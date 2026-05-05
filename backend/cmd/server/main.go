package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/api"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/cache"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/clients"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/config"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/db"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/portfolio"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/seed"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	rootCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// connect to postgres
	gdb, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	if sdb, derr := gdb.DB(); derr == nil {
		defer sdb.Close()
	}

	// db migration(tables setup)
	if err := db.Migrate(gdb); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Printf("db migrated")

	// setup clients to connect to other services
	exchangeClient := clients.NewExchangeClient(cfg.ExchangeBaseURL)
	forexClient := clients.NewForexClient(cfg.ForexBaseURL)

	// seeding bought prices for holdings
	seeder := &seed.Seeder{
		DB:            gdb,
		StocksPath:    cfg.StocksDataPath,
		ExchangesPath: cfg.ExchangesPath,
	}
	if err := seeder.Run(rootCtx); err != nil {
		log.Fatalf("seed: %v", err)
	}

	if cfg.RedisURL == "" {
		log.Fatal("REDIS_URL is required")
	}
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("parse redis url: %v", err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.Ping(rootCtx).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer rdb.Close()
	log.Printf("cache: redis at %s", cfg.RedisURL)

	priceCache := cache.NewRedis[map[string]float64](rdb, "price", cfg.PriceCacheTTL)
	lastGoodPrice := cache.NewRedis[map[string]float64](rdb, "lastgood-price", cfg.LastGoodTTL)
	forexCache := cache.NewRedis[*clients.RatesResponse](rdb, "forex", cfg.ForexCacheTTL)
	lastGoodForex := cache.NewRedis[*clients.RatesResponse](rdb, "lastgood-forex", cfg.LastGoodTTL)

	svc := portfolio.New(gdb, forexClient, exchangeClient, priceCache, lastGoodPrice, forexCache, lastGoodForex)

	// setup rate limiter
	ipLimiter := api.NewIPLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	srv := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: (&api.Server{
			Portfolio:      svc,
			StreamInterval: cfg.PriceCacheTTL,
			RateLimiter:    ipLimiter,
		}).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("backend listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-rootCtx.Done()
	log.Printf("shutting down...")

	// ensuring graceful shutdown
	shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()
	_ = srv.Shutdown(shutdownCtx)
	os.Exit(0)
}
