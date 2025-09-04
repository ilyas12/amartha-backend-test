package mysql

import (
	"amartha-backend-test/internal/domain/loan"
	"context"

	"gorm.io/gorm"
)

type LoanRepository struct{ db *gorm.DB }

func NewLoanRepository(db *gorm.DB) *LoanRepository { return &LoanRepository{db: db} }

// Tx runs fn in a db transaction, passing a repo bound to the tx
func (r *LoanRepository) Tx(ctx context.Context, fn func(repo loan.Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&LoanRepository{db: tx})
	})
}

func (r *LoanRepository) Create(ctx context.Context, l *loan.Loan) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *LoanRepository) Save(ctx context.Context, l *loan.Loan) error {
	return r.db.WithContext(ctx).Save(l).Error
}

func (r *LoanRepository) GetByLoanID(ctx context.Context, loanID string) (*loan.Loan, error) {
	var out loan.Loan
	res := r.db.WithContext(ctx).Where("loan_id = ?", loanID).First(&out)
	return &out, res.Error
}
