package calc

import (
	"math/big"
	"testing"
)

func newTestAlgReal(coeffs []int64, lowNum, lowDen, highNum, highDen int64) *AlgebraicReal {
	poly := make([]*big.Int, len(coeffs))
	for i, c := range coeffs {
		poly[i] = big.NewInt(c)
	}
	return &AlgebraicReal{
		Poly: poly,
		Low:  big.NewRat(lowNum, lowDen),
		High: big.NewRat(highNum, highDen),
	}
}

func TestAlgReal_BasicArithmetic(t *testing.T) {
	// sqrt(2): x^2 - 2 in [1, 2]
	sqrt2 := newTestAlgReal([]int64{-2, 0, 1}, 1, 1, 2, 1)
	// sqrt(3): x^2 - 3 in [1, 2]
	sqrt3 := newTestAlgReal([]int64{-3, 0, 1}, 1, 1, 2, 1)

	// 1. Comparison
	cmp := AlgCompare(sqrt2, sqrt3)
	if cmp >= 0 {
		t.Errorf("expected sqrt(2) < sqrt(3), got cmp = %d", cmp)
	}

	cmpSelf := AlgCompare(sqrt2, sqrt2)
	if cmpSelf != 0 {
		t.Errorf("expected sqrt(2) == sqrt(2), got cmp = %d", cmpSelf)
	}

	// 2. Subtraction: sqrt(2) - sqrt(2) == 0 (Algebraic Zero Test)
	diff := AlgSub(sqrt2, sqrt2)
	if !AlgIsZero(diff) {
		t.Errorf("expected sqrt(2) - sqrt(2) == 0, got %v", diff)
	}

	// 3. Addition: sqrt(2) + sqrt(3) -> min poly x^4 - 10x^2 + 1 = 0
	sum := AlgAdd(sqrt2, sqrt3)
	if sum == nil {
		t.Fatalf("expected non-nil sum for sqrt(2) + sqrt(3)")
	}
	// Check root interval contains ~3.14626
	valLow2 := big.NewRat(2, 1)
	val4 := big.NewRat(4, 1)
	if sum.Low.Cmp(valLow2) < 0 || sum.High.Cmp(val4) > 0 {
		t.Errorf("expected sum in [2, 4], got [%v, %v]", sum.Low, sum.High)
	}
	target := big.NewRat(3146, 1000)
	if sum.Low.Cmp(target) > 0 || sum.High.Cmp(target) < 0 {
		t.Errorf("expected sum interval [%v, %v] to contain %v", sum.Low, sum.High, target)
	}

	// 4. Multiplication: sqrt(2) * sqrt(3) = sqrt(6) -> x^2 - 6 in [2, 3]
	prod := AlgMul(sqrt2, sqrt3)
	if prod == nil {
		t.Fatalf("expected non-nil prod for sqrt(2) * sqrt(3)")
	}
	val2 := big.NewRat(2, 1)
	val3 := big.NewRat(3, 1)
	if prod.Low.Cmp(val2) < 0 || prod.High.Cmp(val3) > 0 {
		t.Errorf("expected prod in [2, 3], got [%v, %v]", prod.Low, prod.High)
	}
}

func TestAlgReal_Division(t *testing.T) {
	sqrt2 := newTestAlgReal([]int64{-2, 0, 1}, 1, 1, 2, 1)
	sqrt6 := newTestAlgReal([]int64{-6, 0, 1}, 2, 1, 3, 1)

	// sqrt(6) / sqrt(2) -> should be in [1, 2] (~1.732)
	quot, err := AlgDiv(sqrt6, sqrt2)
	if err != nil {
		t.Fatalf("unexpected error in AlgDiv: %v", err)
	}
	if quot == nil {
		t.Fatalf("expected non-nil quot")
	}

	val1 := big.NewRat(1, 1)
	val2 := big.NewRat(2, 1)
	if quot.Low.Cmp(val1) < 0 || quot.High.Cmp(val2) > 0 {
		t.Errorf("expected quot in [1, 2], got [%v, %v]", quot.Low, quot.High)
	}

	// Division by zero
	zero := NewAlgebraicRealFromRat(big.NewRat(0, 1))
	_, errDivZero := AlgDiv(sqrt6, zero)
	if errDivZero == nil {
		t.Errorf("expected error dividing by zero")
	}
}

func TestAlgReal_QuinticRoot(t *testing.T) {
	// x^5 - x - 1 in [1, 2] (~1.1673)
	alpha := newTestAlgReal([]int64{-1, -1, 0, 0, 0, 1}, 1, 1, 2, 1)

	// alpha - alpha == 0
	diff := AlgSub(alpha, alpha)
	if !AlgIsZero(diff) {
		t.Errorf("expected alpha - alpha == 0, got %v", diff)
	}

	// alpha * alpha -> root in [1, 2] (~1.362)
	sq := AlgMul(alpha, alpha)
	if sq == nil {
		t.Fatalf("expected non-nil sq")
	}
	val1 := big.NewRat(1, 1)
	val2 := big.NewRat(2, 1)
	if sq.Low.Cmp(val1) < 0 || sq.High.Cmp(val2) > 0 {
		t.Errorf("expected sq in [1, 2], got [%v, %v]", sq.Low, sq.High)
	}
}

func TestAlgReal_ZeroStraddlingDivision(t *testing.T) {
	// a = 2 (rational)
	a := NewAlgebraicRealFromRat(big.NewRat(2, 1))

	// b = sqrt(2), but with a coarse isolating interval [-1, 2] that straddles 0!
	b := newTestAlgReal([]int64{-2, 0, 1}, -1, 1, 2, 1)

	// a / b should refine b to isolate it from 0 and return sqrt(2)
	res, err := AlgDiv(a, b)
	if err != nil {
		t.Fatalf("expected AlgDiv to succeed after interval refinement, got error: %v", err)
	}
	if res.Low.Cmp(res.High) > 0 {
		t.Fatalf("invalid interval: Low (%v) > High (%v)", res.Low, res.High)
	}
	// res ~ 1.414... in [1, 2]
	one := big.NewRat(1, 1)
	two := big.NewRat(2, 1)
	if res.Low.Cmp(one) < 0 || res.High.Cmp(two) > 0 {
		t.Errorf("expected res in [1, 2], got [%v, %v]", res.Low, res.High)
	}
}
