package clients

import (
	"time"

	"github.com/sony/gobreaker/v2"
)

func NewBreaker[T any](name string) *gobreaker.CircuitBreaker[T] {
	return gobreaker.NewCircuitBreaker[T](gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			if c.Requests < 3 {
				return false
			}
			ratio := float64(c.TotalFailures) / float64(c.Requests)
			return ratio >= 0.5 // if failures rates is more than 50%, break the connection 
		},
	})
}
