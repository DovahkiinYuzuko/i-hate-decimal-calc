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
