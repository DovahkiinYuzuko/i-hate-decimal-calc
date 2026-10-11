package calc

import (
	"math/big"
	"strings"
	"testing"
)

func TestEllipticDescentFSM(t *testing.T) {
	fsm := NewEllipticDescentLifecycleFSM()
	if fsm.CurrentState() != StateDescentInit {
		t.Fatalf("expected StateDescentInit, got %v", fsm.CurrentState())
	}

	// Valid sequential transitions
	validSteps := []EcDescentState{
		StateDescentNormalized,
		StateDescentTwoTorsionFactored,
		StateDescentIsogenyConstructed,
		StateDescentDivisorsEnumerated,
		StateDescentLocallyFiltered,
		StateDescentSelmerSolved,
		StateDescentPointsSearched,
		StateDescentCompleted,
	}

	for _, step := range validSteps {
		if err := fsm.TransitionTo(step); err != nil {
			t.Fatalf("unexpected error transitioning to %v: %v", step, err)
		}
	}

	// Invalid backward transition
	if err := fsm.TransitionTo(StateDescentInit); err == nil {
		t.Fatalf("expected error on backward transition from Completed to Init")
	}
}

func TestFindTwoTorsionRationalRoots(t *testing.T) {
	tests := []struct {
		name      string
		a         int64
		b         int64
		expected  []int64
		expectErr bool
	}{
		{
			name:     "Split cubic: y^2 = x^3 - x",
			a:        -1,
			b:        0,
			expected: []int64{-1, 0, 1},
		},
		{
			name:     "Congruent 5: y^2 = x^3 - 25x",
			a:        -25,
			b:        0,
			expected: []int64{-5, 0, 5},
		},
		{
			name:     "One root: y^2 = x^3 - 2x",
			a:        -2,
			b:        0,
			expected: []int64{0},
		},
		{
			name:     "One root: y^2 = x^3 + x",
			a:        1,
			b:        0,
			expected: []int64{0},
		},
		{
			name:     "No rational root: y^2 = x^3 + x + 1",
			a:        1,
			b:        1,
			expected: []int64{},
		},
		{
			name:      "Singular curve: y^2 = x^3 - 3x + 2",
			a:         -3,
			b:         2,
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			aRat := big.NewRat(tc.a, 1)
			bRat := big.NewRat(tc.b, 1)
			roots, err := FindTwoTorsionRationalRoots(aRat, bRat)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for singular curve, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(roots) != len(tc.expected) {
				t.Fatalf("expected %d roots, got %d (%v)", len(tc.expected), len(roots), roots)
			}
			for i, exp := range tc.expected {
				expRat := big.NewRat(exp, 1)
				if roots[i].Cmp(expRat) != 0 {
					t.Errorf("root[%d]: expected %s, got %s", i, expRat.RatString(), roots[i].RatString())
				}
			}
		})
	}
}

