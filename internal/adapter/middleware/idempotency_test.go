package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

// helper: new Echo with the middleware and a simple route
func setupEcho(rdb *redis.Client, ttl time.Duration, handler echo.HandlerFunc) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.Use(IdempotencyMiddleware(rdb, ttl))
	e.POST("/loans", handler)
	e.GET("/loans", handler) // for non-mutating bypass test
	return e
}

func mkJSONBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func doReq(t *testing.T, e *echo.Echo, method, path string, body io.Reader, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func newMiniredisClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, rdb
}

// simple handler to exercise respRecorder capture & saveFinal
func okCreatedHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]any{"ok": true})
}

func Test_BypassOnGET_NoHeadersRequired(t *testing.T) {
	mr, rdb := newMiniredisClient(t)
	defer mr.Close()
	e := setupEcho(rdb, 30*time.Second, func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "get ok"})
	})
	rec := doReq(t, e, http.MethodGet, "/loans", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func Test_ValidationFailures(t *testing.T) {
	mr, rdb := newMiniredisClient(t)
	defer mr.Close()
	e := setupEcho(rdb, 30*time.Second, okCreatedHandler)

	// base headers (valid) to start from
	valid := map[string]string{
		"Ax-Request-Id":  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 32-hex (valid)
		"Ax-Request-At":  time.Now().UTC().Format(time.RFC3339),
		"Ax-Borrower-Id": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}

	// missing Ax-Request-Id
	h := map[string]string{
		"Ax-Request-At":  valid["Ax-Request-At"],
		"Ax-Borrower-Id": valid["Ax-Borrower-Id"],
	}
	rec := doReq(t, e, http.MethodPost, "/loans", mkJSONBody(t, map[string]int{"x": 1}), h)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing Ax-Request-Id => want 400, got %d", rec.Code)
	}

	// invalid Ax-Request-Id
	h = map[string]string{
		"Ax-Request-Id":  "NOT-VALID",
		"Ax-Request-At":  valid["Ax-Request-At"],
		"Ax-Borrower-Id": valid["Ax-Borrower-Id"],
	}
	rec = doReq(t, e, http.MethodPost, "/loans", mkJSONBody(t, map[string]int{"x": 1}), h)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid Ax-Request-Id => want 400, got %d", rec.Code)
	}

	// invalid Ax-Request-At format
	h = map[string]string{
		"Ax-Request-Id":  valid["Ax-Request-Id"],
		"Ax-Request-At":  "not-a-time",
		"Ax-Borrower-Id": valid["Ax-Borrower-Id"],
	}
	rec = doReq(t, e, http.MethodPost, "/loans", mkJSONBody(t, map[string]int{"x": 1}), h)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid Ax-Request-At => want 400, got %d", rec.Code)
	}

	// Ax-Request-At too skewed (past)
	h = map[string]string{
		"Ax-Request-Id":  valid["Ax-Request-Id"],
		"Ax-Request-At":  time.Now().UTC().Add(-maxClockSkew - time.Minute).Format(time.RFC3339),
		"Ax-Borrower-Id": valid["Ax-Borrower-Id"],
	}
	rec = doReq(t, e, http.MethodPost, "/loans", mkJSONBody(t, map[string]int{"x": 1}), h)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("Ax-Request-At skew => want 400, got %d", rec.Code)
	}

	// missing Ax-Borrower-Id
	h = map[string]string{
		"Ax-Request-Id": valid["Ax-Request-Id"],
		"Ax-Request-At": valid["Ax-Request-At"],
	}
	rec = doReq(t, e, http.MethodPost, "/loans", mkJSONBody(t, map[string]int{"x": 1}), h)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing Ax-Borrower-Id => want 400, got %d", rec.Code)
	}

	// invalid Ax-Borrower-Id
	h = map[string]string{
		"Ax-Request-Id":  valid["Ax-Request-Id"],
		"Ax-Request-At":  valid["Ax-Request-At"],
		"Ax-Borrower-Id": "not32hex",
	}
	rec = doReq(t, e, http.MethodPost, "/loans", mkJSONBody(t, map[string]int{"x": 1}), h)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid Ax-Borrower-Id => want 400, got %d", rec.Code)
	}
}

