package calc

import (
	"strings"
	"testing"
)

func TestCadFSM_Transitions(t *testing.T) {
	fsm := NewCadLifecycleFSM()
	if fsm.CurrentState() != CadStateInit {
		t.Fatalf("expected state Init, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateNormalized); err != nil {
		t.Fatalf("failed to transition to Normalized: %v", err)
	}
	if fsm.CurrentState() != CadStateNormalized {
		t.Fatalf("expected state Normalized, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateProjected); err != nil {
		t.Fatalf("failed to transition to Projected: %v", err)
	}
	if fsm.CurrentState() != CadStateProjected {
		t.Fatalf("expected state Projected, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateValidated); err != nil {
		t.Fatalf("failed to transition to Validated: %v", err)
	}
	if fsm.CurrentState() != CadStateValidated {
		t.Fatalf("expected state Validated, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateSampled); err != nil {
		t.Fatalf("failed to transition to Sampled: %v", err)
	}
	if fsm.CurrentState() != CadStateSampled {
		t.Fatalf("expected state Sampled, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateLifted); err != nil {
		t.Fatalf("failed to transition to Lifted: %v", err)
	}
	if fsm.CurrentState() != CadStateLifted {
		t.Fatalf("expected state Lifted, got %v", fsm.CurrentState())
	}

	if err := fsm.TransitionTo(CadStateDecided); err != nil {
		t.Fatalf("failed to transition to Decided: %v", err)
	}
	if fsm.CurrentState() != CadStateDecided {
		t.Fatalf("expected state Decided, got %v", fsm.CurrentState())
	}
}

func TestCad_1D_Linear(t *testing.T) {
	env := NewEnv()
	expr, err := Parse("x - 3 > 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rel, ok := expr.(*RelOpNode)
	if !ok {
		t.Fatalf("expected RelOpNode, got %T", expr)
	}

	sol, err := SolveInequality(rel, "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	str := sol.String()
	if !strings.Contains(str, "3") || !strings.Contains(str, "inf") {
		t.Fatalf("expected interval [3, inf], got %s", str)
	}
}

func TestCad_1D_Quadratic_Strict(t *testing.T) {
	env := NewEnv()
	// x^2 - 4 < 0 -> interval (-2, 2)
	expr, err := Parse("x^2 - 4 < 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rel, ok := expr.(*RelOpNode)
	if !ok {
		t.Fatalf("expected RelOpNode, got %T", expr)
	}

	sol, err := SolveInequality(rel, "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	str := sol.String()
	if !strings.Contains(str, "-2") || !strings.Contains(str, "2") {
		t.Fatalf("expected interval with -2 and 2, got %s", str)
	}
}

func TestCad_1D_Quadratic_NonStrict_Radical(t *testing.T) {
	env := NewEnv()
	// x^2 - 2 <= 0 -> interval [-sqrt(2), sqrt(2)]
	expr, err := Parse("x^2 - 2 <= 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rel, ok := expr.(*RelOpNode)
	if !ok {
		t.Fatalf("expected RelOpNode, got %T", expr)
	}

	sol, err := SolveInequality(rel, "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	str := sol.String()
	if !strings.Contains(str, "2") {
		t.Fatalf("expected interval with sqrt(2), got %s", str)
	}
}

func TestCad_1D_RepeatedRoot_Singleton(t *testing.T) {
	env := NewEnv()
	// (x - 1)^2 <= 0 -> singleton [1, 1]
	expr, err := Parse("(x - 1)^2 <= 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	rel, ok := expr.(*RelOpNode)
	if !ok {
		t.Fatalf("expected RelOpNode, got %T", expr)
	}

	sol, err := SolveInequality(rel, "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	str := sol.String()
	if !strings.Contains(str, "1") {
		t.Fatalf("expected solution containing 1, got %s", str)
	}
}

func TestCad_1D_Tautology_And_UNSAT(t *testing.T) {
	env := NewEnv()

	// x^2 + 1 > 0 -> true (Tautology)
	exprTrue, err := Parse("x^2 + 1 > 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	solTrue, err := SolveInequality(exprTrue.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if solTrue.String() != "true" {
		t.Fatalf("expected true, got %s", solTrue.String())
	}

	// x^2 + 1 < 0 -> false (UNSAT)
	exprFalse, err := Parse("x^2 + 1 < 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	solFalse, err := SolveInequality(exprFalse.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if solFalse.String() != "false" {
		t.Fatalf("expected false, got %s", solFalse.String())
	}
}

func TestCad_Solve_BuiltinIntegration(t *testing.T) {
	env := NewEnv()

	// solve(x^2 - 9 < 0, x) -> [[-3, 3]]
	expr, err := Parse("solve(x^2 - 9 < 0, x)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	str := evaled.String()
	if !strings.Contains(str, "-3") || !strings.Contains(str, "3") {
		t.Fatalf("expected interval [-3, 3], got %s", str)
	}
}

func TestCad_CADDecompose_Builtin(t *testing.T) {
	env := NewEnv()

	// cad([x^2 + y^2 - 1], [x, y])
	expr, err := Parse("cad([x^2 + y^2 - 1], [x, y])")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	evaled, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	list, ok := evaled.(*ListNode)
	if !ok || len(list.Elements) == 0 {
		t.Fatalf("expected non-empty list of samples, got %v", evaled)
	}
}

func TestDecompose1DCAD_Structure(t *testing.T) {
	env := NewEnv()

	// Polynomial with 2 real roots: x^2 - 4 = (x - 2)(x + 2)
	pNode, _ := Parse("x^2 - 4")
	p, ok := extractPoly(pNode, "x")
	if !ok {
		t.Fatalf("failed to extract poly")
	}

	cells, err := Decompose1DCAD([]*univariatePoly{p}, "x", env)
	if err != nil {
		t.Fatalf("Decompose1DCAD failed: %v", err)
	}

	// 2 roots => 2*2 + 1 = 5 cells
	if len(cells) != 5 {
		t.Fatalf("expected 5 cells (2m + 1), got %d", len(cells))
	}

	// Check alternating Sector (Dim 1, IsSection false) and Section (Dim 0, IsSection true)
	expectedTypes := []struct {
		dim       int
		isSection bool
	}{
		{dim: 1, isSection: false}, // (-inf, -2)
		{dim: 0, isSection: true},  // {-2}
		{dim: 1, isSection: false}, // (-2, 2)
		{dim: 0, isSection: true},  // {2}
		{dim: 1, isSection: false}, // (2, +inf)
	}

	for i, exp := range expectedTypes {
		if cells[i].Dimension != exp.dim {
			t.Errorf("cell %d: expected dim %d, got %d", i, exp.dim, cells[i].Dimension)
		}
		if cells[i].IsSection != exp.isSection {
			t.Errorf("cell %d: expected isSection %v, got %v", i, exp.isSection, cells[i].IsSection)
		}
		if len(cells[i].SamplePoint) != 1 {
			t.Errorf("cell %d: expected 1 sample point, got %d", i, len(cells[i].SamplePoint))
		}
	}
}

func TestDecompose1DCAD_CloseRoots(t *testing.T) {
	env := NewEnv()

	// Inequality with extremely close roots: (x - 1)*(x - 10001/10000) < 0
	// Distance between roots is 1/10000 = 0.0001, which broke old floatToRatSample (f * 1000).
	expr, err := Parse("(x - 1) * (x - 10001/10000) < 0")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	sol, err := SolveInequality(expr.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("SolveInequality failed: %v", err)
	}

	str := sol.String()
	if !strings.Contains(str, "1") || !strings.Contains(str, "10001/10000") {
		t.Fatalf("expected interval between 1 and 10001/10000, got: %s", str)
	}
}

func TestDecompose1DCAD_DegenerateAndEqualities(t *testing.T) {
	env := NewEnv()

	// 1. Double root isolated point: (x - 5)^2 <= 0 => [5, 5]
	exprDouble, _ := Parse("(x - 5)^2 <= 0")
	solDouble, err := SolveInequality(exprDouble.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if !strings.Contains(solDouble.String(), "5") {
		t.Fatalf("expected singleton with 5, got %s", solDouble.String())
	}

	// 2. Strict inequality with double root: (x - 5)^2 < 0 => false
	exprStrict, _ := Parse("(x - 5)^2 < 0")
	solStrict, err := SolveInequality(exprStrict.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if solStrict.String() != "false" {
		t.Fatalf("expected false, got %s", solStrict.String())
	}

	// 3. No real roots tautology: x^2 + 4 >= 0 => true
	exprTautology, _ := Parse("x^2 + 4 >= 0")
	solTautology, err := SolveInequality(exprTautology.(*RelOpNode), "x", env)
	if err != nil {
		t.Fatalf("solve failed: %v", err)
	}
	if solTautology.String() != "true" {
		t.Fatalf("expected true, got %s", solTautology.String())
	}
}
