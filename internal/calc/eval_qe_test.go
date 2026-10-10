package calc

import (
	"strings"
	"testing"
)

func TestQeLifecycleFSM(t *testing.T) {
	fsm := NewQeLifecycleFSM()
	if fsm.CurrentState() != QeStateInit {
		t.Fatalf("expected QeStateInit, got %v", fsm.CurrentState())
	}

	// Valid sequence: Init -> PrenexNormalized -> VariableOrdered -> CadDecomposed -> TruthEvaluated -> FormulaConstructed
	if err := fsm.TransitionTo(QeStatePrenexNormalized); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(QeStateVariableOrdered); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(QeStateCadDecomposed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(QeStateTruthEvaluated); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fsm.TransitionTo(QeStateFormulaConstructed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid transition from terminal state
	if err := fsm.TransitionTo(QeStateInit); err == nil {
		t.Fatalf("expected error on transition from terminal state, got nil")
	}

	// Test history tracking
	history := fsm.History()
	expectedStates := []QeState{
		QeStateInit,
		QeStatePrenexNormalized,
		QeStateVariableOrdered,
		QeStateCadDecomposed,
		QeStateTruthEvaluated,
		QeStateFormulaConstructed,
	}
	if len(history) != len(expectedStates) {
		t.Fatalf("expected %d history states, got %d", len(expectedStates), len(history))
	}
	for i, s := range expectedStates {
		if history[i] != s {
			t.Errorf("history[%d]: expected %v, got %v", i, s, history[i])
		}
	}
}

func TestQEClosedSentences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "forall always positive quadratic",
			input:    "qe(forall([x], x^2 + 1 > 0))",
			expected: "true",
		},
		{
			name:     "forall not always positive quadratic",
			input:    "qe(forall([x], x^2 - 1 > 0))",
			expected: "false",
		},
		{
			name:     "forall square non-negative",
			input:    "qe(forall([x], x^2 >= 0))",
			expected: "true",
		},
		{
			name:     "exists real root for x^2 - 2 == 0",
			input:    "qe(exists([x], x^2 - 2 == 0))",
			expected: "true",
		},
		{
			name:     "exists real root for x^2 + 1 == 0",
			input:    "qe(exists([x], x^2 + 1 == 0))",
			expected: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("EvalString(%q) returned error: %v", tt.input, err)
			}
			got := res.String()
			if got != tt.expected {
				t.Errorf("EvalString(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestQEParametric(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "forall quadratic positive leading coeff 1",
			input:    "qe(forall([x], x^2 + a*x + b > 0))",
			expected: "-4*b + a^2 < 0",
		},
		{
			name:     "exists quadratic root with param",
			input:    "qe(exists([x], x^2 - 4*b == 0))",
			expected: "16*b >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("EvalString(%q) returned error: %v", tt.input, err)
			}
			got := Format(res)
			if got != tt.expected {
				t.Errorf("EvalString(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestQEGeneralCAD(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "cubic real root exists for all a",
			input:    "qe(exists([x], x^3 - a == 0))",
			expected: "true",
		},
		{
			name:     "cubic not zero for all x",
			input:    "qe(forall([x], x^3 - a == 0))",
			expected: "false",
		},
		{
			name:     "alternating forall x exists y (x + y == 0)",
			input:    "qe(forall([x], exists([y], x + y == 0)))",
			expected: "true",
		},
		{
			name:     "alternating exists x forall y (x + y == 0)",
			input:    "qe(exists([x], forall([y], x + y == 0)))",
			expected: "false",
		},
		{
			name:     "compound inequality exists in open interval",
			input:    "qe(exists([x], and(x > 0, x < 2)))",
			expected: "true",
		},
		{
			name:     "compound inequality forall in open interval",
			input:    "qe(forall([x], and(x > 0, x < 2)))",
			expected: "false",
		},
		{
			name:     "parametric strict inequality exists x^2 - a < 0",
			input:    "qe(exists([x], x^2 - a < 0))",
			expected: "a > 0",
		},
		{
			name:     "parametric non-strict inequality exists x^2 - a <= 0",
			input:    "qe(exists([x], x^2 - a <= 0))",
			expected: "a >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("EvalString(%q) returned error: %v", tt.input, err)
			}
			got := Format(res)
			if got != tt.expected {
				t.Errorf("EvalString(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestQEParametricDegeneracy(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "exists quadratic root covers a=0 linear and zero polynomial",
			input:    "qe(exists([x], a*x^2 + b*x + c == 0))",
			expected: "or(and(a != 0, -4*a*c + b^2 >= 0), or(and(a == 0, b != 0), and(a == 0, and(b == 0, c == 0))))",
		},
		{
			name:     "forall quadratic geq zero covers a=0 degenerate",
			input:    "qe(forall([x], a*x^2 + b*x + c >= 0))",
			expected: "or(and(a > 0, -4*a*c + b^2 <= 0), and(a == 0, and(b == 0, c >= 0)))",
		},
		{
			name:     "forall quadratic gt zero covers a=0 degenerate",
			input:    "qe(forall([x], a*x^2 + b*x + c > 0))",
			expected: "or(and(a > 0, -4*a*c + b^2 < 0), and(a == 0, and(b == 0, c > 0)))",
		},
		{
			name:     "forall quadratic leq zero covers a=0 degenerate",
			input:    "qe(forall([x], a*x^2 + b*x + c <= 0))",
			expected: "or(and(a < 0, -4*a*c + b^2 <= 0), and(a == 0, and(b == 0, c <= 0)))",
		},
		{
			name:     "forall quadratic lt zero covers a=0 degenerate",
			input:    "qe(forall([x], a*x^2 + b*x + c < 0))",
			expected: "or(and(a < 0, -4*a*c + b^2 < 0), and(a == 0, and(b == 0, c < 0)))",
		},
		{
			name:     "exists quadratic gt zero covers a=0 and discriminant",
			input:    "qe(exists([x], a*x^2 + b*x + c > 0))",
			expected: "or(a > 0, or(and(a < 0, -4*a*c + b^2 > 0), or(and(a == 0, b != 0), and(a == 0, and(b == 0, c > 0)))))",
		},
		{
			name:     "exists quadratic geq zero covers a=0 and discriminant",
			input:    "qe(exists([x], a*x^2 + b*x + c >= 0))",
			expected: "or(a > 0, or(and(a < 0, -4*a*c + b^2 >= 0), or(and(a == 0, b != 0), and(a == 0, and(b == 0, c >= 0)))))",
		},
		{
			name:     "forall quadratic identically zero",
			input:    "qe(forall([x], a*x^2 + b*x + c == 0))",
			expected: "and(a == 0, and(b == 0, c == 0))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("EvalString(%q) returned error: %v", tt.input, err)
			}
			got := Format(res)
			if got != tt.expected {
				t.Errorf("EvalString(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestConstructCylindricalAlgebraicFormula_2D(t *testing.T) {
	// Satisfied cell: sector x in (0, 2), sector y in (0, 3)
	cell := CadCell{
		Dimension:   2,
		SamplePoint: []Node{mustRational(1, 1), mustRational(1, 1)},
		IsSection:   false,
	}

	caf, err := ConstructCylindricalAlgebraicFormula([]CadCell{cell}, []string{"x", "y"})
	if err != nil {
		t.Fatalf("ConstructCylindricalAlgebraicFormula failed: %v", err)
	}
	if caf == nil {
		t.Fatalf("expected non-nil CAF")
	}
	s := Format(caf)
	if s == "" {
		t.Errorf("expected non-empty formula string")
	}
}

func TestSimplifyCylindricalFormula_SiblingMerge(t *testing.T) {
	// Formula: and(0 < x, x < 1) or (x == 1) or and(1 < x, x < 2)
	p1, err1 := Parse("and(0 < x, x < 1)")
	if err1 != nil {
		t.Fatalf("parse p1 failed: %v", err1)
	}
	p2, err2 := Parse("x == 1")
	if err2 != nil {
		t.Fatalf("parse p2 failed: %v", err2)
	}
	p3, err3 := Parse("and(1 < x, x < 2)")
	if err3 != nil {
		t.Fatalf("parse p3 failed: %v", err3)
	}
	orNode := &FuncNode{Name: "or", Args: []Node{p1, p2, p3}}

	simplified, err := SimplifyCylindricalFormula(orNode)
	if err != nil {
		t.Fatalf("SimplifyCylindricalFormula failed: %v", err)
	}
	s := Format(simplified)
	if strings.Contains(s, "x == 1") {
		t.Errorf("expected x == 1 to be absorbed into merged interval, got: %s", s)
	}
}

func TestCADSolveFormula_MultivariateBoundaryFormula(t *testing.T) {
	env := NewEnv()
	// and(x > 1, y > 2)
	expr, err := Parse("and(x > 1, y > 2)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	sol, err := CADSolveFormula(expr, []string{"x", "y"}, env)
	if err != nil {
		t.Fatalf("CADSolveFormula failed: %v", err)
	}
	if sol == nil {
		t.Fatalf("expected non-nil solution")
	}

	// In Phase 3, solution must NOT be a raw list of sample points [[1, 2], ...],
	// but a reconstructed formula (FuncNode or RelOpNode or VarNode)
	if _, isList := sol.(*ListNode); isList {
		t.Errorf("multivariate QE must return a boundary formula, not a raw ListNode sample point list: %v", sol)
	}
}



