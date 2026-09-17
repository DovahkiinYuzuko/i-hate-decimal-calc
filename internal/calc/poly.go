package calc

import (
	"container/heap"
	"fmt"
	"math/big"
	"sort"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// ComparePolyMonomials compares two monomials m1 and m2 according to the specified monomial order.
// Returns 1 if m1 > m2, -1 if m1 < m2, and 0 if m1 == m2.
func ComparePolyMonomials(m1, m2 ast.Monomial, order ast.MonomialOrder) int {
	switch order {
	case ast.OrderGrevLex:
		// Graded Reverse Lexicographical order:
		// 1. Compare total degree (sum of exponents)
		deg1 := 0
		deg2 := 0
		for _, e := range m1.Exponents {
			deg1 += e
		}
		for _, e := range m2.Exponents {
			deg2 += e
		}
		if deg1 > deg2 {
			return 1
		}
		if deg1 < deg2 {
			return -1
		}
		// 2. If total degrees are equal, compare from right to left (last variable to first),
		// reverse comparison (smaller exponent is greater)
		for i := len(m1.Exponents) - 1; i >= 0; i-- {
			if m1.Exponents[i] < m2.Exponents[i] {
				return 1
			}
			if m1.Exponents[i] > m2.Exponents[i] {
				return -1
			}
		}
		return 0

	case ast.OrderLex:
		fallthrough
	default:
		// Lexicographical order: compare exponents from first variable to last
		for i := 0; i < len(m1.Exponents) && i < len(m2.Exponents); i++ {
			if m1.Exponents[i] > m2.Exponents[i] {
				return 1
			}
			if m1.Exponents[i] < m2.Exponents[i] {
				return -1
			}
		}
		return 0
	}
}

// areExponentsEqual checks if two exponent slices are identical.
func areExponentsEqual(e1, e2 []int) bool {
	if len(e1) != len(e2) {
		return false
	}
	for i, v := range e1 {
		if v != e2[i] {
			return false
		}
	}
	return true
}

// NewPolyNode creates a strictly canonical PolyNode from a list of terms.
// It combines duplicate terms, removes zero-coefficient terms, and sorts in descending order.
func NewPolyNode(vars []string, order ast.MonomialOrder, terms []ast.Monomial) *ast.PolyNode {
	vCopy := make([]string, len(vars))
	copy(vCopy, vars)

	if len(terms) == 0 {
		return &ast.PolyNode{
			Vars:  vCopy,
			Order: order,
			Terms: nil,
		}
	}

	// Filter out invalid or zero-coeff terms initially, and clone exponents
	validTerms := make([]ast.Monomial, 0, len(terms))
	for _, t := range terms {
		if t.Coeff != nil && t.Coeff.Sign() != 0 {
			expCopy := make([]int, len(vars))
			for i := 0; i < len(vars) && i < len(t.Exponents); i++ {
				expCopy[i] = t.Exponents[i]
			}
			validTerms = append(validTerms, ast.Monomial{
				Coeff:     new(big.Rat).Set(t.Coeff),
				Exponents: expCopy,
			})
		}
	}

	if len(validTerms) == 0 {
		return &ast.PolyNode{
			Vars:  vCopy,
			Order: order,
			Terms: nil,
		}
	}

	// Sort terms in descending order according to monomial order
	sort.Slice(validTerms, func(i, j int) bool {
		return ComparePolyMonomials(validTerms[i], validTerms[j], order) > 0
	})

	// Combine duplicate terms
	merged := make([]ast.Monomial, 0, len(validTerms))
	for _, t := range validTerms {
		if len(merged) > 0 && areExponentsEqual(merged[len(merged)-1].Exponents, t.Exponents) {
			merged[len(merged)-1].Coeff.Add(merged[len(merged)-1].Coeff, t.Coeff)
		} else {
			merged = append(merged, t)
		}
	}

	// Filter out any terms that cancelled to zero after addition
	finalTerms := make([]ast.Monomial, 0, len(merged))
	for _, t := range merged {
		if t.Coeff.Sign() != 0 {
			finalTerms = append(finalTerms, t)
		}
	}

	return &ast.PolyNode{
		Vars:  vCopy,
		Order: order,
		Terms: finalTerms,
	}
}

// AddPoly adds two canonical polynomials p1 and p2 using a two-pointer merge traversal (O(N+M)).
func AddPoly(p1, p2 *ast.PolyNode) *ast.PolyNode {
	if p1 == nil && p2 == nil {
		return nil
	}
	if p1 == nil || len(p1.Terms) == 0 {
		if p2 == nil {
			return nil
		}
		return p2.Clone()
	}
	if p2 == nil || len(p2.Terms) == 0 {
		return p1.Clone()
	}

	order := p1.Order
	vars := p1.Vars

	i, j := 0, 0
	resultTerms := make([]ast.Monomial, 0, len(p1.Terms)+len(p2.Terms))

	for i < len(p1.Terms) && j < len(p2.Terms) {
		cmp := ComparePolyMonomials(p1.Terms[i], p2.Terms[j], order)
		if cmp > 0 {
			resultTerms = append(resultTerms, p1.Terms[i].Clone())
			i++
		} else if cmp < 0 {
			resultTerms = append(resultTerms, p2.Terms[j].Clone())
			j++
		} else {
			sumCoeff := new(big.Rat).Add(p1.Terms[i].Coeff, p2.Terms[j].Coeff)
			if sumCoeff.Sign() != 0 {
				expCopy := make([]int, len(vars))
				copy(expCopy, p1.Terms[i].Exponents)
				resultTerms = append(resultTerms, ast.Monomial{
					Coeff:     sumCoeff,
					Exponents: expCopy,
				})
			}
			i++
			j++
		}
	}

	for i < len(p1.Terms) {
		resultTerms = append(resultTerms, p1.Terms[i].Clone())
		i++
	}
	for j < len(p2.Terms) {
		resultTerms = append(resultTerms, p2.Terms[j].Clone())
		j++
	}

	return &ast.PolyNode{
		Vars:  vars,
		Order: order,
		Terms: resultTerms,
	}
}

// SubPoly subtracts p2 from p1 (p1 - p2).
func SubPoly(p1, p2 *ast.PolyNode) *ast.PolyNode {
	if p2 == nil || len(p2.Terms) == 0 {
		if p1 == nil {
			return nil
		}
		return p1.Clone()
	}
	// Negate p2
	negTerms := make([]ast.Monomial, len(p2.Terms))
	for idx, t := range p2.Terms {
		negCoeff := new(big.Rat).Neg(t.Coeff)
		expCopy := make([]int, len(t.Exponents))
		copy(expCopy, t.Exponents)
		negTerms[idx] = ast.Monomial{Coeff: negCoeff, Exponents: expCopy}
	}
	negP2 := &ast.PolyNode{
		Vars:  p2.Vars,
		Order: p2.Order,
		Terms: negTerms,
	}
	return AddPoly(p1, negP2)
}

// -------------------------------------------------------------------------
// Monagan-Pearce / Johnson Heap-based Polynomial Multiplication
// -------------------------------------------------------------------------

type heapItem struct {
	i, j      int          // indices into p1.Terms and p2.Terms
	monomial  ast.Monomial // product monomial p1.Terms[i] * p2.Terms[j]
	order     ast.MonomialOrder
}

type polyHeap []*heapItem

func (h polyHeap) Len() int { return len(h) }
func (h polyHeap) Less(i, j int) bool {
	// Max-heap: highest monomial order comes first
	return ComparePolyMonomials(h[i].monomial, h[j].monomial, h[i].order) > 0
}
func (h polyHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *polyHeap) Push(x any)   { *h = append(*h, x.(*heapItem)) }
func (h *polyHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// MulPoly multiplies two canonical polynomials p1 and p2 using Monagan-Pearce heap merge.
func MulPoly(p1, p2 *ast.PolyNode) *ast.PolyNode {
	if p1 == nil || len(p1.Terms) == 0 || p2 == nil || len(p2.Terms) == 0 {
		vars := []string{}
		order := ast.OrderLex
		if p1 != nil {
			vars = p1.Vars
			order = p1.Order
		} else if p2 != nil {
			vars = p2.Vars
			order = p2.Order
		}
		return &ast.PolyNode{
			Vars:  vars,
			Order: order,
			Terms: nil,
		}
	}

	// Ensure p1 has fewer or equal terms than p2 for optimal heap size
	A, B := p1, p2
	if len(A.Terms) > len(B.Terms) {
		A, B = B, A
	}

	numVars := len(A.Vars)
	order := A.Order

	multMonomial := func(i, j int) ast.Monomial {
		t1 := A.Terms[i]
		t2 := B.Terms[j]
		coeff := new(big.Rat).Mul(t1.Coeff, t2.Coeff)
		exps := make([]int, numVars)
		for k := 0; k < numVars; k++ {
			exps[k] = t1.Exponents[k] + t2.Exponents[k]
		}
		return ast.Monomial{Coeff: coeff, Exponents: exps}
	}

	h := &polyHeap{}
	heap.Init(h)

	// Insert (i, 0) for all i in A
	for i := 0; i < len(A.Terms); i++ {
		heap.Push(h, &heapItem{
			i:        i,
			j:        0,
			monomial: multMonomial(i, 0),
			order:    order,
		})
	}

	resultTerms := make([]ast.Monomial, 0, len(A.Terms)*len(B.Terms))

	for h.Len() > 0 {
		top := heap.Pop(h).(*heapItem)
		currMon := top.monomial

		// Push next element from row top.i if available
		if top.j+1 < len(B.Terms) {
			heap.Push(h, &heapItem{
				i:        top.i,
				j:        top.j + 1,
				monomial: multMonomial(top.i, top.j+1),
				order:    order,
			})
		}

		// Accumulate matching terms with identical exponent vectors
		accCoeff := new(big.Rat).Set(currMon.Coeff)
		for h.Len() > 0 && areExponentsEqual((*h)[0].monomial.Exponents, currMon.Exponents) {
			nextTop := heap.Pop(h).(*heapItem)
			accCoeff.Add(accCoeff, nextTop.monomial.Coeff)
			if nextTop.j+1 < len(B.Terms) {
				heap.Push(h, &heapItem{
					i:        nextTop.i,
					j:        nextTop.j + 1,
					monomial: multMonomial(nextTop.i, nextTop.j+1),
					order:    order,
				})
			}
		}

		if accCoeff.Sign() != 0 {
			resultTerms = append(resultTerms, ast.Monomial{
				Coeff:     accCoeff,
				Exponents: currMon.Exponents,
			})
		}
	}

	return &ast.PolyNode{
		Vars:  A.Vars,
		Order: order,
		Terms: resultTerms,
	}
}

// -------------------------------------------------------------------------
// Polynomial Pseudo-Division & DivRem
// -------------------------------------------------------------------------

// LeadingTerm returns the leading monomial of the polynomial (terms[0]). Returns false if polynomial is zero.
func LeadingTerm(p *ast.PolyNode) (ast.Monomial, bool) {
	if p == nil || len(p.Terms) == 0 {
		return ast.Monomial{}, false
	}
	return p.Terms[0], true
}

// DegreeInVar returns the maximum degree of the specified variable name in the polynomial.
func DegreeInVar(p *ast.PolyNode, varName string) int {
	if p == nil || len(p.Terms) == 0 {
		return -1
	}
	varIdx := -1
	for idx, v := range p.Vars {
		if v == varName {
			varIdx = idx
			break
		}
	}
	if varIdx == -1 {
		return 0
	}
	maxDeg := 0
	for _, t := range p.Terms {
		if t.Exponents[varIdx] > maxDeg {
			maxDeg = t.Exponents[varIdx]
		}
	}
	return maxDeg
}

// LeadingCoeffInVar returns the leading coefficient of p with respect to varName.
func LeadingCoeffInVar(p *ast.PolyNode, varName string) *ast.PolyNode {
	if p == nil || len(p.Terms) == 0 {
		return NewPolyNode(p.Vars, p.Order, nil)
	}
	deg := DegreeInVar(p, varName)
	if deg < 0 {
		return NewPolyNode(p.Vars, p.Order, nil)
	}
	varIdx := -1
	for idx, v := range p.Vars {
		if v == varName {
			varIdx = idx
			break
		}
	}

	var coeffTerms []ast.Monomial
	for _, t := range p.Terms {
		if t.Exponents[varIdx] == deg {
			exps := make([]int, len(p.Vars))
			copy(exps, t.Exponents)
			exps[varIdx] = 0 // var degree is factored out
			coeffTerms = append(coeffTerms, ast.Monomial{
				Coeff:     new(big.Rat).Set(t.Coeff),
				Exponents: exps,
			})
		}
	}
	return NewPolyNode(p.Vars, p.Order, coeffTerms)
}

// PseudoDivRem computes the pseudo-division of dividend by divisor with respect to mainVar:
// premFactor * dividend = quotient * divisor + remainder, where deg(remainder) < deg(divisor).
func PseudoDivRem(dividend, divisor *ast.PolyNode, mainVar string) (quotient, remainder *ast.PolyNode, premFactor *big.Rat, err error) {
	if divisor == nil || len(divisor.Terms) == 0 {
		return nil, nil, nil, fmt.Errorf("polynomial division by zero")
	}

	degDivisor := DegreeInVar(divisor, mainVar)
	degDividend := DegreeInVar(dividend, mainVar)

	if degDividend < degDivisor {
		zeroQ := NewPolyNode(dividend.Vars, dividend.Order, nil)
		return zeroQ, dividend.Clone(), big.NewRat(1, 1), nil
	}

	// In single-variable or scalar leading coefficient cases:
	lcDivisor := LeadingCoeffInVar(divisor, mainVar)
	if len(lcDivisor.Terms) == 0 {
		return nil, nil, nil, fmt.Errorf("divisor leading coefficient is zero")
	}

	// delta = degDividend - degDivisor + 1
	delta := degDividend - degDivisor + 1
	varIdx := -1
	for idx, v := range dividend.Vars {
		if v == mainVar {
			varIdx = idx
			break
		}
	}

	// Multiplication factor: lc(divisor)^delta
	scalarLC := lcDivisor.Terms[0].Coeff
	factor := big.NewRat(1, 1)
	for i := 0; i < delta; i++ {
		factor.Mul(factor, scalarLC)
	}

	// Scale dividend
	scaleTerm := ast.Monomial{Coeff: factor, Exponents: make([]int, len(dividend.Vars))}
	scalePoly := NewPolyNode(dividend.Vars, dividend.Order, []ast.Monomial{scaleTerm})
	rem := MulPoly(dividend, scalePoly)
	quo := NewPolyNode(dividend.Vars, dividend.Order, nil)

	for DegreeInVar(rem, mainVar) >= degDivisor && len(rem.Terms) > 0 {
		remDeg := DegreeInVar(rem, mainVar)
		lcRem := LeadingCoeffInVar(rem, mainVar)
		if len(lcRem.Terms) == 0 {
			break
		}

		// termQuo = (lcRem / lcDivisor) * mainVar^(remDeg - degDivisor)
		qCoeff := new(big.Rat).Quo(lcRem.Terms[0].Coeff, scalarLC)
		qExp := make([]int, len(dividend.Vars))
		qExp[varIdx] = remDeg - degDivisor

		termPoly := NewPolyNode(dividend.Vars, dividend.Order, []ast.Monomial{
			{Coeff: qCoeff, Exponents: qExp},
		})

		quo = AddPoly(quo, termPoly)
		subPart := MulPoly(termPoly, divisor)
		rem = SubPoly(rem, subPart)
	}

	return quo, rem, factor, nil
}

// -------------------------------------------------------------------------
// AST <-> PolyNode Conversion
// -------------------------------------------------------------------------

// NodeToPoly converts an AST expression into a canonical PolyNode.
func NodeToPoly(node ast.Node, vars []string, order ast.MonomialOrder) (*ast.PolyNode, error) {
	if node == nil {
		return NewPolyNode(vars, order, nil), nil
	}

	varMap := make(map[string]int)
	for idx, v := range vars {
		varMap[v] = idx
	}

	switch v := node.(type) {
	case *ast.RationalNode:
		if v.Val.Sign() == 0 {
			return NewPolyNode(vars, order, nil), nil
		}
		exps := make([]int, len(vars))
		return NewPolyNode(vars, order, []ast.Monomial{
			{Coeff: new(big.Rat).Set(v.Val), Exponents: exps},
		}), nil

	case *ast.VarNode:
		idx, found := varMap[v.Name]
		if !found {
			return nil, fmt.Errorf("variable %s not present in polynomial variable list %v", v.Name, vars)
		}
		exps := make([]int, len(vars))
		exps[idx] = 1
		return NewPolyNode(vars, order, []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: exps},
		}), nil

	case *ast.AddNode:
		var acc *ast.PolyNode
		for _, child := range v.Terms {
			childPoly, err := NodeToPoly(child, vars, order)
			if err != nil {
				return nil, err
			}
			acc = AddPoly(acc, childPoly)
		}
		return acc, nil

	case *ast.MulNode:
		var acc *ast.PolyNode
		for i, child := range v.Factors {
			childPoly, err := NodeToPoly(child, vars, order)
			if err != nil {
				return nil, err
			}
			if i == 0 {
				acc = childPoly
			} else {
				acc = MulPoly(acc, childPoly)
			}
		}
		return acc, nil

	case *ast.UnaryOpNode:
		if v.Op == "-" {
			childPoly, err := NodeToPoly(v.Expr, vars, order)
			if err != nil {
				return nil, err
			}
			return SubPoly(NewPolyNode(vars, order, nil), childPoly), nil
		} else if v.Op == "+" {
			return NodeToPoly(v.Expr, vars, order)
		}
		return nil, fmt.Errorf("unsupported unary operator %s in polynomial", v.Op)

	case *ast.PowNode:
		basePoly, err := NodeToPoly(v.Base, vars, order)
		if err != nil {
			return nil, err
		}
		expRat, ok := v.Exp.(*ast.RationalNode)
		if !ok || !expRat.Val.IsInt() || expRat.Val.Sign() < 0 {
			return nil, fmt.Errorf("polynomial exponent must be a non-negative integer, got %s", v.Exp.String())
		}
		expInt := expRat.Val.Num().Int64()
		if expInt > 1000 {
			return nil, fmt.Errorf("polynomial exponent %d exceeds reasonable limit", expInt)
		}

		// Binary exponentiation
		res := NewPolyNode(vars, order, []ast.Monomial{
			{Coeff: big.NewRat(1, 1), Exponents: make([]int, len(vars))},
		})
		cur := basePoly
		for expInt > 0 {
			if expInt%2 == 1 {
				res = MulPoly(res, cur)
			}
			if expInt > 1 {
				cur = MulPoly(cur, cur)
			}
			expInt /= 2
		}
		return res, nil

	case *ast.PolyNode:
		return v.Clone(), nil

	default:
		return nil, fmt.Errorf("unsupported node type %T in polynomial conversion", node)
	}
}

