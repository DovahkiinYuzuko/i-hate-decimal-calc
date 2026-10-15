package calc

import (
	"strings"
	"testing"
)

// TestEvalCollection_Slice_Basic verifies standard 0-indexed slicing patterns.
func TestEvalCollection_Slice_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[0, 1, 2, 3, 4][:]", "[0, 1, 2, 3, 4]"},
		{"[0, 1, 2, 3, 4][1:3]", "[1, 2]"},
		{"[0, 1, 2, 3, 4][:2]", "[0, 1]"},
		{"[0, 1, 2, 3, 4][2:]", "[2, 3, 4]"},
		{"[0, 1, 2, 3, 4][::2]", "[0, 2, 4]"},
		{"[0, 1, 2, 3, 4][1::2]", "[1, 3]"},
		{"[0, 1, 2, 3, 4][:4:2]", "[0, 2]"},
		{"[0, 1, 2, 3, 4][1:4:2]", "[1, 3]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

// TestEvalCollection_Slice_Negative verifies negative indexing and negative steps.
func TestEvalCollection_Slice_Negative(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[0, 1, 2, 3, 4][-1]", "4"},
		{"[0, 1, 2, 3, 4][-2]", "3"},
		{"[0, 1, 2, 3, 4][-3:-1]", "[2, 3]"},
		{"[0, 1, 2, 3, 4][::-1]", "[4, 3, 2, 1, 0]"},
		{"[0, 1, 2, 3, 4][3:1:-1]", "[3, 2]"},
		{"[0, 1, 2, 3, 4][:-3:-1]", "[4, 3]"},
		{"[0, 1, 2, 3, 4][-2::-1]", "[3, 2, 1, 0]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

// TestEvalCollection_Slice_Clipping verifies CPython PySlice_GetIndicesEx clipping behavior.
func TestEvalCollection_Slice_Clipping(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[0, 1, 2, 3, 4][10:20]", "[]"},
		{"[0, 1, 2, 3, 4][-20:-10]", "[]"},
		{"[0, 1, 2, 3, 4][4:2]", "[]"},
		{"[0, 1, 2, 3, 4][1:3:-1]", "[]"},
		{"[][:]", "[]"},
		{"[][1:3]", "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

// TestEvalCollection_Slice_Errors verifies error conditions (step=0, out of bounds single index).
func TestEvalCollection_Slice_Errors(t *testing.T) {
	tests := []struct {
		input       string
		errContains string
	}{
		{"[0, 1, 2, 3, 4][::0]", "step"},
		{"[0, 1, 2, 3, 4][5]", "index"},
		{"[0, 1, 2, 3, 4][-6]", "index"},
		{"[][0]", "index"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := EvalString(tt.input)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.input)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tt.errContains) {
				t.Errorf("expected error containing %q, got %v", tt.errContains, err)
			}
		})
	}
}

// TestEvalCollection_Range verifies sequence generation including exact rational steps.
func TestEvalCollection_Range(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"range(5)", "[0, 1, 2, 3, 4]"},
		{"range(0)", "[]"},
		{"range(2, 6)", "[2, 3, 4, 5]"},
		{"range(0, 10, 3)", "[0, 3, 6, 9]"},
		{"range(5, 0, -1)", "[5, 4, 3, 2, 1]"},
		{"range(5, 2)", "[]"},
		{"range(2, 5, -1)", "[]"},
		{"range(0, 1, 1/3)", "[0, 1/3, 2/3]"},
		{"range(1/2, 2, 1/2)", "[1/2, 1, 3/2]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

// TestEvalCollection_Range_Errors verifies zero-step error on range.
func TestEvalCollection_Range_Errors(t *testing.T) {
	_, err := EvalString("range(0, 5, 0)")
	if err == nil {
		t.Fatalf("expected error for range step=0, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "step") {
		t.Errorf("expected error containing 'step', got %v", err)
	}
}

// TestEvalCollection_Map verifies map with built-in functions and lambdas.
func TestEvalCollection_Map(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"map(sin, [0, pi/2, pi])", "[0, 1, 0]"},
		{"map(lambda(x, x^2), [1, 2, 3, 4])", "[1, 4, 9, 16]"},
		{"map(lambda(x, x^2), range(5))", "[0, 1, 4, 9, 16]"},
		{"map(lambda(x, x + 1), [])", "[]"},
		{"map(lambda([x], x * 2), [1, 2, 3])", "[2, 4, 6]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

// TestEvalCollection_Filter verifies filter with predicate functions and lambdas.
func TestEvalCollection_Filter(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"filter(is_prime, range(1, 11))", "[2, 3, 5, 7]"},
		{"filter(lambda(x, x > 2), [1, 2, 3, 4])", "[3, 4]"},
		{"filter(lambda(x, x < 0), [1, 2, 3])", "[]"},
		{"filter(lambda(x, x == 2), [1, 2, 2, 3, 2])", "[2, 2, 2]"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

// TestEvalCollection_ShadowScope verifies that temporary parameters inside lambda do not leak or overwrite outer env.
func TestEvalCollection_ShadowScope(t *testing.T) {
	env := NewEnv()
	// Set outer x = 100
	_, err := EvalStringWithEnv("x = 100", env)
	if err != nil {
		t.Fatalf("failed to set outer x: %v", err)
	}

	// Run map using x as parameter
	mapRes, err := EvalStringWithEnv("map(lambda(x, x + 1), [1, 2, 3])", env)
	if err != nil {
		t.Fatalf("map failed: %v", err)
	}
	if mapRes.String() != "[2, 3, 4]" {
		t.Errorf("map result got %s, expected [2, 3, 4]", mapRes.String())
	}

	// Verify outer x is still 100
	xVal, err := EvalStringWithEnv("x", env)
	if err != nil {
		t.Fatalf("eval x failed: %v", err)
	}
	if xVal.String() != "100" {
		t.Errorf("outer variable x was contaminated: got %s, expected 100", xVal.String())
	}
}

// TestEvalCollection_LambdaInvocation verifies first-class lambda bindings and immediate invocation.
func TestEvalCollection_LambdaInvocation(t *testing.T) {
	tests := []struct {
		setup    string
		input    string
		expected string
	}{
		{"", "lambda(x, x + 10)(5)", "15"},
		{"", "lambda([a, b], a * b + 1)(3, 4)", "13"},
		{"f = lambda(x, x^3)", "f(4)", "64"},
		{"g = lambda([x, y], x^2 + y^2)", "g(3, 4)", "25"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			env := NewEnv()
			if tt.setup != "" {
				_, err := EvalStringWithEnv(tt.setup, env)
				if err != nil {
					t.Fatalf("setup %s failed: %v", tt.setup, err)
				}
			}
			node, err := EvalStringWithEnv(tt.input, env)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

// TestEvalCollection_Composition verifies syntax precedence, chains, and combinations with operators.
func TestEvalCollection_Composition(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"[1, 2, 3][0] + [4, 5, 6][1]", "6"},
		{"-[1, 2, 3][0]", "-1"},
		{"[2, 3, 4][0]^3", "8"},
		{"2 * [10, 20][1]", "40"},
		{"range(10)[2:5]", "[2, 3, 4]"},
		{"[[1, 2], [3, 4]][1][0]", "3"},
		{"map(lambda(x, x^2), range(5))[1:4]", "[1, 4, 9]"},
		{"filter(is_prime, range(1, 10))[-1]", "7"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := EvalString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.input, err)
			}
			if node.String() != tt.expected {
				t.Errorf("input %s: got %s, expected %s", tt.input, node.String(), tt.expected)
			}
		})
	}
}

// TestEvalCollection_ExplainTrace verifies that --explain outputs trace rewrite steps.
func TestEvalCollection_ExplainTrace(t *testing.T) {
	sink := NewMemoryTraceSink()
	var err error
	WithTraceSink(sink, func() {
		_, err = EvalString("map(lambda(x, x^2), [1, 2])")
	})
	if err != nil {
		t.Fatalf("map failed: %v", err)
	}

	events := sink.Events()
	foundMapRule := false
	for _, e := range events {
		if e.Rule == RuleMap {
			foundMapRule = true
			break
		}
	}
	if !foundMapRule {
		t.Errorf("expected TraceEvent with RuleMap, events: %+v", events)
	}
}
