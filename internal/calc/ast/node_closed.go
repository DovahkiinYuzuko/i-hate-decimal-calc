package ast

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// PadicFormatter formats a PadicNode into canonical p-adic series notation.
// Injected by calc package init() to reuse padic.ExpansionString without import cycles.
var PadicFormatter func(n *PadicNode) string

// PadicNode represents a first-class p-adic number in Q_p modulo O(p^(v+r)).
type PadicNode struct {
	Prime     *big.Int
	Valuation int
	Unit      *big.Int
	Precision int
	IsZero    bool
}

// NewPadicNode validates, canonicalizes, and constructs an immutable PadicNode.
func NewPadicNode(prime *big.Int, valuation int, unit *big.Int, prec int, isZero bool) (*PadicNode, error) {
	if prime == nil || prime.Cmp(big.NewInt(2)) < 0 {
		return nil, newDomainErr("padic.err_prime_less_than_2", "")
	}
	pCopy := new(big.Int).Set(prime)

	if isZero || unit == nil {
		return &PadicNode{
			Prime:     pCopy,
			Valuation: 0,
			Unit:      big.NewInt(0),
			Precision: prec,
			IsZero:    true,
		}, nil
	}

	if prec <= 0 {
		return nil, newDomainErr("padic.err_prec_positive", "")
	}

	pk := new(big.Int).Exp(pCopy, big.NewInt(int64(prec)), nil)
	u := new(big.Int).Mod(unit, pk)
	if u.Sign() < 0 {
		u.Add(u, pk)
	}

	if u.Sign() == 0 {
		return &PadicNode{
			Prime:     pCopy,
			Valuation: 0,
			Unit:      big.NewInt(0),
			Precision: valuation + prec,
			IsZero:    true,
		}, nil
	}

	rem := new(big.Int)
	extraV := 0
	for {
		rem.Mod(u, pCopy)
		if rem.Sign() == 0 {
			extraV++
			u.Div(u, pCopy)
		} else {
			break
		}
	}

	newPrec := prec - extraV
	if newPrec <= 0 {
		return &PadicNode{
			Prime:     pCopy,
			Valuation: 0,
			Unit:      big.NewInt(0),
			Precision: valuation + prec,
			IsZero:    true,
		}, nil
	}

	return &PadicNode{
		Prime:     pCopy,
		Valuation: valuation + extraV,
		Unit:      u,
		Precision: newPrec,
		IsZero:    false,
	}, nil
}

func (n *PadicNode) Type() NodeType { return NodePadic }

func (n *PadicNode) String() string {
	if n == nil {
		return "<nil padic>" // txthc-ignore
	}
	if PadicFormatter != nil {
		return PadicFormatter(n)
	}
	if n.IsZero {
		if n.Precision < 0 {
			return fmt.Sprintf("0 + O(%s^(%d))", n.Prime.String(), n.Precision)
		}
		return fmt.Sprintf("0 + O(%s^%d)", n.Prime.String(), n.Precision)
	}
	absPrec := n.Valuation + n.Precision
	return fmt.Sprintf("padic(%s, %d, %s, %d)", n.Prime.String(), n.Valuation, n.Unit.String(), absPrec)
}

func (n *PadicNode) Equal(other Node) bool {
	if n == nil {
		return false
	}
	o, ok := other.(*PadicNode)
	if !ok || o == nil {
		return false
	}
	if n.Prime.Cmp(o.Prime) != 0 || n.IsZero != o.IsZero || n.Precision != o.Precision {
		return false
	}
	if n.IsZero {
		return true
	}
	return n.Valuation == o.Valuation && n.Unit.Cmp(o.Unit) == 0
}

// Clone returns a deep copy of PadicNode.
func (n *PadicNode) Clone() *PadicNode {
	if n == nil {
		return nil
	}
	return &PadicNode{
		Prime:     new(big.Int).Set(n.Prime),
		Valuation: n.Valuation,
		Unit:      new(big.Int).Set(n.Unit),
		Precision: n.Precision,
		IsZero:    n.IsZero,
	}
}

// PuiseuxTermAST represents a single rational-coefficient term c * x^(p/q) in a PuiseuxNode.
type PuiseuxTermAST struct {
	Coeff *big.Rat
	Exp   *big.Rat
}

// PuiseuxNode represents a canonical univariate Puiseux series sum c_k * Var^(e_k) + O(Var^TruncationOrder).
type PuiseuxNode struct {
	Var               string
	Terms             []PuiseuxTermAST
	TruncationOrder   *big.Rat
	RamificationIndex int
}