// PolyToNode converts a canonical PolyNode into an AST expression tree.
func PolyToNode(poly *ast.PolyNode) ast.Node {
	if poly == nil || len(poly.Terms) == 0 {
		r, _ := ast.NewRational(0, 1)
		return r
	}

	var addTerms []ast.Node
	for _, t := range poly.Terms {
		var factors []ast.Node

		// Coefficient factor
		absRat := new(big.Rat).Abs(t.Coeff)
		isOne := absRat.Cmp(big.NewRat(1, 1)) == 0

		isConst := true
		for _, exp := range t.Exponents {
			if exp != 0 {
				isConst = false
				break
			}
		}

		if isConst || !isOne {
			factors = append(factors, ast.NewRationalFromBigRat(absRat))
		}

		// Variable factors
		for vIdx, exp := range t.Exponents {
			if exp == 0 {
				continue
			}
			vName := poly.Vars[vIdx]
			vNode := ast.NewVar(vName)
			if exp == 1 {
				factors = append(factors, vNode)
			} else {
				eNode, _ := ast.NewRational(int64(exp), 1)
				powNode, _ := ast.NewPow(vNode, eNode)
				factors = append(factors, powNode)
			}
		}

		var termNode ast.Node
		if len(factors) == 0 {
			r, _ := ast.NewRational(1, 1)
			termNode = r
		} else if len(factors) == 1 {
			termNode = factors[0]
		} else {
			termNode = ast.NewMul(factors)
		}

		if t.Coeff.Sign() < 0 {
			negNode, _ := ast.NewUnaryOp("-", termNode)
			termNode = negNode
		}
		addTerms = append(addTerms, termNode)
	}

	if len(addTerms) == 1 {
		return addTerms[0]
	}
	return ast.NewAdd(addTerms)
}
