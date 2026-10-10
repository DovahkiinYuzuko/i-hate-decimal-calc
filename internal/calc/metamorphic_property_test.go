package calc_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

// =========================================================================
// Metamorphic Property-Based Tests for CAS Core
// Reference: Chen et al. (1998), Segura et al. (2016), Donaldson et al. (2017)
// =========================================================================

// MR1: Polynomial Factorization & Expansion Round-Trip Inversion
// Relation: expand(factor(P)) - P == 0
func TestMetamorphic_MR1_PolynomialFactorExpand(t *testing.T) {
	env := calc.NewEnv()

	testPolys := []string{
		"x^2 - 1",
		"x^2 - 4*x + 4",
		"x^3 - 1",
		"x^3 + 6*x^2 + 11*x + 6",
		"x^4 - 16",
		"x^4 - 5*x^2 + 4",
		"2*x^3 - 3*x^2 - 11*x + 6",
		"x^4 - 2*x^3 - 7*x^2 + 8*x + 12",
		"x^5 - 1",
		"x^3 - 3*x + 2",
	}

	for _, polyStr := range testPolys {
		polyNode, err := calc.Parse(polyStr)
		if err != nil {
			t.Fatalf("[%s] parse failed: %v", polyStr, err)
		}
		evalPoly, err := calc.EvalWithEnv(polyNode, env)
		if err != nil {
			t.Fatalf("[%s] eval poly failed: %v", polyStr, err)
		}

		factorExpr, err := calc.Parse(fmt.Sprintf("factor(%s)", polyStr))
		if err != nil {
			t.Fatalf("[%s] parse factor failed: %v", polyStr, err)
		}
		factored, err := calc.EvalWithEnv(factorExpr, env)
		if err != nil {
			t.Fatalf("[%s] factor failed: %v", polyStr, err)
		}

		expandExpr, err := calc.Parse(fmt.Sprintf("expand(%s)", factored.String()))
		if err != nil {
			t.Fatalf("[%s] parse expand failed: %v", polyStr, err)
		}
		expanded, err := calc.EvalWithEnv(expandExpr, env)
		if err != nil {
			t.Fatalf("[%s] expand failed: %v", polyStr, err)
		}

		t.Logf("[%s] factored=%s, expanded=%s, evalPoly=%s", polyStr, factored.String(), expanded.String(), evalPoly.String())

		// Direct AST subtraction: expanded - evalPoly == 0
		diffExpr, err := calc.Parse(fmt.Sprintf("expand((%s) - (%s))", expanded.String(), evalPoly.String()))
		if err != nil {
			t.Fatalf("[%s] parse diff failed: %v", polyStr, err)
		}
		diffVal, err := calc.EvalWithEnv(diffExpr, env)
		if err != nil {
			t.Fatalf("[%s] eval diff failed: %v", polyStr, err)
		}

		if r, ok := diffVal.(*calc.RationalNode); !ok || r.Val.Sign() != 0 {
			t.Errorf("[%s] MR1 violated: expand(factor(P)) - P = %s, expected 0", polyStr, diffVal.String())
		}
	}
}

