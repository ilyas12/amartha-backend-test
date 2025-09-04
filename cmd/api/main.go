package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	httpadp "amartha-backend-test/internal/adapter/http"
	"amartha-backend-test/internal/config"
)

func main() {
	cfg := config.Load()
	h := httpadp.NewHandler()

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger(), middleware.Recover())

	// routes
	e.GET("/health", h.Health)

	addr := ":" + cfg.AppPort
	log.Printf("listening on %s", addr)
	if err := e.Start(addr); err != nil {
		log.Fatal(err)
	}
}
