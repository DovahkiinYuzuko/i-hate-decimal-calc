package calc

import (
	"fmt"
	"math/big"
	"sort"
)

// Factor decomposes an integer into prime factors or a univariate rational polynomial
// into irreducible factors over the rationals.
func Factor(expr Node, varName ...string) (Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("factor: nil expression")
	}

	simplified, err := Eval(expr)
	if err != nil {
		return nil, err
	}

	// 1. Integer or Rational factorization
	if rat, ok := simplified.(*RationalNode); ok {
		if rat.Val.IsInt() {
			return factorInteger(rat.Val.Num())
		}
		// Fraction: factor numerator and denominator separately
		numNode, err := factorInteger(rat.Val.Num())
		if err != nil {
			return nil, err
		}
		denomNode, err := factorInteger(rat.Val.Denom())
		if err != nil {
			return nil, err
		}
		inv, err := NewPow(denomNode, mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		return NewMul([]Node{numNode, inv}), nil
	}

	// 2. Polynomial factorization
	v := ""
	if len(varName) > 0 && varName[0] != "" {
		v = varName[0]
	} else {
		vars := collectVariables(simplified)
		if len(vars) == 0 {
			// No variables, return simplified
			return simplified, nil
		}
		// Pick first variable, preferring 'x', 'y', 'z', 't'
		v = selectMainVariable(vars)
	}

	return factorPolynomial(simplified, v)
}

// -------------------------------------------------------------------------
// Integer Prime Factorization (Trial Division with 6k +/- 1 Wheel)
// -------------------------------------------------------------------------

type primeFactor struct {
	prime *big.Int
	exp   int
}

func factorInteger(n *big.Int) (Node, error) {
	if n.Sign() == 0 {
		return mustRational(0, 1), nil
	}

	isNeg := n.Sign() < 0
	val := new(big.Int).Abs(n)

	if val.Cmp(big.NewInt(1)) == 0 {
		if isNeg {
			return mustRational(-1, 1), nil
		}
		return mustRational(1, 1), nil
	}

	var factors []primeFactor
	two := big.NewInt(2)
	three := big.NewInt(3)
	rem := new(big.Int)

	// Factor 2
	count2 := 0
	for {
		rem.Mod(val, two)
		if rem.Sign() == 0 {
			count2++
			val.Div(val, two)
		} else {
			break
		}
	}
	if count2 > 0 {
		factors = append(factors, primeFactor{prime: big.NewInt(2), exp: count2})
	}

	// Factor 3
	count3 := 0
	for {
		rem.Mod(val, three)
		if rem.Sign() == 0 {
			count3++
			val.Div(val, three)
		} else {
			break
		}
	}
	if count3 > 0 {
		factors = append(factors, primeFactor{prime: big.NewInt(3), exp: count3})
	}

	// If val is already prime, stop trial division early!
	if val.Cmp(big.NewInt(1)) > 0 && val.ProbablyPrime(20) {
		factors = append(factors, primeFactor{prime: new(big.Int).Set(val), exp: 1})
		val.SetInt64(1)
	}

	// Wheel 6k +/- 1: d = 5, 7, 11, 13, 17, 19, 23, 25...
	d := big.NewInt(5)
	step := big.NewInt(2)
	d2 := new(big.Int).Mul(d, d)

	for d2.Cmp(val) <= 0 {
		if val.ProbablyPrime(20) {
			break
		}

		count := 0
		for {
			rem.Mod(val, d)
			if rem.Sign() == 0 {
				count++
				val.Div(val, d)
			} else {
				break
			}
		}
		if count > 0 {
			factors = append(factors, primeFactor{prime: new(big.Int).Set(d), exp: count})
			d2.Mul(d, d)
		}

		d.Add(d, step)
		if step.Cmp(two) == 0 {
			step.SetInt64(4)
		} else {
			step.SetInt64(2)
		}
		d2.Mul(d, d)
	}

	if val.Cmp(big.NewInt(1)) > 0 {
		factors = append(factors, primeFactor{prime: new(big.Int).Set(val), exp: 1})
	}

	// Build AST
	var factorNodes []Node
	if isNeg {
		factorNodes = append(factorNodes, mustRational(-1, 1))
	}

	for _, pf := range factors {
		pNode := &RationalNode{Val: new(big.Rat).SetInt(pf.prime)}
		if pf.exp == 1 {
			factorNodes = append(factorNodes, pNode)
		} else {
			expNode := &RationalNode{Val: big.NewRat(int64(pf.exp), 1)}
			powNode, err := NewPow(pNode, expNode)
			if err != nil {
				return nil, err
			}
			factorNodes = append(factorNodes, powNode)
		}
	}

	if len(factorNodes) == 1 {
		return factorNodes[0], nil
	}
	return NewMul(factorNodes), nil
}

