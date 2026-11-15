package calc

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/lp"
)

// sosMaxBasisCandidates guards against combinatorial explosion in Half Newton lattice enumeration.
const sosMaxBasisCandidates = 4096

// sosBasisAnalysis captures preprocessing, exact Half Newton lattice extraction,
// Reznick diagonal pruning, support coverage verification, and sign-symmetry blocking.
type sosBasisAnalysis struct {
	Vars          []string
	Order         MonomialOrder
	Poly          *MPoly
	CoeffMap      map[string]*big.Rat
	TotalDeg      int
	HalfDeg       int
	IsHomogeneous bool
	Basis         []Monomial
	Blocks        [][]int
	BlockOf       []int
	Impossible    bool
	Refutation    map[string]*big.Rat
}

// monomialKey returns a deterministic canonical key for a monomial exponent slice.
func monomialKey(m Monomial) string {
	var b strings.Builder
	for i, e := range m.Exps {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Itoa(e))
	}
	return b.String()
}

// coeffAt looks up the rational coefficient of a monomial in polyCoeffMap (zero if absent).
func coeffAt(coeffMap map[string]*big.Rat, key string) *big.Rat {
	if c, ok := coeffMap[key]; ok {
		return new(big.Rat).Set(c)
	}
	return big.NewRat(0, 1)
}

// analyzeSOSBasis runs structural preprocessing, Half Newton Polytope lattice extraction via LP,
// Reznick pruning, support coverage validation, and sign-symmetry block partitioning.
func analyzeSOSBasis(p *MPoly, vars []string, order MonomialOrder) (*sosBasisAnalysis, error) {
	n := len(vars)
	analysis := &sosBasisAnalysis{
		Vars:     append([]string(nil), vars...),
		Order:    order,
		Poly:     p,
		CoeffMap: make(map[string]*big.Rat, len(p.Terms)),
	}

	if p == nil || p.IsZero() {
		analysis.IsHomogeneous = true
		return analysis, nil
	}

	minDeg := p.Terms[0].Mon.TotalDeg
	maxDeg := minDeg
	minExp := append([]int(nil), p.Terms[0].Mon.Exps...)
	maxExp := append([]int(nil), p.Terms[0].Mon.Exps...)

	for _, t := range p.Terms {
		key := monomialKey(t.Mon)
		analysis.CoeffMap[key] = new(big.Rat).Set(t.Coeff)
		if t.Mon.TotalDeg < minDeg {
			minDeg = t.Mon.TotalDeg
		}
		if t.Mon.TotalDeg > maxDeg {
			maxDeg = t.Mon.TotalDeg
		}
		for i := 0; i < n; i++ {
			if t.Mon.Exps[i] < minExp[i] {
				minExp[i] = t.Mon.Exps[i]
			}
			if t.Mon.Exps[i] > maxExp[i] {
				maxExp[i] = t.Mon.Exps[i]
			}
		}
	}

	analysis.TotalDeg = maxDeg
	analysis.IsHomogeneous = (minDeg == maxDeg)

	// Odd total degree cannot be globally nonnegative unless refuted or degenerate.
	if maxDeg%2 != 0 {
		analysis.Impossible = true
		analysis.Refutation = findOddDegreeRefutation(p, vars)
		return analysis, nil
	}
	analysis.HalfDeg = maxDeg / 2

	// Vertex parity and sign check (Reznick 1978): every vertex of New(P) must lie in (2Z)^n
	// and have a strictly positive coefficient.
	vertexViolated, err := hasImpossibleNewtonVertex(p, n)
	if err != nil {
		return nil, err
	}
	if vertexViolated {
		analysis.Impossible = true
		analysis.Refutation = findCheapRefutation(p, vars)
		return analysis, nil
	}

	// Degree-0 constant polynomial
	if maxDeg == 0 {
		c := p.Terms[0].Coeff
		if c.Sign() < 0 {
			analysis.Impossible = true
			pt := make(map[string]*big.Rat, n)
			for _, v := range vars {
				pt[v] = big.NewRat(0, 1)
			}
			analysis.Refutation = pt
			return analysis, nil
		}
		analysis.Basis = []Monomial{zeroMonomial(n)}
		analysis.Blocks = [][]int{{0}}
		analysis.BlockOf = []int{0}
		return analysis, nil
	}

	halfMinExp := make([]int, n)
	halfMaxExp := make([]int, n)
	for i := 0; i < n; i++ {
		halfMinExp[i] = (minExp[i] + 1) / 2
		halfMaxExp[i] = maxExp[i] / 2
		if halfMinExp[i] > halfMaxExp[i] {
			analysis.Impossible = true
			return analysis, nil
		}
	}
	halfMinDeg := (minDeg + 1) / 2
	halfMaxDeg := maxDeg / 2

	candidates, err := enumerateLatticeCandidates(n, halfMinExp, halfMaxExp, halfMinDeg, halfMaxDeg)
	if err != nil {
		return nil, err
	}

	// Filter candidates strictly by 2*alpha in conv(supp(P)).
	basis := make([]Monomial, 0, len(candidates))
	for _, cand := range candidates {
		doubleMon := cand.Mul(cand)
		if _, inSupp := analysis.CoeffMap[monomialKey(doubleMon)]; inSupp {
			basis = append(basis, cand)
			continue
		}
		inside, err := isPointInHalfNewtonPolytope(p, analysis.CoeffMap, cand, n)
		if err != nil {
			return nil, err
		}
		if inside {
			basis = append(basis, cand)
		}
	}

	// Sort basis in requested monomial order (descending) for deterministic canonical behavior.
	sort.Slice(basis, func(i, j int) bool {
		return CompareMonomials(basis[i], basis[j], order) > 0
	})

	// Iterative Reznick diagonal-zero pruning + uniqueness negative-diag detection.
	pruned, impossible := pruneBasisByReznick(basis, analysis.CoeffMap)
	if impossible {
		analysis.Impossible = true
		analysis.Refutation = findCheapRefutation(p, vars)
		return analysis, nil
	}
	analysis.Basis = pruned

	// Support coverage verification: every nonzero monomial in P must be expressible as b_i * b_j.
	if !verifySupportCoverage(p, analysis.Basis) {
		analysis.Impossible = true
		analysis.Refutation = findCheapRefutation(p, vars)
		return analysis, nil
	}

	if len(analysis.Basis) > 0 {
		analysis.Blocks, analysis.BlockOf = partitionSignSymmetryBlocks(p, analysis.Basis, n)
	}
	return analysis, nil
}

