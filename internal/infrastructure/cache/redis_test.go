package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestOpenRedis_Success(t *testing.T) {
	// Start in-memory Redis
	s := miniredis.RunT(t)
	defer s.Close()

	// Use a non-zero DB to verify it's set
	c, err := OpenRedis(s.Addr(), 2)
	if err != nil {
		t.Fatalf("OpenRedis returned error: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	// Check the client actually works and uses the right DB
	if got := c.Options().DB; got != 2 {
		t.Fatalf("client DB = %d, want 2", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := c.Set(ctx, "k", "v", 0).Err(); err != nil {
		t.Fatalf("SET err: %v", err)
	}
	v, err := c.Get(ctx, "k").Result()
	if err != nil {
		t.Fatalf("GET err: %v", err)
	}
	if v != "v" {
		t.Fatalf("GET value = %q, want %q", v, "v")
	}
}

func TestOpenRedis_Failure(t *testing.T) {
	// Unresolvable host → Ping should fail immediately (no 5s delay)
	if _, err := OpenRedis("not-a-real-host:6379", 0); err == nil {
		t.Fatal("expected error, got nil")
	}
}
