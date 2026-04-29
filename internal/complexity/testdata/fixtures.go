// Package fixtures contains hand-crafted Go functions with known
// cyclomatic and cognitive complexity scores. Used by analyzer/visitor tests.
//
// This file lives under testdata/ so the toolchain ignores it for normal
// `go build` / `go test` runs. The complexity analyzer parses it directly
// from disk via parser.ParseFile.
package fixtures

// Linear is the no-branch baseline.
// cyclomatic = 1, cognitive = 0
func Linear() int {
	return 1 + 2
}

// IfOnly is a single-branch if.
// cyclomatic = 2, cognitive = 1
func IfOnly(x int) int {
	if x > 0 {
		return 1
	}
	return 0
}

// IfElse adds a final else.
// cyclomatic = 2, cognitive = 2
func IfElse(x int) int {
	if x > 0 {
		return 1
	} else {
		return 0
	}
}

// IfElseIf chains conditions.
// cyclomatic = 3, cognitive = 2
func IfElseIf(x int) int {
	if x > 0 {
		return 1
	} else if x < 0 {
		return -1
	}
	return 0
}

// AndOr is a single boolean chain (one operator transition).
// cyclomatic = 2, cognitive = 1
func AndOr(x, y int) bool {
	return x > 0 && y > 0
}

// MixedBool alternates && and || in one expression.
// cyclomatic = 3, cognitive = 2
func MixedBool(x, y, z int) bool {
	return x > 0 && y > 0 || z > 0
}

// RangeIf nests an if inside a range loop.
// cyclomatic = 3, cognitive = 3
func RangeIf(items []int) int {
	sum := 0
	for _, x := range items {
		if x > 0 {
			sum += x
		}
	}
	return sum
}

// DeeplyNested has two-deep loop nesting plus an if at depth 2.
// cyclomatic = 4, cognitive = 6
func DeeplyNested(matrix [][]int) int {
	total := 0
	for _, row := range matrix {
		for _, val := range row {
			if val > 0 {
				total += val
			}
		}
	}
	return total
}

// SwitchThree has three non-default cases.
// cyclomatic = 4, cognitive = 1
func SwitchThree(x int) string {
	switch x {
	case 1:
		return "one"
	case 2:
		return "two"
	case 3:
		return "three"
	default:
		return "other"
	}
}

// TypeSwitchTwo has two type cases.
// cyclomatic = 3, cognitive = 1
func TypeSwitchTwo(x interface{}) string {
	switch v := x.(type) {
	case int:
		_ = v
		return "int"
	case string:
		return "string"
	}
	return "other"
}

// Recursive calls itself directly. Cognitive picks up +1 for the recursion.
// cyclomatic = 2, cognitive = 2
func Recursive(n int) int {
	if n <= 1 {
		return 1
	}
	return n * Recursive(n-1)
}

// WithClosure has an if inside a function literal that participates in the
// outer total.
// cyclomatic = 2, cognitive = 1
func WithClosure() int {
	f := func(x int) int {
		if x > 0 {
			return x
		}
		return 0
	}
	return f(5)
}
