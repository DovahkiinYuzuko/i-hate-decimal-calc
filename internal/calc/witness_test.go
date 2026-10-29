package calc

import (
	"strings"
	"testing"
)

func TestFindCounterexample_ValueMismatch(t *testing.T) {
	lhs, err := Parse("x^2")
	if err != nil {
		t.Fatalf("parse lhs: %v", err)
	}
	rhs, err := Parse("x")
	if err != nil {
		t.Fatalf("parse rhs: %v", err)
	}

	w, err := FindCounterexample(lhs, rhs, NewEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected counterexample witness, got nil")
	}
	if w.Kind != WitnessValueMismatch {
		t.Errorf("expected WitnessValueMismatch, got %v", w.Kind)
	}
	if len(w.Assignments) != 1 {
		t.Errorf("expected 1 assignment, got %d", len(w.Assignments))
	}
	if !strings.Contains(w.Message, "LHS") || !strings.Contains(w.Message, "RHS") {
		t.Errorf("expected message to describe LHS/RHS mismatch, got: %s", w.Message)
	}
}

func TestFindCounterexample_DomainMismatch(t *testing.T) {
	lhs, err := Parse("1 / (x - 1)")
	if err != nil {
		t.Fatalf("parse lhs: %v", err)
	}
	rhs, err := Parse("2")
	if err != nil {
		t.Fatalf("parse rhs: %v", err)
	}

	w, err := FindCounterexample(lhs, rhs, NewEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected counterexample witness, got nil")
	}
	if !strings.Contains(w.Message, "undefined") && !strings.Contains(w.Message, "Domain mismatch") && w.Kind != WitnessDomainMismatch && w.Kind != WitnessValueMismatch {
		t.Errorf("expected domain or value mismatch witness, got: %v (%s)", w.Kind, w.Message)
	}
}

func TestFindCounterexample_IdenticalHolds(t *testing.T) {
	lhs, err := Parse("x + x")
	if err != nil {
		t.Fatalf("parse lhs: %v", err)
	}
	rhs, err := Parse("2 * x")
	if err != nil {
		t.Fatalf("parse rhs: %v", err)
	}

	w, err := FindCounterexample(lhs, rhs, NewEnv())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != nil {
		t.Fatalf("expected no counterexample for identical expressions, got: %s", w.Message)
	}
}