// hasImpossibleNewtonVertex checks whether any term in P with odd parity or negative coefficient
// is an extreme point (vertex) of conv(supp(P)). Any such vertex proves P is not SOS.
func hasImpossibleNewtonVertex(p *MPoly, n int) (bool, error) {
	m := len(p.Terms)
	if m == 0 {
		return false, nil
	}
	if m == 1 {
		t := p.Terms[0]
		if t.Coeff.Sign() < 0 {
			return true, nil
		}
		for _, e := range t.Mon.Exps {
			if e%2 != 0 {
				return true, nil
			}
		}
		return false, nil
	}

	if n == 1 {
		minIdx, maxIdx := 0, 0
		for i, t := range p.Terms {
			if t.Mon.Exps[0] < p.Terms[minIdx].Mon.Exps[0] {
				minIdx = i
			}
			if t.Mon.Exps[0] > p.Terms[maxIdx].Mon.Exps[0] {
				maxIdx = i
			}
		}
		for _, idx := range []int{minIdx, maxIdx} {
			if p.Terms[idx].Coeff.Sign() < 0 || p.Terms[idx].Mon.Exps[0]%2 != 0 {
				return true, nil
			}
		}
		return false, nil
	}

	for idx, t := range p.Terms {
		suspect := t.Coeff.Sign() < 0
		if !suspect {
			for _, e := range t.Mon.Exps {
				if e%2 != 0 {
					suspect = true
					break
				}
			}
		}
		if !suspect {
			continue
		}

		// Test whether t.Mon belongs to conv(supp(P) \ {t.Mon}). If infeasible, t.Mon is a vertex.
		otherCount := m - 1
		varNames := make([]string, otherCount)
		obj := make([]*big.Rat, otherCount)
		sumCoeffs := make([]*big.Rat, otherCount)
		for k := 0; k < otherCount; k++ {
			varNames[k] = "v_" + strconv.Itoa(k) // txthc-ignore
			obj[k] = big.NewRat(0, 1)
			sumCoeffs[k] = big.NewRat(1, 1)
		}

		constraints := make([]lp.Constraint, 0, 1+n)
		constraints = append(constraints, lp.Constraint{
			Name:   "convex_sum", // txthc-ignore
			Coeffs: sumCoeffs,
			Sense:  lp.RelEq,
			RHS:    big.NewRat(1, 1),
		})

		for coord := 0; coord < n; coord++ {
			row := make([]*big.Rat, otherCount)
			col := 0
			for j, tj := range p.Terms {
				if j == idx {
					continue
				}
				row[col] = big.NewRat(int64(tj.Mon.Exps[coord]), 1)
				col++
			}
			constraints = append(constraints, lp.Constraint{
				Name:   "coord_" + strconv.Itoa(coord), // txthc-ignore
				Coeffs: row,
				Sense:  lp.RelEq,
				RHS:    big.NewRat(int64(t.Mon.Exps[coord]), 1),
			})
		}

		prob := &lp.Problem{
			Sense:       lp.OptMinimize,
			Objective:   obj,
			ObjConstant: big.NewRat(0, 1),
			VarNames:    varNames,
			Constraints: constraints,
		}
		res, err := lp.Solve(prob)
		if err != nil {
			return false, err
		}
		if res.Status == lp.StatusInfeasible {
			return true, nil
		}
	}

	return false, nil
}

