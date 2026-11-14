package calc

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/padic"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// gradedAbsPrec represents a graded valuation/truncation pair: valuation v and absolute truncation order N.
// When isZero is true, v is ignored and only absN constrains O(t^N).
type gradedAbsPrec struct {
	isZero bool
	val    *big.Rat
	absN   *big.Rat
}

func minRat(a, b *big.Rat) *big.Rat {
	if a.Cmp(b) <= 0 {
		return new(big.Rat).Set(a)
	}
	return new(big.Rat).Set(b)
}

func gradedMulAbsOrder(a, b gradedAbsPrec) *big.Rat {
	if a.isZero && b.isZero {
		return new(big.Rat).Add(a.absN, b.absN)
	}
	if a.isZero {
		return new(big.Rat).Add(a.absN, b.val)
	}
	if b.isZero {
		return new(big.Rat).Add(a.val, b.absN)
	}
	opt1 := new(big.Rat).Add(a.absN, b.val)
	opt2 := new(big.Rat).Add(b.absN, a.val)
	return minRat(opt1, opt2)
}

func gradedInvAbsOrder(a gradedAbsPrec) *big.Rat {
	twoV := new(big.Rat).Mul(big.NewRat(2, 1), a.val)
	return new(big.Rat).Sub(a.absN, twoV)
}

func padicNodeToNum(n *PadicNode) *padic.PadicNumber {
	return &padic.PadicNumber{
		Prime:     n.Prime,
		Valuation: n.Valuation,
		Unit:      n.Unit,
		Precision: n.Precision,
		IsZero:    n.IsZero,
	}
}

func padicNumToNode(p *padic.PadicNumber) (*PadicNode, error) {
	return NewPadicNode(p.Prime, p.Valuation, p.Unit, p.Precision, p.IsZero)
}

func intervalNodeToRatInterval(n *IntervalNode) RationalInterval {
	return NewRationalInterval(n.Low, n.High)
}

func ratIntervalToNode(iv RationalInterval) *IntervalNode {
	return NewIntervalNode(iv.Low, iv.High)
}

func isClosedDomainNode(n Node) bool {
	switch n.(type) {
	case *IntervalNode, *PadicNode, *PuiseuxNode:
		return true
	default:
		return false
	}
}

