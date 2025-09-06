package http

import (
	"context"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainApproval "amartha-backend-test/internal/domain/approval"
	domainLoan "amartha-backend-test/internal/domain/loan"
	"amartha-backend-test/internal/domain/uow"
	"amartha-backend-test/internal/testutil/approvalmock"
	"amartha-backend-test/internal/testutil/loanmock"
	"amartha-backend-test/internal/testutil/uowmock"
	ucApproval "amartha-backend-test/internal/usecase/approval"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Local helper for field-error assertions (keeps this file self-contained)
func hasFieldDetail(details []FieldError, field, contains string) bool {
	for _, d := range details {
		if d.Field == field && strings.Contains(d.Message, contains) {
			return true
		}
	}
	return false
}

func TestApproveLoan_Success(t *testing.T) {
	e := newEchoWithValidator()

	// Mocks for usecase dependencies
	loans := &loanmock.Repo{
		GetByLoanIDForUpdateFn: func(ctx context.Context, loanID string) (*domainLoan.Loan, error) {
			return &domainLoan.Loan{ID: 777, LoanID: loanID, State: domainLoan.StateProposed}, nil
		},
		SaveFn: func(ctx context.Context, l *domainLoan.Loan) error { return nil },
	}
	apprs := &approvalmock.Repo{
		GetByLoanIDFn: func(ctx context.Context, numeric uint64) (*domainApproval.Approval, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateFn: func(ctx context.Context, a *domainApproval.Approval) error { return nil },
	}
	tx := &uowmock.UoW{
		WithinTxFn: func(ctx context.Context, fn func(r uow.Repos) error) error {
			return fn(uow.Repos{Loans: loans, Approvals: apprs})
		},
	}
	uc := ucApproval.NewUsecase(loans, apprs, tx)
	h := NewApprovalHandler(uc)

	body := map[string]any{
		"photo_url":             "https://cdn.example.com/img.jpg",
		"validator_employee_id": strings.Repeat("a", 32), // hex32
		"approval_date":         "2025-09-06",
	}
	req := httptest.NewRequest(stdhttp.MethodPost, "/loans/llllllllllllllllllllllllllllllll/approve", mustJSON(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loan_id")
	c.SetParamValues("llllllllllllllllllllllllllllllll")

	if err := h.ApproveLoan(c); err != nil {
		t.Fatalf("ApproveLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var dto ucApproval.ApprovalDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if dto.LoanID != "llllllllllllllllllllllllllllllll" {
		t.Fatalf("dto.LoanID = %s, want path loan_id", dto.LoanID)
	}
	if dto.PhotoURL != "https://cdn.example.com/img.jpg" {
		t.Fatalf("dto.PhotoURL mismatch: %s", dto.PhotoURL)
	}
}

func TestApproveLoan_MissingPathParam(t *testing.T) {
	e := newEchoWithValidator()

	// usecase won’t be called because we fail early
	uc := ucApproval.NewUsecase(nil, nil, nil)
	h := NewApprovalHandler(uc)

	req := httptest.NewRequest(stdhttp.MethodPost, "/loans//approve", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// NOTE: do not set params

	if err := h.ApproveLoan(c); err != nil {
		t.Fatalf("ApproveLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var er ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &er)
	if er.Error != "missing loan_id path param" {
		t.Fatalf("error = %q, want %q", er.Error, "missing loan_id path param")
	}
}

func TestApproveLoan_BindError(t *testing.T) {
	e := newEchoWithValidator()
	uc := ucApproval.NewUsecase(nil, nil, nil)
	h := NewApprovalHandler(uc)

	req := httptest.NewRequest(stdhttp.MethodPost, "/loans/abcd/approve", strings.NewReader(`{"photo_url":`)) // broken JSON
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loan_id")
	c.SetParamValues("abcd")

	if err := h.ApproveLoan(c); err != nil {
		t.Fatalf("ApproveLoan error: %v", err)
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

func TestApproveLoan_ValidationError(t *testing.T) {
	e := newEchoWithValidator()
	uc := ucApproval.NewUsecase(nil, nil, nil) // won’t be called
	h := NewApprovalHandler(uc)

	// invalid: bad URL, not hex32, wrong date format
	body := map[string]any{
		"photo_url":             "not-a-url",
		"validator_employee_id": "NOTHEX",
		"approval_date":         "2025/09/06",
	}
	req := httptest.NewRequest(stdhttp.MethodPost, "/loans/xyz/approve", mustJSON(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loan_id")
	c.SetParamValues("xyz")

	if err := h.ApproveLoan(c); err != nil {
		t.Fatalf("ApproveLoan error: %v", err)
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
	// ensure at least some field details are present
	if !(hasFieldDetail(er.Details, "PhotoURL", "url") || hasFieldDetail(er.Details, "ValidatorEmployeeID", "32") || hasFieldDetail(er.Details, "ApprovalDate", "datetime")) {
		t.Fatalf("missing expected field errors: %+v", er.Details)
	}
}

func TestApproveLoan_NotFound(t *testing.T) {
	e := newEchoWithValidator()

	loans := &loanmock.Repo{
		GetByLoanIDForUpdateFn: func(ctx context.Context, loanID string) (*domainLoan.Loan, error) {
			return nil, errors.New("no rows")
		},
	}
	apprs := &approvalmock.Repo{}
	tx := &uowmock.UoW{
		WithinTxFn: func(ctx context.Context, fn func(r uow.Repos) error) error {
			return fn(uow.Repos{Loans: loans, Approvals: apprs})
		},
	}
	uc := ucApproval.NewUsecase(loans, apprs, tx)
	h := NewApprovalHandler(uc)

	body := map[string]any{
		"photo_url":             "https://cdn/img.jpg",
		"validator_employee_id": strings.Repeat("a", 32),
		"approval_date":         "2025-09-06",
	}
	req := httptest.NewRequest(stdhttp.MethodPost, "/loans/LN-404/approve", mustJSON(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loan_id")
	c.SetParamValues("LN-404")

	if err := h.ApproveLoan(c); err != nil {
		t.Fatalf("ApproveLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var er ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &er)
	if er.Error != "loan not found" {
		t.Fatalf("error = %q, want %q", er.Error, "loan not found")
	}
}

func TestApproveLoan_AlreadyApproved(t *testing.T) {
	e := newEchoWithValidator()

	loans := &loanmock.Repo{
		GetByLoanIDForUpdateFn: func(ctx context.Context, loanID string) (*domainLoan.Loan, error) {
			return &domainLoan.Loan{ID: 1, LoanID: loanID, State: domainLoan.StateApproved}, nil
		},
	}
	apprs := &approvalmock.Repo{}
	tx := &uowmock.UoW{
		WithinTxFn: func(ctx context.Context, fn func(r uow.Repos) error) error {
			return fn(uow.Repos{Loans: loans, Approvals: apprs})
		},
	}
	uc := ucApproval.NewUsecase(loans, apprs, tx)
	h := NewApprovalHandler(uc)

	body := map[string]any{
		"photo_url":             "https://cdn/img.jpg",
		"validator_employee_id": strings.Repeat("a", 32),
		"approval_date":         "2025-09-06",
	}
	req := httptest.NewRequest(stdhttp.MethodPost, "/loans/LN-409/approve", mustJSON(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loan_id")
	c.SetParamValues("LN-409")

	if err := h.ApproveLoan(c); err != nil {
		t.Fatalf("ApproveLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	var er ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &er)
	if er.Error != "loan already approved" {
		t.Fatalf("error = %q, want %q", er.Error, "loan already approved")
	}
}

func TestApproveLoan_InvalidTransition(t *testing.T) {
	e := newEchoWithValidator()

	loans := &loanmock.Repo{
		GetByLoanIDForUpdateFn: func(ctx context.Context, loanID string) (*domainLoan.Loan, error) {
			return &domainLoan.Loan{ID: 1, LoanID: loanID, State: domainLoan.StateRejected}, nil
		},
	}
	apprs := &approvalmock.Repo{}
	tx := &uowmock.UoW{
		WithinTxFn: func(ctx context.Context, fn func(r uow.Repos) error) error {
			return fn(uow.Repos{Loans: loans, Approvals: apprs})
		},
	}
	uc := ucApproval.NewUsecase(loans, apprs, tx)
	h := NewApprovalHandler(uc)

	body := map[string]any{
		"photo_url":             "https://cdn/img.jpg",
		"validator_employee_id": strings.Repeat("a", 32),
		"approval_date":         "2025-09-06",
	}
	req := httptest.NewRequest(stdhttp.MethodPost, "/loans/LN-BAD/approve", mustJSON(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loan_id")
	c.SetParamValues("LN-BAD")

	if err := h.ApproveLoan(c); err != nil {
		t.Fatalf("ApproveLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	var er ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &er)
	if er.Error != "loan not in a state that can be approved" {
		t.Fatalf("error = %q, want %q", er.Error, "loan not in a state that can be approved")
	}
}

func TestApproveLoan_GenericError(t *testing.T) {
	e := newEchoWithValidator()

	loans := &loanmock.Repo{
		GetByLoanIDForUpdateFn: func(ctx context.Context, loanID string) (*domainLoan.Loan, error) {
			return &domainLoan.Loan{ID: 7, LoanID: loanID, State: domainLoan.StateProposed}, nil
		},
		SaveFn: func(ctx context.Context, l *domainLoan.Loan) error { return nil },
	}
	apprs := &approvalmock.Repo{
		GetByLoanIDFn: func(ctx context.Context, id uint64) (*domainApproval.Approval, error) {
			return nil, gorm.ErrRecordNotFound
		},
		CreateFn: func(ctx context.Context, a *domainApproval.Approval) error {
			return errors.New("insert failed")
		},
	}
	tx := &uowmock.UoW{
		WithinTxFn: func(ctx context.Context, fn func(r uow.Repos) error) error {
			return fn(uow.Repos{Loans: loans, Approvals: apprs})
		},
	}
	uc := ucApproval.NewUsecase(loans, apprs, tx)
	h := NewApprovalHandler(uc)

	body := map[string]any{
		"photo_url":             "https://cdn/img.jpg",
		"validator_employee_id": strings.Repeat("a", 32),
		"approval_date":         "2025-09-06",
	}
	req := httptest.NewRequest(stdhttp.MethodPost, "/loans/LN-ERR/approve", mustJSON(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("loan_id")
	c.SetParamValues("LN-ERR")

	if err := h.ApproveLoan(c); err != nil {
		t.Fatalf("ApproveLoan error: %v", err)
	}
	if rec.Code != stdhttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var er ErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &er)
	if er.Error != "insert failed" {
		t.Fatalf("error = %q, want %q", er.Error, "insert failed")
	}
}
