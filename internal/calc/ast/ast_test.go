package ast

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestRationalNode_Success(t *testing.T) {
	r, err := NewRational(4, 6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Type() != NodeRational {
		t.Errorf("expected NodeRational, got %v", r.Type())
	}
	expected := "2/3"
	if r.String() != expected {
		t.Errorf("expected %s, got %s", expected, r.String())
	}

	rInt, err := NewRational(5, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rInt.String() != "5" {
		t.Errorf("expected '5', got %s", rInt.String())
	}
}

func TestRationalNode_ZeroDenominator_Error(t *testing.T) {
	_, err := NewRational(1, 0)
	if err == nil {
		t.Fatal("expected error for zero denominator, got nil")
	}
}

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

func TestFuncNode_ValidationHook(t *testing.T) {
	// Without validator, NewFunc succeeds
	FuncValidator = nil
	f, err := NewFunc("custom", []Node{NewVar("x")})
	if err != nil {
		t.Fatalf("expected no error without validator, got %v", err)
	}
	if f.String() != "custom(x)" {
		t.Errorf("expected 'custom(x)', got %s", f.String())
	}

	// With validator hook returning error
	FuncValidator = func(name string, args []Node) error {
		if name == "invalid_func" {
			return fmt.Errorf("rejected func: %s", name)
		}
		return nil
	}
	defer func() { FuncValidator = nil }()

	_, err = NewFunc("invalid_func", []Node{NewVar("x")})
	if err == nil {
		t.Fatal("expected error from validator hook, got nil")
	}
}

func TestUnaryOp_FactorialErrors(t *testing.T) {
	neg, _ := NewRational(-3, 1)
	frac, _ := NewRational(3, 2)
	pos, _ := NewRational(4, 1)

	_, err := NewUnaryOp("!", neg)
	if err == nil {
		t.Fatal("expected error for factorial of negative integer, got nil")
	}

	_, err = NewUnaryOp("!", frac)
	if err == nil {
		t.Fatal("expected error for factorial of non-integer rational, got nil")
	}

	op, err := NewUnaryOp("!", pos)
	if err != nil {
		t.Fatalf("expected no error for 4!, got %v", err)
	}
	if op.String() != "4!" {
		t.Errorf("expected '4!', got %s", op.String())
	}
}

func TestPow_ZeroToNegative_Error(t *testing.T) {
	zero, _ := NewRational(0, 1)
	neg, _ := NewRational(-2, 1)

	_, err := NewPow(zero, neg)
	if err == nil {
		t.Fatal("expected error for 0^(negative), got nil")
	}
}

func TestNaryFlattening(t *testing.T) {
	n1, _ := NewRational(1, 1)
	n2, _ := NewRational(2, 1)
	n3, _ := NewRational(3, 1)

	innerAdd := NewAdd([]Node{n1, n2})
	outerAdd := NewAdd([]Node{innerAdd, n3})
	if len(outerAdd.Terms) != 3 {
		t.Fatalf("expected 3 flattened terms, got %d", len(outerAdd.Terms))
	}

	innerMul := NewMul([]Node{n1, n2})
	outerMul := NewMul([]Node{innerMul, n3})
	if len(outerMul.Factors) != 3 {
		t.Fatalf("expected 3 flattened factors, got %d", len(outerMul.Factors))
	}
}

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

func TestNodeEqual(t *testing.T) {
	r1, _ := NewRational(2, 3)
	r2, _ := NewRational(4, 6)
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

func TestAST_WalkAndInspect_ManualTree(t *testing.T) {
	// Build tree: sin(x + 2*y) + sqrt(z)
	two, _ := NewRational(2, 1)
	innerMul := NewMul([]Node{two, NewVar("y")})
	innerAdd := NewAdd([]Node{NewVar("x"), innerMul})
	sinFunc := &FuncNode{Name: "sin", Args: []Node{innerAdd}}
	sqrtNode := NewSqrt(NewVar("z"))
	rootAdd := NewAdd([]Node{sinFunc, sqrtNode})

	visitedTypes := make(map[NodeType]int)
	Inspect(rootAdd, func(n Node) {
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
	Walk(rootAdd, func(n Node) bool {
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

func TestAST_TransformAndSubstitute_ManualTree(t *testing.T) {
	// Build tree: x^2 + 2*x + 1
	two, _ := NewRational(2, 1)
	one, _ := NewRational(1, 1)
	pow := &PowNode{Base: NewVar("x"), Exp: two}
	mul := NewMul([]Node{two, NewVar("x")})
	poly := NewAdd([]Node{pow, mul, one})

	val, _ := NewRational(3, 1)
	subbed := Substitute(poly, "x", val)

	str := subbed.String()
	if strings.Contains(str, "x") {
		t.Errorf("expected no 'x' in subbed expression, got: %s", str)
	}
}

func TestAST_ContainsVarAndExtractFreeVariables_ManualTree(t *testing.T) {
	// Build tree: x^2 + 2*y + sin(z)
	two, _ := NewRational(2, 1)
	pow := &PowNode{Base: NewVar("x"), Exp: two}
	mul := NewMul([]Node{two, NewVar("y")})
	sinFunc := &FuncNode{Name: "sin", Args: []Node{NewVar("z")}}
	poly := NewAdd([]Node{pow, mul, sinFunc})

	if !ContainsVar(poly, "x") || !ContainsVar(poly, "y") || !ContainsVar(poly, "z") {
		t.Errorf("expected ContainsVar for x, y, z to be true")
	}
	if ContainsVar(poly, "w") {
		t.Errorf("expected ContainsVar for w to be false")
	}

	vars := ExtractFreeVariables(poly)
	if len(vars) != 3 || vars[0] != "x" || vars[1] != "y" || vars[2] != "z" {
		t.Errorf("ExtractFreeVariables = %v, want [x, y, z]", vars)
	}

	rad, _ := NewRational(2, 1)
	half, _ := NewRational(1, 2)
	constExpr := NewAdd([]Node{half, NewSqrt(rad)})
	if ContainsVar(constExpr, "x") {
		t.Errorf("expected false for constExpr")
	}
	if len(ExtractFreeVariables(constExpr)) != 0 {
		t.Errorf("expected empty vars for constExpr")
	}
}

func TestPolyNode_BasicOperations(t *testing.T) {
	// Poly: 3*x^2*y - 2*y + 5
	p := &PolyNode{
		Vars:  []string{"x", "y"},
		Order: OrderLex,
		Terms: []Monomial{
			{Coeff: big.NewRat(3, 1), Exponents: []int{2, 1}},
			{Coeff: big.NewRat(-2, 1), Exponents: []int{0, 1}},
			{Coeff: big.NewRat(5, 1), Exponents: []int{0, 0}},
		},
	}

	if p.Type() != NodePoly {
		t.Errorf("expected NodePoly, got %v", p.Type())
	}

	str := p.String()
	expected := "3*x^2*y - 2*y + 5"
	if str != expected {
		t.Errorf("p.String() = %q, want %q", str, expected)
	}

	cloned := p.Clone()
	if !p.Equal(cloned) {
		t.Errorf("expected cloned PolyNode to equal original")
	}

	// ContainsVar & ExtractFreeVariables
	if !ContainsVar(p, "x") || !ContainsVar(p, "y") {
		t.Errorf("expected ContainsVar to find x and y")
	}
	if ContainsVar(p, "z") {
		t.Errorf("expected ContainsVar not to find z")
	}

	vars := ExtractFreeVariables(p)
	if len(vars) != 2 || vars[0] != "x" || vars[1] != "y" {
		t.Errorf("ExtractFreeVariables(p) = %v, want [x, y]", vars)
	}

	// Zero poly
	zeroP := &PolyNode{Vars: []string{"x"}, Order: OrderLex, Terms: nil}
	if zeroP.String() != "0" {
		t.Errorf("zeroP.String() = %q, want \"0\"", zeroP.String())
	}
}

