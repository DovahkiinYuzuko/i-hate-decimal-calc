package calc

import (
	"math/big"
	"strings"
	"testing"
)

// Helper to make a constant MPoly
func testConstPoly(val int64, numVars int) *MPoly {
	if val == 0 {
		return NewZeroMPoly(numVars)
	}
	return &MPoly{
		Terms: []Term{{Coeff: big.NewRat(val, 1), Mon: zeroMonomial(numVars)}},
		Sugar: 0,
	}
}

// Helper to make a single variable poly: x_i^deg
func testVarPowPoly(varIdx, deg, numVars int) *MPoly {
	exps := make([]int, numVars)
	exps[varIdx] = deg
	return &MPoly{
		Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial(exps)}},
		Sugar: deg,
	}
}

func TestIdealIntersection(t *testing.T) {
	// numVars = 2: x (0), y (1)
	numVars := 2
	order := OrderGRevLex

	// I = <x>, J = <y>
	// I cap J = <xy>
	polyX := testVarPowPoly(0, 1, numVars)
	polyY := testVarPowPoly(1, 1, numVars)

	I := []*MPoly{polyX}
	J := []*MPoly{polyY}

	inter, err := IdealIntersection(I, J, order)
	if err != nil {
		t.Fatalf("IdealIntersection failed: %v", err)
	}

	// Expected: xy (deg 1 in x, deg 1 in y)
	xyMon := newMonomial([]int{1, 1})
	expectedXY := &MPoly{
		Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: xyMon}},
		Sugar: 2,
	}

	eq, err := IdealEquals(inter, []*MPoly{expectedXY}, order)
	if err != nil {
		t.Fatalf("IdealEquals failed: %v", err)
	}
	if !eq {
		t.Errorf("expected <xy>, got %v", inter)
	}
}

func TestIdealColon(t *testing.T) {
	// numVars = 1: x (0)
	numVars := 1
	order := OrderGRevLex

	// I = <x^2>, f = x
	// I : x = <x>
	polyX2 := testVarPowPoly(0, 2, numVars)
	polyX := testVarPowPoly(0, 1, numVars)

	I := []*MPoly{polyX2}
	colon, err := IdealColon(I, polyX, order)
	if err != nil {
		t.Fatalf("IdealColon failed: %v", err)
	}

	eq, err := IdealEquals(colon, []*MPoly{polyX}, order)
	if err != nil {
		t.Fatalf("IdealEquals failed: %v", err)
	}
	if !eq {
		t.Errorf("expected <x>, got %v", colon)
	}
}

func TestIdealSaturation(t *testing.T) {
	// numVars = 1: x (0)
	numVars := 1
	order := OrderGRevLex

	// I = <x^2>, f = x
	// I : x^inf = <1>, stabilization index m = 2
	polyX2 := testVarPowPoly(0, 2, numVars)
	polyX := testVarPowPoly(0, 1, numVars)

	I := []*MPoly{polyX2}
	sat, m, err := IdealSaturation(I, polyX, order)
	if err != nil {
		t.Fatalf("IdealSaturation failed: %v", err)
	}

	if m != 2 {
		t.Errorf("expected stabilization index m = 2, got %d", m)
	}

	onePoly := testConstPoly(1, numVars)
	eq, err := IdealEquals(sat, []*MPoly{onePoly}, order)
	if err != nil {
		t.Fatalf("IdealEquals failed: %v", err)
	}
	if !eq {
		t.Errorf("expected <1>, got %v", sat)
	}
}

func TestIdealContainsAndEquals(t *testing.T) {
	// numVars = 2: x (0), y (1)
	numVars := 2
	order := OrderGRevLex

	polyX := testVarPowPoly(0, 1, numVars)
	polyY := testVarPowPoly(1, 1, numVars)
	polyX2 := testVarPowPoly(0, 2, numVars)

	// I = <x, y>, J = <x^2, y>
	// J subset I (true), I subset J (false)
	I := []*MPoly{polyX, polyY}
	J := []*MPoly{polyX2, polyY}

	contains, err := IdealContains(I, J, order)
	if err != nil || !contains {
		t.Errorf("expected I contains J, got %v (err: %v)", contains, err)
	}

	containsReverse, err := IdealContains(J, I, order)
	if err != nil || containsReverse {
		t.Errorf("expected J does not contain I, got %v (err: %v)", containsReverse, err)
	}

	// K = <x+y, x-y> -> equals <x, y>
	// x+y
	polyXplusY := polyX.Add(polyY, order)
	// x-y
	polyXminusY := polyX.Sub(polyY, order)
	K := []*MPoly{polyXplusY, polyXminusY}

	eq, err := IdealEquals(I, K, order)
	if err != nil || !eq {
		t.Errorf("expected <x, y> == <x+y, x-y>, got %v (err: %v)", eq, err)
	}
}

