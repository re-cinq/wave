package complexity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseFixtures(t *testing.T) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "testdata/fixtures.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse fixtures: %v", err)
	}
	return file
}

func funcByName(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == name {
			return fn
		}
	}
	t.Fatalf("function %q not found in fixtures", name)
	return nil
}

func TestCyclomaticComplexity(t *testing.T) {
	file := parseFixtures(t)
	cases := []struct {
		fn   string
		want int
	}{
		{"Linear", 1},
		{"IfOnly", 2},
		{"IfElse", 2},
		{"IfElseIf", 3},
		{"AndOr", 2},
		{"MixedBool", 3},
		{"RangeIf", 3},
		{"DeeplyNested", 4},
		{"SwitchThree", 4},
		{"TypeSwitchTwo", 3},
		{"Recursive", 2},
		{"WithClosure", 2},
	}
	for _, tc := range cases {
		t.Run(tc.fn, func(t *testing.T) {
			fn := funcByName(t, file, tc.fn)
			got := CyclomaticComplexity(fn.Body)
			if got != tc.want {
				t.Fatalf("CyclomaticComplexity(%s) = %d, want %d", tc.fn, got, tc.want)
			}
		})
	}
}

func TestCyclomaticComplexity_NilBody(t *testing.T) {
	if got := CyclomaticComplexity(nil); got != 1 {
		t.Fatalf("CyclomaticComplexity(nil) = %d, want 1", got)
	}
}
