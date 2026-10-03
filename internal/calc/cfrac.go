package calc

import (
	"fmt"
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
		return nil, fmt.Errorf("%s", i18n.T("cfrac.err_cannot_compute_continued_fraction_of"))
	}

	var elems []Node
	cur := new(big.Rat).Set(r)

	for {
		a := floorRat(cur)
		elems = append(elems, NewRationalFromBigRat(new(big.Rat).SetInt(a)))

		aRat := new(big.Rat).SetInt(a)
		frac := new(big.Rat).Sub(cur, aRat)
		if frac.Sign() == 0 {
			break
		}
		cur.Inv(frac)
	}

	return &ListNode{Elements: elems}, nil
}

// cfracSqrt computes the periodic continued fraction expansion of sqrt(D) using arbitrary precision big.Int.
// Returns [a0, [a1, a2, ..., am]] where a0 is the floor and [a1...am] is the repeating period.
func cfracSqrt(d *big.Int) (*ListNode, error) {
	if d.Sign() < 0 {
		return nil, fmt.Errorf("%s", i18n.T("cfrac.err_continued_fraction_of_imaginary_square", d.String()))
	}
	if d.Sign() == 0 {
		return &ListNode{Elements: []Node{mustRational(0, 1)}}, nil
	}

	a0 := new(big.Int).Sqrt(d)
	a0Sq := new(big.Int).Mul(a0, a0)
	if a0Sq.Cmp(d) == 0 {
		// Perfect square
		return &ListNode{Elements: []Node{NewRationalFromBigRat(new(big.Rat).SetInt(a0))}}, nil
	}

	m := big.NewInt(0)
	denom := big.NewInt(1)
	a := new(big.Int).Set(a0)
	twoA0 := new(big.Int).Lsh(a0, 1)
	var period []Node

	// Continued fraction expansion of sqrt(d) algorithm
	const maxIterations = 2000
	for iter := 0; iter < maxIterations; iter++ {
		// m = denom * a - m
		m = new(big.Int).Sub(new(big.Int).Mul(denom, a), m)
		// denom = (d - m^2) / denom
		mSq := new(big.Int).Mul(m, m)
		denom = new(big.Int).Div(new(big.Int).Sub(d, mSq), denom)
		// a = (a0 + m) / denom
		a = new(big.Int).Div(new(big.Int).Add(a0, m), denom)
		period = append(period, NewRationalFromBigRat(new(big.Rat).SetInt(a)))
		if a.Cmp(twoA0) == 0 {
			break
		}
	}

	periodList := &ListNode{Elements: period}
	return &ListNode{Elements: []Node{NewRationalFromBigRat(new(big.Rat).SetInt(a0)), periodList}}, nil
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
			return cfracSqrt(rat.Val.Num())
		}
	default:
		if sq, ok := extractRadicalSquare(evaled); ok && sq.IsInt() && sq.Sign() >= 0 {
			return cfracSqrt(sq.Num())
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
			return nil, fmt.Errorf("%s", i18n.T("cfrac.err_division_by_zero_in_continued"))
		}
		inv := new(big.Rat).Inv(cur)
		cur = new(big.Rat).Add(rats[k], inv)
	}

	return &RationalNode{Val: cur}, nil
}

// evalSolvePell solves Pell's equation x^2 - d * y^2 = 1 for the fundamental positive integer solution (x0, y0).
func evalSolvePell(arg Node) (Node, error) {
	evaled, err := Eval(arg)
	if err != nil {
		return nil, err
	}

	rat, ok := evaled.(*RationalNode)
	if !ok || !rat.Val.IsInt() || rat.Val.Sign() <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("cfrac.err_pell_d_must_be_positive_integer"))
	}

	d := rat.Val.Num()

	// Check if d is a square
	a0 := new(big.Int).Sqrt(d)
	a0Sq := new(big.Int).Mul(a0, a0)
	if a0Sq.Cmp(d) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("cfrac.err_pell_d_square"))
	}

	// Continued fraction expansion convergents of sqrt(d)
	m := big.NewInt(0)
	denom := big.NewInt(1)
	a := new(big.Int).Set(a0)

	pPrev2 := big.NewInt(1)
	pPrev1 := new(big.Int).Set(a0)
	qPrev2 := big.NewInt(0)
	qPrev1 := big.NewInt(1)

	const maxPellIterations = 100000
	for iter := 0; iter < maxPellIterations; iter++ {
		// Test if pPrev1^2 - d * qPrev1^2 == 1
		pSq := new(big.Int).Mul(pPrev1, pPrev1)
		qSq := new(big.Int).Mul(qPrev1, qPrev1)
		dqSq := new(big.Int).Mul(d, qSq)
		diff := new(big.Int).Sub(pSq, dqSq)
		if diff.Cmp(big.NewInt(1)) == 0 {
			xNode := &RationalNode{Val: new(big.Rat).SetInt(pPrev1)}
			yNode := &RationalNode{Val: new(big.Rat).SetInt(qPrev1)}
			resNode := &ListNode{Elements: []Node{xNode, yNode}}
			RecordTraceRewrite(RuleSolvePell, NewSqrt(rat), resNode, i18n.T("trace.solve_pell", pPrev1.String(), d.String(), qPrev1.String(), pPrev1.String(), qPrev1.String()))
			return resNode, nil
		}

		// Next partial quotient
		m = new(big.Int).Sub(new(big.Int).Mul(denom, a), m)
		mSq := new(big.Int).Mul(m, m)
		denom = new(big.Int).Div(new(big.Int).Sub(d, mSq), denom)
		a = new(big.Int).Div(new(big.Int).Add(a0, m), denom)

		pNext := new(big.Int).Add(new(big.Int).Mul(a, pPrev1), pPrev2)
		qNext := new(big.Int).Add(new(big.Int).Mul(a, qPrev1), qPrev2)

		pPrev2, pPrev1 = pPrev1, pNext
		qPrev2, qPrev1 = qPrev1, qNext
	}

	return nil, fmt.Errorf("%s", i18n.T("cfrac.err_pell_max_iterations_exceeded"))
}

