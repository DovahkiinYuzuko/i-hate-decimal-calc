package calc

import (
	"math/big"
)

const (
	nttP1 uint64 = 998244353  // 119 * 2^23 + 1
	nttP2 uint64 = 1004535809 // 479 * 2^21 + 1
	nttP3 uint64 = 469762049  // 7 * 2^26 + 1

	nttG1 uint64 = 3
	nttG2 uint64 = 3
	nttG3 uint64 = 3

	nttBase uint64 = 1000000000 // 10^9 radix
)

var (
	nttInvP1P2 uint64
	nttInvP1P3 uint64
	nttInvP2P3 uint64

	nttP1Big    *big.Int
	nttP1P2Big  *big.Int
	nttBaseBig  *big.Int
)

func init() {
	nttInvP1P2 = nttModInv(nttP1%nttP2, nttP2)
	nttInvP1P3 = nttModInv(nttP1%nttP3, nttP3)
	nttInvP2P3 = nttModInv(nttP2%nttP3, nttP3)

	nttP1Big = new(big.Int).SetUint64(nttP1)
	p2Big := new(big.Int).SetUint64(nttP2)
	nttP1P2Big = new(big.Int).Mul(nttP1Big, p2Big)
	nttBaseBig = new(big.Int).SetUint64(nttBase)
}

func nttModPow(base, exp, mod uint64) uint64 {
	res := uint64(1)
	base %= mod
	for exp > 0 {
		if exp&1 == 1 {
			res = (res * base) % mod
		}
		base = (base * base) % mod
		exp >>= 1
	}
	return res
}

func nttModInv(n, mod uint64) uint64 {
	return nttModPow(n, mod-2, mod)
}

// forwardNTT computes in-place radix-2 Cooley-Tukey NTT.
func forwardNTT(a []uint64, p, g uint64) {
	nttInternal(a, p, g, false)
}

// inverseNTT computes in-place inverse NTT.
func inverseNTT(a []uint64, p, g uint64) {
	nttInternal(a, p, g, true)
}

func nttInternal(a []uint64, p, g uint64, invert bool) {
	n := len(a)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			a[i], a[j] = a[j], a[i]
		}
	}

	for length := 2; length <= n; length <<= 1 {
		wlen := nttModPow(g, (p-1)/uint64(length), p)
		if invert {
			wlen = nttModInv(wlen, p)
		}
		half := length >> 1
		for i := 0; i < n; i += length {
			w := uint64(1)
			for j := 0; j < half; j++ {
				u := a[i+j]
				v := (a[i+j+half] * w) % p
				if u+v >= p {
					a[i+j] = u + v - p
				} else {
					a[i+j] = u + v
				}
				if u < v {
					a[i+j+half] = u + p - v
				} else {
					a[i+j+half] = u - v
				}
				w = (w * wlen) % p
			}
		}
	}

	if invert {
		nInv := nttModInv(uint64(n), p)
		for i := 0; i < n; i++ {
			a[i] = (a[i] * nInv) % p
		}
	}
}

// garnerReconstruct3 recovers x in [0, P1*P2*P3) using Garner's algorithm.
func garnerReconstruct3(r1, r2, r3 uint64) *big.Int {
	v1 := r1

	// v2 = (r2 - v1) * inv(P1, P2) mod P2
	var diff2 uint64
	if r2 >= v1%nttP2 {
		diff2 = r2 - (v1 % nttP2)
	} else {
		diff2 = r2 + nttP2 - (v1 % nttP2)
	}
	v2 := (diff2 * nttInvP1P2) % nttP2

	// v3 = ((r3 - v1)*inv(P1, P3) - v2) * inv(P2, P3) mod P3
	var diff3 uint64
	if r3 >= v1%nttP3 {
		diff3 = r3 - (v1 % nttP3)
	} else {
		diff3 = r3 + nttP3 - (v1 % nttP3)
	}
	t3 := (diff3 * nttInvP1P3) % nttP3
	var diff3v2 uint64
	if t3 >= v2%nttP3 {
		diff3v2 = t3 - (v2 % nttP3)
	} else {
		diff3v2 = t3 + nttP3 - (v2 % nttP3)
	}
	v3 := (diff3v2 * nttInvP2P3) % nttP3

	// x = v1 + v2 * P1 + v3 * (P1 * P2)
	x := new(big.Int).SetUint64(v1)
	if v2 > 0 {
		term2 := new(big.Int).Mul(nttP1Big, new(big.Int).SetUint64(v2))
		x.Add(x, term2)
	}
	if v3 > 0 {
		term3 := new(big.Int).Mul(nttP1P2Big, new(big.Int).SetUint64(v3))
		x.Add(x, term3)
	}
	return x
}

