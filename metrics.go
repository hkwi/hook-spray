package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var durationBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 15, 30, 60,
}

var bodySizeBuckets = []float64{
	1024,
	4 * 1024,
	16 * 1024,
	64 * 1024,
	256 * 1024,
	1 * 1024 * 1024,
	2 * 1024 * 1024,
	4 * 1024 * 1024,
	8 * 1024 * 1024,
	10 * 1024 * 1024,
	16 * 1024 * 1024,
	32 * 1024 * 1024,
	64 * 1024 * 1024,
}

type relayMetrics struct {
	inFlight         prometheus.Gauge
	requests         *prometheus.CounterVec
	requestBodyBytes *prometheus.HistogramVec
	bodyReadDuration prometheus.Histogram
	requestDuration  *prometheus.HistogramVec
	upstreamRequests *prometheus.CounterVec
	upstreamDuration *prometheus.HistogramVec
}

func newRelayMetrics(registerer prometheus.Registerer) *relayMetrics {
	metrics := &relayMetrics{
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "hook_spray",
			Name:      "in_flight_requests",
			Help:      "Number of relay requests currently being handled.",
		}),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "hook_spray",
			Name:      "requests_total",
			Help:      "Total number of relay requests.",
		}, []string{"method", "status"}),
		requestBodyBytes: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "hook_spray",
			Name:      "request_body_bytes",
			Help:      "Size of relay request bodies in bytes.",
			Buckets:   bodySizeBuckets,
		}, []string{"method"}),
		bodyReadDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "hook_spray",
			Name:      "request_body_read_duration_seconds",
			Help:      "Time spent reading POST request bodies.",
			Buckets:   durationBuckets,
		}),
		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "hook_spray",
			Name:      "request_duration_seconds",
			Help:      "Time spent handling relay requests.",
			Buckets:   durationBuckets,
		}, []string{"method"}),
		upstreamRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "hook_spray",
			Name:      "upstream_requests_total",
			Help:      "Total number of requests sent to relay destinations.",
		}, []string{"destination", "status"}),
		upstreamDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "hook_spray",
			Name:      "upstream_request_duration_seconds",
			Help:      "Time spent waiting for relay destinations.",
			Buckets:   durationBuckets,
		}, []string{"destination"}),
	}
	registerer.MustRegister(
		metrics.inFlight,
		metrics.requests,
		metrics.requestBodyBytes,
		metrics.bodyReadDuration,
		metrics.requestDuration,
		metrics.upstreamRequests,
		metrics.upstreamDuration,
	)
	return metrics
}

func (metrics *relayMetrics) observeRequest(method string, status, bodyBytes int, elapsed time.Duration) {
	method = metricMethod(method)
	metrics.requests.WithLabelValues(method, strconv.Itoa(status)).Inc()
	metrics.requestBodyBytes.WithLabelValues(method).Observe(float64(bodyBytes))
	metrics.requestDuration.WithLabelValues(method).Observe(elapsed.Seconds())
}

func metricMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodPost:
		return method
	default:
		return "other"
	}
}

func (metrics *relayMetrics) observeUpstream(destination, status string, elapsed time.Duration) {
	metrics.upstreamRequests.WithLabelValues(destination, status).Inc()
	metrics.upstreamDuration.WithLabelValues(destination).Observe(elapsed.Seconds())
}
