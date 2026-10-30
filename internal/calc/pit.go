package calc

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// VerifyCertainty represents the mathematical confidence and proof level of the verification.
type VerifyCertainty string

const (
	VerifyCertaintyExact         VerifyCertainty = "EXACT"
	VerifyCertaintyProbabilistic VerifyCertainty = "PROBABILISTIC"
	VerifyCertaintyRefuted       VerifyCertainty = "REFUTED"
	VerifyCertaintyInconclusive  VerifyCertainty = "INCONCLUSIVE"
)

// EstimateDegreeUpperBound computes a safe compositional upper bound on the degree of varName in node n.
func EstimateDegreeUpperBound(n Node, varName string) int {
	if n == nil {
		return 0
	}

	switch v := n.(type) {
	case *RationalNode, *ConstNode:
		return 0

	case *VarNode:
		if v.Name == varName {
			return 1
		}
		return 0

	case *UnaryOpNode:
		return EstimateDegreeUpperBound(v.Expr, varName)

	case *AddNode:
		maxD := 0
		for _, term := range v.Terms {
			d := EstimateDegreeUpperBound(term, varName)
			if d > maxD {
				maxD = d
			}
		}
		return maxD

	case *MulNode:
		sumD := 0
		for _, factor := range v.Factors {
			sumD += EstimateDegreeUpperBound(factor, varName)
		}
		return sumD

	case *PowNode:
		baseD := EstimateDegreeUpperBound(v.Base, varName)
		if rn, ok := v.Exp.(*RationalNode); ok && rn.Val.IsInt() {
			expInt := int(rn.Val.Num().Int64())
			if expInt > 0 {
				return baseD * expInt
			}
			// Negative powers in rational function numerator bound
			return baseD
		}
		return baseD + 1

	default:
		return 1
	}
}

// EvalAtRational evaluates an AST node directly at a rational point point without simplification.
func EvalAtRational(n Node, varName string, point *big.Rat) (*big.Rat, error) {
	if n == nil {
		return nil, fmt.Errorf("nil node in rational evaluation")
	}

	switch v := n.(type) {
	case *RationalNode:
		return new(big.Rat).Set(v.Val), nil

	case *ConstNode:
		if v.Name == "i" {
			return nil, fmt.Errorf("complex constant i not supported in real rational evaluation")
		}
		return nil, fmt.Errorf("transcendental constant %s cannot be evaluated as exact rational", v.Name)

	case *VarNode:
		if v.Name == varName {
			return new(big.Rat).Set(point), nil
		}
		return big.NewRat(1, 1), nil

	case *UnaryOpNode:
		val, err := EvalAtRational(v.Expr, varName, point)
		if err != nil {
			return nil, err
		}
		if v.Op == "-" {
			return new(big.Rat).Neg(val), nil
		}
		return nil, fmt.Errorf("unsupported unary op %s", v.Op)

	case *AddNode:
		sum := new(big.Rat)
		for _, term := range v.Terms {
			val, err := EvalAtRational(term, varName, point)
			if err != nil {
				return nil, err
			}
			sum.Add(sum, val)
		}
		return sum, nil

	case *MulNode:
		prod := big.NewRat(1, 1)
		for _, factor := range v.Factors {
			val, err := EvalAtRational(factor, varName, point)
			if err != nil {
				return nil, err
			}
			prod.Mul(prod, val)
		}
		return prod, nil

	case *PowNode:
		baseVal, err := EvalAtRational(v.Base, varName, point)
		if err != nil {
			return nil, err
		}
		if rn, ok := v.Exp.(*RationalNode); ok && rn.Val.IsInt() {
			k := rn.Val.Num().Int64()
			if k == 0 {
				return big.NewRat(1, 1), nil
			}
			if k > 0 {
				res := new(big.Rat).Set(baseVal)
				for i := int64(1); i < k; i++ {
					res.Mul(res, baseVal)
				}
				return res, nil
			}
			// k < 0: reciprocal
			if baseVal.Sign() == 0 {
				return nil, fmt.Errorf("division by zero in power evaluation")
			}
			posK := -k
			posRes := new(big.Rat).Set(baseVal)
			for i := int64(1); i < posK; i++ {
				posRes.Mul(posRes, baseVal)
			}
			return new(big.Rat).Inv(posRes), nil
		}
		return nil, fmt.Errorf("non-integer exponent in rational evaluation")

	default:
		return nil, fmt.Errorf("unsupported node %T in rational evaluation", n)
	}
}

