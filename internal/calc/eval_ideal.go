package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Polynomial Embedding & Variable Elimination Helpers
// -------------------------------------------------------------------------

// embedPolyWithPrependVar shifts all variable indices by +1 and sets the new leading variable exponent to 0.
// New ring has numVars + 1 variables: [t, x_1, ..., x_n].
func embedPolyWithPrependVar(p *MPoly, numVars int) *MPoly {
	if p.IsZero() {
		return NewZeroMPoly(numVars + 1)
	}
	newTerms := make([]Term, len(p.Terms))
	for i, t := range p.Terms {
		newExps := make([]int, numVars+1)
		newExps[0] = 0 // exponent of t
		copy(newExps[1:], t.Mon.Exps)
		newTerms[i] = Term{
			Coeff: new(big.Rat).Set(t.Coeff),
			Mon:   newMonomial(newExps),
		}
	}
	return &MPoly{Terms: newTerms, Sugar: p.Sugar}
}

// polyT returns the polynomial "t" in the ring Q[t, x_1, ..., x_n].
func polyT(numVars int) *MPoly {
	exps := make([]int, numVars+1)
	exps[0] = 1
	return &MPoly{
		Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial(exps)}},
		Sugar: 1,
	}
}

// polyOneMinusT returns the polynomial "1 - t" in Q[t, x_1, ..., x_n].
func polyOneMinusT(numVars int) *MPoly {
	// 1
	constExps := make([]int, numVars+1)
	tExps := make([]int, numVars+1)
	tExps[0] = 1
	return &MPoly{
		Terms: []Term{
			{Coeff: big.NewRat(-1, 1), Mon: newMonomial(tExps)},
			{Coeff: big.NewRat(1, 1), Mon: newMonomial(constExps)},
		},
		Sugar: 1,
	}
}

// makeConstMPoly returns a constant polynomial c in Q[x_1, ..., x_n].
func makeConstMPoly(val int64, numVars int) *MPoly {
	if val == 0 {
		return NewZeroMPoly(numVars)
	}
	return &MPoly{
		Terms: []Term{{Coeff: big.NewRat(val, 1), Mon: zeroMonomial(numVars)}},
		Sugar: 0,
	}
}

// makeVarPowMPoly returns x_i^deg in Q[x_1, ..., x_n].
func makeVarPowMPoly(varIdx, deg, numVars int) *MPoly {
	exps := make([]int, numVars)
	exps[varIdx] = deg
	return &MPoly{
		Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: newMonomial(exps)}},
		Sugar: deg,
	}
}

// extractPolyEliminatingPrependVar checks if p does not contain t (exps[0] == 0 for all terms),
// and if so, projects it down to Q[x_1, ..., x_n].
func extractPolyEliminatingPrependVar(p *MPoly, numVars int) (*MPoly, bool) {
	if p.IsZero() {
		return NewZeroMPoly(numVars), true
	}
	newTerms := make([]Term, len(p.Terms))
	for i, t := range p.Terms {
		if t.Mon.Exps[0] > 0 {
			return nil, false
		}
		origExps := make([]int, numVars)
		copy(origExps, t.Mon.Exps[1:])
		newTerms[i] = Term{
			Coeff: new(big.Rat).Set(t.Coeff),
			Mon:   newMonomial(origExps),
		}
	}
	return &MPoly{Terms: newTerms, Sugar: p.Sugar}, true
}

func polyNumVars(p *MPoly) int {
	if p != nil && len(p.Terms) > 0 {
		return len(p.Terms[0].Mon.Exps)
	}
	return 0
}

func idealNumVars(I []*MPoly) int {
	for _, p := range I {
		if n := polyNumVars(p); n > 0 {
			return n
		}
	}
	return 1
}

// -------------------------------------------------------------------------
// Core Ideal Operations
// -------------------------------------------------------------------------

// IdealContains returns true if J is a subset of I (i.e. every generator of J reduces to 0 mod GB(I)).
func IdealContains(I, J []*MPoly, order MonomialOrder) (bool, error) {
	if len(J) == 0 {
		return true, nil
	}
	if len(I) == 0 {
		for _, g := range J {
			if !g.IsZero() {
				return false, nil
			}
		}
		return true, nil
	}

	gbI, err := buchberger(I, order)
	if err != nil {
		return false, err
	}
	redGBI := reduceGroebnerBasis(gbI, order)

	for _, g := range J {
		if g.IsZero() {
			continue
		}
		rem := polyReduce(g, redGBI, order)
		if !rem.IsZero() {
			return false, nil
		}
	}
	return true, nil
}

// IdealEquals returns true if I == J (mutual containment).
func IdealEquals(I, J []*MPoly, order MonomialOrder) (bool, error) {
	c1, err := IdealContains(I, J, order)
	if err != nil || !c1 {
		return false, err
	}
	c2, err := IdealContains(J, I, order)
	if err != nil || !c2 {
		return false, err
	}
	return true, nil
}

// IdealIntersection computes the intersection I cap J using the auxiliary variable t:
// <t*I, (1-t)*J> cap Q[x_1, ..., x_n] with elimination order (Lex where t > x_i).
func IdealIntersection(I, J []*MPoly, order MonomialOrder) ([]*MPoly, error) {
	numVars := idealNumVars(I)
	if n := idealNumVars(J); n > numVars {
		numVars = n
	}
	if len(I) == 0 || len(J) == 0 {
		return []*MPoly{NewZeroMPoly(numVars)}, nil
	}

	tPoly := polyT(numVars)
	oneMinusTPoly := polyOneMinusT(numVars)

	// Build generators in Q[t, x_1, ..., x_n]
	var extGenerators []*MPoly
	for _, fi := range I {
		if fi.IsZero() {
			continue
		}
		emb := embedPolyWithPrependVar(fi, numVars)
		extGenerators = append(extGenerators, emb.Mul(tPoly, OrderLex))
	}
	for _, gj := range J {
		if gj.IsZero() {
			continue
		}
		emb := embedPolyWithPrependVar(gj, numVars)
		extGenerators = append(extGenerators, emb.Mul(oneMinusTPoly, OrderLex))
	}

	if len(extGenerators) == 0 {
		return []*MPoly{NewZeroMPoly(numVars)}, nil
	}

	// Compute Groebner basis under Lex order (eliminating t)
	extGB, err := buchberger(extGenerators, OrderLex)
	if err != nil {
		return nil, err
	}
	extRedGB := reduceGroebnerBasis(extGB, OrderLex)
	var res []*MPoly
	for _, p := range extRedGB {
		proj, ok := extractPolyEliminatingPrependVar(p, numVars)
		if ok && !proj.IsZero() {
			res = append(res, proj)
		}
	}

	if len(res) == 0 {
		return []*MPoly{NewZeroMPoly(numVars)}, nil
	}

	// Clean up and return in requested order
	resGB, err := buchberger(res, order)
	if err != nil {
		return nil, err
	}
	return reduceGroebnerBasis(resGB, order), nil
}

