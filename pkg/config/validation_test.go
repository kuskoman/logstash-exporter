package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return path
}

func expectError(t *testing.T, err error, contains string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", contains)
	}
	if !strings.Contains(err.Error(), contains) {
		t.Errorf("expected an error containing %q, got %v", contains, err)
	}
}

func TestBasicAuthConfig_GetUsers(t *testing.T) {
	t.Parallel()

	t.Run("returns_the_configured_users", func(t *testing.T) {
		t.Parallel()

		users, err := (&BasicAuthConfig{Users: map[string]string{"prometheus": "secret"}}).GetUsers()
		if err != nil {
			t.Fatal(err)
		}
		if users["prometheus"] != "secret" || len(users) != 1 {
			t.Errorf("unexpected users %v", users)
		}
	})

	t.Run("reads_users_from_a_file", func(t *testing.T) {
		t.Parallel()

		path := writeTestFile(t, "# comment\n\nprometheus: secret\n  admin:pass:with:colons  \n")
		users, err := (&BasicAuthConfig{UsersFile: path}).GetUsers()
		if err != nil {
			t.Fatal(err)
		}
		if users["prometheus"] != "secret" || users["admin"] != "pass:with:colons" || len(users) != 2 {
			t.Errorf("unexpected users %v", users)
		}
	})

	t.Run("rejects_invalid_lines_in_the_users_file", func(t *testing.T) {
		t.Parallel()

		_, err := (&BasicAuthConfig{UsersFile: writeTestFile(t, "no-colon-here\n")}).GetUsers()
		expectError(t, err, "invalid line in users file")
	})

	t.Run("returns_an_error_for_a_missing_users_file", func(t *testing.T) {
		t.Parallel()

		_, err := (&BasicAuthConfig{UsersFile: filepath.Join(t.TempDir(), "missing")}).GetUsers()
		expectError(t, err, "failed to read users file")
	})

	t.Run("returns_an_error_without_users", func(t *testing.T) {
		t.Parallel()

		_, err := (&BasicAuthConfig{}).GetUsers()
		expectError(t, err, "no authentication configuration provided")
	})
}

func TestBasicAuthConfig_ValidateBasicAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config BasicAuthConfig
		err    string
	}{
		{name: "users", config: BasicAuthConfig{Users: map[string]string{"a": "b"}}},
		{name: "users_file", config: BasicAuthConfig{UsersFile: "/users"}},
		{name: "both", config: BasicAuthConfig{Users: map[string]string{"a": "b"}, UsersFile: "/users"}, err: "mutually exclusive"},
		{name: "neither", config: BasicAuthConfig{}, err: "no authentication configuration provided"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.ValidateBasicAuth()
			if tt.err == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				return
			}
			expectError(t, err, tt.err)
		})
	}
}

func TestServerConfig_ValidateServerTLS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config ServerConfig
		err    string
	}{
		{name: "no_tls", config: ServerConfig{}},
		{name: "cert_and_key", config: ServerConfig{TLSConfig: &TLSServerConfig{CertFile: "c", KeyFile: "k"}}},
		{name: "missing_cert", config: ServerConfig{TLSConfig: &TLSServerConfig{KeyFile: "k"}}, err: "cert_file must be specified"},
		{name: "missing_key", config: ServerConfig{TLSConfig: &TLSServerConfig{CertFile: "c"}}, err: "key_file must be specified"},
		{name: "invalid_basic_auth", config: ServerConfig{BasicAuth: &BasicAuthConfig{}}, err: "no authentication configuration provided"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.ValidateServerTLS()
			if tt.err == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				return
			}
			expectError(t, err, tt.err)
		})
	}
}

