package k8s_controller

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

// InstanceManager adds and removes the Logstash instances that are scraped.
// It is implemented by collector_manager.CollectorManager.
type InstanceManager interface {
	AddInstance(id string, instance *config.LogstashInstance)
	RemoveInstance(id string)
}

// ResourceHandler defines the interface for handling different Kubernetes resource types
type ResourceHandler interface {
	// Start starts watching the resource
	Start(ctx context.Context, namespaces []string) error
	// Stop stops watching the resource
	Stop()
	// Name returns the name of the resource handler
	Name() string
}

// BaseResourceHandler contains common functionality for all resource handlers
type BaseResourceHandler struct {
	client         kubernetes.Interface
	collectorMgr   InstanceManager
	config         config.KubernetesConfig
	resourceConfig config.ResourceConfig
	mu             sync.RWMutex
	informers      []cache.SharedIndexInformer
	stores         []cache.Store
	stopCh         chan struct{}
}

// newBaseResourceHandler creates a new base resource handler
func newBaseResourceHandler(
	client kubernetes.Interface,
	collectorMgr InstanceManager,
	kubeConfig config.KubernetesConfig,
	resourceConfig config.ResourceConfig,
) *BaseResourceHandler {
	return &BaseResourceHandler{
		client:         client,
		collectorMgr:   collectorMgr,
		config:         kubeConfig,
		resourceConfig: resourceConfig,
		stopCh:         make(chan struct{}),
	}
}

// Stop stops watching the resource
func (h *BaseResourceHandler) Stop() {
	close(h.stopCh)
}

// extractLogstashInfo extracts Logstash connection info from object annotations
func (h *BaseResourceHandler) extractLogstashInfo(annotations map[string]string, resourceName string) (string, *config.LogstashInstance) {
	logstashURL, ok := annotations[h.config.LogstashURLAnnotation]
	if !ok {
		return "", nil
	}

	// Create a new LogstashInstance with the discovered information
	instance := &config.LogstashInstance{
		Host: logstashURL,
		Name: resourceName,
	}

	// Use basic authentication if the credentials annotations are set
	if username := annotations[h.config.LogstashUsernameAnnotation]; username != "" {
		instance.BasicAuth = &config.ClientAuthConfig{
			Username: username,
			Password: annotations[h.config.LogstashPasswordAnnotation],
		}
	}

	// Configure TLS if the URL is HTTPS
	if strings.HasPrefix(logstashURL, "https://") {
		instance.TLSConfig = &config.TLSClientConfig{
			InsecureSkipVerify: true, // Default to insecure for discovered instances
		}
	}

	return resourceName, instance
}

// PodResourceHandler handles Pod resources
type PodResourceHandler struct {
	*BaseResourceHandler
}

// NewPodResourceHandler creates a new Pod resource handler
func NewPodResourceHandler(
	client kubernetes.Interface,
	collectorMgr InstanceManager,
	kubeConfig config.KubernetesConfig,
) ResourceHandler {
	return &PodResourceHandler{
		BaseResourceHandler: newBaseResourceHandler(
			client,
			collectorMgr,
			kubeConfig,
			kubeConfig.Resources.Pods,
		),
	}
}

// Name returns the name of the resource handler
func (h *PodResourceHandler) Name() string {
	return "pods"
}

// Start starts watching pods
func (h *PodResourceHandler) Start(ctx context.Context, namespaces []string) error {
	if !h.resourceConfig.Enabled {
		slog.Info("pod monitoring is disabled")
		return nil
	}

	slog.Info("starting pod monitoring",
		"annotationPrefix", h.resourceConfig.AnnotationPrefix,
		"namespaces", namespaces)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Create an informer for each namespace
	for _, namespace := range namespaces {
		podListWatcher := &cache.ListWatch{
			ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
				return h.client.CoreV1().Pods(namespace).List(ctx, options)
			},
			WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
				return h.client.CoreV1().Pods(namespace).Watch(ctx, options)
			},
		}

		informer := cache.NewSharedIndexInformer(
			podListWatcher,
			&corev1.Pod{},
			h.config.ResyncPeriod,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		)

		_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    h.onPodAdd,
			UpdateFunc: h.onPodUpdate,
			DeleteFunc: h.onPodDelete,
		})
		if err != nil {
			return fmt.Errorf("failed to add event handler to pod informer: %w", err)
		}

		h.informers = append(h.informers, informer)
		h.stores = append(h.stores, informer.GetStore())
	}

	// Start all informers
	for _, informer := range h.informers {
		go informer.Run(h.stopCh)
	}

	return nil
}

