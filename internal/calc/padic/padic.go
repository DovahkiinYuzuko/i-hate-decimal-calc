package padic

import (
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// PadicNumber represents a truncated p-adic number in Q_p:
// x = p^Valuation * Unit (mod p^Precision)
type PadicNumber struct {
	Prime     *big.Int
	Valuation int
	Unit      *big.Int
	Precision int
	IsZero    bool
}

// extGCD computes the Extended Euclidean Algorithm on a and b,
// returning gcd, x, and y such that a*x + b*y = gcd.
func extGCD(a, b *big.Int) (*big.Int, *big.Int, *big.Int) {
	if b.Sign() == 0 {
		return new(big.Int).Set(a), big.NewInt(1), big.NewInt(0)
	}
	g, x1, y1 := extGCD(b, new(big.Int).Mod(a, b))
	q := new(big.Int).Div(a, b)
	x := new(big.Int).Set(y1)
	y := new(big.Int).Sub(x1, new(big.Int).Mul(q, y1))
	return g, x, y
}

// invMod computes the modular inverse of a modulo m: a * x = 1 (mod m).
func invMod(a, m *big.Int) (*big.Int, error) {
	modA := new(big.Int).Mod(a, m)
	if modA.Sign() < 0 {
		modA.Add(modA, m)
	}
	g, x, _ := extGCD(modA, m)
	if g.Cmp(big.NewInt(1)) != 0 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_non_invertible"))
	}
	x.Mod(x, m)
	if x.Sign() < 0 {
		x.Add(x, m)
	}
	return x, nil
}

// isDeterministicPrime checks whether p is prime deterministically for uint64
// (Miller-Rabin bases {2, 325, 9375, 28178, 450775, 9780504, 1795265022})
// and uses Baillie-PSW (big.Int.ProbablyPrime(0)) for larger integers.
func isDeterministicPrime(p *big.Int) bool {
	if p == nil || p.Sign() <= 0 {
		return false
	}
	if p.IsUint64() {
		n := p.Uint64()
		if n < 2 {
			return false
		}
		smallPrimes := [...]uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}
		for _, sp := range smallPrimes {
			if n == sp {
				return true
			}
			if n%sp == 0 {
				return false
			}
		}
		d := n - 1
		s := bits.TrailingZeros64(d)
		d >>= s
		bases := [...]uint64{2, 325, 9375, 28178, 450775, 9780504, 1795265022}
		for _, a := range bases {
			if a%n == 0 {
				continue
			}
			if !millerRabinWitness64(n, d, s, a) {
				return false
			}
		}
		return true
	}
	return p.ProbablyPrime(0)
}

func mulMod64(a, b, mod uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	_, rem := bits.Div64(hi, lo, mod)
	return rem
}

func powMod64(base, exp, mod uint64) uint64 {
	res := uint64(1)
	b := base % mod
	for exp > 0 {
		if exp&1 == 1 {
			res = mulMod64(res, b, mod)
		}
		b = mulMod64(b, b, mod)
		exp >>= 1
	}
	return res
}

func millerRabinWitness64(n, d uint64, s int, a uint64) bool {
	x := powMod64(a, d, n)
	if x == 1 || x == n-1 {
		return true
	}
	for r := 1; r < s; r++ {
		x = mulMod64(x, x, n)
		if x == n-1 {
			return true
		}
	}
	return false
}

