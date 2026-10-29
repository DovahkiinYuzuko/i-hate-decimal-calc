package calc_test

import (
	"fmt"
	"math"
	"math/big"
	"math/cmplx"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// OracleVerdict represents the outcome classification of the 3-tier oracle evaluation.
type OracleVerdict int

const (
	// VerdictExactMatch: Tier 1 structural canonical DAG / ast.Identical match (O(1)).
	VerdictExactMatch OracleVerdict = iota
	// VerdictSymbolicZero: Tier 2 symbolic reduction Sub(before, after) == 0.
	VerdictSymbolicZero
	// VerdictFingerprintMatch: Tier 3 numerical fingerprint match via Schwartz-Zippel sampling
	// (Canonicalization gap / soft finding; Tier 1/2 failed to reduce).
	VerdictFingerprintMatch
	// VerdictIndeterminate: Tier 3 could not evaluate any valid sample point (e.g., poles or symbolic-only constructs).
	VerdictIndeterminate
	// VerdictSoundnessBug: Tier 3 numerical fingerprint mismatch (Definite correctness bug).
	VerdictSoundnessBug
)

func (v OracleVerdict) String() string {
	switch v {
	case VerdictExactMatch:
		return "VerdictExactMatch"
	case VerdictSymbolicZero:
		return "VerdictSymbolicZero"
	case VerdictFingerprintMatch:
		return "VerdictFingerprintMatch"
	case VerdictIndeterminate:
		return "VerdictIndeterminate"
	case VerdictSoundnessBug:
		return "VerdictSoundnessBug"
	default:
		return fmt.Sprintf("OracleVerdict(%d)", v)
	}
}

// gaussianRat represents an exact Gaussian rational number a + bi in Q[i] using arbitrary-precision big.Rat.
type gaussianRat struct {
	re *big.Rat
	im *big.Rat
}

func newGaussianRat(re, im *big.Rat) gaussianRat {
	r := new(big.Rat)
	if re != nil {
		r.Set(re)
	}
	i := new(big.Rat)
	if im != nil {
		i.Set(im)
	}
	return gaussianRat{re: r, im: i}
}

func newGaussianRatFromInt(re, im int64) gaussianRat {
	return gaussianRat{
		re: big.NewRat(re, 1),
		im: big.NewRat(im, 1),
	}
}

func (g gaussianRat) isZero() bool {
	return g.re.Sign() == 0 && g.im.Sign() == 0
}

func (g gaussianRat) isReal() bool {
	return g.im.Sign() == 0
}

func (g gaussianRat) equal(other gaussianRat) bool {
	return g.re.Cmp(other.re) == 0 && g.im.Cmp(other.im) == 0
}

func (g gaussianRat) neg() gaussianRat {
	return gaussianRat{
		re: new(big.Rat).Neg(g.re),
		im: new(big.Rat).Neg(g.im),
	}
}

func (g gaussianRat) add(other gaussianRat) gaussianRat {
	return gaussianRat{
		re: new(big.Rat).Add(g.re, other.re),
		im: new(big.Rat).Add(g.im, other.im),
	}
}


func (g gaussianRat) mul(other gaussianRat) gaussianRat {
	// (a + bi)(c + di) = (ac - bd) + (ad + bc)i
	ac := new(big.Rat).Mul(g.re, other.re)
	bd := new(big.Rat).Mul(g.im, other.im)
	ad := new(big.Rat).Mul(g.re, other.im)
	bc := new(big.Rat).Mul(g.im, other.re)
	return gaussianRat{
		re: ac.Sub(ac, bd),
		im: ad.Add(ad, bc),
	}
}

func (g gaussianRat) inv() (gaussianRat, bool) {
	// 1 / (a + bi) = (a - bi) / (a^2 + b^2)
	a2 := new(big.Rat).Mul(g.re, g.re)
	b2 := new(big.Rat).Mul(g.im, g.im)
	norm := a2.Add(a2, b2)
	if norm.Sign() == 0 {
		return gaussianRat{}, false
	}
	return gaussianRat{
		re: new(big.Rat).Quo(g.re, norm),
		im: new(big.Rat).Quo(new(big.Rat).Neg(g.im), norm),
	}, true
}

func (g gaussianRat) powInt(exp int64) (gaussianRat, bool) {
	if exp == 0 {
		if g.isZero() {
			return gaussianRat{}, false
		}
		return newGaussianRatFromInt(1, 0), true
	}
	// Bound exponent magnitude to prevent bit-length blowup in the independent test oracle.
	if exp > 64 || exp < -64 {
		return gaussianRat{}, false
	}
	base := g
	if exp < 0 {
		var ok bool
		base, ok = g.inv()
		if !ok {
			return gaussianRat{}, false
		}
		exp = -exp
	}
	result := newGaussianRatFromInt(1, 0)
	cur := base
	for exp > 0 {
		if exp&1 == 1 {
			result = result.mul(cur)
		}
		cur = cur.mul(cur)
		exp >>= 1
	}
	return result, true
}

// EvaluateInvariant executes the Three-Tier Equality Oracle between two AST expressions:
// Tier 1: Canonical DAG pointer identity via ast.Identical (O(1)).
// Tier 2: Symbolic reduction of Sub(before, after) == 0.
// Tier 3: Independent exact Gaussian rational Q[i] evaluation (with complex fallback for transcendentals)
// over K=6 multivariate Schwartz-Zippel sample vectors where each variable receives an independent value.
func EvaluateInvariant(before, after ast.Node, domainProfile ProfileType) OracleVerdict {
	// --- Tier 1: O(1) Canonical DAG Pointer Identity ---
	if ast.Identical(before, after) {
		return VerdictExactMatch
	}

	env := calc.NewEnv()

	// --- Tier 2: Symbolic Zero Reduction ---
	negAfter, err := ast.NewUnaryOp("-", after)
	if err == nil {
		diffNode := ast.NewAdd([]ast.Node{before, negAfter})
		evalDiff, err := calc.EvalWithEnv(diffNode, env)
		if err == nil && isLiteralZero(evalDiff) {
			return VerdictSymbolicZero
		}
	}

	// --- Tier 3: Schwartz-Zippel Independent Fingerprinting ---
	vars := extractVariables(before, after)
	const numSamples = 6

	mismatchDetected := false
	validEvaluations := 0

	for sampleIdx := 0; sampleIdx < numSamples; sampleIdx++ {
		assignment := buildIndependentSampleAssignment(vars, sampleIdx, domainProfile)
		subBefore := substituteVarsMap(before, assignment)
		subAfter := substituteVarsMap(after, assignment)

		// Sub-tier 3A: Exact Gaussian Rational Q[i] Evaluation (zero rounding error)
		exactB, okExactB := evalExactGaussianRat(subBefore)
		exactA, okExactA := evalExactGaussianRat(subAfter)
		if okExactB && okExactA {
			validEvaluations++
			if !exactB.equal(exactA) {
				mismatchDetected = true
				break
			}
			continue
		}

		// Sub-tier 3B: Complex128 fallback for transcendental functions / irrational radicals
		valB, okB := evalNumericalComplex(subBefore)
		valA, okA := evalNumericalComplex(subAfter)
		if !okB || !okA {
			continue
		}

		validEvaluations++
		diff := cmplx.Abs(valB - valA)
		if math.IsNaN(diff) || math.IsInf(diff, 0) {
			continue
		}
		relTol := 1e-9 * (1.0 + cmplx.Abs(valB) + cmplx.Abs(valA))
		if diff > relTol && !approxAlmostEqual(valB, valA, 1e-9) {
			mismatchDetected = true
			break
		}
	}

	if mismatchDetected {
		return VerdictSoundnessBug
	}
	if validEvaluations == 0 {
		return VerdictIndeterminate
	}

	return VerdictFingerprintMatch
}

// isLiteralZero returns true if the AST node represents symbolic zero.
func isLiteralZero(n ast.Node) bool {
	if n == nil {
		return false
	}
	switch v := n.(type) {
	case *ast.RationalNode:
		return v.Val.Sign() == 0
	case *ast.ComplexNode:
		return isLiteralZero(v.Real) && isLiteralZero(v.Imag)
	case *ast.QuantityNode:
		return isLiteralZero(v.Value)
	case *ast.PolyNode:
		return len(v.Terms) == 0
	}
	return false
}

// extractVariables collects all distinct variable names appearing in the given AST nodes in deterministic sorted order.
func extractVariables(nodes ...ast.Node) []string {
	varMap := make(map[string]bool)
	for _, n := range nodes {
		if n == nil {
			continue
		}
		ast.Walk(n, func(curr ast.Node) bool {
			switch v := curr.(type) {
			case *ast.VarNode:
				varMap[v.Name] = true
			case *ast.PolyNode:
				for _, name := range v.Vars {
					varMap[name] = true
				}
			}
			return true
		})
	}

	vars := make([]string, 0, len(varMap))
	for v := range varMap {
		vars = append(vars, v)
	}
	sort.Strings(vars)
	return vars
}

// primePool provides distinct primes for constructing independent rational sample vectors.
var primePool = []int64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61}

