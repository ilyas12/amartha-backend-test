package mysql

import (
	loanDomain "amartha-backend-test/internal/domain/loan"
	"context"

	"gorm.io/gorm"
)

type LoanRepository struct{ db *gorm.DB }

func NewLoanRepository(db *gorm.DB) *LoanRepository { return &LoanRepository{db: db} }

// Tx runs fn in a db transaction, passing a repo bound to the tx
func (r *LoanRepository) Tx(ctx context.Context, fn func(repo loanDomain.Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&LoanRepository{db: tx})
	})
}

func (r *LoanRepository) Create(ctx context.Context, l *loanDomain.Loan) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *LoanRepository) Save(ctx context.Context, l *loanDomain.Loan) error {
	return r.db.WithContext(ctx).Save(l).Error
}

func (r *LoanRepository) GetByLoanID(ctx context.Context, loanID string) (*loanDomain.Loan, error) {
	var out loanDomain.Loan
	res := r.db.WithContext(ctx).Where("loan_id = ?", loanID).First(&out)
	return &out, res.Error
}

func (r *LoanRepository) GetPendingLoanByBorrowerID(ctx context.Context, borrowerID string) (*loanDomain.Loan, error) {
	var out loanDomain.Loan
	res := r.db.WithContext(ctx).
		Where("borrower_id = ? AND state = ?", borrowerID, loanDomain.StateProposed).
		Order("state_updated_at DESC, id DESC").
		First(&out)
	return &out, res.Error
}
