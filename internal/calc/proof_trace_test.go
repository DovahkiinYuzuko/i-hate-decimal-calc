package calc

import (
	"fmt"
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

func TestConvertCertificateToProofTrace_IntegerRelation(t *testing.T) {
	el1 := &ast.VarNode{Name: "a"}
	el2 := &ast.VarNode{Name: "b"}
	c1 := &ast.RationalNode{Val: big.NewRat(1, 1)}
	c2 := &ast.RationalNode{Val: big.NewRat(-1, 1)}
	res := &ast.RationalNode{Val: big.NewRat(0, 1)}

	cert := NewIntegerRelationCertificate([]ast.Node{el1, el2}, []ast.Node{c1, c2}, res, true, "Integer relation certified")
	trace, err := ConvertCertificateToProofTrace(cert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trace.StepCount() != 1 {
		t.Fatalf("expected 1 step, got %d", trace.StepCount())
	}
	if trace.Steps[0].Rule != "IntegerRelationLLL" {
		t.Errorf("expected rule IntegerRelationLLL, got %s", trace.Steps[0].Rule)
	}
	if trace.Steps[0].RuleDescription == "" {
		t.Errorf("expected non-empty rule description")
	}
}

func TestConvertCertificateToProofTrace_EllipticPoint(t *testing.T) {
	p1 := &ast.ListNode{Elements: []ast.Node{&ast.RationalNode{Val: big.NewRat(-1, 1)}, &ast.RationalNode{Val: big.NewRat(0, 1)}}}
	p2 := &ast.ListNode{Elements: []ast.Node{&ast.RationalNode{Val: big.NewRat(0, 1)}, &ast.RationalNode{Val: big.NewRat(0, 1)}}}
	sum := &ast.ListNode{Elements: []ast.Node{&ast.RationalNode{Val: big.NewRat(1, 1)}, &ast.RationalNode{Val: big.NewRat(0, 1)}}}
	a := &ast.RationalNode{Val: big.NewRat(-1, 1)}
	b := &ast.RationalNode{Val: big.NewRat(0, 1)}
	res := &ast.RationalNode{Val: big.NewRat(0, 1)}

	cert := NewEllipticPointCertificate(a, b, p1, p2, sum, res, true, "Elliptic addition certified")
	trace, err := ConvertCertificateToProofTrace(cert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trace.StepCount() != 1 {
		t.Fatalf("expected 1 step, got %d", trace.StepCount())
	}
	if trace.Steps[0].Rule != "EllipticCurveAddition" {
		t.Errorf("expected rule EllipticCurveAddition, got %s", trace.Steps[0].Rule)
	}
}

func TestConvertCertificateToProofTrace_PrimitiveElement(t *testing.T) {
	minPolyTheta, _ := Parse("x^4 - 10*x^2 + 1")
	minPolyAlpha, _ := Parse("x^2 - 2")
	minPolyBeta, _ := Parse("x^2 - 3")
	repAlpha, _ := Parse("theta^3/2 - 9/2*theta")
	repBeta, _ := Parse("11/2*theta - theta^3/2")
	quoG, _ := Parse("x^2 + 1")
	quoF, _ := Parse("x^2 - 1")
	res := &ast.RationalNode{Val: big.NewRat(0, 1)}

	cert := NewPrimitiveElementCertificate(minPolyTheta, "theta", 1, repAlpha, repBeta, minPolyAlpha, minPolyBeta, quoG, quoF, res, true, "Primitive element certified")
	trace, err := ConvertCertificateToProofTrace(cert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if trace.StepCount() != 1 {
		t.Fatalf("expected 1 step, got %d", trace.StepCount())
	}
	if trace.Steps[0].Rule != "PrimitiveElementIsomorphism" {
		t.Errorf("expected rule PrimitiveElementIsomorphism, got %s", trace.Steps[0].Rule)
	}
	if trace.Metadata["symbol_theta"] != "theta" {
		t.Errorf("expected symbol_theta 'theta', got %s", trace.Metadata["symbol_theta"])
	}
}

func TestConvertCertificateToProofTrace_AllDomains_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Smith Normal Form
	matA := &ast.MatrixNode{Data: [][]ast.Node{
		{&ast.RationalNode{Val: big.NewRat(2, 1)}, &ast.RationalNode{Val: big.NewRat(4, 1)}},
		{&ast.RationalNode{Val: big.NewRat(4, 1)}, &ast.RationalNode{Val: big.NewRat(2, 1)}},
	}}
	matD := &ast.MatrixNode{Data: [][]ast.Node{
		{&ast.RationalNode{Val: big.NewRat(2, 1)}, &ast.RationalNode{Val: big.NewRat(0, 1)}},
		{&ast.RationalNode{Val: big.NewRat(0, 1)}, &ast.RationalNode{Val: big.NewRat(6, 1)}},
	}}
	snfCert := NewSmithNormalFormCertificate(matA, matA, matA, matD, []Node{&ast.RationalNode{Val: big.NewRat(2, 1)}, &ast.RationalNode{Val: big.NewRat(6, 1)}}, &ast.RationalNode{Val: big.NewRat(0, 1)}, true, "SNF certified")

	// 2. Linear Solve
	target := &ast.ListNode{Elements: []ast.Node{&ast.RationalNode{Val: big.NewRat(5, 1)}, &ast.RationalNode{Val: big.NewRat(11, 1)}}}
	sol := &ast.ListNode{Elements: []ast.Node{&ast.RationalNode{Val: big.NewRat(1, 1)}, &ast.RationalNode{Val: big.NewRat(2, 1)}}}
	solveCert := NewLinearSolveCertificate(matA, sol, target, &ast.RationalNode{Val: big.NewRat(0, 1)}, true, "Linear solve certified")

	// 3. Pell equation
	dNode := &ast.RationalNode{Val: big.NewRat(2, 1)}
	xNode := &ast.RationalNode{Val: big.NewRat(3, 1)}
	yNode := &ast.RationalNode{Val: big.NewRat(2, 1)}
	pellCert := NewPellCertificate(dNode, xNode, yNode, &ast.RationalNode{Val: big.NewRat(0, 1)}, true, "Pell certified")

	// 4. Polynomial Root
	poly, _ := Parse("x^2 - 2")
	root, _ := Parse("sqrt(2)")
	rootCert := NewPolynomialRootCertificate(poly, "x", root, &ast.RationalNode{Val: big.NewRat(0, 1)}, true, "Root certified")

	certs := []Certificate{snfCert, solveCert, pellCert, rootCert}

	for i, c := range certs {
		trace, err := ConvertCertificateToProofTrace(c)
		if err != nil {
			t.Fatalf("[%d] ConvertCertificateToProofTrace failed: %v", i, err)
		}
		path := filepath.Join(tempDir, fmt.Sprintf("cert_%d.json", i))
		if err := SaveProofTraceJSON(trace, path); err != nil {
			t.Fatalf("[%d] SaveProofTraceJSON failed: %v", i, err)
		}
		loaded, err := LoadProofTraceFromJSON(path)
		if err != nil {
			t.Fatalf("[%d] LoadProofTraceFromJSON failed: %v", i, err)
		}
		if loaded.StepCount() != 1 {
			t.Errorf("[%d] expected 1 step after roundtrip, got %d", i, loaded.StepCount())
		}
		if !loaded.Steps[0].IsSelfVerified {
			t.Errorf("[%d] expected step to remain self-verified", i)
		}
	}
}



