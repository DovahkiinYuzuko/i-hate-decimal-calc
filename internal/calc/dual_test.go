package calc

import (
	"math/big"
	"testing"
)

func TestDualArithmetic(t *testing.T) {
	// f(x) = x^3 + 2*x + 5 at x = 2
	// f(2) = 8 + 4 + 5 = 17
	// f'(x) = 3*x^2 + 2 => f'(2) = 12 + 2 = 14
	x := NewDualVar(big.NewRat(2, 1))
	x3, err := DualPowInt(x, 3)
	if err != nil {
		t.Fatalf("DualPowInt failed: %v", err)
	}

	two := NewDualConst(big.NewRat(2, 1))
	twoX := DualMul(two, x)

	five := NewDualConst(big.NewRat(5, 1))

	res := DualAdd(DualAdd(x3, twoX), five)

	expectedReal := big.NewRat(17, 1)
	expectedDual := big.NewRat(14, 1)

	if res.Real.Cmp(expectedReal) != 0 {
		t.Errorf("expected Real = %s, got %s", expectedReal.String(), res.Real.String())
	}
	if res.Dual.Cmp(expectedDual) != 0 {
		t.Errorf("expected Dual = %s, got %s", expectedDual.String(), res.Dual.String())
	}
}

func TestDualDivision(t *testing.T) {
	// f(x) = (x + 1) / (x - 1) at x = 3
	// f(3) = 4 / 2 = 2
	// f'(x) = ((1)*(x-1) - (x+1)*(1)) / (x-1)^2 = -2 / (x-1)^2 => f'(3) = -2 / 4 = -1/2
	x := NewDualVar(big.NewRat(3, 1))
	num := DualAdd(x, NewDualConst(big.NewRat(1, 1)))
	den := DualSub(x, NewDualConst(big.NewRat(1, 1)))

	res, err := DualQuo(num, den)
	if err != nil {
		t.Fatalf("DualQuo failed: %v", err)
	}

	expectedReal := big.NewRat(2, 1)
	expectedDual := big.NewRat(-1, 2)

	if res.Real.Cmp(expectedReal) != 0 {
		t.Errorf("expected Real = %s, got %s", expectedReal.String(), res.Real.String())
	}
	if res.Dual.Cmp(expectedDual) != 0 {
		t.Errorf("expected Dual = %s, got %s", expectedDual.String(), res.Dual.String())
	}
}

func TestEvalNodeDual(t *testing.T) {
	// AST of f(x) = 3*x^2 + 5*x - 7 at x = 4
	// f(4) = 3*16 + 20 - 7 = 48 + 13 = 61
	// f'(x) = 6*x + 5 => f'(4) = 24 + 5 = 29
	x2, err := NewPow(NewVar("x"), mustRational(2, 1))
	if err != nil {
		t.Fatalf("NewPow failed: %v", err)
	}
	expr := NewAdd([]Node{
		NewMul([]Node{mustRational(3, 1), x2}),
		NewMul([]Node{mustRational(5, 1), NewVar("x")}),
		mustRational(-7, 1),
	})

	point := big.NewRat(4, 1)
	res, err := EvalNodeDual(expr, "x", point)
	if err != nil {
		t.Fatalf("EvalNodeDual failed: %v", err)
	}

	expectedReal := big.NewRat(61, 1)
	expectedDual := big.NewRat(29, 1)

	if res.Real.Cmp(expectedReal) != 0 {
		t.Errorf("expected Real = %s, got %s", expectedReal.String(), res.Real.String())
	}
	if res.Dual.Cmp(expectedDual) != 0 {
		t.Errorf("expected Dual = %s, got %s", expectedDual.String(), res.Dual.String())
	}
}
