package main

import (
	"encoding/json"
	"hash/fnv"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type Stock struct {
	Ticker   string `json:"ticker"`
	Exchange string `json:"exchange"`
	Currency string `json:"currency"`
}

type Exchange struct {
	ShortName string `json:"shortName"`
	Currency  string `json:"currency"`
}

type PriceSnapshot struct {
	Ticker   string    `json:"ticker"`
	Price    float64   `json:"price"`
	Currency string    `json:"currency"`
	Exchange string    `json:"exchange"`
	AsOf     time.Time `json:"asOf"`
}

type snapshotsResponse struct {
	Exchange  string          `json:"exchange"`
	Currency  string          `json:"currency"`
	Snapshots []PriceSnapshot `json:"snapshots"`
}

var (
	stocks    []Stock
	exchanges []Exchange
)

func loadJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// computes current price
// other way could have been calling the db for the bought price
// but then this service would have been dependent on backend service, which it ideally shouldn't be
// as its responsibility is to send current price
// therefore made this independent by creating a hash per stock and then calculating final price from there
// the bought price and current price has no relations with it right now, which is the realistic condition of markets
func priceFor(symbol, currency string) float64 {
	// hash the ticker, this remains same for a stock here
	h := fnv.New32a()
	h.Write([]byte(symbol))

	// pick a base price in the currency-appropriate band
	base := basePrice(currency, h.Sum32())

	centered := rand.Float64() - 0.5 // uniform random shifted to [-0.5, 0.5)
	wobble := centered * 0.04        // scale to +-2% wobble
	multiplier := 1 + wobble         // muliplier to 1 for factor

	return base * multiplier
}

// base price ranges across exchanges
func basePrice(currency string, seed uint32) float64 {
	switch strings.ToUpper(currency) {
	case "INR":
		return 500 + float64(seed%4500)
	case "JPY":
		return 1000 + float64(seed%9000)
	case "HKD":
		return 50 + float64(seed%750)
	default:
		return 50 + float64(seed%450)
	}
}

func snapshotsForExchange(short string) ([]PriceSnapshot, string) {
	short = strings.ToUpper(short)
	now := time.Now().UTC()
	out := []PriceSnapshot{}
	currency := ""
	for _, s := range stocks {
		if !strings.EqualFold(s.Exchange, short) {
			continue
		}
		if currency == "" {
			currency = s.Currency
		}
		out = append(out, PriceSnapshot{
			Ticker:   s.Ticker,                       // the unique code per stock
			Price:    priceFor(s.Ticker, s.Currency), // the base price
			Currency: s.Currency,
			Exchange: s.Exchange,
			AsOf:     now,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ticker < out[j].Ticker })
	return out, currency
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func handleExchanges(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, exchanges)
}

func handleSnapshots(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}
	short := parts[1]
	snaps, currency := snapshotsForExchange(short)
	if len(snaps) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "exchange not found or has no listings"})
		return
	}
	writeJSON(w, http.StatusOK, snapshotsResponse{Exchange: strings.ToUpper(short), Currency: currency, Snapshots: snaps})
}

func handleSingleSnapshot(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		http.NotFound(w, r)
		return
	}
	short := strings.ToUpper(parts[1]) // the short code for exchange
	ticker := parts[3]                 // for the stock
	for _, s := range stocks {
		if strings.EqualFold(s.Exchange, short) && strings.EqualFold(s.Ticker, ticker) {
			now := time.Now().UTC()
			writeJSON(w, http.StatusOK, PriceSnapshot{
				Ticker: s.Ticker, Price: priceFor(s.Ticker, s.Currency), Currency: s.Currency,
				Exchange: s.Exchange, AsOf: now,
			})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "ticker not found"})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "exchange", "stocks": len(stocks), "exchanges": len(exchanges)})
}

func handleStocks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, stocks)
}

func main() {
	stocksPath := envOr("STOCKS_DATA", "/app/data/stocks.json")
	exchangesPath := envOr("EXCHANGES_DATA", "/app/data/exchanges.json")
	if err := loadJSON(stocksPath, &stocks); err != nil {
		log.Fatalf("load stocks: %v", err)
	}
	if err := loadJSON(exchangesPath, &exchanges); err != nil {
		log.Fatalf("load exchanges: %v", err)
	}
	addr := envOr("LISTEN_ADDR", ":8082")
	mux := http.NewServeMux()
	mux.HandleFunc("/exchanges", handleExchanges)
	mux.HandleFunc("/stocks", handleStocks)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/exchange/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		switch {
		case len(parts) == 3 && parts[2] == "snapshots":
			handleSnapshots(w, r)
		case len(parts) == 4 && parts[2] == "snapshot":
			handleSingleSnapshot(w, r)
		default:
			http.NotFound(w, r)
		}
	})
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
