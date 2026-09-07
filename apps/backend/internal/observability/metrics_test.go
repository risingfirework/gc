package observability

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHTTPMetricsUseRoutePattern(t *testing.T) {
	metrics := NewMetrics()
	router := chi.NewRouter()
	router.Use(metrics.Middleware)
	router.Get("/api/items/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/items/123?token=secret", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected response code: %d", response.Code)
	}

	metricsResponse := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(metricsResponse, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	result := metricsResponse.Result()
	t.Cleanup(func() {
		if err := result.Body.Close(); err != nil {
			t.Errorf("close metrics response: %v", err)
		}
	})
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("read metrics response: %v", err)
	}
	want := `tka_http_requests_total{method="GET",route="/api/items/{id}",status="204"} 1`
	if !strings.Contains(string(body), want) {
		t.Fatalf("route metric missing; wanted %q", want)
	}
	if strings.Contains(string(body), "123") || strings.Contains(string(body), "secret") {
		t.Fatal("metrics leaked raw path or query data")
	}
}
