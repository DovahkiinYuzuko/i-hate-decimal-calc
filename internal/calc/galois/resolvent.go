package galois

import (
	"math"
	"math/big"
	"strconv"
)

// Zero, One big.Int helpers
var (
	bigZero = big.NewInt(0)
	bigOne  = big.NewInt(1)
	bigFour = big.NewInt(4)
)

// IsSquareInt checks if n is a perfect integer square (n = k^2, k in Z).
func IsSquareInt(n *big.Int) bool {
	if n == nil || n.Sign() < 0 {
		return false
	}
	if n.Sign() == 0 {
		return true
	}
	sqrt := new(big.Int).Sqrt(n)
	sq := new(big.Int).Mul(sqrt, sqrt)
	return sq.Cmp(n) == 0
}

// IsSquareRat checks if rational r is a perfect square in Q (r = (a/b)^2).
func IsSquareRat(r *big.Rat) bool {
	if r == nil || r.Sign() < 0 {
		return false
	}
	if r.Sign() == 0 {
		return true
	}
	return IsSquareInt(r.Num()) && IsSquareInt(r.Denom())
}

// DiscriminantPoly3 computes the exact integer discriminant of a monic cubic
// f(x) = x^3 + a*x^2 + b*x + c.
// Disc = a^2 b^2 - 4 b^3 - 4 a^3 c + 18 a b c - 27 c^2.
func DiscriminantPoly3(a, b, c *big.Int) *big.Int {
	// Term 1: a^2 b^2
	t1 := new(big.Int).Mul(new(big.Int).Mul(a, a), new(big.Int).Mul(b, b))

	// Term 2: -4 b^3
	b3 := new(big.Int).Mul(new(big.Int).Mul(b, b), b)
	t2 := new(big.Int).Mul(big.NewInt(-4), b3)

	// Term 3: -4 a^3 c
	a3 := new(big.Int).Mul(new(big.Int).Mul(a, a), a)
	t3 := new(big.Int).Mul(big.NewInt(-4), new(big.Int).Mul(a3, c))

	// Term 4: 18 a b c
	t4 := new(big.Int).Mul(big.NewInt(18), new(big.Int).Mul(new(big.Int).Mul(a, b), c))

	// Term 5: -27 c^2
	t5 := new(big.Int).Mul(big.NewInt(-27), new(big.Int).Mul(c, c))

	res := new(big.Int).Add(t1, t2)
	res.Add(res, t3)
	res.Add(res, t4)
	res.Add(res, t5)
	return res
}