func TestBuildDualIsogenousPair(t *testing.T) {
	// y^2 = x^3 - 25x with root x0 = 0
	aRat := big.NewRat(-25, 1)
	rootZero := big.NewRat(0, 1)

	c, d, cPrime, dPrime, _, err := BuildDualIsogenousPair(aRat, rootZero)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// For root x0 = 0:
	// c = 0, d = -25
	// c' = 0, d' = c^2 - 4d = 0 - 4(-25) = 100
	if c.Cmp(big.NewInt(0)) != 0 || d.Cmp(big.NewInt(-25)) != 0 {
		t.Fatalf("expected (c=0, d=-25), got (c=%s, d=%s)", c.String(), d.String())
	}
	if cPrime.Cmp(big.NewInt(0)) != 0 || dPrime.Cmp(big.NewInt(100)) != 0 {
		t.Fatalf("expected (cPrime=0, dPrime=100), got (cPrime=%s, dPrime=%s)", cPrime.String(), dPrime.String())
	}

	// y^2 = x^3 - 25x with root x0 = -5
	rootNeg5 := big.NewRat(-5, 1)
	c2, d2, cPrime2, dPrime2, _, err := BuildDualIsogenousPair(aRat, rootNeg5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// x0 = -5:
	// c = 3(-5) = -15
	// d = 3(-5)^2 - 25 = 75 - 25 = 50
	// c' = -2(-15) = 30
	// d' = (-15)^2 - 4(50) = 225 - 200 = 25
	if c2.Cmp(big.NewInt(-15)) != 0 || d2.Cmp(big.NewInt(50)) != 0 {
		t.Fatalf("expected (c=-15, d=50), got (c=%s, d=%s)", c2.String(), d2.String())
	}
	if cPrime2.Cmp(big.NewInt(30)) != 0 || dPrime2.Cmp(big.NewInt(25)) != 0 {
		t.Fatalf("expected (cPrime=30, dPrime=25), got (cPrime=%s, dPrime=%s)", cPrime2.String(), dPrime2.String())
	}
}

func TestIsQuarticRealSoluble(t *testing.T) {
	tests := []struct {
		name     string
		d1, c, d2 int64
		expected bool
	}{
		{"d1 > 0: positive leading", 1, -10, -5, true},
		{"d2 > 0: positive constant", -5, -10, 1, true},
		{"d1 < 0, d2 < 0, but vertex positive in R+", -1, 3, -1, true},
		{"d1 < 0, d2 < 0, negative vertex: no real solution", -1, -3, -1, false},
		{"d1 < 0, d2 < 0, negative discriminant: no real solution", -2, 1, -2, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := IsQuarticRealSoluble(big.NewInt(tc.d1), big.NewInt(tc.c), big.NewInt(tc.d2))
			if res != tc.expected {
				t.Fatalf("expected %v, got %v for (d1=%d, c=%d, d2=%d)", tc.expected, res, tc.d1, tc.c, tc.d2)
			}
		})
	}
}

func TestIsQuarticTwoAdicSoluble(t *testing.T) {
	// N^2 = M^4 - 25 e^4 (d1 = 1, c = 0, d2 = -25)
	// (M=1, e=0): N^2 = 1 => soluble in Q_2!
	if !IsQuarticTwoAdicSoluble(big.NewInt(1), big.NewInt(0), big.NewInt(-25)) {
		t.Fatalf("expected M^4 - 25e^4 to be 2-adically soluble")
	}

	// N^2 = -M^4 - e^4 (d1 = -1, c = 0, d2 = -1)
	// For any (M, e) in Z_2 not both even, -M^4 - e^4 <= -1 < 0, but over Q_2, -1 is not a square (neither mod 4 nor mod 8).
	// In fact -M^4 - e^4 can never be 1 mod 8 or square.
	if IsQuarticTwoAdicSoluble(big.NewInt(-1), big.NewInt(0), big.NewInt(-1)) {
		t.Fatalf("expected -M^4 - e^4 to be 2-adically insoluble")
	}

	// N^2 = 2 M^4 + 3 e^4 (d1 = 2, c = 0, d2 = 3)
	// If e is odd: 2 M^4 + 3 e^4 mod 8. If M is even: 0 + 3 = 3 mod 8 (not square).
	// If M is odd: 2 + 3 = 5 mod 8 (not square).
	// If e is even (e=2k), M must be odd: 2(1) + 48 k^4 = 2 mod 4 (v_2 = 1 is odd, not square).
	// So 2 M^4 + 3 e^4 has NO 2-adic solution!
	if IsQuarticTwoAdicSoluble(big.NewInt(2), big.NewInt(0), big.NewInt(3)) {
		t.Fatalf("expected 2 M^4 + 3 e^4 to be 2-adically insoluble")
	}
}

