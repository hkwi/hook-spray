package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServeMuxUsesConfiguredEntryPath(t *testing.T) {
	relay := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	metrics := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	mux, err := newServeMux("/webhook", relay, metrics)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		path string
		want int
	}{
		{path: "/webhook", want: http.StatusNoContent},
		{path: metricsPath, want: http.StatusAccepted},
		{path: "/hook", want: http.StatusNotFound},
	} {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, test.path, nil))
		if recorder.Code != test.want {
			t.Errorf("%s status = %d, want %d", test.path, recorder.Code, test.want)
		}
	}
}

func TestNewServeMuxRejectsInvalidEntryPaths(t *testing.T) {
	for _, entry := range []string{"", "webhook", metricsPath} {
		if _, err := newServeMux(entry, http.NotFoundHandler(), http.NotFoundHandler()); err == nil {
			t.Errorf("newServeMux(%q) returned no error", entry)
		}
	}
}
