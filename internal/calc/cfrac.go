package calc

import (
	"fmt"
	"math"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// floorRat computes the mathematical floor of a rational number floor(p/q).
// It correctly rounds down for both positive and negative values.
func floorRat(r *big.Rat) *big.Int {
	num := r.Num()
	den := r.Denom()
	div := new(big.Int).Quo(num, den)
	rem := new(big.Int).Rem(num, den)
	if rem.Sign() != 0 && num.Sign() < 0 {
		div.Sub(div, big.NewInt(1))
	}
	return div
}

// cfracRational computes the finite regular continued fraction expansion of a rational number.
// Returns a ListNode [a0, a1, ..., an] where each ak is an integer.
func cfracRational(r *big.Rat) (*ListNode, error) {
	if r == nil {
		return nil, fmt.Errorf("cannot compute continued fraction of nil rational")
	}

	var elems []Node
	cur := new(big.Rat).Set(r)

	for {
		a := floorRat(cur)
		elems = append(elems, mustRational(a.Int64(), 1))

		aRat := new(big.Rat).SetInt(a)
		frac := new(big.Rat).Sub(cur, aRat)
		if frac.Sign() == 0 {
			break
		}
		cur.Inv(frac)
	}

	return &ListNode{Elements: elems}, nil
}

// cfracSqrt computes the periodic continued fraction expansion of sqrt(D).
// Returns [a0, [a1, a2, ..., am]] where a0 is the floor and [a1...am] is the repeating period.
func cfracSqrt(d int64) (*ListNode, error) {
	if d < 0 {
		return nil, fmt.Errorf("continued fraction of imaginary square root sqrt(%d) is not supported", d)
	}
	if d == 0 {
		return &ListNode{Elements: []Node{mustRational(0, 1)}}, nil
	}

	a0 := int64(math.Floor(math.Sqrt(float64(d))))
	if a0*a0 == d {
		// Perfect square
		return &ListNode{Elements: []Node{mustRational(a0, 1)}}, nil
	}

	m := int64(0)
	denom := int64(1)
	a := a0
	var period []Node

	// Continued fraction expansion of sqrt(d) algorithm
	const maxIterations = 2000
	for iter := 0; iter < maxIterations; iter++ {
		m = denom*a - m
		denom = (d - m*m) / denom
		a = (a0 + m) / denom
		period = append(period, mustRational(a, 1))
		if a == 2*a0 {
			break
		}
	}

	periodList := &ListNode{Elements: period}
	return &ListNode{Elements: []Node{mustRational(a0, 1), periodList}}, nil
}

// evalCFrac evaluates the cfrac(expr) function.
func evalCFrac(arg Node) (Node, error) {
	evaled, err := Eval(arg)
	if err != nil {
		return nil, err
	}

	switch v := evaled.(type) {
	case *RationalNode:
		return cfracRational(v.Val)
	case *SqrtNode:
		if rat, ok := v.Radicand.(*RationalNode); ok && rat.Val.IsInt() && rat.Val.Sign() >= 0 {
			return cfracSqrt(rat.Val.Num().Int64())
		}
	default:
		if sq, ok := extractRadicalSquare(evaled); ok && sq.IsInt() && sq.Sign() >= 0 {
			return cfracSqrt(sq.Num().Int64())
		}
	}

	return nil, fmt.Errorf("%s", i18n.T("errors.cfrac_unsupported"))
}

// evalFromCFrac evaluates from_cfrac(list), converting a continued fraction [a0, a1, ...]
// back into an exact rational number.
func evalFromCFrac(arg Node) (Node, error) {
	evaled, err := Eval(arg)
	if err != nil {
		return nil, err
	}

	listNode, ok := evaled.(*ListNode)
	if !ok || len(listNode.Elements) == 0 {
		return nil, fmt.Errorf(i18n.T("errors.from_cfrac_invalid"), evaled.String())
	}

	n := len(listNode.Elements)
	// Check that elements are integers
	rats := make([]*big.Rat, n)
	for i, elem := range listNode.Elements {
		elemEvaled, err := Eval(elem)
		if err != nil {
			return nil, err
		}
		rNode, ok := elemEvaled.(*RationalNode)
		if !ok || !rNode.Val.IsInt() {
			return nil, fmt.Errorf(i18n.T("errors.from_cfrac_invalid"), listNode.String())
		}
		rats[i] = new(big.Rat).Set(rNode.Val)
	}

	// Fold backwards: cur = a_{n-1}; for k = n-2 downto 0: cur = a_k + 1/cur
	cur := new(big.Rat).Set(rats[n-1])
	for k := n - 2; k >= 0; k-- {
		if cur.Sign() == 0 {
			return nil, fmt.Errorf("division by zero in continued fraction reconstruction")
		}
		inv := new(big.Rat).Inv(cur)
		cur = new(big.Rat).Add(rats[k], inv)
	}

	return &RationalNode{Val: cur}, nil
}