// Valuation computes the p-adic valuation v_p(x) of a rational number x.
// Returns an error if rat == 0 or prime is not a prime number.
func Valuation(rat *big.Rat, prime *big.Int) (int, error) {
	if rat.Sign() == 0 {
		return 0, fmt.Errorf("%s", i18n.T("padic.err_zero_valuation"))
	}
	if prime.Cmp(big.NewInt(2)) < 0 {
		return 0, fmt.Errorf("%s", i18n.T("padic.err_prime_less_than_2"))
	}
	if !isDeterministicPrime(prime) {
		return 0, fmt.Errorf("%s", i18n.T("padic.err_prime_must_be_prime", prime.String()))
	}

	num := new(big.Int).Abs(rat.Num())
	denom := new(big.Int).Abs(rat.Denom())

	vNum := 0
	rem := new(big.Int)
	for {
		rem.Mod(num, prime)
		if rem.Sign() == 0 {
			vNum++
			num.Div(num, prime)
		} else {
			break
		}
	}

	vDenom := 0
	for {
		rem.Mod(denom, prime)
		if rem.Sign() == 0 {
			vDenom++
			denom.Div(denom, prime)
		} else {
			break
		}
	}

	return vNum - vDenom, nil
}

// Norm computes the p-adic norm |x|_p = p^(-v_p(x)) of a rational number x.
// If x == 0, returns 0.
func Norm(rat *big.Rat, prime *big.Int) (*big.Rat, error) {
	if rat.Sign() == 0 {
		return new(big.Rat), nil
	}
	v, err := Valuation(rat, prime)
	if err != nil {
		return nil, err
	}

	res := new(big.Rat)
	if v >= 0 {
		pow := new(big.Int).Exp(prime, big.NewInt(int64(v)), nil)
		res.SetFrac(big.NewInt(1), pow)
	} else {
		pow := new(big.Int).Exp(prime, big.NewInt(int64(-v)), nil)
		res.SetInt(pow)
	}
	return res, nil
}

// NewPadicFromRat constructs a PadicNumber from a rational number x, prime p, and precision k.
func NewPadicFromRat(rat *big.Rat, prime *big.Int, prec int) (*PadicNumber, error) {
	if prime.Cmp(big.NewInt(2)) < 0 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_prime_less_than_2"))
	}
	if !isDeterministicPrime(prime) {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_prime_must_be_prime", prime.String()))
	}
	if prec <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_prec_positive"))
	}

	pCopy := new(big.Int).Set(prime)
	if rat.Sign() == 0 {
		return &PadicNumber{
			Prime:     pCopy,
			Valuation: 0,
			Unit:      big.NewInt(0),
			Precision: prec,
			IsZero:    true,
		}, nil
	}

	v, err := Valuation(rat, prime)
	if err != nil {
		return nil, err
	}

	// a' and b' coprime to p
	aPrime := new(big.Int).Set(rat.Num())
	bPrime := new(big.Int).Set(rat.Denom())

	rem := new(big.Int)
	for {
		rem.Mod(aPrime, prime)
		if rem.Sign() == 0 {
			aPrime.Div(aPrime, prime)
		} else {
			break
		}
	}
	for {
		rem.Mod(bPrime, prime)
		if rem.Sign() == 0 {
			bPrime.Div(bPrime, prime)
		} else {
			break
		}
	}

	pk := new(big.Int).Exp(prime, big.NewInt(int64(prec)), nil)
	invB, err := invMod(bPrime, pk)
	if err != nil {
		return nil, err
	}

	unit := new(big.Int).Mul(aPrime, invB)
	unit.Mod(unit, pk)
	if unit.Sign() < 0 {
		unit.Add(unit, pk)
	}

	return &PadicNumber{
		Prime:     pCopy,
		Valuation: v,
		Unit:      unit,
		Precision: prec,
		IsZero:    false,
	}, nil
}

// Clone returns a deep copy of a PadicNumber.
func (a *PadicNumber) Clone() *PadicNumber {
	return &PadicNumber{
		Prime:     new(big.Int).Set(a.Prime),
		Valuation: a.Valuation,
		Unit:      new(big.Int).Set(a.Unit),
		Precision: a.Precision,
		IsZero:    a.IsZero,
	}
}

func absPrec(a *PadicNumber) int {
	if a.IsZero {
		return a.Precision
	}
	return a.Valuation + a.Precision
}

