package calc

import (
	"fmt"
	"math/big"
)

// extractRat extracts *big.Rat from a Node if it is a RationalNode.
func extractRat(n Node, context string) (*big.Rat, error) {
	rat, ok := n.(*RationalNode)
	if !ok {
		return nil, fmt.Errorf("%s: expected rational number, got %s", context, n.String())
	}
	return new(big.Rat).Set(rat.Val), nil
}

// extractNonNegativeInt extracts a non-negative *big.Int from a RationalNode with denominator 1.
func extractNonNegativeInt(n Node, context string) (*big.Int, error) {
	rat, ok := n.(*RationalNode)
	if !ok || !rat.Val.IsInt() {
		return nil, fmt.Errorf("%s: %s", context, MsgErrProbNonNegativeInt)
	}
	val := rat.Val.Num()
	if val.Sign() < 0 {
		return nil, fmt.Errorf("%s: %s", context, MsgErrProbNonNegativeInt)
	}
	return new(big.Int).Set(val), nil
}

// extractPositiveInt extracts a positive *big.Int (> 0) from a RationalNode with denominator 1.
func extractPositiveInt(n Node, context string) (*big.Int, error) {
	rat, ok := n.(*RationalNode)
	if !ok || !rat.Val.IsInt() {
		return nil, fmt.Errorf("%s: %s", context, MsgErrProbPositiveInt)
	}
	val := rat.Val.Num()
	if val.Sign() <= 0 {
		return nil, fmt.Errorf("%s: %s", context, MsgErrProbPositiveInt)
	}
	return new(big.Int).Set(val), nil
}

// extractProbRat extracts a probability in [0, 1] from a RationalNode.
func extractProbRat(n Node, context string) (*big.Rat, error) {
	rat, err := extractRat(n, context)
	if err != nil {
		return nil, err
	}
	zero := big.NewRat(0, 1)
	one := big.NewRat(1, 1)
	if rat.Cmp(zero) < 0 || rat.Cmp(one) > 0 {
		return nil, fmt.Errorf("%s: %s (got %s)", context, MsgErrProbOutOfRange, rat.RatString())
	}
	return rat, nil
}

// combBig calculates binomial coefficient C(n, k) for big.Int.
func combBig(n, k *big.Int) *big.Int {
	zero := big.NewInt(0)
	if k.Cmp(zero) < 0 || k.Cmp(n) > 0 {
		return big.NewInt(0)
	}
	if k.Cmp(zero) == 0 || k.Cmp(n) == 0 {
		return big.NewInt(1)
	}

	// C(n, k) = C(n, n-k)
	nMinusK := new(big.Int).Sub(n, k)
	kEff := new(big.Int).Set(k)
	if kEff.Cmp(nMinusK) > 0 {
		kEff.Set(nMinusK)
	}

	result := big.NewInt(1)
	i := big.NewInt(1)
	one := big.NewInt(1)

	for i.Cmp(kEff) <= 0 {
		// result = result * (n - i + 1) / i
		numTerm := new(big.Int).Sub(n, i)
		numTerm.Add(numTerm, one)
		result.Mul(result, numTerm)
		result.Quo(result, i)
		i.Add(i, one)
	}

	return result
}

// ratPowInt raises big.Rat to a non-negative big.Int power.
func ratPowInt(base *big.Rat, exp *big.Int) *big.Rat {
	zero := big.NewInt(0)
	if exp.Cmp(zero) == 0 {
		return big.NewRat(1, 1)
	}
	if base.Sign() == 0 {
		return big.NewRat(0, 1)
	}

	num := new(big.Int).Exp(base.Num(), exp, nil)
	denom := new(big.Int).Exp(base.Denom(), exp, nil)
	res := new(big.Rat).SetFrac(num, denom)
	return res
}

// getDistName extracts the distribution name from a VarNode, ConstNode, or string representation.
func getDistName(n Node) string {
	switch v := n.(type) {
	case *VarNode:
		return v.Name
	case *ConstNode:
		return v.Name
	default:
		return n.String()
	}
}

// EvalBinomPMF calculates binomial distribution PMF P(X = k) = C(n, k) * p^k * (1-p)^(n-k).
func EvalBinomPMF(args []Node) (Node, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("binom requires 3 arguments (n, k, p), got %d", len(args))
	}

	n, err := extractNonNegativeInt(args[0], "binom: n")
	if err != nil {
		return nil, err
	}
	k, err := extractNonNegativeInt(args[1], "binom: k")
	if err != nil {
		return nil, err
	}
	p, err := extractProbRat(args[2], "binom: p")
	if err != nil {
		return nil, err
	}

	if k.Cmp(n) > 0 {
		return mustRational(0, 1), nil
	}

	c := combBig(n, k)
	cRat := new(big.Rat).SetInt(c)

	pk := ratPowInt(p, k)

	one := big.NewRat(1, 1)
	oneMinusP := new(big.Rat).Sub(one, p)
	nMinusK := new(big.Int).Sub(n, k)
	pRest := ratPowInt(oneMinusP, nMinusK)

	ans := new(big.Rat).Mul(cRat, pk)
	ans.Mul(ans, pRest)

	return NewRationalFromBigRat(ans), nil
}

