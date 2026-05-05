package portfolio

import (
	"fmt"
	"math"
	"strings"
)

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
	if rFrom == 0 { // basic math divide by 0 case
		return 0, fmt.Errorf("zero rate for %s", from)
	}
	return rTo / rFrom, nil
}
