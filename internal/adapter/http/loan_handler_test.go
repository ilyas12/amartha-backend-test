package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "amartha-backend-test/internal/domain/loan"
	loanmock "amartha-backend-test/internal/testutil/loanmock"
	uc "amartha-backend-test/internal/usecase/loan"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// -------- helpers --------

func newEchoWithValidator() *echo.Echo {
	e := echo.New()
	e.Validator = NewValidator()
	return e
}

func mustJSON(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

// -------- tests --------

func TestCreateLoan_Success(t *testing.T) {
	e := newEchoWithValidator()

	repo := &loanmock.Repo{
		// No pending loan found
		GetPendingLoanByBorrowerIDFn: func(ctx context.Context, borrowerID string) (*domain.Loan, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateFn: func(ctx context.Context, l *domain.Loan) error {
			if l.CreatedAt.IsZero() {
				l.CreatedAt = time.Now().UTC()
			}
			return nil
		},
	}
	usecase := uc.NewUsecase(repo)
	h := NewLoanHandler(usecase)

	reqBody := map[string]any{
		"borrower_id": strings.Repeat("b", 32),
		"principal":   5000000,
		"rate":        1.29,
		"roi":         0.90,
	}
	req := httptest.NewRequest(stdhttp.MethodPost, "/loans", mustJSON(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateLoan(c); err != nil {
		t.Fatalf("CreateLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	var got uc.LoanDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if got.BorrowerID != strings.Repeat("b", 32) || got.Principal != 5000000 {
		t.Fatalf("unexpected dto: %+v", got)
	}
	if got.State != string(domain.StateProposed) {
		t.Fatalf("state = %s, want proposed", got.State)
	}
}

func TestCreateLoan_BindError(t *testing.T) {
	e := newEchoWithValidator()
	usecase := uc.NewUsecase(&loanmock.Repo{})
	h := NewLoanHandler(usecase)

	req := httptest.NewRequest(stdhttp.MethodPost, "/loans", strings.NewReader(`{"borrower_id":`)) // broken JSON
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateLoan(c); err != nil {
		t.Fatalf("CreateLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var er ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &er)
	if er.Error != "invalid body" {
		t.Fatalf("error = %q, want %q", er.Error, "invalid body")
	}
}

func TestCreateLoan_ValidationError(t *testing.T) {
	e := newEchoWithValidator()
	usecase := uc.NewUsecase(&loanmock.Repo{}) // won't be called
	h := NewLoanHandler(usecase)

	// invalid: borrower_id not hex32, principal not intlike, rate too many decimals, roi below min
	reqBody := map[string]any{
		"borrower_id": "NOT_HEX_32",
		"principal":   5000000.01,
		"rate":        1.234,
		"roi":         0.89,
	}
	req := httptest.NewRequest(stdhttp.MethodPost, "/loans", mustJSON(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateLoan(c); err != nil {
		t.Fatalf("CreateLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	var er ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &er); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if er.Error != "validation failed" {
		t.Fatalf("error = %q, want %q", er.Error, "validation failed")
	}
	if !containsFieldMsg(er.Details, "BorrowerID", "32-char lowercase hex") {
		t.Fatalf("missing hex32 detail: %+v", er.Details)
	}
	if !containsFieldMsg(er.Details, "Principal", "integer value") {
		t.Fatalf("missing intlike detail for principal: %+v", er.Details)
	}
	if !containsFieldMsg(er.Details, "Rate", "at most 2 decimal places") {
		t.Fatalf("missing dec2 detail for rate: %+v", er.Details)
	}
}

func TestCreateLoan_PendingLoanConflict(t *testing.T) {
	e := newEchoWithValidator()

	// Simulate existing pending/proposed loan for this borrower => Usecase should reject
	repo := &loanmock.Repo{
		GetPendingLoanByBorrowerIDFn: func(ctx context.Context, borrowerID string) (*domain.Loan, error) {
			return &domain.Loan{
				LoanID:         "existingpendingexistingpendingexistin",
				BorrowerID:     borrowerID,
				State:          domain.StateProposed,
				StateUpdatedAt: time.Now().UTC(),
				CreatedAt:      time.Now().UTC(),
			}, nil
		},
	}
	usecase := uc.NewUsecase(repo)
	h := NewLoanHandler(usecase)

	reqBody := map[string]any{
		"borrower_id": strings.Repeat("b", 32),
		"principal":   5000000,
		"rate":        1.29,
		"roi":         0.90,
	}
	req := httptest.NewRequest(stdhttp.MethodPost, "/loans", mustJSON(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := h.CreateLoan(c); err != nil {
		t.Fatalf("CreateLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusBadRequest { // handler maps usecase err to 400
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestGetLoan_Success(t *testing.T) {
	e := echo.New()

	repo := &loanmock.Repo{
		GetByLoanIDFn: func(ctx context.Context, loanID string) (*domain.Loan, error) {
			if loanID != "llllllllllllllllllllllllllllllll" {
				return nil, errors.New("not found")
			}
			return &domain.Loan{
				LoanID:     loanID,
				BorrowerID: strings.Repeat("b", 32),
				Principal:  7000000,
				Rate:       1.5,
				ROI:        1.0,
				State:      domain.StateProposed,
				CreatedAt:  time.Now().UTC(),
			}, nil
		},
	}
	usecase := uc.NewUsecase(repo)
	h := NewLoanHandler(usecase)

	req := httptest.NewRequest(stdhttp.MethodGet, "/loans/llllllllllllllllllllllllllllllll", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loan_id")
	c.SetParamValues("llllllllllllllllllllllllllllllll")

	if err := h.GetLoan(c); err != nil {
		t.Fatalf("GetLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var dto uc.LoanDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if dto.LoanID != "llllllllllllllllllllllllllllllll" {
		t.Fatalf("loan_id = %s, want %s", dto.LoanID, "llllllllllllllllllllllllllllllll")
	}
}

func TestGetLoan_NotFound(t *testing.T) {
	e := echo.New()
	repo := &loanmock.Repo{
		GetByLoanIDFn: func(ctx context.Context, loanID string) (*domain.Loan, error) {
			return nil, errors.New("not found")
		},
	}
	usecase := uc.NewUsecase(repo)
	h := NewLoanHandler(usecase)

	req := httptest.NewRequest(stdhttp.MethodGet, "/loans/xxx", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loan_id")
	c.SetParamValues("xxx")

	if err := h.GetLoan(c); err != nil {
		t.Fatalf("GetLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var m map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &m)
	if m["error"] != "not found" {
		t.Fatalf("error = %q, want %q", m["error"], "not found")
	}
}
