package portfolio

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/cache"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/clients"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/models"
)

type Service struct {
	DB           *gorm.DB
	Forex        *clients.ForexClient
	Exchange     *clients.ExchangeClient
	PriceCache   *cache.Redis[map[string]float64]
	ForexCache   *cache.Redis[*clients.RatesResponse]
	exchangesMu  sync.RWMutex
	exchangesAll []models.Exchange
}

func New(db *gorm.DB, fx *clients.ForexClient, ex *clients.ExchangeClient,
	priceCache *cache.Redis[map[string]float64], forexCache *cache.Redis[*clients.RatesResponse]) *Service {
	return &Service{
		DB:         db,
		Forex:      fx,
		Exchange:   ex,
		PriceCache: priceCache,
		ForexCache: forexCache,
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
	var out []holdingRow
	err := s.DB.WithContext(ctx).
		Table("holdings").
		Select("s.ticker, s.name, s.exchange_short_name, s.country, s.currency, s.sector, holdings.quantity, holdings.bought_price_local").
		Joins("JOIN stocks s ON s.ticker = holdings.ticker").
		Order("s.exchange_short_name, s.ticker").
		Scan(&out).Error
	return out, err
}

func (s *Service) Exchanges(ctx context.Context) ([]models.Exchange, error) {
	// list all exchanges in the cache
	s.exchangesMu.RLock()
	if len(s.exchangesAll) > 0 {
		out := make([]models.Exchange, len(s.exchangesAll))
		copy(out, s.exchangesAll)
		s.exchangesMu.RUnlock()
		return out, nil
	}
	s.exchangesMu.RUnlock()

	// also get from db if some new exchange was added and sync it to the cachemap
	var out []models.Exchange
	if err := s.DB.WithContext(ctx).Order("short_name").Find(&out).Error; err != nil {
		return nil, err
	}
	s.exchangesMu.Lock()
	s.exchangesAll = out
	s.exchangesMu.Unlock()
	return out, nil
}

func (s *Service) snapshotsFor(ctx context.Context, exchange string) (map[string]float64, error) {
	return s.PriceCache.GetOrLoad(ctx, strings.ToUpper(exchange), func() (map[string]float64, error) {
		snaps, err := s.Exchange.Snapshots(ctx, exchange)
		if err != nil {
			return nil, err
		}
		m := make(map[string]float64, len(snaps))
		for _, q := range snaps {
			m[q.Ticker] = q.Price
		}
		return m, nil
	})
}

func (s *Service) ratesINR(ctx context.Context) (*clients.RatesResponse, error) {
	return s.ForexCache.GetOrLoad(ctx, "INR", func() (*clients.RatesResponse, error) {
		return s.Forex.Rates(ctx, "INR")
	})
}

func (s *Service) SupportedCurrencies(ctx context.Context) ([]string, error) {
	r, err := s.ratesINR(ctx)
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
	rates, err := s.ratesINR(ctx)
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
		err    error
	}
	ch := make(chan snapshotsResult, len(exchanges)) // for each exchange we are getting snapshots, thus should we independent of exchange , thus we are spawning goroutines here
	for ex := range exchanges {
		go func() {
			p, err := s.snapshotsFor(ctx, ex) // get all the price snapshorts for stocks of an exchange from exchange-service(also cache aware)
			ch <- snapshotsResult{ex: ex, prices: p, err: err}
		}()
	}
	// now extract prices, example priceMap["NYSE"]["AAPL"]
	priceMap := map[string]map[string]float64{}
	for i := 0; i < len(exchanges); i++ {
		r := <-ch
		if r.err != nil {
			return nil, fmt.Errorf("exchange %s: %w", r.ex, r.err)
		}
		priceMap[r.ex] = r.prices
	}

	views := make([]models.HoldingView, 0, len(holdings))
	var totalNet, totalInvested float64
	for _, h := range holdings {
		curLocal := priceMap[h.Exchange][h.Ticker]
		localToBase, err := convertFactor(rates.Rates, h.Currency, baseCurrency)
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
		})
		totalInvested += invested
		totalNet += networth
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
	return &models.PortfolioResponse{
		BaseCurrency: baseCurrency,
		AsOf:         time.Now().UTC(),
		Summary: models.Summary{
			BaseCurrency:    baseCurrency,
			NetWorth:        round2(totalNet),
			Invested:        round2(totalInvested),
			UnrealizedGains: round2(gains),
			GainsPct:        round2(pct),
		},
		Holdings:  views,
		Exchanges: exchangesList,
	}, nil
}

func convertFactor(ratesINRBase map[string]float64, from, to string) (float64, error) {
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)
	if from == to {
		return 1, nil
	}
	rFrom, ok := ratesINRBase[from]
	if !ok {
		return 0, fmt.Errorf("unsupported currency: %s", from)
	}
	rTo, ok := ratesINRBase[to]
	if !ok {
		return 0, fmt.Errorf("unsupported currency: %s", to)
	}
	if rFrom == 0 {
		return 0, fmt.Errorf("zero rate for %s", from)
	}
	return rTo / rFrom, nil
}

func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}

func round4(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*10000) / 10000
}
