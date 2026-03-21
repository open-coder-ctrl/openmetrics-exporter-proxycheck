package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	content := `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test-proxy"
    type: http
    address: "127.0.0.1:8080"
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Server.Listen != ":9099" {
		t.Errorf("expected listen :9099, got %s", cfg.Server.Listen)
	}
	if cfg.Server.Interval != 30*time.Second {
		t.Errorf("expected interval 30s, got %v", cfg.Server.Interval)
	}
	if cfg.Defaults.Target != "http://example.com" {
		t.Errorf("expected target http://example.com, got %s", cfg.Defaults.Target)
	}
	if len(cfg.Proxies) != 1 {
		t.Fatalf("expected 1 proxy, got %d", len(cfg.Proxies))
	}
	if cfg.Proxies[0].Name != "test-proxy" {
		t.Errorf("expected name test-proxy, got %s", cfg.Proxies[0].Name)
	}
}

func TestLoadConfigWithAuth(t *testing.T) {
	content := `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "auth-proxy"
    type: http
    address: "127.0.0.1:8080"
    username: "user"
    password: "pass"
    expect_regex: ".*ok.*"
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Proxies[0].Username != "user" {
		t.Errorf("expected username user, got %s", cfg.Proxies[0].Username)
	}
	if cfg.Proxies[0].Password != "pass" {
		t.Errorf("expected password pass, got %s", cfg.Proxies[0].Password)
	}
	if cfg.Proxies[0].ExpectRegex != ".*ok.*" {
		t.Errorf("expected regex .*ok.*, got %s", cfg.Proxies[0].ExpectRegex)
	}
}

func TestLoadAndValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{
			name: "valid config",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
`,
			wantError: false,
		},
		{
			name: "invalid config - missing listen",
			config: `
server:
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
`,
			wantError: true,
		},
		{
			name: "invalid config - empty proxies",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies: []
`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())
			if _, err := tmpfile.Write([]byte(tt.config)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadAndValidateConfig(tmpfile.Name())
			if (err != nil) != tt.wantError {
				t.Errorf("LoadAndValidateConfig() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError && cfg == nil {
				t.Error("expected non-nil config on success")
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		wantError bool
	}{
		{
			name: "valid config",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
`,
			wantError: false,
		},
		{
			name: "missing listen",
			config: `
server:
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
`,
			wantError: true,
		},
		{
			name: "missing target",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
`,
			wantError: true,
		},
		{
			name: "empty proxies",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies: []
`,
			wantError: true,
		},
		{
			name: "invalid proxy type",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: invalid
    address: "127.0.0.1:8080"
`,
			wantError: true,
		},
		{
			name: "missing proxy name",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - type: http
    address: "127.0.0.1:8080"
`,
			wantError: true,
		},
		{
			name: "missing proxy address",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
`,
			wantError: true,
		},
		{
			name: "invalid address format (missing port)",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1"
`,
			wantError: true,
		},
		{
			name: "invalid interval (zero)",
			config: `
server:
  listen: ":9099"
  interval: 0s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
`,
			wantError: true,
		},
		{
			name: "invalid interval (too small)",
			config: `
server:
  listen: ":9099"
  interval: 500ms
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
`,
			wantError: true,
		},
		{
			name: "invalid timeout (zero)",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 0s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
`,
			wantError: true,
		},
		{
			name: "duplicate proxy names",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "duplicate"
    type: http
    address: "127.0.0.1:8080"
  - name: "duplicate"
    type: socks5
    address: "127.0.0.1:1080"
`,
			wantError: true,
		},
		{
			name: "invalid regex pattern",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
    expect_regex: "[invalid"
`,
			wantError: true,
		},
		{
			name: "valid regex pattern",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
    expect_regex: ".*ok.*"
`,
			wantError: false,
		},
		{
			name: "invalid default target URL",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "not a valid url"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
`,
			wantError: true,
		},
		{
			name: "invalid per-proxy target URL",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
    target: ":::invalid"
`,
			wantError: true,
		},
		{
			name: "valid per-proxy target URL",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "127.0.0.1:8080"
    target: "https://api.example.com/health"
`,
			wantError: false,
		},
		{
			name: "address with only colon",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: ":"
`,
			wantError: true,
		},
		{
			name: "address with missing host",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: ":8080"
`,
			wantError: false,
		},
		{
			name: "address with missing port",
			config: `
server:
  listen: ":9099"
  interval: 30s
defaults:
  target: "http://example.com"
  timeout: 10s
proxies:
  - name: "test"
    type: http
    address: "localhost:"
`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpfile, err := os.CreateTemp("", "config-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())
			if _, err := tmpfile.Write([]byte(tt.config)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadConfig(tmpfile.Name())
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			err = cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestProxyConfigGetCompiledRegex(t *testing.T) {
	tests := []struct {
		name        string
		expectRegex string
		wantNil     bool
	}{
		{
			name:        "no regex",
			expectRegex: "",
			wantNil:     true,
		},
		{
			name:        "valid regex",
			expectRegex: ".*test.*",
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ProxyConfig{
				Name:        "test",
				Type:        "http",
				Address:     "127.0.0.1:8080",
				ExpectRegex: tt.expectRegex,
			}

			if err := p.Validate(); err != nil {
				t.Fatalf("Validate failed: %v", err)
			}

			regex := p.GetCompiledRegex()
			if (regex == nil) != tt.wantNil {
				t.Errorf("GetCompiledRegex() nil = %v, wantNil %v", regex == nil, tt.wantNil)
			}

			if regex != nil && !regex.MatchString("this is a test string") {
				t.Error("compiled regex should match test string")
			}
		})
	}
}
