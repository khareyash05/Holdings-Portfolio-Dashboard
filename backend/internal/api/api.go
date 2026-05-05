package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/portfolio"
)

const (
	fallbackStreamInterval  = 3 * time.Second // fallback when StreamInternval is 0(introduced for decoupling on interval fetches between backend and exchange service)
	minStreamInterval       = 1 * time.Second // minimum wait for streamInterval
	portfolioRequestTimeout = 5 * time.Second
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
	mux.Handle("/api/portfolio", s.withRateLimiter(s.handlePortfolio))
	mux.Handle("/api/portfolio/stream", s.withRateLimiter(s.handlePortfolioStream))
	mux.Handle("/api/currencies", s.withRateLimiter(s.handleCurrencies))
	mux.Handle("/api/exchanges", s.withRateLimiter(s.handleExchanges))
	return withCORS(mux)
}

func (s *Server) withRateLimiter(h http.HandlerFunc) http.Handler {
	if s.RateLimiter == nil {
		return h
	}
	return s.RateLimiter.Middleware(h)
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

// handlePortfolioStream is the Server-Sent Events endpoint — a long-lived HTTP response that pushes a fresh
// portfolio snapshot to one connected user every 3 s

// SSE cases handled -
// 1. Headers before any data write
// 2. Flush after every event
// 3. Stop work when the client leaves
func (s *Server) handlePortfolioStream(w http.ResponseWriter, r *http.Request) {
	// Ensure the response writer supports streaming (SSE requires flushing)
	flusher, ok := s.ensureFlusher(w)
	if !ok {
		return
	}

	// Extract query params (base currency + stream interval)
	base := getBase(r)
	interval := s.getStreamInterval(r)

	// Setup SSE headers + gzip compression
	// `out` is where we write
	out, gz := setupStreamWriter(w, r)
	defer closeGzip(gz)

	// setup portfolio function, will fetch and write data
	send := s.makePortfolioSender(r, base, out, flusher, gz)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Send first response immediately (avoids initial client-side loading delay)
	if !send() {
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if !send() {
				return
			}
		}
	}
}
