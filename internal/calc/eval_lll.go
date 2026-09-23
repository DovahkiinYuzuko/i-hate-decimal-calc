package calc

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// LLLState represents discrete states of the LLL lattice reduction lifecycle.
type LLLState int

const (
	LLLStateInit LLLState = iota
	LLLStateGramSchmidtComputed
	LLLStateSizeReduced
	LLLStateLovaszTested
	LLLStateVectorSwapped
	LLLStateBasisReduced
	LLLStateRelationIdentified
	LLLStateFailed
)

// String returns the human-readable name of the LLLState.
func (s LLLState) String() string {
	switch s {
	case LLLStateInit:
		return "Init"
	case LLLStateGramSchmidtComputed:
		return "GramSchmidtComputed"
	case LLLStateSizeReduced:
		return "SizeReduced"
	case LLLStateLovaszTested:
		return "LovaszTested"
	case LLLStateVectorSwapped:
		return "VectorSwapped"
	case LLLStateBasisReduced:
		return "BasisReduced"
	case LLLStateRelationIdentified:
		return "RelationIdentified"
	case LLLStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("UnknownState(%d)", int(s))
	}
}

// LLLLifecycleFSM manages and audits state transitions of the LLL lattice reduction algorithm.
type LLLLifecycleFSM struct {
	mu           sync.RWMutex
	currentState LLLState
	history      []LLLState
}

// NewLLLLifecycleFSM instantiates a new FSM initialized to LLLStateInit.
func NewLLLLifecycleFSM() *LLLLifecycleFSM {
	return &LLLLifecycleFSM{
		currentState: LLLStateInit,
		history:      []LLLState{LLLStateInit},
	}
}

// CurrentState returns the current state in a thread-safe manner.
func (f *LLLLifecycleFSM) CurrentState() LLLState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.currentState
}

// History returns a copy of the transition history.
func (f *LLLLifecycleFSM) History() []LLLState {
	f.mu.RLock()
	defer f.mu.RUnlock()
	res := make([]LLLState, len(f.history))
	copy(res, f.history)
	return res
}

// TransitionTo validates and performs a state transition.
func (f *LLLLifecycleFSM) TransitionTo(target LLLState) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	valid := false
	switch f.currentState {
	case LLLStateInit:
		valid = (target == LLLStateGramSchmidtComputed || target == LLLStateFailed)
	case LLLStateGramSchmidtComputed:
		valid = (target == LLLStateSizeReduced || target == LLLStateBasisReduced || target == LLLStateFailed)
	case LLLStateSizeReduced:
		valid = (target == LLLStateLovaszTested || target == LLLStateBasisReduced || target == LLLStateFailed)
	case LLLStateLovaszTested:
		valid = (target == LLLStateVectorSwapped || target == LLLStateSizeReduced || target == LLLStateBasisReduced || target == LLLStateFailed)
	case LLLStateVectorSwapped:
		valid = (target == LLLStateGramSchmidtComputed || target == LLLStateSizeReduced || target == LLLStateFailed)
	case LLLStateBasisReduced:
		valid = (target == LLLStateRelationIdentified || target == LLLStateFailed)
	case LLLStateRelationIdentified:
		valid = (target == LLLStateFailed)
	case LLLStateFailed:
		valid = false
	}

	if !valid {
		return fmt.Errorf("%s", i18n.T("lll.err_invalid_lll_fsm_transition", f.currentState.String(), target.String()))
	}

	f.currentState = target
	f.history = append(f.history, target)
	return nil
}

// LLLResult contains the output of LLL lattice basis reduction.
type LLLResult struct {
	FSM                *LLLLifecycleFSM
	ReducedBasis       [][]*big.Int
	GramSchmidtCoeffs  [][]*big.Rat
	OrthogonalNormSq   []*big.Rat
}

