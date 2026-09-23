package calc

import (
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

