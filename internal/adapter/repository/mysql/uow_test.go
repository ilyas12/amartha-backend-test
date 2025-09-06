package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	approvalDomain "amartha-backend-test/internal/domain/approval"
	loanDomain "amartha-backend-test/internal/domain/loan"
	"amartha-backend-test/internal/domain/uow"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openUowTestDB migrates both tables, so UoW can orchestrate both repos.
func openUowTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&loanSQLite{}, &approvalSQLite{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return db
}

func makeLoanDomain(loanID, borrowerID string) *loanDomain.Loan {
	return &loanDomain.Loan{
		LoanID:         loanID,
		BorrowerID:     borrowerID,
		Principal:      1_000_000.00,
		Rate:           0.22,
		ROI:            0.18,
		State:          loanDomain.StateProposed,
		StateUpdatedAt: time.Now().UTC(),
	}
}

func makeApprovalDomain(apprID string, loanNumericID uint64, when time.Time) *approvalDomain.Approval {
	return &approvalDomain.Approval{
		ApprovalID:          apprID,
		LoanID:              loanNumericID,
		PhotoURL:            "https://example.com/a.jpg",
		ValidatorEmployeeID: "EMP-1",
		ApprovalDate:        when.UTC(),
	}
}

// ----------------------------- Tests -----------------------------

func TestGormUoW_WithinTx_Commit(t *testing.T) {
	db := openUowTestDB(t)
	ctx := context.Background()

	guow := NewGormUoW(db)
	loanRepo := NewLoanRepository(db)
	apprRepo := NewApprovalRepository(db)

	err := guow.WithinTx(ctx, func(rRepos uow.Repos) error {
		// Create loan, then approval referencing loan numeric ID
		l := makeLoanDomain("LN-COMMIT", "BR-1")
		if err := rRepos.Loans.Create(ctx, l); err != nil {
			return err
		}
		if l.ID == 0 {
			t.Fatalf("loan auto ID not set")
		}
		return rRepos.Approvals.Create(ctx, makeApprovalDomain("APR-COMMIT", l.ID, time.Now()))
	})
	if err != nil {
		t.Fatalf("WithinTx commit err: %v", err)
	}

	// Verify post-commit visibility
	if _, err := loanRepo.GetByLoanID(ctx, "LN-COMMIT"); err != nil {
		t.Fatalf("loan not visible after commit: %v", err)
	}
	if _, err := apprRepo.GetByApprovalID(ctx, "APR-COMMIT"); err != nil {
		t.Fatalf("approval not visible after commit: %v", err)
	}
}

func TestGormUoW_WithinTx_Rollback(t *testing.T) {
	db := openUowTestDB(t)
	ctx := context.Background()

	guow := NewGormUoW(db)
	loanRepo := NewLoanRepository(db)
	apprRepo := NewApprovalRepository(db)

	sentinel := errors.New("boom")

	_ = guow.WithinTx(ctx, func(rRepos uow.Repos) error {
		l := makeLoanDomain("LN-ROLL", "BR-2")
		if err := rRepos.Loans.Create(ctx, l); err != nil {
			return err
		}
		if err := rRepos.Approvals.Create(ctx, makeApprovalDomain("APR-ROLL", l.ID, time.Now())); err != nil {
			return err
		}
		return sentinel // force rollback
	})

	// None should exist after rollback
	if _, err := loanRepo.GetByLoanID(ctx, "LN-ROLL"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected loan not found after rollback, got %v", err)
	}
	if _, err := apprRepo.GetByApprovalID(ctx, "APR-ROLL"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected approval not found after rollback, got %v", err)
	}
}

