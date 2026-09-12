package calc

import (
	"fmt"
	"math/big"
)

// Sign represents the exact algebraic sign of an expression.
type Sign int

const (
	SignNegative Sign = -1
	SignZero     Sign = 0
	SignPositive Sign = 1
	SignUnknown  Sign = 2
)

// String returns the human-readable string representation of a Sign.
func (s Sign) String() string {
	switch s {
	case SignNegative:
		return "Negative"
	case SignZero:
		return "Zero"
	case SignPositive:
		return "Positive"
	default:
		return "Unknown"
	}
}

// ExactSign determines the exact algebraic sign of an AST node (Positive, Negative, Zero, or Unknown).
// It completely avoids floating-point epsilon comparisons and evaluates rational numbers, radical terms,
// and quadratic irrationals (a + b*sqrt(c), a*sqrt(b) + c*sqrt(d)) via exact algebraic squaring and norm comparisons.
func ExactSign(n Node) (Sign, error) {
	if n == nil {
		return SignUnknown, fmt.Errorf("cannot determine sign of nil node")
	}

	simplified, err := Eval(n)
	if err != nil {
		return SignUnknown, err
	}

	return exactSignEval(simplified, 0), nil
}

// signNode evaluates the exact sign as an int (-1, 0, 1) or returns an error if unknown.
// This function replaces the old float64-based signNode in geometry calculations.
func signNode(n Node) (int, error) {
	s, err := ExactSign(n)
	if err != nil {
		return 0, err
	}
	switch s {
	case SignPositive:
		return 1, nil
	case SignNegative:
		return -1, nil
	case SignZero:
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot determine exact sign for expression: %s", n.String())
	}
}

func exactSignEval(n Node, depth int) Sign {
	if n == nil {
		return SignUnknown
	}

	if isZeroNode(n) || isZero(n) {
		return SignZero
	}

	switch v := n.(type) {
	case *RationalNode:
		cmp := v.Val.Sign()
		if cmp > 0 {
			return SignPositive
		} else if cmp < 0 {
			return SignNegative
		}
		return SignZero

	case *SqrtNode:
		radSign := exactSignEval(v.Radicand, depth)
		switch radSign {
		case SignPositive:
			return SignPositive
		case SignZero:
			return SignZero
		default:
			return SignUnknown
		}

	case *MulNode:
		negCount := 0
		for _, f := range v.Factors {
			s := exactSignEval(f, depth)
			if s == SignZero {
				return SignZero
			}
			if s == SignUnknown {
				return SignUnknown
			}
			if s == SignNegative {
				negCount++
			}
		}
		if negCount%2 == 1 {
			return SignNegative
		}
		return SignPositive

	case *PowNode:
		baseSign := exactSignEval(v.Base, depth)
		if baseSign == SignZero {
			return SignZero
		}

		if expRat, ok := v.Exp.(*RationalNode); ok {
			num := expRat.Val.Num()
			denom := expRat.Val.Denom()

			// If denominator is even (e.g. 1/2 for square root)
			if new(big.Int).Rem(denom, big.NewInt(2)).Sign() == 0 {
				if baseSign == SignPositive {
					// Principal root of positive real number is strictly positive
					return SignPositive
				}
				// Even root of negative number is non-real (imaginary)
				return SignUnknown
			}

			// Denominator is odd (e.g. 1/1, 1/3, 1/5)
			if new(big.Int).Rem(num, big.NewInt(2)).Sign() == 0 {
				// Even power of non-zero real number is strictly positive
				return SignPositive
			}
			// Odd power preserves base sign
			return baseSign
		}
		return SignUnknown

	case *AddNode:
		return exactSignAdd(v, depth)

	default:
		return SignUnknown
	}
}

