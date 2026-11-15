package calc

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/lp"
)

const (
	sosMaxCuttingPlaneIters   = 24
	sosMaxFacialRounds        = 6
	sosFacialStallThreshold   = 3
	sosMaxFacialCandidatesTry = 6
)

type sosFSMState int

const (
	sosStateInit sosFSMState = iota
	sosStateDirectLadder
	sosStateDSOSLP
	sosStateWarmStartDR
	sosStateCuttingPlaneLP
	sosStateFacialReduction
	sosStateVerify
	sosStateCertified
	sosStateRefuted
	sosStateExhausted
)

type sosBlockCut struct {
	Block  int
	Vector []*big.Rat
	Key    string
}

// SOSDecomposeFSM implements a deterministic state machine searching the affine Gram space
// via direct structural points, DSOS/scaled-DSOS LPs, cutting-plane LPs, and partial facial reduction.
type SOSDecomposeFSM struct {
	state            sosFSMState
	poly             *MPoly
	vars             []string
	order            MonomialOrder
	analysis         *sosBasisAnalysis
	space            *sosAffineSpace
	cuts             []sosBlockCut
	cutSeen          map[string]bool
	facialVectors    []sosBlockCut
	facialSeen       map[string]bool
	candidateLambda  []*big.Rat
	candidateLDLTs   []sosLDLResult
	prevZeroLambda   []*big.Rat
	zeroMarginStreak int
	cutIter          int
	facialRounds     int
	result           *SOSResult
}

// runSOSDecomposeFSM executes the deterministic SOS decomposition state machine on p.
func runSOSDecomposeFSM(p *MPoly, vars []string, order MonomialOrder) (*SOSResult, error) {
	fsm := &SOSDecomposeFSM{
		state:      sosStateInit,
		poly:       p,
		vars:       vars,
		order:      order,
		cutSeen:    make(map[string]bool),
		facialSeen: make(map[string]bool),
	}

	for {
		switch fsm.state {
		case sosStateInit:
			if err := fsm.stepInit(); err != nil {
				return nil, err
			}
		case sosStateDirectLadder:
			fsm.stepDirectLadder()
		case sosStateDSOSLP:
			if err := fsm.stepDSOSLP(); err != nil {
				return nil, err
			}
		case sosStateWarmStartDR:
			if err := fsm.stepWarmStartDR(); err != nil {
				return nil, err
			}
		case sosStateCuttingPlaneLP:
			if err := fsm.stepCuttingPlaneLP(); err != nil {
				return nil, err
			}
		case sosStateFacialReduction:
			if err := fsm.stepFacialReduction(); err != nil {
				return nil, err
			}
		case sosStateVerify:
			fsm.stepVerify()
		case sosStateCertified, sosStateRefuted, sosStateExhausted:
			if fsm.result != nil {
				return fsm.result, nil
			}
			return &SOSResult{
				Status:   SOSNoCertificateFound,
				Terms:    nil,
				Residual: p.Clone(),
				Vars:     vars,
			}, nil
		}
	}
}

func (f *SOSDecomposeFSM) stepInit() error {
	analysis, err := analyzeSOSBasis(f.poly, f.vars, f.order)
	if err != nil {
		return err
	}
	f.analysis = analysis

	if analysis.Refutation != nil {
		f.result = &SOSResult{
			Status:         SOSRefuted,
			Counterexample: analysis.Refutation,
			Residual:       f.poly.Clone(),
			Vars:           f.vars,
		}
		f.state = sosStateRefuted
		return nil
	}

	if analysis.Impossible || len(analysis.Basis) == 0 {
		f.state = sosStateExhausted
		return nil
	}

	f.space = buildSOSAffineSpace(analysis)
	f.state = sosStateDirectLadder
	return nil
}

