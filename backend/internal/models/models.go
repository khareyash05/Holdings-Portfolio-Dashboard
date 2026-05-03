package models

import "time"

type Exchange struct {
	ShortName string `gorm:"primaryKey;column:short_name" json:"shortName"`
	Currency  string `gorm:"not null" json:"currency"`
}

type Stock struct {
	Ticker   string `gorm:"primaryKey" json:"ticker"`
	Name     string `gorm:"not null" json:"name"`
	Exchange string `gorm:"column:exchange_short_name;not null;index:idx_stocks_exchange" json:"exchange"`
	Country  string `gorm:"not null" json:"country"`
	Currency string `gorm:"not null" json:"currency"`
	Sector   string `gorm:"not null" json:"sector"`
}

type Holding struct {
	ID               int64     `gorm:"primaryKey" json:"id"`
	Ticker           string    `gorm:"uniqueIndex;not null" json:"ticker"`
	Quantity         float64   `gorm:"type:numeric(18,4);not null" json:"quantity"`
	BoughtPriceLocal float64   `gorm:"type:numeric(18,4);not null;column:bought_price_local" json:"boughtPriceLocal"`
	BoughtAt         time.Time `gorm:"not null;default:now()" json:"boughtAt"`
}

type PriceSnapshot struct {
	Ticker   string    `json:"ticker"`
	Price    float64   `json:"price"`
	Currency string    `json:"currency"`
	Exchange string    `json:"exchange"`
	AsOf     time.Time `json:"asOf"`
}

type Summary struct {
	BaseCurrency    string  `json:"baseCurrency"`
	NetWorth        float64 `json:"netWorth"`
	Invested        float64 `json:"invested"`
	UnrealizedGains float64 `json:"unrealizedGains"`
	GainsPct        float64 `json:"gainsPct"`
	Stale           bool    `json:"stale"`
}

type HoldingView struct {
	Ticker            string  `json:"ticker"`
	Name              string  `json:"name"`
	Exchange          string  `json:"exchange"`
	Country           string  `json:"country"`
	Sector            string  `json:"sector"`
	LocalCurrency     string  `json:"localCurrency"`
	Quantity          float64 `json:"quantity"`
	CurrentPriceLocal float64 `json:"currentPriceLocal"`
	BoughtPriceLocal  float64 `json:"boughtPriceLocal"`
	CurrentPriceBase  float64 `json:"currentPriceBase"`
	InvestedBase      float64 `json:"investedBase"`
	NetWorthBase      float64 `json:"netWorthBase"`
	UnrealizedGains   float64 `json:"unrealizedGains"`
	GainsPct          float64 `json:"gainsPct"`
	Stale             bool    `json:"stale"`
}

type PortfolioResponse struct {
	BaseCurrency string        `json:"baseCurrency"`
	AsOf         time.Time     `json:"asOf"`
	Summary      Summary       `json:"summary"`
	Holdings     []HoldingView `json:"holdings"`
	Exchanges    []Exchange    `json:"exchanges"`
}
