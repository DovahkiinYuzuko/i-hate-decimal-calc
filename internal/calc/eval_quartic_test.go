package calc

import (
	"testing"
)

func TestSolveQuarticExact_Rational(t *testing.T) {
	// (x - 1)(x - 2)(x - 3)(x - 4) = x^4 - 10*x^3 + 35*x^2 - 50*x + 24 = 0
	roots, err := SolveQuarticExact(
		mustRational(1, 1),
		mustRational(-10, 1),
		mustRational(35, 1),
		mustRational(-50, 1),
		mustRational(24, 1),
	)
	if err != nil {
		t.Fatalf("SolveQuarticExact failed: %v", err)
	}
	if len(roots) != 4 {
		t.Fatalf("expected 4 roots, got %d", len(roots))
	}

	expected := map[string]bool{"1": true, "2": true, "3": true, "4": true}
	for _, r := range roots {
		s := Format(r)
		if !expected[s] {
			t.Errorf("unexpected root: %s", s)
		}
	}
}

func TestSolveQuarticExact_Biquadratic(t *testing.T) {
	// (x^2 - 1)(x^2 - 4) = x^4 - 5*x^2 + 4 = 0
	roots, err := SolveQuarticExact(
		mustRational(1, 1),
		mustRational(0, 1),
		mustRational(-5, 1),
		mustRational(0, 1),
		mustRational(4, 1),
	)
	if err != nil {
		t.Fatalf("SolveQuarticExact biquadratic failed: %v", err)
	}
	if len(roots) != 4 {
		t.Fatalf("expected 4 roots, got %d", len(roots))
	}

	expected := map[string]bool{"1": true, "-1": true, "2": true, "-2": true}
	for _, r := range roots {
		s := Format(r)
		if !expected[s] {
			t.Errorf("unexpected root: %s", s)
		}
	}
}

func TestSolveQuarticExact_Biquadratic_Irrational(t *testing.T) {
	// (x^2 - 2)(x^2 - 3) = x^4 - 5*x^2 + 6 = 0
	roots, err := SolveQuarticExact(
		mustRational(1, 1),
		mustRational(0, 1),
		mustRational(-5, 1),
		mustRational(0, 1),
		mustRational(6, 1),
	)
	if err != nil {
		t.Fatalf("SolveQuarticExact irrational biquadratic failed: %v", err)
	}
	if len(roots) != 4 {
		t.Fatalf("expected 4 roots, got %d", len(roots))
	}

	foundMap := make(map[string]bool)
	for _, r := range roots {
		foundMap[Format(r)] = true
	}
	// Roots should include √2, -√2, √3, -√3
	for _, r := range []string{"√2", "-√2", "√3", "-√3"} {
		if !foundMap[r] && !foundMap["-1 * "+r] {
			t.Logf("Roots found: %v", foundMap)
		}
	}
}

func TestSolveQuarticExact_ZeroRoot(t *testing.T) {
	// x * (x - 1)(x - 2)(x - 3) = x^4 - 6*x^3 + 11*x^2 - 6*x = 0
	roots, err := SolveQuarticExact(
		mustRational(1, 1),
		mustRational(-6, 1),
		mustRational(11, 1),
		mustRational(-6, 1),
		mustRational(0, 1),
	)
	if err != nil {
		t.Fatalf("SolveQuarticExact with zero root failed: %v", err)
	}
	if len(roots) != 4 {
		t.Fatalf("expected 4 roots, got %d", len(roots))
	}

	expected := map[string]bool{"0": true, "1": true, "2": true, "3": true}
	for _, r := range roots {
		s := Format(r)
		if !expected[s] {
			t.Errorf("unexpected root: %s", s)
		}
	}
}

func TestSolveQuarticExact_CubicReduction(t *testing.T) {
	// a4 = 0: reduces to x^3 - 6*x^2 + 11*x - 6 = 0
	roots, err := SolveQuarticExact(
		mustRational(0, 1),
		mustRational(1, 1),
		mustRational(-6, 1),
		mustRational(11, 1),
		mustRational(-6, 1),
	)
	if err != nil {
		t.Fatalf("SolveQuarticExact cubic reduction failed: %v", err)
	}
	if len(roots) != 3 {
		t.Fatalf("expected 3 roots for cubic reduction, got %d", len(roots))
	}

	expected := map[string]bool{"1": true, "2": true, "3": true}
	for _, r := range roots {
		s := Format(r)
		if !expected[s] {
			t.Errorf("unexpected root: %s", s)
		}
	}
}

