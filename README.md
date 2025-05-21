# nexen-metrics

A Prometheus-based metrics and instrumentation package for the Nexen platform.

## Overview

`nexen-metrics` is a thin wrapper around the Prometheus Go client that standardizes metric registration, exposes a `/metrics` HTTP handler, and provides common counters and histograms for HTTP request duration, error rates, and custom application events.

## Installation

```bash
go get github.com/nexen-io/nexen-metrics
```

## Basic Usage

```go
package main

import (
    "net/http"

    "github.com/nexen-io/nexen-metrics"
)

func main() {
    // Create a new metrics instance
    m := metrics.New(metrics.WithServiceName("my-service"))

    // Create your HTTP server
    mux := http.NewServeMux()

    // Add your routes
    mux.HandleFunc("/api/v1/example", exampleHandler)

    // Expose metrics endpoint
    mux.Handle("/metrics", m.Handler())

    // Wrap your handler with metrics middleware
    instrumentedHandler := m.Instrument(mux)

    // Start your server with the instrumented handler
    http.ListenAndServe(":8080", instrumentedHandler)
}

func exampleHandler(w http.ResponseWriter, r *http.Request) {
    // Your handler logic
}
```

## Features

* HTTP request metrics (count, duration, error rates)
* Custom application event tracking
* Service-specific gauges
* Registry for custom metrics
* Prometheus-compatible `/metrics` endpoint

## Configuration Options

* `WithServiceName(name string)` - Set the service name for metric labels
* `WithHistogramBuckets(buckets []float64)` - Configure custom histogram buckets
* `WithRegistry(registry *prometheus.Registry)` - Use a custom Prometheus registry

## Advanced Usage

See [documentation](docs/) for advanced usage examples and complete API reference.

## License

See the [LICENSE](LICENSE) file for license rights and limitations.
