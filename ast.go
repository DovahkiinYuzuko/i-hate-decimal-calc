package main

import (
	"fmt"
	"math/big"
	"strings"
)

// NodeType identifies the specific kind of AST node.
type NodeType int

const (
	NodeRational NodeType = iota
	NodeSqrt
	NodeConst
	NodeFunc
	NodeComplex
	NodeAdd
	NodeMul
	NodePow
	NodeUnaryOp
)

// Node represents any node in the mathematical expression tree.
type Node interface {
	Type() NodeType
	String() string
	Equal(other Node) bool
}

// -------------------------------------------------------------------------
// RationalNode (Fraction)
// -------------------------------------------------------------------------

// RationalNode represents an exact rational number using big.Rat.
type RationalNode struct {
	Val *big.Rat
}

// NewRational creates a new RationalNode with the given numerator and denominator.
// Automatically reduces the fraction. Returns an error if denominator is zero.
func NewRational(num, denom int64) (*RationalNode, error) {
	if denom == 0 {
		return nil, fmt.Errorf("division by zero: denominator cannot be zero")
	}
	r := new(big.Rat).SetFrac64(num, denom)
	return &RationalNode{Val: r}, nil
}

// NewRationalFromBigRat wraps an existing big.Rat into a RationalNode.
func NewRationalFromBigRat(r *big.Rat) *RationalNode {
	copied := new(big.Rat).Set(r)
	return &RationalNode{Val: copied}
}

func (n *RationalNode) Type() NodeType { return NodeRational }

func (n *RationalNode) String() string {
	if n.Val.IsInt() {
		return n.Val.Num().String()
	}
	return n.Val.String()
}

func (n *RationalNode) Equal(other Node) bool {
	o, ok := other.(*RationalNode)
	if !ok {
		return false
	}
	return n.Val.Cmp(o.Val) == 0
}

// -------------------------------------------------------------------------
// SqrtNode (Square Root)
// -------------------------------------------------------------------------

// SqrtNode represents a square root symbol: sqrt(Radicand).
type SqrtNode struct {
	Radicand Node
}

// NewSqrt creates a new SqrtNode.
func NewSqrt(radicand Node) *SqrtNode {
	return &SqrtNode{Radicand: radicand}
}

func (n *SqrtNode) Type() NodeType { return NodeSqrt }

func (n *SqrtNode) String() string {
	return fmt.Sprintf("sqrt(%s)", n.Radicand.String())
}

func (n *SqrtNode) Equal(other Node) bool {
	o, ok := other.(*SqrtNode)
	if !ok {
		return false
	}
	return n.Radicand.Equal(o.Radicand)
}

// -------------------------------------------------------------------------
// ConstNode (Constants: pi, e)
// -------------------------------------------------------------------------

// ConstNode represents a mathematical constant symbol like pi or e.
type ConstNode struct {
	Name string
}

// NewConst creates a new ConstNode. Allowed names: "pi", "e".
func NewConst(name string) (*ConstNode, error) {
	switch name {
	case "pi", "e":
		return &ConstNode{Name: name}, nil
	default:
		return nil, fmt.Errorf("unknown constant: %s", name)
	}
}

func (n *ConstNode) Type() NodeType { return NodeConst }

func (n *ConstNode) String() string { return n.Name }

func (n *ConstNode) Equal(other Node) bool {
	o, ok := other.(*ConstNode)
	if !ok {
		return false
	}
	return n.Name == o.Name
}

// -------------------------------------------------------------------------
// FuncNode (Functions: sin, cos, tan, log, ln)
// -------------------------------------------------------------------------

// FuncNode represents a function call with one or more arguments.
type FuncNode struct {
	Name string
	Args []Node
}

