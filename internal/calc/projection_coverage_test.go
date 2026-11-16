package calc_test

import (
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

func TestProjectionCoverage_Slice1_InvModAndCRT(t *testing.T) {
	cases := []struct {
		name         string
		expr         string
		wantRules    []string
		wantLeanSubs []string
	}{
		{
			name:         "inv_mod concrete",
			expr:         "inv_mod(3, 11)",
			wantRules:    []string{provenance.RuleNTBezout, provenance.RuleNTInvMod},
			wantLeanSubs: []string{"norm_num", "#print axioms"},
		},
		{
			name:         "crt coprime",
			expr:         "crt([2, 3, 2], [3, 5, 7])",
			wantRules:    []string{provenance.RuleNTCRT},
			wantLeanSubs: []string{"norm_num", "#print axioms"},
		},
		{
			name:         "crt compatible non-coprime",
			expr:         "crt([1, 3], [4, 6])",
			wantRules:    []string{provenance.RuleNTCRT},
			wantLeanSubs: []string{"norm_num", "#print axioms"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := calc.NewEnv()
			tracer := provenance.NewTraceRecorder()
			env.Tracer = tracer

			parsed, err := calc.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			res, err := calc.EvalWithEnv(parsed, env)
			if err != nil {
				t.Fatalf("EvalWithEnv failed: %v", err)
			}

			root := tracer.Root()
			if root == nil {
				t.Fatalf("expected non-nil provenance root for %s", tc.expr)
			}

			// 1. --explain
			explainStr := provenance.NewExplainVisitor().RenderTree(root, tc.expr)
			if strings.TrimSpace(explainStr) == "" {
				t.Errorf("ExplainVisitor returned empty output")
			}

			// 2. --typst
			typstStr := provenance.NewTypstVisitor().RenderDocument(root, tc.expr)
			if !strings.Contains(typstStr, "Derivation Steps") || !strings.Contains(typstStr, "$") {
				t.Errorf("TypstVisitor output unexpected: %s", typstStr)
			}

			// 3. --verify
			cert, err := calc.VerifyComputation(parsed, res, env)
			if err != nil {
				t.Fatalf("VerifyComputation failed: %v", err)
			}
			if !cert.IsVerified {
				t.Fatalf("expected IsVerified=true, got false: %s", cert.Details)
			}

			// 4. --lean (certificate & calc)
			leanCertCode, err := calc.GenerateLeanSource("test_cert", cert, parsed, res)
			if err != nil {
				t.Fatalf("GenerateLeanSource failed: %v", err)
			}
			for _, sub := range tc.wantLeanSubs {
				if !strings.Contains(leanCertCode, sub) {
					t.Errorf("GenerateLeanSource missing %q:\n%s", sub, leanCertCode)
				}
			}
			if strings.Contains(leanCertCode, "sorry") || strings.Contains(leanCertCode, "native_decide") {
				t.Errorf("GenerateLeanSource contains forbidden tactic:\n%s", leanCertCode)
			}

			leanCalcCode, err := provenance.NewLeanCalcVisitor().RenderCalc(root, "test_calc")
			if err != nil {
				t.Fatalf("LeanCalcVisitor.RenderCalc failed: %v", err)
			}
			if strings.Contains(leanCalcCode, "sorry") || strings.Contains(leanCalcCode, "native_decide") {
				t.Errorf("LeanCalcVisitor.RenderCalc contains forbidden tactic:\n%s", leanCalcCode)
			}
			for _, rule := range tc.wantRules {
				jsonBytes, _ := provenance.NewExplainVisitor().RenderJSON(root, tc.expr)
				if !strings.Contains(string(jsonBytes), rule) {
					t.Errorf("expected rule %q in provenance JSON:\n%s", rule, string(jsonBytes))
				}
			}
		})
	}
}

func TestProjectionCoverage_Slice2_ApartAndDiophantine(t *testing.T) {
	cases := []struct {
		name         string
		expr         string
		wantRules    []string
		wantLeanSubs []string
	}{
		{
			name:         "apart distinct linear",
			expr:         "apart(1 / (x^2 - 1), x)",
			wantRules:    []string{provenance.RuleAlgebraApartPoly, provenance.RuleAlgebraApartRational},
			wantLeanSubs: []string{"field_simp", "ring"},
		},
		{
			name:         "apart improper fraction",
			expr:         "apart((x^2 + 1) / (x - 1), x)",
			wantRules:    []string{provenance.RuleAlgebraApartPoly, provenance.RuleAlgebraApartRational},
			wantLeanSubs: []string{"field_simp", "ring"},
		},
		{
			name:         "apart repeated linear factor",
			expr:         "apart(1 / ((x - 1)^2 * (x + 1)), x)",
			wantRules:    []string{provenance.RuleAlgebraApartPoly, provenance.RuleAlgebraApartRational},
			wantLeanSubs: []string{"field_simp", "ring"},
		},
		{
			name:         "diophantine linear",
			expr:         "solve_diophantine(3*x + 5*y == 1, [x, y])",
			wantRules:    []string{provenance.RuleDiophantineLinear},
			wantLeanSubs: []string{"intro", "ring"},
		},
		{
			name:         "diophantine pell D=2",
			expr:         "solve_diophantine(x^2 - 2*y^2 == 1, [x, y])",
			wantRules:    []string{provenance.RuleDiophantinePell},
			wantLeanSubs: []string{"norm_num"},
		},
		{
			name:         "diophantine neg pell D=5",
			expr:         "solve_diophantine(x^2 - 5*y^2 == -1, [x, y])",
			wantRules:    []string{provenance.RuleDiophantinePell},
			wantLeanSubs: []string{"norm_num"},
		},
		{
			name:         "diophantine pythagorean",
			expr:         "solve_diophantine(x^2 + y^2 == z^2, [x, y, z])",
			wantRules:    []string{provenance.RuleDiophantinePythagorean},
			wantLeanSubs: []string{"ring"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := calc.NewEnv()
			tracer := provenance.NewTraceRecorder()
			env.Tracer = tracer

			parsed, err := calc.Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			res, err := calc.EvalWithEnv(parsed, env)
			if err != nil {
				t.Fatalf("EvalWithEnv failed: %v", err)
			}

			root := tracer.Root()
			if root == nil {
				t.Fatalf("expected non-nil provenance root for %s", tc.expr)
			}

			explainStr := provenance.NewExplainVisitor().RenderTree(root, tc.expr)
			if strings.TrimSpace(explainStr) == "" {
				t.Errorf("ExplainVisitor returned empty output")
			}

			typstStr := provenance.NewTypstVisitor().RenderDocument(root, tc.expr)
			if !strings.Contains(typstStr, "Derivation Steps") {
				t.Errorf("TypstVisitor output unexpected: %s", typstStr)
			}

			cert, err := calc.VerifyComputation(parsed, res, env)
			if err != nil {
				t.Fatalf("VerifyComputation failed: %v", err)
			}
			if !cert.IsVerified {
				t.Fatalf("expected IsVerified=true, got false: %s", cert.Details)
			}

			leanCertCode, err := calc.GenerateLeanSource("test_cert", cert, parsed, res)
			if err != nil {
				t.Fatalf("GenerateLeanSource failed: %v", err)
			}
			if strings.Contains(leanCertCode, "sorry") || strings.Contains(leanCertCode, "native_decide") {
				t.Errorf("GenerateLeanSource contains forbidden tactic:\n%s", leanCertCode)
			}

			leanCalcCode, err := provenance.NewLeanCalcVisitor().RenderCalc(root, "test_calc")
			if err != nil {
				t.Fatalf("LeanCalcVisitor.RenderCalc failed: %v", err)
			}
			for _, sub := range tc.wantLeanSubs {
				if !strings.Contains(leanCalcCode, sub) {
					t.Errorf("LeanCalcVisitor missing %q:\n%s", sub, leanCalcCode)
				}
			}
			if strings.Contains(leanCalcCode, "sorry") || strings.Contains(leanCalcCode, "native_decide") {
				t.Errorf("LeanCalcVisitor contains forbidden tactic:\n%s", leanCalcCode)
			}

			jsonBytes, _ := provenance.NewExplainVisitor().RenderJSON(root, tc.expr)
			for _, rule := range tc.wantRules {
				if !strings.Contains(string(jsonBytes), rule) {
					t.Errorf("expected rule %q in provenance JSON:\n%s", rule, string(jsonBytes))
				}
			}
		})
	}
}

func TestProjectionCoverage_NegativeAndRefutation(t *testing.T) {
	unsolvableCases := []string{
		"inv_mod(2, 4)",
		"crt([1, 2], [4, 6])",
		"solve_diophantine(6*x + 9*y == 4, [x, y])",
	}
	for _, expr := range unsolvableCases {
		t.Run("unsolvable/"+expr, func(t *testing.T) {
			parsed, err := calc.Parse(expr)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			env := calc.NewEnv()
			env.Tracer = provenance.NewTraceRecorder()
			res, evalErr := calc.EvalWithEnv(parsed, env)
			if evalErr == nil {
				t.Errorf("expected error for unsolvable expression %q, got %v", expr, res)
			}
		})
	}

	refutationCases := []struct {
		name  string
		expr  string
		wrong string
	}{
		{"inv_mod wrong", "inv_mod(3, 11)", "5"},
		{"crt wrong", "crt([2, 3, 2], [3, 5, 7])", "24"},
		{"apart wrong", "apart(1/(x^2 - 1), x)", "1/(x - 1)"},
		{"diophantine linear wrong", "solve_diophantine(3*x + 5*y == 1, [x, y])", "[x == 0, y == 0]"},
		{"diophantine linear non-integer", "solve_diophantine(3*x + 5*y == 1, [x, y])", "[x == 1/3, y == 0]"},
		{"diophantine pell wrong", "solve_diophantine(x^2 - 2*y^2 == 1, [x, y])", "[x == 2, y == 2]"},
		{"diophantine pell non-integer", "solve_diophantine(x^2 - 2*y^2 == 1, [x, y])", "[x == -9/7, y == -4/7]"},
	}
	for _, tc := range refutationCases {
		t.Run("refute/"+tc.name, func(t *testing.T) {
			parsed, err := calc.Parse(tc.expr)
			if err != nil {
				t.Fatalf("parse expr failed: %v", err)
			}
			wrongNode, err := calc.Parse(tc.wrong)
			if err != nil {
				t.Fatalf("parse wrong failed: %v", err)
			}
			cert, err := calc.VerifyComputation(parsed, wrongNode, nil)
			if err != nil {
				return
			}
			if cert != nil && cert.IsVerified {
				t.Errorf("expected VerifyComputation to reject wrong result %s for %s, but got IsVerified=true (%s)", tc.wrong, tc.expr, cert.Details)
			}
		})
	}
}

func TestProjectionCoverage_LeanSyntaxPowMod(t *testing.T) {
	negPowNode, err := calc.Parse("(x - 1)^(-1)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	negPowLean, err := provenance.ToLeanSyntax(negPowNode)
	if err != nil {
		t.Fatalf("ToLeanSyntax negative power failed: %v", err)
	}
	if !strings.Contains(negPowLean, "(-1 : ℤ)") || !strings.HasPrefix(negPowLean, "(x -") {
		t.Errorf("expected parenthesized base and integer cast in negative power Lean syntax, got: %s", negPowLean)
	}

	modNode, err := calc.Parse("mod(17, 5)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	modLean, err := provenance.ToLeanSyntax(modNode)
	if err != nil {
		t.Fatalf("ToLeanSyntax mod failed: %v", err)
	}
	if !strings.Contains(modLean, "%") || !strings.Contains(modLean, ": ℤ") {
		t.Errorf("expected Lean integer modulo syntax for mod(17, 5), got: %s", modLean)
	}

	// Verify Lean transpiler handles unevaluated (x - 1)^(-1) and apart original denominator NonZero binder
	parsedApart, err := calc.Parse("apart(1/(x^2 - 1), x)")
	if err != nil {
		t.Fatalf("parse apart failed: %v", err)
	}
	env := calc.NewEnv()
	tracer := provenance.NewTraceRecorder()
	env.Tracer = tracer
	_, err = calc.EvalWithEnv(parsedApart, env)
	if err != nil {
		t.Fatalf("eval apart failed: %v", err)
	}
	leanCalc, err := provenance.NewLeanCalcVisitor().RenderCalc(tracer.Root(), "apart_nonzero_test")
	if err != nil {
		t.Fatalf("RenderCalc failed: %v", err)
	}
	if !strings.Contains(leanCalc, "x ^ 2") {
		t.Errorf("expected original denominator NonZero assumption (x^2 - 1 != 0) in Lean calc:\n%s", leanCalc)
	}

	// Verify CRT unevaluated arg expressions and no double crt(crt(...)) equation string
	parsedCRT, err := calc.Parse("crt([1 + 1, 3], [3, 5])")
	if err != nil {
		t.Fatalf("parse crt failed: %v", err)
	}
	resCRT, err := calc.Eval(parsedCRT)
	if err != nil {
		t.Fatalf("eval crt failed: %v", err)
	}
	certCRT, err := calc.VerifyComputation(parsedCRT, resCRT, nil)
	if err != nil || !certCRT.IsVerified {
		t.Fatalf("VerifyComputation for unevaluated CRT args failed: cert=%v err=%v", certCRT, err)
	}
	if strings.Contains(certCRT.Equation, "crt(crt(") {
		t.Errorf("CRT certificate equation duplicated crt(): %s", certCRT.Equation)
	}
}