// ExactLLLReduction performs Lenstra-Lenstra-Lovász lattice basis reduction
// on integer vectors over pure big.Rat and big.Int arithmetic with zero floating-point approximation.
func ExactLLLReduction(basis [][]*big.Int, delta *big.Rat) (*LLLResult, error) {
	fsm := NewLLLLifecycleFSM()

	n := len(basis)
	if n == 0 {
		_ = fsm.TransitionTo(LLLStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("lll.err_empty_basis"))
	}
	m := len(basis[0])
	if m == 0 {
		_ = fsm.TransitionTo(LLLStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("lll.err_empty_basis"))
	}

	for i := 1; i < n; i++ {
		if len(basis[i]) != m {
			_ = fsm.TransitionTo(LLLStateFailed)
			return nil, fmt.Errorf("%s", i18n.T("lll.err_inconsistent_dimension", i, len(basis[i]), m))
		}
	}

	// Default delta = 3/4
	if delta == nil {
		delta = big.NewRat(3, 4)
	} else {
		quarter := big.NewRat(1, 4)
		one := big.NewRat(1, 1)
		if delta.Cmp(quarter) <= 0 || delta.Cmp(one) >= 0 {
			_ = fsm.TransitionTo(LLLStateFailed)
			return nil, fmt.Errorf("%s", i18n.T("lll.err_invalid_delta", delta.RatString()))
		}
	}

	// Clone input basis
	b := cloneBigIntMatrix(basis)

	// Single vector base case
	if n == 1 {
		b0Norm := vectorDotBigInt(b[0], b[0])
		if b0Norm.Sign() == 0 {
			_ = fsm.TransitionTo(LLLStateFailed)
			return nil, fmt.Errorf("%s", i18n.T("lll.err_linearly_dependent_basis"))
		}
		_ = fsm.TransitionTo(LLLStateGramSchmidtComputed)
		_ = fsm.TransitionTo(LLLStateBasisReduced)
		return &LLLResult{
			FSM:               fsm,
			ReducedBasis:      b,
			GramSchmidtCoeffs: [][]*big.Rat{{big.NewRat(1, 1)}},
			OrthogonalNormSq:  []*big.Rat{new(big.Rat).SetInt(b0Norm)},
		}, nil
	}

	// Compute initial Gram-Schmidt orthogonalization
	bStar, mu, normSq, err := gramSchmidtExact(b)
	if err != nil {
		_ = fsm.TransitionTo(LLLStateFailed)
		return nil, err
	}
	_ = fsm.TransitionTo(LLLStateGramSchmidtComputed)

	k := 1
	for k < n {
		// Size reduction on k against k-1
		sizeReduce(b, mu, k, k-1)
		_ = fsm.TransitionTo(LLLStateSizeReduced)

		// Lovász condition:
		// normSq[k] >= (delta - mu[k][k-1]^2) * normSq[k-1]
		muSq := new(big.Rat).Mul(mu[k][k-1], mu[k][k-1])
		lovaszRHSFactor := new(big.Rat).Sub(delta, muSq)
		lovaszRHS := new(big.Rat).Mul(lovaszRHSFactor, normSq[k-1])

		_ = fsm.TransitionTo(LLLStateLovaszTested)

		if normSq[k].Cmp(lovaszRHS) >= 0 {
			// Lovász condition holds: size-reduce against remaining previous vectors j = k-2 downto 0
			for j := k - 2; j >= 0; j-- {
				sizeReduce(b, mu, k, j)
			}
			k++
		} else {
			// Lovász condition fails: swap b[k] and b[k-1]
			b[k], b[k-1] = b[k-1], b[k]
			_ = fsm.TransitionTo(LLLStateVectorSwapped)

			// Recompute Gram-Schmidt for precision and robust exactness
			bStar, mu, normSq, err = gramSchmidtExact(b)
			if err != nil {
				_ = fsm.TransitionTo(LLLStateFailed)
				return nil, err
			}
			_ = fsm.TransitionTo(LLLStateGramSchmidtComputed)

			if k > 1 {
				k--
			}
		}
	}

	_ = fsm.TransitionTo(LLLStateBasisReduced)
	_ = bStar // retain for debugging if needed

	return &LLLResult{
		FSM:               fsm,
		ReducedBasis:      b,
		GramSchmidtCoeffs: mu,
		OrthogonalNormSq:  normSq,
	}, nil
}

