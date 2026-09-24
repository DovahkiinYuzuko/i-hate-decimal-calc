package calc

import (
	"testing"
)

func TestEvalGaloisGroupAndSolvability(t *testing.T) {
	tests := []struct {
		inputExpr    string
		expectedGrp  string
		expectedSolv string
	}{
		// Degree 3
		{"x^3 - 2", "S3", "true"},
		{"x^3 - 3*x + 1", "A3", "true"},
		// Degree 4
		{"x^4 - 2", "D4", "true"},
		{"x^4 + x^3 + x^2 + x + 1", "C4", "true"},
		{"x^4 - 10*x^2 + 1", "V4", "true"},
		// Degree 5 - Solvable
		{"x^5 - 2", "F20", "true"},
		{"x^5 - 5*x + 12", "D5", "true"},
		// Degree 5 - Non-Solvable (Abel-Ruffini)
		{"x^5 - 4*x + 2", "S5", "false"},
		{"x^5 - x - 1", "S5", "false"},
		{"x^5 + 20*x + 16", "A5", "false"},
	}

	for _, tc := range tests {
		t.Run(tc.inputExpr, func(t *testing.T) {
			env := NewEnv()
			// 1. Test galois_group
			grpCmd := "galois_group(" + tc.inputExpr + ", x)"
			astNode, err := Parse(grpCmd)
			if err != nil {
				t.Fatalf("Parse failed for %s: %v", grpCmd, err)
			}
			resNode, err := EvalWithEnv(astNode, env)
			if err != nil {
				t.Fatalf("Eval failed for %s: %v", grpCmd, err)
			}
			if resNode.String() != tc.expectedGrp {
				t.Errorf("galois_group(%s) = %s; want %s", tc.inputExpr, resNode.String(), tc.expectedGrp)
			}

			// 2. Test is_solvable_by_radicals
			solvCmd := "is_solvable_by_radicals(" + tc.inputExpr + ", x)"
			astSolv, err := Parse(solvCmd)
			if err != nil {
				t.Fatalf("Parse failed for %s: %v", solvCmd, err)
			}
			resSolv, err := EvalWithEnv(astSolv, env)
			if err != nil {
				t.Fatalf("Eval failed for %s: %v", solvCmd, err)
			}
			if resSolv.String() != tc.expectedSolv {
				t.Errorf("is_solvable_by_radicals(%s) = %s; want %s", tc.inputExpr, resSolv.String(), tc.expectedSolv)
			}

			// 3. Verify Abel-Ruffini certificate when not solvable
			if tc.expectedSolv == "false" {
				if env.LastCert == nil {
					t.Fatalf("expected Abel-Ruffini certificate in env.LastCert, got nil")
				}
				if env.LastCert.Domain != DomainImpossibility {
					t.Errorf("expected DomainImpossibility, got %s", env.LastCert.Domain)
				}
				if !env.LastCert.IsVerified {
					t.Errorf("expected certificate to be verified")
				}
			}
		})
	}
}