// exactPolyDiv divides p by d, assuming p is exactly divisible by d.
// Returns q such that p = d * q.
func exactPolyDiv(p, d *MPoly, order MonomialOrder) (*MPoly, error) {
	if d.IsZero() {
		return nil, fmt.Errorf("%s", i18n.T("poly.err_division_by_zero"))
	}
	numVars := polyNumVars(p)
	if numVars == 0 {
		numVars = polyNumVars(d)
	}
	if p.IsZero() {
		return NewZeroMPoly(numVars), nil
	}

	rem := p.Clone()
	var quotTerms []Term
	ltD := d.LeadingTerm()

	maxSteps := 50000
	steps := 0
	for !rem.IsZero() {
		steps++
		if steps > maxSteps {
			return nil, fmt.Errorf("exact division step limit exceeded")
		}
		ltRem := rem.LeadingTerm()
		if !ltD.Mon.Divides(ltRem.Mon) {
			return nil, fmt.Errorf("polynomial is not divisible")
		}

		coeff := new(big.Rat).Quo(ltRem.Coeff, ltD.Coeff)
		mon := ltRem.Mon.Div(ltD.Mon)
		qTerm := Term{Coeff: coeff, Mon: mon}
		quotTerms = append(quotTerms, qTerm)

		subPoly := d.MulTerm(qTerm, order)
		rem = rem.Sub(subPoly, order)
	}

	return &MPoly{Terms: quotTerms, Sugar: p.Sugar}, nil
}

// IdealColon computes the ideal quotient I : f = { g in Q[X] | f * g in I }.
// It computes I cap <f> and divides each generator by f.
func IdealColon(I []*MPoly, f *MPoly, order MonomialOrder) ([]*MPoly, error) {
	numVars := polyNumVars(f)
	if n := idealNumVars(I); n > numVars {
		numVars = n
	}
	if f.IsZero() {
		return []*MPoly{makeConstMPoly(1, numVars)}, nil
	}
	if len(I) == 0 {
		return []*MPoly{NewZeroMPoly(numVars)}, nil
	}

	// If f is a non-zero constant, I : c = I
	if f.LeadingMonomial().IsZero() {
		gb, err := buchberger(I, order)
		if err != nil {
			return nil, err
		}
		return reduceGroebnerBasis(gb, order), nil
	}

	// 1. Compute intersection J = I cap <f>
	inter, err := IdealIntersection(I, []*MPoly{f}, order)
	if err != nil {
		return nil, err
	}

	// 2. Divide each generator in inter by f
	var quotientGenerators []*MPoly
	for _, gen := range inter {
		if gen.IsZero() {
			continue
		}
		q, err := exactPolyDiv(gen, f, order)
		if err != nil {
			return nil, err
		}
		if !q.IsZero() {
			quotientGenerators = append(quotientGenerators, q)
		}
	}

	if len(quotientGenerators) == 0 {
		return []*MPoly{NewZeroMPoly(numVars)}, nil
	}

	resGB, err := buchberger(quotientGenerators, order)
	if err != nil {
		return nil, err
	}
	return reduceGroebnerBasis(resGB, order), nil
}

// IdealSaturation computes the saturation I : f^infinity and the minimal stabilization index m >= 0
// such that I : f^m == I : f^{m+1} == I : f^infinity.
func IdealSaturation(I []*MPoly, f *MPoly, order MonomialOrder) ([]*MPoly, int, error) {
	numVars := polyNumVars(f)
	if n := idealNumVars(I); n > numVars {
		numVars = n
	}
	if f.IsZero() {
		return []*MPoly{makeConstMPoly(1, numVars)}, 0, nil
	}
	if len(I) == 0 {
		return []*MPoly{NewZeroMPoly(numVars)}, 0, nil
	}

	// Chain: I_0 = I, I_{k+1} = I_k : f
	current := I
	maxIter := 50
	for k := 0; k < maxIter; k++ {
		next, err := IdealColon(current, f, order)
		if err != nil {
			return nil, 0, err
		}

		eq, err := IdealEquals(current, next, order)
		if err != nil {
			return nil, 0, err
		}
		if eq {
			// Stabilized at index k
			return current, k, nil
		}
		current = next
	}

	// Fallback to auxiliary variable method if iterative chain exceeds 50
	// <I, 1 - t*f> cap Q[x_1, ..., x_n]
	tPoly := polyT(numVars)
	oneConst := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars + 1)}}, Sugar: 0}
	embF := embedPolyWithPrependVar(f, numVars)
	tf := embF.Mul(tPoly, OrderLex)
	oneMinusTF := oneConst.Sub(tf, OrderLex)

	var extG []*MPoly
	for _, p := range I {
		if !p.IsZero() {
			extG = append(extG, embedPolyWithPrependVar(p, numVars))
		}
	}
	extG = append(extG, oneMinusTF)

	extGB, err := buchberger(extG, OrderLex)
	if err != nil {
		return nil, 0, err
	}
	extRedGB := reduceGroebnerBasis(extGB, OrderLex)

	var res []*MPoly
	for _, p := range extRedGB {
		if proj, ok := extractPolyEliminatingPrependVar(p, numVars); ok && !proj.IsZero() {
			res = append(res, proj)
		}
	}
	if len(res) == 0 {
		res = []*MPoly{NewZeroMPoly(numVars)}
	}
	finalGB, err := buchberger(res, order)
	if err != nil {
		return nil, 0, err
	}
	return reduceGroebnerBasis(finalGB, order), maxIter, nil
}

// -------------------------------------------------------------------------
// Dimension & Maximal Independent Set
// -------------------------------------------------------------------------

// IdealDimensionAndMaximalIndependentSet computes the Krull dimension dim(Q[X]/I)
// and returns a maximum-cardinality independent set of variable indices U.
// If I is the unit ideal <1>, dimension is -1 and U is empty.
func IdealDimensionAndMaximalIndependentSet(G []*MPoly, numVars int) (int, []int) {
	// Check for unit ideal <1>
	for _, g := range G {
		if !g.IsZero() && g.LeadingMonomial().IsZero() {
			return -1, nil
		}
	}

	// Extract non-zero leading monomials
	var lms []Monomial
	for _, g := range G {
		if !g.IsZero() {
			lms = append(lms, g.LeadingMonomial())
		}
	}
	if len(lms) == 0 {
		// Zero ideal <0> has dimension numVars and U = all variables
		u := make([]int, numVars)
		for i := 0; i < numVars; i++ {
			u[i] = i
		}
		return numVars, u
	}

	// Helper to check if a variable subset U is independent modulo LM(G).
	// U is independent iff for every m in lms, m has at least one variable not in U.
	isIndependent := func(uMap map[int]bool) bool {
		for _, m := range lms {
			allInU := true
			for i, exp := range m.Exps {
				if exp > 0 && !uMap[i] {
					allInU = false
					break
				}
			}
			if allInU {
				return false
			}
		}
		return true
	}

	// Search subsets from size numVars down to 0
	var findSubsets func(start, k int, current []int) []int
	findSubsets = func(start, k int, current []int) []int {
		if len(current) == k {
			uMap := make(map[int]bool, k)
			for _, idx := range current {
				uMap[idx] = true
			}
			if isIndependent(uMap) {
				res := make([]int, k)
				copy(res, current)
				return res
			}
			return nil
		}
		for i := start; i < numVars; i++ {
			current = append(current, i)
			if res := findSubsets(i+1, k, current); res != nil {
				return res
			}
			current = current[:len(current)-1]
		}
		return nil
	}

	for k := numVars; k >= 0; k-- {
		if u := findSubsets(0, k, nil); u != nil {
			return k, u
		}
	}

	return 0, nil
}

// IsZeroDimensional returns true iff dim(Q[X]/I) == 0.
func IsZeroDimensional(G []*MPoly, numVars int) bool {
	dim, _ := IdealDimensionAndMaximalIndependentSet(G, numVars)
	return dim == 0
}

