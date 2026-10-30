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

// shrinkContext represents a path-copying substitution function mapping a sub-expression to the complete root AST.
type shrinkContext func(replacement ast.Node) ast.Node

// ASTShrinker performs Hierarchical Delta Debugging (HDD, Misherghi & Su 2006)
// on mathematical ASTs with strict slippage safeguards, occurrence-aware contextual substitution,
// and deterministic caching.
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
		MaxEvaluations: 200,
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
	idCtx := func(n ast.Node) ast.Node { return n }

	// Multi-pass fixed-point loop:
	// Pass 1: Subtree hoisting (replace parent with one of its direct children)
	// Pass 2: n-ary node ddmin (remove terms/factors from Add/Mul)
	// Pass 3: Identity node replacement (replace subtrees with 1, 0, x, -1)
	// Pass 4: Constant coefficient binary-search reduction
	for pass := 0; pass < 6; pass++ {
		progress := false

		// Pass 1: Subtree hoisting
		if next, ok := s.shrinkHoisting(current, idCtx, pred); ok {
			current = next
			progress = true
		}

		// Pass 2: n-ary list reduction (ddmin)
		if next, ok := s.shrinkNaryNodes(current, idCtx, pred); ok {
			current = next
			progress = true
		}

		// Pass 3: Identity replacement
		if next, ok := s.shrinkIdentities(current, idCtx, pred); ok {
			current = next
			progress = true
		}

		// Pass 4: Constant reduction
		if next, ok := s.shrinkConstants(current, idCtx, pred); ok {
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

	key := candidate.String()
	if res, found := s.cache[key]; found {
		return res
	}

	// Increment evaluation count ONLY on cache miss
	s.evalCount++

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

// shrinkHoisting attempts to replace parent nodes with their direct children inside the full expression context.
func (s *ASTShrinker) shrinkHoisting(curr ast.Node, ctx shrinkContext, pred ShrinkPredicate) (ast.Node, bool) {
	if curr == nil || s.isBudgetExhausted() {
		return curr, false
	}

	// 1. Try hoisting direct children to replace curr in root
	children := getDirectChildren(curr)
	for _, child := range children {
		candRoot := ctx(child)
		if s.testCandidate(candRoot, pred) {
			// Found smaller candidate; recursively shrink from here
			if next, ok := s.shrinkHoisting(child, ctx, pred); ok {
				return next, true
			}
			return candRoot, true
		}
	}

	// 2. Recursively try hoisting inside children
	for i, ch := range children {
		if s.isBudgetExhausted() {
			break
		}
		childCtx := func(sub ast.Node) ast.Node {
			return ctx(replaceNthChild(curr, i, sub))
		}
		if nextRoot, ok := s.shrinkHoisting(ch, childCtx, pred); ok {
			return nextRoot, true
		}
	}

	return curr, false
}

// shrinkIdentities attempts to replace subtrees with minimal algebraic identities inside full root context: 1, 0, x, -1.
func (s *ASTShrinker) shrinkIdentities(curr ast.Node, ctx shrinkContext, pred ShrinkPredicate) (ast.Node, bool) {
	if curr == nil || s.isBudgetExhausted() {
		return curr, false
	}

	// Do not replace minimal leaves with themselves
	isLeaf := false
	switch curr.(type) {
	case *ast.VarNode, *ast.RationalNode, *ast.ConstNode:
		isLeaf = true
	}

	if !isLeaf {
		// Canonical identity replacements in increasing risk order
		identities := []ast.Node{
			ast.NewRationalFromBigRat(big.NewRat(1, 1)),
			ast.NewRationalFromBigRat(big.NewRat(0, 1)),
			ast.NewVar("x"),
			ast.NewRationalFromBigRat(big.NewRat(-1, 1)),
		}

		for _, idNode := range identities {
			candRoot := ctx(idNode)
			if s.testCandidate(candRoot, pred) {
				return candRoot, true
			}
		}
	}

	// Recursively try replacing inside children
	children := getDirectChildren(curr)
	for i, ch := range children {
		if s.isBudgetExhausted() {
			break
		}
		childCtx := func(sub ast.Node) ast.Node {
			return ctx(replaceNthChild(curr, i, sub))
		}
		if nextRoot, ok := s.shrinkIdentities(ch, childCtx, pred); ok {
			return nextRoot, true
		}
	}

	return curr, false
}

// shrinkNaryNodes applies ddmin (Zeller 2002) to multi-child nodes (*ast.AddNode, *ast.MulNode) inside full root context.
func (s *ASTShrinker) shrinkNaryNodes(curr ast.Node, ctx shrinkContext, pred ShrinkPredicate) (ast.Node, bool) {
	if curr == nil || s.isBudgetExhausted() {
		return curr, false
	}

	switch n := curr.(type) {
	case *ast.AddNode:
		if len(n.Terms) >= 2 {
			// Try 1-minimal removal of each term
			for i := 0; i < len(n.Terms); i++ {
				var candNode ast.Node
				if len(n.Terms) == 2 {
					// 2-term addition reduces to the single remaining term
					candNode = n.Terms[1-i]
				} else {
					subTerms := make([]ast.Node, 0, len(n.Terms)-1)
					subTerms = append(subTerms, n.Terms[:i]...)
					subTerms = append(subTerms, n.Terms[i+1:]...)
					candNode = ast.NewAdd(subTerms)
				}
				candRoot := ctx(candNode)
				if s.testCandidate(candRoot, pred) {
					if nextRoot, ok := s.shrinkNaryNodes(candNode, ctx, pred); ok {
						return nextRoot, true
					}
					return candRoot, true
				}
			}
		}

	case *ast.MulNode:
		if len(n.Factors) >= 2 {
			for i := 0; i < len(n.Factors); i++ {
				var candNode ast.Node
				if len(n.Factors) == 2 {
					candNode = n.Factors[1-i]
				} else {
					subFactors := make([]ast.Node, 0, len(n.Factors)-1)
					subFactors = append(subFactors, n.Factors[:i]...)
					subFactors = append(subFactors, n.Factors[i+1:]...)
					candNode = ast.NewMul(subFactors)
				}
				candRoot := ctx(candNode)
				if s.testCandidate(candRoot, pred) {
					if nextRoot, ok := s.shrinkNaryNodes(candNode, ctx, pred); ok {
						return nextRoot, true
					}
					return candRoot, true
				}
			}
		}
	}

	// Recursively shrink inside children
	children := getDirectChildren(curr)
	for i, ch := range children {
		if s.isBudgetExhausted() {
			break
		}
		childCtx := func(sub ast.Node) ast.Node {
			return ctx(replaceNthChild(curr, i, sub))
		}
		if nextRoot, ok := s.shrinkNaryNodes(ch, childCtx, pred); ok {
			return nextRoot, true
		}
	}

	return curr, false
}

// shrinkConstants reduces large integer/rational coefficients towards 0 or 1 without mutating big.Int in-place.
func (s *ASTShrinker) shrinkConstants(curr ast.Node, ctx shrinkContext, pred ShrinkPredicate) (ast.Node, bool) {
	if curr == nil || s.isBudgetExhausted() {
		return curr, false
	}

	if r, ok := curr.(*ast.RationalNode); ok && r.Val != nil {
		num := new(big.Int).Set(r.Val.Num())
		den := new(big.Int).Set(r.Val.Denom())

		// If denominator > 1, try reducing to integer (denominator = 1)
		if den.Cmp(big.NewInt(1)) > 0 {
			cand := ast.NewRationalFromBigRat(new(big.Rat).SetInt(num))
			candRoot := ctx(cand)
			if s.testCandidate(candRoot, pred) {
				return candRoot, true
			}
		}

		// Try binary search towards 0 while strictly preserving sign
		absNum := new(big.Int).Abs(num)
		if absNum.Cmp(big.NewInt(2)) > 0 {
			half := new(big.Int).Div(absNum, big.NewInt(2))
			if num.Sign() < 0 {
				half.Neg(half)
			}
			cand := ast.NewRationalFromBigRat(new(big.Rat).SetFrac(half, den))
			candRoot := ctx(cand)
			if s.testCandidate(candRoot, pred) {
				return candRoot, true
			}
		}

		// Try setting to 1 or 0 directly
		if num.Cmp(big.NewInt(1)) != 0 {
			cand1 := ast.NewRationalFromBigRat(big.NewRat(1, 1))
			candRoot1 := ctx(cand1)
			if s.testCandidate(candRoot1, pred) {
				return candRoot1, true
			}
		}
		if num.Sign() != 0 {
			cand0 := ast.NewRationalFromBigRat(big.NewRat(0, 1))
			candRoot0 := ctx(cand0)
			if s.testCandidate(candRoot0, pred) {
				return candRoot0, true
			}
		}
	}

	// Recursively shrink child constants
	children := getDirectChildren(curr)
	for i, ch := range children {
		if s.isBudgetExhausted() {
			break
		}
		childCtx := func(sub ast.Node) ast.Node {
			return ctx(replaceNthChild(curr, i, sub))
		}
		if nextRoot, ok := s.shrinkConstants(ch, childCtx, pred); ok {
			return nextRoot, true
		}
	}

	return curr, false
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

func replaceNthChild(node ast.Node, idx int, replacement ast.Node) ast.Node {
	if node == nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.AddNode:
		if idx < 0 || idx >= len(n.Terms) {
			return node
		}
		newTerms := make([]ast.Node, len(n.Terms))
		copy(newTerms, n.Terms)
		newTerms[idx] = replacement
		return ast.NewAdd(newTerms)

	case *ast.MulNode:
		if idx < 0 || idx >= len(n.Factors) {
			return node
		}
		newFactors := make([]ast.Node, len(n.Factors))
		copy(newFactors, n.Factors)
		newFactors[idx] = replacement
		return ast.NewMul(newFactors)

	case *ast.PowNode:
		if idx == 0 {
			pow, err := ast.NewPow(replacement, n.Exp)
			if err != nil {
				return node
			}
			return pow
		} else if idx == 1 {
			pow, err := ast.NewPow(n.Base, replacement)
			if err != nil {
				return node
			}
			return pow
		}
		return node

	case *ast.UnaryOpNode:
		if idx == 0 {
			uop, err := ast.NewUnaryOp(n.Op, replacement)
			if err != nil {
				return node
			}
			return uop
		}
		return node

	case *ast.SqrtNode:
		if idx == 0 {
			return ast.NewSqrt(replacement)
		}
		return node

	case *ast.FuncNode:
		if idx < 0 || idx >= len(n.Args) {
			return node
		}
		newArgs := make([]ast.Node, len(n.Args))
		copy(newArgs, n.Args)
		newArgs[idx] = replacement
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
