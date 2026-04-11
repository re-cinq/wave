package pipeline

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// emptyArrayError is returned when a JSON path indexes into an empty array
// (index 0 on length 0). This distinguishes a "no results" condition from
// genuine out-of-bounds errors, allowing callers to produce friendlier messages.
type emptyArrayError struct {
	Field string
}

func (e *emptyArrayError) Error() string {
	return fmt.Sprintf("no items in %s", e.Field)
}

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
				if arrayIdx == 0 && len(arr) == 0 {
					return "", &emptyArrayError{Field: field}
				}
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

// ContainsWildcard returns true if a json_path string contains the [*] array wildcard syntax.
func ContainsWildcard(path string) bool {
	return strings.Contains(path, "[*]")
}

// ExtractJSONPathAll extracts multiple values from JSON data using a path containing [*] wildcard.
// It splits the path at [*] into a prefix and suffix, navigates to the array at the prefix,
// then extracts the suffix sub-path from each element.
// Returns []string of extracted values. Empty arrays return an empty slice without error.
// Non-array values at the wildcard position return an error.
func ExtractJSONPathAll(data []byte, path string) ([]string, error) {
	if path == "" {
		return nil, fmt.Errorf("empty JSON path")
	}

	// Split at [*]
	idx := strings.Index(path, "[*]")
	if idx == -1 {
		return nil, fmt.Errorf("path %q does not contain [*] wildcard", path)
	}

	prefix := path[:idx]
	suffix := path[idx+3:] // skip "[*]"

	// Strip leading dot from prefix
	if len(prefix) > 0 && prefix[0] == '.' {
		prefix = prefix[1:]
	}

	// Parse the JSON data
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Navigate to the array using prefix
	current := root
	if prefix != "" {
		parts := strings.Split(prefix, ".")
		for _, part := range parts {
			if part == "" {
				continue
			}
			obj, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("cannot navigate into non-object at %q", part)
			}
			val, exists := obj[part]
			if !exists {
				return nil, fmt.Errorf("key %q not found", part)
			}
			current = val
		}
	}

	// Current must be an array
	arr, ok := current.([]any)
	if !ok {
		return nil, fmt.Errorf("value at wildcard position is not an array")
	}

	// Empty array → empty slice, no error
	if len(arr) == 0 {
		return []string{}, nil
	}

	// Extract suffix sub-path from each element
	results := make([]string, 0, len(arr))
	for i, elem := range arr {
		if suffix == "" {
			// No suffix — element itself is the value
			str, ok := elem.(string)
			if !ok {
				return nil, fmt.Errorf("array element [%d] is not a string", i)
			}
			results = append(results, str)
		} else {
			// Marshal the element back to JSON and extract via suffix
			elemData, err := json.Marshal(elem)
			if err != nil {
				return nil, fmt.Errorf("cannot serialize array element [%d]: %w", i, err)
			}
			val, err := ExtractJSONPath(elemData, suffix)
			if err != nil {
				return nil, fmt.Errorf("extraction failed on element [%d]: %w", i, err)
			}
			results = append(results, val)
		}
	}

	return results, nil
}
