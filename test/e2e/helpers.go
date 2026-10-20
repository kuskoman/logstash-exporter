package e2e

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const (
	startupTimeout  = 10 * time.Second
	shutdownTimeout = 5 * time.Second
	maxStartRetries = 3
)

// Exporter is a running exporter binary under test
type Exporter struct {
	Port       int
	ConfigPath string

	t        *testing.T
	scheme   string
	client   *http.Client
	cmd      *exec.Cmd
	logs     *syncBuffer
	exited   chan struct{}
	exitErr  error
	stopOnce sync.Once
}

// ExporterOption configures how the exporter is started
type ExporterOption func(*exporterSettings)

type exporterSettings struct {
	args   []string
	env    []string
	scheme string
}

// WithWatch enables configuration hot reload
func WithWatch() ExporterOption {
	return func(s *exporterSettings) { s.args = append(s.args, "-watch") }
}

// WithEnv adds environment variables in KEY=VALUE form
func WithEnv(env ...string) ExporterOption {
	return func(s *exporterSettings) { s.env = append(s.env, env...) }
}

// WithHTTPS makes the test client talk to the exporter over HTTPS
func WithHTTPS() ExporterOption {
	return func(s *exporterSettings) { s.scheme = "https" }
}

// StartExporter writes the configuration, starts the exporter and waits until
// it serves HTTP. It picks a free port and retries when the port is taken by
// another process before the exporter binds it. The exporter is stopped when
// the test ends, and its logs are printed if the test failed.
func StartExporter(t *testing.T, cb *ConfigBuilder, opts ...ExporterOption) *Exporter {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yml")
	return StartExporterWithConfigFile(t, configPath, func(port int) {
		cb.WithServerPort(port)
		if err := os.WriteFile(configPath, []byte(cb.Build()), 0644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}
	}, opts...)
}

// StartExporterWithConfigFile starts the exporter with the config file at
// configPath. writeConfig is called with the chosen port before every start
// attempt and must write a configuration that listens on it.
func StartExporterWithConfigFile(t *testing.T, configPath string, writeConfig func(port int), opts ...ExporterOption) *Exporter {
	t.Helper()

	settings := &exporterSettings{scheme: "http"}
	for _, opt := range opts {
		opt(settings)
	}

	var lastErr error
	for attempt := 1; attempt <= maxStartRetries; attempt++ {
		port, err := GetFreePort()
		if err != nil {
			t.Fatalf("failed to get free port: %v", err)
		}
		writeConfig(port)

		exporter := startProcess(t, configPath, port, settings)
		err = exporter.waitUntilServing(startupTimeout)
		if err == nil {
			t.Cleanup(exporter.cleanup)
			return exporter
		}

		exporter.Stop()
		lastErr = err
		if !strings.Contains(exporter.Logs(), "address already in use") {
			t.Fatalf("exporter did not start: %v\nexporter logs:\n%s", err, exporter.Logs())
		}
		t.Logf("port %d was taken, retrying (attempt %d/%d)", port, attempt, maxStartRetries)
	}

	t.Fatalf("exporter did not start after %d attempts: %v", maxStartRetries, lastErr)
	return nil
}

func startProcess(t *testing.T, configPath string, port int, settings *exporterSettings) *Exporter {
	t.Helper()

	args := append([]string{"-config", configPath}, settings.args...)
	cmd := exec.Command(GetBinaryPath(t), args...)
	cmd.Env = append(os.Environ(), settings.env...)

	logs := &syncBuffer{}
	cmd.Stdout = logs
	cmd.Stderr = logs

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start exporter: %v", err)
	}

	exporter := &Exporter{
		Port:       port,
		ConfigPath: configPath,
		t:          t,
		scheme:     settings.scheme,
		client:     newTestHTTPClient(),
		cmd:        cmd,
		logs:       logs,
		exited:     make(chan struct{}),
	}

	go func() {
		exporter.exitErr = cmd.Wait()
		close(exporter.exited)
	}()

	return exporter
}

func newTestHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			// The exporter uses a self-signed certificate in TLS tests
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			DisableKeepAlives: true,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (e *Exporter) waitUntilServing(timeout time.Duration) error {
	return poll(timeout, func() error {
		select {
		case <-e.exited:
			return fmt.Errorf("exporter exited: %v", e.exitErr)
		default:
		}

		resp, err := e.Get("/version")
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("/version returned %d", resp.StatusCode)
		}
		return nil
	})
}

func (e *Exporter) cleanup() {
	e.Stop()
	if e.t.Failed() {
		e.t.Logf("exporter logs:\n%s", e.Logs())
	}
}

