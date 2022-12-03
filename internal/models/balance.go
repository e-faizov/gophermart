package models

import "time"

type Withdraw struct {
	Order     string    `json:"order"`
	Sum       float64   `json:"sum"`
	Processed time.Time `json:"processed_at,omitempty"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}
