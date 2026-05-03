package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/portfolio"
)

const (
	defaultStreamInterval = 3 * time.Second // 3s tick, it is matched with price cache TTL
	minStreamInterval     = 1 * time.Second
)

type Server struct {
	Portfolio *portfolio.Service
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/portfolio", s.handlePortfolio)
	mux.HandleFunc("/api/portfolio/stream", s.handlePortfolioStream)
	mux.HandleFunc("/api/currencies", s.handleCurrencies)
	mux.HandleFunc("/api/exchanges", s.handleExchanges)
	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "backend"})
}

func (s *Server) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	base := strings.ToUpper(r.URL.Query().Get("base"))
	if base == "" {
		base = "INR"
	}
	resp, err := s.Portfolio.Portfolio(r.Context(), base)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handlePortfolioStream is the Server-Sent Events endpoint — a long-lived HTTP response that pushes a fresh
// portfolio snapshot to one connected browser every 3 s

// SSE cases handled -
// 1. Headers before any data write
// 2. Flush after every event
// 3. Stop work when the client leaves
func (s *Server) handlePortfolioStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	base := strings.ToUpper(r.URL.Query().Get("base"))
	if base == "" {
		base = "INR"
	}

	interval := defaultStreamInterval
	if v := r.URL.Query().Get("interval"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= minStreamInterval {
			interval = d
		}
	}

	w.Header().Set("Content-Type", "text/event-stream") //SSE
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// sending portfolio after marshalling
	send := func() bool {
		resp, err := s.Portfolio.Portfolio(r.Context(), base)
		if err != nil {
			payload, _ := json.Marshal(map[string]string{"error": err.Error()})
			if _, werr := fmt.Fprintf(w, "event: app-error\ndata: %s\n\n", payload); werr != nil {
				return false
			}
			flusher.Flush()
			return true
		}
		payload, err := json.Marshal(resp)
		if err != nil {
			return false
		}
		if _, werr := fmt.Fprintf(w, "data: %s\n\n", payload); werr != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	// send immediately on connect, before the ticker starts
	// without this, there will be a loading screen for 3s initially
	if !send() {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {

		// client disconnect case
		case <-r.Context().Done():
			return

		// normal case
		case <-ticker.C:
			if !send() {
				return
			}
		}
	}
}

func (s *Server) handleCurrencies(w http.ResponseWriter, r *http.Request) {
	cs, err := s.Portfolio.SupportedCurrencies(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"currencies": cs})
}

func (s *Server) handleExchanges(w http.ResponseWriter, r *http.Request) {
	es, err := s.Portfolio.Exchanges(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, es)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
