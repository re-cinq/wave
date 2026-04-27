package complexity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyze_EmptyPaths(t *testing.T) {
	report, err := Analyze(nil, Options{})
	if err != nil {
		t.Fatalf("Analyze(nil) error = %v, want nil", err)
	}
	if len(report.Scores) != 0 {
		t.Fatalf("Analyze(nil) scored %d funcs, want 0", len(report.Scores))
	}
}

func TestAnalyze_FixturesFile(t *testing.T) {
	report, err := Analyze([]string{"testdata/fixtures.go"}, Options{})
	if err != nil {
		t.Fatalf("Analyze fixtures: %v", err)
	}
	if len(report.Scores) != 12 {
		t.Fatalf("Analyze produced %d scores, want 12", len(report.Scores))
	}
	// All scores should carry their package name.
	for _, s := range report.Scores {
		if s.Package != "fixtures" {
			t.Fatalf("score for %s has package=%q, want fixtures", s.Function, s.Package)
		}
	}
}

func TestAnalyze_BrokenFile(t *testing.T) {
	_, err := Analyze([]string{"testdata/broken/broken.go"}, Options{})
	if err == nil {
		t.Fatalf("Analyze(broken) error = nil, want parse error")
	}
	if !strings.Contains(err.Error(), "broken.go") {
		t.Fatalf("error = %q, want path in message", err.Error())
	}
}

func TestAnalyze_MixedDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ok.go"),
		[]byte("package x\nfunc F() int { return 0 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"),
		[]byte("# not Go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "extra_test.go"),
		[]byte("package x\nfunc TestF() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Analyze([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Analyze mixed dir: %v", err)
	}
	if len(report.Scores) != 1 {
		t.Fatalf("expected 1 scored func (ok.go::F only), got %d", len(report.Scores))
	}
	// Now include tests; should pick up both.
	report, err = Analyze([]string{dir}, Options{IncludeTests: true})
	if err != nil {
		t.Fatalf("Analyze mixed dir with tests: %v", err)
	}
	if len(report.Scores) != 2 {
		t.Fatalf("expected 2 scored funcs with IncludeTests, got %d", len(report.Scores))
	}
}

func TestAnalyze_ExcludePattern(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "keep.go"),
		[]byte("package x\nfunc K() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.go"),
		[]byte("package x\nfunc S() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Analyze([]string{dir}, Options{Excludes: []string{"skip"}})
	if err != nil {
		t.Fatalf("Analyze with exclude: %v", err)
	}
	if len(report.Scores) != 1 || report.Scores[0].Function != "K" {
		t.Fatalf("expected only K, got %+v", report.Scores)
	}
}

func TestAnalyze_VendorSkipped(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"),
		[]byte("package x\nfunc M() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	vendor := filepath.Join(dir, "vendor", "lib")
	if err := os.MkdirAll(vendor, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendor, "lib.go"),
		[]byte("package lib\nfunc L() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Analyze([]string{dir}, Options{})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(report.Scores) != 1 || report.Scores[0].Function != "M" {
		t.Fatalf("expected only M (vendor skipped), got %+v", report.Scores)
	}
}

func TestAnalyze_RaceSafe(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 8; i++ {
		path := filepath.Join(dir, "f"+string(rune('a'+i))+".go")
		body := "package x\nfunc F" + string(rune('A'+i)) + "() int { if true { return 1 }; return 0 }\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	report, err := Analyze([]string{dir}, Options{Concurrency: 4})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(report.Scores) != 8 {
		t.Fatalf("expected 8 scored funcs, got %d", len(report.Scores))
	}
}

func TestAnalyze_MissingPath(t *testing.T) {
	_, err := Analyze([]string{filepath.Join(t.TempDir(), "does-not-exist")}, Options{})
	if err == nil {
		t.Fatalf("Analyze(missing) error = nil, want stat error")
	}
}