// buildIndependentSampleAssignment assigns an independent rational number to each variable
// so that multivariate expressions are never collapsed onto the diagonal x_1 = x_2 = ... = x_m.
func buildIndependentSampleAssignment(vars []string, sampleIdx int, domainProfile ProfileType) map[string]*big.Rat {
	assignment := make(map[string]*big.Rat, len(vars))
	for varIdx, name := range vars {
		// Discrete time variables in Z-transform / recurrence profiles require non-negative integer points.
		if domainProfile == ProfileTransforms || name == "n" || name == "k" {
			val := int64(sampleIdx + 2 + varIdx*3)
			assignment[name] = big.NewRat(val, 1)
			continue
		}
		// Choose distinct primes for numerator and denominator to avoid poles at integers and keep variables independent.
		numPrime := primePool[(sampleIdx*3+varIdx*5)%len(primePool)]
		denPrime := primePool[(sampleIdx*2+varIdx*7+1)%len(primePool)]
		if numPrime == denPrime {
			denPrime = primePool[(sampleIdx*2+varIdx*7+2)%len(primePool)]
		}
		assignment[name] = big.NewRat(numPrime, denPrime)
	}
	return assignment
}

// polyNodeToAST converts a *ast.PolyNode into standard Add/Mul/Pow AST nodes for substitution and evaluation.
func polyNodeToAST(p *ast.PolyNode) ast.Node {
	if p == nil || len(p.Terms) == 0 {
		return ast.NewRationalFromBigRat(big.NewRat(0, 1))
	}
	terms := make([]ast.Node, 0, len(p.Terms))
	for _, m := range p.Terms {
		factors := []ast.Node{ast.NewRationalFromBigRat(m.Coeff)}
		for idx, exp := range m.Exponents {
			if exp == 0 || idx >= len(p.Vars) {
				continue
			}
			pow, err := ast.NewPow(ast.NewVar(p.Vars[idx]), ast.NewRationalFromBigRat(big.NewRat(int64(exp), 1)))
			if err == nil {
				factors = append(factors, pow)
			}
		}
		terms = append(terms, ast.NewMul(factors))
	}
	return ast.NewAdd(terms)
}

