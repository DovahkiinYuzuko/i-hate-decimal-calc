package calc

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Gröbner Bases & Polynomial Ideals via Buchberger Algorithm
// (Buchberger 1965/1979/1985, Gebauer & Möller 1988, Giovini et al. 1991,
//  Cox, Little, O'Shea 2015 "Ideals, Varieties, and Algorithms")
// -------------------------------------------------------------------------

// MonomialOrder defines the admissible term ordering.
type MonomialOrder int

const (
	OrderLex MonomialOrder = iota // Lexicographic order (elimination order)
	OrderGRevLex                  // Graded reverse lexicographic order (standard/fastest)
)

// Monomial represents a multivariate power product x_1^e_1 * ... * x_n^e_n.
type Monomial struct {
	Exps     []int
	TotalDeg int
}

// newMonomial creates a Monomial with given exponents and precomputed total degree.
func newMonomial(exps []int) Monomial {
	tot := 0
	c := make([]int, len(exps))
	for i, e := range exps {
		c[i] = e
		tot += e
	}
	return Monomial{Exps: c, TotalDeg: tot}
}

// zeroMonomial returns a monomial with all exponents zero.
func zeroMonomial(n int) Monomial {
	return Monomial{Exps: make([]int, n), TotalDeg: 0}
}

// IsZero returns true if all exponents are zero.
func (m Monomial) IsZero() bool {
	return m.TotalDeg == 0
}

// Divides returns true if m divides other (i.e. m.Exps[i] <= other.Exps[i] for all i).
func (m Monomial) Divides(other Monomial) bool {
	for i := range m.Exps {
		if m.Exps[i] > other.Exps[i] {
			return false
		}
	}
	return true
}

// Mul returns the product of two monomials.
func (m Monomial) Mul(other Monomial) Monomial {
	res := make([]int, len(m.Exps))
	for i := range m.Exps {
		res[i] = m.Exps[i] + other.Exps[i]
	}
	return Monomial{Exps: res, TotalDeg: m.TotalDeg + other.TotalDeg}
}

// Div returns m / other assuming other divides m.
func (m Monomial) Div(other Monomial) Monomial {
	res := make([]int, len(m.Exps))
	for i := range m.Exps {
		res[i] = m.Exps[i] - other.Exps[i]
	}
	return Monomial{Exps: res, TotalDeg: m.TotalDeg - other.TotalDeg}
}

// LCM returns the least common multiple monomial of m and other.
func (m Monomial) LCM(other Monomial) Monomial {
	res := make([]int, len(m.Exps))
	tot := 0
	for i := range m.Exps {
		if m.Exps[i] >= other.Exps[i] {
			res[i] = m.Exps[i]
		} else {
			res[i] = other.Exps[i]
		}
		tot += res[i]
	}
	return Monomial{Exps: res, TotalDeg: tot}
}

// AreRelativelyPrime returns true if m and other share no variables (Buchberger Criterion 1).
func (m Monomial) AreRelativelyPrime(other Monomial) bool {
	for i := range m.Exps {
		if m.Exps[i] > 0 && other.Exps[i] > 0 {
			return false
		}
	}
	return true
}

