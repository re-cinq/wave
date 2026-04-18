package pipeline_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/pipeline"
	"gopkg.in/yaml.v3"
)

func TestWaveTestHardeningGraph(t *testing.T) {
	p := filepath.Join("..", "..", ".agents", "pipelines", "wave-test-hardening.yaml")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}

	var pl pipeline.Pipeline
	// Use the same strict decoder as YAMLPipelineLoader.Unmarshal
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&pl); err != nil {
		t.Fatalf("yaml parse: %v", err)
	}

	t.Logf("Steps: %d", len(pl.Steps))
	for _, s := range pl.Steps {
		t.Logf("  Step %q type=%q edges=%d", s.ID, s.Type, len(s.Edges))
		for _, e := range s.Edges {
			t.Logf("    Edge target=%q sentinel=%q match=%v", e.Target, pipeline.EdgeTargetComplete, e.Target == pipeline.EdgeTargetComplete)
		}
	}

	v := &pipeline.DAGValidator{}
	if err := v.ValidateGraph(&pl); err != nil {
		t.Fatalf("validation error: %v", err)
	}
}