// -------------------------------------------------------------------------
// Zero-Dimensional Quotient Algebra & Normal Form Linear Algebra
// -------------------------------------------------------------------------

// StandardMonomialBasis computes the standard monomials (normal form vector basis)
// of Q[X]/I from a Groebner basis G.
func StandardMonomialBasis(G []*MPoly, numVars int) []Monomial {
	var lms []Monomial
	for _, g := range G {
		if !g.IsZero() {
			lms = append(lms, g.LeadingMonomial())
		}
	}
	isDivisibleByLM := func(m Monomial) bool {
		for _, lm := range lms {
			if lm.Divides(m) {
				return true
			}
		}
		return false
	}

	startMon := zeroMonomial(numVars)
	if isDivisibleByLM(startMon) {
		return nil
	}

	var basis []Monomial
	visited := make(map[string]bool)
	queue := []Monomial{startMon}
	visited[fmt.Sprint(startMon.Exps)] = true

	maxSize := 2000 // safe guard for CAS quotient ring
	for len(queue) > 0 {
		m := queue[0]
		queue = queue[1:]

		if isDivisibleByLM(m) {
			continue
		}
		basis = append(basis, m)
		if len(basis) >= maxSize {
			break
		}

		for i := 0; i < numVars; i++ {
			nextExps := make([]int, numVars)
			copy(nextExps, m.Exps)
			nextExps[i]++
			nextMon := newMonomial(nextExps)
			key := fmt.Sprint(nextExps)
			if !visited[key] {
				visited[key] = true
				queue = append(queue, nextMon)
			}
		}
	}

	return basis
}

// normalFormVector returns the coordinate vector of polyReduce(p, redGB) over basis B.
func normalFormVector(p *MPoly, redGB []*MPoly, B []Monomial, order MonomialOrder) []*big.Rat {
	nf := polyReduce(p, redGB, order)
	vec := make([]*big.Rat, len(B))
	for i := range vec {
		vec[i] = new(big.Rat)
	}
	for _, term := range nf.Terms {
		for idx, bm := range B {
			if bm.Equal(term.Mon) {
				vec[idx].Set(term.Coeff)
				break
			}
		}
	}
	return vec
}

// solveLinearSystem solves M * c = b for c over big.Rat using Gaussian elimination.
// M is rows x cols, b has length rows. Returns c of length cols, or nil if inconsistent.
func solveLinearSystem(M [][]*big.Rat, b []*big.Rat) []*big.Rat {
	rows := len(M)
	if rows == 0 {
		return nil
	}
	cols := len(M[0])

	// Create augmented matrix [M | b]
	A := make([][]*big.Rat, rows)
	for i := 0; i < rows; i++ {
		A[i] = make([]*big.Rat, cols+1)
		for j := 0; j < cols; j++ {
			A[i][j] = new(big.Rat).Set(M[i][j])
		}
		A[i][cols] = new(big.Rat).Set(b[i])
	}

	// Forward elimination with partial pivoting
	pivotRow := 0
	pivotCols := make([]int, 0, cols)

	for col := 0; col < cols && pivotRow < rows; col++ {
		// Find non-zero pivot
		sel := -1
		for r := pivotRow; r < rows; r++ {
			if A[r][col].Sign() != 0 {
				sel = r
				break
			}
		}
		if sel == -1 {
			continue
		}

		A[pivotRow], A[sel] = A[sel], A[pivotRow]
		pivotCols = append(pivotCols, col)

		// Normalize pivot row
		pv := new(big.Rat).Set(A[pivotRow][col])
		for c := col; c <= cols; c++ {
			A[pivotRow][c].Quo(A[pivotRow][c], pv)
		}

		// Eliminate below and above
		for r := 0; r < rows; r++ {
			if r != pivotRow && A[r][col].Sign() != 0 {
				factor := new(big.Rat).Set(A[r][col])
				for c := col; c <= cols; c++ {
					term := new(big.Rat).Mul(factor, A[pivotRow][c])
					A[r][c].Sub(A[r][c], term)
				}
			}
		}

		pivotRow++
	}

	// Check consistency (0 == non-zero in rhs)
	for r := pivotRow; r < rows; r++ {
		if A[r][cols].Sign() != 0 {
			return nil
		}
	}

	// Extract solution
	res := make([]*big.Rat, cols)
	for i := range res {
		res[i] = new(big.Rat)
	}
	for r, c := range pivotCols {
		res[c] = A[r][cols]
	}

	return res
}

// MinimalPolynomialCoefficients computes the coefficients [c_0, c_1, ..., c_k] (monic, c_k = 1)
// of the minimal polynomial P(T) = c_k T^k + ... + c_0 such that P(u) = 0 in Q[X]/I.
func MinimalPolynomialCoefficients(u *MPoly, G []*MPoly, numVars int, order MonomialOrder) ([]*big.Rat, error) {
	redGB := reduceGroebnerBasis(G, order)
	B := StandardMonomialBasis(redGB, numVars)
	if len(B) == 0 {
		return []*big.Rat{big.NewRat(1, 1)}, nil
	}

	dim := len(B)
	var vectors [][]*big.Rat
	currPower := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}}, Sugar: 0}

	for k := 0; k <= dim; k++ {
		v := normalFormVector(currPower, redGB, B, order)
		vectors = append(vectors, v)

		if k > 0 {
			// Test if v_k is linearly dependent on [v_0, ..., v_{k-1}]:
			// M * c = -v_k
			M := make([][]*big.Rat, dim)
			for r := 0; r < dim; r++ {
				M[r] = make([]*big.Rat, k)
				for c := 0; c < k; c++ {
					M[r][c] = vectors[c][r]
				}
			}
			negVk := make([]*big.Rat, dim)
			for r := 0; r < dim; r++ {
				negVk[r] = new(big.Rat).Neg(v[r])
			}

			sol := solveLinearSystem(M, negVk)
			if sol != nil {
				// Found minimal polynomial: T^k + sum_{j=0}^{k-1} sol[j] * T^j
				coeffs := make([]*big.Rat, k+1)
				for j := 0; j < k; j++ {
					coeffs[j] = new(big.Rat).Set(sol[j])
				}
				coeffs[k] = big.NewRat(1, 1)
				return coeffs, nil
			}
		}

		currPower = currPower.Mul(u, order)
	}

	return nil, fmt.Errorf("failed to find minimal polynomial within dimension bound %d", dim)
}

// MinimalPolynomialOfElement computes the minimal monic polynomial P(u) such that P(u) = 0 in Q[X]/I.
func MinimalPolynomialOfElement(u *MPoly, G []*MPoly, numVars int, order MonomialOrder) (*MPoly, error) {
	coeffs, err := MinimalPolynomialCoefficients(u, G, numVars, order)
	if err != nil {
		return nil, err
	}

	k := len(coeffs) - 1
	if k == 0 {
		return &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}}}, nil
	}

	res := NewZeroMPoly(numVars)
	termPower := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}}, Sugar: 0}
	for j := 0; j <= k; j++ {
		c := coeffs[j]
		if c.Sign() != 0 {
			scaled := termPower.MulTerm(Term{Coeff: c, Mon: zeroMonomial(numVars)}, order)
			res = res.Add(scaled, order)
		}
		if j < k {
			termPower = termPower.Mul(u, order)
		}
	}
	return res, nil
}

