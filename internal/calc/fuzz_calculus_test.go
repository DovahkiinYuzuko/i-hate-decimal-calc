package calc_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// FuzzCalculus_DiffIntegrateInversion tests the Fundamental Theorem of Calculus:
// d/dx ( ∫ f(x) dx ) == f(x) for integrable expressions.
func FuzzCalculus_DiffIntegrateInversion(f *testing.F) {
	// Seed corpus registration
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x02, 0x03, 0x04})
	f.Add([]byte{0x05, 0x01, 0x02, 0x00, 0x02})
	f.Add([]byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60})
	f.Add([]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)
		nodeBudget := 10
		maxDepth := 3

		fNode := provider.GenNode(maxDepth, &nodeBudget, ProfileCalculus)
		if fNode == nil {
			return
		}

		err := RunWithTimeout(500*time.Millisecond, func() error {
			env := calc.NewEnv()

			// Simplify/Evaluate f(x)
			fEval, errF := calc.EvalWithEnv(fNode, env)
			if errF != nil {
				AssertI18nErrorIntegrity(t, errF)
				return nil
			}
			if CountNodes(fEval) > 5000 {
				return nil
			}

			// F = integrate(f(x), x)
			intFunc, errI := ast.NewFunc("integrate", []ast.Node{fEval, ast.NewVar("x")})
			if errI != nil {
				return nil
			}
			intRes, errInt := calc.EvalWithEnv(intFunc, env)
			if errInt != nil {
				// Non-integrable or error: ensure i18n integrity and safely return
				AssertI18nErrorIntegrity(t, errInt)
				return nil
			}
			if CountNodes(intRes) > 5000 {
				return nil
			}

			// If integration returned unevaluated integrate(...) node, skip inversion check safely
			if fn, ok := intRes.(*ast.FuncNode); ok && fn.Name == "integrate" {
				return nil
			}

			// diffRes = diff(F, x)
			diffFunc, errD := ast.NewFunc("diff", []ast.Node{intRes, ast.NewVar("x")})
			if errD != nil {
				return nil
			}
			diffRes, errDiff := calc.EvalWithEnv(diffFunc, env)
			if errDiff != nil {
				AssertI18nErrorIntegrity(t, errDiff)
				return nil
			}
			if CountNodes(diffRes) > 5000 {
				return nil
			}

			// Invariant verification: diff(integrate(f(x), x), x) == f(x)
			verdict := EvaluateInvariant(diffRes, fEval, ProfileCalculus)
			if verdict == VerdictSoundnessBug {
				t.Fatalf("FTC Diff-Integrate inversion violated: d/dx(int(%s)) != %s. Got d/dx = %s",
					fEval.String(), fEval.String(), diffRes.String())
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("DiffIntegrateInversion unexpected error: %v", err)
		}
	})
}

// FuzzCalculus_DiffLinearity tests linearity of differentiation:
// d/dx ( c1 * f(x) + c2 * g(x) ) == c1 * d/dx(f(x)) + c2 * d/dx(g(x)).
func FuzzCalculus_DiffLinearity(f *testing.F) {
	// Seed corpus registration
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x02, 0x05, 0x07, 0x01})
	f.Add([]byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC})
	f.Add([]byte{0xFE, 0xDC, 0xBA, 0x98, 0x76, 0x54})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)
		nodeBudget := 8
		maxDepth := 3

		fNode := provider.GenNode(maxDepth, &nodeBudget, ProfileCalculus)
		gNode := provider.GenNode(maxDepth, &nodeBudget, ProfileCalculus)
		if fNode == nil || gNode == nil {
			return
		}

		c1Num := provider.NextInt(-4, 4)
		c2Num := provider.NextInt(-4, 4)
		c1 := ast.NewRationalFromBigRat(big.NewRat(c1Num, 1))
		c2 := ast.NewRationalFromBigRat(big.NewRat(c2Num, 1))

		err := RunWithTimeout(500*time.Millisecond, func() error {
			env := calc.NewEnv()

			// LHS: diff(c1*f + c2*g, x)
			term1 := ast.NewMul([]ast.Node{c1, fNode})
			term2 := ast.NewMul([]ast.Node{c2, gNode})
			comb := ast.NewAdd([]ast.Node{term1, term2})

			lhsFunc, errL := ast.NewFunc("diff", []ast.Node{comb, ast.NewVar("x")})
			if errL != nil {
				return nil
			}
			lhsRes, errLhs := calc.EvalWithEnv(lhsFunc, env)
			if errLhs != nil {
				AssertI18nErrorIntegrity(t, errLhs)
				return nil
			}
			if CountNodes(lhsRes) > 5000 {
				return nil
			}

			// RHS: c1 * diff(f, x) + c2 * diff(g, x)
			diffF, errDF := ast.NewFunc("diff", []ast.Node{fNode, ast.NewVar("x")})
			if errDF != nil {
				return nil
			}
			resDF, errResDF := calc.EvalWithEnv(diffF, env)
			if errResDF != nil {
				AssertI18nErrorIntegrity(t, errResDF)
				return nil
			}

			diffG, errDG := ast.NewFunc("diff", []ast.Node{gNode, ast.NewVar("x")})
			if errDG != nil {
				return nil
			}
			resDG, errResDG := calc.EvalWithEnv(diffG, env)
			if errResDG != nil {
				AssertI18nErrorIntegrity(t, errResDG)
				return nil
			}

			rhsComb := ast.NewAdd([]ast.Node{
				ast.NewMul([]ast.Node{c1, resDF}),
				ast.NewMul([]ast.Node{c2, resDG}),
			})
			rhsRes, errRhs := calc.EvalWithEnv(rhsComb, env)
			if errRhs != nil {
				AssertI18nErrorIntegrity(t, errRhs)
				return nil
			}
			if CountNodes(rhsRes) > 5000 {
				return nil
			}

			// Invariant verification: LHS == RHS
			verdict := EvaluateInvariant(lhsRes, rhsRes, ProfileCalculus)
			if verdict == VerdictSoundnessBug {
				t.Fatalf("Diff linearity violated: diff(%s) != %s. LHS: %s, RHS: %s",
					comb.String(), rhsComb.String(), lhsRes.String(), rhsRes.String())
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("DiffLinearity unexpected error: %v", err)
		}
	})
}

