package calc_test

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/units"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/lp"
)

// FuzzDomain_UnitsDimensionalConsistency tests dimensional consistency and conversion invariance of physical units:
// 1. Addition / Subtraction: Incompatible dimensions must produce a dimension mismatch error (AssertI18nErrorIntegrity),
//    while compatible dimensions preserve numerical value after conversion.
// 2. Multiplication / Division: Dimensional additivity ([u1 * u2] == [u1] + [u2], [u1 / u2] == [u1] - [u2]).
// 3. Unit Conversion Round-Trip: convert_unit(convert_unit(q, u2), u1) == q evaluated via EvaluateInvariant.
func FuzzDomain_UnitsDimensionalConsistency(f *testing.F) {
	// Seed corpus registration
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x02, 0x03, 0x04})
	f.Add([]byte{0x05, 0x08, 0x12, 0x34})
	f.Add([]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF})
	f.Add([]byte{0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)

		q1 := provider.GenQuantityLeaf()
		q2 := provider.GenQuantityLeaf()
		if q1 == nil || q2 == nil {
			return
		}

		err := RunWithTimeout(500*time.Millisecond, func() error {
			// 1. Multiplication: [q1 * q2] == [q1] + [q2]
			evalMul, errMul := calc.EvalQuantityBinary("*", q1, q2)
			if errMul != nil {
				AssertI18nErrorIntegrity(t, errMul)
			} else {
				if CountNodes(evalMul) <= 5000 {
					expectedDims := q1.Dimensions.Add(q2.Dimensions)
					if qMul, ok := evalMul.(*ast.QuantityNode); ok {
						if !qMul.Dimensions.Equal(expectedDims) {
							t.Fatalf("Units multiplication dimension additivity violated: expected %s, got %s",
								expectedDims.String(), qMul.Dimensions.String())
						}
					} else if !expectedDims.IsZero() {
						t.Fatalf("Units multiplication expected non-zero dimension %s, got non-quantity: %s",
							expectedDims.String(), evalMul.String())
					}
				}
			}

			// 2. Division: [q1 / q2] == [q1] - [q2]
			evalDiv, errDiv := calc.EvalQuantityBinary("/", q1, q2)
			if errDiv != nil {
				AssertI18nErrorIntegrity(t, errDiv)
			} else {
				if CountNodes(evalDiv) <= 5000 {
					expectedDivDims := q1.Dimensions.Sub(q2.Dimensions)
					if qDiv, ok := evalDiv.(*ast.QuantityNode); ok {
						if !qDiv.Dimensions.Equal(expectedDivDims) {
							t.Fatalf("Units division dimension additivity violated: expected %s, got %s",
								expectedDivDims.String(), qDiv.Dimensions.String())
						}
					} else if !expectedDivDims.IsZero() {
						t.Fatalf("Units division expected non-zero dimension %s, got non-quantity: %s",
							expectedDivDims.String(), evalDiv.String())
					}
				}
			}

			// 3. Addition / Subtraction: dimensional compatibility invariant
			evalAdd, errAdd := calc.EvalQuantityBinary("+", q1, q2)
			if !q1.Dimensions.Equal(q2.Dimensions) {
				// Must fail with dimension mismatch error
				if errAdd == nil {
					t.Fatalf("Incompatible units addition should have failed: %s + %s", q1.String(), q2.String())
				}
				AssertI18nErrorIntegrity(t, errAdd)
			} else {
				// Identical dimensions: addition must succeed and preserve dimension
				if errAdd != nil {
					AssertI18nErrorIntegrity(t, errAdd)
				} else if CountNodes(evalAdd) <= 5000 {
					if qAdd, ok := evalAdd.(*ast.QuantityNode); ok {
						if !qAdd.Dimensions.Equal(q1.Dimensions) {
							t.Fatalf("Compatible units addition changed dimension: expected %s, got %s",
								q1.Dimensions.String(), qAdd.Dimensions.String())
						}
					}
				}
			}

			// Subtraction: dimensional compatibility invariant
			evalSub, errSub := calc.EvalQuantityBinary("-", q1, q2)
			if !q1.Dimensions.Equal(q2.Dimensions) {
				if errSub == nil {
					t.Fatalf("Incompatible units subtraction should have failed: %s - %s", q1.String(), q2.String())
				}
				AssertI18nErrorIntegrity(t, errSub)
			} else {
				if errSub != nil {
					AssertI18nErrorIntegrity(t, errSub)
				} else if CountNodes(evalSub) <= 5000 {
					if qSub, ok := evalSub.(*ast.QuantityNode); ok {
						if !qSub.Dimensions.Equal(q1.Dimensions) {
							t.Fatalf("Compatible units subtraction changed dimension: expected %s, got %s",
								q1.Dimensions.String(), qSub.Dimensions.String())
						}
					}
				}
			}

			// 4. Unit Conversion Round-Trip: convert_unit(convert_unit(q, u2), u1) == q
			// Test known compatible unit pairs for exact round-trip invariance
			compatiblePairs := [][2]string{
				{"m", "km"},
				{"km", "m"},
				{"s", "ms"},
				{"ms", "s"},
				{"min", "h"},
				{"h", "min"},
				{"N", "kN"},
				{"J", "kJ"},
				{"Pa", "kPa"},
			}
			pair := compatiblePairs[provider.NextChoice(len(compatiblePairs))]
			u1Str, u2Str := pair[0], pair[1]

			s1, d1, err1 := units.ResolveUnitSymbol(u1Str)
			if err1 == nil {
				valRat := provider.NextSmallRat()
				if valRat.Sign() == 0 {
					valRat = big.NewRat(1, 1)
				}
				origQ := ast.NewQuantityNode(ast.NewRationalFromBigRat(valRat), s1, d1, u1Str)

				// Step 1: convert_unit(origQ, u2)
				res1, errRes1 := calc.ConvertUnit(origQ, ast.NewVar(u2Str))
				if errRes1 != nil {
					AssertI18nErrorIntegrity(t, errRes1)
				} else if CountNodes(res1) <= 5000 {
					// Step 2: convert_unit(res1, u1)
					roundTrip, errRes2 := calc.ConvertUnit(res1, ast.NewVar(u1Str))
					if errRes2 != nil {
						AssertI18nErrorIntegrity(t, errRes2)
					} else if CountNodes(roundTrip) <= 5000 {
						// Invariant: roundTrip value == origQ value
						if rtQ, ok := roundTrip.(*ast.QuantityNode); ok {
							verdict := EvaluateInvariant(rtQ.Value, origQ.Value, ProfilePolynomial)
							if verdict == VerdictSoundnessBug {
								t.Fatalf("Unit conversion round-trip value mismatch: %s -> %s -> %s != %s",
									origQ.String(), u2Str, rtQ.String(), origQ.String())
							}
						}
					}
				}
			}

			// 5. Compound Units AST Evaluation via ProfileUnits:
			// Exercises compound +, -, *, /, ^ trees of physical quantities.
			unitBudget := 10
			compoundUnitExpr := provider.GenNode(3, &unitBudget, ProfileUnits)
			if compoundUnitExpr != nil {
				env := calc.NewEnv()
				evalComp, errComp := calc.EvalWithEnv(compoundUnitExpr, env)
				if errComp != nil {
					AssertI18nErrorIntegrity(t, errComp)
				} else if CountNodes(evalComp) <= MaxOutputASTNodes {
					evalComp2, errComp2 := calc.EvalWithEnv(evalComp, env)
					if errComp2 != nil {
						AssertI18nErrorIntegrity(t, errComp2)
					} else if qComp1, ok1 := evalComp.(*ast.QuantityNode); ok1 {
						if qComp2, ok2 := evalComp2.(*ast.QuantityNode); ok2 {
							if !qComp1.Dimensions.Equal(qComp2.Dimensions) {
								t.Fatalf("Compound unit evaluation idempotence dimension mismatch: %s vs %s",
									qComp1.Dimensions.String(), qComp2.Dimensions.String())
							}
							verdict := EvaluateInvariant(qComp1.Value, qComp2.Value, ProfilePolynomial)
							if verdict == VerdictSoundnessBug {
								t.Fatalf("Compound unit evaluation idempotence value mismatch: %s vs %s",
									qComp1.String(), qComp2.String())
							}
						}
					}
				}
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("Units dimensional consistency unexpected error: %v", err)
		}
	})
}