func TestIsQuarticOddPrimeSoluble(t *testing.T) {
	// p = 5: N^2 = M^4 + 2 e^4
	// (M=1, e=0) => N^2 = 1 => soluble at p=5
	p5 := big.NewInt(5)
	if !IsQuarticOddPrimeSoluble(big.NewInt(1), big.NewInt(0), big.NewInt(2), p5) {
		t.Fatalf("expected M^4 + 2e^4 to be 5-adically soluble")
	}

	// p = 3: N^2 = 2 M^4 + 2 e^4
	// Mod 3: M, e in {0, 1, 2}.
	// If e=0, M=1: 2 mod 3 (2 is not a square mod 3: 1^2=1, 2^2=1).
	// If e=1, M=0: 2 mod 3 (not a square).
	// If e=1, M=1: 2+2 = 4 = 1 mod 3! (1 is a square mod 3, derivative is non-zero). Soluble!
	p3 := big.NewInt(3)
	if !IsQuarticOddPrimeSoluble(big.NewInt(2), big.NewInt(0), big.NewInt(2), p3) {
		t.Fatalf("expected 2 M^4 + 2 e^4 to be 3-adically soluble via (1, 1)")
	}

	// p = 3: N^2 = 3 M^4 + 3 e^4
	// Both coefficients divisible by 3.
	// F(M,e) = 3(M^4 + e^4). Valuation v_3(F) is 1 (odd) unless M^4 + e^4 = 0 mod 3.
	// But in F_3, M^4 + e^4 = M^2 + e^2 can only be 0 if M=0, e=0.
	// So v_3(F) is always 1 (odd). Cannot be a square in Q_3!
	if IsQuarticOddPrimeSoluble(big.NewInt(3), big.NewInt(0), big.NewInt(3), p3) {
		t.Fatalf("expected 3 M^4 + 3 e^4 to be 3-adically insoluble")
	}
}

func TestComputeTwoDescentRankBound(t *testing.T) {
	tests := []struct {
		name          string
		a, b          int64
		expectedUpper int
		expectExact   bool
		expectErr     bool
	}{
		{"Congruent 1: y^2 = x^3 - x", -1, 0, 0, true, false},
		{"Congruent 2: y^2 = x^3 - 4x", -4, 0, 0, true, false},
		{"Congruent 3: y^2 = x^3 - 9x", -9, 0, 0, true, false},
		{"Congruent 5: y^2 = x^3 - 25x", -25, 0, 1, false, false},
		{"Congruent 6: y^2 = x^3 - 36x", -36, 0, 1, false, false},
		{"One root rank 1: y^2 = x^3 - 2x", -2, 0, 1, false, false},
		{"One root rank 0: y^2 = x^3 + x", 1, 0, 0, true, false},
		{"Sha[2] non-trivial trap: y^2 = x^3 - 289x (n=17)", -289, 0, 2, false, false},
		{"Irreducible cubic (Case C): y^2 = x^3 + x + 1", 1, 1, 0, false, true},
		{"Singular curve: y^2 = x^3 - 3x + 2", -3, 2, 0, false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			aRat := big.NewRat(tc.a, 1)
			bRat := big.NewRat(tc.b, 1)
			res, err := ComputeTwoDescentRankBound(aRat, bRat, nil)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.RankUpperBound != tc.expectedUpper {
				t.Fatalf("expected RankUpperBound = %d, got %d (SelmerPhiDim=%d, SelmerPsiDim=%d)",
					tc.expectedUpper, res.RankUpperBound, res.SelmerPhiDim, res.SelmerPsiDim)
			}
		})
	}
}

