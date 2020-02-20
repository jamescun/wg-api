package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Handler responds to JSON-RPC requests.
type Handler interface {
	ServeJSONRPC(w ResponseWriter, r *Request)
}

// HandlerFunc adapts a function into a handler.
type HandlerFunc func(ResponseWriter, *Request)

// ServeJSONRPC responds to JSON-RPC requests with hf(w, r).
func (hf HandlerFunc) ServeJSONRPC(w ResponseWriter, r *Request) {
	hf(w, r)
}

// Request contains the JSON-RPC paramaters submitted by the client.
type Request struct {
	Version string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id"`

	ctx   context.Context
	raddr string
}

// Context returns the execution context of the request, or the background
// context if one is not set.
func (r *Request) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}

	return r.ctx
}

// RemoteAddr returns the remote ip:port of the client.
func (r *Request) RemoteAddr() string {
	return r.raddr
}

// ResponseWriter marshals the JSON-RPC response to the client.
type ResponseWriter interface {
	// Write marshals anything given to it as the Result of the JSON-RPC
	// interaction. If the given argument implements then Error interface,
	// it will be marshalled as the Error.
	Write(interface{}) error
}

type response struct {
	Version string          `json:"jsonrpc"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
}

func (r *response) Write(res interface{}) error {
	if r.Result != nil && r.Error != nil {
		return fmt.Errorf("response already written")
	}

	if rpcErr, ok := res.(*Error); ok {
		r.Error = rpcErr
	} else if err, ok := res.(error); ok {
		r.Error = InternalError(err.Error(), nil)
	} else {
		r.Result = res
	}

	return nil
}

// ContentType is the MIME Type expected of clients and returned by the server.
const ContentType = "application/json"

// HTTP adapts a JSON-RPC Handler to a HTTP Handler for use in
// HTTP(S) exchanges.
func HTTP(hf Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if hdr := r.Header.Get("Content-Type"); !strings.HasPrefix(hdr, ContentType) {
			http.Error(w, fmt.Sprintf("unknown content type %q", hdr), http.StatusBadRequest)
			return
		}

		req := new(Request)
		err := json.NewDecoder(r.Body).Decode(req)
		if err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}
		req.raddr = r.RemoteAddr

		res := &response{Version: "2.0", ID: req.ID}

		hf.ServeJSONRPC(res, req)

		w.Header().Set("Content-Type", ContentType)
		json.NewEncoder(w).Encode(res)
	})
}

// Error implements a top-level JSON-RPC error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`

	Data interface{} `json:"data,omitempty"`
}

func (e Error) Error() string {
	return fmt.Sprintf("Error(%d): %s", e.Code, e.Message)
}

// ParseError returns a JSON-RPC Parse Error (-32700).
func ParseError(message string, data interface{}) *Error {
	return &Error{Code: -32700, Message: message, Data: data}
}

// InvalidRequest returns a JSON-RPC Invalid Request error (-32600).
func InvalidRequest(message string, data interface{}) *Error {
	return &Error{Code: -32600, Message: message, Data: data}
}

// MethodNotFound returns a JSON-RPC Method Not Found error (-32601).
func MethodNotFound(message string, data interface{}) *Error {
	return &Error{Code: -32601, Message: message, Data: data}
}

// InvalidParams returns a JSON-RPC Invalid Params error (-32602).
func InvalidParams(message string, data interface{}) *Error {
	return &Error{Code: -32602, Message: message, Data: data}
}

// InternalError returns a JSON-RPC Internal Server error (-32603).
func InternalError(message string, data interface{}) *Error {
	return &Error{Code: -32603, Message: message, Data: data}
}

// ServerError returns a JSON-RPC Server Error, which must be given a code
// between -32000 and -32099.
func ServerError(code int, message string, data interface{}) *Error {
	return &Error{Code: code, Message: message, Data: data}
}
