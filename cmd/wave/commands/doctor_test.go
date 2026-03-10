package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/recinq/wave/internal/doctor"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newDoctorCmdWithRoot creates a doctor command under a root that has
// the standard persistent flags, mirroring the real CLI structure.
func newDoctorCmdWithRoot() *cobra.Command {
	root := &cobra.Command{Use: "wave"}
	root.PersistentFlags().String("manifest", "wave.yaml", "Path to wave.yaml")
	root.PersistentFlags().String("output", "auto", "Output format")
	root.PersistentFlags().Bool("json", false, "JSON output")
	root.PersistentFlags().Bool("quiet", false, "Quiet output")
	root.PersistentFlags().Bool("verbose", false, "Verbose output")
	root.PersistentFlags().Bool("debug", false, "Debug mode")
	root.PersistentFlags().Bool("no-color", false, "Disable color")
	root.PersistentFlags().Bool("no-tui", false, "Disable TUI")
	doctorCmd := NewDoctorCmd()
	root.AddCommand(doctorCmd)
	return root
}

func TestDoctorOptimize_DryRunRequiresOptimize(t *testing.T) {
	root := newDoctorCmdWithRoot()
	root.SetArgs([]string{"doctor", "--dry-run"})

	var errBuf bytes.Buffer
	root.SetErr(&errBuf)

	err := root.Execute()
	require.Error(t, err, "expected error when --dry-run used without --optimize")
	assert.Contains(t, err.Error(), "--dry-run requires --optimize")
}

func TestDoctorOptimize_YesRequiresOptimize(t *testing.T) {
	root := newDoctorCmdWithRoot()
	root.SetArgs([]string{"doctor", "--yes"})

	var errBuf bytes.Buffer
	root.SetErr(&errBuf)

	err := root.Execute()
	require.Error(t, err, "expected error when --yes used without --optimize")
	assert.Contains(t, err.Error(), "--yes requires --optimize")
}

func TestDoctorOptimize_SkipAIRequiresOptimize(t *testing.T) {
	root := newDoctorCmdWithRoot()
	root.SetArgs([]string{"doctor", "--skip-ai"})

	var errBuf bytes.Buffer
	root.SetErr(&errBuf)

	err := root.Execute()
	require.Error(t, err, "expected error when --skip-ai used without --optimize")
	assert.Contains(t, err.Error(), "--skip-ai requires --optimize")
}

func TestDoctorOptimize_YesShortFlag(t *testing.T) {
	root := newDoctorCmdWithRoot()

	// Just verify the short flag -y is recognized (it will fail for other reasons
	// since we don't have a real project, but the flag parsing should work)
	root.SetArgs([]string{"doctor", "-y"})

	err := root.Execute()
	require.Error(t, err, "expected error since -y requires --optimize")
	assert.Contains(t, err.Error(), "--yes requires --optimize")
}

func TestRenderConfigChange_WithChange(t *testing.T) {
	var buf bytes.Buffer
	change := doctor.ConfigChange{
		Field:    "project.test_command",
		Current:  "go test ./...",
		Proposed: "make test",
		Reason:   "Makefile 'test' target detected",
		Source:   "makefile",
	}

	renderConfigChange(&buf, change)

	output := buf.String()
	assert.Contains(t, output, "project.test_command:")
	assert.Contains(t, output, "- current:  go test ./...")
	assert.Contains(t, output, "+ proposed: make test")
	assert.Contains(t, output, "Makefile 'test' target detected")
}

func TestRenderConfigChange_NoChange(t *testing.T) {
	var buf bytes.Buffer
	change := doctor.ConfigChange{
		Field:    "project.language",
		Current:  "go",
		Proposed: "go",
		Reason:   "confirmed",
		Source:   "profile",
	}

	renderConfigChange(&buf, change)

	output := buf.String()
	assert.Contains(t, output, "project.language:")
	assert.Contains(t, output, "(no change)")
	assert.NotContains(t, output, "- current:")
	assert.NotContains(t, output, "+ proposed:")
}

func TestRenderConfigChange_EmptyCurrent(t *testing.T) {
	var buf bytes.Buffer
	change := doctor.ConfigChange{
		Field:    "project.build_command",
		Current:  "",
		Proposed: "go build ./...",
		Reason:   "detected Go project",
		Source:   "profile",
	}

	renderConfigChange(&buf, change)

	output := buf.String()
	assert.Contains(t, output, "- current:  (not set)")
	assert.Contains(t, output, "+ proposed: go build ./...")
}

func TestRenderPipelineRec_Recommended(t *testing.T) {
	var buf bytes.Buffer
	rec := doctor.PipelineRecommendation{
		Name:        "gh-implement",
		Recommended: true,
		Reason:      "matches detected forge (github)",
	}

	renderPipelineRec(&buf, rec)

	output := buf.String()
	assert.Contains(t, output, "+ gh-implement")
	assert.Contains(t, output, "matches detected forge (github)")
}

func TestRenderPipelineRec_NotRecommended(t *testing.T) {
	var buf bytes.Buffer
	rec := doctor.PipelineRecommendation{
		Name:        "bb-implement",
		Recommended: false,
		Reason:      "requires Bitbucket forge, but project uses github",
	}

	renderPipelineRec(&buf, rec)

	output := buf.String()
	assert.Contains(t, output, "x bb-implement")
	assert.Contains(t, output, "requires Bitbucket forge")
}