func (f *SOSDecomposeFSM) evaluateBlocksPSD(lambda []*big.Rat) ([]sosLDLResult, bool) {
	numBlocks := len(f.space.BlockBase)
	results := make([]sosLDLResult, numBlocks)
	allPSD := true
	for b := 0; b < numBlocks; b++ {
		Qb := f.space.EvaluateBlock(b, lambda)
		resB := exactRationalLDLTWithWitness(Qb)
		results[b] = resB
		if !resB.IsPSD {
			allPSD = false
		}
	}
	return results, allPSD
}

func (f *SOSDecomposeFSM) stepDirectLadder() {
	if f.space.Dim == 0 {
		if results, ok := f.evaluateBlocksPSD(nil); ok {
			f.candidateLambda = nil
			f.candidateLDLTs = results
			f.state = sosStateVerify
			return
		}
		f.state = sosStateExhausted
		return
	}

	candidates := [][]*big.Rat{
		makeZeroRatVec(f.space.Dim),
		f.space.DiagHeavyLambda,
		f.space.EqualLambda,
	}
	if f.space.Dim == 1 {
		for _, num := range []int64{-2, -1, 1, 2} {
			candidates = append(candidates, []*big.Rat{big.NewRat(num, 1)})
			candidates = append(candidates, []*big.Rat{big.NewRat(num, 2)})
		}
	}

	for _, lam := range candidates {
		if results, ok := f.evaluateBlocksPSD(lam); ok {
			f.candidateLambda = lam
			f.candidateLDLTs = results
			f.state = sosStateVerify
			return
		}
	}

	f.state = sosStateDSOSLP
}

func (f *SOSDecomposeFSM) stepDSOSLP() error {
	// Try standard DSOS (u_i = 1) first, then diagonal-estimated scaled DSOS.
	if lam, ok, err := f.solveScaledDSOSLP(false); err != nil {
		return err
	} else if ok {
		if results, psd := f.evaluateBlocksPSD(lam); psd {
			f.candidateLambda = lam
			f.candidateLDLTs = results
			f.state = sosStateVerify
			return nil
		}
	}

	if lam, ok, err := f.solveScaledDSOSLP(true); err != nil {
		return err
	} else if ok {
		if results, psd := f.evaluateBlocksPSD(lam); psd {
			f.candidateLambda = lam
			f.candidateLDLTs = results
			f.state = sosStateVerify
			return nil
		}
	}

	f.seedInitialCuttingPlanes()
	f.state = sosStateWarmStartDR
	return nil
}

func (f *SOSDecomposeFSM) stepWarmStartDR() error {
	warm := runSOSWarmStartDR(f.space)
	for _, lam := range warm.LambdaCandidates {
		if results, psd := f.evaluateBlocksPSD(lam); psd {
			f.candidateLambda = lam
			f.candidateLDLTs = results
			f.state = sosStateVerify
			return nil
		} else {
			for b, rb := range results {
				if !rb.IsPSD && len(rb.Witness) > 0 {
					f.addCut(b, rb.Witness)
				}
			}
		}
	}

	// Add near-kernel eigenvector hints as cuts and test whether any are strictly zero-forced
	// across the cutting-plane polyhedron (sound facial reduction).
	if len(warm.KernelHints) > 0 {
		for _, hint := range warm.KernelHints {
			f.addCut(hint.Block, hint.Vector)
		}
		d := f.space.Dim
		totalVars := 2 * d
		varNames := make([]string, totalVars)
		for k := 0; k < d; k++ {
			varNames[2*k] = "lp_" + strconv.Itoa(k)   // txthc-ignore
			varNames[2*k+1] = "lm_" + strconv.Itoa(k) // txthc-ignore
		}
		baseConstraints := f.buildPolyhedronConstraints(totalVars, false, -1)
		for _, hint := range warm.KernelHints {
			if f.facialSeen[hint.Key] {
				continue
			}
			forcedZero, err := f.isCutForcedZeroInPolyhedron(totalVars, varNames, baseConstraints, hint)
			if err != nil {
				return err
			}
			if forcedZero {
				f.addFacialVector(hint.Block, hint.Vector)
			}
		}
	}

	f.state = sosStateCuttingPlaneLP
	return nil
}

