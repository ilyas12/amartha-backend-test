package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "amartha-backend-test/internal/domain/loan"
	"amartha-backend-test/pkg/id"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- SQLite-friendly schema only for tests (no ENUM) ---

type loanSQLite struct {
	ID             uint64         `gorm:"primaryKey;column:id"`
	LoanID         string         `gorm:"size:32;column:loan_id"`
	BorrowerID     string         `gorm:"size:32;column:borrower_id"`
	Principal      float64        `gorm:"column:principal"`
	Rate           float64        `gorm:"column:rate"`
	ROI            float64        `gorm:"column:roi"`
	AgreementLink  string         `gorm:"column:agreement_link"`
	State          string         `gorm:"type:text;column:state"` // ← no enum
	StateUpdatedAt time.Time      `gorm:"column:state_updated_at"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at"`
	DeletedBy      string         `gorm:"column:deleted_by"`
}

func (loanSQLite) TableName() string { return "loans" }

// openTestDB creates an in-memory sqlite DB and migrates ONLY the sqlite-safe schema.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// IMPORTANT: migrate the sqlite-safe model, NOT the domain model.
	if err := db.AutoMigrate(&loanSQLite{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return db
}

func makeLoan(loanID, borrowerID string) *domain.Loan {
	return &domain.Loan{
		LoanID:         loanID,
		BorrowerID:     borrowerID,
		Principal:      1_000_000.00,
		Rate:           0.2200,
		ROI:            0.1800,
		State:          domain.StateProposed,
		StateUpdatedAt: time.Now().UTC(),
	}
}

func TestCreateAndGetByLoanID(t *testing.T) {
	db := openTestDB(t)
	repo := NewLoanRepository(db)
	ctx := context.Background()

	loanID := id.NewID32()   // 32-char
	borrower := id.NewID32() // 32-char

	l := makeLoan(loanID, borrower)
	if err := repo.Create(ctx, l); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if l.ID == 0 {
		t.Fatalf("Create did not set auto-increment ID")
	}

	got, err := repo.GetByLoanID(ctx, loanID)
	if err != nil {
		t.Fatalf("GetByLoanID: %v", err)
	}
	if got.LoanID != loanID || got.BorrowerID != borrower {
		t.Errorf("unexpected loan: %+v", got)
	}
}

func TestSaveUpdates(t *testing.T) {
	db := openTestDB(t)
	repo := NewLoanRepository(db)
	ctx := context.Background()

	loanID := id.NewID32()
	l := makeLoan(loanID, "dddddddddddddddddddddddddddddddd")

	if err := repo.Create(ctx, l); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update a field and persist
	const link = "https://example.com/agreement.pdf"
	l.AgreementLink = link
	if err := repo.Save(ctx, l); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByLoanID(ctx, loanID)
	if err != nil {
		t.Fatalf("GetByLoanID: %v", err)
	}
	if got.AgreementLink != link {
		t.Errorf("AgreementLink not updated, got=%q want=%q", got.AgreementLink, link)
	}
}

func TestGetByLoanID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewLoanRepository(db)
	ctx := context.Background()

	_, err := repo.GetByLoanID(ctx, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestTx_Commit(t *testing.T) {
	db := openTestDB(t)
	repo := NewLoanRepository(db)
	ctx := context.Background()

	loanID := id.NewID32()
	err := repo.Tx(ctx, func(r domain.Repository) error {
		return r.Create(ctx, makeLoan(loanID, "11111111111111111111111111111111"))
	})
	if err != nil {
		t.Fatalf("Tx commit: %v", err)
	}

	// Should be visible after commit
	if _, err := repo.GetByLoanID(ctx, loanID); err != nil {
		t.Fatalf("GetByLoanID after commit: %v", err)
	}
}

func TestTx_Rollback(t *testing.T) {
	db := openTestDB(t)
	repo := NewLoanRepository(db)
	ctx := context.Background()

	loanID := id.NewID32()
	wantErr := errors.New("boom")

	_ = repo.Tx(ctx, func(r domain.Repository) error {
		if err := r.Create(ctx, makeLoan(loanID, "22222222222222222222222222222222")); err != nil {
			return err
		}
		return wantErr // force rollback
	})

	// Should not exist after rollback
	_, err := repo.GetByLoanID(ctx, loanID)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found after rollback, got %v", err)
	}
}
