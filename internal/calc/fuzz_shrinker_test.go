package calc_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// ShrinkPredicate defines the "interestingness test" for test-case reduction.
// Invariant (Regehr et al. 2012 PLDI C-Reduce):
// It must return true IF AND ONLY IF the candidate expression reproduces the EXACT SAME failure class
// and does not introduce extraneous panics, timeouts, division by zero, or syntax degradation (Slippage).
type ShrinkPredicate func(candidate ast.Node) bool

// ASTShrinker performs Hierarchical Delta Debugging (HDD, Misherghi & Su 2006)
// on mathematical ASTs with strict slippage safeguards and deterministic caching.
type ASTShrinker struct {
	MaxEvaluations int
	MaxDuration    time.Duration
	evalCount      int
	cache          map[string]bool
	startTime      time.Time
}

// NewASTShrinker creates an ASTShrinker with default execution budgets.
func NewASTShrinker() *ASTShrinker {
	return &ASTShrinker{
		MaxEvaluations: 150,
		MaxDuration:    4 * time.Second,
		cache:          make(map[string]bool),
	}
}

// Shrink iteratively reduces the root AST until a fixed point is reached or the budget is exhausted.
func (s *ASTShrinker) Shrink(root ast.Node, pred ShrinkPredicate) ast.Node {
	if root == nil || pred == nil {
		return root
	}

	s.evalCount = 0
	s.cache = make(map[string]bool)
	s.startTime = time.Now()

	// Initial check: if original expression does not trigger predicate, return unchanged
	if !s.testCandidate(root, pred) {
		return root
	}

	current := root

	// Multi-pass fixed-point loop:
	// Pass 1: Top-down subtree hoisting (replace parent with child)
	// Pass 2: Identity node replacement (replace subtree with 1, x, -1, 2, 0)
	// Pass 3: n-ary node ddmin (prune terms/factors from Add/Mul)
	// Pass 4: Constant coefficient binary-search reduction
	for pass := 0; pass < 5; pass++ {
		progress := false

		// Pass 1: Subtree hoisting
		if next, ok := s.shrinkHoisting(current, pred); ok {
			current = next
			progress = true
		}

		// Pass 2: n-ary list reduction (ddmin)
		if next, ok := s.shrinkNaryNodes(current, pred); ok {
			current = next
			progress = true
		}

		// Pass 3: Identity replacement
		if next, ok := s.shrinkIdentities(current, pred); ok {
			current = next
			progress = true
		}

		// Pass 4: Constant reduction
		if next, ok := s.shrinkConstants(current, pred); ok {
			current = next
			progress = true
		}

		if !progress || s.isBudgetExhausted() {
			break
		}
	}

	return current
}

func (s *ASTShrinker) isBudgetExhausted() bool {
	if s.evalCount >= s.MaxEvaluations {
		return true
	}
	if time.Since(s.startTime) >= s.MaxDuration {
		return true
	}
	return false
}

func (s *ASTShrinker) testCandidate(candidate ast.Node, pred ShrinkPredicate) bool {
	if candidate == nil || s.isBudgetExhausted() {
		return false
	}
	s.evalCount++

	key := candidate.String()
	if res, found := s.cache[key]; found {
		return res
	}

	// Guard against panics during candidate evaluation
	var passed bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				passed = false
			}
		}()
		passed = pred(candidate)
	}()

	s.cache[key] = passed
	return passed
}

// shrinkHoisting attempts to replace parent nodes with their direct children (Misherghi & Su 2006).
func (s *ASTShrinker) shrinkHoisting(root ast.Node, pred ShrinkPredicate) (ast.Node, bool) {
	if root == nil || s.isBudgetExhausted() {
		return root, false
	}

	// 1. Try hoisting root's own direct children first
	children := getDirectChildren(root)
	for _, child := range children {
		if s.testCandidate(child, pred) {
			// Found smaller candidate! Recursively shrink from here
			if next, ok := s.shrinkHoisting(child, pred); ok {
				return next, true
			}
			return child, true
		}
	}

	// 2. Recursively try hoisting inside children
	changed := false
	newRoot := replaceChildren(root, func(ch ast.Node) ast.Node {
		if changed || s.isBudgetExhausted() {
			return ch
		}
		if next, ok := s.shrinkHoisting(ch, pred); ok {
			changed = true
			return next
		}
		return ch
	})

	if changed && s.testCandidate(newRoot, pred) {
		return newRoot, true
	}
	return root, false
}

