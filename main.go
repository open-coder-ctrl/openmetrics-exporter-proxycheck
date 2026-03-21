// Proxycheck Exporter is a Prometheus exporter that monitors the health and
// performance of HTTP, HTTPS, and SOCKS5 proxy servers.
//
// The exporter periodically checks configured proxies by making requests through
// them to a target URL and exposes metrics about proxy availability, response
// times, and error rates at the /metrics endpoint.
//
// Usage:
//
//	proxycheck-exporter -config /path/to/config.yaml
//
// Endpoints:
//
//	/metrics - Prometheus metrics endpoint
//	/health  - Simple health check endpoint (returns "OK" if server is running)
//
// Configuration is loaded from a YAML file specified by the -config flag
// (default: config.yaml). See config.yaml for an example configuration.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
)

func main() {
	flag.Parse()

	// Load and validate configuration
	cfg, err := LoadAndValidateConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Configuration loaded: %d proxies configured", len(cfg.Proxies))

	// Initialize components
	checker := NewChecker(cfg.Defaults.Target, cfg.Defaults.Timeout)
	collector := NewCollector()

	// Start check loop
	stopCh := make(chan struct{})
	go runCheckLoop(cfg, checker, collector, stopCh)

	// Setup HTTP server
	http.Handle("/metrics", promhttp.Handler())
	// Health endpoint provides a simple liveness check.
	// It indicates the HTTP server is running and accepting requests.
	// It does NOT indicate the health of configured proxies.
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    cfg.Server.Listen,
		Handler: nil, // uses DefaultServeMux
	}

	log.Printf("Starting proxycheck-exporter on %s", cfg.Server.Listen)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutting down...")

	// Stop the check loop
	close(stopCh)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}

func runCheckLoop(cfg *Config, checker *Checker, collector *Collector, stopCh <-chan struct{}) {
	// Run initial check immediately
	runCheck(cfg.Proxies, checker, collector)

	ticker := time.NewTicker(cfg.Server.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			runCheck(cfg.Proxies, checker, collector)
		case <-stopCh:
			return
		}
	}
}

// runCheck performs health checks on all configured proxies concurrently.
// The function spawns a goroutine for each proxy check and uses a mutex to
// protect concurrent metric updates. All checks must complete (via wg.Wait)
// before returning, ensuring metric snapshots are atomic per check cycle -
// i.e., all metrics reflect the same check iteration rather than partial
// updates from different times.
func runCheck(proxies []ProxyConfig, checker *Checker, collector *Collector) {
	var wg sync.WaitGroup
	var mu sync.Mutex // Protect collector updates

	for _, proxy := range proxies {
		wg.Add(1)
		go func(p ProxyConfig) {
			defer wg.Done()
			result := checker.Check(p)

			mu.Lock()
			collector.Update(p, result)
			mu.Unlock()

			if result.Success {
				log.Printf("[%s] OK - %dms", p.Name, result.ResponseTime.Milliseconds())
			} else {
				log.Printf("[%s] FAILED - %s: %s", p.Name, result.ErrorType, result.Error)
			}
		}(proxy)
	}
	wg.Wait() // Wait for all checks to complete before next interval
}