// Stop terminates the exporter. It sends SIGTERM and kills the process if it
// does not exit within the shutdown timeout.
func (e *Exporter) Stop() {
	e.stopOnce.Do(func() {
		select {
		case <-e.exited:
			return
		default:
		}

		_ = e.cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-e.exited:
		case <-time.After(shutdownTimeout):
			_ = e.cmd.Process.Kill()
			<-e.exited
		}
	})
}

// Signal sends a signal to the exporter process
func (e *Exporter) Signal(sig os.Signal) error {
	return e.cmd.Process.Signal(sig)
}

// WaitForExit waits for the exporter to exit and returns its exit code
func (e *Exporter) WaitForExit(timeout time.Duration) (int, error) {
	select {
	case <-e.exited:
		return exitCode(e.exitErr), nil
	case <-time.After(timeout):
		return 0, fmt.Errorf("exporter did not exit within %s", timeout)
	}
}

// Logs returns everything the exporter wrote to stdout and stderr
func (e *Exporter) Logs() string {
	return e.logs.String()
}

// URL returns the URL of the given path on the exporter
func (e *Exporter) URL(path string) string {
	return fmt.Sprintf("%s://127.0.0.1:%d%s", e.scheme, e.Port, path)
}

// Response is a fully read HTTP response
type Response struct {
	StatusCode int
	Header     http.Header
	Body       string
}

// Get requests the given path on the exporter
func (e *Exporter) Get(path string) (*Response, error) {
	return e.GetWithAuth(path, "", "")
}

// GetWithAuth requests the given path using basic authentication when a
// username is given
func (e *Exporter) GetWithAuth(path, username, password string) (*Response, error) {
	req, err := http.NewRequest(http.MethodGet, e.URL(path), nil)
	if err != nil {
		return nil, err
	}
	if username != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: string(body)}, nil
}

// Metrics fetches and parses the metrics endpoint
func (e *Exporter) Metrics() (*Metrics, error) {
	resp, err := e.Get("/metrics")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/metrics returned %d: %s", resp.StatusCode, resp.Body)
	}
	return ParseMetrics(resp.Body)
}

// WaitForMetrics polls the metrics endpoint until check succeeds and returns
// the metrics that satisfied it
func (e *Exporter) WaitForMetrics(check func(*Metrics) error) *Metrics {
	e.t.Helper()

	var result *Metrics
	Eventually(e.t, defaultWaitTimeout, func() error {
		metrics, err := e.Metrics()
		if err != nil {
			return err
		}
		if err := check(metrics); err != nil {
			return err
		}
		result = metrics
		return nil
	})
	return result
}

// WaitForInstances waits until metrics are exported for exactly the given
// number of Logstash instances
func (e *Exporter) WaitForInstances(count int) *Metrics {
	e.t.Helper()

	return e.WaitForMetrics(func(m *Metrics) error {
		if instances := m.Instances(); len(instances) != count {
			return fmt.Errorf("expected %d instances, got %d: %v", count, len(instances), instances)
		}
		return nil
	})
}

// WriteConfig replaces the configuration file in place. The exporter port is
// kept unless the builder sets a different one.
func (e *Exporter) WriteConfig(cb *ConfigBuilder) {
	e.t.Helper()

	if cb.serverPort == 0 {
		cb.WithServerPort(e.Port)
	}
	if err := os.WriteFile(e.ConfigPath, []byte(cb.Build()), 0644); err != nil {
		e.t.Fatalf("failed to write config file: %v", err)
	}
}

// RunResult is the outcome of running the exporter to completion
type RunResult struct {
	ExitCode int
	Output   string
}

// RunExporter runs the exporter binary with the given arguments until it
// exits, and fails the test if it does not exit within the timeout
func RunExporter(t *testing.T, timeout time.Duration, args ...string) RunResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, GetBinaryPath(t), args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("exporter did not exit within %s\noutput:\n%s", timeout, output)
	}

	return RunResult{ExitCode: exitCode(err), Output: string(output)}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// GetBinaryPath returns the path to the exporter binary built by
// `make e2e-prepare`
func GetBinaryPath(t *testing.T) string {
	t.Helper()

	binaryPath := filepath.Join("..", "..", "test", "e2e", "bin", "logstash-exporter-e2e")
	if _, err := os.Stat(binaryPath); err != nil {
		t.Fatalf("e2e binary not found at %s. Please run 'make e2e-prepare' first", binaryPath)
	}

	return binaryPath
}

// GetFreePort asks the operating system for a free TCP port. Another process
// can still take it before the exporter binds it, which StartExporter handles
// by retrying.
func GetFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

// syncBuffer is a bytes.Buffer safe for concurrent writes and reads
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