func Test_HappyPath_Then_Replay(t *testing.T) {
	mr, rdb := newMiniredisClient(t)
	defer mr.Close()
	e := setupEcho(rdb, 2*time.Minute, okCreatedHandler)

	h := map[string]string{
		"Ax-Request-Id":  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"Ax-Request-At":  time.Now().UTC().Format(time.RFC3339),
		"Ax-Borrower-Id": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	body := mkJSONBody(t, map[string]any{"principal": 5000000})

	// First request -> goes through handler (201, {"ok":true})
	rec1 := doReq(t, e, http.MethodPost, "/loans", body, h)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first request => want 201, got %d, body: %s", rec1.Code, rec1.Body.String())
	}

	// Second request with SAME headers & body -> replay stored response (also 201)
	rec2 := doReq(t, e, http.MethodPost, "/loans", mkJSONBody(t, map[string]any{"principal": 5000000}), h)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("replay => want 201, got %d, body: %s", rec2.Code, rec2.Body.String())
	}
	if rec1.Body.String() != rec2.Body.String() {
		t.Fatalf("replay body mismatch: %q vs %q", rec1.Body.String(), rec2.Body.String())
	}
}

func Test_Conflict_When_InProgress(t *testing.T) {
	mr, rdb := newMiniredisClient(t)
	defer mr.Close()
	e := setupEcho(rdb, 2*time.Minute, okCreatedHandler)

	method := http.MethodPost
	path := "/loans"
	reqID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	borrowerID := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	body := []byte(`{"x":1}`)

	// Seed provisional "in-progress" entry (so SetNX will fail and loadEntry sees InProgress=true)
	key := buildKey(method, path, borrowerID, reqID)
	entry := idempEntry{
		InProgress:  true,
		BodySHA256:  bodyHash(body),
		RequestID:   reqID,
		RequestAtMS: time.Now().UnixMilli(),
		CreatedAt:   time.Now().UTC(),
	}
	// Store into Redis as JSON via the same helper used by middleware
	if ok, err := provisionalSet(context.Background(), rdb, key, entry); err != nil || !ok {
		t.Fatalf("seed provisional failed, ok=%v err=%v", ok, err)
	}

	h := map[string]string{
		"Ax-Request-Id":  reqID,
		"Ax-Request-At":  time.Now().UTC().Format(time.RFC3339),
		"Ax-Borrower-Id": borrowerID,
	}
	rec := doReq(t, e, method, path, bytes.NewReader(body), h)

	if rec.Code != http.StatusConflict {
		t.Fatalf("in-progress => want 409, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func Test_Conflict_When_SameReqID_DifferentBody(t *testing.T) {
	mr, rdb := newMiniredisClient(t)
	defer mr.Close()
	e := setupEcho(rdb, 2*time.Minute, okCreatedHandler)

	method := http.MethodPost
	path := "/loans"
	reqID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	borrowerID := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	body1 := []byte(`{"x":1}`)
	body2 := []byte(`{"x":2}`)

	// Seed FINAL entry with body hash of body1 (so SetNX fails, loadEntry returns final,
	// and branch detects different body -> 409)
	key := buildKey(method, path, borrowerID, reqID)
	final := idempEntry{
		InProgress:  false,
		Code:        http.StatusCreated,
		Body:        []byte(`{"ok":true}`), // any stored body
		BodySHA256:  bodyHash(body1),
		RequestID:   reqID,
		RequestAtMS: time.Now().UnixMilli(),
		CreatedAt:   time.Now().UTC(),
	}
	if err := saveFinal(context.Background(), rdb, key, final, time.Minute*5); err != nil {
		t.Fatalf("seed final failed: %v", err)
	}

	h := map[string]string{
		"Ax-Request-Id":  reqID,
		"Ax-Request-At":  time.Now().UTC().Format(time.RFC3339),
		"Ax-Borrower-Id": borrowerID,
	}
	rec := doReq(t, e, method, path, bytes.NewReader(body2), h)

	if rec.Code != http.StatusConflict {
		t.Fatalf("different body same reqID => want 409, got %d", rec.Code)
	}
}

func Test_StoreUnavailable_Returns503(t *testing.T) {
	// Create a client that points to a closed address → SetNX error
	// (fast fail vs waiting the whole context)
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	e := setupEcho(rdb, time.Minute, okCreatedHandler)

	h := map[string]string{
		"Ax-Request-Id":  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"Ax-Request-At":  time.Now().UTC().Format(time.RFC3339),
		"Ax-Borrower-Id": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	rec := doReq(t, e, http.MethodPost, "/loans", bytes.NewReader([]byte(`{}`)), h)

	if rec.Code != http.StatusServiceUnavailable && rec.Code != http.StatusBadGateway {
		// expect 503 from the middleware path
		t.Fatalf("store unavailable => want 503-ish, got %d", rec.Code)
	}
}
