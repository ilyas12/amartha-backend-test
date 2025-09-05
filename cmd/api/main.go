package main

import (
	"amartha-backend-test/internal/config"
	"amartha-backend-test/internal/infrastructure/cache"
	"log"
	"os"
	"time"

	httpadp "amartha-backend-test/internal/adapter/http"
	idmp "amartha-backend-test/internal/adapter/middleware"
	repomysql "amartha-backend-test/internal/adapter/repository/mysql"
	dbinfra "amartha-backend-test/internal/infrastructure/db"
	usecaseLoan "amartha-backend-test/internal/usecase/loan"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	_ = godotenv.Load(".env")
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("bad config: %v", err)
	}

	gormDB, err := dbinfra.OpenGorm(cfg.MySQLDSN())
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	rdb, err := cache.OpenRedis(cfg.RedisAddr, cfg.RedisDB)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	repo := repomysql.NewLoanRepository(gormDB)
	uc := usecaseLoan.NewUsecase(repo)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger(), middleware.Recover())
	e.Validator = httpadp.NewValidator()
	e.Logger.SetOutput(os.Stdout)
	log.SetOutput(os.Stdout)
	// global idempotency for mutating methods, TTL in seconds
	e.Use(idmp.IdempotencyMiddleware(rdb, time.Duration(cfg.IdempTTLSecs)*time.Second))
	h := httpadp.NewHandler()
	hLoan := httpadp.NewLoanHandler(uc)
	// routes
	e.GET("/health", h.Health)

	e.POST("/loans", hLoan.CreateLoan)
	e.GET("/loans/:loan_id", hLoan.GetLoan)

	addr := ":" + cfg.AppPort
	log.Printf("listening on %s", addr)
	if err := e.Start(addr); err != nil {
		log.Fatal(err)
	}
}
