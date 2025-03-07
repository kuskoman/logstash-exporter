package k8s_controller

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kuskoman/logstash-exporter/pkg/collector_manager"
	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// Controller watches Kubernetes pods with specific annotations and configures
// the collector to monitor Logstash instances based on those annotations.
type Controller struct {
	client        kubernetes.Interface
	config        config.KubernetesConfig
	collectorMgr  *collector_manager.CollectorManager
	stopCh        chan struct{}
	mu            sync.Mutex
	podInformers  []cache.SharedIndexInformer
	podStores     []cache.Store
	runningWorker bool
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

	return &Controller{
		client:       client,
		config:       kubeConfig,
		collectorMgr: collectorMgr,
		stopCh:       make(chan struct{}),
	}, nil
}

// Start starts the controller
func (c *Controller) Start(ctx context.Context) error {
	if c == nil {
		// Controller is nil when Kubernetes is not enabled
		return nil
	}

	slog.Info("starting Kubernetes controller", 
		"annotationPrefix", c.config.PodAnnotationPrefix,
		"namespaces", c.config.Namespaces)

	// If no namespaces are specified, watch all namespaces
	namespaces := c.config.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{metav1.NamespaceAll}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Create an informer for each namespace
	for _, namespace := range namespaces {
		// Create a pod informer that filters for pods with our annotation prefix
		fieldSelector := fields.Everything()
		
		podListWatcher := cache.NewListWatchFromClient(
			c.client.CoreV1().RESTClient(),
			"pods",
			namespace,
			fieldSelector,
		)

		informer := cache.NewSharedIndexInformer(
			podListWatcher,
			&corev1.Pod{},
			c.config.ResyncPeriod,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		)

		informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onPodAdd,
			UpdateFunc: c.onPodUpdate,
			DeleteFunc: c.onPodDelete,
		})

		c.podInformers = append(c.podInformers, informer)
		c.podStores = append(c.podStores, informer.GetStore())
	}

	// Start all informers
	for _, informer := range c.podInformers {
		go informer.Run(c.stopCh)
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
	close(c.stopCh)
	return nil
}

// onPodAdd is called when a pod is added
func (c *Controller) onPodAdd(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		slog.Warn("unexpected type in pod add event handler")
		return
	}

	c.processPod(pod)
}

// onPodUpdate is called when a pod is updated
func (c *Controller) onPodUpdate(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*corev1.Pod)
	if !ok {
		slog.Warn("unexpected type in pod update event handler (old object)")
		return
	}

	newPod, ok := newObj.(*corev1.Pod)
	if !ok {
		slog.Warn("unexpected type in pod update event handler (new object)")
		return
	}

	// Check if annotations have changed
	if haveSameAnnotations(oldPod, newPod) {
		return
	}

	c.processPod(newPod)
}

// onPodDelete is called when a pod is deleted
func (c *Controller) onPodDelete(obj interface{}) {
	var pod *corev1.Pod
	switch t := obj.(type) {
	case *corev1.Pod:
		pod = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*corev1.Pod)
		if !ok {
			slog.Warn("unexpected type in pod delete event handler")
			return
		}
	default:
		slog.Warn("unexpected type in pod delete event handler")
		return
	}

	c.removePod(pod)
}

// haveSameAnnotations checks if two pods have the same relevant annotations
func haveSameAnnotations(pod1, pod2 *corev1.Pod) bool {
	return fmt.Sprintf("%v", pod1.Annotations) == fmt.Sprintf("%v", pod2.Annotations)
}

// processPod processes a pod to see if it has the required annotations
func (c *Controller) processPod(pod *corev1.Pod) {
	if pod.Status.Phase != corev1.PodRunning {
		return
	}

	logstashURL, ok := pod.Annotations[c.config.LogstashURLAnnotation]
	if !ok {
		return
	}

	// Get auth credentials from annotations if needed
	// TODO: Add auth support when implementing the collector manager auth API
	_ = pod.Annotations[c.config.LogstashUsernameAnnotation]
	_ = pod.Annotations[c.config.LogstashPasswordAnnotation]

	instanceName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	slog.Info("discovered logstash instance from pod annotation", 
		"instance", instanceName, 
		"url", logstashURL)

	// Create a new LogstashInstance with the discovered information
	instance := &config.LogstashInstance{
		Host: logstashURL,
		Name: instanceName,
	}

	// Set HttpInsecure to true if the URL is HTTPS
	if strings.HasPrefix(logstashURL, "https://") {
		instance.HttpInsecure = true
	}

	// Add the instance to the collector manager
	c.collectorMgr.AddInstance(instanceName, instance)
}

// removePod removes a pod from monitoring
func (c *Controller) removePod(pod *corev1.Pod) {
	instanceName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	slog.Info("removing logstash instance", "instance", instanceName)

	// Remove the instance from the collector manager
	c.collectorMgr.RemoveInstance(instanceName)
}

// worker performs periodic reconciliation
func (c *Controller) worker() {
	slog.Debug("kubernetes controller worker running")
	
	// Get all pods from all stores
	pods := []interface{}{}
	for _, store := range c.podStores {
		storePods := store.List()
		pods = append(pods, storePods...)
	}

	// Process all pods
	for _, obj := range pods {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}
		c.processPod(pod)
	}
}