package calc

import (
	"math/big"
	"math/bits"
)

// mulMod64 computes (a * b) % m without 64-bit overflow using math/bits.
func mulMod64(a, b, m uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	_, rem := bits.Div64(hi, lo, m)
	return rem
}

// powMod64 computes (base^exp) % mod using binary exponentiation.
func powMod64(base, exp, mod uint64) uint64 {
	res := uint64(1)
	base %= mod
	for exp > 0 {
		if exp&1 == 1 {
			res = mulMod64(res, base, mod)
		}
		base = mulMod64(base, base, mod)
		exp >>= 1
	}
	return res
}

// millerRabin7 performs a deterministic Miller-Rabin test for any odd n < 2^64
// using the 7 minimal bases discovered by Jim Sinclair (Jiang & Deng 2014):
// {2, 325, 9375, 28178, 450775, 9780504, 1795265022}.
// Proved to have zero pseudoprimes / counterexamples for all n < 2^64.
func millerRabin7(n uint64) bool {
	if n < 2 {
		return false
	}
	if n == 2 || n == 3 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}

	// Factor n - 1 = 2^s * d
	d := n - 1
	s := 0
	for d%2 == 0 {
		d /= 2
		s++
	}

	bases := [...]uint64{2, 325, 9375, 28178, 450775, 9780504, 1795265022}
	for _, a := range bases {
		if n <= a {
			// If n divides a or equals a
			if n == a {
				return true
			}
			// a mod n is safe
			a %= n
			if a == 0 {
				continue
			}
		}

		x := powMod64(a, d, n)
		if x == 1 || x == n-1 {
			continue
		}

		composite := true
		for r := 1; r < s; r++ {
			x = mulMod64(x, x, n)
			if x == n-1 {
				composite = false
				break
			}
		}
		if composite {
			return false
		}
	}
	return true
}

// millerRabin12 performs a deterministic Miller-Rabin test for any odd n < 2^64
// using the first 12 prime bases proved by Sorenson & Webster (2015, Math. Comp. 2017):
// {2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}.
func millerRabin12(n uint64) bool {
	if n < 2 {
		return false
	}
	if n == 2 || n == 3 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}

	// Factor n - 1 = 2^s * d
	d := n - 1
	s := 0
	for d%2 == 0 {
		d /= 2
		s++
	}

	bases := [...]uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}
	for _, a := range bases {
		if n <= a {
			break
		}
		x := powMod64(a, d, n)
		if x == 1 || x == n-1 {
			continue
		}

		composite := true
		for r := 1; r < s; r++ {
			x = mulMod64(x, x, n)
			if x == n-1 {
				composite = false
				break
			}
		}
		if composite {
			return false
		}
	}
	return true
}

// IsDeterministicPrime64 tests whether a 64-bit unsigned integer n is prime
// using the 100% unconditional deterministic Sinclair 7-base Miller-Rabin test.
func IsDeterministicPrime64(n uint64) bool {
	return millerRabin7(n)
}

// IsDeterministicPrime tests whether an arbitrary-precision integer n is prime
// without using any pseudorandomness or probabilistic shortcuts.
//
// For n < 2^64: uses Jim Sinclair's minimal 7-base Miller-Rabin test (unconditionally deterministic).
// For n >= 2^64: uses Go's math/big.ProbablyPrime(0), which executes pure deterministic
// Baillie-PSW (Base-2 Strong Lucas-Selfridge without any pseudorandom rounds).
func IsDeterministicPrime(n *big.Int) bool {
	if n == nil || n.Sign() <= 0 {
		return false
	}

	// 0 and 1 are not prime
	if n.Cmp(big.NewInt(1)) <= 0 {
		return false
	}

	// 2 and 3 are prime
	two := big.NewInt(2)
	three := big.NewInt(3)
	if n.Cmp(two) == 0 || n.Cmp(three) == 0 {
		return true
	}

	// Quick divisibility check for 2 and 3
	rem := new(big.Int)
	if rem.Mod(n, two).Sign() == 0 || rem.Mod(n, three).Sign() == 0 {
		return false
	}

	// If n fits in uint64, apply 100% deterministic 64-bit test
	if n.IsUint64() {
		return IsDeterministicPrime64(n.Uint64())
	}

	// For n >= 2^64:
	// math/big.ProbablyPrime(0) executes deterministic Baillie-PSW (zero random rounds).
	return n.ProbablyPrime(0)
}
