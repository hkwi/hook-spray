package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/slog"
)

const metricsPath = "/metrics"

type ss []string

func (self ss) String() string {
	return fmt.Sprintf("%v", []string(self))
}

func (self *ss) Set(value string) error {
	*self = append(*self, value)
	return nil
}

func main() {
	slog.Info(fmt.Sprintf("%#v", flag.Args()))
	addr := flag.String("listen", ":8080", "Addr to listen")
	allowGet := flag.Bool("allow-get", false, "normal webhook use POST")
	entry := flag.String("entry", "/hook", "URL path to receive webhook requests")
	var dests ss
	flag.Var(&dests, "dest", "URL to send to")
	flag.Parse()
	slog.Info("config", "dests", dests, "entry", *entry)

	metrics := newRelayMetrics(prometheus.DefaultRegisterer)
	mux, err := newServeMux(
		*entry,
		newRelayHandler(dests, *allowGet, http.DefaultClient, metrics),
		promhttp.Handler(),
	)
	if err != nil {
		slog.Error("invalid entry path", "err", err)
		return
	}

	fmt.Printf("Relay to %v", dests)
	http.ListenAndServe(*addr, mux)
}

func newServeMux(entry string, relay, metrics http.Handler) (*http.ServeMux, error) {
	if entry == "" || entry[0] != '/' {
		return nil, fmt.Errorf("entry path %q must start with /", entry)
	}
	if entry == metricsPath {
		return nil, fmt.Errorf("entry path must differ from %s", metricsPath)
	}

	mux := http.NewServeMux()
	mux.Handle(entry, relay)
	mux.Handle(metricsPath, metrics)
	return mux, nil
}
