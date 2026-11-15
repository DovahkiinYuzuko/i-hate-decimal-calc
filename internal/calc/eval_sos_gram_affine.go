package calc

import (
	"math/big"
	"sort"
)

// sosAffineDirection represents a single sparse direction matrix A_k corresponding to free parameter lambda_k.
// Setting Q_{free} = lambda_k shifts the group's pivot entry Q_{pivot} by -(w_free / w_pivot) * lambda_k,
// maintaining v(x)^T Q(lambda) v(x) = P(x) identically for any rational lambda.
type sosAffineDirection struct {
	ParamIndex  int
	PivotBlock  int
	PivotLocalI int
	PivotLocalJ int
	PivotCoeff  *big.Rat // -(w_free / w_pivot)
	FreeBlock   int
	FreeLocalI  int
	FreeLocalJ  int // entry weight +1 on symmetric pair (i,j)/(j,i)
}

type sosGramPair struct {
	Block      int
	LocalI     int
	LocalJ     int
	GlobalI    int
	GlobalJ    int
	Weight     int64 // 1 if diagonal (i == j), 2 if off-diagonal (i < j)
	IsDiagonal bool
}

// sosAffineGroup represents a single product monomial equation sum_r w_r Q_{r} = TargetCoeff.
type sosAffineGroup struct {
	TargetCoeff *big.Rat
	Pairs       []sosGramPair
}

// sosAffineSpace encodes the block-diagonal affine parameterization Q(lambda) = Q_0 + sum_k lambda_k A_k.
type sosAffineSpace struct {
	Analysis        *sosBasisAnalysis
	BlockBase       [][][]*big.Rat
	Directions      []sosAffineDirection
	Groups          []sosAffineGroup
	Dim             int
	EqualLambda     []*big.Rat
	DiagHeavyLambda []*big.Rat
}

