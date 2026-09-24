package calc

import (
	"fmt"
	"strings"
	"testing"
)

func TestWuContentPrimitive(t *testing.T) {
	// 4*x^2 + 6*x + 8 -> content is 2, prim is 2*x^2 + 3*x + 4
	expr, err := Parse("4*x^2 + 6*x + 8")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	cont, prim, err := ContentPrimitive(expr, "x", []string{"x"})
	if err != nil {
		t.Fatalf("ContentPrimitive failed: %v", err)
	}
	if cont.String() != "2" {
		t.Errorf("expected content 2, got %s", cont.String())
	}
	// verify prim
	diffExpr, err := Parse(fmt.Sprintf("(%s) - (2*x^2 + 3*x + 4)", prim.String()))
	if err != nil {
		t.Fatalf("Parse diffExpr failed: %v", err)
	}
	diff, err := Eval(diffExpr)
	if err != nil {
		t.Fatalf("Eval diff failed: %v", err)
	}
	if !isZero(diff) {
		t.Errorf("expected primitive poly 2*x^2 + 3*x + 4, got %s", prim.String())
	}
}

func TestWuPseudoDividePrimitive(t *testing.T) {
	// f = 3*x^2 + 2*x + 1
	// g = 2*x + 1
	// prem(f, g, x)
	f, _ := Parse("3*x^2 + 2*x + 1")
	g, _ := Parse("2*x + 1")

	_, r, exp, init, err := PseudoDividePrimitive(f, g, "x", []string{"x"})
	if err != nil {
		t.Fatalf("PseudoDividePrimitive failed: %v", err)
	}

	if init.String() != "2" {
		t.Errorf("expected initial 2, got %s", init.String())
	}
	if exp <= 0 {
		t.Errorf("expected power exp > 0, got %d", exp)
	}

	// Remainder r should have degree in x strictly less than deg(g) = 1
	polyR, okR := extractPoly(r, "x")
	if okR && polyR.degree() >= 1 {
		t.Errorf("expected rem degree < 1, got %d for %s", polyR.degree(), r.String())
	}

	// Verify exact vanishing when f is a multiple of g:
	// fExact = (2*x + 1)^2 = 4*x^2 + 4*x + 1
	fExact, _ := Parse("4*x^2 + 4*x + 1")
	_, rExact, _, _, err := PseudoDividePrimitive(fExact, g, "x", []string{"x"})
	if err != nil {
		t.Fatalf("PseudoDividePrimitive exact failed: %v", err)
	}
	if !isZero(rExact) {
		t.Errorf("expected exact vanishing remainder 0, got %s", rExact.String())
	}
}

func TestWuLinearSubstitutionFastPath(t *testing.T) {
	// f1 = 2*x1 - u1 (linear in x1)
	// f2 = x1^2 + u1
	f1, _ := Parse("2*x1 - u1")
	f2, _ := Parse("x1^2 + u1")

	order := []string{"u1", "x1"}
	reduced, subs := LinearSubstitutionFastPath([]Node{f1, f2}, order)
	if len(subs) == 0 {
		t.Fatalf("expected linear substitution to eliminate x1")
	}
	if _, ok := subs["x1"]; !ok {
		t.Fatalf("expected x1 in substitution map")
	}

	// reduced polynomials should not contain x1
	for _, p := range reduced {
		vars := collectVariables(p)
		for _, v := range vars {
			if v == "x1" {
				t.Errorf("variable x1 was not eliminated: %s", p.String())
			}
		}
	}
}

func TestWuGeometryPredicates(t *testing.T) {
	tests := []struct {
		exprStr string
		expectEq int
	}{
		{"midpoint(M, A, B)", 2},
		{"collinear(A, B, C)", 1},
		{"parallel(A, B, C, D)", 1},
		{"perpendicular(A, B, C, D)", 1},
		{"equal_length_sq(A, B, C, D)", 1},
		{"circle_concyclic(A, B, C, D)", 1},
	}

	for _, tc := range tests {
		node, err := Parse(tc.exprStr)
		if err != nil {
			t.Fatalf("Parse(%s) failed: %v", tc.exprStr, err)
		}
		trans, err := TranslateGeometricPredicate(node)
		if err != nil {
			t.Fatalf("TranslateGeometricPredicate(%s) failed: %v", tc.exprStr, err)
		}
		if len(trans.Equations) != tc.expectEq {
			t.Errorf("predicate %s: expected %d equations, got %d", tc.exprStr, tc.expectEq, len(trans.Equations))
		}
	}
}

