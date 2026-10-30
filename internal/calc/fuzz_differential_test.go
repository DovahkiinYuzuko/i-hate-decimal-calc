package calc_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
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
	// VerdictDiffExactMatch: Tier-0 structural match or Tier-1 symbolic zero reduction.
	VerdictDiffExactMatch DifferentialVerdict = iota
	// VerdictDiffExactMismatch: Symbolic reduction proves non-equivalence (non-zero constant diff).
	VerdictDiffExactMismatch
	// VerdictDiffSampledAgreement: Evaluated at k >= 5 independent sample points with zero discrepancy.
	VerdictDiffSampledAgreement
	// VerdictDiffSampledMismatch: Valid sample points yield distinct values (Confirmed bug candidate).
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
		sampleSize: 6, // k >= 5 independent sample points
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
	isZero, isConstantNonzero, simpStr, err := o.client.SubtractiveZero(ctx, ihdJSON, sympJSON)
	if err == nil {
		if isZero {
			return VerdictDiffExactMatch, "Tier-1 subtractive algebraic zero"
		}
		if isConstantNonzero {
			return VerdictDiffExactMismatch, fmt.Sprintf("Tier-1 exact non-zero difference: %s", simpStr)
		}
	}

	// Tier 2: Schwartz-Zippel Rational Identity Testing (RIT) over Q[i]
	vars := extractVariables(ihdAST, sympyAST)
	if len(vars) == 0 {
		// Purely constant expression
		valIhd, okIhd := evalExactGaussianRat(ihdAST)
		evals, errSym := o.client.EvalPoints(ctx, sympJSON, []map[string]map[string]string{{}})
		if okIhd && errSym == nil && len(evals) == 1 && !evals[0].Undefined {
			if valSym, okSym := evals[0].ToGaussianRat(); okSym {
				if valIhd.re.Cmp(valSym.re) == 0 && valIhd.im.Cmp(valSym.im) == 0 {
					return VerdictDiffSampledAgreement, "Tier-2 constant exact rational match"
				}
				return VerdictDiffSampledMismatch, fmt.Sprintf("constant mismatch: ihd=%v+%vi, sympy=%v+%vi", valIhd.re, valIhd.im, valSym.re, valSym.im)
			}
		}

		// Tier 3 Fallback for transcendental constant expressions (e.g. nested radicals, Morrie's law)
		numRes, numErr := o.client.EvalNumerical(ctx, ihdJSON, sympJSON, []map[string]map[string]string{{}}, 1e-12, 1e-9)
		if numErr == nil && len(numRes.PointResults) > 0 && numRes.PointResults[0].Valid {
			if numRes.PointResults[0].Agree {
				return VerdictDiffSampledAgreement, fmt.Sprintf("Tier-3 numerical constant match (residual=%e)", numRes.MaxResidual)
			}
			return VerdictDiffSampledMismatch, fmt.Sprintf("Tier-3 numerical constant mismatch (residual=%e)", numRes.MaxResidual)
		}
		return VerdictDiffInconclusive, "constant evaluation undefined or failed"
	}

	// Deterministic PRNG seeded from fuzz input
	rng := rand.New(rand.NewSource(seed))

	validSamples := 0
	const maxAttempts = 25
	pointsList := make([]map[string]map[string]string, 0, o.sampleSize)

	for attempt := 0; attempt < maxAttempts && validSamples < o.sampleSize; attempt++ {
		pointMap := make(map[string]gaussianRat)
		symPointMap := make(map[string]map[string]string)

		for _, vName := range vars {
			// Bias towards safe first-quadrant wedge Re in [1, 30], Im in [0, 15] to avoid negative real axis branch cuts
			re := int64(rng.Intn(30) + 1)
			im := int64(0)
			if rng.Intn(2) == 1 {
				im = int64(rng.Intn(15) + 1)
			}

			pointMap[vName] = newGaussianRatFromInt(re, im)
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
			// Expression has transcendentals or division by zero -> save point for Tier-3 fallback
			pointsList = append(pointsList, symPointMap)
			continue
		}

		// Substitute in SymPy
		evals, errSym := o.client.EvalPoints(ctx, sympJSON, []map[string]map[string]string{symPointMap})
		if errSym != nil || len(evals) != 1 || evals[0].Undefined {
			pointsList = append(pointsList, symPointMap)
			continue
		}
		valSym, okSym := evals[0].ToGaussianRat()
		if !okSym {
			pointsList = append(pointsList, symPointMap)
			continue
		}

		if valIhd.re.Cmp(valSym.re) != 0 || valIhd.im.Cmp(valSym.im) != 0 {
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
		pointsList = append(pointsList, symPointMap)
	}

	if validSamples >= 5 {
		return VerdictDiffSampledAgreement, fmt.Sprintf("Tier-2 agreement across %d independent Q[i] sample points", validSamples)
	}

	// Tier 3: High-precision numerical evaluation fallback (eliminates silent Inconclusive passes on transcendentals)
	if len(pointsList) >= 5 {
		numRes, numErr := o.client.EvalNumerical(ctx, ihdJSON, sympJSON, pointsList, 1e-12, 1e-9)
		if numErr == nil && numRes.SampleCount >= 5 {
			if numRes.AllAgree {
				return VerdictDiffSampledAgreement, fmt.Sprintf("Tier-3 numerical agreement across %d points (max_residual=%e)", numRes.SampleCount, numRes.MaxResidual)
			}
			return VerdictDiffSampledMismatch, fmt.Sprintf("Tier-3 numerical mismatch (max_residual=%e)", numRes.MaxResidual)
		}
	}

	return VerdictDiffInconclusive, fmt.Sprintf("too few valid non-pole sample points: %d/%d", validSamples, o.sampleSize)
}