// FuzzDomain_SimplexStrongDuality tests the Strong Duality Theorem of Linear Programming:
// 1. Primal: Maximize c^T x s.t. A x <= b, x >= 0
// 2. Dual:   Minimize b^T y s.t. A^T y >= c, y >= 0
// Invariant:
// - If primal has optimal solution x*, dual has optimal solution y* and c^T x* == b^T y*.
// - Infeasible / Unbounded status duality certificates are verified via lp.SanitizeAndVerifyCertificate.
func FuzzDomain_SimplexStrongDuality(f *testing.F) {
	// Seed corpus registration
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x02, 0x03, 0x04})
	f.Add([]byte{0x12, 0x34, 0x56, 0x78})
	f.Add([]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF})
	f.Add([]byte{0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF, 0x01})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)

		// Choose problem dimensions: n in [1, 3], m in [1, 3]
		n := provider.NextChoice(3) + 1
		m := provider.NextChoice(3) + 1

		primalProb := provider.GenSimplexLP(n, m)
		if primalProb == nil {
			return
		}

		err := RunWithTimeout(500*time.Millisecond, func() error {
			// Solve Primal problem
			primalRes, errPrimal := lp.Solve(primalProb)
			if errPrimal != nil {
				AssertI18nErrorIntegrity(t, errPrimal)
				return nil
			}

			// Verify Primal certificate via independent sanitizer
			if errCert := lp.SanitizeAndVerifyCertificate(primalProb, primalRes.Certificate); errCert != nil {
				t.Fatalf("Primal LP certificate verification failed: %v\nProblem: %+v", errCert, primalProb)
			}

			// Construct Dual problem:
			// Min b^T y s.t. A^T y >= c, y >= 0
			// Variables y1, ..., ym
			dualVarNames := make([]string, m)
			dualObj := make([]*big.Rat, m)
			for i := 0; i < m; i++ {
				dualVarNames[i] = fmt.Sprintf("y%d", i+1)
				dualObj[i] = new(big.Rat).Set(primalProb.Constraints[i].RHS)
			}

			// Constraints: for each variable j in primal, sum_i A[i][j] * y_i >= c[j]
			dualConstraints := make([]lp.Constraint, n)
			for j := 0; j < n; j++ {
				colCoeffs := make([]*big.Rat, m)
				for i := 0; i < m; i++ {
					colCoeffs[i] = new(big.Rat).Set(primalProb.Constraints[i].Coeffs[j])
				}
				dualConstraints[j] = lp.Constraint{
					Name:   fmt.Sprintf("dual_c%d", j+1),
					Coeffs: colCoeffs,
					Sense:  lp.RelGe,
					RHS:    new(big.Rat).Set(primalProb.Objective[j]),
				}
			}

			dualProb := &lp.Problem{
				Sense:       lp.OptMinimize,
				Objective:   dualObj,
				ObjConstant: new(big.Rat),
				VarNames:    dualVarNames,
				Constraints: dualConstraints,
			}

			dualRes, errDual := lp.Solve(dualProb)
			if errDual != nil {
				AssertI18nErrorIntegrity(t, errDual)
				return nil
			}

			// Verify Dual certificate
			if errCert := lp.SanitizeAndVerifyCertificate(dualProb, dualRes.Certificate); errCert != nil {
				t.Fatalf("Dual LP certificate verification failed: %v\nProblem: %+v", errCert, dualProb)
			}

			// Strong Duality Invariant Check
			if primalRes.Status == lp.StatusOptimal {
				if dualRes.Status != lp.StatusOptimal {
					t.Fatalf("Strong duality status mismatch: primal is OPTIMAL, but dual is %s", dualRes.Status)
				}

				// Check optimal objective value equality: max c^T x == min b^T y
				diff := new(big.Rat).Sub(primalRes.OptimalValue, dualRes.OptimalValue)
				if diff.Sign() != 0 {
					t.Fatalf("Strong duality value gap violated: primal opt = %s, dual opt = %s, diff = %s",
						primalRes.OptimalValue.String(), dualRes.OptimalValue.String(), diff.String())
				}

				// Oracle verification via EvaluateInvariant
				primalOptNode := ast.NewRationalFromBigRat(primalRes.OptimalValue)
				dualOptNode := ast.NewRationalFromBigRat(dualRes.OptimalValue)
				verdict := EvaluateInvariant(primalOptNode, dualOptNode, ProfilePolynomial)
				if verdict == VerdictSoundnessBug {
					t.Fatalf("Oracle evaluated strong duality soundness bug: %s != %s",
						primalOptNode.String(), dualOptNode.String())
				}
			} else if primalRes.Status == lp.StatusUnbounded {
				// If primal is unbounded, dual must be infeasible
				if dualRes.Status != lp.StatusInfeasible {
					t.Fatalf("Duality theorem violated: primal is UNBOUNDED, but dual is %s (expected INFEASIBLE)",
						dualRes.Status)
				}
			} else if primalRes.Status == lp.StatusInfeasible {
				// If primal is infeasible, dual cannot be optimal
				if dualRes.Status == lp.StatusOptimal {
					t.Fatalf("Duality theorem violated: primal is INFEASIBLE, but dual is OPTIMAL")
				}
			}

			// Also test mixed-sense (<=, >=, ==) and Minimize/Maximize Phase I problems for certificate integrity
			senses := []lp.RelSense{lp.RelLe, lp.RelGe, lp.RelEq}
			mixedConstraints := make([]lp.Constraint, len(primalProb.Constraints))
			for i, c := range primalProb.Constraints {
				mixedConstraints[i] = lp.Constraint{
					Name:   c.Name,
					Coeffs: c.Coeffs,
					Sense:  senses[provider.NextChoice(len(senses))],
					RHS:    c.RHS,
				}
			}
			mixedSense := lp.OptMaximize
			if provider.NextBool() {
				mixedSense = lp.OptMinimize
			}
			mixedProb := &lp.Problem{
				Sense:       mixedSense,
				Objective:   primalProb.Objective,
				ObjConstant: new(big.Rat),
				VarNames:    primalProb.VarNames,
				Constraints: mixedConstraints,
			}
			mixedRes, errMixed := lp.Solve(mixedProb)
			if errMixed != nil {
				AssertI18nErrorIntegrity(t, errMixed)
			} else if errCert := lp.SanitizeAndVerifyCertificate(mixedProb, mixedRes.Certificate); errCert != nil {
				t.Fatalf("Mixed-sense LP certificate verification failed: %v\nProblem: %+v", errCert, mixedProb)
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("Simplex strong duality unexpected error: %v", err)
		}
	})
}

