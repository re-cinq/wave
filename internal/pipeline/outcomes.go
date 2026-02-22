package pipeline

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ExtractJSONPath extracts a value from JSON data using simple dot-notation path.
// Supported syntax: ".field", ".field.nested", ".field.nested.deep", ".items[0].url"
// Returns the extracted value as a string, or an error if the path is invalid or not found.
func ExtractJSONPath(data []byte, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty JSON path")
	}

	// Strip leading dot
	if path[0] == '.' {
		path = path[1:]
	}
	if path == "" {
		return "", fmt.Errorf("JSON path contains only a dot")
	}

	parts := strings.Split(path, ".")

	var current any
	if err := json.Unmarshal(data, &current); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	for _, part := range parts {
		// Check for array index syntax: "field[N]"
		if idx := strings.Index(part, "["); idx != -1 {
			field := part[:idx]
			indexStr := strings.TrimSuffix(part[idx+1:], "]")
			arrayIdx, err := strconv.Atoi(indexStr)
			if err != nil {
				return "", fmt.Errorf("invalid array index %q in %q", indexStr, part)
			}

			// Navigate to the field first
			obj, ok := current.(map[string]any)
			if !ok {
				return "", fmt.Errorf("cannot navigate into non-object at %q", field)
			}
			val, exists := obj[field]
			if !exists {
				return "", fmt.Errorf("key %q not found", field)
			}

			// Index into the array
			arr, ok := val.([]any)
			if !ok {
				return "", fmt.Errorf("value at %q is not an array", field)
			}
			if arrayIdx < 0 || arrayIdx >= len(arr) {
				return "", fmt.Errorf("array index %d out of bounds (length %d) at %q", arrayIdx, len(arr), field)
			}
			current = arr[arrayIdx]
			continue
		}

		obj, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("cannot navigate into non-object at %q", part)
		}
		val, exists := obj[part]
		if !exists {
			return "", fmt.Errorf("key %q not found", part)
		}
		current = val
	}

	// Convert the final value to string
	switch v := current.(type) {
	case string:
		return v, nil
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v)), nil
		}
		return fmt.Sprintf("%g", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case nil:
		return "", fmt.Errorf("value at path is null")
	default:
		// For nested objects/arrays, return JSON representation
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("cannot serialize value: %w", err)
		}
		return string(b), nil
	}
}
