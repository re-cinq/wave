package pipeline

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

const defaultHashLength = 8

// GenerateRunID generates a unique pipeline runtime ID in the format "{name}-{hex_suffix}".
// hashLength specifies the number of hex characters in the suffix (0 uses default of 8).
// Uses crypto/rand for entropy with a timestamp-based fallback.
func GenerateRunID(name string, hashLength int) string {
	if hashLength <= 0 {
		hashLength = defaultHashLength
	}

	suffix := generateHexSuffix(hashLength)
	return fmt.Sprintf("%s-%s", name, suffix)
}

// generateHexSuffix generates a random hex string of the specified length.
// Falls back to timestamp-based entropy if crypto/rand fails.
func generateHexSuffix(length int) string {
	// Number of bytes needed (each byte = 2 hex chars)
	numBytes := (length + 1) / 2

	b := make([]byte, numBytes)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based entropy
		return timestampFallback(length)
	}

	return hex.EncodeToString(b)[:length]
}

// timestampFallback generates a hex string from the current timestamp.
func timestampFallback(length int) string {
	ts := time.Now().UnixNano()
	hex := fmt.Sprintf("%016x", ts)
	if len(hex) >= length {
		return hex[len(hex)-length:]
	}
	return hex
}