// MR2: Calculus Fundamental Theorem & Linearity Inversion
// Relation: diff(integrate(f(x), x), x) - f(x) == 0
// Relation: integrate(3*f - 5*g, x) == 3*integrate(f, x) - 5*integrate(g, x)
func TestMetamorphic_MR2_CalculusInversionAndLinearity(t *testing.T) {
	env := calc.NewEnv()

	testFunctions := []string{
		"x^3 - 2*x + 1",
		"3*x^4 + 5*x^2 - 7",
		"1 / (x + 1)",
		"x / (x^2 + 1)",
		"sin(x)",
		"cos(2*x)",
		"exp(3*x)",
		"x * exp(x)",
	}

	for _, fStr := range testFunctions {
		// 1. Fundamental Theorem of Calculus: d/dx \int f dx == f
		queryStr := fmt.Sprintf("expand(diff(integrate(%s, x), x) - (%s))", fStr, fStr)
		query, err := calc.Parse(queryStr)
		if err != nil {
			t.Fatalf("[%s] parse calculus query failed: %v", fStr, err)
		}
		diffVal, err := calc.EvalWithEnv(query, env)
		if err != nil {
			t.Fatalf("[%s] eval calculus query failed: %v", fStr, err)
		}

		if r, ok := diffVal.(*calc.RationalNode); !ok || r.Val.Sign() != 0 {
			t.Errorf("[%s] MR2 Fundamental Theorem violated: diff(integrate(f)) - f = %s, expected 0", fStr, diffVal.String())
		}
	}

	// 2. Linearity: integrate(2*f + 3*g, x) - (2*integrate(f, x) + 3*integrate(g, x)) == 0
	f := "x^2"
	g := "sin(x)"
	linQueryStr := fmt.Sprintf("expand(integrate(2*(%s) + 3*(%s), x) - (2*integrate(%s, x) + 3*integrate(%s, x)))", f, g, f, g)
	linQuery, err := calc.Parse(linQueryStr)
	if err != nil {
		t.Fatalf("parse linearity failed: %v", err)
	}
	linVal, err := calc.EvalWithEnv(linQuery, env)
	if err != nil {
		t.Fatalf("eval linearity failed: %v", err)
	}
	if r, ok := linVal.(*calc.RationalNode); !ok || r.Val.Sign() != 0 {
		t.Errorf("MR2 Linearity violated: %s, expected 0", linVal.String())
	}
}

// MR3: Linear Algebra Multiplicative & Matrix Decomposition Invariants
// Relation: det(A * B) == det(A) * det(B)
// Relation: P^T * L * U == A
// Relation: Q * R == A and Q^T * Q == I
func TestMetamorphic_MR3_LinearAlgebraInvariants(t *testing.T) {
	env := calc.NewEnv()

	// 1. Multiplicativity of Determinant: det(A * B) == det(A) * det(B)
	matPairs := []struct {
		A string
		B string
	}{
		{
			A: "[[1, 2], [3, 4]]",
			B: "[[5, 6], [7, 8]]",
		},
		{
			A: "[[2, -1], [1, 3]]",
			B: "[[4, 0], [-2, 1]]",
		},
		{
			A: "[[1, 2, 3], [0, 1, 4], [5, 6, 0]]",
			B: "[[2, 0, -1], [1, 3, 2], [0, -2, 1]]",
		},
	}

	for i, pair := range matPairs {
		queryStr := fmt.Sprintf("det((%s) * (%s)) - (det(%s) * det(%s))", pair.A, pair.B, pair.A, pair.B)
		query, err := calc.Parse(queryStr)
		if err != nil {
			t.Fatalf("[pair %d] parse failed: %v", i, err)
		}
		val, err := calc.EvalWithEnv(query, env)
		if err != nil {
			t.Fatalf("[pair %d] eval failed: %v", i, err)
		}
		if r, ok := val.(*calc.RationalNode); !ok || r.Val.Sign() != 0 {
			t.Errorf("[pair %d] MR3 det(A*B) = det(A)*det(B) violated: got %s, expected 0", i, val.String())
		}
	}

	// 2. LU Decomposition Reconstruction: P^T * L * U == A
	luMats := []string{
		"[[4, 3], [6, 3]]",
		"[[2, 1, 1], [4, -6, 0], [-2, 7, 2]]",
	}

	for _, mStr := range luMats {
		// lu(A) returns [P, L, U]
		queryStr := fmt.Sprintf("lu(%s)", mStr)
		query, err := calc.Parse(queryStr)
		if err != nil {
			t.Fatalf("[%s] parse lu failed: %v", mStr, err)
		}
		res, err := calc.EvalWithEnv(query, env)
		if err != nil {
			t.Fatalf("[%s] eval lu failed: %v", mStr, err)
		}

		list, ok := res.(*calc.ListNode)
		if !ok || len(list.Elements) != 3 {
			t.Fatalf("[%s] expected [P, L, U], got: %s", mStr, res.String())
		}
		pMat := list.Elements[0].String()
		lMat := list.Elements[1].String()
		uMat := list.Elements[2].String()

		reconQuery, err := calc.Parse(fmt.Sprintf("transpose(%s) * (%s) * (%s) - (%s)", pMat, lMat, uMat, mStr))
		if err != nil {
			t.Fatalf("[%s] parse recon failed: %v", mStr, err)
		}
		reconDiff, err := calc.EvalWithEnv(reconQuery, env)
		if err != nil {
			t.Fatalf("[%s] eval recon failed: %v", mStr, err)
		}

		// reconDiff should be zero matrix
		if matNode, ok := reconDiff.(*calc.MatrixNode); ok {
			for r := 0; r < matNode.Rows; r++ {
				for c := 0; c < matNode.Cols; c++ {
					elem := matNode.Data[r][c]
					if rat, ok := elem.(*calc.RationalNode); !ok || rat.Val.Sign() != 0 {
						t.Errorf("[%s] MR3 LU reconstruction non-zero at (%d,%d): %s", mStr, r, c, elem.String())
					}
				}
			}
		} else {
			t.Errorf("[%s] expected MatrixNode for LU diff, got: %s", mStr, reconDiff.String())
		}
	}

	// 3. QR Decomposition Reconstruction: Q * R == A
	qrMats := []string{
		"[[12, -51], [6, 167]]",
		"[[1, 2, 4], [3, 8, 14], [2, 6, 13]]",
	}

	for _, mStr := range qrMats {
		// qr(A) returns [Q, R]
		queryStr := fmt.Sprintf("qr(%s)", mStr)
		query, err := calc.Parse(queryStr)
		if err != nil {
			t.Fatalf("[%s] parse qr failed: %v", mStr, err)
		}
		res, err := calc.EvalWithEnv(query, env)
		if err != nil {
			t.Fatalf("[%s] eval qr failed: %v", mStr, err)
		}

		list, ok := res.(*calc.ListNode)
		if !ok || len(list.Elements) != 2 {
			t.Fatalf("[%s] expected [Q, R], got: %s", mStr, res.String())
		}
		qMat := list.Elements[0].String()
		rMat := list.Elements[1].String()

		reconQuery, err := calc.Parse(fmt.Sprintf("(%s) * (%s) - (%s)", qMat, rMat, mStr))
		if err != nil {
			t.Fatalf("[%s] parse qr recon failed: %v", mStr, err)
		}
		reconDiff, err := calc.EvalWithEnv(reconQuery, env)
		if err != nil {
			t.Fatalf("[%s] eval qr recon failed: %v", mStr, err)
		}

		if matNode, ok := reconDiff.(*calc.MatrixNode); ok {
			for r := 0; r < matNode.Rows; r++ {
				for c := 0; c < matNode.Cols; c++ {
					elem := matNode.Data[r][c]
					// check if elem simplifies to 0
					simpElem, _ := calc.EvalWithEnv(elem, env)
					if rat, ok := simpElem.(*calc.RationalNode); !ok || rat.Val.Sign() != 0 {
						t.Errorf("[%s] MR3 QR reconstruction non-zero at (%d,%d): %s", mStr, r, c, elem.String())
					}
				}
			}
		} else {
			t.Errorf("[%s] expected MatrixNode for QR diff, got: %s", mStr, reconDiff.String())
		}
	}
}

