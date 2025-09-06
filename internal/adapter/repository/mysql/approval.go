package mysql

import (
	"context"

	approvalDomain "amartha-backend-test/internal/domain/approval"

	"gorm.io/gorm"
)

type ApprovalRepository struct{ db *gorm.DB }

func NewApprovalRepository(db *gorm.DB) *ApprovalRepository { return &ApprovalRepository{db: db} }

// Tx helper (optional) — bind this repo to a transaction when needed.
func (r *ApprovalRepository) Tx(ctx context.Context, fn func(repo *ApprovalRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&ApprovalRepository{db: tx})
	})
}

func (r *ApprovalRepository) Create(ctx context.Context, a *approvalDomain.Approval) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *ApprovalRepository) GetByLoanID(ctx context.Context, loanNumericID uint64) (*approvalDomain.Approval, error) {
	var out approvalDomain.Approval
	res := r.db.WithContext(ctx).
		Where("loan_id = ? AND deleted_at IS NULL", loanNumericID).
		First(&out)
	return &out, res.Error
}

func (r *ApprovalRepository) GetByApprovalID(ctx context.Context, approvalID string) (*approvalDomain.Approval, error) {
	var out approvalDomain.Approval
	res := r.db.WithContext(ctx).
		Where("approval_id = ? AND deleted_at IS NULL", approvalID).
		First(&out)
	return &out, res.Error
}