func TestTwoDescentGenerators(t *testing.T) {
	tests := []struct {
		name          string
		a, b          int64
		expectedRank  int
		expectedExact bool
	}{
		{"Rank 1: y^2 = x^3 - 25x (n=5)", -25, 0, 1, true},
		{"Rank 1: y^2 = x^3 - 36x (n=6)", -36, 0, 1, true},
		{"Rank 1: y^2 = x^3 - 2x (one root)", -2, 0, 1, true},
		{"Rank 0: y^2 = x^3 - x (n=1)", -1, 0, 0, true},
		{"Rank 0: y^2 = x^3 + x (one root)", 1, 0, 0, true},
		{"Sha[2] trap: y^2 = x^3 - 289x (n=17)", -289, 0, 0, false},
		{"Metamorphic scaled: y^2 = x^3 - 400x", -400, 0, 1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			aRat := big.NewRat(tc.a, 1)
			bRat := big.NewRat(tc.b, 1)
			res, err := ComputeTwoDescentRankBound(aRat, bRat, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.IsExact != tc.expectedExact {
				t.Errorf("IsExact: expected %v, got %v", tc.expectedExact, res.IsExact)
			}
			if res.RankLowerBound != tc.expectedRank {
				t.Errorf("RankLowerBound: expected %d, got %d", tc.expectedRank, res.RankLowerBound)
			}

			// Verify all returned generators lie exactly on curve and have infinite order
			for i, gen := range res.Generators {
				if gen.IsInf {
					t.Errorf("generator[%d] cannot be infinity", i)
				}
				if !IsOnCurve(gen, aRat, bRat) {
					t.Errorf("generator[%d] %s does not lie on curve y^2 = x^3 + (%d)x + (%d)", i, gen.String(), tc.a, tc.b)
				}

				// Verify infinite order by Mazur test: m*P != O for m = 1..12
				for m := int64(1); m <= 12; m++ {
					mP, err := ScalarMulEllipticPoint(gen, big.NewInt(m), aRat, bRat)
					if err != nil {
						t.Fatalf("scalar mul error: %v", err)
					}
					if mP.IsInf {
						t.Errorf("generator[%d] %s has finite torsion order dividing %d", i, gen.String(), m)
					}
				}
			}
		})
	}
}

