package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

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

func (s *Server) ensureFlusher(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return nil, false
	}
	return flusher, true
}

func getBase(r *http.Request) string {
	base := strings.ToUpper(r.URL.Query().Get("base"))
	if base == "" {
		return "INR"
	}
	return base
}

func (s *Server) getStreamInterval(r *http.Request) time.Duration {
	interval := s.StreamInterval
	if interval < minStreamInterval {
		interval = fallbackStreamInterval
	}

	if v := r.URL.Query().Get("interval"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= minStreamInterval {
			return d
		}
	}

	return interval
}

func (s *Server) makePortfolioSender(
	r *http.Request,
	base string,
	out io.Writer,
	flusher http.Flusher,
	gz *gzip.Writer,
) func() bool {

	return func() bool {
		resp, err := s.Portfolio.Portfolio(r.Context(), base)
		if err != nil {
			return writeEvent(out, flusher, gz, "app-error", map[string]string{
				"error": err.Error(),
			})
		}

		return writeEvent(out, flusher, gz, "", resp)
	}
}

func writeEvent(
	out io.Writer,
	flusher http.Flusher,
	gz *gzip.Writer,
	event string,
	data any,
) bool {
	payload, err := json.Marshal(data)
	if err != nil {
		return false
	}

	var writeErr error
	if event != "" {
		_, writeErr = fmt.Fprintf(out, "event: %s\ndata: %s\n\n", event, payload)
	} else {
		_, writeErr = fmt.Fprintf(out, "data: %s\n\n", payload)
	}

	if writeErr != nil {
		return false
	}

	if gz != nil {
		if err := gz.Flush(); err != nil {
			return false
		}
	}

	flusher.Flush()
	return true
}

func setupStreamWriter(w http.ResponseWriter, r *http.Request) (io.Writer, *gzip.Writer) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")

	var (
		out io.Writer = w
		gz  *gzip.Writer
	)

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		gz, _ = gzip.NewWriterLevel(w, gzip.BestSpeed)
		out = gz
	}

	w.WriteHeader(http.StatusOK)
	return out, gz
}

func closeGzip(gz *gzip.Writer) {
	if gz != nil {
		gz.Close()
	}
}