func containsClosedDomainDeep(nodes []Node) bool {
	found := false
	for _, n := range nodes {
		Walk(n, func(curr Node) bool {
			if found {
				return false
			}
			if isClosedDomainNode(curr) {
				found = true
				return false
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

// promoteRatToPadic converts an exact rational q into a PadicNode with sufficient precision
// so it never degrades partner's absolute precision.
func promoteRatToPadic(q *big.Rat, partner *PadicNode) (*padic.PadicNumber, error) {
	partnerAbs := partner.Precision
	if !partner.IsZero {
		partnerAbs = partner.Valuation + partner.Precision
	}
	if q.Sign() == 0 {
		prec := partnerAbs + 4
		if partner.Precision > 0 {
			prec += partner.Precision
		}
		return &padic.PadicNumber{
			Prime:     new(big.Int).Set(partner.Prime),
			Valuation: 0,
			Unit:      big.NewInt(0),
			Precision: prec,
			IsZero:    true,
		}, nil
	}
	v, err := padic.Valuation(q, partner.Prime)
	if err != nil {
		return nil, err
	}
	relNeeded := partnerAbs - v + 2
	if partner.Precision > 0 {
		relNeeded += partner.Precision
	}
	if v+relNeeded < partnerAbs {
		relNeeded = partnerAbs - v
	}
	if relNeeded < partner.Precision {
		relNeeded = partner.Precision
	}
	if relNeeded < 1 {
		relNeeded = 1
	}
	return padic.NewPadicFromRat(q, partner.Prime, relNeeded)
}

// promoteRatToPuiseux converts an exact rational q into a constant PuiseuxNode whose truncation order
// never degrades partner's precision under addition or multiplication.
func promoteRatToPuiseux(q *big.Rat, partner *PuiseuxNode) (*PuiseuxNode, error) {
	partnerAbs := new(big.Rat).Set(partner.TruncationOrder)
	var minVal *big.Rat
	if len(partner.Terms) > 0 {
		minVal = partner.Terms[0].Exp
	}
	neededOrder := new(big.Rat).Set(partnerAbs)
	if minVal != nil && minVal.Sign() < 0 {
		// Ensure N_q + v_partner >= N_partner  =>  N_q >= N_partner - v_partner
		diff := new(big.Rat).Sub(partnerAbs, minVal)
		if diff.Cmp(neededOrder) > 0 {
			neededOrder = diff
		}
	}
	neededOrder.Add(neededOrder, big.NewRat(1, 1))
	var terms []PuiseuxTermAST
	if q.Sign() != 0 {
		terms = []PuiseuxTermAST{{
			Coeff: new(big.Rat).Set(q),
			Exp:   big.NewRat(0, 1),
		}}
	}
	return NewPuiseuxNode(partner.Var, terms, neededOrder)
}

func puiseuxGradedInfo(p *PuiseuxNode) gradedAbsPrec {
	if len(p.Terms) == 0 {
		return gradedAbsPrec{
			isZero: true,
			val:    nil,
			absN:   p.TruncationOrder,
		}
	}
	return gradedAbsPrec{
		isZero: false,
		val:    p.Terms[0].Exp,
		absN:   p.TruncationOrder,
	}
}

// PuiseuxAdd adds two PuiseuxNodes in the same variable with exact truncation order min(R_a, R_b).
func PuiseuxAdd(a, b *PuiseuxNode) (*PuiseuxNode, error) {
	if a.Var != b.Var {
		return nil, NewDomainError("puiseux.err_var_mismatch", "", a.Var, b.Var)
	}
	nRes := minRat(a.TruncationOrder, b.TruncationOrder)
	combined := make([]PuiseuxTermAST, 0, len(a.Terms)+len(b.Terms))
	combined = append(combined, a.Terms...)
	combined = append(combined, b.Terms...)
	return NewPuiseuxNode(a.Var, combined, nRes)
}

// PuiseuxNeg negates all coefficients of a PuiseuxNode while preserving truncation order.
func PuiseuxNeg(a *PuiseuxNode) (*PuiseuxNode, error) {
	terms := make([]PuiseuxTermAST, len(a.Terms))
	for i, t := range a.Terms {
		terms[i] = PuiseuxTermAST{
			Coeff: new(big.Rat).Neg(t.Coeff),
			Exp:   new(big.Rat).Set(t.Exp),
		}
	}
	return NewPuiseuxNode(a.Var, terms, a.TruncationOrder)
}

// PuiseuxMul multiplies two PuiseuxNodes using the shared graded precision rule N_mul = min(N1+v2, N2+v1).
func PuiseuxMul(a, b *PuiseuxNode) (*PuiseuxNode, error) {
	if a.Var != b.Var {
		return nil, NewDomainError("puiseux.err_var_mismatch", "", a.Var, b.Var)
	}
	nMul := gradedMulAbsOrder(puiseuxGradedInfo(a), puiseuxGradedInfo(b))
	if len(a.Terms) == 0 || len(b.Terms) == 0 {
		return NewPuiseuxNode(a.Var, nil, nMul)
	}
	prodTerms := make([]PuiseuxTermAST, 0, len(a.Terms)*len(b.Terms))
	for _, ta := range a.Terms {
		for _, tb := range b.Terms {
			expSum := new(big.Rat).Add(ta.Exp, tb.Exp)
			if expSum.Cmp(nMul) >= 0 {
				// Since both term slices are sorted ascending, any further tb gives larger expSum.
				break
			}
			prodTerms = append(prodTerms, PuiseuxTermAST{
				Coeff: new(big.Rat).Mul(ta.Coeff, tb.Coeff),
				Exp:   expSum,
			})
		}
	}
	return NewPuiseuxNode(a.Var, prodTerms, nMul)
}

// PuiseuxInv computes the multiplicative inverse 1/a of a non-zero PuiseuxNode via Newton iteration.
func PuiseuxInv(a *PuiseuxNode) (*PuiseuxNode, error) {
	if len(a.Terms) == 0 {
		return nil, NewZeroDivisionError(i18n.T("puiseux.err_zero_division"))
	}
	lead := a.Terms[0]
	v0 := lead.Exp
	c0 := lead.Coeff
	c0Inv := new(big.Rat).Inv(c0)

	info := puiseuxGradedInfo(a)
	nInv := gradedInvAbsOrder(info)
	relPrec := new(big.Rat).Sub(a.TruncationOrder, v0)

	// Normalize unit u(x) = a / (c0 * x^v0) = 1 + sum_{e > 0} (c_i/c0) x^e mod O(x^relPrec)
	unitTerms := make([]PuiseuxTermAST, 0, len(a.Terms))
	for _, t := range a.Terms {
		shiftExp := new(big.Rat).Sub(t.Exp, v0)
		if shiftExp.Cmp(relPrec) >= 0 {
			break
		}
		unitTerms = append(unitTerms, PuiseuxTermAST{
			Coeff: new(big.Rat).Mul(t.Coeff, c0Inv),
			Exp:   shiftExp,
		})
	}
	unitNode, err := NewPuiseuxNode(a.Var, unitTerms, relPrec)
	if err != nil {
		return nil, err
	}

	// Fast path: monomial c0 * x^v0 + O(x^N)
	if len(unitNode.Terms) == 1 {
		negV0 := new(big.Rat).Neg(v0)
		return NewPuiseuxNode(a.Var, []PuiseuxTermAST{{Coeff: c0Inv, Exp: negV0}}, nInv)
	}

	// Newton iteration: inv_{k+1} = inv_k * (2 - unit * inv_k) mod x^relPrec
	twoNode, err := NewPuiseuxNode(a.Var, []PuiseuxTermAST{{Coeff: big.NewRat(2, 1), Exp: big.NewRat(0, 1)}}, relPrec)
	if err != nil {
		return nil, err
	}
	currInv, err := NewPuiseuxNode(a.Var, []PuiseuxTermAST{{Coeff: big.NewRat(1, 1), Exp: big.NewRat(0, 1)}}, relPrec)
	if err != nil {
		return nil, err
	}

	for iter := 0; iter < 64; iter++ {
		prod, err := PuiseuxMul(unitNode, currInv)
		if err != nil {
			return nil, err
		}
		negProd, err := PuiseuxNeg(prod)
		if err != nil {
			return nil, err
		}
		twoMinus, err := PuiseuxAdd(twoNode, negProd)
		if err != nil {
			return nil, err
		}
		nextInv, err := PuiseuxMul(currInv, twoMinus)
		if err != nil {
			return nil, err
		}
		if nextInv.Equal(currInv) {
			currInv = nextInv
			break
		}
		currInv = nextInv
	}

	// Scale currInv by c0Inv * x^(-v0) with final absolute order nInv
	scaledTerms := make([]PuiseuxTermAST, 0, len(currInv.Terms))
	for _, t := range currInv.Terms {
		expShifted := new(big.Rat).Sub(t.Exp, v0)
		if expShifted.Cmp(nInv) >= 0 {
			break
		}
		scaledTerms = append(scaledTerms, PuiseuxTermAST{
			Coeff: new(big.Rat).Mul(t.Coeff, c0Inv),
			Exp:   expShifted,
		})
	}
	return NewPuiseuxNode(a.Var, scaledTerms, nInv)
}

// PuiseuxPowInt raises a PuiseuxNode to integer exponent k.
func PuiseuxPowInt(a *PuiseuxNode, k *big.Int) (*PuiseuxNode, error) {
	if k == nil {
		return nil, NewDomainError("puiseux.err_exponent_out_of_range", "", "<nil>")
	}
	if k.Sign() == 0 {
		relOrder := new(big.Rat).SetInt64(1)
		if len(a.Terms) > 0 {
			diff := new(big.Rat).Sub(a.TruncationOrder, a.Terms[0].Exp)
			if diff.Cmp(relOrder) > 0 {
				relOrder = diff
			}
		} else if a.TruncationOrder.Cmp(relOrder) > 0 {
			relOrder = new(big.Rat).Set(a.TruncationOrder)
		}
		return NewPuiseuxNode(a.Var, []PuiseuxTermAST{{Coeff: big.NewRat(1, 1), Exp: big.NewRat(0, 1)}}, relOrder)
	}
	if k.Sign() < 0 {
		invA, err := PuiseuxInv(a)
		if err != nil {
			return nil, err
		}
		return PuiseuxPowInt(invA, new(big.Int).Neg(k))
	}
	if !k.IsInt64() {
		return nil, NewDomainError("puiseux.err_exponent_out_of_range", "", k.String())
	}
	exp := k.Int64()
	if len(a.Terms) == 0 {
		return NewPuiseuxNode(a.Var, nil, new(big.Rat).Mul(a.TruncationOrder, new(big.Rat).SetInt64(exp)))
	}

	var res *PuiseuxNode
	var err error
	base := a.Clone()
	first := true
	for exp > 0 {
		if exp&1 == 1 {
			if first {
				res = base.Clone()
				first = false
			} else {
				res, err = PuiseuxMul(res, base)
				if err != nil {
					return nil, err
				}
			}
		}
		exp >>= 1
		if exp > 0 {
			base, err = PuiseuxMul(base, base)
			if err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

// DifferentiatePuiseux computes d/d(varName) of a PuiseuxNode, lowering truncation bound R -> R - 1.
func DifferentiatePuiseux(p *PuiseuxNode, varName string) (Node, error) {
	if p.Var != varName {
		return mustRational(0, 1), nil
	}
	dTerms := make([]PuiseuxTermAST, 0, len(p.Terms))
	one := big.NewRat(1, 1)
	for _, t := range p.Terms {
		if t.Exp.Sign() == 0 {
			continue
		}
		dTerms = append(dTerms, PuiseuxTermAST{
			Coeff: new(big.Rat).Mul(t.Coeff, t.Exp),
			Exp:   new(big.Rat).Sub(t.Exp, one),
		})
	}
	newOrder := new(big.Rat).Sub(p.TruncationOrder, one)
	return NewPuiseuxNode(p.Var, dTerms, newOrder)
}

func negateClosedDomain(n Node) (Node, error) {
	switch v := n.(type) {
	case *IntervalNode:
		inv := intervalNodeToRatInterval(v).Neg()
		return ratIntervalToNode(inv), nil
	case *PadicNode:
		zeroPadic := &padic.PadicNumber{
			Prime:     new(big.Int).Set(v.Prime),
			Valuation: 0,
			Unit:      big.NewInt(0),
			Precision: v.Precision + v.Valuation + 1,
			IsZero:    true,
		}
		if v.IsZero {
			return v.Clone(), nil
		}
		res, err := padic.Sub(zeroPadic, padicNodeToNum(v))
		if err != nil {
			return nil, err
		}
		return padicNumToNode(res)
	case *PuiseuxNode:
		return PuiseuxNeg(v)
	default:
		return nil, fmt.Errorf("negateClosedDomain: unsupported node %T", n) // txthc-ignore
	}
}

func simplifyClosedDomainPow(base Node, exp *big.Int) (Node, error) {
	switch v := base.(type) {
	case *IntervalNode:
		if !exp.IsInt64() {
			return NewPow(base, &RationalNode{Val: new(big.Rat).SetInt(exp)})
		}
		iv := intervalNodeToRatInterval(v)
		res, err := iv.PowIntChecked(int(exp.Int64()))
		if err != nil {
			return nil, NewZeroDivisionError(err.Error())
		}
		return ratIntervalToNode(res), nil
	case *PadicNode:
		res, err := padic.PowInt(padicNodeToNum(v), exp)
		if err != nil {
			if v.IsZero && exp.Sign() < 0 {
				return nil, NewZeroDivisionError(err.Error())
			}
			return nil, NewDomainError("", "%s", err.Error())
		}
		return padicNumToNode(res)
	case *PuiseuxNode:
		return PuiseuxPowInt(v, exp)
	default:
		return NewPow(base, &RationalNode{Val: new(big.Rat).SetInt(exp)})
	}
}

func addCompatiblePadic(acc *PadicNode, item *PadicNode) (*PadicNode, error) {
	res, err := padic.Add(padicNodeToNum(acc), padicNodeToNum(item))
	if err != nil {
		return nil, err
	}
	return padicNumToNode(res)
}

func mulCompatiblePadic(acc *PadicNode, item *PadicNode) (*PadicNode, error) {
	res, err := padic.Mul(padicNodeToNum(acc), padicNodeToNum(item))
	if err != nil {
		return nil, err
	}
	return padicNumToNode(res)
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// simplifyClosedDomainAdd folds compatible closed-domain nodes while forbidding x - x -> 0.
func simplifyClosedDomainAdd(terms []Node) (Node, error) {
	var intervalAcc *RationalInterval
	padicGroups := make(map[string]*PadicNode)
	puiseuxGroups := make(map[string]*PuiseuxNode)
	ratSum := big.NewRat(0, 1)
	var symbolicTerms []Node

	for _, t := range terms {
		switch v := t.(type) {
		case *RationalNode:
			ratSum.Add(ratSum, v.Val)
		case *IntervalNode:
			iv := intervalNodeToRatInterval(v)
			if intervalAcc == nil {
				intervalAcc = &iv
			} else {
				sum := intervalAcc.Add(iv)
				intervalAcc = &sum
			}
		case *PadicNode:
			pKey := v.Prime.String()
			if existing, ok := padicGroups[pKey]; ok {
				sum, err := addCompatiblePadic(existing, v)
				if err != nil {
					return nil, err
				}
				padicGroups[pKey] = sum
			} else {
				padicGroups[pKey] = v.Clone()
			}
		case *PuiseuxNode:
			if existing, ok := puiseuxGroups[v.Var]; ok {
				sum, err := PuiseuxAdd(existing, v)
				if err != nil {
					return nil, err
				}
				puiseuxGroups[v.Var] = sum
			} else {
				puiseuxGroups[v.Var] = v.Clone()
			}
		default:
			symbolicTerms = append(symbolicTerms, t)
		}
	}

	// Count distinct active closed domains; promote exact rational sum only when domain is unambiguous.
	activeDomains := 0
	if intervalAcc != nil {
		activeDomains++
	}
	activeDomains += len(padicGroups) + len(puiseuxGroups)

	var out []Node
	if activeDomains == 1 && ratSum.Sign() != 0 {
		if intervalAcc != nil {
			sum := intervalAcc.Add(NewExactRationalInterval(ratSum))
			intervalAcc = &sum
			ratSum.SetInt64(0)
		} else if len(padicGroups) == 1 {
			for k, pNode := range padicGroups {
				prom, err := promoteRatToPadic(ratSum, pNode)
				if err != nil {
					return nil, err
				}
				sum, err := padic.Add(padicNodeToNum(pNode), prom)
				if err != nil {
					return nil, err
				}
				node, err := padicNumToNode(sum)
				if err != nil {
					return nil, err
				}
				padicGroups[k] = node
			}
			ratSum.SetInt64(0)
		} else if len(puiseuxGroups) == 1 {
			for k, pui := range puiseuxGroups {
				prom, err := promoteRatToPuiseux(ratSum, pui)
				if err != nil {
					return nil, err
				}
				sum, err := PuiseuxAdd(pui, prom)
				if err != nil {
					return nil, err
				}
				puiseuxGroups[k] = sum
			}
			ratSum.SetInt64(0)
		}
	}

	if intervalAcc != nil {
		out = append(out, ratIntervalToNode(*intervalAcc))
	}
	for _, k := range sortedMapKeys(padicGroups) {
		out = append(out, padicGroups[k])
	}
	for _, k := range sortedMapKeys(puiseuxGroups) {
		out = append(out, puiseuxGroups[k])
	}
	if len(symbolicTerms) > 0 {
		if len(symbolicTerms) == 1 {
			out = append(out, symbolicTerms[0])
		} else {
			symSum, err := simplifyAddWithGuard(symbolicTerms, false)
			if err != nil {
				return nil, err
			}
			if !isZero(symSum) {
				out = append(out, symSum)
			}
		}
	}
	if ratSum.Sign() != 0 {
		out = append(out, &RationalNode{Val: new(big.Rat).Set(ratSum)})
	}

	if len(out) == 0 {
		return mustRational(0, 1), nil
	}
	if len(out) == 1 {
		return out[0], nil
	}
	return NewAdd(out), nil
}

// simplifyClosedDomainMul multiplies compatible closed-domain factors while forbidding x*x^-1 -> 1 and distribution.
func simplifyClosedDomainMul(factors []Node) (Node, error) {
	var intervalAcc *RationalInterval
	padicGroups := make(map[string]*PadicNode)
	puiseuxGroups := make(map[string]*PuiseuxNode)
	ratProd := big.NewRat(1, 1)
	var otherFactors []Node

	for _, f := range factors {
		// Handle PowNode(ClosedDomain, Integer) before merging
		if pow, ok := f.(*PowNode); ok && isClosedDomainNode(pow.Base) {
			if rExp, okR := pow.Exp.(*RationalNode); okR && rExp.Val.IsInt() {
				powered, err := simplifyClosedDomainPow(pow.Base, rExp.Val.Num())
				if err != nil {
					return nil, err
				}
				f = powered
			}
		}

		switch v := f.(type) {
		case *RationalNode:
			ratProd.Mul(ratProd, v.Val)
		case *IntervalNode:
			iv := intervalNodeToRatInterval(v)
			if intervalAcc == nil {
				intervalAcc = &iv
			} else {
				prod := intervalAcc.Mul(iv)
				intervalAcc = &prod
			}
		case *PadicNode:
			pKey := v.Prime.String()
			if existing, ok := padicGroups[pKey]; ok {
				prod, err := mulCompatiblePadic(existing, v)
				if err != nil {
					return nil, err
				}
				padicGroups[pKey] = prod
			} else {
				padicGroups[pKey] = v.Clone()
			}
		case *PuiseuxNode:
			if existing, ok := puiseuxGroups[v.Var]; ok {
				prod, err := PuiseuxMul(existing, v)
				if err != nil {
					return nil, err
				}
				puiseuxGroups[v.Var] = prod
			} else {
				puiseuxGroups[v.Var] = v.Clone()
			}
		default:
			otherFactors = append(otherFactors, f)
		}
	}

	activeDomains := 0
	if intervalAcc != nil {
		activeDomains++
	}
	activeDomains += len(padicGroups) + len(puiseuxGroups)

	one := big.NewRat(1, 1)
	if activeDomains == 1 && ratProd.Cmp(one) != 0 {
		if intervalAcc != nil {
			prod := intervalAcc.Mul(NewExactRationalInterval(ratProd))
			intervalAcc = &prod
			ratProd.SetInt64(1)
		} else if len(padicGroups) == 1 {
			for k, pNode := range padicGroups {
				if ratProd.Sign() == 0 && len(otherFactors) == 0 {
					// 0 * x = 0 in Z/Q_p? Exact rational 0 annihilates to exact 0 if scalar; or p-adic zero
					return mustRational(0, 1), nil
				}
				prom, err := promoteRatToPadic(ratProd, pNode)
				if err != nil {
					return nil, err
				}
				prod, err := padic.Mul(padicNodeToNum(pNode), prom)
				if err != nil {
					return nil, err
				}
				node, err := padicNumToNode(prod)
				if err != nil {
					return nil, err
				}
				padicGroups[k] = node
			}
			ratProd.SetInt64(1)
		} else if len(puiseuxGroups) == 1 {
			for k, pui := range puiseuxGroups {
				if ratProd.Sign() == 0 && len(otherFactors) == 0 {
					return mustRational(0, 1), nil
				}
				prom, err := promoteRatToPuiseux(ratProd, pui)
				if err != nil {
					return nil, err
				}
				prod, err := PuiseuxMul(pui, prom)
				if err != nil {
					return nil, err
				}
				puiseuxGroups[k] = prod
			}
			ratProd.SetInt64(1)
		}
	}

	var out []Node
	if ratProd.Cmp(one) != 0 {
		if ratProd.Sign() == 0 && len(otherFactors) == 0 && intervalAcc == nil {
			return mustRational(0, 1), nil
		}
		out = append(out, &RationalNode{Val: new(big.Rat).Set(ratProd)})
	}
	if intervalAcc != nil {
		out = append(out, ratIntervalToNode(*intervalAcc))
	}
	for _, k := range sortedMapKeys(padicGroups) {
		out = append(out, padicGroups[k])
	}
	for _, k := range sortedMapKeys(puiseuxGroups) {
		out = append(out, puiseuxGroups[k])
	}
	out = append(out, otherFactors...)

	if len(out) == 0 {
		return mustRational(1, 1), nil
	}
	if len(out) == 1 {
		return out[0], nil
	}
	return NewMul(out), nil
}
