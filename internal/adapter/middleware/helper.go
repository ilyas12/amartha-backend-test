package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func bodyHash(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

func nowUTC() time.Time { return time.Now().UTC() }

func buildKey(method, path, borrowerID, requestID string) string {
	return "idemp:ax:" + strings.ToLower(method) + ":" + path + ":" + borrowerID + ":" + requestID
}

var (
	reUUID  = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[1-5][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	reHex32 = regexp.MustCompile(`^[a-f0-9]{32}$`)
)

func validReqID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return reUUID.MatchString(id) || reHex32.MatchString(id)
}

// parseAxRequestAt accepts:
//   - epoch seconds (e.g., "1736123456")
//   - epoch milliseconds (e.g., "1736123456789")
//   - RFC3339 / RFC3339Nano **with timezone** (e.g., "2025-09-05T10:00:00+07:00" or "...Z")
//
// Naive local timestamps **without** timezone are rejected.
func parseAxRequestAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("missing Ax-Request-At")
	}
	// Epoch?
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if n > 1e12 { // ms
			return time.UnixMilli(n).UTC(), nil
		}
		return time.Unix(n, 0).UTC(), nil // seconds
	}
	// RFC3339 / RFC3339Nano (requires zone or 'Z')
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errors.New("Ax-Request-At must be epoch (s/ms) or RFC3339 with timezone")
}

// ---- Redis helpers ----
func provisionalSet(ctx context.Context, rdb *redis.Client, key string, entry idempEntry) (bool, error) {
	payload, _ := json.Marshal(entry)
	return rdb.SetNX(ctx, key, payload, provisionalLockTTL).Result()
}

func loadEntry(ctx context.Context, rdb *redis.Client, key string) (idempEntry, error) {
	var e idempEntry
	v, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		return e, err
	}
	_ = json.Unmarshal(v, &e)
	return e, nil
}

func saveFinal(ctx context.Context, rdb *redis.Client, key string, entry idempEntry, ttl time.Duration) error {
	payload, _ := json.Marshal(entry)
	return rdb.Set(ctx, key, payload, ttl).Err()
}