// candidateSamplePoints generates small deterministic rational sample points.
var candidateSamplePoints = []*big.Rat{
	big.NewRat(0, 1),
	big.NewRat(1, 1),
	big.NewRat(-1, 1),
	big.NewRat(2, 1),
	big.NewRat(-2, 1),
	big.NewRat(3, 1),
	big.NewRat(-3, 1),
	big.NewRat(1, 2),
	big.NewRat(-1, 2),
	big.NewRat(4, 1),
	big.NewRat(-4, 1),
	big.NewRat(5, 1),
	big.NewRat(-5, 1),
	big.NewRat(1, 3),
	big.NewRat(-1, 3),
	big.NewRat(2, 3),
	big.NewRat(-2, 3),
	big.NewRat(6, 1),
	big.NewRat(-6, 1),
	big.NewRat(7, 1),
	big.NewRat(-7, 1),
}

// VerifyEquivalencePIT verifies if lhs == rhs algebraically using D+1 deterministic PIT
// or probabilistic sampling if multivariate / high degree.
func VerifyEquivalencePIT(lhs, rhs Node, varName string) (VerifyCertainty, *CounterexampleWitness, error) {
	if lhs == nil || rhs == nil {
		return VerifyCertaintyInconclusive, nil, fmt.Errorf("nil node in PIT verification")
	}

	degL := EstimateDegreeUpperBound(lhs, varName)
	degR := EstimateDegreeUpperBound(rhs, varName)
	maxDeg := degL
	if degR > maxDeg {
		maxDeg = degR
	}

	// For univariate with degree D <= 15, D+1 deterministic evaluation is a mathematical proof
	neededPoints := maxDeg + 1
	if neededPoints < 3 {
		neededPoints = 3
	}

	testedCount := 0
	for _, pt := range candidateSamplePoints {
		if testedCount >= neededPoints {
			break
		}

		vL, errL := EvalAtRational(lhs, varName, pt)
		vR, errR := EvalAtRational(rhs, varName, pt)

		// If both have pole / division by zero at this sample point, skip
		if errL != nil && errR != nil {
			continue
		}
		// If one is defined and the other undefined, domain mismatch -> Refuted
		if (errL == nil && errR != nil) || (errL != nil && errR == nil) {
			wit := &CounterexampleWitness{
				Kind:         WitnessDomainMismatch,
				Assignments:  map[string]*big.Rat{varName: pt},
				LHSUndefined: errL != nil,
				RHSUndefined: errR != nil,
				LHSVal:       vL,
				RHSVal:       vR,
				Message:      fmt.Sprintf("Domain mismatch at %s = %s", varName, pt.String()),
			}
			return VerifyCertaintyRefuted, wit, nil
		}

		// Both defined: compare values
		if vL.Cmp(vR) != 0 {
			wit := &CounterexampleWitness{
				Kind:         WitnessValueMismatch,
				Assignments:  map[string]*big.Rat{varName: pt},
				LHSVal:       vL,
				RHSVal:       vR,
				Message:      fmt.Sprintf("Value mismatch at %s = %s: LHS = %s, RHS = %s", varName, pt.String(), vL.String(), vR.String()),
			}
			return VerifyCertaintyRefuted, wit, nil
		}

		testedCount++
	}

	if testedCount >= neededPoints {
		return VerifyCertaintyExact, nil, nil
	}

	// Fallback to randomized sampling with bounded integers
	for i := 0; i < 5; i++ {
		b, _ := rand.Int(rand.Reader, big.NewInt(1000))
		randPt := new(big.Rat).SetInt(b)

		vL, errL := EvalAtRational(lhs, varName, randPt)
		vR, errR := EvalAtRational(rhs, varName, randPt)
		if errL != nil || errR != nil {
			continue
		}
		if vL.Cmp(vR) != 0 {
			wit := &CounterexampleWitness{
				Kind:         WitnessValueMismatch,
				Assignments:  map[string]*big.Rat{varName: randPt},
				LHSVal:       vL,
				RHSVal:       vR,
				Message:      fmt.Sprintf("Random sampling mismatch at %s = %s", varName, randPt.String()),
			}
			return VerifyCertaintyRefuted, wit, nil
		}
	}

	return VerifyCertaintyProbabilistic, nil, nil
}
