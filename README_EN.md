# Proxy Check Exporter

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

English | [简体中文](README.md)

A Prometheus exporter for monitoring HTTP, HTTPS, and SOCKS5 proxy server health and performance.

## Features

- Monitor multiple proxy servers concurrently
- Support for HTTP, HTTPS, and SOCKS5 proxy types
- Configurable check intervals and timeouts
- Response body regex validation
- Basic authentication support
- Prometheus-compatible metrics endpoint
- `/health` health check endpoint

## Quick Start

### Requirements

- Go 1.21 or higher

### Installation

Build with Makefile (recommended):

```bash
make build
# Output: .build/openmetric-proxycheck-exporter-<os>-<arch>
```

Or build directly:

```bash
go build -o proxycheck-exporter .
```

### Minimal Configuration

Create a `config.yaml` file:

```yaml
server:
  listen: ":9099"
  interval: 30s

defaults:
  target: "https://api.ipify.org"
  timeout: 10s

proxies:
  - name: "my-proxy"
    type: http
    address: "192.168.1.100:8080"
```

### Run

```bash
./proxycheck-exporter -config config.yaml
```

### Verify Installation

```bash
# Health check
curl http://localhost:9099/health
# Output: OK

# View metrics
curl http://localhost:9099/metrics | grep proxy_
```

## Configuration

### Full Configuration Example

```yaml
server:
  listen: ":9099"        # Listen address
  interval: 30s          # Check interval

defaults:
  target: "https://api.ipify.org"  # Default target URL
  timeout: 10s                      # Default timeout

proxies:
  - name: "http-proxy-1"
    type: http
    address: "192.168.1.100:8080"

  - name: "https-proxy-auth"
    type: https
    address: "proxy.example.com:443"
    username: "user"
    password: "pass"
    target: "https://api.ipify.org?format=json"
    expect_regex: ".*ip.*"

  - name: "socks5-proxy-1"
    type: socks5
    address: "127.0.0.1:1080"
    timeout: 5s
```

### Server Settings

| Field | Description | Default |
|-------|-------------|---------|
| `listen` | HTTP server listen address | None (required) |
| `interval` | Proxy health check interval | None (required) |

### Default Settings

| Field | Description | Default |
|-------|-------------|---------|
| `target` | Default target URL to check | None (required) |
| `timeout` | Default request timeout | None (required) |

### Proxy Configuration

| Field | Description | Required | Example |
|-------|-------------|----------|---------|
| `name` | Proxy name (used in metrics labels) | Yes | `"my-proxy"` |
| `type` | Proxy type: `http`, `https`, or `socks5` | Yes | `http` |
| `address` | Proxy address in `host:port` format | Yes | `"192.168.1.100:8080"` |
| `target` | Override default target URL | No | `"https://example.com"` |
| `timeout` | Override default timeout | No | `5s` |
| `username` | Authentication username | No | `"user"` |
| `password` | Authentication password | No | `"pass"` |
| `expect_regex` | Regex pattern to match in response body | No | `".*ok.*"` |

## Metrics

### Metric List

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `proxy_up` | Gauge | `name`, `type`, `address` | 1 if proxy is available, 0 otherwise |
| `proxy_response_seconds` | Gauge | `name`, `type` | Response time in seconds |
| `proxy_status_code` | Gauge | `name`, `type` | HTTP status code from target |
| `proxy_check_errors_total` | Counter | `name`, `type`, `error_type` | Total check errors by type |

### Metrics Output Example

```
# HELP proxy_up Whether the proxy is up (1) or down (0)
# TYPE proxy_up gauge
proxy_up{name="my-proxy",type="http",address="192.168.1.100:8080"} 1

# HELP proxy_response_seconds Response time of the proxy check in seconds
# TYPE proxy_response_seconds gauge
proxy_response_seconds{name="my-proxy",type="http"} 0.234

# HELP proxy_status_code HTTP status code returned by the target through the proxy
# TYPE proxy_status_code gauge
proxy_status_code{name="my-proxy",type="http"} 200

# HELP proxy_check_errors_total Total number of proxy check errors by type
# TYPE proxy_check_errors_total counter
proxy_check_errors_total{name="my-proxy",type="http",error_type="timeout"} 2
```

### Error Types

| Type | Description |
|------|-------------|
| `timeout` | Request timed out |
| `connection` | Connection failed |
| `status` | HTTP status code was not 2xx |
| `regex` | Response body didn't match expected regex |
| `auth` | Authentication failed |

## Prometheus Integration

### Scrape Configuration

