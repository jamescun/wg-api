package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jamescun/wg-api/server/jsonrpc"
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

// AuthTokens only allows a request to continue if one of the pre-configured
// tokens is provided by the client in the Authorization header, otherwise
// a HTTP 403 Forbidden is returned and the request terminated.
func AuthTokens(tokens ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Token "))

			if !stringInSlice(token, tokens) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func stringInSlice(s string, vv []string) bool {
	for _, v := range vv {
		if v == s {
			return true
		}
	}

	return false
}
