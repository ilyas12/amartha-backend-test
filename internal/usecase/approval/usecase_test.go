package approval

import (
	"context"
	"errors"
	"testing"
	"time"

	"amartha-backend-test/internal/domain/approval"
	"amartha-backend-test/internal/domain/loan"
	"amartha-backend-test/internal/domain/uow"
	"amartha-backend-test/internal/testutil/approvalmock"
	"amartha-backend-test/internal/testutil/loanmock"
	"amartha-backend-test/internal/testutil/uowmock"

	"gorm.io/gorm"
)

func TestUsecase_Approve(t *testing.T) {
	now := time.Date(2025, 9, 6, 10, 0, 0, 0, time.UTC)
	in := ApproveInput{
		LoanID:              "LN-123",
		PhotoURL:            "https://img/x.jpg",
		ValidatorEmployeeID: "EMP-9",
		ApprovalDate:        now,
	}

	newProposedLoan := func() *loan.Loan {
		return &loan.Loan{ID: 777, LoanID: "LN-123", State: loan.StateProposed}
	}

	tests := []struct {
		name    string
		setup   func() *Usecase
		wantErr error
		check   func(*ApprovalDTO) error
	}{
		{
			name: "happy path proposed -> approved",
			setup: func() *Usecase {
				loans := &loanmock.Repo{
					GetByLoanIDForUpdateFn: func(ctx context.Context, loanID string) (*loan.Loan, error) {
						return newProposedLoan(), nil
					},
					SaveFn: func(ctx context.Context, l *loan.Loan) error {
						if l.State != loan.StateApproved {
							t.Fatalf("expected state=approved, got %s", l.State)
						}
						return nil
					},
				}
				apprs := &approvalmock.Repo{
					GetByLoanIDFn: func(ctx context.Context, id uint64) (*approval.Approval, error) {
						return nil, gorm.ErrRecordNotFound
					},
					CreateFn: func(ctx context.Context, a *approval.Approval) error {
						if a.LoanID != 777 || a.PhotoURL != in.PhotoURL {
							t.Fatalf("approval mismatch: %+v", a)
						}
						return nil
					},
				}
				tx := &uowmock.UoW{
					WithinTxFn: func(ctx context.Context, fn func(r uow.Repos) error) error {
						return fn(uow.Repos{Loans: loans, Approvals: apprs})
					},
				}
				return NewUsecase(loans, apprs, tx)
			},
			wantErr: nil,
			check: func(dto *ApprovalDTO) error {
				if dto == nil {
					return errors.New("dto is nil")
				}
				if dto.LoanID != "LN-123" {
					return errors.New("dto LoanID mismatch")
				}
				return nil
			},
		},
		{
			name: "loan not found",
			setup: func() *Usecase {
				loans := &loanmock.Repo{
					GetByLoanIDForUpdateFn: func(context.Context, string) (*loan.Loan, error) {
						return nil, errors.New("no rows")
					},
				}
				apprs := &approvalmock.Repo{}
				tx := &uowmock.UoW{
					WithinTxFn: func(ctx context.Context, fn func(r uow.Repos) error) error {
						return fn(uow.Repos{Loans: loans, Approvals: apprs})
					},
				}
				return NewUsecase(loans, apprs, tx)
			},
			wantErr: loan.ErrNotFound,
		},
		{
			name: "already approved state",
			setup: func() *Usecase {
				loans := &loanmock.Repo{
					GetByLoanIDForUpdateFn: func(context.Context, string) (*loan.Loan, error) {
						return &loan.Loan{ID: 1, State: loan.StateApproved}, nil
					},
				}
				apprs := &approvalmock.Repo{}
				tx := &uowmock.UoW{
					WithinTxFn: func(ctx context.Context, fn func(r uow.Repos) error) error {
						return fn(uow.Repos{Loans: loans, Approvals: apprs})
					},
				}
				return NewUsecase(loans, apprs, tx)
			},
			wantErr: loan.ErrAlreadyApproved,
		},
		{
			name: "duplicate approval exists",
			setup: func() *Usecase {
				loans := &loanmock.Repo{
					GetByLoanIDForUpdateFn: func(context.Context, string) (*loan.Loan, error) {
						return newProposedLoan(), nil
					},
				}
				apprs := &approvalmock.Repo{
					GetByLoanIDFn: func(context.Context, uint64) (*approval.Approval, error) {
						return &approval.Approval{ApprovalID: "EXIST"}, nil
					},
				}
				tx := &uowmock.UoW{
					WithinTxFn: func(ctx context.Context, fn func(r uow.Repos) error) error {
						return fn(uow.Repos{Loans: loans, Approvals: apprs})
					},
				}
				return NewUsecase(loans, apprs, tx)
			},
			wantErr: loan.ErrAlreadyApproved,
		},
		{
			name: "nil UoW",
			setup: func() *Usecase {
				return NewUsecase(nil, nil, nil)
			},
			wantErr: loan.ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			uc := tt.setup()
			dto, err := uc.Approve(context.Background(), in)

			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("want err=%v, got %v", tt.wantErr, err)
			}
			if tt.check != nil && err == nil {
				if cerr := tt.check(dto); cerr != nil {
					t.Fatalf("dto check failed: %v", cerr)
				}
			}
		})
	}
}
