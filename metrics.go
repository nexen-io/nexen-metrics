// Package metrics provides a Prometheus-based instrumentation wrapper for Nexen services.
// It exposes a standard registry, default metrics (HTTP request count, duration, errors),
// and a ready-to-use HTTP handler for scrape endpoints.
package metrics

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/nexen-io/nexen-metrics/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Flags for the metrics server
var (
	listenAddress = flag.String("metrics.listen-address", ":8080", "The address to listen on for HTTP requests.")
	metricsPath   = flag.String("metrics.path", "/metrics", "The path to listen on for HTTP requests.")
)

// Namespace and subsystem for all Nexen metrics
const (
	namespace = "nexen"
	subsystem = "service"
)

// Option is a functional option for configuring the Metrics.
type Option func(*Metrics)

// WithHistogramBuckets configures custom histogram buckets for HTTP duration metrics.
func WithHistogramBuckets(buckets []float64) Option {
	return func(m *Metrics) {
		m.histogramBuckets = buckets
	}
}

// WithServiceName sets a custom service name for metric labels.
func WithServiceName(serviceName string) Option {
	return func(m *Metrics) {
		m.serviceName = serviceName
	}
}

// WithRegistry allows providing a custom prometheus registry.
func WithRegistry(registry *prometheus.Registry) Option {
	return func(m *Metrics) {
		m.registry = registry
	}
}

// Metrics holds common instrumenters and the Prometheus registry.
type Metrics struct {
	registry         *prometheus.Registry
	httpRequests     *prometheus.CounterVec
	httpDuration     *prometheus.HistogramVec
	httpErrors       *prometheus.CounterVec
	applicationEvent *prometheus.CounterVec
	serviceGauge     *prometheus.GaugeVec
	scrapeHandler    http.Handler
	histogramBuckets []float64
	serviceName      string
}

// New constructs a Metrics instance, registers standard collectors, and returns it.
func New(opts ...Option) *Metrics {
	m := &Metrics{
		registry:         prometheus.NewRegistry(),
		histogramBuckets: internal.DefaultHTTPBuckets(),
		serviceName:      "default",
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Standard process and Go runtime metrics
	m.registry.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)

	// HTTP request count, partitioned by method, path and service
	m.httpRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests received",
		},
		[]string{"method", "path", "service"},
	)
	m.registry.MustRegister(m.httpRequests)

	// HTTP request duration histogram
	m.httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_request_duration_seconds",
			Help:      "Histogram of HTTP request durations",
			Buckets:   m.histogramBuckets,
		},
		[]string{"method", "path", "service"},
	)
	m.registry.MustRegister(m.httpDuration)

	// HTTP error count, partitioned by method, path, status code and service
	m.httpErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_errors_total",
			Help:      "Total number of HTTP responses with error status codes",
		},
		[]string{"method", "path", "code", "service"},
	)
	m.registry.MustRegister(m.httpErrors)

	// Generic application event counter for custom events
	m.applicationEvent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "application_events_total",
			Help:      "Count of application-specific events",
		},
		[]string{"event", "service"},
	)
	m.registry.MustRegister(m.applicationEvent)

	// Service-specific gauge for arbitrary numeric values
	m.serviceGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "gauge",
			Help:      "Service-specific gauge for arbitrary values",
		},
		[]string{"name", "service"},
	)
	m.registry.MustRegister(m.serviceGauge)

	// Prometheus HTTP handler for /metrics
	m.scrapeHandler = promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})

	return m
}

// Handler returns the HTTP handler to expose the /metrics endpoint.
func (m *Metrics) Handler() http.Handler {
	return m.scrapeHandler
}

// Registry returns the underlying Prometheus registry.
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// Instrument wraps an HTTP handler to collect request count, duration, and errors.
// It should be used as middleware at the outermost layer.
func (m *Metrics) Instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		method := r.Method

		// Increment request count
		m.httpRequests.WithLabelValues(method, path, m.serviceName).Inc()

		// Create timer to observe duration
		start := time.Now()

		// Capture status code via ResponseWriter wrapper
		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		// Record duration
		duration := time.Since(start).Seconds()
		m.httpDuration.WithLabelValues(method, path, m.serviceName).Observe(duration)

		// If status code >= 400, increment error counter
		statusCode := rw.statusCode
		if statusCode >= 400 {
			m.httpErrors.WithLabelValues(method, path, http.StatusText(statusCode), m.serviceName).Inc()
		}
	})
}

// RecordEvent increments a counter for application-specific events.
func (m *Metrics) RecordEvent(event string) {
	m.applicationEvent.WithLabelValues(event, m.serviceName).Inc()
}

// SetGauge sets the value of a named gauge.
func (m *Metrics) SetGauge(name string, value float64) {
	m.serviceGauge.WithLabelValues(name, m.serviceName).Set(value)
}

// IncrementGauge increments a named gauge by 1.
func (m *Metrics) IncrementGauge(name string) {
	m.serviceGauge.WithLabelValues(name, m.serviceName).Inc()
}

// DecrementGauge decrements a named gauge by 1.
func (m *Metrics) DecrementGauge(name string) {
	m.serviceGauge.WithLabelValues(name, m.serviceName).Dec()
}

// RegisterCounter creates and registers a new counter with the given name and help text.
func (m *Metrics) RegisterCounter(name, help string, labels []string) (*prometheus.CounterVec, error) {
	allLabels := append(labels, "service")
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		allLabels,
	)

	err := m.registry.Register(counter)
	if err != nil {
		return nil, fmt.Errorf("failed to register counter %s: %w", name, err)
	}
	return counter, nil
}

// RegisterHistogram creates and registers a new histogram with the given name, help text, and buckets.
func (m *Metrics) RegisterHistogram(name, help string, buckets []float64, labels []string) (*prometheus.HistogramVec, error) {
	if buckets == nil {
		buckets = m.histogramBuckets
	}

	allLabels := append(labels, "service")
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		allLabels,
	)

	err := m.registry.Register(histogram)
	if err != nil {
		return nil, fmt.Errorf("failed to register histogram %s: %w", name, err)
	}
	return histogram, nil
}

// RegisterGauge creates and registers a new gauge with the given name and help text.
func (m *Metrics) RegisterGauge(name, help string, labels []string) (*prometheus.GaugeVec, error) {
	allLabels := append(labels, "service")
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      name,
			Help:      help,
		},
		allLabels,
	)

	err := m.registry.Register(gauge)
	if err != nil {
		return nil, fmt.Errorf("failed to register gauge %s: %w", name, err)
	}
	return gauge, nil
}

// responseWriter wraps http.ResponseWriter to capture status codes
// without altering its behaviour.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and delegates to the real writer.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write ensures that if WriteHeader was not called explicitly,
// we still capture the default 200 status code.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}