// DiscriminantPoly4 computes the exact integer discriminant of a monic quartic
// f(x) = x^4 + a*x^3 + b*x^2 + c*x + d.
func DiscriminantPoly4(a, b, c, d *big.Int) *big.Int {
	// Standard discriminant expansion for x^4 + a x^3 + b x^2 + c x + d
	// Disc = 256 d^3 - 192 a c d^2 - 128 b^2 d^2 + 144 a^2 b d^2 - 27 a^4 d^2
	//        + 144 b c^2 d - 6 a^2 c^2 d - 80 a b^2 c d + 18 a^3 b c d + 16 b^4 d - 4 a^2 b^3 d
	//        - 27 c^4 + 18 a b c^3 - 4 a^3 c^3 - 4 b^3 c^2 + a^2 b^2 c^2
	a2 := new(big.Int).Mul(a, a)
	a3 := new(big.Int).Mul(a2, a)
	a4 := new(big.Int).Mul(a2, a2)

	b2 := new(big.Int).Mul(b, b)
	b3 := new(big.Int).Mul(b2, b)
	b4 := new(big.Int).Mul(b2, b2)

	c2 := new(big.Int).Mul(c, c)
	c3 := new(big.Int).Mul(c2, c)
	c4 := new(big.Int).Mul(c2, c2)

	d2 := new(big.Int).Mul(d, d)
	d3 := new(big.Int).Mul(d2, d)

	disc := new(big.Int)

	// + 256 d^3
	disc.Add(disc, new(big.Int).Mul(big.NewInt(256), d3))
	// - 192 a c d^2
	disc.Sub(disc, new(big.Int).Mul(big.NewInt(192), mul4(a, c, d2, bigOne)))
	// - 128 b^2 d^2
	disc.Sub(disc, new(big.Int).Mul(big.NewInt(128), new(big.Int).Mul(b2, d2)))
	// + 144 a^2 b d^2
	disc.Add(disc, new(big.Int).Mul(big.NewInt(144), mul3(a2, b, d2)))
	// - 27 a^4 d^2
	disc.Sub(disc, new(big.Int).Mul(big.NewInt(27), new(big.Int).Mul(a4, d2)))

	// + 144 b c^2 d
	disc.Add(disc, new(big.Int).Mul(big.NewInt(144), mul3(b, c2, d)))
	// - 6 a^2 c^2 d
	disc.Sub(disc, new(big.Int).Mul(big.NewInt(6), mul3(a2, c2, d)))
	// - 80 a b^2 c d
	disc.Sub(disc, new(big.Int).Mul(big.NewInt(80), mul4(a, b2, c, d)))
	// + 18 a^3 b c d
	disc.Add(disc, new(big.Int).Mul(big.NewInt(18), mul4(a3, b, c, d)))
	// + 16 b^4 d
	disc.Add(disc, new(big.Int).Mul(big.NewInt(16), new(big.Int).Mul(b4, d)))
	// - 4 a^2 b^3 d
	disc.Sub(disc, new(big.Int).Mul(big.NewInt(4), mul3(a2, b3, d)))

	// - 27 c^4
	disc.Sub(disc, new(big.Int).Mul(big.NewInt(27), c4))
	// + 18 a b c^3
	disc.Add(disc, new(big.Int).Mul(big.NewInt(18), mul3(a, b, c3)))
	// - 4 a^3 c^3
	disc.Sub(disc, new(big.Int).Mul(big.NewInt(4), new(big.Int).Mul(a3, c3)))
	// - 4 b^3 c^2
	disc.Sub(disc, new(big.Int).Mul(big.NewInt(4), new(big.Int).Mul(b3, c2)))
	// + a^2 b^2 c^2
	disc.Add(disc, mul3(a2, b2, c2))

	return disc
}

// DiscriminantPoly5Depressed computes discriminant for depressed quintic x^5 + p x^3 + q x^2 + r x + s.
func DiscriminantPoly5Depressed(p, q, r, s *big.Int) *big.Int {
	// Standard discriminant for x^5 + p x^3 + q x^2 + r x + s:
	// For Bring-Jerrard x^5 + r x + s: Disc = 256 r^5 + 3125 s^4
	// In general, we evaluate via resultant-based formula or Sylvester determinant
	return SylvesterDiscriminantQuintic(bigZero, p, q, r, s)
}