func TestIdealDimensionAndMaximalIndependentSet(t *testing.T) {
	order := OrderGRevLex

	// Case 1: I = <xy, xz> in Q[x, y, z] (3 vars: 0:x, 1:y, 2:z)
	// LM = {xy, xz}.
	// Subsets:
	// {y, z} (size 2): does any LM have vars subset of {y, z}?
	// xy has x (not in {y, z}), xz has x (not in {y, z}). So {y, z} is independent!
	// Maximum dimension must be 2, with independent set {1, 2} (y, z).
	numVars := 3
	xy := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 1, 0})}}}
	xz := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 0, 1})}}}
	G1, err := buchberger([]*MPoly{xy, xz}, order)
	if err != nil {
		t.Fatalf("buchberger failed: %v", err)
	}
	redG1 := reduceGroebnerBasis(G1, order)

	dim, u := IdealDimensionAndMaximalIndependentSet(redG1, numVars)
	if dim != 2 {
		t.Errorf("expected dimension 2 for <xy, xz>, got %d", dim)
	}
	if len(u) != 2 || u[0] != 1 || u[1] != 2 {
		t.Errorf("expected maximal independent set [1, 2] ({y, z}), got %v", u)
	}

	if IsZeroDimensional(redG1, numVars) {
		t.Errorf("<xy, xz> should not be zero dimensional")
	}

	// Case 2: I = <x^2, y^2> in Q[x, y] (2 vars: 0:x, 1:y)
	// Dimension must be 0, zero-dimensional must be true.
	numVars2 := 2
	x2 := testVarPowPoly(0, 2, numVars2)
	y2 := testVarPowPoly(1, 2, numVars2)
	G2, err := buchberger([]*MPoly{x2, y2}, order)
	if err != nil {
		t.Fatalf("buchberger failed: %v", err)
	}
	redG2 := reduceGroebnerBasis(G2, order)

	dim2, u2 := IdealDimensionAndMaximalIndependentSet(redG2, numVars2)
	if dim2 != 0 {
		t.Errorf("expected dimension 0 for <x^2, y^2>, got %d", dim2)
	}
	if len(u2) != 0 {
		t.Errorf("expected empty independent set for zero dim, got %v", u2)
	}
	if !IsZeroDimensional(redG2, numVars2) {
		t.Errorf("<x^2, y^2> should be zero dimensional")
	}
}

func TestZeroDimensionalRadical(t *testing.T) {
	order := OrderGRevLex
	numVars := 2 // x: 0, y: 1

	// I = <x^2, y^2> -> sqrt(I) = <x, y>
	x2 := testVarPowPoly(0, 2, numVars)
	y2 := testVarPowPoly(1, 2, numVars)
	I := []*MPoly{x2, y2}

	rad, err := ZeroDimensionalRadical(I, numVars, order)
	if err != nil {
		t.Fatalf("ZeroDimensionalRadical failed: %v", err)
	}

	x := testVarPowPoly(0, 1, numVars)
	y := testVarPowPoly(1, 1, numVars)
	expected := []*MPoly{x, y}

	eq, err := IdealEquals(rad, expected, order)
	if err != nil || !eq {
		t.Errorf("expected sqrt(<x^2, y^2>) == <x, y>, got %v (err: %v)", rad, err)
	}
}

