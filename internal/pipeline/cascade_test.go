package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetStaleDownstream_LinearPipeline(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{
				ID:              "step-a",
				OutputArtifacts: []ArtifactDef{{Name: "output-a", Path: ".wave/output/a.json"}},
			},
			{
				ID: "step-b",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{{Step: "step-a", Artifact: "output-a", As: "input_a"}},
				},
				OutputArtifacts: []ArtifactDef{{Name: "output-b", Path: ".wave/output/b.json"}},
			},
			{
				ID: "step-c",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{{Step: "step-b", Artifact: "output-b", As: "input_b"}},
				},
			},
		},
	}

	detector := NewCascadeDetector()
	stale, err := detector.GetStaleDownstream(p, "step-a", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stale) != 2 {
		t.Fatalf("expected 2 stale steps, got %d", len(stale))
	}

	if stale[0].StepID != "step-b" {
		t.Errorf("expected first stale step to be step-b, got %q", stale[0].StepID)
	}
	if stale[1].StepID != "step-c" {
		t.Errorf("expected second stale step to be step-c, got %q", stale[1].StepID)
	}

	// step-b should be directly stale (consumes from modified step)
	foundDirect := false
	for _, r := range stale[0].Reasons {
		if strings.Contains(r, "modified step") {
			foundDirect = true
			break
		}
	}
	if !foundDirect {
		t.Errorf("expected step-b to have a direct staleness reason, got reasons: %v", stale[0].Reasons)
	}

	// step-c should be transitively stale
	foundTransitive := false
	for _, r := range stale[1].Reasons {
		if strings.Contains(r, "transitively stale") {
			foundTransitive = true
			break
		}
	}
	if !foundTransitive {
		t.Errorf("expected step-c to have a transitive staleness reason, got reasons: %v", stale[1].Reasons)
	}
}

func TestGetStaleDownstream_DiamondDAG(t *testing.T) {
	// Diamond: A -> B, A -> C, B -> D, C -> D
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "diamond-pipeline"},
		Steps: []Step{
			{
				ID:              "step-a",
				OutputArtifacts: []ArtifactDef{{Name: "output-a", Path: ".wave/output/a.json"}},
			},
			{
				ID: "step-b",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{{Step: "step-a", Artifact: "output-a", As: "input_a"}},
				},
				OutputArtifacts: []ArtifactDef{{Name: "output-b", Path: ".wave/output/b.json"}},
			},
			{
				ID: "step-c",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{{Step: "step-a", Artifact: "output-a", As: "input_a"}},
				},
				OutputArtifacts: []ArtifactDef{{Name: "output-c", Path: ".wave/output/c.json"}},
			},
			{
				ID: "step-d",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step-b", Artifact: "output-b", As: "input_b"},
						{Step: "step-c", Artifact: "output-c", As: "input_c"},
					},
				},
			},
		},
	}

	detector := NewCascadeDetector()
	stale, err := detector.GetStaleDownstream(p, "step-a", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stale) != 3 {
		t.Fatalf("expected 3 stale steps (B, C, D), got %d", len(stale))
	}

	staleIDs := make(map[string]bool)
	for _, s := range stale {
		staleIDs[s.StepID] = true
	}

	for _, expected := range []string{"step-b", "step-c", "step-d"} {
		if !staleIDs[expected] {
			t.Errorf("expected %q to be stale, but it was not found", expected)
		}
	}

	// Verify pipeline-definition order is preserved.
	if stale[0].StepID != "step-b" {
		t.Errorf("expected first stale step to be step-b, got %q", stale[0].StepID)
	}
	if stale[1].StepID != "step-c" {
		t.Errorf("expected second stale step to be step-c, got %q", stale[1].StepID)
	}
	if stale[2].StepID != "step-d" {
		t.Errorf("expected third stale step to be step-d, got %q", stale[2].StepID)
	}
}

func TestGetStaleDownstream_NoDownstream(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{
				ID:              "step-a",
				OutputArtifacts: []ArtifactDef{{Name: "output-a", Path: ".wave/output/a.json"}},
			},
			{
				ID: "step-b",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{{Step: "step-a", Artifact: "output-a", As: "input_a"}},
				},
				OutputArtifacts: []ArtifactDef{{Name: "output-b", Path: ".wave/output/b.json"}},
			},
			{
				ID: "step-c",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{{Step: "step-b", Artifact: "output-b", As: "input_b"}},
				},
			},
		},
	}

	detector := NewCascadeDetector()
	stale, err := detector.GetStaleDownstream(p, "step-c", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stale) != 0 {
		t.Errorf("expected 0 stale steps when modifying last step, got %d", len(stale))
	}
}

