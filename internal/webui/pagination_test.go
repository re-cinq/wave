//go:build webui

package webui

import (
	"net/http/httptest"
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