// mpolyDerivative computes dp/dx_varIdx for an MPoly.
func mpolyDerivative(p *MPoly, varIdx int) *MPoly {
	var terms []Term
	for _, t := range p.Terms {
		deg := t.Mon.Exps[varIdx]
		if deg > 0 {
			newExps := make([]int, len(t.Mon.Exps))
			copy(newExps, t.Mon.Exps)
			newExps[varIdx]--
			coeff := new(big.Rat).Mul(t.Coeff, big.NewRat(int64(deg), 1))
			terms = append(terms, Term{Coeff: coeff, Mon: newMonomial(newExps)})
		}
	}
	return &MPoly{Terms: terms, Sugar: p.Sugar}
}

// univariateSquareFree computes square-free part p / gcd(p, p') for univariate polynomial p(x_varIdx).
func univariateSquareFree(p *MPoly, varIdx int, order MonomialOrder) (*MPoly, error) {
	dp := mpolyDerivative(p, varIdx)
	if dp.IsZero() {
		return p.Monic(), nil
	}

	// GCD via univariate Euclidean algorithm
	a := p.Monic()
	b := dp.Monic()

	for !b.IsZero() {
		rem := polyReduce(a, []*MPoly{b}, order)
		a = b
		if rem.IsZero() {
			break
		}
		b = rem.Monic()
	}

	gcdPoly := a.Monic()
	sqfree, err := exactPolyDiv(p, gcdPoly, order)
	if err != nil {
		return nil, err
	}
	return sqfree.Monic(), nil
}

// ZeroDimensionalRadical computes the radical of a zero-dimensional ideal I
// via Seidenberg's Lemma (Condition P): sqrt(I) = I + <sqfree(g_0), ..., sqfree(g_{n-1})>.
func ZeroDimensionalRadical(I []*MPoly, numVars int, order MonomialOrder) ([]*MPoly, error) {
	if len(I) == 0 {
		return []*MPoly{NewZeroMPoly(numVars)}, nil
	}

	gbI, err := buchberger(I, order)
	if err != nil {
		return nil, err
	}
	redGB := reduceGroebnerBasis(gbI, order)

	if !IsZeroDimensional(redGB, numVars) {
		return nil, fmt.Errorf("ZeroDimensionalRadical requires a zero-dimensional ideal")
	}

	// For each variable x_i, compute its elimination minimal polynomial g_i(x_i)
	generators := append([]*MPoly{}, redGB...)

	for i := 0; i < numVars; i++ {
		xiPoly := makeVarPowMPoly(i, 1, numVars)
		gi, err := MinimalPolynomialOfElement(xiPoly, redGB, numVars, order)
		if err != nil {
			return nil, err
		}

		sqfree, err := univariateSquareFree(gi, i, order)
		if err != nil {
			return nil, err
		}
		generators = append(generators, sqfree)
	}

	radGB, err := buchberger(generators, order)
	if err != nil {
		return nil, err
	}
	return reduceGroebnerBasis(radGB, order), nil
}

// FindSeparatingLinearForm finds a deterministic linear form u = sum c_j x_j
// that separates all points of the reduced zero-dimensional ideal sqrt(I).
func FindSeparatingLinearForm(I []*MPoly, numVars int, order MonomialOrder) ([]int, *MPoly, error) {
	rad, err := ZeroDimensionalRadical(I, numVars, order)
	if err != nil {
		return nil, nil, err
	}

	B_rad := StandardMonomialBasis(rad, numVars)
	dimRad := len(B_rad)
	if dimRad <= 1 {
		// Trivially separated
		c := make([]int, numVars)
		c[0] = 1
		uPoly := makeVarPowMPoly(0, 1, numVars)
		minP, _ := MinimalPolynomialOfElement(uPoly, I, numVars, order)
		return c, minP, nil
	}

	// Deterministic candidates for coefficient vectors
	candidates := [][]int{
		{1, 0}, {0, 1}, {1, 1}, {1, 2}, {1, -1}, {2, 1}, {1, 3}, {1, -2},
		{1, 1, 1}, {1, 2, 3}, {1, 2, 4}, {1, -1, 2},
	}

	// Generate small integer tuples
	var genCandidates func(curr []int, depth int) [][]int
	genCandidates = func(curr []int, depth int) [][]int {
		if depth == numVars {
			// Must have at least one non-zero
			hasNonZero := false
			for _, v := range curr {
				if v != 0 {
					hasNonZero = true
					break
				}
			}
			if hasNonZero {
				c := make([]int, numVars)
				copy(c, curr)
				return [][]int{c}
			}
			return nil
		}
		var list [][]int
		for val := 0; val <= 3; val++ {
			list = append(list, genCandidates(append(curr, val), depth+1)...)
		}
		for val := -1; val >= -2; val-- {
			list = append(list, genCandidates(append(curr, val), depth+1)...)
		}
		return list
	}

	allCandidates := append(candidates, genCandidates(nil, 0)...)

	for _, cand := range allCandidates {
		if len(cand) != numVars {
			continue
		}

		// Build u = sum cand[j] * x_j
		u := NewZeroMPoly(numVars)
		for j, cj := range cand {
			if cj != 0 {
				xj := makeVarPowMPoly(j, 1, numVars)
				scaled := xj.MulTerm(Term{Coeff: big.NewRat(int64(cj), 1), Mon: zeroMonomial(numVars)}, order)
				u = u.Add(scaled, order)
			}
		}

		// Compute minimal polynomial on rad
		minRad, err := MinimalPolynomialOfElement(u, rad, numVars, order)
		if err == nil && minRad.Degree() == dimRad {
			// Found separating linear form!
			// Compute minimal polynomial on original ideal I
			minI, err := MinimalPolynomialOfElement(u, I, numVars, order)
			if err != nil {
				return nil, nil, err
			}
			return cand, minI, nil
		}
	}

	return nil, nil, fmt.Errorf("failed to find separating linear form")
}

// -------------------------------------------------------------------------
// Zero-Dimensional Primary Decomposition (Slice 4)
// -------------------------------------------------------------------------

// PrimaryComponent represents a primary ideal component Q and its associated prime P = sqrt(Q).
type PrimaryComponent struct {
	Primary []*MPoly // Q_i
	Prime   []*MPoly // P_i = sqrt(Q_i)
}