func TestGetStaleDownstream_InvalidStep(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "test-pipeline"},
		Steps: []Step{
			{ID: "step-a"},
		},
	}

	detector := NewCascadeDetector()
	_, err := detector.GetStaleDownstream(p, "nonexistent", "")
	if err == nil {
		t.Fatal("expected error for nonexistent step, got nil")
	}

	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention step ID, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error to mention 'not found', got: %v", err)
	}
}

func TestGetStaleDownstream_IndependentBranches(t *testing.T) {
	// Two independent chains: A -> B, C -> D
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "branched-pipeline"},
		Steps: []Step{
			{
				ID:              "step-a",
				OutputArtifacts: []ArtifactDef{{Name: "output-a", Path: ".wave/output/a.json"}},
			},
			{
				ID: "step-b",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{{Step: "step-a", Artifact: "output-a", As: "input_a"}},
				},
			},
			{
				ID:              "step-c",
				OutputArtifacts: []ArtifactDef{{Name: "output-c", Path: ".wave/output/c.json"}},
			},
			{
				ID: "step-d",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{{Step: "step-c", Artifact: "output-c", As: "input_c"}},
				},
			},
		},
	}

	detector := NewCascadeDetector()
	stale, err := detector.GetStaleDownstream(p, "step-a", "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(stale) != 1 {
		t.Fatalf("expected 1 stale step, got %d", len(stale))
	}

	if stale[0].StepID != "step-b" {
		t.Errorf("expected only step-b to be stale, got %q", stale[0].StepID)
	}
}

func TestSelectCascadeTargets(t *testing.T) {
	stale := []StaleStep{
		{StepID: "step-b", Reasons: []string{"direct"}},
		{StepID: "step-c", Reasons: []string{"transitive"}},
		{StepID: "step-d", Reasons: []string{"transitive"}},
	}

	t.Run("select subset", func(t *testing.T) {
		selected := SelectCascadeTargets(stale, []string{"step-b", "step-d"})
		if len(selected) != 2 {
			t.Fatalf("expected 2 selected targets, got %d", len(selected))
		}
		if selected[0].StepID != "step-b" {
			t.Errorf("expected first selected to be step-b, got %q", selected[0].StepID)
		}
		if selected[1].StepID != "step-d" {
			t.Errorf("expected second selected to be step-d, got %q", selected[1].StepID)
		}
	})

	t.Run("empty selection", func(t *testing.T) {
		selected := SelectCascadeTargets(stale, []string{})
		if len(selected) != 0 {
			t.Errorf("expected 0 selected targets for empty selection, got %d", len(selected))
		}
	})

	t.Run("select nonexistent ID", func(t *testing.T) {
		selected := SelectCascadeTargets(stale, []string{"step-x"})
		if len(selected) != 0 {
			t.Errorf("expected 0 selected targets for nonexistent ID, got %d", len(selected))
		}
	})

	t.Run("select all", func(t *testing.T) {
		selected := SelectCascadeTargets(stale, []string{"step-b", "step-c", "step-d"})
		if len(selected) != 3 {
			t.Fatalf("expected 3 selected targets, got %d", len(selected))
		}
	})
}

func TestFormatStaleReport(t *testing.T) {
	stale := []StaleStep{
		{
			StepID:            "step-b",
			Reasons:           []string{`consumes artifact "output-a" from modified step "step-a"`},
			AffectedArtifacts: []string{"step-a:output-a"},
		},
		{
			StepID:            "step-c",
			Reasons:           []string{`consumes artifact "output-b" from transitively stale step "step-b"`},
			AffectedArtifacts: []string{"step-b:output-b"},
		},
	}

	report := FormatStaleReport(stale)

	if report == "" {
		t.Fatal("expected non-empty report")
	}

	// Verify it contains step IDs.
	if !strings.Contains(report, "step-b") {
		t.Error("report should contain step-b")
	}
	if !strings.Contains(report, "step-c") {
		t.Error("report should contain step-c")
	}

	// Verify it contains reasons.
	if !strings.Contains(report, "output-a") {
		t.Error("report should mention artifact output-a")
	}
	if !strings.Contains(report, "output-b") {
		t.Error("report should mention artifact output-b")
	}

	// Verify it mentions the count.
	if !strings.Contains(report, "2 downstream") {
		t.Error("report should mention '2 downstream'")
	}

	// Verify affected artifacts appear.
	if !strings.Contains(report, "step-a:output-a") {
		t.Error("report should contain affected artifact key step-a:output-a")
	}
}

