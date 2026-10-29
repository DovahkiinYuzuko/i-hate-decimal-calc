package calc

import (
	"strings"
	"testing"
)

func TestExtractDomainConditions_RationalDenominator(t *testing.T) {
	// 1 / (x - 1) -> condition: x - 1 != 0
	expr, err := Parse("1 / (x - 1)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	conds := ExtractDomainConditions(expr)
	if conds.Len() != 1 {
		t.Fatalf("expected 1 condition, got %d: %s", conds.Len(), conds.String())
	}
	c := conds.Conditions()[0]
	if c.Kind != CondNonZero {
		t.Errorf("expected CondNonZero, got %v", c.Kind)
	}
	if !strings.Contains(c.Expr.String(), "x") {
		t.Errorf("expected expression to contain 'x', got: %s", c.Expr.String())
	}
}

func TestExtractDomainConditions_ConstantDenominatorIgnored(t *testing.T) {
	// x / 2 -> constant denominator 2 should not produce condition
	expr, err := Parse("x / 2")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	conds := ExtractDomainConditions(expr)
	if conds.Len() != 0 {
		t.Fatalf("expected 0 conditions for constant denominator, got %d: %s", conds.Len(), conds.String())
	}
}

func TestExtractDomainConditions_SqrtAndLog(t *testing.T) {
	// sqrt(x) + log(y) -> x >= 0, y > 0
	expr, err := Parse("sqrt(x) + log(y)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	conds := ExtractDomainConditions(expr)
	if conds.Len() != 2 {
		t.Fatalf("expected 2 conditions, got %d: %s", conds.Len(), conds.String())
	}

	str := conds.String()
	if !strings.Contains(str, "x >= 0") {
		t.Errorf("expected 'x >= 0', got: %s", str)
	}
	if !strings.Contains(str, "y > 0") {
		t.Errorf("expected 'y > 0', got: %s", str)
	}
}

func TestSimplifyWithConditions_Rational(t *testing.T) {
	expr, err := Parse("1 / (x - 1)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	res, conds, err := SimplifyWithConditions(expr, NewEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if conds.Len() != 1 {
		t.Fatalf("expected 1 condition, got %d: %s", conds.Len(), conds.String())
	}
}
