package approval

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("approval not found")
)

// Table: approvals (matches your DDL)
type Approval struct {
	// Internal numeric PK
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement"`
	// Public identifier (32-char lowercase hex)
	ApprovalID string `gorm:"column:approval_id;type:char(32);not null;uniqueIndex:ux_approvals_approval_id_active"`
	// FK to loans.id (numeric)
	LoanID              uint64         `gorm:"column:loan_id;not null;index;uniqueIndex:ux_approvals_loan_active"`
	PhotoURL            string         `gorm:"column:photo_url;type:text;not null"`
	ValidatorEmployeeID string         `gorm:"column:validator_employee_id;type:char(32);not null"`
	ApprovalDate        time.Time      `gorm:"column:approval_date;type:date;not null"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;index"`
	DeletedBy           *string        `gorm:"column:deleted_by;type:char(32);"`
}

func (Approval) TableName() string { return "approvals" }
