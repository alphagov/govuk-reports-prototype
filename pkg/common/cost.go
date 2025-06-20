package common

import "time"

// CostData represents cost information for a service
type CostData struct {
	Service     string    `json:"service"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Granularity string    `json:"granularity"`
}