func TestWuGeoProveMidpointTheorem(t *testing.T) {
	// Triangle ABC:
	// A = (0, 0), B = (u1, 0), C = (u2, u3)
	// M = midpoint of AB -> (x1, x2): 2*x1 - u1 = 0, 2*x2 - 0 = 0 => x2 = 0
	// N = midpoint of AC -> (x3, x4): 2*x3 - u2 = 0, 2*x4 - u3 = 0
	// Conclusion: parallel(M, N, B, C) -> (x3 - x1)*(u3 - 0) - (x4 - x2)*(u2 - u1) == 0
	// (x3 - x1)*u3 - x4*(u2 - u1) == 0
	h1, _ := Parse("2*x1 - u1")
	h2, _ := Parse("2*x2")
	h3, _ := Parse("2*x3 - u2")
	h4, _ := Parse("2*x4 - u3")
	concl, _ := Parse("(x3 - x1)*u3 - (x4 - x2)*(u2 - u1)")

	env := NewEnv()
	args := []Node{
		&ListNode{Elements: []Node{h1, h2, h3, h4}},
		concl,
	}

	res, err := EvalGeoProve(args, env)
	if err != nil {
		t.Fatalf("EvalGeoProve failed: %v", err)
	}

	constNode, ok := res.(*ConstNode)
	if !ok || constNode.Name != "true" {
		if env.LastCert != nil {
			t.Logf("Midpoint cert details: %s, equation: %s, residual: %s", env.LastCert.Details, env.LastCert.Equation, env.LastCert.Residual.String())
		}
		t.Fatalf("expected true, got %s", res.String())
	}

	if env.LastCert == nil {
		t.Fatalf("expected LastCert to be set")
	}
	if !env.LastCert.IsVerified {
		t.Errorf("expected LastCert.IsVerified to be true")
	}
	if env.LastCert.Domain != DomainGeometry {
		t.Errorf("expected DomainGeometry, got %v", env.LastCert.Domain)
	}
}

func TestWuGeoProveRefutedCase(t *testing.T) {
	// False theorem: x2 = 2*u1 => x2 = 3*u1
	h1, _ := Parse("x2 - 2*u1")
	concl, _ := Parse("x2 - 3*u1")

	env := NewEnv()
	args := []Node{
		&ListNode{Elements: []Node{h1}},
		concl,
	}

	res, err := EvalGeoProve(args, env)
	if err != nil {
		t.Fatalf("EvalGeoProve failed: %v", err)
	}

	constNode, ok := res.(*ConstNode)
	if !ok || constNode.Name != "false" {
		t.Fatalf("expected false, got %s", res.String())
	}

	if env.LastCert == nil {
		t.Fatalf("expected LastCert to be set")
	}
	if env.LastCert.IsVerified {
		t.Errorf("expected LastCert.IsVerified to be false")
	}
}

func TestWuGeometricCertificateAndLeanTranspile(t *testing.T) {
	h1, _ := Parse("2*x1 - u1")
	concl, _ := Parse("2*x1 - u1")

	env := NewEnv()
	args := []Node{
		&ListNode{Elements: []Node{h1}},
		concl,
	}

	_, err := EvalGeoProve(args, env)
	if err != nil {
		t.Fatalf("EvalGeoProve failed: %v", err)
	}

	cert := env.LastCert
	if cert == nil {
		t.Fatalf("expected LastCert")
	}

	callNode := &FuncNode{Name: "geo_prove", Args: args}
	leanCode, err := GenerateLeanSource("geometric_theorem", cert, callNode, &ConstNode{Name: "true"})
	if err != nil {
		t.Fatalf("GenerateLeanSource failed: %v", err)
	}

	if !strings.Contains(leanCode, "theorem geometric_theorem") {
		t.Errorf("expected theorem geometric_theorem in Lean output, got: %s", leanCode)
	}
	if !strings.Contains(leanCode, "Mathlib") {
		t.Errorf("expected Mathlib import, got: %s", leanCode)
	}
}

func TestWuZeroDecompositionTree(t *testing.T) {
	p1, _ := Parse("x1^2 - 1")
	vars := []string{"x1"}
	chain, initials, err := BuildAscendingChain([]Node{p1}, vars)
	if err != nil {
		t.Fatalf("BuildAscendingChain failed: %v", err)
	}
	root := &ZeroDecompositionTree{
		Chain:              chain,
		SaturationInitials: initials,
	}
	if root.Chain == nil {
		t.Fatalf("expected non-nil chain in ZeroDecompositionTree")
	}
	if len(root.Chain.Elements) == 0 {
		t.Fatalf("expected non-empty chain elements")
	}
}
