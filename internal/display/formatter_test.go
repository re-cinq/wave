package display

import (
	"testing"
)

func TestNewFormatter(t *testing.T) {
	f := NewFormatter()
	if f == nil {
		t.Fatal("NewFormatter returned nil")
	}
	if f.codec == nil {
		t.Error("Expected codec to be initialized")
	}
}
