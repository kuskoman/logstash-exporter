//go:build k8se2e

// Package k8s contains end-to-end tests of the Kubernetes controller. They run
// against a real cluster with the controller installed by the chart, see the
// "Kubernetes e2e" workflow and README.md. They are excluded from regular test
// runs by the k8se2e build tag.
package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// Namespaces the controller watches, set by the workflow
	watchedNamespace      = "e2e-a"
	otherWatchedNamespace = "e2e-b"
	// Namespace the controller does not watch
	unwatchedNamespace = "e2e-c"

	urlAnnotation      = "logstash-exporter.io/url"
	usernameAnnotation = "logstash-exporter.io/username"
	passwordAnnotation = "logstash-exporter.io/password"

	waitTimeout = 3 * time.Minute
)

var (
	client           kubernetes.Interface
	controllerNS     = envOrDefault("E2E_CONTROLLER_NAMESPACE", "default")
	controllerSvc    = envOrDefault("E2E_CONTROLLER_SERVICE", "controller-logstash-exporter")
	logstashImage    = envOrDefault("E2E_LOGSTASH_IMAGE", "docker.elastic.co/logstash/logstash:9.5.5")
	mockImage        = envOrDefault("E2E_MOCK_IMAGE", "busybox:1.37")
	podURL           = map[string]string{urlAnnotation: "http://$(pod_ip):9600"}
	mockCredentials  = map[string]string{usernameAnnotation: "monitoring", passwordAnnotation: "secret"}
	podSelectorLabel = "e2e-app"
)

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func TestMain(m *testing.M) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot load kubeconfig %s: %v\n", kubeconfig, err)
		os.Exit(1)
	}
	client, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot create Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// TestController runs the scenarios in order against one controller. They
// share the cluster, so they are not parallel.
func TestController(t *testing.T) {
	ctx := context.Background()
	for _, namespace := range []string{watchedNamespace, otherWatchedNamespace, unwatchedNamespace} {
		createMockConfig(t, ctx, namespace)
	}

	t.Run("discovers_pods_started_after_the_controller", func(t *testing.T) {
		createMockDeployment(t, ctx, watchedNamespace, "mock", 2, podURL, false)
		waitForInstances(t, ctx, map[string]float64{
			"e2e-a/mock-*": 1,
		}, 2)
	})

	t.Run("follows_scaling_up_and_down", func(t *testing.T) {
		scaleDeployment(t, ctx, watchedNamespace, "mock", 3)
		waitForInstances(t, ctx, map[string]float64{"e2e-a/mock-*": 1}, 3)

		scaleDeployment(t, ctx, watchedNamespace, "mock", 1)
		waitForInstances(t, ctx, map[string]float64{"e2e-a/mock-*": 1}, 1)
	})

	t.Run("replaces_a_deleted_pod_with_its_successor", func(t *testing.T) {
		before := podNames(t, ctx, watchedNamespace, "mock")
		if err := client.CoreV1().Pods(watchedNamespace).Delete(ctx, before[0], metav1.DeleteOptions{}); err != nil {
			t.Fatal(err)
		}
		instances := waitForInstances(t, ctx, map[string]float64{"e2e-a/mock-*": 1}, 1)
		if _, ok := instances["e2e-a/"+before[0]]; ok {
			t.Errorf("expected the deleted pod %s to be removed", before[0])
		}
	})

	t.Run("ignores_pods_without_the_annotation", func(t *testing.T) {
		createMockDeployment(t, ctx, watchedNamespace, "plain", 1, nil, false)
		waitForPodsReady(t, ctx, watchedNamespace, "plain", 1)
		instances := waitForInstances(t, ctx, map[string]float64{"e2e-a/mock-*": 1}, 1)
		assertNoInstance(t, instances, "e2e-a/plain-")
	})

	t.Run("ignores_namespaces_that_are_not_watched", func(t *testing.T) {
		createMockDeployment(t, ctx, unwatchedNamespace, "mock", 1, podURL, false)
		waitForPodsReady(t, ctx, unwatchedNamespace, "mock", 1)
		instances := waitForInstances(t, ctx, map[string]float64{"e2e-a/mock-*": 1}, 1)
		assertNoInstance(t, instances, "e2e-c/")
	})

	t.Run("uses_the_credentials_annotations", func(t *testing.T) {
		annotations := map[string]string{}
		for key, value := range podURL {
			annotations[key] = value
		}
		for key, value := range mockCredentials {
			annotations[key] = value
		}
		createMockDeployment(t, ctx, otherWatchedNamespace, "secured", 1, annotations, true)
		waitForInstances(t, ctx, map[string]float64{"e2e-a/mock-*": 1, "e2e-b/secured-*": 1}, 2)
	})

	t.Run("reports_an_instance_with_wrong_credentials_as_down", func(t *testing.T) {
		annotations := map[string]string{urlAnnotation: "http://$(pod_ip):9600", usernameAnnotation: "monitoring", passwordAnnotation: "wrong"}
		createMockDeployment(t, ctx, otherWatchedNamespace, "rejected", 1, annotations, true)
		waitForInstances(t, ctx, map[string]float64{"e2e-a/mock-*": 1, "e2e-b/secured-*": 1, "e2e-b/rejected-*": 0}, 3)
	})

	t.Run("stops_scraping_a_pod_when_the_annotation_is_removed", func(t *testing.T) {
		patchPodTemplateAnnotations(t, ctx, otherWatchedNamespace, "rejected", `{"logstash-exporter.io/url":null}`)
		waitForInstances(t, ctx, map[string]float64{"e2e-a/mock-*": 1, "e2e-b/secured-*": 1}, 2)
	})

	t.Run("discovers_annotated_services", func(t *testing.T) {
		createService(t, ctx, watchedNamespace, "mock", map[string]string{urlAnnotation: "http://mock.e2e-a.svc:9600"})
		waitForInstances(t, ctx, map[string]float64{"e2e-a/mock-*": 1, "e2e-b/secured-*": 1, "e2e-a/mock": 1}, 3)
	})

	t.Run("scrapes_a_real_logstash", func(t *testing.T) {
		createLogstashDeployment(t, ctx, otherWatchedNamespace)
		waitForInstances(t, ctx, map[string]float64{
			"e2e-a/mock-*": 1, "e2e-b/secured-*": 1, "e2e-a/mock": 1, "e2e-b/logstash-*": 1,
		}, 4)

		metrics := scrapeController(t, ctx)
		if !hasSeries(metrics["logstash_stats_pipeline_up"], "instance_name", "e2e-b/logstash-") {
			t.Error("expected pipeline metrics from the real Logstash")
		}
	})
}

