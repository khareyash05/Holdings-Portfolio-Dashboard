package portfolio

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/cache"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/clients"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/models"
)

const (
	holdingsCacheTTL  = 30 * time.Second
	exchangesCacheTTL = 5 * time.Minute
)

type Service struct {
	DB            *gorm.DB
	Forex         *clients.ForexClient
	Exchange      *clients.ExchangeClient
	PriceCache    *cache.Redis[map[string]float64]
	LastGoodPrice *cache.Redis[map[string]float64]
	ForexCache    *cache.Redis[*clients.RatesResponse]
	LastGoodForex *cache.Redis[*clients.RatesResponse] // used when the hot cache misses and upstream fails (returns stale flag)

	// in-process holdings cache.for now, these are global , in future will be pinned by userID
	holdingsMu sync.RWMutex
	holdings   []holdingRow
	holdingsAt time.Time

	// exchanges list cache with TTL
	exchangesMu  sync.RWMutex
	exchangesAll []models.Exchange
	exchangesAt  time.Time
}

func New(db *gorm.DB, fx *clients.ForexClient, ex *clients.ExchangeClient,
	priceCache *cache.Redis[map[string]float64], lastGoodPrice *cache.Redis[map[string]float64],
	forexCache *cache.Redis[*clients.RatesResponse], lastGoodForex *cache.Redis[*clients.RatesResponse]) *Service {
	return &Service{
		DB:            db,
		Forex:         fx,
		Exchange:      ex,
		PriceCache:    priceCache,
		LastGoodPrice: lastGoodPrice,
		ForexCache:    forexCache,
		LastGoodForex: lastGoodForex,
	}
}

type holdingRow struct {
	Ticker      string  `gorm:"column:ticker"`
	Name        string  `gorm:"column:name"`
	Exchange    string  `gorm:"column:exchange_short_name"`
	Country     string  `gorm:"column:country"`
	Currency    string  `gorm:"column:currency"`
	Sector      string  `gorm:"column:sector"`
	Quantity    float64 `gorm:"column:quantity"`
	BoughtPrice float64 `gorm:"column:bought_price_local"`
}

func (s *Service) loadHoldings(ctx context.Context) ([]holdingRow, error) {

	// fast check on holdingCache first
	s.holdingsMu.RLock()
	if s.holdings != nil && time.Since(s.holdingsAt) < holdingsCacheTTL {
		out := make([]holdingRow, len(s.holdings))
		copy(out, s.holdings)
		s.holdingsMu.RUnlock()
		return out, nil
	}
	s.holdingsMu.RUnlock()

	var rows []holdingRow
	err := s.DB.WithContext(ctx).
		Table("holdings").
		Select("s.ticker, s.name, s.exchange_short_name, s.country, s.currency, s.sector, holdings.quantity, holdings.bought_price_local").
		Joins("JOIN stocks s ON s.ticker = holdings.ticker").
		Order("s.exchange_short_name, s.ticker").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	s.holdingsMu.Lock()
	s.holdings = rows
	s.holdingsAt = time.Now()
	s.holdingsMu.Unlock()

	out := make([]holdingRow, len(rows))
	copy(out, rows)
	return out, nil
}

func (s *Service) Exchanges(ctx context.Context) ([]models.Exchange, error) {

	// first check from hot cache,99% exchanges will be returned from here
	s.exchangesMu.RLock()
	if len(s.exchangesAll) > 0 && time.Since(s.exchangesAt) < exchangesCacheTTL {
		out := make([]models.Exchange, len(s.exchangesAll))
		copy(out, s.exchangesAll)
		s.exchangesMu.RUnlock()
		return out, nil
	}
	s.exchangesMu.RUnlock()

	// failsafe check here
	// if not found in cache, there might be a case where during the previous reading, a new field was added, we again need to check
	// also update the cache here
	// this helps in cases where the if 100 failed first step, if not this step, all 100 would call DB
	s.exchangesMu.Lock()
	defer s.exchangesMu.Unlock()
	if len(s.exchangesAll) > 0 && time.Since(s.exchangesAt) < exchangesCacheTTL {
		out := make([]models.Exchange, len(s.exchangesAll))
		copy(out, s.exchangesAll)
		return out, nil
	}

	// if again not found, query the db
	var fresh []models.Exchange
	if err := s.DB.WithContext(ctx).Order("short_name").Find(&fresh).Error; err != nil {
		return nil, err
	}
	s.exchangesAll = fresh
	s.exchangesAt = time.Now()
	out := make([]models.Exchange, len(fresh))
	copy(out, fresh)
	return out, nil
}

func (s *Service) snapshotsFor(ctx context.Context, exchange string) (map[string]float64, bool, error) {
	key := strings.ToUpper(exchange)

	if v, ok, err := s.PriceCache.Get(ctx, key); err == nil && ok {
		return v, false, nil
	}

	snaps, ferr := s.Exchange.Snapshots(ctx, exchange)
	if ferr != nil {
		// the exchange service wasn't connected, fallback to the last price
		if v, ok, gerr := s.LastGoodPrice.Get(ctx, key); gerr == nil && ok {
			return v, true, nil
		}
		return nil, true, ferr
	}
	m := make(map[string]float64, len(snaps))
	for _, q := range snaps {
		m[q.Ticker] = q.Price
	}

	// update both the caches(price cache -> exchange up, LastGoodPrice -> exchange down)
	if serr := s.PriceCache.Set(ctx, key, m); serr != nil {
		log.Printf("price cache set %s: %v", key, serr)
	}
	if serr := s.LastGoodPrice.Set(ctx, key, m); serr != nil {
		log.Printf("lastgood cache set %s: %v", key, serr)
	}
	return m, false, nil
}