func (f *SOSDecomposeFSM) isCutForcedZeroInPolyhedron(totalVars int, varNames []string, baseConstraints []lp.Constraint, cut sosBlockCut) (bool, error) {
	d := f.space.Dim
	coeffs, c0 := f.space.DirectionQuadFormRow(cut.Block, cut.Vector)
	obj := makeZeroRatVec(totalVars)
	capRow := makeZeroRatVec(totalVars)
	for k := 0; k < d; k++ {
		if coeffs[k].Sign() == 0 {
			continue
		}
		obj[2*k] = new(big.Rat).Set(coeffs[k])
		obj[2*k+1] = new(big.Rat).Neg(coeffs[k])
		capRow[2*k] = new(big.Rat).Set(coeffs[k])
		capRow[2*k+1] = new(big.Rat).Neg(coeffs[k])
	}
	cons := append([]lp.Constraint(nil), baseConstraints...)
	cons = append(cons, lp.Constraint{
		Name:   "facial_cap", // txthc-ignore
		Coeffs: capRow,
		Sense:  lp.RelLe,
		RHS:    new(big.Rat).Sub(big.NewRat(1, 1), c0),
	})

	prob := &lp.Problem{
		Sense:       lp.OptMaximize,
		Objective:   obj,
		ObjConstant: new(big.Rat).Set(c0),
		VarNames:    varNames,
		Constraints: cons,
	}
	res, err := lp.Solve(prob)
	if err != nil {
		return false, err
	}
	return res.Status == lp.StatusOptimal && res.OptimalValue.Sign() == 0, nil
}