// MR4: Quantifier Elimination Dualities & Invariants
// Relation: qe(forall([x], phi)) == not(qe(exists([x], not(phi))))
// Relation: qe(exists([y], P(y) == 0)) == qe(exists([y], P(y - c) == 0))
func TestMetamorphic_MR4_QuantifierEliminationDualities(t *testing.T) {
	env := calc.NewEnv()

	// 1. De Morgan Duality for Quantifiers: forall(x, phi) <=> !exists(x, !phi)
	formulas := []struct {
		phi        string
		negPhi     string
		boundVar   string
	}{
		{
			phi:      "x^2 + 1 > 0",
			negPhi:   "x^2 + 1 <= 0",
			boundVar: "x",
		},
		{
			phi:      "x^2 - 4 <= 0",
			negPhi:   "x^2 - 4 > 0",
			boundVar: "x",
		},
		{
			phi:      "x^2 + y^2 >= 0",
			negPhi:   "x^2 + y^2 < 0",
			boundVar: "x",
		},
	}

	for i, f := range formulas {
		// qe(forall([x], phi))
		forallQuery, err := calc.Parse(fmt.Sprintf("qe(forall([%s], %s))", f.boundVar, f.phi))
		if err != nil {
			t.Fatalf("[formula %d] parse forall failed: %v", i, err)
		}
		forallRes, err := calc.EvalWithEnv(forallQuery, env)
		if err != nil {
			t.Fatalf("[formula %d] eval forall failed: %v", i, err)
		}

		// qe(exists([x], negPhi))
		existsQuery, err := calc.Parse(fmt.Sprintf("qe(exists([%s], %s))", f.boundVar, f.negPhi))
		if err != nil {
			t.Fatalf("[formula %d] parse exists failed: %v", i, err)
		}
		existsRes, err := calc.EvalWithEnv(existsQuery, env)
		if err != nil {
			t.Fatalf("[formula %d] eval exists failed: %v", i, err)
		}

		// Check boolean duality: forallRes == !existsRes
		fStr := forallRes.String()
		eStr := existsRes.String()

		if (fStr == "true" && eStr != "false") || (fStr == "false" && eStr != "true") {
			t.Errorf("[formula %d] MR4 De Morgan duality violated: forall(%s) = %s, but exists(%s) = %s",
				i, f.phi, fStr, f.negPhi, eStr)
		}
	}

	// 2. Translation Invariance: exists y (P(y) == 0) <=> exists y (P(y - 3) == 0)
	poly := "y^3 + 2*y + 1 == 0"
	translatedPoly := "(y - 3)^3 + 2*(y - 3) + 1 == 0"

	p1, _ := calc.Parse(fmt.Sprintf("qe(exists([y], %s))", poly))
	res1, err1 := calc.EvalWithEnv(p1, env)
	if err1 != nil {
		t.Fatalf("p1 eval failed: %v", err1)
	}

	p2, _ := calc.Parse(fmt.Sprintf("qe(exists([y], %s))", translatedPoly))
	res2, err2 := calc.EvalWithEnv(p2, env)
	if err2 != nil {
		t.Fatalf("p2 eval failed: %v", err2)
	}

	if res1.String() != res2.String() {
		t.Errorf("MR4 Translation Invariance violated: original=%s, translated=%s", res1.String(), res2.String())
	}
}