func TestRenderOptimizeText_FullResult(t *testing.T) {
	var buf bytes.Buffer
	result := &doctor.OptimizeResult{
		ProjectChanges: []doctor.ConfigChange{
			{
				Field:    "project.language",
				Current:  "go",
				Proposed: "go",
				Reason:   "confirmed",
				Source:   "profile",
			},
			{
				Field:    "project.test_command",
				Current:  "go test ./...",
				Proposed: "make test",
				Reason:   "Makefile target",
				Source:   "makefile",
			},
		},
		PipelineRecs: []doctor.PipelineRecommendation{
			{Name: "gh-implement", Recommended: true, Reason: "GitHub project"},
			{Name: "bb-implement", Recommended: false, Reason: "Not a Bitbucket project"},
		},
		Conventions: []string{
			"commit format: conventional commits",
			"editorconfig configured",
		},
	}

	renderOptimizeText(&buf, result)

	output := buf.String()

	// Project changes section
	assert.Contains(t, output, "Proposed wave.yaml changes:")
	assert.Contains(t, output, "project.language:")
	assert.Contains(t, output, "(no change)")
	assert.Contains(t, output, "project.test_command:")
	assert.Contains(t, output, "- current:")
	assert.Contains(t, output, "+ proposed:")

	// Pipeline recommendations section
	assert.Contains(t, output, "Pipeline recommendations:")
	assert.Contains(t, output, "+ gh-implement")
	assert.Contains(t, output, "x bb-implement")

	// Conventions section
	assert.Contains(t, output, "Detected conventions:")
	assert.Contains(t, output, "conventional commits")
	assert.Contains(t, output, "editorconfig configured")
}

func TestRenderOptimizeText_EmptyResult(t *testing.T) {
	var buf bytes.Buffer
	result := &doctor.OptimizeResult{}

	renderOptimizeText(&buf, result)

	output := buf.String()
	assert.Contains(t, output, "No wave.yaml changes proposed.")
	assert.NotContains(t, output, "Pipeline recommendations:")
	assert.NotContains(t, output, "Detected conventions:")
}

func TestDisplayValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "(not set)"},
		{"go test ./...", "go test ./..."},
		{"make build", "make build"},
	}

	for _, tt := range tests {
		got := displayValue(tt.input)
		assert.Equal(t, tt.expected, got, "displayValue(%q)", tt.input)
	}
}

func TestPromptConfirm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
		wantErr  bool
	}{
		{"empty input (default yes)", "\n", true, false},
		{"y", "y\n", true, false},
		{"Y", "Y\n", true, false},
		{"yes", "yes\n", true, false},
		{"Yes", "Yes\n", true, false},
		{"n", "n\n", false, false},
		{"N", "N\n", false, false},
		{"no", "no\n", false, false},
		{"arbitrary text", "maybe\n", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf bytes.Buffer
			in := strings.NewReader(tt.input)

			got, err := promptConfirm(in, &outBuf, "Confirm? ")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
			assert.Contains(t, outBuf.String(), "Confirm?")
		})
	}
}

func TestPromptConfirm_EOF(t *testing.T) {
	var outBuf bytes.Buffer
	in := strings.NewReader("") // EOF immediately

	got, err := promptConfirm(in, &outBuf, "Confirm? ")
	require.NoError(t, err)
	assert.False(t, got, "EOF should return false")
}

func TestDiscoverPipelineNames(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some pipeline files
	require.NoError(t, writeTestFile(tmpDir, "gh-implement.yaml", "kind: Pipeline"))
	require.NoError(t, writeTestFile(tmpDir, "speckit-flow.yml", "kind: Pipeline"))
	require.NoError(t, writeTestFile(tmpDir, "bb-implement.yaml", "kind: Pipeline"))
	require.NoError(t, writeTestFile(tmpDir, "README.md", "not a pipeline"))

	names := discoverPipelineNames(tmpDir)
	assert.Len(t, names, 3)
	assert.Contains(t, names, "gh-implement")
	assert.Contains(t, names, "speckit-flow")
	assert.Contains(t, names, "bb-implement")
}

func TestDiscoverPipelineNames_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	names := discoverPipelineNames(tmpDir)
	assert.Empty(t, names)
}

func TestDiscoverPipelineNames_NonExistentDir(t *testing.T) {
	names := discoverPipelineNames("/nonexistent/path")
	assert.Nil(t, names)
}

func TestFormatScanSummary(t *testing.T) {
	tests := []struct {
		name     string
		profile  *doctor.ProjectProfile
		expected string
	}{
		{"nil profile", nil, "no data"},
		{"zero files", &doctor.ProjectProfile{}, "scan complete"},
		{"with files", &doctor.ProjectProfile{FilesScanned: 1247}, "1247 files scanned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatScanSummary(tt.profile)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// writeTestFile creates a file in the given directory with the given content.
func writeTestFile(dir, name, content string) error {
	return os.WriteFile(dir+"/"+name, []byte(content), 0o644)
}
