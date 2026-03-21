package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestCheckProxyHTTP(t *testing.T) {
	// Create a target server that returns 200
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer targetServer.Close()

	// Create a proxy server
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			// For HTTPS, just respond OK (simplified)
			w.WriteHeader(http.StatusOK)
			return
		}
		// Forward the request
		resp, err := http.Get(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
	}))
	defer proxyServer.Close()

	proxy := ProxyConfig{
		Name:    "test-http",
		Type:    "http",
		Address: proxyServer.URL[7:], // remove "http://" prefix
		Timeout: 5 * time.Second,
	}

	checker := NewChecker(targetServer.URL, 5*time.Second)
	result := checker.Check(proxy)

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}
	if result.ResponseTime <= 0 {
		t.Errorf("expected positive response time, got %v", result.ResponseTime)
	}
}

func TestCheckProxyHTTPS(t *testing.T) {
	// Create a target server that returns 200
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer targetServer.Close()

	// Create a proxy server (HTTPS proxies use CONNECT method)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			w.WriteHeader(http.StatusOK)
			return
		}
		resp, err := http.Get(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
	}))
	defer proxyServer.Close()

	proxy := ProxyConfig{
		Name:    "test-https",
		Type:    "https",
		Address: proxyServer.URL[7:], // remove "http://" prefix
		Timeout: 5 * time.Second,
	}

	checker := NewChecker(targetServer.URL, 5*time.Second)
	result := checker.Check(proxy)

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if result.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}
}

func TestCheckProxyTimeout(t *testing.T) {
	// Create a slow server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	proxy := ProxyConfig{
		Name:    "test-timeout",
		Type:    "http",
		Address: "127.0.0.1:1", // Non-routable, will fail fast or timeout
		Timeout: 100 * time.Millisecond,
	}

	checker := NewChecker(slowServer.URL, 100*time.Millisecond)
	result := checker.Check(proxy)

	if result.Success {
		t.Error("expected failure due to timeout")
	}
	if result.ErrorType != "timeout" && result.ErrorType != "connection" {
		t.Errorf("expected timeout or connection error, got %s", result.ErrorType)
	}
}

func TestCheckProxyStatusCodeNon2xx(t *testing.T) {
	// Create a target server that returns 404
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer targetServer.Close()

	// Create a proxy server
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			w.WriteHeader(http.StatusOK)
			return
		}
		resp, err := http.Get(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
	}))
	defer proxyServer.Close()

	proxy := ProxyConfig{
		Name:    "test-status",
		Type:    "http",
		Address: proxyServer.URL[7:],
		Timeout: 5 * time.Second,
	}

	checker := NewChecker(targetServer.URL, 5*time.Second)
	result := checker.Check(proxy)

	if result.Success {
		t.Error("expected failure due to non-2xx status code")
	}
	if result.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", result.StatusCode)
	}
	if result.ErrorType != "status" {
		t.Errorf("expected error type 'status', got %s", result.ErrorType)
	}
}

func TestCheckProxyRegexMatch(t *testing.T) {
	// Create a target server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","ip":"1.2.3.4"}`))
	}))
	defer targetServer.Close()

	// Create a proxy server
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			w.WriteHeader(http.StatusOK)
			return
		}
		resp, err := http.Get(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.WriteHeader(resp.StatusCode)
		b := make([]byte, 1024)
		n, _ := resp.Body.Read(b)
		w.Write(b[:n])
	}))
	defer proxyServer.Close()

	// Test successful regex match
	t.Run("regex matches", func(t *testing.T) {
		proxy := ProxyConfig{
			Name:        "test-regex-match",
			Type:        "http",
			Address:     proxyServer.URL[7:],
			Timeout:     5 * time.Second,
			ExpectRegex: ".*\"ip\".*",
		}
		proxy.compiledRegex = regexp.MustCompile(proxy.ExpectRegex)

		checker := NewChecker(targetServer.URL, 5*time.Second)
		result := checker.Check(proxy)

		if !result.Success {
			t.Errorf("expected success with matching regex, got error: %s", result.Error)
		}
	})

	// Test failed regex match
	t.Run("regex does not match", func(t *testing.T) {
		proxy := ProxyConfig{
			Name:        "test-regex-nomatch",
			Type:        "http",
			Address:     proxyServer.URL[7:],
			Timeout:     5 * time.Second,
			ExpectRegex: ".*\"nonexistent\".*",
		}
		proxy.compiledRegex = regexp.MustCompile(proxy.ExpectRegex)

		checker := NewChecker(targetServer.URL, 5*time.Second)
		result := checker.Check(proxy)

		if result.Success {
			t.Error("expected failure with non-matching regex")
		}
		if result.ErrorType != "regex" {
			t.Errorf("expected error type 'regex', got %s", result.ErrorType)
		}
	})
}

