package pipeline

import (
	"strings"
	"testing"
)

func TestBuildFailureContextSection(t *testing.T) {
	got := buildFailureContextSection("Pipeline: fail-demo\n- step build [contract_failed]: boom")
	for _, want := range []string{"## Previous Attempt Failed", "avoids these failures", "Pipeline: fail-demo", "boom"} {
		if !strings.Contains(got, want) {
			t.Errorf("section missing %q:\n%s", want, got)
		}
	}
	e := NewMetaPipelineExecutor(nil, WithFailureContext("ctx"))
	if e.FailureContext() != "ctx" {
		t.Errorf("WithFailureContext not applied")
	}
}
