package config

import (
	"os"
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
