package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func main() {
	// Initialize Echo
	e := echo.New()

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"message": "service is healthy",
		})
	})

	// Start server
	e.Logger.Fatal(e.Start(":8080"))
}
