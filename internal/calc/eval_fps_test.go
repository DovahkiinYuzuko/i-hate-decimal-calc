package calc

import (
	"math/big"
	"testing"
)

func TestDenseFPS_BasicArithmetic(t *testing.T) {
	// A(x) = 1 + 2x + 3x^2 (prec=4)
	a := &DenseFPS{
		Var: "x",
		Coeffs: []*big.Rat{
			big.NewRat(1, 1),
			big.NewRat(2, 1),
			big.NewRat(3, 1),
			big.NewRat(0, 1),
		},
		Precision: 4,
	}

	// B(x) = 2 - x (prec=4)
	b := &DenseFPS{
		Var: "x",
		Coeffs: []*big.Rat{
			big.NewRat(2, 1),
			big.NewRat(-1, 1),
			big.NewRat(0, 1),
			big.NewRat(0, 1),
		},
		Precision: 4,
	}

	// Add: 3 + x + 3x^2
	sum := a.Add(b)
	expectedSum := []*big.Rat{big.NewRat(3, 1), big.NewRat(1, 1), big.NewRat(3, 1), big.NewRat(0, 1)}
	for i := 0; i < 4; i++ {
		if sum.Coeffs[i].Cmp(expectedSum[i]) != 0 {
			t.Errorf("Add coeff[%d] = %s, want %s", i, sum.Coeffs[i].String(), expectedSum[i].String())
		}
	}

	// MulTrunc (mod x^4):
	// (1 + 2x + 3x^2) * (2 - x) = 2 + 3x + 4x^2 - 3x^3
	prod := a.MulTrunc(b, 4)
	expectedProd := []*big.Rat{big.NewRat(2, 1), big.NewRat(3, 1), big.NewRat(4, 1), big.NewRat(-3, 1)}
	for i := 0; i < 4; i++ {
		if prod.Coeffs[i].Cmp(expectedProd[i]) != 0 {
			t.Errorf("MulTrunc coeff[%d] = %s, want %s", i, prod.Coeffs[i].String(), expectedProd[i].String())
		}
	}
}

func TestDenseFPS_Inv_Geometric(t *testing.T) {
	// A(x) = 1 - x (prec=5)
	// 1 / (1 - x) = 1 + x + x^2 + x^3 + x^4
	a := &DenseFPS{
		Var: "x",
		Coeffs: []*big.Rat{
			big.NewRat(1, 1),
			big.NewRat(-1, 1),
			big.NewRat(0, 1),
			big.NewRat(0, 1),
			big.NewRat(0, 1),
		},
		Precision: 5,
	}

	invODE, err := a.InvRecurrence(5)
	if err != nil {
		t.Fatalf("InvRecurrence failed: %v", err)
	}
	for i := 0; i < 5; i++ {
		if invODE.Coeffs[i].Cmp(big.NewRat(1, 1)) != 0 {
			t.Errorf("invODE[%d] = %s, want 1", i, invODE.Coeffs[i].String())
		}
	}

	invNewton, err := a.InvNewton(5)
	if err != nil {
		t.Fatalf("InvNewton failed: %v", err)
	}
	for i := 0; i < 5; i++ {
		if invNewton.Coeffs[i].Cmp(big.NewRat(1, 1)) != 0 {
			t.Errorf("invNewton[%d] = %s, want 1", i, invNewton.Coeffs[i].String())
		}
	}
}

func TestDenseFPS_Log_Exp_Roundtrip(t *testing.T) {
	// A(x) = 1 + x (prec=5)
	// ln(1 + x) = x - 1/2*x^2 + 1/3*x^3 - 1/4*x^4
	a := &DenseFPS{
		Var: "x",
		Coeffs: []*big.Rat{
			big.NewRat(1, 1),
			big.NewRat(1, 1),
			big.NewRat(0, 1),
			big.NewRat(0, 1),
			big.NewRat(0, 1),
		},
		Precision: 5,
	}

	logA, err := a.Log(5)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	expectedLog := []*big.Rat{
		big.NewRat(0, 1),
		big.NewRat(1, 1),
		big.NewRat(-1, 2),
		big.NewRat(1, 3),
		big.NewRat(-1, 4),
	}
	for i := 0; i < 5; i++ {
		if logA.Coeffs[i].Cmp(expectedLog[i]) != 0 {
			t.Errorf("logA[%d] = %s, want %s", i, logA.Coeffs[i].String(), expectedLog[i].String())
		}
	}

	// exp(log(1 + x)) should recover 1 + x mod x^5
	expLogA, err := logA.Exp(5)
	if err != nil {
		t.Fatalf("Exp failed: %v", err)
	}
	for i := 0; i < 5; i++ {
		if expLogA.Coeffs[i].Cmp(a.Coeffs[i]) != 0 {
			t.Errorf("exp(log(1+x))[%d] = %s, want %s", i, expLogA.Coeffs[i].String(), a.Coeffs[i].String())
		}
	}
}