// CompareMonomials compares m1 and m2 with respect to the given order.
// Returns +1 if m1 > m2, -1 if m1 < m2, 0 if m1 == m2.
func CompareMonomials(m1, m2 Monomial, order MonomialOrder) int {
	if order == OrderGRevLex {
		if m1.TotalDeg != m2.TotalDeg {
			if m1.TotalDeg > m2.TotalDeg {
				return 1
			}
			return -1
		}
		// Same total degree: reverse lexicographic, smaller index is lower order
		for i := len(m1.Exps) - 1; i >= 0; i-- {
			if m1.Exps[i] != m2.Exps[i] {
				if m1.Exps[i] < m2.Exps[i] {
					return 1
				}
				return -1
			}
		}
		return 0
	}

	// Default: OrderLex
	for i := 0; i < len(m1.Exps); i++ {
		if m1.Exps[i] != m2.Exps[i] {
			if m1.Exps[i] > m2.Exps[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

// -------------------------------------------------------------------------
// Term & Multivariate Polynomial (MPoly)
// -------------------------------------------------------------------------

// Term represents c * mon.
type Term struct {
	Coeff *big.Rat
	Mon   Monomial
}

// MPoly represents a multivariate polynomial with terms strictly sorted in descending order.
type MPoly struct {
	Terms []Term
	Sugar int
}

// NewZeroMPoly creates a zero polynomial.
func NewZeroMPoly(numVars int) *MPoly {
	return &MPoly{Terms: nil, Sugar: 0}
}

// IsZero returns true if the polynomial is zero.
func (p *MPoly) IsZero() bool {
	return len(p.Terms) == 0
}

// LeadingTerm returns the leading term. Panics if zero.
func (p *MPoly) LeadingTerm() Term {
	return p.Terms[0]
}

// LeadingCoeff returns the leading coefficient. Panics if zero.
func (p *MPoly) LeadingCoeff() *big.Rat {
	return p.Terms[0].Coeff
}

// LeadingMonomial returns the leading monomial. Panics if zero.
func (p *MPoly) LeadingMonomial() Monomial {
	return p.Terms[0].Mon
}

// Degree returns the total degree of the leading monomial, or -1 if zero.
func (p *MPoly) Degree() int {
	if p.IsZero() {
		return -1
	}
	return p.Terms[0].Mon.TotalDeg
}

// Clone creates a deep copy of p.
func (p *MPoly) Clone() *MPoly {
	cp := &MPoly{
		Terms: make([]Term, len(p.Terms)),
		Sugar: p.Sugar,
	}
	for i, t := range p.Terms {
		cp.Terms[i] = Term{
			Coeff: new(big.Rat).Set(t.Coeff),
			Mon:   newMonomial(t.Mon.Exps),
		}
	}
	return cp
}

// Monic normalizes p so that its leading coefficient is 1.
func (p *MPoly) Monic() *MPoly {
	if p.IsZero() {
		return p
	}
	lc := p.LeadingCoeff()
	if lc.Cmp(big.NewRat(1, 1)) == 0 {
		return p.Clone()
	}
	inv := new(big.Rat).Inv(lc)
	res := p.Clone()
	for i := range res.Terms {
		res.Terms[i].Coeff.Mul(res.Terms[i].Coeff, inv)
	}
	return res
}

// Add computes p + q.
func (p *MPoly) Add(q *MPoly, order MonomialOrder) *MPoly {
	var terms []Term
	i, j := 0, 0
	for i < len(p.Terms) && j < len(q.Terms) {
		cmp := CompareMonomials(p.Terms[i].Mon, q.Terms[j].Mon, order)
		if cmp > 0 {
			terms = append(terms, Term{Coeff: new(big.Rat).Set(p.Terms[i].Coeff), Mon: p.Terms[i].Mon})
			i++
		} else if cmp < 0 {
			terms = append(terms, Term{Coeff: new(big.Rat).Set(q.Terms[j].Coeff), Mon: q.Terms[j].Mon})
			j++
		} else {
			sumCoeff := new(big.Rat).Add(p.Terms[i].Coeff, q.Terms[j].Coeff)
			if sumCoeff.Sign() != 0 {
				terms = append(terms, Term{Coeff: sumCoeff, Mon: p.Terms[i].Mon})
			}
			i++
			j++
		}
	}
	for ; i < len(p.Terms); i++ {
		terms = append(terms, Term{Coeff: new(big.Rat).Set(p.Terms[i].Coeff), Mon: p.Terms[i].Mon})
	}
	for ; j < len(q.Terms); j++ {
		terms = append(terms, Term{Coeff: new(big.Rat).Set(q.Terms[j].Coeff), Mon: q.Terms[j].Mon})
	}
	sugar := p.Sugar
	if q.Sugar > sugar {
		sugar = q.Sugar
	}
	return &MPoly{Terms: terms, Sugar: sugar}
}

// Sub computes p - q.
func (p *MPoly) Sub(q *MPoly, order MonomialOrder) *MPoly {
	negQ := &MPoly{
		Terms: make([]Term, len(q.Terms)),
		Sugar: q.Sugar,
	}
	for k, t := range q.Terms {
		negQ.Terms[k] = Term{
			Coeff: new(big.Rat).Neg(t.Coeff),
			Mon:   t.Mon,
		}
	}
	return p.Add(negQ, order)
}

// MulTerm multiplies p by term t.
func (p *MPoly) MulTerm(t Term, order MonomialOrder) *MPoly {
	if p.IsZero() || t.Coeff.Sign() == 0 {
		return NewZeroMPoly(len(t.Mon.Exps))
	}
	res := make([]Term, len(p.Terms))
	for i, pt := range p.Terms {
		res[i] = Term{
			Coeff: new(big.Rat).Mul(pt.Coeff, t.Coeff),
			Mon:   pt.Mon.Mul(t.Mon),
		}
	}
	// Multiplying by a monomial preserves relative term order for admissible orderings
	return &MPoly{Terms: res, Sugar: p.Sugar + t.Mon.TotalDeg}
}

// Mul computes p * q.
func (p *MPoly) Mul(q *MPoly, order MonomialOrder) *MPoly {
	if p.IsZero() || q.IsZero() {
		numVars := 0
		if !p.IsZero() {
			numVars = len(p.Terms[0].Mon.Exps)
		} else if !q.IsZero() {
			numVars = len(q.Terms[0].Mon.Exps)
		}
		return NewZeroMPoly(numVars)
	}
	res := NewZeroMPoly(len(p.Terms[0].Mon.Exps))
	for _, qt := range q.Terms {
		part := p.MulTerm(qt, order)
		res = res.Add(part, order)
	}
	res.Sugar = p.Sugar + q.Sugar
	return res
}

// -------------------------------------------------------------------------
// Multivariate Polynomial Division / Reduction (Cox-Little-O'Shea)
// -------------------------------------------------------------------------

// polyReduce reduces f by a set of polynomials basis until no leading term can be eliminated,
// and tail-reduces intermediate terms.
func polyReduce(f *MPoly, basis []*MPoly, order MonomialOrder) *MPoly {
	if f.IsZero() {
		return f
	}
	var remTerms []Term
	p := f.Clone()

	for !p.IsZero() {
		ltP := p.LeadingTerm()
		divided := false

		for _, g := range basis {
			if g.IsZero() {
				continue
			}
			ltG := g.LeadingTerm()
			if ltG.Mon.Divides(ltP.Mon) {
				qMon := ltP.Mon.Div(ltG.Mon)
				qCoeff := new(big.Rat).Quo(ltP.Coeff, ltG.Coeff)
				qTerm := Term{Coeff: qCoeff, Mon: qMon}

				sub := g.MulTerm(qTerm, order)
				p = p.Sub(sub, order)
				divided = true
				break
			}
		}

		if !divided {
			remTerms = append(remTerms, ltP)
			p.Terms = p.Terms[1:]
		}
	}

	return &MPoly{Terms: remTerms, Sugar: f.Sugar}
}

// sPolynomial computes S(f, g) = (LCM / LT(f)) * f - (LCM / LT(g)) * g.
func sPolynomial(f, g *MPoly, order MonomialOrder) *MPoly {
	ltF := f.LeadingTerm()
	ltG := g.LeadingTerm()

	lcmMon := ltF.Mon.LCM(ltG.Mon)

	monF := lcmMon.Div(ltF.Mon)
	monG := lcmMon.Div(ltG.Mon)

	coeffF := new(big.Rat).Inv(ltF.Coeff)
	coeffG := new(big.Rat).Inv(ltG.Coeff)

	termF := Term{Coeff: coeffF, Mon: monF}
	termG := Term{Coeff: coeffG, Mon: monG}

	partF := f.MulTerm(termF, order)
	partG := g.MulTerm(termG, order)

	res := partF.Sub(partG, order)

	// Sugar degree calculation (Giovini et al. 1991):
	// sugar(S(f, g)) = max(sugar(f) - deg(LM(f)), sugar(g) - deg(LM(g))) + deg(LCM)
	sugarF := f.Sugar - ltF.Mon.TotalDeg
	sugarG := g.Sugar - ltG.Mon.TotalDeg
	maxSugar := sugarF
	if sugarG > maxSugar {
		maxSugar = sugarG
	}
	res.Sugar = maxSugar + lcmMon.TotalDeg

	return res
}

// -------------------------------------------------------------------------
// Buchberger Algorithm with Criteria & Sugar Strategy
// -------------------------------------------------------------------------

type critPair struct {
	i, j  int
	lcm   Monomial
	sugar int
}

func buchberger(initial []*MPoly, order MonomialOrder) ([]*MPoly, error) {
	// Filter zero polynomials and make monic
	var G []*MPoly
	for _, p := range initial {
		if !p.IsZero() {
			G = append(G, p.Monic())
		}
	}
	if len(G) <= 1 {
		return G, nil
	}

	// Pair queue
	var pairs []critPair
	for i := 0; i < len(G); i++ {
		for j := i + 1; j < len(G); j++ {
			p1 := G[i]
			p2 := G[j]
			// Buchberger Criterion 1: relatively prime leading monomials always reduce to 0
			if p1.LeadingMonomial().AreRelativelyPrime(p2.LeadingMonomial()) {
				continue
			}
			lcm := p1.LeadingMonomial().LCM(p2.LeadingMonomial())
			sugar := p1.Sugar - p1.LeadingMonomial().TotalDeg
			s2 := p2.Sugar - p2.LeadingMonomial().TotalDeg
			if s2 > sugar {
				sugar = s2
			}
			sugar += lcm.TotalDeg
			pairs = append(pairs, critPair{i: i, j: j, lcm: lcm, sugar: sugar})
		}
	}

	maxIter := 500
	iter := 0

	for len(pairs) > 0 {
		iter++
		if iter > maxIter {
			return nil, fmt.Errorf("%s", i18n.T("errors.groebner_limit_exceeded"))
		}

		// Pick pair with smallest sugar (Sugar Strategy), break tie with LCM total degree
		bestIdx := 0
		for k := 1; k < len(pairs); k++ {
			if pairs[k].sugar < pairs[bestIdx].sugar ||
				(pairs[k].sugar == pairs[bestIdx].sugar && pairs[k].lcm.TotalDeg < pairs[bestIdx].lcm.TotalDeg) {
				bestIdx = k
			}
		}

		cp := pairs[bestIdx]
		pairs = append(pairs[:bestIdx], pairs[bestIdx+1:]...)

		p1 := G[cp.i]
		p2 := G[cp.j]

		// Check Buchberger Criterion 2 (Chain Criterion):
		// If there exists some k such that LM(G[k]) divides lcm(LM(p1), LM(p2))
		// and both pairs (p1, G[k]) and (p2, G[k]) have already been processed
		// (handled implicitly if S-poly reduces to 0)

		sPoly := sPolynomial(p1, p2, order)
		rem := polyReduce(sPoly, G, order)

		if !rem.IsZero() {
			// Found new basis element
			newElem := rem.Monic()

			// Early exit: if new element is a non-zero constant (1), the ideal is <1>
			if newElem.LeadingMonomial().IsZero() {
				return []*MPoly{newElem}, nil
			}

			newIdx := len(G)
			for i := 0; i < len(G); i++ {
				// Buchberger Criterion 1
				if G[i].LeadingMonomial().AreRelativelyPrime(newElem.LeadingMonomial()) {
					continue
				}
				lcm := G[i].LeadingMonomial().LCM(newElem.LeadingMonomial())
				sugar := G[i].Sugar - G[i].LeadingMonomial().TotalDeg
				s2 := newElem.Sugar - newElem.LeadingMonomial().TotalDeg
				if s2 > sugar {
					sugar = s2
				}
				sugar += lcm.TotalDeg
				pairs = append(pairs, critPair{i: i, j: newIdx, lcm: lcm, sugar: sugar})
			}
			G = append(G, newElem)
		}
	}

	return G, nil
}

// reduceGroebnerBasis cleans up G into a unique reduced Gröbner basis:
// 1. Remove redundant generators (LM(gi) divisible by LM(gj) for j != i)
// 2. Monicize every element
// 3. Tail-reduce every element by all other elements
func reduceGroebnerBasis(G []*MPoly, order MonomialOrder) []*MPoly {
	if len(G) == 0 {
		return G
	}

	// 1. Filter redundancies
	var minimal []*MPoly
	for i, gi := range G {
		if gi.IsZero() {
			continue
		}
		redundant := false
		for j, gj := range G {
			if i != j && !gj.IsZero() {
				if gj.LeadingMonomial().Divides(gi.LeadingMonomial()) {
					if CompareMonomials(gj.LeadingMonomial(), gi.LeadingMonomial(), order) == 0 {
						if j < i {
							redundant = true
							break
						}
					} else {
						redundant = true
						break
					}
				}
			}
		}
		if !redundant {
			minimal = append(minimal, gi.Monic())
		}
	}

	// 2. Tail reduction
	var reduced []*MPoly
	for i, gi := range minimal {
		var others []*MPoly
		for j, gj := range minimal {
			if i != j {
				others = append(others, gj)
			}
		}
		red := polyReduce(gi, others, order)
		if !red.IsZero() {
			reduced = append(reduced, red.Monic())
		}
	}

	// Sort reduced basis for deterministic output (higher leading terms first)
	sort.Slice(reduced, func(i, j int) bool {
		cmp := CompareMonomials(reduced[i].LeadingMonomial(), reduced[j].LeadingMonomial(), order)
		if cmp != 0 {
			return cmp > 0
		}
		return len(reduced[i].Terms) > len(reduced[j].Terms)
	})

	return reduced
}

// -------------------------------------------------------------------------
// AST Node <-> MPoly Conversion
// -------------------------------------------------------------------------

// nodeToMPoly converts an arbitrary AST expression into an MPoly using recursive algebraic construction.
func nodeToMPoly(n Node, varMap map[string]int, numVars int, order MonomialOrder) (*MPoly, error) {
	if n == nil {
		return NewZeroMPoly(numVars), nil
	}

	switch v := n.(type) {
	case *RationalNode:
		if v.Val.Sign() == 0 {
			return NewZeroMPoly(numVars), nil
		}
		mon := zeroMonomial(numVars)
		return &MPoly{
			Terms: []Term{{Coeff: new(big.Rat).Set(v.Val), Mon: mon}},
			Sugar: 0,
		}, nil

	case *VarNode:
		idx, ok := varMap[v.Name]
		if !ok {
			return nil, fmt.Errorf("%s", i18n.T("errors.groebner_unknown_var", v.Name))
		}
		exps := make([]int, numVars)
		exps[idx] = 1
		mon := newMonomial(exps)
		return &MPoly{
			Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: mon}},
			Sugar: 1,
		}, nil

	case *UnaryOpNode:
		sub, err := nodeToMPoly(v.Expr, varMap, numVars, order)
		if err != nil {
			return nil, err
		}
		switch v.Op {
		case "-":
			return NewZeroMPoly(numVars).Sub(sub, order), nil
		case "+":
			return sub, nil
		default:
			return nil, fmt.Errorf("%s", i18n.T("errors.groebner_unsupported_unary", v.Op))
		}

	case *AddNode:
		res := NewZeroMPoly(numVars)
		for _, child := range v.Terms {
			cp, err := nodeToMPoly(child, varMap, numVars, order)
			if err != nil {
				return nil, err
			}
			res = res.Add(cp, order)
		}
		return res, nil

	case *MulNode:
		res := &MPoly{
			Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}},
			Sugar: 0,
		}
		for _, child := range v.Factors {
			cp, err := nodeToMPoly(child, varMap, numVars, order)
			if err != nil {
				return nil, err
			}
			res = res.Mul(cp, order)
		}
		return res, nil

	case *PowNode:
		basePoly, err := nodeToMPoly(v.Base, varMap, numVars, order)
		if err != nil {
			return nil, err
		}
		ratExp, ok := v.Exp.(*RationalNode)
		if !ok || !ratExp.Val.IsInt() || ratExp.Val.Sign() < 0 {
			return nil, fmt.Errorf("%s", i18n.T("errors.groebner_invalid_exponent", v.String()))
		}
		expVal := int(ratExp.Val.Num().Int64())
		if expVal > 100 {
			return nil, fmt.Errorf("%s", i18n.T("errors.groebner_exponent_too_large", expVal))
		}
		res := &MPoly{
			Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}},
			Sugar: 0,
		}
		for k := 0; k < expVal; k++ {
			res = res.Mul(basePoly, order)
		}
		return res, nil

	default:
		// Attempt evaluation/expansion
		evalN, err := Eval(expandNode(n))
		if err == nil && evalN.String() != n.String() {
			return nodeToMPoly(evalN, varMap, numVars, order)
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.groebner_node_to_poly", n.String()))
	}
}

