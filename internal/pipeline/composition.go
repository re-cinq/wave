package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SubPipelineLoader loads a pipeline by name from a directory.
// This avoids a circular dependency on the tui package.
type SubPipelineLoader func(dir, name string) (*Pipeline, error)

// mergeJSONArrays takes a JSON string containing multiple arrays and merges them.
// If the input is an array of arrays, the inner arrays are flattened into one.
// If the input is already a flat array (no inner arrays), it is returned as-is.
//
// When key is non-empty, each element is expected to be a JSON object and the
// value at that key is extracted before merging. This supports the common pattern
// where sub-pipelines produce {"findings": [...], "summary": "..."} envelopes
// and only the array field should be merged.
func mergeJSONArrays(data string, key string) (string, error) {
	var elements []json.RawMessage
	if err := json.Unmarshal([]byte(data), &elements); err != nil {
		return "", fmt.Errorf("cannot parse as JSON array: %w", err)
	}

	// When a key is specified, extract that field from each element first.
	if key != "" {
		extracted := make([]json.RawMessage, 0, len(elements))
		for i, elem := range elements {
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(elem, &obj); err != nil {
				return "", fmt.Errorf("element %d: expected JSON object for key extraction, got: %s", i, truncateJSON(elem))
			}
			val, ok := obj[key]
			if !ok {
				return "", fmt.Errorf("element %d: key %q not found in object", i, key)
			}
			// The extracted value should be an array; unwrap its items directly.
			var inner []json.RawMessage
			if err := json.Unmarshal(val, &inner); err != nil {
				return "", fmt.Errorf("element %d: value at key %q is not a JSON array", i, key)
			}
			extracted = append(extracted, inner...)
		}
		result, err := json.Marshal(extracted)
		if err != nil {
			return "", err
		}
		return string(result), nil
	}

	// Check if any element is itself an array -- if so, flatten all.
	hasSubArray := false
	for _, elem := range elements {
		trimmed := strings.TrimSpace(string(elem))
		if len(trimmed) > 0 && trimmed[0] == '[' {
			hasSubArray = true
			break
		}
	}

	if !hasSubArray {
		// Already a flat array, return as-is
		return data, nil
	}

	// Flatten: each element must be an array
	var merged []json.RawMessage
	for _, elem := range elements {
		var inner []json.RawMessage
		if err := json.Unmarshal(elem, &inner); err != nil {
			// Not an array element -- include it directly
			merged = append(merged, elem)
			continue
		}
		merged = append(merged, inner...)
	}

	result, err := json.Marshal(merged)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// truncateJSON returns a short representation of a JSON value for error messages.
func truncateJSON(data json.RawMessage) string {
	s := string(data)
	if len(s) > 60 {
		return s[:60] + "..."
	}
	return s
}

// ValidateCompositionTemplates checks that all template references in a composition
// pipeline resolve to known step IDs or valid expressions.
func ValidateCompositionTemplates(p *Pipeline) []string {
	var errors []string
	stepIDs := make(map[string]bool)
	for _, step := range p.Steps {
		stepIDs[step.ID] = true
	}

	for _, step := range p.Steps {
		if step.Iterate != nil {
			errors = append(errors, validateTemplateRefs(step.Iterate.Over, stepIDs, step.ID, "iterate.over")...)
		}
		if step.Branch != nil {
			errors = append(errors, validateTemplateRefs(step.Branch.On, stepIDs, step.ID, "branch.on")...)
		}
		if step.SubInput != "" {
			errors = append(errors, validateTemplateRefs(step.SubInput, stepIDs, step.ID, "input")...)
		}
		if step.Loop != nil && step.Loop.Until != "" {
			errors = append(errors, validateTemplateRefs(step.Loop.Until, stepIDs, step.ID, "loop.until")...)
		}
		if step.Aggregate != nil {
			errors = append(errors, validateTemplateRefs(step.Aggregate.From, stepIDs, step.ID, "aggregate.from")...)
		}
	}

	return errors
}

// validateTemplateRefs checks that step references in a template exist.
func validateTemplateRefs(tmpl string, stepIDs map[string]bool, stepID, field string) []string {
	var errors []string
	matches := templatePattern.FindAllStringSubmatch(tmpl, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		expr := strings.TrimSpace(match[1])
		// Skip built-in expressions
		if expr == "input" || expr == "item" || expr == "iteration" || strings.HasPrefix(expr, "item.") || strings.HasPrefix(expr, "env.") {
			continue
		}
		// Must be step_id.output or step_id.output.field
		parts := strings.SplitN(expr, ".", 2)
		if len(parts) >= 2 && (parts[1] == "output" || strings.HasPrefix(parts[1], "output.")) {
			if !stepIDs[parts[0]] {
				errors = append(errors, fmt.Sprintf("step %q, %s: references unknown step %q", stepID, field, parts[0]))
			}
		}
	}
	return errors
}
