package calc

import (
	"math/big"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func makePoly(terms []ast.Monomial, varName string) *ast.PolyNode {
	return NewPolyNode([]string{varName}, ast.OrderLex, terms)
}

// TestPrimitiveElement_Sqrt2_Sqrt3 tests primitive element derivation for Q(sqrt(2), sqrt(3)).
func TestPrimitiveElement_Sqrt2_Sqrt3(t *testing.T) {
	// minPoly1: x^2 - 2 = 0
	m1 := makePoly([]ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(-2, 1), Exponents: []int{0}},
	}, "x")

	// minPoly2: x^2 - 3 = 0
	m2 := makePoly([]ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(-3, 1), Exponents: []int{0}},
	}, "x")

	res, err := FindPrimitiveElement(m1, m2, "theta")
	if err != nil {
		t.Fatalf("FindPrimitiveElement failed: %v", err)
	}

	if res.Degree != 4 {
		t.Errorf("Expected extension degree 4, got %d (minPoly: %s)", res.Degree, res.MinPolyTheta.String())
	}

	if res.RepAlpha == nil || res.RepBeta == nil {
		t.Fatalf("Expected non-nil generator representations")
	}

	// Verify algebraic relations: g(RepBeta) mod MinPolyTheta == 0, f(RepAlpha) mod MinPolyTheta == 0
	compG := substitutePolyIntoPoly(m2, res.RepBeta, "theta")
	_, remG, errG := PolyDivRem(compG, res.MinPolyTheta, "theta")
	if errG != nil || len(remG.Terms) > 0 {
		t.Errorf("g(RepBeta) mod m_theta != 0: %v (rem: %v)", errG, remG)
	}

	compF := substitutePolyIntoPoly(m1, res.RepAlpha, "theta")
	_, remF, errF := PolyDivRem(compF, res.MinPolyTheta, "theta")
	if errF != nil || len(remF.Terms) > 0 {
		t.Errorf("f(RepAlpha) mod m_theta != 0: %v (rem: %v)", errF, remF)
	}
}

// TestPrimitiveElement_Certificate_Lean tests Certificate IR generation and Lean 4 code output.
func TestPrimitiveElement_Certificate_Lean(t *testing.T) {
	m1 := makePoly([]ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(-2, 1), Exponents: []int{0}},
	}, "x")

	m2 := makePoly([]ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(-3, 1), Exponents: []int{0}},
	}, "x")

	res, err := FindPrimitiveElement(m1, m2, "theta")
	if err != nil {
		t.Fatalf("FindPrimitiveElement failed: %v", err)
	}

	cert := NewPrimitiveElementCertificate(
		PolyToNode(res.MinPolyTheta),
		res.SymbolTheta,
		res.C,
		PolyToNode(res.RepAlpha),
		PolyToNode(res.RepBeta),
		PolyToNode(res.MinPolyAlpha),
		PolyToNode(res.MinPolyBeta),
		PolyToNode(res.QuotientG),
		PolyToNode(res.QuotientF),
		mustRational(0, 1),
		true,
		"Primitive element isomorphism verification",
	)

	leanCode, err := TranspileCertificateIRToLean(cert)
	if err != nil {
		t.Fatalf("TranspileCertificateIRToLean failed: %v", err)
	}

	if !strings.Contains(leanCode, "theorem ihd_verified_proof") {
		t.Errorf("Expected lean code to contain theorem declaration, got: %s", leanCode)
	}
	if !strings.Contains(leanCode, "by intro x ; constructor <;> ring") {
		t.Errorf("Expected lean code to contain ring tactic, got: %s", leanCode)
	}
}

// TestPrimitiveElement_MultipleExtensions tests collapsing 3 extensions: Q(sqrt(2), sqrt(3), sqrt(5)).
func TestPrimitiveElement_MultipleExtensions(t *testing.T) {
	m1 := makePoly([]ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(-2, 1), Exponents: []int{0}},
	}, "x")

	m2 := makePoly([]ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(-3, 1), Exponents: []int{0}},
	}, "x")

	m3 := makePoly([]ast.Monomial{
		{Coeff: big.NewRat(1, 1), Exponents: []int{2}},
		{Coeff: big.NewRat(-5, 1), Exponents: []int{0}},
	}, "x")

	multiRes, err := CollapseMultipleExtensions([]*ast.PolyNode{m1, m2, m3}, "theta")
	if err != nil {
		t.Fatalf("CollapseMultipleExtensions failed: %v", err)
	}

	if multiRes.Degree != 8 {
		t.Errorf("Expected extension degree 8 for Q(sqrt2, sqrt3, sqrt5), got %d (poly: %s)", multiRes.Degree, multiRes.MinPolyTheta.String())
	}
	if len(multiRes.Reps) != 3 {
		t.Errorf("Expected 3 generator representations, got %d", len(multiRes.Reps))
	}
}

// TestEvalToPrimitiveElement_CLI tests evaluation of to_primitive_element function.
func TestEvalToPrimitiveElement_CLI(t *testing.T) {
	env := NewEnv()

	// Call: to_primitive_element([sqrt(2), sqrt(3)])
	sqrt2 := ast.NewSqrt(mustRational(2, 1))
	sqrt3 := ast.NewSqrt(mustRational(3, 1))
	inputList := &ast.ListNode{Elements: []ast.Node{sqrt2, sqrt3}}

	resNode, err := EvalToPrimitiveElement([]ast.Node{inputList}, env)
	if err != nil {
		t.Fatalf("EvalToPrimitiveElement failed: %v", err)
	}

	listRes, ok := resNode.(*ast.ListNode)
	if !ok || len(listRes.Elements) != 3 {
		t.Fatalf("Expected 3-element list result [theta, minpoly, [reps...]], got: %s", resNode.String())
	}

	thetaVar, ok := listRes.Elements[0].(*ast.VarNode)
	if !ok || thetaVar.Name != "theta" {
		t.Errorf("Expected theta variable, got: %s", listRes.Elements[0].String())
	}
}

// TestPrimitiveElementFSM_Transitions tests valid and invalid FSM transitions.
func TestPrimitiveElementFSM_Transitions(t *testing.T) {
	fsm := NewPrimitiveElementFSM()
	if fsm.Current() != PrimitiveStateInit {
		t.Fatalf("Expected INIT state, got %v", fsm.Current())
	}

	// Invalid direct transition from INIT to SUCCESS
	err := fsm.Transition(PrimitiveStateSuccess)
	if err == nil {
		t.Fatalf("Expected error on invalid transition INIT -> SUCCESS")
	}

	// Recreate and do valid path
	fsm2 := NewPrimitiveElementFSM()
	if err := fsm2.Transition(PrimitiveStateScanningC); err != nil {
		t.Fatalf("Transition to SCANNING_C failed: %v", err)
	}
	if err := fsm2.Transition(PrimitiveStateResultant); err != nil {
		t.Fatalf("Transition to RESULTANT failed: %v", err)
	}
	if err := fsm2.Transition(PrimitiveStateSquareFreeCheck); err != nil {
		t.Fatalf("Transition to SQUARE_FREE_CHECK failed: %v", err)
	}
	if err := fsm2.Transition(PrimitiveStateSolveGenerators); err != nil {
		t.Fatalf("Transition to SOLVE_GENERATORS failed: %v", err)
	}
	if err := fsm2.Transition(PrimitiveStateCertify); err != nil {
		t.Fatalf("Transition to CERTIFY failed: %v", err)
	}
	if err := fsm2.Transition(PrimitiveStateSuccess); err != nil {
		t.Fatalf("Transition to SUCCESS failed: %v", err)
	}
}