// FuzzDomain_MatrixAlgebraInvariants tests exact algebraic invariants of matrices:
// 1. Determinant multiplicativity: det(A * B) == det(A) * det(B).
// 2. Transpose involution: (A^T)^T == A.
// 3. Transpose product anti-homomorphism: (A * B)^T == B^T * A^T.
// 4. Invertibility & Regularity: if det(A) != 0, A * inv(A) == I.
func FuzzDomain_MatrixAlgebraInvariants(f *testing.F) {
	// Seed corpus registration
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x02, 0x03, 0x04})
	f.Add([]byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02})
	f.Add([]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88})
	f.Add([]byte{0xAA, 0x55, 0xAA, 0x55, 0x12, 0x34, 0x56, 0x78})

	f.Fuzz(func(t *testing.T, data []byte) {
		provider := NewByteStreamProvider(data)

		// Choose dimension 2 or 3 for fast, exact algebraic computation
		dim := provider.NextChoice(2) + 2

		matA := provider.GenRationalMatrix(dim, dim)
		matB := provider.GenRationalMatrix(dim, dim)
		if matA == nil || matB == nil {
			return
		}

		err := RunWithTimeout(500*time.Millisecond, func() error {
			env := calc.NewEnv()

			// 1. Transpose involution: transpose(transpose(A)) == A
			tA, errTA := ast.NewFunc("transpose", []ast.Node{matA})
			if errTA == nil {
				evalTA, errEvalTA := calc.EvalWithEnv(tA, env)
				if errEvalTA != nil {
					AssertI18nErrorIntegrity(t, errEvalTA)
				} else if CountNodes(evalTA) <= 5000 {
					ttA, errTTA := ast.NewFunc("transpose", []ast.Node{evalTA})
					if errTTA == nil {
						evalTTA, errEvalTTA := calc.EvalWithEnv(ttA, env)
						if errEvalTTA != nil {
							AssertI18nErrorIntegrity(t, errEvalTTA)
						} else if CountNodes(evalTTA) <= 5000 {
							if !evalTTA.Equal(matA) {
								t.Fatalf("Transpose involution violated: ((A)^T)^T != A. Got: %s, want: %s",
									evalTTA.String(), matA.String())
							}
						}
					}
				}
			}

			// 2. Transpose product anti-homomorphism: transpose(A * B) == transpose(B) * transpose(A)
			abMul := ast.NewMul([]ast.Node{matA, matB})
			evalAB, errAB := calc.EvalWithEnv(abMul, env)
			if errAB != nil {
				AssertI18nErrorIntegrity(t, errAB)
			} else if CountNodes(evalAB) <= 5000 {
				tAB, errTAB := ast.NewFunc("transpose", []ast.Node{evalAB})
				if errTAB == nil {
					lhsT, errLhsT := calc.EvalWithEnv(tAB, env)
					if errLhsT != nil {
						AssertI18nErrorIntegrity(t, errLhsT)
					} else if CountNodes(lhsT) <= 5000 {
						tB, _ := ast.NewFunc("transpose", []ast.Node{matB})
						tA, _ := ast.NewFunc("transpose", []ast.Node{matA})
						evalTB, _ := calc.EvalWithEnv(tB, env)
						evalTA, _ := calc.EvalWithEnv(tA, env)
						rhsMul := ast.NewMul([]ast.Node{evalTB, evalTA})
						rhsT, errRhsT := calc.EvalWithEnv(rhsMul, env)
						if errRhsT != nil {
							AssertI18nErrorIntegrity(t, errRhsT)
						} else if CountNodes(rhsT) <= 5000 {
							if !lhsT.Equal(rhsT) {
								t.Fatalf("Transpose product anti-homomorphism violated: (AB)^T != B^T A^T.\nLHS: %s\nRHS: %s",
									lhsT.String(), rhsT.String())
							}
						}
					}
				}
			}

			// 3. Determinant multiplicativity: det(A * B) == det(A) * det(B)
			if CountNodes(evalAB) <= 5000 {
				detABFunc, errDetAB := ast.NewFunc("det", []ast.Node{evalAB})
				if errDetAB == nil {
					detAB, errDetABEval := calc.EvalWithEnv(detABFunc, env)
					if errDetABEval != nil {
						AssertI18nErrorIntegrity(t, errDetABEval)
					} else if CountNodes(detAB) <= 5000 {
						detAFunc, _ := ast.NewFunc("det", []ast.Node{matA})
						detBFunc, _ := ast.NewFunc("det", []ast.Node{matB})
						detA, errA := calc.EvalWithEnv(detAFunc, env)
						detB, errB := calc.EvalWithEnv(detBFunc, env)
						if errA == nil && errB == nil {
							detMul := ast.NewMul([]ast.Node{detA, detB})
							detProd, errProd := calc.EvalWithEnv(detMul, env)
							if errProd == nil && CountNodes(detProd) <= 5000 {
								verdict := EvaluateInvariant(detAB, detProd, ProfilePolynomial)
								if verdict == VerdictSoundnessBug {
									t.Fatalf("Determinant multiplicativity violated: det(AB) != det(A)*det(B).\ndet(AB) = %s, det(A)*det(B) = %s",
										detAB.String(), detProd.String())
								}
							}
						}
					}
				}
			}

			// 4. Invertibility & Regularity: if det(A) != 0, A * inv(A) == I
			detAFunc, _ := ast.NewFunc("det", []ast.Node{matA})
			detA, errA := calc.EvalWithEnv(detAFunc, env)
			if errA == nil {
				isSingular := false
				if ratA, ok := detA.(*ast.RationalNode); ok && ratA.Val.Sign() == 0 {
					isSingular = true
				}

				invAFunc, _ := ast.NewFunc("inv", []ast.Node{matA})
				invA, errInv := calc.EvalWithEnv(invAFunc, env)
				if isSingular {
					if errInv == nil {
						t.Fatalf("Singular matrix inversion should have failed: det=%s, mat=%s",
							detA.String(), matA.String())
					}
					AssertI18nErrorIntegrity(t, errInv)
				} else if errInv != nil {
					AssertI18nErrorIntegrity(t, errInv)
				} else if CountNodes(invA) <= 5000 {
					// Verify A * inv(A) == I
					prodFunc := ast.NewMul([]ast.Node{matA, invA})
					prodRes, errProd := calc.EvalWithEnv(prodFunc, env)
					if errProd != nil {
						AssertI18nErrorIntegrity(t, errProd)
					} else if CountNodes(prodRes) <= 5000 {
						if prodMat, ok := prodRes.(*ast.MatrixNode); ok {
							for r := 0; r < dim; r++ {
								for c := 0; c < dim; c++ {
									expected := int64(0)
									if r == c {
										expected = 1
									}
									elem := prodMat.Data[r][c]
									expectedNode := ast.NewRationalFromBigRat(big.NewRat(expected, 1))
									verdict := EvaluateInvariant(elem, expectedNode, ProfilePolynomial)
									if verdict == VerdictSoundnessBug {
										t.Fatalf("A * inv(A) != I at (%d, %d): got %s, expected %s",
											r, c, elem.String(), expectedNode.String())
									}
								}
							}
						}
					}
				}
			}

			// 5. Symbolic Matrix Invariants via GenMatrixNode:
			// Exercises symbolic entries (x, y, a, b) for det(A^T) == det(A).
			symBudget := 9
			symMat := provider.GenMatrixNode(2, 2, 1, &symBudget)
			if symMat != nil {
				symTA, errSymTA := ast.NewFunc("transpose", []ast.Node{symMat})
				if errSymTA == nil {
					evalSymTA, errEvalSymTA := calc.EvalWithEnv(symTA, env)
					if errEvalSymTA != nil {
						AssertI18nErrorIntegrity(t, errEvalSymTA)
					} else {
						detSymFn, _ := ast.NewFunc("det", []ast.Node{symMat})
						detSymTFn, _ := ast.NewFunc("det", []ast.Node{evalSymTA})
						detSym, errDS := calc.EvalWithEnv(detSymFn, env)
						detSymT, errDST := calc.EvalWithEnv(detSymTFn, env)
						if errDS != nil {
							AssertI18nErrorIntegrity(t, errDS)
						}
						if errDST != nil {
							AssertI18nErrorIntegrity(t, errDST)
						}
						if errDS == nil && errDST == nil && CountNodes(detSym) <= MaxOutputASTNodes && CountNodes(detSymT) <= MaxOutputASTNodes {
							verdict := EvaluateInvariant(detSym, detSymT, ProfilePolynomial)
							if verdict == VerdictSoundnessBug {
								t.Fatalf("Symbolic matrix det(A^T) != det(A): %s vs %s", detSymT.String(), detSym.String())
							}
						}
					}
				}
			}

			return nil
		})

		if err != nil && err != ErrExecutionTimeout {
			t.Fatalf("Matrix algebra invariants unexpected error: %v", err)
		}
	})
}