// onPodAdd is called when a pod is added
func (h *PodResourceHandler) onPodAdd(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		slog.Warn("unexpected type in pod add event handler")
		return
	}

	h.processPod(pod)
}

// onPodUpdate is called when a pod is updated
func (h *PodResourceHandler) onPodUpdate(oldObj, newObj interface{}) {
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

	// Pods are created before they run and get an IP, and can stop running
	// or lose their annotation later, so compare what each version means for
	// scraping instead of only the annotations
	_, oldInstance := h.podInstance(oldPod)
	resourceName, newInstance := h.podInstance(newPod)

	switch {
	case newInstance == nil && oldInstance != nil:
		h.removePod(newPod)
	case newInstance != nil && !reflect.DeepEqual(oldInstance, newInstance):
		slog.Info("discovered logstash instance from pod annotation",
			"instance", resourceName,
			"url", newInstance.Host)
		h.collectorMgr.AddInstance(resourceName, newInstance)
	}
}

// onPodDelete is called when a pod is deleted
func (h *PodResourceHandler) onPodDelete(obj interface{}) {
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

	h.removePod(pod)
}

// processPod adds a pod to monitoring if it should be scraped
func (h *PodResourceHandler) processPod(pod *corev1.Pod) {
	resourceName, instance := h.podInstance(pod)
	if instance == nil {
		return
	}

	slog.Info("discovered logstash instance from pod annotation",
		"instance", resourceName,
		"url", instance.Host)

	// Add the instance to the collector manager
	h.collectorMgr.AddInstance(resourceName, instance)
}

// podInstance returns the Logstash instance to scrape for a pod, or nil if
// the pod is not running or not annotated
func (h *PodResourceHandler) podInstance(pod *corev1.Pod) (string, *config.LogstashInstance) {
	if pod.Status.Phase != corev1.PodRunning {
		return "", nil
	}

	instanceName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	resourceName, instance := h.extractLogstashInfo(pod.Annotations, instanceName)

	if instance == nil {
		return "", nil
	}

	// Replace $(pod_ip) template variable with the actual pod IP
	if pod.Status.PodIP != "" && instance.Host != "" {
		// Replace $(pod_ip) in the URL string
		replacedURL := strings.ReplaceAll(instance.Host, "$(pod_ip)", pod.Status.PodIP)

		// Validate the URL after replacement
		if _, err := url.Parse(replacedURL); err == nil {
			instance.Host = replacedURL
		} else {
			slog.Warn("failed to parse URL after pod_ip replacement",
				"instance", instanceName,
				"url", replacedURL,
				"error", err)
			return "", nil
		}
	}

	return resourceName, instance
}

// removePod removes a pod from monitoring
func (h *PodResourceHandler) removePod(pod *corev1.Pod) {
	instanceName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	slog.Info("removing logstash instance", "instance", instanceName)

	// Remove the instance from the collector manager
	h.collectorMgr.RemoveInstance(instanceName)
}

// ServiceResourceHandler handles Service resources
type ServiceResourceHandler struct {
	*BaseResourceHandler
}

// NewServiceResourceHandler creates a new Service resource handler
func NewServiceResourceHandler(
	client kubernetes.Interface,
	collectorMgr InstanceManager,
	kubeConfig config.KubernetesConfig,
) ResourceHandler {
	return &ServiceResourceHandler{
		BaseResourceHandler: newBaseResourceHandler(
			client,
			collectorMgr,
			kubeConfig,
			kubeConfig.Resources.Services,
		),
	}
}

// Name returns the name of the resource handler
func (h *ServiceResourceHandler) Name() string {
	return "services"
}

