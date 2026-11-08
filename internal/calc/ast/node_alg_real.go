package ast

import (
	"fmt"
	"math/big"
	"strings"
)

// AlgebraicRealNode represents an isolated real algebraic number root of an irreducible
// integer polynomial Poly(x) within the rational interval [Low, High].
type AlgebraicRealNode struct {
	Poly      []*big.Int // Ascending order: a_0 + a_1 x + ... + a_d x^d
	Low       *big.Rat   // Lower bound of isolating interval
	High      *big.Rat   // Upper bound of isolating interval
	RootIndex int        // 1-based index among real roots
}

// NewAlgebraicRealNode creates a new AlgebraicRealNode.
func NewAlgebraicRealNode(poly []*big.Int, low, high *big.Rat, rootIndex int) *AlgebraicRealNode {
	polyCopy := make([]*big.Int, len(poly))
	for i, c := range poly {
		if c != nil {
			polyCopy[i] = new(big.Int).Set(c)
		} else {
			polyCopy[i] = big.NewInt(0)
		}
	}
	var lowCopy, highCopy *big.Rat
	if low != nil {
		lowCopy = new(big.Rat).Set(low)
	}
	if high != nil {
		highCopy = new(big.Rat).Set(high)
	}
	return &AlgebraicRealNode{
		Poly:      polyCopy,
		Low:       lowCopy,
		High:      highCopy,
		RootIndex: rootIndex,
	}
}

func (n *AlgebraicRealNode) Type() NodeType {
	return NodeAlgebraicReal
}

func (n *AlgebraicRealNode) String() string {
	var polyTerms []string
	for i := len(n.Poly) - 1; i >= 0; i-- {
		c := n.Poly[i]
		if c == nil || c.Sign() == 0 {
			continue
		}
		term := ""
		if i == 0 {
			term = c.String()
		} else if i == 1 {
			if c.Cmp(big.NewInt(1)) == 0 {
				term = "x"
			} else if c.Cmp(big.NewInt(-1)) == 0 {
				term = "-x"
			} else {
				term = fmt.Sprintf("%sx", c.String())
			}
		} else {
			if c.Cmp(big.NewInt(1)) == 0 {
				term = fmt.Sprintf("x^%d", i)
			} else if c.Cmp(big.NewInt(-1)) == 0 {
				term = fmt.Sprintf("-x^%d", i)
			} else {
				term = fmt.Sprintf("%sx^%d", c.String(), i)
			}
		}
		if len(polyTerms) > 0 && !strings.HasPrefix(term, "-") {
			term = "+" + term
		}
		polyTerms = append(polyTerms, term)
	}
	polyStr := strings.Join(polyTerms, "")
	if polyStr == "" {
		polyStr = "0"
	}
	lowStr := "?"
	if n.Low != nil {
		lowStr = n.Low.RatString()
	}
	highStr := "?"
	if n.High != nil {
		highStr = n.High.RatString()
	}
	return fmt.Sprintf("AlgReal(%s, [%s, %s], %d)", polyStr, lowStr, highStr, n.RootIndex)
}

func (n *AlgebraicRealNode) Equal(other Node) bool {
	o, ok := other.(*AlgebraicRealNode)
	if !ok {
		return false
	}
	if n.RootIndex != o.RootIndex {
		return false
	}
	if len(n.Poly) != len(o.Poly) {
		return false
	}
	for i := range n.Poly {
		if (n.Poly[i] == nil) != (o.Poly[i] == nil) {
			return false
		}
		if n.Poly[i] != nil && n.Poly[i].Cmp(o.Poly[i]) != 0 {
			return false
		}
	}
	if (n.Low == nil) != (o.Low == nil) {
		return false
	}
	if n.Low != nil && n.Low.Cmp(o.Low) != 0 {
		return false
	}
	if (n.High == nil) != (o.High == nil) {
		return false
	}
	if n.High != nil && n.High.Cmp(o.High) != 0 {
		return false
	}
	return true
}

// CFNode represents a continued fraction term stream generator.
type CFNode struct {
	Terms       []*big.Int // Evaluated continued fraction coefficients [a_0; a_1, a_2, ...]
	Generator   any        // Engine or generator function producing further terms
	IsExhausted bool       // True if finite continued fraction has completed
}

// NewCFNode creates a new CFNode.
func NewCFNode(terms []*big.Int, generator any, isExhausted bool) *CFNode {
	termsCopy := make([]*big.Int, len(terms))
	for i, t := range terms {
		if t != nil {
			termsCopy[i] = new(big.Int).Set(t)
		}
	}
	return &CFNode{
		Terms:       termsCopy,
		Generator:   generator,
		IsExhausted: isExhausted,
	}
}

func (n *CFNode) Type() NodeType {
	return NodeCF
}

func (n *CFNode) String() string {
	var parts []string
	for _, t := range n.Terms {
		if t != nil {
			parts = append(parts, t.String())
		}
	}
	body := strings.Join(parts, ", ")
	if !n.IsExhausted {
		if len(parts) > 0 {
			body += ", ..."
		} else {
			body = "..."
		}
	}
	return fmt.Sprintf("CF([%s])", body)
}

func (n *CFNode) Equal(other Node) bool {
	o, ok := other.(*CFNode)
	if !ok {
		return false
	}
	if n.IsExhausted != o.IsExhausted {
		return false
	}
	if len(n.Terms) != len(o.Terms) {
		return false
	}
	for i := range n.Terms {
		if (n.Terms[i] == nil) != (o.Terms[i] == nil) {
			return false
		}
		if n.Terms[i] != nil && n.Terms[i].Cmp(o.Terms[i]) != 0 {
			return false
		}
	}
	return true
}