// -------------------------------------------------------------------------
// Univariate Polynomial Factorization over Rationals
// -------------------------------------------------------------------------

type polyFactorEntry struct {
	factor Node
	exp    int
}

func factorPolynomial(expr Node, varName string) (Node, error) {
	// Expand the polynomial first
	expanded := expandNode(expr)

	// Extract coefficients
	coeffsMap, err := extractPolyCoeffs(expanded, varName)
	if err != nil {
		return nil, err
	}

	maxDeg := 0
	for d := range coeffsMap {
		if d > maxDeg {
			maxDeg = d
		}
	}

	if maxDeg <= 1 {
		// Degree 0 or 1 is already irreducible or trivial
		return expr, nil
	}

	// Check that all coefficients are rational numbers
	intCoeffs := make(map[int]*big.Int)
	var denoms []*big.Int

	for d, cNode := range coeffsMap {
		cRat, ok := cNode.(*RationalNode)
		if !ok {
			return nil, fmt.Errorf("factor: non-rational coefficient in degree %d: %s", d, cNode.String())
		}
		denoms = append(denoms, cRat.Val.Denom())
	}

	// 1. Extract common variable factors x^k
	minDeg := maxDeg
	for d := range coeffsMap {
		if d < minDeg {
			minDeg = d
		}
	}

	// 2. Compute LCM of all denominators
	commonLcm := big.NewInt(1)
	for _, denom := range denoms {
		commonLcm = lcmInt(commonLcm, denom)
	}

	// Convert to integer coefficients: coeff_scaled = coeff * commonLcm
	for d, cNode := range coeffsMap {
		cRat := cNode.(*RationalNode)
		// num * (lcm / denom)
		mult := new(big.Int).Div(commonLcm, cRat.Val.Denom())
		scaled := new(big.Int).Mul(cRat.Val.Num(), mult)
		intCoeffs[d-minDeg] = scaled
	}

	polyDeg := maxDeg - minDeg

	// 3. Compute GCD of all integer coefficients (Content extraction)
	commonGcd := new(big.Int).Set(intCoeffs[0])
	for i := 1; i <= polyDeg; i++ {
		if c, ok := intCoeffs[i]; ok {
			commonGcd.GCD(nil, nil, commonGcd, c)
		}
	}
	commonGcd.Abs(commonGcd)

	// Keep leading coefficient positive
	leadCoeff := intCoeffs[polyDeg]
	contentSign := big.NewInt(1)
	if leadCoeff.Sign() < 0 {
		contentSign.SetInt64(-1)
	}

	// Content = (contentSign * commonGcd) / commonLcm
	contentNum := new(big.Int).Mul(contentSign, commonGcd)
	contentRat := new(big.Rat).SetFrac(contentNum, commonLcm)

	// Primitive polynomial: P[i] = intCoeffs[i] / (contentSign * commonGcd)
	primDiv := new(big.Int).Mul(contentSign, commonGcd)
	poly := make([]*big.Int, polyDeg+1)
	for i := 0; i <= polyDeg; i++ {
		if c, ok := intCoeffs[i]; ok {
			poly[i] = new(big.Int).Div(c, primDiv)
		} else {
			poly[i] = big.NewInt(0)
		}
	}

	// 4. Factor primitive polynomial via Rational Root Theorem and Synthetic Division
	var factors []polyFactorEntry

	// Find rational roots
	currentPoly := poly
	for len(currentPoly) > 2 { // degree >= 2
		a0 := currentPoly[0]

		if a0.Sign() == 0 {
			// This shouldn't happen because minDeg was already extracted
			break
		}

		roots := findRationalRoots(currentPoly)
		if len(roots) == 0 {
			// No rational roots
			break
		}

		rootFound := false
		for _, r := range roots {
			p := r.num
			q := r.denom
			// Check if (q*x - p) divides currentPoly
			quot, ok := syntheticDivide(currentPoly, p, q)
			if ok {
				// (q*x - p) is a factor!
				factorNode := buildLinearFactorNode(varName, p, q)
				addFactorEntry(&factors, factorNode)
				currentPoly = quot
				rootFound = true
				break
			}
		}

		if !rootFound {
			break
		}
	}

	// If remaining polynomial has degree >= 2, check if quadratic can be factored
	if len(currentPoly) == 3 {
		// A*x^2 + B*x + C
		A := currentPoly[2]
		B := currentPoly[1]
		C := currentPoly[0]

		// D = B^2 - 4*A*C
		bSq := new(big.Int).Mul(B, B)
		fourAC := new(big.Int).Mul(big.NewInt(4), new(big.Int).Mul(A, C))
		disc := new(big.Int).Sub(bSq, fourAC)

		if disc.Sign() >= 0 {
			sqrtD := new(big.Int).Sqrt(disc)
			if new(big.Int).Mul(sqrtD, sqrtD).Cmp(disc) == 0 {
				// Discriminant is a perfect square! Factors exist.
				// Roots: (-B +/- sqrtD) / (2*A)
				twoA := new(big.Int).Mul(big.NewInt(2), A)

				r1Num := new(big.Int).Add(new(big.Int).Neg(B), sqrtD)
				g1 := new(big.Int).GCD(nil, nil, r1Num, twoA)
				p1 := new(big.Int).Div(r1Num, g1)
				q1 := new(big.Int).Div(twoA, g1)
				if q1.Sign() < 0 {
					q1.Neg(q1)
					p1.Neg(p1)
				}

				r2Num := new(big.Int).Sub(new(big.Int).Neg(B), sqrtD)
				g2 := new(big.Int).GCD(nil, nil, r2Num, twoA)
				p2 := new(big.Int).Div(r2Num, g2)
				q2 := new(big.Int).Div(twoA, g2)
				if q2.Sign() < 0 {
					q2.Neg(q2)
					p2.Neg(p2)
				}

				f1 := buildLinearFactorNode(varName, p1, q1)
				f2 := buildLinearFactorNode(varName, p2, q2)
				addFactorEntry(&factors, f1)
				addFactorEntry(&factors, f2)
				currentPoly = []*big.Int{big.NewInt(1)}
			}
		}
	}

	// Remaining irreducible polynomial factor
	if len(currentPoly) > 1 {
		remNode := polyToNode(currentPoly, varName)
		addFactorEntry(&factors, remNode)
	}

	// 5. Assemble all factors into final AST
	var resultNodes []Node

	// Content constant (if not 1)
	if contentRat.Cmp(big.NewRat(1, 1)) != 0 {
		resultNodes = append(resultNodes, &RationalNode{Val: contentRat})
	}

	// Variable monomial factor x^minDeg
	if minDeg > 0 {
		xNode := &VarNode{Name: varName}
		if minDeg == 1 {
			resultNodes = append(resultNodes, xNode)
		} else {
			expNode := &RationalNode{Val: big.NewRat(int64(minDeg), 1)}
			powNode, _ := NewPow(xNode, expNode)
			resultNodes = append(resultNodes, powNode)
		}
	}

	// Polynomial factors with exponents
	for _, fe := range factors {
		if fe.exp == 1 {
			resultNodes = append(resultNodes, fe.factor)
		} else {
			expNode := &RationalNode{Val: big.NewRat(int64(fe.exp), 1)}
			powNode, _ := NewPow(fe.factor, expNode)
			resultNodes = append(resultNodes, powNode)
		}
	}

	if len(resultNodes) == 0 {
		return mustRational(1, 1), nil
	}
	if len(resultNodes) == 1 {
		return resultNodes[0], nil
	}
	return NewMul(resultNodes), nil
}