// solveScaledDSOSLP encodes u_i^2 * Q_{ii}(lambda) - sum_{j != i} u_i * u_j * s_{ij} - t >= 0,
// |Q_{ij}(lambda)| <= s_{ij}, and 0 <= t <= 1 via internal/lp.
func (f *SOSDecomposeFSM) solveScaledDSOSLP(useDiagScaling bool) ([]*big.Rat, bool, error) {
	d := f.space.Dim
	numBlocks := len(f.space.BlockBase)

	// Per-block rational scaling weights u[b][i] >= 1 (positive integer ratios)
	u := make([][]*big.Rat, numBlocks)
	for b := 0; b < numBlocks; b++ {
		sz := len(f.space.BlockBase[b])
		u[b] = make([]*big.Rat, sz)
		for i := 0; i < sz; i++ {
			if !useDiagScaling {
				u[b][i] = big.NewRat(1, 1)
				continue
			}
			diagVal := f.space.BlockBase[b][i][i]
			w := int64(1)
			if diagVal.Sign() > 0 {
				num := new(big.Int).Quo(diagVal.Num(), diagVal.Denom())
				if num.Sign() > 0 && num.IsInt64() {
					w = intSqrtApprox(num.Int64())
					if w < 1 {
						w = 1
					}
				}
			}
			u[b][i] = big.NewRat(w, 1)
		}
	}

	// Variable layout:
	//   [0 .. 2*d - 1]: lp_k, lm_k (lambda_k = lp_k - lm_k)
	//   [2*d]:          margin t (>= 0, <= 1)
	//   [2*d+1 .. ]:    s_{b, i, j} for b in blocks, 0 <= i < j < m_b
	type sPair struct{ b, i, j, varIdx int }
	var sPairs []sPair
	sLookup := make(map[int]map[[2]int]int, numBlocks)

	nextVar := 2*d + 1
	for b := 0; b < numBlocks; b++ {
		sz := len(f.space.BlockBase[b])
		sLookup[b] = make(map[[2]int]int)
		for i := 0; i < sz; i++ {
			for j := i + 1; j < sz; j++ {
				sLookup[b][[2]int{i, j}] = nextVar
				sPairs = append(sPairs, sPair{b: b, i: i, j: j, varIdx: nextVar})
				nextVar++
			}
		}
	}

	totalVars := nextVar
	varNames := make([]string, totalVars)
	obj := make([]*big.Rat, totalVars)
	for k := 0; k < d; k++ {
		varNames[2*k] = "lp_" + strconv.Itoa(k)   // txthc-ignore
		varNames[2*k+1] = "lm_" + strconv.Itoa(k) // txthc-ignore
		obj[2*k] = big.NewRat(0, 1)
		obj[2*k+1] = big.NewRat(0, 1)
	}
	tIdx := 2 * d
	varNames[tIdx] = "t_margin" // txthc-ignore
	obj[tIdx] = big.NewRat(1, 1)
	for _, sp := range sPairs {
		varNames[sp.varIdx] = "s_" + strconv.Itoa(sp.varIdx) // txthc-ignore
		obj[sp.varIdx] = big.NewRat(0, 1)
	}

	var constraints []lp.Constraint
	// Upper bound t <= 1
	tBound := makeZeroRatVec(totalVars)
	tBound[tIdx] = big.NewRat(1, 1)
	constraints = append(constraints, lp.Constraint{
		Name:   "t_cap", // txthc-ignore
		Coeffs: tBound,
		Sense:  lp.RelLe,
		RHS:    big.NewRat(1, 1),
	})

	// Helper to compute linear form Q^{(b)}_{ij}(lambda) = c0 + sum coeffs[k]*lambda_k
	entryLinearForm := func(b, i, j int) ([]*big.Rat, *big.Rat) {
		c0 := new(big.Rat).Set(f.space.BlockBase[b][i][j])
		coeffs := makeZeroRatVec(d)
		for k, dir := range f.space.Directions {
			if dir.PivotBlock == b && ((dir.PivotLocalI == i && dir.PivotLocalJ == j) || (dir.PivotLocalI == j && dir.PivotLocalJ == i)) {
				coeffs[k].Add(coeffs[k], dir.PivotCoeff)
			}
			if dir.FreeBlock == b && ((dir.FreeLocalI == i && dir.FreeLocalJ == j) || (dir.FreeLocalI == j && dir.FreeLocalJ == i)) {
				coeffs[k].Add(coeffs[k], big.NewRat(1, 1))
			}
		}
		return coeffs, c0
	}

	// Off-diagonal abs bounds: Q_{ij}(lambda) - s <= 0 and -Q_{ij}(lambda) - s <= 0
	for _, sp := range sPairs {
		coeffs, c0 := entryLinearForm(sp.b, sp.i, sp.j)

		rowPos := makeZeroRatVec(totalVars)
		rowNeg := makeZeroRatVec(totalVars)
		for k := 0; k < d; k++ {
			if coeffs[k].Sign() == 0 {
				continue
			}
			rowPos[2*k] = new(big.Rat).Set(coeffs[k])
			rowPos[2*k+1] = new(big.Rat).Neg(coeffs[k])
			rowNeg[2*k] = new(big.Rat).Neg(coeffs[k])
			rowNeg[2*k+1] = new(big.Rat).Set(coeffs[k])
		}
		rowPos[sp.varIdx] = big.NewRat(-1, 1)
		rowNeg[sp.varIdx] = big.NewRat(-1, 1)

		constraints = append(constraints, lp.Constraint{
			Name:   "abs_pos_" + strconv.Itoa(sp.varIdx), // txthc-ignore
			Coeffs: rowPos,
			Sense:  lp.RelLe,
			RHS:    new(big.Rat).Neg(c0),
		})
		constraints = append(constraints, lp.Constraint{
			Name:   "abs_neg_" + strconv.Itoa(sp.varIdx), // txthc-ignore
			Coeffs: rowNeg,
			Sense:  lp.RelLe,
			RHS:    new(big.Rat).Set(c0),
		})
	}

	// Diagonal dominance per row: u_i^2 * Q_{ii}(lambda) - sum_{j != i} u_i*u_j*s_{ij} - t >= 0
	for b := 0; b < numBlocks; b++ {
		sz := len(f.space.BlockBase[b])
		for i := 0; i < sz; i++ {
			coeffs, c0 := entryLinearForm(b, i, i)
			ui2 := new(big.Rat).Mul(u[b][i], u[b][i])

			row := makeZeroRatVec(totalVars)
			for k := 0; k < d; k++ {
				if coeffs[k].Sign() == 0 {
					continue
				}
				scaled := new(big.Rat).Mul(coeffs[k], ui2)
				row[2*k] = scaled
				row[2*k+1] = new(big.Rat).Neg(scaled)
			}
			for j := 0; j < sz; j++ {
				if j == i {
					continue
				}
				a, cIdx := i, j
				if a > cIdx {
					a, cIdx = cIdx, a
				}
				sVar := sLookup[b][[2]int{a, cIdx}]
				uij := new(big.Rat).Mul(u[b][i], u[b][j])
				row[sVar] = new(big.Rat).Neg(uij)
			}
			row[tIdx] = big.NewRat(-1, 1)

			rhs := new(big.Rat).Mul(ui2, c0)
			rhs.Neg(rhs)
			constraints = append(constraints, lp.Constraint{
				Name:   "dd_" + strconv.Itoa(b) + "_" + strconv.Itoa(i), // txthc-ignore
				Coeffs: row,
				Sense:  lp.RelGe,
				RHS:    rhs,
			})
		}
	}

	prob := &lp.Problem{
		Sense:       lp.OptMaximize,
		Objective:   obj,
		ObjConstant: big.NewRat(0, 1),
		VarNames:    varNames,
		Constraints: constraints,
	}
	res, err := lp.Solve(prob)
	if err != nil {
		return nil, false, err
	}
	if res.Status != lp.StatusOptimal {
		return nil, false, nil
	}
	return extractLambdaFromLP(res.PrimalSolution, varNames, d), true, nil
}

