package suggest

import (
	"encoding/json"
	"testing"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		s    Status
		want string
	}{
		{StatusOK, "ok"},
		{StatusWarn, "warn"},
		{StatusErr, "error"},
		{Status(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestStatusMarshalJSON(t *testing.T) {
	tests := []struct {
		s    Status
		want string
	}{
		{StatusOK, `"ok"`},
		{StatusWarn, `"warn"`},
		{StatusErr, `"error"`},
		{Status(99), `"unknown"`},
	}
	for _, tt := range tests {
		got, err := json.Marshal(tt.s)
		if err != nil {
			t.Fatalf("Marshal(%v) error: %v", tt.s, err)
		}
		if string(got) != tt.want {
			t.Errorf("Marshal(%v) = %s, want %s", tt.s, got, tt.want)
		}
	}
}