// -------------------------------------------------------------------------
// Helpers: Rational Root Theorem & Synthetic Division
// -------------------------------------------------------------------------

type rationalRootCandidate struct {
	num   *big.Int
	denom *big.Int
}

func findRationalRoots(poly []*big.Int) []rationalRootCandidate {
	deg := len(poly) - 1
	a0 := poly[0]
	an := poly[deg]

	divsP := integerDivisors(new(big.Int).Abs(a0))
	divsQ := integerDivisors(new(big.Int).Abs(an))

	var candidates []rationalRootCandidate
	seen := make(map[string]bool)

	for _, p := range divsP {
		for _, q := range divsQ {
			// Try +p/q and -p/q
			for _, sign := range []int64{1, -1} {
				num := new(big.Int).Mul(p, big.NewInt(sign))
				g := new(big.Int).GCD(nil, nil, num, q)
				reducedNum := new(big.Int).Div(num, g)
				reducedDenom := new(big.Int).Div(q, g)

				key := fmt.Sprintf("%s/%s", reducedNum.String(), reducedDenom.String())
				if !seen[key] {
					seen[key] = true
					// Evaluate polynomial at root
					if evalPolyInt(poly, reducedNum, reducedDenom).Sign() == 0 {
						candidates = append(candidates, rationalRootCandidate{
							num:   reducedNum,
							denom: reducedDenom,
						})
					}
				}
			}
		}
	}

	return candidates
}