func TestEvalEcRankBoundAndGenerators(t *testing.T) {
	env := NewEnv()

	// Rank 1 test: ec_rank_bound(-25, 0)
	nodeRank, err := Parse("ec_rank_bound(-25, 0)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	resRank, err := EvalWithEnv(nodeRank, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	listRank, ok := resRank.(*ListNode)
	if !ok || len(listRank.Elements) != 3 {
		t.Fatalf("expected ListNode with 3 elements, got %s", resRank.String())
	}
	if listRank.Elements[0].String() != "1" || listRank.Elements[1].String() != "1" || listRank.Elements[2].String() != "true" {
		t.Errorf("expected [1, 1, true], got %s", resRank.String())
	}

	// Generator test: ec_generators(-25, 0)
	nodeGens, err := Parse("ec_generators(-25, 0)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	resGens, err := EvalWithEnv(nodeGens, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	listGens, ok := resGens.(*ListNode)
	if !ok || len(listGens.Elements) != 1 {
		t.Fatalf("expected 1 generator, got %s", resGens.String())
	}

	// Sha[2] trap test: ec_rank_bound(-289, 0)
	nodeSha, err := Parse("ec_rank_bound(-289, 0)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	resSha, err := EvalWithEnv(nodeSha, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	listSha, ok := resSha.(*ListNode)
	if !ok || len(listSha.Elements) != 3 {
		t.Fatalf("expected ListNode with 3 elements, got %s", resSha.String())
	}
	if listSha.Elements[0].String() != "0" || listSha.Elements[1].String() != "2" || listSha.Elements[2].String() != "false" {
		t.Errorf("expected [0, 2, false], got %s", resSha.String())
	}
}

func TestEllipticDescentVerify(t *testing.T) {
	env := NewEnv()

	// Verify ec_rank_bound
	exprRank, err := Parse("ec_rank_bound(-25, 0)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	resRank, err := EvalWithEnv(exprRank, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	certRank, err := VerifyComputation(exprRank, resRank, env)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !certRank.IsVerified {
		t.Fatalf("expected certified, got %v", certRank)
	}
	if certRank.Domain != DomainElliptic {
		t.Errorf("expected DomainElliptic, got %s", certRank.Domain)
	}

	// Verify ec_generators
	exprGens, err := Parse("ec_generators(-25, 0)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	resGens, err := EvalWithEnv(exprGens, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	certGens, err := VerifyComputation(exprGens, resGens, env)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !certGens.IsVerified {
		t.Fatalf("expected certified, got %v", certGens)
	}

	// Test Lean 4 generation from certificate
	leanCode, err := GenerateLeanSource("ihd_elliptic_generator_proof", certGens, exprGens, resGens)
	if err != nil {
		t.Fatalf("GenerateLeanSource failed: %v", err)
	}
	if !strings.Contains(leanCode, "theorem ihd_elliptic_generator_proof") {
		t.Errorf("missing theorem name in lean code: %s", leanCode)
	}
	if !strings.Contains(leanCode, ": ℚ)^2 = (") {
		t.Errorf("missing Weierstrass equation in lean code: %s", leanCode)
	}
}

func TestEllipticDescentTrace(t *testing.T) {
	sink := NewMemoryTraceSink()
	WithTraceSink(sink, func() {
		_, err := EvalString("ec_rank_bound(-36, 0)")
		if err != nil {
			t.Fatalf("eval failed: %v", err)
		}
	})

	events := sink.Events()
	foundDescent := false
	for _, e := range events {
		if e.Rule == RuleEllipticDescent {
			foundDescent = true
			break
		}
	}
	if !foundDescent {
		t.Errorf("expected trace event for RuleEllipticDescent, but not found among %d events", len(events))
	}
}

func TestComputeTwoDescentDiscriminants(t *testing.T) {
	// y^2 = x^3 + 13x^2 + 128x
	// d = 128, c = 13
	// d' = 13^2 - 4(128) = 169 - 512 = -343
	d := big.NewInt(128)
	dPrime := big.NewInt(-343)

	deltaE, deltaEPrime := ComputeTwoDescentDiscriminants(d, dPrime)

	// Delta(E) = 16 * d^2 * d' = 16 * 128^2 * (-343) = -89915392
	expectedDeltaE := big.NewInt(-89915392)
	if deltaE.Cmp(expectedDeltaE) != 0 {
		t.Errorf("deltaE: expected %s, got %s", expectedDeltaE.String(), deltaE.String())
	}

	// Delta(E') = 256 * dPrime^2 * d = 256 * (-343)^2 * 128 = 3855122432
	expectedDeltaEPrime := big.NewInt(3855122432)
	if deltaEPrime.Cmp(expectedDeltaEPrime) != 0 {
		t.Errorf("deltaEPrime: expected %s, got %s", expectedDeltaEPrime.String(), deltaEPrime.String())
	}
}

func TestDualIsogenousPair_SingularGuard(t *testing.T) {
	// y^2 = x^3 - 3x + 2: roots 1 (double), -2 (single)
	// For root x0 = 1: x -> x + 1 => y^2 = (x+1)^3 - 3(x+1) + 2 = x^3 + 3x^2
	// Here d = 0, so shifted curve is singular!
	root1 := big.NewRat(1, 1)
	aRatForCubic := big.NewRat(-3, 1)
	_, _, _, _, _, err := BuildDualIsogenousPair(aRatForCubic, root1)
	if err == nil {
		t.Fatalf("expected error for singular shifted curve d=0, got nil")
	}

	// Direct singular dual curve d' = 0 test
	d := big.NewInt(1)
	dPrime := big.NewInt(0)
	deltaE, deltaEPrime := ComputeTwoDescentDiscriminants(d, dPrime)
	if deltaE.Sign() != 0 || deltaEPrime.Sign() != 0 {
		t.Errorf("expected both discriminants 0 when d'=0, got %s, %s", deltaE.String(), deltaEPrime.String())
	}
}

func TestMapDualPointToCurve_FormulaAndBranches(t *testing.T) {
	// E: y^2 = x^3 - 25x (c=0, d=-25)
	// E': Y^2 = X^3 + 100X (c'=0, d'=100)
	// P = (-4, 6) on E
	// 2P = (1681/144, -62279/1728)
	// phi(P) = (9/4, -123/8) on E'
	q := NewEllipticPoint(big.NewRat(9, 4), big.NewRat(-123, 8))
	dPrime := big.NewInt(100)

	mapped, err := MapDualPointToCurve(q, dPrime)
	if err != nil {
		t.Fatalf("unexpected error in MapDualPointToCurve: %v", err)
	}

	// Double P = (-4, 6) on E: y^2 = x^3 - 25x (A = -25, B = 0)
	aRat := big.NewRat(-25, 1)
	bRat := big.NewRat(0, 1)
	p := NewEllipticPoint(big.NewRat(-4, 1), big.NewRat(6, 1))
	twoP, err := ScalarMulEllipticPoint(p, big.NewInt(2), aRat, bRat)
	if err != nil {
		t.Fatalf("ScalarMulEllipticPoint failed: %v", err)
	}

	// Verify mapped point equals 2P (both X and Y must match!)
	if mapped.X.Cmp(twoP.X) != 0 {
		t.Errorf("mapped.X: expected %s, got %s", twoP.X.RatString(), mapped.X.RatString())
	}
	// Note: in our engine, points are typically stored with Y >= 0 or tested up to sign,
	// but mapDualPointToCurve should match 2P coordinate-wise.
	// In Go engine, points on curve satisfy y^2 = f(x).
	if mapped.Y.Cmp(twoP.Y) != 0 {
		// Test if opposite sign or exact
		negY := new(big.Rat).Neg(twoP.Y)
		if mapped.Y.Cmp(negY) != 0 {
			t.Errorf("mapped.Y: expected %s or %s, got %s", twoP.Y.RatString(), negY.RatString(), mapped.Y.RatString())
		}
	}

	// Branch tests:
	// Q = (0, 0)' should map to Infinity
	qZero := NewEllipticPoint(big.NewRat(0, 1), big.NewRat(0, 1))
	mappedZero, err := MapDualPointToCurve(qZero, dPrime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mappedZero.IsInf {
		t.Errorf("expected InfinityPoint for Q=(0,0), got %v", mappedZero)
	}

	// Q = Infinity should map to Infinity
	mappedInf, err := MapDualPointToCurve(InfinityPoint(), dPrime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mappedInf.IsInf {
		t.Errorf("expected InfinityPoint for Q=Inf, got %v", mappedInf)
	}
}

func TestIsPointTorsion_MazurOrders(t *testing.T) {
	// y^2 = x^3 - 25x (A = -25, B = 0)
	aRat := big.NewRat(-25, 1)
	bRat := big.NewRat(0, 1)

	// (0, 0) is 2-torsion point
	pt2 := NewEllipticPoint(big.NewRat(0, 1), big.NewRat(0, 1))
	if !IsPointTorsion(pt2, aRat, bRat) {
		t.Errorf("expected (0, 0) to be identified as torsion")
	}

	// (25/4, 75/8) is generator of infinite order
	ptGen := NewEllipticPoint(big.NewRat(25, 4), big.NewRat(75, 8))
	if IsPointTorsion(ptGen, aRat, bRat) {
		t.Errorf("expected (25/4, 75/8) to be identified as infinite order (NOT torsion)")
	}
}

func TestQuarticSolvability_TwoAdicHenselAndSturmReal(t *testing.T) {
	// 1. Real solvability tests
	realCases := []struct {
		d1, c, d2 int64
		expected  bool
	}{
		{-1, 3, -1, true},   // Delta = 9 - 4 = 5 > 0, c = 3 > 0
		{-1, 2, -1, true},   // Delta = 4 - 4 = 0, c = 2 > 0 -> root t=1 gives F=0, soluble
		{-1, 1, -1, false},  // Delta = 1 - 4 = -3 < 0
		{-1, -3, -1, false}, // c = -3 <= 0, no positive vertex
		{1, -100, -5, true}, // d1 > 0 -> e=0 boundary point (1, 0) gives F=1 > 0
		{-5, -100, 1, true}, // d2 > 0 -> M=0 gives F=1 > 0
	}

	for _, rc := range realCases {
		res := IsQuarticRealSoluble(big.NewInt(rc.d1), big.NewInt(rc.c), big.NewInt(rc.d2))
		if res != rc.expected {
			t.Errorf("IsQuarticRealSoluble(%d, %d, %d): expected %v, got %v", rc.d1, rc.c, rc.d2, rc.expected, res)
		}
	}

	// 2. 2-adic solvability tests
	twoAdicCases := []struct {
		name      string
		d1, c, d2 int64
		expected  bool
	}{
		{"Square leading: M^4 - 25e^4", 1, 0, -25, true},
		{"Insoluble: 2M^4 + 3e^4", 2, 0, 3, false},
		{"Insoluble negative: -M^4 - e^4", -1, 0, -1, false},
		{"Units 17 mod 8: 17M^4 + 17e^4", 17, 0, 17, true},
		{"Integer solution (-2)^4 + 32 = 16: -M^4 + 32e^4", -1, 0, 32, true},
		{"Insoluble mod 8: 3M^4 + 5e^4", 3, 0, 5, false},
		{"Insoluble mod 8: 3M^4 + 3e^4", 3, 0, 3, false},
	}

	for _, tc := range twoAdicCases {
		res := IsQuarticTwoAdicSoluble(big.NewInt(tc.d1), big.NewInt(tc.c), big.NewInt(tc.d2))
		if res != tc.expected {
			t.Errorf("IsQuarticTwoAdicSoluble(%s): expected %v, got %v", tc.name, tc.expected, res)
		}
	}
}

func TestTwoDescentRankIntervalAndShaResilience(t *testing.T) {
	// SHA-CONGRUENT-17: y^2 = x^3 - 289x (congruent number n=17)
	// Has rank 0, but Selmer rank 2 due to non-trivial Sha[2] = Z/2Z x Z/2Z!
	aRat := big.NewRat(-289, 1)
	bRat := big.NewRat(0, 1)

	res, err := ComputeTwoDescentRankBound(aRat, bRat, nil)
	if err != nil {
		t.Fatalf("ComputeTwoDescentRankBound failed: %v", err)
	}

	// Must certify interval [0, 2] and flag sha_2_possible
	if res.RankLowerBound != 0 {
		t.Errorf("expected RankLowerBound = 0, got %d", res.RankLowerBound)
	}
	if res.RankUpperBound != 2 {
		t.Errorf("expected RankUpperBound = 2, got %d", res.RankUpperBound)
	}
	if res.IsExact {
		t.Errorf("expected IsExact = false for interval [0, 2], got true")
	}
	if !res.Sha2Possible {
		t.Errorf("expected Sha2Possible = true for [0, 2] interval gap, got false")
	}
	if len(res.Coverings) == 0 {
		t.Errorf("expected Coverings to be tracked in TwoDescentResult")
	}

	// Check CLI response: ec_rank_bound(-289, 0)
	node, err := EvalString("ec_rank_bound(-289, 0)")
	if err != nil {
		t.Fatalf("eval ec_rank_bound failed: %v", err)
	}
	listNode, ok := node.(*ListNode)
	if !ok || len(listNode.Elements) < 3 {
		t.Fatalf("expected ListNode with at least 3 elements, got %v", node)
	}
	lowerRat, ok1 := listNode.Elements[0].(*RationalNode)
	upperRat, ok2 := listNode.Elements[1].(*RationalNode)
	exactVar, ok3 := listNode.Elements[2].(*VarNode)
	if !ok1 || !ok2 || !ok3 {
		t.Fatalf("unexpected node types in ec_rank_bound result: %v", listNode.Elements)
	}
	if lowerRat.Val.Num().Int64() != 0 || upperRat.Val.Num().Int64() != 2 || exactVar.Name != "false" {
		t.Errorf("expected [0, 2, false], got [%v, %v, %v]", lowerRat.Val, upperRat.Val, exactVar.Name)
	}
}