// enumerateLatticeCandidates enumerates all integer lattice points alpha in N^n satisfying
// coordinate and total-degree box bounds derived from New(P)/2, pruning dead branches via suffix bounds.
func enumerateLatticeCandidates(n int, minExp, maxExp []int, minDeg, maxDeg int) ([]Monomial, error) {
	if minDeg > maxDeg {
		return nil, nil
	}
	for i := 0; i < n; i++ {
		if minExp[i] > maxExp[i] {
			return nil, nil
		}
	}

	suffixMin := make([]int, n+1)
	suffixMax := make([]int, n+1)
	for i := n - 1; i >= 0; i-- {
		suffixMin[i] = suffixMin[i+1] + minExp[i]
		suffixMax[i] = suffixMax[i+1] + maxExp[i]
	}

	var result []Monomial
	current := make([]int, n)

	var dfs func(idx int, degSoFar int) error
	dfs = func(idx int, degSoFar int) error {
		if len(result) >= sosMaxBasisCandidates {
			return fmt.Errorf("%s", i18n.T("errors.sos_basis_limit_exceeded", sosMaxBasisCandidates))
		}
		if degSoFar+suffixMax[idx] < minDeg || degSoFar+suffixMin[idx] > maxDeg {
			return nil
		}
		if idx == n {
			result = append(result, newMonomial(current))
			return nil
		}
		lo := minExp[idx]
		if need := minDeg - degSoFar - suffixMax[idx+1]; need > lo {
			lo = need
		}
		hi := maxExp[idx]
		if rem := maxDeg - degSoFar - suffixMin[idx+1]; rem < hi {
			hi = rem
		}
		for e := lo; e <= hi; e++ {
			current[idx] = e
			if err := dfs(idx+1, degSoFar+e); err != nil {
				return err
			}
		}
		return nil
	}

	if err := dfs(0, 0); err != nil {
		return nil, err
	}
	return result, nil
}

// isPointInHalfNewtonPolytope checks whether 2*alpha belongs to conv(supp(P)) using internal/lp.
func isPointInHalfNewtonPolytope(p *MPoly, _ map[string]*big.Rat, alpha Monomial, n int) (bool, error) {
	if n <= 1 {
		// Box bounds on 1D already equal conv(supp(P))/2 ∩ N^1.
		return true, nil
	}

	m := len(p.Terms)
	varNames := make([]string, m)
	obj := make([]*big.Rat, m)
	sumCoeffs := make([]*big.Rat, m)
	for j := 0; j < m; j++ {
		varNames[j] = "theta_" + strconv.Itoa(j) // txthc-ignore
		obj[j] = big.NewRat(0, 1)
		sumCoeffs[j] = big.NewRat(1, 1)
	}

	constraints := make([]lp.Constraint, 0, 1+n)
	constraints = append(constraints, lp.Constraint{
		Name:   "convex_sum", // txthc-ignore
		Coeffs: sumCoeffs,
		Sense:  lp.RelEq,
		RHS:    big.NewRat(1, 1),
	})

	for coord := 0; coord < n; coord++ {
		row := make([]*big.Rat, m)
		for j, t := range p.Terms {
			row[j] = big.NewRat(int64(t.Mon.Exps[coord]), 1)
		}
		constraints = append(constraints, lp.Constraint{
			Name:   "half_coord_" + strconv.Itoa(coord), // txthc-ignore
			Coeffs: row,
			Sense:  lp.RelEq,
			RHS:    big.NewRat(int64(2*alpha.Exps[coord]), 1),
		})
	}

	prob := &lp.Problem{
		Sense:       lp.OptMinimize,
		Objective:   obj,
		ObjConstant: big.NewRat(0, 1),
		VarNames:    varNames,
		Constraints: constraints,
	}
	res, err := lp.Solve(prob)
	if err != nil {
		return false, err
	}
	return res.Status == lp.StatusOptimal, nil
}

