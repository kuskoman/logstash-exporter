package startup_manager

import (
	"testing"
	"time"
	"net"
	"strconv"
	"context"

	"github.com/kuskoman/logstash-exporter/internal/flags"
)

func TestAppServerNoTLS(t *testing.T) {
	flagsConfig := &flags.FlagsConfig{ConfigLocation: "../../fixtures/valid_config.yml"}

	ctx := context.TODO()
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

	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)), timeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil {
		defer conn.Close()
	}
}

func TestAppServerTLS(t *testing.T) {
	flagsConfig := &flags.FlagsConfig{ConfigLocation: "../../fixtures/valid_config.yml"}

	ctx := context.TODO()
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

	t.Log("Swaggg")
	t.Logf("Host: %s, port: %d", cfg.Server.Host, cfg.Server.Port)
	go func() {
		sm.startServer(cfg)
	}()

	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)), timeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != nil {
		defer conn.Close()
	}
}
