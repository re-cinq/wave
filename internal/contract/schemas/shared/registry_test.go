package shared

import (
	"encoding/json"
	"testing"
)

func TestRegistryContainsCanonicalTypes(t *testing.T) {
	expected := []string{
		"issue_ref",
		"pr_ref",
		"branch_ref",
		"spec_ref",
		"findings_report",
		"plan_ref",
		"workspace_ref",
		"work_item_ref",
	}
	for _, name := range expected {
		data, err := schemasFS.ReadFile(name + ".json")
		if err != nil {
			t.Errorf("embedded schema %q missing: %v", name, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("schema %q: empty bytes", name)
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			t.Errorf("schema %q is not valid JSON: %v", name, err)
		}
		if !Exists(name) {
			t.Errorf("Exists(%q) = false, want true", name)
		}
	}
}

func TestStringSentinel(t *testing.T) {
	if !Exists("string") {
		t.Error("Exists(\"string\") = false, want true")
	}
	if !Exists("") {
		t.Error("Exists(\"\") = false, want true (empty treated as string)")
	}
}

func TestUnknownType(t *testing.T) {
	if Exists("not_a_real_type_xyz") {
		t.Error("Exists(unknown) = true, want false")
	}
}

func TestNamesSorted(t *testing.T) {
	names := Names()
	if len(names) < 8 {
		t.Errorf("Names() returned %d entries, want at least 8", len(names))
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Errorf("Names() not sorted: %q >= %q", names[i-1], names[i])
		}
	}
	// string sentinel must not leak into Names()
	for _, n := range names {
		if n == "string" {
			t.Error("Names() must not include \"string\" sentinel")
		}
	}
}
