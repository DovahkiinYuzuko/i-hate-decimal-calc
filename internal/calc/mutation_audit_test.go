package calc

import (
	"testing"
)

// IdentityTestCase represents an equation to verify along with its expected validity.
type IdentityTestCase struct {
	Name        string
	LHS         Node
	RHS         Node
	ShouldHold  bool
	Description string
}

func makePow(base Node, exp int64) Node {
	p, err := NewPow(base, mustRational(exp, 1))
	if err != nil {
		panic(err)
	}
	return p
}

// GetGoldenVerificationCorpus returns frozen positive (valid) and planted negative (invalid) identities.
func GetGoldenVerificationCorpus() []IdentityTestCase {
	return []IdentityTestCase{
		// Positive cases (Must always PASS)
		{
			Name: "DifferenceOfSquares",
			LHS: NewMul([]Node{
				NewAdd([]Node{NewVar("x"), mustRational(-1, 1)}),
				NewAdd([]Node{NewVar("x"), mustRational(1, 1)}),
			}),
			RHS: NewAdd([]Node{
				makePow(NewVar("x"), 2),
				mustRational(-1, 1),
			}),
			ShouldHold:  true,
			Description: "(x - 1)*(x + 1) == x^2 - 1",
		},
		{
			Name: "PerfectCube",
			LHS:  makePow(NewAdd([]Node{NewVar("x"), mustRational(1, 1)}), 3),
			RHS: NewAdd([]Node{
				makePow(NewVar("x"), 3),
				NewMul([]Node{mustRational(3, 1), makePow(NewVar("x"), 2)}),
				NewMul([]Node{mustRational(3, 1), NewVar("x")}),
				mustRational(1, 1),
			}),
			ShouldHold:  true,
			Description: "(x + 1)^3 == x^3 + 3*x^2 + 3*x + 1",
		},
		{
			Name: "PolynomialCubicDifference",
			LHS: NewMul([]Node{
				NewAdd([]Node{NewVar("x"), mustRational(-1, 1)}),
				NewAdd([]Node{makePow(NewVar("x"), 2), NewVar("x"), mustRational(1, 1)}),
			}),
			RHS: NewAdd([]Node{
				makePow(NewVar("x"), 3),
				mustRational(-1, 1),
			}),
			ShouldHold:  true,
			Description: "(x - 1)*(x^2 + x + 1) == x^3 - 1",
		},

		// Negative cases (Planted Invalids - Must always REFUTE)
		{
			Name: "SignFlipInSquare",
			LHS:  makePow(NewAdd([]Node{NewVar("x"), mustRational(-1, 1)}), 2),
			RHS: NewAdd([]Node{
				makePow(NewVar("x"), 2),
				mustRational(-1, 1),
			}),
			ShouldHold:  false,
			Description: "(x - 1)^2 == x^2 - 1 (Sign mutant)",
		},
		{
			Name: "DroppedLinearTermInSquare",
			LHS:  makePow(NewAdd([]Node{NewVar("x"), mustRational(1, 1)}), 2),
			RHS: NewAdd([]Node{
				makePow(NewVar("x"), 2),
				mustRational(1, 1),
			}),
			ShouldHold:  false,
			Description: "(x + 1)^2 == x^2 + 1 (Dropped cross term)",
		},
		{
			Name: "WrongDerivativeCoeff",
			LHS:  NewMul([]Node{mustRational(3, 1), makePow(NewVar("x"), 2)}),
			RHS:  NewMul([]Node{mustRational(2, 1), makePow(NewVar("x"), 2)}),
			ShouldHold:  false,
			Description: "3*x^2 == 2*x^2 (Coefficient mutation)",
		},
		{
			Name: "OffByOneFactorConstant",
			LHS: NewMul([]Node{
				NewAdd([]Node{NewVar("x"), mustRational(-1, 1)}),
				NewAdd([]Node{NewVar("x"), mustRational(2, 1)}),
			}),
			RHS: NewAdd([]Node{
				makePow(NewVar("x"), 2),
				mustRational(-1, 1),
			}),
			ShouldHold:  false,
			Description: "(x - 1)*(x + 2) == x^2 - 1 (Constant mutation)",
		},
	}
}

// TestGoldenVerificationCorpus audits all positive and negative identities against the verifier.
func TestGoldenVerificationCorpus(t *testing.T) {
	env := NewEnv()
	corpus := GetGoldenVerificationCorpus()

	for _, tc := range corpus {
		t.Run(tc.Name, func(t *testing.T) {
			cert, err := VerifyAlgebraicEquivalence(tc.LHS, tc.RHS, env)
			if err != nil {
				t.Fatalf("unexpected execution error for %s: %v", tc.Description, err)
			}

			if tc.ShouldHold {
				if !cert.IsVerified {
					t.Errorf("expected valid identity to HOLD, but REFUTED: %s (Details: %s)", tc.Description, cert.Details)
				}
				if cert.State != VerifyStateCertified {
					t.Errorf("expected state Certified, got %v", cert.State)
				}
			} else {
				if cert.IsVerified {
					t.Errorf("SOUNDNESS BUG: invalid identity accepted as valid! %s", tc.Description)
				}
				if cert.State != VerifyStateRefuted {
					t.Errorf("expected state Refuted for invalid identity, got %v", cert.State)
				}
				if cert.Witness == nil {
					t.Errorf("expected counterexample witness for refuted identity: %s", tc.Description)
				}
			}
		})
	}
}

// TestMutationAudit_SimulatedMutants verifies that 100% of injected non-equivalent mutants are killed.
func TestMutationAudit_SimulatedMutants(t *testing.T) {
	env := NewEnv()

	mutants := []struct {
		name       string
		f          func() (Node, Node)
		shouldKill bool
	}{
		{
			name: "Mutant_FlipAddSub",
			f: func() (Node, Node) {
				lhs := NewMul([]Node{
					NewAdd([]Node{NewVar("x"), mustRational(2, 1)}),
					NewAdd([]Node{NewVar("x"), mustRational(2, 1)}),
				})
				rhs := NewAdd([]Node{
					makePow(NewVar("x"), 2),
					mustRational(-4, 1),
				})
				return lhs, rhs
			},
			shouldKill: true,
		},
		{
			name: "Mutant_CoefficientPerturbation",
			f: func() (Node, Node) {
				lhs := NewMul([]Node{mustRational(5, 1), NewVar("x")})
				rhs := NewMul([]Node{mustRational(6, 1), NewVar("x")})
				return lhs, rhs
			},
			shouldKill: true,
		},
		{
			name: "Mutant_PowerPerturbation",
			f: func() (Node, Node) {
				lhs := makePow(NewVar("x"), 4)
				rhs := makePow(NewVar("x"), 3)
				return lhs, rhs
			},
			shouldKill: true,
		},
	}

	killed := 0
	for _, m := range mutants {
		t.Run(m.name, func(t *testing.T) {
			lhs, rhs := m.f()
			cert, err := VerifyAlgebraicEquivalence(lhs, rhs, env)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if !cert.IsVerified {
				killed++
			} else {
				t.Errorf("Mutant SURVIVED: %s", m.name)
			}
		})
	}

	if killed != len(mutants) {
		t.Errorf("Mutation score: %d/%d (expected 100%% killed)", killed, len(mutants))
	}
}
