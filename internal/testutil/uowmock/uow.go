package uowmock

import (
	"context"
	"errors"

	"amartha-backend-test/internal/domain/loan"
	"amartha-backend-test/internal/domain/uow"
)

// Ensure compile-time compliance
var _ uow.UnitOfWork = (*UoW)(nil)

var errUnimplemented = errors.New("uowmock: method not implemented")

// UoW is a function-backed mock that satisfies uow.UnitOfWork.
// Fill in the function fields you need in a test; unfilled ones return errUnimplemented.
type UoW struct {
	WithinTxFn     func(ctx context.Context, fn func(r uow.Repos) error) error
	WithinLoanTxFn func(ctx context.Context, loanID string, fn func(r uow.Repos, l *loan.Loan) error) error
}

// Convenience fluent setters
func New() *UoW { return &UoW{} }
func (m *UoW) WithWithinTx(fn func(context.Context, func(uow.Repos) error) error) *UoW {
	m.WithinTxFn = fn
	return m
}
func (m *UoW) WithWithinLoanTx(fn func(context.Context, string, func(uow.Repos, *loan.Loan) error) error) *UoW {
	m.WithinLoanTxFn = fn
	return m
}
func (m *UoW) Reset() { *m = UoW{} }

// Methods implementing UnitOfWork
func (m *UoW) WithinTx(ctx context.Context, fn func(r uow.Repos) error) error {
	if m.WithinTxFn != nil {
		return m.WithinTxFn(ctx, fn)
	}
	return errUnimplemented
}
func (m *UoW) WithinLoanTx(ctx context.Context, loanID string, fn func(r uow.Repos, l *loan.Loan) error) error {
	if m.WithinLoanTxFn != nil {
		return m.WithinLoanTxFn(ctx, loanID, fn)
	}
	return errUnimplemented
}