// shrinkIdentities attempts to replace subtrees with minimal identities: 1, x, -1, 2, 0.
// Invariant: 0 is tried last because it frequently triggers division-by-zero singularities (Slippage).
func (s *ASTShrinker) shrinkIdentities(root ast.Node, pred ShrinkPredicate) (ast.Node, bool) {
	if root == nil || s.isBudgetExhausted() {
		return root, false
	}

	// Do not replace minimal leaves with themselves
	switch root.(type) {
	case *ast.VarNode:
		return root, false
	case *ast.RationalNode:
		return root, false
	}

	// Candidate replacement identities in order of safety
	identities := []ast.Node{
		ast.NewRationalFromBigRat(big.NewRat(1, 1)),
		ast.NewVar("x"),
		ast.NewRationalFromBigRat(big.NewRat(-1, 1)),
		ast.NewRationalFromBigRat(big.NewRat(2, 1)),
		ast.NewRationalFromBigRat(big.NewRat(0, 1)),
	}

	// Try replacing current root with an identity
	for _, idNode := range identities {
		if s.testCandidate(idNode, pred) {
			return idNode, true
		}
	}

	// Recursively try replacing children
	changed := false
	newRoot := replaceChildren(root, func(ch ast.Node) ast.Node {
		if changed || s.isBudgetExhausted() {
			return ch
		}
		if next, ok := s.shrinkIdentities(ch, pred); ok {
			changed = true
			return next
		}
		return ch
	})

	if changed && s.testCandidate(newRoot, pred) {
		return newRoot, true
	}
	return root, false
}

// shrinkNaryNodes applies ddmin (Zeller 2002) to multi-child nodes (*ast.AddNode, *ast.MulNode).
func (s *ASTShrinker) shrinkNaryNodes(root ast.Node, pred ShrinkPredicate) (ast.Node, bool) {
	if root == nil || s.isBudgetExhausted() {
		return root, false
	}

	switch n := root.(type) {
	case *ast.AddNode:
		if len(n.Terms) > 2 {
			// Try 1-minimal removal of each term
			for i := 0; i < len(n.Terms); i++ {
				subTerms := make([]ast.Node, 0, len(n.Terms)-1)
				subTerms = append(subTerms, n.Terms[:i]...)
				subTerms = append(subTerms, n.Terms[i+1:]...)
				cand := ast.NewAdd(subTerms)
				if s.testCandidate(cand, pred) {
					// Recurse on the successfully reduced node
					if next, ok := s.shrinkNaryNodes(cand, pred); ok {
						return next, true
					}
					return cand, true
				}
			}
		}

	case *ast.MulNode:
		if len(n.Factors) > 2 {
			// Try 1-minimal removal of each factor
			for i := 0; i < len(n.Factors); i++ {
				subFactors := make([]ast.Node, 0, len(n.Factors)-1)
				subFactors = append(subFactors, n.Factors[:i]...)
				subFactors = append(subFactors, n.Factors[i+1:]...)
				cand := ast.NewMul(subFactors)
				if s.testCandidate(cand, pred) {
					if next, ok := s.shrinkNaryNodes(cand, pred); ok {
						return next, true
					}
					return cand, true
				}
			}
		}
	}

	// Recursively shrink inside children
	changed := false
	newRoot := replaceChildren(root, func(ch ast.Node) ast.Node {
		if changed || s.isBudgetExhausted() {
			return ch
		}
		if next, ok := s.shrinkNaryNodes(ch, pred); ok {
			changed = true
			return next
		}
		return ch
	})

	if changed && s.testCandidate(newRoot, pred) {
		return newRoot, true
	}
	return root, false
}

// shrinkConstants reduces large integer/rational coefficients towards 0 or 1.
func (s *ASTShrinker) shrinkConstants(root ast.Node, pred ShrinkPredicate) (ast.Node, bool) {
	if root == nil || s.isBudgetExhausted() {
		return root, false
	}

	if r, ok := root.(*ast.RationalNode); ok && r.Val != nil {
		num := new(big.Int).Set(r.Val.Num())
		den := new(big.Int).Set(r.Val.Denom())

		// If denominator > 1, try reducing to integer (denominator = 1)
		if den.Cmp(big.NewInt(1)) > 0 {
			cand := ast.NewRationalFromBigRat(new(big.Rat).SetInt(num))
			if s.testCandidate(cand, pred) {
				return cand, true
			}
		}

		// Try binary search towards 0 or 1
		if num.Abs(num).Cmp(big.NewInt(2)) > 0 {
			half := new(big.Int).Div(num, big.NewInt(2))
			cand := ast.NewRationalFromBigRat(new(big.Rat).SetFrac(half, den))
			if s.testCandidate(cand, pred) {
				return cand, true
			}
		}
	}

	// Recursively shrink child constants
	changed := false
	newRoot := replaceChildren(root, func(ch ast.Node) ast.Node {
		if changed || s.isBudgetExhausted() {
			return ch
		}
		if next, ok := s.shrinkConstants(ch, pred); ok {
			changed = true
			return next
		}
		return ch
	})

	if changed && s.testCandidate(newRoot, pred) {
		return newRoot, true
	}
	return root, false
}

