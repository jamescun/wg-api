package server

import (
	"log"
	"net/http"
	"time"

	"github.com/jamescun/wireguard-api/server/jsonrpc"
)

// PreventReferer blocks any request that contains a Referer or Origin header,
// as this would indicate a web browser is submitting the request and this
// server should NOT be directly accessible that way.
func PreventReferer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if headersExist(r.Header, "Referer", "Origin") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func headersExist(h http.Header, keys ...string) bool {
	for _, key := range keys {
		if _, ok := h[key]; ok {
			return true
		}
	}

	return false
}

// Logger logs JSON-RPC requests.
func Logger(next jsonrpc.Handler) jsonrpc.Handler {
	return jsonrpc.HandlerFunc(func(w jsonrpc.ResponseWriter, r *jsonrpc.Request) {
		t1 := time.Now()
		next.ServeJSONRPC(w, r)
		t2 := time.Now()

		log.Printf("info: request: method=%q remote_addr=%s duration=%s\n", r.Method, r.RemoteAddr(), t2.Sub(t1))
	})
}
