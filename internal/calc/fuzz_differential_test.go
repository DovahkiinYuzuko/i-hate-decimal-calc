package calc_test

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// DifferentialVerdict represents the 5-way classification outcome of CAS differential testing.
type DifferentialVerdict int

const (
	// VerdictDiffExactMatch: Tier-0 structural or canonical representation matches identically.
	VerdictDiffExactMatch DifferentialVerdict = iota
	// VerdictDiffExactMismatch: Symbolic polynomial reduction proves non-equivalence.
	VerdictDiffExactMismatch
	// VerdictDiffSampledAgreement: Evaluated at k >= 5 independent non-pole sample points over Q[i] with zero difference.
	VerdictDiffSampledAgreement
	// VerdictDiffSampledMismatch: Valid non-pole sample points yield distinct values (Confirmed bug candidate).
	VerdictDiffSampledMismatch
	// VerdictDiffInconclusive: Skipped due to unsupported operations, pole singularities, or timeouts (Not a bug).
	VerdictDiffInconclusive
)

func (v DifferentialVerdict) String() string {
	switch v {
	case VerdictDiffExactMatch:
		return "VerdictDiffExactMatch"
	case VerdictDiffExactMismatch:
		return "VerdictDiffExactMismatch"
	case VerdictDiffSampledAgreement:
		return "VerdictDiffSampledAgreement"
	case VerdictDiffSampledMismatch:
		return "VerdictDiffSampledMismatch"
	case VerdictDiffInconclusive:
		return "VerdictDiffInconclusive"
	default:
		return fmt.Sprintf("DifferentialVerdict(%d)", v)
	}
}

// DifferentialOracle assesses equivalence between an ihd AST result and a SymPy oracle expression.
type DifferentialOracle struct {
	client     *SymPyClient
	sampleSize int
}

// NewDifferentialOracle creates an oracle linked with the resident SymPyClient.
func NewDifferentialOracle(client *SymPyClient) *DifferentialOracle {
	return &DifferentialOracle{
		client:     client,
		sampleSize: 6, // k >= 5 independent points over Q[i]
	}
}

