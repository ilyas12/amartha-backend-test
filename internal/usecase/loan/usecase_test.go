package loan

import (
	domain "amartha-backend-test/internal/domain/loan"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

// ----- test doubles -----

// mockRepo implements domain.Repository (only methods used by these tests).
type mockRepo struct {
	CreateFn                     func(ctx context.Context, l *domain.Loan) error
	GetByLoanIDFn                func(ctx context.Context, loanID string) (*domain.Loan, error)
	GetPendingLoanByBorrowerIDFn func(ctx context.Context, borrowerID string) (*domain.Loan, error)
	SaveFn                       func(ctx context.Context, l *domain.Loan) error
}

func (m *mockRepo) Create(ctx context.Context, l *domain.Loan) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, l)
	}
	return nil
}

func (m *mockRepo) GetByLoanID(ctx context.Context, loanID string) (*domain.Loan, error) {
	if m.GetByLoanIDFn != nil {
		return m.GetByLoanIDFn(ctx, loanID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRepo) Save(ctx context.Context, l *domain.Loan) error {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, l)
	}
	return nil
}

func (m *mockRepo) GetPendingLoanByBorrowerID(ctx context.Context, borrowerID string) (*domain.Loan, error) {
	if m.GetPendingLoanByBorrowerIDFn != nil {
		return m.GetPendingLoanByBorrowerID(ctx, borrowerID)
	}
	return nil, errors.New("not implemented")
}

// If your real Repository interface also includes Save(...) etc.,
// add no-op methods here so the mock satisfies it. Example:
// func (m *mockRepo) Save(ctx context.Context, l *domain.Loan) error { return nil }

// ----- tests -----
func TestCreate_Success_NoPendingLoan(t *testing.T) {
	uc := NewUsecase(&mockRepo{
		// Simulate: no pending loan ⇒ repo returns gorm.ErrRecordNotFound
		GetPendingLoanByBorrowerIDFn: func(ctx context.Context, borrowerID string) (*domain.Loan, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateFn: func(ctx context.Context, l *domain.Loan) error {
			if l.CreatedAt.IsZero() {
				l.CreatedAt = time.Now().UTC()
			}
			return nil
		},
	})

	in := CreateLoanInput{
		BorrowerID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Principal:  5_000_000,
		Rate:       0.22, ROI: 0.18,
	}
	dto, err := uc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("Create err: %v", err)
	}
	if len(dto.LoanID) != 32 {
		t.Fatalf("LoanID length: %d", len(dto.LoanID))
	}
	if dto.State != string(domain.StateProposed) {
		t.Fatalf("state=%s", dto.State)
	}
}

func TestCreate_Rejects_WhenPendingLoanExists(t *testing.T) {
	const borrowerID = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	const existingLoanID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	uc := NewUsecase(&mockRepo{
		// Simulate: a pending (proposed) loan already exists
		GetPendingLoanByBorrowerIDFn: func(ctx context.Context, id string) (*domain.Loan, error) {
			if id != borrowerID {
				return nil, fmt.Errorf("unexpected borrower id: %s", id)
			}
			return &domain.Loan{
				LoanID:         existingLoanID,
				BorrowerID:     borrowerID,
				Principal:      1_000_000,
				Rate:           0.22,
				ROI:            0.18,
				State:          domain.StateProposed,
				StateUpdatedAt: time.Now().UTC(),
				CreatedAt:      time.Now().UTC(),
			}, nil
		},
		// Create() should never be called in this scenario; guard it:
		CreateFn: func(ctx context.Context, l *domain.Loan) error {
			t.Fatalf("Create must not be called when pending loan exists")
			return nil
		},
	})

	_, err := uc.Create(context.Background(), CreateLoanInput{
		BorrowerID: borrowerID,
		Principal:  7_000_000,
		Rate:       0.21, ROI: 0.17,
	})
	if err == nil {
		t.Fatalf("expected error due to existing pending loan, got nil")
	}
	// Loose assertion: contains the message (exact text may differ in implementation)
	if want := "already has a pending loan"; !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}

func TestGet_Success(t *testing.T) {
	const LID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	now := time.Now().UTC()
	uc := NewUsecase(&mockRepo{
		GetByLoanIDFn: func(ctx context.Context, loanID string) (*domain.Loan, error) {
			return &domain.Loan{
				LoanID: LID, BorrowerID: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Principal: 1_000_000, Rate: 0.22, ROI: 0.18,
				State: domain.StateProposed, CreatedAt: now,
			}, nil
		},
	})
	dto, err := uc.Get(context.Background(), LID)
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if dto.LoanID != LID {
		t.Fatalf("got %s", dto.LoanID)
	}
}

func TestCreate_InvalidInput(t *testing.T) {
	uc := NewUsecase(&mockRepo{})
	_, err := uc.Create(context.Background(), CreateLoanInput{
		BorrowerID: "short", Principal: 0, Rate: 0.2, ROI: 0.1,
	})
	if err == nil {
		t.Fatal("want error")
	}
}
