package http

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
	})
}
