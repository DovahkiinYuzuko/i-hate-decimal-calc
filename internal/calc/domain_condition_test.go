package calc

import (
	"strings"
	"testing"
)

func TestDomainConditionSet_Deduplication(t *testing.T) {
	set := NewDomainConditionSet()
	x := &VarNode{Name: "x"}

	set.Add(CondNonZero, x)
	set.Add(CondNonZero, x)

	if set.Len() != 1 {
		t.Fatalf("expected 1 condition, got %d", set.Len())
	}

	conds := set.Conditions()
	if len(conds) != 1 {
		t.Fatalf("expected 1 condition slice, got %d", len(conds))
	}
	if conds[0].Kind != CondNonZero || !conds[0].Expr.Equal(x) {
		t.Errorf("unexpected condition: %v", conds[0])
	}
}

func TestDomainConditionSet_MultipleKinds(t *testing.T) {
	set := NewDomainConditionSet()
	x := &VarNode{Name: "x"}
	y := &VarNode{Name: "y"}

	set.Add(CondNonZero, x)
	set.Add(CondNonNegative, y)
	set.Add(CondPositive, x)

	if set.Len() != 3 {
		t.Fatalf("expected 3 conditions, got %d", set.Len())
	}

	str := set.String()
	if !strings.Contains(str, "x != 0") {
		t.Errorf("expected string to contain 'x != 0', got: %s", str)
	}
	if !strings.Contains(str, "y >= 0") {
		t.Errorf("expected string to contain 'y >= 0', got: %s", str)
	}
	if !strings.Contains(str, "x > 0") {
		t.Errorf("expected string to contain 'x > 0', got: %s", str)
	}
}