// buildSOSAffineSpace constructs the per-monomial decoupled affine subspace Q(lambda)
// together with structural starting points EqualLambda and DiagHeavyLambda.
func buildSOSAffineSpace(analysis *sosBasisAnalysis) *sosAffineSpace {
	numBlocks := len(analysis.Blocks)
	basis := analysis.Basis

	// Compute local index within each sign-symmetry block.
	localIdx := make([]int, len(basis))
	blockBase := make([][][]*big.Rat, numBlocks)
	for b, blk := range analysis.Blocks {
		sz := len(blk)
		mat := make([][]*big.Rat, sz)
		for r := 0; r < sz; r++ {
			mat[r] = make([]*big.Rat, sz)
			for c := 0; c < sz; c++ {
				mat[r][c] = big.NewRat(0, 1)
			}
		}
		blockBase[b] = mat
		for lIdx, gIdx := range blk {
			localIdx[gIdx] = lIdx
		}
	}

	// Group intra-block unordered pairs (i <= j) by product monomial gamma = b_i * b_j.
	groups := make(map[string][]sosGramPair)
	groupMons := make(map[string]Monomial)
	for i := 0; i < len(basis); i++ {
		bi := analysis.BlockOf[i]
		li := localIdx[i]
		for j := i; j < len(basis); j++ {
			if analysis.BlockOf[j] != bi {
				continue
			}
			lj := localIdx[j]
			prod := basis[i].Mul(basis[j])
			key := monomialKey(prod)
			w := int64(2)
			if i == j {
				w = 1
			}
			groups[key] = append(groups[key], sosGramPair{
				Block:      bi,
				LocalI:     li,
				LocalJ:     lj,
				GlobalI:    i,
				GlobalJ:    j,
				Weight:     w,
				IsDiagonal: (i == j),
			})
			if _, ok := groupMons[key]; !ok {
				groupMons[key] = prod
			}
		}
	}

	// Sort product monomial keys canonically so direction numbering is deterministic.
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		return CompareMonomials(groupMons[keys[a]], groupMons[keys[b]], analysis.Order) > 0
	})

	var directions []sosAffineDirection
	var affineGroups []sosAffineGroup
	var equalVals []*big.Rat
	var diagVals []*big.Rat

	for _, key := range keys {
		pairs := groups[key]
		c := coeffAt(analysis.CoeffMap, key)
		affineGroups = append(affineGroups, sosAffineGroup{
			TargetCoeff: new(big.Rat).Set(c),
			Pairs:       pairs,
		})
		m := int64(len(pairs))

		// Choose pivot pair: first pair in deterministic order.
		piv := pairs[0]
		baseVal := new(big.Rat).Quo(c, big.NewRat(piv.Weight, 1))
		setSymmetricMatrixEntry(blockBase[piv.Block], piv.LocalI, piv.LocalJ, baseVal)

		if len(pairs) <= 1 {
			continue
		}

		// Precompute structural values for all free pairs (r >= 1) in this monomial group.
		diagCount := int64(0)
		offDiagCount := int64(0)
		for _, pr := range pairs {
			if pr.IsDiagonal {
				diagCount++
			} else {
				offDiagCount++
			}
		}

		for r := 1; r < len(pairs); r++ {
			fr := pairs[r]
			paramIdx := len(directions)
			// Pivot adjustment = -(w_free / w_pivot)
			pivCoeff := big.NewRat(-fr.Weight, piv.Weight)

			directions = append(directions, sosAffineDirection{
				ParamIndex:  paramIdx,
				PivotBlock:  piv.Block,
				PivotLocalI: piv.LocalI,
				PivotLocalJ: piv.LocalJ,
				PivotCoeff:  pivCoeff,
				FreeBlock:   fr.Block,
				FreeLocalI:  fr.LocalI,
				FreeLocalJ:  fr.LocalJ,
			})

			// Equal split assigns c / (m * w_fr) to each pair.
			eqVal := new(big.Rat).Quo(c, big.NewRat(m*fr.Weight, 1))
			equalVals = append(equalVals, eqVal)

			// Diagonal-heavy split puts nonnegative coefficients onto diagonal pairs
			// and negative coefficients onto off-diagonal pairs when possible.
			var dVal *big.Rat
			if c.Sign() > 0 && diagCount > 0 {
				if fr.IsDiagonal {
					dVal = new(big.Rat).Quo(c, big.NewRat(diagCount*fr.Weight, 1))
				} else {
					dVal = big.NewRat(0, 1)
				}
			} else if c.Sign() < 0 && offDiagCount > 0 {
				if !fr.IsDiagonal {
					dVal = new(big.Rat).Quo(c, big.NewRat(offDiagCount*fr.Weight, 1))
				} else {
					dVal = big.NewRat(0, 1)
				}
			} else {
				dVal = new(big.Rat).Set(eqVal)
			}
			diagVals = append(diagVals, dVal)
		}
	}

	return &sosAffineSpace{
		Analysis:        analysis,
		BlockBase:       blockBase,
		Directions:      directions,
		Groups:          affineGroups,
		Dim:             len(directions),
		EqualLambda:     equalVals,
		DiagHeavyLambda: diagVals,
	}
}

func setSymmetricMatrixEntry(mat [][]*big.Rat, i, j int, val *big.Rat) {
	mat[i][j] = new(big.Rat).Set(val)
	if i != j {
		mat[j][i] = new(big.Rat).Set(val)
	}
}

func addSymmetricMatrixEntry(mat [][]*big.Rat, i, j int, delta *big.Rat) {
	if delta.Sign() == 0 {
		return
	}
	mat[i][j] = new(big.Rat).Add(mat[i][j], delta)
	if i != j {
		mat[j][i] = new(big.Rat).Add(mat[j][i], delta)
	}
}

// EvaluateBlock evaluates Q^{(b)}(lambda) as an exact rational symmetric matrix.
func (s *sosAffineSpace) EvaluateBlock(b int, lambda []*big.Rat) [][]*big.Rat {
	base := s.BlockBase[b]
	sz := len(base)
	Q := make([][]*big.Rat, sz)
	for i := 0; i < sz; i++ {
		Q[i] = make([]*big.Rat, sz)
		for j := 0; j < sz; j++ {
			Q[i][j] = new(big.Rat).Set(base[i][j])
		}
	}

	for k, dir := range s.Directions {
		if k >= len(lambda) || lambda[k] == nil || lambda[k].Sign() == 0 {
			continue
		}
		lk := lambda[k]
		if dir.PivotBlock == b {
			delta := new(big.Rat).Mul(lk, dir.PivotCoeff)
			addSymmetricMatrixEntry(Q, dir.PivotLocalI, dir.PivotLocalJ, delta)
		}
		if dir.FreeBlock == b {
			addSymmetricMatrixEntry(Q, dir.FreeLocalI, dir.FreeLocalJ, lk)
		}
	}
	return Q
}

