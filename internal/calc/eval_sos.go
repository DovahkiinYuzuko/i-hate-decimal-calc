package calc

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Multivariate Polynomial Sum-of-Squares (SOS) Decomposition
// (Powers & Wörmann 1998, Reznick 1989, Harrison 2007)
// -------------------------------------------------------------------------

// SOSStatus defines the 3-valued verification status of a polynomial's SOS decomposition.
type SOSStatus int

const (
	// SOSCertified indicates that an exact rational SOS decomposition P = sum c_i q_i^2 (c_i > 0) was found.
	SOSCertified SOSStatus = iota
	// SOSNoCertificateFound indicates that no rational SOS certificate was found under the current basis.
	SOSNoCertificateFound
	// SOSRefuted indicates that a rational witness point a in Q^n was found such that P(a) < 0.
	SOSRefuted
)

func (s SOSStatus) String() string {
	switch s {
	case SOSCertified:
		return "certified"
	case SOSNoCertificateFound:
		return "no_certificate_found"
	case SOSRefuted:
		return "refuted"
	default:
		return "unknown"
	}
}

// SOSTerm represents a single weighted square component c_i * q_i(x)^2 in an SOS decomposition.
type SOSTerm struct {
	Weight *big.Rat // c_i > 0 in Q
	Poly   *MPoly   // q_i(x) in Q[x_1, ..., x_n]
}

// SOSResult encapsulates the complete outcome of an SOS decomposition attempt.
type SOSResult struct {
	Status         SOSStatus
	Terms          []SOSTerm
	Residual       *MPoly
	Counterexample map[string]*big.Rat
	Vars           []string
}

// evaluateAt evaluates an MPoly at a rational point.
func evaluateAt(p *MPoly, point []*big.Rat) *big.Rat {
	res := new(big.Rat)
	for _, t := range p.Terms {
		termVal := new(big.Rat).Set(t.Coeff)
		for i, exp := range t.Mon.Exps {
			if exp > 0 {
				xi := point[i]
				powVal := new(big.Rat).Set(xi)
				for e := 1; e < exp; e++ {
					powVal.Mul(powVal, xi)
				}
				termVal.Mul(termVal, powVal)
			}
		}
		res.Add(res, termVal)
	}
	return res
}

// findOddDegreeRefutation uses Cauchy's root bound to mathematically certify a witness point where P(x) < 0.
func findOddDegreeRefutation(p *MPoly, vars []string) map[string]*big.Rat {
	n := len(vars)
	if n == 0 || p.IsZero() {
		return nil
	}

	// Try each variable as the principal variable
	for k := 0; k < n; k++ {
		// Try combinations of fixing other variables to 0 or 1
		for _, fixVal := range []int64{0, 1} {
			point := make([]*big.Rat, n)
			for j := 0; j < n; j++ {
				if j != k {
					point[j] = big.NewRat(fixVal, 1)
				}
			}

			// Specialize p to a univariate polynomial in x_k: p_uni(x_k) = sum coeff_d * x_k^d
			coeffs := make(map[int]*big.Rat)
			for _, t := range p.Terms {
				val := new(big.Rat).Set(t.Coeff)
				for j, exp := range t.Mon.Exps {
					if j != k && exp > 0 {
						if fixVal == 0 {
							val = nil
							break
						}
						// fixVal == 1 => 1^exp == 1, val unchanged
					}
				}
				if val == nil || val.Sign() == 0 {
					continue
				}
				deg := t.Mon.Exps[k]
				if cur, exists := coeffs[deg]; exists {
					cur.Add(cur, val)
				} else {
					coeffs[deg] = val
				}
			}

			// Find highest degree of x_k
			maxDeg := -1
			var leadingCoeff *big.Rat
			for deg, c := range coeffs {
				if c.Sign() != 0 && deg > maxDeg {
					maxDeg = deg
					leadingCoeff = c
				}
			}

			// If highest degree is odd, Cauchy bound guarantees existence of a negative point
			if maxDeg > 0 && maxDeg%2 != 0 && leadingCoeff != nil && leadingCoeff.Sign() != 0 {
				// Compute Cauchy root bound: R = 1 + max_{d < maxDeg} |coeff_d / leadingCoeff|
				absLC := new(big.Rat).Abs(leadingCoeff)
				maxRatio := new(big.Rat)
				for deg, c := range coeffs {
					if deg < maxDeg && c.Sign() != 0 {
						ratio := new(big.Rat).Abs(c)
						ratio.Quo(ratio, absLC)
						if ratio.Cmp(maxRatio) > 0 {
							maxRatio.Set(ratio)
						}
					}
				}
				bound := new(big.Rat).Add(big.NewRat(1, 1), maxRatio)
				// Ceiling integer M = ceil(bound) + 1
				num := bound.Num()
				denom := bound.Denom()
				ceilInt := new(big.Int).Add(num, denom)
				ceilInt.Sub(ceilInt, big.NewInt(1))
				ceilInt.Quo(ceilInt, denom)
				ceilInt.Add(ceilInt, big.NewInt(1))

				var xVal *big.Rat
				if leadingCoeff.Sign() > 0 {
					// Leading term > 0 => as x_k -> -inf, x_k^maxDeg -> -inf => p_uni -> -inf
					negM := new(big.Int).Neg(ceilInt)
					xVal = new(big.Rat).SetInt(negM)
				} else {
					// Leading term < 0 => as x_k -> +inf, x_k^maxDeg -> +inf => p_uni -> -inf
					xVal = new(big.Rat).SetInt(ceilInt)
				}

				point[k] = xVal
				// Mechanically verify that P(point) < 0
				val := evaluateAt(p, point)
				if val.Sign() < 0 {
					witness := make(map[string]*big.Rat, n)
					for j, v := range vars {
						witness[v] = new(big.Rat).Set(point[j])
					}
					return witness
				}
			}
		}
	}

	return nil
}

