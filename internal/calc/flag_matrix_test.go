package calc_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
)

// =========================================================================
// Flag Matrix Test Suite
// Verifies 30 representative CAS domain expressions against:
// 1. Evaluation (Base Engine)
// 2. --verify (Asymmetric Verifier / Proof Certificates / Skeptic fail-safe)
// 3. --lean (Lean 4 Proof Transpiler / Fail-safe unsupported checks)
// 4. --ascii (ASCII / Pretty formatter)
// =========================================================================

func TestFlagMatrix_30Domains(t *testing.T) {
	testMatrix := []struct {
		name        string
		expr        string
		allowUnver  bool // whether this domain is expected to return [UNVERIFIED: ...] safely
		allowNoLean bool // whether this domain is expected to return lean unsupported error safely
	}{
		// 1. Polynomial Factorization
		{name: "Factor Quadratic", expr: "factor(x^2 - 1)", allowUnver: false, allowNoLean: false},
		{name: "Factor Cubic", expr: "factor(x^3 - 1)", allowUnver: false, allowNoLean: false},
		{name: "Factor Quartic", expr: "factor(x^4 - 16)", allowUnver: false, allowNoLean: false},

		// 2. Differential & Integral Calculus
		{name: "Indefinite Integral Poly", expr: "integrate(x^3 + 2*x, x)", allowUnver: false, allowNoLean: false},
		{name: "Indefinite Integral Trig", expr: "integrate(sin(x), x)", allowUnver: false, allowNoLean: false},
		{name: "Differentiation Poly", expr: "diff(x^4 + 3*x^2, x)", allowUnver: true, allowNoLean: true},
		{name: "Differentiation Trig", expr: "diff(cos(2*x), x)", allowUnver: true, allowNoLean: true},

		// 3. Limits & Asymptotics
		{name: "Limit Rational", expr: "limit((x^2 - 1)/(x - 1), x, 1)", allowUnver: true, allowNoLean: true},
		{name: "Limit Trig", expr: "limit(sin(x)/x, x, 0)", allowUnver: true, allowNoLean: true},

		// 4. Linear Algebra & Matrix Decompositions
		{name: "Matrix Inverse 2x2", expr: "inv([[1, 2], [3, 4]])", allowUnver: false, allowNoLean: false},
		{name: "Matrix Det 3x3", expr: "det([[1, 2, 3], [0, 1, 4], [5, 6, 0]])", allowUnver: true, allowNoLean: true},
		{name: "Matrix LU", expr: "lu([[4, 3], [6, 3]])", allowUnver: true, allowNoLean: true},
		{name: "Matrix QR", expr: "qr([[12, -51], [6, 167]])", allowUnver: true, allowNoLean: true},
		{name: "Matrix SNF", expr: "snf([[2, 4], [6, 8]])", allowUnver: false, allowNoLean: false},
		{name: "Matrix HNF", expr: "hnf([[2, 4], [6, 8]])", allowUnver: false, allowNoLean: false},

		// 5. Ordinary Differential Equations (ODE)
		{name: "ODE Linear 1st", expr: "dsolve(diff(y, x) + y == 0, y, x)", allowUnver: true, allowNoLean: true},
		{name: "ODE Harmonic", expr: "dsolve(diff(diff(y, x), x) + 4*y == 0, y, x)", allowUnver: true, allowNoLean: true},

		// 6. Cylindrical Algebraic Decomposition (CAD) & QE
		{name: "CAD 1D Linear", expr: "cad([x - 3], [x])", allowUnver: true, allowNoLean: true},
		{name: "CAD 1D Quadratic", expr: "cad([x^2 - 4], [x])", allowUnver: true, allowNoLean: true},
		{name: "QE Forall Exists Cubic", expr: "qe(forall([x], exists([y], y^3 + x*y + a == 0)))", allowUnver: true, allowNoLean: true},
		{name: "QE Exists Circle", expr: "qe(exists([x, y], x^2 + y^2 < 1))", allowUnver: true, allowNoLean: true},

		// 7. Diophantine Equations
		{name: "Diophantine Pell D=2", expr: "solve_diophantine(x^2 - 2*y^2 == 1, [x, y])", allowUnver: true, allowNoLean: true},
		{name: "Diophantine NegPell D=5", expr: "solve_diophantine(x^2 - 5*y^2 == -1, [x, y])", allowUnver: true, allowNoLean: true},
		{name: "Diophantine Linear", expr: "solve_diophantine(3*x + 5*y == 1, [x, y])", allowUnver: true, allowNoLean: true},

		// 8. Number Theory & Prime
		{name: "Prime Factor Integer", expr: "factor(360)", allowUnver: true, allowNoLean: true},
		{name: "Deterministic IsPrime", expr: "is_prime(104729)", allowUnver: true, allowNoLean: true},

		// 9. Polynomial Solvers & Galois Theory
		{name: "Solve Cubic Cardano", expr: "solve(x^3 - 3*x + 1 == 0, x)", allowUnver: true, allowNoLean: true},
		{name: "Is Solvable By Radicals Abel-Ruffini", expr: "is_solvable_by_radicals(x^5 - 4*x - 2, x)", allowUnver: false, allowNoLean: false},
		{name: "Galois Group", expr: "galois_group(x^3 - 2, x)", allowUnver: true, allowNoLean: true},

		// 10. Automated Geometric Theorem Proving (Wu's Method)
		{name: "Geo Prove Midpoint", expr: "geo_prove([midpoint(M, A, B), midpoint(N, A, C)], parallel(M, N, B, C))", allowUnver: false, allowNoLean: false},
	}

	for idx, tc := range testMatrix {
		t.Run(fmt.Sprintf("%02d_%s", idx+1, tc.name), func(t *testing.T) {
			env := calc.NewEnv()

			// 1. Base Evaluation
			parsed, err := calc.Parse(tc.expr)
			if err != nil {
				t.Fatalf("[%s] parse failed: %v", tc.expr, err)
			}
			res, err := calc.EvalWithEnv(parsed, env)
			if err != nil {
				t.Fatalf("[%s] eval failed: %v", tc.expr, err)
			}
			if res == nil {
				t.Fatalf("[%s] eval returned nil node", tc.expr)
			}

			// 2. ASCII & Pretty Formatting
			asciiStr := calc.FormatWithOptions(res, calc.FormatOptions{AsciiOnly: true})
			if strings.TrimSpace(asciiStr) == "" {
				t.Errorf("[%s] ascii formatting returned empty string", tc.expr)
			}
			prettyStr := calc.FormatWithOptions(res, calc.FormatOptions{AsciiOnly: false})
			if strings.TrimSpace(prettyStr) == "" {
				t.Errorf("[%s] pretty formatting returned empty string", tc.expr)
			}

			// 3. Panic-free Execution Guarantee
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("[%s] panic encountered during matrix testing: %v", tc.expr, r)
				}
			}()

			// 4. --verify (Verification Engine & Governance)
			cert, err := calc.VerifyComputation(parsed, res, env)
			if err != nil {
				t.Fatalf("[%s] VerifyComputation returned error: %v", tc.expr, err)
			}
			if cert == nil {
				t.Fatalf("[%s] VerifyComputation returned nil certificate", tc.expr)
			}

			if !tc.allowUnver {
				if !cert.IsVerified {
					t.Errorf("[%s] expected verified certificate, but got unverified: state=%s, details=%s",
						tc.expr, cert.State.String(), cert.Details)
				}
			} else {
				// Must either be verified, or safely unverified (no panic, no circular fake verify)
				if !cert.IsVerified && cert.State != calc.VerifyStateUnsupportedDomain && cert.State != calc.VerifyStateRefuted {
					t.Errorf("[%s] expected certified or safe unsupported/refuted state, got: %s", tc.expr, cert.State.String())
				}
			}

			// 5. --lean (Lean 4 Transpiler)
			if cert.IsVerified {
				leanCode, leanErr := calc.GenerateLeanSource("test_matrix_thm", cert, parsed, res)
				if !tc.allowNoLean {
					if leanErr != nil {
						t.Errorf("[%s] GenerateLeanSource failed: %v", tc.expr, leanErr)
					}
					if !strings.Contains(leanCode, "theorem") {
						t.Errorf("[%s] generated Lean 4 code missing 'theorem':\n%s", tc.expr, leanCode)
					}
				} else {
					if leanErr != nil {
						t.Logf("[%s] safely rejected by Lean 4 transpiler: %v", tc.expr, leanErr)
					}
				}
			}
		})
	}
}
