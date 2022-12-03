package models

import "time"

type Order struct {
	Number   string    `json:"number"`
	Status   string    `json:"status"`
	Accrual  *float64  `json:"accrual,omitempty"`
	Uploaded time.Time `json:"uploaded_at"`
}
