package farewell

import (
	"fmt"
	"io"
	"strings"
)

const (
	genericTemplate = "Farewell — see you next wave."
	namedTemplate   = "Farewell, %s — see you next wave."
)

// Farewell returns the farewell line for the given recipient name.
// Empty or whitespace-only name yields the generic default.
func Farewell(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return genericTemplate
	}
	return fmt.Sprintf(namedTemplate, trimmed)
}

// WriteFarewell writes Farewell(name) followed by "\n" to w, unless suppress
// is true, in which case it is a no-op. Returns any write error.
func WriteFarewell(w io.Writer, name string, suppress bool) error {
	if suppress {
		return nil
	}
	_, err := io.WriteString(w, Farewell(name)+"\n")
	return err
}
