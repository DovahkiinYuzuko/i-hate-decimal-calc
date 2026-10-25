package calc

import (
	"testing"
)

func TestCompareExpressions_Equivalence(t *testing.T) {
	env := NewEnv()

	// Identical
	n1, err := Parse("(x + 1)^2")
	if err != nil {
		t.Fatal(err)
	}
	n2, err := Parse("(x + 1)^2")
	if err != nil {
		t.Fatal(err)
	}
	status, err := CompareExpressions(n1, n2, env)
	if err != nil {
		t.Fatalf("CompareExpressions failed: %v", err)
	}
	if status != DiffIdentical {
		t.Errorf("expected DiffIdentical, got %v", status)
	}

	// Semantically equivalent (expanded vs factored)
	n3, err := Parse("x^2 + 2*x + 1")
	if err != nil {
		t.Fatal(err)
	}
	status, err = CompareExpressions(n1, n3, env)
	if err != nil {
		t.Fatalf("CompareExpressions failed: %v", err)
	}
	if status != DiffEquivalent {
		t.Errorf("expected DiffEquivalent, got %v", status)
	}

	// Semantically changed
	n4, err := Parse("x^2 + 2*x + 2")
	if err != nil {
		t.Fatal(err)
	}
	status, err = CompareExpressions(n1, n4, env)
	if err != nil {
		t.Fatalf("CompareExpressions failed: %v", err)
	}
	if status != DiffChanged {
		t.Errorf("expected DiffChanged, got %v", status)
	}
}

func TestCompareScripts_StatementDiff(t *testing.T) {
	env := NewEnv()

	leftScript := `
a = 10
f = (x + 1)^2
b = 20
`
	rightScript := `
a = 10
f = x^2 + 2*x + 1
b = 25
c = 30
`

	diffs, err := CompareScripts(leftScript, rightScript, env)
	if err != nil {
		t.Fatalf("CompareScripts failed: %v", err)
	}

	if len(diffs) == 0 {
		t.Fatal("expected diffs, got empty")
	}

	foundEq := false
	foundChanged := false
	foundAdded := false

	for _, d := range diffs {
		if d.Status == DiffEquivalent && d.LeftRaw != "" && d.RightRaw != "" {
			foundEq = true
		}
		if d.Status == DiffChanged && d.LeftRaw != "" && d.RightRaw != "" {
			foundChanged = true
		}
		if d.Status == DiffAdded {
			foundAdded = true
		}
	}

	if !foundEq {
		t.Error("expected to find DiffEquivalent for f")
	}
	if !foundChanged {
		t.Error("expected to find DiffChanged for b")
	}
	if !foundAdded {
		t.Error("expected to find DiffAdded for c")
	}
}

func TestCompareScripts_RelationalEqualityNotAssignment(t *testing.T) {
	env := NewEnv()

	leftScript := `
x == 2
`
	rightScript := `
x == 2
`
	diffs, err := CompareScripts(leftScript, rightScript, env)
	if err != nil {
		t.Fatalf("CompareScripts failed: %v", err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Status != DiffIdentical {
		t.Errorf("expected DiffIdentical for 'x == 2', got %v", diffs[0].Status)
	}

	// Equivalent relations: x - 2 == 0 vs x == 2
	leftRel := `x - 2 == 0`
	rightRel := `x == 2`
	diffsRel, err := CompareScripts(leftRel, rightRel, env)
	if err != nil {
		t.Fatalf("CompareScripts failed: %v", err)
	}
	if len(diffsRel) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffsRel))
	}
	if diffsRel[0].Status != DiffEquivalent {
		t.Errorf("expected DiffEquivalent for 'x - 2 == 0' and 'x == 2', got %v", diffsRel[0].Status)
	}
}
