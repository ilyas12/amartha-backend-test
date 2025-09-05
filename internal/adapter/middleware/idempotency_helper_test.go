package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// --- small helpers ---

func newMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, rdb
}

// --- bodyHash ---

func Test_bodyHash(t *testing.T) {
	data := []byte("hello world")
	got := bodyHash(data)

	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])

	if got != want {
		t.Fatalf("bodyHash mismatch: got %s want %s", got, want)
	}
}

// --- nowUTC ---

func Test_nowUTC(t *testing.T) {
	u := nowUTC()
	if u.Location() != time.UTC {
		t.Fatalf("nowUTC must be UTC, got %v", u.Location())
	}
	if d := time.Since(u); d < 0 || d > 2*time.Second {
		t.Fatalf("nowUTC too far from now: %v", d)
	}
}

// --- buildKey ---

func Test_buildKey(t *testing.T) {
	k := buildKey("POST", "/loans", strings.Repeat("b", 32), strings.Repeat("a", 32))
	wantPrefix := "idemp:ax:post:/loans:"
	if !strings.HasPrefix(k, wantPrefix) {
		t.Fatalf("buildKey prefix mismatch: got %q want prefix %q", k, wantPrefix)
	}
	if !strings.Contains(k, ":"+strings.Repeat("b", 32)+":") || !strings.HasSuffix(k, strings.Repeat("a", 32)) {
		t.Fatalf("buildKey missing borrower/request segments: %q", k)
	}
}

// --- validReqID ---

func Test_validReqID(t *testing.T) {
	t.Run("accepts uuid v4 and 32-hex", func(t *testing.T) {
		valid := []string{
			"3f9a6a1b-3d54-4fbe-8b3a-6b3e8d6b2c88", // UUID v4 (lowercase)
			strings.Repeat("a", 32),                // 32-char lowercase hex
			"3f9a6a1b3d544fbe8b3a6b3e8d6b2c88",     // 32-char lowercase hex (no dashes)
		}
		for _, s := range valid {
			if !validReqID(s) {
				t.Fatalf("validReqID should accept %q", s)
			}
		}
	})

	t.Run("rejects bad formats", func(t *testing.T) {
		invalid := []string{
			"",
			"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",     // uppercase hex (should reject)
			"3f9a6a1b3d544fbe8b3a6b3e8d6b2c8",      // 31 chars
			"3f9a6a1b3d544fbe8b3a6b3e8d6b2c880",    // 33 chars
			"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",     // non-hex chars
			"3F9A6A1B-3D54-4FBE-8B3A-6B3E8D6B2C88", // uppercase UUID
			"3f9a6a1b-3d54-9fbe-8b3a-6b3e8d6b2c88", // invalid UUID version '9'
		}
		for _, s := range invalid {
			if validReqID(s) {
				t.Fatalf("validReqID should reject %q", s)
			}
		}
	})
}

// --- parseAxRequestAt ---

func Test_parseAxRequestAt_EpochSeconds(t *testing.T) {
	sec := time.Now().UTC().Unix()
	ts, err := parseAxRequestAt(strconv64(sec))
	if err != nil {
		t.Fatalf("parseAxRequestAt sec: %v", err)
	}
	if !ts.Equal(time.Unix(sec, 0).UTC()) {
		t.Fatalf("epoch seconds mismatch: got %v want %v", ts, time.Unix(sec, 0).UTC())
	}
}

func Test_parseAxRequestAt_EpochMillis(t *testing.T) {
	ms := time.Now().UTC().UnixMilli()
	ts, err := parseAxRequestAt(strconv64(ms))
	if err != nil {
		t.Fatalf("parseAxRequestAt ms: %v", err)
	}
	if !ts.Equal(time.UnixMilli(ms).UTC()) {
		t.Fatalf("epoch millis mismatch: got %v want %v", ts, time.UnixMilli(ms).UTC())
	}
}

func Test_parseAxRequestAt_RFC3339_WithTZ(t *testing.T) {
	raw := "2025-09-05T10:00:00+07:00"
	ts, err := parseAxRequestAt(raw)
	if err != nil {
		t.Fatalf("parseAxRequestAt rfc3339: %v", err)
	}
	// 10:00 +07:00 == 03:00 UTC
	want := time.Date(2025, 9, 5, 3, 0, 0, 0, time.UTC)
	if !ts.Equal(want) {
		t.Fatalf("rfc3339 tz mismatch: got %v want %v", ts, want)
	}
}