// EvalHyperPMF calculates hypergeometric distribution PMF P(X = k) = C(K, k) * C(N-K, n-k) / C(N, n).
func EvalHyperPMF(args []Node) (Node, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("hyper requires 4 arguments (N, K, n, k), got %d", len(args))
	}

	N, err := extractPositiveInt(args[0], "hyper: N")
	if err != nil {
		return nil, err
	}
	K, err := extractNonNegativeInt(args[1], "hyper: K")
	if err != nil {
		return nil, err
	}
	n, err := extractNonNegativeInt(args[2], "hyper: n")
	if err != nil {
		return nil, err
	}
	k, err := extractNonNegativeInt(args[3], "hyper: k")
	if err != nil {
		return nil, err
	}

	if K.Cmp(N) > 0 || n.Cmp(N) > 0 {
		return nil, fmt.Errorf("%s: K and n cannot exceed N (got N=%s, K=%s, n=%s)", MsgErrHyperParams, N.String(), K.String(), n.String())
	}

	// Check support condition: max(0, n - (N - K)) <= k <= min(n, K)
	nMinusKAll := new(big.Int).Sub(N, K)
	nMinusKDraw := new(big.Int).Sub(n, k)

	if k.Cmp(K) > 0 || k.Cmp(n) > 0 || nMinusKDraw.Sign() < 0 || nMinusKDraw.Cmp(nMinusKAll) > 0 {
		return mustRational(0, 1), nil
	}

	combKk := combBig(K, k)
	combRest := combBig(nMinusKAll, nMinusKDraw)
	num := new(big.Int).Mul(combKk, combRest)

	denom := combBig(N, n)
	if denom.Sign() == 0 {
		return nil, fmt.Errorf("%s: division by zero in hypergeometric denominator", MsgErrHyperParams)
	}

	ans := new(big.Rat).SetFrac(num, denom)
	return NewRationalFromBigRat(ans), nil
}

// EvalGeomPMF calculates geometric distribution PMF P(X = k) = (1-p)^(k-1) * p.
func EvalGeomPMF(args []Node) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("geom requires 2 arguments (p, k), got %d", len(args))
	}

	p, err := extractProbRat(args[0], "geom: p")
	if err != nil {
		return nil, err
	}
	if p.Sign() == 0 {
		return nil, fmt.Errorf("geom: p cannot be zero")
	}

	k, err := extractPositiveInt(args[1], "geom: k")
	if err != nil {
		return nil, err
	}

	one := big.NewRat(1, 1)
	oneMinusP := new(big.Rat).Sub(one, p)
	kMinusOne := new(big.Int).Sub(k, big.NewInt(1))

	qPow := ratPowInt(oneMinusP, kMinusOne)
	ans := new(big.Rat).Mul(qPow, p)

	return NewRationalFromBigRat(ans), nil
}

// EvalBayes calculates posterior probability P(A|B) = P(B|A) * P(A) / P(B).
func EvalBayes(args []Node) (Node, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("bayes requires 3 arguments (prior, likelihood, marginal), got %d", len(args))
	}

	prior, err := extractProbRat(args[0], "bayes: prior P(A)")
	if err != nil {
		return nil, err
	}
	likelihood, err := extractProbRat(args[1], "bayes: likelihood P(B|A)")
	if err != nil {
		return nil, err
	}
	marginal, err := extractProbRat(args[2], "bayes: marginal P(B)")
	if err != nil {
		return nil, err
	}

	if marginal.Sign() == 0 {
		return nil, fmt.Errorf("%s", MsgErrBayesZeroEvidence)
	}

	num := new(big.Rat).Mul(prior, likelihood)
	ans := new(big.Rat).Quo(num, marginal)

	one := big.NewRat(1, 1)
	if ans.Cmp(one) > 0 {
		return nil, fmt.Errorf("%s: posterior exceeds 1 (got %s)", MsgErrProbOutOfRange, ans.RatString())
	}

	return NewRationalFromBigRat(ans), nil
}

type probPair struct {
	x Node
	p *big.Rat
}

