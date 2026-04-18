package skill

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/recinq/wave/internal/timeouts"
)

// Timeout constants — re-exported from manifest.Timeouts for backward compatibility.
// Configure via runtime.timeouts in wave.yaml.
var (
	CLITimeout        = timeouts.SkillCLI
	HTTPTimeout       = timeouts.SkillHTTP
	HTTPHeaderTimeout = timeouts.SkillHTTPHeader
)

// SourceAdapter handles installation of skills from a specific source type.
type SourceAdapter interface {
	Install(ctx context.Context, ref string, store Store) (*InstallResult, error)
	Prefix() string
}

// InstallResult represents the outcome of a source adapter installation.
type InstallResult struct {
	Skills   []Skill
	Warnings []string
}

// SourceReference represents a parsed source string.
type SourceReference struct {
	Prefix    string
	Reference string
	Raw       string
}

// DependencyError indicates a required CLI tool is not installed.
type DependencyError struct {
	Binary       string
	Instructions string
}

func (e *DependencyError) Error() string {
	return fmt.Sprintf("required tool %q not found: install with: %s", e.Binary, e.Instructions)
}

// CLIDependency describes an external CLI tool required by an adapter.
type CLIDependency struct {
	Binary       string
	Instructions string
}

// SourceRouter dispatches source strings to the appropriate adapter.
type SourceRouter struct {
	adapters map[string]SourceAdapter
}

// NewSourceRouter creates a router with the given adapters registered.
func NewSourceRouter(adapters ...SourceAdapter) *SourceRouter {
	r := &SourceRouter{
		adapters: make(map[string]SourceAdapter),
	}
	for _, a := range adapters {
		r.Register(a)
	}
	return r
}

// Register adds an adapter to the router.
func (r *SourceRouter) Register(adapter SourceAdapter) {
	r.adapters[adapter.Prefix()] = adapter
}

// Parse splits a source string into the matched adapter and reference.
// URL-scheme prefixes (https://, http://) are checked first, then the generic prefix:reference split.
func (r *SourceRouter) Parse(source string) (SourceAdapter, string, error) {
	if source == "" {
		return nil, "", fmt.Errorf("empty source string")
	}

	// Reject plaintext HTTP — HTTPS only
	if strings.HasPrefix(source, "http://") {
		return nil, "", fmt.Errorf("only HTTPS URLs are allowed; got %q", source)
	}

	// Check HTTPS URL scheme prefix
	if strings.HasPrefix(source, "https://") {
		adapter, ok := r.adapters["https://"]
		if !ok {
			return nil, "", fmt.Errorf("no adapter registered for URL sources")
		}
		return adapter, source, nil
	}

	// Split on first colon for standard prefixes
	idx := strings.Index(source, ":")
	if idx < 0 {
		return nil, "", fmt.Errorf("no source prefix in %q: use a prefix like %s", source, strings.Join(r.Prefixes(), ", "))
	}

	prefix := source[:idx]
	ref := source[idx+1:]

	adapter, ok := r.adapters[prefix]
	if !ok {
		return nil, "", fmt.Errorf("unknown source prefix %q: recognized prefixes are %s", prefix, strings.Join(r.Prefixes(), ", "))
	}

	return adapter, ref, nil
}

// Install parses the source string and delegates to the matched adapter.
func (r *SourceRouter) Install(ctx context.Context, source string, store Store) (*InstallResult, error) {
	adapter, ref, err := r.Parse(source)
	if err != nil {
		return nil, err
	}
	return adapter.Install(ctx, ref, store)
}

// Prefixes returns all registered prefix strings, sorted.
func (r *SourceRouter) Prefixes() []string {
	prefixes := make([]string, 0, len(r.adapters))
	for p := range r.adapters {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)
	return prefixes
}

// NewDefaultRouter creates a SourceRouter with the minimal install path: file: only.
// Bare local paths (without prefix) are accepted by the file adapter via convention.
func NewDefaultRouter(projectRoot string) *SourceRouter {
	return NewSourceRouter(
		NewFileAdapter(projectRoot),
	)
}
