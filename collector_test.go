package main

import (
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewCollector(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewCollectorWithRegistry(registry)
	if collector == nil {
		t.Fatal("expected non-nil collector")
	}
}

func TestCollectorUpdate(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewCollectorWithRegistry(registry)

	proxy := ProxyConfig{
		Name:    "test-proxy",
		Type:    "http",
		Address: "127.0.0.1:8080",
	}

	result := CheckResult{
		Success:      true,
		StatusCode:   200,
		ResponseTime: 100 * time.Millisecond,
	}

	// This should not panic
	collector.Update(proxy, result)
}

func TestCollectorUpdateWithError(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewCollectorWithRegistry(registry)

	proxy := ProxyConfig{
		Name:    "error-proxy",
		Type:    "socks5",
		Address: "127.0.0.1:1080",
	}

	result := CheckResult{
		Success:      false,
		StatusCode:   0,
		ResponseTime: 50 * time.Millisecond,
		Error:        "connection refused",
		ErrorType:    "connection",
	}

	// This should not panic
	collector.Update(proxy, result)
}

func TestCollectorConcurrentUpdate(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewCollectorWithRegistry(registry)

	// Simulate concurrent updates from multiple proxies (the actual usage pattern)
	var wg sync.WaitGroup
	numGoroutines := 10
	numUpdates := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numUpdates; j++ {
				proxy := ProxyConfig{
					Name:    "concurrent-proxy",
					Type:    "http",
					Address: "127.0.0.1:8080",
				}

				result := CheckResult{
					Success:      j%2 == 0, // Alternate success/failure
					StatusCode:   200,
					ResponseTime: time.Duration(j) * time.Millisecond,
				}
				if !result.Success {
					result.ErrorType = "connection"
					result.Error = "simulated error"
				}

				collector.Update(proxy, result)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify the collector handled concurrent updates without panicking
	// and that metrics are still queryable
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	if len(families) == 0 {
		t.Error("expected some metric families after concurrent updates")
	}
}