func (f *SOSDecomposeFSM) addCut(block int, vec []*big.Rat) bool {
	norm := normalizeWitnessVector(vec)
	key := cutKey(block, norm)
	if f.cutSeen[key] {
		return false
	}
	f.cutSeen[key] = true
	f.cuts = append(f.cuts, sosBlockCut{
		Block:  block,
		Vector: norm,
		Key:    key,
	})
	return true
}

func (f *SOSDecomposeFSM) addFacialVector(block int, vec []*big.Rat) bool {
	norm := normalizeWitnessVector(vec)
	key := cutKey(block, norm)
	if f.facialSeen[key] {
		return false
	}
	f.facialSeen[key] = true
	f.facialVectors = append(f.facialVectors, sosBlockCut{
		Block:  block,
		Vector: norm,
		Key:    key,
	})
	return true
}

func (f *SOSDecomposeFSM) seedInitialCuttingPlanes() {
	numBlocks := len(f.space.BlockBase)
	for b := 0; b < numBlocks; b++ {
		sz := len(f.space.BlockBase[b])
		for i := 0; i < sz; i++ {
			e := makeZeroRatVec(sz)
			e[i] = big.NewRat(1, 1)
			f.addCut(b, e)
		}
	}
}

func (f *SOSDecomposeFSM) buildPolyhedronConstraints(totalVars int, includeMargin bool, marginVarIdx int) []lp.Constraint {
	d := f.space.Dim
	constraints := make([]lp.Constraint, 0, len(f.cuts)+len(f.facialVectors)*4+1)

	for idx, cut := range f.cuts {
		coeffs, c0 := f.space.DirectionQuadFormRow(cut.Block, cut.Vector)
		row := makeZeroRatVec(totalVars)
		for k := 0; k < d; k++ {
			if coeffs[k].Sign() == 0 {
				continue
			}
			row[2*k] = new(big.Rat).Set(coeffs[k])
			row[2*k+1] = new(big.Rat).Neg(coeffs[k])
		}
		if includeMargin {
			row[marginVarIdx] = big.NewRat(-1, 1)
		}
		constraints = append(constraints, lp.Constraint{
			Name:   "cut_" + strconv.Itoa(idx), // txthc-ignore
			Coeffs: row,
			Sense:  lp.RelGe,
			RHS:    new(big.Rat).Neg(c0),
		})
	}

	// Partial facial reduction linear subspace constraints: Q^{(b)}(lambda) v = 0 (per coordinate row r)
	for fIdx, fv := range f.facialVectors {
		sz := len(f.space.BlockBase[fv.Block])
		for r := 0; r < sz; r++ {
			// Compute row r of Q^{(b)}(lambda) * fv.Vector
			c0 := big.NewRat(0, 1)
			for c := 0; c < sz; c++ {
				if fv.Vector[c].Sign() == 0 || f.space.BlockBase[fv.Block][r][c].Sign() == 0 {
					continue
				}
				prod := new(big.Rat).Mul(f.space.BlockBase[fv.Block][r][c], fv.Vector[c])
				c0.Add(c0, prod)
			}

			coeffs := makeZeroRatVec(d)
			hasNonZero := false
			for k, dir := range f.space.Directions {
				val := big.NewRat(0, 1)
				if dir.PivotBlock == fv.Block {
					if dir.PivotLocalI == r && fv.Vector[dir.PivotLocalJ].Sign() != 0 {
						val.Add(val, new(big.Rat).Mul(dir.PivotCoeff, fv.Vector[dir.PivotLocalJ]))
					} else if dir.PivotLocalJ == r && dir.PivotLocalI != dir.PivotLocalJ && fv.Vector[dir.PivotLocalI].Sign() != 0 {
						val.Add(val, new(big.Rat).Mul(dir.PivotCoeff, fv.Vector[dir.PivotLocalI]))
					}
				}
				if dir.FreeBlock == fv.Block {
					if dir.FreeLocalI == r && fv.Vector[dir.FreeLocalJ].Sign() != 0 {
						val.Add(val, fv.Vector[dir.FreeLocalJ])
					} else if dir.FreeLocalJ == r && dir.FreeLocalI != dir.FreeLocalJ && fv.Vector[dir.FreeLocalI].Sign() != 0 {
						val.Add(val, fv.Vector[dir.FreeLocalI])
					}
				}
				if val.Sign() != 0 {
					coeffs[k] = val
					hasNonZero = true
				}
			}

			if !hasNonZero {
				continue
			}

			row := makeZeroRatVec(totalVars)
			for k := 0; k < d; k++ {
				if coeffs[k].Sign() == 0 {
					continue
				}
				row[2*k] = new(big.Rat).Set(coeffs[k])
				row[2*k+1] = new(big.Rat).Neg(coeffs[k])
			}
			constraints = append(constraints, lp.Constraint{
				Name:   "face_" + strconv.Itoa(fIdx) + "_" + strconv.Itoa(r), // txthc-ignore
				Coeffs: row,
				Sense:  lp.RelEq,
				RHS:    new(big.Rat).Neg(c0),
			})
		}
	}

	return constraints
}