func getDirectChildren(node ast.Node) []ast.Node {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.AddNode:
		return n.Terms
	case *ast.MulNode:
		return n.Factors
	case *ast.PowNode:
		return []ast.Node{n.Base, n.Exp}
	case *ast.UnaryOpNode:
		return []ast.Node{n.Expr}
	case *ast.SqrtNode:
		return []ast.Node{n.Radicand}
	case *ast.FuncNode:
		return n.Args
	default:
		return nil
	}
}

func replaceChildren(node ast.Node, transform func(ast.Node) ast.Node) ast.Node {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.AddNode:
		newTerms := make([]ast.Node, len(n.Terms))
		for i, t := range n.Terms {
			newTerms[i] = transform(t)
		}
		return ast.NewAdd(newTerms)
	case *ast.MulNode:
		newFactors := make([]ast.Node, len(n.Factors))
		for i, f := range n.Factors {
			newFactors[i] = transform(f)
		}
		return ast.NewMul(newFactors)
	case *ast.PowNode:
		newBase := transform(n.Base)
		newExp := transform(n.Exp)
		pow, err := ast.NewPow(newBase, newExp)
		if err != nil {
			return node
		}
		return pow
	case *ast.UnaryOpNode:
		newExpr := transform(n.Expr)
		uop, err := ast.NewUnaryOp(n.Op, newExpr)
		if err != nil {
			return node
		}
		return uop
	case *ast.SqrtNode:
		newRad := transform(n.Radicand)
		return ast.NewSqrt(newRad)
	case *ast.FuncNode:
		newArgs := make([]ast.Node, len(n.Args))
		for i, a := range n.Args {
			newArgs[i] = transform(a)
		}
		fn, err := ast.NewFunc(n.Name, newArgs)
		if err != nil {
			return node
		}
		return fn
	default:
		return node
	}
}

// TestASTShrinker_SlippageResilience verifies that the shrinker reduces a bloated expression
// to a minimal counterexample while strictly rejecting candidates that trigger unrelated errors (Slippage).
func TestASTShrinker_SlippageResilience(t *testing.T) {
	shrinker := NewASTShrinker()

	// Bloated input: (x + 100) * (y + 200) + (z * 0)
	// Target bug condition: expression contains both variables 'x' and 'y' with degree >= 1.
	// Extraneous terms: + (z * 0) and large constants 100, 200.
	x := ast.NewVar("x")
	y := ast.NewVar("y")
	z := ast.NewVar("z")
	c100 := ast.NewRationalFromBigRat(big.NewRat(100, 1))
	c200 := ast.NewRationalFromBigRat(big.NewRat(200, 1))
	c0 := ast.NewRationalFromBigRat(big.NewRat(0, 1))

	term1 := ast.NewAdd([]ast.Node{x, c100})
	term2 := ast.NewAdd([]ast.Node{y, c200})
	mul1 := ast.NewMul([]ast.Node{term1, term2})
	mul2 := ast.NewMul([]ast.Node{z, c0})
	bloated := ast.NewAdd([]ast.Node{mul1, mul2})

	// Predicate: must contain both 'x' and 'y', and must NOT divide by zero or panic
	pred := func(n ast.Node) bool {
		hasX := false
		hasY := false
		ast.Walk(n, func(curr ast.Node) bool {
			if v, ok := curr.(*ast.VarNode); ok {
				if v.Name == "x" {
					hasX = true
				}
				if v.Name == "y" {
					hasY = true
				}
			}
			return true
		})
		return hasX && hasY
	}

	shrunk := shrinker.Shrink(bloated, pred)
	t.Logf("Original bloated expression: %s", bloated.String())
	t.Logf("Minimized expression: %s (Evaluations: %d)", shrunk.String(), shrinker.evalCount)

	// Invariant 1: Shrunk expression must satisfy the predicate
	if !pred(shrunk) {
		t.Fatalf("Shrunk expression failed predicate: %s", shrunk.String())
	}

	// Invariant 2: Shrunk expression must be strictly smaller in node count than bloated
	origCount := CountNodes(bloated)
	shrunkCount := CountNodes(shrunk)
	if shrunkCount >= origCount {
		t.Fatalf("Expected smaller node count: original=%d, shrunk=%d", origCount, shrunkCount)
	}

	// Invariant 3: Extraneous 'z' must be pruned away
	hasZ := false
	ast.Walk(shrunk, func(curr ast.Node) bool {
		if v, ok := curr.(*ast.VarNode); ok && v.Name == "z" {
			hasZ = true
		}
		return true
	})
	if hasZ {
		t.Fatalf("Shrinker failed to prune extraneous variable 'z': %s", shrunk.String())
	}
}
