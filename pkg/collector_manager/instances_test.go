package collector_manager

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kuskoman/logstash-exporter/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

// newLogstashServer serves the node info and node stats fixtures. When
// credentials are given, requests without them get HTTP 401.
func newLogstashServer(t *testing.T, useTLS bool, username, password string) *httptest.Server {
	t.Helper()

	responses := map[string][]byte{}
	for path, fixture := range map[string]string{"/": "node_info.json", "/_node/stats": "node_stats.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", fixture))
		if err != nil {
			t.Fatalf("failed to read fixture: %v", err)
		}
		responses[path] = data
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if username != "" {
			user, pass, ok := r.BasicAuth()
			if !ok || user != username || pass != password {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		body, ok := responses[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})

	var server *httptest.Server
	if useTLS {
		server = httptest.NewTLSServer(handler)
	} else {
		server = httptest.NewServer(handler)
	}
	t.Cleanup(server.Close)
	return server
}

// upInstances collects the manager's metrics and returns logstash_info_up by
// instance name
func upInstances(t *testing.T, manager *CollectorManager) map[string]float64 {
	t.Helper()

	registry := prometheus.NewRegistry()
	registry.MustRegister(manager)
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	up := map[string]float64{}
	for _, family := range families {
		if family.GetName() != "logstash_info_up" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "instance_name" {
					up[label.GetValue()] = metric.GetGauge().GetValue()
				}
			}
		}
	}
	return up
}

func TestCollectorManager_Instances(t *testing.T) {
	t.Parallel()

	t.Run("adds_and_removes_instances", func(t *testing.T) {
		t.Parallel()

		first := newLogstashServer(t, false, "", "")
		second := newLogstashServer(t, false, "", "")
		manager := NewCollectorManager([]*config.LogstashInstance{{Host: first.URL, Name: "first"}}, 2*time.Second)

		manager.AddInstance("default/second", &config.LogstashInstance{Host: second.URL, Name: "default/second"})
		if up := upInstances(t, manager); up["first"] != 1 || up["default/second"] != 1 {
			t.Errorf("expected both instances to be up, got %v", up)
		}

		manager.RemoveInstance("default/second")
		if up := upInstances(t, manager); len(up) != 1 || up["first"] != 1 {
			t.Errorf("expected only the first instance, got %v", up)
		}
	})

	t.Run("replaces_an_instance_with_the_same_id", func(t *testing.T) {
		t.Parallel()

		server := newLogstashServer(t, false, "", "")
		manager := NewCollectorManager(nil, 2*time.Second)

		manager.AddInstance("default/logstash", &config.LogstashInstance{Host: "http://127.0.0.1:1", Name: "default/logstash"})
		manager.AddInstance("default/logstash", &config.LogstashInstance{Host: server.URL, Name: "default/logstash"})

		if up := upInstances(t, manager); len(up) != 1 || up["default/logstash"] != 1 {
			t.Errorf("expected the replaced instance to be up, got %v", up)
		}
	})

	t.Run("ignores_removing_an_unknown_instance", func(t *testing.T) {
		t.Parallel()

		server := newLogstashServer(t, false, "", "")
		manager := NewCollectorManager([]*config.LogstashInstance{{Host: server.URL, Name: "first"}}, 2*time.Second)

		manager.RemoveInstance("default/unknown")
		if up := upInstances(t, manager); up["first"] != 1 {
			t.Errorf("expected the first instance to stay, got %v", up)
		}
	})
}

func TestCollectorManager_InstanceClients(t *testing.T) {
	t.Parallel()

	passwordFile := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(passwordFile, []byte("secret\n"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		useTLS   bool
		username string
		instance func(url string) *config.LogstashInstance
		expected float64
	}{
		{
			name:     "sends_basic_auth_credentials",
			username: "monitoring",
			instance: func(url string) *config.LogstashInstance {
				return &config.LogstashInstance{Host: url, Name: "instance",
					BasicAuth: &config.ClientAuthConfig{Username: "monitoring", Password: "secret"}}
			},
			expected: 1,
		},
		{
			name:     "reads_the_password_from_a_file",
			username: "monitoring",
			instance: func(url string) *config.LogstashInstance {
				return &config.LogstashInstance{Host: url, Name: "instance",
					BasicAuth: &config.ClientAuthConfig{Username: "monitoring", PasswordFile: passwordFile}}
			},
			expected: 1,
		},
		{
			name:     "is_rejected_without_credentials",
			username: "monitoring",
			instance: func(url string) *config.LogstashInstance {
				return &config.LogstashInstance{Host: url, Name: "instance"}
			},
			expected: 0,
		},
		{
			name:   "skips_tls_verification_when_configured",
			useTLS: true,
			instance: func(url string) *config.LogstashInstance {
				return &config.LogstashInstance{Host: url, Name: "instance",
					TLSConfig: &config.TLSClientConfig{InsecureSkipVerify: true}}
			},
			expected: 1,
		},
		{
			name:   "verifies_tls_certificates_by_default",
			useTLS: true,
			instance: func(url string) *config.LogstashInstance {
				return &config.LogstashInstance{Host: url, Name: "instance"}
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newLogstashServer(t, tt.useTLS, tt.username, "secret")
			manager := NewCollectorManager([]*config.LogstashInstance{tt.instance(server.URL)}, 2*time.Second)

			if up := upInstances(t, manager)["instance"]; up != tt.expected {
				t.Errorf("expected logstash_info_up %v, got %v", tt.expected, up)
			}
		})
	}
}
