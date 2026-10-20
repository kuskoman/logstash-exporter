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
		sm.registerer.Unregister(sm.prometheusCollector)
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
		sm.registerer.Unregister(sm.prometheusCollector)
	}

	collectorManager := collector_manager.NewCollectorManager(
		cfg.Logstash.Instances,
		cfg.Logstash.HttpTimeout,
	)

	sm.prometheusCollector = collectorManager

	// Try to register, but handle errors gracefully
	if err := sm.registerer.Register(sm.prometheusCollector); err != nil {
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

// newAppServer creates the HTTP server for the given configuration and
// applies its TLS settings
func newAppServer(cfg *config.Config) (AppServer, error) {
	slog.Debug("creating new app server instance", "config", fmt.Sprintf("%+v", cfg.Server))
	appServer := server.NewAppServer(cfg)

	if err := cfg.Server.ValidateServerTLS(); err != nil {
		return nil, fmt.Errorf("invalid TLS configuration: %w", err)
	}

	if cfg.Server.TLSConfig != nil {
		tlsConfig, err := tls.ConfigureServerTLS(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		appServer.TLSConfig = tlsConfig
	}

	return appServer, nil
}

// startServer creates the server and starts listening in a separate
// goroutine. Errors, including the server being closed, are sent to
// serverErrorChan.
func (sm *StartupManager) startServer(cfg *config.Config) {
	appServer, err := sm.newServer(cfg)
	if err != nil {
		go func() { sm.serverErrorChan <- err }()
		return
	}
	sm.server = appServer

	go func() {
		slog.Info("starting server", "host", cfg.Server.Host, "port", cfg.Server.Port)

		if cfg.Server.TLSConfig != nil {
			slog.Info("starting HTTPS server")

			// The certificate and key are already part of the server's TLS
			// configuration, so they are not passed here
			sm.serverErrorChan <- appServer.ListenAndServeTLS("", "")
		} else {
			sm.serverErrorChan <- appServer.ListenAndServe()
		}
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
