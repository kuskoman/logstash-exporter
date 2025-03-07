package k8s_controller

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kuskoman/logstash-exporter/pkg/collector_manager"
	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// Controller watches Kubernetes resources with specific annotations and configures
// the collector to monitor Logstash instances based on those annotations.
type Controller struct {
	client           kubernetes.Interface
	config           config.KubernetesConfig
	collectorMgr     *collector_manager.CollectorManager
	stopCh           chan struct{}
	mu               sync.Mutex
	resourceHandlers map[string]ResourceHandler
	runningWorker    bool
}

// NewController creates a new Kubernetes controller
func NewController(kubeConfig config.KubernetesConfig, collectorMgr *collector_manager.CollectorManager) (*Controller, error) {
	if !kubeConfig.Enabled {
		return nil, nil
	}

	var config *rest.Config
	var err error

	if kubeConfig.KubeConfig != "" {
		// Use kubeconfig file if specified
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig.KubeConfig)
	} else {
		// Use in-cluster config
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client config: %v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	// Create controller with empty resource handlers map
	controller := &Controller{
		client:           client,
		config:           kubeConfig,
		collectorMgr:     collectorMgr,
		stopCh:           make(chan struct{}),
		resourceHandlers: make(map[string]ResourceHandler),
	}

	// Register resource handlers
	podHandler := NewPodResourceHandler(client, collectorMgr, kubeConfig)
	controller.resourceHandlers[podHandler.Name()] = podHandler

	serviceHandler := NewServiceResourceHandler(client, collectorMgr, kubeConfig)
	controller.resourceHandlers[serviceHandler.Name()] = serviceHandler

	return controller, nil
}

// Start starts the controller
func (c *Controller) Start(ctx context.Context) error {
	if c == nil {
		// Controller is nil when Kubernetes is not enabled
		return nil
	}

	slog.Info("starting Kubernetes controller", 
		"namespaces", c.config.Namespaces)

	// If no namespaces are specified, watch all namespaces
	namespaces := c.config.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{metav1.NamespaceAll}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Start all resource handlers
	for name, handler := range c.resourceHandlers {
		slog.Debug("starting resource handler", "name", name)
		if err := handler.Start(ctx, namespaces); err != nil {
			return fmt.Errorf("failed to start resource handler %s: %v", name, err)
		}
	}

	// Start the worker
	if !c.runningWorker {
		c.runningWorker = true
		go wait.Until(c.worker, c.config.ScrapeInterval, c.stopCh)
	}

	return nil
}

// Stop stops the controller
func (c *Controller) Stop(ctx context.Context) error {
	if c == nil {
		// Controller is nil when Kubernetes is not enabled
		return nil
	}

	slog.Info("stopping Kubernetes controller")

	// Stop all resource handlers
	for name, handler := range c.resourceHandlers {
		slog.Debug("stopping resource handler", "name", name)
		handler.Stop()
	}

	close(c.stopCh)
	return nil
}

// worker performs periodic reconciliation
func (c *Controller) worker() {
	slog.Debug("kubernetes controller worker running")
	
	// Worker is now just a heartbeat since each resource handler
	// manages its own resources
}