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
	usecaseApproval "amartha-backend-test/internal/usecase/approval"
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

	loanRepo := repomysql.NewLoanRepository(gormDB)
	ucLoan := usecaseLoan.NewUsecase(loanRepo)
	approvalRepo := repomysql.NewApprovalRepository(gormDB)
	// UoW (one generic Unit-of-Work for all flows)
	uow := repomysql.NewGormUoW(gormDB)

	// Usecase (inject repos + UoW)
	ucApproval := usecaseApproval.NewUsecase(loanRepo, approvalRepo, uow)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger(), middleware.Recover())
	e.Validator = httpadp.NewValidator()
	e.Logger.SetOutput(os.Stdout)
	log.SetOutput(os.Stdout)
	// global idempotency for mutating methods, TTL in seconds
	e.Use(idmp.IdempotencyMiddleware(rdb, time.Duration(cfg.IdempTTLSecs)*time.Second))
	h := httpadp.NewHandler()
	hLoan := httpadp.NewLoanHandler(ucLoan)
	hApproval := httpadp.NewApprovalHandler(ucApproval)

	// routes
	e.GET("/health", h.Health)

	e.POST("/loans", hLoan.CreateLoan)
	e.POST("/loans/:loan_id/approve", hApproval.ApproveLoan)
	e.GET("/loans/:loan_id", hLoan.GetLoan)

	addr := ":" + cfg.AppPort
	log.Printf("listening on %s", addr)
	if err := e.Start(addr); err != nil {
		log.Fatal(err)
	}
}
