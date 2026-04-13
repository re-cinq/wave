// Package tools provides shared utilities for checking CLI tool availability.
package tools

import "os/exec"

// PathResult is the outcome of checking a single tool on PATH.
type PathResult struct {
	Name  string
	Found bool
}

// CheckOnPath checks each tool name using lookPath and returns results in
// input order. Empty tool names are skipped. If lookPath is nil, exec.LookPath
// is used. This is the shared core for doctor's checkRequiredTools and
// preflight's CheckTools — both packages build their own result types on top.
func CheckOnPath(lookPath func(string) (string, error), tools []string) []PathResult {
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	results := make([]PathResult, 0, len(tools))
	for _, t := range tools {
		if t == "" {
			continue
		}
		_, err := lookPath(t)
		results = append(results, PathResult{Name: t, Found: err == nil})
	}
	return results
}
