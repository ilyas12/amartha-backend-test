package approval

import (
	"time"
)

type ApproveInput struct {
	LoanID              string
	PhotoURL            string
	ValidatorEmployeeID string    // 32-char hex
	ApprovalDate        time.Time // date-only is fine; store .UTC()
}

type ApprovalDTO struct {
	ApprovalID string    `json:"approval_id"`
	LoanID     string    `json:"loan_id"`
	PhotoURL   string    `json:"photo_url"`
	ApprovedAt time.Time `json:"approved_at"` // equals input date @ 00:00:00 UTC
}
