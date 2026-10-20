package calc

import (
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

func parseAndEval(input string, env *Env) (Node, error) {
	node, err := Parse(input)
	if err != nil {
		return nil, err
	}
	return EvalWithEnv(node, env)
}

// TestEvalLP_Maximize_Basic tests basic 2-variable LP maximization in expression mode.
func TestEvalLP_Maximize_Basic(t *testing.T) {
	env := NewEnv()
	res, err := parseAndEval("lp_maximize(2*x + 3*y, [x + y <= 4, x >= 0, y >= 0])", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resStr := res.String()
	// Expect [x == 0, y == 4, optimal_value == 12]
	if !strings.Contains(resStr, "optimal_value == 12") {
		t.Errorf("expected optimal_value == 12, got: %s", resStr)
	}
	if !strings.Contains(resStr, "x == 0") || !strings.Contains(resStr, "y == 4") {
		t.Errorf("expected x == 0 and y == 4, got: %s", resStr)
	}
}

// TestEvalLP_Minimize_Fraction tests minimization with fractions.
// Minimize 1/3*x + 2/7*y s.t. x + y >= 5, x >= 0, y >= 0
// Since 2/7 = 6/21 < 1/3 = 7/21, optimal is x = 0, y = 5, min = 10/7.
func TestEvalLP_Minimize_Fraction(t *testing.T) {
	env := NewEnv()
	res, err := parseAndEval("lp_minimize(1/3*x + 2/7*y, [x + y >= 5, x >= 0, y >= 0])", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resStr := res.String()
	if !strings.Contains(resStr, "optimal_value == 10/7") {
		t.Errorf("expected optimal_value == 10/7, got: %s", resStr)
	}
	if !strings.Contains(resStr, "x == 0") || !strings.Contains(resStr, "y == 5") {
		t.Errorf("expected x == 0 and y == 5, got: %s", resStr)
	}
}

// TestEvalLP_MatrixMode tests matrix & vector input mode:
// lp_maximize([2, 3], [[1, 1]], ["<="], [4])
func TestEvalLP_MatrixMode(t *testing.T) {
	env := NewEnv()
	res, err := parseAndEval("lp_maximize([2, 3], [[1, 1]], [\"<=\"], [4])", env)
	if err != nil {
		t.Fatalf("unexpected error in matrix mode: %v", err)
	}

	resStr := res.String()
	if !strings.Contains(resStr, "optimal_value == 12") {
		t.Errorf("expected optimal_value == 12, got: %s", resStr)
	}
}

// TestEvalLP_EnvVariableSubstitution verifies integration with REPL environment variables.
func TestEvalLP_EnvVariableSubstitution(t *testing.T) {
	env := NewEnv()
	// Set bound variable L = 10
	limitNode := mustRational(10, 1)
	env.Set("L", limitNode)

	res, err := parseAndEval("lp_maximize(x + y, [x <= L, y <= L, x >= 0, y >= 0])", env)
	if err != nil {
		t.Fatalf("unexpected error with env substitution: %v", err)
	}

	resStr := res.String()
	// x <= 10, y <= 10 => optimal_value == 20
	if !strings.Contains(resStr, "optimal_value == 20") {
		t.Errorf("expected optimal_value == 20, got: %s", resStr)
	}
}

// TestEvalLP_PhysicalUnits verifies integration with Q^7 physical units and dimensional analysis.
func TestEvalLP_PhysicalUnits(t *testing.T) {
	env := NewEnv()

	// 1. Valid dimensional matching:
	// Maximize 2*m*x + 3*m*y s.t. 1*s*x + 2*s*y <= 4*s, x >= 0, y >= 0
	// x and y are dimensionless scalars, objective is in meters [m], constraint is in seconds [s].
	res, err := parseAndEval("lp_maximize(2*m*x + 3*m*y, [1*s*x + 2*s*y <= 4*s, x >= 0, y >= 0])", env)
	if err != nil {
		t.Fatalf("unexpected error with valid units: %v", err)
	}

	resStr := res.String()
	if !strings.Contains(resStr, "optimal_value == 8") {
		t.Errorf("expected optimal_value == 8, got: %s", resStr)
	}

	// 2. Incompatible dimension error detection:
	// 2*m*x + 3*s*y has incompatible dimensions (length vs time) in same objective!
	_, err = parseAndEval("lp_maximize(2*m*x + 3*s*y, [x + y <= 4, x >= 0, y >= 0])", env)
	if err == nil {
		t.Errorf("expected error for incompatible dimensions in objective, but succeeded")
	}
}

// TestEvalLP_ProvenanceExplain verifies that explain mode records pivot steps into Tracer.
func TestEvalLP_ProvenanceExplain(t *testing.T) {
	env := NewEnv()
	tracer := provenance.NewTraceRecorder()
	env.Tracer = tracer

	res, err := parseAndEval("lp_maximize(2*x + 3*y, [x + y <= 4, x >= 0, y >= 0])", env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil {
		t.Fatalf("expected non-nil result")
	}

	root := tracer.Root()
	if root == nil {
		t.Fatalf("expected active root derivation node in tracer")
	}

	if root.RuleID != "lp_maximize" {
		t.Errorf("expected root RuleID 'lp_maximize', got %q", root.RuleID)
	}

	// At least 1 pivot step should be recorded
	if len(root.Children) == 0 {
		t.Errorf("expected recorded pivot steps in tracer children")
	}
}
