package webui

import (
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEncodeDecodeCursor(t *testing.T) {
	now := time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC)
	runID := "test-run-123"

	encoded := encodeCursor(now, runID)
	if encoded == "" {
		t.Fatal("encoded cursor should not be empty")
	}

	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("failed to decode cursor: %v", err)
	}

	if decoded.Timestamp != now.Unix() {
		t.Errorf("timestamp mismatch: got %d, want %d", decoded.Timestamp, now.Unix())
	}
	if decoded.RunID != runID {
		t.Errorf("run ID mismatch: got %q, want %q", decoded.RunID, runID)
	}
}

func TestDecodeCursor_Empty(t *testing.T) {
	c, err := decodeCursor("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != nil {
		t.Error("expected nil cursor for empty string")
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	_, err := decodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid cursor")
	}
}

func TestParsePageSize(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"default", "", defaultPageSize},
		{"valid", "limit=50", 50},
		{"max exceeded", "limit=200", maxPageSize},
		{"zero", "limit=0", defaultPageSize},
		{"negative", "limit=-1", defaultPageSize},
		{"non-numeric", "limit=abc", defaultPageSize},
		{"minimum", "limit=1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/runs?"+tt.query, nil)
			got := parsePageSize(req)
			if got != tt.expected {
				t.Errorf("parsePageSize() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// TestDecodeCursor_InvalidJSON verifies that valid base64 containing invalid
// JSON returns an error with a descriptive message.
func TestDecodeCursor_InvalidJSON(t *testing.T) {
	// Encode something that is valid base64 but not valid JSON.
	encoded := base64.URLEncoding.EncodeToString([]byte("this is not json"))
	_, err := decodeCursor(encoded)
	if err == nil {
		t.Fatal("expected error for invalid JSON in cursor")
	}
	if !strings.Contains(err.Error(), "invalid cursor format") {
		t.Errorf("expected error to mention 'invalid cursor format', got: %v", err)
	}
}

// TestDecodeCursor_InvalidBase64 verifies that a string with invalid base64
// characters returns an error with a descriptive message.
func TestDecodeCursor_InvalidBase64(t *testing.T) {
	_, err := decodeCursor("%%%invalid-base64%%%")
	if err == nil {
		t.Fatal("expected error for invalid base64 cursor")
	}
	if !strings.Contains(err.Error(), "invalid cursor encoding") {
		t.Errorf("expected error to mention 'invalid cursor encoding', got: %v", err)
	}
}

// TestDecodeCursor_EmptyString verifies that an empty cursor string returns
// nil cursor and nil error (not an error condition).
func TestDecodeCursor_EmptyString(t *testing.T) {
	c, err := decodeCursor("")
	if err != nil {
		t.Fatalf("expected no error for empty cursor, got: %v", err)
	}
	if c != nil {
		t.Errorf("expected nil cursor for empty string, got: %+v", c)
	}
}

// TestDecodeCursor_VeryOldTimestamp verifies that a cursor with a very old
// timestamp (Unix epoch) decodes correctly without error.
func TestDecodeCursor_VeryOldTimestamp(t *testing.T) {
	epoch := time.Unix(0, 0)
	encoded := encodeCursor(epoch, "old-run")

	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("failed to decode cursor with epoch timestamp: %v", err)
	}
	if decoded.Timestamp != 0 {
		t.Errorf("expected timestamp 0 (epoch), got %d", decoded.Timestamp)
	}
	if decoded.RunID != "old-run" {
		t.Errorf("expected run ID 'old-run', got %q", decoded.RunID)
	}
}

// TestDecodeCursor_VeryNewTimestamp verifies that a cursor with a far-future
// timestamp (year 2100) decodes correctly without error.
func TestDecodeCursor_VeryNewTimestamp(t *testing.T) {
	future := time.Date(2100, 12, 31, 23, 59, 59, 0, time.UTC)
	encoded := encodeCursor(future, "future-run")

	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("failed to decode cursor with future timestamp: %v", err)
	}
	if decoded.Timestamp != future.Unix() {
		t.Errorf("expected timestamp %d, got %d", future.Unix(), decoded.Timestamp)
	}
	if decoded.RunID != "future-run" {
		t.Errorf("expected run ID 'future-run', got %q", decoded.RunID)
	}
}

// TestDecodeCursor_NegativeTimestamp verifies that a cursor with a negative
// timestamp (before Unix epoch) decodes correctly.
func TestDecodeCursor_NegativeTimestamp(t *testing.T) {
	beforeEpoch := time.Date(1960, 1, 1, 0, 0, 0, 0, time.UTC)
	encoded := encodeCursor(beforeEpoch, "pre-epoch-run")

	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("failed to decode cursor with negative timestamp: %v", err)
	}
	if decoded.Timestamp != beforeEpoch.Unix() {
		t.Errorf("expected timestamp %d, got %d", beforeEpoch.Unix(), decoded.Timestamp)
	}
}

// TestParsePageSize_VeryLarge verifies that a very large page size value
// is clamped to maxPageSize.
func TestParsePageSize_VeryLarge(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/runs?limit=999999", nil)
	got := parsePageSize(req)
	if got != maxPageSize {
		t.Errorf("expected very large limit to be clamped to %d, got %d", maxPageSize, got)
	}
}

// TestParsePageSize_MaxBoundary verifies that a limit exactly at maxPageSize
// is accepted as-is.
func TestParsePageSize_MaxBoundary(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/runs?limit=100", nil)
	got := parsePageSize(req)
	if got != maxPageSize {
		t.Errorf("expected limit=100 to equal maxPageSize (%d), got %d", maxPageSize, got)
	}
}

// TestParsePageSize_JustAboveMax verifies that a limit one above maxPageSize
// is clamped.
func TestParsePageSize_JustAboveMax(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/runs?limit=101", nil)
	got := parsePageSize(req)
	if got != maxPageSize {
		t.Errorf("expected limit=101 to be clamped to %d, got %d", maxPageSize, got)
	}
}

// TestEncodeCursor_Roundtrip verifies that encoding and decoding a cursor
// preserves all fields exactly.
func TestEncodeCursor_Roundtrip(t *testing.T) {
	cases := []struct {
		name  string
		time  time.Time
		runID string
	}{
		{"normal", time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC), "run-abc-123"},
		{"empty run ID", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ""},
		{"special chars", time.Now().UTC(), "run/with+special=chars"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded := encodeCursor(tc.time, tc.runID)
			decoded, err := decodeCursor(encoded)
			if err != nil {
				t.Fatalf("roundtrip decode failed: %v", err)
			}
			if decoded.Timestamp != tc.time.Unix() {
				t.Errorf("timestamp mismatch: got %d, want %d", decoded.Timestamp, tc.time.Unix())
			}
			if decoded.RunID != tc.runID {
				t.Errorf("runID mismatch: got %q, want %q", decoded.RunID, tc.runID)
			}
		})
	}
}

// TestDecodeCursor_EmptyJSON verifies that valid base64 containing an empty
// JSON object decodes to a zero-value cursor (no error).
func TestDecodeCursor_EmptyJSON(t *testing.T) {
	encoded := base64.URLEncoding.EncodeToString([]byte("{}"))
	decoded, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("expected no error for empty JSON object, got: %v", err)
	}
	if decoded.Timestamp != 0 {
		t.Errorf("expected zero timestamp for empty JSON, got %d", decoded.Timestamp)
	}
	if decoded.RunID != "" {
		t.Errorf("expected empty run ID for empty JSON, got %q", decoded.RunID)
	}
}