func (f *SOSDecomposeFSM) stepCuttingPlaneLP() error {
	d := f.space.Dim
	if f.cutIter >= sosMaxCuttingPlaneIters {
		f.state = sosStateExhausted
		return nil
	}
	f.cutIter++

	totalVars := 2*d + 1
	tIdx := 2 * d
	varNames := make([]string, totalVars)
	obj := makeZeroRatVec(totalVars)
	for k := 0; k < d; k++ {
		varNames[2*k] = "lp_" + strconv.Itoa(k)   // txthc-ignore
		varNames[2*k+1] = "lm_" + strconv.Itoa(k) // txthc-ignore
	}
	varNames[tIdx] = "t_margin" // txthc-ignore
	obj[tIdx] = big.NewRat(1, 1)

	constraints := f.buildPolyhedronConstraints(totalVars, true, tIdx)
	tCap := makeZeroRatVec(totalVars)
	tCap[tIdx] = big.NewRat(1, 1)
	constraints = append(constraints, lp.Constraint{
		Name:   "t_cap", // txthc-ignore
		Coeffs: tCap,
		Sense:  lp.RelLe,
		RHS:    big.NewRat(1, 1),
	})

	prob := &lp.Problem{
		Sense:       lp.OptMaximize,
		Objective:   obj,
		ObjConstant: big.NewRat(0, 1),
		VarNames:    varNames,
		Constraints: constraints,
	}
	res, err := lp.Solve(prob)
	if err != nil {
		return err
	}
	if res.Status == lp.StatusInfeasible {
		f.state = sosStateExhausted
		return nil
	}
	if res.Status != lp.StatusOptimal {
		f.state = sosStateExhausted
		return nil
	}

	lam := extractLambdaFromLP(res.PrimalSolution, varNames, d)
	if results, psd := f.evaluateBlocksPSD(lam); psd {
		f.candidateLambda = lam
		f.candidateLDLTs = results
		f.state = sosStateVerify
		return nil
	}

	// On zero-margin boundary solutions, try exact rational midpoint with previous iterate
	// to move inside relative interior of active boundary cones.
	if res.OptimalValue.Sign() == 0 && f.prevZeroLambda != nil {
		mid := make([]*big.Rat, d)
		half := big.NewRat(1, 2)
		for k := 0; k < d; k++ {
			mid[k] = new(big.Rat).Add(f.prevZeroLambda[k], lam[k])
			mid[k].Mul(mid[k], half)
		}
		if results, psd := f.evaluateBlocksPSD(mid); psd {
			f.candidateLambda = mid
			f.candidateLDLTs = results
			f.state = sosStateVerify
			return nil
		}
	}

	// Collect all block refutation witnesses at lam as exact cuts.
	results, _ := f.evaluateBlocksPSD(lam)
	addedAny := false
	for b, rb := range results {
		if !rb.IsPSD && len(rb.Witness) > 0 {
			if f.addCut(b, rb.Witness) {
				addedAny = true
			}
		}
	}

	f.prevZeroLambda = lam
	if res.OptimalValue.Sign() == 0 {
		f.zeroMarginStreak++
	} else {
		f.zeroMarginStreak = 0
	}

	if f.zeroMarginStreak >= sosFacialStallThreshold && f.facialRounds < sosMaxFacialRounds {
		f.state = sosStateFacialReduction
		return nil
	}

	if !addedAny && f.facialRounds >= sosMaxFacialRounds {
		f.state = sosStateExhausted
		return nil
	}

	return nil
}

