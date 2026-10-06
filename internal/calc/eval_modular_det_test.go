package calc

import (
	"math/big"
	"testing"
)

func TestDetModularCRT_Basic(t *testing.T) {
	// 1x1
	mat1 := [][]*big.Int{
		{big.NewInt(42)},
	}
	if got := DetModularCRT(mat1); got.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("1x1 got %s, want 42", got)
	}

	// 2x2
	mat2 := [][]*big.Int{
		{big.NewInt(1), big.NewInt(2)},
		{big.NewInt(3), big.NewInt(4)},
	}
	if got := DetModularCRT(mat2); got.Cmp(big.NewInt(-2)) != 0 {
		t.Errorf("2x2 got %s, want -2", got)
	}

	// 3x3 identity
	mat3Id := [][]*big.Int{
		{big.NewInt(1), big.NewInt(0), big.NewInt(0)},
		{big.NewInt(0), big.NewInt(1), big.NewInt(0)},
		{big.NewInt(0), big.NewInt(0), big.NewInt(1)},
	}
	if got := DetModularCRT(mat3Id); got.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("3x3 identity got %s, want 1", got)
	}

	// 3x3 general: det = 1*(5*9 - 6*8) - 2*(4*9 - 6*7) + 3*(4*8 - 5*7) = 1*(-3) - 2*(-6) + 3*(-3) = -3 + 12 - 9 = 0
	mat3Singular := [][]*big.Int{
		{big.NewInt(1), big.NewInt(2), big.NewInt(3)},
		{big.NewInt(4), big.NewInt(5), big.NewInt(6)},
		{big.NewInt(7), big.NewInt(8), big.NewInt(9)},
	}
	if got := DetModularCRT(mat3Singular); got.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("3x3 singular got %s, want 0", got)
	}
}

func TestDetModularCRT_LargeCoeffs(t *testing.T) {
	// 3x3 with large integers
	a, _ := new(big.Int).SetString("1000000000000000000", 10) // 10^18
	b, _ := new(big.Int).SetString("2000000000000000000", 10)
	c, _ := new(big.Int).SetString("3000000000000000000", 10)

	mat := [][]*big.Int{
		{a, big.NewInt(1), big.NewInt(0)},
		{big.NewInt(0), b, big.NewInt(2)},
		{big.NewInt(3), big.NewInt(0), c},
	}
	// det = a*(b*c - 0) - 1*(0 - 6) = a*b*c + 6
	expected := new(big.Int).Mul(a, b)
	expected.Mul(expected, c)
	expected.Add(expected, big.NewInt(6))

	got := DetModularCRT(mat)
	if got.Cmp(expected) != 0 {
		t.Fatalf("LargeCoeffs mismatch:\ngot  %s\nwant %s", got, expected)
	}
}

func TestDet_IntegrationAndResultant(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"det([[1, 2], [3, 4]])", "-2"},
		{"det([[1, 2, 3], [4, 5, 6], [7, 8, 9]])", "0"},
		{"det([[2, 0, 1], [3, 1, 2], [1, 2, 3]])", "3"},
		{"resultant(x^2 - 2, x - 1, x)", "-1"},
		{"resultant(x^2 + 1, x^2 - 1, x)", "4"},
	}

	for _, tc := range tests {
		res, err := EvalString(tc.expr)
		if err != nil {
			t.Fatalf("EvalString(%q) failed: %v", tc.expr, err)
		}
		if res.String() != tc.expected {
			t.Errorf("EvalString(%q) = %s; want %s", tc.expr, res.String(), tc.expected)
		}
	}
}

