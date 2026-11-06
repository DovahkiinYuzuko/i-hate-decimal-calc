package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// extractRat extracts *big.Rat from a Node if it is a RationalNode.
func extractRat(n Node, context string) (*big.Rat, error) {
	rat, ok := n.(*RationalNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_expected_rational_number_got", context, n.String()))
	}
	return new(big.Rat).Set(rat.Val), nil
}

// extractNonNegativeInt extracts a non-negative *big.Int from a RationalNode with denominator 1.
func extractNonNegativeInt(n Node, context string) (*big.Int, error) {
	rat, ok := n.(*RationalNode)
	if !ok || !rat.Val.IsInt() {
		return nil, fmt.Errorf("%s: %s", context, i18n.T("errors.prob_non_negative_int"))
	}
	val := rat.Val.Num()
	if val.Sign() < 0 {
		return nil, fmt.Errorf("%s: %s", context, i18n.T("errors.prob_non_negative_int"))
	}
	return new(big.Int).Set(val), nil
}

// extractPositiveInt extracts a positive *big.Int (> 0) from a RationalNode with denominator 1.
func extractPositiveInt(n Node, context string) (*big.Int, error) {
	rat, ok := n.(*RationalNode)
	if !ok || !rat.Val.IsInt() {
		return nil, fmt.Errorf("%s: %s", context, i18n.T("errors.prob_positive_int"))
	}
	val := rat.Val.Num()
	if val.Sign() <= 0 {
		return nil, fmt.Errorf("%s: %s", context, i18n.T("errors.prob_positive_int"))
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
		return nil, fmt.Errorf("%s: %s (got %s)", context, i18n.T("errors.prob_out_of_range"), rat.RatString())
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
		return nil, fmt.Errorf("%s", i18n.T("prob.err_binom_requires_3_arguments_n", len(args)))
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
		return nil, fmt.Errorf("%s", i18n.T("prob.err_hyper_requires_4_arguments_n", len(args)))
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
		return nil, fmt.Errorf("%s: K and n cannot exceed N (got N=%s, K=%s, n=%s)", i18n.T("errors.hyper_params"), N.String(), K.String(), n.String())
	}
	if k.Cmp(K) > 0 || k.Cmp(n) > 0 {
		return mustRational(0, 1), nil
	}
	// Support case where k < 0 or k < n - (N - K)
	nMinusK := new(big.Int).Sub(N, K)
	minK := new(big.Int).Sub(n, nMinusK)
	if minK.Sign() > 0 && k.Cmp(minK) < 0 {
		return mustRational(0, 1), nil
	}

	combKk := combBig(K, k)
	nMinusK_nMinusK := combBig(nMinusK, new(big.Int).Sub(n, k))
	num := new(big.Int).Mul(combKk, nMinusK_nMinusK)
	denom := combBig(N, n)
	if denom.Sign() == 0 {
		return nil, fmt.Errorf("%s: division by zero in hypergeometric denominator", i18n.T("errors.hyper_params"))
	}

	ans := new(big.Rat).SetFrac(num, denom)
	return NewRationalFromBigRat(ans), nil
}

// EvalGeomPMF calculates geometric distribution PMF P(X = k) = (1-p)^(k-1) * p.
func EvalGeomPMF(args []Node) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_geom_requires_2_arguments_p", len(args)))
	}

	p, err := extractProbRat(args[0], "geom: p")
	if err != nil {
		return nil, err
	}
	if p.Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_geom_p_cannot_be_zero"))
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
		return nil, fmt.Errorf("%s", i18n.T("prob.err_bayes_requires_3_arguments_prior", len(args)))
	}

	priorRat, err := extractProbRat(args[0], "bayes: prior P(A)")
	if err != nil {
		return nil, err
	}
	likelihoodRat, err := extractProbRat(args[1], "bayes: likelihood P(B|A)")
	if err != nil {
		return nil, err
	}
	marginalRat, err := extractProbRat(args[2], "bayes: marginal P(B)")
	if err != nil {
		return nil, err
	}

	if marginalRat.Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.bayes_zero_evidence"))
	}

	num := new(big.Rat).Mul(priorRat, likelihoodRat)
	ans := new(big.Rat).Quo(num, marginalRat)

	one := big.NewRat(1, 1)
	if ans.Cmp(one) > 0 {
		return nil, fmt.Errorf("%s: posterior exceeds 1 (got %s)", i18n.T("errors.prob_out_of_range"), ans.RatString())
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
				return nil, fmt.Errorf("%s", i18n.T("prob.err_each_element_of_discrete_distribution", elem.String()))
			}
			pRat, err := extractProbRat(pairList.Elements[1], "discrete list: prob")
			if err != nil {
				return nil, err
			}
			pairs = append(pairs, probPair{x: pairList.Elements[0], p: pRat})
		}

	case *MatrixNode:
		if v.Cols != 2 {
			return nil, fmt.Errorf("%s", i18n.T("prob.err_discrete_distribution_matrix_must_have", v.Cols))
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
		return nil, fmt.Errorf("%s", i18n.T("prob.err_discrete_distribution_cannot_be_empty"))
	}

	sumProb := big.NewRat(0, 1)
	for _, pair := range pairs {
		sumProb.Add(sumProb, pair.p)
	}
	one := big.NewRat(1, 1)
	if sumProb.Cmp(one) != 0 {
		return nil, fmt.Errorf("%s (sum = %s)", i18n.T("errors.prob_sum_not_one"), sumProb.RatString())
	}

	return pairs, nil
}