// gramSchmidtExact performs arbitrary-precision exact Gram-Schmidt orthogonalization
// on the given integer basis using *big.Rat vectors.
func gramSchmidtExact(b [][]*big.Int) (bStar [][]*big.Rat, mu [][]*big.Rat, normSq []*big.Rat, err error) {
	n := len(b)
	m := len(b[0])

	bStar = make([][]*big.Rat, n)
	mu = make([][]*big.Rat, n)
	normSq = make([]*big.Rat, n)

	for i := 0; i < n; i++ {
		mu[i] = make([]*big.Rat, n)
		for j := 0; j < n; j++ {
			mu[i][j] = big.NewRat(0, 1)
		}
		mu[i][i] = big.NewRat(1, 1)

		// Convert b[i] to rational vector
		curStar := make([]*big.Rat, m)
		for col := 0; col < m; col++ {
			curStar[col] = new(big.Rat).SetInt(b[i][col])
		}

		for j := 0; j < i; j++ {
			// mu[i][j] = <b[i], bStar[j]> / ||bStar[j]||^2
			dot := big.NewRat(0, 1)
			for col := 0; col < m; col++ {
				term := new(big.Rat).Mul(new(big.Rat).SetInt(b[i][col]), bStar[j][col])
				dot.Add(dot, term)
			}
			mu[i][j] = new(big.Rat).Quo(dot, normSq[j])

			// curStar = curStar - mu[i][j] * bStar[j]
			for col := 0; col < m; col++ {
				sub := new(big.Rat).Mul(mu[i][j], bStar[j][col])
				curStar[col].Sub(curStar[col], sub)
			}
		}

		// normSq[i] = <curStar, curStar>
		curNormSq := big.NewRat(0, 1)
		for col := 0; col < m; col++ {
			term := new(big.Rat).Mul(curStar[col], curStar[col])
			curNormSq.Add(curNormSq, term)
		}

		if curNormSq.Sign() == 0 {
			return nil, nil, nil, fmt.Errorf("%s", i18n.T("lll.err_linearly_dependent_basis"))
		}

		bStar[i] = curStar
		normSq[i] = curNormSq
	}

	return bStar, mu, normSq, nil
}

// sizeReduce performs size reduction of vector k with respect to vector j (j < k).
func sizeReduce(b [][]*big.Int, mu [][]*big.Rat, k, j int) {
	q := nearestInteger(mu[k][j])
	if q.Sign() == 0 {
		return
	}

	m := len(b[k])
	qRat := new(big.Rat).SetInt(q)

	// b[k] = b[k] - q * b[j]
	for col := 0; col < m; col++ {
		prod := new(big.Int).Mul(q, b[j][col])
		b[k][col].Sub(b[k][col], prod)
	}

	// Update mu[k][l] for l <= j
	for l := 0; l < j; l++ {
		sub := new(big.Rat).Mul(qRat, mu[j][l])
		mu[k][l].Sub(mu[k][l], sub)
	}
	mu[k][j].Sub(mu[k][j], qRat)
}

// nearestInteger computes floor(r + 1/2) for exact round-to-nearest integer.
func nearestInteger(r *big.Rat) *big.Int {
	half := big.NewRat(1, 2)
	shifted := new(big.Rat).Add(r, half)
	num := shifted.Num()
	denom := shifted.Denom()

	res := new(big.Int).Quo(num, denom)
	rem := new(big.Int).Rem(num, denom)

	// If remainder is negative (due to Go's truncated division towards zero), adjust downward
	if rem.Sign() < 0 {
		res.Sub(res, big.NewInt(1))
	}
	return res
}

// cloneBigIntMatrix creates a deep copy of a 2D big.Int slice.
func cloneBigIntMatrix(m [][]*big.Int) [][]*big.Int {
	res := make([][]*big.Int, len(m))
	for i := range m {
		res[i] = make([]*big.Int, len(m[i]))
		for j := range m[i] {
			res[i][j] = new(big.Int).Set(m[i][j])
		}
	}
	return res
}

// vectorDotBigInt computes the dot product of two integer vectors.
func vectorDotBigInt(v1, v2 []*big.Int) *big.Int {
	res := big.NewInt(0)
	for i := 0; i < len(v1); i++ {
		term := new(big.Int).Mul(v1[i], v2[i])
		res.Add(res, term)
	}
	return res
}

// vectorDotBigRat computes the dot product of two rational vectors.
func vectorDotBigRat(v1, v2 []*big.Rat) *big.Rat {
	res := big.NewRat(0, 1)
	for i := 0; i < len(v1); i++ {
		term := new(big.Rat).Mul(v1[i], v2[i])
		res.Add(res, term)
	}
	return res
}