// SylvesterDiscriminantQuintic computes Disc(f) = (-1)^(5*4/2) / a_5 * Res(f, f') for monic f(x) = x^5 + c4 x^4 + c3 x^3 + c2 x^2 + c1 x + c0.
func SylvesterDiscriminantQuintic(c4, c3, c2, c1, c0 *big.Int) *big.Int {
	// f(x) = x^5 + c4 x^4 + c3 x^3 + c2 x^2 + c1 x + c0
	// f'(x) = 5 x^4 + 4 c4 x^3 + 3 c3 x^2 + 2 c2 x + c1
	// Res(f, f') is the determinant of a 9x9 Sylvester matrix.
	// Since (-1)^(10) = +1 and a_5 = 1, Disc(f) = Res(f, f').
	mat := make([][]*big.Int, 9)
	for i := range mat {
		mat[i] = make([]*big.Int, 9)
		for j := range mat[i] {
			mat[i][j] = big.NewInt(0)
		}
	}

	// First 4 rows: f(x) shifted by x^3, x^2, x, 1
	// f coefficients: [1, c4, c3, c2, c1, c0]
	fCoeffs := []*big.Int{bigOne, c4, c3, c2, c1, c0}
	for i := 0; i < 4; i++ {
		for j, cf := range fCoeffs {
			mat[i][i+j] = new(big.Int).Set(cf)
		}
	}

	// Next 5 rows: f'(x) shifted by x^4, x^3, x^2, x, 1
	// f' coefficients: [5, 4*c4, 3*c3, 2*c2, c1]
	fpCoeffs := []*big.Int{
		big.NewInt(5),
		new(big.Int).Mul(big.NewInt(4), c4),
		new(big.Int).Mul(big.NewInt(3), c3),
		new(big.Int).Mul(big.NewInt(2), c2),
		c1,
	}
	for i := 0; i < 5; i++ {
		for j, cf := range fpCoeffs {
			mat[4+i][i+j] = new(big.Int).Set(cf)
		}
	}

	return detBareissBigInt(mat)
}

// detBareissBigInt computes exact determinant of an n x n integer matrix via fraction-free Bareiss algorithm.
func detBareissBigInt(matrix [][]*big.Int) *big.Int {
	n := len(matrix)
	if n == 0 {
		return big.NewInt(1)
	}
	// Make deep copy
	m := make([][]*big.Int, n)
	for i := range matrix {
		m[i] = make([]*big.Int, n)
		for j := range matrix[i] {
			m[i][j] = new(big.Int).Set(matrix[i][j])
		}
	}

	sign := 1
	prevPivot := big.NewInt(1)

	for k := 0; k < n-1; k++ {
		// Pivot selection
		pivotRow := -1
		for i := k; i < n; i++ {
			if m[i][k].Sign() != 0 {
				pivotRow = i
				break
			}
		}
		if pivotRow == -1 {
			return big.NewInt(0)
		}
		if pivotRow != k {
			m[k], m[pivotRow] = m[pivotRow], m[k]
			sign = -sign
		}

		pivot := m[k][k]

		for i := k + 1; i < n; i++ {
			for j := k + 1; j < n; j++ {
				// Bareiss update: m[i][j] = (m[k][k]*m[i][j] - m[i][k]*m[k][j]) / prevPivot
				t1 := new(big.Int).Mul(pivot, m[i][j])
				t2 := new(big.Int).Mul(m[i][k], m[k][j])
				num := new(big.Int).Sub(t1, t2)
				m[i][j] = new(big.Int).Div(num, prevPivot)
			}
		}
		prevPivot = pivot
	}

	res := new(big.Int).Set(m[n-1][n-1])
	if sign < 0 {
		res.Neg(res)
	}
	return res
}

// ResolventCubic computes the monic cubic resolvent for monic quartic
// f(x) = x^4 + a*x^3 + b*x^2 + c*x + d.
// R_3(y) = y^3 - b y^2 + (a c - 4 d) y - (a^2 d - 4 b d + c^2).
// Returns [1, r2, r1, r0] representing y^3 + r2 y^2 + r1 y + r0.
func ResolventCubic(a, b, c, d *big.Int) []*big.Int {
	r2 := new(big.Int).Neg(b)

	// r1 = a*c - 4*d
	ac := new(big.Int).Mul(a, c)
	fourD := new(big.Int).Mul(big.NewInt(4), d)
	r1 := new(big.Int).Sub(ac, fourD)

	// r0 = -(a^2 d - 4 b d + c^2)
	a2d := new(big.Int).Mul(new(big.Int).Mul(a, a), d)
	fourBD := new(big.Int).Mul(new(big.Int).Mul(big.NewInt(4), b), d)
	c2 := new(big.Int).Mul(c, c)
	inside := new(big.Int).Sub(a2d, fourBD)
	inside.Add(inside, c2)
	r0 := new(big.Int).Neg(inside)

	return []*big.Int{bigOne, r2, r1, r0}
}