// substituteVarsMap substitutes each variable in assignment with its corresponding RationalNode.
func substituteVarsMap(node ast.Node, assignment map[string]*big.Rat) ast.Node {
	if node == nil {
		return nil
	}
	if p, ok := node.(*ast.PolyNode); ok {
		node = polyNodeToAST(p)
	}
	curr := node
	for name, val := range assignment {
		curr = ast.Substitute(curr, name, ast.NewRationalFromBigRat(new(big.Rat).Set(val)))
	}
	return curr
}

// evalExactGaussianRat evaluates an AST expression exactly over the Gaussian rationals Q[i].
// It returns (value, true) if and only if the entire expression can be evaluated without
// irrational numbers or transcendental approximations.
func evalExactGaussianRat(n ast.Node) (gaussianRat, bool) {
	if n == nil {
		return gaussianRat{}, false
	}
	switch v := n.(type) {
	case *ast.RationalNode:
		return newGaussianRat(v.Val, nil), true
	case *ast.ConstNode:
		if v.Name == "i" {
			return newGaussianRatFromInt(0, 1), true
		}
		return gaussianRat{}, false
	case *ast.ComplexNode:
		re, ok1 := evalExactGaussianRat(v.Real)
		im, ok2 := evalExactGaussianRat(v.Imag)
		if !ok1 || !ok2 || !re.isReal() || !im.isReal() {
			return gaussianRat{}, false
		}
		return newGaussianRat(re.re, im.re), true
	case *ast.SqrtNode:
		rad, ok := evalExactGaussianRat(v.Radicand)
		if !ok || !rad.isReal() || rad.re.Sign() < 0 {
			return gaussianRat{}, false
		}
		numSqrt := new(big.Int).Sqrt(rad.re.Num())
		if new(big.Int).Mul(numSqrt, numSqrt).Cmp(rad.re.Num()) != 0 {
			return gaussianRat{}, false
		}
		denSqrt := new(big.Int).Sqrt(rad.re.Denom())
		if new(big.Int).Mul(denSqrt, denSqrt).Cmp(rad.re.Denom()) != 0 {
			return gaussianRat{}, false
		}
		return newGaussianRat(new(big.Rat).SetFrac(numSqrt, denSqrt), nil), true
	case *ast.PolyNode:
		return evalExactGaussianRat(polyNodeToAST(v))
	case *ast.QuantityNode:
		return evalExactGaussianRat(v.Value)
	case *ast.UnaryOpNode:
		val, ok := evalExactGaussianRat(v.Expr)
		if !ok {
			return gaussianRat{}, false
		}
		switch v.Op {
		case "-":
			return val.neg(), true
		case "+":
			return val, true
		default:
			return gaussianRat{}, false
		}
	case *ast.AddNode:
		sum := newGaussianRatFromInt(0, 0)
		for _, term := range v.Terms {
			c, ok := evalExactGaussianRat(term)
			if !ok {
				return gaussianRat{}, false
			}
			sum = sum.add(c)
		}
		return sum, true
	case *ast.MulNode:
		prod := newGaussianRatFromInt(1, 0)
		for _, factor := range v.Factors {
			c, ok := evalExactGaussianRat(factor)
			if !ok {
				return gaussianRat{}, false
			}
			prod = prod.mul(c)
		}
		return prod, true
	case *ast.PowNode:
		base, ok1 := evalExactGaussianRat(v.Base)
		exp, ok2 := evalExactGaussianRat(v.Exp)
		if !ok1 || !ok2 || !exp.isReal() || !exp.re.IsInt() {
			return gaussianRat{}, false
		}
		expNum := exp.re.Num()
		if !expNum.IsInt64() {
			return gaussianRat{}, false
		}
		return base.powInt(expNum.Int64())
	case *ast.FuncNode:
		if len(v.Args) == 1 {
			arg, ok := evalExactGaussianRat(v.Args[0])
			if !ok {
				return gaussianRat{}, false
			}
			switch v.Name {
			case "abs":
				if arg.isReal() {
					return newGaussianRat(new(big.Rat).Abs(arg.re), nil), true
				}
				if arg.re.Sign() == 0 {
					return newGaussianRat(new(big.Rat).Abs(arg.im), nil), true
				}
				return gaussianRat{}, false
			case "re":
				return newGaussianRat(arg.re, nil), true
			case "im":
				return newGaussianRat(arg.im, nil), true
			case "conj":
				return newGaussianRat(arg.re, new(big.Rat).Neg(arg.im)), true
			case "u":
				if !arg.isReal() {
					return gaussianRat{}, false
				}
				if arg.re.Sign() >= 0 {
					return newGaussianRatFromInt(1, 0), true
				}
				return newGaussianRatFromInt(0, 0), true
			case "delta":
				if !arg.isReal() {
					return gaussianRat{}, false
				}
				if arg.re.Sign() == 0 {
					return newGaussianRatFromInt(1, 0), true
				}
				return newGaussianRatFromInt(0, 0), true
			case "floor":
				if !arg.isReal() {
					return gaussianRat{}, false
				}
				q := new(big.Int).Div(arg.re.Num(), arg.re.Denom())
				return newGaussianRat(new(big.Rat).SetInt(q), nil), true
			case "ceil":
				if !arg.isReal() {
					return gaussianRat{}, false
				}
				negRe := new(big.Rat).Neg(arg.re)
				q := new(big.Int).Div(negRe.Num(), negRe.Denom())
				q.Neg(q)
				return newGaussianRat(new(big.Rat).SetInt(q), nil), true
			}
		}
		return gaussianRat{}, false
	default:
		return gaussianRat{}, false
	}
}