// FuzzCalculus_ZTransformRoundTrip tests the round-trip invariant of symbolic Z-transform:
// iztrans(ztrans(x[n], n, z), z, n) == x[n].
func FuzzCalculus_ZTransformRoundTrip(f *testing.F) {
	// Seed corpus registration
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x02, 0x03, 0x01})
	f.Add([]byte{0x04, 0x05, 0x02, 0x01})
	f.Add([]byte{0x11, 0x22, 0x33, 0x44, 0x55})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)
		nodeBudget := 8
		maxDepth := 3

		sigNode := provider.GenNode(maxDepth, &nodeBudget, ProfileTransforms)
		if sigNode == nil {
			return
		}

		err := RunWithTimeout(500*time.Millisecond, func() error {
			env := calc.NewEnv()

			evalSig, errSig := calc.EvalWithEnv(sigNode, env)
			if errSig != nil {
				AssertI18nErrorIntegrity(t, errSig)
				return nil
			}
			if CountNodes(evalSig) > 5000 {
				return nil
			}

			// Forward Z-transform: X(z) = ztrans(x[n], n, z)
			ztransFunc, errZ := ast.NewFunc("ztrans", []ast.Node{evalSig, ast.NewVar("n"), ast.NewVar("z")})
			if errZ != nil {
				return nil
			}
			fwdRes, errFwd := calc.EvalWithEnv(ztransFunc, env)
			if errFwd != nil {
				AssertI18nErrorIntegrity(t, errFwd)
				return nil
			}
			if CountNodes(fwdRes) > 5000 {
				return nil
			}

			// Inverse Z-transform: x_rec[n] = iztrans(X(z), z, n)
			iztransFunc, errIZ := ast.NewFunc("iztrans", []ast.Node{fwdRes, ast.NewVar("z"), ast.NewVar("n")})
			if errIZ != nil {
				return nil
			}
			recRes, errRec := calc.EvalWithEnv(iztransFunc, env)
			if errRec != nil {
				AssertI18nErrorIntegrity(t, errRec)
				return nil
			}
			if CountNodes(recRes) > 5000 {
				return nil
			}

			// Invariant verification: recRes == evalSig
			verdict := EvaluateInvariant(recRes, evalSig, ProfileTransforms)
			if verdict == VerdictSoundnessBug {
				t.Fatalf("Z-transform round-trip soundness bug: orig: %s, fwd: %s, recovered: %s",
					evalSig.String(), fwdRes.String(), recRes.String())
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("ZTransformRoundTrip unexpected error: %v", err)
		}
	})
}

