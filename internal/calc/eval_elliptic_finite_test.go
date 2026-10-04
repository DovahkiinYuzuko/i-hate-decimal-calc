package calc

import (
	"math/big"
	"testing"
)

func TestFiniteCurvePointArithmetic(t *testing.T) {
	// E: y^2 = x^3 + x + 1 mod 5
	curve, err := NewFiniteCurve(big.NewInt(1), big.NewInt(1), big.NewInt(5))
	if err != nil {
		t.Fatalf("failed to create curve: %v", err)
	}

	p := NewFinitePoint(big.NewInt(0), big.NewInt(1), curve.P)
	if !curve.IsOnCurve(p) {
		t.Fatalf("point (0, 1) should be on curve")
	}

	// 2*P = (4, 2)
	twoP := curve.Add(p, p)
	expected2P := NewFinitePoint(big.NewInt(4), big.NewInt(2), curve.P)
	if !twoP.Equal(expected2P) {
		t.Errorf("expected 2P = (4, 2), got %s", twoP.String())
	}

	// 3*P = (0, 1) + (4, 2) = (2, 1)
	threeP := curve.Add(p, twoP)
	expected3P := NewFinitePoint(big.NewInt(2), big.NewInt(1), curve.P)
	if !threeP.Equal(expected3P) {
		t.Errorf("expected 3P = (2, 1), got %s", threeP.String())
	}

	// P + (-P) = O
	negP := curve.Neg(p)
	inf := curve.Add(p, negP)
	if !inf.IsInf {
		t.Errorf("expected P + (-P) = O, got %s", inf.String())
	}

	// Scalar multiplication: 2 * P
	scalar2P := curve.ScalarMul(big.NewInt(2), p)
	if !scalar2P.Equal(expected2P) {
		t.Errorf("expected ScalarMul(2, P) = (4, 2), got %s", scalar2P.String())
	}

	// Scalar multiplication: 3 * P
	scalar3P := curve.ScalarMul(big.NewInt(3), p)
	if !scalar3P.Equal(expected3P) {
		t.Errorf("expected ScalarMul(3, P) = (2, 1), got %s", scalar3P.String())
	}
}

func TestSchoofOrderDetermination(t *testing.T) {
	cases := []struct {
		a, b, p       int64
		expectedOrder int64
		expectedTrace int64
	}{
		{a: 1, b: 1, p: 5, expectedOrder: 9, expectedTrace: -3},
		{a: 2, b: 3, p: 7, expectedOrder: 6, expectedTrace: 2},
		{a: 1, b: 0, p: 13, expectedOrder: 20, expectedTrace: -6},
		{a: 0, b: 1, p: 13, expectedOrder: 12, expectedTrace: 2},
		{a: 1, b: 1, p: 17, expectedOrder: 18, expectedTrace: 0},
		{a: 1, b: 1, p: 19, expectedOrder: 21, expectedTrace: -1},
		{a: 1, b: 1, p: 97, expectedOrder: 97, expectedTrace: 1},
		{a: 2, b: 3, p: 1009, expectedOrder: 1068, expectedTrace: -58},
	}

	for _, tc := range cases {
		curve, err := NewFiniteCurve(big.NewInt(tc.a), big.NewInt(tc.b), big.NewInt(tc.p))
		if err != nil {
			t.Fatalf("failed to create curve [%d, %d] mod %d: %v", tc.a, tc.b, tc.p, err)
		}

		order, err := curve.EcPOrder()
		if err != nil {
			t.Fatalf("EcPOrder failed for [%d, %d] mod %d: %v", tc.a, tc.b, tc.p, err)
		}
		if order.Int64() != tc.expectedOrder {
			t.Errorf("curve [%d, %d] mod %d: expected order %d, got %s", tc.a, tc.b, tc.p, tc.expectedOrder, order.String())
		}

		trace, err := curve.EcPTrace()
		if err != nil {
			t.Fatalf("EcPTrace failed for [%d, %d] mod %d: %v", tc.a, tc.b, tc.p, err)
		}
		if trace.Int64() != tc.expectedTrace {
			t.Errorf("curve [%d, %d] mod %d: expected trace %d, got %s", tc.a, tc.b, tc.p, tc.expectedTrace, trace.String())
		}
	}
}

func TestEvalFiniteEllipticHandlers(t *testing.T) {
	env := NewEnv()

	// 1. ec_p_order([1, 1], 5) => 9
	resOrder, err := EvalStringWithEnv("ec_p_order([1, 1], 5)", env)
	if err != nil {
		t.Fatalf("ec_p_order failed: %v", err)
	}
	if resOrder.String() != "9" {
		t.Errorf("expected 9, got %s", resOrder.String())
	}

	// 2. ec_p_trace([1, 1], 5) => -3
	resTrace, err := EvalStringWithEnv("ec_p_trace([1, 1], 5)", env)
	if err != nil {
		t.Fatalf("ec_p_trace failed: %v", err)
	}
	if resTrace.String() != "-3" {
		t.Errorf("expected -3, got %s", resTrace.String())
	}

	// 3. ec_p_add([1, 1], [0, 1], [0, 4], 5) => [0, 0] (O)
	resAddInf, err := EvalStringWithEnv("ec_p_add([1, 1], [0, 1], [0, 4], 5)", env)
	if err != nil {
		t.Fatalf("ec_p_add inf failed: %v", err)
	}
	if resAddInf.String() != "[0, 0]" {
		t.Errorf("expected [0, 0], got %s", resAddInf.String())
	}

	// 4. ec_p_mul([1, 1], 2, [0, 1], 5) => [4, 2]
	resMul, err := EvalStringWithEnv("ec_p_mul([1, 1], 2, [0, 1], 5)", env)
	if err != nil {
		t.Fatalf("ec_p_mul failed: %v", err)
	}
	if resMul.String() != "[4, 2]" {
		t.Errorf("expected [4, 2], got %s", resMul.String())
	}
}

func TestSingularAndCompositeRejection(t *testing.T) {
	// Singular curve: [1, 3] mod 19 has 4(1) + 27(9) = 247 = 19 * 13 = 0 mod 19
	_, err := NewFiniteCurve(big.NewInt(1), big.NewInt(3), big.NewInt(19))
	if err == nil {
		t.Errorf("expected error for singular curve [1, 3] mod 19, got nil")
	}

	// Composite characteristic: p = 15
	_, err = NewFiniteCurve(big.NewInt(1), big.NewInt(1), big.NewInt(15))
	if err == nil {
		t.Errorf("expected error for composite characteristic 15, got nil")
	}
}

func TestFpPolyArithmetic(t *testing.T) {
	p := big.NewInt(7)
	// (2 + 3x) + (4 + 5x) = 6 + 8x = 6 + x mod 7
	poly1 := fpPoly{big.NewInt(2), big.NewInt(3)}
	poly2 := fpPoly{big.NewInt(4), big.NewInt(5)}
	sum := fpPolyAdd(p, poly1, poly2)

	if len(sum) != 2 || sum[0].Int64() != 6 || sum[1].Int64() != 1 {
		t.Errorf("expected 6 + x, got %v", sum)
	}
}


