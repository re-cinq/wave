package shared

import (
	"testing"
)

func TestNamesSorted(t *testing.T) {
	names := Names()
	if len(names) < 8 {
		t.Errorf("Names() returned %d entries, want at least 8", len(names))
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Errorf("Names() not sorted: %q >= %q", names[i-1], names[i])
		}
	}
	// string sentinel must not leak into Names()
	for _, n := range names {
		if n == "string" {
			t.Error("Names() must not include \"string\" sentinel")
		}
	}
}
