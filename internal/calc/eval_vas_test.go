package calc

import (
	"math/big"
	"testing"
)

func TestCountSignVariations(t *testing.T) {
	tests := []struct {
		coeffs   []int64
		expected int
	}{
		{[]int64{1, -1, 1}, 2},
		{[]int64{1, 0, -1}, 1},
		{[]int64{1, 2, 3}, 0},
		{[]int64{-1, -2, -3}, 0},
		{[]int64{1, -2, 0, 3, -4}, 3},
	}

	for _, tc := range tests {
		bigCoeffs := make([]*big.Int, len(tc.coeffs))
		for i, c := range tc.coeffs {
			bigCoeffs[i] = big.NewInt(c)
		}
		got := CountSignVariations(bigCoeffs)
		if got != tc.expected {
			t.Errorf("coeffs %v: expected %d, got %d", tc.coeffs, tc.expected, got)
		}
	}
}

func TestIsolateRealRootsVAS_Basic(t *testing.T) {
	// 1. x^2 - 2: coeffs [-2, 0, 1]
	polySqrt2 := []*big.Int{big.NewInt(-2), big.NewInt(0), big.NewInt(1)}
	rootsSqrt2 := IsolateRealRootsVAS(polySqrt2)
	if len(rootsSqrt2) != 2 {
		t.Fatalf("expected 2 roots for x^2 - 2, got %d", len(rootsSqrt2))
	}
	// Check roots are negative and positive
	if rootsSqrt2[0].High.Sign() >= 0 || rootsSqrt2[1].Low.Sign() <= 0 {
		t.Errorf("expected one negative and one positive interval, got [%v, %v] and [%v, %v]",
			rootsSqrt2[0].Low, rootsSqrt2[0].High, rootsSqrt2[1].Low, rootsSqrt2[1].High)
	}

	// 2. x^3 - 2: coeffs [-2, 0, 0, 1]
	polyCbrt2 := []*big.Int{big.NewInt(-2), big.NewInt(0), big.NewInt(0), big.NewInt(1)}
	rootsCbrt2 := IsolateRealRootsVAS(polyCbrt2)
	if len(rootsCbrt2) != 1 {
		t.Fatalf("expected 1 root for x^3 - 2, got %d", len(rootsCbrt2))
	}
	if rootsCbrt2[0].Low.Cmp(big.NewRat(1, 1)) < 0 || rootsCbrt2[0].High.Cmp(big.NewRat(2, 1)) > 0 {
		t.Errorf("expected root of x^3 - 2 in [1, 2], got [%v, %v]", rootsCbrt2[0].Low, rootsCbrt2[0].High)
	}

	// 3. x^5 - x - 1: coeffs [-1, -1, 0, 0, 0, 1]
	polyDeg5 := []*big.Int{big.NewInt(-1), big.NewInt(-1), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(1)}
	rootsDeg5 := IsolateRealRootsVAS(polyDeg5)
	if len(rootsDeg5) != 1 {
		t.Fatalf("expected 1 real root for x^5 - x - 1, got %d", len(rootsDeg5))
	}
	if rootsDeg5[0].Low.Cmp(big.NewRat(1, 1)) < 0 || rootsDeg5[0].High.Cmp(big.NewRat(2, 1)) > 0 {
		t.Errorf("expected root in [1, 2], got [%v, %v]", rootsDeg5[0].Low, rootsDeg5[0].High)
	}
}

