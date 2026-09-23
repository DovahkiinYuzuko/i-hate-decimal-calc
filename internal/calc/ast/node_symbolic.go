package ast

import (
	"fmt"
	"strings"
)

// -------------------------------------------------------------------------
// ConstNode (Constants: pi, e)
// -------------------------------------------------------------------------

// ConstNode represents a mathematical constant symbol like pi or e.
type ConstNode struct {
	Name string
}

// NewConst creates a new ConstNode. Allowed names: "pi", "e", "deg".
func NewConst(name string) (*ConstNode, error) {
	switch name {
	case "pi", "e", "deg":
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
	if FuncValidator != nil {
		if err := FuncValidator(name, args); err != nil {
			return nil, err
		}
	}
	return &FuncNode{Name: name, Args: args}, nil
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
// VarNode (Variable Symbol)
// -------------------------------------------------------------------------

// VarNode represents a variable symbol (e.g., "x", "ans").
type VarNode struct {
	Name string
}

// NewVar creates a new VarNode with the given name.
func NewVar(name string) *VarNode {
	return &VarNode{Name: name}
}

func (n *VarNode) Type() NodeType { return NodeVar }

func (n *VarNode) String() string {
	return n.Name
}

func (n *VarNode) Equal(other Node) bool {
	o, ok := other.(*VarNode)
	if !ok {
		return false
	}
	return n.Name == o.Name
}

// -------------------------------------------------------------------------
// AssignStmt (Variable Assignment Statement)
// -------------------------------------------------------------------------

// AssignStmt represents a variable assignment statement (e.g., "x = 1/2").
type AssignStmt struct {
	Name  string
	Value Node
}