func (f *SOSDecomposeFSM) stepFacialReduction() error {
	f.facialRounds++
	d := f.space.Dim
	lam := f.prevZeroLambda
	if lam == nil {
		f.state = sosStateCuttingPlaneLP
		return nil
	}

	// Evaluate all blocks once at lam to locate cuts that are tight (w^T Q(lam) w == 0).
	blocks := make([][][]*big.Rat, len(f.space.BlockBase))
	for b := range blocks {
		blocks[b] = f.space.EvaluateBlock(b, lam)
	}

	// Prefer testing the most recently added cuts first, since those reflect boundary faces.
	addedFacial := false
	tried := 0
	totalVars := 2 * d
	varNames := make([]string, totalVars)
	for k := 0; k < d; k++ {
		varNames[2*k] = "lp_" + strconv.Itoa(k)   // txthc-ignore
		varNames[2*k+1] = "lm_" + strconv.Itoa(k) // txthc-ignore
	}
	baseConstraints := f.buildPolyhedronConstraints(totalVars, false, -1)

	for idx := len(f.cuts) - 1; idx >= 0 && tried < sosMaxFacialCandidatesTry; idx-- {
		cut := f.cuts[idx]
		if f.facialSeen[cut.Key] {
			continue
		}
		if evalSymQuadForm(blocks[cut.Block], cut.Vector).Sign() != 0 {
			continue
		}
		tried++

		forcedZero, err := f.isCutForcedZeroInPolyhedron(totalVars, varNames, baseConstraints, cut)
		if err != nil {
			return err
		}
		if forcedZero {
			if f.addFacialVector(cut.Block, cut.Vector) {
				addedFacial = true
			}
		}
	}

	if addedFacial {
		f.zeroMarginStreak = 0
	} else {
		// Prevent infinite re-entry into facial reduction when no tight cut is globally zero-forced.
		f.facialRounds = sosMaxFacialRounds
	}

	f.state = sosStateCuttingPlaneLP
	return nil
}