func newPadicZero(p *big.Int, absN int) *PadicNumber {
	return &PadicNumber{
		Prime:     new(big.Int).Set(p),
		Valuation: 0,
		Unit:      big.NewInt(0),
		Precision: absN,
		IsZero:    true,
	}
}

func truncatePadicToAbs(a *PadicNumber, absN int) *PadicNumber {
	if a.IsZero {
		if a.Precision < absN {
			return a.Clone()
		}
		return newPadicZero(a.Prime, absN)
	}
	if absN <= a.Valuation {
		return newPadicZero(a.Prime, absN)
	}
	rel := absN - a.Valuation
	if rel >= a.Precision {
		return a.Clone()
	}
	pk := new(big.Int).Exp(a.Prime, big.NewInt(int64(rel)), nil)
	u := new(big.Int).Mod(a.Unit, pk)
	if u.Sign() == 0 {
		return newPadicZero(a.Prime, absN)
	}
	return &PadicNumber{
		Prime:     new(big.Int).Set(a.Prime),
		Valuation: a.Valuation,
		Unit:      u,
		Precision: rel,
		IsZero:    false,
	}
}

// Add computes a + b in Q_p with strict absolute precision tracking (N = v + r).
func Add(a, b *PadicNumber) (*PadicNumber, error) {
	if a.Prime.Cmp(b.Prime) != 0 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_prime_mismatch", a.Prime.String(), b.Prime.String()))
	}

	absRes := absPrec(a)
	if bAbs := absPrec(b); bAbs < absRes {
		absRes = bAbs
	}

	if a.IsZero && b.IsZero {
		return newPadicZero(a.Prime, absRes), nil
	}
	if a.IsZero {
		return truncatePadicToAbs(b, absRes), nil
	}
	if b.IsZero {
		return truncatePadicToAbs(a, absRes), nil
	}

	vMin := a.Valuation
	if b.Valuation < vMin {
		vMin = b.Valuation
	}

	relWork := absRes - vMin
	if relWork <= 0 {
		return newPadicZero(a.Prime, absRes), nil
	}

	p := a.Prime
	pk := new(big.Int).Exp(p, big.NewInt(int64(relWork)), nil)

	shiftTerm := func(x *PadicNumber) *big.Int {
		shift := x.Valuation - vMin
		if shift >= relWork {
			return big.NewInt(0)
		}
		powShift := new(big.Int).Exp(p, big.NewInt(int64(shift)), nil)
		term := new(big.Int).Mul(x.Unit, powShift)
		term.Mod(term, pk)
		return term
	}

	sum := new(big.Int).Add(shiftTerm(a), shiftTerm(b))
	sum.Mod(sum, pk)

	if sum.Sign() == 0 {
		return newPadicZero(p, absRes), nil
	}

	// Factor out powers of p from sum and update relative precision rNew = absRes - vNew.
	rem := new(big.Int)
	extraV := 0
	for {
		rem.Mod(sum, p)
		if rem.Sign() == 0 && sum.Sign() != 0 {
			extraV++
			sum.Div(sum, p)
		} else {
			break
		}
	}

	vNew := vMin + extraV
	rNew := absRes - vNew
	if rNew <= 0 {
		return newPadicZero(p, absRes), nil
	}

	return &PadicNumber{
		Prime:     new(big.Int).Set(p),
		Valuation: vNew,
		Unit:      sum,
		Precision: rNew,
		IsZero:    false,
	}, nil
}

// Sub computes a - b in Q_p.
func Sub(a, b *PadicNumber) (*PadicNumber, error) {
	if a.Prime.Cmp(b.Prime) != 0 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_prime_mismatch", a.Prime.String(), b.Prime.String()))
	}
	if b.IsZero {
		return Add(a, b)
	}

	pk := new(big.Int).Exp(b.Prime, big.NewInt(int64(b.Precision)), nil)
	negUnit := new(big.Int).Sub(pk, new(big.Int).Mod(b.Unit, pk))
	negUnit.Mod(negUnit, pk)

	negB := &PadicNumber{
		Prime:     new(big.Int).Set(b.Prime),
		Valuation: b.Valuation,
		Unit:      negUnit,
		Precision: b.Precision,
		IsZero:    false,
	}

	return Add(a, negB)
}