func TestFindSeparatingLinearForm(t *testing.T) {
	order := OrderGRevLex
	numVars := 2 // x: 0, y: 1

	// I = <x^2 - 2, y^2 - 2>
	// x^2 - 2
	x2Minus2 := &MPoly{Terms: []Term{
		{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{2, 0})},
		{Coeff: big.NewRat(-2, 1), Mon: newMonomial([]int{0, 0})},
	}}
	// y^2 - 2
	y2Minus2 := &MPoly{Terms: []Term{
		{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{0, 2})},
		{Coeff: big.NewRat(-2, 1), Mon: newMonomial([]int{0, 0})},
	}}
	I := []*MPoly{x2Minus2, y2Minus2}

	coeffs, minPoly, err := FindSeparatingLinearForm(I, numVars, order)
	if err != nil {
		t.Fatalf("FindSeparatingLinearForm failed: %v", err)
	}

	if len(coeffs) != 2 {
		t.Fatalf("expected 2 coeffs, got %v", coeffs)
	}
	if minPoly == nil || minPoly.Degree() != 4 {
		t.Errorf("expected separating minimal polynomial of degree 4, got %v (deg %d)", minPoly, minPoly.Degree())
	}
}

func TestDecomposeZeroDimensionalIdeal(t *testing.T) {
	order := OrderGRevLex
	numVars := 2 // x: 0, y: 1

	// Test Case 1: I = <x^2 - 1, y> -> Two maximal primes: <x-1, y> and <x+1, y>
	x2Minus1 := &MPoly{Terms: []Term{
		{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{2, 0})},
		{Coeff: big.NewRat(-1, 1), Mon: newMonomial([]int{0, 0})},
	}}
	yPoly := testVarPowPoly(1, 1, numVars)
	I1 := []*MPoly{x2Minus1, yPoly}

	comps1, err := DecomposeZeroDimensionalIdeal(I1, numVars, order)
	if err != nil {
		t.Fatalf("DecomposeZeroDimensionalIdeal failed: %v", err)
	}
	if len(comps1) != 2 {
		t.Fatalf("expected 2 components for <x^2 - 1, y>, got %d", len(comps1))
	}

	// Verify intersection of components equals original ideal
	var inter []*MPoly
	if len(comps1) > 0 {
		inter = comps1[0].Primary
		for i := 1; i < len(comps1); i++ {
			inter, err = IdealIntersection(inter, comps1[i].Primary, order)
			if err != nil {
				t.Fatalf("IdealIntersection failed: %v", err)
			}
		}
	}
	for i, c := range comps1 {
		t.Logf("comp %d Primary:", i)
		for _, p := range c.Primary {
			t.Logf("    %v", p)
		}
	}
	t.Logf("intersection:")
	for _, p := range inter {
		t.Logf("    %v", p)
	}
	t.Logf("I1:")
	for _, p := range I1 {
		t.Logf("    %v", p)
	}
	eq1, err := IdealEquals(inter, I1, order)
	if err != nil || !eq1 {
		t.Errorf("expected intersection of components == I1, got eq=%v, err=%v", eq1, err)
	}

	// Test Case 2: I = <x^2, y> -> Single primary component: Q = <x^2, y>, P = <x, y>
	x2Poly := testVarPowPoly(0, 2, numVars)
	I2 := []*MPoly{x2Poly, yPoly}
	comps2, err := DecomposeZeroDimensionalIdeal(I2, numVars, order)
	if err != nil {
		t.Fatalf("DecomposeZeroDimensionalIdeal failed: %v", err)
	}
	if len(comps2) != 1 {
		t.Fatalf("expected 1 primary component for <x^2, y>, got %d", len(comps2))
	}
	eq2Q, err := IdealEquals(comps2[0].Primary, I2, order)
	if err != nil || !eq2Q {
		t.Errorf("expected primary component == <x^2, y>, got %v", comps2[0].Primary)
	}
	xPoly := testVarPowPoly(0, 1, numVars)
	expectedP := []*MPoly{xPoly, yPoly}
	eq2P, err := IdealEquals(comps2[0].Prime, expectedP, order)
	if err != nil || !eq2P {
		t.Errorf("expected prime radical == <x, y>, got %v", comps2[0].Prime)
	}
}

func TestPrimaryDecomposition_TwoPlanes(t *testing.T) {
	order := OrderGRevLex
	numVars := 3 // x: 0, y: 1, z: 2

	// I = <xy, xz>
	xy := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 1, 0})}}}
	xz := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 0, 1})}}}
	I := []*MPoly{xy, xz}

	comps, err := PrimaryDecomposition(I, numVars, order)
	if err != nil {
		t.Fatalf("PrimaryDecomposition failed: %v", err)
	}

	if len(comps) != 2 {
		t.Fatalf("expected 2 components for <xy, xz>, got %d", len(comps))
	}

	// Verify intersection of components == I
	inter, err := IdealIntersection(comps[0].Primary, comps[1].Primary, order)
	if err != nil {
		t.Fatalf("IdealIntersection failed: %v", err)
	}
	eq, err := IdealEquals(inter, I, order)
	if err != nil || !eq {
		t.Errorf("expected intersection == I, got eq=%v, err=%v", eq, err)
	}
}