// buildUnivariateNodeFromCoeffs converts coefficients [c_0, c_1, ..., c_k] into an AST Node:
// c_k * T^k + ... + c_1 * T + c_0.
func buildUnivariateNodeFromCoeffs(coeffs []*big.Rat, varName string) Node {
	var terms []Node
	for deg, c := range coeffs {
		if c.Sign() == 0 {
			continue
		}
		cNode := &RationalNode{Val: new(big.Rat).Set(c)}
		if deg == 0 {
			terms = append(terms, cNode)
		} else if deg == 1 {
			if c.Cmp(big.NewRat(1, 1)) == 0 {
				terms = append(terms, &VarNode{Name: varName})
			} else {
				terms = append(terms, &MulNode{Factors: []Node{cNode, &VarNode{Name: varName}}})
			}
		} else {
			powNode := &PowNode{Base: &VarNode{Name: varName}, Exp: &RationalNode{Val: big.NewRat(int64(deg), 1)}}
			if c.Cmp(big.NewRat(1, 1)) == 0 {
				terms = append(terms, powNode)
			} else {
				terms = append(terms, &MulNode{Factors: []Node{cNode, powNode}})
			}
		}
	}
	if len(terms) == 0 {
		return &RationalNode{Val: big.NewRat(0, 1)}
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return &AddNode{Terms: terms}
}

type univariateFactor struct {
	polyNode Node
	exp      int
}

// extractFactorsFromNode breaks down a factored expression into its components.
func extractFactorsFromNode(n Node) []univariateFactor {
	switch v := n.(type) {
	case *MulNode:
		var res []univariateFactor
		for _, f := range v.Factors {
			res = append(res, extractFactorsFromNode(f)...)
		}
		return res
	case *PowNode:
		if rat, ok := v.Exp.(*RationalNode); ok && rat.Val.IsInt() {
			exp := int(rat.Val.Num().Int64())
			return []univariateFactor{{polyNode: v.Base, exp: exp}}
		}
		return []univariateFactor{{polyNode: n, exp: 1}}
	case *RationalNode:
		return nil
	default:
		return []univariateFactor{{polyNode: n, exp: 1}}
	}
}

// evalNodeWithPoly substitutes varName with the multivariate polynomial u.
func evalNodeWithPoly(n Node, varName string, u *MPoly, numVars int, order MonomialOrder) (*MPoly, error) {
	switch v := n.(type) {
	case *VarNode:
		if v.Name == varName {
			return u.Clone(), nil
		}
		return nil, fmt.Errorf("unexpected variable: %s", v.Name)
	case *RationalNode:
		return &MPoly{Terms: []Term{{Coeff: new(big.Rat).Set(v.Val), Mon: zeroMonomial(numVars)}}}, nil
	case *AddNode:
		res := NewZeroMPoly(numVars)
		for _, term := range v.Terms {
			evalTerm, err := evalNodeWithPoly(term, varName, u, numVars, order)
			if err != nil {
				return nil, err
			}
			res = res.Add(evalTerm, order)
		}
		return res, nil
	case *MulNode:
		res := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}}}
		for _, factor := range v.Factors {
			evalFac, err := evalNodeWithPoly(factor, varName, u, numVars, order)
			if err != nil {
				return nil, err
			}
			res = res.Mul(evalFac, order)
		}
		return res, nil
	case *PowNode:
		base, err := evalNodeWithPoly(v.Base, varName, u, numVars, order)
		if err != nil {
			return nil, err
		}
		rat, ok := v.Exp.(*RationalNode)
		if !ok || !rat.Val.IsInt() || rat.Val.Sign() < 0 {
			return nil, fmt.Errorf("unsupported exponent: %v", v.Exp)
		}
		exp := int(rat.Val.Num().Int64())
		res := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}}}
		for i := 0; i < exp; i++ {
			res = res.Mul(base, order)
		}
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported AST node: %T", n)
	}
}

// DecomposeZeroDimensionalIdeal decomposes a zero-dimensional polynomial ideal I
// into its irredundant primary components Q_i and associated primes P_i = sqrt(Q_i).
func DecomposeZeroDimensionalIdeal(I []*MPoly, numVars int, order MonomialOrder) ([]PrimaryComponent, error) {
	if len(I) == 0 {
		return nil, nil
	}

	gbI, err := buchberger(I, order)
	if err != nil {
		return nil, err
	}
	redGB := reduceGroebnerBasis(gbI, order)

	// Check if unit ideal
	for _, g := range redGB {
		if !g.IsZero() && g.LeadingMonomial().IsZero() {
			return nil, nil
		}
	}

	if !IsZeroDimensional(redGB, numVars) {
		return nil, fmt.Errorf("ideal is not zero-dimensional")
	}

	B := StandardMonomialBasis(redGB, numVars)
	if len(B) <= 1 {
		return []PrimaryComponent{{Primary: redGB, Prime: redGB}}, nil
	}

	// 1. Find separating linear form u
	coeffsCand, _, err := FindSeparatingLinearForm(redGB, numVars, order)
	if err != nil {
		return nil, err
	}

	u := NewZeroMPoly(numVars)
	for j, cj := range coeffsCand {
		if cj != 0 {
			xj := makeVarPowMPoly(j, 1, numVars)
			scaled := xj.MulTerm(Term{Coeff: big.NewRat(int64(cj), 1), Mon: zeroMonomial(numVars)}, order)
			u = u.Add(scaled, order)
		}
	}

	// 2. Minimal polynomial coefficients of u modulo I
	minCoeffs, err := MinimalPolynomialCoefficients(u, redGB, numVars, order)
	if err != nil {
		return nil, err
	}

	// 3. Build univariate AST node P(T) and factor it over Q
	nodeP := buildUnivariateNodeFromCoeffs(minCoeffs, "T")
	factoredNode, err := factorPolynomial(nodeP, "T")
	if err != nil {
		return nil, err
	}

	factors := extractFactorsFromNode(factoredNode)

	type nonTrivialFactor struct {
		fullPoly *MPoly
	}
	var validFactors []nonTrivialFactor

	for _, fac := range factors {
		basePoly, err := evalNodeWithPoly(fac.polyNode, "T", u, numVars, order)
		if err != nil {
			return nil, err
		}
		if basePoly.IsZero() || basePoly.LeadingMonomial().IsZero() {
			continue
		}

		fullPoly := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}}}
		for e := 0; e < fac.exp; e++ {
			fullPoly = fullPoly.Mul(basePoly, order)
		}

		validFactors = append(validFactors, nonTrivialFactor{
			fullPoly: fullPoly.Monic(),
		})
	}

	if len(validFactors) <= 1 {
		// Already primary!
		radI, err := ZeroDimensionalRadical(redGB, numVars, order)
		if err != nil {
			radI = redGB
		}
		return []PrimaryComponent{
			{
				Primary: redGB,
				Prime:   radI,
			},
		}, nil
	}

	// Split into branches: I_k = I + <q_k(u)^{e_k}>
	var allComponents []PrimaryComponent
	for _, vf := range validFactors {
		branchG := append([]*MPoly{}, redGB...)
		branchG = append(branchG, vf.fullPoly)

		branchComps, err := DecomposeZeroDimensionalIdeal(branchG, numVars, order)
		if err != nil {
			return nil, err
		}
		allComponents = append(allComponents, branchComps...)
	}

	return allComponents, nil
}

// -------------------------------------------------------------------------
// Positive-Dimensional GTZ Pipeline & Irredundancy Cleanup (Slice 5)
// -------------------------------------------------------------------------

// permutePolyVars re-indexes variables according to perm mapping:
// new_var_idx = perm[old_var_idx].
func permutePolyVars(p *MPoly, perm []int) *MPoly {
	if p == nil || p.IsZero() {
		return NewZeroMPoly(len(perm))
	}
	n := len(perm)
	newTerms := make([]Term, len(p.Terms))
	for i, t := range p.Terms {
		newExps := make([]int, n)
		for origIdx, exp := range t.Mon.Exps {
			if origIdx < len(perm) {
				newExps[perm[origIdx]] = exp
			}
		}
		newTerms[i] = Term{
			Coeff: new(big.Rat).Set(t.Coeff),
			Mon:   newMonomial(newExps),
		}
	}
	return &MPoly{Terms: newTerms, Sugar: p.Sugar}
}

// invertPermutation returns inv such that inv[perm[i]] = i.
func invertPermutation(perm []int) []int {
	inv := make([]int, len(perm))
	for i, p := range perm {
		inv[p] = i
	}
	return inv
}

