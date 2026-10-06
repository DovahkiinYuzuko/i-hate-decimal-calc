package calc

import (
	"math/big"
	"path/filepath"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func TestProofTrace_Basic(t *testing.T) {
	orig := &ast.VarNode{Name: "x"}
	res := &ast.RationalNode{Val: big.NewRat(1, 1)}

	trace := NewProofTrace(orig, res)
	if trace == nil {
		t.Fatalf("expected non-nil trace")
	}
	if trace.StepCount() != 0 {
		t.Fatalf("expected 0 steps, got %d", trace.StepCount())
	}
	if trace.LastStep() != nil {
		t.Fatalf("expected nil last step on empty trace")
	}

	step1 := ProofStep{
		Before:          orig,
		After:           res,
		Rule:            "IdentityReduction",
		RuleDescription: "Reduce x to 1",
		Residual:        &ast.RationalNode{Val: big.NewRat(0, 1)},
		IsSelfVerified:  true,
	}
	trace.AddStep(step1)

	if trace.StepCount() != 1 {
		t.Fatalf("expected 1 step, got %d", trace.StepCount())
	}
	last := trace.LastStep()
	if last == nil || last.Rule != "IdentityReduction" {
		t.Fatalf("unexpected last step: %+v", last)
	}
	if !last.IsSelfVerified {
		t.Errorf("expected IsSelfVerified true")
	}

	str := step1.String()
	if str == "" || !containsSubstr(str, "IdentityReduction") || !containsSubstr(str, "[VERIFIED]") {
		t.Errorf("unexpected step string: %s", str)
	}
}

func TestConvertCertificateToProofTrace_Identity(t *testing.T) {
	// (x - 1)*(x + 1) == x^2 - 1
	lhs := &ast.MulNode{
		Factors: []ast.Node{
			&ast.AddNode{Terms: []ast.Node{&ast.VarNode{Name: "x"}, &ast.RationalNode{Val: big.NewRat(-1, 1)}}},
			&ast.AddNode{Terms: []ast.Node{&ast.VarNode{Name: "x"}, &ast.RationalNode{Val: big.NewRat(1, 1)}}},
		},
	}
	rhs := &ast.AddNode{
		Terms: []ast.Node{
			&ast.PowNode{Base: &ast.VarNode{Name: "x"}, Exp: &ast.RationalNode{Val: big.NewRat(2, 1)}},
			&ast.RationalNode{Val: big.NewRat(-1, 1)},
		},
	}
	res := &ast.RationalNode{Val: big.NewRat(0, 1)}

	cert := NewIdentityCertificate(IdentityKindFactor, lhs, rhs, res, true, "Difference of squares factor")
	trace, err := ConvertCertificateToProofTrace(cert)
	if err != nil {
		t.Fatalf("unexpected error converting identity cert: %v", err)
	}
	if trace.StepCount() != 1 {
		t.Fatalf("expected 1 step, got %d", trace.StepCount())
	}
	step := trace.Steps[0]
	if step.Rule != "PolynomialFactorization" {
		t.Errorf("expected rule PolynomialFactorization, got %s", step.Rule)
	}
	if !step.IsSelfVerified {
		t.Errorf("expected step to be verified")
	}
	if trace.Metadata["kind"] != string(IdentityKindFactor) {
		t.Errorf("expected metadata kind to be factor, got %s", trace.Metadata["kind"])
	}
}

func TestConvertCertificateToProofTrace_Deriv(t *testing.T) {
	integrand := &ast.VarNode{Name: "x"}
	antideriv := &ast.MulNode{
		Factors: []ast.Node{
			&ast.RationalNode{Val: big.NewRat(1, 2)},
			&ast.PowNode{Base: &ast.VarNode{Name: "x"}, Exp: &ast.RationalNode{Val: big.NewRat(2, 1)}},
		},
	}
	res := &ast.RationalNode{Val: big.NewRat(0, 1)}

	cert := NewDerivCertificate(integrand, antideriv, res, "x", true, "Power rule antiderivative")
	trace, err := ConvertCertificateToProofTrace(cert)
	if err != nil {
		t.Fatalf("unexpected error converting deriv cert: %v", err)
	}
	if trace.StepCount() != 1 {
		t.Fatalf("expected 1 step, got %d", trace.StepCount())
	}
	step := trace.Steps[0]
	if step.Rule != "IndefiniteIntegration" {
		t.Errorf("expected rule IndefiniteIntegration, got %s", step.Rule)
	}
	if trace.Metadata["variable"] != "x" {
		t.Errorf("expected metadata variable 'x', got %s", trace.Metadata["variable"])
	}
}

func TestConvertCertificateToProofTrace_Invertibility(t *testing.T) {
	mat := &ast.MatrixNode{
		Data: [][]ast.Node{
			{&ast.RationalNode{Val: big.NewRat(1, 1)}, &ast.RationalNode{Val: big.NewRat(0, 1)}},
			{&ast.RationalNode{Val: big.NewRat(0, 1)}, &ast.RationalNode{Val: big.NewRat(1, 1)}},
		},
	}
	det := &ast.RationalNode{Val: big.NewRat(1, 1)}
	cert := NewInvertibilityCertificate(mat, mat, det, &ast.RationalNode{Val: big.NewRat(0, 1)}, true, "Identity matrix self-inverse")
	trace, err := ConvertCertificateToProofTrace(cert)
	if err != nil {
		t.Fatalf("unexpected error converting matrix cert: %v", err)
	}
	if trace.StepCount() != 1 {
		t.Fatalf("expected 1 step, got %d", trace.StepCount())
	}
	if trace.Steps[0].Rule != "MatrixInversion" {
		t.Errorf("expected rule MatrixInversion, got %s", trace.Steps[0].Rule)
	}
}

func TestConvertCertificateToProofTrace_Nil(t *testing.T) {
	_, err := ConvertCertificateToProofTrace(nil)
	if err == nil {
		t.Errorf("expected error on nil certificate")
	}
}

func TestVerifyProofTrace_Success(t *testing.T) {
	env := NewEnv()
	// (x - 1)*(x + 1) == x^2 - 1
	p1, _ := Parse("(x - 1) * (x + 1)")
	p2, _ := Parse("x^2 - 1")

	trace := NewProofTrace(p1, p2)
	trace.AddStep(ProofStep{
		Before:          p1,
		After:           p2,
		Rule:            "PolynomialExpansion",
		RuleDescription: "Difference of squares",
	})

	ok, err := VerifyProofTrace(trace, env)
	if err != nil {
		t.Fatalf("unexpected verification error: %v", err)
	}
	if !ok {
		t.Fatalf("expected verification to succeed")
	}
	if !trace.Steps[0].IsSelfVerified {
		t.Errorf("expected step to be marked as self-verified")
	}
}

func TestVerifyProofTrace_Failure(t *testing.T) {
	env := NewEnv()
	p1, _ := Parse("x + 1")
	p2, _ := Parse("x + 2") // Not equal!

	trace := NewProofTrace(p1, p2)
	trace.AddStep(ProofStep{
		Before:          p1,
		After:           p2,
		Rule:            "BogusRule",
		RuleDescription: "Invalid transformation",
	})

	ok, err := VerifyProofTrace(trace, env)
	if err == nil {
		t.Errorf("expected error on invalid transformation, got nil")
	}
	if ok {
		t.Errorf("expected verification to fail")
	}
}

func TestProofTrace_RenderExplain(t *testing.T) {
	env := NewEnv()
	p1, _ := Parse("x^2 - 1")
	p2, _ := Parse("(x - 1)*(x + 1)")

	trace := NewProofTrace(p1, p2)
	trace.AddStep(ProofStep{
		Before:          p1,
		After:           p2,
		Rule:            "Factorization",
		RuleDescription: "Difference of squares",
	})

	_, err := VerifyProofTrace(trace, env)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	explainJa := trace.RenderExplain("ja")
	if !containsSubstr(explainJa, "検証済") {
		t.Errorf("expected Japanese explain to contain '検証済', got:\n%s", explainJa)
	}

	explainEn := trace.RenderExplain("en")
	if !containsSubstr(explainEn, "Verified") {
		t.Errorf("expected English explain to contain 'Verified', got:\n%s", explainEn)
	}
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && len(sub) > 0 && (s[:len(sub)] == sub || containsSubstr(s[1:], sub))))
}