func TestPrimaryDecomposition_EmbeddedPoint(t *testing.T) {
	order := OrderGRevLex
	numVars := 2 // x: 0, y: 1

	// I = <x^2, xy>
	x2 := testVarPowPoly(0, 2, numVars)
	xy := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 1})}}}
	I := []*MPoly{x2, xy}

	comps, err := PrimaryDecomposition(I, numVars, order)
	if err != nil {
		t.Fatalf("PrimaryDecomposition failed: %v", err)
	}

	if len(comps) != 2 {
		t.Fatalf("expected 2 components (line + embedded point), got %d", len(comps))
	}

	inter, err := IdealIntersection(comps[0].Primary, comps[1].Primary, order)
	if err != nil {
		t.Fatalf("IdealIntersection failed: %v", err)
	}
	eq, err := IdealEquals(inter, I, order)
	if err != nil || !eq {
		t.Errorf("expected intersection == I, got eq=%v, err=%v", eq, err)
	}
}

func TestIdealRadical_EmbeddedPoint(t *testing.T) {
	order := OrderGRevLex
	numVars := 2 // x: 0, y: 1

	// I = <x^2, xy> -> sqrt(I) = <x>
	x2 := testVarPowPoly(0, 2, numVars)
	xy := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 1})}}}
	I := []*MPoly{x2, xy}

	rad, err := IdealRadical(I, numVars, order)
	if err != nil {
		t.Fatalf("IdealRadical failed: %v", err)
	}

	xPoly := testVarPowPoly(0, 1, numVars)
	expected := []*MPoly{xPoly}
	eq, err := IdealEquals(rad, expected, order)
	if err != nil || !eq {
		t.Errorf("expected sqrt(<x^2, xy>) == <x>, got %v (err: %v)", rad, err)
	}
}

func TestEvalString_IdealCommands(t *testing.T) {
	// 1. ideal_dimension
	resDim, err := EvalString("ideal_dimension([x*y, x*z], [x, y, z])")
	if err != nil {
		t.Fatalf("ideal_dimension failed: %v", err)
	}
	if Format(resDim) != "2" {
		t.Errorf("expected dimension 2, got: %s", Format(resDim))
	}

	// 2. ideal_radical
	resRad, err := EvalString("ideal_radical([x^2, x*y], [x, y])")
	if err != nil {
		t.Fatalf("ideal_radical failed: %v", err)
	}
	if Format(resRad) != "[x]" {
		t.Errorf("expected [x], got: %s", Format(resRad))
	}

	// 3. ideal_colon
	resColon, err := EvalString("ideal_colon([x^2], x, [x])")
	if err != nil {
		t.Fatalf("ideal_colon failed: %v", err)
	}
	if Format(resColon) != "[x]" {
		t.Errorf("expected [x], got: %s", Format(resColon))
	}

	// 4. ideal_saturation
	resSat, err := EvalString("ideal_saturation([x^2], x, [x])")
	if err != nil {
		t.Fatalf("ideal_saturation failed: %v", err)
	}
	if Format(resSat) != "[1]" {
		t.Errorf("expected [1], got: %s", Format(resSat))
	}

	// 5. primary_decomposition
	resDecomp, err := EvalString("primary_decomposition([x^2 - 1, y], [x, y])")
	if err != nil {
		t.Fatalf("primary_decomposition failed: %v", err)
	}
	listNode, ok := resDecomp.(*ListNode)
	if !ok || len(listNode.Elements) != 2 {
		t.Errorf("expected 2 components from primary_decomposition, got: %s", Format(resDecomp))
	}
}

