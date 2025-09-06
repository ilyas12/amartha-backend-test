package uow

import (
	"amartha-backend-test/internal/domain/approval"
	"amartha-backend-test/internal/domain/loan"
	"context"
)

// domain/uow/uow.go
type Repos struct {
	Loans     loan.Repository
	Approvals approval.Repository
}

type UnitOfWork interface {
	// plain tx
	WithinTx(ctx context.Context, fn func(r Repos) error) error
	// convenience: lock loan first, then pass it in
	WithinLoanTx(ctx context.Context, loanID string, fn func(r Repos, l *loan.Loan) error) error
}
