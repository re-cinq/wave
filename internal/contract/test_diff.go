package contract

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// testDiffValidator enforces a ceiling on net test-function deletions in the
// current diff. Catches the "satisfy tests-pass gate by deleting failing
// test" failure mode (real-world hit during epic #1565 phase 1.5a where a
// persona removed a passing-by-design test instead of correcting its
// build-tag scope).
//
// Operates on the unified diff of `*_test.go` files between HEAD and the
// working tree. Net deletions = (count of `^-func Test*` lines) -
// (count of `^+func Test*` lines). When the result exceeds MaxTestDeletions
// the contract fails. Renames (one removed + one added) net to zero, so
// legitimate refactors pass.
type testDiffValidator struct{}

// testFuncLineRe matches lines that introduce a top-level `func Test*` in
// either a + or - diff line. Tabs/spaces accepted before `func`.
var testFuncLineRe = regexp.MustCompile(`^[ \t]*func[ \t]+(Test|Example|Benchmark|Fuzz)[A-Za-z0-9_]*\b`)

func (v *testDiffValidator) Validate(cfg ContractConfig, workspacePath string) error {
	max := cfg.MaxTestDeletions
	// max defaults to 0 — any net deletion is rejected unless the pipeline
	// yaml opts in to a higher tolerance.

	cmd := exec.Command("git", "diff", "HEAD", "--", "*_test.go")
	cmd.Dir = workspacePath
	out, err := cmd.Output()
	if err != nil {
		// No HEAD (initial commit) or git not present — skip silently.
		// The companion source_diff contract will surface that anomaly.
		return nil
	}

	added, removed := 0, 0
	for _, line := range strings.Split(string(out), "\n") {
		// diff metadata lines (`+++`, `---`, `@@`) start with the same
		// characters but include text after the marker that doesn't match
		// our regex, so they are filtered naturally by the regex below.
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			continue
		case strings.HasPrefix(line, "+"):
			if testFuncLineRe.MatchString(line[1:]) {
				added++
			}
		case strings.HasPrefix(line, "-"):
			if testFuncLineRe.MatchString(line[1:]) {
				removed++
			}
		}
	}

	net := removed - added
	if net > max {
		return fmt.Errorf("test_diff: net %d test function(s) removed (max allowed %d) — "+
			"removed=%d added=%d. Persona must replace deleted tests, not net-delete them. "+
			"Common cause: build-tag scoping issue mistaken for a broken test.",
			net, max, removed, added)
	}
	return nil
}
