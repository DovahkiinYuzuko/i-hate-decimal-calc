package main

import (
	"math/big"
	"testing"
)

// TestRationalNode_Success validates creation and string representation of rational numbers.
func TestRationalNode_Success(t *testing.T) {
	r, err := NewRational(4, 6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Type() != NodeRational {
		t.Errorf("expected NodeRational, got %v", r.Type())
	}
	// 4/6 should be automatically reduced to 2/3
	expected := "2/3"
	if r.String() != expected {
		t.Errorf("expected %s, got %s", expected, r.String())
	}

	// Integer representation (denominator is 1)
	rInt, err := NewRational(5, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rInt.String() != "5" {
		t.Errorf("expected '5', got %s", rInt.String())
	}
}

// TestRationalNode_ZeroDenominator_Error validates error handling when denominator is zero.
func TestRationalNode_ZeroDenominator_Error(t *testing.T) {
	_, err := NewRational(1, 0)
	if err == nil {
		t.Fatal("expected error for zero denominator, got nil")
	}
}

// TestSqrtNode validates sqrt node creation and string representation.
func TestSqrtNode(t *testing.T) {
	rad, _ := NewRational(2, 1)
	s := NewSqrt(rad)
	if s.Type() != NodeSqrt {
		t.Errorf("expected NodeSqrt, got %v", s.Type())
	}
	if s.String() != "sqrt(2)" {
		t.Errorf("expected 'sqrt(2)', got %s", s.String())
	}
}

// TestConstNode_SuccessAndError validates pi and e constants, and rejects unknown ones.
func TestConstNode_SuccessAndError(t *testing.T) {
	pi, err := NewConst("pi")
	if err != nil {
		t.Fatalf("expected no error for pi, got %v", err)
	}
	if pi.String() != "pi" {
		t.Errorf("expected 'pi', got %s", pi.String())
	}

	e, err := NewConst("e")
	if err != nil {
		t.Fatalf("expected no error for e, got %v", err)
	}
	if e.String() != "e" {
		t.Errorf("expected 'e', got %s", e.String())
	}

	_, err = NewConst("unknown_const")
	if err == nil {
		t.Fatal("expected error for unknown constant, got nil")
	}
}

// TestFuncNode_LogDomainErrors validates error detection on logarithmic domain violations.
func TestFuncNode_LogDomainErrors(t *testing.T) {
	zero, _ := NewRational(0, 1)
	negative, _ := NewRational(-5, 1)
	one, _ := NewRational(1, 1)
	two, _ := NewRational(2, 1)

	// log(0) -> Error
	_, err := NewFunc("log", []Node{zero})
	if err == nil {
		t.Fatal("expected error for log(0), got nil")
	}

	// log(-5) -> Error
	_, err = NewFunc("log", []Node{negative})
	if err == nil {
		t.Fatal("expected error for log(-5), got nil")
	}

	// log_1(2) (base is 1) -> Error
	_, err = NewFunc("log", []Node{one, two})
	if err == nil {
		t.Fatal("expected error for log with base 1, got nil")
	}

	// log_(-2)(2) (base <= 0) -> Error
	_, err = NewFunc("log", []Node{negative, two})
	if err == nil {
		t.Fatal("expected error for log with negative base, got nil")
	}

	// Unknown function -> Error
	_, err = NewFunc("unknown_func", []Node{two})
	if err == nil {
		t.Fatal("expected error for unknown function name, got nil")
	}
}

// TestUnaryOp_FactorialErrors validates error detection on factorial with negative or non-integer numbers.
func TestUnaryOp_FactorialErrors(t *testing.T) {
	neg, _ := NewRational(-3, 1)
	frac, _ := NewRational(3, 2)
	pos, _ := NewRational(4, 1)

	// -3! -> Error
	_, err := NewUnaryOp("!", neg)
	if err == nil {
		t.Fatal("expected error for factorial of negative integer, got nil")
	}

	// (3/2)! -> Error
	_, err = NewUnaryOp("!", frac)
	if err == nil {
		t.Fatal("expected error for factorial of non-integer rational, got nil")
	}

	// 4! -> Success
	op, err := NewUnaryOp("!", pos)
	if err != nil {
		t.Fatalf("expected no error for 4!, got %v", err)
	}
	if op.String() != "4!" {
		t.Errorf("expected '4!', got %s", op.String())
	}
}

// TestPow_ZeroToNegative_Error validates 0^(negative) division by zero error.
func TestPow_ZeroToNegative_Error(t *testing.T) {
	zero, _ := NewRational(0, 1)
	neg, _ := NewRational(-2, 1)

	_, err := NewPow(zero, neg)
	if err == nil {
		t.Fatal("expected error for 0^(negative), got nil")
	}
}

// TestNaryFlattening validates that AddNode and MulNode flatten nested additions/multiplications.
func TestNaryFlattening(t *testing.T) {
	n1, _ := NewRational(1, 1)
	n2, _ := NewRational(2, 1)
	n3, _ := NewRational(3, 1)

	// Nested Add: (1 + 2) + 3 should be flattened to [1, 2, 3]
	innerAdd := NewAdd([]Node{n1, n2})
	outerAdd := NewAdd([]Node{innerAdd, n3})

	if len(outerAdd.Terms) != 3 {
		t.Fatalf("expected 3 flattened terms, got %d", len(outerAdd.Terms))
	}

	// Nested Mul: (1 * 2) * 3 should be flattened to [1, 2, 3]
	innerMul := NewMul([]Node{n1, n2})
	outerMul := NewMul([]Node{innerMul, n3})

	if len(outerMul.Factors) != 3 {
		t.Fatalf("expected 3 flattened factors, got %d", len(outerMul.Factors))
	}
}

// TestComplexNode validates complex node representation.
func TestComplexNode(t *testing.T) {
	r, _ := NewRational(3, 1)
	i, _ := NewRational(4, 1)

	c := NewComplex(r, i)
	if c.Type() != NodeComplex {
		t.Errorf("expected NodeComplex, got %v", c.Type())
	}
	if c.String() != "3 + 4*i" {
		t.Errorf("expected '3 + 4*i', got %s", c.String())
	}
}

// TestNodeEqual validates deep structural equality comparison.
func TestNodeEqual(t *testing.T) {
	r1, _ := NewRational(2, 3)
	r2, _ := NewRational(4, 6) // simplifies to 2/3
	r3, _ := NewRational(5, 6)

	if !r1.Equal(r2) {
		t.Error("expected r1 and r2 to be equal")
	}
	if r1.Equal(r3) {
		t.Error("expected r1 and r3 to be not equal")
	}

	pi1, _ := NewConst("pi")
	pi2, _ := NewConst("pi")
	e, _ := NewConst("e")

	if !pi1.Equal(pi2) {
		t.Error("expected pi1 and pi2 to be equal")
	}
	if pi1.Equal(e) {
		t.Error("expected pi and e to be not equal")
	}
}

// Suppress unused big.Rat import
var _ = big.NewRat
