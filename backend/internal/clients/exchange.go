package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/khareyash05/Holdings-Portfolio-Dashboard/internal/models"
	"github.com/sony/gobreaker/v2"
)

type ExchangeClient struct {
	BaseURL string
	HTTP    *http.Client
	breaker *gobreaker.CircuitBreaker[[]models.PriceSnapshot]
}

type snapshotsResponse struct {
	Exchange  string                 `json:"exchange"`
	Currency  string                 `json:"currency"`
	Snapshots []models.PriceSnapshot `json:"snapshots"`
}

func NewExchangeClient(baseURL string) *ExchangeClient {
	t := &http.Transport{
		MaxConnsPerHost:     50,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	}
	return &ExchangeClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 5 * time.Second, Transport: t},
		breaker: newBreaker[[]models.PriceSnapshot]("exchange"),
	}
}

func (c *ExchangeClient) Snapshots(ctx context.Context, exchange string) ([]models.PriceSnapshot, error) {
	return c.breaker.Execute(func() ([]models.PriceSnapshot, error) {
		u := fmt.Sprintf("%s/exchange/%s/snapshots", c.BaseURL, strings.ToUpper(exchange))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("exchange snapshots %s: status %d", exchange, resp.StatusCode)
		}
		var out snapshotsResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
		return out.Snapshots, nil
	})
}
