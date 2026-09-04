package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
)

func TestRelayPostDebugBodyLogging(t *testing.T) {
	for _, test := range []struct {
		name       string
		debug      bool
		status     int
		wantBodies bool
	}{
		{name: "enabled for HTTP 500", debug: true, status: http.StatusInternalServerError, wantBodies: true},
		{name: "disabled for HTTP 500", debug: false, status: http.StatusInternalServerError},
		{name: "enabled for HTTP 400", debug: true, status: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			var logOutput bytes.Buffer
			previousLogger := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(&logOutput, nil)))
			defer slog.SetDefault(previousLogger)

			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: test.status,
					Status:     http.StatusText(test.status),
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("upstream failure")),
					Request:    r,
				}, nil
			})}
			handler := newRelayHandler(
				[]string{"http://upstream.example/update"},
				false,
				test.debug,
				client,
				newRelayMetrics(prometheus.NewRegistry()),
			)
			handler.ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("payload")),
			)

			for _, body := range []string{
				`"request_body":"payload"`,
				`"response_body":"upstream failure"`,
			} {
				if got := strings.Contains(logOutput.String(), body); got != test.wantBodies {
					t.Errorf("log contains %q = %t, want %t\n%s", body, got, test.wantBodies, logOutput.String())
				}
			}
		})
	}
}
