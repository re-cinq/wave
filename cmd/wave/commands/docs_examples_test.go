package commands

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Owner rule: command examples in the docs use codex with gpt-6.2-sol or
// claude with sonnet. Regression for docs/reference/cli.md showing
// `--adapter opencode --model zai-coding-plan/glm-5-turbo`.
var (
	docsWaveCmd     = regexp.MustCompile(`\bwave (run|init|meta|resume)\b`)
	docsAdapterFlag = regexp.MustCompile(`--adapter[ =]"?([A-Za-z0-9_.<>/-]+)`)
	docsModelFlag   = regexp.MustCompile(`--model[ =]"?([A-Za-z0-9_.<>/:-]+)`)
)

var allowedExampleAdapters = map[string]bool{"claude": true, "codex": true}

// Tier names are adapter-neutral; <id>-style placeholders are not examples.
var allowedExampleModels = map[string]bool{
	"sonnet": true, "gpt-6.2-sol": true,
	"cheapest": true, "balanced": true, "strongest": true,
}

func TestDocsCommandExamplesUseOwnerAdapters(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	files := []string{filepath.Join(root, "AGENTS.md"), filepath.Join(root, "README.md")}
	err := filepath.Walk(filepath.Join(root, "docs"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && (info.Name() == "node_modules" || strings.HasPrefix(info.Name(), ".")) {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk docs: %v", err)
	}

	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for n := 1; sc.Scan(); n++ {
			line := sc.Text()
			if !docsWaveCmd.MatchString(line) {
				continue
			}
			for _, m := range docsAdapterFlag.FindAllStringSubmatch(line, -1) {
				if a := m[1]; !allowedExampleAdapters[a] && !strings.HasPrefix(a, "<") {
					t.Errorf("%s:%d: example uses --adapter %s; use codex or claude", path, n, a)
				}
			}
			for _, m := range docsModelFlag.FindAllStringSubmatch(line, -1) {
				if mo := m[1]; !allowedExampleModels[mo] && !strings.HasPrefix(mo, "<") {
					t.Errorf("%s:%d: example uses --model %s; use gpt-6.2-sol (codex) or sonnet (claude)", path, n, mo)
				}
			}
		}
		_ = f.Close()
		if err := sc.Err(); err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
	}
}
