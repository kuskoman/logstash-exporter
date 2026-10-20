package k8s_controller

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

const urlAnnotation = "logstash-exporter.io/url"

// fakeInstanceManager records the instances the handlers add and remove
type fakeInstanceManager struct {
	mu        sync.Mutex
	instances map[string]*config.LogstashInstance
	calls     int
}

func newFakeInstanceManager() *fakeInstanceManager {
	return &fakeInstanceManager{instances: map[string]*config.LogstashInstance{}}
}

func (m *fakeInstanceManager) AddInstance(id string, instance *config.LogstashInstance) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.instances[id] = instance
	m.calls++
}

func (m *fakeInstanceManager) RemoveInstance(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.instances, id)
	m.calls++
}

// Hosts returns the scraped URLs by instance id
func (m *fakeInstanceManager) Hosts() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	hosts := map[string]string{}
	for id, instance := range m.instances {
		hosts[id] = instance.Host
	}
	return hosts
}

func (m *fakeInstanceManager) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func assertHosts(t *testing.T, manager *fakeInstanceManager, expected map[string]string) {
	t.Helper()

	actual := manager.Hosts()
	if fmt.Sprint(sortedEntries(actual)) != fmt.Sprint(sortedEntries(expected)) {
		t.Errorf("expected instances %v, got %v", expected, actual)
	}
}

func sortedEntries(hosts map[string]string) []string {
	entries := make([]string, 0, len(hosts))
	for id, host := range hosts {
		entries = append(entries, id+"="+host)
	}
	sort.Strings(entries)
	return entries
}

func waitForHosts(t *testing.T, manager *fakeInstanceManager, expected map[string]string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if fmt.Sprint(sortedEntries(manager.Hosts())) == fmt.Sprint(sortedEntries(expected)) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	assertHosts(t, manager, expected)
}

func newPod(namespace, name string, phase corev1.PodPhase, ip string, annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, Annotations: annotations},
		Status:     corev1.PodStatus{Phase: phase, PodIP: ip},
	}
}

func newService(namespace, name string, annotations map[string]string) *corev1.Service {
	return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, Annotations: annotations}}
}

func newTestPodHandler(manager InstanceManager) *PodResourceHandler {
	return NewPodResourceHandler(fake.NewClientset(), manager, config.DefaultKubernetesConfig()).(*PodResourceHandler)
}

func newTestServiceHandler(manager InstanceManager) *ServiceResourceHandler {
	return NewServiceResourceHandler(fake.NewClientset(), manager, config.DefaultKubernetesConfig()).(*ServiceResourceHandler)
}

var podURL = map[string]string{urlAnnotation: "http://$(pod_ip):9600"}

func TestPodResourceHandler_Events(t *testing.T) {
	t.Parallel()

	t.Run("adds_a_running_annotated_pod_with_its_ip", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		newTestPodHandler(manager).onPodAdd(newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL))

		assertHosts(t, manager, map[string]string{"logging/logstash-0": "http://10.0.0.5:9600"})
	})

	t.Run("ignores_pods_without_the_url_annotation", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		newTestPodHandler(manager).onPodAdd(newPod("logging", "web-0", corev1.PodRunning, "10.0.0.6", nil))

		assertHosts(t, manager, map[string]string{})
	})

	t.Run("ignores_pods_that_are_not_running", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		newTestPodHandler(manager).onPodAdd(newPod("logging", "logstash-0", corev1.PodPending, "", podURL))

		assertHosts(t, manager, map[string]string{})
	})

	t.Run("adds_a_pod_when_it_starts_running", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestPodHandler(manager)
		pending := newPod("logging", "logstash-0", corev1.PodPending, "", podURL)
		running := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL)

		handler.onPodAdd(pending)
		handler.onPodUpdate(pending, running)

		assertHosts(t, manager, map[string]string{"logging/logstash-0": "http://10.0.0.5:9600"})
	})

	t.Run("removes_a_pod_when_it_stops_running", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestPodHandler(manager)
		running := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL)
		failed := newPod("logging", "logstash-0", corev1.PodFailed, "10.0.0.5", podURL)

		handler.onPodAdd(running)
		handler.onPodUpdate(running, failed)

		assertHosts(t, manager, map[string]string{})
	})

	t.Run("updates_the_url_when_the_pod_ip_changes", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestPodHandler(manager)
		before := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL)
		after := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.9", podURL)

		handler.onPodAdd(before)
		handler.onPodUpdate(before, after)

		assertHosts(t, manager, map[string]string{"logging/logstash-0": "http://10.0.0.9:9600"})
	})

	t.Run("updates_the_url_when_the_annotation_changes", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestPodHandler(manager)
		before := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL)
		after := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5",
			map[string]string{urlAnnotation: "http://$(pod_ip):9601"})

		handler.onPodAdd(before)
		handler.onPodUpdate(before, after)

		assertHosts(t, manager, map[string]string{"logging/logstash-0": "http://10.0.0.5:9601"})
	})

	t.Run("removes_a_pod_when_the_annotation_is_removed", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestPodHandler(manager)
		annotated := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL)
		plain := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", nil)

		handler.onPodAdd(annotated)
		handler.onPodUpdate(annotated, plain)

		assertHosts(t, manager, map[string]string{})
	})

	t.Run("ignores_resyncs_of_unchanged_pods", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestPodHandler(manager)
		pod := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL)

		handler.onPodAdd(pod)
		calls := manager.Calls()
		handler.onPodUpdate(pod, pod)

		if manager.Calls() != calls {
			t.Errorf("expected no changes for an unchanged pod, got %d new calls", manager.Calls()-calls)
		}
	})

	t.Run("removes_a_deleted_pod", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestPodHandler(manager)
		pod := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL)

		handler.onPodAdd(pod)
		handler.onPodDelete(pod)

		assertHosts(t, manager, map[string]string{})
	})

	t.Run("removes_a_pod_deleted_while_the_watch_was_disconnected", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestPodHandler(manager)
		pod := newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL)

		handler.onPodAdd(pod)
		handler.onPodDelete(cache.DeletedFinalStateUnknown{Key: "logging/logstash-0", Obj: pod})

		assertHosts(t, manager, map[string]string{})
	})
}