func TestLogstashInstance_ValidateClientTLS(t *testing.T) {
	t.Parallel()

	caFile := writeTestFile(t, "ca")

	tests := []struct {
		name     string
		instance LogstashInstance
		err      string
	}{
		{name: "no_tls_or_auth", instance: LogstashInstance{}},
		{name: "existing_ca_file", instance: LogstashInstance{TLSConfig: &TLSClientConfig{CAFile: caFile}}},
		{name: "missing_ca_file", instance: LogstashInstance{TLSConfig: &TLSClientConfig{CAFile: "/does/not/exist"}}, err: "does not exist"},
		{name: "valid_basic_auth", instance: LogstashInstance{BasicAuth: &ClientAuthConfig{Username: "u", Password: "p"}}},
		{name: "invalid_basic_auth", instance: LogstashInstance{BasicAuth: &ClientAuthConfig{Username: "u"}}, err: "either password or password_file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.instance.ValidateClientTLS()
			if tt.err == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				return
			}
			expectError(t, err, tt.err)
		})
	}
}

func TestClientAuthConfig_ValidateClientAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config ClientAuthConfig
		err    string
	}{
		{name: "password", config: ClientAuthConfig{Username: "u", Password: "p"}},
		{name: "password_file", config: ClientAuthConfig{Username: "u", PasswordFile: "/p"}},
		{name: "missing_username", config: ClientAuthConfig{Password: "p"}, err: "username must be specified"},
		{name: "missing_password", config: ClientAuthConfig{Username: "u"}, err: "either password or password_file"},
		{name: "both_passwords", config: ClientAuthConfig{Username: "u", Password: "p", PasswordFile: "/p"}, err: "mutually exclusive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.ValidateClientAuth()
			if tt.err == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				return
			}
			expectError(t, err, tt.err)
		})
	}
}

func TestClientAuthConfig_GetPassword(t *testing.T) {
	t.Parallel()

	t.Run("returns_the_configured_password", func(t *testing.T) {
		t.Parallel()

		password, err := (&ClientAuthConfig{Password: "secret"}).GetPassword()
		if err != nil || password != "secret" {
			t.Errorf("expected secret, got %q (%v)", password, err)
		}
	})

	t.Run("reads_the_password_from_a_file", func(t *testing.T) {
		t.Parallel()

		password, err := (&ClientAuthConfig{PasswordFile: writeTestFile(t, "secret")}).GetPassword()
		if err != nil || password != "secret" {
			t.Errorf("expected secret, got %q (%v)", password, err)
		}
	})

	t.Run("returns_an_error_for_a_missing_password_file", func(t *testing.T) {
		t.Parallel()

		_, err := (&ClientAuthConfig{PasswordFile: filepath.Join(t.TempDir(), "missing")}).GetPassword()
		expectError(t, err, "failed to read password file")
	})

	t.Run("returns_an_error_without_a_password", func(t *testing.T) {
		t.Parallel()

		_, err := (&ClientAuthConfig{}).GetPassword()
		expectError(t, err, "neither password nor password_file specified")
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("accepts_a_valid_config", func(t *testing.T) {
		t.Parallel()

		config := &Config{Logstash: LogstashConfig{Instances: []*LogstashInstance{{Host: "http://logstash:9600"}}}}
		if err := config.Validate(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("rejects_invalid_server_tls", func(t *testing.T) {
		t.Parallel()

		config := &Config{Server: ServerConfig{TLSConfig: &TLSServerConfig{}}}
		expectError(t, config.Validate(), "invalid server TLS configuration")
	})

	t.Run("names_the_invalid_instance", func(t *testing.T) {
		t.Parallel()

		config := &Config{Logstash: LogstashConfig{Instances: []*LogstashInstance{
			{Host: "http://logstash:9600"},
			{Host: "http://logstash2:9600", BasicAuth: &ClientAuthConfig{Username: "u"}},
		}}}
		expectError(t, config.Validate(), "invalid Logstash instance 1")
	})
}

func TestHandleLegacyServersProperty(t *testing.T) {
	t.Parallel()

	config := &Config{Logstash: LogstashConfig{
		Instances:     []*LogstashInstance{{Host: "http://new:9600"}},
		LegacyServers: []*LogstashInstance{{Host: "http://old:9600"}},
	}}
	config.handleLegacyServersProperty()

	if len(config.Logstash.Instances) != 2 || config.Logstash.Instances[1].Host != "http://old:9600" {
		t.Errorf("expected the legacy servers to be added to the instances, got %v", config.Logstash.Instances)
	}
}
