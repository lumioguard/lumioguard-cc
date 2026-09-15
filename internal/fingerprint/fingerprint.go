// Package fingerprint provides the hashing and canonical-JSON primitives used
// for snapshot identities, finding identifiers and configuration hashes.
package fingerprint

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// SHA256 returns the lowercase hexadecimal SHA-256 digest of data.
func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SHA256String hashes the UTF-8 bytes of s.
func SHA256String(s string) string {
	return SHA256([]byte(s))
}

// StableJSON encodes v as canonical JSON: object keys sorted, arrays in
// order, no insignificant whitespace. Equal values always produce equal bytes.
func StableJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode value: %w", err)
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, fmt.Errorf("normalise value: %w", err)
	}
	stable, err := json.Marshal(generic)
	if err != nil {
		return nil, fmt.Errorf("encode normalised value: %w", err)
	}
	return stable, nil
}

// HashJSON returns the SHA-256 digest of the canonical JSON encoding of v.
func HashJSON(v any) (string, error) {
	stable, err := StableJSON(v)
	if err != nil {
		return "", err
	}
	return SHA256(stable), nil
}

// ShortID derives a 20-character identifier from parts joined with NUL bytes.
// It is used for finding identifiers that must be stable across runs.
func ShortID(parts ...string) string {
	return SHA256String(strings.Join(parts, "\x00"))[:20]
}

// NewRunID returns a random RFC 4122 version 4 UUID string.
func NewRunID() string {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		// crypto/rand failing is unrecoverable for a CLI; the run ID is volatile metadata only.
		panic(fmt.Sprintf("generate run id: %v", err))
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