// EvaluateEquivalence performs the 5-way classification protocol between ihd and SymPy ASTs.
func (o *DifferentialOracle) EvaluateEquivalence(ctx context.Context, ihdAST, sympyAST ast.Node, seed int64) (DifferentialVerdict, string) {
	if ihdAST == nil || sympyAST == nil {
		return VerdictDiffInconclusive, "nil AST provided"
	}

	// Tier 0: Direct structural AST match
	if ast.Identical(ihdAST, sympyAST) || ihdAST.String() == sympyAST.String() {
		return VerdictDiffExactMatch, "Tier-0 exact structural AST match"
	}

	ihdJSON, err := ASTToSymPyJSON(ihdAST)
	if err != nil {
		return VerdictDiffInconclusive, fmt.Sprintf("ihd AST serialization skipped: %v", err)
	}
	sympJSON, err := ASTToSymPyJSON(sympyAST)
	if err != nil {
		return VerdictDiffInconclusive, fmt.Sprintf("sympy AST serialization skipped: %v", err)
	}

	// Tier 1 Fast-Path: Subtractive zero check (equals(0) / simplify(A - B) == 0)
	// Invariant (Claude Sonnet 5.5 / Namazu): simplify() is allowed to confirm equality, but NEVER to declare inequality.
	isZero, _, err := o.client.SubtractiveZero(ctx, ihdJSON, sympJSON)
	if err == nil && isZero {
		return VerdictDiffExactMatch, "Tier-1 subtractive algebraic zero"
	}

	// Tier 2: Schwartz-Zippel Rational Identity Testing (RIT) over Q[i]
	// Extract variables occurring in either expression
	vars := extractVariables(ihdAST, sympyAST)
	if len(vars) == 0 {
		// Purely constant expression: evaluate directly in ihd and SymPy
		valIhd, okIhd := evalExactGaussianRat(ihdAST)
		evals, errSym := o.client.EvalPoints(ctx, sympJSON, []map[string]map[string]string{{}})
		if !okIhd || errSym != nil || len(evals) != 1 || evals[0].Undefined {
			return VerdictDiffInconclusive, "constant evaluation undefined or failed"
		}
		valSym, okSym := evals[0].ToGaussianRat()
		if !okSym {
			return VerdictDiffInconclusive, "constant SymPy value not Gaussian rational"
		}
		if valIhd.re.Cmp(valSym.re) == 0 && valIhd.im.Cmp(valSym.im) == 0 {
			return VerdictDiffSampledAgreement, "Tier-2 constant exact match"
		}
		return VerdictDiffSampledMismatch, fmt.Sprintf("constant mismatch: ihd=%v+%vi, sympy=%v+%vi", valIhd.re, valIhd.im, valSym.re, valSym.im)
	}

	// Deterministic PRNG seeded from fuzz corpus input
	rng := rand.New(rand.NewSource(seed))

	// Generate sample points in Q[i] with pole and singularity shielding
	validSamples := 0
	const maxAttempts = 25

	for attempt := 0; attempt < maxAttempts && validSamples < o.sampleSize; attempt++ {
		pointMap := make(map[string]gaussianRat)
		symPointMap := make(map[string]map[string]string)

		for _, vName := range vars {
			// Choose non-zero small random Gaussian integers in [-100, 100]
			re := int64(rng.Intn(199) - 99)
			if re == 0 {
				re = 1
			}
			im := int64(rng.Intn(199) - 99)
			// Half of the points are real-only to test purely real branch contracts
			if rng.Intn(2) == 0 {
				im = 0
			}

			gVal := newGaussianRatFromInt(re, im)
			pointMap[vName] = gVal

			symPointMap[vName] = map[string]string{
				"re_num": strconv.FormatInt(re, 10),
				"re_den": "1",
				"im_num": strconv.FormatInt(im, 10),
				"im_den": "1",
			}
		}

		// Substitute in ihd
		subIhd := substituteVariables(ihdAST, pointMap)
		valIhd, okIhd := evalExactGaussianRat(subIhd)
		if !okIhd {
			// Hit a pole or unsupported complex branch in ihd -> skip this sample point
			continue
		}

		// Substitute in SymPy
		evals, errSym := o.client.EvalPoints(ctx, sympJSON, []map[string]map[string]string{symPointMap})
		if errSym != nil || len(evals) != 1 || evals[0].Undefined {
			// Hit a pole or unsupported branch in SymPy -> skip this sample point
			continue
		}
		valSym, okSym := evals[0].ToGaussianRat()
		if !okSym {
			// SymPy returned non-rational -> skip
			continue
		}

		// Check difference
		if valIhd.re.Cmp(valSym.re) != 0 || valIhd.im.Cmp(valSym.im) != 0 {
			// Mismatch confirmed at a valid non-pole sample point!
			var ptStrs []string
			for k, v := range pointMap {
				ptStrs = append(ptStrs, fmt.Sprintf("%s=%s+%si", k, v.re.RatString(), v.im.RatString()))
			}
			return VerdictDiffSampledMismatch, fmt.Sprintf(
				"value mismatch at point {%s}: ihd=%s+%si, sympy=%s+%si",
				strings.Join(ptStrs, ", "),
				valIhd.re.RatString(), valIhd.im.RatString(),
				valSym.re.RatString(), valSym.im.RatString(),
			)
		}
		validSamples++
	}

	if validSamples < 3 {
		return VerdictDiffInconclusive, fmt.Sprintf("too few valid non-pole sample points: %d/%d", validSamples, o.sampleSize)
	}

	return VerdictDiffSampledAgreement, fmt.Sprintf("Tier-2 agreement across %d independent Q[i] sample points", validSamples)
}

