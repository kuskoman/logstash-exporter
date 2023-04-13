package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	kubeMeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	// If provided, use Kubernetes' API to get service endpoints for each Logstash replica.
	LogstashNamespace = func() string { return os.Getenv("LOGSTASH_KUBERNETES_NAMESPACE") }
	LogstashService   = func() string { return os.Getenv("LOGSTASH_KUBERNETES_SERVICE") }
	LogstashApiPort   = getEnvWithDefault("LOGSTASH_API_PORT", "9600")
)

type ServiceEndpoint struct {
	Service  string
	PortName string
	Labels   map[string]string
	Ip       string
	Port     uint16
}

func UseKubernetesEndpoints() bool {
	return LogstashNamespace() != "" && LogstashService() != ""
}

// Retrieves the set of Logstash Internal API endpoints
// for all replicas in the same cluster, matching the ENV filters.
func GetKubernetesLogstashApiEndpoints() ([]ServiceEndpoint, error) {
	kubeApiClient, err := initKubernetesClient(); 
	if err != nil {
		return nil, err
	}
	endpoints, err := fetchKubernetesServiceMetadata(kubeApiClient, LogstashNamespace(), LogstashService(), LogstashApiPort)
	if err != nil {
		return nil, err
	}
	for _, ep := range endpoints {
		log.Printf("Found Logstash Replica at: %s:%d", ep.Ip, ep.Port)
	}
	return endpoints, nil
}

// Mockable K8s API setup goes here
func initKubernetesClient() (*kubernetes.Clientset, error) {
	// Assumes Logstash runs in the same K8s cluster
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("Unable to init Kubernetes Client. Is process running in Kubernetes? Error: %w", err)
	}

	kubeApiClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Kubernetes client config failed. Error: %w", err)
	}
	return kubeApiClient, nil
}

func fetchKubernetesServiceMetadata(clientset kubernetes.Interface, namespace string, service string, apiPort string) ([]ServiceEndpoint, error) {
	// Query Kubernetes API for Logstash's Service endpoints
	ctx := context.Background()
	endpoints, err := clientset.CoreV1().Endpoints(namespace).Get(ctx, service, kubeMeta.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Unable to get Logstash service metadata. Error: %w", err)
	}

	// Gather all replicas' endpoints matching the Logstash API Port
	var endpointsList []ServiceEndpoint
	logstashApiPort, err := strconv.ParseInt(apiPort, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("Logstash API port (%s) is invalid: %w", apiPort, err)
	}
	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			for _, port := range subset.Ports {
				if int32(logstashApiPort) != port.Port {
					continue
				}
				endpoint := ServiceEndpoint{
					Service:  endpoints.Name,
					Labels:   endpoints.Labels,
					PortName: port.Name,
					Ip:       address.IP,
					Port:     uint16(port.Port),
				}
				endpointsList = append(endpointsList, endpoint)
			}
		}
	}

	return endpointsList, nil
}
