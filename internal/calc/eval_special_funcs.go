package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Elementary Special Functions & Combinatorics Pack
// (Euler 1729/1734, Legendre 1809, Akiyama-Tanigawa 2000)
// -------------------------------------------------------------------------

// EvalGamma evaluates the Gamma function Γ(z) with exact algebraic values:
// - Positive integer n: Γ(n) = (n-1)!
// - Non-positive integer n <= 0: undefined (pole)
// - Half-integer k/2: exact rational * sqrt(pi) via reflection/duplication
// - General symbols or unsupported arguments: retained as symbolic gamma(z)
func EvalGamma(arg Node, env *Env) (Node, error) {
	evalArg, err := EvalWithEnv(arg, env)
	if err != nil {
		evalArg = arg
	}

	r, ok := evalArg.(*RationalNode)
	if !ok {
		return &FuncNode{Name: "gamma", Args: []Node{evalArg}}, nil
	}

	// 1. Integer arguments
	if r.Val.IsInt() {
		n := r.Val.Num().Int64()
		if n <= 0 {
			return nil, fmt.Errorf("%s", i18n.T("errors.gamma_pole", evalArg.String()))
		}
		if n == 1 {
			return mustRational(1, 1), nil
		}
		fact := factorialInt(int(n - 1))
		return &RationalNode{Val: new(big.Rat).SetInt(fact)}, nil
	}

	// 2. Half-integer arguments: denominator == 2
	if r.Val.Denom().Int64() == 2 {
		k := r.Val.Num().Int64() // odd integer
		return evalHalfIntegerGamma(k)
	}

	// Other rational fractions: retain symbolic
	return &FuncNode{Name: "gamma", Args: []Node{evalArg}}, nil
}

// gammaRationalCoeff returns the rational coefficient c and whether sqrt(pi) is present
// such that Γ(r) = c * (sqrt(pi) if hasSqrtPi else 1).
func gammaRationalCoeff(r *big.Rat) (*big.Rat, bool, error) {
	if r.IsInt() {
		n := r.Num().Int64()
		if n <= 0 {
			return nil, false, fmt.Errorf("%s", i18n.T("errors.gamma_pole", r.String()))
		}
		if n == 1 {
			return big.NewRat(1, 1), false, nil
		}
		fact := factorialInt(int(n - 1))
		return new(big.Rat).SetInt(fact), false, nil
	}

	if r.Denom().Int64() == 2 {
		k := r.Num().Int64()
		if k > 0 {
			n := (k - 1) / 2
			if n == 0 {
				return big.NewRat(1, 1), true, nil
			}
			doubleFact := doubleFactorialInt(2*n - 1)
			denom := new(big.Int).Exp(big.NewInt(2), big.NewInt(n), nil)
			return new(big.Rat).SetFrac(doubleFact, denom), true, nil
		}

		// k < 0: negative half-integers
		m := (1 - k) / 2
		denomProd := big.NewRat(1, 1)
		for j := int64(0); j < m; j++ {
			termNum := k + 2*j
			termRat := big.NewRat(termNum, 2)
			denomProd.Mul(denomProd, termRat)
		}
		return new(big.Rat).Inv(denomProd), true, nil
	}

	return nil, false, fmt.Errorf("%s", i18n.T("errors.gamma_invalid_arg", r.String()))
}

// evalHalfIntegerGamma computes Γ(k/2) for odd integer k as rational * sqrt(pi).
func evalHalfIntegerGamma(k int64) (Node, error) {
	r := big.NewRat(k, 2)
	coeff, hasSqrtPi, err := gammaRationalCoeff(r)
	if err != nil {
		return nil, err
	}
	if !hasSqrtPi {
		return &RationalNode{Val: coeff}, nil
	}

	sqrtPi, errS := simplifySqrt(&VarNode{Name: "pi"})
	if errS != nil {
		sqrtPi = &SqrtNode{Radicand: &VarNode{Name: "pi"}}
	}

	if coeff.Cmp(big.NewRat(1, 1)) == 0 {
		return sqrtPi, nil
	}
	return simplifyMul([]Node{&RationalNode{Val: coeff}, sqrtPi})
}

// doubleFactorialInt returns (2n - 1)!! = 1 * 3 * 5 * ... * (2n - 1).
func doubleFactorialInt(n int64) *big.Int {
	res := big.NewInt(1)
	for i := int64(3); i <= n; i += 2 {
		res.Mul(res, big.NewInt(i))
	}
	return res
}

