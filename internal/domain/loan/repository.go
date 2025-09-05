package loan

import "context"

type Repository interface {
	// Basic Case
	Create(ctx context.Context, l *Loan) error
	GetByLoanID(ctx context.Context, loanID string) (*Loan, error)
	GetPendingLoanByBorrowerID(ctx context.Context, borrowerID string) (*Loan, error)
	Save(ctx context.Context, l *Loan) error
}