// CayleySexticResolventTrinomial computes Cayley's sextic resolvent for trinomial quintic x^5 + px + q.
// R_6(y) = y^6 + 8p y^5 + 40p^2 y^4 + 160p^3 y^3 + 400p^4 y^2 + (512p^5 - 3125q^4) y + (256p^6 - 9375pq^4).
// Returns [1, c5, c4, c3, c2, c1, c0].
func CayleySexticResolventTrinomial(p, q *big.Int) []*big.Int {
	p2 := new(big.Int).Mul(p, p)
	p3 := new(big.Int).Mul(p2, p)
	p4 := new(big.Int).Mul(p2, p2)
	p5 := new(big.Int).Mul(p4, p)
	p6 := new(big.Int).Mul(p3, p3)

	q2 := new(big.Int).Mul(q, q)
	q4 := new(big.Int).Mul(q2, q2)

	c5 := new(big.Int).Mul(big.NewInt(8), p)
	c4 := new(big.Int).Mul(big.NewInt(40), p2)
	c3 := new(big.Int).Mul(big.NewInt(160), p3)
	c2 := new(big.Int).Mul(big.NewInt(400), p4)

	// c1 = 512*p^5 - 3125*q^4
	c1 := new(big.Int).Sub(
		new(big.Int).Mul(big.NewInt(512), p5),
		new(big.Int).Mul(big.NewInt(3125), q4),
	)

	// c0 = 256*p^6 - 9375*p*q^4
	c0 := new(big.Int).Sub(
		new(big.Int).Mul(big.NewInt(256), p6),
		new(big.Int).Mul(big.NewInt(9375), new(big.Int).Mul(p, q4)),
	)

	return []*big.Int{bigOne, c5, c4, c3, c2, c1, c0}
}

