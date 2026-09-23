package ast

import (
	"fmt"
)

// Hooks for error factories and validation to allow decoupling from higher-level calc/errors/registry packages.
var (
	// FuncValidator checks function argument constraints. If nil, arguments are not validated.
	FuncValidator func(name string, args []Node) error

	// ZeroDivisionErrorFactory creates structured zero-division errors. If nil, standard fmt.Errorf is used.
	ZeroDivisionErrorFactory func(msg string) error

	// DomainErrorFactory creates structured domain errors. If nil, standard fmt.Errorf is used.
	DomainErrorFactory func(msgKey, fallback string, args ...any) error
)

func newZeroDivisionErr(msg string) error {
	if ZeroDivisionErrorFactory != nil {
		return ZeroDivisionErrorFactory(msg)
	}
	return fmt.Errorf("%s", msg)
}

func newDomainErr(msgKey, fallback string, args ...any) error {
	if DomainErrorFactory != nil {
		return DomainErrorFactory(msgKey, fallback, args...)
	}
	return fmt.Errorf(fallback, args...)
}

func isNegative(n Node) bool {
	if rat, ok := n.(*RationalNode); ok {
		return rat.Val.Sign() < 0
	}
	if uop, ok := n.(*UnaryOpNode); ok && uop.Op == "-" {
		return true
	}
	if mul, ok := n.(*MulNode); ok && len(mul.Factors) > 0 {
		if rat, ok := mul.Factors[0].(*RationalNode); ok {
			return rat.Val.Sign() < 0
		}
	}
	if c, ok := n.(*ComplexNode); ok {
		if isZero(c.Real) && isNegative(c.Imag) {
			return true
		}
	}
	return false
}

func isZero(n Node) bool {
	if rat, ok := n.(*RationalNode); ok {
		return rat.Val.Sign() == 0
	}
	return false
}

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
	NodeVar
	NodeList
	NodeMatrix
	NodePlot
	NodeRelOp
	NodePoly
	NodeAlgebraicNumber
	NodeQuantifier
)

// Node represents any node in the mathematical expression tree.
type Node interface {
	Type() NodeType
	String() string
	Equal(other Node) bool
}