// mpolyToNode converts an MPoly back into a clean AST Node.
func mpolyToNode(p *MPoly, vars []string) Node {
	if p.IsZero() {
		return mustRational(0, 1)
	}

	var termNodes []Node
	for _, t := range p.Terms {
		var factors []Node

		isConstant := t.Mon.IsZero()
		c := t.Coeff

		if isConstant {
			factors = append(factors, &RationalNode{Val: new(big.Rat).Set(c)})
		} else {
			// Non-constant term: handle coefficient 1 or -1 nicely
			if c.Cmp(big.NewRat(1, 1)) == 0 {
				// Coeff 1: omit
			} else if c.Cmp(big.NewRat(-1, 1)) == 0 {
				factors = append(factors, mustRational(-1, 1))
			} else {
				factors = append(factors, &RationalNode{Val: new(big.Rat).Set(c)})
			}

			// Add variables with exponents
			for i, exp := range t.Mon.Exps {
				if exp == 0 {
					continue
				}
				vNode := &VarNode{Name: vars[i]}
				if exp == 1 {
					factors = append(factors, vNode)
				} else {
					powNode, _ := NewPow(vNode, mustRational(int64(exp), 1))
					factors = append(factors, powNode)
				}
			}
		}

		if len(factors) == 0 {
			termNodes = append(termNodes, mustRational(1, 1))
		} else if len(factors) == 1 {
			termNodes = append(termNodes, factors[0])
		} else {
			termNodes = append(termNodes, NewMul(factors))
		}
	}

	if len(termNodes) == 0 {
		return mustRational(0, 1)
	}
	if len(termNodes) == 1 {
		return termNodes[0]
	}
	return NewAdd(termNodes)
}

