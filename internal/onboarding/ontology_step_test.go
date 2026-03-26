package onboarding

import (
	"testing"

	"github.com/recinq/wave/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOntologyStep_Name(t *testing.T) {
	step := &OntologyStep{}
	assert.Equal(t, "Project Ontology", step.Name())
}

func TestOntologyStep_NonInteractive_NoExisting(t *testing.T) {
	step := &OntologyStep{}
	cfg := &WizardConfig{Interactive: false}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	assert.True(t, result.Skipped, "should skip when no telos and no contexts")
}

func TestOntologyStep_NonInteractive_Reconfigure(t *testing.T) {
	step := &OntologyStep{}
	cfg := &WizardConfig{
		Interactive: false,
		Reconfigure: true,
		Existing: &manifest.Manifest{
			Ontology: &manifest.Ontology{
				Telos: "Enable conversational finance",
				Contexts: []manifest.OntologyContext{
					{Name: "identity"},
					{Name: "analytics"},
				},
			},
		},
	}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	assert.False(t, result.Skipped)
	assert.Equal(t, "Enable conversational finance", result.Data["telos"])
	contexts, ok := result.Data["contexts"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"identity", "analytics"}, contexts)
}

func TestOntologyStep_NonInteractive_ReconfigureNoOntology(t *testing.T) {
	step := &OntologyStep{}
	cfg := &WizardConfig{
		Interactive: false,
		Reconfigure: true,
		Existing:    &manifest.Manifest{},
	}

	result, err := step.Run(cfg)
	require.NoError(t, err)
	assert.True(t, result.Skipped, "should skip when existing manifest has no ontology")
}

func TestParseContextNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"spaces only", "   ", nil},
		{"single", "identity", []string{"identity"}},
		{"multiple", "identity, analytics, auth", []string{"identity", "analytics", "auth"}},
		{"trailing comma", "identity, analytics,", []string{"identity", "analytics"}},
		{"extra spaces", "  identity ,  analytics  ", []string{"identity", "analytics"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseContextNames(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
