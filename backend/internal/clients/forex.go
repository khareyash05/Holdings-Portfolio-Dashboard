package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sony/gobreaker/v2"
)

type ForexClient struct {
	BaseURL string
	HTTP    *http.Client
	breaker *gobreaker.CircuitBreaker[*RatesResponse]
}

type RatesResponse struct {
	Base      string             `json:"base"`
	AsOf      string             `json:"asOf"`
	Rates     map[string]float64 `json:"rates"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

func NewForexClient(baseURL string) *ForexClient {
	t := &http.Transport{
		IdleConnTimeout:     90 * time.Second,
	}
	return &ForexClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 5 * time.Second, Transport: t},
		breaker: NewBreaker[*RatesResponse]("forex"),
	}
}

func (c *ForexClient) Rates(ctx context.Context, base string) (*RatesResponse, error) {
	return c.breaker.Execute(func() (*RatesResponse, error) {
		u, _ := url.Parse(c.BaseURL + "/rates")
		q := u.Query()
		q.Set("base", strings.ToUpper(base))
		u.RawQuery = q.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("forex rates: status %d", resp.StatusCode)
		}
		var out RatesResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return nil, err
		}
		return &out, nil
	})
}