func extractDiscretePairs(n Node) ([]probPair, error) {
	var pairs []probPair

	switch v := n.(type) {
	case *ListNode:
		for _, elem := range v.Elements {
			pairList, ok := elem.(*ListNode)
			if !ok || len(pairList.Elements) != 2 {
				return nil, fmt.Errorf("each element of discrete distribution must be a [value, prob] pair, got %s", elem.String())
			}
			pRat, err := extractProbRat(pairList.Elements[1], "discrete list: prob")
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, probPair{x: pairList.Elements[0], p: pRat})
		}

	case *MatrixNode:
		if v.Cols != 2 {
			return nil, fmt.Errorf("discrete distribution matrix must have exactly 2 columns [value, prob], got %d cols", v.Cols)
		}
		for r := 0; r < v.Rows; r++ {
			pRat, err := extractProbRat(v.Data[r][1], "discrete matrix: prob")
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, probPair{x: v.Data[r][0], p: pRat})
		}

	default:
		return nil, nil // not a list or matrix
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("discrete distribution cannot be empty")
	}

	sumProb := big.NewRat(0, 1)
	for _, pair := range pairs {
		sumProb.Add(sumProb, pair.p)
	}
	one := big.NewRat(1, 1)
	if sumProb.Cmp(one) != 0 {
		return nil, fmt.Errorf("%s (sum = %s)", MsgErrProbSumNotOne, sumProb.RatString())
	}

	return pairs, nil
}

// EvalExpect calculates expected value E[X].
func EvalExpect(args []Node) (Node, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("expect requires at least 1 argument")
	}

	// Case 1: Discrete list or matrix distribution [[x1, p1], [x2, p2], ...]
	pairs, err := extractDiscretePairs(args[0])
	if err != nil {
		return nil, err
	}
	if pairs != nil {
		return evalPairsExpect(pairs)
	}

	// Case 2: Named distribution
	dist := getDistName(args[0])
	switch dist {
	case "binom":
		if len(args) != 3 {
			return nil, fmt.Errorf("expect(binom) requires 3 arguments (binom, n, p), got %d", len(args))
		}
		n, err := extractNonNegativeInt(args[1], "expect: n")
		if err != nil {
			return nil, err
		}
		p, err := extractProbRat(args[2], "expect: p")
		if err != nil {
			return nil, err
		}
		nRat := new(big.Rat).SetInt(n)
		ans := new(big.Rat).Mul(nRat, p)
		return NewRationalFromBigRat(ans), nil

	case "geom":
		if len(args) != 2 {
			return nil, fmt.Errorf("expect(geom) requires 2 arguments (geom, p), got %d", len(args))
		}
		p, err := extractProbRat(args[1], "expect: p")
		if err != nil {
			return nil, err
		}
		if p.Sign() == 0 {
			return nil, fmt.Errorf("expect: p cannot be zero")
		}
		one := big.NewRat(1, 1)
		ans := new(big.Rat).Quo(one, p)
		return NewRationalFromBigRat(ans), nil

	case "hyper":
		if len(args) != 4 {
			return nil, fmt.Errorf("expect(hyper) requires 4 arguments (hyper, N, K, n), got %d", len(args))
		}
		N, err := extractPositiveInt(args[1], "expect: N")
		if err != nil {
			return nil, err
		}
		K, err := extractNonNegativeInt(args[2], "expect: K")
		if err != nil {
			return nil, err
		}
		n, err := extractNonNegativeInt(args[3], "expect: n")
		if err != nil {
			return nil, err
		}
		if K.Cmp(N) > 0 || n.Cmp(N) > 0 {
			return nil, fmt.Errorf("%s", MsgErrHyperParams)
		}
		// E = n * K / N
		num := new(big.Int).Mul(n, K)
		ans := new(big.Rat).SetFrac(num, N)
		return NewRationalFromBigRat(ans), nil

	default:
		return nil, fmt.Errorf("%s: %s", MsgErrInvalidDistribution, args[0].String())
	}
}

