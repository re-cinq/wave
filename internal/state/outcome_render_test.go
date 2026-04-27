package state

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutcomeRecordString_Branch(t *testing.T) {
	t.Setenv("NERD_FONT", "")
	t.Setenv("TERM", "dumb")
	t.Setenv("TERMINAL_EMULATOR", "")

	r := &OutcomeRecord{
		Type:        OutcomeTypeBranch,
		Label:       "feat/branch",
		Value:       "/ws/path",
		Description: "desc",
		StepID:      "step-1",
	}

	out := r.String()
	assert.NotEmpty(t, out)
	assert.True(t, strings.Contains(out, "/ws/path"), "expected String() to contain path, got %q", out)
}

func TestOutcomeRecordString_Issue(t *testing.T) {
	t.Setenv("NERD_FONT", "")
	t.Setenv("TERM", "dumb")
	t.Setenv("TERMINAL_EMULATOR", "")

	r := &OutcomeRecord{
		Type:   OutcomeTypeIssue,
		Label:  "Issue #1",
		Value:  "https://github.com/org/repo/issues/1",
		StepID: "step-2",
	}

	out := r.String()
	assert.NotEmpty(t, out)
	assert.True(t, strings.Contains(out, "https://github.com/org/repo/issues/1"))
}

func TestOutcomeRecordString_FileUsesFileURI(t *testing.T) {
	t.Setenv("NERD_FONT", "")
	r := &OutcomeRecord{Type: OutcomeTypeFile, Value: "/tmp/out.json", StepID: "s1", Label: "out"}
	out := r.String()
	assert.True(t, strings.Contains(out, "file://"), "file outcome should render via file:// URI, got %q", out)
}

func TestOutcomeRecordIsTemporary(t *testing.T) {
	cases := []struct {
		name string
		r    OutcomeRecord
		want bool
	}{
		{"log type", OutcomeRecord{Type: OutcomeTypeLog}, true},
		{"description contains temporary", OutcomeRecord{Type: OutcomeTypeFile, Description: "TEMPORARY junk"}, true},
		{"label contains temp", OutcomeRecord{Type: OutcomeTypeFile, Label: "tempfile.txt"}, true},
		{"plain file", OutcomeRecord{Type: OutcomeTypeFile, Label: "report.md", Description: "report"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.r.IsTemporary())
		})
	}
}

func TestHasNerdFont(t *testing.T) {
	t.Setenv("NERD_FONT", "1")
	assert.True(t, hasNerdFont())

	t.Setenv("NERD_FONT", "")
	t.Setenv("TERM", "kitty")
	t.Setenv("TERMINAL_EMULATOR", "")
	assert.True(t, hasNerdFont())

	t.Setenv("TERM", "dumb")
	t.Setenv("TERMINAL_EMULATOR", "JetBrains")
	assert.True(t, hasNerdFont())

	t.Setenv("TERMINAL_EMULATOR", "")
	assert.False(t, hasNerdFont())
}