// FuzzCalculus_RSolveInvariance tests that the solution y[n] produced by rsolve
// satisfies the linear recurrence relation identically.
func FuzzCalculus_RSolveInvariance(f *testing.F) {
	// Seed corpus registration: deterministic representations of difference equations
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x02, 0x01})
	f.Add([]byte{0x03, 0x01, 0x02, 0x04})
	f.Add([]byte{0x05, 0x02, 0x01, 0x03, 0x01})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)

		// Deterministically construct a valid linear recurrence relation of order 1 or 2
		orderChoice := provider.NextChoice(2) + 1 // 1 or 2
		c1Val := provider.NextInt(-3, 3)
		if c1Val == 0 {
			c1Val = 2
		}

		c2Val := provider.NextInt(-3, 3)
		if c2Val == 0 {
			c2Val = -1
		}

		var eqNode ast.Node
		var initsNode ast.Node
		var constTerm int64
		var init0Val, init1Val int64
		var bCoeff, cCoeff, rhsConst int64

		if orderChoice == 1 {
			// a(n+1) == c1*a(n) + f(n)
			// LHS: a(n+1)
			lhs, _ := ast.NewFunc("a", []ast.Node{
				ast.NewAdd([]ast.Node{ast.NewVar("n"), ast.NewRationalFromBigRat(big.NewRat(1, 1))}),
			})

			// RHS: c1*a(n) + constTerm
			constTerm = provider.NextInt(-2, 2)
			aN, errAN := ast.NewFunc("a", []ast.Node{ast.NewVar("n")})
			if errAN != nil {
				return
			}
			rhsTerms := []ast.Node{
				ast.NewMul([]ast.Node{
					ast.NewRationalFromBigRat(big.NewRat(c1Val, 1)),
					aN,
				}),
			}
			if constTerm != 0 {
				rhsTerms = append(rhsTerms, ast.NewRationalFromBigRat(big.NewRat(constTerm, 1)))
			}
			rhs := ast.NewAdd(rhsTerms)
			eqNode = ast.NewRelOp(lhs, "==", rhs)

			// Initial condition: a(0) == init0Val
			init0Val = provider.NextInt(-3, 4)
			initLHS, errInit0 := ast.NewFunc("a", []ast.Node{ast.NewRationalFromBigRat(big.NewRat(0, 1))})
			if errInit0 != nil {
				return
			}
			initCond := ast.NewRelOp(initLHS, "==", ast.NewRationalFromBigRat(big.NewRat(init0Val, 1)))
			initsNode = ast.NewList([]ast.Node{initCond})
		} else {
			// Order 2: a(n+2) - (r1+r2)*a(n+1) + r1*r2*a(n) == rhsConst
			// Includes distinct roots, repeated roots (r1 == r2, resonance), and negative roots.
			r1 := provider.NextInt(-3, 3)
			if r1 == 0 {
				r1 = 1
			}
			r2 := provider.NextInt(-3, 3)
			if r2 == 0 {
				r2 = 2
			}
			bCoeff = -(r1 + r2)
			cCoeff = r1 * r2
			rhsConst = provider.NextInt(-2, 2)

			lhsTerm1, errT1 := ast.NewFunc("a", []ast.Node{
				ast.NewAdd([]ast.Node{ast.NewVar("n"), ast.NewRationalFromBigRat(big.NewRat(2, 1))}),
			})
			if errT1 != nil {
				return
			}
			aNPlus1, errAN1 := ast.NewFunc("a", []ast.Node{
				ast.NewAdd([]ast.Node{ast.NewVar("n"), ast.NewRationalFromBigRat(big.NewRat(1, 1))}),
			})
			if errAN1 != nil {
				return
			}
			lhsTerm2 := ast.NewMul([]ast.Node{
				ast.NewRationalFromBigRat(big.NewRat(bCoeff, 1)),
				aNPlus1,
			})
			aN, errAN := ast.NewFunc("a", []ast.Node{ast.NewVar("n")})
			if errAN != nil {
				return
			}
			lhsTerm3 := ast.NewMul([]ast.Node{
				ast.NewRationalFromBigRat(big.NewRat(cCoeff, 1)),
				aN,
			})

			lhs := ast.NewAdd([]ast.Node{lhsTerm1, lhsTerm2, lhsTerm3})
			eqNode = ast.NewRelOp(lhs, "==", ast.NewRationalFromBigRat(big.NewRat(rhsConst, 1)))

			init0Val = provider.NextInt(-3, 4)
			init1Val = provider.NextInt(-3, 4)
			init0LHS, errI0 := ast.NewFunc("a", []ast.Node{ast.NewRationalFromBigRat(big.NewRat(0, 1))})
			if errI0 != nil {
				return
			}
			init1LHS, errI1 := ast.NewFunc("a", []ast.Node{ast.NewRationalFromBigRat(big.NewRat(1, 1))})
			if errI1 != nil {
				return
			}
			init0 := ast.NewRelOp(init0LHS, "==", ast.NewRationalFromBigRat(big.NewRat(init0Val, 1)))
			init1 := ast.NewRelOp(init1LHS, "==", ast.NewRationalFromBigRat(big.NewRat(init1Val, 1)))
			initsNode = ast.NewList([]ast.Node{init0, init1})
		}

		err := RunWithTimeout(500*time.Millisecond, func() error {
			env := calc.NewEnv()

			seqTarget, errSeq := ast.NewFunc("a", []ast.Node{ast.NewVar("n")})
			if errSeq != nil {
				return nil
			}
			rsolveCall, errR := ast.NewFunc("rsolve", []ast.Node{eqNode, seqTarget, initsNode})
			if errR != nil {
				return nil
			}

			sol, errSolve := calc.EvalWithEnv(rsolveCall, env)
			if errSolve != nil {
				AssertI18nErrorIntegrity(t, errSolve)
				return nil
			}
			if CountNodes(sol) > MaxOutputASTNodes {
				return nil
			}

			// 1. Independent Oracle Verification: evaluate a(n) at n = 0..6 using exact Q[i] arithmetic
			seqVals := make([]gaussianRat, 7)
			for nVal := int64(0); nVal <= 6; nVal++ {
				subN := ast.Substitute(sol, "n", ast.NewRationalFromBigRat(big.NewRat(nVal, 1)))
				val, ok := evalExactGaussianRat(subN)
				if !ok {
					// Fallback through EvalWithEnv only if symbolic functions remain
					ev, errEv := calc.EvalWithEnv(subN, env)
					if errEv != nil {
						t.Fatalf("Failed to evaluate rsolve solution %s at n=%d: %v", sol.String(), nVal, errEv)
					}
					val, ok = evalExactGaussianRat(ev)
					if !ok {
						t.Fatalf("rsolve solution %s at n=%d did not evaluate to exact rational: %s", sol.String(), nVal, ev.String())
					}
				}
				seqVals[nVal] = val
			}

			if orderChoice == 1 {
				if !seqVals[0].equal(newGaussianRatFromInt(init0Val, 0)) {
					t.Fatalf("Order-1 initial condition violated for %s: expected a(0)=%d, got %v (sol=%s)",
						eqNode.String(), init0Val, seqVals[0].re, sol.String())
				}
				for nVal := 0; nVal <= 4; nVal++ {
					expectedNext := seqVals[nVal].mul(newGaussianRatFromInt(c1Val, 0)).add(newGaussianRatFromInt(constTerm, 0))
					if !seqVals[nVal+1].equal(expectedNext) {
						t.Fatalf("Order-1 recurrence violated at n=%d for %s: a(%d)=%v != %d*a(%d)+%d=%v (sol=%s)",
							nVal, eqNode.String(), nVal+1, seqVals[nVal+1].re, c1Val, nVal, constTerm, expectedNext.re, sol.String())
					}
				}
			} else {
				if !seqVals[0].equal(newGaussianRatFromInt(init0Val, 0)) || !seqVals[1].equal(newGaussianRatFromInt(init1Val, 0)) {
					t.Fatalf("Order-2 initial conditions violated for %s: expected a(0)=%d, a(1)=%d, got a(0)=%v, a(1)=%v (sol=%s)",
						eqNode.String(), init0Val, init1Val, seqVals[0].re, seqVals[1].re, sol.String())
				}
				for nVal := 0; nVal <= 4; nVal++ {
					lhsVal := seqVals[nVal+2].
						add(seqVals[nVal+1].mul(newGaussianRatFromInt(bCoeff, 0))).
						add(seqVals[nVal].mul(newGaussianRatFromInt(cCoeff, 0)))
					rhsExpected := newGaussianRatFromInt(rhsConst, 0)
					if !lhsVal.equal(rhsExpected) {
						t.Fatalf("Order-2 recurrence violated at n=%d for %s: LHS=%v != RHS=%d (sol=%s)",
							nVal, eqNode.String(), lhsVal.re, rhsConst, sol.String())
					}
				}
			}

			// 2. Also verify consistency with internal certificate verifier
			cert, errVerify := calc.VerifyComputation(rsolveCall, sol, env)
			if errVerify != nil {
				AssertI18nErrorIntegrity(t, errVerify)
				return nil
			}

			if cert != nil && !cert.IsVerified {
				t.Fatalf("RSolve solution verification failed: %s, details: %s", eqNode.String(), cert.Details)
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("RSolveInvariance unexpected error: %v", err)
		}
	})
}