// DirectionQuadFormRow computes (coeffs, constVal) for a witness vector w in block b:
// w^T Q^{(b)}(lambda) w = constVal + sum_k coeffs[k] * lambda_k.
func (s *sosAffineSpace) DirectionQuadFormRow(b int, w []*big.Rat) ([]*big.Rat, *big.Rat) {
	constVal := evalSymQuadForm(s.BlockBase[b], w)
	coeffs := make([]*big.Rat, s.Dim)
	for k := range coeffs {
		coeffs[k] = big.NewRat(0, 1)
	}

	pairWeight := func(i, j int) *big.Rat {
		prod := new(big.Rat).Mul(w[i], w[j])
		if i != j {
			prod.Mul(prod, big.NewRat(2, 1))
		}
		return prod
	}

	for k, dir := range s.Directions {
		if dir.PivotBlock == b {
			pw := pairWeight(dir.PivotLocalI, dir.PivotLocalJ)
			pw.Mul(pw, dir.PivotCoeff)
			coeffs[k].Add(coeffs[k], pw)
		}
		if dir.FreeBlock == b {
			fw := pairWeight(dir.FreeLocalI, dir.FreeLocalJ)
			coeffs[k].Add(coeffs[k], fw)
		}
	}

	return coeffs, constVal
}

// evalSymQuadForm computes w^T A w for exact rational symmetric matrix A and vector w.
func evalSymQuadForm(A [][]*big.Rat, w []*big.Rat) *big.Rat {
	sum := big.NewRat(0, 1)
	for i := 0; i < len(w); i++ {
		if w[i].Sign() == 0 {
			continue
		}
		diag := new(big.Rat).Mul(w[i], w[i])
		diag.Mul(diag, A[i][i])
		sum.Add(sum, diag)
		for j := i + 1; j < len(w); j++ {
			if w[j].Sign() == 0 || A[i][j].Sign() == 0 {
				continue
			}
			off := new(big.Rat).Mul(w[i], w[j])
			off.Mul(off, A[i][j])
			off.Mul(off, big.NewRat(2, 1))
			sum.Add(sum, off)
		}
	}
	return sum
}

// reconstructMPolyFromAffine expands sum_b v_b^T Q^{(b)}(lambda) v_b into an MPoly.
func reconstructMPolyFromAffine(s *sosAffineSpace, lambda []*big.Rat, order MonomialOrder) *MPoly {
	n := len(s.Analysis.Vars)
	termAcc := make(map[string]Term)
	for b, blk := range s.Analysis.Blocks {
		Qb := s.EvaluateBlock(b, lambda)
		for li, gi := range blk {
			for lj, gj := range blk {
				c := Qb[li][lj]
				if c.Sign() == 0 {
					continue
				}
				mon := s.Analysis.Basis[gi].Mul(s.Analysis.Basis[gj])
				key := monomialKey(mon)
				if prev, ok := termAcc[key]; ok {
					prev.Coeff = new(big.Rat).Add(prev.Coeff, c)
					termAcc[key] = prev
				} else {
					termAcc[key] = Term{Coeff: new(big.Rat).Set(c), Mon: mon}
				}
			}
		}
	}

	terms := make([]Term, 0, len(termAcc))
	sugar := 0
	for _, t := range termAcc {
		if t.Coeff.Sign() != 0 {
			terms = append(terms, t)
			if t.Mon.TotalDeg > sugar {
				sugar = t.Mon.TotalDeg
			}
		}
	}
	sortPolyTerms(terms, order)
	if len(terms) == 0 {
		return NewZeroMPoly(n)
	}
	return &MPoly{Terms: terms, Sugar: sugar}
}
