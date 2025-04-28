package models

type RateUpdate struct {
	Timestamp   int64   `json:"timestamp"`
	Rate        float64 `json:"rate"`
	Assets      string  `json:"assets"`
	TotalSupply string  `json:"total_supply"`
}
