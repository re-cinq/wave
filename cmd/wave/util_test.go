package main

import "testing"

func TestClamp(t *testing.T) {
	tests := []struct {
		name      string
		n, lo, hi int
		want      int
	}{
		{"within range", 5, 0, 10, 5},
		{"below lo", -3, 0, 10, 0},
		{"above hi", 99, 0, 10, 10},
		{"equal lo", 0, 0, 10, 0},
		{"equal hi", 10, 0, 10, 10},
		{"inverted range", 5, 10, 0, 5},
		{"inverted below", -1, 10, 0, 0},
		{"inverted above", 11, 10, 0, 10},
		{"negative range", -5, -10, -1, -5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Clamp(tt.n, tt.lo, tt.hi); got != tt.want {
				t.Fatalf("Clamp(%d,%d,%d) = %d, want %d", tt.n, tt.lo, tt.hi, got, tt.want)
			}
		})
	}
}
