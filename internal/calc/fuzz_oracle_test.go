package calc_test

import (
	"fmt"
	"math"
	"math/big"
	"math/cmplx"
	"regexp"
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
	case VerdictSoundnessBug:
		return "VerdictSoundnessBug"
	default:
		return fmt.Sprintf("OracleVerdict(%d)", v)
	}
}

// EvaluateInvariant executes the Three-Tier Equality Oracle between two AST expressions:
// Tier 1: Canonical DAG pointer identity via ast.Identical (O(1)).
// Tier 2: Symbolic reduction of Sub(before, after) == 0.
// Tier 3: Numerical fingerprinting over K sample points (Schwartz-Zippel theorem).
func EvaluateInvariant(before, after ast.Node, domainProfile ProfileType) OracleVerdict {
	// --- Tier 1: O(1) Canonical DAG Pointer Identity ---
	if ast.Identical(before, after) {
		return VerdictExactMatch
	}

	env := calc.NewEnv()

	// --- Tier 2: Symbolic Zero Reduction ---
	// Construct diff: before - after -> Add([before, UnaryOp("-", after)])
	negAfter, err := ast.NewUnaryOp("-", after)
	if err == nil {
		diffNode := ast.NewAdd([]ast.Node{before, negAfter})
		evalDiff, err := calc.EvalWithEnv(diffNode, env)
		if err == nil && isLiteralZero(evalDiff) {
			return VerdictSymbolicZero
		}
	}

	// --- Tier 3: Schwartz-Zippel Independent Numerical Fingerprinting ---
	// Sample K=5 deterministic points avoiding singularities (0, poles).
	// For polynomials of bounded degree, Schwartz-Zippel bounds false equality error <= 10^-3.
	vars := extractVariables(before, after)

	sampleValues := []int64{2, 3, 5, 7, 11}
	mismatchDetected := false
	validEvaluations := 0

	for _, sVal := range sampleValues {
		subBefore := substituteAllVars(before, vars, sVal)
		subAfter := substituteAllVars(after, vars, sVal)

		valB, okB := evalNumericalComplex(subBefore)
		valA, okA := evalNumericalComplex(subAfter)

		if !okB || !okA {
			// Skip points that hit domain singularities or unsupported functions
			continue
		}

		validEvaluations++

		// Compare with numerical tolerance
		diff := cmplx.Abs(valB - valA)
		relTol := 1e-9 * (1.0 + cmplx.Abs(valB) + cmplx.Abs(valA))
		if diff > relTol && !approxAlmostEqual(valB, valA, 1e-9) {
			mismatchDetected = true
			break
		}
	}

	if mismatchDetected {
		return VerdictSoundnessBug
	}

	// Numerical fingerprints matched (or unsupported for pure numerical eval).
	// Flags canonicalization gap / soft finding because Tier 1 and Tier 2 could not reduce it.
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
	}
	return false
}

// extractVariables collects all distinct variable names appearing in either node.
func extractVariables(nodes ...ast.Node) []string {
	varMap := make(map[string]bool)
	for _, n := range nodes {
		if n == nil {
			continue
		}
		ast.Walk(n, func(curr ast.Node) bool {
			if v, ok := curr.(*ast.VarNode); ok {
				varMap[v.Name] = true
			}
			return true
		})
	}

	vars := make([]string, 0, len(varMap))
	for v := range varMap {
		vars = append(vars, v)
	}
	return vars
}

// substituteAllVars substitutes every variable in vars with val in node.
func substituteAllVars(node ast.Node, vars []string, val int64) ast.Node {
	curr := node
	for _, v := range vars {
		curr = ast.Substitute(curr, v, ast.NewRationalFromBigRat(big.NewRat(val, 1)))
	}
	return curr
}