func isValidComplex(c complex128) bool {
	if cmplx.IsNaN(c) || cmplx.IsInf(c) {
		return false
	}
	// Reject values near IEEE-754 overflow where mantissa precision collapses
	const maxSafeMag = 1e100
	if math.Abs(real(c)) > maxSafeMag || math.Abs(imag(c)) > maxSafeMag {
		return false
	}
	return true
}

// evalNumericalComplex recursively evaluates an AST node to complex128 independently as a fallback for transcendentals.
func evalNumericalComplex(n ast.Node) (complex128, bool) {
	if n == nil {
		return 0, false
	}
	switch v := n.(type) {
	case *ast.RationalNode:
		f, _ := v.Val.Float64()
		res := complex(f, 0)
		return res, isValidComplex(res)
	case *ast.ConstNode:
		switch v.Name {
		case "pi":
			return complex(math.Pi, 0), true
		case "e":
			return complex(math.E, 0), true
		case "i":
			return complex(0, 1), true
		case "deg":
			return complex(math.Pi/180.0, 0), true
		default:
			return 0, false
		}
	case *ast.PolyNode:
		return evalNumericalComplex(polyNodeToAST(v))
	case *ast.QuantityNode:
		return evalNumericalComplex(v.Value)
	case *ast.SqrtNode:
		c, ok := evalNumericalComplex(v.Radicand)
		if !ok {
			return 0, false
		}
		res := cmplx.Sqrt(c)
		return res, isValidComplex(res)
	case *ast.ComplexNode:
		re, ok1 := evalNumericalComplex(v.Real)
		im, ok2 := evalNumericalComplex(v.Imag)
		if !ok1 || !ok2 {
			return 0, false
		}
		res := complex(real(re), real(im))
		return res, isValidComplex(res)
	case *ast.UnaryOpNode:
		val, ok := evalNumericalComplex(v.Expr)
		if !ok {
			return 0, false
		}
		switch v.Op {
		case "-":
			return -val, isValidComplex(-val)
		case "+":
			return val, isValidComplex(val)
		default:
			return 0, false
		}
	case *ast.AddNode:
		var sum complex128
		for _, term := range v.Terms {
			c, ok := evalNumericalComplex(term)
			if !ok {
				return 0, false
			}
			sum += c
			if !isValidComplex(sum) {
				return 0, false
			}
		}
		return sum, true
	case *ast.MulNode:
		prod := complex(1, 0)
		for _, factor := range v.Factors {
			c, ok := evalNumericalComplex(factor)
			if !ok {
				return 0, false
			}
			prod *= c
			if !isValidComplex(prod) {
				return 0, false
			}
		}
		return prod, true
	case *ast.PowNode:
		base, ok1 := evalNumericalComplex(v.Base)
		exp, ok2 := evalNumericalComplex(v.Exp)
		if !ok1 || !ok2 {
			return 0, false
		}
		if base == 0 && real(exp) <= 0 {
			return 0, false
		}
		res := cmplx.Pow(base, exp)
		return res, isValidComplex(res)
	case *ast.FuncNode:
		if len(v.Args) == 1 {
			arg, ok := evalNumericalComplex(v.Args[0])
			if !ok {
				return 0, false
			}
			var res complex128
			switch v.Name {
			case "sin":
				res = cmplx.Sin(arg)
			case "cos":
				res = cmplx.Cos(arg)
			case "tan":
				res = cmplx.Tan(arg)
			case "asin":
				res = cmplx.Asin(arg)
			case "acos":
				res = cmplx.Acos(arg)
			case "atan":
				res = cmplx.Atan(arg)
			case "sinh":
				res = cmplx.Sinh(arg)
			case "cosh":
				res = cmplx.Cosh(arg)
			case "tanh":
				res = cmplx.Tanh(arg)
			case "abs":
				res = complex(cmplx.Abs(arg), 0)
			case "ln":
				if arg == 0 {
					return 0, false
				}
				res = cmplx.Log(arg)
			case "exp":
				res = cmplx.Exp(arg)
			case "re":
				res = complex(real(arg), 0)
			case "im":
				res = complex(imag(arg), 0)
			case "conj":
				res = cmplx.Conj(arg)
			case "u":
				if math.Abs(imag(arg)) > 1e-12 {
					return 0, false
				}
				if real(arg) >= 0 {
					return complex(1, 0), true
				}
				return complex(0, 0), true
			case "delta":
				if math.Abs(imag(arg)) > 1e-12 {
					return 0, false
				}
				if math.Abs(real(arg)) <= 1e-12 {
					return complex(1, 0), true
				}
				return complex(0, 0), true
			default:
				return 0, false
			}
			return res, isValidComplex(res)
		}
		return 0, false
	default:
		return 0, false
	}
}

