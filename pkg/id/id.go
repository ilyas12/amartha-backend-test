package id

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID32 returns exactly 32 hex characters (no separators/prefixes).
func NewID32() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
