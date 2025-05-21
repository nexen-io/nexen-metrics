// Package internal provides private helpers for the metrics package.
package internal

// DefaultHTTPBuckets returns the default histogram buckets for HTTP request durations.
// These are optimized for typical API request patterns in the Nexen platform.
func DefaultHTTPBuckets() []float64 {
	return []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
}

// DefaultLLMLatencyBuckets returns histogram buckets optimized for LLM inference latency.
// These have a wider range to accommodate the variability in LLM response times.
func DefaultLLMLatencyBuckets() []float64 {
	return []float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 30, 60, 120}
}

// DefaultMemoryBuckets returns histogram buckets suitable for memory usage metrics (in MB).
func DefaultMemoryBuckets() []float64 {
	return []float64{50, 100, 250, 500, 1000, 2000, 5000, 10000}
}