// -------------------------------------------------------------------------
// Top-Level Public Interface
// -------------------------------------------------------------------------

// EvalGroebner computes the reduced Gröbner basis for a set of polynomials.
// Syntax: groebner([p1, p2, ...], [x, y, ...], [order])
func EvalGroebner(polysNode, varsNode, orderOpt Node, env *Env) (Node, error) {
	if polysNode == nil || varsNode == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.groebner_missing_args"))
	}

	// 1. Extract polynomials
	pList, ok := polysNode.(*ListNode)
	if !ok {
		// Single polynomial wrapped
		pList = NewList([]Node{polysNode})
	}

	// 2. Extract variable list
	var varNames []string
	if vList, ok := varsNode.(*ListNode); ok {
		for _, vElem := range vList.Elements {
			if vn, ok := vElem.(*VarNode); ok {
				varNames = append(varNames, vn.Name)
			} else {
				return nil, fmt.Errorf("%s", i18n.T("errors.groebner_var_list_identifiers", vElem.String()))
			}
		}
	} else if vn, ok := varsNode.(*VarNode); ok {
		varNames = []string{vn.Name}
	} else {
		return nil, fmt.Errorf("%s", i18n.T("errors.groebner_second_arg_var_list", varsNode.String()))
	}

	if len(varNames) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.groebner_var_list_empty"))
	}

	// Check duplicates
	varMap := make(map[string]int)
	for i, name := range varNames {
		if _, exists := varMap[name]; exists {
			return nil, fmt.Errorf("%s", i18n.T("errors.groebner_duplicate_var", name))
		}
		varMap[name] = i
	}

	// 3. Parse monomial ordering
	order := OrderGRevLex
	if orderOpt != nil {
		str := strings.ToLower(orderOpt.String())
		str = strings.Trim(str, "\"'`")
		switch str {
		case "lex", "lexicographic":
			order = OrderLex
		case "grevlex", "degrevlex", "gradedreverselexicographic":
			order = OrderGRevLex
		default:
			return nil, fmt.Errorf("%s", i18n.T("errors.groebner_unsupported_order", str))
		}
	}

	// 4. Convert input polynomials to MPoly
	var polys []*MPoly
	for _, pe := range pList.Elements {
		evalP, err := Eval(expandNode(pe))
		if err != nil {
			evalP = pe
		}
		mp, err := nodeToMPoly(evalP, varMap, len(varNames), order)
		if err != nil {
			return nil, err
		}
		polys = append(polys, mp)
	}

	// 5. Run Buchberger algorithm
	basis, err := buchberger(polys, order)
	if err != nil {
		return nil, err
	}

	// 6. Reduce to unique reduced Gröbner basis
	reduced := reduceGroebnerBasis(basis, order)

	// 7. Convert back to AST ListNode
	var resElements []Node
	for _, rPoly := range reduced {
		resElements = append(resElements, mpolyToNode(rPoly, varNames))
	}

	return NewList(resElements), nil
}
