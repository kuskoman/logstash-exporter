package config

import (
	"context"
	"os"
	"reflect"
	"testing"

	kubeCore "k8s.io/api/core/v1"
	kubeMeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestUseKubernetesEndpoints(t *testing.T) {
	t.Run("should return false when no LOGSTASH_KUBERNETES ENVs are set", func(t *testing.T) {
		expected := false
		actual := UseKubernetesEndpoints()
		if actual != expected {
			t.Errorf("expected %t but got %t", expected, actual)
		}
	})
	t.Run("should return false when not all LOGSTASH_KUBERNETES ENVs are set", func(t *testing.T) {
		expected := false
		os.Setenv("LOGSTASH_KUBERNETES_NAMESPACE", "monitoring")
		actual := UseKubernetesEndpoints()
		if actual != expected {
			t.Errorf("expected %t but got %t", expected, actual)
		}
	})

	t.Run("should return true when all LOGSTASH_KUBERNETES ENVs are set", func(t *testing.T) {
		expected := true
		os.Setenv("LOGSTASH_KUBERNETES_NAMESPACE", "monitoring")
		os.Setenv("LOGSTASH_KUBERNETES_SERVICE", "logstash")
		actual := UseKubernetesEndpoints()
		if actual != expected {
			t.Errorf("expected %t but got %t", expected, actual)
		}
	})
}
 
func TestGetServiceEndpoints(t *testing.T) {

	// These would be passed by ENVs (via values.yaml)
	serviceName := "mock-logstash"
	namespace := "monitoring"
	logstashApiPort := "9600"

	// create dummy endpoints
	validEndpoints := &kubeCore.Endpoints{
		ObjectMeta: kubeMeta.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Subsets: []kubeCore.EndpointSubset{
			{
				Addresses: []kubeCore.EndpointAddress{
					{IP: "10.0.0.1"},
					{IP: "10.0.0.2"},
				},
				Ports: []kubeCore.EndpointPort{
					{Name: "logstash-pipeline0", Port: 8080},
					{Name: "logstash-api", Port: 9600},
					{Name: "logstash-pipeline1", Port: 8081},
				},
			},
		},
	}
	irrelevantEndpoints := &kubeCore.Endpoints{
		ObjectMeta: kubeMeta.ObjectMeta{
			Name:      "some other service",
			Namespace: namespace,
		},
		Subsets: []kubeCore.EndpointSubset{
			{
				Addresses: []kubeCore.EndpointAddress{
					{IP: "10.0.6.5"},
					{IP: "10.0.6.6"},
				},
				Ports: []kubeCore.EndpointPort{
					{Name: "logstash-pipeline0", Port: 8080},
					{Name: "logstash-api", Port: 9600},
					{Name: "logstash-pipeline1", Port: 8081},
				},
			},
		},
	}

	
	t.Run("should return the mock-logstash endpoints matching the namespace, service, and port", func(t *testing.T) {
		// register the dummy endpoints to the fake clientset
		clientset := fake.NewSimpleClientset()
		clientset.CoreV1().Endpoints(namespace).Create(context.Background(), validEndpoints, kubeMeta.CreateOptions{})
		
		expectedEndpoints := []ServiceEndpoint{
			{Service: "mock-logstash", PortName: "logstash-api", Ip: "10.0.0.1", Port: 9600},
			{Service: "mock-logstash", PortName: "logstash-api", Ip: "10.0.0.2", Port: 9600},
		}

		// Retrieve the fake endpoints from the mocked client, using the code under test
		actualEndpoints, err := fetchKubernetesServiceMetadata(clientset, namespace, serviceName, logstashApiPort)
		if err != nil {
			t.Error(err)
		}

		// check that the returned array is equal to the expected array
		if !reflect.DeepEqual(actualEndpoints, expectedEndpoints) {
			t.Errorf("Expected endpoints array: %v, but got: %v", expectedEndpoints, actualEndpoints)
		}
	})

	t.Run("should return the no irrelevant service endpoints", func(t *testing.T) {

		// register the dummy endpoints to the fake clientset
		clientset := fake.NewSimpleClientset()
		clientset.CoreV1().Endpoints(namespace).Create(context.Background(), irrelevantEndpoints, kubeMeta.CreateOptions{})

		// Retrieve the fake endpoints from the mocked client, using the code under test
		expectedErr := "Unable to get Logstash service metadata. Error: endpoints \"mock-logstash\" not found"
		_, err := fetchKubernetesServiceMetadata(clientset, namespace, serviceName, logstashApiPort)
		if err == nil || err.Error() != expectedErr {
			t.Errorf("Expected to fail with error %v but got %v", expectedErr, err)
		}
	})
}
