package config

import (
	"time"
)

// ResourceConfig represents the configuration for monitoring a specific type of Kubernetes resource
type ResourceConfig struct {
	// Enable monitoring this resource type
	Enabled bool `yaml:"enabled"`
	
	// AnnotationPrefix is the prefix used for annotations that the controller will watch
	AnnotationPrefix string `yaml:"annotationPrefix"`
}

// KubernetesConfig holds configuration for the Kubernetes controller
type KubernetesConfig struct {
	// Enable Kubernetes controller
	Enabled bool `yaml:"enabled"`

	// Namespaces to watch, empty for all namespaces
	Namespaces []string `yaml:"namespaces,omitempty"`

	// ResourceTypes defines which types of Kubernetes resources to monitor
	Resources struct {
		// Pods configuration
		Pods ResourceConfig `yaml:"pods"`
		
		// Services configuration
		Services ResourceConfig `yaml:"services"`
	} `yaml:"resources"`

	// ResyncPeriod is the period for resynchronizing the cache
	ResyncPeriod time.Duration `yaml:"resyncPeriod"`

	// ScrapeInterval is the interval at which logstash instances will be scraped
	ScrapeInterval time.Duration `yaml:"scrapeInterval"`

	// LogstashURLAnnotation is the annotation that contains the URL of the logstash instance
	LogstashURLAnnotation string `yaml:"logstashURLAnnotation"`

	// LogstashUsernameAnnotation is the annotation that contains the username for logstash authentication
	LogstashUsernameAnnotation string `yaml:"logstashUsernameAnnotation,omitempty"`

	// LogstashPasswordAnnotation is the annotation that contains the password for logstash authentication
	LogstashPasswordAnnotation string `yaml:"logstashPasswordAnnotation,omitempty"`

	// KubeConfig is the path to the kubeconfig file
	KubeConfig string `yaml:"kubeConfig,omitempty"`
}

// DefaultKubernetesConfig returns the default Kubernetes controller configuration
func DefaultKubernetesConfig() KubernetesConfig {
	config := KubernetesConfig{
		Enabled:                 false,
		ResyncPeriod:            10 * time.Minute,
		ScrapeInterval:          15 * time.Second,
		LogstashURLAnnotation:   "logstash-exporter.io/url",
		LogstashUsernameAnnotation: "logstash-exporter.io/username",
		LogstashPasswordAnnotation: "logstash-exporter.io/password",
	}
	
	// Default resource configurations
	config.Resources.Pods = ResourceConfig{
		Enabled:         true,
		AnnotationPrefix: "logstash-exporter.io",
	}
	
	config.Resources.Services = ResourceConfig{
		Enabled:         false,
		AnnotationPrefix: "logstash-exporter.io",
	}
	
	return config
}