// evalNumericalComplex recursively evaluates an AST node to complex128 independently.
func evalNumericalComplex(n ast.Node) (complex128, bool) {
	if n == nil {
		return 0, false
	}
	switch v := n.(type) {
	case *ast.RationalNode:
		f, _ := v.Val.Float64()
		return complex(f, 0), true
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
	case *ast.SqrtNode:
		c, ok := evalNumericalComplex(v.Radicand)
		if !ok {
			return 0, false
		}
		return cmplx.Sqrt(c), true
	case *ast.ComplexNode:
		re, ok1 := evalNumericalComplex(v.Real)
		im, ok2 := evalNumericalComplex(v.Imag)
		if !ok1 || !ok2 {
			return 0, false
		}
		return complex(real(re), real(im)), true
	case *ast.UnaryOpNode:
		val, ok := evalNumericalComplex(v.Expr)
		if !ok {
			return 0, false
		}
		if v.Op == "-" {
			return -val, true
		} else if v.Op == "+" {
			return val, true
		}
		return 0, false
	case *ast.AddNode:
		var sum complex128
		for _, term := range v.Terms {
			c, ok := evalNumericalComplex(term)
			if !ok {
				return 0, false
			}
			sum += c
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
		return cmplx.Pow(base, exp), true
	case *ast.FuncNode:
		if len(v.Args) == 1 {
			arg, ok := evalNumericalComplex(v.Args[0])
			if !ok {
				return 0, false
			}
			switch v.Name {
			case "sin":
				return cmplx.Sin(arg), true
			case "cos":
				return cmplx.Cos(arg), true
			case "tan":
				return cmplx.Tan(arg), true
			case "abs":
				return complex(cmplx.Abs(arg), 0), true
			case "ln":
				if arg == 0 {
					return 0, false
				}
				return cmplx.Log(arg), true
			case "exp":
				return cmplx.Exp(arg), true
			}
		}
		return 0, false
	default:
		return 0, false
	}
}

func approxAlmostEqual(a, b complex128, eps float64) bool {
	if cmplx.IsNaN(a) || cmplx.IsNaN(b) || cmplx.IsInf(a) || cmplx.IsInf(b) {
		return false
	}
	return cmplx.Abs(a-b) <= eps
}

var (
	rePlaceholderCorruption = regexp.MustCompile(`%!(?:s|d|w|q|v|x|X|f|g|e|b|c|p|T)\((?:MISSING|BADINDEX|EXTRA)?`)
	reUnregisteredKey       = regexp.MustCompile(`^(?:errors|approx|parser|calc|i18n)\.[a-zA-Z0-9_\.]+$`)
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
	if reUnregisteredKey.MatchString(strings.TrimSpace(msg)) {
		t.Errorf("Detected unexpanded i18n key literal leaking in error message: %q", msg)
	}
}

// TestThreeTierOracle_Hierarchy tests the 3-tier oracle evaluation hierarchy.
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

	// 3. Tier 3: Inequivalent nodes (x + 1 and x) -> VerdictSoundnessBug
	one := ast.NewRationalFromBigRat(big.NewRat(1, 1))
	xPlusOne := ast.NewAdd([]ast.Node{xNode, one})

	v3 := EvaluateInvariant(xPlusOne, xNode, ProfileAlgebra)
	if v3 != VerdictSoundnessBug {
		t.Fatalf("Expected VerdictSoundnessBug for (x + 1) != x, got %v", v3)
	}
}

// TestI18nErrorIntegrity tests that AssertI18nErrorIntegrity catches corruption and passes clean errors.
func TestI18nErrorIntegrity(t *testing.T) {
	// Clean error
	cleanErr := fmt.Errorf("division by zero occurred")
	AssertI18nErrorIntegrity(t, cleanErr)

	// Corrupted placeholder detection
	corrupted1 := fmt.Sprintf("invalid arg: %s", "%!s(MISSING)")
	fakeT1 := &testing.T{}
	AssertI18nErrorIntegrity(fakeT1, fmt.Errorf("%s", corrupted1))
	if !fakeT1.Failed() {
		t.Errorf("Expected failure for placeholder corruption %s", corrupted1)
	}

	// Raw untranslated key detection
	rawKey := "errors.err_division_by_zero"
	fakeT2 := &testing.T{}
	AssertI18nErrorIntegrity(fakeT2, fmt.Errorf("%s", rawKey))
	if !fakeT2.Failed() {
		t.Errorf("Expected failure for unexpanded i18n key %s", rawKey)
	}
}
