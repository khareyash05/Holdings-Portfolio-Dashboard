package api

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/portfolio"
)

const (
	fallbackStreamInterval  = 3 * time.Second // fallback when StreamInternval is 0(introduced for decoupling on interval fetches between backend and exchange service)
	minStreamInterval       = 1 * time.Second // minimum wait for streamInterval
	portfolioRequestTimeout = 5 * time.Second // bounds the non-streaming /api/portfolio call , safe check to not pin a goroutine forever
)

type Server struct {
	Portfolio      *portfolio.Service
	StreamInterval time.Duration
	RateLimiter    *IPLimiter
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/health", s.handleHealth)

	rateLimiterMiddleware := func(h http.HandlerFunc) http.Handler {
		if s.RateLimiter == nil {
			return h
		}
		return s.RateLimiter.Middleware(h)
	}
	mux.Handle("/api/portfolio", rateLimiterMiddleware(s.handlePortfolio))
	mux.Handle("/api/portfolio/stream", rateLimiterMiddleware(s.handlePortfolioStream))
	mux.Handle("/api/currencies", rateLimiterMiddleware(s.handleCurrencies))
	mux.Handle("/api/exchanges", rateLimiterMiddleware(s.handleExchanges))
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
	ctx, cancel := context.WithTimeout(r.Context(), portfolioRequestTimeout)
	defer cancel()
	resp, err := s.Portfolio.Portfolio(ctx, base)
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

	interval := s.StreamInterval
	if interval < minStreamInterval {
		interval = fallbackStreamInterval
	}
	if v := r.URL.Query().Get("interval"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= minStreamInterval {
			interval = d
		}
	}

	w.Header().Set("Content-Type", "text/event-stream") //SSE
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")

	// introducing a gzip compression on data sent on the SSE stream(tested --> resulted in 3.1x compression, when scaling this results in about 3x lower bills)
	// ideally ,we should do it on some CDN layer, but this is a kind of fallback
	var (
		out io.Writer = w
		gz  *gzip.Writer
	)
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		gz, _ = gzip.NewWriterLevel(w, gzip.BestSpeed)
		defer gz.Close()
		out = gz
	}
	w.WriteHeader(http.StatusOK)

	// sending portfolio after marshalling
	send := func() bool {
		resp, err := s.Portfolio.Portfolio(r.Context(), base)
		if err != nil {
			payload, _ := json.Marshal(map[string]string{"error": err.Error()})
			if _, werr := fmt.Fprintf(out, "event: app-error\ndata: %s\n\n", payload); werr != nil {
				return false
			}
			if gz != nil {
				if ferr := gz.Flush(); ferr != nil {
					return false
				}
			}
			flusher.Flush()
			return true
		}
		payload, err := json.Marshal(resp)
		if err != nil {
			return false
		}
		if _, werr := fmt.Fprintf(out, "data: %s\n\n", payload); werr != nil {
			return false
		}
		if gz != nil {
			if ferr := gz.Flush(); ferr != nil {
				return false
			}
		}
		flusher.Flush()
		return true
	}

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