func TestDenseFPS_Sqrt_Roundtrip(t *testing.T) {
	// A(x) = 1 - 4x (prec=5)
	// sqrt(1 - 4x) = 1 - 2x - 2x^2 - 4x^3 - 14x^4
	a := &DenseFPS{
		Var: "x",
		Coeffs: []*big.Rat{
			big.NewRat(1, 1),
			big.NewRat(-4, 1),
			big.NewRat(0, 1),
			big.NewRat(0, 1),
			big.NewRat(0, 1),
		},
		Precision: 5,
	}

	sqrtA, err := a.Sqrt(5)
	if err != nil {
		t.Fatalf("Sqrt failed: %v", err)
	}

	expectedSqrt := []*big.Rat{
		big.NewRat(1, 1),
		big.NewRat(-2, 1),
		big.NewRat(-2, 1),
		big.NewRat(-4, 1),
		big.NewRat(-10, 1),
	}
	for i := 0; i < 5; i++ {
		if sqrtA.Coeffs[i].Cmp(expectedSqrt[i]) != 0 {
			t.Errorf("sqrtA[%d] = %s, want %s", i, sqrtA.Coeffs[i].String(), expectedSqrt[i].String())
		}
	}

	// (sqrt(A))^2 mod x^5 == A
	sq := sqrtA.MulTrunc(sqrtA, 5)
	for i := 0; i < 5; i++ {
		if sq.Coeffs[i].Cmp(a.Coeffs[i]) != 0 {
			t.Errorf("sqrt(A)^2[%d] = %s, want %s", i, sq.Coeffs[i].String(), a.Coeffs[i].String())
		}
	}
}

func TestDenseFPS_Preconditions(t *testing.T) {
	// 1. Inv with A(0) == 0 should fail
	zeroConst := &DenseFPS{
		Var:       "x",
		Coeffs:    []*big.Rat{big.NewRat(0, 1), big.NewRat(1, 1)},
		Precision: 2,
	}
	if _, err := zeroConst.Inv(2); err == nil {
		t.Errorf("expected error for Inv with A(0)=0, got nil")
	}

	// 2. Log with A(0) != 1 should fail
	nonOneConst := &DenseFPS{
		Var:       "x",
		Coeffs:    []*big.Rat{big.NewRat(2, 1), big.NewRat(1, 1)},
		Precision: 2,
	}
	if _, err := nonOneConst.Log(2); err == nil {
		t.Errorf("expected error for Log with A(0)!=1, got nil")
	}

	// 3. Exp with A(0) != 0 should fail
	nonZeroConst := &DenseFPS{
		Var:       "x",
		Coeffs:    []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 1)},
		Precision: 2,
	}
	if _, err := nonZeroConst.Exp(2); err == nil {
		t.Errorf("expected error for Exp with A(0)!=0, got nil")
	}

	// 4. Sqrt with non-square A(0) should fail
	nonSquareConst := &DenseFPS{
		Var:       "x",
		Coeffs:    []*big.Rat{big.NewRat(2, 1), big.NewRat(1, 1)},
		Precision: 2,
	}
	if _, err := nonSquareConst.Sqrt(2); err == nil {
		t.Errorf("expected error for Sqrt with A(0)=2 (non-square), got nil")
	}
}

func TestCombinatorics(t *testing.T) {
	// partitions(5) = 7, partitions(10) = 42
	p5, err := EvalPartitions(5)
	if err != nil || p5.Cmp(big.NewInt(7)) != 0 {
		t.Errorf("partitions(5) = %v, want 7, err=%v", p5, err)
	}
	p10, err := EvalPartitions(10)
	if err != nil || p10.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("partitions(10) = %v, want 42, err=%v", p10, err)
	}

	// bell(5) = 52, bell(10) = 115975
	b5, err := EvalBell(5)
	if err != nil || b5.Cmp(big.NewInt(52)) != 0 {
		t.Errorf("bell(5) = %v, want 52, err=%v", b5, err)
	}
	b10, err := EvalBell(10)
	if err != nil || b10.Cmp(big.NewInt(115975)) != 0 {
		t.Errorf("bell(10) = %v, want 115975, err=%v", b10, err)
	}

	// catalan(4) = 14, catalan(5) = 42
	c4, err := EvalCatalan(4)
	if err != nil || c4.Cmp(big.NewInt(14)) != 0 {
		t.Errorf("catalan(4) = %v, want 14, err=%v", c4, err)
	}
	c5, err := EvalCatalan(5)
	if err != nil || c5.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("catalan(5) = %v, want 42, err=%v", c5, err)
	}

	// stirling1(4, 2) = 11
	s1, err := EvalStirling1(4, 2)
	if err != nil || s1.Cmp(big.NewInt(11)) != 0 {
		t.Errorf("stirling1(4, 2) = %v, want 11, err=%v", s1, err)
	}

	// stirling2(4, 2) = 7
	s2, err := EvalStirling2(4, 2)
	if err != nil || s2.Cmp(big.NewInt(7)) != 0 {
		t.Errorf("stirling2(4, 2) = %v, want 7, err=%v", s2, err)
	}
}

func TestDenseFPS_MinPrecisionRule(t *testing.T) {
	// FLINT / SageMath standard: prec(f + g) = min(prec(f), prec(g))
	f := NewDenseFPS("x", 5)
	f.Coeffs[0].SetInt64(1)
	f.Coeffs[1].SetInt64(2)

	g := NewDenseFPS("x", 3)
	g.Coeffs[0].SetInt64(3)
	g.Coeffs[1].SetInt64(4)

	sum := f.Add(g)
	if sum.Precision != 3 {
		t.Errorf("expected sum precision to be min(5, 3)=3, got %d", sum.Precision)
	}

	diff := f.Sub(g)
	if diff.Precision != 3 {
		t.Errorf("expected diff precision to be min(5, 3)=3, got %d", diff.Precision)
	}
}
