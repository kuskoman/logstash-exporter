package startup_manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kuskoman/logstash-exporter/internal/server"
	"github.com/kuskoman/logstash-exporter/pkg/collector_manager"
	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

// shutdownPrometheus unregisters the Prometheus collector
func (sm *StartupManager) shutdownPrometheus() {
	if sm.prometheusCollector != nil {
		slog.Info("unregistering prometheus collector")
		prometheus.Unregister(sm.prometheusCollector)
	} else {
		slog.Debug("prometheus collector is nil")
	}
}

// shutdownServer shuts down the HTTP server
func (sm *StartupManager) shutdownServer(ctx context.Context) error {
	if sm.server == nil {
		slog.Debug("server is nil")
		return nil
	}

	slog.Info("shutting down server")
	err := sm.server.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		slog.Debug("server closed gracefully")
		return nil
	}

	if err != nil {
		slog.Debug("server shutdown error", "error", err)
		return err
	}

	slog.Debug("server closed")
	return nil
}

// startPrometheus initializes and registers the Prometheus collector
func (sm *StartupManager) startPrometheus(cfg *config.Config) {
	// First, unregister any existing collector to prevent duplicate registration errors
	// This is especially important in tests where multiple StartupManager instances might be created
	if sm.prometheusCollector != nil {
		prometheus.Unregister(sm.prometheusCollector)
	}
	
	collectorManager := collector_manager.NewCollectorManager(
		cfg.Logstash.Instances,
		cfg.Logstash.HttpTimeout,
	)

	sm.prometheusCollector = collectorManager
	
	// Try to register, but handle errors gracefully
	if err := prometheus.Register(sm.prometheusCollector); err != nil {
		// In production, we want to panic, but in tests, we should handle this more gracefully
		alreadyRegistered := false
		if reg, ok := err.(prometheus.AlreadyRegisteredError); ok {
			sm.prometheusCollector = reg.ExistingCollector
			alreadyRegistered = true
		}
		
		if !alreadyRegistered {
			// If it's another type of error, panic
			panic(err)
		}
	}
}

// startServer initializes and starts the HTTP server
func (sm *StartupManager) startServer(cfg *config.Config) {
	slog.Debug("creating new app server instance", "config", fmt.Sprintf("%+v", cfg.Server))
	appServer := server.NewAppServer(cfg)
	sm.server = appServer

	go func() {
		slog.Info("starting server", "host", cfg.Server.Host, "port", cfg.Server.Port)
		
		var err error
		if cfg.Server.EnableSSL {
			// Validate TLS configuration
			if cfg.Server.CertFile == "" || cfg.Server.KeyFile == "" {
				err = fmt.Errorf("TLS is enabled but certFile or keyFile is missing")
				sm.serverErrorChan <- err
				return
			}
			
			slog.Info("starting HTTPS server", "certFile", cfg.Server.CertFile, "keyFile", cfg.Server.KeyFile)
			err = appServer.ListenAndServeTLS(cfg.Server.CertFile, cfg.Server.KeyFile)
		} else {
			err = appServer.ListenAndServe()
		}
		
		sm.serverErrorChan <- err
	}()
}