func Test_parseAxRequestAt_RFC3339_Z(t *testing.T) {
	raw := "2025-09-05T03:00:00Z"
	ts, err := parseAxRequestAt(raw)
	if err != nil {
		t.Fatalf("parseAxRequestAt rfc3339 Z: %v", err)
	}
	want := time.Date(2025, 9, 5, 3, 0, 0, 0, time.UTC)
	if !ts.Equal(want) {
		t.Fatalf("rfc3339 Z mismatch: got %v want %v", ts, want)
	}
}

func Test_parseAxRequestAt_Invalid(t *testing.T) {
	cases := []string{
		"",                    // missing
		"not-a-time",          // garbage
		"2025-09-05T10:00:00", // naive (no TZ)
		"1736123456abc",       // junk
	}
	for _, raw := range cases {
		if _, err := parseAxRequestAt(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

// strconv64 is a tiny helper to avoid importing strconv in multiple places in tests
func strconv64(n int64) string { return strconvFormatInt(n) }

func strconvFormatInt(n int64) string {
	// local minimal int->string to avoid extra imports; fine to use strconv if you prefer
	return fmt.Sprintf("%d", n)
}

// --- Redis helpers: provisionalSet, loadEntry, saveFinal ---

func Test_provisionalSet_LoadEntry(t *testing.T) {
	mr, rdb := newMiniRedis(t)
	defer mr.Close()

	key := "idemp:ax:post:/loans:" + strings.Repeat("b", 32) + ":" + strings.Repeat("a", 32)
	entry := idempEntry{
		InProgress:  true,
		BodySHA256:  bodyHash([]byte(`{"a":1}`)),
		RequestID:   strings.Repeat("a", 32),
		RequestAtMS: time.Now().UnixMilli(),
		CreatedAt:   nowUTC(),
	}

	// First SetNX should succeed
	ok, err := provisionalSet(context.Background(), rdb, key, entry)
	if err != nil || !ok {
		t.Fatalf("provisionalSet 1: ok=%v err=%v", ok, err)
	}

	// TTL should be close to provisionalLockTTL
	ttl := rdb.TTL(context.Background(), key).Val()
	if ttl <= 0 || ttl > provisionalLockTTL {
		t.Fatalf("provisional TTL not set correctly: %v", ttl)
	}

	// Second SetNX should fail (already exists)
	ok, err = provisionalSet(context.Background(), rdb, key, entry)
	if err != nil {
		t.Fatalf("provisionalSet 2 err: %v", err)
	}
	if ok {
		t.Fatalf("provisionalSet 2 should be false, got true")
	}

	// loadEntry returns the same content (spot check a few fields)
	got, err := loadEntry(context.Background(), rdb, key)
	if err != nil {
		t.Fatalf("loadEntry err: %v", err)
	}
	if !got.InProgress || got.RequestID != entry.RequestID || got.BodySHA256 != entry.BodySHA256 {
		t.Fatalf("loaded entry mismatch: %+v vs %+v", got, entry)
	}
}

func Test_saveFinal_Load_TTL(t *testing.T) {
	mr, rdb := newMiniRedis(t)
	defer mr.Close()

	key := "idemp:ax:post:/loans:" + strings.Repeat("b", 32) + ":" + strings.Repeat("a", 32)
	final := idempEntry{
		InProgress:  false,
		Code:        201,
		Body:        []byte(`{"ok":true}`),
		BodySHA256:  bodyHash([]byte(`{"ok":true}`)),
		RequestID:   strings.Repeat("a", 32),
		RequestAtMS: time.Now().UnixMilli(),
		CreatedAt:   nowUTC(),
	}

	ttlWant := 5 * time.Second
	if err := saveFinal(context.Background(), rdb, key, final, ttlWant); err != nil {
		t.Fatalf("saveFinal err: %v", err)
	}

	// Check TTL is set (allow a small drift)
	ttl := rdb.TTL(context.Background(), key).Val()
	if ttl <= 0 || ttl > ttlWant {
		t.Fatalf("final TTL out of range: got %v want <= %v", ttl, ttlWant)
	}

	// And the content is retrievable
	got, err := loadEntry(context.Background(), rdb, key)
	if err != nil {
		t.Fatalf("load after final err: %v", err)
	}
	if got.Code != 201 || string(got.Body) != `{"ok":true}` || got.InProgress {
		t.Fatalf("final entry mismatch: %+v", got)
	}
}
