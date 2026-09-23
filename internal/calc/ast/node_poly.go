package ast

import (
	"fmt"
	"math/big"
	"strings"
)

// -------------------------------------------------------------------------
// PolyNode (Canonical Sparse Multivariate Polynomial)
// -------------------------------------------------------------------------

// MonomialOrder defines the term ordering for multivariate polynomials.
type MonomialOrder int

const (
	// OrderLex represents lexicographic term order.
	OrderLex MonomialOrder = iota
	// OrderGrevLex represents graded reverse lexicographic term order.
	OrderGrevLex
)

// Monomial represents a single term in a multivariate polynomial: Coeff * x1^e1 * x2^e2 * ...
type Monomial struct {
	Coeff     *big.Rat
	Exponents []int // Parallel to PolyNode.Vars
}

// Clone returns a deep copy of the Monomial.
func (m Monomial) Clone() Monomial {
	cp := Monomial{
		Coeff:     new(big.Rat).Set(m.Coeff),
		Exponents: make([]int, len(m.Exponents)),
	}
	copy(cp.Exponents, m.Exponents)
	return cp
}

// PolyNode represents a canonical sparse multivariate polynomial.
// Invariant: Terms are strictly sorted in descending order according to Order,
// with no zero coefficients and no duplicate exponent vectors.
type PolyNode struct {
	Vars  []string
	Order MonomialOrder
	Terms []Monomial
}

func (n *PolyNode) Type() NodeType { return NodePoly }

func (n *PolyNode) String() string {
	if len(n.Terms) == 0 {
		return "0"
	}
	var sb strings.Builder
	for i, t := range n.Terms {
		isNeg := t.Coeff.Sign() < 0
		absRat := new(big.Rat).Abs(t.Coeff)
		isOne := absRat.Cmp(big.NewRat(1, 1)) == 0

		isConst := true
		for _, exp := range t.Exponents {
			if exp != 0 {
				isConst = false
				break
			}
		}

		if i > 0 {
			if isNeg {
				sb.WriteString(" - ")
			} else {
				sb.WriteString(" + ")
			}
		} else {
			if isNeg {
				sb.WriteString("-")
			}
		}

		coeffStr := ""
		if isConst || !isOne {
			if absRat.IsInt() {
				coeffStr = absRat.Num().String()
			} else {
				coeffStr = "(" + absRat.String() + ")"
			}
		}

		termStr := ""
		for vIdx, exp := range t.Exponents {
			if exp == 0 {
				continue
			}
			vName := n.Vars[vIdx]
			if termStr != "" {
				termStr += "*"
			}
			if exp == 1 {
				termStr += vName
			} else {
				termStr += fmt.Sprintf("%s^%d", vName, exp)
			}
		}

		if coeffStr != "" && termStr != "" {
			sb.WriteString(coeffStr)
			sb.WriteString("*")
			sb.WriteString(termStr)
		} else if coeffStr != "" {
			sb.WriteString(coeffStr)
		} else if termStr != "" {
			sb.WriteString(termStr)
		} else {
			sb.WriteString("1")
		}
	}
	return sb.String()
}

func (n *PolyNode) Equal(other Node) bool {
	o, ok := other.(*PolyNode)
	if !ok {
		return false
	}
	if len(n.Vars) != len(o.Vars) || n.Order != o.Order || len(n.Terms) != len(o.Terms) {
		return false
	}
	for i, v := range n.Vars {
		if v != o.Vars[i] {
			return false
		}
	}
	for i, t := range n.Terms {
		ot := o.Terms[i]
		if t.Coeff.Cmp(ot.Coeff) != 0 {
			return false
		}
		if len(t.Exponents) != len(ot.Exponents) {
			return false
		}
		for j, exp := range t.Exponents {
			if exp != ot.Exponents[j] {
				return false
			}
		}
	}
	return true
}

func (n *PolyNode) Clone() *PolyNode {
	cp := &PolyNode{
		Vars:  make([]string, len(n.Vars)),
		Order: n.Order,
		Terms: make([]Monomial, len(n.Terms)),
	}
	copy(cp.Vars, n.Vars)
	for i, t := range n.Terms {
		cp.Terms[i] = t.Clone()
	}
	return cp
}

// -------------------------------------------------------------------------
// AlgebraicNumberNode (Element of Algebraic Number Field Q(alpha))
// -------------------------------------------------------------------------

// AlgebraicNumberNode represents an algebraic number in Q(alpha), defined modulo
// an irreducible monic polynomial MinPoly(alpha) = 0.
// RepPoly represents the canonical element as a polynomial in alpha with deg(RepPoly) < deg(MinPoly).
type AlgebraicNumberNode struct {
	MinPoly *PolyNode // Minimal polynomial m(x) where m(alpha) = 0
	RepPoly *PolyNode // Representative polynomial r(x) where element is r(alpha)
	Symbol  string    // Generator symbol, default "alpha"
}

func (n *AlgebraicNumberNode) Type() NodeType { return NodeAlgebraicNumber }

func (n *AlgebraicNumberNode) String() string {
	sym := n.Symbol
	if sym == "" {
		sym = "alpha"
	}
	repStr := "0"
	if n.RepPoly != nil {
		repStr = n.RepPoly.String()
	}
	minStr := "?"
	if n.MinPoly != nil {
		minStr = n.MinPoly.String()
	}
	return fmt.Sprintf("AlgNum(%s mod %s=0)", repStr, minStr)
}

func (n *AlgebraicNumberNode) Equal(other Node) bool {
	o, ok := other.(*AlgebraicNumberNode)
	if !ok {
		return false
	}
	if n.Symbol != o.Symbol {
		return false
	}
	if (n.MinPoly == nil) != (o.MinPoly == nil) {
		return false
	}
	if n.MinPoly != nil && !n.MinPoly.Equal(o.MinPoly) {
		return false
	}
	if (n.RepPoly == nil) != (o.RepPoly == nil) {
		return false
	}
	if n.RepPoly != nil && !n.RepPoly.Equal(o.RepPoly) {
		return false
	}
	return true
}

func (n *AlgebraicNumberNode) Clone() *AlgebraicNumberNode {
	cp := &AlgebraicNumberNode{
		Symbol: n.Symbol,
	}
	if n.MinPoly != nil {
		cp.MinPoly = n.MinPoly.Clone()
	}
	if n.RepPoly != nil {
		cp.RepPoly = n.RepPoly.Clone()
	}
	return cp
}
