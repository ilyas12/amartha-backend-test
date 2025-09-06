package loan

import (
	"time"

	"gorm.io/gorm"
)

type State string

const (
	StateProposed  State = "proposed"
	StateApproved  State = "approved"
	StateInvested  State = "invested"
	StateDisbursed State = "disbursed"
	StateRejected  State = "rejected"
)

type Loan struct {
	ID             uint64         `gorm:"primaryKey;column:id" json:"-"`
	LoanID         string         `gorm:"size:32;uniqueIndex:ux_loans_loan_id_active" json:"loan_id"`
	BorrowerID     string         `gorm:"size:32;index:idx_loans_borrower_active" json:"borrower_id"`
	Principal      float64        `gorm:"type:decimal(18,2)" json:"principal"`
	Rate           float64        `gorm:"type:decimal(6,4)" json:"rate"`
	ROI            float64        `gorm:"type:decimal(6,4)" json:"roi"`
	AgreementLink  string         `gorm:"type:text" json:"agreement_link"`
	State          State          `gorm:"type:enum('proposed','rejected','approved','invested','disbursed');default:'proposed'" json:"state"`
	StateUpdatedAt time.Time      `gorm:"autoCreateTime" json:"state_updated_at"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	DeletedBy      string         `gorm:"size:32" json:"-"`
}

func (Loan) TableName() string { return "loans" }