func TestProofTrace_JSONSerialization(t *testing.T) {
	orig, _ := Parse("x^2 - 1")
	res, _ := Parse("(x - 1)*(x + 1)")

	trace := NewProofTrace(orig, res)
	trace.AddStep(ProofStep{
		Before:          orig,
		After:           res,
		Rule:            "Factorization",
		RuleDescription: "Difference of squares",
		Residual:        &ast.RationalNode{Val: big.NewRat(0, 1)},
		IsSelfVerified:  true,
	})

	jsonData, err := trace.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	jsonStr := string(jsonData)
	if !containsSubstr(jsonStr, "\"version\": \"1.0\"") {
		t.Errorf("expected version 1.0, got: %s", jsonStr)
	}
	if !containsSubstr(jsonStr, "\"is_verified\": true") {
		t.Errorf("expected is_verified true, got: %s", jsonStr)
	}
	if !containsSubstr(jsonStr, "Factorization") {
		t.Errorf("expected rule Factorization, got: %s", jsonStr)
	}
}

func TestProofTrace_SaveAndLoadJSON(t *testing.T) {
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "test_cert.json")

	orig, _ := Parse("x^2 - 1")
	res, _ := Parse("(x - 1)*(x + 1)")

	trace := NewProofTrace(orig, res)
	trace.Metadata["domain"] = "algebra"
	trace.AddStep(ProofStep{
		Before:          orig,
		After:           res,
		Rule:            "Factorization",
		RuleDescription: "Difference of squares",
		Residual:        &ast.RationalNode{Val: big.NewRat(0, 1)},
		IsSelfVerified:  true,
	})

	if err := SaveProofTraceJSON(trace, certPath); err != nil {
		t.Fatalf("SaveProofTraceJSON failed: %v", err)
	}

	loadedTrace, err := LoadProofTraceFromJSON(certPath)
	if err != nil {
		t.Fatalf("LoadProofTraceFromJSON failed: %v", err)
	}

	if loadedTrace.StepCount() != 1 {
		t.Fatalf("expected 1 step, got %d", loadedTrace.StepCount())
	}
	if loadedTrace.Metadata["domain"] != "algebra" {
		t.Errorf("expected metadata domain 'algebra', got %s", loadedTrace.Metadata["domain"])
	}
	step := loadedTrace.Steps[0]
	if step.Rule != "Factorization" {
		t.Errorf("expected rule Factorization, got %s", step.Rule)
	}
	if !step.IsSelfVerified {
		t.Errorf("expected IsSelfVerified true")
	}
}


