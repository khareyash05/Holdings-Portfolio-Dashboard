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
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	rootCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	gdb, err := connectWithRetry(rootCtx, cfg.DatabaseURL, 30)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	if sdb, derr := gdb.DB(); derr == nil {
		defer sdb.Close()
	}

	if err := db.Migrate(gdb); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Printf("db migrated")

	exchangeClient := clients.NewExchangeClient(cfg.ExchangeBaseURL)
	forexClient := clients.NewForexClient(cfg.ForexBaseURL)

	if err := waitForService(rootCtx, cfg.ExchangeBaseURL+"/health", 30); err != nil {
		log.Fatalf("exchange-service unreachable: %v", err)
	}
	if err := waitForService(rootCtx, cfg.ForexBaseURL+"/health", 30); err != nil {
		log.Fatalf("forex-service unreachable: %v", err)
	}

	seeder := &seed.Seeder{
		DB:            gdb,
		StocksPath:    cfg.StocksDataPath,
		ExchangesPath: cfg.ExchangesPath,
		Exchange:      exchangeClient,
		Salt:          cfg.SeedSalt,
	}
	if err := seeder.Run(rootCtx); err != nil {
		log.Fatalf("seed: %v", err)
	}

	if cfg.RedisURL == "" {
		log.Fatal("REDIS_URL is required")
	}
	rdb, err := connectRedisWithRetry(rootCtx, cfg.RedisURL, 15)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer rdb.Close()
	log.Printf("cache: redis at %s", cfg.RedisURL)
	priceCache := cache.NewRedis[map[string]float64](rdb, "price", cfg.PriceCacheTTL)
	lastGoodPrice := cache.NewRedis[map[string]float64](rdb, "lastgood-price", cfg.LastGoodTTL)
	forexCache := cache.NewRedis[*clients.RatesResponse](rdb, "forex", cfg.ForexCacheTTL)
	svc := portfolio.New(gdb, forexClient, exchangeClient, priceCache, lastGoodPrice, forexCache)
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           (&api.Server{Portfolio: svc}).Routes(),
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
	shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()
	_ = srv.Shutdown(shutdownCtx)
	os.Exit(0)
}

func connectWithRetry(ctx context.Context, url string, attempts int) (*gorm.DB, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		gdb, err := db.Connect(url)
		if err == nil {
			return gdb, nil
		}
		lastErr = err
		log.Printf("db not ready (%d/%d): %v", i+1, attempts, err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, lastErr
}

func connectRedisWithRetry(ctx context.Context, url string, attempts int) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	var lastErr error
	for i := 0; i < attempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := client.Ping(pingCtx).Err()
		cancel()
		if err == nil {
			return client, nil
		}
		lastErr = err
		log.Printf("redis not ready (%d/%d): %v", i+1, attempts, err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	_ = client.Close()
	return nil, lastErr
}

func waitForService(ctx context.Context, url string, attempts int) error {
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for i := 0; i < attempts; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = err
		} else {
			lastErr = err
		}
		log.Printf("waiting for %s (%d/%d)", url, i+1, attempts)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return lastErr
}
