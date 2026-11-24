package state

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Creating a fresh state database on the first `wave run` must not print one
// "Applying migration N" line per migration before any progress output.
func TestNewStateStore_FreshDatabaseIsSilent(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	store, openErr := NewStateStore(filepath.Join(t.TempDir(), "state.db"))
	os.Stderr = orig
	_ = w.Close()
	out, _ := io.ReadAll(r)
	if openErr != nil {
		t.Fatalf("NewStateStore: %v", openErr)
	}
	defer store.Close()
	if len(out) != 0 {
		t.Fatalf("fresh database printed to stderr:\n%s", out)
	}
}
