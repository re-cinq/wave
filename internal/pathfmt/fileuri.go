package pathfmt

import "strings"

// FileURI prefixes absolute file paths with the file:// URI scheme so they
// become clickable hyperlinks in modern terminal emulators. Relative paths,
// empty strings, and paths that already contain a URI scheme are returned
// unchanged.
func FileURI(path string) string {
	if path == "" {
		return path
	}
	// Skip paths that already contain a URI scheme (e.g., file://, https://)
	if strings.Contains(path, "://") {
		return path
	}
	// Only prefix absolute paths (starting with /)
	if !strings.HasPrefix(path, "/") {
		return path
	}
	return "file://" + path
}
