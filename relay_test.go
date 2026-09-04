package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestRelayPostMetrics(t *testing.T) {
	var received string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		received = string(body)
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Status:     "204 No Content",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	})}

	registry := prometheus.NewRegistry()
	metrics := newRelayMetrics(registry)
	handler := newRelayHandler([]string{"http://upstream.example/update"}, false, client, metrics)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("payload"))
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if received != "payload" {
		t.Fatalf("upstream body = %q, want payload", received)
	}

	metricsRecorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		metricsRecorder,
		httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)
	output := metricsRecorder.Body.String()
	for _, want := range []string{
		`hook_spray_in_flight_requests 0`,
		`hook_spray_request_body_bytes_sum{method="POST"} 7`,
		`hook_spray_request_body_read_duration_seconds_count 1`,
		`hook_spray_request_duration_seconds_count{method="POST"} 1`,
		`hook_spray_requests_total{method="POST",status="200"} 1`,
		`hook_spray_upstream_request_duration_seconds_count{destination="upstream.example/update"} 1`,
		`hook_spray_upstream_requests_total{destination="upstream.example/update",status="204"} 1`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("metrics output does not contain %q\n%s", want, output)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestUnsupportedMethodMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := newRelayMetrics(registry)
	handler := newRelayHandler(nil, false, http.DefaultClient, metrics)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/hook", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}

	metricsRecorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(
		metricsRecorder,
		httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)
	want := `hook_spray_requests_total{method="other",status="400"} 1`
	if !strings.Contains(metricsRecorder.Body.String(), want) {
		t.Errorf("metrics output does not contain %q\n%s", want, metricsRecorder.Body.String())
	}
}