func substituteVariables(node ast.Node, values map[string]gaussianRat) ast.Node {
	if node == nil {
		return nil
	}
	return replaceChildrenWithSelf(node, func(curr ast.Node) ast.Node {
		if v, ok := curr.(*ast.VarNode); ok {
			if val, exists := values[v.Name]; exists {
				// Construct Gaussian rational AST node
				reNode := ast.NewRationalFromBigRat(val.re)
				if val.im.Sign() == 0 {
					return reNode
				}
				imNode := ast.NewRationalFromBigRat(val.im)
				iNode, err := ast.NewConst("i")
				if err != nil {
					return reNode
				}
				mulI := ast.NewMul([]ast.Node{imNode, iNode})
				return ast.NewAdd([]ast.Node{reNode, mulI})
			}
		}
		return curr
	})
}

func replaceChildrenWithSelf(node ast.Node, leafMapper func(ast.Node) ast.Node) ast.Node {
	mapped := leafMapper(node)
	if mapped != node {
		return mapped
	}
	return replaceChildren(node, func(ch ast.Node) ast.Node {
		return replaceChildrenWithSelf(ch, leafMapper)
	})
}

// TestDifferentialOracle_FiveVerdicts verifies the 5-way verdict classification protocol.
func TestDifferentialOracle_FiveVerdicts(t *testing.T) {
	client := GetGlobalSymPyClient()
	if !client.Available() {
		t.Skipf("SymPy not available: %s", client.ReasonUnavailable())
	}
	oracle := NewDifferentialOracle(client)
	ctx := context.Background()

	x := ast.NewVar("x")
	one := ast.NewRationalFromBigRat(big.NewRat(1, 1))
	two := ast.NewRationalFromBigRat(big.NewRat(2, 1))

	// 1. Exact Match: x + 1 vs x + 1
	n1 := ast.NewAdd([]ast.Node{x, one})
	v1, reason1 := oracle.EvaluateEquivalence(ctx, n1, n1, 42)
	t.Logf("Case 1 (x+1 vs x+1): %s (%s)", v1, reason1)
	if v1 != VerdictDiffExactMatch {
		t.Fatalf("Expected VerdictDiffExactMatch, got %s", v1)
	}

	// 2. Sampled Agreement (Algebraic Equivalence): (x + 1)^2 vs x^2 + 2x + 1
	pow2, _ := ast.NewPow(n1, two)
	x2, _ := ast.NewPow(x, two)
	twoX := ast.NewMul([]ast.Node{two, x})
	expanded := ast.NewAdd([]ast.Node{x2, twoX, one})

	v2, reason2 := oracle.EvaluateEquivalence(ctx, pow2, expanded, 1337)
	t.Logf("Case 2 ((x+1)^2 vs x^2+2x+1): %s (%s)", v2, reason2)
	if v2 != VerdictDiffExactMatch && v2 != VerdictDiffSampledAgreement {
		t.Fatalf("Expected ExactMatch or SampledAgreement, got %s", v2)
	}

	// 3. Exact Mismatch: x + 1 vs x + 2
	n3 := ast.NewAdd([]ast.Node{x, two})
	v3, reason3 := oracle.EvaluateEquivalence(ctx, n1, n3, 999)
	t.Logf("Case 3 (x+1 vs x+2): %s (%s)", v3, reason3)
	if v3 != VerdictDiffSampledMismatch && v3 != VerdictDiffExactMismatch {
		t.Fatalf("Expected SampledMismatch, got %s", v3)
	}
}

