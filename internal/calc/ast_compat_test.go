package calc

import (
	"strings"
	"testing"
)

func TestAST_WalkAndInspect(t *testing.T) {
	// Build expression: sin(x + 2*y) + sqrt(z)
	expr, err := Parse("sin(x + 2*y) + sqrt(z)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	visitedTypes := make(map[NodeType]int)
	Inspect(expr, func(n Node) {
		visitedTypes[n.Type()]++
	})

	if visitedTypes[NodeAdd] != 2 {
		t.Errorf("expected 2 Add nodes, got %d", visitedTypes[NodeAdd])
	}
	if visitedTypes[NodeFunc] != 1 {
		t.Errorf("expected 1 Func node, got %d", visitedTypes[NodeFunc])
	}
	if visitedTypes[NodeSqrt] != 1 {
		t.Errorf("expected 1 Sqrt node, got %d", visitedTypes[NodeSqrt])
	}
	if visitedTypes[NodeVar] != 3 {
		t.Errorf("expected 3 Var nodes, got %d", visitedTypes[NodeVar])
	}

	walkCount := 0
	Walk(expr, func(n Node) bool {
		walkCount++
		if n.Type() == NodeFunc {
			return false
		}
		return true
	})
	if walkCount >= len(visitedTypes) && visitedTypes[NodeFunc] == 0 {
		t.Errorf("expected walkCount to be pruned")
	}
}

func TestAST_TransformAndSubstitute(t *testing.T) {
	expr, err := Parse("x^2 + 2*x + 1")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	val, _ := NewRational(3, 1)
	subbed := Substitute(expr, "x", val)

	str := subbed.String()
	if strings.Contains(str, "x") {
		t.Errorf("expected no 'x' in subbed expression, got: %s", str)
	}

	evaled, err := Eval(subbed)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	want, _ := NewRational(16, 1)
	if !evaled.Equal(want) {
		t.Errorf("Eval(Substitute) = %s, want %s", evaled.String(), want.String())
	}
}

func TestAST_ContainsVarAndExtractFreeVariables(t *testing.T) {
	expr, err := Parse("x^2 + 2*y + sin(z)")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if !ContainsVar(expr, "x") {
		t.Errorf("expected ContainsVar(expr, 'x') to be true")
	}
	if !ContainsVar(expr, "y") {
		t.Errorf("expected ContainsVar(expr, 'y') to be true")
	}
	if !ContainsVar(expr, "z") {
		t.Errorf("expected ContainsVar(expr, 'z') to be true")
	}
	if ContainsVar(expr, "w") {
		t.Errorf("expected ContainsVar(expr, 'w') to be false")
	}

	vars := ExtractFreeVariables(expr)
	if len(vars) != 3 || vars[0] != "x" || vars[1] != "y" || vars[2] != "z" {
		t.Errorf("ExtractFreeVariables = %v, want [x, y, z]", vars)
	}

	constExpr, _ := Parse("1/2 + sqrt(2)")
	if ContainsVar(constExpr, "x") {
		t.Errorf("expected false for constExpr")
	}
	if len(ExtractFreeVariables(constExpr)) != 0 {
		t.Errorf("expected empty vars for constExpr")
	}
}
