package models

import "time"

type CostData struct {
	Service     string    `json:"service"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Granularity string    `json:"granularity"`
}

type CostSummary struct {
	TotalCost     float64    `json:"total_cost"`
	Currency      string     `json:"currency"`
	PeriodStart   time.Time  `json:"period_start"`
	PeriodEnd     time.Time  `json:"period_end"`
	Services      []CostData `json:"services"`
	LastUpdated   time.Time  `json:"last_updated"`
}

type HealthCheck struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}