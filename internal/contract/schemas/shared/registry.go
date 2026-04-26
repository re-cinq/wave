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
)

//go:embed *.json
var schemasFS embed.FS

// TypeString is the sentinel type name for free-text (unstructured) I/O.
// Pipelines with no typed schema default to "string".
const TypeString = "string"

// registry is populated at init time from the embedded *.json files.
// Keys are the filename sans extension (e.g. "issue_ref").
var registry = func() map[string][]byte {
	out := make(map[string][]byte)
	entries, err := fs.ReadDir(schemasFS, ".")
	if err != nil {
		//nolint:forbidigo // package-init guard, embedded FS read cannot fail at runtime
		panic(fmt.Sprintf("shared schemas: failed to read embedded FS: %v", err))
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := schemasFS.ReadFile(e.Name())
		if err != nil {
			//nolint:forbidigo // package-init guard, embedded FS read cannot fail at runtime
			panic(fmt.Sprintf("shared schemas: failed to read %s: %v", e.Name(), err))
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		out[name] = data
	}
	return out
}()

// Lookup returns the raw JSON schema bytes for a named type.
// The special name "string" resolves to (nil, true) — callers treat it as
// free-text with no schema validation.
// Returns (nil, false) if the type is unknown.
func Lookup(name string) ([]byte, bool) {
	if name == "" || name == TypeString {
		return nil, true
	}
	data, ok := registry[name]
	return data, ok
}

// Exists reports whether the given type name is registered (or is the
// "string" sentinel).
func Exists(name string) bool {
	if name == "" || name == TypeString {
		return true
	}
	_, ok := registry[name]
	return ok
}

// Names returns the sorted list of registered typed schema names.
// The "string" sentinel is not included.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
