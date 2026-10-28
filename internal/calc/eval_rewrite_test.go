package calc

import (
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/provenance"
)

func parseRewriteTestExpr(t *testing.T, input string) ast.Node {
	t.Helper()
	node, err := Parse(input)
	if err != nil {
		t.Fatalf("failed to parse %q: %v", input, err)
	}
	return node
}

func evalRewriteTestExpr(t *testing.T, input string, env *Env) ast.Node {
	t.Helper()
	node := parseRewriteTestExpr(t, input)
	res, err := EvalWithEnv(node, env)
	if err != nil {
		t.Fatalf("failed to eval %q: %v", input, err)
	}
	return res
}

func TestMatchAST_Wildcards(t *testing.T) {
	// Syntactic wildcard matching: f(_x) against f(a + b)
	pat := parseRewriteTestExpr(t, "f(_x)")
	target := parseRewriteTestExpr(t, "f(a + b)")

	bindings, ok := MatchAST(pat, target, PatternBindings{})
	if !ok {
		t.Fatalf("expected pattern to match, got false")
	}
	if bound, exists := bindings["_x"]; !exists || bound.String() != "a + b" {
		t.Errorf("expected _x to bind to 'a + b', got %v", bound)
	}
}

func TestMatchAST_NonLinearPattern(t *testing.T) {
	// Non-linear: f(_a, _a) should match f(x + 1, x + 1) but NOT f(x + 1, x + 2)
	pat := parseRewriteTestExpr(t, "f(_a, _a)")

	targetSuccess := parseRewriteTestExpr(t, "f(x + 1, x + 1)")
	if _, ok := MatchAST(pat, targetSuccess, PatternBindings{}); !ok {
		t.Errorf("expected non-linear pattern to match identical subterms")
	}

	targetFail := parseRewriteTestExpr(t, "f(x + 1, x + 2)")
	if _, ok := MatchAST(pat, targetFail, PatternBindings{}); ok {
		t.Errorf("expected non-linear pattern NOT to match different subterms")
	}
}

func TestReplace_Basic(t *testing.T) {
	env := NewEnv()
	// replace(sin(t)^2, rule(sin(_x)^2, 1 - cos(_x)^2))
	res := evalRewriteTestExpr(t, "replace(sin(t)^2, rule(sin(_x)^2, 1 - cos(_x)^2))", env)
	resStr := res.String()
	if !strings.Contains(resStr, "1") || !strings.Contains(resStr, "cos(t)^2") {
		t.Errorf("expected replacement containing 1 and cos(t)^2, got %q", resStr)
	}
}

func TestReplace_ACPartialAdd(t *testing.T) {
	env := NewEnv()
	// rule(sin(_x)^2 + cos(_x)^2, 1) applied to sin(t)^2 + cos(t)^2 + y
	// AC partial match should replace the trigonometric sum with 1, leaving 1 + y
	res := evalRewriteTestExpr(t, "replace(sin(t)^2 + cos(t)^2 + y, rule(sin(_x)^2 + cos(_x)^2, 1))", env)
	resStr := res.String()
	if !strings.Contains(resStr, "1") || !strings.Contains(resStr, "y") {
		t.Errorf("expected AC partial replacement with 1 and y, got %q", resStr)
	}
}

func TestRewrite_FixedPoint(t *testing.T) {
	env := NewEnv()
	// rewrite(sin(sin(sin(x))), rule(sin(_t), _t)) -> should reduce to x
	res := evalRewriteTestExpr(t, "rewrite(sin(sin(sin(x))), rule(sin(_t), _t))", env)
	if res.String() != "x" {
		t.Errorf("expected 'x', got %q", res.String())
	}
}

func TestRewrite_MultipleRulesList(t *testing.T) {
	env := NewEnv()
	// Multiple rules: [rule(a, b), rule(b, c)]
	res := evalRewriteTestExpr(t, "rewrite(sin(a), [rule(a, b), rule(b, c)])", env)
	if res.String() != "sin(c)" {
		t.Errorf("expected 'sin(c)', got %q", res.String())
	}
}

func TestRewrite_CycleDetection(t *testing.T) {
	env := NewEnv()
	// Oscillating rule: a -> b, b -> a
	// Cycle detection must safely terminate without hanging
	res := evalRewriteTestExpr(t, "rewrite(a, [rule(a, b), rule(b, a)])", env)
	resStr := res.String()
	if resStr != "a" && resStr != "b" {
		t.Errorf("expected cycle detection termination with 'a' or 'b', got %q", resStr)
	}
}

func TestRewrite_MaxStepsArg(t *testing.T) {
	env := NewEnv()
	// rule(_n, _n + 1) will increase indefinitely. Setting max_steps = 5 must stop after 5 steps.
	res := evalRewriteTestExpr(t, "rewrite(0, rule(_n, _n + 1), 5)", env)
	if res == nil {
		t.Fatalf("expected non-nil result from bounded rewrite")
	}
}

func TestRewrite_ProvenanceTrace(t *testing.T) {
	env := NewEnv()
	recorder := provenance.NewTraceRecorder()
	env.Tracer = recorder

	_ = evalRewriteTestExpr(t, "replace(sin(x)^2, rule(sin(_t)^2, 1 - cos(_t)^2))", env)
	root := recorder.Root()
	if root == nil {
		t.Fatalf("expected provenance trace step to be recorded")
	}
	if root.RuleID != "TRS_RULE_APPLY" {
		t.Errorf("expected RuleID 'TRS_RULE_APPLY', got %q", root.RuleID)
	}
}
