package calc

import (
	"testing"
)

func TestFactorialLegendre_Basic(t *testing.T) {
	tests := []struct {
		n        int64
		expected string
	}{
		{0, "1"},
		{1, "1"},
		{2, "2"},
		{3, "6"},
		{4, "24"},
		{5, "120"},
		{6, "720"},
		{7, "5040"},
		{8, "40320"},
		{9, "362880"},
		{10, "3628800"},
		{12, "479001600"},
		{20, "2432902008176640000"},
	}

	for _, tc := range tests {
		got := FactorialLegendre(tc.n)
		if got.String() != tc.expected {
			t.Errorf("FactorialLegendre(%d) = %s; want %s", tc.n, got.String(), tc.expected)
		}
	}
}

func TestFactorialLegendre_MatchesProductTree(t *testing.T) {
	checkPoints := []int64{25, 50, 100, 200, 500, 1000, 5000}
	for _, n := range checkPoints {
		expected := ParallelProductTree(1, n, 64)
		got := FactorialLegendre(n)
		if got.Cmp(expected) != 0 {
			t.Fatalf("FactorialLegendre(%d) mismatch with ParallelProductTree", n)
		}
	}
}

func TestFactorialLegendre_LargeScale(t *testing.T) {
	// 10,000! has 35,660 decimal digits
	got := FactorialLegendre(10000)
	digits := ComputeExactDecimalDigits(got)
	if digits != 35660 {
		t.Errorf("digits of 10000! = %d; want 35660", digits)
	}
}
