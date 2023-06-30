package main

import (
	"bytes"
	"flag"
	"fmt"
	ulid "github.com/oklog/ulid/v2"
	"golang.org/x/exp/slog"
	"net/http"
	"net/url"
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

	http.HandleFunc("/hook", func(w http.ResponseWriter, r *http.Request) {
		xid := ulid.Make()
		slog.Info(r.Method, "xid", xid)
		defer r.Body.Close()
		if r.Method == "POST" {
			contentType := r.Header.Get("Content-Type")
			buf := bytes.Buffer{}
			buf.ReadFrom(r.Body)
			defer r.Body.Close()

			for _, dest := range dests {
				if resp, err := http.Post(dest, contentType, bytes.NewReader(buf.Bytes())); err != nil {
					slog.Error("call failed", "xid", xid, "err", err)
				} else {
					slog.Info(resp.Status, "xid", xid, "dest", dest)
					defer resp.Body.Close()
				}
			}
		} else if r.Method == "GET" && *allowGet {
			for _, dest := range dests {
				if u, err := url.Parse(dest); err != nil {
					slog.Error("URL error", "xid", xid, "err", err)
				} else if m, err := url.ParseQuery(u.RawQuery); err != nil {
					slog.Error("URL failed", "xid", xid, "err", err)
				} else {
					for k, vs := range r.Form {
						for _, v := range vs {
							m.Add(k, v)
						}
					}
					u.RawQuery = m.Encode()
					if resp, err := http.Get(u.String()); err != nil {
						slog.Error("call failed", "xid", xid, "err", err)
					} else {
						slog.Info("ok", "xid", xid, "dest", dest, "status", resp.Status)
						defer resp.Body.Close()
					}
				}
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Unsupported method")
		}
	})
	fmt.Printf("Relay to %v", dests)
	http.ListenAndServe(*addr, nil)
}