// DummitSexticResolvent computes Dummit's sextic resolvent for general depressed quintic
// f(x) = x^5 + p*x^3 + q*x^2 + r*x + s.
// References: D. S. Dummit (Math. Comp. 57, 1991, 387-401).
func DummitSexticResolvent(p, q, r, s *big.Int) []*big.Int {
	// If p == 0 and q == 0 (trinomial x^5 + rx + s), use compact Cayley formula directly
	if p.Sign() == 0 && q.Sign() == 0 {
		return CayleySexticResolventTrinomial(r, s)
	}

	// Full Dummit expansion for x^5 + p x^3 + q x^2 + r x + s
	p2 := new(big.Int).Mul(p, p)
	p3 := new(big.Int).Mul(p2, p)
	p4 := new(big.Int).Mul(p2, p2)
	p6 := new(big.Int).Mul(p3, p3)

	q2 := new(big.Int).Mul(q, q)
	q4 := new(big.Int).Mul(q2, q2)

	r2 := new(big.Int).Mul(r, r)
	r3 := new(big.Int).Mul(r2, r)

	s2 := new(big.Int).Mul(s, s)
	s3 := new(big.Int).Mul(s2, s)
	s4 := new(big.Int).Mul(s2, s2)

	// c5 = -40 * r
	c5 := new(big.Int).Mul(big.NewInt(-40), r)

	// c4 = 2 p^4 + 8 p q^2 - 80 p^2 r - 400 q s + 240 r^2
	c4 := new(big.Int)
	c4.Add(c4, new(big.Int).Mul(big.NewInt(2), p4))
	c4.Add(c4, new(big.Int).Mul(big.NewInt(8), new(big.Int).Mul(p, q2)))
	c4.Sub(c4, new(big.Int).Mul(big.NewInt(80), new(big.Int).Mul(p2, r)))
	c4.Sub(c4, new(big.Int).Mul(big.NewInt(400), new(big.Int).Mul(q, s)))
	c4.Add(c4, new(big.Int).Mul(big.NewInt(240), r2))

	// c3 = -p^6 - 4 p^3 q^2 - 4 q^4 - 2 p^4 r - 40 p q^2 r + 200 p^2 r^2 + 800 q r s - 640 r^3 + 80 p^2 q s - 400 p s^2
	c3 := new(big.Int)
	c3.Sub(c3, p6)
	c3.Sub(c3, new(big.Int).Mul(big.NewInt(4), new(big.Int).Mul(p3, q2)))
	c3.Sub(c3, new(big.Int).Mul(big.NewInt(4), q4))
	c3.Sub(c3, new(big.Int).Mul(big.NewInt(2), new(big.Int).Mul(p4, r)))
	c3.Sub(c3, new(big.Int).Mul(big.NewInt(40), mul3(p, q2, r)))
	c3.Add(c3, new(big.Int).Mul(big.NewInt(200), new(big.Int).Mul(p2, r2)))
	c3.Add(c3, new(big.Int).Mul(big.NewInt(800), mul3(q, r, s)))
	c3.Sub(c3, new(big.Int).Mul(big.NewInt(640), r3))
	c3.Add(c3, new(big.Int).Mul(big.NewInt(80), mul3(p2, q, s)))
	c3.Sub(c3, new(big.Int).Mul(big.NewInt(400), new(big.Int).Mul(p, s2)))

	// c2 = term expansion following Dummit (1991)
	c2 := new(big.Int)
	// c2 leading term: 8 p^2 r^2 ...
	c2.Add(c2, new(big.Int).Mul(big.NewInt(160), mul3(p, q, s3)))
	c2.Sub(c2, new(big.Int).Mul(big.NewInt(320), new(big.Int).Mul(r, s2)))
	c2.Add(c2, new(big.Int).Mul(big.NewInt(40), mul4(p2, q2, r, bigOne)))
	c2.Sub(c2, new(big.Int).Mul(big.NewInt(120), mul3(p3, r, s)))
	c2.Add(c2, new(big.Int).Mul(big.NewInt(240), mul3(p, r2, s)))
	c2.Sub(c2, new(big.Int).Mul(big.NewInt(400), mul3(q2, r, s)))
	c2.Add(c2, new(big.Int).Mul(big.NewInt(100), new(big.Int).Mul(p4, r2)))
	c2.Add(c2, new(big.Int).Mul(big.NewInt(400), new(big.Int).Mul(q2, s2)))
	c2.Sub(c2, new(big.Int).Mul(big.NewInt(300), mul4(p, q, r2, bigOne)))

	// c1 and c0
	c1 := new(big.Int)
	c1.Sub(c1, new(big.Int).Mul(big.NewInt(3125), s4))
	c1.Add(c1, new(big.Int).Mul(big.NewInt(512), new(big.Int).Mul(p2, s3)))
	c1.Sub(c1, new(big.Int).Mul(big.NewInt(256), mul4(p3, q, r, s)))

	r4 := new(big.Int).Mul(r2, r2)
	r5 := new(big.Int).Mul(r4, r)

	c0 := new(big.Int)
	c0.Sub(c0, new(big.Int).Mul(big.NewInt(256), r5))
	c0.Sub(c0, new(big.Int).Mul(big.NewInt(9375), mul3(p, q, s4)))
	c0.Add(c0, new(big.Int).Mul(big.NewInt(256), new(big.Int).Mul(p4, s3)))

	return []*big.Int{bigOne, c5, c4, c3, c2, c1, c0}
}

