package calc

import (
	"math/big"
	"strings"
	"testing"
)

func TestDecomposeSOS_UnivariateSquare(t *testing.T) {
	// P(x) = x^2 + 2x + 1 = (x + 1)^2
	vars := []string{"x"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0}

	p, err := nodeToMPoly(mustParseNode(t, "x^2 + 2*x + 1"), varMap, 1, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	res, err := DecomposeSOS(p, vars, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}

	if res.Status != SOSCertified {
		t.Fatalf("expected SOSCertified, got %v", res.Status)
	}

	// Verify reconstruction: sum c_i * q_i^2 == P
	reconstructed := NewZeroMPoly(1)
	for _, term := range res.Terms {
		sq := term.Poly.Mul(term.Poly, order)
		sqScaled := sq.MulTerm(Term{Coeff: term.Weight, Mon: zeroMonomial(1)}, order)
		reconstructed = reconstructed.Add(sqScaled, order)
	}

	diff := p.Sub(reconstructed, order)
	if !diff.IsZero() {
		t.Fatalf("reconstruction mismatch: p - sum c_i*q_i^2 != 0, got %v", mpolyToNode(diff, vars))
	}
}

func TestDecomposeSOS_BivariateQuadratic(t *testing.T) {
	// P(x, y) = 2*x^2 + 2*x*y + 2*y^2
	// Expected: 2 * (x + 1/2*y)^2 + 3/2 * y^2
	vars := []string{"x", "y"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0, "y": 1}

	p, err := nodeToMPoly(mustParseNode(t, "2*x^2 + 2*x*y + 2*y^2"), varMap, 2, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	res, err := DecomposeSOS(p, vars, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}

	if res.Status != SOSCertified {
		t.Fatalf("expected SOSCertified, got %v", res.Status)
	}

	for _, term := range res.Terms {
		if term.Weight.Sign() <= 0 {
			t.Fatalf("expected positive weight, got %v", term.Weight)
		}
	}

	// Verify reconstruction
	reconstructed := NewZeroMPoly(2)
	for _, term := range res.Terms {
		sq := term.Poly.Mul(term.Poly, order)
		sqScaled := sq.MulTerm(Term{Coeff: term.Weight, Mon: zeroMonomial(2)}, order)
		reconstructed = reconstructed.Add(sqScaled, order)
	}

	diff := p.Sub(reconstructed, order)
	if !diff.IsZero() {
		t.Fatalf("reconstruction mismatch: p - sum c_i*q_i^2 != 0, got %v", mpolyToNode(diff, vars))
	}
}

func TestDecomposeSOS_Refuted(t *testing.T) {
	// P(x) = x^2 - 1 => P(0) = -1 < 0
	vars := []string{"x"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0}

	p, err := nodeToMPoly(mustParseNode(t, "x^2 - 1"), varMap, 1, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	res, err := DecomposeSOS(p, vars, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}

	if res.Status != SOSRefuted {
		t.Fatalf("expected SOSRefuted, got %v", res.Status)
	}
	if len(res.Counterexample) == 0 {
		t.Fatalf("expected non-empty counterexample point")
	}
}

func TestDecomposeSOS_MotzkinPolynomial(t *testing.T) {
	// Motzkin polynomial: M(x, y) = x^4*y^2 + x^2*y^4 - 3*x^2*y^2 + 1 >= 0 on R, but NOT a sum of squares!
	vars := []string{"x", "y"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0, "y": 1}

	p, err := nodeToMPoly(mustParseNode(t, "x^4*y^2 + x^2*y^4 - 3*x^2*y^2 + 1"), varMap, 2, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	res, err := DecomposeSOS(p, vars, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}

	// Must NOT be falsely certified!
	if res.Status == SOSCertified {
		t.Fatalf("Motzkin polynomial must not be certified as SOS!")
	}
	if res.Status != SOSNoCertificateFound {
		t.Fatalf("expected SOSNoCertificateFound, got %v", res.Status)
	}
}


func TestEvalSOSDecompose_Builtin(t *testing.T) {
	env := NewEnv()
	node, err := Parse("sos_decompose(x^2 + 2*x + 1, [x])")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	res, err := EvalWithEnv(node, env)
	if err != nil {
		t.Fatalf("EvalWithEnv error: %v", err)
	}

	listNode, ok := res.(*ListNode)
	if !ok || len(listNode.Elements) != 2 {
		t.Fatalf("expected 2-element list result, got: %s", Format(res))
	}
	statusVar, ok := listNode.Elements[0].(*VarNode)
	if !ok || statusVar.Name != "certified" {
		t.Fatalf("expected certified, got: %s", Format(listNode.Elements[0]))
	}
}

func TestDecomposeSOS_OddDegreeCauchyRefutation(t *testing.T) {
	// P(x) = x^3 + 10000 (degree 3, odd)
	// Must mathematically prove a negative witness where P(w) < 0, NOT a fake witness like -10 (which gives +9000).
	p := &MPoly{
		Terms: []Term{
			{Coeff: big.NewRat(1, 1), Mon: Monomial{Exps: []int{3}, TotalDeg: 3}},
			{Coeff: big.NewRat(10000, 1), Mon: Monomial{Exps: []int{0}, TotalDeg: 0}},
		},
		Sugar: 3,
	}
	order := OrderGRevLex
	res, err := DecomposeSOS(p, []string{"x"}, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}
	if res.Status != SOSRefuted {
		t.Fatalf("expected SOSRefuted, got %v", res.Status)
	}
	if len(res.Counterexample) == 0 {
		t.Fatalf("expected counterexample witness point")
	}
	// Verify that P(w) is strictly negative
	pt := []*big.Rat{res.Counterexample["x"]}
	val := evaluateAt(p, pt)
	if val.Sign() >= 0 {
		t.Fatalf("counterexample %v is invalid! P(w) = %s >= 0", res.Counterexample, val.String())
	}
}

func TestDecomposeSOS_SymmetricAveraging(t *testing.T) {
	// P(x, y) = x^4 + 2*x^2*y^2 + y^4 = (x^2 + y^2)^2
	// Monomial basis includes [x^2, x*y, y^2].
	// The cross term x^2 * y^2 can be formed by (x^2) * (y^2) and (x*y) * (x*y).
	// Averaging and LDL^T must successfully decompose it into a valid SOS without double-subtraction.
	p := &MPoly{
		Terms: []Term{
			{Coeff: big.NewRat(1, 1), Mon: Monomial{Exps: []int{4, 0}, TotalDeg: 4}},
			{Coeff: big.NewRat(2, 1), Mon: Monomial{Exps: []int{2, 2}, TotalDeg: 4}},
			{Coeff: big.NewRat(1, 1), Mon: Monomial{Exps: []int{0, 4}, TotalDeg: 4}},
		},
		Sugar: 4,
	}
	order := OrderGRevLex
	res, err := DecomposeSOS(p, []string{"x", "y"}, order)
	if err != nil {
		t.Fatalf("DecomposeSOS error: %v", err)
	}
	if res.Status != SOSCertified {
		t.Fatalf("expected SOSCertified for (x^2 + y^2)^2, got %v", res.Status)
	}
}

func TestAnalyzeSOSBasis_HalfNewtonInteriorAndReznickPruning(t *testing.T) {
	// P(x) = (x^3 + x^2 - (1/2)*x^4)^2 = (1/4)*x^8 - x^7 + 2*x^5 + x^4
	// Note that x^6 coefficient is 0, so x^3 * x^3 = x^6 is absent from supp(P),
	// yet x^6 = x^2 * x^4 allows x^3 to survive Reznick pruning!
	vars := []string{"x"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0}

	rawP := mustParseNode(t, "(1/4)*x^8 - x^7 + 2*x^5 + x^4")
	evalP, err := Eval(expandNode(rawP))
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	p, err := nodeToMPoly(evalP, varMap, 1, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	analysis, err := analyzeSOSBasis(p, vars, order)
	if err != nil {
		t.Fatalf("analyzeSOSBasis error: %v", err)
	}
	if analysis.Impossible {
		t.Fatalf("expected feasible basis, got Impossible=true")
	}

	hasX3 := false
	for _, m := range analysis.Basis {
		if len(m.Exps) == 1 && m.Exps[0] == 3 {
			hasX3 = true
		}
		if m.TotalDeg < 2 || m.TotalDeg > 4 {
			t.Fatalf("unexpected basis monomial outside [2,4]: %v", m.Exps)
		}
	}
	if !hasX3 {
		t.Fatalf("expected interior Half Newton monomial x^3 in basis, got %+v", analysis.Basis)
	}
}

func TestAnalyzeSOSBasis_SignSymmetryAndAffineIdentity(t *testing.T) {
	// P(x, y) = (x^2 - y^2)^2 + (x*y)^2 = x^4 - x^2*y^2 + y^4
	// All exponents are even mod 2 -> W = {(0,0)} -> two blocks: {x^2, y^2} and {x*y}.
	// Verify v^T Q(lambda) v == P holds identically for arbitrary rational lambda.
	vars := []string{"x", "y"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0, "y": 1}

	p, err := nodeToMPoly(mustParseNode(t, "x^4 - x^2*y^2 + y^4"), varMap, 2, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	analysis, err := analyzeSOSBasis(p, vars, order)
	if err != nil {
		t.Fatalf("analyzeSOSBasis error: %v", err)
	}
	if len(analysis.Blocks) != 2 {
		t.Fatalf("expected 2 sign-symmetry blocks for x^4 - x^2*y^2 + y^4, got %d (%v)", len(analysis.Blocks), analysis.Blocks)
	}

	space := buildSOSAffineSpace(analysis)
	if space.Dim < 1 {
		t.Fatalf("expected affine dimension >= 1 for shared monomial x^2*y^2, got %d", space.Dim)
	}

	// Test reconstruction invariance at arbitrary rational lambda entries
	probeLambda := make([]*big.Rat, space.Dim)
	for k := range probeLambda {
		probeLambda[k] = big.NewRat(int64(3*k-5), 7)
	}
	reconstructed := reconstructMPolyFromAffine(space, probeLambda, order)
	diff := p.Sub(reconstructed, order)
	if !diff.IsZero() {
		t.Fatalf("affine Gram identity violated for arbitrary lambda: diff=%v", mpolyToNode(diff, vars))
	}
}

func TestAnalyzeSOSBasis_SupportCoverageImpossible(t *testing.T) {
	// P(x, y) = x^4 + y^4 + x^2*y (vertex parity / support coverage violation: x^2*y cannot be represented cleanly if forced)
	// Or P(x, y) = x^4 + y^4 - x^2*y^2? Wait: x^2*y^2 IS inside half Newton. Instead x^4 + y^4 + x*y:
	// x*y requires x and y in basis, whose squares x^2 and y^2 are in conv({(4,0),(0,4),(1,1)})/2,
	// WAIT: is (1, 0) in 1/2 conv({(4,0),(0,4),(1,1)})? No! 2*(1,0) = (2,0) HAS y=0, convex combo requires coeff_y=0 -> only (4,0), then x in 2*(1,0) vs (4,0) needs lambda*(4,0) + (1-lambda)*(0,4) -> y>0 unless lambda=1!
	// So (2,0) IS on the segment [(0,0), (4,0)]? No, (0,0) is NOT in supp! Only (4,0),(0,4),(1,1)!
	// Hence (1,0) and (0,1) are NOT in Half Newton Polytope! Only {(2,0),(0,2)} remain, and they CANNOT form (1,1) = x*y!
	vars := []string{"x", "y"}
	order := OrderGRevLex
	varMap := map[string]int{"x": 0, "y": 1}

	p, err := nodeToMPoly(mustParseNode(t, "x^4 + y^4 + x*y"), varMap, 2, order)
	if err != nil {
		t.Fatalf("nodeToMPoly error: %v", err)
	}

	analysis, err := analyzeSOSBasis(p, vars, order)
	if err != nil {
		t.Fatalf("analyzeSOSBasis error: %v", err)
	}
	if !analysis.Impossible {
		t.Fatalf("expected Impossible=true for x^4 + y^4 + x*y due to missing support coverage, got Basis=%v", analysis.Basis)
	}
}

func TestExactRationalLDLTWithWitness_ZeroPivotAndIndefinite(t *testing.T) {
	// Rank-deficient PSD matrix [[1, 0, -1], [0, 0, 0], [-1, 0, 1]]
	psdMat := [][]*big.Rat{
		{big.NewRat(1, 1), big.NewRat(0, 1), big.NewRat(-1, 1)},
		{big.NewRat(0, 1), big.NewRat(0, 1), big.NewRat(0, 1)},
		{big.NewRat(-1, 1), big.NewRat(0, 1), big.NewRat(1, 1)},
	}
	resPSD := exactRationalLDLTWithWitness(psdMat)
	if !resPSD.IsPSD {
		t.Fatalf("expected rank-deficient PSD matrix to pass LDL^T")
	}
	ker := extractKernelBasisFromLDLT(resPSD.L, resPSD.D)
	if len(ker) != 2 {
		t.Fatalf("expected 2-dimensional kernel for rank-1 3x3 matrix, got %d", len(ker))
	}

	// Zero-diagonal matrix with nonzero off-diagonal [[1, 0, 0], [0, 0, 2], [0, 2, 1]] (indefinite)
	indefMat := [][]*big.Rat{
		{big.NewRat(1, 1), big.NewRat(0, 1), big.NewRat(0, 1)},
		{big.NewRat(0, 1), big.NewRat(0, 1), big.NewRat(2, 1)},
		{big.NewRat(0, 1), big.NewRat(2, 1), big.NewRat(1, 1)},
	}
	resIndef := exactRationalLDLTWithWitness(indefMat)
	if resIndef.IsPSD {
		t.Fatalf("expected indefinite matrix to fail LDL^T")
	}
	if len(resIndef.Witness) != 3 {
		t.Fatalf("expected 3-vector refutation witness, got %v", resIndef.Witness)
	}
	qVal := evalSymQuadForm(indefMat, resIndef.Witness)
	if qVal.Sign() >= 0 {
		t.Fatalf("expected w^T A w < 0 for refutation witness, got %s", qVal.String())
	}
}

func assertSOSCertifiedAndExact(t *testing.T, expr string, vars []string) {
	t.Helper()
	order := OrderGRevLex
	varMap := make(map[string]int, len(vars))
	for i, v := range vars {
		varMap[v] = i
	}
	raw := mustParseNode(t, expr)
	evalP, err := Eval(expandNode(raw))
	if err != nil {
		t.Fatalf("Eval(%s) error: %v", expr, err)
	}
	p, err := nodeToMPoly(evalP, varMap, len(vars), order)
	if err != nil {
		t.Fatalf("nodeToMPoly(%s) error: %v", expr, err)
	}
	res, err := DecomposeSOS(p, vars, order)
	if err != nil {
		t.Fatalf("DecomposeSOS(%s) error: %v", expr, err)
	}
	if res.Status != SOSCertified {
		t.Fatalf("expected SOSCertified for %s, got %v", expr, res.Status)
	}
	reconstructed := NewZeroMPoly(len(vars))
	for _, term := range res.Terms {
		if term.Weight.Sign() <= 0 {
			t.Fatalf("non-positive weight %s in %s", term.Weight.String(), expr)
		}
		sq := term.Poly.Mul(term.Poly, order)
		sqScaled := sq.MulTerm(Term{Coeff: term.Weight, Mon: zeroMonomial(len(vars))}, order)
		reconstructed = reconstructed.Add(sqScaled, order)
	}
	diff := p.Sub(reconstructed, order)
	if !diff.IsZero() {
		t.Fatalf("reconstruction mismatch for %s: diff=%v", expr, mpolyToNode(diff, vars))
	}
}

func TestDecomposeSOS_AffineAndBoundaryCases(t *testing.T) {
	cases := []struct {
		name string
		expr string
		vars []string
	}{
		{"UnivariateInteriorHalfNewton", "(1/4)*x^8 - x^7 + 2*x^5 + x^4", []string{"x"}},
		{"BivariateCoupledAffine", "(x^2 + y)^2 + (x + 1)^2", []string{"x", "y"}},
		{"UnivariateLPRequired", "(x^2 - 1)^2 + (x^2 + x)^2", []string{"x"}},
		{"BivariateSignSymmetryBlocks", "(x^2 - y^2)^2 + (x*y)^2", []string{"x", "y"}},
		{"BivariateBoundaryRankDeficient", "(x*y - 1)^2 + (x - 1)^2*(y + 1)^2", []string{"x", "y"}},
		{"BivariateDenseQuadraticBasis", "(x^2 + x*y + y^2 - 1)^2 + (x - y)^2", []string{"x", "y"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertSOSCertifiedAndExact(t, tc.expr, tc.vars)
		})
	}
}

func TestDecomposeSOS_NonSOSPSDCounterforms(t *testing.T) {
	// Motzkin and Choi-Lam polynomials are nonnegative on R^n but not SOS.
	// Engine must return SOSNoCertificateFound without false certification or false refutation.
	cases := []struct {
		expr string
		vars []string
	}{
		{"x^4*y^2 + x^2*y^4 - 3*x^2*y^2 + 1", []string{"x", "y"}},
		{"x^4*y^2 + y^4*z^2 + z^4*x^2 - 3*x^2*y^2*z^2", []string{"x", "y", "z"}},
	}
	order := OrderGRevLex
	for _, tc := range cases {
		varMap := make(map[string]int, len(tc.vars))
		for i, v := range tc.vars {
			varMap[v] = i
		}
		p, err := nodeToMPoly(mustParseNode(t, tc.expr), varMap, len(tc.vars), order)
		if err != nil {
			t.Fatalf("nodeToMPoly(%s) error: %v", tc.expr, err)
		}
		res, err := DecomposeSOS(p, tc.vars, order)
		if err != nil {
			t.Fatalf("DecomposeSOS(%s) error: %v", tc.expr, err)
		}
		if res.Status != SOSNoCertificateFound {
			t.Fatalf("expected SOSNoCertificateFound for non-SOS PSD %s, got %v", tc.expr, res.Status)
		}
	}
}

func TestDecomposeSOS_MetamorphicScalingAndShiftInvariance(t *testing.T) {
	// Deductive & Metamorphic invariant:
	// Multiplying a certified SOS by a positive rational alpha > 0, or shifting variables x -> x + 1,
	// preserves SOS certifiability and zero reconstruction residual.
	assertSOSCertifiedAndExact(t, "(7/3)*((x^2 + y)^2 + (x + 1)^2)", []string{"x", "y"})
	assertSOSCertifiedAndExact(t, "((x + 1)^2 + (y - 2))^2 + ((x + 1) + 1)^2", []string{"x", "y"})
	// Positive perturbation of Motzkin (making it strictly SOS with sufficient pure power padding)
	assertSOSCertifiedAndExact(t, "x^4*y^2 + x^2*y^4 - 3*x^2*y^2 + 1 + 2*(x^6 + y^6 + 1)", []string{"x", "y"})
}

func TestVerifyAndLeanSOSDecompose(t *testing.T) {
	expr := "sos_decompose((1/4)*x^8 - x^7 + 2*x^5 + x^4)"
	node := mustParseNode(t, expr)
	env := NewEnv()
	res, err := EvalWithEnv(node, env)
	if err != nil {
		t.Fatalf("EvalWithEnv error: %v", err)
	}
	cert, err := VerifyComputation(node, res, env)
	if err != nil {
		t.Fatalf("VerifyComputation error: %v", err)
	}
	if cert == nil || !cert.IsVerified {
		t.Fatalf("expected verified SOS certificate, got %+v", cert)
	}
	leanCode, err := GenerateLeanSource("sos_quartic_test", cert, node, res)
	if err != nil {
		t.Fatalf("GenerateLeanSource error: %v", err)
	}
	if !strings.Contains(leanCode, "positivity") || !strings.Contains(leanCode, "by ring") {
		t.Fatalf("unexpected Lean proof code:\n%s", leanCode)
	}

	// Non-SOS PSD polynomial must NOT be certified and must NOT produce a Lean positivity certificate.
	motzExpr := "sos_decompose(x^4*y^2 + x^2*y^4 - 3*x^2*y^2 + 1)"
	motzNode := mustParseNode(t, motzExpr)
	motzRes, err := EvalWithEnv(motzNode, env)
	if err != nil {
		t.Fatalf("EvalWithEnv(motzkin) error: %v", err)
	}
	motzCert, err := VerifyComputation(motzNode, motzRes, env)
	if err != nil {
		t.Fatalf("VerifyComputation(motzkin) error: %v", err)
	}
	if motzCert != nil && motzCert.IsVerified {
		t.Fatalf("Motzkin polynomial must never yield IsVerified certificate")
	}
}
