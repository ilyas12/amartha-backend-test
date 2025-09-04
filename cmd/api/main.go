package main

import (
	"log"
	"time"

	"amartha-backend-test/internal/infrastructure/cache"

	httpadp "amartha-backend-test/internal/adapter/http"
	idmp "amartha-backend-test/internal/adapter/middleware"
	"amartha-backend-test/internal/config"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := config.Load()
	h := httpadp.NewHandler()

	rdb, err := cache.OpenRedis(cfg.RedisAddr, cfg.RedisDB)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger(), middleware.Recover())
	// global idempotency for mutating methods, TTL in seconds
	e.Use(idmp.IdempotencyMiddleware(rdb, time.Duration(cfg.IdempTTLSecs)*time.Second))

	// routes
	e.GET("/health", h.Health)

	addr := ":" + cfg.AppPort
	log.Printf("listening on %s", addr)
	if err := e.Start(addr); err != nil {
		log.Fatal(err)
	}
}