func TestIsolateRealRootsVAS_ExactRationalRoots(t *testing.T) {
	// (x - 1)(x + 2)x = x^3 + x^2 - 2x -> coeffs [0, -2, 1, 1]
	poly := []*big.Int{big.NewInt(0), big.NewInt(-2), big.NewInt(1), big.NewInt(1)}
	roots := IsolateRealRootsVAS(poly)
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots, got %d", len(roots))
	}

	// Check roots contain -2, 0, 1
	hasNeg2 := false
	hasZero := false
	hasOne := false
	for _, r := range roots {
		if r.Low.Cmp(big.NewRat(-2, 1)) <= 0 && r.High.Cmp(big.NewRat(-2, 1)) >= 0 {
			hasNeg2 = true
		}
		if r.Low.Cmp(big.NewRat(0, 1)) <= 0 && r.High.Cmp(big.NewRat(0, 1)) >= 0 {
			hasZero = true
		}
		if r.Low.Cmp(big.NewRat(1, 1)) <= 0 && r.High.Cmp(big.NewRat(1, 1)) >= 0 {
			hasOne = true
		}
	}

	if !hasNeg2 || !hasZero || !hasOne {
		t.Errorf("expected roots -2, 0, 1, got %+v", roots)
	}
}

func TestIsolateRealRootsVAS_CompareWithSturm(t *testing.T) {
	testPolys := []struct {
		name        string
		poly        []*big.Int
		expectedNum int
	}{
		{
			name:        "x^4 - 4x^2 + 2",
			poly:        []*big.Int{big.NewInt(2), big.NewInt(0), big.NewInt(-4), big.NewInt(0), big.NewInt(1)},
			expectedNum: 4,
		},
		{
			name:        "x^3 - 3x + 1",
			poly:        []*big.Int{big.NewInt(1), big.NewInt(-3), big.NewInt(0), big.NewInt(1)},
			expectedNum: 3,
		},
		{
			name:        "x^5 - 5x + 1",
			poly:        []*big.Int{big.NewInt(1), big.NewInt(-5), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(1)},
			expectedNum: 3,
		},
	}

	for _, tc := range testPolys {
		t.Run(tc.name, func(t *testing.T) {
			roots := IsolateRealRootsVAS(tc.poly)
			if len(roots) != tc.expectedNum {
				t.Errorf("%s: expected %d roots, got %d", tc.name, tc.expectedNum, len(roots))
			}
			// Verify all intervals are disjoint and properly ordered
			for i := 0; i < len(roots)-1; i++ {
				if roots[i].High.Cmp(roots[i+1].Low) > 0 {
					t.Errorf("%s: overlapping intervals [%v, %v] and [%v, %v]",
						tc.name, roots[i].Low, roots[i].High, roots[i+1].Low, roots[i+1].High)
				}
			}
		})
	}
}

func TestIsolateRealRootsVAS_MultipleRoots(t *testing.T) {
	cases := []struct {
		name        string
		poly        []*big.Int
		expectedNum int
	}{
		{
			name:        "(x - 2)^2 = x^2 - 4x + 4",
			poly:        []*big.Int{big.NewInt(4), big.NewInt(-4), big.NewInt(1)},
			expectedNum: 1,
		},
		{
			name:        "(x - 1)^2 * (x + 1) = x^3 - x^2 - x + 1",
			poly:        []*big.Int{big.NewInt(1), big.NewInt(-1), big.NewInt(-1), big.NewInt(1)},
			expectedNum: 2,
		},
		{
			name:        "(x^2 - 2)^2 = x^4 - 4x^2 + 4",
			poly:        []*big.Int{big.NewInt(4), big.NewInt(0), big.NewInt(-4), big.NewInt(0), big.NewInt(1)},
			expectedNum: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roots := IsolateRealRootsVAS(tc.poly)
			if len(roots) != tc.expectedNum {
				t.Fatalf("%s: expected %d roots, got %d: %v", tc.name, tc.expectedNum, len(roots), roots)
			}
			for i := 0; i < len(roots)-1; i++ {
				if roots[i].High.Cmp(roots[i+1].Low) > 0 {
					t.Errorf("%s: overlapping intervals [%v, %v] and [%v, %v]",
						tc.name, roots[i].Low, roots[i].High, roots[i+1].Low, roots[i+1].High)
				}
			}
		})
	}
}