// NewFunc creates and validates a function node.
func NewFunc(name string, args []Node) (*FuncNode, error) {
	switch name {
	case "sin", "cos", "tan", "ln":
		if len(args) != 1 {
			return nil, fmt.Errorf("%s requires exactly 1 argument, got %d", name, len(args))
		}
		if name == "ln" {
			if rat, ok := args[0].(*RationalNode); ok && rat.Val.Sign() <= 0 {
				return nil, fmt.Errorf("ln domain error: argument must be positive, got %s", rat.String())
			}
		}
		return &FuncNode{Name: name, Args: args}, nil

	case "log":
		if len(args) == 1 {
			// Common log (base 10): log(x)
			if rat, ok := args[0].(*RationalNode); ok && rat.Val.Sign() <= 0 {
				return nil, fmt.Errorf("log domain error: argument must be positive, got %s", rat.String())
			}
		} else if len(args) == 2 {
			// Arbitrary base: log(base, x)
			base := args[0]
			arg := args[1]
			if ratBase, ok := base.(*RationalNode); ok {
				if ratBase.Val.Sign() <= 0 {
					return nil, fmt.Errorf("log base error: base must be positive, got %s", ratBase.String())
				}
				one := big.NewRat(1, 1)
				if ratBase.Val.Cmp(one) == 0 {
					return nil, fmt.Errorf("log base error: base cannot be 1")
				}
			}
			if ratArg, ok := arg.(*RationalNode); ok && ratArg.Val.Sign() <= 0 {
				return nil, fmt.Errorf("log domain error: argument must be positive, got %s", ratArg.String())
			}
		} else {
			return nil, fmt.Errorf("log requires 1 or 2 arguments, got %d", len(args))
		}
		return &FuncNode{Name: name, Args: args}, nil

	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

func (n *FuncNode) Type() NodeType { return NodeFunc }

func (n *FuncNode) String() string {
	argStrs := make([]string, len(n.Args))
	for i, a := range n.Args {
		argStrs[i] = a.String()
	}
	return fmt.Sprintf("%s(%s)", n.Name, strings.Join(argStrs, ", "))
}

func (n *FuncNode) Equal(other Node) bool {
	o, ok := other.(*FuncNode)
	if !ok || n.Name != o.Name || len(n.Args) != len(o.Args) {
		return false
	}
	for i := range n.Args {
		if !n.Args[i].Equal(o.Args[i]) {
			return false
		}
	}
	return true
}

// -------------------------------------------------------------------------
// ComplexNode (Complex Number: a + b*i)
// -------------------------------------------------------------------------

// ComplexNode represents a complex number with real and imaginary parts.
type ComplexNode struct {
	Real Node
	Imag Node
}

// NewComplex creates a new ComplexNode.
func NewComplex(real, imag Node) *ComplexNode {
	return &ComplexNode{Real: real, Imag: imag}
}

func (n *ComplexNode) Type() NodeType { return NodeComplex }

func (n *ComplexNode) String() string {
	return fmt.Sprintf("%s + %s*i", n.Real.String(), n.Imag.String())
}

func (n *ComplexNode) Equal(other Node) bool {
	o, ok := other.(*ComplexNode)
	if !ok {
		return false
	}
	return n.Real.Equal(o.Real) && n.Imag.Equal(o.Imag)
}

// -------------------------------------------------------------------------
// AddNode (N-ary Addition)
// -------------------------------------------------------------------------

// AddNode represents an addition of multiple terms (N-ary flat tree).
type AddNode struct {
	Terms []Node
}

// NewAdd creates an AddNode and flattens nested additions according to associative law.
func NewAdd(terms []Node) *AddNode {
	var flattened []Node
	for _, t := range terms {
		if add, ok := t.(*AddNode); ok {
			flattened = append(flattened, add.Terms...)
		} else {
			flattened = append(flattened, t)
		}
	}
	return &AddNode{Terms: flattened}
}

func (n *AddNode) Type() NodeType { return NodeAdd }

func (n *AddNode) String() string {
	strs := make([]string, len(n.Terms))
	for i, t := range n.Terms {
		strs[i] = t.String()
	}
	return strings.Join(strs, " + ")
}

func (n *AddNode) Equal(other Node) bool {
	o, ok := other.(*AddNode)
	if !ok || len(n.Terms) != len(o.Terms) {
		return false
	}
	for i := range n.Terms {
		if !n.Terms[i].Equal(o.Terms[i]) {
			return false
		}
	}
	return true
}

// -------------------------------------------------------------------------
// MulNode (N-ary Multiplication)
// -------------------------------------------------------------------------

// MulNode represents a multiplication of multiple factors (N-ary flat tree).
type MulNode struct {
	Factors []Node
}

// NewMul creates a MulNode and flattens nested multiplications according to associative law.
func NewMul(factors []Node) *MulNode {
	var flattened []Node
	for _, f := range factors {
		if mul, ok := f.(*MulNode); ok {
			flattened = append(flattened, mul.Factors...)
		} else {
			flattened = append(flattened, f)
		}
	}
	return &MulNode{Factors: flattened}
}

func (n *MulNode) Type() NodeType { return NodeMul }

func (n *MulNode) String() string {
	if len(n.Factors) == 2 {
		if r, ok := n.Factors[0].(*RationalNode); ok && !r.Val.IsInt() {
			if r.Val.Num().Cmp(big.NewInt(1)) == 0 {
				// 1/d * X -> X/d
				return fmt.Sprintf("%s/%s", n.Factors[1].String(), r.Val.Denom().String())
			} else if r.Val.Num().Cmp(big.NewInt(-1)) == 0 {
				// -1/d * X -> -X/d
				return fmt.Sprintf("-%s/%s", n.Factors[1].String(), r.Val.Denom().String())
			}
		}
	}
	strs := make([]string, len(n.Factors))
	for i, f := range n.Factors {
		strs[i] = f.String()
	}
	return strings.Join(strs, " * ")
}

func (n *MulNode) Equal(other Node) bool {
	o, ok := other.(*MulNode)
	if !ok || len(n.Factors) != len(o.Factors) {
		return false
	}
	for i := range n.Factors {
		if !n.Factors[i].Equal(o.Factors[i]) {
			return false
		}
	}
	return true
}

// -------------------------------------------------------------------------
// PowNode (Power / Exponentiation)
// -------------------------------------------------------------------------

// PowNode represents Base^Exp.
type PowNode struct {
	Base Node
	Exp  Node
}

// NewPow creates a PowNode. Validates that 0^(negative) returns a division by zero error.
func NewPow(base, exp Node) (*PowNode, error) {
	if ratBase, ok := base.(*RationalNode); ok && ratBase.Val.Sign() == 0 {
		if ratExp, ok := exp.(*RationalNode); ok && ratExp.Val.Sign() < 0 {
			return nil, fmt.Errorf("division by zero: 0^(negative number) is undefined")
		}
	}
	return &PowNode{Base: base, Exp: exp}, nil
}

func (n *PowNode) Type() NodeType { return NodePow }

func (n *PowNode) String() string {
	return fmt.Sprintf("%s^%s", n.Base.String(), n.Exp.String())
}

func (n *PowNode) Equal(other Node) bool {
	o, ok := other.(*PowNode)
	if !ok {
		return false
	}
	return n.Base.Equal(o.Base) && n.Exp.Equal(o.Exp)
}

// -------------------------------------------------------------------------
// UnaryOpNode (Unary Operators: -, !)
// -------------------------------------------------------------------------

// UnaryOpNode represents a unary operation (negation '-' or factorial '!').
type UnaryOpNode struct {
	Op   string
	Expr Node
}

// NewUnaryOp creates a UnaryOpNode. Validates that factorial is applied only to non-negative integers.
func NewUnaryOp(op string, expr Node) (*UnaryOpNode, error) {
	switch op {
	case "-":
		return &UnaryOpNode{Op: op, Expr: expr}, nil

	case "!":
		if rat, ok := expr.(*RationalNode); ok {
			if !rat.Val.IsInt() {
				return nil, fmt.Errorf("factorial domain error: factorial of non-integer rational (%s) is undefined", rat.String())
			}
			if rat.Val.Sign() < 0 {
				return nil, fmt.Errorf("factorial domain error: factorial of negative integer (%s) is undefined", rat.String())
			}
		}
		return &UnaryOpNode{Op: op, Expr: expr}, nil

	default:
		return nil, fmt.Errorf("unknown unary operator: %s", op)
	}
}

func (n *UnaryOpNode) Type() NodeType { return NodeUnaryOp }

func (n *UnaryOpNode) String() string {
	if n.Op == "!" {
		return fmt.Sprintf("%s!", n.Expr.String())
	}
	return fmt.Sprintf("%s%s", n.Op, n.Expr.String())
}

func (n *UnaryOpNode) Equal(other Node) bool {
	o, ok := other.(*UnaryOpNode)
	if !ok || n.Op != o.Op {
		return false
	}
	return n.Expr.Equal(o.Expr)
}
