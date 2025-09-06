package uowmock

import (
	"context"
	"errors"
	"testing"

	"amartha-backend-test/internal/domain/loan"
	"amartha-backend-test/internal/domain/uow"
	"amartha-backend-test/internal/testutil/approvalmock"
	"amartha-backend-test/internal/testutil/loanmock"
)

func TestUoW_WithinTx_Happy(t *testing.T) {
	ctx := context.Background()

	loans := &loanmock.Repo{}
	apprs := &approvalmock.Repo{}
	repos := uow.Repos{Loans: loans, Approvals: apprs}

	innerCalled := false
	m := &UoW{
		WithinTxFn: func(gotCtx context.Context, fn func(r uow.Repos) error) error {
			if gotCtx != ctx {
				t.Fatalf("WithinTx: ctx mismatch")
			}
			if fn == nil {
				t.Fatalf("WithinTx: fn is nil")
			}
			// simulate transaction body
			return fn(repos)
		},
	}

	err := m.WithinTx(ctx, func(r uow.Repos) error {
		innerCalled = true
		if r.Loans != loans || r.Approvals != apprs {
			t.Fatalf("WithinTx: repos not forwarded correctly")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithinTx: unexpected err: %v", err)
	}
	if !innerCalled {
		t.Fatalf("WithinTx: inner fn not called")
	}
}

func TestUoW_WithinTx_PropagatesError(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("boom")

	m := &UoW{
		WithinTxFn: func(context.Context, func(uow.Repos) error) error {
			return sentinel
		},
	}
	if err := m.WithinTx(ctx, func(uow.Repos) error { return nil }); !errors.Is(err, sentinel) {
		t.Fatalf("WithinTx: want %v, got %v", sentinel, err)
	}
}

func TestUoW_WithinTx_Default_Unimplemented(t *testing.T) {
	ctx := context.Background()
	m := &UoW{} // no funcs set
	if err := m.WithinTx(ctx, func(uow.Repos) error { return nil }); !errors.Is(err, errUnimplemented) {
		t.Fatalf("WithinTx default: want errUnimplemented, got %v", err)
	}
}

func TestUoW_WithinLoanTx_Happy(t *testing.T) {
	ctx := context.Background()

	loans := &loanmock.Repo{}
	apprs := &approvalmock.Repo{}
	repos := uow.Repos{Loans: loans, Approvals: apprs}
	lock := &loan.Loan{ID: 7, LoanID: "LN-7"}

	innerCalled := false
	m := &UoW{
		WithinLoanTxFn: func(gotCtx context.Context, loanID string, fn func(r uow.Repos, l *loan.Loan) error) error {
			if gotCtx != ctx {
				t.Fatalf("WithinLoanTx: ctx mismatch")
			}
			if loanID != "LN-7" {
				t.Fatalf("WithinLoanTx: loanID mismatch, got %s", loanID)
			}
			return fn(repos, lock)
		},
	}

	err := m.WithinLoanTx(ctx, "LN-7", func(r uow.Repos, l *loan.Loan) error {
		innerCalled = true
		if r.Loans != loans || r.Approvals != apprs {
			t.Fatalf("WithinLoanTx: repos not forwarded")
		}
		if l != lock || l.LoanID != "LN-7" {
			t.Fatalf("WithinLoanTx: loan not forwarded correctly: %+v", l)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithinLoanTx: unexpected err: %v", err)
	}
	if !innerCalled {
		t.Fatalf("WithinLoanTx: inner fn not called")
	}
}

func TestUoW_WithinLoanTx_PropagatesError(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("stop")

	m := &UoW{
		WithinLoanTxFn: func(context.Context, string, func(uow.Repos, *loan.Loan) error) error {
			return sentinel
		},
	}
	if err := m.WithinLoanTx(ctx, "LN-X", func(uow.Repos, *loan.Loan) error { return nil }); !errors.Is(err, sentinel) {
		t.Fatalf("WithinLoanTx: want %v, got %v", sentinel, err)
	}
}

func TestUoW_Default_Unimplemented_WithinLoanTx(t *testing.T) {
	ctx := context.Background()
	m := &UoW{} // no funcs set
	if err := m.WithinLoanTx(ctx, "LN-X", func(uow.Repos, *loan.Loan) error { return nil }); !errors.Is(err, errUnimplemented) {
		t.Fatalf("WithinLoanTx default: want errUnimplemented, got %v", err)
	}
}

func TestUoW_FluentSetters_And_Reset(t *testing.T) {
	m := New()
	if m.WithinTxFn != nil || m.WithinLoanTxFn != nil {
		t.Fatalf("New should start with nil funcs")
	}

	// set via fluent setters
	m.WithWithinTx(func(context.Context, func(uow.Repos) error) error { return nil }).
		WithWithinLoanTx(func(context.Context, string, func(uow.Repos, *loan.Loan) error) error { return nil })

	if m.WithinTxFn == nil || m.WithinLoanTxFn == nil {
		t.Fatalf("fluent setters didn't assign funcs")
	}

	// reset clears funcs
	m.Reset()
	if m.WithinTxFn != nil || m.WithinLoanTxFn != nil {
		t.Fatalf("Reset should clear function fields")
	}
}