Add to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'proxycheck'
    static_configs:
      - targets: ['localhost:9099']
```

### PromQL Query Examples

```promql
# All proxy availability status
proxy_up

# Average response time (grouped by proxy type)
avg by (type) (proxy_response_seconds)

# Error rate over the last 5 minutes
rate(proxy_check_errors_total[5m])

# Proxies with response time over 1 second
proxy_response_seconds > 1

# Count of available proxies
count(proxy_up == 1)
```

## Grafana Integration

### Recommended Panels

| Panel Name | PromQL Query | Visualization Type |
|------------|--------------|---------------------|
| Proxy Availability | `proxy_up` | Stat / Gauge |
| Response Time Trend | `proxy_response_seconds` | Time series |
| Error Rate | `rate(proxy_check_errors_total[5m])` | Graph |
| Proxy Status Overview | `count by (type) (proxy_up == 1)` | Pie chart |

### Import Dashboard

1. Create a new dashboard in Grafana
2. Add a Prometheus data source (pointing to your Prometheus instance)
3. Create panels using the PromQL queries above

## Alerting Rules

Add to Prometheus `rules.yml`:

```yaml
groups:
  - name: proxy_alerts
    rules:
      - alert: ProxyDown
        expr: proxy_up == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Proxy {{ $labels.name }} is down"
          description: "Proxy {{ $labels.name }} ({{ $labels.address }}) has been unavailable for more than 2 minutes"

      - alert: ProxyHighErrorRate
        expr: rate(proxy_check_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Proxy {{ $labels.name }} has high error rate"
          description: "Proxy {{ $labels.name }} error rate over the last 5 minutes is {{ $value | humanize }}/s"

      - alert: ProxySlowResponse
        expr: proxy_response_seconds > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Proxy {{ $labels.name }} is slow"
          description: "Proxy {{ $labels.name }} response time exceeds 5 seconds"
```

## Makefile Usage

### Common Commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary for current platform |
| `make build-linux-amd64` | Build for Linux AMD64 |
| `make build-linux-arm64` | Build for Linux ARM64 |
| `make build-darwin-amd64` | Build for macOS AMD64 |
| `make build-darwin-arm64` | Build for macOS ARM64 |
| `make build-all` | Build for all platforms |
| `make test` | Run unit tests |
| `make test-race` | Run tests with race detection |
| `make test-cover` | Generate test coverage report |
| `make fmt` | Format code |
| `make lint` | Run linter |
| `make check` | Run fmt + lint + test |
| `make clean` | Clean build artifacts |
| `make run` | Build and run |
| `make help` | Show help message |

### Cross-compilation Example

```bash
# Build for Linux AMD64
make build-linux-amd64

# Build for all platforms
make build-all
```

## Troubleshooting

### Common Issues

| Issue | Possible Cause | Solution |
|-------|----------------|----------|
| `proxy_up == 0` | Wrong proxy address or network unreachable | Check address format `host:port`, verify network connectivity |
| Auth failed (`auth` error) | Wrong username or password | Verify `username` and `password` configuration |
| Timeout (`timeout` error) | High network latency or target unreachable | Increase `timeout` value, check `target` URL |
| Status error (`status` error) | Target returned non-2xx status | Check if `target` URL is correct |
| Regex mismatch (`regex` error) | Response body doesn't match `expect_regex` | Test actual response with `curl`, adjust regex |

### Debug Commands

```bash
# Check if service is running
curl http://localhost:9099/health

# View raw metrics
curl http://localhost:9099/metrics

# Test proxy connection manually
curl -x http://192.168.1.100:8080 https://api.ipify.org
```

### Log Analysis

Logs are output on startup:

```
2026/03/21 10:00:00 Configuration loaded: 3 proxies configured
2026/03/21 10:00:00 Starting proxycheck-exporter on :9099
2026/03/21 10:00:00 [http-proxy-1] OK - 234ms
2026/03/21 10:00:00 [https-proxy-auth] FAILED - auth: authentication failed
```

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/metrics` | GET | Prometheus metrics endpoint |
| `/health` | GET | Health check (returns "OK" if server is running) |

## Security Considerations

- **Credential Storage**: Avoid storing plaintext credentials in the configuration file for production deployments. Consider using environment variables or a secrets manager.
- **File Permissions**: Restrict access to the configuration file: `chmod 600 config.yaml`
- **Network Security**: The `/metrics` endpoint exposes proxy addresses. Use network policies or authentication to restrict access if needed.
- **Health Check Note**: The `/health` endpoint only indicates the HTTP service is running, not proxy availability.

## License

MIT License
