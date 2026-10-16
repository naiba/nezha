package main

import (
	"net/http"
	"time"
)

const dashboardReadHeaderTimeout = 5 * time.Second

func newDashboardHTTPServer(handler http.Handler) *http.Server {
	// ReadHeaderTimeout protects incomplete HTTP/1 requests and is cleared after
	// headers. Unlike ReadTimeout it does not impose a lifetime on HTTP/2 streams.
	return &http.Server{Handler: handler, ReadHeaderTimeout: dashboardReadHeaderTimeout}
}