func approxAlmostEqual(a, b complex128, eps float64) bool {
	if !isValidComplex(a) || !isValidComplex(b) {
		return false
	}
	return cmplx.Abs(a-b) <= eps
}

var (
	// Matches any Go fmt error verb sequence such as %!s(MISSING), %!d(BADINDEX), %!v(PANIC=...), %!(EXTRA ...), %!(NOVERB).
	rePlaceholderCorruption = regexp.MustCompile(`%!(?:[a-zA-Z0-9_]*\([^)]*\)|\((?:EXTRA[^)]*|MISSING|BADINDEX|NOVERB)\))`)
	// Matches unexpanded dotted i18n keys anywhere in the message or as the entire message.
	reUnregisteredKey = regexp.MustCompile(`(?:^|\s)(?:errors|approx|parser|calc|i18n|iztrans|ztrans|rsolve|calculus|simplify|rational|verify|ast|units|lp|matrix|cad|ode|poly|number_theory|stats|geo|cli|repl)\.[a-zA-Z0-9_.-]+`)
	reRawDottedKey    = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-zA-Z0-9_-]+)+$`)
)

// AssertI18nErrorIntegrity verifies that an error does not leak formatting placeholders
// or untranslated key strings.
func AssertI18nErrorIntegrity(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}

	msg := err.Error()

	// Check 1: fmt format placeholder corruptions
	if rePlaceholderCorruption.MatchString(msg) {
		t.Errorf("Detected corrupted fmt placeholder in error message: %q", msg)
	}

	// Check 2: Raw untranslated key leaking as error message
	trimmed := strings.TrimSpace(msg)
	if reUnregisteredKey.MatchString(trimmed) || reRawDottedKey.MatchString(trimmed) {
		t.Errorf("Detected unexpanded i18n key literal leaking in error message: %q", msg)
	}
}

// TestThreeTierOracle_Hierarchy tests the 3-tier oracle evaluation hierarchy and bug-detection guarantees.
func TestThreeTierOracle_Hierarchy(t *testing.T) {
	// 1. Tier 1: Identical nodes
	pool := ast.NewInternPool()
	rawNode1 := ast.NewAdd([]ast.Node{ast.NewVar("x"), ast.NewRationalFromBigRat(big.NewRat(1, 1))})
	c1 := pool.ToCanonical(rawNode1)
	c2 := pool.ToCanonical(rawNode1)

	v1 := EvaluateInvariant(c1, c2, ProfileAlgebra)
	if v1 != VerdictExactMatch {
		t.Fatalf("Expected VerdictExactMatch for identical canonical nodes, got %v", v1)
	}

	// 2. Tier 2: Equivalent representations (x + 0 and x)
	zero := ast.NewRationalFromBigRat(big.NewRat(0, 1))
	xNode := ast.NewVar("x")
	xPlusZero := ast.NewAdd([]ast.Node{xNode, zero})

	v2 := EvaluateInvariant(xPlusZero, xNode, ProfileAlgebra)
	if v2 != VerdictExactMatch && v2 != VerdictSymbolicZero {
		t.Fatalf("Expected VerdictExactMatch or VerdictSymbolicZero for (x + 0) == x, got %v", v2)
	}

	// 3. Tier 3: Inequivalent univariate nodes (x + 1 and x) -> VerdictSoundnessBug
	one := ast.NewRationalFromBigRat(big.NewRat(1, 1))
	xPlusOne := ast.NewAdd([]ast.Node{xNode, one})

	v3 := EvaluateInvariant(xPlusOne, xNode, ProfileAlgebra)
	if v3 != VerdictSoundnessBug {
		t.Fatalf("Expected VerdictSoundnessBug for (x + 1) != x, got %v", v3)
	}

	// 4. Tier 3 Multivariate Schwartz-Zippel: asymmetric expressions (x * y^2 vs x^2 * y)
	// When x == y these coincide, so independent variable sampling is required to catch the bug.
	yNode := ast.NewVar("y")
	two := ast.NewRationalFromBigRat(big.NewRat(2, 1))
	yPow2, _ := ast.NewPow(yNode, two)
	xPow2, _ := ast.NewPow(xNode, two)
	xy2 := ast.NewMul([]ast.Node{xNode, yPow2})
	x2y := ast.NewMul([]ast.Node{xPow2, yNode})
	v4 := EvaluateInvariant(xy2, x2y, ProfileAlgebra)
	if v4 != VerdictSoundnessBug {
		t.Fatalf("Expected VerdictSoundnessBug for asymmetric multivariate (x*y^2 != x^2*y), got %v", v4)
	}

	// 5. Tier 3 Sub-1e-9 Exact Rational Precision: x + 1/10^15 vs x
	// Floating-point complex128 with 1e-9 tolerance misses this, but exact Q[i] catches it.
	tinyDenom := new(big.Int).Exp(big.NewInt(10), big.NewInt(15), nil)
	tinyRat := ast.NewRationalFromBigRat(new(big.Rat).SetFrac(big.NewInt(1), tinyDenom))
	subB := substituteVarsMap(ast.NewAdd([]ast.Node{xNode, tinyRat}), map[string]*big.Rat{"x": big.NewRat(2, 3)})
	subA := substituteVarsMap(xNode, map[string]*big.Rat{"x": big.NewRat(2, 3)})
	eb, okB := evalExactGaussianRat(subB)
	ea, okA := evalExactGaussianRat(subA)
	if !okB || !okA || eb.equal(ea) {
		t.Fatalf("Expected exact Q[i] oracle to distinguish x + 10^-15 from x")
	}

	// 6. Tier 3 Unsupported nodes -> VerdictIndeterminate (never silent VerdictFingerprintMatch)
	unknownA, _ := ast.NewFunc("unknown_oracle_fn_a", []ast.Node{xNode})
	unknownB, _ := ast.NewFunc("unknown_oracle_fn_b", []ast.Node{xNode})
	v6 := EvaluateInvariant(unknownA, unknownB, ProfileAlgebra)
	if v6 != VerdictIndeterminate {
		t.Fatalf("Expected VerdictIndeterminate when 0 sample points are evaluatable, got %v", v6)
	}
}

// TestI18nErrorIntegrity tests that AssertI18nErrorIntegrity catches all fmt corruptions and leaked keys.
func TestI18nErrorIntegrity(t *testing.T) {
	// Clean error
	cleanErr := fmt.Errorf("division by zero occurred")
	AssertI18nErrorIntegrity(t, cleanErr)

	corruptedCases := []string{
		"invalid arg: %!s(MISSING)",
		"index error: %!d(BADINDEX)",
		"extra arg: %!(EXTRA int=42)",
		"type mismatch: %!d(string=hello)",
		"errors.err_division_by_zero",
		"iztrans.err_invalid_rational_expr",
		"rsolve.err_non_constant_coeff",
		"lp.err_infeasible_system",
	}

	for _, c := range corruptedCases {
		fakeT := &testing.T{}
		AssertI18nErrorIntegrity(fakeT, fmt.Errorf("%s", c))
		if !fakeT.Failed() {
			t.Errorf("Expected AssertI18nErrorIntegrity to fail on %q, but it passed", c)
		}
	}
}