// MR5: Diophantine Equation Invariants (Pell Solvers)
// Relation: For fundamental solution (x1, y1), x1^2 - D * y1^2 == 1 (or -1) exactly.
func TestMetamorphic_MR5_DiophantinePellInvariants(t *testing.T) {
	env := calc.NewEnv()

	// Helper to extract x and y from solve_diophantine result: [x == xVal, y == yVal]
	extractXY := func(res calc.Node) (*big.Int, *big.Int, error) {
		list, ok := res.(*calc.ListNode)
		if !ok || len(list.Elements) < 2 {
			return nil, nil, fmt.Errorf("expected [x == ..., y == ...] list, got: %s", res.String())
		}
		relX, okX := list.Elements[0].(*calc.RelOpNode)
		relY, okY := list.Elements[1].(*calc.RelOpNode)
		if !okX || !okY {
			return nil, nil, fmt.Errorf("expected RelOpNode elements, got: %s", res.String())
		}
		rX, okRX := relX.RHS.(*calc.RationalNode)
		rY, okRY := relY.RHS.(*calc.RationalNode)
		if !okRX || !okRY {
			return nil, nil, fmt.Errorf("non-rational RHS: x=%s, y=%s", relX.RHS.String(), relY.RHS.String())
		}
		return rX.Val.Num(), rY.Val.Num(), nil
	}

	// Positive Pell: x^2 - D*y^2 == 1
	dPositive := []int{2, 3, 5, 6, 7, 10, 11, 13, 14, 15, 61, 109}

	for _, d := range dPositive {
		queryStr := fmt.Sprintf("solve_diophantine(x^2 - %d*y^2 == 1, [x, y])", d)
		query, err := calc.Parse(queryStr)
		if err != nil {
			t.Fatalf("[D=%d] parse failed: %v", d, err)
		}
		res, err := calc.EvalWithEnv(query, env)
		if err != nil {
			t.Fatalf("[D=%d] eval failed: %v", d, err)
		}

		xInt, yInt, err := extractXY(res)
		if err != nil {
			t.Fatalf("[D=%d] %v", d, err)
		}

		// Assert x > 0 and y > 0
		if xInt.Sign() <= 0 || yInt.Sign() <= 0 {
			t.Errorf("[D=%d] MR5 violated: non-positive fundamental solution (%s, %s)", d, xInt.String(), yInt.String())
		}

		// Assert x^2 - D * y^2 == 1 in big.Int
		xSq := new(big.Int).Mul(xInt, xInt)
		ySq := new(big.Int).Mul(yInt, yInt)
		dySq := new(big.Int).Mul(big.NewInt(int64(d)), ySq)
		diff := new(big.Int).Sub(xSq, dySq)

		if diff.Cmp(big.NewInt(1)) != 0 {
			t.Errorf("[D=%d] MR5 violated: x^2 - %d*y^2 = %s, expected 1", d, d, diff.String())
		}
	}

	// Negative Pell: x^2 - D*y^2 == -1
	dNegative := []int{2, 5, 10, 13, 17, 26, 29, 37, 41, 53, 61, 73, 89, 97, 109}

	for _, d := range dNegative {
		queryStr := fmt.Sprintf("solve_diophantine(x^2 - %d*y^2 == -1, [x, y])", d)
		query, err := calc.Parse(queryStr)
		if err != nil {
			t.Fatalf("[neg D=%d] parse failed: %v", d, err)
		}
		res, err := calc.EvalWithEnv(query, env)
		if err != nil {
			t.Fatalf("[neg D=%d] eval failed: %v", d, err)
		}

		xInt, yInt, err := extractXY(res)
		if err != nil {
			t.Fatalf("[neg D=%d] %v", d, err)
		}

		if xInt.Sign() <= 0 || yInt.Sign() <= 0 {
			t.Errorf("[neg D=%d] MR5 violated: non-positive fundamental solution (%s, %s)", d, xInt.String(), yInt.String())
		}

		xSq := new(big.Int).Mul(xInt, xInt)
		ySq := new(big.Int).Mul(yInt, yInt)
		dySq := new(big.Int).Mul(big.NewInt(int64(d)), ySq)
		diff := new(big.Int).Sub(xSq, dySq)

		if diff.Cmp(big.NewInt(-1)) != 0 {
			t.Errorf("[neg D=%d] MR5 violated: x^2 - %d*y^2 = %s, expected -1", d, d, diff.String())
		}
	}
}