func exactSignAdd(add *AddNode, depth int) Sign {
	terms := add.Terms
	if len(terms) == 0 {
		return SignZero
	}
	if len(terms) == 1 {
		return exactSignEval(terms[0], depth)
	}

	// 1. Try to evaluate signs of all terms
	allPositive := true
	allNegative := true
	allZero := true
	hasUnknown := false

	termSigns := make([]Sign, len(terms))
	for i, t := range terms {
		s := exactSignEval(t, depth)
		termSigns[i] = s
		switch s {
		case SignUnknown:
			hasUnknown = true
			allPositive = false
			allNegative = false
			allZero = false
		case SignPositive:
			allNegative = false
			allZero = false
		case SignNegative:
			allPositive = false
			allZero = false
		}
	}

	if allZero && !hasUnknown {
		return SignZero
	}
	if allPositive && !hasUnknown {
		return SignPositive
	}
	if allNegative && !hasUnknown {
		return SignNegative
	}

	// 2. Algebraic radical comparison for two terms: A*sqrt(B) + C*sqrt(D)
	if len(terms) == 2 {
		c1, r1, ok1 := extractRadicalTerm(terms[0])
		c2, r2, ok2 := extractRadicalTerm(terms[1])
		if ok1 && ok2 {
			// terms[0] = c1 * sqrt(r1)
			// terms[1] = c2 * sqrt(r2)
			// If one is positive and one is negative:
			if (c1.Sign() > 0 && c2.Sign() < 0) || (c1.Sign() < 0 && c2.Sign() > 0) {
				// Compare squares: (c1 * sqrt(r1))^2 vs (|c2| * sqrt(r2))^2
				// sq1 = c1^2 * r1
				// sq2 = c2^2 * r2
				c1Sq := new(big.Rat).Mul(c1, c1)
				sq1 := new(big.Rat).Mul(c1Sq, r1)

				c2Sq := new(big.Rat).Mul(c2, c2)
				sq2 := new(big.Rat).Mul(c2Sq, r2)

				cmp := sq1.Cmp(sq2)
				if c1.Sign() > 0 {
					// term0 > 0, term1 < 0
					if cmp > 0 {
						return SignPositive
					} else if cmp < 0 {
						return SignNegative
					}
					return SignZero
				} else {
					// term0 < 0, term1 > 0
					if cmp > 0 {
						return SignNegative
					} else if cmp < 0 {
						return SignPositive
					}
					return SignZero
				}
			}
		}
	}

	// 3. Difference of squares for positive terms vs negative terms (P - N)
	if depth < 2 && !hasUnknown {
		var posTerms []Node
		var negTerms []Node
		for i, t := range terms {
			switch termSigns[i] {
			case SignPositive:
				posTerms = append(posTerms, t)
			case SignNegative:
				// Abs of negative term: -t
				negAbs, err := geoMul(mustRational(-1, 1), t)
				if err == nil {
					negTerms = append(negTerms, negAbs)
				}
			}
		}

		if len(posTerms) > 0 && len(negTerms) > 0 {
			var pNode, nNode Node
			if len(posTerms) == 1 {
				pNode = posTerms[0]
			} else {
				pNode = NewAdd(posTerms)
			}
			if len(negTerms) == 1 {
				nNode = negTerms[0]
			} else {
				nNode = NewAdd(negTerms)
			}

			// Compute P^2 - N^2 = (P - N)(P + N). Since P > 0 and N > 0, sign(P - N) == sign(P^2 - N^2).
			pSq, err1 := geoMul(pNode, pNode)
			nSq, err2 := geoMul(nNode, nNode)
			if err1 == nil && err2 == nil {
				diffSq, err3 := geoSub(pSq, nSq)
				if err3 == nil {
					simplifiedDiff, err4 := Eval(diffSq)
					if err4 == nil {
						sDiff := exactSignEval(simplifiedDiff, depth+1)
						if sDiff != SignUnknown {
							return sDiff
						}
					}
				}
			}
		}
	}

	return SignUnknown
}

// extractRadicalTerm decomposes a node into coeff * sqrt(radicand) where coeff and radicand are rational numbers.
// Returns (coeff, radicand, true) if match succeeds, or (nil, nil, false) otherwise.
func extractRadicalTerm(n Node) (*big.Rat, *big.Rat, bool) {
	if n == nil {
		return nil, nil, false
	}

	switch v := n.(type) {
	case *RationalNode:
		// c * sqrt(1)
		return new(big.Rat).Set(v.Val), big.NewRat(1, 1), true

	case *SqrtNode:
		// sqrt(r) = 1 * sqrt(r)
		if baseRat, ok := v.Radicand.(*RationalNode); ok && baseRat.Val.Sign() >= 0 {
			return big.NewRat(1, 1), new(big.Rat).Set(baseRat.Val), true
		}

	case *PowNode:
		// r^(1/2) = 1 * sqrt(r)
		if expRat, ok := v.Exp.(*RationalNode); ok && expRat.Val.Cmp(big.NewRat(1, 2)) == 0 {
			if baseRat, ok := v.Base.(*RationalNode); ok && baseRat.Val.Sign() >= 0 {
				return big.NewRat(1, 1), new(big.Rat).Set(baseRat.Val), true
			}
		}

	case *MulNode:
		// Check for c * sqrt(r)
		var ratCoeff *big.Rat
		var radicand *big.Rat
		seenRadical := false

		for _, factor := range v.Factors {
			if rNode, ok := factor.(*RationalNode); ok {
				if ratCoeff == nil {
					ratCoeff = new(big.Rat).Set(rNode.Val)
				} else {
					ratCoeff.Mul(ratCoeff, rNode.Val)
				}
			} else if sqrtNode, ok := factor.(*SqrtNode); ok {
				if baseRat, ok := sqrtNode.Radicand.(*RationalNode); ok && baseRat.Val.Sign() >= 0 {
					if seenRadical {
						return nil, nil, false
					}
					seenRadical = true
					radicand = new(big.Rat).Set(baseRat.Val)
				} else {
					return nil, nil, false
				}
			} else if powNode, ok := factor.(*PowNode); ok {
				if expRat, ok := powNode.Exp.(*RationalNode); ok && expRat.Val.Cmp(big.NewRat(1, 2)) == 0 {
					if baseRat, ok := powNode.Base.(*RationalNode); ok && baseRat.Val.Sign() >= 0 {
						if seenRadical {
							return nil, nil, false // multiple radicals in product not handled here
						}
						seenRadical = true
						radicand = new(big.Rat).Set(baseRat.Val)
					} else {
						return nil, nil, false
					}
				} else {
					return nil, nil, false
				}
			} else {
				return nil, nil, false
			}
		}

		if seenRadical && radicand != nil {
			if ratCoeff == nil {
				ratCoeff = big.NewRat(1, 1)
			}
			return ratCoeff, radicand, true
		}
	}

	return nil, nil, false
}
