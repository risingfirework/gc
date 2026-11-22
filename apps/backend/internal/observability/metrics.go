// Package observability menyediakan metrik Prometheus dan sentry.
package observability

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

func NewMetrics() *Metrics {
	metrics := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "tka", Subsystem: "http", Name: "requests_total",
			Help: "Total HTTP requests handled by the API.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "tka", Subsystem: "http", Name: "request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.15, 0.25, 0.5, 1, 2.5, 5},
		}, []string{"method", "route", "status"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "tka", Subsystem: "http", Name: "in_flight_requests",
			Help: "Current number of in-flight API requests.",
		}),
	}
	metrics.registry.MustRegister(
		metrics.requests,
		metrics.duration,
		metrics.inFlight,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return metrics
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
}

// Gather mengembalikan snapshot metrik untuk konsumsi UI (JSON).
func (m *Metrics) Gather() (map[string]any, error) {
	if m == nil || m.registry == nil {
		return map[string]any{}, nil
	}
	families, err := m.registry.Gather()
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	for _, f := range families {
		name := f.GetName()
		switch {
		case name == "tka_http_requests_total":
			total := 0.0
			errors5xx := 0.0
			for _, metric := range f.GetMetric() {
				for _, lp := range metric.GetLabel() {
					if lp.GetName() == "status" {
						v := metric.GetCounter().GetValue()
						total += v
						if len(lp.GetValue()) > 0 && lp.GetValue()[0] == '5' {
							errors5xx += v
						}
					}
				}
			}
			result["requests_total"] = total
			result["errors_5xx_total"] = errors5xx
		case name == "tka_http_in_flight_requests":
			if len(f.GetMetric()) > 0 {
				result["in_flight"] = f.GetMetric()[0].GetGauge().GetValue()
			}
		}
		// Fallback fleksibel untuk memori, CPU, goroutine.
		if result["memory_bytes"] == nil && (strings.Contains(name, "memory") && strings.Contains(name, "resident")) {
			if len(f.GetMetric()) > 0 {
				result["memory_bytes"] = f.GetMetric()[0].GetGauge().GetValue()
			}
		}
		if result["memory_bytes"] == nil && strings.Contains(name, "memstats") && strings.Contains(name, "alloc") {
			if len(f.GetMetric()) > 0 {
				result["memory_bytes"] = f.GetMetric()[0].GetGauge().GetValue()
			}
		}
		if result["cpu_seconds_total"] == nil && strings.Contains(name, "cpu") && strings.Contains(name, "seconds") {
			if len(f.GetMetric()) > 0 {
				if f.GetType() == dto.MetricType_COUNTER {
					result["cpu_seconds_total"] = f.GetMetric()[0].GetCounter().GetValue()
				} else if f.GetType() == dto.MetricType_GAUGE {
					result["cpu_seconds_total"] = f.GetMetric()[0].GetGauge().GetValue()
				}
			}
		}
		if result["goroutines"] == nil && strings.Contains(name, "goroutine") {
			if len(f.GetMetric()) > 0 {
				result["goroutines"] = f.GetMetric()[0].GetGauge().GetValue()
			}
		}
	}
	return result, nil
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		startedAt := time.Now()
		writer := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		m.inFlight.Inc()
		defer func() {
			m.inFlight.Dec()
			status := writer.Status()
			if status == 0 {
				status = http.StatusOK
			}
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			labels := []string{r.Method, route, strconv.Itoa(status)}
			m.requests.WithLabelValues(labels...).Inc()
			m.duration.WithLabelValues(labels...).Observe(time.Since(startedAt).Seconds())
		}()
		next.ServeHTTP(writer, r)
	})
}
