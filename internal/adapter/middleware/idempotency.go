package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type idempEntry struct {
	InProgress bool      `json:"in_progress"`
	Code       int       `json:"code"`
	Body       []byte    `json:"body"`
	BodySHA256 string    `json:"body_sha256"`
	CreatedAt  time.Time `json:"created_at"`
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

func bodyHash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// IdempotencyMiddleware enforces idempotency for mutating methods using Redis.
func IdempotencyMiddleware(rdb *redis.Client, ttl time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			m := c.Request().Method
			if m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions {
				return next(c)
			}

			idKey := c.Request().Header.Get("Idempotency-Key")
			if idKey == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing Idempotency-Key"})
			}

			var bodyBytes []byte
			if c.Request().Body != nil {
				bodyBytes, _ = io.ReadAll(c.Request().Body)
			}
			c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			bhash := bodyHash(bodyBytes)

			redisKey := "idemp:" + strings.ToLower(m) + ":" + c.Path() + ":" + idKey
			ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
			defer cancel()

			entry := idempEntry{InProgress: true, BodySHA256: bhash, CreatedAt: time.Now()}
			pl, _ := json.Marshal(entry)
			ok, err := rdb.SetNX(ctx, redisKey, pl, 60*time.Second).Result()
			if err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "idempotency store unavailable"})
			}
			if !ok {
				var cur idempEntry
				v, err := rdb.Get(ctx, redisKey).Bytes()
				if err == nil {
					_ = json.Unmarshal(v, &cur)
				}
				if cur.BodySHA256 != "" && cur.BodySHA256 != bhash {
					return c.JSON(http.StatusConflict, map[string]string{"error": "idempotency key re-used with different body"})
				}
				if !cur.InProgress && cur.Code != 0 {
					return c.Blob(cur.Code, echo.MIMEApplicationJSON, cur.Body)
				}
				return c.JSON(http.StatusConflict, map[string]string{"error": "idempotent request in progress"})
			}

			rec := &respRecorder{w: c.Response().Writer, buf: &bytes.Buffer{}, code: http.StatusOK}
			c.Response().Writer = rec
			err = next(c)
			if err != nil {
				c.Error(err)
			}

			final := idempEntry{InProgress: false, Code: rec.code, Body: rec.buf.Bytes(), BodySHA256: bhash, CreatedAt: time.Now()}
			fv, _ := json.Marshal(final)
			_ = rdb.Set(context.Background(), redisKey, fv, ttl).Err()
			return nil
		}
	}
}
