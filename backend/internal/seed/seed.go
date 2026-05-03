package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"os"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/clients"
	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/models"
)

type Seeder struct {
	DB            *gorm.DB
	StocksPath    string
	ExchangesPath string
	Exchange      *clients.ExchangeClient
	Salt          string
}

type stockFile struct {
	Ticker   string  `json:"ticker"`
	Name     string  `json:"name"`
	Exchange string  `json:"exchange"`
	Country  string  `json:"country"`
	Currency string  `json:"currency"`
	Sector   string  `json:"sector"`
	Volume   float64 `json:"volume"`
}

type exchangeFile struct {
	ShortName string `json:"shortName"`
	Currency  string `json:"currency"`
}

func loadJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (s *Seeder) Run(ctx context.Context) error {
	db := s.DB.WithContext(ctx)

	// load exchanges
	var exchangeCount int64
	if err := db.Model(&models.Exchange{}).Count(&exchangeCount).Error; err != nil {
		return err
	}
	if exchangeCount == 0 {
		var exchanges []exchangeFile
		if err := loadJSON(s.ExchangesPath, &exchanges); err != nil {
			return fmt.Errorf("load exchanges: %w", err)
		}
		rows := make([]models.Exchange, 0, len(exchanges))
		for _, e := range exchanges {
			rows = append(rows, models.Exchange{ShortName: e.ShortName, Currency: e.Currency})
		}
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
			return err
		}
		log.Printf("seed: inserted %d exchanges", len(exchanges))
	}

	// load stocks
	var stockCount int64
	if err := db.Model(&models.Stock{}).Count(&stockCount).Error; err != nil {
		return err
	}
	var stocks []stockFile
	if stockCount == 0 {
		if err := loadJSON(s.StocksPath, &stocks); err != nil {
			return fmt.Errorf("load stocks: %w", err)
		}
		rows := make([]models.Stock, 0, len(stocks))
		for _, st := range stocks {
			rows = append(rows, models.Stock{
				Ticker:   st.Ticker,
				Name:     st.Name,
				Exchange: st.Exchange,
				Country:  st.Country,
				Currency: st.Currency,
				Sector:   st.Sector,
			})
		}
		if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error; err != nil {
			return err
		}
		log.Printf("seed: inserted %d stocks", len(stocks))
	}

	// load holdings
	var holdingCount int64
	if err := db.Model(&models.Holding{}).Count(&holdingCount).Error; err != nil {
		return err
	}
	if holdingCount > 0 {
		// holdings already present, all ok
		log.Printf("seed: %d holdings already present, skipping", holdingCount)
		return nil
	}

	// holdings aren't there,we need to now build them

	// to prevent empty stock checking while creating holdings
	if len(stocks) == 0 {
		if err := loadJSON(s.StocksPath, &stocks); err != nil {
			return err
		}
	}

	exchangePrices := map[string]map[string]float64{}
	exchangeSet := map[string]struct{}{}
	for _, st := range stocks {
		exchangeSet[st.Exchange] = struct{}{}
	}
	for ex := range exchangeSet {
		snaps, err := s.Exchange.Snapshots(ctx, ex)
		if err != nil {
			return fmt.Errorf("fetch snapshots for %s: %w", ex, err)
		}
		m := make(map[string]float64, len(snaps))
		for _, q := range snaps {
			m[q.Ticker] = q.Price
		}
		exchangePrices[ex] = m
	}

	holdings := make([]models.Holding, 0, len(stocks))
	for _, st := range stocks {
		current := exchangePrices[st.Exchange][st.Ticker]
		if current <= 0 {
			current = 100
		}
		factor := boughtFactor(s.Salt, st.Ticker)
		bought := round2(current * factor)
		holdings = append(holdings, models.Holding{
			Ticker:           st.Ticker,
			Quantity:         st.Volume,
			BoughtPriceLocal: bought,
		})
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&holdings).Error; err != nil {
		return err
	}
	log.Printf("seed: inserted %d holdings", len(holdings))
	return nil
}

// a simple way to show variance across prices, using salt and ticker here
// we hash (salt| ticker) and divide it by 10000 such because our holdings lies in 10000's
// this results in varied continuous factor from the bought price
func boughtFactor(salt, ticker string) float64 {
	h := fnv.New64a() // can use crypto/sha256 here as well, but this library is lightweight in nautre, and crypt/sha256 generates big numbers which we don't need for this scrope
	h.Write([]byte(salt + "|" + ticker))
	// 550 buckets / 1000 → [0, 0.55); +0.7 → bought price lands in [0.7, 1.25) of current; ~55% gainers
	return 0.7 + float64(h.Sum64()%550)/1000
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
