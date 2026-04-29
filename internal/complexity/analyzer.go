package complexity

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Options controls Analyze behavior.
type Options struct {
	// MaxCyclomatic is the fail threshold for cyclomatic complexity.
	// Functions with score > MaxCyclomatic produce a high-severity finding.
	// Default 15.
	MaxCyclomatic int
	// MaxCognitive is the fail threshold for cognitive complexity. Default 15.
	MaxCognitive int
	// WarnCyclomatic is the warn threshold; functions in
	// (WarnCyclomatic, MaxCyclomatic] produce a medium-severity finding.
	// Default 10.
	WarnCyclomatic int
	// WarnCognitive analogue. Default 10.
	WarnCognitive int
	// IncludeTests, when true, also scores `_test.go` files. Default false.
	IncludeTests bool
	// Excludes is a list of substrings matched against file paths; matching
	// files are skipped. Common defaults always apply (vendor/, .git/).
	Excludes []string
	// Concurrency caps per-file workers. Defaults to runtime.NumCPU().
	Concurrency int
}

// FunctionScore is a per-function complexity result.
type FunctionScore struct {
	File       string `json:"file"`
	Package    string `json:"package"`
	Function   string `json:"function"`
	Line       int    `json:"line"`
	Cyclomatic int    `json:"cyclomatic"`
	Cognitive  int    `json:"cognitive"`
}

// Report is the aggregated analyzer output.
type Report struct {
	Scores    []FunctionScore `json:"scores"`
	FileCount int             `json:"file_count"`
	ScannedAt time.Time       `json:"scanned_at"`
}

// withDefaults returns a copy of opts with zero fields populated.
func (o Options) withDefaults() Options {
	if o.MaxCyclomatic <= 0 {
		o.MaxCyclomatic = 15
	}
	if o.MaxCognitive <= 0 {
		o.MaxCognitive = 15
	}
	if o.WarnCyclomatic <= 0 {
		o.WarnCyclomatic = 10
	}
	if o.WarnCognitive <= 0 {
		o.WarnCognitive = 10
	}
	if o.Concurrency <= 0 {
		o.Concurrency = runtime.NumCPU()
	}
	return o
}

// Analyze scans the given paths for Go source files and returns a Report
// containing per-function cyclomatic and cognitive complexity scores.
//
// Each path may be a file or a directory; directories are walked recursively.
// `vendor/`, `.git/`, and `node_modules/` are always skipped. `_test.go` files
// are skipped unless opts.IncludeTests is true.
//
// Per-file parsing and scoring runs in parallel up to opts.Concurrency.
// A parse error in any one file fails the whole call (the error wraps the
// file path). Empty path lists produce an empty report, not an error.
func Analyze(paths []string, opts Options) (Report, error) {
	opts = opts.withDefaults()
	report := Report{ScannedAt: time.Now().UTC()}
	if len(paths) == 0 {
		return report, nil
	}
	files, err := discoverFiles(paths, opts)
	if err != nil {
		return report, err
	}
	report.FileCount = len(files)
	if len(files) == 0 {
		return report, nil
	}

	var (
		mu      sync.Mutex
		results []FunctionScore
	)
	g := new(errgroup.Group)
	g.SetLimit(opts.Concurrency)
	for _, file := range files {
		file := file
		g.Go(func() error {
			scores, err := scoreFile(file)
			if err != nil {
				return fmt.Errorf("%s: %w", file, err)
			}
			mu.Lock()
			results = append(results, scores...)
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return report, err
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].File != results[j].File {
			return results[i].File < results[j].File
		}
		return results[i].Line < results[j].Line
	})
	report.Scores = results
	return report, nil
}

func discoverFiles(paths []string, opts Options) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, root := range paths {
		if err := walkRoot(root, opts, add); err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

func walkRoot(root string, opts Options, add func(string)) error {
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("stat %s: %w", root, err)
	}
	if !info.IsDir() {
		if shouldInclude(root, opts) {
			add(root)
		}
		return nil
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if isSkippedDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if shouldInclude(path, opts) {
			add(path)
		}
		return nil
	})
}

func isSkippedDir(name string) bool {
	switch name {
	case "vendor", ".git", "node_modules", "testdata":
		return true
	}
	return false
}

func shouldInclude(path string, opts Options) bool {
	if !strings.HasSuffix(path, ".go") {
		return false
	}
	if !opts.IncludeTests && strings.HasSuffix(path, "_test.go") {
		return false
	}
	for _, ex := range opts.Excludes {
		if ex == "" {
			continue
		}
		if strings.Contains(path, ex) {
			return false
		}
	}
	return true
}

func scoreFile(path string) ([]FunctionScore, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	pkg := ""
	if file.Name != nil {
		pkg = file.Name.Name
	}
	var scores []FunctionScore
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		pos := fset.Position(fn.Pos())
		scores = append(scores, FunctionScore{
			File:       path,
			Package:    pkg,
			Function:   funcName(fn),
			Line:       pos.Line,
			Cyclomatic: CyclomaticComplexity(fn.Body),
			Cognitive:  CognitiveComplexity(fn),
		})
	}
	return scores, nil
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		recv := exprString(fn.Recv.List[0].Type)
		if recv != "" {
			return fmt.Sprintf("(%s).%s", recv, fn.Name.Name)
		}
	}
	return fn.Name.Name
}

func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	case *ast.IndexExpr:
		return exprString(v.X)
	case *ast.IndexListExpr:
		return exprString(v.X)
	}
	return ""
}

// ErrNoPaths is returned when Analyze is invoked with no resolvable paths.
var ErrNoPaths = errors.New("complexity: no paths provided")