func (f *SOSDecomposeFSM) stepVerify() {
	var allTerms []SOSTerm
	for b, blk := range f.analysis.Blocks {
		blockMonomials := make([]Monomial, len(blk))
		for li, gi := range blk {
			blockMonomials[li] = f.analysis.Basis[gi]
		}
		termsB := extractSOSTermsFromLDLT(f.candidateLDLTs[b].L, f.candidateLDLTs[b].D, blockMonomials, f.order)
		allTerms = append(allTerms, termsB...)
	}

	// Independent full MPoly re-expansion check before certifying.
	n := len(f.vars)
	reconstructed := NewZeroMPoly(n)
	unitMon := zeroMonomial(n)
	for _, t := range allTerms {
		if t.Weight.Sign() <= 0 {
			f.state = sosStateExhausted
			return
		}
		sq := t.Poly.Mul(t.Poly, f.order)
		scaled := sq.MulTerm(Term{Coeff: t.Weight, Mon: unitMon}, f.order)
		reconstructed = reconstructed.Add(scaled, f.order)
	}
	diff := f.poly.Sub(reconstructed, f.order)
	if !diff.IsZero() {
		f.state = sosStateExhausted
		return
	}

	f.result = &SOSResult{
		Status:   SOSCertified,
		Terms:    allTerms,
		Residual: NewZeroMPoly(n),
		Vars:     f.vars,
	}
	f.state = sosStateCertified
}

// extractSOSTermsFromLDLT builds weighted squared polynomials D_j * (sum_{i>=j} L_{ij} v_i)^2.
func extractSOSTermsFromLDLT(L [][]*big.Rat, D []*big.Rat, basis []Monomial, order MonomialOrder) []SOSTerm {
	var terms []SOSTerm
	m := len(basis)
	for j := 0; j < m; j++ {
		dVal := D[j]
		if dVal.Sign() <= 0 {
			continue
		}
		var qTerms []Term
		for i := j; i < m; i++ {
			coeff := L[i][j]
			if coeff.Sign() != 0 {
				qTerms = append(qTerms, Term{Coeff: new(big.Rat).Set(coeff), Mon: basis[i]})
			}
		}
		if len(qTerms) > 0 {
			sortPolyTerms(qTerms, order)
			terms = append(terms, SOSTerm{
				Weight: new(big.Rat).Set(dVal),
				Poly:   &MPoly{Terms: qTerms, Sugar: qTerms[0].Mon.TotalDeg},
			})
		}
	}
	return terms
}

func extractLambdaFromLP(sol map[string]*big.Rat, varNames []string, d int) []*big.Rat {
	lam := make([]*big.Rat, d)
	for k := 0; k < d; k++ {
		plus := sol[varNames[2*k]]
		minus := sol[varNames[2*k+1]]
		if plus == nil {
			plus = big.NewRat(0, 1)
		}
		if minus == nil {
			minus = big.NewRat(0, 1)
		}
		lam[k] = new(big.Rat).Sub(plus, minus)
	}
	return lam
}

func cutKey(block int, w []*big.Rat) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(block))
	b.WriteByte(':')
	for i, x := range w {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(x.RatString())
	}
	return b.String()
}

func makeZeroRatVec(n int) []*big.Rat {
	v := make([]*big.Rat, n)
	for i := range v {
		v[i] = big.NewRat(0, 1)
	}
	return v
}

func intSqrtApprox(x int64) int64 {
	if x <= 1 {
		return 1
	}
	lo, hi := int64(1), x
	if hi > 1<<30 {
		hi = 1 << 30
	}
	ans := int64(1)
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if mid <= x/mid {
			ans = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return ans
}
