package loanmock

import (
	"context"
	"errors"
	"testing"

	domain "amartha-backend-test/internal/domain/loan"
)

func TestRepo_Create(t *testing.T) {
	ctx := context.Background()
	l := &domain.Loan{LoanID: "LN-1"}

	// Uses provided func
	called := false
	wantErr := errors.New("boom")
	m := &Repo{
		CreateFn: func(gotCtx context.Context, got *domain.Loan) error {
			called = true
			if gotCtx != ctx {
				t.Fatalf("Create ctx mismatch")
			}
			if got != l {
				t.Fatalf("Create arg mismatch")
			}
			return wantErr
		},
	}
	if err := m.Create(ctx, l); !errors.Is(err, wantErr) {
		t.Fatalf("Create: want %v, got %v", wantErr, err)
	}
	if !called {
		t.Fatalf("CreateFn not called")
	}

	// Default (nil func) → no-op, nil error
	m = &Repo{}
	if err := m.Create(ctx, l); err != nil {
		t.Fatalf("Create default: want nil, got %v", err)
	}
}

func TestRepo_GetByLoanID(t *testing.T) {
	ctx := context.Background()
	want := &domain.Loan{LoanID: "LN-2"}

	// Uses provided func
	called := false
	m := &Repo{
		GetByLoanIDFn: func(gotCtx context.Context, loanID string) (*domain.Loan, error) {
			called = true
			if gotCtx != ctx {
				t.Fatalf("GetByLoanID ctx mismatch")
			}
			if loanID != "LN-2" {
				t.Fatalf("GetByLoanID loanID mismatch: got %s", loanID)
			}
			return want, nil
		},
	}
	got, err := m.GetByLoanID(ctx, "LN-2")
	if err != nil {
		t.Fatalf("GetByLoanID: unexpected err: %v", err)
	}
	if got != want {
		t.Fatalf("GetByLoanID: want %+v, got %+v", want, got)
	}
	if !called {
		t.Fatalf("GetByLoanIDFn not called")
	}

	// Default (nil func) → context.Canceled
	m = &Repo{}
	got, err = m.GetByLoanID(ctx, "LN-2")
	if err != context.Canceled {
		t.Fatalf("GetByLoanID default: want context.Canceled, got %v", err)
	}
	if got != nil {
		t.Fatalf("GetByLoanID default: want nil loan, got %+v", got)
	}
}

func TestRepo_Save(t *testing.T) {
	ctx := context.Background()
	l := &domain.Loan{LoanID: "LN-3"}

	// Uses provided func
	called := false
	wantErr := errors.New("save-fail")
	m := &Repo{
		SaveFn: func(gotCtx context.Context, got *domain.Loan) error {
			called = true
			if gotCtx != ctx {
				t.Fatalf("Save ctx mismatch")
			}
			if got != l {
				t.Fatalf("Save arg mismatch")
			}
			return wantErr
		},
	}
	if err := m.Save(ctx, l); !errors.Is(err, wantErr) {
		t.Fatalf("Save: want %v, got %v", wantErr, err)
	}
	if !called {
		t.Fatalf("SaveFn not called")
	}

	// Default (nil func) → no-op, nil error
	m = &Repo{}
	if err := m.Save(ctx, l); err != nil {
		t.Fatalf("Save default: want nil, got %v", err)
	}
}

func TestRepo_GetPendingLoanByBorrowerID(t *testing.T) {
	ctx := context.Background()
	want := &domain.Loan{LoanID: "LN-4"}

	// Uses provided func
	called := false
	m := &Repo{
		GetPendingLoanByBorrowerIDFn: func(gotCtx context.Context, borrowerID string) (*domain.Loan, error) {
			called = true
			if gotCtx != ctx {
				t.Fatalf("GetPending ctx mismatch")
			}
			if borrowerID != "BR-1" {
				t.Fatalf("GetPending borrowerID mismatch: got %s", borrowerID)
			}
			return want, nil
		},
	}
	got, err := m.GetPendingLoanByBorrowerID(ctx, "BR-1")
	if err != nil {
		t.Fatalf("GetPendingLoanByBorrowerID: unexpected err: %v", err)
	}
	if got != want {
		t.Fatalf("GetPendingLoanByBorrowerID: want %+v, got %+v", want, got)
	}
	if !called {
		t.Fatalf("GetPendingLoanByBorrowerIDFn not called")
	}

	// Default (nil func) → context.Canceled
	m = &Repo{}
	got, err = m.GetPendingLoanByBorrowerID(ctx, "BR-1")
	if err != context.Canceled {
		t.Fatalf("GetPending default: want context.Canceled, got %v", err)
	}
	if got != nil {
		t.Fatalf("GetPending default: want nil loan, got %+v", got)
	}
}

func TestRepo_GetByLoanIDForUpdate(t *testing.T) {
	ctx := context.Background()
	want := &domain.Loan{LoanID: "LN-5"}

	// Uses provided func
	called := false
	m := &Repo{
		GetByLoanIDForUpdateFn: func(gotCtx context.Context, loanID string) (*domain.Loan, error) {
			called = true
			if gotCtx != ctx {
				t.Fatalf("GetByLoanIDForUpdate ctx mismatch")
			}
			if loanID != "LN-5" {
				t.Fatalf("GetByLoanIDForUpdate loanID mismatch: got %s", loanID)
			}
			return want, nil
		},
	}
	got, err := m.GetByLoanIDForUpdate(ctx, "LN-5")
	if err != nil {
		t.Fatalf("GetByLoanIDForUpdate: unexpected err: %v", err)
	}
	if got != want {
		t.Fatalf("GetByLoanIDForUpdate: want %+v, got %+v", want, got)
	}
	if !called {
		t.Fatalf("GetByLoanIDForUpdateFn not called")
	}

	// Default (nil func) → context.Canceled
	m = &Repo{}
	got, err = m.GetByLoanIDForUpdate(ctx, "LN-5")
	if err != context.Canceled {
		t.Fatalf("GetByLoanIDForUpdate default: want context.Canceled, got %v", err)
	}
	if got != nil {
		t.Fatalf("GetByLoanIDForUpdate default: want nil loan, got %+v", got)
	}
}