// MR6: Algebraic Number Minimal Polynomial Invariant
// Relation: find_min_poly(alpha, deg) evaluated at alpha vanishes to zero.
func TestMetamorphic_MR6_MinimalPolynomialInvariants(t *testing.T) {
	env := calc.NewEnv()

	testAlgs := []struct {
		alpha   string
		maxDeg  int
		minpoly string
	}{
		{alpha: "sqrt(2)", maxDeg: 2, minpoly: "x^2 - 2"},
		{alpha: "sqrt(3)", maxDeg: 2, minpoly: "x^2 - 3"},
		{alpha: "sqrt(2) + 1", maxDeg: 2, minpoly: "x^2 - 2*x - 1"},
		{alpha: "sqrt(2) + sqrt(3)", maxDeg: 4, minpoly: "x^4 - 10*x^2 + 1"},
	}

	for _, item := range testAlgs {
		queryStr := fmt.Sprintf("find_min_poly(%s, %d)", item.alpha, item.maxDeg)
		query, err := calc.Parse(queryStr)
		if err != nil {
			t.Fatalf("[%s] parse find_min_poly failed: %v", item.alpha, err)
		}
		res, err := calc.EvalWithEnv(query, env)
		if err != nil {
			t.Fatalf("[%s] eval find_min_poly failed: %v", item.alpha, err)
		}

		expectedPoly, err := calc.Parse(item.minpoly)
		if err != nil {
			t.Fatalf("[%s] parse expected failed: %v", item.alpha, err)
		}

		// Difference between computed minpoly and canonical minpoly should be 0
		diffExpr, err := calc.Parse(fmt.Sprintf("expand((%s) - (%s))", res.String(), expectedPoly.String()))
		if err != nil {
			t.Fatalf("[%s] parse diff failed: %v", item.alpha, err)
		}
		diffVal, err := calc.EvalWithEnv(diffExpr, env)
		if err != nil {
			t.Fatalf("[%s] eval diff failed: %v", item.alpha, err)
		}

		if r, ok := diffVal.(*calc.RationalNode); !ok || r.Val.Sign() != 0 {
			t.Errorf("[%s] MR6 minpoly mismatch: got %s, expected %s", item.alpha, res.String(), item.minpoly)
		}
	}
}