// createMockConfig creates the ConfigMap the mock Logstash serves: the node
// info and node stats fixtures, and an httpd configuration for basic auth
func createMockConfig(t *testing.T, ctx context.Context, namespace string) {
	t.Helper()

	if _, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}

	data := map[string]string{"httpd.conf": "/:monitoring:secret\n"}
	for key, file := range map[string]string{"node_info": "node_info.json", "node_stats": "node_stats.json"} {
		content, err := os.ReadFile(filepath.Join("..", "..", "fixtures", file))
		if err != nil {
			t.Fatal(err)
		}
		data[key] = string(content)
	}

	_, err := client.CoreV1().ConfigMaps(namespace).Create(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "logstash-mock"},
		Data:       data,
	}, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatal(err)
	}
}

// createMockDeployment deploys pods serving the Logstash API fixtures with
// busybox httpd, optionally behind basic authentication
func createMockDeployment(t *testing.T, ctx context.Context, namespace, name string, replicas int32, annotations map[string]string, auth bool) {
	t.Helper()

	command := []string{"httpd", "-f", "-p", "9600", "-h", "/www"}
	mounts := []corev1.VolumeMount{{Name: "fixtures", MountPath: "/www"}}
	if auth {
		command = append(command, "-c", "/etc/httpd/httpd.conf")
		mounts = append(mounts, corev1.VolumeMount{Name: "httpd", MountPath: "/etc/httpd"})
	}

	pod := corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:           "logstash-mock",
			Image:          mockImage,
			Command:        command,
			Ports:          []corev1.ContainerPort{{Name: "api", ContainerPort: 9600}},
			VolumeMounts:   mounts,
			ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(9600)}}},
		}},
		Volumes: []corev1.Volume{
			{Name: "fixtures", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "logstash-mock"},
				Items: []corev1.KeyToPath{
					{Key: "node_info", Path: "index.html"},
					{Key: "node_stats", Path: "_node/stats"},
				},
			}}},
			{Name: "httpd", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "logstash-mock"},
				Items:                []corev1.KeyToPath{{Key: "httpd.conf", Path: "httpd.conf"}},
			}}},
		},
	}
	createDeployment(t, ctx, namespace, name, replicas, annotations, pod)
}

// createLogstashDeployment deploys a real Logstash with its default pipeline
func createLogstashDeployment(t *testing.T, ctx context.Context, namespace string) {
	t.Helper()

	createDeployment(t, ctx, namespace, "logstash", 1, podURL, corev1.PodSpec{
		Containers: []corev1.Container{{
			Name:  "logstash",
			Image: logstashImage,
			Env: []corev1.EnvVar{
				{Name: "API_HTTP_HOST", Value: "0.0.0.0"},
				{Name: "LS_JAVA_OPTS", Value: "-Xms256m -Xmx256m"},
			},
			Ports: []corev1.ContainerPort{{Name: "api", ContainerPort: 9600}},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler:        corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/", Port: intstr.FromInt32(9600)}},
				InitialDelaySeconds: 20,
				PeriodSeconds:       5,
			},
		}},
	})
}

func createDeployment(t *testing.T, ctx context.Context, namespace, name string, replicas int32, annotations map[string]string, pod corev1.PodSpec) {
	t.Helper()

	labels := map[string]string{podSelectorLabel: name}
	_, err := client.AppsV1().Deployments(namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels, Annotations: annotations},
				Spec:       pod,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func createService(t *testing.T, ctx context.Context, namespace, name string, annotations map[string]string) {
	t.Helper()

	_, err := client.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Annotations: annotations},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{podSelectorLabel: name},
			Ports:    []corev1.ServicePort{{Name: "api", Port: 9600, TargetPort: intstr.FromInt32(9600)}},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
}

