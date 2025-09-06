package http

import (
	"errors"
	"net/http"
	"time"

	domainLoan "amartha-backend-test/internal/domain/loan"
	ucapproval "amartha-backend-test/internal/usecase/approval"

	"github.com/labstack/echo/v4"
)

type ApprovalHandler struct{ uc *ucapproval.Usecase }

func NewApprovalHandler(uc *ucapproval.Usecase) *ApprovalHandler { return &ApprovalHandler{uc: uc} }

type approveLoanReq struct {
	PhotoURL            string `json:"photo_url"             validate:"required,url"`
	ValidatorEmployeeID string `json:"validator_employee_id" validate:"required,hex32"`
	// canonical date `YYYY-MM-DD` (matches MySQL DATE)
	ApprovalDate string `json:"approval_date"               validate:"required,datetime=2006-01-02"`
}

func (h *ApprovalHandler) ApproveLoan(c echo.Context) error {
	// path param
	loanID := c.Param("loan_id")
	if loanID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "missing loan_id path param"})
	}

	// bind + validate
	var req approveLoanReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid body"})
	}
	// Validate request body
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "validation failed",
			Details: ToFieldErrors(err),
		})
	}

	// parse approval_date
	ad, err := time.Parse("2006-01-02", req.ApprovalDate)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "validation failed",
			Details: []FieldError{{Field: "ApprovalDate", Message: "must be YYYY-MM-DD"}},
		})
	}

	// call usecase
	dto, uerr := h.uc.Approve(
		c.Request().Context(),
		ucapproval.ApproveInput{
			LoanID:              loanID,
			PhotoURL:            req.PhotoURL,
			ValidatorEmployeeID: req.ValidatorEmployeeID,
			ApprovalDate:        ad,
		},
	)
	if uerr != nil {
		switch {
		case errors.Is(uerr, domainLoan.ErrNotFound):
			return c.JSON(http.StatusNotFound, ErrorResponse{Error: "loan not found"})
		case errors.Is(uerr, domainLoan.ErrAlreadyApproved):
			return c.JSON(http.StatusConflict, ErrorResponse{Error: "loan already approved"})
		case errors.Is(uerr, domainLoan.ErrInvalidTransition):
			return c.JSON(http.StatusConflict, ErrorResponse{Error: "loan not in a state that can be approved"})
		default:
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: uerr.Error()})
		}
	}

	// success
	return c.JSON(http.StatusOK, dto)
}