// findCheapRefutation tests common small rational points to detect P(a) < 0.
func findCheapRefutation(p *MPoly, vars []string) map[string]*big.Rat {
	n := len(vars)
	testPoints := [][]*big.Rat{
		make([]*big.Rat, n), // Origin (0, ..., 0)
	}
	for i := 0; i < n; i++ {
		testPoints[0][i] = new(big.Rat)
	}

	// Basis unit vectors and small integer points
	for i := 0; i < n; i++ {
		for _, v := range []int64{1, -1, 2, -2} {
			pt := make([]*big.Rat, n)
			for j := 0; j < n; j++ {
				if j == i {
					pt[j] = big.NewRat(v, 1)
				} else {
					pt[j] = new(big.Rat)
				}
			}
			testPoints = append(testPoints, pt)
		}
	}

	// Diagonal points (1, 1, ...), (-1, -1, ...)
	for _, v := range []int64{1, -1} {
		pt := make([]*big.Rat, n)
		for j := 0; j < n; j++ {
			pt[j] = big.NewRat(v, 1)
		}
		testPoints = append(testPoints, pt)
	}

	for _, pt := range testPoints {
		val := evaluateAt(p, pt)
		if val.Sign() < 0 {
			res := make(map[string]*big.Rat, n)
			for i, vName := range vars {
				res[vName] = pt[i]
			}
			return res
		}
	}

	return nil
}

// tryGreedyQuadraticCompletion handles polynomials of total degree <= 2 by completing the square (exact LDL^T).
func tryGreedyQuadraticCompletion(p *MPoly, vars []string, order MonomialOrder) (*SOSResult, bool) {
	if p.Degree() > 2 {
		return nil, false
	}
	n := len(vars)
	numMonomials := 1 + n // Basis: [1, x_1, ..., x_n]

	basis := make([]Monomial, numMonomials)
	basis[0] = zeroMonomial(n)
	for i := 0; i < n; i++ {
		exps := make([]int, n)
		exps[i] = 1
		basis[i+1] = newMonomial(exps)
	}

	// Build Gram matrix Q of size (n+1) x (n+1)
	Q := make([][]*big.Rat, numMonomials)
	for i := range Q {
		Q[i] = make([]*big.Rat, numMonomials)
		for j := range Q[i] {
			Q[i][j] = new(big.Rat)
		}
	}

	// Map coefficient from p:
	// p = c_0 + sum c_i x_i + sum c_ii x_i^2 + sum_{i < j} c_ij x_i x_j
	for _, t := range p.Terms {
		deg := t.Mon.TotalDeg
		if deg == 0 {
			Q[0][0].Add(Q[0][0], t.Coeff)
		} else if deg == 1 {
			// Linear term: split evenly between Q[0][i+1] and Q[i+1][0]
			for idx, exp := range t.Mon.Exps {
				if exp == 1 {
					half := new(big.Rat).Mul(t.Coeff, big.NewRat(1, 2))
					Q[0][idx+1].Add(Q[0][idx+1], half)
					Q[idx+1][0].Add(Q[idx+1][0], half)
					break
				}
			}
		} else if deg == 2 {
			var first, second int
			first = -1
			second = -1
			for idx, exp := range t.Mon.Exps {
				if exp == 2 {
					first = idx
					second = idx
					break
				} else if exp == 1 {
					if first == -1 {
						first = idx
					} else {
						second = idx
					}
				}
			}
			if first == second {
				// Pure quadratic x_i^2: on the diagonal Q[first+1][first+1]
				Q[first+1][first+1].Add(Q[first+1][first+1], t.Coeff)
			} else {
				// Cross term x_i x_j: split evenly
				half := new(big.Rat).Mul(t.Coeff, big.NewRat(1, 2))
				Q[first+1][second+1].Add(Q[first+1][second+1], half)
				Q[second+1][first+1].Add(Q[second+1][first+1], half)
			}
		}
	}

	// Exact rational LDL^T decomposition
	ldl := exactRationalLDLTWithWitness(Q)
	if !ldl.IsPSD {
		return nil, false
	}

	terms := extractSOSTermsFromLDLT(ldl.L, ldl.D, basis, order)

	return &SOSResult{
		Status:   SOSCertified,
		Terms:    terms,
		Residual: NewZeroMPoly(n),
		Vars:     vars,
	}, true
}