func TestFormatStaleReport_Empty(t *testing.T) {
	report := FormatStaleReport(nil)
	if !strings.Contains(report, "No stale steps") {
		t.Errorf("expected 'No stale steps' message for empty input, got: %q", report)
	}

	report2 := FormatStaleReport([]StaleStep{})
	if !strings.Contains(report2, "No stale steps") {
		t.Errorf("expected 'No stale steps' message for empty slice, got: %q", report2)
	}
}

func TestVerifyStaleByMtime(t *testing.T) {
	tmpDir := t.TempDir()
	pipelineName := "mtime-pipeline"

	// Create workspace directories for source step (step-a) and consumer step (step-b).
	srcWs := filepath.Join(tmpDir, ".wave", "workspaces", pipelineName, "step-a")
	consumerWs := filepath.Join(tmpDir, ".wave", "workspaces", pipelineName, "step-b")

	if err := os.MkdirAll(srcWs, 0o755); err != nil {
		t.Fatalf("failed to create srcWs: %v", err)
	}
	if err := os.MkdirAll(consumerWs, 0o755); err != nil {
		t.Fatalf("failed to create consumerWs: %v", err)
	}

	// Create files in both workspaces.
	srcFile := filepath.Join(srcWs, "output.json")
	consumerFile := filepath.Join(consumerWs, "result.json")

	if err := os.WriteFile(srcFile, []byte(`{"status":"ok"}`), 0o644); err != nil {
		t.Fatalf("failed to write srcFile: %v", err)
	}
	if err := os.WriteFile(consumerFile, []byte(`{"status":"ok"}`), 0o644); err != nil {
		t.Fatalf("failed to write consumerFile: %v", err)
	}

	p := &Pipeline{
		Metadata: PipelineMetadata{Name: pipelineName},
		Steps: []Step{
			{ID: "step-a", OutputArtifacts: []ArtifactDef{{Name: "output-a"}}},
			{
				ID: "step-b",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{{Step: "step-a", Artifact: "output-a", As: "input_a"}},
				},
			},
		},
	}

	detector := NewCascadeDetector()

	t.Run("source newer than consumer means stale", func(t *testing.T) {
		// Make source workspace newer than consumer workspace.
		past := time.Now().Add(-1 * time.Hour)
		future := time.Now().Add(1 * time.Hour)

		if err := os.Chtimes(consumerFile, past, past); err != nil {
			t.Fatalf("failed to set consumerFile mtime: %v", err)
		}
		if err := os.Chtimes(srcFile, future, future); err != nil {
			t.Fatalf("failed to set srcFile mtime: %v", err)
		}

		staleInput := []StaleStep{
			{
				StepID:            "step-b",
				Reasons:           []string{"direct"},
				AffectedArtifacts: []string{"step-a:output-a"},
			},
		}

		confirmed := detector.VerifyStaleByMtime(staleInput, p, tmpDir)
		if len(confirmed) != 1 {
			t.Fatalf("expected 1 confirmed stale step, got %d", len(confirmed))
		}
		if confirmed[0].StepID != "step-b" {
			t.Errorf("expected confirmed stale step to be step-b, got %q", confirmed[0].StepID)
		}
	})

	t.Run("consumer newer than source means not stale", func(t *testing.T) {
		// Make consumer workspace newer than source workspace.
		past := time.Now().Add(-1 * time.Hour)
		future := time.Now().Add(1 * time.Hour)

		if err := os.Chtimes(srcFile, past, past); err != nil {
			t.Fatalf("failed to set srcFile mtime: %v", err)
		}
		if err := os.Chtimes(consumerFile, future, future); err != nil {
			t.Fatalf("failed to set consumerFile mtime: %v", err)
		}

		staleInput := []StaleStep{
			{
				StepID:            "step-b",
				Reasons:           []string{"direct"},
				AffectedArtifacts: []string{"step-a:output-a"},
			},
		}

		confirmed := detector.VerifyStaleByMtime(staleInput, p, tmpDir)
		if len(confirmed) != 0 {
			t.Errorf("expected 0 confirmed stale steps when consumer is newer, got %d", len(confirmed))
		}
	})

	t.Run("missing consumer workspace means stale", func(t *testing.T) {
		staleInput := []StaleStep{
			{
				StepID:            "step-nonexistent",
				Reasons:           []string{"direct"},
				AffectedArtifacts: []string{"step-a:output-a"},
			},
		}

		confirmed := detector.VerifyStaleByMtime(staleInput, p, tmpDir)
		if len(confirmed) != 1 {
			t.Fatalf("expected 1 confirmed stale step for missing workspace, got %d", len(confirmed))
		}
	})

	t.Run("missing source workspace means stale", func(t *testing.T) {
		staleInput := []StaleStep{
			{
				StepID:            "step-b",
				Reasons:           []string{"direct"},
				AffectedArtifacts: []string{"step-missing:output-x"},
			},
		}

		// Consumer workspace exists but source does not -- should be stale.
		// Reset consumer to a known time so the test is deterministic.
		now := time.Now()
		if err := os.Chtimes(consumerFile, now, now); err != nil {
			t.Fatalf("failed to set consumerFile mtime: %v", err)
		}

		confirmed := detector.VerifyStaleByMtime(staleInput, p, tmpDir)
		if len(confirmed) != 1 {
			t.Fatalf("expected 1 confirmed stale step for missing source workspace, got %d", len(confirmed))
		}
	})
}