// Mul computes a * b in Q_p with graded precision tracking.
func Mul(a, b *PadicNumber) (*PadicNumber, error) {
	if a.Prime.Cmp(b.Prime) != 0 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_prime_mismatch", a.Prime.String(), b.Prime.String()))
	}
	if a.IsZero && b.IsZero {
		return newPadicZero(a.Prime, a.Precision+b.Precision), nil
	}
	if a.IsZero {
		return newPadicZero(a.Prime, a.Precision+b.Valuation), nil
	}
	if b.IsZero {
		return newPadicZero(a.Prime, a.Valuation+b.Precision), nil
	}

	minPrec := a.Precision
	if b.Precision < minPrec {
		minPrec = b.Precision
	}

	pk := new(big.Int).Exp(a.Prime, big.NewInt(int64(minPrec)), nil)
	u := new(big.Int).Mul(a.Unit, b.Unit)
	u.Mod(u, pk)
	if u.Sign() == 0 {
		return newPadicZero(a.Prime, a.Valuation+b.Valuation+minPrec), nil
	}

	return &PadicNumber{
		Prime:     new(big.Int).Set(a.Prime),
		Valuation: a.Valuation + b.Valuation,
		Unit:      u,
		Precision: minPrec,
		IsZero:    false,
	}, nil
}

