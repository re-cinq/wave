package adaptertest

import (
	"encoding/json"
	"sort"
	"strings"
)

// schemaPromptMarker precedes the JSON schema that the executor injects into
// the prompt of every step with a json_schema contract (see
// pipeline.buildContractPrompt).
const schemaPromptMarker = "**Schema** (your output must conform to this):\n```json\n"

// extractPromptSchema returns the JSON schema embedded in a step prompt, or
// nil when the prompt carries none.
func extractPromptSchema(prompt string) map[string]any {
	i := strings.Index(prompt, schemaPromptMarker)
	if i < 0 {
		return nil
	}
	rest := prompt[i+len(schemaPromptMarker):]
	j := strings.Index(rest, "\n```")
	if j < 0 {
		return nil
	}
	var schema map[string]any
	if err := json.Unmarshal([]byte(rest[:j]), &schema); err != nil {
		return nil
	}
	return schema
}

// schemaInstanceJSON builds a minimal JSON document that satisfies the given
// schema, so --mock runs pass json_schema contracts without an LLM.
func schemaInstanceJSON(schema map[string]any) (string, bool) {
	v := schemaInstance(schema, schema, 0)
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", false
	}
	return string(data), true
}

func schemaInstance(s, root map[string]any, depth int) any {
	if s == nil || depth > 12 {
		return nil
	}
	if ref, ok := s["$ref"].(string); ok {
		if r := resolveRef(ref, root); r != nil {
			return schemaInstance(r, root, depth+1)
		}
		return map[string]any{}
	}
	if c, ok := s["const"]; ok {
		return c
	}
	if e, ok := s["enum"].([]any); ok && len(e) > 0 {
		return e[0]
	}
	if d, ok := s["default"]; ok {
		return d
	}
	for _, key := range []string{"oneOf", "anyOf"} {
		if alts, ok := s[key].([]any); ok && len(alts) > 0 {
			if m, ok := alts[0].(map[string]any); ok {
				return schemaInstance(m, root, depth+1)
			}
		}
	}
	if all, ok := s["allOf"].([]any); ok && len(all) > 0 {
		merged := map[string]any{}
		for _, a := range all {
			if m, ok := a.(map[string]any); ok {
				for k, v := range m {
					merged[k] = v
				}
			}
		}
		return schemaInstance(merged, root, depth+1)
	}

	switch schemaType(s) {
	case "object":
		out := map[string]any{}
		props, _ := s["properties"].(map[string]any)
		for _, name := range requiredFields(s, props) {
			ps, _ := props[name].(map[string]any)
			if ps == nil {
				out[name] = "mock"
				continue
			}
			out[name] = schemaInstance(ps, root, depth+1)
		}
		return out
	case "array":
		n := 0
		if mi, ok := s["minItems"].(float64); ok {
			n = int(mi)
		}
		items, _ := s["items"].(map[string]any)
		if n == 0 && items != nil {
			n = 1
		}
		if mx, ok := s["maxItems"].(float64); ok && n > int(mx) {
			n = int(mx)
		}
		arr := make([]any, 0, n)
		for k := 0; k < n; k++ {
			arr = append(arr, schemaInstance(items, root, depth+1))
		}
		return arr
	case "string":
		return mockString(s)
	case "integer":
		return mockNumber(s, true)
	case "number":
		return mockNumber(s, false)
	case "boolean":
		return true
	case "null":
		return nil
	}
	return "mock"
}

func schemaType(s map[string]any) string {
	switch t := s["type"].(type) {
	case string:
		return t
	case []any:
		for _, x := range t {
			if str, ok := x.(string); ok && str != "null" {
				return str
			}
		}
	}
	if _, ok := s["properties"]; ok {
		return "object"
	}
	if _, ok := s["items"]; ok {
		return "array"
	}
	return ""
}

// requiredFields returns the required property names; when none are
// required, all declared properties are filled (sorted for determinism).
func requiredFields(s map[string]any, props map[string]any) []string {
	var names []string
	if req, ok := s["required"].([]any); ok {
		for _, r := range req {
			if name, ok := r.(string); ok {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		for name := range props {
			names = append(names, name)
		}
		sort.Strings(names)
	}
	return names
}

func mockString(s map[string]any) string {
	switch s["format"] {
	case "date-time":
		return "2026-01-01T00:00:00Z"
	case "date":
		return "2026-01-01"
	case "uri", "url":
		return "https://example.com/mock"
	case "email":
		return "mock@example.com"
	}
	v := "mock output"
	if ml, ok := s["minLength"].(float64); ok && len(v) < int(ml) {
		v += strings.Repeat("x", int(ml)-len(v))
	}
	if mx, ok := s["maxLength"].(float64); ok && len(v) > int(mx) {
		v = v[:int(mx)]
	}
	return v
}

func mockNumber(s map[string]any, integer bool) any {
	v := 1.0
	if mi, ok := s["minimum"].(float64); ok && v < mi {
		v = mi
	}
	if emi, ok := s["exclusiveMinimum"].(float64); ok && v <= emi {
		v = emi + 1
	}
	if mx, ok := s["maximum"].(float64); ok && v > mx {
		v = mx
	}
	if integer {
		return int(v)
	}
	return v
}

func resolveRef(ref string, root map[string]any) map[string]any {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	var cur any = root
	for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[part]
	}
	m, _ := cur.(map[string]any)
	return m
}