func TestBuildArtifactGraph(t *testing.T) {
	p := &Pipeline{
		Metadata: PipelineMetadata{Name: "graph-pipeline"},
		Steps: []Step{
			{
				ID: "step-a",
				OutputArtifacts: []ArtifactDef{
					{Name: "output-a1", Path: ".wave/output/a1.json"},
					{Name: "output-a2", Path: ".wave/output/a2.json"},
				},
			},
			{
				ID: "step-b",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step-a", Artifact: "output-a1", As: "input_a1"},
					},
				},
				OutputArtifacts: []ArtifactDef{
					{Name: "output-b", Path: ".wave/output/b.json"},
				},
			},
			{
				ID: "step-c",
				Memory: MemoryConfig{
					InjectArtifacts: []ArtifactRef{
						{Step: "step-a", Artifact: "output-a2", As: "input_a2"},
						{Step: "step-b", Artifact: "output-b", As: "input_b"},
					},
				},
			},
		},
	}

	detector := NewCascadeDetector()
	produces, consumers := detector.buildArtifactGraph(p)

	// Verify produces map.
	if len(produces["step-a"]) != 2 {
		t.Errorf("expected step-a to produce 2 artifacts, got %d", len(produces["step-a"]))
	}
	if produces["step-a"][0] != "output-a1" || produces["step-a"][1] != "output-a2" {
		t.Errorf("unexpected produces for step-a: %v", produces["step-a"])
	}

	if len(produces["step-b"]) != 1 {
		t.Errorf("expected step-b to produce 1 artifact, got %d", len(produces["step-b"]))
	}
	if produces["step-b"][0] != "output-b" {
		t.Errorf("unexpected produces for step-b: %v", produces["step-b"])
	}

	// step-c has no output artifacts.
	if len(produces["step-c"]) != 0 {
		t.Errorf("expected step-c to produce 0 artifacts, got %d", len(produces["step-c"]))
	}

	// Verify consumers map.
	if len(consumers["step-a"]) != 0 {
		t.Errorf("expected step-a to consume 0 artifacts, got %d", len(consumers["step-a"]))
	}

	if len(consumers["step-b"]) != 1 {
		t.Errorf("expected step-b to consume 1 artifact, got %d", len(consumers["step-b"]))
	}
	if consumers["step-b"][0].SourceStep != "step-a" || consumers["step-b"][0].ArtifactName != "output-a1" {
		t.Errorf("unexpected consumers for step-b: %+v", consumers["step-b"])
	}

	if len(consumers["step-c"]) != 2 {
		t.Errorf("expected step-c to consume 2 artifacts, got %d", len(consumers["step-c"]))
	}
	if consumers["step-c"][0].SourceStep != "step-a" || consumers["step-c"][0].ArtifactName != "output-a2" {
		t.Errorf("unexpected first consumer for step-c: %+v", consumers["step-c"][0])
	}
	if consumers["step-c"][1].SourceStep != "step-b" || consumers["step-c"][1].ArtifactName != "output-b" {
		t.Errorf("unexpected second consumer for step-c: %+v", consumers["step-c"][1])
	}
}

func TestSplitArtifactKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "standard key",
			input:    "step:artifact",
			expected: []string{"step", "artifact"},
		},
		{
			name:     "hyphenated key",
			input:    "step-a:output-1",
			expected: []string{"step-a", "output-1"},
		},
		{
			name:     "no colon",
			input:    "nocolon",
			expected: []string{"nocolon"},
		},
		{
			name:     "multiple colons only splits on first",
			input:    "step:artifact:extra",
			expected: []string{"step", "artifact:extra"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "colon at start",
			input:    ":artifact",
			expected: []string{"", "artifact"},
		},
		{
			name:     "colon at end",
			input:    "step:",
			expected: []string{"step", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitArtifactKey(tt.input)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d parts, got %d: %v", len(tt.expected), len(result), result)
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("part[%d]: expected %q, got %q", i, tt.expected[i], v)
				}
			}
		})
	}
}