// Start starts watching services
func (h *ServiceResourceHandler) Start(ctx context.Context, namespaces []string) error {
	if !h.resourceConfig.Enabled {
		slog.Info("service monitoring is disabled")
		return nil
	}

	slog.Info("starting service monitoring",
		"annotationPrefix", h.resourceConfig.AnnotationPrefix,
		"namespaces", namespaces)

	h.mu.Lock()
	defer h.mu.Unlock()

	// Create an informer for each namespace
	for _, namespace := range namespaces {
		serviceListWatcher := &cache.ListWatch{
			ListWithContextFunc: func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
				return h.client.CoreV1().Services(namespace).List(ctx, options)
			},
			WatchFuncWithContext: func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
				return h.client.CoreV1().Services(namespace).Watch(ctx, options)
			},
		}

		informer := cache.NewSharedIndexInformer(
			serviceListWatcher,
			&corev1.Service{},
			h.config.ResyncPeriod,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		)

		_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    h.onServiceAdd,
			UpdateFunc: h.onServiceUpdate,
			DeleteFunc: h.onServiceDelete,
		})
		if err != nil {
			return fmt.Errorf("failed to add event handler to service informer: %w", err)
		}

		h.informers = append(h.informers, informer)
		h.stores = append(h.stores, informer.GetStore())
	}

	// Start all informers
	for _, informer := range h.informers {
		go informer.Run(h.stopCh)
	}

	return nil
}

// onServiceAdd is called when a service is added
func (h *ServiceResourceHandler) onServiceAdd(obj interface{}) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		slog.Warn("unexpected type in service add event handler")
		return
	}

	h.processService(service)
}

// onServiceUpdate is called when a service is updated
func (h *ServiceResourceHandler) onServiceUpdate(oldObj, newObj interface{}) {
	oldService, ok := oldObj.(*corev1.Service)
	if !ok {
		slog.Warn("unexpected type in service update event handler (old object)")
		return
	}

	newService, ok := newObj.(*corev1.Service)
	if !ok {
		slog.Warn("unexpected type in service update event handler (new object)")
		return
	}

	_, oldInstance := h.serviceInstance(oldService)
	resourceName, newInstance := h.serviceInstance(newService)

	switch {
	case newInstance == nil && oldInstance != nil:
		h.removeService(newService)
	case newInstance != nil && !reflect.DeepEqual(oldInstance, newInstance):
		slog.Info("discovered logstash instance from service annotation",
			"instance", resourceName,
			"url", newInstance.Host)
		h.collectorMgr.AddInstance(resourceName, newInstance)
	}
}

// onServiceDelete is called when a service is deleted
func (h *ServiceResourceHandler) onServiceDelete(obj interface{}) {
	var service *corev1.Service
	switch t := obj.(type) {
	case *corev1.Service:
		service = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		service, ok = t.Obj.(*corev1.Service)
		if !ok {
			slog.Warn("unexpected type in service delete event handler")
			return
		}
	default:
		slog.Warn("unexpected type in service delete event handler")
		return
	}

	h.removeService(service)
}

// serviceInstance returns the Logstash instance to scrape for a service, or
// nil if the service is not annotated
func (h *ServiceResourceHandler) serviceInstance(service *corev1.Service) (string, *config.LogstashInstance) {
	instanceName := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	return h.extractLogstashInfo(service.Annotations, instanceName)
}

// processService adds a service to monitoring if it should be scraped
func (h *ServiceResourceHandler) processService(service *corev1.Service) {
	resourceName, instance := h.serviceInstance(service)
	if instance == nil {
		return
	}

	slog.Info("discovered logstash instance from service annotation",
		"instance", resourceName,
		"url", instance.Host)

	// Add the instance to the collector manager
	h.collectorMgr.AddInstance(resourceName, instance)
}

// removeService removes a service from monitoring
func (h *ServiceResourceHandler) removeService(service *corev1.Service) {
	instanceName := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	slog.Info("removing logstash instance", "instance", instanceName)

	// Remove the instance from the collector manager
	h.collectorMgr.RemoveInstance(instanceName)
}