// evalPolyInt computes P(num/denom) * denom^deg to test if P(num/denom) == 0 using exact integers.
func evalPolyInt(poly []*big.Int, num, denom *big.Int) *big.Int {
	deg := len(poly) - 1
	sum := big.NewInt(0)
	for i := 0; i <= deg; i++ {
		pPow := new(big.Int).Exp(num, big.NewInt(int64(i)), nil)
		qPow := new(big.Int).Exp(denom, big.NewInt(int64(deg-i)), nil)
		term := new(big.Int).Mul(poly[i], pPow)
		term.Mul(term, qPow)
		sum.Add(sum, term)
	}

	return sum
}

// syntheticDivide divides polynomial poly by (q*x - p) using exact integer arithmetic.
// Returns (quotient, true) if remainder is zero, or (nil, false) otherwise.
func syntheticDivide(poly []*big.Int, p, q *big.Int) ([]*big.Int, bool) {
	deg := len(poly) - 1
	if deg < 1 {
		return nil, false
	}

	// Horner division by (x - p/q):
	// b_n = a_n
	// b_{k-1} = a_k + (p/q) * b_k
	// In integer arithmetic with factor (q*x - p):
	quot := make([]*big.Int, deg)
	carry := big.NewInt(0)

	remVal := new(big.Int)
	remVal.Mod(poly[deg], q)
	if remVal.Sign() != 0 {
		return nil, false
	}
	quot[deg-1] = new(big.Int).Div(poly[deg], q)
	carry.Mul(quot[deg-1], p)

	for i := deg - 1; i >= 1; i-- {
		curr := new(big.Int).Add(poly[i], carry)
		remVal.Mod(curr, q)
		if remVal.Sign() != 0 {
			return nil, false
		}
		quot[i-1] = new(big.Int).Div(curr, q)
		carry.Mul(quot[i-1], p)
	}

	// Remainder
	rem := new(big.Int).Add(poly[0], carry)
	if rem.Sign() != 0 {
		return nil, false
	}

	return quot, true
}

func buildLinearFactorNode(varName string, p, q *big.Int) Node {
	xNode := &VarNode{Name: varName}
	var qxNode Node
	if q.Cmp(big.NewInt(1)) == 0 {
		qxNode = xNode
	} else {
		qRat := &RationalNode{Val: new(big.Rat).SetInt(q)}
		qxNode = NewMul([]Node{qRat, xNode})
	}

	negP := new(big.Int).Neg(p)
	pRat := &RationalNode{Val: new(big.Rat).SetInt(negP)}

	if pRat.Val.Sign() == 0 {
		return qxNode
	}
	return NewAdd([]Node{qxNode, pRat})
}

