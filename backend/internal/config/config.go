package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr      string
	DatabaseURL     string
	ForexBaseURL    string
	ExchangeBaseURL string
	RedisURL        string
	PriceCacheTTL   time.Duration
	LastGoodTTL     time.Duration
	ForexCacheTTL   time.Duration
	StocksDataPath  string
	ExchangesPath   string
	SeedSalt        string
	// introducing a token bucket algorithm for rate limiting
	// the tokens are filled after a specific interval
	// each token is consumed when a request is sent
	RateLimitRPS   float64 // rate of refilling tokens per second
	RateLimitBurst int     // number of tokens in the bucket
}

func Load() Config {
	return Config{
		ListenAddr:      envOr("LISTEN_ADDR", ":8080"),
		DatabaseURL:     envOr("DATABASE_URL", "postgres://paasa:paasa@db:5432/paasa?sslmode=disable"),
		ForexBaseURL:    envOr("FOREX_BASE_URL", "http://forex-service:8081"),
		ExchangeBaseURL: envOr("EXCHANGE_BASE_URL", "http://exchange-service:8082"),
		RedisURL:        envOr("REDIS_URL", ""),
		PriceCacheTTL:   envDuration("PRICE_CACHE_TTL", 3*time.Second),
		LastGoodTTL:     envDuration("LAST_GOOD_TTL", 24*time.Hour),
		ForexCacheTTL:   envDuration("FOREX_CACHE_TTL", 60*time.Second),
		StocksDataPath:  envOr("STOCKS_DATA", "/app/data/stocks.json"),
		ExchangesPath:   envOr("EXCHANGES_DATA", "/app/data/exchanges.json"),
		SeedSalt:        envOr("SEED_SALT", "paasa"),
		RateLimitRPS:    envFloat("RATE_LIMIT_RPS", 10),
		RateLimitBurst:  envInt("RATE_LIMIT_BURST", 20),
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envDuration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func envFloat(k string, def float64) float64 {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func envInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