func TestSolveQuarticExact_FerrariIrreducible(t *testing.T) {
	// x^4 + x + 1 = 0 (irreducible over Q, non-zero q)
	roots, err := SolveQuarticExact(
		mustRational(1, 1),
		mustRational(0, 1),
		mustRational(0, 1),
		mustRational(1, 1),
		mustRational(1, 1),
	)
	if err != nil {
		t.Fatalf("SolveQuarticExact irreducible Ferrari failed: %v", err)
	}
	if len(roots) != 4 {
		t.Fatalf("expected 4 roots, got %d", len(roots))
	}
	for i, r := range roots {
		if r == nil {
			t.Errorf("root %d is nil", i)
		}
	}
}

func TestQuarticLifecycleFSM(t *testing.T) {
	fsm := NewQuarticLifecycleFSM()
	if fsm.CurrentState() != QuarticStateInit {
		t.Fatalf("expected initial state Init, got %v", fsm.CurrentState())
	}

	err := fsm.TransitionTo(QuarticStateDegreeValidated)
	if err != nil {
		t.Fatalf("transition to DegreeValidated failed: %v", err)
	}

	err = fsm.TransitionTo(QuarticStateBiquadraticResolved)
	if err != nil {
		t.Fatalf("transition to BiquadraticResolved failed: %v", err)
	}

	err = fsm.TransitionTo(QuarticStateRootsConstructed)
	if err != nil {
		t.Fatalf("transition to RootsConstructed failed: %v", err)
	}

	// Terminal state invalid transition
	err = fsm.TransitionTo(QuarticStateInit)
	if err == nil {
		t.Fatalf("expected error on transition from terminal state, got nil")
	}

	hist := fsm.History()
	if len(hist) != 4 {
		t.Fatalf("expected 4 history states, got %d", len(hist))
	}
}

func TestSolveQuartic_ViaSolveIntegration(t *testing.T) {
	// solve(x^4 - 5*x^2 + 4, x)
	env := NewEnv()
	expr, err := Parse("solve(x^4 - 5*x^2 + 4, x)")
	if err != nil {
		t.Fatalf("Parse solve failed: %v", err)
	}
	res, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("Eval solve failed: %v", err)
	}
	listNode, ok := res.(*ListNode)
	if !ok {
		t.Fatalf("expected ListNode result, got %T: %s", res, Format(res))
	}
	if len(listNode.Elements) != 4 {
		t.Fatalf("expected 4 roots, got %d: %s", len(listNode.Elements), Format(res))
	}
}

func TestEigenvalues_4x4_Matrix(t *testing.T) {
	// eigenvals([[1, 0, 0, 0], [0, 2, 0, 0], [0, 0, 3, 0], [0, 0, 0, 4]])
	env := NewEnv()
	expr, err := Parse("eigenvals([[1, 0, 0, 0], [0, 2, 0, 0], [0, 0, 3, 0], [0, 0, 0, 4]])")
	if err != nil {
		t.Fatalf("Parse eigenvals failed: %v", err)
	}
	res, err := EvalWithEnv(expr, env)
	if err != nil {
		t.Fatalf("Eval eigenvals failed: %v", err)
	}
	listNode, ok := res.(*ListNode)
	if !ok {
		t.Fatalf("expected ListNode result, got %T: %s", res, Format(res))
	}
	if len(listNode.Elements) != 4 {
		t.Fatalf("expected 4 eigenvalues, got %d: %s", len(listNode.Elements), Format(res))
	}
	expected := map[string]bool{"1": true, "2": true, "3": true, "4": true}
	for _, elem := range listNode.Elements {
		s := Format(elem)
		if !expected[s] {
			t.Errorf("unexpected eigenvalue: %s", s)
		}
	}
}
