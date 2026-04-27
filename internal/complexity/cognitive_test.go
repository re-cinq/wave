package complexity

import "testing"

func TestCognitiveComplexity(t *testing.T) {
	file := parseFixtures(t)
	cases := []struct {
		fn   string
		want int
	}{
		{"Linear", 0},
		{"IfOnly", 1},
		{"IfElse", 2},
		{"IfElseIf", 2},
		{"AndOr", 1},
		{"MixedBool", 2},
		{"RangeIf", 3},
		{"DeeplyNested", 6},
		{"SwitchThree", 1},
		{"TypeSwitchTwo", 1},
		{"Recursive", 2},
		{"WithClosure", 1},
	}
	for _, tc := range cases {
		t.Run(tc.fn, func(t *testing.T) {
			fn := funcByName(t, file, tc.fn)
			got := CognitiveComplexity(fn)
			if got != tc.want {
				t.Fatalf("CognitiveComplexity(%s) = %d, want %d", tc.fn, got, tc.want)
			}
		})
	}
}

func TestCognitiveComplexity_NilFunc(t *testing.T) {
	if got := CognitiveComplexity(nil); got != 0 {
		t.Fatalf("CognitiveComplexity(nil) = %d, want 0", got)
	}
}
