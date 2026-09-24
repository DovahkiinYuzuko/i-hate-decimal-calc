package calc

import (
	"math/big"
	"strings"
	"testing"
)

func TestToLeanSyntax_Rational(t *testing.T) {
	// Integer
	r1, _ := NewRational(42, 1)
	s1, err := ToLeanSyntax(r1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s1 != "42" {
		t.Errorf("expected 42, got %s", s1)
	}

	// Negative integer
	r2, _ := NewRational(-7, 1)
	s2, err := ToLeanSyntax(r2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s2 != "(-7)" {
		t.Errorf("expected (-7), got %s", s2)
	}

	// Fraction
	r3, _ := NewRational(3, 5)
	s3, err := ToLeanSyntax(r3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s3 != "((3 : ℚ) / 5)" {
		t.Errorf("expected ((3 : ℚ) / 5), got %s", s3)
	}
}

func TestToLeanSyntax_Polynomial(t *testing.T) {
	// x^2 - 1
	x := NewVar("x")
	two := &RationalNode{Val: big.NewRat(2, 1)}
	pow := &PowNode{Base: x, Exp: two}
	minusOne := &RationalNode{Val: big.NewRat(-1, 1)}
	add := &AddNode{Terms: []Node{pow, minusOne}}

	s, err := ToLeanSyntax(add)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "x ^ 2 - 1"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}

func TestToLeanSyntax_ReservedKeywords(t *testing.T) {
	v := NewVar("theorem")
	s, err := ToLeanSyntax(v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "«theorem»" {
		t.Errorf("expected «theorem», got %s", s)
	}
}

func TestCollectFreeVariables(t *testing.T) {
	x := NewVar("x")
	y := NewVar("y")
	add := &AddNode{Terms: []Node{x, y, x}}

	vars := CollectFreeVariables(add)
	if len(vars) != 2 || vars[0] != "x" || vars[1] != "y" {
		t.Errorf("expected [x y], got %v", vars)
	}
}

func TestTranspileCertificateToLean_Factor(t *testing.T) {
	x := NewVar("x")
	two := &RationalNode{Val: big.NewRat(2, 1)}
	pow := &PowNode{Base: x, Exp: two}
	minusOne := &RationalNode{Val: big.NewRat(-1, 1)}
	lhs := &AddNode{Terms: []Node{pow, minusOne}} // x^2 - 1

	factorCall := &FuncNode{Name: "factor", Args: []Node{lhs}}

	plusOne := &RationalNode{Val: big.NewRat(1, 1)}
	term1 := &AddNode{Terms: []Node{x, minusOne}} // x - 1
	term2 := &AddNode{Terms: []Node{x, plusOne}}  // x + 1
	rhs := &MulNode{Factors: []Node{term1, term2}}

	cert := &VerificationCertificate{
		Domain:     DomainFactor,
		Equation:   "x^2 - 1 = (x - 1)*(x + 1)",
		IsVerified: true,
	}

	code, err := TranspileCertificateToLean(cert, factorCall, rhs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(code, "theorem ihd_verified_proof :") {
		t.Errorf("missing theorem declaration: %s", code)
	}
	if !strings.Contains(code, "by ring") {
		t.Errorf("missing 'by ring' tactic: %s", code)
	}
}

func TestGenerateLeanSource(t *testing.T) {
	x := NewVar("x")
	two := &RationalNode{Val: big.NewRat(2, 1)}
	pow := &PowNode{Base: x, Exp: two}
	minusOne := &RationalNode{Val: big.NewRat(-1, 1)}
	lhs := &AddNode{Terms: []Node{pow, minusOne}}

	plusOne := &RationalNode{Val: big.NewRat(1, 1)}
	term1 := &AddNode{Terms: []Node{x, minusOne}}
	term2 := &AddNode{Terms: []Node{x, plusOne}}
	rhs := &MulNode{Factors: []Node{term1, term2}}

	cert := &VerificationCertificate{
		Domain:     DomainFactor,
		Equation:   "x^2 - 1 = (x - 1)*(x + 1)",
		IsVerified: true,
	}

	src, err := GenerateLeanSource("factor_identity", cert, lhs, rhs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(src, "import Mathlib.Tactic.Ring") {
		t.Errorf("missing Mathlib import: %s", src)
	}
	if !strings.Contains(src, "variable (x : ℚ)") {
		t.Errorf("missing variable declaration: %s", src)
	}
	if !strings.Contains(src, "theorem factor_identity :") {
		t.Errorf("missing custom theorem name: %s", src)
	}
	if !strings.Contains(src, "by ring") {
		t.Errorf("missing tactic: %s", src)
	}
}

func TestTranspileCertificateToLean_UnverifiedError(t *testing.T) {
	cert := &VerificationCertificate{
		Domain:     DomainGeneral,
		IsVerified: false,
		Details:    "residual is non-zero",
	}

	_, err := TranspileCertificateToLean(cert, NewVar("x"), NewVar("y"))
	if err == nil {
		t.Fatal("expected error for unverified certificate, got nil")
	}
}

func TestTranspileCertificateIRToLean_Direct(t *testing.T) {
	// 1. Identity IR
	lhs := NewVar("x")
	rhs := NewVar("x")
	res, _ := NewRational(0, 1)
	idCert := NewIdentityCertificate(IdentityKindFactor, lhs, rhs, res, true, "Holds")
	leanCode, err := TranspileCertificateIRToLean(idCert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(leanCode, "theorem ihd_verified_proof : x = x := by ring") {
		t.Errorf("unexpected Lean output: %s", leanCode)
	}

	// 2. Deriv IR
	f := NewVar("x")
	F := NewVar("F")
	derivCert := NewDerivCertificate(f, F, res, "x", true, "Holds")
	leanDeriv, err := TranspileCertificateIRToLean(derivCert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(leanDeriv, "deriv (fun x => F) x = x := by simp") {
		t.Errorf("unexpected Lean output: %s", leanDeriv)
	}

	// 3. Matrix Invertibility IR
	m := NewVar("A")
	inv := NewVar("B")
	invCert := NewInvertibilityCertificate(m, inv, nil, res, true, "Holds")
	leanInv, err := TranspileCertificateIRToLean(invCert)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(leanInv, "A * B = 1 := by ext <;> ring") {
		t.Errorf("unexpected Lean output: %s", leanInv)
	}
}

func TestGenerateLeanSource_RealAndGeometry(t *testing.T) {
	// 1. Real function typing test
	x := NewVar("x")
	sinX := &FuncNode{Name: "sin", Args: []Node{x}}
	res, _ := NewRational(0, 1)
	derivCert := NewDerivCertificate(sinX, sinX, res, "x", true, "Holds")
	vCert := &VerificationCertificate{
		Domain:     DomainIntegral,
		Equation:   "deriv == sin",
		IsVerified: true,
		CertIR:     derivCert,
	}

	src, err := GenerateLeanSource("real_deriv_proof", vCert, sinX, sinX)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(src, "variable (x : ℝ)") {
		t.Errorf("expected real typing 'variable (x : ℝ)', got: %s", src)
	}
	if !strings.Contains(src, "import Mathlib.Analysis.Calculus.Deriv.Basic") {
		t.Errorf("missing calculus deriv import: %s", src)
	}
	if !strings.Contains(src, "import Mathlib.Analysis.SpecialFunctions.Trigonometric.Basic") {
		t.Errorf("missing trigonometric import: %s", src)
	}

	// 2. Geometry certificate variable extraction test
	ax := NewVar("A_x")
	bx := NewVar("B_x")
	mx := NewVar("M_x")
	two := mustRational(2, 1)
	twoMX, _ := simplifyMul([]Node{two, mx})
	negAX, _ := simplifyUnaryOp("-", ax)
	negBX, _ := simplifyUnaryOp("-", bx)
	hyp, _ := simplifyAdd([]Node{twoMX, negAX, negBX}) // 2*M_x - A_x - B_x
	concl := NewVar("M_x")

	geoCert := NewGeometricCertificate([]Node{hyp}, concl, []Node{hyp}, nil, nil, nil, mustRational(0, 1), true, "Holds")
	vGeoCert := &VerificationCertificate{
		Domain:     DomainGeometry,
		Equation:   "prem == 0",
		IsVerified: true,
		CertIR:     geoCert,
	}

	geoSrc, err := GenerateLeanSource("geo_midpoint", vGeoCert, NewVar("Dummy"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(geoSrc, "A_x") || !strings.Contains(geoSrc, "B_x") {
		t.Errorf("missing coordinate variables in geometric proof: %s", geoSrc)
	}
	if !strings.Contains(geoSrc, "variable (") || !strings.Contains(geoSrc, ": ℚ)") {
		t.Errorf("expected rational variable declaration in geometric proof: %s", geoSrc)
	}

	// 3. Full pipeline test with midpoint theorem
	env := NewEnv()
	parsed, err := ParseStatement("geo_prove([midpoint(M, A, B), midpoint(N, A, C)], parallel(M, N, B, C))")
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	fn := parsed.(*FuncNode)
	resNode, err := EvalGeoProve(fn.Args, env)
	if err != nil || resNode.String() != "true" {
		t.Fatalf("failed to prove geometric theorem: %v", err)
	}
	fullGeoSrc, err := GenerateLeanSource("midpoint_theorem", env.LastCert, fn, resNode)
	if err != nil {
		t.Fatalf("failed to generate lean source: %v", err)
	}
	if !strings.Contains(fullGeoSrc, "theorem midpoint_theorem") {
		t.Errorf("expected theorem midpoint_theorem, got: %s", fullGeoSrc)
	}
	if !strings.Contains(fullGeoSrc, ":= by ring") {
		t.Errorf("expected by ring tactic, got: %s", fullGeoSrc)
	}
	if !strings.Contains(fullGeoSrc, "Mathlib.Analysis.SpecialFunctions.Trigonometric.Deriv") {
		t.Errorf("expected Trigonometric.Deriv import, got: %s", fullGeoSrc)
	}
	if !strings.Contains(fullGeoSrc, "#print axioms midpoint_theorem") {
		t.Errorf("expected #print axioms in output, got: %s", fullGeoSrc)
	}
}
