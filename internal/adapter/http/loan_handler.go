package http

import (
	"net/http"

	"amartha-backend-test/internal/usecase/loan"

	"github.com/labstack/echo/v4"
)

type LoanHandler struct{ uc *loan.Usecase }

func NewLoanHandler(uc *loan.Usecase) *LoanHandler { return &LoanHandler{uc: uc} }

type createLoanReq struct {
	BorrowerID string  `json:"borrower_id"`
	Principal  float64 `json:"principal"`
	Rate       float64 `json:"rate"`
	ROI        float64 `json:"roi"`
}

func (h *LoanHandler) CreateLoan(c echo.Context) error {
	var req createLoanReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
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
