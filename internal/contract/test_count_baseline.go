package contract

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

// testCountBaselineValidator is the post-commit "last line of defense"
// against test deletions. Where test_diff inspects the working-tree
// diff, this one compares COMMITTED tree counts: HEAD vs BaseRef
// (default HEAD~1). Catches deletions that slipped past diff inspection
// (file moves, force-pushes within session, multi-commit sequences).
//
// Language-agnostic: shares TestFilePattern + TestFuncPattern with
// test_diff so a project configures patterns once.
//
// Operation:
//  1. `git ls-tree -r --name-only <ref>` → filter by TestFilePattern globs
//  2. `git show <ref>:<path>` per file → count regex matches
//  3. Fail if (base - head) > MaxTestDeletions
type testCountBaselineValidator struct{}

func (v *testCountBaselineValidator) Validate(cfg ContractConfig, workspacePath string) error {
	baseRef := cfg.BaseRef
	if baseRef == "" {
		baseRef = "HEAD~1"
	}
	max := cfg.MaxTestDeletions

	globs := cfg.TestFilePattern
	if len(globs) == 0 {
		globs = []string{defaultTestFilePathspec}
	}
	patternStr := cfg.TestFuncPattern
	if patternStr == "" {
		patternStr = defaultTestFuncPattern
	}
	re, err := regexp.Compile(patternStr)
	if err != nil {
		return fmt.Errorf("test_count_baseline: invalid TestFuncPattern %q: %w", patternStr, err)
	}

	headCount, err := countTestFuncsAtRef(workspacePath, "HEAD", globs, re)
	if err != nil {
		return nil
	}
	baseCount, err := countTestFuncsAtRef(workspacePath, baseRef, globs, re)
	if err != nil {
		return nil
	}

	net := baseCount - headCount
	if net > max {
		return fmt.Errorf("test_count_baseline: HEAD has %d test declarations vs %s=%d (net deletion %d, max allowed %d); persona must replace removed tests, not net-delete them across commits",
			headCount, baseRef, baseCount, net, max)
	}
	return nil
}

func countTestFuncsAtRef(dir, ref string, globs []string, re *regexp.Regexp) (int, error) {
	out, err := runGitCmd(dir, "ls-tree", "-r", "--name-only", ref)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, p := range strings.Split(strings.TrimSpace(out), "\n") {
		if p == "" || !matchesAnyGlob(p, globs) {
			continue
		}
		blob, err := runGitCmd(dir, "show", ref+":"+p)
		if err != nil {
			continue
		}
		total += len(re.FindAllString(blob, -1))
	}
	return total, nil
}

// matchesAnyGlob checks both the full path and the basename against each
// pattern. path.Match doesn't grok `**`, so basename match is a
// pragmatic fallback covering `*_test.go`, `test_*.py`, `*.test.ts`, etc.
func matchesAnyGlob(p string, globs []string) bool {
	base := path.Base(p)
	for _, g := range globs {
		if ok, _ := path.Match(g, p); ok {
			return true
		}
		if ok, _ := path.Match(g, base); ok {
			return true
		}
	}
	return false
}

func runGitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}
