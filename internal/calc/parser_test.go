package calc

import (
	"strings"
	"testing"
)

// TestParser_PrecedenceAndAssociativity validates operator precedence and associativity.
func TestParser_PrecedenceAndAssociativity(t *testing.T) {
	// 1 + 2 * 3 -> 1 + (2 * 3)
	node, err := Parse("1 + 2 * 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	add, ok := node.(*AddNode)
	if !ok || len(add.Terms) != 2 {
		t.Fatalf("expected AddNode with 2 terms, got %T: %s", node, node.String())
	}
	if _, ok := add.Terms[1].(*MulNode); !ok {
		t.Errorf("expected second term to be MulNode, got %T", add.Terms[1])
	}

	// 2^3^2 -> Right associative: 2^(3^2)
	node, err = Parse("2 ^ 3 ^ 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pow, ok := node.(*PowNode)
	if !ok {
		t.Fatalf("expected PowNode, got %T", node)
	}
	if pow.Base.String() != "2" {
		t.Errorf("expected base 2, got %s", pow.Base.String())
	}
	if _, ok := pow.Exp.(*PowNode); !ok {
		t.Errorf("expected exponent to be PowNode (right associative), got %T: %s", pow.Exp, pow.Exp.String())
	}

	// -3^2 -> Unary minus has lower precedence than power: -(3^2)
	node, err = Parse("-3^2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	uop, ok := node.(*UnaryOpNode)
	if !ok || uop.Op != "-" {
		t.Fatalf("expected unary '-' at root, got %T: %s", node, node.String())
	}
	if _, ok := uop.Expr.(*PowNode); !ok {
		t.Errorf("expected inner expr to be PowNode (3^2), got %T: %s", uop.Expr, uop.Expr.String())
	}

	// 2^3! -> Factorial has higher precedence than power: 2^(3!)
	node, err = Parse("2^3!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pow, ok = node.(*PowNode)
	if !ok {
		t.Fatalf("expected PowNode, got %T", node)
	}
	if bang, ok := pow.Exp.(*UnaryOpNode); !ok || bang.Op != "!" {
		t.Errorf("expected exponent to be 3!, got %T: %s", pow.Exp, pow.Exp.String())
	}
}

// TestParser_DecimalParsing validates parsing decimals directly into RationalNode.
func TestParser_DecimalParsing(t *testing.T) {
	node, err := Parse("4 + 4 * 6.441")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	add, ok := node.(*AddNode)
	if !ok {
		t.Fatalf("expected AddNode, got %T", node)
	}
	mul, ok := add.Terms[1].(*MulNode)
	if !ok {
		t.Fatalf("expected MulNode, got %T", add.Terms[1])
	}
	rat, ok := mul.Factors[1].(*RationalNode)
	if !ok {
		t.Fatalf("expected RationalNode, got %T", mul.Factors[1])
	}
	if rat.String() != "6441/1000" {
		t.Errorf("expected 6441/1000, got %s", rat.String())
	}
}

// TestParser_FunctionCallsAndAliases validates function parsing and Unicode aliases.
func TestParser_FunctionCallsAndAliases(t *testing.T) {
	// sqrt(4) vs √4
	node1, err := Parse("sqrt(4)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	node2, err := Parse("√(4)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !node1.Equal(node2) {
		t.Errorf("expected sqrt(4) and √(4) to be equal, got %s vs %s", node1.String(), node2.String())
	}

	// sin(pi/6)
	node, err := Parse("sin(pi / 6)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fn, ok := node.(*FuncNode)
	if !ok || fn.Name != "sin" || len(fn.Args) != 1 {
		t.Fatalf("expected sin(...) FuncNode, got %T: %s", node, node.String())
	}

	// log(2, 4)
	node, err = Parse("log(2, 4)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fn, ok = node.(*FuncNode)
	if !ok || fn.Name != "log" || len(fn.Args) != 2 {
		t.Fatalf("expected log(2, 4) FuncNode, got %T: %s", node, node.String())
	}
}

// TestParser_ImplicitMultiplication_ProhibitedError validates that omitted '*' triggers syntax error.
func TestParser_ImplicitMultiplication_ProhibitedError(t *testing.T) {
	testCases := []string{
		"2pi",
		"2*pi*(1+2)(3+4)",
		"3sqrt(2)",
		"(1+2)(3+4)",
		"2(3)",
	}

	for _, tc := range testCases {
		_, err := Parse(tc)
		if err == nil {
			t.Errorf("expected error for omitted '*' in %q, got nil", tc)
		} else if !strings.Contains(err.Error(), "*") && !strings.Contains(err.Error(), "syntax") && !strings.Contains(err.Error(), "unexpected") {
			t.Logf("got error for %q: %v", tc, err)
		}
	}
}

// TestParser_SpecificationExample validates the complex expression from specification line 3.
func TestParser_SpecificationExample(t *testing.T) {
	// "563*76^2/3*sqrt(4)/5.888"
	input := "563*76^2/3*sqrt(4)/5.888"
	node, err := Parse(input)
	if err != nil {
		t.Fatalf("failed to parse specification example: %v", err)
	}
	if node == nil {
		t.Fatal("expected non-nil node")
	}
}

// TestParser_SyntaxErrors validates handling of unbalanced parens and trailing operators.
func TestParser_SyntaxErrors(t *testing.T) {
	testCases := []string{
		"((1 + 2)",
		"(1 + 2))",
		"1 + ",
		"* 5",
		"",
	}

	for _, tc := range testCases {
		_, err := Parse(tc)
		if err == nil {
			t.Errorf("expected syntax error for %q, got nil", tc)
		}
	}
}