// pruneBasisByReznick iteratively removes monomials b_k whose square 2*b_k cannot be produced
// by any other pair in the basis and has zero coefficient in P.
// If such a unique diagonal square has negative coefficient in P, returns impossible = true.
func pruneBasisByReznick(basis []Monomial, coeffMap map[string]*big.Rat) ([]Monomial, bool) {
	cur := append([]Monomial(nil), basis...)
	for len(cur) > 0 {
		pairCount := make(map[string]int, len(cur)*len(cur))
		diagKeys := make([]string, len(cur))
		for i := 0; i < len(cur); i++ {
			for j := i; j < len(cur); j++ {
				key := monomialKey(cur[i].Mul(cur[j]))
				pairCount[key]++
				if i == j {
					diagKeys[i] = key
				}
			}
		}

		changed := false
		next := make([]Monomial, 0, len(cur))
		for i, m := range cur {
			dKey := diagKeys[i]
			if pairCount[dKey] == 1 {
				c, ok := coeffMap[dKey]
				if !ok || c.Sign() == 0 {
					changed = true
					continue
				}
				if c.Sign() < 0 {
					return nil, true
				}
			}
			next = append(next, m)
		}
		if !changed {
			return cur, false
		}
		cur = next
	}
	return cur, false
}

// verifySupportCoverage ensures every nonzero term of P can be expressed as b_i * b_j.
func verifySupportCoverage(p *MPoly, basis []Monomial) bool {
	if p == nil || p.IsZero() {
		return true
	}
	if len(basis) == 0 {
		return false
	}
	covered := make(map[string]bool, len(basis)*len(basis))
	for i := 0; i < len(basis); i++ {
		for j := i; j < len(basis); j++ {
			covered[monomialKey(basis[i].Mul(basis[j]))] = true
		}
	}
	for _, t := range p.Terms {
		if t.Coeff.Sign() != 0 && !covered[monomialKey(t.Mon)] {
			return false
		}
	}
	return true
}

// partitionSignSymmetryBlocks implements Gatermann-Parrilo / Löfberg sign-symmetry block
// decomposition over F_2^n: W = span(supp(P) mod 2), grouping b_i by their coset b_i mod W.
func partitionSignSymmetryBlocks(p *MPoly, basis []Monomial, n int) ([][]int, []int) {
	// Build row-echelon F_2 basis for W = span(supp(P) mod 2).
	pivotCol := make([]int, 0, n)
	wBasis := make([][]int, 0, n)

	reduceModW := func(vec []int) []int {
		v := append([]int(nil), vec...)
		for r, pc := range pivotCol {
			if v[pc] != 0 {
				row := wBasis[r]
				for c := pc; c < n; c++ {
					v[c] ^= row[c]
				}
			}
		}
		return v
	}

	for _, t := range p.Terms {
		if t.Coeff.Sign() == 0 {
			continue
		}
		par := make([]int, n)
		for i := 0; i < n; i++ {
			par[i] = t.Mon.Exps[i] & 1
		}
		red := reduceModW(par)
		firstOne := -1
		for c := 0; c < n; c++ {
			if red[c] != 0 {
				firstOne = c
				break
			}
		}
		if firstOne >= 0 {
			pivotCol = append(pivotCol, firstOne)
			wBasis = append(wBasis, red)
		}
	}

	// Keep W rows reduced nicely by sorting pivots? Standard sequential reduction already uniquely
	// canonicalizes mod span only if back-substituted; perform back-substitution into RREF over F_2.
	for i := len(wBasis) - 1; i >= 0; i-- {
		pc := pivotCol[i]
		for j := 0; j < i; j++ {
			if wBasis[j][pc] != 0 {
				for c := 0; c < n; c++ {
					wBasis[j][c] ^= wBasis[i][c]
				}
			}
		}
	}

	canonicalCoset := func(m Monomial) string {
		par := make([]int, n)
		for i := 0; i < n; i++ {
			par[i] = m.Exps[i] & 1
		}
		red := reduceModW(par)
		var b strings.Builder
		for i, bit := range red {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte(byte('0' + bit))
		}
		return b.String()
	}

	cosetIdx := make(map[string]int)
	var blocks [][]int
	blockOf := make([]int, len(basis))

	for i, m := range basis {
		ck := canonicalCoset(m)
		bIdx, exists := cosetIdx[ck]
		if !exists {
			bIdx = len(blocks)
			cosetIdx[ck] = bIdx
			blocks = append(blocks, nil)
		}
		blocks[bIdx] = append(blocks[bIdx], i)
		blockOf[i] = bIdx
	}

	return blocks, blockOf
}