// EvalExpect calculates expected value E[X].
func EvalExpect(args []Node) (Node, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_expect_requires_at_least_1"))
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
			return nil, fmt.Errorf("%s", i18n.T("prob.err_expect_binom_requires_3_arguments", len(args)))
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
			return nil, fmt.Errorf("%s", i18n.T("prob.err_expect_geom_requires_2_arguments", len(args)))
		}
		p, err := extractProbRat(args[1], "expect: p")
		if err != nil {
			return nil, err
		}
		if p.Sign() == 0 {
			return nil, fmt.Errorf("%s", i18n.T("prob.err_expect_p_cannot_be_zero"))
		}
		one := big.NewRat(1, 1)
		ans := new(big.Rat).Quo(one, p)
		return NewRationalFromBigRat(ans), nil

	case "hyper":
		if len(args) != 4 {
			return nil, fmt.Errorf("%s", i18n.T("prob.err_expect_hyper_requires_4_arguments", len(args)))
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
		if N.Sign() <= 0 || n.Cmp(N) > 0 {
			return nil, fmt.Errorf("%s", i18n.T("errors.hyper_params"))
		}
		// E[X] = n * K / N
		num := new(big.Int).Mul(n, K)
		rat := new(big.Rat).SetFrac(num, N)
		return NewRationalFromBigRat(rat), nil

	default:
		return nil, fmt.Errorf("%s: %s", i18n.T("errors.invalid_distribution"), args[0].String())
	}
}