func TestPrimaryDecomposition_ComplexCombinations(t *testing.T) {
	order := OrderGRevLex

	// Tricky Case 1: Ideal with higher powers and intersection
	// I = <x^3, y^2> in Q[x, y] -> dim = 0, single primary component with radical <x, y>
	x3 := makeVarPowMPoly(0, 3, 2)
	y2 := makeVarPowMPoly(1, 2, 2)
	I1 := []*MPoly{x3, y2}

	comps1, err := PrimaryDecomposition(I1, 2, order)
	if err != nil {
		t.Fatalf("PrimaryDecomposition of <x^3, y^2> failed: %v", err)
	}
	if len(comps1) != 1 {
		t.Errorf("expected 1 primary component for <x^3, y^2>, got %d", len(comps1))
	}

	// Radical must be <x, y>
	rad1, err := IdealRadical(I1, 2, order)
	if err != nil {
		t.Fatalf("IdealRadical failed: %v", err)
	}
	expectedRad1 := []*MPoly{makeVarPowMPoly(0, 1, 2), makeVarPowMPoly(1, 1, 2)}
	eqRad, err := IdealEquals(rad1, expectedRad1, order)
	if err != nil || !eqRad {
		t.Errorf("expected radical <x, y>, got %v", rad1)
	}

	// Tricky Case 2: Reducible curve <x^2 - y^2> = <(x-y)(x+y)>
	// Decomposition must yield 2 components: <x - y> and <x + y>
	xMinusY := &MPoly{Terms: []Term{
		{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 0})},
		{Coeff: big.NewRat(-1, 1), Mon: newMonomial([]int{0, 1})},
	}}
	xPlusY := &MPoly{Terms: []Term{
		{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 0})},
		{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{0, 1})},
	}}
	x2MinusY2 := xMinusY.Mul(xPlusY, order)
	I2 := []*MPoly{x2MinusY2}

	comps2, err := PrimaryDecomposition(I2, 2, order)
	if err != nil {
		t.Fatalf("PrimaryDecomposition of <x^2 - y^2> failed: %v", err)
	}
	if len(comps2) != 2 {
		t.Fatalf("expected 2 components for <x^2 - y^2>, got %d", len(comps2))
	}

	// Intersection must equal I2
	inter2, err := IdealIntersection(comps2[0].Primary, comps2[1].Primary, order)
	if err != nil {
		t.Fatalf("IdealIntersection failed: %v", err)
	}
	eq2, err := IdealEquals(inter2, I2, order)
	if err != nil || !eq2 {
		t.Errorf("expected intersection == <x^2 - y^2>, got eq=%v, err=%v", eq2, err)
	}
}

func TestIdealOperations_CLI_EndToEnd(t *testing.T) {
	// 1. Colon ideal: <x*y> : x == <y>
	res1, err := EvalString("ideal_colon([x*y], x, [x, y])")
	if err != nil {
		t.Fatalf("ideal_colon failed: %v", err)
	}
	if Format(res1) != "[y]" {
		t.Errorf("expected [y], got: %s", Format(res1))
	}

	// 2. Saturation: <x*y^2> : y^inf == <x>
	res2, err := EvalString("ideal_saturation([x*y^2], y, [x, y])")
	if err != nil {
		t.Fatalf("ideal_saturation failed: %v", err)
	}
	if Format(res2) != "[x]" {
		t.Errorf("expected [x], got: %s", Format(res2))
	}

	// 3. Intersection: <x> cap <y> == <x*y>
	res3, err := EvalString("ideal_intersection([x], [y], [x, y])")
	if err != nil {
		t.Fatalf("ideal_intersection failed: %v", err)
	}
	if Format(res3) != "[x*y]" {
		t.Errorf("expected [x*y], got: %s", Format(res3))
	}

	// 4. Dimension of twisted cubic: <y - x^2, z - x^3> in Q[x, y, z] -> dim = 1
	res4, err := EvalString("ideal_dimension([y - x^2, z - x^3], [x, y, z])")
	if err != nil {
		t.Fatalf("ideal_dimension of twisted cubic failed: %v", err)
	}
	if Format(res4) != "1" {
		t.Errorf("expected dimension 1 for twisted cubic, got: %s", Format(res4))
	}
}

