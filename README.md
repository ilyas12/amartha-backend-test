# Loan Services — Clean Architecture

This repository is the Loan Service assessment implementation: a clean-architecture Go API built with Echo v4, GORM (MySQL), and Redis-backed idempotency. It delivers the required loan lifecycle—proposed → approved → invested → disbursed—with forward-only transitions, and 32-character public identifiers (no DB-generated UUIDs exposed).

## Folder structure

```
.
├─ cmd/                     # Application entrypoints (composition root)
│  └─ api/                  # Main HTTP service wiring (Echo, routes, DI)
├─ db/                      # Database assets
│  └─ migrations/           # SQL migrations (schema, indexes, seeds)
├─ internal/                # Application code (Clean Architecture)
│  ├─ domain/               # Core domain model & repository interfaces
│  │  └─ loan/              # Loan entities and contracts
│  ├─ usecase/              # Business rules and orchestration
│  │  └─ loan/              # Loan lifecycle use cases
│  ├─ adapter/              # I/O adapters (framework & external tech)
│  │  ├─ http/              # Echo handlers (transport layer)
│  │  ├─ repository/        # Persistence adapters
│  │  │  └─ mysql/          # GORM implementation of repositories
│  │  └─ middleware/        # Cross-cutting (e.g., Redis idempotency)
│  └─ infrastructure/       # Runtime infrastructure clients
│     ├─ db/                # GORM connector (MySQL)
│     └─ cache/             # Redis client
├─ pkg/                     # Shared, framework-agnostic utilities
│  └─ id/                   # 32-char public ID generator
├─ .env.example             # Example environment configuration
├─ docker-compose.yml       # Local MySQL & Redis for development
└─ README.md

```

### Layering (Clean Architecture)

* **domain/**
  Pure domain model and contracts (entities, repository interfaces). No framework imports here This is the **center** of the app.
* **usecase/**
  Application/business logic. Orchestrates domain workflows and enforces invariants (state transitions, investment totals, required fields).
* **adapter/**
  * **repository/mysql/**: Implements `domain.Repository` using **GORM**. Contains SQL/GORM specifics and transactional boundaries.
  * **http/**: Echo handlers that translate HTTP ↔ DTOs, calling use cases.
  * **middleware/**: Cross-cutting concerns like **idempotency**.
* **infrastructure/**
  Concrete runtime dependencies: GORM connector (MySQL) and Redis client.
* **cmd/api/**
  Composition root: build dependencies, assemble routes, set middleware, and start the server.
## Request flow (end-to-end)

1. **Echo handler** parses/validates input and enforces idempotency (middleware).
2. Calls **usecase** (e.g., `Approve`, `Invest`), passing a context + DTO.
3. Usecase opens a **transaction** via repository `Tx` and loads aggregates with **FOR UPDATE** where needed.
4. Usecase applies **business rules** (state checks, totals, required fields).
5. Repository persists changes through **GORM**; usecase returns a DTO.
6. Handler formats JSON response.

## Endpoints (current)

* `GET  /health`

> **IDs**: All public identifiers are **32-char lowercase hex** strings (no database-generated UUIDs exposed). Internal numeric PKs are never returned.

## Idempotency (Redis)

* Applied to **mutating** methods (POST/PUT/PATCH/DELETE) by global middleware.
* Requires header `Idempotency-Key`.
* Stores `{code, body, body_sha256}` in Redis with TTL (`IDEMPOTENCY_TTL_SECONDS`).
* Same key + **same body** → previous response **replayed**.
* Same key + **different body** → **409 Conflict**.
* “In progress” duplicate (lock window) → **409 Conflict**.

## Environment variables

Create `.env` from `.env.example`:

```
APP_PORT=8080

# MySQL
MYSQL_HOST=127.0.0.1
MYSQL_PORT=3306
MYSQL_DB=app
MYSQL_USER=app
MYSQL_PASS=app

# Redis
REDIS_ADDR=127.0.0.1:6379
REDIS_DB=0

# Idempotency
IDEMPOTENCY_TTL_SECONDS=300
```

`config.MySQLDSN()` formats the DSN with `parseTime=true` and `utf8mb4`.

## Quickstart

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api             # runs API on :8080

curl -s localhost:8080/health
```

## Conventions & notes

* **Public IDs**: 32-char lowercase hex (generated in `pkg/id`).
* **Dates**: ISO-8601 (`YYYY-MM-DD`) where used; timestamps stored UTC.
* **Security**: This sample expects an API Gateway to handle AuthN/Z; handlers focus on business logic.
* **Transactions**: Multi-step updates run inside `repo.Tx(...)` with row locking for consistency.

---

**Note:** Public identifiers are fixed-length **string-32**; internal numeric PKs are not exposed.
