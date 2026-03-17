package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// StatsProvider is a function that returns the current stats.
type StatsProvider func() interface{}

// Server exposes stats via HTTP.
type Server struct {
	port       int
	getStats   StatsProvider
	httpServer *http.Server
}

// New creates a new HTTP server for exposing stats.
func New(port int, getStats StatsProvider) *Server {
	return &Server{
		port:     port,
		getStats: getStats,
	}
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/stats", s.HandleStats)
	mux.HandleFunc("/api/health", s.HandleHealth)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// HandleStats handles GET /api/stats requests.
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	stats := s.getStats()
	if stats == nil {
		http.Error(w, "stats not available", http.StatusServiceUnavailable)
		return
	}

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "encoding error", http.StatusInternalServerError)
		return
	}
}

// HandleHealth handles GET /api/health requests.
func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(`{"status":"ok"}`))
}