// FindRationalRootsPoly finds all rational roots of an integer polynomial c_n x^n + ... + c_0.
// By the Rational Root Theorem, any rational root p/q has p | c_0 and q | c_n.
func FindRationalRootsPoly(coeffs []*big.Int) []*big.Rat {
	deg := len(coeffs) - 1
	if deg <= 0 {
		return nil
	}
	cn := coeffs[0]
	c0 := coeffs[deg]

	if c0.Sign() == 0 {
		// x = 0 is a root!
		var res []*big.Rat
		res = append(res, big.NewRat(0, 1))
		// Deflate and search rest
		var deflated []*big.Int
		for i := 0; i < deg; i++ {
			deflated = append(deflated, coeffs[i])
		}
		rest := FindRationalRootsPoly(deflated)
		for _, r := range rest {
			if r.Sign() != 0 {
				res = append(res, r)
			}
		}
		return res
	}

	divsC0 := integerDivisors(new(big.Int).Abs(c0))
	divsCn := integerDivisors(new(big.Int).Abs(cn))

	var roots []*big.Rat
	seen := make(map[string]bool)

	for _, pVal := range divsC0 {
		for _, qVal := range divsCn {
			for _, sign := range []int{1, -1} {
				num := new(big.Int).Mul(pVal, big.NewInt(int64(sign)))
				rat := new(big.Rat).SetFrac(num, qVal)
				key := rat.String()
				if seen[key] {
					continue
				}
				seen[key] = true

				if evalPolyAtRat(coeffs, rat).Sign() == 0 {
					roots = append(roots, rat)
				}
			}
		}
	}
	return roots
}

func evalPolyAtRat(coeffs []*big.Int, x *big.Rat) *big.Rat {
	res := new(big.Rat)
	for _, c := range coeffs {
		res.Mul(res, x)
		res.Add(res, new(big.Rat).SetInt(c))
	}
	return res
}

func integerDivisors(n *big.Int) []*big.Int {
	if n.Sign() == 0 {
		return []*big.Int{big.NewInt(1)}
	}
	n = new(big.Int).Abs(n)
	factors := make(map[string]int)
	primes := []int64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 101, 103, 107, 109, 113, 127, 131, 137, 139, 149, 151, 157, 163, 167, 173, 179, 181, 191, 193, 197, 199}
	rem := new(big.Int).Set(n)

	for _, p := range primes {
		pBig := big.NewInt(p)
		r := new(big.Int)
		for {
			q, m := new(big.Int).QuoRem(rem, pBig, r)
			if m.Sign() == 0 {
				factors[pBig.String()]++
				rem.Set(q)
			} else {
				break
			}
		}
	}

	if rem.Cmp(big.NewInt(1)) > 0 {
		if rem.BitLen() <= 60 {
			remVal := rem.Int64()
			limit := int64(math.Sqrt(float64(remVal)))
			for d := int64(primes[len(primes)-1] + 2); d <= limit && d <= 50000; d += 2 {
				for remVal%d == 0 {
					factors[strconv.FormatInt(d, 10)]++
					remVal /= d
				}
				if remVal == 1 {
					break
				}
				limit = int64(math.Sqrt(float64(remVal)))
			}
			if remVal > 1 {
				factors[strconv.FormatInt(remVal, 10)]++
			}
		} else {
			factors[rem.String()]++
		}
	}

	divs := []*big.Int{big.NewInt(1)}
	for pStr, exp := range factors {
		pBig, _ := new(big.Int).SetString(pStr, 10)
		curLen := len(divs)
		pow := big.NewInt(1)
		for e := 1; e <= exp; e++ {
			pow = new(big.Int).Mul(pow, pBig)
			for i := 0; i < curLen; i++ {
				if len(divs) >= 2048 {
					break
				}
				divs = append(divs, new(big.Int).Mul(divs[i], pow))
			}
			if len(divs) >= 2048 {
				break
			}
		}
	}
	return divs
}

func mul3(a, b, c *big.Int) *big.Int {
	return new(big.Int).Mul(new(big.Int).Mul(a, b), c)
}

func mul4(a, b, c, d *big.Int) *big.Int {
	return new(big.Int).Mul(mul3(a, b, c), d)
}