// EvalVariance calculates variance V[X] = E[X^2] - (E[X])^2.
func EvalVariance(args []Node) (Node, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_variance_requires_at_least_1"))
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
			return nil, fmt.Errorf("%s", i18n.T("prob.err_variance_binom_requires_3_arguments", len(args)))
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
			return nil, fmt.Errorf("%s", i18n.T("prob.err_variance_geom_requires_2_arguments", len(args)))
		}
		p, err := extractProbRat(args[1], "variance: p")
		if err != nil {
			return nil, err
		}
		if p.Sign() == 0 {
			return nil, fmt.Errorf("%s", i18n.T("prob.err_variance_p_cannot_be_zero"))
		}
		// V = (1 - p) / p^2
		one := big.NewRat(1, 1)
		oneMinusP := new(big.Rat).Sub(one, p)
		pSquared := new(big.Rat).Mul(p, p)
		ans := new(big.Rat).Quo(oneMinusP, pSquared)
		return NewRationalFromBigRat(ans), nil

	case "hyper":
		if len(args) != 4 {
			return nil, fmt.Errorf("%s", i18n.T("prob.err_variance_hyper_requires_4_arguments", len(args)))
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
		// If N == 1, variance is 0
		oneInt := big.NewInt(1)
		if N.Cmp(oneInt) == 0 {
			return mustRational(0, 1), nil
		}
		if N.Sign() <= 0 || n.Cmp(N) > 0 {
			return nil, fmt.Errorf("%s", i18n.T("errors.hyper_params"))
		}
		numK := new(big.Int).Mul(K, new(big.Int).Sub(N, K))
		numN := new(big.Int).Mul(n, new(big.Int).Sub(N, n))
		num := new(big.Int).Mul(numK, numN)

		denomN := new(big.Int).Mul(N, N)
		denomNMinus1 := new(big.Int).Sub(N, big.NewInt(1))
		denom := new(big.Int).Mul(denomN, denomNMinus1)
		rat := new(big.Rat).SetFrac(num, denom)
		return NewRationalFromBigRat(rat), nil

	default:
		return nil, fmt.Errorf("%s: %s", i18n.T("errors.invalid_distribution"), args[0].String())
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


// EvalDiceProb calculates exact discrete probability distributions for dice rolls using PGF.
// Supported calls:
// - dice_prob(n, sides): returns [E, V] as a ListNode containing [expected_value, variance]
// - dice_prob(n, sides, target): returns exact rational probability P(sum == target)
func EvalDiceProb(args []Node) (Node, error) {
	if len(args) != 2 && len(args) != 3 {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_dice_prob_arg_count", len(args)))
	}

	nBig, err := extractNonNegativeInt(args[0], "dice_prob: n")
	if err != nil {
		return nil, err
	}
	sidesBig, err := extractPositiveInt(args[1], "dice_prob: sides")
	if err != nil {
		return nil, err
	}

	if !nBig.IsInt64() || nBig.Int64() > 10000 {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_dice_n_too_large"))
	}
	if !sidesBig.IsInt64() || sidesBig.Int64() > 1000 {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_dice_sides_too_large"))
	}

	n := int(nBig.Int64())
	sides := int(sidesBig.Int64())

	if len(args) == 2 {
		// Expectation: E = n * (sides + 1) / 2
		// Variance: V = n * (sides^2 - 1) / 12
		if n == 0 {
			return &ListNode{Elements: []Node{mustRational(0, 1), mustRational(0, 1)}}, nil
		}
		eNum := new(big.Int).Mul(nBig, big.NewInt(int64(sides+1)))
		eRat := new(big.Rat).SetFrac(eNum, big.NewInt(2))

		sidesSqMinus1 := int64(sides)*int64(sides) - 1
		vNum := new(big.Int).Mul(nBig, big.NewInt(sidesSqMinus1))
		vRat := new(big.Rat).SetFrac(vNum, big.NewInt(12))

		return &ListNode{
			Elements: []Node{
				NewRationalFromBigRat(eRat),
				NewRationalFromBigRat(vRat),
			},
		}, nil
	}

	// 3 arguments: dice_prob(n, sides, target)
	targetRat, ok := args[2].(*RationalNode)
	if !ok || !targetRat.Val.IsInt() {
		return mustRational(0, 1), nil
	}
	targetInt := targetRat.Val.Num()

	if n == 0 {
		if targetInt.Sign() == 0 {
			return mustRational(1, 1), nil
		}
		return mustRational(0, 1), nil
	}

	minSum := int64(n)
	maxSum := int64(n) * int64(sides)
	if targetInt.Cmp(big.NewInt(minSum)) < 0 || targetInt.Cmp(big.NewInt(maxSum)) > 0 {
		return mustRational(0, 1), nil
	}

	// Polynomial multiplication via binary exponentiation:
	// P(x) = sum_{k=1}^sides x^k
	polyBase := make([]*big.Int, sides+1)
	for i := range polyBase {
		polyBase[i] = big.NewInt(0)
	}
	for i := 1; i <= sides; i++ {
		polyBase[i] = big.NewInt(1)
	}

	polyPow := func(base []*big.Int, exp int) []*big.Int {
		res := []*big.Int{big.NewInt(1)}
		cur := make([]*big.Int, len(base))
		copy(cur, base)
		e := exp
		for e > 0 {
			if e%2 == 1 {
				nextRes := make([]*big.Int, len(res)+len(cur)-1)
				for i := range nextRes {
					nextRes[i] = big.NewInt(0)
				}
				for i := range res {
					if res[i].Sign() == 0 {
						continue
					}
					for j := range cur {
						if cur[j].Sign() == 0 {
							continue
						}
						prod := new(big.Int).Mul(res[i], cur[j])
						nextRes[i+j].Add(nextRes[i+j], prod)
					}
				}
				res = nextRes
			}
			e /= 2
			if e > 0 {
				nextCur := make([]*big.Int, len(cur)+len(cur)-1)
				for i := range nextCur {
					nextCur[i] = big.NewInt(0)
				}
				for i := range cur {
					if cur[i].Sign() == 0 {
						continue
					}
					for j := range cur {
						if cur[j].Sign() == 0 {
							continue
						}
						prod := new(big.Int).Mul(cur[i], cur[j])
						nextCur[i+j].Add(nextCur[i+j], prod)
					}
				}
				cur = nextCur
			}
		}
		return res
	}

	resPoly := polyPow(polyBase, n)
	targetIdx := int(targetInt.Int64())
	var ways *big.Int
	if targetIdx >= 0 && targetIdx < len(resPoly) {
		ways = resPoly[targetIdx]
	} else {
		ways = big.NewInt(0)
	}

	// Total outcomes = sides^n
	totalOutcomes := new(big.Int).Exp(big.NewInt(int64(sides)), big.NewInt(int64(n)), nil)
	probRat := new(big.Rat).SetFrac(ways, totalOutcomes)
	return NewRationalFromBigRat(probRat), nil
}

// EvalMarkovStationary calculates the exact stationary distribution pi for a Markov transition matrix P,
// such that pi * P = pi and sum(pi) = 1.
// If the Markov chain is reducible or has multiple stationary distributions, an error is returned.
func EvalMarkovStationary(args []Node) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_markov_arg_count", len(args)))
	}

	var mat *MatrixNode
	switch v := args[0].(type) {
	case *MatrixNode:
		mat = v
	default:
		return nil, fmt.Errorf("%s", i18n.T("prob.err_markov_matrix_required"))
	}

	if mat.Rows != mat.Cols || mat.Rows == 0 {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_markov_square_required", mat.Rows, mat.Cols))
	}
	n := mat.Rows

	// Check row stochasticity: each element >= 0, row sum == 1
	zeroRat := big.NewRat(0, 1)
	oneRat := big.NewRat(1, 1)

	P := make([][]*big.Rat, n)
	for r := 0; r < n; r++ {
		P[r] = make([]*big.Rat, n)
		rowSum := big.NewRat(0, 1)
		for c := 0; c < n; c++ {
			ratNode, err := Eval(mat.Data[r][c])
			if err != nil {
				return nil, err
			}
			rNode, ok := ratNode.(*RationalNode)
			if !ok {
				return nil, fmt.Errorf("%s", i18n.T("prob.err_markov_rational_required", r, c))
			}
			val := new(big.Rat).Set(rNode.Val)
			if val.Cmp(zeroRat) < 0 {
				return nil, fmt.Errorf("%s", i18n.T("prob.err_markov_negative_prob", r, c, val.RatString()))
			}
			rowSum.Add(rowSum, val)
			P[r][c] = val
		}
		if rowSum.Cmp(oneRat) != 0 {
			return nil, fmt.Errorf("%s", i18n.T("prob.err_markov_row_sum_not_one", r, rowSum.RatString()))
		}
	}

	// We solve:
	// sum_{i=0}^{n-1} pi_i (P[i][j] - delta_{ij}) = 0   for j = 0..n-1
	// and sum_{i=0}^{n-1} pi_i = 1
	//
	// Augmented matrix of size (n + 1) x (n + 1):
	// Row 0..n-1: column j of (P^T - I) with RHS 0
	// Row n: [1, 1, ..., 1 | 1]
	augData := make([][]Node, n+1)
	for j := 0; j < n; j++ {
		augData[j] = make([]Node, n+1)
		for i := 0; i < n; i++ {
			coef := new(big.Rat).Set(P[i][j])
			if i == j {
				coef.Sub(coef, oneRat)
			}
			augData[j][i] = NewRationalFromBigRat(coef)
		}
		augData[j][n] = mustRational(0, 1) // RHS
	}

	augData[n] = make([]Node, n+1)
	for i := 0; i < n; i++ {
		augData[n][i] = mustRational(1, 1)
	}
	augData[n][n] = mustRational(1, 1) // RHS

	augMat, err := NewMatrix(n+1, n+1, augData)
	if err != nil {
		return nil, err
	}

	rrefMat, pivotCols, err := computeRREF(augMat)
	if err != nil {
		return nil, err
	}

	// For a unique solution, the system must have rank n in the n unknowns,
	// meaning pivot columns must be exactly 0, 1, ..., n-1, and row n must be 0...0|0 (consistent).
	// If pivot column includes column n (RHS), it's inconsistent (0 = 1).
	// If rank < n, multiple solutions exist (reducible).
	hasInconsistency := false
	rank := len(pivotCols)
	for _, pCol := range pivotCols {
		if pCol == n {
			hasInconsistency = true
			break
		}
	}

	if hasInconsistency || rank != n {
		return nil, fmt.Errorf("%s", i18n.T("prob.err_markov_non_unique_stationary"))
	}

	// Extract solution pi_0, ..., pi_{n-1}
	sol := make([]Node, n)
	for r := 0; r < n; r++ {
		sol[r] = rrefMat.Data[r][n]
	}

	return &ListNode{Elements: sol}, nil
}