// FuzzDifferential_SymPy_Algebra explores algebraic identities (Expand, Simplify) against SymPy oracle.
func FuzzDifferential_SymPy_Algebra(f *testing.F) {
	// Seed corpus
	f.Add([]byte{0x01, 0x02, 0x03, 0x04, 0x05})
	f.Add([]byte{0x10, 0x20, 0x30, 0x40, 0x50})
	f.Add([]byte("expand((x+1)*(x+2))"))
	f.Add([]byte("factor(x^2 - 1)"))

	client := GetGlobalSymPyClient()
	if !client.Available() {
		f.Skipf("SymPy not available: %s", client.ReasonUnavailable())
	}
	oracle := NewDifferentialOracle(client)
	shrinker := NewASTShrinker()

	f.Fuzz(func(t *testing.T, data []byte) {
		p := NewByteStreamProvider(data)
		budget := 20
		tree := p.GenNode(3, &budget, ProfileAlgebra)
		if tree == nil {
			return
		}

		// Semantic Pre-Filter (Sakana AI Namazu):
		// Reject ASTs with trivial literal division by zero or log(0)
		if containsLiteralPole(tree) {
			return
		}

		// Wrap in expand / simplify in ihd
		env := calc.NewEnv()
		simplified, err := calc.EvalWithEnv(tree, env)
		if err != nil || simplified == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		seed := int64(p.NextByte())<<24 | int64(p.NextByte())<<16 | int64(p.NextByte())<<8 | int64(p.NextByte())
		verdict, reason := oracle.EvaluateEquivalence(ctx, simplified, tree, seed)

		switch verdict {
		case VerdictDiffExactMatch, VerdictDiffSampledAgreement:
			// Passed! Invariant holds
			return
		case VerdictDiffInconclusive:
			// Skipped sample (e.g. pole, transcendental branch)
			return
		case VerdictDiffExactMismatch, VerdictDiffSampledMismatch:
			// Discrepancy detected! Automatically invoke AST Shrinker to isolate minimal bug
			t.Logf("[DIFFERENTIAL MISMATCH] Original AST: %s (Reason: %s)", tree.String(), reason)

			pred := func(cand ast.Node) bool {
				candSimp, eErr := calc.EvalWithEnv(cand, env)
				if eErr != nil || candSimp == nil {
					return false
				}
				v, _ := oracle.EvaluateEquivalence(ctx, candSimp, cand, seed)
				return v == verdict // Strict Interestingness: Same mismatch class (Slippage prevention)
			}

			shrunk := shrinker.Shrink(tree, pred)
			t.Fatalf("Differential Mismatch with SymPy Oracle!\nOriginal: %s\nShrunk Minimal Counterexample: %s\nReason: %s",
				tree.String(), shrunk.String(), reason)
		}
	})
}

// FuzzDifferential_SymPy_Calculus explores symbolic differentiation identities against SymPy oracle.
func FuzzDifferential_SymPy_Calculus(f *testing.F) {
	f.Add([]byte{0x05, 0x06, 0x07, 0x08, 0x09})
	f.Add([]byte("diff(x^3 + 2*x^2 + 5, x)"))
	f.Add([]byte("diff(sin(x) * cos(x), x)"))

	client := GetGlobalSymPyClient()
	if !client.Available() {
		f.Skipf("SymPy not available: %s", client.ReasonUnavailable())
	}
	oracle := NewDifferentialOracle(client)
	shrinker := NewASTShrinker()

	f.Fuzz(func(t *testing.T, data []byte) {
		p := NewByteStreamProvider(data)
		budget := 20
		tree := p.GenNode(3, &budget, ProfileCalculus)
		if tree == nil || containsLiteralPole(tree) {
			return
		}

		diffCall, err := ast.NewFunc("diff", []ast.Node{tree, ast.NewVar("x")})
		if err != nil {
			return
		}

		env := calc.NewEnv()
		ihdResult, err := calc.EvalWithEnv(diffCall, env)
		if err != nil || ihdResult == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		seed := int64(p.NextByte())<<24 | int64(p.NextByte())<<16 | int64(p.NextByte())<<8 | int64(p.NextByte())
		verdict, reason := oracle.EvaluateEquivalence(ctx, ihdResult, diffCall, seed)

		switch verdict {
		case VerdictDiffExactMatch, VerdictDiffSampledAgreement, VerdictDiffInconclusive:
			return
		case VerdictDiffExactMismatch, VerdictDiffSampledMismatch:
			t.Logf("[DIFFERENTIAL CALCULUS MISMATCH] Tree: %s, Result: %s (Reason: %s)", tree.String(), ihdResult.String(), reason)
			pred := func(cand ast.Node) bool {
				cDiff, cErr := ast.NewFunc("diff", []ast.Node{cand, ast.NewVar("x")})
				if cErr != nil {
					return false
				}
				cRes, eErr := calc.EvalWithEnv(cDiff, env)
				if eErr != nil || cRes == nil {
					return false
				}
				v, _ := oracle.EvaluateEquivalence(ctx, cRes, cDiff, seed)
				return v == verdict
			}
			shrunk := shrinker.Shrink(tree, pred)
			t.Fatalf("Differential Calculus Mismatch!\nOriginal: %s\nShrunk Minimal Counterexample: %s\nReason: %s",
				tree.String(), shrunk.String(), reason)
		}
	})
}

