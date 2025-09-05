package id

import (
	"encoding/hex"
	"regexp"
	"testing"
)

var reHex32 = regexp.MustCompile(`^[a-f0-9]{32}$`)

func TestNewID32_FormatAndDecode(t *testing.T) {
	got := NewID32()

	// length
	if len(got) != 32 {
		t.Fatalf("length = %d, want 32 (got=%q)", len(got), got)
	}
	// lowercase hex only (no separators/prefixes)
	if !reHex32.MatchString(got) {
		t.Fatalf("not 32-char lowercase hex: %q", got)
	}
	// decodes to exactly 16 bytes
	b, err := hex.DecodeString(got)
	if err != nil {
		t.Fatalf("hex.DecodeString error: %v", err)
	}
	if len(b) != 16 {
		t.Fatalf("decoded bytes = %d, want 16", len(b))
	}
}

func TestNewID32_Uniqueness(t *testing.T) {
	const n = 200
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := NewID32()
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id after %d iterations: %q", i, id)
		}
		seen[id] = struct{}{}
	}
}

func TestNewID32_NoUppercaseOrHyphen(t *testing.T) {
	id := NewID32()
	for _, r := range id {
		if r >= 'A' && r <= 'Z' {
			t.Fatalf("found uppercase letter in id: %q", id)
		}
		if r == '-' {
			t.Fatalf("found hyphen in id: %q", id)
		}
	}
}