// DecomposeSOS finds a rational sum-of-squares decomposition for multivariate polynomial p.
func DecomposeSOS(p *MPoly, vars []string, order MonomialOrder) (*SOSResult, error) {
	n := len(vars)
	if p.IsZero() {
		return &SOSResult{
			Status:   SOSCertified,
			Terms:    nil,
			Residual: NewZeroMPoly(n),
			Vars:     vars,
		}, nil
	}

	// 1. Cheap refutation: test small rational points
	if witness := findCheapRefutation(p, vars); witness != nil {
		return &SOSResult{
			Status:         SOSRefuted,
			Counterexample: witness,
			Residual:       p.Clone(),
			Vars:           vars,
		}, nil
	}

	// 2. Odd total degree refutation via Cauchy root bound
	if p.Degree()%2 != 0 {
		if witness := findOddDegreeRefutation(p, vars); witness != nil {
			return &SOSResult{
				Status:         SOSRefuted,
				Counterexample: witness,
				Residual:       p.Clone(),
				Vars:           vars,
			}, nil
		}
		// If unable to certify a negative witness mechanically, do not falsely claim refuted
		return &SOSResult{
			Status:   SOSNoCertificateFound,
			Residual: p.Clone(),
			Vars:     vars,
		}, nil
	}

	// 3. Fast path: Greedy completion of squares for quadratic forms (deg <= 2)
	if res, ok := tryGreedyQuadraticCompletion(p, vars, order); ok {
		return res, nil
	}

	// 4. Deterministic FSM-driven affine Gram exploration:
	// Half Newton Polytope + Reznick pruning + Sign-Symmetry blocking +
	// DSOS/SDSOS LP + Cutting-Plane LP + Partial Facial Reduction + Re-expansion Verification.
	return runSOSDecomposeFSM(p, vars, order)
}

// EvalSOSDecompose evaluates the sos_decompose builtin function.
// Syntax: sos_decompose(poly, [vars])
func EvalSOSDecompose(polyNode, varsNode Node, env *Env) (Node, error) {
	var varList *ListNode
	if varsNode != nil {
		vl, ok := varsNode.(*ListNode)
		if ok {
			varList = vl
		}
	}
	if varList == nil || len(varList.Elements) == 0 {
		freeVars := CollectFreeVariables(polyNode)
		if len(freeVars) == 0 {
			freeVars = []string{"x"}
		}
		varNodes := make([]Node, len(freeVars))
		for i, v := range freeVars {
			varNodes[i] = &VarNode{Name: v}
		}
		varList = NewList(varNodes)
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

	order := OrderGRevLex
	evalP, err := Eval(expandNode(polyNode))
	if err != nil {
		evalP = polyNode
	}
	mp, err := nodeToMPoly(evalP, varMap, len(varNames), order)
	if err != nil {
		return nil, err
	}

	res, err := DecomposeSOS(mp, varNames, order)
	if err != nil {
		return nil, err
	}

	// Return [status, terms]
	statusNode := &VarNode{Name: res.Status.String()}
	if res.Status == SOSCertified {
		var termNodes []Node
		for _, t := range res.Terms {
			qNode := mpolyToNode(t.Poly, varNames)
			termNodes = append(termNodes, NewList([]Node{
				NewRationalFromBigRat(t.Weight),
				qNode,
			}))
		}
		return NewList([]Node{statusNode, NewList(termNodes)}), nil
	} else if res.Status == SOSRefuted {
		var ceNodes []Node
		for _, vName := range varNames {
			if val, exists := res.Counterexample[vName]; exists {
				ceNodes = append(ceNodes, NewList([]Node{
					&VarNode{Name: vName},
					NewRationalFromBigRat(val),
				}))
			}
		}
		return NewList([]Node{statusNode, NewList(ceNodes)}), nil
	}

	return NewList([]Node{statusNode, NewList(nil)}), nil
}

func sortPolyTerms(terms []Term, order MonomialOrder) {
	sort.Slice(terms, func(i, j int) bool {
		return CompareMonomials(terms[i].Mon, terms[j].Mon, order) > 0
	})
}