// EvalBeta evaluates the Beta function B(p, q) = Γ(p)Γ(q) / Γ(p+q).
func EvalBeta(pNode, qNode Node, env *Env) (Node, error) {
	pEval, errP := EvalWithEnv(pNode, env)
	if errP != nil {
		pEval = pNode
	}
	qEval, errQ := EvalWithEnv(qNode, env)
	if errQ != nil {
		qEval = qNode
	}

	// Check if both p and q are rational numbers with denominator 1 or 2
	pRat, okP := pEval.(*RationalNode)
	qRat, okQ := qEval.(*RationalNode)

	if okP && okQ && (pRat.Val.IsInt() || pRat.Val.Denom().Int64() == 2) &&
		(qRat.Val.IsInt() || qRat.Val.Denom().Int64() == 2) {
		coeffP, hasPiP, err := gammaRationalCoeff(pRat.Val)
		if err != nil {
			return nil, err
		}
		coeffQ, hasPiQ, err := gammaRationalCoeff(qRat.Val)
		if err != nil {
			return nil, err
		}

		sumRat := new(big.Rat).Add(pRat.Val, qRat.Val)
		coeffSum, hasPiSum, err := gammaRationalCoeff(sumRat)
		if err != nil {
			return nil, err
		}

		// Beta coefficient = (coeffP * coeffQ) / coeffSum
		prodNum := new(big.Rat).Mul(coeffP, coeffQ)
		betaCoeff := new(big.Rat).Quo(prodNum, coeffSum)

		// Determine power of sqrt(pi):
		sqrtPiCount := 0
		if hasPiP {
			sqrtPiCount++
		}
		if hasPiQ {
			sqrtPiCount++
		}
		if hasPiSum {
			sqrtPiCount--
		}

		if sqrtPiCount == 0 {
			return &RationalNode{Val: betaCoeff}, nil
		}
		if sqrtPiCount == 2 {
			if betaCoeff.Cmp(big.NewRat(1, 1)) == 0 {
				return &VarNode{Name: "pi"}, nil
			}
			return simplifyMul([]Node{&RationalNode{Val: betaCoeff}, &VarNode{Name: "pi"}})
		}
	}

	return &FuncNode{Name: "beta", Args: []Node{pEval, qEval}}, nil
}

// EvalBernoulli computes the n-th Bernoulli number B_n as an exact rational number.
func EvalBernoulli(nNode Node) (Node, error) {
	r, ok := nNode.(*RationalNode)
	if !ok || !r.Val.IsInt() || r.Val.Sign() < 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.bernoulli_non_negative", nNode.String()))
	}

	n := int(r.Val.Num().Int64())
	resRat := bernoulliRat(n)
	return &RationalNode{Val: resRat}, nil
}

// bernoulliRat computes B_n using the Akiyama-Tanigawa algorithm (Kaneko 2000).
func bernoulliRat(n int) *big.Rat {
	if n == 0 {
		return big.NewRat(1, 1)
	}
	if n == 1 {
		// Modern standard: B_1 = -1/2
		return big.NewRat(-1, 2)
	}
	if n%2 != 0 {
		// All odd Bernoulli numbers for n >= 3 are 0
		return big.NewRat(0, 1)
	}

	// Akiyama-Tanigawa algorithm for even n:
	// Row 0: a[0][m] = 1 / (m + 1) for m = 0, ..., n
	a := make([]*big.Rat, n+1)
	for m := 0; m <= n; m++ {
		a[m] = big.NewRat(1, int64(m+1))
	}

	for j := 1; j <= n; j++ {
		for m := 0; m <= n-j; m++ {
			// a[m] = (m + 1) * (a[m] - a[m+1])
			diff := new(big.Rat).Sub(a[m], a[m+1])
			multiplier := big.NewRat(int64(m+1), 1)
			a[m] = new(big.Rat).Mul(multiplier, diff)
		}
	}

	return a[0]
}

// EvalZeta evaluates the Riemann zeta function ζ(s).
// For s = 2k (positive even integer), evaluates exactly via Euler formula:
// ζ(2k) = (-1)^(k-1) * 2^(2k-1) * B_{2k} / (2k)! * pi^(2k)
func EvalZeta(sNode Node, env *Env) (Node, error) {
	evalS, err := EvalWithEnv(sNode, env)
	if err != nil {
		evalS = sNode
	}

	r, ok := evalS.(*RationalNode)
	if ok && r.Val.IsInt() {
		s := r.Val.Num().Int64()
		if s == 1 {
			// Pole at s = 1
			return nil, fmt.Errorf("%s", i18n.T("errors.zeta_domain_error"))
		}
		if s == 0 {
			// ζ(0) = -1/2
			return mustRational(-1, 2), nil
		}
		if s > 0 && s%2 == 0 {
			// Positive even integer: s = 2k
			// 1. Bernoulli number B_{2k}
			b2k := bernoulliRat(int(s))

			// 2. Factorial (2k)!
			fact := factorialInt(int(s))

			// 3. Power 2^(2k - 1)
			twoPow := new(big.Int).Exp(big.NewInt(2), big.NewInt(s-1), nil)

			// 4. Coefficient = |2^(2k-1) * B_{2k} / (2k)!|
			num := new(big.Int).Mul(twoPow, b2k.Num())
			den := new(big.Int).Mul(fact, b2k.Denom())
			coeff := new(big.Rat).SetFrac(num, den)
			coeff.Abs(coeff) // Euler's formula always yields positive coefficients for ζ(2k)

			piPow := &PowNode{
				Base: &VarNode{Name: "pi"},
				Exp:  mustRational(s, 1),
			}

			if coeff.Cmp(big.NewRat(1, 1)) == 0 {
				return piPow, nil
			}

			return simplifyMul([]Node{&RationalNode{Val: coeff}, piPow})
		}
	}

	return &FuncNode{Name: "zeta", Args: []Node{evalS}}, nil
}