// PowInt computes a^k in Q_p for any integer exponent k.
func PowInt(a *PadicNumber, k *big.Int) (*PadicNumber, error) {
	if k == nil {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_integer", "PowInt", "<nil>"))
	}
	if k.Sign() == 0 {
		prec := a.Precision
		if prec <= 0 {
			prec = 1
		}
		return &PadicNumber{
			Prime:     new(big.Int).Set(a.Prime),
			Valuation: 0,
			Unit:      big.NewInt(1),
			Precision: prec,
			IsZero:    false,
		}, nil
	}
	if k.Sign() < 0 {
		invA, err := Inv(a)
		if err != nil {
			return nil, err
		}
		posK := new(big.Int).Neg(k)
		return PowInt(invA, posK)
	}
	if a.IsZero {
		if !k.IsInt64() {
			return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_integer", "PowInt", k.String()))
		}
		kVal := k.Int64()
		if a.Precision != 0 {
			absPrec := int64(a.Precision)
			if absPrec < 0 {
				absPrec = -absPrec
			}
			if kVal > int64(math.MaxInt)/absPrec {
				return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_integer", "PowInt", k.String()))
			}
		}
		return newPadicZero(a.Prime, a.Precision*int(kVal)), nil
	}
	if !k.IsInt64() {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_integer", "PowInt", k.String()))
	}

	res := &PadicNumber{
		Prime:     new(big.Int).Set(a.Prime),
		Valuation: 0,
		Unit:      big.NewInt(1),
		Precision: a.Precision,
		IsZero:    false,
	}
	base := a.Clone()
	exp := k.Int64()
	for exp > 0 {
		if exp&1 == 1 {
			var err error
			res, err = Mul(res, base)
			if err != nil {
				return nil, err
			}
		}
		exp >>= 1
		if exp > 0 {
			var err error
			base, err = Mul(base, base)
			if err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

// Inv computes the multiplicative inverse 1 / a in Q_p.
func Inv(a *PadicNumber) (*PadicNumber, error) {
	if a.IsZero {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_zero_division"))
	}

	pk := new(big.Int).Exp(a.Prime, big.NewInt(int64(a.Precision)), nil)
	invU, err := invMod(a.Unit, pk)
	if err != nil {
		return nil, err
	}

	return &PadicNumber{
		Prime:     new(big.Int).Set(a.Prime),
		Valuation: -a.Valuation,
		Unit:      invU,
		Precision: a.Precision,
		IsZero:    false,
	}, nil
}

// Div computes a / b in Q_p.
func Div(a, b *PadicNumber) (*PadicNumber, error) {
	if b.IsZero {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_zero_division"))
	}
	invB, err := Inv(b)
	if err != nil {
		return nil, err
	}
	return Mul(a, invB)
}

// ExpansionDigits returns the p-adic series expansion digits d_0, d_1, ..., d_{k-1}
// such that Unit = sum_{i=0}^{k-1} d_i * p^i (mod p^Precision), with 0 <= d_i < p.
func (a *PadicNumber) ExpansionDigits() []*big.Int {
	if a.IsZero {
		return []*big.Int{big.NewInt(0)}
	}

	digits := make([]*big.Int, 0, a.Precision)
	curr := new(big.Int).Set(a.Unit)
	p := a.Prime

	for i := 0; i < a.Precision; i++ {
		d := new(big.Int).Mod(curr, p)
		digits = append(digits, d)
		curr.Sub(curr, d)
		curr.Div(curr, p)
	}

	return digits
}

// ExpansionString formats the p-adic number as a power series string.
// Example: "1 + 2*5 + 3*5^2 + O(5^3)" or "5^(-1) * (1 + 2*5 + O(5^2))".
func ExpansionString(a *PadicNumber) string {
	if a.IsZero {
		if a.Precision < 0 {
			return fmt.Sprintf("0 + O(%s^(%d))", a.Prime.String(), a.Precision)
		}
		return fmt.Sprintf("0 + O(%s^%d)", a.Prime.String(), a.Precision)
	}

	digits := a.ExpansionDigits()
	terms := make([]string, 0, len(digits))

	pStr := a.Prime.String()
	for i, d := range digits {
		if d.Sign() == 0 {
			continue
		}
		switch i {
		case 0:
			terms = append(terms, d.String())
		case 1:
			if d.Cmp(big.NewInt(1)) == 0 {
				terms = append(terms, pStr)
			} else {
				terms = append(terms, fmt.Sprintf("%s*%s", d.String(), pStr))
			}
		default:
			if d.Cmp(big.NewInt(1)) == 0 {
				terms = append(terms, fmt.Sprintf("%s^%d", pStr, i))
			} else {
				terms = append(terms, fmt.Sprintf("%s*%s^%d", d.String(), pStr, i))
			}
		}

	}

	seriesPart := "0"
	if len(terms) > 0 {
		seriesPart = strings.Join(terms, " + ")
	}

	if a.Valuation == 0 {
		return fmt.Sprintf("%s + O(%s^%d)", seriesPart, pStr, a.Precision)
	} else if a.Valuation > 0 {
		return fmt.Sprintf("%s^%d * (%s) + O(%s^%d)", pStr, a.Valuation, seriesPart, pStr, a.Valuation+a.Precision)
	} else {
		return fmt.Sprintf("%s^(%d) * (%s) + O(%s^%d)", pStr, a.Valuation, seriesPart, pStr, a.Valuation+a.Precision)
	}
}

// IsUltrametric strictly verifies the ultrametric inequality |a + b|_p <= max(|a|_p, |b|_p)
// which is equivalent to v_p(a + b) >= min(v_p(a), v_p(b)).
func IsUltrametric(a, b *PadicNumber) bool {
	sum, err := Add(a, b)
	if err != nil {
		return false
	}
	if sum.IsZero {
		// Valuation of zero is +infinity, which is >= any finite valuation.
		return true
	}
	if a.IsZero {
		return sum.Valuation == b.Valuation
	}
	if b.IsZero {
		return sum.Valuation == a.Valuation
	}

	minV := a.Valuation
	if b.Valuation < minV {
		minV = b.Valuation
	}

	return sum.Valuation >= minV
}
