package calc_test

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// FuzzAlgebra_SimplifyIdempotence tests that AST simplification via Eval is idempotent:
// simplify(simplify(e)) == simplify(e).
func FuzzAlgebra_SimplifyIdempotence(f *testing.F) {
	// Seed corpus registration
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x02, 0x03, 0x04})
	f.Add([]byte{0x05, 0x09, 0x00, 0x01})
	f.Add([]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02, 0x03, 0x04})
	f.Add([]byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)
		nodeBudget := 12
		maxDepth := 4

		// Profile selection: polynomial or rational
		profile := ProfilePolynomial
		if provider.NextBool() {
			profile = ProfileRational
		}

		node := provider.GenNode(maxDepth, &nodeBudget, profile)
		if node == nil {
			return
		}

		err := RunWithTimeout(500*time.Millisecond, func() error {
			env := calc.NewEnv()

			// First simplification: e1 = simplify(e)
			e1, err1 := calc.EvalWithEnv(node, env)
			if err1 != nil {
				AssertI18nErrorIntegrity(t, err1)
				return nil
			}
			if CountNodes(e1) > 5000 {
				return nil
			}

			// Second simplification: e2 = simplify(e1)
			e2, err2 := calc.EvalWithEnv(e1, env)
			if err2 != nil {
				AssertI18nErrorIntegrity(t, err2)
				return nil
			}
			if CountNodes(e2) > 5000 {
				return nil
			}

			// Invariant verification: e1 and e2 must be structurally / algebraically equivalent
			verdict := EvaluateInvariant(e1, e2, profile)
			if verdict == VerdictSoundnessBug {
				t.Fatalf("Idempotence violated: simplify(simplify(%s)) != simplify(%s). e1: %s, e2: %s",
					node.String(), node.String(), e1.String(), e2.String())
			}

			// Check Tier 1 strict identical match or Equal
			if !ast.Identical(e1, e2) && !e1.Equal(e2) {
				t.Fatalf("Simplifier idempotence canonical DAG or AST mismatch: e1: %s, e2: %s",
					e1.String(), e2.String())
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("Simplifier idempotence unexpected error: %v", err)
		}
	})
}

// FuzzAlgebra_FactorExpandRoundTrip tests the round-trip invariant: expand(factor(P)) == P
// for polynomial expressions.
func FuzzAlgebra_FactorExpandRoundTrip(f *testing.F) {
	// Seed corpus registration
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x02, 0x03, 0x04})
	f.Add([]byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80})
	f.Add([]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x00, 0x11})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)
		nodeBudget := 12
		maxDepth := 4

		polyNode := provider.GenNode(maxDepth, &nodeBudget, ProfilePolynomial)
		if polyNode == nil {
			return
		}

		err := RunWithTimeout(500*time.Millisecond, func() error {
			env := calc.NewEnv()

			// Evaluate original poly first to ensure well-formed standard expression
			evalPoly, errP := calc.EvalWithEnv(polyNode, env)
			if errP != nil {
				AssertI18nErrorIntegrity(t, errP)
				return nil
			}
			if CountNodes(evalPoly) > 5000 {
				return nil
			}

			// Construct factor(P) AST call
			factorFunc, errF := ast.NewFunc("factor", []ast.Node{evalPoly})
			if errF != nil {
				return nil
			}
			factored, errFactor := calc.EvalWithEnv(factorFunc, env)
			if errFactor != nil {
				AssertI18nErrorIntegrity(t, errFactor)
				return nil
			}
			if CountNodes(factored) > 5000 {
				return nil
			}

			// Construct expand(factor(P)) AST call
			expandFunc, errE := ast.NewFunc("expand", []ast.Node{factored})
			if errE != nil {
				return nil
			}
			expanded, errExpand := calc.EvalWithEnv(expandFunc, env)
			if errExpand != nil {
				AssertI18nErrorIntegrity(t, errExpand)
				return nil
			}
			if CountNodes(expanded) > 5000 {
				return nil
			}

			// Invariant: expand(factor(P)) == evalPoly
			verdict := EvaluateInvariant(evalPoly, expanded, ProfilePolynomial)
			if verdict == VerdictSoundnessBug {
				t.Fatalf("Factor-Expand round-trip soundness bug: orig: %s, factored: %s, expanded: %s",
					evalPoly.String(), factored.String(), expanded.String())
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("FactorExpand round-trip unexpected error: %v", err)
		}
	})
}

// FuzzAlgebra_RationalTogetherApartRoundTrip tests the round-trip invariant: together(apart(R)) == R
// for rational expressions.
func FuzzAlgebra_RationalTogetherApartRoundTrip(f *testing.F) {
	// Seed corpus registration
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x02, 0x04, 0x06, 0x08})
	f.Add([]byte{0x13, 0x37, 0x42, 0x55, 0x66, 0x77, 0x88, 0x99})
	f.Add([]byte{0xCA, 0xFE, 0xBA, 0xBE, 0x01, 0x02, 0x03, 0x04})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)
		nodeBudget := 12
		maxDepth := 4

		ratNode := provider.GenNode(maxDepth, &nodeBudget, ProfileRational)
		if ratNode == nil {
			return
		}

		err := RunWithTimeout(500*time.Millisecond, func() error {
			env := calc.NewEnv()

			// Simplify original rational expression
			evalR, errR := calc.EvalWithEnv(ratNode, env)
			if errR != nil {
				AssertI18nErrorIntegrity(t, errR)
				return nil
			}
			if CountNodes(evalR) > 5000 {
				return nil
			}

			// apart(R)
			apartFunc, errA := ast.NewFunc("apart", []ast.Node{evalR})
			if errA != nil {
				return nil
			}
			aparted, errApart := calc.EvalWithEnv(apartFunc, env)
			if errApart != nil {
				AssertI18nErrorIntegrity(t, errApart)
				return nil
			}
			if CountNodes(aparted) > 5000 {
				return nil
			}

			// together(apart(R))
			togetherFunc, errT := ast.NewFunc("together", []ast.Node{aparted})
			if errT != nil {
				return nil
			}
			togethered, errTog := calc.EvalWithEnv(togetherFunc, env)
			if errTog != nil {
				AssertI18nErrorIntegrity(t, errTog)
				return nil
			}
			if CountNodes(togethered) > 5000 {
				return nil
			}

			// Invariant: together(apart(R)) == evalR
			verdict := EvaluateInvariant(evalR, togethered, ProfileRational)
			if verdict == VerdictSoundnessBug {
				t.Fatalf("Rational together-apart round-trip soundness bug: orig: %s, aparted: %s, togethered: %s",
					evalR.String(), aparted.String(), togethered.String())
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("Rational together-apart round-trip unexpected error: %v", err)
		}
	})
}

// Suppress unused imports
var _ = fmt.Sprintf
var _ = big.NewRat
