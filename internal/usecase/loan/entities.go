package loan

import (
	"time"
)

type CreateLoanInput struct {
	BorrowerID string  `json:"borrower_id"`
	Principal  float64 `json:"principal"`
	Rate       float64 `json:"rate"`
	ROI        float64 `json:"roi"`
}

type LoanDTO struct {
	LoanID     string    `json:"loan_id"`
	BorrowerID string    `json:"borrower_id"`
	Principal  float64   `json:"principal"`
	Rate       float64   `json:"rate"`
	ROI        float64   `json:"roi"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"created_at"`
}