func TestClassifyError(t *testing.T) {
	checker := NewChecker("http://example.com", 5*time.Second)

	tests := []struct {
		name     string
		errMsg   string
		wantType string
	}{
		{
			name:     "timeout error",
			errMsg:   "context deadline exceeded",
			wantType: "timeout",
		},
		{
			name:     "connection refused",
			errMsg:   "connection refused",
			wantType: "connection",
		},
		{
			name:     "no route to host",
			errMsg:   "no route to host",
			wantType: "connection",
		},
		{
			name:     "auth error",
			errMsg:   "407 Proxy Authentication Required",
			wantType: "auth",
		},
		{
			name:     "unauthorized",
			errMsg:   "authentication failed",
			wantType: "auth",
		},
		{
			name:     "generic error",
			errMsg:   "some unknown error",
			wantType: "connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock error and test classification
			result := checker.classifyError(&mockError{msg: tt.errMsg})
			if result != tt.wantType {
				t.Errorf("classifyError(%q) = %q, want %q", tt.errMsg, result, tt.wantType)
			}
		})
	}
}

func TestCheckProxySOCKS5ConnectionFailure(t *testing.T) {
	// Test SOCKS5 connection to non-existent proxy
	// This tests the SOCKS5 code path including the goroutine-based timeout handling
	// Note: Port 1 behavior varies by platform - may fail fast (connection refused) or timeout
	proxy := ProxyConfig{
		Name:    "test-socks5-fail",
		Type:    "socks5",
		Address: "127.0.0.1:1", // Non-routable port, behavior varies by OS
		Timeout: 500 * time.Millisecond,
	}

	checker := NewChecker("http://example.com", 500*time.Millisecond)
	result := checker.Check(proxy)

	if result.Success {
		t.Error("expected failure due to non-routable SOCKS5 proxy")
	}
	if result.ErrorType != "timeout" && result.ErrorType != "connection" {
		t.Errorf("expected timeout or connection error, got %s", result.ErrorType)
	}
	t.Logf("SOCKS5 connection failure resulted in error type: %s, error: %s", result.ErrorType, result.Error)
}

func TestCheckProxySOCKS5Timeout(t *testing.T) {
	// Test SOCKS5 timeout behavior
	// This specifically tests the goroutine-based context-aware timeout handling
	proxy := ProxyConfig{
		Name:    "test-socks5-timeout",
		Type:    "socks5",
		Address: "10.255.255.1:1080", // Non-routable IP that will hang
		Timeout: 200 * time.Millisecond,
	}

	checker := NewChecker("http://example.com", 200*time.Millisecond)
	result := checker.Check(proxy)

	if result.Success {
		t.Error("expected failure due to timeout")
	}
	// Should be timeout since we're using a non-routable address with short timeout
	if result.ErrorType != "timeout" && result.ErrorType != "connection" {
		t.Errorf("expected timeout or connection error, got %s", result.ErrorType)
	}
	t.Logf("SOCKS5 timeout test resulted in error type: %s, error: %s", result.ErrorType, result.Error)
	// Verify the timeout is respected (should complete within reasonable time)
	if result.ResponseTime > 1*time.Second {
		t.Errorf("timeout took too long: %v", result.ResponseTime)
	}
}

func TestCheckProxyUnsupportedType(t *testing.T) {
	proxy := ProxyConfig{
		Name:    "test-unsupported",
		Type:    "invalid",
		Address: "127.0.0.1:8080",
		Timeout: 1 * time.Second,
	}

	checker := NewChecker("http://example.com", 1*time.Second)
	result := checker.Check(proxy)

	if result.Success {
		t.Error("expected failure for unsupported proxy type")
	}
	if result.ErrorType != "connection" {
		t.Errorf("expected connection error for unsupported type, got %s", result.ErrorType)
	}
	if !strings.Contains(result.Error, "unsupported proxy type") {
		t.Errorf("expected unsupported proxy type error, got: %s", result.Error)
	}
}

// mockError is a simple error implementation for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
