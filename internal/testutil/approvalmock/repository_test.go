package loanmock

import (
	"context"
	"errors"
	"testing"

	domain "amartha-backend-test/internal/domain/approval"
)

func TestRepo_Create(t *testing.T) {
	ctx := context.Background()
	a := &domain.Approval{ApprovalID: "APR-1", LoanID: 123}

	// Uses provided func
	called := false
	wantErr := errors.New("boom")
	m := &Repo{
		CreateFn: func(gotCtx context.Context, got *domain.Approval) error {
			called = true
			if gotCtx != ctx {
				t.Fatalf("ctx mismatch")
			}
			if got != a {
				t.Fatalf("arg mismatch")
			}
			return wantErr
		},
	}
	if err := m.Create(ctx, a); !errors.Is(err, wantErr) {
		t.Fatalf("Create: want %v, got %v", wantErr, err)
	}
	if !called {
		t.Fatalf("CreateFn not called")
	}

	// Default (nil func) → no-op, nil error
	m = &Repo{}
	if err := m.Create(ctx, a); err != nil {
		t.Fatalf("Create default: want nil, got %v", err)
	}
}

func TestRepo_GetByLoanID(t *testing.T) {
	ctx := context.Background()
	want := &domain.Approval{ApprovalID: "APR-2", LoanID: 456}

	// Uses provided func
	called := false
	m := &Repo{
		GetByLoanIDFn: func(gotCtx context.Context, id uint64) (*domain.Approval, error) {
			called = true
			if gotCtx != ctx {
				t.Fatalf("ctx mismatch")
			}
			if id != 456 {
				t.Fatalf("loanNumericID mismatch: got %d", id)
			}
			return want, nil
		},
	}
	got, err := m.GetByLoanID(ctx, 456)
	if err != nil {
		t.Fatalf("GetByLoanID: unexpected err %v", err)
	}
	if got != want {
		t.Fatalf("GetByLoanID: want %+v, got %+v", want, got)
	}
	if !called {
		t.Fatalf("GetByLoanIDFn not called")
	}

	// Default (nil func) → context.Canceled
	m = &Repo{}
	got, err = m.GetByLoanID(ctx, 999)
	if err != context.Canceled {
		t.Fatalf("GetByLoanID default: want context.Canceled, got %v", err)
	}
	if got != nil {
		t.Fatalf("GetByLoanID default: want nil, got %+v", got)
	}
}

func TestRepo_GetByApprovalID(t *testing.T) {
	ctx := context.Background()
	want := &domain.Approval{ApprovalID: "APR-3", LoanID: 789}

	// Uses provided func
	called := false
	m := &Repo{
		GetByApprovalIDFn: func(gotCtx context.Context, id string) (*domain.Approval, error) {
			called = true
			if gotCtx != ctx {
				t.Fatalf("ctx mismatch")
			}
			if id != "APR-3" {
				t.Fatalf("approvalID mismatch: got %s", id)
			}
			return want, nil
		},
	}
	got, err := m.GetByApprovalID(ctx, "APR-3")
	if err != nil {
		t.Fatalf("GetByApprovalID: unexpected err %v", err)
	}
	if got != want {
		t.Fatalf("GetByApprovalID: want %+v, got %+v", want, got)
	}
	if !called {
		t.Fatalf("GetByApprovalIDFn not called")
	}

	// Default (nil func) → context.Canceled
	m = &Repo{}
	got, err = m.GetByApprovalID(ctx, "APR-3")
	if err != context.Canceled {
		t.Fatalf("GetByApprovalID default: want context.Canceled, got %v", err)
	}
	if got != nil {
		t.Fatalf("GetByApprovalID default: want nil, got %+v", got)
	}
}
