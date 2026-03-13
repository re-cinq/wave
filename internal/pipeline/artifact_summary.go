package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SummarizeArtifact reads a file and produces a size-bounded summary.
// For JSON files, it extracts top-level keys and truncated values.
// For markdown, it extracts headings and first paragraph.
// For binary content, it returns a placeholder.
// maxBytes limits the output size.
func SummarizeArtifact(path string, maxBytes int) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read artifact: %w", err)
	}

	if len(data) == 0 {
		return "[empty file]", nil
	}

	// Detect binary content (null bytes in first 512 bytes)
	checkLen := len(data)
	if checkLen > 512 {
		checkLen = 512
	}
	if bytes.ContainsRune(data[:checkLen], 0) {
		return fmt.Sprintf("[binary file, %d bytes]", len(data)), nil
	}

	content := string(data)

	// Try JSON summarization
	if isJSON(data) {
		return summarizeJSON(data, path, maxBytes), nil
	}

	// Try markdown summarization (check for headings)
	if strings.HasPrefix(strings.TrimSpace(content), "#") || strings.Contains(content, "\n#") {
		return summarizeMarkdown(content, path, maxBytes), nil
	}

	// Plain text: first N lines
	return summarizePlainText(content, path, maxBytes), nil
}

func isJSON(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	return (trimmed[0] == '{' || trimmed[0] == '[') && json.Valid(trimmed)
}

func summarizeJSON(data []byte, path string, maxBytes int) string {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return truncateWithNote(string(data), path, maxBytes)
	}

	// For objects, show top-level keys with truncated values
	if obj, ok := raw.(map[string]interface{}); ok {
		var b strings.Builder
		b.WriteString("{\n")
		for key, val := range obj {
			valStr := formatJSONValue(val)
			if len(valStr) > 200 {
				valStr = valStr[:197] + "..."
			}
			line := fmt.Sprintf("  %q: %s,\n", key, valStr)
			if b.Len()+len(line) > maxBytes-50 {
				b.WriteString("  ...\n")
				break
			}
			b.WriteString(line)
		}
		b.WriteString("}")
		return truncateWithNote(b.String(), path, maxBytes)
	}

	// For arrays or other JSON, just truncate
	return truncateWithNote(string(data), path, maxBytes)
}

func formatJSONValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		if len(v) > 100 {
			return fmt.Sprintf("%q", v[:97]+"...")
		}
		return fmt.Sprintf("%q", v)
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v))
		}
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case nil:
		return "null"
	default:
		data, _ := json.Marshal(v)
		s := string(data)
		if len(s) > 200 {
			return s[:197] + "..."
		}
		return s
	}
}

func summarizeMarkdown(content, path string, maxBytes int) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder

	for _, line := range lines {
		if b.Len()+len(line)+1 > maxBytes-60 {
			b.WriteString("\n...")
			break
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(line)
	}

	return truncateWithNote(b.String(), path, maxBytes)
}

func summarizePlainText(content, path string, maxBytes int) string {
	if len(content) <= maxBytes {
		return content
	}
	return truncateWithNote(content, path, maxBytes)
}

func truncateWithNote(content, path string, maxBytes int) string {
	if len(content) <= maxBytes {
		return content
	}
	note := fmt.Sprintf("\n... (truncated, full content at %s)", path)
	cutoff := maxBytes - len(note)
	if cutoff < 0 {
		cutoff = 0
	}
	return content[:cutoff] + note
}
