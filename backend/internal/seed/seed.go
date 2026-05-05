package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/models"
)

type Seeder struct {
	DB            *gorm.DB
	StocksPath    string
	ExchangesPath string
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

	holdings := make([]models.Holding, 0, len(stocks))
	for _, st := range stocks {
		bought := round2(boughtPriceFor(st.Currency))
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

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

func boughtPriceFor(currency string) float64 {
	switch strings.ToUpper(currency) {
	case "INR":
		return 500 + rand.Float64()*4500
	case "JPY":
		return 1000 + rand.Float64()*9000
	case "HKD":
		return 50 + rand.Float64()*750
	default:
		return 50 + rand.Float64()*450
	}
}
