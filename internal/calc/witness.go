package calc

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// WitnessKind identifies the category of counterexample found.
type WitnessKind int

const (
	// WitnessValueMismatch indicates both sides evaluated to unequal defined values.
	WitnessValueMismatch WitnessKind = iota
	// WitnessDomainMismatch indicates one side is defined while the other is undefined.
	WitnessDomainMismatch
)

// CounterexampleWitness represents a concrete numerical assignment refuting an equality.
type CounterexampleWitness struct {
	Kind         WitnessKind
	Assignments  map[string]*big.Rat
	LHSVal       *big.Rat // nil if undefined or non-rational
	RHSVal       *big.Rat // nil if undefined or non-rational
	LHSUndefined bool
	RHSUndefined bool
	Message      string
}

// String returns human-readable description of the witness.
func (w *CounterexampleWitness) String() string {
	if w == nil {
		return ""
	}
	return w.Message
}

func copyAssignments(src map[string]*big.Rat) map[string]*big.Rat {
	dst := make(map[string]*big.Rat, len(src))
	for k, v := range src {
		dst[k] = new(big.Rat).Set(v)
	}
	return dst
}

// FindCounterexample searches for a concrete rational counterexample to lhs == rhs.
// It searches a small deterministic grid, correctly distinguishing value mismatches from domain mismatches.
func FindCounterexample(lhs, rhs Node, env *Env) (*CounterexampleWitness, error) {
	if lhs == nil || rhs == nil {
		return nil, nil
	}

	// 1. Collect free variables in canonical sorted order
	varsLHS := ExtractFreeVariables(lhs)
	varsRHS := ExtractFreeVariables(rhs)
	varSet := make(map[string]bool)
	for _, v := range varsLHS {
		varSet[v] = true
	}
	for _, v := range varsRHS {
		varSet[v] = true
	}

	if len(varSet) == 0 {
		// Both are constants; no variable assignments to search
		return nil, nil
	}

	allVars := make([]string, 0, len(varSet))
	for v := range varSet {
		allVars = append(allVars, v)
	}
	sort.Strings(allVars)

	// 2. Deterministic candidate pool of exact rationals
	gridInts := []int64{0, 1, -1, 2, -2, 3, -3}
	gridFracs := [][2]int64{{1, 2}, {-1, 2}, {1, 3}, {-1, 3}}

	candidates := make([]*big.Rat, 0, len(gridInts)+len(gridFracs))
	for _, i := range gridInts {
		candidates = append(candidates, big.NewRat(i, 1))
	}
	for _, f := range gridFracs {
		candidates = append(candidates, big.NewRat(f[0], f[1]))
	}

	// 3. Generate candidate vectors up to budget using backtracking (zero allocation per step)
	maxPoints := 250
	pointCount := 0
	current := make(map[string]*big.Rat, len(allVars))

	var searchGrid func(varIdx int) *CounterexampleWitness
	searchGrid = func(varIdx int) *CounterexampleWitness {
		if pointCount >= maxPoints {
			return nil
		}

		if varIdx == len(allVars) {
			pointCount++
			// Evaluate both sides with this assignment
			evalEnv := NewEnv()
			if env != nil {
				evalEnv = env.Clone()
			}
			for v, val := range current {
				evalEnv.Set(v, NewRationalFromBigRat(val))
			}

			evalL, errL := EvalWithEnv(lhs, evalEnv)
			evalR, errR := EvalWithEnv(rhs, evalEnv)

			// Both undefined -> skip (not a counterexample)
			if errL != nil && errR != nil {
				return nil
			}

			formatAssigns := func() string {
				parts := make([]string, len(allVars))
				for i, v := range allVars {
					parts[i] = fmt.Sprintf("%s = %s", v, current[v].RatString())
				}
				return strings.Join(parts, ", ")
			}

			// Domain mismatch: one defined, one undefined
			if errL != nil && errR == nil {
				rStr := evalR.String()
				return &CounterexampleWitness{
					Kind:         WitnessDomainMismatch,
					Assignments:  copyAssignments(current),
					LHSVal:       nil,
					RHSVal:       tryExtractRat(evalR),
					LHSUndefined: true,
					Message:      i18n.T("verify.witness_domain_mismatch_lhs", formatAssigns(), errL, rStr),
				}
			}
			if errL == nil && errR != nil {
				lStr := evalL.String()
				return &CounterexampleWitness{
					Kind:         WitnessDomainMismatch,
					Assignments:  copyAssignments(current),
					LHSVal:       tryExtractRat(evalL),
					RHSVal:       nil,
					RHSUndefined: true,
					Message:      i18n.T("verify.witness_domain_mismatch_rhs", formatAssigns(), lStr, errR),
				}
			}

			// 1. Check structural AST equality first (handles Complex(0,0), matrices, symbols)
			if evalL.Equal(evalR) {
				return nil
			}

			// 2. Both evaluate to rationals: compare values
			ratL := tryExtractRat(evalL)
			ratR := tryExtractRat(evalR)
			if ratL != nil && ratR != nil {
				if ratL.Cmp(ratR) != 0 {
					return &CounterexampleWitness{
						Kind:        WitnessValueMismatch,
						Assignments: copyAssignments(current),
						LHSVal:      ratL,
						RHSVal:      ratR,
						Message:     i18n.T("verify.witness_counterexample", formatAssigns(), ratL.RatString(), ratR.RatString()),
					}
				}
				return nil
			}

			// 3. Fallback: check algebraic zero equivalence
			negR, _ := simplifyUnaryOp("-", evalR)
			diffNode := NewAdd([]Node{evalL, negR})
			if checkIsZeroAlgebraically(diffNode, evalEnv) {
				return nil
			}

			return &CounterexampleWitness{
				Kind:        WitnessValueMismatch,
				Assignments: copyAssignments(current),
				LHSVal:      ratL,
				RHSVal:      ratR,
				Message:     i18n.T("verify.witness_counterexample", formatAssigns(), evalL.String(), evalR.String()),
			}
		}

		vName := allVars[varIdx]
		for _, cand := range candidates {
			current[vName] = cand
			if w := searchGrid(varIdx + 1); w != nil {
				return w
			}
			delete(current, vName)
			if pointCount >= maxPoints {
				break
			}
		}
		return nil
	}

	witness := searchGrid(0)
	return witness, nil
}

func tryExtractRat(n Node) *big.Rat {
	if n == nil {
		return nil
	}
	if r, ok := n.(*RationalNode); ok && r.Val != nil {
		return new(big.Rat).Set(r.Val)
	}
	return nil
}
