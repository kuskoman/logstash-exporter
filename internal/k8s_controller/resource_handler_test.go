package k8s_controller

import (
	"testing"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

func TestExtractLogstashInfo(t *testing.T) {
	t.Parallel()

	handler := &BaseResourceHandler{config: config.DefaultKubernetesConfig()}

	t.Run("ignores_resources_without_the_url_annotation", func(t *testing.T) {
		t.Parallel()

		name, instance := handler.extractLogstashInfo(map[string]string{"other": "value"}, "default/logstash-0")
		if name != "" || instance != nil {
			t.Errorf("expected no instance, got %q %+v", name, instance)
		}
	})

	t.Run("creates_an_instance_from_the_url_annotation", func(t *testing.T) {
		t.Parallel()

		name, instance := handler.extractLogstashInfo(map[string]string{
			"logstash-exporter.io/url": "http://$(pod_ip):9600",
		}, "default/logstash-0")

		if name != "default/logstash-0" {
			t.Errorf("expected name default/logstash-0, got %q", name)
		}
		if instance.Host != "http://$(pod_ip):9600" || instance.Name != "default/logstash-0" {
			t.Errorf("unexpected instance %+v", instance)
		}
		if instance.BasicAuth != nil || instance.TLSConfig != nil {
			t.Errorf("expected no authentication or TLS, got %+v", instance)
		}
	})

	t.Run("uses_the_credentials_annotations", func(t *testing.T) {
		t.Parallel()

		_, instance := handler.extractLogstashInfo(map[string]string{
			"logstash-exporter.io/url":      "http://logstash:9600",
			"logstash-exporter.io/username": "monitoring",
			"logstash-exporter.io/password": "secret",
		}, "default/logstash-0")

		if instance.BasicAuth == nil {
			t.Fatal("expected basic authentication to be configured")
		}
		if instance.BasicAuth.Username != "monitoring" || instance.BasicAuth.Password != "secret" {
			t.Errorf("expected the annotated credentials, got %+v", instance.BasicAuth)
		}
	})

	t.Run("uses_tls_without_verification_for_https", func(t *testing.T) {
		t.Parallel()

		_, instance := handler.extractLogstashInfo(map[string]string{
			"logstash-exporter.io/url": "https://logstash:9600",
		}, "default/logstash-0")

		if instance.TLSConfig == nil || !instance.TLSConfig.InsecureSkipVerify {
			t.Errorf("expected TLS without certificate verification, got %+v", instance.TLSConfig)
		}
	})
}
