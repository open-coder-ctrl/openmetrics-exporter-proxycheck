package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector manages Prometheus metrics
type Collector struct {
	proxyUp           *prometheus.GaugeVec
	proxyResponseTime *prometheus.GaugeVec
	proxyStatusCode   *prometheus.GaugeVec
	proxyCheckErrors  *prometheus.CounterVec
}

// NewCollector creates and registers Prometheus metrics with the default registry
func NewCollector() *Collector {
	return NewCollectorWithRegistry(prometheus.DefaultRegisterer)
}

// NewCollectorWithRegistry creates and registers Prometheus metrics with a custom registry
func NewCollectorWithRegistry(reg prometheus.Registerer) *Collector {
	c := &Collector{
		proxyUp: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "proxy_up",
			Help: "Whether the proxy is up (1) or down (0)",
		}, []string{"name", "type", "address"}),

		proxyResponseTime: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "proxy_response_seconds",
			Help: "Response time of the proxy check in seconds",
		}, []string{"name", "type"}),

		proxyStatusCode: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "proxy_status_code",
			Help: "HTTP status code returned by the target through the proxy",
		}, []string{"name", "type"}),

		proxyCheckErrors: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "proxy_check_errors_total",
			Help: "Total number of proxy check errors by type",
		}, []string{"name", "type", "error_type"}),
	}

	return c
}

// Update updates the metrics with a new check result
func (c *Collector) Update(proxy ProxyConfig, result CheckResult) {
	labels := prometheus.Labels{
		"name":    proxy.Name,
		"type":    proxy.Type,
		"address": proxy.Address,
	}

	// Update proxy_up
	if result.Success {
		c.proxyUp.With(labels).Set(1)
	} else {
		c.proxyUp.With(labels).Set(0)
	}

	// Update response time
	responseLabels := prometheus.Labels{
		"name": proxy.Name,
		"type": proxy.Type,
	}
	c.proxyResponseTime.With(responseLabels).Set(result.ResponseTime.Seconds())

	// Update status code
	c.proxyStatusCode.With(responseLabels).Set(float64(result.StatusCode))

	// Increment error counter if there's an error
	if !result.Success && result.ErrorType != "" {
		errorLabels := prometheus.Labels{
			"name":       proxy.Name,
			"type":       proxy.Type,
			"error_type": result.ErrorType,
		}
		c.proxyCheckErrors.With(errorLabels).Inc()
	}
}