// EvalVariance calculates variance V[X] = E[X^2] - (E[X])^2.
func EvalVariance(args []Node) (Node, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("variance requires at least 1 argument")
	}

	// Case 1: Discrete list or matrix distribution [[x1, p1], [x2, p2], ...]
	pairs, err := extractDiscretePairs(args[0])
	if err != nil {
		return nil, err
	}
	if pairs != nil {
		return evalPairsVariance(pairs)
	}

	// Case 2: Named distribution
	dist := getDistName(args[0])
	switch dist {
	case "binom":
		if len(args) != 3 {
			return nil, fmt.Errorf("variance(binom) requires 3 arguments (binom, n, p), got %d", len(args))
		}
		n, err := extractNonNegativeInt(args[1], "variance: n")
		if err != nil {
			return nil, err
		}
		p, err := extractProbRat(args[2], "variance: p")
		if err != nil {
			return nil, err
		}
		// V = n * p * (1 - p)
		one := big.NewRat(1, 1)
		oneMinusP := new(big.Rat).Sub(one, p)
		nRat := new(big.Rat).SetInt(n)
		ans := new(big.Rat).Mul(nRat, p)
		ans.Mul(ans, oneMinusP)
		return NewRationalFromBigRat(ans), nil

	case "geom":
		if len(args) != 2 {
			return nil, fmt.Errorf("variance(geom) requires 2 arguments (geom, p), got %d", len(args))
		}
		p, err := extractProbRat(args[1], "variance: p")
		if err != nil {
			return nil, err
		}
		if p.Sign() == 0 {
			return nil, fmt.Errorf("variance: p cannot be zero")
		}
		// V = (1 - p) / p^2
		one := big.NewRat(1, 1)
		oneMinusP := new(big.Rat).Sub(one, p)
		pSquared := new(big.Rat).Mul(p, p)
		ans := new(big.Rat).Quo(oneMinusP, pSquared)
		return NewRationalFromBigRat(ans), nil

	case "hyper":
		if len(args) != 4 {
			return nil, fmt.Errorf("variance(hyper) requires 4 arguments (hyper, N, K, n), got %d", len(args))
		}
		N, err := extractPositiveInt(args[1], "variance: N")
		if err != nil {
			return nil, err
		}
		K, err := extractNonNegativeInt(args[2], "variance: K")
		if err != nil {
			return nil, err
		}
		n, err := extractNonNegativeInt(args[3], "variance: n")
		if err != nil {
			return nil, err
		}
		if K.Cmp(N) > 0 || n.Cmp(N) > 0 {
			return nil, fmt.Errorf("%s", MsgErrHyperParams)
		}
		// If N == 1, variance is 0
		oneInt := big.NewInt(1)
		if N.Cmp(oneInt) == 0 {
			return mustRational(0, 1), nil
		}
		// V = n * (K/N) * ((N-K)/N) * ((N-n)/(N-1))
		nMinusK := new(big.Int).Sub(N, K)
		nMinusDraw := new(big.Int).Sub(N, n)
		nMinusOne := new(big.Int).Sub(N, oneInt)

		num := new(big.Int).Mul(n, K)
		num.Mul(num, nMinusK)
		num.Mul(num, nMinusDraw)

		denom := new(big.Int).Mul(N, N)
		denom.Mul(denom, nMinusOne)

		ans := new(big.Rat).SetFrac(num, denom)
		return NewRationalFromBigRat(ans), nil

	default:
		return nil, fmt.Errorf("%s: %s", MsgErrInvalidDistribution, args[0].String())
	}
}

// EvalStdDev calculates standard deviation sigma = sqrt(V[X]).
func EvalStdDev(args []Node) (Node, error) {
	vNode, err := EvalVariance(args)
	if err != nil {
		return nil, err
	}
	return simplifySqrt(vNode)
}

// evalPairsExpect calculates E[X] = sum(x_i * p_i) for discrete probability pairs.
func evalPairsExpect(pairs []probPair) (Node, error) {
	terms := make([]Node, len(pairs))
	for i, pair := range pairs {
		pNode := NewRationalFromBigRat(pair.p)
		terms[i] = &MulNode{Factors: []Node{pNode, pair.x}}
	}
	return Eval(&AddNode{Terms: terms})
}

// evalPairsVariance calculates V[X] = E[X^2] - (E[X])^2 for discrete probability pairs.
func evalPairsVariance(pairs []probPair) (Node, error) {
	mean, err := evalPairsExpect(pairs)
	if err != nil {
		return nil, err
	}

	termsSq := make([]Node, len(pairs))
	for i, pair := range pairs {
		pNode := NewRationalFromBigRat(pair.p)
		xSq := &PowNode{
			Base: pair.x,
			Exp:  mustRational(2, 1),
		}
		termsSq[i] = &MulNode{Factors: []Node{pNode, xSq}}
	}

	meanSq, err := Eval(&AddNode{Terms: termsSq})
	if err != nil {
		return nil, err
	}

	meanSquared := &PowNode{
		Base: mean,
		Exp:  mustRational(2, 1),
	}
	evaledMeanSquared, err := Eval(meanSquared)
	if err != nil {
		return nil, err
	}

	diff := &AddNode{
		Terms: []Node{
			meanSq,
			&MulNode{
				Factors: []Node{
					mustRational(-1, 1),
					evaledMeanSquared,
				},
			},
		},
	}
	return Eval(diff)
}