func substituteVariables(node ast.Node, values map[string]gaussianRat) ast.Node {
	if node == nil {
		return nil
	}
	return ast.Transform(node, func(curr ast.Node) ast.Node {
		if v, ok := curr.(*ast.VarNode); ok {
			if val, exists := values[v.Name]; exists {
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

// CounterexampleRecord defines the JSON schema for storing minimized differential counterexamples.
type CounterexampleRecord struct {
	SchemaVersion int    `json:"schema_version"`
	ID            string `json:"id"`
	Domain        string `json:"domain"`
	OriginalAST   string `json:"original_ast"`
	ShrunkAST     string `json:"shrunk_ast"`
	Verdict       string `json:"verdict"`
	Reason        string `json:"reason"`
	Seed          int64  `json:"seed"`
	DiscoveredAt  string `json:"discovered_at"`
	Status        string `json:"status"` // "open", "fixed", "wontfix"
}

func saveCounterexample(domain string, orig, shrunk ast.Node, verdict DifferentialVerdict, reason string, seed int64) {
	dir := filepath.Join("test-scripts", "go", "fuzz", "counterexamples", "pending")
	_ = os.MkdirAll(dir, 0755)

	shrunkStr := shrunk.String()
	h := sha256.Sum256([]byte(domain + ":" + shrunkStr))
	id := hex.EncodeToString(h[:8])

	record := CounterexampleRecord{
		SchemaVersion: 1,
		ID:            id,
		Domain:        domain,
		OriginalAST:   orig.String(),
		ShrunkAST:     shrunkStr,
		Verdict:       verdict.String(),
		Reason:        reason,
		Seed:          seed,
		DiscoveredAt:  time.Now().UTC().Format(time.RFC3339),
		Status:        "open",
	}

	bytes, err := json.MarshalIndent(record, "", "  ")
	if err == nil {
		targetFile := filepath.Join(dir, fmt.Sprintf("%s_%s.json", domain, id))
		_ = os.WriteFile(targetFile, bytes, 0644)
	}
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

	// 2. Sampled Agreement: (x + 1)^2 vs x^2 + 2x + 1
	pow2, _ := ast.NewPow(n1, two)
	x2, _ := ast.NewPow(x, two)
	twoX := ast.NewMul([]ast.Node{two, x})
	expanded := ast.NewAdd([]ast.Node{x2, twoX, one})

	v2, reason2 := oracle.EvaluateEquivalence(ctx, pow2, expanded, 1337)
	t.Logf("Case 2 ((x+1)^2 vs x^2+2x+1): %s (%s)", v2, reason2)
	if v2 != VerdictDiffExactMatch && v2 != VerdictDiffSampledAgreement {
		t.Fatalf("Expected ExactMatch or SampledAgreement, got %s", v2)
	}

	// 3. Exact Mismatch: x + 1 vs x + 2 (Difference is -1 != 0)
	n3 := ast.NewAdd([]ast.Node{x, two})
	v3, reason3 := oracle.EvaluateEquivalence(ctx, n1, n3, 999)
	t.Logf("Case 3 (x+1 vs x+2): %s (%s)", v3, reason3)
	if v3 != VerdictDiffExactMismatch && v3 != VerdictDiffSampledMismatch {
		t.Fatalf("Expected ExactMismatch or SampledMismatch, got %s", v3)
	}
}

// TestDifferentialOracle_AdversarialSuite evaluates the classical CAS pitfalls provided by external AI review.
func TestDifferentialOracle_AdversarialSuite(t *testing.T) {
	client := GetGlobalSymPyClient()
	if !client.Available() {
		t.Skipf("SymPy not available: %s", client.ReasonUnavailable())
	}
	oracle := NewDifferentialOracle(client)
	ctx := context.Background()

	x := ast.NewVar("x")
	one := ast.NewRationalFromBigRat(big.NewRat(1, 1))
	two := ast.NewRationalFromBigRat(big.NewRat(2, 1))

	tests := []struct {
		name     string
		lhs      ast.Node
		rhs      ast.Node
		expected DifferentialVerdict
	}{
		{
			name:     "Polynomial Identity: (x-1)*(x+1) vs x^2 - 1",
			lhs:      ast.NewMul([]ast.Node{ast.NewAdd([]ast.Node{x, ast.NewRationalFromBigRat(big.NewRat(-1, 1))}), ast.NewAdd([]ast.Node{x, one})}),
			rhs:      ast.NewAdd([]ast.Node{mustPow(x, two), ast.NewRationalFromBigRat(big.NewRat(-1, 1))}),
			expected: VerdictDiffExactMatch,
		},
		{
			name:     "Degree 5 Geometric Sum: (x^5 - 1)/(x - 1) vs x^4 + x^3 + x^2 + x + 1",
			lhs:      ast.NewMul([]ast.Node{ast.NewAdd([]ast.Node{mustPow(x, ast.NewRationalFromBigRat(big.NewRat(5, 1))), ast.NewRationalFromBigRat(big.NewRat(-1, 1))}), mustPow(ast.NewAdd([]ast.Node{x, ast.NewRationalFromBigRat(big.NewRat(-1, 1))}), ast.NewRationalFromBigRat(big.NewRat(-1, 1)))}),
			rhs:      ast.NewAdd([]ast.Node{mustPow(x, ast.NewRationalFromBigRat(big.NewRat(4, 1))), mustPow(x, ast.NewRationalFromBigRat(big.NewRat(3, 1))), mustPow(x, two), x, one}),
			expected: VerdictDiffSampledAgreement,
		},
		{
			name:     "Trig Identity: sin(x)^2 + cos(x)^2 vs 1",
			lhs:      ast.NewAdd([]ast.Node{mustPow(mustFunc("sin", []ast.Node{x}), two), mustPow(mustFunc("cos", []ast.Node{x}), two)}),
			rhs:      one,
			expected: VerdictDiffExactMatch,
		},
		{
			name:     "Trig Zero Identity: 2*sin(x)^2 - sin(2x)*tan(x) vs 0 (issue #21683)",
			lhs:      ast.NewAdd([]ast.Node{ast.NewMul([]ast.Node{two, mustPow(mustFunc("sin", []ast.Node{x}), two)}), mustUnaryOp("-", ast.NewMul([]ast.Node{mustFunc("sin", []ast.Node{ast.NewMul([]ast.Node{two, x})}), mustFunc("tan", []ast.Node{x})}))}),
			rhs:      ast.NewRationalFromBigRat(big.NewRat(0, 1)),
			expected: VerdictDiffSampledAgreement,
		},
		{
			name: "Nested Radical (Ramanujan-like): sqrt(3 + 2*sqrt(2)) vs sqrt(2) + 1",
			lhs: ast.NewSqrt(ast.NewAdd([]ast.Node{
				ast.NewRationalFromBigRat(big.NewRat(3, 1)),
				ast.NewMul([]ast.Node{two, ast.NewSqrt(two)}),
			})),
			rhs: ast.NewAdd([]ast.Node{
				ast.NewSqrt(two),
				one,
			}),
			expected: VerdictDiffSampledAgreement,
		},
		{
			name:     "Non-zero Discrepancy: x vs x + 1",
			lhs:      x,
			rhs:      ast.NewAdd([]ast.Node{x, one}),
			expected: VerdictDiffExactMismatch,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, reason := oracle.EvaluateEquivalence(ctx, tc.lhs, tc.rhs, 42)
			t.Logf("[%s] -> Verdict: %s (Reason: %s)", tc.name, v, reason)
			if tc.expected == VerdictDiffExactMatch {
				if v != VerdictDiffExactMatch && v != VerdictDiffSampledAgreement {
					t.Fatalf("Expected agreement, got %s (%s)", v, reason)
				}
			} else if tc.expected == VerdictDiffSampledAgreement {
				if v != VerdictDiffSampledAgreement && v != VerdictDiffExactMatch {
					t.Fatalf("Expected agreement, got %s (%s)", v, reason)
				}
			} else if tc.expected == VerdictDiffExactMismatch {
				if v != VerdictDiffExactMismatch && v != VerdictDiffSampledMismatch {
					t.Fatalf("Expected mismatch, got %s (%s)", v, reason)
				}
			}
		})
	}
}

func mustPow(base, exp ast.Node) ast.Node {
	p, err := ast.NewPow(base, exp)
	if err != nil {
		panic(err)
	}
	return p
}

func mustFunc(name string, args []ast.Node) ast.Node {
	f, err := ast.NewFunc(name, args)
	if err != nil {
		panic(err)
	}
	return f
}

func mustUnaryOp(op string, expr ast.Node) ast.Node {
	uop, err := ast.NewUnaryOp(op, expr)
	if err != nil {
		panic(err)
	}
	return uop
}

// TestDifferentialOracle_RegressionLedger reads and validates all counterexamples saved in counterexamples/.
func TestDifferentialOracle_RegressionLedger(t *testing.T) {
	ledgerDir := filepath.Join("test-scripts", "go", "fuzz", "counterexamples")
	files, err := filepath.Glob(filepath.Join(ledgerDir, "*.json"))
	if err != nil || len(files) == 0 {
		t.Log("No confirmed counterexample ledger fixtures found (clean state).")
		return
	}

	client := GetGlobalSymPyClient()
	if !client.Available() {
		t.Skipf("SymPy not available: %s", client.ReasonUnavailable())
	}
	oracle := NewDifferentialOracle(client)
	ctx := context.Background()

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			bytes, rErr := os.ReadFile(file)
			if rErr != nil {
				t.Fatalf("Failed to read fixture: %v", rErr)
			}
			var record CounterexampleRecord
			if uErr := json.Unmarshal(bytes, &record); uErr != nil {
				t.Fatalf("Failed to unmarshal fixture: %v", uErr)
			}
			if record.Status == "wontfix" {
				t.Skip("Marked wontfix by specification.")
			}

			ihdNode, err1 := calc.Parse(record.ShrunkAST)
			if err1 != nil {
				t.Fatalf("Failed to parse shrunk AST '%s': %v", record.ShrunkAST, err1)
			}
			env := calc.NewEnv()
			simplified, err2 := calc.EvalWithEnv(ihdNode, env)
			if err2 != nil {
				t.Fatalf("Eval error on shrunk AST: %v", err2)
			}

			v, reason := oracle.EvaluateEquivalence(ctx, simplified, ihdNode, record.Seed)
			t.Logf("Regression fixture %s -> verdict: %s (%s)", record.ID, v, reason)
			if record.Status == "fixed" {
				if v != VerdictDiffExactMatch && v != VerdictDiffSampledAgreement {
					t.Fatalf("Regression failure on fixed fixture %s: expected agreement, got %s", record.ID, v)
				}
			}
		})
	}
}

// FuzzDifferential_SymPy_Algebra explores algebraic identities against SymPy oracle.
func FuzzDifferential_SymPy_Algebra(f *testing.F) {
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
			t.Logf("[DIFFERENTIAL MISMATCH] Original AST: %s (Reason: %s)", tree.String(), reason)

			// Fresh context for shrinker with full budget duration
			shrinkCtx, shrinkCancel := context.WithTimeout(context.Background(), shrinker.MaxDuration)
			defer shrinkCancel()

			pred := func(cand ast.Node) bool {
				candSimp, eErr := calc.EvalWithEnv(cand, env)
				if eErr != nil || candSimp == nil {
					return false
				}
				v, _ := oracle.EvaluateEquivalence(shrinkCtx, candSimp, cand, seed)
				return v == verdict
			}

			shrunk := shrinker.Shrink(tree, pred)
			saveCounterexample("algebra", tree, shrunk, verdict, reason, seed)
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

			shrinkCtx, shrinkCancel := context.WithTimeout(context.Background(), shrinker.MaxDuration)
			defer shrinkCancel()

			pred := func(cand ast.Node) bool {
				cDiff, cErr := ast.NewFunc("diff", []ast.Node{cand, ast.NewVar("x")})
				if cErr != nil {
					return false
				}
				cRes, eErr := calc.EvalWithEnv(cDiff, env)
				if eErr != nil || cRes == nil {
					return false
				}
				v, _ := oracle.EvaluateEquivalence(shrinkCtx, cRes, cDiff, seed)
				return v == verdict
			}
			shrunk := shrinker.Shrink(tree, pred)
			saveCounterexample("calculus", tree, shrunk, verdict, reason, seed)
			t.Fatalf("Differential Calculus Mismatch!\nOriginal: %s\nShrunk Minimal Counterexample: %s\nReason: %s",
				tree.String(), shrunk.String(), reason)
		}
	})
}

// FuzzDifferential_SymPy_Rational explores rational fraction identities against SymPy oracle.
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

			shrinkCtx, shrinkCancel := context.WithTimeout(context.Background(), shrinker.MaxDuration)
			defer shrinkCancel()

			pred := func(cand ast.Node) bool {
				candSimp, eErr := calc.EvalWithEnv(cand, env)
				if eErr != nil || candSimp == nil {
					return false
				}
				v, _ := oracle.EvaluateEquivalence(shrinkCtx, candSimp, cand, seed)
				return v == verdict
			}
			shrunk := shrinker.Shrink(tree, pred)
			saveCounterexample("rational", tree, shrunk, verdict, reason, seed)
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