// findIntegerRelationCandidates computes candidate integer coefficient vectors
// by applying LLL reduction to an (n) x (n+1) lattice.
func findIntegerRelationCandidates(values []*big.Rat, scaleBits int) ([][]*big.Int, *LLLResult, error) {
	n := len(values)
	if n < 2 {
		return nil, nil, fmt.Errorf("%s", i18n.T("lll.err_empty_basis"))
	}
	if scaleBits <= 0 {
		scaleBits = 96
	}

	// Large scale M = 2^scaleBits
	mInt := new(big.Int).Lsh(big.NewInt(1), uint(scaleBits))
	mRat := new(big.Rat).SetInt(mInt)

	// Construct lattice basis of size n x (n + 1):
	// Row i: [0, ..., 0, 1, 0, ..., 0, round(M * values[i])]
	basis := make([][]*big.Int, n)
	for i := 0; i < n; i++ {
		basis[i] = make([]*big.Int, n+1)
		for j := 0; j < n+1; j++ {
			basis[i][j] = big.NewInt(0)
		}
		basis[i][i] = big.NewInt(1)

		// scaledVal = round(M * values[i])
		scaled := new(big.Rat).Mul(values[i], mRat)
		basis[i][n] = nearestInteger(scaled)
	}

	res, err := ExactLLLReduction(basis, big.NewRat(3, 4))
	if err != nil {
		return nil, nil, err
	}

	var candidates [][]*big.Int
	for _, row := range res.ReducedBasis {
		isAllZero := true
		for j := 0; j < n; j++ {
			if row[j].Sign() != 0 {
				isAllZero = false
				break
			}
		}
		if isAllZero {
			continue
		}

		c := make([]*big.Int, n)
		for j := 0; j < n; j++ {
			c[j] = new(big.Int).Set(row[j])
		}
		candidates = append(candidates, c)
	}

	return candidates, res, nil
}

// FindIntegerRelation searches for a non-trivial integer vector c = (c_1, ..., c_n)
// such that sum_{i=1}^n c_i * values[i] = 0 using LLL lattice reduction.
func FindIntegerRelation(values []*big.Rat, scaleBits int) ([]*big.Int, error) {
	n := len(values)
	candidates, res, err := findIntegerRelationCandidates(values, scaleBits)
	if err != nil {
		return nil, err
	}

	// First pass: look for exact zero relation
	for _, c := range candidates {
		dot := big.NewRat(0, 1)
		for j := 0; j < n; j++ {
			term := new(big.Rat).Mul(new(big.Rat).SetInt(c[j]), values[j])
			dot.Add(dot, term)
		}

		if dot.Sign() == 0 {
			_ = res.FSM.TransitionTo(LLLStateRelationIdentified)
			return c, nil
		}
	}

	return nil, fmt.Errorf("%s", i18n.T("lll.err_relation_not_found"))
}

