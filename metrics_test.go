package metrics

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewMetrics(t *testing.T) {
	metrics := New()
	if metrics == nil {
		t.Fatal("Expected metrics instance to be non-nil")
	}

	if metrics.registry == nil {
		t.Fatal("Expected registry to be initialized")
	}

	if metrics.httpRequests == nil {
		t.Fatal("Expected httpRequests counter to be initialized")
	}

	if metrics.httpDuration == nil {
		t.Fatal("Expected httpDuration histogram to be initialized")
	}

	if metrics.httpErrors == nil {
		t.Fatal("Expected httpErrors counter to be initialized")
	}
}

func TestMetricsHandler(t *testing.T) {
	metrics := New()
	handler := metrics.Handler()

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Check that the response contains basic Prometheus metrics
	if !strings.Contains(string(body), "# HELP") {
		t.Fatal("Expected metrics response to contain '# HELP'")
	}
}

func TestInstrument(t *testing.T) {
	metrics := New(WithServiceName("test-service"))

	// Create a test handler that returns different status codes
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with metrics instrumentation
	instrumentedHandler := metrics.Instrument(testHandler)

	// Test successful request
	req := httptest.NewRequest("GET", "/success", nil)
	w := httptest.NewRecorder()
	instrumentedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code 200, got %d", w.Code)
	}

	// Test error request
	req = httptest.NewRequest("GET", "/error", nil)
	w = httptest.NewRecorder()
	instrumentedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status code 500, got %d", w.Code)
	}

	// Check metrics output
	metricsReq := httptest.NewRequest("GET", "/metrics", nil)
	metricsW := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(metricsW, metricsReq)

	body, _ := ioutil.ReadAll(metricsW.Result().Body)
	bodyStr := string(body)

	// Should have incremented request counters
	if !strings.Contains(bodyStr, "nexen_service_http_requests_total") {
		t.Fatal("Expected metrics to contain http_requests_total")
	}

	// Should have recorded error
	if !strings.Contains(bodyStr, "nexen_service_http_errors_total") {
		t.Fatal("Expected metrics to contain http_errors_total")
	}
}

func TestCustomMetrics(t *testing.T) {
	metrics := New(WithServiceName("test-service"))

	// Record a custom event
	metrics.RecordEvent("test-event")

	// Set a custom gauge
	metrics.SetGauge("test-gauge", 42.0)

	// Register a custom counter
	counter, err := metrics.RegisterCounter("test_counter", "A test counter", []string{"label1"})
	if err != nil {
		t.Fatalf("Failed to register counter: %v", err)
	}

	// Increment the custom counter
	counter.WithLabelValues("value1", "test-service").Inc()

	// Check metrics output
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(w, req)

	body, _ := ioutil.ReadAll(w.Result().Body)
	bodyStr := string(body)

	if !strings.Contains(bodyStr, "nexen_service_application_events_total") {
		t.Fatal("Expected metrics to contain application_events_total")
	}

	if !strings.Contains(bodyStr, "nexen_service_gauge") {
		t.Fatal("Expected metrics to contain gauge")
	}

	if !strings.Contains(bodyStr, "nexen_service_test_counter") {
		t.Fatal("Expected metrics to contain test_counter")
	}
}