func TestPrimaryDecomposition_AdversarialBenchmarks(t *testing.T) {
	order := OrderGRevLex

	// 1. Counterexample Case: I = <x - y, xy> == <x - y, x^2>
	// Must not be decomposed into radical <x, y>; intersection must strictly equal I!
	numVars2 := 2 // x: 0, y: 1
	xMinusY := &MPoly{Terms: []Term{
		{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 0})},
		{Coeff: big.NewRat(-1, 1), Mon: newMonomial([]int{0, 1})},
	}}
	xy := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 1})}}}
	ICounter := []*MPoly{xMinusY, xy}

	compsCounter, err := PrimaryDecomposition(ICounter, numVars2, order)
	if err != nil {
		t.Fatalf("decomposition of <x - y, xy> failed: %v", err)
	}
	interCounter := compsCounter[0].Primary
	for i := 1; i < len(compsCounter); i++ {
		interCounter, err = IdealIntersection(interCounter, compsCounter[i].Primary, order)
		if err != nil {
			t.Fatalf("IdealIntersection failed: %v", err)
		}
	}
	eqCounter, err := IdealEquals(interCounter, ICounter, order)
	if err != nil || !eqCounter {
		t.Errorf("counterexample <x - y, xy> failed: intersection != I (eq=%v, err=%v)", eqCounter, err)
	}

	// 2. Killer Benchmark T05: I = <x^2 - 2, y^2 - 2>
	// Coordinate eliminants are irreducible over Q, but ideal splits into 2 components:
	// <x^2 - 2, y - x> and <x^2 - 2, y + x>
	x2Minus2 := &MPoly{Terms: []Term{
		{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{2, 0})},
		{Coeff: big.NewRat(-2, 1), Mon: zeroMonomial(numVars2)},
	}}
	y2Minus2 := &MPoly{Terms: []Term{
		{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{0, 2})},
		{Coeff: big.NewRat(-2, 1), Mon: zeroMonomial(numVars2)},
	}}
	IT05 := []*MPoly{x2Minus2, y2Minus2}

	compsT05, err := PrimaryDecomposition(IT05, numVars2, order)
	if err != nil {
		t.Fatalf("decomposition of T05 <x^2 - 2, y^2 - 2> failed: %v", err)
	}
	if len(compsT05) != 2 {
		t.Fatalf("expected 2 components for T05 <x^2 - 2, y^2 - 2>, got %d", len(compsT05))
	}
	interT05, err := IdealIntersection(compsT05[0].Primary, compsT05[1].Primary, order)
	if err != nil {
		t.Fatalf("IdealIntersection failed for T05: %v", err)
	}
	eqT05, err := IdealEquals(interT05, IT05, order)
	if err != nil || !eqT05 {
		t.Errorf("expected intersection == IT05, got eq=%v, err=%v", eqT05, err)
	}

	// 3. Killer Benchmark T03: I = <xz, yz, z^2> in Q[x, y, z]
	// Components: <z> and embedded point <x, y, z^2>
	numVars3 := 3 // x: 0, y: 1, z: 2
	xz := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 0, 1})}}}
	yz := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{0, 1, 1})}}}
	z2 := makeVarPowMPoly(2, 2, numVars3)
	IT03 := []*MPoly{xz, yz, z2}

	compsT03, err := PrimaryDecomposition(IT03, numVars3, order)
	if err != nil {
		t.Fatalf("decomposition of T03 <xz, yz, z^2> failed: %v", err)
	}
	if len(compsT03) != 2 {
		t.Fatalf("expected 2 components for T03, got %d", len(compsT03))
	}
	interT03, err := IdealIntersection(compsT03[0].Primary, compsT03[1].Primary, order)
	if err != nil {
		t.Fatalf("IdealIntersection failed for T03: %v", err)
	}
	eqT03, err := IdealEquals(interT03, IT03, order)
	if err != nil || !eqT03 {
		t.Errorf("expected intersection == IT03, got eq=%v, err=%v", eqT03, err)
	}

	// 4. Killer Benchmark T07: 3 coordinate axes I = <xy, xz, yz> in Q[x, y, z]
	// Components: <x, y>, <x, z>, <y, z>
	xy3 := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial([]int{1, 1, 0})}}}
	IT07 := []*MPoly{xy3, xz, yz}

	compsT07, err := PrimaryDecomposition(IT07, numVars3, order)
	if err != nil {
		t.Fatalf("decomposition of T07 <xy, xz, yz> failed: %v", err)
	}
	if len(compsT07) != 3 {
		t.Fatalf("expected 3 components for T07, got %d", len(compsT07))
	}
	interT07 := compsT07[0].Primary
	for i := 1; i < len(compsT07); i++ {
		interT07, err = IdealIntersection(interT07, compsT07[i].Primary, order)
		if err != nil {
			t.Fatalf("IdealIntersection failed for T07: %v", err)
		}
	}
	eqT07, err := IdealEquals(interT07, IT07, order)
	if err != nil || !eqT07 {
		t.Errorf("expected intersection == IT07, got eq=%v, err=%v", eqT07, err)
	}

	// 5. Killer Benchmark T11: Fat point I = <x^2, xy, y^2> in Q[x, y]
	// Already primary, must NOT be split!
	x2 := makeVarPowMPoly(0, 2, numVars2)
	y2 := makeVarPowMPoly(1, 2, numVars2)
	IT11 := []*MPoly{x2, xy, y2}

	compsT11, err := PrimaryDecomposition(IT11, numVars2, order)
	if err != nil {
		t.Fatalf("decomposition of T11 <x^2, xy, y^2> failed: %v", err)
	}
	if len(compsT11) != 1 {
		t.Errorf("expected exactly 1 primary component for T11 <x^2, xy, y^2>, got %d", len(compsT11))
	}
}