func polyToNode(poly []*big.Int, varName string) Node {
	var terms []Node
	for deg := len(poly) - 1; deg >= 0; deg-- {
		coeff := poly[deg]
		if coeff.Sign() == 0 {
			continue
		}

		cRat := &RationalNode{Val: new(big.Rat).SetInt(coeff)}
		switch deg {
		case 0:
			terms = append(terms, cRat)
		case 1:
			xNode := &VarNode{Name: varName}
			if coeff.Cmp(big.NewInt(1)) == 0 {
				terms = append(terms, xNode)
			} else if coeff.Cmp(big.NewInt(-1)) == 0 {
				terms = append(terms, NewMul([]Node{mustRational(-1, 1), xNode}))
			} else {
				terms = append(terms, NewMul([]Node{cRat, xNode}))
			}
		default:
			xNode := &VarNode{Name: varName}
			powNode, _ := NewPow(xNode, mustRational(int64(deg), 1))
			if coeff.Cmp(big.NewInt(1)) == 0 {
				terms = append(terms, powNode)
			} else if coeff.Cmp(big.NewInt(-1)) == 0 {
				terms = append(terms, NewMul([]Node{mustRational(-1, 1), powNode}))
			} else {
				terms = append(terms, NewMul([]Node{cRat, powNode}))
			}
		}
	}

	if len(terms) == 0 {
		return mustRational(0, 1)
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return NewAdd(terms)
}

func addFactorEntry(factors *[]polyFactorEntry, f Node) {
	fStr := f.String()
	for i, entry := range *factors {
		if entry.factor.String() == fStr {
			(*factors)[i].exp++
			return
		}
	}
	*factors = append(*factors, polyFactorEntry{factor: f, exp: 1})
}

func integerDivisors(n *big.Int) []*big.Int {
	if n.Sign() == 0 {
		return []*big.Int{big.NewInt(1)}
	}

	var divs []*big.Int
	one := big.NewInt(1)
	val := new(big.Int).Abs(n)

	i := big.NewInt(1)
	i2 := new(big.Int)
	rem := new(big.Int)

	for {
		i2.Mul(i, i)
		if i2.Cmp(val) > 0 {
			break
		}
		rem.Mod(val, i)
		if rem.Sign() == 0 {
			divs = append(divs, new(big.Int).Set(i))
			other := new(big.Int).Div(val, i)
			if other.Cmp(i) != 0 {
				divs = append(divs, other)
			}
		}
		i.Add(i, one)
	}

	sort.Slice(divs, func(a, b int) bool {
		return divs[a].Cmp(divs[b]) < 0
	})

	return divs
}

func lcmInt(a, b *big.Int) *big.Int {
	gcd := new(big.Int).GCD(nil, nil, a, b)
	if gcd.Sign() == 0 {
		return big.NewInt(0)
	}
	res := new(big.Int).Mul(a, b)
	res.Div(res, gcd)
	return res.Abs(res)
}

func collectVariables(n Node) []string {
	seen := make(map[string]bool)
	var vars []string

	var walk func(node Node)
	walk = func(node Node) {
		if node == nil {
			return
		}
		switch v := node.(type) {
		case *VarNode:
			if !seen[v.Name] {
				seen[v.Name] = true
				vars = append(vars, v.Name)
			}
		case *AddNode:
			for _, t := range v.Terms {
				walk(t)
			}
		case *MulNode:
			for _, f := range v.Factors {
				walk(f)
			}
		case *PowNode:
			walk(v.Base)
			walk(v.Exp)
		case *FuncNode:
			for _, a := range v.Args {
				walk(a)
			}
		}
	}

	walk(n)
	sort.Strings(vars)
	return vars
}

func selectMainVariable(vars []string) string {
	for _, pref := range []string{"x", "y", "z", "t", "a", "b", "c", "n", "k"} {
		for _, v := range vars {
			if v == pref {
				return v
			}
		}
	}
	return vars[0]
}
