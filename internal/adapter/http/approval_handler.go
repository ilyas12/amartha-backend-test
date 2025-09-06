package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type ApprovalHandler struct{}

func NewApprovalHandler() *ApprovalHandler { return &ApprovalHandler{} }

type approveLoanReq struct {
	PhotoURL            string `json:"photo_url"              validate:"required,url"`
	ValidatorEmployeeID string `json:"validator_employee_id"  validate:"required,hex32"`
	// Accept canonical date `YYYY-MM-DD` (aligns with schema DATE)
	ApprovalDate string `json:"approval_date"          validate:"required,datetime=2006-01-02"`
}

func (h *ApprovalHandler) ApproveLoan(c echo.Context) error {
	// Validate path param
	loanID := c.Param("loan_id")
	if loanID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "missing loan_id path param"})
	}
	// Bind + validate body payload JSON
	var req approveLoanReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid body"})
	}
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "validation failed",
			Details: ToFieldErrors(err),
		})
	}
	// Call Usecase Approve Loan
	dto := map[string]interface{}{}
	// Map domain errors → HTTP codes

	return c.JSON(http.StatusOK, dto)
}
