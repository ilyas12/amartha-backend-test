package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	approvalDomain "amartha-backend-test/internal/domain/approval"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- SQLite-friendly schema only for tests (no enums/engine specifics) ---
type approvalSQLite struct {
	ID                  uint64         `gorm:"primaryKey;column:id;autoIncrement"`
	ApprovalID          string         `gorm:"size:64;uniqueIndex;column:approval_id"`
	LoanID              uint64         `gorm:"column:loan_id"`
	PhotoURL            string         `gorm:"column:photo_url"`
	ValidatorEmployeeID string         `gorm:"column:validator_employee_id"`
	ApprovalDate        time.Time      `gorm:"column:approval_date"`
	CreatedAt           time.Time      `gorm:"column:created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at"`
	DeletedBy           string         `gorm:"column:deleted_by"`
}

func (approvalSQLite) TableName() string { return "approvals" }

// openApprovalTestDB creates an in-memory sqlite DB and migrates ONLY the sqlite-safe schema.
func openApprovalTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// IMPORTANT: migrate the sqlite-safe model, NOT the domain model.
	if err := db.AutoMigrate(&approvalSQLite{}); err != nil {
		t.Fatalf("auto-migrate: %v", err)
	}
	return db
}

func makeApproval(approvalID string, loanNumericID uint64, when time.Time) *approvalDomain.Approval {
	return &approvalDomain.Approval{
		ApprovalID:          approvalID,
		LoanID:              loanNumericID,
		PhotoURL:            "https://example.com/a.jpg",
		ValidatorEmployeeID: "EMP-1",
		ApprovalDate:        when.UTC(),
	}
}

func TestApproval_CreateAndGet(t *testing.T) {
	db := openApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	in := makeApproval("APR-001", 777, now)

	if err := repo.Create(ctx, in); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// By LoanID
	gotByLoan, err := repo.GetByLoanID(ctx, 777)
	if err != nil {
		t.Fatalf("GetByLoanID: %v", err)
	}
	if gotByLoan == nil || gotByLoan.ApprovalID != "APR-001" || gotByLoan.LoanID != 777 {
		t.Errorf("unexpected row by loan: %+v", gotByLoan)
	}
	if !gotByLoan.ApprovalDate.Equal(now) {
		t.Errorf("ApprovalDate not preserved as UTC: got=%v want=%v", gotByLoan.ApprovalDate, now)
	}

	// By ApprovalID
	gotByID, err := repo.GetByApprovalID(ctx, "APR-001")
	if err != nil {
		t.Fatalf("GetByApprovalID: %v", err)
	}
	if gotByID == nil || gotByID.LoanID != 777 || gotByID.ApprovalID != "APR-001" {
		t.Errorf("unexpected row by id: %+v", gotByID)
	}
}

func TestApproval_NotFound(t *testing.T) {
	db := openApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	_, err := repo.GetByLoanID(ctx, 999)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound for GetByLoanID, got %v", err)
	}

	_, err = repo.GetByApprovalID(ctx, "NOPE")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound for GetByApprovalID, got %v", err)
	}
}

func TestApproval_Tx_Commit(t *testing.T) {
	db := openApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	err := repo.Tx(ctx, func(r *ApprovalRepository) error {
		return r.Create(ctx, makeApproval("APR-COMMIT", 123, time.Now()))
	})
	if err != nil {
		t.Fatalf("Tx commit path err: %v", err)
	}

	got, err := repo.GetByApprovalID(ctx, "APR-COMMIT")
	if err != nil {
		t.Fatalf("post-commit fetch err: %v", err)
	}
	if got == nil || got.LoanID != 123 {
		t.Fatalf("unexpected row after commit: %+v", got)
	}
}

func TestApproval_Tx_Rollback(t *testing.T) {
	db := openApprovalTestDB(t)
	repo := NewApprovalRepository(db)
	ctx := context.Background()

	sentinel := errors.New("boom")

	err := repo.Tx(ctx, func(r *ApprovalRepository) error {
		if err := r.Create(ctx, makeApproval("APR-ROLL", 456, time.Now())); err != nil {
			return err
		}
		// Force rollback by returning an error from the tx fn.
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}

	_, err = repo.GetByApprovalID(ctx, "APR-ROLL")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected no row after rollback, got %v", err)
	}
}