// unpermutePolyVars restores original variable indices from permuted indices:
// orig_var_idx = invPerm[perm_var_idx].
func unpermutePolyVars(p *MPoly, invPerm []int) *MPoly {
	if p == nil || p.IsZero() {
		return NewZeroMPoly(len(invPerm))
	}
	n := len(invPerm)
	newTerms := make([]Term, len(p.Terms))
	for i, t := range p.Terms {
		origExps := make([]int, n)
		for permIdx, exp := range t.Mon.Exps {
			if permIdx < len(invPerm) {
				origExps[invPerm[permIdx]] = exp
			}
		}
		newTerms[i] = Term{
			Coeff: new(big.Rat).Set(t.Coeff),
			Mon:   newMonomial(origExps),
		}
	}
	return &MPoly{Terms: newTerms, Sugar: p.Sugar}
}

// cleanIrredundantComponents removes redundant primary components:
// 1. Merges components with identical associated prime P (via ideal intersection).
// 2. Removes component Q_i if the intersection of all other components is contained in Q_i.
func cleanIrredundantComponents(comps []PrimaryComponent, order MonomialOrder) ([]PrimaryComponent, error) {
	if len(comps) <= 1 {
		return comps, nil
	}

	// 1. Merge components with identical associated prime P
	var merged []PrimaryComponent
	used := make([]bool, len(comps))

	for i := 0; i < len(comps); i++ {
		if used[i] {
			continue
		}
		currQ := comps[i].Primary
		currP := comps[i].Prime
		used[i] = true

		for j := i + 1; j < len(comps); j++ {
			if used[j] {
				continue
			}
			eqPrime, err := IdealEquals(currP, comps[j].Prime, order)
			if err != nil {
				return nil, err
			}
			if eqPrime {
				interQ, err := IdealIntersection(currQ, comps[j].Primary, order)
				if err != nil {
					return nil, err
				}
				currQ = interQ
				used[j] = true
			}
		}
		merged = append(merged, PrimaryComponent{Primary: currQ, Prime: currP})
	}

	if len(merged) <= 1 {
		return merged, nil
	}

	// 2. Remove redundant components
	var result []PrimaryComponent
	for i := 0; i < len(merged); i++ {
		var otherInter []*MPoly
		first := true
		var err error
		for j := 0; j < len(merged); j++ {
			if i == j {
				continue
			}
			if first {
				otherInter = merged[j].Primary
				first = false
			} else {
				otherInter, err = IdealIntersection(otherInter, merged[j].Primary, order)
				if err != nil {
					return nil, err
				}
			}
		}

		isSubset, err := IdealContains(merged[i].Primary, otherInter, order)
		if err != nil {
			return nil, err
		}
		if isSubset {
			continue // Redundant component!
		}
		result = append(result, merged[i])
	}

	return result, nil
}

// polySqrt computes the polynomial square root of p, such that H^2 == p.
// Returns nil if p is not a perfect square in Q[X].
func polySqrt(p *MPoly, order MonomialOrder) *MPoly {
	if p.IsZero() {
		return p.Clone()
	}
	lt := p.LeadingTerm()
	if lt.Coeff.Sign() <= 0 {
		return nil
	}
	halfExps := make([]int, len(lt.Mon.Exps))
	for i, e := range lt.Mon.Exps {
		if e%2 != 0 {
			return nil
		}
		halfExps[i] = e / 2
	}

	sqrtNum := new(big.Int).Sqrt(lt.Coeff.Num())
	sqrtDenom := new(big.Int).Sqrt(lt.Coeff.Denom())
	if new(big.Int).Mul(sqrtNum, sqrtNum).Cmp(lt.Coeff.Num()) != 0 ||
		new(big.Int).Mul(sqrtDenom, sqrtDenom).Cmp(lt.Coeff.Denom()) != 0 {
		return nil
	}
	c0 := new(big.Rat).SetFrac(sqrtNum, sqrtDenom)

	H := &MPoly{Terms: []Term{{Coeff: c0, Mon: newMonomial(halfExps)}}}
	twoH0 := Term{Coeff: new(big.Rat).Mul(big.NewRat(2, 1), c0), Mon: newMonomial(halfExps)}

	maxSteps := len(p.Terms) + 5
	for step := 0; step < maxSteps; step++ {
		h2 := H.Mul(H, order)
		diff := p.Sub(h2, order)
		if diff.IsZero() {
			return H
		}
		ltDiff := diff.LeadingTerm()
		if !twoH0.Mon.Divides(ltDiff.Mon) {
			return nil
		}
		nextMon := ltDiff.Mon.Div(twoH0.Mon)
		nextCoeff := new(big.Rat).Quo(ltDiff.Coeff, twoH0.Coeff)
		nextTerm := Term{Coeff: nextCoeff, Mon: nextMon}

		H = H.Add(&MPoly{Terms: []Term{nextTerm}}, order)
	}

	if p.Sub(H.Mul(H, order), order).IsZero() {
		return H
	}
	return nil
}

// findNontrivialFactor attempts to find a non-trivial factor of p.
func findNontrivialFactor(p *MPoly, numVars int, order MonomialOrder) *MPoly {
	if p.IsZero() || p.Degree() <= 1 {
		return nil
	}

	// 1. Common monomial factor (x_i)
	for i := 0; i < numVars; i++ {
		minExp := -1
		for _, t := range p.Terms {
			e := t.Mon.Exps[i]
			if minExp == -1 || e < minExp {
				minExp = e
			}
		}
		if minExp > 0 {
			return makeVarPowMPoly(i, 1, numVars)
		}
	}

	// 2. Linear factors of the form x_i +/- x_j +/- c, or x_i +/- c
	var candidates []*MPoly
	for i := 0; i < numVars; i++ {
		xi := makeVarPowMPoly(i, 1, numVars)
		for _, c := range []int64{1, -1, 2, -2, 3, -3} {
			cPoly := makeConstMPoly(c, numVars)
			candidates = append(candidates, xi.Add(cPoly, order))
		}
		for j := i + 1; j < numVars; j++ {
			xj := makeVarPowMPoly(j, 1, numVars)
			candidates = append(candidates, xi.Sub(xj, order))
			candidates = append(candidates, xi.Add(xj, order))
			for _, c := range []int64{1, -1} {
				cPoly := makeConstMPoly(c, numVars)
				candidates = append(candidates, xi.Sub(xj, order).Add(cPoly, order))
				candidates = append(candidates, xi.Add(xj, order).Add(cPoly, order))
			}
		}
	}

	for _, cand := range candidates {
		_, rem := polyDivisionWithQuotients(p, []*MPoly{cand}, order)
		if rem.IsZero() {
			return cand
		}
	}

	// 3. Quadratic polynomial in variable x_i with constant leading coeff: A*x_i^2 + B*x_i + C
	for i := 0; i < numVars; i++ {
		maxDeg := 0
		hasOther := false
		for _, t := range p.Terms {
			if t.Mon.Exps[i] > maxDeg {
				maxDeg = t.Mon.Exps[i]
			}
			if t.Mon.Exps[i] != 2 && t.Mon.Exps[i] != 1 && t.Mon.Exps[i] != 0 {
				hasOther = true
			}
		}
		if maxDeg == 2 && !hasOther {
			A := NewZeroMPoly(numVars)
			B := NewZeroMPoly(numVars)
			C := NewZeroMPoly(numVars)
			for _, t := range p.Terms {
				e := t.Mon.Exps[i]
				exps := make([]int, numVars)
				copy(exps, t.Mon.Exps)
				exps[i] = 0
				reducedTerm := Term{Coeff: new(big.Rat).Set(t.Coeff), Mon: newMonomial(exps)}
				singlePoly := &MPoly{Terms: []Term{reducedTerm}}
				switch e {
				case 2:
					A = A.Add(singlePoly, order)
				case 1:
					B = B.Add(singlePoly, order)
				case 0:
					C = C.Add(singlePoly, order)
				}
			}

			if !A.IsZero() && A.LeadingMonomial().IsZero() {
				cA := A.Terms[0].Coeff
				b2 := B.Mul(B, order)
				fourAC := C.MulTerm(Term{Coeff: new(big.Rat).Mul(big.NewRat(4, 1), cA), Mon: zeroMonomial(numVars)}, order)
				D := b2.Sub(fourAC, order)
				sqrtD := polySqrt(D, order)
				if sqrtD != nil {
					twoA := new(big.Rat).Mul(big.NewRat(2, 1), cA)
					twoAx := makeVarPowMPoly(i, 1, numVars).MulTerm(Term{Coeff: twoA, Mon: zeroMonomial(numVars)}, order)
					f := twoAx.Add(B, order).Sub(sqrtD, order)
					if !f.IsZero() && f.Degree() < p.Degree() {
						return f
					}
				}
			}
		}
	}

	return nil
}