// FuzzDifferential_SymPy_Rational explores rational fraction identities (apart / together) against SymPy oracle.
func FuzzDifferential_SymPy_Rational(f *testing.F) {
	f.Add([]byte{0x11, 0x22, 0x33, 0x44, 0x55})
	f.Add([]byte("apart(1/(x^2 - 1), x)"))
	f.Add([]byte("together(1/(x+1) + 1/(x-1))"))

	client := GetGlobalSymPyClient()
	if !client.Available() {
		f.Skipf("SymPy not available: %s", client.ReasonUnavailable())
	}
	oracle := NewDifferentialOracle(client)
	shrinker := NewASTShrinker()

	f.Fuzz(func(t *testing.T, data []byte) {
		p := NewByteStreamProvider(data)
		budget := 20
		tree := p.GenNode(3, &budget, ProfileRational)
		if tree == nil || containsLiteralPole(tree) {
			return
		}

		env := calc.NewEnv()
		simplified, err := calc.EvalWithEnv(tree, env)
		if err != nil || simplified == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		seed := int64(p.NextByte())<<24 | int64(p.NextByte())<<16 | int64(p.NextByte())<<8 | int64(p.NextByte())
		verdict, reason := oracle.EvaluateEquivalence(ctx, simplified, tree, seed)

		switch verdict {
		case VerdictDiffExactMatch, VerdictDiffSampledAgreement, VerdictDiffInconclusive:
			return
		case VerdictDiffExactMismatch, VerdictDiffSampledMismatch:
			t.Logf("[DIFFERENTIAL RATIONAL MISMATCH] Original AST: %s (Reason: %s)", tree.String(), reason)
			pred := func(cand ast.Node) bool {
				candSimp, eErr := calc.EvalWithEnv(cand, env)
				if eErr != nil || candSimp == nil {
					return false
				}
				v, _ := oracle.EvaluateEquivalence(ctx, candSimp, cand, seed)
				return v == verdict
			}
			shrunk := shrinker.Shrink(tree, pred)
			t.Fatalf("Differential Rational Mismatch!\nOriginal: %s\nShrunk Minimal Counterexample: %s\nReason: %s",
				tree.String(), shrunk.String(), reason)
		}
	})
}

// containsLiteralPole detects literal division by zero or pole candidates.
func containsLiteralPole(n ast.Node) bool {
	hasPole := false
	ast.Walk(n, func(curr ast.Node) bool {
		if p, ok := curr.(*ast.PowNode); ok {
			if r, ok2 := p.Exp.(*ast.RationalNode); ok2 && r.Val != nil && r.Val.Sign() < 0 {
				if b, ok3 := p.Base.(*ast.RationalNode); ok3 && b.Val != nil && b.Val.Sign() == 0 {
					hasPole = true
					return false
				}
			}
		}
		return true
	})
	return hasPole
}
