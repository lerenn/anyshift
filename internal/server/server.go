package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/challenge-github-events/internal/store"
)

// Server serves /health and /stats. Depends only on Store interface.
type Server struct {
	store store.Store
	http  *http.Server
}

// NewServer returns an HTTP server that uses the given Store.
func NewServer(addr string, s store.Store) *Server {
	mux := http.NewServeMux()
	srv := &Server{store: s}
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/stats", srv.handleStats)
	srv.http = &http.Server{Addr: addr, Handler: mux}
	return srv
}

// Start starts the HTTP server (blocking).
func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		slog.Debug("health check method not allowed", "method", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store != nil {
		if err := s.store.Ping(r.Context()); err != nil {
			slog.Warn("health check failed", "err", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "error": err.Error()})
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		slog.Debug("stats method not allowed", "method", r.Method)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	netLines, err := s.store.GlobalNetLines(r.Context())
	if err != nil {
		slog.Error("stats: global net lines", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	eventsCount, err := s.store.EventsSeenCount(r.Context())
	if err != nil {
		slog.Error("stats: events count", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	slog.Debug("stats served", "net_lines", netLines, "events_count", eventsCount)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"global_net_lines_current": netLines,
		"events_seen_since_start":  eventsCount,
	})
}