// ConvolveNTT computes polynomial multiplication of two coefficient slices (base 10^9).
func ConvolveNTT(a, b []uint64) []*big.Int {
	degA := len(a)
	degB := len(b)
	if degA == 0 || degB == 0 {
		return nil
	}

	needed := degA + degB - 1
	n := 1
	for n < needed {
		n <<= 1
	}

	// 1. Modulo P1
	fa1 := make([]uint64, n)
	fb1 := make([]uint64, n)
	for i := 0; i < degA; i++ {
		fa1[i] = a[i] % nttP1
	}
	for i := 0; i < degB; i++ {
		fb1[i] = b[i] % nttP1
	}
	forwardNTT(fa1, nttP1, nttG1)
	forwardNTT(fb1, nttP1, nttG1)
	for i := 0; i < n; i++ {
		fa1[i] = (fa1[i] * fb1[i]) % nttP1
	}
	inverseNTT(fa1, nttP1, nttG1)

	// 2. Modulo P2
	fa2 := make([]uint64, n)
	fb2 := make([]uint64, n)
	for i := 0; i < degA; i++ {
		fa2[i] = a[i] % nttP2
	}
	for i := 0; i < degB; i++ {
		fb2[i] = b[i] % nttP2
	}
	forwardNTT(fa2, nttP2, nttG2)
	forwardNTT(fb2, nttP2, nttG2)
	for i := 0; i < n; i++ {
		fa2[i] = (fa2[i] * fb2[i]) % nttP2
	}
	inverseNTT(fa2, nttP2, nttG2)

	// 3. Modulo P3
	fa3 := make([]uint64, n)
	fb3 := make([]uint64, n)
	for i := 0; i < degA; i++ {
		fa3[i] = a[i] % nttP3
	}
	for i := 0; i < degB; i++ {
		fb3[i] = b[i] % nttP3
	}
	forwardNTT(fa3, nttP3, nttG3)
	forwardNTT(fb3, nttP3, nttG3)
	for i := 0; i < n; i++ {
		fa3[i] = (fa3[i] * fb3[i]) % nttP3
	}
	inverseNTT(fa3, nttP3, nttG3)

	// Garner reconstruction
	res := make([]*big.Int, needed)
	for i := 0; i < needed; i++ {
		res[i] = garnerReconstruct3(fa1[i], fa2[i], fa3[i])
	}
	return res
}

func toDigitsNTT(n *big.Int) []uint64 {
	if n.Sign() == 0 {
		return []uint64{0}
	}
	temp := new(big.Int).Abs(n)
	rem := new(big.Int)
	var digits []uint64
	for temp.Sign() > 0 {
		temp.QuoRem(temp, nttBaseBig, rem)
		digits = append(digits, rem.Uint64())
	}
	return digits
}

// NTTMultiply computes the exact product a * b using 3-prime NTT and Garner's algorithm.
func NTTMultiply(a, b *big.Int) *big.Int {
	if a.Sign() == 0 || b.Sign() == 0 {
		return big.NewInt(0)
	}

	// For small numbers, standard multiplication is fastest
	if a.BitLen() < 1024 && b.BitLen() < 1024 {
		return new(big.Int).Mul(a, b)
	}

	digitsA := toDigitsNTT(a)
	digitsB := toDigitsNTT(b)

	polyProduct := ConvolveNTT(digitsA, digitsB)

	// Carry propagation
	carry := new(big.Int)
	var normalizedDigits []*big.Int
	rem := new(big.Int)

	for i := 0; i < len(polyProduct) || carry.Sign() > 0; i++ {
		sum := new(big.Int).Set(carry)
		if i < len(polyProduct) {
			sum.Add(sum, polyProduct[i])
		}
		sum.QuoRem(sum, nttBaseBig, rem)
		carry.Set(sum)
		normalizedDigits = append(normalizedDigits, new(big.Int).Set(rem))
	}

	// Reconstruct big.Int from base 10^9 digits using balanced divide-and-conquer
	result := reconstructBigIntNTT(normalizedDigits)

	// Apply sign
	if (a.Sign() < 0 && b.Sign() > 0) || (a.Sign() > 0 && b.Sign() < 0) {
		result.Neg(result)
	}
	return result
}

// reconstructBigIntNTT converts base 10^9 digits back to *big.Int via balanced divide-and-conquer.
func reconstructBigIntNTT(digits []*big.Int) *big.Int {
	if len(digits) == 0 {
		return big.NewInt(0)
	}
	if len(digits) == 1 {
		return digits[0]
	}
	return reconstructRangeNTT(digits, 0, len(digits))
}

func reconstructRangeNTT(digits []*big.Int, low, high int) *big.Int {
	length := high - low
	if length == 1 {
		return digits[low]
	}
	mid := low + length/2
	lowPart := reconstructRangeNTT(digits, low, mid)
	highPart := reconstructRangeNTT(digits, mid, high)

	shiftExp := big.NewInt(int64(mid - low))
	scale := new(big.Int).Exp(nttBaseBig, shiftExp, nil)

	term := new(big.Int).Mul(highPart, scale)
	return new(big.Int).Add(term, lowPart)
}