func scaleDeployment(t *testing.T, ctx context.Context, namespace, name string, replicas int32) {
	t.Helper()

	patch := fmt.Sprintf(`{"spec":{"replicas":%d}}`, replicas)
	if _, err := client.AppsV1().Deployments(namespace).Patch(ctx, name, types.MergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		t.Fatal(err)
	}
}

func patchPodTemplateAnnotations(t *testing.T, ctx context.Context, namespace, name, annotations string) {
	t.Helper()

	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":%s}}}}`, annotations)
	if _, err := client.AppsV1().Deployments(namespace).Patch(ctx, name, types.MergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		t.Fatal(err)
	}
}

func podNames(t *testing.T, ctx context.Context, namespace, app string) []string {
	t.Helper()

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: podSelectorLabel + "=" + app})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp == nil {
			names = append(names, pod.Name)
		}
	}
	sort.Strings(names)
	return names
}

func waitForPodsReady(t *testing.T, ctx context.Context, namespace, app string, count int) {
	t.Helper()

	eventually(t, func() error {
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: podSelectorLabel + "=" + app})
		if err != nil {
			return err
		}
		ready := 0
		for _, pod := range pods.Items {
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
					ready++
				}
			}
		}
		if ready != count {
			return fmt.Errorf("%d of %d pods of %s/%s ready", ready, count, namespace, app)
		}
		return nil
	})
}

// scrapeController fetches the controller's metrics through the API server
func scrapeController(t *testing.T, ctx context.Context) map[string]*dto.MetricFamily {
	t.Helper()

	families, err := tryScrapeController(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return families
}

func tryScrapeController(ctx context.Context) (map[string]*dto.MetricFamily, error) {
	body, err := client.CoreV1().Services(controllerNS).ProxyGet("http", controllerSvc, "9198", "/metrics", nil).DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape the controller: %w", err)
	}
	parser := expfmt.NewTextParser(model.UTF8Validation)
	families, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}
	return families, nil
}

// instancesUp returns logstash_info_up by instance name
func instancesUp(families map[string]*dto.MetricFamily) map[string]float64 {
	up := map[string]float64{}
	family := families["logstash_info_up"]
	if family == nil {
		return up
	}
	for _, metric := range family.GetMetric() {
		for _, label := range metric.GetLabel() {
			if label.GetName() == "instance_name" {
				up[label.GetValue()] = metric.GetGauge().GetValue()
			}
		}
	}
	return up
}

// waitForInstances waits until the controller scrapes exactly count
// instances, each matching one of the expected patterns with the given
// logstash_info_up value. A pattern ending in * matches any suffix, which is
// needed for generated pod names.
func waitForInstances(t *testing.T, ctx context.Context, expected map[string]float64, count int) map[string]float64 {
	t.Helper()

	var instances map[string]float64
	eventually(t, func() error {
		families, err := tryScrapeController(ctx)
		if err != nil {
			return err
		}
		instances = instancesUp(families)
		if len(instances) != count {
			return fmt.Errorf("expected %d instances, got %v", count, instances)
		}
		for name, up := range instances {
			pattern, ok := matchPattern(expected, name)
			if !ok {
				return fmt.Errorf("unexpected instance %s in %v", name, instances)
			}
			if up != expected[pattern] {
				return fmt.Errorf("expected %s to have logstash_info_up %v, got %v", name, expected[pattern], up)
			}
		}
		for pattern := range expected {
			if !anyMatches(pattern, instances) {
				return fmt.Errorf("no instance matches %s in %v", pattern, instances)
			}
		}
		return nil
	})
	return instances
}

func matchPattern(expected map[string]float64, name string) (string, bool) {
	for pattern := range expected {
		if matches(pattern, name) {
			return pattern, true
		}
	}
	return "", false
}

func anyMatches(pattern string, instances map[string]float64) bool {
	for name := range instances {
		if matches(pattern, name) {
			return true
		}
	}
	return false
}

func matches(pattern, name string) bool {
	if prefix, ok := strings.CutSuffix(pattern, "*"); ok {
		return strings.HasPrefix(name, prefix)
	}
	return pattern == name
}

func assertNoInstance(t *testing.T, instances map[string]float64, prefix string) {
	t.Helper()

	for name := range instances {
		if strings.HasPrefix(name, prefix) {
			t.Errorf("did not expect instance %s to be scraped", name)
		}
	}
}

func hasSeries(family *dto.MetricFamily, label, prefix string) bool {
	if family == nil {
		return false
	}
	for _, metric := range family.GetMetric() {
		for _, pair := range metric.GetLabel() {
			if pair.GetName() == label && strings.HasPrefix(pair.GetValue(), prefix) {
				return true
			}
		}
	}
	return false
}

func eventually(t *testing.T, condition func() error) {
	t.Helper()

	deadline := time.Now().Add(waitTimeout)
	for {
		err := condition()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("condition not met within %s: %v", waitTimeout, err)
		}
		time.Sleep(2 * time.Second)
	}
}
