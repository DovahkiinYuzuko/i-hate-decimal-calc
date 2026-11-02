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

// Equal returns true if m and other have the exact same exponents.
func (m Monomial) Equal(other Monomial) bool {
	if m.TotalDeg != other.TotalDeg || len(m.Exps) != len(other.Exps) {
		return false
	}
	for i := range m.Exps {
		if m.Exps[i] != other.Exps[i] {
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

// polyDivisionWithQuotients divides f by basis = [g_1, ..., g_s] using Cox-Little-O'Shea division algorithm.
// It returns quotients [q_1, ..., q_s] and remainder r such that f = sum q_i * g_i + r.
func polyDivisionWithQuotients(f *MPoly, basis []*MPoly, order MonomialOrder) ([]*MPoly, *MPoly) {
	numVars := polyNumVars(f)
	for _, g := range basis {
		if n := polyNumVars(g); n > numVars {
			numVars = n
		}
	}
	qs := make([]*MPoly, len(basis))
	for i := range qs {
		qs[i] = NewZeroMPoly(numVars)
	}
	if f.IsZero() {
		return qs, NewZeroMPoly(numVars)
	}

	var remTerms []Term
	p := f.Clone()

	for !p.IsZero() {
		ltP := p.LeadingTerm()
		divided := false

		for i, g := range basis {
			if g.IsZero() {
				continue
			}
			ltG := g.LeadingTerm()
			if ltG.Mon.Divides(ltP.Mon) {
				qMon := ltP.Mon.Div(ltG.Mon)
				qCoeff := new(big.Rat).Quo(ltP.Coeff, ltG.Coeff)
				qTerm := Term{Coeff: qCoeff, Mon: qMon}

				qs[i] = qs[i].Add(&MPoly{Terms: []Term{qTerm}, Sugar: qMon.TotalDeg}, order)

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

	return qs, &MPoly{Terms: remTerms, Sugar: f.Sugar}
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

func critPairKey(i, j int) uint64 {
	if i > j {
		i, j = j, i
	}
	return (uint64(i) << 32) | uint64(j)
}

// updatePairsGebauerMoller implements the Gebauer & Möller (1988) Update algorithm.
// When newElem (at newIdx) is added to G, it prunes existing pairs in pairs and adds minimal new pairs.
func updatePairsGebauerMoller(G []*MPoly, pairs []critPair, newElem *MPoly, newIdx int, processed map[uint64]bool) []critPair {
	lmNew := newElem.LeadingMonomial()

	// 1. Form candidate pairs with newElem:
	// (i, newIdx) for i in 0..newIdx-1
	var candidates []critPair
	for i := 0; i < newIdx; i++ {
		lmI := G[i].LeadingMonomial()
		key := critPairKey(i, newIdx)
		// Criterion 1: relatively prime leading monomials always reduce to 0
		if lmI.AreRelativelyPrime(lmNew) {
			processed[key] = true
			continue
		}
		lcm := lmI.LCM(lmNew)
		sugar := G[i].Sugar - lmI.TotalDeg
		sNew := newElem.Sugar - lmNew.TotalDeg
		if sNew > sugar {
			sugar = sNew
		}
		sugar += lcm.TotalDeg
		candidates = append(candidates, critPair{i: i, j: newIdx, lcm: lcm, sugar: sugar})
	}

	// 2. Apply Criterion 2 on new pairs (Gebauer-Möller M-Criterion):
	// A candidate (i, newIdx) is superfluous if there exists another candidate (j, newIdx) (j != i)
	// such that LCM(LM(Gj), LM(newElem)) divides LCM(LM(Gi), LM(newElem))
	// and LCM(LM(Gj), LM(newElem)) != LCM(LM(Gi), LM(newElem))
	// AND the pair (i, j) has already been processed!
	var newPairs []critPair
	for a := 0; a < len(candidates); a++ {
		keep := true
		for b := 0; b < len(candidates); b++ {
			if a == b {
				continue
			}
			idxI := candidates[a].i
			idxJ := candidates[b].i
			// Only valid if (idxI, idxJ) has already been processed!
			if !processed[critPairKey(idxI, idxJ)] {
				continue
			}
			lcmA := candidates[a].lcm
			lcmB := candidates[b].lcm
			if lcmB.Divides(lcmA) {
				if !lcmB.Equal(lcmA) {
					keep = false
					break
				}
				if b < a {
					keep = false
					break
				}
			}
		}
		if keep {
			newPairs = append(newPairs, candidates[a])
		}
	}

	// 3. Prune existing pairs in B (Gebauer-Möller F-Criterion):
	// For each pair (i, j) in pairs, if LM(newElem) divides LCM(LM(Gi), LM(Gj))
	// and LCM(LM(Gi), LM(newElem)) != LCM(LM(Gi), LM(Gj))
	// and LCM(LM(Gj), LM(newElem)) != LCM(LM(Gi), LM(Gj))
	// AND both (i, newIdx) and (j, newIdx) are covered (in newPairs or processed),
	// then the pair (i, j) can be pruned.
	covered := make(map[int]bool)
	for _, np := range newPairs {
		covered[np.i] = true
	}
	for i := 0; i < newIdx; i++ {
		if processed[critPairKey(i, newIdx)] {
			covered[i] = true
		}
	}

	var prunedPairs []critPair
	for _, cp := range pairs {
		if lmNew.Divides(cp.lcm) && covered[cp.i] && covered[cp.j] {
			lcm1 := G[cp.i].LeadingMonomial().LCM(lmNew)
			lcm2 := G[cp.j].LeadingMonomial().LCM(lmNew)
			if !lcm1.Equal(cp.lcm) && !lcm2.Equal(cp.lcm) {
				// Redundant by chain through newElem: mark processed and prune!
				processed[critPairKey(cp.i, cp.j)] = true
				continue
			}
		}
		prunedPairs = append(prunedPairs, cp)
	}

	return append(prunedPairs, newPairs...)
}

// canApplyChainCriterion checks Cox-Little-O'Shea dynamic Criterion 2:
// There exists some k in 0..len(G)-1 (k != i, k != j) such that:
// 1. LM(G[k]) divides LCM(LM(p1), LM(p2))
// 2. Both pairs (i, k) and (j, k) have already been processed (in processed map).
func canApplyChainCriterion(p1, p2 *MPoly, i, j int, G []*MPoly, processed map[uint64]bool) bool {
	lcm := p1.LeadingMonomial().LCM(p2.LeadingMonomial())

	for k := 0; k < len(G); k++ {
		if k == i || k == j {
			continue
		}
		lmK := G[k].LeadingMonomial()
		if lmK.Divides(lcm) {
			if processed[critPairKey(i, k)] && processed[critPairKey(j, k)] {
				return true
			}
		}
	}
	return false
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

	processed := make(map[uint64]bool)
	var pairs []critPair
	var currentG []*MPoly
	for i, p := range G {
		if i > 0 {
			pairs = updatePairsGebauerMoller(currentG, pairs, p, i, processed)
		}
		currentG = append(currentG, p)
	}
	G = currentG

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
		EmitSearchEvent(EventGroebnerPairSelected, "buchberger", map[string]any{"iter": iter, "pair_i": cp.i, "pair_j": cp.j, "sugar": cp.sugar})

		// Dynamic check of Buchberger Criterion 2 (Chain Criterion):
		// If there exists some k such that LM(G[k]) divides lcm(LM(p1), LM(p2))
		// and both pairs (p1, G[k]) and (p2, G[k]) have already been processed
		// (in processed map), S-poly is guaranteed to reduce to 0.
		if canApplyChainCriterion(p1, p2, cp.i, cp.j, G, processed) {
			processed[critPairKey(cp.i, cp.j)] = true
			continue
		}

		sPoly := sPolynomial(p1, p2, order)
		rem := polyReduce(sPoly, G, order)
		processed[critPairKey(cp.i, cp.j)] = true

		if !rem.IsZero() {
			// Found new basis element
			newElem := rem.Monic()

			// Early exit: if new element is a non-zero constant (1), the ideal is <1>
			if newElem.LeadingMonomial().IsZero() {
				return []*MPoly{newElem}, nil
			}

			newIdx := len(G)
			pairs = updatePairsGebauerMoller(G, pairs, newElem, newIdx, processed)
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
	type tailTask struct {
		gi     *MPoly
		others []*MPoly
	}
	tasks := make([]tailTask, len(minimal))
	for i, gi := range minimal {
		var others []*MPoly
		for j, gj := range minimal {
			if i != j {
				others = append(others, gj)
			}
		}
		tasks[i] = tailTask{gi: gi, others: others}
	}

	reducedRaw, _ := ParallelBatchMap(tasks, func(t tailTask) (*MPoly, error) {
		red := polyReduce(t.gi, t.others, order)
		if !red.IsZero() {
			return red.Monic(), nil
		}
		return nil, nil
	}, 4)

	var reduced []*MPoly
	for _, r := range reducedRaw {
		if r != nil {
			reduced = append(reduced, r)
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

// parseMPolyInputArgs parses polynomial list, variable list, and optional monomial order.
func parseMPolyInputArgs(polysNode, varsNode, orderOpt Node) ([]*MPoly, []string, MonomialOrder, error) {
	if polysNode == nil || varsNode == nil {
		return nil, nil, OrderGRevLex, fmt.Errorf("%s", i18n.T("errors.groebner_missing_args"))
	}

	pList, ok := polysNode.(*ListNode)
	if !ok {
		pList = NewList([]Node{polysNode})
	}

	var varNames []string
	if vList, ok := varsNode.(*ListNode); ok {
		for _, vElem := range vList.Elements {
			if vn, ok := vElem.(*VarNode); ok {
				varNames = append(varNames, vn.Name)
			} else {
				return nil, nil, OrderGRevLex, fmt.Errorf("%s", i18n.T("errors.groebner_var_list_identifiers", vElem.String()))
			}
		}
	} else if vn, ok := varsNode.(*VarNode); ok {
		varNames = []string{vn.Name}
	} else {
		return nil, nil, OrderGRevLex, fmt.Errorf("%s", i18n.T("errors.groebner_second_arg_var_list", varsNode.String()))
	}

	if len(varNames) == 0 {
		return nil, nil, OrderGRevLex, fmt.Errorf("%s", i18n.T("errors.groebner_var_list_empty"))
	}

	varMap := make(map[string]int)
	for i, name := range varNames {
		if _, exists := varMap[name]; exists {
			return nil, nil, OrderGRevLex, fmt.Errorf("%s", i18n.T("errors.groebner_duplicate_var", name))
		}
		varMap[name] = i
	}

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
			return nil, nil, OrderGRevLex, fmt.Errorf("%s", i18n.T("errors.groebner_unsupported_order", str))
		}
	}

	var polys []*MPoly
	for _, pe := range pList.Elements {
		evalP, err := Eval(expandNode(pe))
		if err != nil {
			evalP = pe
		}
		mp, err := nodeToMPoly(evalP, varMap, len(varNames), order)
		if err != nil {
			return nil, nil, OrderGRevLex, err
		}
		polys = append(polys, mp)
	}

	return polys, varNames, order, nil
}

// EvalGroebner computes the reduced Gröbner basis for a set of polynomials.
// Syntax: groebner([p1, p2, ...], [x, y, ...], [order])
func EvalGroebner(polysNode, varsNode, orderOpt Node, env *Env) (Node, error) {
	polys, varNames, order, err := parseMPolyInputArgs(polysNode, varsNode, orderOpt)
	if err != nil {
		return nil, err
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


// -------------------------------------------------------------------------
// Extended Buchberger Algorithm & Ideal Membership with Cofactors
// (Cox, Little, O'Shea 2015, Möller, Mora, Traverso 1992, Schreyer 1980)
// -------------------------------------------------------------------------

func oneMPoly(numVars int) *MPoly {
	return &MPoly{
		Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}},
		Sugar: 0,
	}
}

func addCofactorVector(v1, v2 []*MPoly, order MonomialOrder) []*MPoly {
	res := make([]*MPoly, len(v1))
	for i := range v1 {
		res[i] = v1[i].Add(v2[i], order)
	}
	return res
}

func subCofactorVector(v1, v2 []*MPoly, order MonomialOrder) []*MPoly {
	res := make([]*MPoly, len(v1))
	for i := range v1 {
		res[i] = v1[i].Sub(v2[i], order)
	}
	return res
}

func cofactorVectorNumVars(v []*MPoly, fallback int) int {
	for _, p := range v {
		if n := polyNumVars(p); n > 0 {
			return n
		}
	}
	if fallback > 0 {
		return fallback
	}
	return 1
}

func scaleCofactorVector(v []*MPoly, scale *big.Rat, order MonomialOrder) []*MPoly {
	numVars := cofactorVectorNumVars(v, 1)
	term := Term{Coeff: scale, Mon: zeroMonomial(numVars)}
	res := make([]*MPoly, len(v))
	for i := range v {
		res[i] = v[i].MulTerm(term, order)
	}
	return res
}

func mulCofactorVectorTerm(v []*MPoly, t Term, order MonomialOrder) []*MPoly {
	res := make([]*MPoly, len(v))
	for i := range v {
		res[i] = v[i].MulTerm(t, order)
	}
	return res
}

func mulCofactorVectorPoly(v []*MPoly, q *MPoly, order MonomialOrder) []*MPoly {
	res := make([]*MPoly, len(v))
	for i := range v {
		res[i] = v[i].Mul(q, order)
	}
	return res
}

// extendedBuchberger tracks cofactors g_j = sum_{i=1}^m A[j][i] * f_i alongside the Gröbner basis computation.
// It also collects syzygies (S-polynomial zero reductions: sum s_i * f_i = 0).
func extendedBuchberger(initial []*MPoly, order MonomialOrder) ([]*MPoly, [][]*MPoly, [][]*MPoly, error) {
	m := len(initial)
	numVars := 1
	for _, p := range initial {
		if n := polyNumVars(p); n > numVars {
			numVars = n
		}
	}

	var G []*MPoly
	var cofactors [][]*MPoly // cofactors[j][i] is coeff of f_i in G[j]
	var syzygies [][]*MPoly  // Syzygy relations: sum_i S[k][i] * f_i = 0

	for i, p := range initial {
		if p.IsZero() {
			continue
		}
		// Initial unit vector in K[x]^m
		unitVec := make([]*MPoly, m)
		for j := 0; j < m; j++ {
			if j == i {
				unitVec[j] = oneMPoly(numVars)
			} else {
				unitVec[j] = NewZeroMPoly(numVars)
			}
		}

		lc := p.LeadingCoeff()
		if lc.Sign() != 0 && (lc.Num().Int64() != 1 || lc.Denom().Int64() != 1) {
			invLC := new(big.Rat).Inv(lc)
			G = append(G, p.Monic())
			cofactors = append(cofactors, scaleCofactorVector(unitVec, invLC, order))
		} else {
			G = append(G, p.Clone())
			cofactors = append(cofactors, unitVec)
		}
	}

	if len(G) <= 1 {
		return G, cofactors, syzygies, nil
	}

	processed := make(map[uint64]bool)
	var pairs []critPair
	var currentG []*MPoly
	for i, p := range G {
		if i > 0 {
			pairs = updatePairsGebauerMoller(currentG, pairs, p, i, processed)
		}
		currentG = append(currentG, p)
	}
	G = currentG

	maxIter := 500
	iter := 0

	for len(pairs) > 0 {
		iter++
		if iter > maxIter {
			return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.groebner_limit_exceeded"))
		}

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

		if canApplyChainCriterion(p1, p2, cp.i, cp.j, G, processed) {
			processed[critPairKey(cp.i, cp.j)] = true
			continue
		}

		// Compute S-polynomial multipliers:
		// S(p1, p2) = (lcm / LT(p1)) * p1 - (lcm / LT(p2)) * p2
		lt1 := p1.LeadingTerm()
		lt2 := p2.LeadingTerm()
		lcmMon := cp.lcm

		t1Mon := lcmMon.Div(lt1.Mon)
		t1Coeff := new(big.Rat).Inv(lt1.Coeff)
		term1 := Term{Coeff: t1Coeff, Mon: t1Mon}

		t2Mon := lcmMon.Div(lt2.Mon)
		t2Coeff := new(big.Rat).Inv(lt2.Coeff)
		term2 := Term{Coeff: t2Coeff, Mon: t2Mon}

		sPoly := p1.MulTerm(term1, order).Sub(p2.MulTerm(term2, order), order)

		// Compute S-poly cofactor vector: term1 * A[cp.i] - term2 * A[cp.j]
		sCofactors := subCofactorVector(
			mulCofactorVectorTerm(cofactors[cp.i], term1, order),
			mulCofactorVectorTerm(cofactors[cp.j], term2, order),
			order,
		)

		// Reduce S-polynomial against current basis G
		qs, rem := polyDivisionWithQuotients(sPoly, G, order)
		processed[critPairKey(cp.i, cp.j)] = true

		// Update cofactor vector for the remainder: remCofactors = sCofactors - sum q_k * cofactors[k]
		remCofactors := make([]*MPoly, m)
		for k := 0; k < m; k++ {
			remCofactors[k] = sCofactors[k].Clone()
		}
		for k, qk := range qs {
			if !qk.IsZero() {
				subK := mulCofactorVectorPoly(cofactors[k], qk, order)
				remCofactors = subCofactorVector(remCofactors, subK, order)
			}
		}

		if rem.IsZero() {
			// S-poly reduced to zero: remCofactors represents a Syzygy relation (sum_i remCofactors[i] * f_i = 0)
			isNonTrivial := false
			for _, c := range remCofactors {
				if !c.IsZero() {
					isNonTrivial = true
					break
				}
			}
			if isNonTrivial {
				syzygies = append(syzygies, remCofactors)
			}
		} else {
			// Found new basis element: normalize to monic
			lcRem := rem.LeadingCoeff()
			invLCRem := new(big.Rat).Inv(lcRem)

			newElem := rem.Monic()
			newCofactors := scaleCofactorVector(remCofactors, invLCRem, order)

			if newElem.LeadingMonomial().IsZero() {
				// Constant 1 in ideal: ideal is <1>
				return []*MPoly{newElem}, [][]*MPoly{newCofactors}, syzygies, nil
			}

			newIdx := len(G)
			pairs = updatePairsGebauerMoller(G, pairs, newElem, newIdx, processed)
			G = append(G, newElem)
			cofactors = append(cofactors, newCofactors)
		}
	}

	return G, cofactors, syzygies, nil
}

// reduceExtendedGroebnerBasis cleans up G and cofactors into a unique reduced Gröbner basis with exact cofactors.
func reduceExtendedGroebnerBasis(G []*MPoly, cofactors [][]*MPoly, order MonomialOrder) ([]*MPoly, [][]*MPoly) {
	if len(G) == 0 {
		return G, cofactors
	}
	m := len(cofactors[0])

	// 1. Filter redundancies
	var minimal []*MPoly
	var minCofactors [][]*MPoly
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
			minCofactors = append(minCofactors, cofactors[i])
		}
	}

	// 2. Interreduction (tail-reduction) with lockstep cofactor transformation
	for i := 0; i < len(minimal); i++ {
		gi := minimal[i]
		ai := minCofactors[i]

		var others []*MPoly
		var otherIndices []int
		for j, gj := range minimal {
			if i != j {
				others = append(others, gj)
				otherIndices = append(otherIndices, j)
			}
		}

		qs, red := polyDivisionWithQuotients(gi, others, order)
		if !red.IsZero() {
			// Update cofactors: ai_new = ai - sum q_k * minCofactors[otherIndices[k]]
			aiNew := make([]*MPoly, m)
			for k := 0; k < m; k++ {
				aiNew[k] = ai[k].Clone()
			}
			for k, qk := range qs {
				if !qk.IsZero() {
					subK := mulCofactorVectorPoly(minCofactors[otherIndices[k]], qk, order)
					aiNew = subCofactorVector(aiNew, subK, order)
				}
			}

			lcRed := red.LeadingCoeff()
			if lcRed.Sign() != 0 && (lcRed.Num().Int64() != 1 || lcRed.Denom().Int64() != 1) {
				invLCRed := new(big.Rat).Inv(lcRed)
				minimal[i] = red.Monic()
				minCofactors[i] = scaleCofactorVector(aiNew, invLCRed, order)
			} else {
				minimal[i] = red
				minCofactors[i] = aiNew
			}
		}
	}

	return minimal, minCofactors
}

// reduceCofactorsModuloSyzygies simplifies cofactors h modulo the syzygy module to minimize monomial term count.
func reduceCofactorsModuloSyzygies(h []*MPoly, syzygies [][]*MPoly, generators []*MPoly, target *MPoly, order MonomialOrder) []*MPoly {
	if len(syzygies) == 0 {
		return h
	}
	res := make([]*MPoly, len(h))
	for i := range h {
		res[i] = h[i].Clone()
	}

	// Greedily reduce high-degree terms if a syzygy can cancel leading monomials
	for _, syz := range syzygies {
		for i := range res {
			if res[i].IsZero() || syz[i].IsZero() {
				continue
			}
			ltH := res[i].LeadingTerm()
			ltS := syz[i].LeadingTerm()
			if ltS.Mon.Divides(ltH.Mon) {
				qMon := ltH.Mon.Div(ltS.Mon)
				qCoeff := new(big.Rat).Quo(ltH.Coeff, ltS.Coeff)
				qTerm := Term{Coeff: qCoeff, Mon: qMon}

				// Candidate simplified cofactors: res - qTerm * syz
				candidate := subCofactorVector(res, mulCofactorVectorTerm(syz, qTerm, order), order)

				// Verify candidate still satisfies identity
				reconstructed := NewZeroMPoly(polyNumVars(target))
				for k := range candidate {
					reconstructed = reconstructed.Add(candidate[k].Mul(generators[k], order), order)
				}
				if target.Sub(reconstructed, order).IsZero() {
					res = candidate
				}
			}
		}
	}
	return res
}

// polyIdealMembershipWithCofactors determines if f in <generators> and computes cofactors h_i such that f = sum h_i f_i.
func polyIdealMembershipWithCofactors(f *MPoly, generators []*MPoly, order MonomialOrder) ([]*MPoly, bool, error) {
	if f.IsZero() {
		numVars := polyNumVars(f)
		m := len(generators)
		zeros := make([]*MPoly, m)
		for i := 0; i < m; i++ {
			zeros[i] = NewZeroMPoly(numVars)
		}
		return zeros, true, nil
	}

	// 1. Run extended Buchberger to obtain Gröbner basis G and representation matrix A
	G, cofactors, syzygies, err := extendedBuchberger(generators, order)
	if err != nil {
		return nil, false, err
	}

	// 2. Clean up to reduced Gröbner basis with exact cofactors
	G, cofactors = reduceExtendedGroebnerBasis(G, cofactors, order)

	// 3. Divide target f by G: f = sum q_j G_j + r
	qs, r := polyDivisionWithQuotients(f, G, order)
	if !r.IsZero() {
		return nil, false, nil // Remainder non-zero => f not in ideal
	}

	// 4. Back-substitute cofactors: h_i = sum_j q_j * A[j][i]
	m := len(generators)
	numVars := polyNumVars(f)
	h := make([]*MPoly, m)
	for i := 0; i < m; i++ {
		h[i] = NewZeroMPoly(numVars)
	}

	for j, qj := range qs {
		if !qj.IsZero() {
			for i := 0; i < m; i++ {
				termI := qj.Mul(cofactors[j][i], order)
				h[i] = h[i].Add(termI, order)
			}
		}
	}

	// 5. Simplify cofactors modulo syzygies to minimize term count
	h = reduceCofactorsModuloSyzygies(h, syzygies, generators, f, order)

	// 6. Defensive self-check: verify exact identity f == sum h_i f_i
	reconstructed := NewZeroMPoly(numVars)
	for i := 0; i < m; i++ {
		reconstructed = reconstructed.Add(h[i].Mul(generators[i], order), order)
	}
	diff := f.Sub(reconstructed, order)
	if !diff.IsZero() {
		return nil, false, fmt.Errorf("internal verification failed: target - sum h_i f_i != 0")
	}

	return h, true, nil
}

// EvalIdealMember checks whether targetNode belongs to the ideal generated by gensNode.
// Syntax: ideal_member(target, [f1, f2, ...], [x, y, ...], [order])
func EvalIdealMember(targetNode, gensNode, varsNode Node, env *Env) (Node, error) {
	varList, ok := varsNode.(*ListNode)
	if !ok || len(varList.Elements) == 0 {
		// Collect free variables from target and gens
		freeVars := CollectFreeVariables(targetNode, gensNode)
		if len(freeVars) == 0 {
			return nil, fmt.Errorf("%s", i18n.T("errors.groebner_no_vars"))
		}
		varNodes := make([]Node, len(freeVars))
		for i, v := range freeVars {
			varNodes[i] = &VarNode{Name: v}
		}
		varList = NewList(varNodes)
	}

	order := OrderGRevLex
	gensList, ok := gensNode.(*ListNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.groebner_polys_not_list"))
	}

	varNames := make([]string, len(varList.Elements))
	varMap := make(map[string]int, len(varList.Elements))
	for i, ve := range varList.Elements {
		vn, ok := ve.(*VarNode)
		if !ok {
			return nil, fmt.Errorf("%s", i18n.T("errors.groebner_var_not_symbol"))
		}
		varNames[i] = vn.Name
		varMap[vn.Name] = i
	}

	evalTarget, err := Eval(expandNode(targetNode))
	if err != nil {
		evalTarget = targetNode
	}
	f, err := nodeToMPoly(evalTarget, varMap, len(varNames), order)
	if err != nil {
		return nil, err
	}

	var gens []*MPoly
	for _, ge := range gensList.Elements {
		evalG, err := Eval(expandNode(ge))
		if err != nil {
			evalG = ge
		}
		gp, err := nodeToMPoly(evalG, varMap, len(varNames), order)
		if err != nil {
			return nil, err
		}
		gens = append(gens, gp)
	}

	cofactors, isMember, err := polyIdealMembershipWithCofactors(f, gens, order)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return &VarNode{Name: "false"}, nil
	}

	var cofNodes []Node
	for _, c := range cofactors {
		cofNodes = append(cofNodes, mpolyToNode(c, varNames))
	}

	return NewList([]Node{&VarNode{Name: "true"}, NewList(cofNodes)}), nil
}