// extractLeadingCoefficientInV extracts the coefficient of the leading monomial in V
// as a polynomial in U (where V has indices 0..lenV-1, and U has indices lenV..n-1).
func extractLeadingCoefficientInV(p *MPoly, lenV, numVars int) *MPoly {
	if p.IsZero() {
		return nil
	}
	var maxVMon Monomial
	var maxVExps []int
	hasV := false
	for _, t := range p.Terms {
		vExps := make([]int, lenV)
		copy(vExps, t.Mon.Exps[:lenV])
		vMon := newMonomial(vExps)
		if !hasV || CompareMonomials(vMon, maxVMon, OrderLex) > 0 {
			maxVMon = vMon
			maxVExps = vExps
			hasV = true
		}
	}
	if !hasV {
		return nil
	}

	var uTerms []Term
	for _, t := range p.Terms {
		match := true
		for i := 0; i < lenV; i++ {
			if t.Mon.Exps[i] != maxVExps[i] {
				match = false
				break
			}
		}
		if match {
			uExps := make([]int, numVars)
			copy(uExps[lenV:], t.Mon.Exps[lenV:])
			uTerms = append(uTerms, Term{
				Coeff: new(big.Rat).Set(t.Coeff),
				Mon:   newMonomial(uExps),
			})
		}
	}

	return &MPoly{Terms: uTerms, Sugar: 0}
}

// PrimaryDecomposition computes the minimal irredundant primary decomposition of ideal I
// over Q[x_1, ..., x_n] using Gianni-Trager-Zacharias (GTZ 1988) algorithm.
func PrimaryDecomposition(I []*MPoly, numVars int, order MonomialOrder) ([]PrimaryComponent, error) {
	if len(I) == 0 {
		return nil, nil
	}

	gbI, err := buchberger(I, order)
	if err != nil {
		return nil, err
	}
	redGB := reduceGroebnerBasis(gbI, order)

	// Check if unit ideal <1>
	for _, g := range redGB {
		if !g.IsZero() && g.LeadingMonomial().IsZero() {
			return nil, nil
		}
	}

	// Splitting by factorization (GTZ / Möller 1993):
	// If any basis element g has a non-trivial factor f1 not in I,
	// split I into (I + <f1>) and (I : f1^infinity).
	for _, g := range redGB {
		if g.Degree() >= 2 {
			f1 := findNontrivialFactor(g, numVars, order)
			if f1 != nil && f1.Degree() > 0 && f1.Degree() < g.Degree() {
				// Check if f1 is already in I
				if !polyReduce(f1, redGB, order).IsZero() {
					I1 := append([]*MPoly{}, redGB...)
					I1 = append(I1, f1)

					I2, _, err := IdealSaturation(redGB, f1, order)
					if err == nil && len(I2) > 0 {
						comps1, err1 := PrimaryDecomposition(I1, numVars, order)
						comps2, err2 := PrimaryDecomposition(I2, numVars, order)
						if err1 == nil && err2 == nil && len(comps1) > 0 && len(comps2) > 0 {
							all := append(comps1, comps2...)
							return cleanIrredundantComponents(all, order)
						}
					}
				}
			}
		}
	}

	dim, U := IdealDimensionAndMaximalIndependentSet(redGB, numVars)
	if dim == -1 {
		return nil, nil
	}
	if dim == 0 {
		comps, err := DecomposeZeroDimensionalIdeal(redGB, numVars, order)
		if err != nil {
			return nil, err
		}
		return cleanIrredundantComponents(comps, order)
	}

	// Positive-Dimensional Ideal (dim >= 1):
	uMap := make(map[int]bool, len(U))
	for _, uIdx := range U {
		uMap[uIdx] = true
	}
	var V []int
	for i := 0; i < numVars; i++ {
		if !uMap[i] {
			V = append(V, i)
		}
	}

	// Permutation: V variables first, U variables second
	perm := make([]int, numVars)
	newIdx := 0
	for _, vIdx := range V {
		perm[vIdx] = newIdx
		newIdx++
	}
	for _, uIdx := range U {
		perm[uIdx] = newIdx
		newIdx++
	}
	invPerm := invertPermutation(perm)

	// Permute I to V > U ring
	permG := make([]*MPoly, len(redGB))
	for i, p := range redGB {
		permG[i] = permutePolyVars(p, perm)
	}

	// Compute Groebner basis under OrderLex (V > U)
	permGB, err := buchberger(permG, OrderLex)
	if err != nil {
		return nil, err
	}
	redPermGB := reduceGroebnerBasis(permGB, OrderLex)

	// Find non-constant leading coefficient in V
	var hPerm *MPoly
	for _, g := range redPermGB {
		lcV := extractLeadingCoefficientInV(g, len(V), numVars)
		if lcV != nil && !lcV.IsZero() && !lcV.LeadingMonomial().IsZero() {
			hPerm = lcV
			break
		}
	}

	if hPerm == nil {
		return []PrimaryComponent{{Primary: redGB, Prime: redGB}}, nil
	}

	// Unpermute h back to original coordinates
	h := unpermutePolyVars(hPerm, invPerm)

	// Saturate I with respect to h: I1 = I : h^infinity, with stabilization index m
	I1, m, err := IdealSaturation(redGB, h, order)
	if err != nil {
		return nil, err
	}

	if m == 0 {
		return []PrimaryComponent{{Primary: redGB, Prime: redGB}}, nil
	}

	// Compute h^m
	hm := &MPoly{Terms: []Term{{Coeff: big.NewRat(1, 1), Mon: zeroMonomial(numVars)}}}
	for k := 0; k < m; k++ {
		hm = hm.Mul(h, order)
	}

	// Form remainder branch I2 = I + <h^m>
	I2 := append([]*MPoly{}, redGB...)
	I2 = append(I2, hm)

	comps1, err := PrimaryDecomposition(I1, numVars, order)
	if err != nil {
		return nil, err
	}
	comps2, err := PrimaryDecomposition(I2, numVars, order)
	if err != nil {
		return nil, err
	}

	allComps := append(comps1, comps2...)
	return cleanIrredundantComponents(allComps, order)
}

