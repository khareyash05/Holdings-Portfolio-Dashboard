package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type forexFile struct {
	Date string             `json:"date"`
	INR  map[string]float64 `json:"inr"`
}

type ratesResponse struct {
	Base      string             `json:"base"`
	AsOf      string             `json:"asOf"`
	Rates     map[string]float64 `json:"rates"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

type convertResponse struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
	Result float64 `json:"result"`
	Rate   float64 `json:"rate"`
}

var forex forexFile

func loadForex(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &forex)
}

func ratesINR() map[string]float64 {
	out := make(map[string]float64, len(forex.INR))
	for k, v := range forex.INR {
		out[strings.ToUpper(k)] = v
	}
	return out
}

func convertAmount(from, to string, amount float64) (float64, float64, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	rates := ratesINR()
	rf, ok := rates[from]
	if !ok {
		return 0, 0, fmt.Errorf("unknown currency: %s", from)
	}
	rt, ok := rates[to]
	if !ok {
		return 0, 0, fmt.Errorf("unknown currency: %s", to)
	}
	rate := rt / rf
	return amount * rate, rate, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func handleRates(w http.ResponseWriter, r *http.Request) {
	base := strings.ToUpper(r.URL.Query().Get("base"))
	if base == "" {
		base = "INR"
	}
	rates := ratesINR()
	rb, ok := rates[base]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown base currency"})
		return
	}
	out := make(map[string]float64, len(rates))
	for k, v := range rates {
		out[k] = v / rb
	}
	writeJSON(w, http.StatusOK, ratesResponse{
		Base:      base,
		AsOf:      forex.Date,
		Rates:     out,
		UpdatedAt: time.Now().UTC(),
	})
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := q.Get("from")
	to := q.Get("to")
	amt, err := strconv.ParseFloat(q.Get("amount"), 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid amount"})
		return
	}
	res, rate, err := convertAmount(from, to, amt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, convertResponse{
		From: strings.ToUpper(from), To: strings.ToUpper(to),
		Amount: amt, Result: res, Rate: rate,
	})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "forex"})
}

func main() {
	if err := loadForex(envOr("FOREX_DATA", "/app/data/forex.json")); err != nil {
		log.Fatalf("load forex: %v", err)
	}
	addr := envOr("LISTEN_ADDR", ":8081")
	mux := http.NewServeMux()
	mux.HandleFunc("/rates", handleRates)
	mux.HandleFunc("/rates/convert", handleConvert)
	mux.HandleFunc("/health", handleHealth)

	log.Printf("forex-service listening on %s (loaded %d rates)", addr, len(forex.INR))
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
