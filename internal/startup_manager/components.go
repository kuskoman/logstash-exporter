package startup_manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kuskoman/logstash-exporter/internal/k8s_controller"
	"github.com/kuskoman/logstash-exporter/internal/server"
	"github.com/kuskoman/logstash-exporter/pkg/collector_manager"
	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/kuskoman/logstash-exporter/pkg/tls"
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

		// First validate the server TLS configuration
		if err := cfg.Server.ValidateServerTLS(); err != nil {
			sm.serverErrorChan <- fmt.Errorf("invalid TLS configuration: %w", err)
			return
		}

		// Configure TLS if either the advanced or legacy TLS configuration is present
		if cfg.Server.TLSConfig != nil || cfg.Server.EnableSSL {
			tlsConfig, err := tls.ConfigureServerTLS(cfg)
			if err != nil {
				sm.serverErrorChan <- fmt.Errorf("failed to configure TLS: %w", err)
				return
			}

			// Apply the TLS configuration to the server
			appServer.TLSConfig = tlsConfig

			slog.Info("starting HTTPS server")

			// Start TLS listener without specifying cert and key files since they're
			// already provided in the TLS configuration
			err = appServer.ListenAndServeTLS("", "")
		} else {
			err = appServer.ListenAndServe()
		}

		sm.serverErrorChan <- err
	}()
}

// startKubernetesController initializes and starts the Kubernetes controller
func (sm *StartupManager) startKubernetesController(cfg *config.Config) {
	if !cfg.Kubernetes.Enabled {
		slog.Info("kubernetes controller is disabled")
		return
	}

	slog.Info("starting kubernetes controller")

	// Get collector manager from Prometheus collector
	var collectorMgr *collector_manager.CollectorManager
	if sm.prometheusCollector != nil {
		var ok bool
		collectorMgr, ok = sm.prometheusCollector.(*collector_manager.CollectorManager)
		if !ok {
			slog.Error("collector is not a CollectorManager, cannot start kubernetes controller")
			return
		}
	}

	if collectorMgr == nil {
		slog.Error("collector manager is nil, cannot start kubernetes controller")
		return
	}

	controller, err := k8s_controller.NewController(cfg.Kubernetes, collectorMgr)
	if err != nil {
		slog.Error("failed to create kubernetes controller", "error", err)
		return
	}

	sm.kubernetesController = controller

	// Start the controller in a separate goroutine
	go func() {
		if err := controller.Start(context.Background()); err != nil {
			slog.Error("failed to start kubernetes controller", "error", err)
		}
	}()
}

// shutdownKubernetesController stops the Kubernetes controller
func (sm *StartupManager) shutdownKubernetesController(ctx context.Context) {
	if sm.kubernetesController == nil {
		slog.Debug("kubernetes controller is nil")
		return
	}

	slog.Info("shutting down kubernetes controller")
	if err := sm.kubernetesController.Stop(ctx); err != nil {
		slog.Error("failed to stop kubernetes controller", "error", err)
	}
}