// NewPuiseuxNode constructs a normalized PuiseuxNode by merging duplicate exponents,
// stripping zero coefficients and terms at or above truncOrder, sorting exponents ascending,
// and computing the canonical RamificationIndex as the LCM of surviving exponent denominators.
func NewPuiseuxNode(varName string, terms []PuiseuxTermAST, truncOrder *big.Rat) (*PuiseuxNode, error) {
	if varName == "" {
		return nil, newDomainErrKey("puiseux.err_arg_must_be_variable", "<empty>")
	}
	if truncOrder == nil {
		return nil, newDomainErrKey("puiseux.err_order_must_be_positive_int", "<nil>")
	}

	tOrder := new(big.Rat).Set(truncOrder)
	merged := make(map[string]PuiseuxTermAST, len(terms))

	for _, t := range terms {
		if t.Coeff == nil || t.Coeff.Sign() == 0 || t.Exp == nil {
			continue
		}
		if t.Exp.Cmp(tOrder) >= 0 {
			continue
		}
		key := t.Exp.RatString()
		if existing, ok := merged[key]; ok {
			existing.Coeff.Add(existing.Coeff, t.Coeff)
			if existing.Coeff.Sign() == 0 {
				delete(merged, key)
			}
		} else {
			merged[key] = PuiseuxTermAST{
				Coeff: new(big.Rat).Set(t.Coeff),
				Exp:   new(big.Rat).Set(t.Exp),
			}
		}
	}

	sorted := make([]PuiseuxTermAST, 0, len(merged))
	for _, t := range merged {
		if t.Coeff.Sign() != 0 {
			sorted = append(sorted, t)
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Exp.Cmp(sorted[j].Exp) < 0
	})

	ramIdx := big.NewInt(1)
	g := new(big.Int)
	for _, t := range sorted {
		d := t.Exp.Denom()
		if d.Cmp(big.NewInt(1)) > 0 {
			g.GCD(nil, nil, ramIdx, d)
			ramIdx.Div(ramIdx, g)
			ramIdx.Mul(ramIdx, d)
		}
	}

	eInt := 1
	if ramIdx.IsInt64() && ramIdx.Int64() >= 1 {
		eInt = int(ramIdx.Int64())
	}

	return &PuiseuxNode{
		Var:               varName,
		Terms:             sorted,
		TruncationOrder:   tOrder,
		RamificationIndex: eInt,
	}, nil
}

func (n *PuiseuxNode) Type() NodeType { return NodePuiseux }

func formatPuiseuxVarPow(varName string, exp *big.Rat) string {
	if exp.IsInt() {
		e := exp.Num()
		if e.Cmp(big.NewInt(1)) == 0 {
			return varName
		}
		if e.Sign() < 0 {
			return fmt.Sprintf("%s^(%s)", varName, e.String())
		}
		return fmt.Sprintf("%s^%s", varName, e.String())
	}
	return fmt.Sprintf("%s^(%s)", varName, exp.RatString())
}

func formatPuiseuxOrder(varName string, order *big.Rat) string {
	if order.IsInt() {
		if order.Sign() < 0 {
			return fmt.Sprintf("O(%s^(%s))", varName, order.Num().String())
		}
		return fmt.Sprintf("O(%s^%s)", varName, order.Num().String())
	}
	return fmt.Sprintf("O(%s^(%s))", varName, order.RatString())
}

func (n *PuiseuxNode) String() string {
	if n == nil {
		return "<nil puiseux>" // txthc-ignore
	}
	var sb strings.Builder
	for i, t := range n.Terms {
		isNeg := t.Coeff.Sign() < 0
		absCoeff := new(big.Rat).Abs(t.Coeff)

		var termStr string
		if t.Exp.Sign() == 0 {
			termStr = absCoeff.RatString()
		} else {
			powStr := formatPuiseuxVarPow(n.Var, t.Exp)
			if absCoeff.Cmp(big.NewRat(1, 1)) == 0 {
				termStr = powStr
			} else if !absCoeff.IsInt() {
				termStr = fmt.Sprintf("(%s)*%s", absCoeff.RatString(), powStr)
			} else {
				termStr = fmt.Sprintf("%s*%s", absCoeff.RatString(), powStr)
			}
		}

		if i == 0 {
			if isNeg {
				sb.WriteByte('-')
			}
			sb.WriteString(termStr)
		} else {
			if isNeg {
				sb.WriteString(" - ")
			} else {
				sb.WriteString(" + ")
			}
			sb.WriteString(termStr)
		}
	}

	if len(n.Terms) == 0 {
		sb.WriteString("0")
	}
	sb.WriteString(" + ")
	sb.WriteString(formatPuiseuxOrder(n.Var, n.TruncationOrder))
	return sb.String()
}

func (n *PuiseuxNode) Equal(other Node) bool {
	if n == nil {
		return false
	}
	o, ok := other.(*PuiseuxNode)
	if !ok || o == nil {
		return false
	}
	if n.Var != o.Var || n.RamificationIndex != o.RamificationIndex {
		return false
	}
	if n.TruncationOrder.Cmp(o.TruncationOrder) != 0 || len(n.Terms) != len(o.Terms) {
		return false
	}
	for i := range n.Terms {
		if n.Terms[i].Exp.Cmp(o.Terms[i].Exp) != 0 || n.Terms[i].Coeff.Cmp(o.Terms[i].Coeff) != 0 {
			return false
		}
	}
	return true
}

// Clone returns a deep copy of PuiseuxNode.
func (n *PuiseuxNode) Clone() *PuiseuxNode {
	if n == nil {
		return nil
	}
	termsCopy := make([]PuiseuxTermAST, len(n.Terms))
	for i, t := range n.Terms {
		termsCopy[i] = PuiseuxTermAST{
			Coeff: new(big.Rat).Set(t.Coeff),
			Exp:   new(big.Rat).Set(t.Exp),
		}
	}
	return &PuiseuxNode{
		Var:               n.Var,
		Terms:             termsCopy,
		TruncationOrder:   new(big.Rat).Set(n.TruncationOrder),
		RamificationIndex: n.RamificationIndex,
	}
}
