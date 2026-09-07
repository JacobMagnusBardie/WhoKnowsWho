package server

import "net/http"

// NewRouter builds the HTTP handler for the application.
func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)

	return mux
}
