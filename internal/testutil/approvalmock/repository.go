package loanmock

import (
	domain "amartha-backend-test/internal/domain/approval"
	"context"
)

// Repo is a function-backed mock that satisfies domain.Repository.
// Only methods you need are included; add more as tests require.
type Repo struct {
	CreateFn          func(ctx context.Context, a *domain.Approval) error
	GetByLoanIDFn     func(ctx context.Context, loanNumericID uint64) (*domain.Approval, error)
	GetByApprovalIDFn func(ctx context.Context, approvalID string) (*domain.Approval, error)
}

func (m *Repo) Create(ctx context.Context, l *domain.Approval) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, l)
	}
	return nil
}

func (m *Repo) GetByLoanID(ctx context.Context, loanNumericID uint64) (*domain.Approval, error) {
	if m.GetByLoanIDFn != nil {
		return m.GetByLoanIDFn(ctx, loanNumericID)
	}
	return nil, context.Canceled
}

func (m *Repo) GetByApprovalID(ctx context.Context, approvalID string) (*domain.Approval, error) {
	if m.GetByApprovalIDFn != nil {
		return m.GetByApprovalIDFn(ctx, approvalID)
	}
	return nil, context.Canceled
}
