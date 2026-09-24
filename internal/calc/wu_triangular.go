package calc

import (
	"fmt"
	"sort"
)

// TriangularSet represents an ascending chain A = (A_1, ..., A_r)
// where mvar(A_1) < mvar(A_2) < ... < mvar(A_r).
type TriangularSet struct {
	Elements []Node
	MainVars []string
	Initials []Node
	Degrees  []int
}

// ZeroDecompositionTree represents a branch tree for Wu's zero decomposition:
// Zero(HYP) = Union (Zero(Chain_i) \ Zero(Initials_i))
type ZeroDecompositionTree struct {
	Chain              *TriangularSet
	SaturationInitials []Node
	DegenerateBranches []*ZeroDecompositionTree
}

// LinearSubstitutionFastPath extracts linear equations of the form c * x + Rest == 0 (deg_x = 1),
// solves for x = -Rest / c, and substitutes them throughout the remaining polynomials.
func LinearSubstitutionFastPath(hypotheses []Node, order []string) ([]Node, map[string]Node) {
	subs := make(map[string]Node)
	remaining := make([]Node, 0, len(hypotheses))

	// Copy and expand
	for _, h := range hypotheses {
		evalH, err := Eval(expandNode(h))
		if err != nil {
			evalH = h
		}
		if !isZero(evalH) {
			remaining = append(remaining, evalH)
		}
	}

	progress := true
	for progress {
		progress = false

		for i := 0; i < len(remaining); i++ {
			p := remaining[i]
			mvar, deg, init := IdentifyMainVarAndRank(p, order)
			if mvar == "" || deg != 1 {
				continue
			}

			// We have a polynomial linear in mvar: I * mvar + C = 0
			// Check if initial I contains only lower variables
			if containsVar(init, mvar) {
				continue
			}

			// Extract rest = P - I * mvar
			poly, ok := extractPoly(p, mvar)
			if !ok || poly.degree() != 1 {
				continue
			}

			c0 := poly.coeffs[0] // constant term w.r.t mvar
			c1 := poly.coeffs[1] // linear term w.r.t mvar (initial)

			// mvar = -c0 / c1
			negC0, _ := simplifyUnaryOp("-", c0)
			invC1, err := simplifyPow(c1, mustRational(-1, 1))
			if err != nil {
				continue
			}
			subExpr, err := simplifyMul([]Node{negC0, invC1})
			if err != nil {
				continue
			}
			evalSub, err := Eval(expandNode(subExpr))
			if err != nil {
				evalSub = subExpr
			}

			subs[mvar] = evalSub
			progress = true

			// Remove current polynomial from remaining
			remaining = append(remaining[:i], remaining[i+1:]...)

			// Substitute mvar = evalSub in all remaining polynomials
			for j := range remaining {
				subbed := Substitute(remaining[j], mvar, evalSub)
				evaled, err := Eval(expandNode(subbed))
				if err != nil {
					evaled = subbed
				}
				remaining[j] = evaled
			}

			break
		}
	}

	// Filter out zeros
	var nonZeroRemaining []Node
	for _, r := range remaining {
		if !isZero(r) {
			nonZeroRemaining = append(nonZeroRemaining, r)
		}
	}

	return nonZeroRemaining, subs
}

// BuildAscendingChain constructs a triangular ascending chain from a set of hypotheses.
func BuildAscendingChain(hypotheses []Node, order []string) (*TriangularSet, []Node, error) {
	// 1. Fast path: apply linear substitution to reduce dimensions
	reducedHyp, subs := LinearSubstitutionFastPath(hypotheses, order)

	var allInitials []Node
	chainMap := make(map[string]Node) // mainVar -> polynomial

	// Queue of polynomials to insert into ascending chain
	queue := make([]Node, 0, len(reducedHyp))
	for _, h := range reducedHyp {
		_, primH, err := ContentPrimitive(h, "", order)
		if err == nil && primH != nil && !isZero(primH) {
			queue = append(queue, primH)
		} else if !isZero(h) {
			queue = append(queue, h)
		}
	}

	maxIters := 100
	iter := 0

	for len(queue) > 0 && iter < maxIters {
		iter++
		p := queue[0]
		queue = queue[1:]

		mvar, deg, init := IdentifyMainVarAndRank(p, order)
		if mvar == "" || isZero(p) {
			continue
		}
		if deg == 0 {
			// Inconsistent hypothesis or scalar contradiction
			if !isZero(p) {
				return nil, nil, fmt.Errorf("inconsistent hypothesis detected: %s", p.String())
			}
			continue
		}

		existing, exists := chainMap[mvar]
		if !exists {
			chainMap[mvar] = p
			if init != nil && !isZero(init) && !isOne(init) {
				allInitials = append(allInitials, init)
			}
			continue
		}

		// Conflict on same mainVar: use Subresultant / pseudo-division to reduce
		_, degExist, _ := IdentifyMainVarAndRank(existing, order)
		var higher, lower Node
		if deg >= degExist {
			higher, lower = p, existing
		} else {
			higher, lower = existing, p
			chainMap[mvar] = p // replace with lower degree polynomial
		}

		rem, err := SubresultantPrem(higher, lower, mvar, order)
		if err != nil {
			return nil, nil, err
		}

		if !isZero(rem) {
			_, primRem, err := ContentPrimitive(rem, "", order)
			if err == nil && primRem != nil && !isZero(primRem) {
				queue = append(queue, primRem)
			} else {
				queue = append(queue, rem)
			}
		}
	}

	// 2. Add linear substitution equations into chainMap
	for mvar, subExpr := range subs {
		pLinear, err := Parse(fmt.Sprintf("%s - (%s)", mvar, subExpr.String()))
		if err == nil {
			evalP, err := Eval(expandNode(pLinear))
			if err == nil && !isZero(evalP) {
				_, primP, _ := ContentPrimitive(evalP, mvar, order)
				if primP != nil && !isZero(primP) {
					chainMap[mvar] = primP
				} else {
					chainMap[mvar] = evalP
				}
			}
		}
	}

	// 3. Sort chain elements by mainVar order
	varRank := make(map[string]int)
	for i, v := range order {
		varRank[v] = i
	}

	type chainEntry struct {
		mvar    string
		rank    int
		deg     int
		initial Node
		poly    Node
	}

	var entries []chainEntry
	for _, poly := range chainMap {
		m, d, init := IdentifyMainVarAndRank(poly, order)
		r := varRank[m]
		entries = append(entries, chainEntry{
			mvar:    m,
			rank:    r,
			deg:     d,
			initial: init,
			poly:    poly,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rank < entries[j].rank
	})

	tSet := &TriangularSet{
		Elements: make([]Node, len(entries)),
		MainVars: make([]string, len(entries)),
		Initials: make([]Node, len(entries)),
		Degrees:  make([]int, len(entries)),
	}

	for i, e := range entries {
		tSet.Elements[i] = e.poly
		tSet.MainVars[i] = e.mvar
		tSet.Initials[i] = e.initial
		tSet.Degrees[i] = e.deg
	}

	return tSet, allInitials, nil
}
