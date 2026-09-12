package calc

import (
	"strings"
	"testing"
)

func TestRegistry_AllFunctionsRegistered(t *testing.T) {
	expectedFuncs := []string{
		"sqrt", "sin", "cos", "tan", "asin", "acos", "atan",
		"ln", "log", "abs", "cbrt",
		"gcd", "lcm", "mod", "perm", "comb", "rand",
		"expand", "diff", "solve", "taylor", "sum",
		"det", "inv", "transpose",
		"dot", "cross", "norm", "grad", "div", "curl",
		"line_intersect", "circle_intersect", "triangle_area", "triangle_centers",
		"binom", "hyper", "geom", "bayes", "expect", "variance", "stddev",
		"plot",
	}

	for _, name := range expectedFuncs {
		if !IsReservedFunc(name) {
			t.Errorf("expected function %q to be registered in FunctionRegistry", name)
		}
		spec, ok := LookupFunction(name)
		if !ok {
			t.Errorf("expected LookupFunction(%q) to succeed", name)
		}
		if spec.Name != name {
			t.Errorf("spec name mismatch: got %q, want %q", spec.Name, name)
		}
	}
}

func TestRegistry_AssignToReservedFunction_Fails(t *testing.T) {
	reservedToTest := []string{
		"sin", "plot", "binom", "hyper", "geom", "bayes", "expect", "variance", "stddev",
		"det", "diff", "solve",
	}

	for _, name := range reservedToTest {
		input := name + " = 10"
		_, err := ParseStatement(input)
		if err == nil {
			t.Errorf("expected assigning to reserved function %q (%s) to fail, but it succeeded", name, input)
		} else if !strings.Contains(err.Error(), "cannot assign to reserved identifier") {
			t.Errorf("expected 'cannot assign to reserved identifier' error for %s, got: %v", input, err)
		}
	}
}

func TestRegistry_MissingArguments_Fails(t *testing.T) {
	missingArgFuncs := []string{
		"sin", "cos", "plot", "bayes", "det", "diff", "solve",
	}

	for _, name := range missingArgFuncs {
		_, err := Parse(name)
		if err == nil {
			t.Errorf("expected bare function %q to fail with missing arguments, but succeeded", name)
		} else if !strings.Contains(err.Error(), "missing arguments") {
			t.Errorf("expected 'missing arguments' error for %q, got: %v", name, err)
		}
	}
}

func TestRegistry_BareDistributionSymbol_AllowedInExpression(t *testing.T) {
	// Distribution identifiers can appear as bare symbols inside arguments (e.g. expect(binom, 10, 1/2))
	distFuncs := []string{"binom", "geom", "hyper"}
	for _, name := range distFuncs {
		node, err := Parse(name)
		if err != nil {
			t.Errorf("expected bare distribution symbol %q to parse as VarNode, got err: %v", name, err)
		}
		if vn, ok := node.(*VarNode); !ok || vn.Name != name {
			t.Errorf("expected *VarNode with name %q, got: %T (%v)", name, node, node)
		}
	}
}

func TestRegistry_ValidateFuncArgs_Constraints(t *testing.T) {
	// Arity test
	rat1 := mustRational(1, 1)
	if err := ValidateFuncArgs("sin", []Node{}); err == nil {
		t.Errorf("expected sin with 0 args to fail")
	}
	if err := ValidateFuncArgs("sin", []Node{rat1, rat1}); err == nil {
		t.Errorf("expected sin with 2 args to fail")
	}
	if err := ValidateFuncArgs("sin", []Node{rat1}); err != nil {
		t.Errorf("expected sin with 1 arg to succeed, got: %v", err)
	}

	// Domain constraint tests
	// asin domain [-1, 1]
	rat2 := mustRational(2, 1)
	if err := ValidateFuncArgs("asin", []Node{rat2}); err == nil {
		t.Errorf("expected asin(2) to fail domain check")
	}

	// ln positive domain
	ratNeg := mustRational(-1, 1)
	if err := ValidateFuncArgs("ln", []Node{ratNeg}); err == nil {
		t.Errorf("expected ln(-1) to fail domain check")
	}

	// log base constraints
	if err := ValidateFuncArgs("log", []Node{rat1, rat2}); err == nil {
		t.Errorf("expected log(1, 2) with base=1 to fail")
	}
	if err := ValidateFuncArgs("log", []Node{ratNeg, rat2}); err == nil {
		t.Errorf("expected log(-1, 2) with base<0 to fail")
	}
}