func (s *Service) ratesINR(ctx context.Context) (*clients.RatesResponse, bool, error) {
	if v, ok, err := s.ForexCache.Get(ctx, "INR"); err == nil && ok {
		return v, false, nil
	}

	// same flow as exchange service -> check cache -> if not found/error -> return lastgoodprice + stale flag
	v, ferr := s.Forex.Rates(ctx, "INR")
	if ferr != nil {
		if g, ok, gerr := s.LastGoodForex.Get(ctx, "INR"); gerr == nil && ok {
			return g, true, nil
		}
		return nil, true, ferr
	}
	if serr := s.ForexCache.Set(ctx, "INR", v); serr != nil {
		log.Printf("forex cache set: %v", serr)
	}
	if serr := s.LastGoodForex.Set(ctx, "INR", v); serr != nil {
		log.Printf("lastgood-forex set: %v", serr)
	}
	return v, false, nil
}

func (s *Service) SupportedCurrencies(ctx context.Context) ([]string, error) {
	r, _, err := s.ratesINR(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(r.Rates))
	for k := range r.Rates {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

func (s *Service) Portfolio(ctx context.Context, baseCurrency string) (*models.PortfolioResponse, error) {
	baseCurrency = strings.ToUpper(strings.TrimSpace(baseCurrency))
	if baseCurrency == "" {
		baseCurrency = "INR"
	}

	// fetch base rates from cache
	rates, ratesStale, err := s.ratesINR(ctx)
	if err != nil {
		return nil, fmt.Errorf("forex: %w", err)
	}
	if _, ok := rates.Rates[baseCurrency]; !ok {
		return nil, fmt.Errorf("unsupported base currency: %s", baseCurrency)
	}

	holdings, err := s.loadHoldings(ctx)
	if err != nil {
		return nil, fmt.Errorf("load holdings: %w", err)
	}

	// It takes ~100 holdings spread across ~10 exchanges, fetches all 10 exchanges prices in parallel, and assembles the results into the map
	exchanges := map[string]struct{}{}
	for _, h := range holdings {
		exchanges[h.Exchange] = struct{}{}
	}

	type snapshotsResult struct {
		ex     string
		prices map[string]float64
		stale  bool
		err    error
	}
	ch := make(chan snapshotsResult, len(exchanges)) // for each exchange we are getting snapshots, thus should we independent of exchange , thus we are spawning goroutines here
	for ex := range exchanges {
		go func() {
			p, stale, err := s.snapshotsFor(ctx, ex) // get all the price snapshorts for stocks of an exchange from exchange-service(also cache aware)
			ch <- snapshotsResult{ex: ex, prices: p, stale: stale, err: err}
		}()
	}
	priceMap := map[string]map[string]float64{}
	staleEx := map[string]bool{}
	for i := 0; i < len(exchanges); i++ {
		r := <-ch
		if r.err != nil {
			staleEx[r.ex] = true
			continue
		}
		priceMap[r.ex] = r.prices
		if r.stale {
			staleEx[r.ex] = true
		}
	}

	views := make([]models.HoldingView, 0, len(holdings))
	var totalNet, totalInvested float64
	anyStale := ratesStale
	
	for _, h := range holdings {
		prices, hasPrices := priceMap[h.Exchange]
		curLocal, hasTicker := 0.0, false
		if hasPrices {
			curLocal, hasTicker = prices[h.Ticker], true
		}
		stale := staleEx[h.Exchange] || ratesStale

		// if it doesnt have price or ticker, falllback to the current price
		if !hasTicker {
			curLocal = h.BoughtPrice
			stale = true
		}
		
		localToBase, err := convertFactor(rates.Rates, h.Currency, baseCurrency) // comvert the current into the currency of the exchange
		if err != nil {
			return nil, err
		}
		
		invested := h.Quantity * h.BoughtPrice * localToBase
		networth := h.Quantity * curLocal * localToBase
		gains := networth - invested
		var gainsPct float64
		if invested > 0 {
			gainsPct = gains / invested * 100
		}
		// return per holding(for each and every holding, used in below section of ui where we show all holdings)
		views = append(views, models.HoldingView{
			Ticker:            h.Ticker,
			Name:              h.Name,
			Exchange:          h.Exchange,
			Country:           h.Country,
			Sector:            h.Sector,
			LocalCurrency:     h.Currency,
			Quantity:          round4(h.Quantity),
			CurrentPriceLocal: round2(curLocal),
			BoughtPriceLocal:  round2(h.BoughtPrice),
			CurrentPriceBase:  round4(curLocal * localToBase),
			InvestedBase:      round2(invested),
			NetWorthBase:      round2(networth),
			UnrealizedGains:   round2(gains),
			GainsPct:          round2(gainsPct),
			Stale:             stale,
		})
		totalInvested += invested
		totalNet += networth
		if stale {
			anyStale = true
		}
	}

	exchangesList, err := s.Exchanges(ctx)
	if err != nil {
		return nil, err
	}

	gains := totalNet - totalInvested
	var pct float64
	if totalInvested > 0 {
		pct = gains / totalInvested * 100
	}
	// return portfolio summary ,the overall summary
	return &models.PortfolioResponse{
		BaseCurrency: baseCurrency,
		AsOf:         time.Now().UTC(),
		Summary: models.Summary{
			BaseCurrency:    baseCurrency,
			NetWorth:        round2(totalNet),
			Invested:        round2(totalInvested),
			UnrealizedGains: round2(gains),
			GainsPct:        round2(pct),
			Stale:           anyStale,
		},
		Holdings:  views,
		Exchanges: exchangesList,
	}, nil
}
