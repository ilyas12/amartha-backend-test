package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestHealth_ReturnsOKWithRFC3339NanoUTC(t *testing.T) {
	e := echo.New()
	h := NewHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	start := time.Now().UTC()

	if err := h.Health(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// Status code
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	// Content-Type
	ct := rec.Header().Get(echo.HeaderContentType)
	if !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	// Body JSON
	var body struct {
		Status string `json:"status"`
		Time   string `json:"time"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v; raw=%s", err, rec.Body.String())
	}

	if body.Status != "ok" {
		t.Fatalf(`expected status "ok", got %q`, body.Status)
	}

	// Time is RFC3339Nano and UTC (with 'Z')
	parsed, err := time.Parse(time.RFC3339Nano, body.Time)
	if err != nil {
		t.Fatalf("time not RFC3339Nano: %v (value=%q)", err, body.Time)
	}
	if parsed.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", parsed.Location())
	}
	// Freshness: should be close to now (within a few seconds)
	now := time.Now().UTC()
	if parsed.Before(start.Add(-2*time.Second)) || parsed.After(now.Add(2*time.Second)) {
		t.Fatalf("time not within expected window: parsed=%v start=%v now=%v", parsed, start, now)
	}
}
