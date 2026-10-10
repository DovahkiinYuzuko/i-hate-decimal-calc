package calc

import (
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// Cyclotomic8 represents an element of the dyadic cyclotomic ring Z[1/sqrt(2), i] = D[omega],
// where omega = e^(i*pi/4) = (1+i)/sqrt(2), omega^4 = -1, omega^8 = 1.
// Any element is uniquely represented in lowest terms as:
//
//	(A0 + A1*omega + A2*omega^2 + A3*omega^3) / sqrt(2)^K
//
// where A0, A1, A2, A3 are integers, K >= 0 is the Smallest Denominator Exponent (SDE).
type Cyclotomic8 struct {
	A0 *big.Int
	A1 *big.Int
	A2 *big.Int
	A3 *big.Int
	K  int
}

// NewCyclotomic8 creates a new Cyclotomic8 element and immediately applies SDE reduction.
func NewCyclotomic8(a0, a1, a2, a3 *big.Int, k int) Cyclotomic8 {
	c := Cyclotomic8{
		A0: new(big.Int).Set(a0),
		A1: new(big.Int).Set(a1),
		A2: new(big.Int).Set(a2),
		A3: new(big.Int).Set(a3),
		K:  k,
	}
	return c.SdeReduce()
}

// NewCyclotomic8Int creates an integer scalar Cyclotomic8 element.
func NewCyclotomic8Int(val int64) Cyclotomic8 {
	return Cyclotomic8{
		A0: big.NewInt(val),
		A1: big.NewInt(0),
		A2: big.NewInt(0),
		A3: big.NewInt(0),
		K:  0,
	}
}

// Cyclotomic8Zero returns the additive identity 0.
func Cyclotomic8Zero() Cyclotomic8 {
	return NewCyclotomic8Int(0)
}

// Cyclotomic8One returns the multiplicative identity 1.
func Cyclotomic8One() Cyclotomic8 {
	return NewCyclotomic8Int(1)
}

// Cyclotomic8Omega returns the 8th root of unity omega = e^(i*pi/4).
func Cyclotomic8Omega() Cyclotomic8 {
	return Cyclotomic8{
		A0: big.NewInt(0),
		A1: big.NewInt(1),
		A2: big.NewInt(0),
		A3: big.NewInt(0),
		K:  0,
	}
}

// Cyclotomic8InvSqrt2 returns 1/sqrt(2), used in Hadamard gates.
func Cyclotomic8InvSqrt2() Cyclotomic8 {
	return Cyclotomic8{
		A0: big.NewInt(1),
		A1: big.NewInt(0),
		A2: big.NewInt(0),
		A3: big.NewInt(0),
		K:  1,
	}
}

// mulSqrt2Numerator multiplies the numerator (a0, a1, a2, a3) by sqrt(2) = omega - omega^3.
// (a0 + a1*w + a2*w^2 + a3*w^3)*(w - w^3) = (a1 - a3) + (a0 + a2)*w + (a1 + a3)*w^2 + (a2 - a0)*w^3
func mulSqrt2Numerator(a0, a1, a2, a3 *big.Int) (*big.Int, *big.Int, *big.Int, *big.Int) {
	r0 := new(big.Int).Sub(a1, a3)
	r1 := new(big.Int).Add(a0, a2)
	r2 := new(big.Int).Add(a1, a3)
	r3 := new(big.Int).Sub(a2, a0)
	return r0, r1, r2, r3
}

// SdeReduce reduces the element to its canonical Smallest Denominator Exponent (SDE).
// A numerator is divisible by sqrt(2) iff:
//
//	b0 == b2 (mod 2) and b1 == b3 (mod 2)
//
// If so and K > 0, we can divide by sqrt(2) and decrement K.
func (c Cyclotomic8) SdeReduce() Cyclotomic8 {
	if c.IsZero() {
		return Cyclotomic8Zero()
	}

	a0 := new(big.Int).Set(c.A0)
	a1 := new(big.Int).Set(c.A1)
	a2 := new(big.Int).Set(c.A2)
	a3 := new(big.Int).Set(c.A3)
	k := c.K

	two := big.NewInt(2)
	m0 := new(big.Int)
	m1 := new(big.Int)
	m2 := new(big.Int)
	m3 := new(big.Int)

	for k > 0 {
		m0.Mod(a0, two)
		m1.Mod(a1, two)
		m2.Mod(a2, two)
		m3.Mod(a3, two)

		// Check divisibility by sqrt(2)
		if m0.Cmp(m2) == 0 && m1.Cmp(m3) == 0 {
			// Quotients:
			// newA0 = (a1 - a3) / 2
			// newA1 = (a0 + a2) / 2
			// newA2 = (a1 + a3) / 2
			// newA3 = (a2 - a0) / 2
			// Wait, let's invert mulSqrt2:
			// If b = sqrt(2)*a, then b0 = a1 - a3, b1 = a0 + a2, b2 = a1 + a3, b3 = a2 - a0
			// b0 + b2 = 2*a1 => a1 = (b0 + b2)/2
			// b2 - b0 = 2*a3 => a3 = (b2 - b0)/2
			// b1 - b3 = 2*a0 => a0 = (b1 - b3)/2
			// b1 + b3 = 2*a2 => a2 = (b1 + b3)/2
			nA0 := new(big.Int).Sub(a1, a3)
			nA0.Div(nA0, two)

			nA1 := new(big.Int).Add(a0, a2)
			nA1.Div(nA1, two)

			nA2 := new(big.Int).Add(a1, a3)
			nA2.Div(nA2, two)

			nA3 := new(big.Int).Sub(a2, a0)
			nA3.Div(nA3, two)

			a0, a1, a2, a3 = nA0, nA1, nA2, nA3
			k--
		} else {
			break
		}
	}

	return Cyclotomic8{A0: a0, A1: a1, A2: a2, A3: a3, K: k}
}

// IsZero returns true if the element is strictly 0.
func (c Cyclotomic8) IsZero() bool {
	return c.A0.Sign() == 0 && c.A1.Sign() == 0 && c.A2.Sign() == 0 && c.A3.Sign() == 0
}

// Equals checks canonical equality of two Cyclotomic8 numbers.
func (c Cyclotomic8) Equals(other Cyclotomic8) bool {
	diff := c.Sub(other)
	return diff.IsZero()
}

// Neg returns the negation -c.
func (c Cyclotomic8) Neg() Cyclotomic8 {
	return Cyclotomic8{
		A0: new(big.Int).Neg(c.A0),
		A1: new(big.Int).Neg(c.A1),
		A2: new(big.Int).Neg(c.A2),
		A3: new(big.Int).Neg(c.A3),
		K:  c.K,
	}
}

// Add computes c + other.
func (c Cyclotomic8) Add(other Cyclotomic8) Cyclotomic8 {
	kMax := c.K
	if other.K > kMax {
		kMax = other.K
	}

	a0, a1, a2, a3 := c.scaleNumeratorTo(kMax)
	b0, b1, b2, b3 := other.scaleNumeratorTo(kMax)

	r0 := new(big.Int).Add(a0, b0)
	r1 := new(big.Int).Add(a1, b1)
	r2 := new(big.Int).Add(a2, b2)
	r3 := new(big.Int).Add(a3, b3)

	return Cyclotomic8{A0: r0, A1: r1, A2: r2, A3: r3, K: kMax}.SdeReduce()
}

// Sub computes c - other.
func (c Cyclotomic8) Sub(other Cyclotomic8) Cyclotomic8 {
	return c.Add(other.Neg())
}

// scaleNumeratorTo scales the numerator by sqrt(2)^(targetK - c.K).
func (c Cyclotomic8) scaleNumeratorTo(targetK int) (*big.Int, *big.Int, *big.Int, *big.Int) {
	a0 := new(big.Int).Set(c.A0)
	a1 := new(big.Int).Set(c.A1)
	a2 := new(big.Int).Set(c.A2)
	a3 := new(big.Int).Set(c.A3)

	diff := targetK - c.K
	for i := 0; i < diff; i++ {
		a0, a1, a2, a3 = mulSqrt2Numerator(a0, a1, a2, a3)
	}
	return a0, a1, a2, a3
}

// Mul computes c * other using polynomial multiplication modulo omega^4 + 1.
func (c Cyclotomic8) Mul(other Cyclotomic8) Cyclotomic8 {
	// Formula modulo omega^4 = -1:
	// r0 = a0*b0 - a1*b3 - a2*b2 - a3*b1
	// r1 = a0*b1 + a1*b0 - a2*b3 - a3*b2
	// r2 = a0*b2 + a1*b1 + a2*b0 - a3*b3
	// r3 = a0*b3 + a1*b2 + a2*b1 + a3*b0
	p := func(x, y *big.Int) *big.Int {
		return new(big.Int).Mul(x, y)
	}

	r0 := new(big.Int).Sub(p(c.A0, other.A0), p(c.A1, other.A3))
	r0.Sub(r0, p(c.A2, other.A2))
	r0.Sub(r0, p(c.A3, other.A1))

	r1 := new(big.Int).Add(p(c.A0, other.A1), p(c.A1, other.A0))
	r1.Sub(r1, p(c.A2, other.A3))
	r1.Sub(r1, p(c.A3, other.A2))

	r2 := new(big.Int).Add(p(c.A0, other.A2), p(c.A1, other.A1))
	r2.Add(r2, p(c.A2, other.A0))
	r2.Sub(r2, p(c.A3, other.A3))

	r3 := new(big.Int).Add(p(c.A0, other.A3), p(c.A1, other.A2))
	r3.Add(r3, p(c.A2, other.A1))
	r3.Add(r3, p(c.A3, other.A0))

	return Cyclotomic8{A0: r0, A1: r1, A2: r2, A3: r3, K: c.K + other.K}.SdeReduce()
}

// Conj computes the complex conjugate:
// conj(omega) = -omega^3, conj(omega^2) = -omega^2, conj(omega^3) = -omega.
// conj(a0 + a1*w + a2*w^2 + a3*w^3) = a0 - a3*w - a2*w^2 - a1*w^3.
func (c Cyclotomic8) Conj() Cyclotomic8 {
	return Cyclotomic8{
		A0: new(big.Int).Set(c.A0),
		A1: new(big.Int).Neg(c.A3),
		A2: new(big.Int).Neg(c.A2),
		A3: new(big.Int).Neg(c.A1),
		K:  c.K,
	}.SdeReduce()
}

// IsRootOfUnity checks whether c is an 8th root of unity omega^k for k in {0, ..., 7}.
// If so, it returns (k, true). Otherwise, it returns (0, false).
func (c Cyclotomic8) IsRootOfUnity() (int, bool) {
	red := c.SdeReduce()
	if red.K != 0 {
		return 0, false
	}

	one := big.NewInt(1)
	negOne := big.NewInt(-1)
	zero := big.NewInt(0)

	// k=0: 1
	if red.A0.Cmp(one) == 0 && red.A1.Cmp(zero) == 0 && red.A2.Cmp(zero) == 0 && red.A3.Cmp(zero) == 0 {
		return 0, true
	}
	// k=1: omega
	if red.A0.Cmp(zero) == 0 && red.A1.Cmp(one) == 0 && red.A2.Cmp(zero) == 0 && red.A3.Cmp(zero) == 0 {
		return 1, true
	}
	// k=2: omega^2 = i
	if red.A0.Cmp(zero) == 0 && red.A1.Cmp(zero) == 0 && red.A2.Cmp(one) == 0 && red.A3.Cmp(zero) == 0 {
		return 2, true
	}
	// k=3: omega^3
	if red.A0.Cmp(zero) == 0 && red.A1.Cmp(zero) == 0 && red.A2.Cmp(zero) == 0 && red.A3.Cmp(one) == 0 {
		return 3, true
	}
	// k=4: -1
	if red.A0.Cmp(negOne) == 0 && red.A1.Cmp(zero) == 0 && red.A2.Cmp(zero) == 0 && red.A3.Cmp(zero) == 0 {
		return 4, true
	}
	// k=5: -omega
	if red.A0.Cmp(zero) == 0 && red.A1.Cmp(negOne) == 0 && red.A2.Cmp(zero) == 0 && red.A3.Cmp(zero) == 0 {
		return 5, true
	}
	// k=6: -i
	if red.A0.Cmp(zero) == 0 && red.A1.Cmp(zero) == 0 && red.A2.Cmp(negOne) == 0 && red.A3.Cmp(zero) == 0 {
		return 6, true
	}
	// k=7: -omega^3
	if red.A0.Cmp(zero) == 0 && red.A1.Cmp(zero) == 0 && red.A2.Cmp(zero) == 0 && red.A3.Cmp(negOne) == 0 {
		return 7, true
	}

	return 0, false
}

// ToASTNode converts Cyclotomic8 to an exact symbolic AST Node.
func (c Cyclotomic8) ToASTNode() ast.Node {
	red := c.SdeReduce()
	if red.IsZero() {
		return ast.NewRationalFromBigRat(big.NewRat(0, 1))
	}

	m := red.K / 2
	isOdd := red.K%2 != 0

	denom2m := new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(m)), nil)
	denom2m1 := new(big.Int).Mul(denom2m, big.NewInt(2))

	var realRatPart, realSqrtPart *big.Rat
	var imagRatPart, imagSqrtPart *big.Rat

	sub13 := new(big.Int).Sub(red.A1, red.A3)
	add13 := new(big.Int).Add(red.A1, red.A3)

	if !isOdd {
		realRatPart = new(big.Rat).SetFrac(red.A0, denom2m)
		realSqrtPart = new(big.Rat).SetFrac(sub13, denom2m1)

		imagRatPart = new(big.Rat).SetFrac(red.A2, denom2m)
		imagSqrtPart = new(big.Rat).SetFrac(add13, denom2m1)
	} else {
		realRatPart = new(big.Rat).SetFrac(sub13, denom2m1)
		realSqrtPart = new(big.Rat).SetFrac(red.A0, denom2m1)

		imagRatPart = new(big.Rat).SetFrac(add13, denom2m1)
		imagSqrtPart = new(big.Rat).SetFrac(red.A2, denom2m1)
	}

	buildPart := func(ratPart, sqrtPart *big.Rat) ast.Node {
		var terms []ast.Node
		if ratPart.Sign() != 0 {
			terms = append(terms, ast.NewRationalFromBigRat(ratPart))
		}
		if sqrtPart.Sign() != 0 {
			sqrt2 := &ast.SqrtNode{Radicand: ast.NewRationalFromBigRat(big.NewRat(2, 1))}
			if sqrtPart.Cmp(big.NewRat(1, 1)) == 0 {
				terms = append(terms, sqrt2)
			} else if sqrtPart.Cmp(big.NewRat(-1, 1)) == 0 {
				terms = append(terms, &ast.MulNode{Factors: []ast.Node{ast.NewRationalFromBigRat(big.NewRat(-1, 1)), sqrt2}})
			} else {
				terms = append(terms, &ast.MulNode{Factors: []ast.Node{ast.NewRationalFromBigRat(sqrtPart), sqrt2}})
			}
		}
		if len(terms) == 0 {
			return ast.NewRationalFromBigRat(big.NewRat(0, 1))
		}
		if len(terms) == 1 {
			return terms[0]
		}
		return &ast.AddNode{Terms: terms}
	}

	realNode := buildPart(realRatPart, realSqrtPart)
	imagNode := buildPart(imagRatPart, imagSqrtPart)

	if imagNode.String() == "0" {
		return realNode
	}
	return ast.NewComplex(realNode, imagNode)
}
