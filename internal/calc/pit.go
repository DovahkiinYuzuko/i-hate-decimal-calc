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

// IsPurePolynomial checks if the expression tree is a pure polynomial in varName
// (no division, no negative exponents, no transcendental functions, and no other free variables).
func IsPurePolynomial(n Node, varName string) bool {
	if n == nil {
		return false
	}

	switch v := n.(type) {
	case *RationalNode:
		return true

	case *ConstNode:
		return false

	case *VarNode:
		return v.Name == varName

	case *UnaryOpNode:
		if v.Op == "-" {
			return IsPurePolynomial(v.Expr, varName)
		}
		return false

	case *AddNode:
		for _, term := range v.Terms {
			if !IsPurePolynomial(term, varName) {
				return false
			}
		}
		return true

	case *MulNode:
		for _, factor := range v.Factors {
			if !IsPurePolynomial(factor, varName) {
				return false
			}
		}
		return true

	case *PowNode:
		if !IsPurePolynomial(v.Base, varName) {
			return false
		}
		rn, ok := v.Exp.(*RationalNode)
		if !ok || !rn.Val.IsInt() {
			return false
		}
		// Negative exponents are rational functions, not polynomials
		if rn.Val.Sign() < 0 {
			return false
		}
		return true

	default:
		return false
	}
}

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
			num := rn.Val.Num()
			if num.Sign() > 0 && num.IsInt64() {
				expInt := int(num.Int64())
				// Guard against arithmetic overflow in degree calculation
				if expInt > 0 && baseD <= (1<<30)/expInt {
					return baseD * expInt
				}
			}
			return baseD
		}
		return baseD + 1

	default:
		return 1
	}
}

// ratPowBigInt computes base^exp using binary exponentiation (O(log exp)) without downcasting to int.
func ratPowBigInt(base *big.Rat, exp *big.Int) (*big.Rat, error) {
	if exp.Sign() == 0 {
		return big.NewRat(1, 1), nil
	}

	isNeg := exp.Sign() < 0
	absExp := new(big.Int).Abs(exp)

	// Binary exponentiation
	result := big.NewRat(1, 1)
	cur := new(big.Rat).Set(base)

	for i := 0; i < absExp.BitLen(); i++ {
		if absExp.Bit(i) == 1 {
			result.Mul(result, cur)
			// Safety budget: prevent intermediate bit swell DoS
			if result.Num().BitLen()+result.Denom().BitLen() > 100000 {
				return nil, fmt.Errorf("bit size budget exceeded in rational power")
			}
		}
		if i+1 < absExp.BitLen() {
			cur.Mul(cur, cur)
			if cur.Num().BitLen()+cur.Denom().BitLen() > 100000 {
				return nil, fmt.Errorf("bit size budget exceeded in rational power")
			}
		}
	}

	if isNeg {
		if result.Sign() == 0 {
			return nil, fmt.Errorf("division by zero in power evaluation")
		}
		result.Inv(result)
	}

	return result, nil
}

// EvalAtRational evaluates an AST node directly at a rational point without simplification.
func EvalAtRational(n Node, varName string, point *big.Rat) (*big.Rat, error) {
	if n == nil {
		return nil, fmt.Errorf("nil node in rational evaluation")
	}

	switch v := n.(type) {
	case *RationalNode:
		return new(big.Rat).Set(v.Val), nil

	case *ConstNode:
		if v.Name == "i" {
			return nil, fmt.Errorf("unsupported: complex constant i")
		}
		return nil, fmt.Errorf("unsupported: transcendental constant %s", v.Name)

	case *VarNode:
		if v.Name == varName {
			return new(big.Rat).Set(point), nil
		}
		return nil, fmt.Errorf("unsupported: multiple variables (encountered %s)", v.Name)

	case *UnaryOpNode:
		val, err := EvalAtRational(v.Expr, varName, point)
		if err != nil {
			return nil, err
		}
		if v.Op == "-" {
			return new(big.Rat).Neg(val), nil
		}
		return nil, fmt.Errorf("unsupported: unary op %s", v.Op)

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
			return ratPowBigInt(baseVal, rn.Val.Num())
		}
		return nil, fmt.Errorf("unsupported: non-integer exponent in rational evaluation")

	default:
		return nil, fmt.Errorf("unsupported: node %T in rational evaluation", n)
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

// isUnsupportedError returns true if the error is due to an unsupported node/function rather than a domain singularity.
func isUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return len(s) >= 12 && s[:12] == "unsupported:"
}

// VerifyEquivalencePIT verifies if lhs == rhs algebraically using D+1 deterministic PIT
// for univariate polynomials, or returns Inconclusive for non-polynomials / multivariate expressions.
func VerifyEquivalencePIT(lhs, rhs Node, varName string) (VerifyCertainty, *CounterexampleWitness, error) {
	if lhs == nil || rhs == nil {
		return VerifyCertaintyInconclusive, nil, fmt.Errorf("nil node in PIT verification")
	}

	// 1. Strict univariate polynomial guard
	// PIT theorem (D+1 point equality => algebraic identity) only holds for univariate polynomials.
	if !IsPurePolynomial(lhs, varName) || !IsPurePolynomial(rhs, varName) {
		return VerifyCertaintyInconclusive, nil, nil
	}

	// 2. Variable count guard: must be strictly univariate
	varsL := ExtractFreeVariables(lhs)
	varsR := ExtractFreeVariables(rhs)
	allVars := make(map[string]bool)
	for _, v := range varsL {
		allVars[v] = true
	}
	for _, v := range varsR {
		allVars[v] = true
	}
	if len(allVars) > 1 || (len(allVars) == 1 && !allVars[varName]) {
		return VerifyCertaintyInconclusive, nil, nil
	}

	degL := EstimateDegreeUpperBound(lhs, varName)
	degR := EstimateDegreeUpperBound(rhs, varName)
	maxDeg := degL
	if degR > maxDeg {
		maxDeg = degR
	}

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

		// If either failed due to an unsupported node type, fail safe to Inconclusive
		if isUnsupportedError(errL) || isUnsupportedError(errR) {
			return VerifyCertaintyInconclusive, nil, nil
		}

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
	randomSuccessCount := 0
	for i := 0; i < 5; i++ {
		b, err := rand.Int(rand.Reader, big.NewInt(1000))
		if err != nil || b == nil {
			continue
		}
		randPt := new(big.Rat).SetInt(b)

		vL, errL := EvalAtRational(lhs, varName, randPt)
		vR, errR := EvalAtRational(rhs, varName, randPt)
		if isUnsupportedError(errL) || isUnsupportedError(errR) {
			return VerifyCertaintyInconclusive, nil, nil
		}
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
		randomSuccessCount++
	}

	if randomSuccessCount > 0 {
		return VerifyCertaintyProbabilistic, nil, nil
	}

	return VerifyCertaintyInconclusive, nil, nil
}