func TestServiceResourceHandler_Events(t *testing.T) {
	t.Parallel()

	serviceURL := map[string]string{urlAnnotation: "http://logstash-headless.logging:9600"}

	t.Run("adds_an_annotated_service", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		newTestServiceHandler(manager).onServiceAdd(newService("logging", "logstash", serviceURL))

		assertHosts(t, manager, map[string]string{"logging/logstash": "http://logstash-headless.logging:9600"})
	})

	t.Run("ignores_services_without_the_url_annotation", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		newTestServiceHandler(manager).onServiceAdd(newService("logging", "web", nil))

		assertHosts(t, manager, map[string]string{})
	})

	t.Run("removes_a_service_when_the_annotation_is_removed", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestServiceHandler(manager)
		annotated := newService("logging", "logstash", serviceURL)
		plain := newService("logging", "logstash", nil)

		handler.onServiceAdd(annotated)
		handler.onServiceUpdate(annotated, plain)

		assertHosts(t, manager, map[string]string{})
	})

	t.Run("removes_a_deleted_service", func(t *testing.T) {
		t.Parallel()

		manager := newFakeInstanceManager()
		handler := newTestServiceHandler(manager)
		service := newService("logging", "logstash", serviceURL)

		handler.onServiceAdd(service)
		handler.onServiceDelete(cache.DeletedFinalStateUnknown{Key: "logging/logstash", Obj: service})

		assertHosts(t, manager, map[string]string{})
	})
}

func TestPodResourceHandler_Start(t *testing.T) {
	t.Parallel()

	t.Run("discovers_pods_in_the_watched_namespaces", func(t *testing.T) {
		t.Parallel()

		client := fake.NewClientset(
			newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL),
			newPod("other", "logstash-0", corev1.PodRunning, "10.0.1.5", podURL),
		)
		manager := newFakeInstanceManager()
		handler := NewPodResourceHandler(client, manager, config.DefaultKubernetesConfig())
		if err := handler.Start(context.Background(), []string{"logging"}); err != nil {
			t.Fatal(err)
		}
		defer handler.Stop()

		waitForHosts(t, manager, map[string]string{"logging/logstash-0": "http://10.0.0.5:9600"})
	})

	t.Run("follows_a_pod_from_creation_to_deletion", func(t *testing.T) {
		t.Parallel()

		client := fake.NewClientset()
		manager := newFakeInstanceManager()
		handler := NewPodResourceHandler(client, manager, config.DefaultKubernetesConfig())
		if err := handler.Start(context.Background(), []string{metav1.NamespaceAll}); err != nil {
			t.Fatal(err)
		}
		defer handler.Stop()

		ctx := context.Background()
		pods := client.CoreV1().Pods("logging")

		// Wait until the informer watches, so the creation is not missed
		time.Sleep(100 * time.Millisecond)

		pod, err := pods.Create(ctx, newPod("logging", "logstash-0", corev1.PodPending, "", podURL), metav1.CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		pod.Status = corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.0.0.5"}
		if _, err := pods.UpdateStatus(ctx, pod, metav1.UpdateOptions{}); err != nil {
			t.Fatal(err)
		}
		waitForHosts(t, manager, map[string]string{"logging/logstash-0": "http://10.0.0.5:9600"})

		if err := pods.Delete(ctx, "logstash-0", metav1.DeleteOptions{}); err != nil {
			t.Fatal(err)
		}
		waitForHosts(t, manager, map[string]string{})
	})

	t.Run("does_nothing_when_pods_are_disabled", func(t *testing.T) {
		t.Parallel()

		kubeConfig := config.DefaultKubernetesConfig()
		kubeConfig.Resources.Pods.Enabled = false
		client := fake.NewClientset(newPod("logging", "logstash-0", corev1.PodRunning, "10.0.0.5", podURL))
		manager := newFakeInstanceManager()
		handler := NewPodResourceHandler(client, manager, kubeConfig)
		if err := handler.Start(context.Background(), []string{metav1.NamespaceAll}); err != nil {
			t.Fatal(err)
		}

		time.Sleep(200 * time.Millisecond)
		assertHosts(t, manager, map[string]string{})
	})
}

func TestServiceResourceHandler_Start(t *testing.T) {
	t.Parallel()

	kubeConfig := config.DefaultKubernetesConfig()
	kubeConfig.Resources.Services.Enabled = true
	client := fake.NewClientset(newService("logging", "logstash",
		map[string]string{urlAnnotation: "http://logstash-headless.logging:9600"}))
	manager := newFakeInstanceManager()
	handler := NewServiceResourceHandler(client, manager, kubeConfig)
	if err := handler.Start(context.Background(), []string{metav1.NamespaceAll}); err != nil {
		t.Fatal(err)
	}
	defer handler.Stop()

	waitForHosts(t, manager, map[string]string{"logging/logstash": "http://logstash-headless.logging:9600"})
}
