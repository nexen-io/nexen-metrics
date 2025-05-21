# Advanced Usage

## Custom Metrics

### Registering Custom Counters

```go
counter, err := metrics.RegisterCounter(
    "api_request_count",
    "Count of API requests by endpoint",
    []string{"endpoint"}
)
if err != nil {
    log.Fatalf("Failed to register counter: %v", err)
}

// Increment the counter
counter.WithLabelValues("/api/v1/completions", "my-service").Inc()
```

### Registering Custom Histograms

```go
// Using LLM-specific latency buckets
histogram, err := metrics.RegisterHistogram(
    "llm_inference_seconds",
    "LLM inference duration in seconds",
    internal.DefaultLLMLatencyBuckets(),
    []string{"model"}
)
if err != nil {
    log.Fatalf("Failed to register histogram: %v", err)
}

// Track inference duration
timer := prometheus.NewTimer(histogram.WithLabelValues("gpt-4", "my-service"))
defer timer.ObserveDuration()

// Perform LLM inference...
```

### Using Gauges

```go
// Set the gauge to the current queue size
metrics.SetGauge("queue_depth", float64(len(queue)))

// Alternatively, increment/decrement
metrics.IncrementGauge("active_connections")
metrics.DecrementGauge("active_connections")
```

## Recording Application Events

```go
// Record specific application events
metrics.RecordEvent("cache_miss")
metrics.RecordEvent("rate_limited")
metrics.RecordEvent("authorization_failure")
```

## Custom HTTP Instrumentation

For more fine-grained control over HTTP instrumentation:

```go
// Create a custom middleware
func customMiddleware(next http.Handler, m *metrics.Metrics) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract tenant ID from request
        tenantID := r.Header.Get("X-Tenant-ID")

        // Create custom counter with tenant ID label
        counter, _ := m.RegisterCounter(
            "tenant_requests_total",
            "Total requests by tenant",
            []string{"tenant_id"}
        )
        counter.WithLabelValues(tenantID, "my-service").Inc()

        // Continue with standard instrumentation
        instrumentedHandler := m.Instrument(next)
        instrumentedHandler.ServeHTTP(w, r)
    })
}
```

## OpenTelemetry Integration

To use both Prometheus and OpenTelemetry:

```go
import (
    "github.com/nexen-io/nexen-metrics"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/prometheus"
    "go.opentelemetry.io/otel/sdk/metric"
)

func setupMetrics() (*metrics.Metrics, error) {
    // Create the Prometheus exporter
    exporter, err := prometheus.New()
    if err != nil {
        return nil, err
    }

    // Create the MeterProvider
    provider := metric.NewMeterProvider(
        metric.WithReader(exporter),
    )
    otel.SetMeterProvider(provider)

    // Create metrics with the Prometheus registry from the exporter
    m := metrics.New(
        metrics.WithRegistry(exporter.Collector()),
        metrics.WithServiceName("my-service"),
    )

    return m, nil
}
```
