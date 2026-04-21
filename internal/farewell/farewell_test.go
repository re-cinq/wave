package farewell

import (
	"bytes"
	"strings"
	"testing"
)

func TestFarewell(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
		substr string
	}{
		{"empty", "", "Farewell — see you next wave.", ""},
		{"named", "Alice", "Farewell, Alice — see you next wave.", "Alice"},
		{"trim", "  Alice  ", "Farewell, Alice — see you next wave.", "Alice"},
		{"whitespace only", "   \t\n", "Farewell — see you next wave.", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Farewell(tc.input)
			if got != tc.want {
				t.Fatalf("Farewell(%q) = %q, want %q", tc.input, got, tc.want)
			}
			if tc.substr != "" && !strings.Contains(got, tc.substr) {
				t.Fatalf("Farewell(%q) = %q, missing substring %q", tc.input, got, tc.substr)
			}
		})
	}
}

func TestFarewellDeterministic(t *testing.T) {
	if Farewell("") != Farewell("") {
		t.Fatal("Farewell(\"\") not deterministic")
	}
	if Farewell("Alice") != Farewell("Alice") {
		t.Fatal("Farewell(\"Alice\") not deterministic")
	}
}

func TestWriteFarewell(t *testing.T) {
	t.Run("writes with newline", func(t *testing.T) {
		var buf bytes.Buffer
		if err := WriteFarewell(&buf, "Alice", false); err != nil {
			t.Fatalf("err: %v", err)
		}
		want := "Farewell, Alice — see you next wave.\n"
		if buf.String() != want {
			t.Fatalf("buf = %q, want %q", buf.String(), want)
		}
	})
	t.Run("suppress is no-op", func(t *testing.T) {
		var buf bytes.Buffer
		if err := WriteFarewell(&buf, "Alice", true); err != nil {
			t.Fatalf("err: %v", err)
		}
		if buf.Len() != 0 {
			t.Fatalf("expected empty buf, got %q", buf.String())
		}
	})
	t.Run("empty name generic", func(t *testing.T) {
		var buf bytes.Buffer
		if err := WriteFarewell(&buf, "", false); err != nil {
			t.Fatalf("err: %v", err)
		}
		want := "Farewell — see you next wave.\n"
		if buf.String() != want {
			t.Fatalf("buf = %q, want %q", buf.String(), want)
		}
	})
}