// IdealRadical computes the radical sqrt(I) of an arbitrary polynomial ideal I.
func IdealRadical(I []*MPoly, numVars int, order MonomialOrder) ([]*MPoly, error) {
	comps, err := PrimaryDecomposition(I, numVars, order)
	if err != nil {
		return nil, err
	}
	if len(comps) == 0 {
		return []*MPoly{NewZeroMPoly(numVars)}, nil
	}

	res := comps[0].Prime
	for i := 1; i < len(comps); i++ {
		res, err = IdealIntersection(res, comps[i].Prime, order)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// -------------------------------------------------------------------------
// Top-Level CAS Interfaces for Ideal Operations (Slice 7)
// -------------------------------------------------------------------------

// EvalPrimaryDecomposition computes the minimal primary decomposition of an ideal.
// Syntax: primary_decomposition([p1, ...], [x, y, ...], [order])
// Returns a list of components, where each component is [primary_generators_list, prime_generators_list].
func EvalPrimaryDecomposition(polysNode, varsNode, orderOpt Node, env *Env) (Node, error) {
	polys, varNames, order, err := parseMPolyInputArgs(polysNode, varsNode, orderOpt)
	if err != nil {
		return nil, err
	}

	comps, err := PrimaryDecomposition(polys, len(varNames), order)
	if err != nil {
		return nil, err
	}

	var compNodes []Node
	var irComps [][]Node
	var irPrimes [][]Node

	for _, c := range comps {
		var qList []Node
		for _, q := range c.Primary {
			qList = append(qList, mpolyToNode(q, varNames))
		}
		var pList []Node
		for _, p := range c.Prime {
			pList = append(pList, mpolyToNode(p, varNames))
		}
		irComps = append(irComps, qList)
		irPrimes = append(irPrimes, pList)

		compPair := NewList([]Node{NewList(qList), NewList(pList)})
		compNodes = append(compNodes, compPair)
	}

	resList := NewList(compNodes)

	// Build certificate if env is present
	if env != nil {
		var idealGenNodes []Node
		if pl, ok := polysNode.(*ListNode); ok {
			idealGenNodes = pl.Elements
		} else {
			idealGenNodes = []Node{polysNode}
		}

		gb, _ := buchberger(polys, order)
		dim, _ := IdealDimensionAndMaximalIndependentSet(gb, len(varNames))

		var fwdSyzygies []IdealSyzygyRelation
		if len(comps) > 0 && len(polys) > 0 {
			comp0GB := comps[0].Primary
			for _, f := range polys {
				qs, rem := polyDivisionWithQuotients(f, comp0GB, order)
				if rem.IsZero() {
					var qNodes, gNodes []Node
					for j, q := range qs {
						if !q.IsZero() {
							qNodes = append(qNodes, mpolyToNode(q, varNames))
							gNodes = append(gNodes, mpolyToNode(comp0GB[j], varNames))
						}
					}
					if len(qNodes) > 0 {
						fwdSyzygies = append(fwdSyzygies, IdealSyzygyRelation{
							TargetNode: mpolyToNode(f, varNames),
							CombCoeffs: qNodes,
							Gens:       gNodes,
						})
					}
				}
			}
		}

		certIR := NewIdealDecompositionCertificate(
			idealGenNodes,
			irComps,
			irPrimes,
			dim,
			fwdSyzygies,
			nil,
			nil,
			mustRational(0, 1),
			true,
			"Primary decomposition verified via Gianni-Trager-Zacharias algorithm",
		)
		env.LastCert = &VerificationCertificate{
			Domain:     DomainIdeal,
			Equation:   certIR.EquationString(),
			Residual:   mustRational(0, 1),
			IsVerified: true,
			Details:    certIR.DetailsMsg,
			State:      VerifyStateCertified,
			CertIR:     certIR,
		}
	}

	return resList, nil
}

// EvalIdealRadical computes the radical of an ideal sqrt(I).
// Syntax: ideal_radical([p1, ...], [x, y, ...], [order])
func EvalIdealRadical(polysNode, varsNode, orderOpt Node, env *Env) (Node, error) {
	polys, varNames, order, err := parseMPolyInputArgs(polysNode, varsNode, orderOpt)
	if err != nil {
		return nil, err
	}

	rad, err := IdealRadical(polys, len(varNames), order)
	if err != nil {
		return nil, err
	}

	var resNodes []Node
	for _, p := range rad {
		resNodes = append(resNodes, mpolyToNode(p, varNames))
	}
	return NewList(resNodes), nil
}

// EvalIdealDimension computes the Krull dimension dim(Q[X]/I).
// Syntax: ideal_dimension([p1, ...], [x, y, ...], [order])
func EvalIdealDimension(polysNode, varsNode, orderOpt Node, env *Env) (Node, error) {
	polys, varNames, order, err := parseMPolyInputArgs(polysNode, varsNode, orderOpt)
	if err != nil {
		return nil, err
	}

	gb, err := buchberger(polys, order)
	if err != nil {
		return nil, err
	}
	redGB := reduceGroebnerBasis(gb, order)

	dim, _ := IdealDimensionAndMaximalIndependentSet(redGB, len(varNames))
	return mustRational(int64(dim), 1), nil
}

// EvalIdealIntersection computes the intersection I1 cap I2.
// Syntax: ideal_intersection([p1, ...], [q1, ...], [x, y, ...], [order])
func EvalIdealIntersection(p1Node, p2Node, varsNode, orderOpt Node, env *Env) (Node, error) {
	polys1, varNames, order, err := parseMPolyInputArgs(p1Node, varsNode, orderOpt)
	if err != nil {
		return nil, err
	}
	polys2, _, _, err := parseMPolyInputArgs(p2Node, varsNode, orderOpt)
	if err != nil {
		return nil, err
	}

	inter, err := IdealIntersection(polys1, polys2, order)
	if err != nil {
		return nil, err
	}

	var resNodes []Node
	for _, p := range inter {
		resNodes = append(resNodes, mpolyToNode(p, varNames))
	}
	return NewList(resNodes), nil
}

// EvalIdealColon computes the ideal quotient I : f.
// Syntax: ideal_colon([p1, ...], f, [x, y, ...], [order])
func EvalIdealColon(polysNode, fNode, varsNode, orderOpt Node, env *Env) (Node, error) {
	polys, varNames, order, err := parseMPolyInputArgs(polysNode, varsNode, orderOpt)
	if err != nil {
		return nil, err
	}
	fPolys, _, _, err := parseMPolyInputArgs(fNode, varsNode, orderOpt)
	if err != nil {
		return nil, err
	}
	if len(fPolys) == 0 {
		return nil, fmt.Errorf("colon polynomial cannot be empty")
	}

	colon, err := IdealColon(polys, fPolys[0], order)
	if err != nil {
		return nil, err
	}

	var resNodes []Node
	for _, p := range colon {
		resNodes = append(resNodes, mpolyToNode(p, varNames))
	}
	return NewList(resNodes), nil
}

// EvalIdealSaturation computes the saturation ideal I : f^infinity.
// Syntax: ideal_saturation([p1, ...], f, [x, y, ...], [order])
func EvalIdealSaturation(polysNode, fNode, varsNode, orderOpt Node, env *Env) (Node, error) {
	polys, varNames, order, err := parseMPolyInputArgs(polysNode, varsNode, orderOpt)
	if err != nil {
		return nil, err
	}
	fPolys, _, _, err := parseMPolyInputArgs(fNode, varsNode, orderOpt)
	if err != nil {
		return nil, err
	}
	if len(fPolys) == 0 {
		return nil, fmt.Errorf("saturation polynomial cannot be empty")
	}

	sat, _, err := IdealSaturation(polys, fPolys[0], order)
	if err != nil {
		return nil, err
	}

	var resNodes []Node
	for _, p := range sat {
		resNodes = append(resNodes, mpolyToNode(p, varNames))
	}
	return NewList(resNodes), nil
}