// FindMinimalPolynomial reconstructs the monic integer minimal polynomial P(x)
// such that P(val) == 0 up to degree maxDegree.
func FindMinimalPolynomial(val Node, maxDegree int, env *Env) (Node, error) {
	if maxDegree <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("lll.err_find_min_poly_degree", maxDegree))
	}

	// Try degrees from 1 up to maxDegree
	for deg := 1; deg <= maxDegree; deg++ {
		// Evaluate powers of val with high precision rational intervals
		// steps = 32 gives >100 bits of precision
		steps := 32
		interval, err := EvalNodeInterval(val, steps)
		if err != nil {
			return nil, err
		}

		// Center approximation in Q: (Low + High) / 2
		alpha := new(big.Rat).Add(interval.Low, interval.High)
		alpha.Mul(alpha, big.NewRat(1, 2))

		// Form vector [1, alpha, alpha^2, ..., alpha^deg]
		powers := make([]*big.Rat, deg+1)
		powers[0] = big.NewRat(1, 1)
		for p := 1; p <= deg; p++ {
			powers[p] = new(big.Rat).Mul(powers[p-1], alpha)
		}

		// Search for integer relation candidates
		// scaleBits is scaled appropriately to stay well within interval precision (<= 128 bits)
		scaleBits := 40 + deg*4
		candidates, res, err := findIntegerRelationCandidates(powers, scaleBits)
		if err != nil {
			continue
		}

		for _, coeffs := range candidates {
			// Leading coefficient must not be 0
			if coeffs[deg].Sign() == 0 {
				continue
			}

			// Ensure positive leading coefficient
			if coeffs[deg].Sign() < 0 {
				for j := range coeffs {
					coeffs[j].Neg(coeffs[j])
				}
			}

			// Remove common integer gcd
			g := new(big.Int).Set(coeffs[deg])
			for _, c := range coeffs {
				if c.Sign() != 0 {
					g.GCD(nil, nil, g, c)
				}
			}
			if g.Sign() > 0 && g.Cmp(big.NewInt(1)) > 0 {
				for j := range coeffs {
					coeffs[j].Quo(coeffs[j], g)
				}
			}

			// Construct polynomial node: sum_{i=0}^deg coeffs[i] * x^i
			xVar := &VarNode{Name: "x"}
			var terms []Node
			for p := deg; p >= 0; p-- {
				c := coeffs[p]
				if c.Sign() == 0 {
					continue
				}

				cNode := &RationalNode{Val: new(big.Rat).SetInt(c)}
				if p == 0 {
					terms = append(terms, cNode)
				} else if p == 1 {
					if c.Cmp(big.NewInt(1)) == 0 {
						terms = append(terms, xVar)
					} else if c.Cmp(big.NewInt(-1)) == 0 {
						terms = append(terms, &UnaryOpNode{Op: "-", Expr: xVar})
					} else {
						terms = append(terms, &MulNode{Factors: []Node{cNode, xVar}})
					}
				} else {
					powNode := &PowNode{Base: xVar, Exp: &RationalNode{Val: big.NewRat(int64(p), 1)}}
					if c.Cmp(big.NewInt(1)) == 0 {
						terms = append(terms, powNode)
					} else if c.Cmp(big.NewInt(-1)) == 0 {
						terms = append(terms, &UnaryOpNode{Op: "-", Expr: powNode})
					} else {
						terms = append(terms, &MulNode{Factors: []Node{cNode, powNode}})
					}
				}
			}

			var polyNode Node
			if len(terms) == 0 {
				polyNode = &RationalNode{Val: big.NewRat(0, 1)}
			} else if len(terms) == 1 {
				polyNode = terms[0]
			} else {
				polyNode = &AddNode{Terms: terms}
			}

			// Rigorously verify that P(val) == 0 algebraically
			subExpr := ast.Substitute(polyNode, "x", val)
			simped, err := Eval(subExpr)
			if err == nil {
				if r, ok := simped.(*RationalNode); ok && r.Val.Sign() == 0 {
					_ = res.FSM.TransitionTo(LLLStateRelationIdentified)
					return polyNode, nil
				}
			}

			// If algebraic simplification was unable to reduce directly to 0 (e.g. nested radicals),
			// verify via high-precision rational interval enclosure
			subInterval, err := EvalNodeInterval(subExpr, steps+20)
			if err == nil && subInterval.ContainsZero() {
				// Sub-interval contains zero and width is extremely small (< 10^-15)
				width := new(big.Rat).Sub(subInterval.High, subInterval.Low)
				bound := big.NewRat(1, 1000000000000000)
				if width.Cmp(bound) < 0 {
					_ = res.FSM.TransitionTo(LLLStateRelationIdentified)
					return polyNode, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("%s", i18n.T("lll.err_relation_not_found"))
}

// HandleLLL evaluates the built-in function `lll(matrix, [delta])`.
func HandleLLL(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s", i18n.T("calc.matrix_dimension_error"))
	}

	matNode, ok := args[0].(*MatrixNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("lll.err_matrix_expected"))
	}

	var delta *big.Rat
	if len(args) == 2 {
		dRat, ok := args[1].(*RationalNode)
		if !ok {
			return nil, fmt.Errorf("%s", i18n.T("lll.err_invalid_delta", args[1].String()))
		}
		delta = dRat.Val
	}

	n := matNode.Rows
	m := matNode.Cols
	basis := make([][]*big.Int, n)
	for i := 0; i < n; i++ {
		basis[i] = make([]*big.Int, m)
		for j := 0; j < m; j++ {
			elemRat, ok := matNode.Data[i][j].(*RationalNode)
			if !ok || !elemRat.Val.IsInt() {
				return nil, fmt.Errorf("%s", i18n.T("lll.err_matrix_expected"))
			}
			basis[i][j] = new(big.Int).Set(elemRat.Val.Num())
		}
	}

	res, err := ExactLLLReduction(basis, delta)
	if err != nil {
		return nil, err
	}

	outData := make([][]Node, n)
	for i := 0; i < n; i++ {
		outData[i] = make([]Node, m)
		for j := 0; j < m; j++ {
			outData[i][j] = &RationalNode{Val: new(big.Rat).SetInt(res.ReducedBasis[i][j])}
		}
	}

	return &MatrixNode{
		Rows: n,
		Cols: m,
		Data: outData,
	}, nil
}

// HandleFindMinPoly evaluates the built-in function `find_min_poly(expr, maxDegree)`.
func HandleFindMinPoly(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("lll.err_find_min_poly_args"))
	}

	degRat, ok := args[1].(*RationalNode)
	if !ok || !degRat.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("lll.err_find_min_poly_degree", 0))
	}
	maxDeg := int(degRat.Val.Num().Int64())

	return FindMinimalPolynomial(args[0], maxDeg, env)
}
