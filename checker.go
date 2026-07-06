package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// CheckResult represents the result of a proxy check
type CheckResult struct {
	Success      bool
	StatusCode   int
	ResponseTime time.Duration
	Error        string
	ErrorType    string
}

// Checker performs proxy health checks
type Checker struct {
	defaultTarget  string
	defaultTimeout time.Duration
	
	// clientPool caches HTTP clients to enable connection reuse
	clientMu sync.Mutex
	clientPool map[string]*http.Client
}

// NewChecker creates a new Checker
func NewChecker(defaultTarget string, defaultTimeout time.Duration) *Checker {
	return &Checker{
		defaultTarget:  defaultTarget,
		defaultTimeout: defaultTimeout,
		clientPool:     make(map[string]*http.Client),
	}
}

// Check performs a health check on the given proxy
func (c *Checker) Check(p ProxyConfig) CheckResult {
	target := c.defaultTarget
	if p.Target != "" {
		target = p.Target
	}

	timeout := c.defaultTimeout
	if p.Timeout > 0 {
		timeout = p.Timeout
	}

	client, err := c.getClient(p, timeout)
	if err != nil {
		return CheckResult{
			Success:   false,
			Error:     err.Error(),
			ErrorType: "connection",
		}
	}

	start := time.Now()
	resp, err := client.Get(target)
	elapsed := time.Since(start)

	if err != nil {
		return CheckResult{
			Success:      false,
			ResponseTime: elapsed,
			Error:        err.Error(),
			ErrorType:    c.classifyError(err),
		}
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CheckResult{
			Success:      false,
			StatusCode:   resp.StatusCode,
			ResponseTime: elapsed,
			Error:        fmt.Sprintf("non-2xx status code: %d", resp.StatusCode),
			ErrorType:    "status",
		}
	}

	// Check regex if specified
	if p.ExpectRegex != "" && p.GetCompiledRegex() != nil {
		// Read body with limit
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Max 1MB
		if err != nil {
			return CheckResult{
				Success:      false,
				StatusCode:   resp.StatusCode,
				ResponseTime: elapsed,
				Error:        fmt.Sprintf("failed to read response body: %v", err),
				ErrorType:    "connection",
			}
		}

		// Use pre-compiled regex from configuration validation
		if !p.GetCompiledRegex().Match(body) {
			return CheckResult{
				Success:      false,
				StatusCode:   resp.StatusCode,
				ResponseTime: elapsed,
				Error:        "response body did not match expected regex",
				ErrorType:    "regex",
			}
		}
	}

	return CheckResult{
		Success:      true,
		StatusCode:   resp.StatusCode,
		ResponseTime: elapsed,
	}
}

// getClient returns an HTTP client for the given proxy configuration.
// It reuses existing clients from the pool when possible to enable connection reuse.
func (c *Checker) getClient(p ProxyConfig, timeout time.Duration) (*http.Client, error) {
	key := fmt.Sprintf("%s:%s:%s:%s", p.Type, p.Address, p.Username, p.Password)
	
	c.clientMu.Lock()
	client, exists := c.clientPool[key]
	c.clientMu.Unlock()
	
	if exists {
		return client, nil
	}
	
	client, err := c.createClient(p, timeout)
	if err != nil {
		return nil, err
	}
	
	c.clientMu.Lock()
	// Double-check after acquiring lock
	if cached, exists := c.clientPool[key]; exists {
		c.clientMu.Unlock()
		return cached, nil
	}
	c.clientPool[key] = client
	c.clientMu.Unlock()
	
	return client, nil
}

// createClient creates an HTTP client configured for the proxy type.
// Clients are cached and reused to enable connection pooling.
func (c *Checker) createClient(p ProxyConfig, timeout time.Duration) (*http.Client, error) {
	switch p.Type {
	case "http", "https":
		// Both HTTP and HTTPS proxies use HTTP CONNECT tunneling.
		// The proxy URL uses http:// scheme; for HTTPS targets, Go automatically
		// uses CONNECT method to establish a tunnel through the proxy.
		proxyURL, err := url.Parse(fmt.Sprintf("http://%s", p.Address))
		if err != nil {
			return nil, err
		}
		if p.Username != "" {
			proxyURL.User = url.UserPassword(p.Username, p.Password)
		}
		return &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
				DialContext: (&net.Dialer{
					Timeout:   timeout,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				// Enable connection pooling
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: timeout,
		}, nil

	case "socks5":
		var auth *proxy.Auth
		if p.Username != "" {
			auth = &proxy.Auth{
				User:     p.Username,
				Password: p.Password,
			}
		}
		// Use a dialer with timeout instead of proxy.Direct to ensure
		// the SOCKS5 connection handshake respects the timeout setting
		forwardDialer := &net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}
		dialer, err := proxy.SOCKS5("tcp", p.Address, auth, forwardDialer)
		if err != nil {
			return nil, err
		}
		return &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					// Use a goroutine with context-aware timeout to ensure
					// the dial respects both the context deadline and timeout
					connCh := make(chan net.Conn, 1)
					errCh := make(chan error, 1)
					go func() {
						conn, err := dialer.Dial(network, addr)
						if err != nil {
							errCh <- err
							return
						}
						connCh <- conn
					}()

					select {
					case conn := <-connCh:
						return conn, nil
					case err := <-errCh:
						return nil, err
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				},
				// Enable connection pooling
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: timeout,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", p.Type)
	}
}

// classifyError determines the error type from an error
func (c *Checker) classifyError(err error) string {
	if err == nil {
		return ""
	}
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline exceeded"):
		return "timeout"
	case strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "network is unreachable"):
		return "connection"
	case strings.Contains(errStr, "authentication") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "407"):
		return "auth"
	default:
		return "connection"
	}
}
