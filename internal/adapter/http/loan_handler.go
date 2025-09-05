package http

import (
	"net/http"

	"amartha-backend-test/internal/usecase/loan"

	"github.com/labstack/echo/v4"
)

type LoanHandler struct{ uc *loan.Usecase }

func NewLoanHandler(uc *loan.Usecase) *LoanHandler { return &LoanHandler{uc: uc} }

type createLoanReq struct {
	BorrowerID string `json:"borrower_id" validate:"required,hex32"`
	// principal: integer in [5000000 .. 100000000]
	Principal float64 `json:"principal"  validate:"required,intlike,gte=5000000,lte=100000000"`
	// rate: [1.29 .. 2.99] with max 2 decimals
	Rate float64 `json:"rate"       validate:"required,dec2,gte=1.29,lte=2.99"`
	// roi: [0.90 .. 1.29] with max 2 decimals
	ROI float64 `json:"roi"        validate:"required,dec2,gte=0.90,lte=1.29"`
}

func (h *LoanHandler) CreateLoan(c echo.Context) error {
	var req createLoanReq
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

	dto, err := h.uc.Create(c.Request().Context(), loan.CreateLoanInput(req))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, dto)
}

func (h *LoanHandler) GetLoan(c echo.Context) error {
	loanID := c.Param("loan_id")
	dto, err := h.uc.Get(c.Request().Context(), loanID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	return c.JSON(http.StatusOK, dto)
}
