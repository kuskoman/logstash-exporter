package startup_manager

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

const testTimeout = 2 * time.Second

// mockServer behaves like an http.Server: ListenAndServe blocks until
// Shutdown is called and then returns http.ErrServerClosed
type mockServer struct {
	mu            sync.Mutex
	listenErr     error
	shutdownErr   error
	usedTLS       bool
	shutdownCalls int

	listening chan struct{}
	stopped   chan struct{}
	stopOnce  sync.Once
}

func newMockServer() *mockServer {
	return &mockServer{
		listening: make(chan struct{}),
		stopped:   make(chan struct{}),
	}
}

func (s *mockServer) ListenAndServe() error {
	return s.serve(false)
}

func (s *mockServer) ListenAndServeTLS(certFile, keyFile string) error {
	return s.serve(true)
}

func (s *mockServer) serve(useTLS bool) error {
	s.mu.Lock()
	s.usedTLS = useTLS
	listenErr := s.listenErr
	s.mu.Unlock()
	close(s.listening)

	if listenErr != nil {
		return listenErr
	}
	<-s.stopped
	return http.ErrServerClosed
}

func (s *mockServer) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.shutdownCalls++
	shutdownErr := s.shutdownErr
	s.mu.Unlock()

	s.stopOnce.Do(func() { close(s.stopped) })
	return shutdownErr
}

func (s *mockServer) ShutdownCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shutdownCalls
}

func (s *mockServer) UsedTLS() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usedTLS
}

// waitUntilListening fails the test if the server is not started in time
func (s *mockServer) waitUntilListening(t *testing.T) {
	t.Helper()

	select {
	case <-s.listening:
	case <-time.After(testTimeout):
		t.Fatal("server was not started")
	}
}

// mockServerFactory creates mock servers and records the configuration each
// server was created with
type mockServerFactory struct {
	mu        sync.Mutex
	err       error
	listenErr error
	configs   []*config.Config
	servers   []*mockServer
	created   chan *mockServer
}

func newMockServerFactory() *mockServerFactory {
	return &mockServerFactory{created: make(chan *mockServer, 10)}
}

func (f *mockServerFactory) create(cfg *config.Config) (AppServer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.configs = append(f.configs, cfg)
	if f.err != nil {
		return nil, f.err
	}

	server := newMockServer()
	server.listenErr = f.listenErr
	f.servers = append(f.servers, server)
	f.created <- server
	return server, nil
}

// Servers returns all servers created so far
func (f *mockServerFactory) Servers() []*mockServer {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*mockServer(nil), f.servers...)
}

// Configs returns the configurations servers were created with
func (f *mockServerFactory) Configs() []*config.Config {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*config.Config(nil), f.configs...)
}

// waitForServer returns the next created server, which is listening
func (f *mockServerFactory) waitForServer(t *testing.T) *mockServer {
	t.Helper()

	select {
	case server := <-f.created:
		server.waitUntilListening(t)
		return server
	case <-time.After(testTimeout):
		t.Fatal("no server was created")
		return nil
	}
}

// mockConfigSource returns a configured config and change status
type mockConfigSource struct {
	mu          sync.Mutex
	cfg         *config.Config
	changed     bool
	err         error
	loadCalls   int
	reloadCalls int
	reloadCtx   context.Context
}

func newMockConfigSource(cfg *config.Config) *mockConfigSource {
	return &mockConfigSource{cfg: cfg, changed: true}
}

func (m *mockConfigSource) LoadAndCompareConfig(ctx context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadCalls++
	return m.changed, m.err
}

func (m *mockConfigSource) ReloadAndCompareConfig(ctx context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadCalls++
	m.reloadCtx = ctx
	return m.changed, m.err
}

func (m *mockConfigSource) GetCurrentConfig() *config.Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// set changes what the next load or reload returns
func (m *mockConfigSource) set(cfg *config.Config, changed bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg, m.changed, m.err = cfg, changed, err
}

func (m *mockConfigSource) ReloadCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reloadCalls
}

// newTestStartupManager creates a StartupManager that uses the given config
// source, creates mock servers and registers collectors in a private registry
func newTestStartupManager(source configSource, factory *mockServerFactory) (*StartupManager, *prometheus.Registry) {
	registry := prometheus.NewRegistry()
	sm := &StartupManager{
		configPath:      "config.yml",
		configManager:   source,
		newServer:       factory.create,
		registerer:      registry,
		serverErrorChan: make(chan error),
	}
	return sm, registry
}

// runInitialize runs Initialize in the background and returns a channel with
// its result. The context is cancelled when the test ends.
func runInitialize(t *testing.T, sm *StartupManager) (context.CancelFunc, <-chan error) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		result <- sm.Initialize(ctx)
		close(done)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(testTimeout):
			t.Error("Initialize did not return after the context was cancelled")
		}
	})

	return cancel, result
}

// waitForResult fails the test if Initialize does not return in time
func waitForResult(t *testing.T, result <-chan error) error {
	t.Helper()

	select {
	case err := <-result:
		return err
	case <-time.After(testTimeout):
		t.Fatal("Initialize did not return")
		return nil
	}
}

// isRegistered reports whether this collector instance is registered in the
// registry. The registry identifies collectors by their metric descriptors,
// so a different collector with the same metrics does not count.
func isRegistered(registry *prometheus.Registry, collector prometheus.Collector) bool {
	err := registry.Register(collector)
	if err == nil {
		registry.Unregister(collector)
		return false
	}

	var alreadyRegistered prometheus.AlreadyRegisteredError
	return errors.As(err, &alreadyRegistered) && alreadyRegistered.ExistingCollector == collector
}
