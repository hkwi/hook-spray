package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/exp/slog"
)

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
	var dests ss
	flag.Var(&dests, "dest", "URL to send to")
	flag.Parse()
	slog.Info("config", "dests", dests)

	metrics := newRelayMetrics(prometheus.DefaultRegisterer)
	mux := http.NewServeMux()
	mux.Handle("/hook", newRelayHandler(dests, *allowGet, http.DefaultClient, metrics))
	mux.Handle("/metrics", promhttp.Handler())

	fmt.Printf("Relay to %v", dests)
	http.ListenAndServe(*addr, mux)
}