func TestPrimaryDecomposition_VerifyCertificate_And_Explain_And_Lean(t *testing.T) {
	env := NewEnv()
	exprStr := "primary_decomposition([x^2 - 1, y], [x, y])"
	parsed, err := Parse(exprStr)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	res, err := EvalWithEnv(parsed, env)
	if err != nil {
		t.Fatalf("EvalWithEnv failed: %v", err)
	}

	// 1. Test VerifyComputation
	cert, err := VerifyComputation(parsed, res, env)
	if err != nil {
		t.Fatalf("VerifyComputation failed: %v", err)
	}
	if cert == nil {
		t.Fatalf("expected non-nil VerificationCertificate")
	}
	if !cert.IsVerified {
		t.Errorf("expected IsVerified == true, got false. Details: %s", cert.Details)
	}
	if cert.Residual.String() != "0" {
		t.Errorf("expected residual 0, got: %s", cert.Residual.String())
	}

	certIR, ok := cert.CertIR.(*IdealDecompositionCertificate)
	if !ok {
		t.Fatalf("expected *IdealDecompositionCertificate in CertIR, got %T", cert.CertIR)
	}
	if len(certIR.ForwardSyzygies) == 0 {
		t.Errorf("expected forward syzygies > 0, got 0")
	}
	if len(certIR.BackwardSyzygies) == 0 {
		t.Errorf("expected backward syzygies > 0, got 0")
	}
	if len(certIR.IrredundancyWitness) == 0 {
		t.Errorf("expected irredundancy witnesses > 0, got 0")
	}

	// 2. Test ProofTrace and Explain steps
	trace, err := cert.ToProofTrace()
	if err != nil {
		t.Fatalf("ToProofTrace failed: %v", err)
	}
	if trace.StepCount() != 5 {
		t.Errorf("expected exactly 5 academic proof steps, got %d", trace.StepCount())
	}
	expectedRules := []string{
		"DimensionEliminationAnalysis",
		"PrimaryDecompositionGTZ",
		"ForwardSyzygyContainment",
		"BackwardSyzygyContainment",
		"IrredundancyMinimalityCheck",
	}
	for i, r := range expectedRules {
		if i < len(trace.Steps) && trace.Steps[i].Rule != r {
			t.Errorf("step %d rule expected %s, got %s", i, r, trace.Steps[i].Rule)
		}
	}
	explainOutput := trace.RenderExplain("ja")
	if !strings.Contains(explainOutput, "DimensionEliminationAnalysis") {
		t.Errorf("expected explain output to contain DimensionEliminationAnalysis, got: %s", explainOutput)
	}

	// 3. Test Lean 4 Theorem Generation
	leanCode, err := GenerateLeanSource("ihd_primary_decomp_test", cert, parsed, res)
	if err != nil {
		t.Fatalf("GenerateLeanSource failed: %v", err)
	}
	if !strings.Contains(leanCode, "theorem ihd_primary_decomp_test") {
		t.Errorf("expected 'theorem ihd_primary_decomp_test' in Lean code, got: %s", leanCode)
	}
	if !strings.Contains(leanCode, "by") {
		t.Errorf("expected proof tactic in Lean code, got: %s", leanCode)
	}
	if strings.Contains(leanCode, "sorry") {
		t.Errorf("Lean code must not contain monkey-patch 'sorry': %s", leanCode)
	}
}



