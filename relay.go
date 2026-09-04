package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	ulid "github.com/oklog/ulid/v2"
	"golang.org/x/exp/slog"
)

type relayHandler struct {
	dests    []string
	allowGet bool
	client   *http.Client
	metrics  *relayMetrics
}

func newRelayHandler(dests []string, allowGet bool, client *http.Client, metrics *relayMetrics) http.Handler {
	return &relayHandler{
		dests:    dests,
		allowGet: allowGet,
		client:   client,
		metrics:  metrics,
	}
}

func (handler *relayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	xid := ulid.Make()
	started := time.Now()
	body := &countingReadCloser{ReadCloser: r.Body}
	r.Body = body
	status := http.StatusOK

	handler.metrics.inFlight.Inc()
	defer func() {
		elapsed := time.Since(started)
		handler.metrics.inFlight.Dec()
		handler.metrics.observeRequest(r.Method, status, body.bytes, elapsed)
	}()
	defer r.Body.Close()

	slog.Info(r.Method, "xid", xid)
	if r.Method == http.MethodPost {
		handler.relayPost(xid.String(), r)
		return
	}
	if r.Method == http.MethodGet && handler.allowGet {
		handler.relayGet(xid.String(), r)
		return
	}

	status = http.StatusBadRequest
	w.WriteHeader(status)
	fmt.Fprint(w, "Unsupported method")
}

func (handler *relayHandler) relayPost(xid string, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	buf := bytes.Buffer{}
	readStarted := time.Now()
	buf.ReadFrom(r.Body)
	handler.metrics.bodyReadDuration.Observe(time.Since(readStarted).Seconds())

	for _, dest := range handler.dests {
		started := time.Now()
		resp, err := handler.client.Post(dest, contentType, bytes.NewReader(buf.Bytes()))
		elapsed := time.Since(started)
		metricDest := metricDestination(dest)
		if err != nil {
			handler.metrics.observeUpstream(metricDest, "error", elapsed)
			slog.Error("call failed", "xid", xid, "err", err)
			continue
		}

		handler.metrics.observeUpstream(metricDest, strconv.Itoa(resp.StatusCode), elapsed)
		slog.Info(resp.Status, "xid", xid, "dest", dest)
		defer resp.Body.Close()
	}
}

func (handler *relayHandler) relayGet(xid string, r *http.Request) {
	for _, dest := range handler.dests {
		u, err := url.Parse(dest)
		if err != nil {
			slog.Error("URL error", "xid", xid, "err", err)
			continue
		}
		m, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			slog.Error("URL failed", "xid", xid, "err", err)
			continue
		}
		for k, values := range r.Form {
			for _, value := range values {
				m.Add(k, value)
			}
		}
		u.RawQuery = m.Encode()

		started := time.Now()
		resp, err := handler.client.Get(u.String())
		elapsed := time.Since(started)
		metricDest := metricDestination(dest)
		if err != nil {
			handler.metrics.observeUpstream(metricDest, "error", elapsed)
			slog.Error("call failed", "xid", xid, "err", err)
			continue
		}

		handler.metrics.observeUpstream(metricDest, strconv.Itoa(resp.StatusCode), elapsed)
		slog.Info("ok", "xid", xid, "dest", dest, "status", resp.Status)
		defer resp.Body.Close()
	}
}

func metricDestination(dest string) string {
	u, err := url.Parse(dest)
	if err != nil || u.Host == "" {
		return "invalid"
	}
	return u.Host + u.EscapedPath()
}

type countingReadCloser struct {
	io.ReadCloser
	bytes int
}

func (reader *countingReadCloser) Read(p []byte) (int, error) {
	n, err := reader.ReadCloser.Read(p)
	reader.bytes += n
	return n, err
}
