// Package shared provides canonical artifact schemas used by the Wave
// pipeline I/O protocol. Each schema is identified by a short type name
// (e.g. "issue_ref", "pr_ref") and is embedded into the binary so the
// manifest loader can resolve and validate typed inputs and outputs
// without touching the filesystem.
//
// See docs/adr/010-pipeline-io-protocol.md for the protocol design.
package shared

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

//go:embed *.json
var schemasFS embed.FS

// TypeString is the sentinel type name for free-text (unstructured) I/O.
// Pipelines with no typed schema default to "string".
const TypeString = "string"

// registry holds the parsed schema bytes keyed by short type name (e.g.
// "issue_ref"). It is populated lazily on the first call to one of the
// public lookup helpers via loadOnce.
//
// Loading is intentionally lazy — and not done in init() — so that an
// embed-FS read failure surfaces as a structured error rather than a
// process-wide panic. Callers that want to fail fast at startup can
// invoke LoadSchemas() once during binary wiring.
var (
	registry  map[string][]byte
	loadOnce  sync.Once
	loadError error
)

// LoadSchemas eagerly populates the in-memory registry from the embedded
// FS. It is safe to call concurrently and idempotent: subsequent calls
// reuse the first result. The returned error (if any) is also surfaced
// from Lookup, Exists, and Names via LoadError after the first call.
//
// Callers that prefer fail-fast semantics should invoke LoadSchemas
// during binary wiring (e.g. from a manifest constructor) and abort on
// non-nil. Callers that prefer best-effort semantics can rely on the
// implicit lazy load performed by Lookup/Exists/Names.
func LoadSchemas() error {
	loadOnce.Do(func() {
		registry, loadError = loadSchemasFromFS()
	})
	return loadError
}

// LoadError returns the error encountered the first time the registry was
// populated, or nil if loading has not been attempted or succeeded.
func LoadError() error {
	return loadError
}

func loadSchemasFromFS() (map[string][]byte, error) {
	out := make(map[string][]byte)
	entries, err := fs.ReadDir(schemasFS, ".")
	if err != nil {
		return out, fmt.Errorf("shared schemas: failed to read embedded FS: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := schemasFS.ReadFile(e.Name())
		if err != nil {
			return out, fmt.Errorf("shared schemas: failed to read %s: %w", e.Name(), err)
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		out[name] = data
	}
	return out, nil
}

// ensureLoaded triggers the lazy load on first call. Subsequent calls
// are no-ops thanks to sync.Once.
func ensureLoaded() {
	_ = LoadSchemas()
}

// Lookup returns the raw JSON schema bytes for a named type.
// The special name "string" resolves to (nil, true) — callers treat it as
// free-text with no schema validation.
// Returns (nil, false) if the type is unknown OR if the embedded FS
// failed to load (see LoadError).
func Lookup(name string) ([]byte, bool) {
	if name == "" || name == TypeString {
		return nil, true
	}
	ensureLoaded()
	if registry == nil {
		return nil, false
	}
	data, ok := registry[name]
	return data, ok
}

// Exists reports whether the given type name is registered (or is the
// "string" sentinel). Returns false if the registry failed to load.
func Exists(name string) bool {
	if name == "" || name == TypeString {
		return true
	}
	ensureLoaded()
	if registry == nil {
		return false
	}
	_, ok := registry[name]
	return ok
}

// Names returns the sorted list of registered typed schema names.
// The "string" sentinel is not included. Returns an empty slice if the
// registry failed to load.
func Names() []string {
	ensureLoaded()
	if registry == nil {
		return nil
	}
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
