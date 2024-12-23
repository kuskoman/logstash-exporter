package startup_manager

import (
	"context"
	"crypto/tls"
	"net"
	"strconv"
	"testing"
	"time"

	"os"
	"crypto/x509"

	"github.com/kuskoman/logstash-exporter/internal/flags"
)


func TestAppServer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	timeout := time.Second

	t.Run("No TLS", func(t *testing.T) {
		t.Parallel()

		flagsConfig := &flags.FlagsConfig{ConfigLocation: "../../fixtures/valid_config.yml"}

		sm, err := NewStartupManager(flagsConfig.ConfigLocation, flagsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = sm.configManager.LoadAndCompareConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg := sm.configManager.GetCurrentConfig()
		if cfg == nil {
			t.Fatal("config is nil")
		}

		go func() {
			sm.startServer(cfg)
		}()

		name := net.JoinHostPort("localhost", strconv.Itoa(cfg.Server.Port))
		go func(t *testing.T) {
			conn, err := net.DialTimeout("tcp", name, timeout)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if conn != nil {
				defer conn.Close()
			}
		}(t)

		err = sm.shutdownServer(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("TLS", func(t *testing.T) {
		t.Parallel()

		flagsConfig := &flags.FlagsConfig{ConfigLocation: "../../fixtures/https/config.yml"}

		sm, err := NewStartupManager(flagsConfig.ConfigLocation, flagsConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = sm.configManager.LoadAndCompareConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg := sm.configManager.GetCurrentConfig()
		if cfg == nil {
			t.Fatal("config is nil")
		}

		go func() {
			sm.startServer(cfg)
		}()

		cert, err := os.ReadFile("../../fixtures/https/ca.crt")
		if err != nil {
			t.Fatalf("Failed to read certificate file: %v", err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cert)

		tlsConfig := &tls.Config{
			RootCAs: caCertPool,
		}

		dialer := net.Dialer{Timeout: timeout}
		name := net.JoinHostPort("localhost", strconv.Itoa(cfg.Server.Port))
		go func(t *testing.T) {
			conn, err := tls.DialWithDialer(&dialer, "tcp", name, tlsConfig)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if conn != nil {
				defer conn.Close()
			}
		}(t)

		err = sm.shutdownServer(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
