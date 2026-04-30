package contract

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// testDiffValidator enforces a ceiling on net test-function deletions in
// the current diff. Catches the "satisfy tests-pass gate by deleting
// failing test" failure mode (real-world hit during epic #1565 phase
// 1.5a where a persona removed a passing-by-design test instead of
// correcting its build-tag scope).
//
// Language-agnostic: callers configure TestFilePattern (pathspecs handed
// to `git diff -- ...`) and TestFuncPattern (regex matching one test
// declaration per line). Defaults match Go (`*_test.go`,
// `^[ \t]*func[ \t]+(Test|Example|Benchmark|Fuzz)\w*\b`) for back-compat
// and out-of-the-box use; project pipelines override per ecosystem.
//
// Net deletions = (count of `-` test-decl lines) - (count of `+` test-decl
// lines). Renames (one removed + one added) net to zero. When the result
// exceeds MaxTestDeletions the contract fails.
type testDiffValidator struct{}

const (
	defaultTestFilePathspec = "*_test.go"
	defaultTestFuncPattern  = `^[ \t]*func[ \t]+(Test|Example|Benchmark|Fuzz)[A-Za-z0-9_]*\b`
)

func (v *testDiffValidator) Validate(cfg ContractConfig, workspacePath string) error {
	max := cfg.MaxTestDeletions

	pathspecs := cfg.TestFilePattern
	if len(pathspecs) == 0 {
		pathspecs = []string{defaultTestFilePathspec}
	}
	patternStr := cfg.TestFuncPattern
	if patternStr == "" {
		patternStr = defaultTestFuncPattern
	}
	re, err := regexp.Compile(patternStr)
	if err != nil {
		return fmt.Errorf("test_diff: invalid TestFuncPattern %q: %w", patternStr, err)
	}

	args := append([]string{"diff", "HEAD", "--"}, pathspecs...)
	cmd := exec.Command("git", args...)
	cmd.Dir = workspacePath
	out, err := cmd.Output()
	if err != nil {
		// No HEAD (initial commit) or git not present — skip silently.
		// The companion source_diff contract surfaces that anomaly.
		return nil
	}

	added, removed := 0, 0
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			continue
		case strings.HasPrefix(line, "+"):
			if re.MatchString(line[1:]) {
				added++
			}
		case strings.HasPrefix(line, "-"):
			if re.MatchString(line[1:]) {
				removed++
			}
		}
	}

	net := removed - added
	if net > max {
		return fmt.Errorf("test_diff: net %d test declaration(s) removed (max allowed %d, removed=%d added=%d); persona must replace deleted tests, not net-delete them",
			net, max, removed, added)
	}
	return nil
}
