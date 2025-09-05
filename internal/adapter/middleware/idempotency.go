package middleware

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

const (
	// How long we hold the "in-progress" lock before it must be refreshed by finishing the handler.
	provisionalLockTTL = 60 * time.Second
	// Allowed client/server clock skew for Ax-Request-At (in UTC).
	maxClockSkew = 10 * time.Minute
)

// ---- Data types ----
type idempEntry struct {
	InProgress  bool      `json:"in_progress"`
	Code        int       `json:"code"`
	Body        []byte    `json:"body"`
	BodySHA256  string    `json:"body_sha256"`
	RequestID   string    `json:"request_id"`
	RequestAtMS int64     `json:"request_at_ms"`
	CreatedAt   time.Time `json:"created_at"`
}

type respRecorder struct {
	w    http.ResponseWriter
	buf  *bytes.Buffer
	code int
}

func (r *respRecorder) Header() http.Header { return r.w.Header() }
func (r *respRecorder) Write(b []byte) (int, error) {
	if r.buf != nil {
		r.buf.Write(b)
	}
	return r.w.Write(b)
}
func (r *respRecorder) WriteHeader(statusCode int) { r.code = statusCode; r.w.WriteHeader(statusCode) }

// IdempotencyMiddleware: key = method + route + user id + request id
// Ax-Request-At **must** be epoch (seconds or ms) OR RFC3339/RFC3339Nano **with** timezone (Z or ±HH:MM).
func IdempotencyMiddleware(rdb *redis.Client, ttl time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			method := req.Method

			// Only enforce on mutating methods
			switch method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				return next(c)
			}

			// Headers Validation
			reqID := strings.TrimSpace(req.Header.Get("Ax-Request-Id"))
			if reqID == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing Ax-Request-Id"})
			}
			if !validReqID(reqID) {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid Ax-Request-Id format"})
			}

			reqAt, err := parseAxRequestAt(req.Header.Get("Ax-Request-At"))
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}
			now := nowUTC()
			if reqAt.Before(now.Add(-maxClockSkew)) || reqAt.After(now.Add(maxClockSkew)) {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Ax-Request-At too skewed"})
			}

			borrowerID := strings.TrimSpace(req.Header.Get("Ax-Borrower-Id"))
			if borrowerID == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing Ax-Borrower-Id"})
			}
			if !reHex32.MatchString(borrowerID) {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid Ax-Borrower-Id"})
			}

			// Buffer & hash body
			var body []byte
			if req.Body != nil {
				body, _ = io.ReadAll(req.Body)
			}
			req.Body = io.NopCloser(bytes.NewBuffer(body))
			bhash := bodyHash(body)

			// 3) Provisional lock key
			key := buildKey(method, c.Path(), borrowerID, reqID)
			ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
			defer cancel()

			entry := idempEntry{
				InProgress:  true,
				BodySHA256:  bhash,
				RequestID:   reqID,
				RequestAtMS: reqAt.UnixMilli(),
				CreatedAt:   nowUTC(),
			}
			ok, err := provisionalSet(ctx, rdb, key, entry)
			if err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "idempotency store unavailable"})
			}
			if !ok {
				// Key exists: body must match, and we may be able to replay
				cur, errLoad := loadEntry(ctx, rdb, key)
				if errLoad != nil {
					log.Printf("Failed To get Load Data %s in Idempotency %s", key, errLoad.Error())
				}

				if cur.BodySHA256 != "" && cur.BodySHA256 != bhash {
					return c.JSON(http.StatusConflict, map[string]string{"error": "Ax-Request-Id reused with different body"})
				}
				if !cur.InProgress && cur.Code != 0 && len(cur.Body) > 0 {
					return c.Blob(cur.Code, echo.MIMEApplicationJSON, cur.Body)
				}
				return c.JSON(http.StatusConflict, map[string]string{"error": "request is already in progress"})
			}

			// 4) Call next and record final response
			rec := &respRecorder{w: c.Response().Writer, buf: &bytes.Buffer{}, code: http.StatusOK}
			c.Response().Writer = rec
			if err := next(c); err != nil {
				c.Error(err)
			}

			final := idempEntry{
				InProgress:  false,
				Code:        rec.code,
				Body:        rec.buf.Bytes(),
				BodySHA256:  bhash,
				RequestID:   reqID,
				RequestAtMS: reqAt.UnixMilli(),
				CreatedAt:   nowUTC(),
			}
			_ = saveFinal(context.Background(), rdb, key, final, ttl)
			return nil
		}
	}
}
