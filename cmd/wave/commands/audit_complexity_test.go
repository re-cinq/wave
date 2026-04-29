package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/recinq/wave/internal/complexity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditCmd_Structure(t *testing.T) {
	cmd := NewAuditCmd()
	assert.Equal(t, "audit", cmd.Use)
	require.NotNil(t, cmd.Commands())
	var hasComplexity bool
	for _, c := range cmd.Commands() {
		if c.Name() == "complexity" {
			hasComplexity = true
			break
		}
	}
	assert.True(t, hasComplexity, "expected complexity subcommand")
}

func TestNewAuditComplexityCmd_Flags(t *testing.T) {
	cmd := newAuditComplexityCmd()
	require.NotNil(t, cmd.Flags().Lookup("max-cyclomatic"))
	require.NotNil(t, cmd.Flags().Lookup("max-cognitive"))
	require.NotNil(t, cmd.Flags().Lookup("warn-cyclomatic"))
	require.NotNil(t, cmd.Flags().Lookup("warn-cognitive"))
	require.NotNil(t, cmd.Flags().Lookup("output"))
	require.NotNil(t, cmd.Flags().Lookup("exclude"))
	require.NotNil(t, cmd.Flags().Lookup("format"))
	require.NotNil(t, cmd.Flags().Lookup("include-tests"))
	assert.Equal(t, "15", cmd.Flags().Lookup("max-cyclomatic").DefValue)
	assert.Equal(t, "15", cmd.Flags().Lookup("max-cognitive").DefValue)
	assert.Equal(t, "10", cmd.Flags().Lookup("warn-cyclomatic").DefValue)
	assert.Equal(t, "10", cmd.Flags().Lookup("warn-cognitive").DefValue)
}

// runAuditComplexity runs the subcommand with the given args, returning stdout,
// stderr, the error from RunE, and the resolved exit code.
func runAuditComplexity(t *testing.T, args []string) (stdout, stderr string, exit int) {
	t.Helper()
	cmd := newAuditComplexityCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), ExitCodeFor(err)
}

func TestAuditComplexity_Pass(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ok.go")
	require.NoError(t, os.WriteFile(srcPath,
		[]byte("package x\nfunc Easy() int { return 1 }\n"), 0o644))
	outPath := filepath.Join(tmp, "findings.json")

	_, stderr, exit := runAuditComplexity(t, []string{
		"--output", outPath,
		srcPath,
	})
	assert.Equal(t, 0, exit, "stderr=%s", stderr)
	body, err := os.ReadFile(outPath)
	require.NoError(t, err)
	var doc complexity.FindingsDocument
	require.NoError(t, json.Unmarshal(body, &doc))
	assert.Empty(t, doc.Findings, "expected zero findings")
	assert.Equal(t, "complexity", doc.ScanType)
}

func TestAuditComplexity_Breach(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "big.go")
	// Function with cyclomatic >= 5 — set a low fail threshold so we breach.
	src := "package x\nfunc Big(x int) int {\n" +
		"  if x > 0 { return 1 }\n" +
		"  if x < 0 { return -1 }\n" +
		"  if x == 5 { return 5 }\n" +
		"  return 0\n}\n"
	require.NoError(t, os.WriteFile(srcPath, []byte(src), 0o644))
	outPath := filepath.Join(tmp, "findings.json")

	stdout, stderr, exit := runAuditComplexity(t, []string{
		"--max-cyclomatic", "2",
		"--warn-cyclomatic", "1",
		"--output", outPath,
		srcPath,
	})
	assert.Equal(t, 1, exit, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stderr, "BREACH")
	assert.Contains(t, stderr, "Big")

	body, err := os.ReadFile(outPath)
	require.NoError(t, err)
	var doc complexity.FindingsDocument
	require.NoError(t, json.Unmarshal(body, &doc))
	assert.True(t, doc.HasBreach())
	require.NotEmpty(t, doc.Findings)
	assert.Equal(t, "high", doc.Findings[0].Severity)
}

func TestAuditComplexity_ParseError(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "broken.go")
	require.NoError(t, os.WriteFile(srcPath,
		[]byte("package x\nfunc Broken() int { return 1 +"), 0o644))
	outPath := filepath.Join(tmp, "findings.json")

	_, _, exit := runAuditComplexity(t, []string{
		"--output", outPath,
		srcPath,
	})
	assert.Equal(t, 2, exit, "expected IO/parse exit code 2")
}

func TestAuditComplexity_MissingPath(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "nope")
	outPath := filepath.Join(tmp, "findings.json")

	_, _, exit := runAuditComplexity(t, []string{
		"--output", outPath,
		missing,
	})
	assert.Equal(t, 2, exit)
}

func TestAuditComplexity_SummaryFormat(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "ok.go")
	require.NoError(t, os.WriteFile(srcPath,
		[]byte("package x\nfunc Easy() int { return 1 }\n"), 0o644))

	stdout, _, exit := runAuditComplexity(t, []string{
		"--format", "summary",
		srcPath,
	})
	assert.Equal(t, 0, exit)
	assert.Contains(t, stdout, "scanned")
}

func TestExitCodeFor(t *testing.T) {
	assert.Equal(t, 0, ExitCodeFor(nil))
	assert.Equal(t, 1, ExitCodeFor(errors.New("plain")))
	assert.Equal(t, 2, ExitCodeFor(cliExitErr(2, errors.New("io"))))
	assert.Equal(t, 1, ExitCodeFor(cliExitErr(1, errors.New("breach"))))
}

func TestNormalizeAuditPaths(t *testing.T) {
	assert.Equal(t, []string{"."}, normalizeAuditPaths(nil))
	assert.Equal(t, []string{"./internal"}, normalizeAuditPaths([]string{"./internal/..."}))
	assert.Equal(t, []string{"a", "b"}, normalizeAuditPaths([]string{"a", "b"}))
}