func TestGormUoW_WithinLoanTx_Commit(t *testing.T) {
	db := openUowTestDB(t)
	ctx := context.Background()

	guow := NewGormUoW(db)
	loanRepo := NewLoanRepository(db)
	apprRepo := NewApprovalRepository(db)

	// Seed a proposed loan (outside tx)
	seed := &loanSQLite{
		LoanID:         "LN-TARGET",
		BorrowerID:     "BR-3",
		Principal:      2_000_000,
		Rate:           0.24,
		ROI:            0.19,
		State:          "proposed",
		StateUpdatedAt: time.Now().UTC().Add(-1 * time.Hour),
	}
	if err := db.Create(seed).Error; err != nil {
		t.Fatalf("seed loan: %v", err)
	}

	// Execute WithinLoanTx: should fetch locked loan and pass to fn
	if err := guow.WithinLoanTx(ctx, "LN-TARGET", func(rRepos uow.Repos, l *loanDomain.Loan) error {
		// Assert the fetched loan is correct and in proposed state
		if l == nil || l.LoanID != "LN-TARGET" || l.State != loanDomain.StateProposed {
			t.Fatalf("unexpected loan passed to fn: %+v", l)
		}

		// Create approval for this numeric loan id
		if err := rRepos.Approvals.Create(ctx, makeApprovalDomain("APR-LOCK", l.ID, time.Now())); err != nil {
			return err
		}

		// Update state → approved
		l.State = loanDomain.StateApproved
		l.StateUpdatedAt = time.Now().UTC()
		return rRepos.Loans.Save(ctx, l)
	}); err != nil {
		t.Fatalf("WithinLoanTx commit err: %v", err)
	}

	// Verify changes
	gotLoan, err := loanRepo.GetByLoanID(ctx, "LN-TARGET")
	if err != nil {
		t.Fatalf("GetByLoanID post-commit: %v", err)
	}
	if gotLoan.State != loanDomain.StateApproved {
		t.Fatalf("loan state not updated, got=%s", gotLoan.State)
	}
	if _, err := apprRepo.GetByApprovalID(ctx, "APR-LOCK"); err != nil {
		t.Fatalf("approval not visible after commit: %v", err)
	}
}

func TestGormUoW_WithinLoanTx_Rollback(t *testing.T) {
	db := openUowTestDB(t)
	ctx := context.Background()

	guow := NewGormUoW(db)
	loanRepo := NewLoanRepository(db)
	apprRepo := NewApprovalRepository(db)

	// Seed proposed loan
	seed := &loanSQLite{
		LoanID:         "LN-RB-TGT",
		BorrowerID:     "BR-4",
		Principal:      3_000_000,
		Rate:           0.25,
		ROI:            0.20,
		State:          "proposed",
		StateUpdatedAt: time.Now().UTC(),
	}
	if err := db.Create(seed).Error; err != nil {
		t.Fatalf("seed loan: %v", err)
	}

	sentinel := errors.New("stop")

	_ = guow.WithinLoanTx(ctx, "LN-RB-TGT", func(rRepos uow.Repos, l *loanDomain.Loan) error {
		// Make changes inside tx
		if err := rRepos.Approvals.Create(ctx, makeApprovalDomain("APR-RB", l.ID, time.Now())); err != nil {
			return err
		}
		l.State = loanDomain.StateApproved
		if err := rRepos.Loans.Save(ctx, l); err != nil {
			return err
		}
		return sentinel // force rollback
	})

	// After rollback: state unchanged, approval absent
	gotLoan, err := loanRepo.GetByLoanID(ctx, "LN-RB-TGT")
	if err != nil {
		t.Fatalf("post-rollback GetByLoanID: %v", err)
	}
	if gotLoan.State != loanDomain.StateProposed {
		t.Fatalf("expected proposed after rollback, got %s", gotLoan.State)
	}
	if _, err := apprRepo.GetByApprovalID(ctx, "APR-RB"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected approval absent after rollback, got %v", err)
	}
}

func TestGormUoW_WithinLoanTx_LoanNotFound(t *testing.T) {
	db := openUowTestDB(t)
	ctx := context.Background()

	guow := NewGormUoW(db)

	err := guow.WithinLoanTx(ctx, "LN-NOPE", func(rRepos uow.Repos, l *loanDomain.Loan) error {
		t.Fatalf("callback should not be called when loan missing")
		return nil
	})
	if err == nil {
		t.Fatalf("expected error when loan not found")
	}
}
