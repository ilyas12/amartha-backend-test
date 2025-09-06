package loanmock

import (
	domain "amartha-backend-test/internal/domain/loan"
	"context"
)

// Repo is a function-backed mock that satisfies domain.Repository.
// Only methods you need are included; add more as tests require.
type Repo struct {
	CreateFn                     func(ctx context.Context, l *domain.Loan) error
	GetByLoanIDFn                func(ctx context.Context, loanID string) (*domain.Loan, error)
	SaveFn                       func(ctx context.Context, l *domain.Loan) error
	GetPendingLoanByBorrowerIDFn func(ctx context.Context, borrowerID string) (*domain.Loan, error)
	GetByLoanIDForUpdateFn       func(ctx context.Context, loanID string) (*domain.Loan, error)
}

func (m *Repo) Create(ctx context.Context, l *domain.Loan) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, l)
	}
	return nil
}
func (m *Repo) GetByLoanID(ctx context.Context, loanID string) (*domain.Loan, error) {
	if m.GetByLoanIDFn != nil {
		return m.GetByLoanIDFn(ctx, loanID)
	}
	return nil, context.Canceled // or errors.New("not implemented")
}
func (m *Repo) Save(ctx context.Context, l *domain.Loan) error {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, l)
	}
	return nil
}

func (m *Repo) GetPendingLoanByBorrowerID(ctx context.Context, borrowerID string) (*domain.Loan, error) {
	if m.GetPendingLoanByBorrowerIDFn != nil {
		return m.GetPendingLoanByBorrowerIDFn(ctx, borrowerID)
	}
	return nil, context.Canceled
}

func (m *Repo) GetByLoanIDForUpdate(ctx context.Context, borrowerID string) (*domain.Loan, error) {
	if m.GetByLoanIDForUpdateFn != nil {
		return m.GetByLoanIDForUpdateFn(ctx, borrowerID)
	}
	return nil, context.Canceled
}
