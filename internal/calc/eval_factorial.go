package calc

import (
	"math/big"
	"math/bits"
)

// FactorialLegendre computes n! using Legendre's prime factorization formula
// and binary splitting product tree multiplication.
// Factors of 2 are evaluated via bit shifting (Lsh).
func FactorialLegendre(n int64) *big.Int {
	if n < 0 {
		return nil
	}
	if n <= 1 {
		return big.NewInt(1)
	}

	// 1. Compute exponent of 2: nu_2(n!) = n - popcount(n)
	nu2 := uint(n - int64(bits.OnesCount64(uint64(n))))

	// 2. Sieve odd primes up to n
	oddPrimes := sieveOddPrimes(n)

	// 3. Compute p^(nu_p(n!)) for each odd prime
	terms := make([]*big.Int, 0, len(oddPrimes))
	for _, p := range oddPrimes {
		e := legendreExponent(n, p)
		if e == 1 {
			terms = append(terms, big.NewInt(p))
		} else {
			pBig := big.NewInt(p)
			eBig := big.NewInt(e)
			term := new(big.Int).Exp(pBig, eBig, nil)
			terms = append(terms, term)
		}
	}

	// 4. Multiply odd prime powers via binary splitting product tree
	oddProduct := binarySplittingProduct(terms)

	// 5. Shift left by nu2 to multiply by 2^(nu_2(n!))
	result := new(big.Int).Lsh(oddProduct, nu2)
	return result
}

// legendreExponent calculates the exponent of prime p in n! using Legendre's formula:
// nu_p(n!) = sum_{k >= 1} floor(n / p^k).
func legendreExponent(n, p int64) int64 {
	var exp int64
	for n >= p {
		n /= p
		exp += n
	}
	return exp
}

// sieveOddPrimes finds all odd primes up to n using the Sieve of Eratosthenes.
func sieveOddPrimes(n int64) []int64 {
	if n < 3 {
		return nil
	}

	// Index i represents the odd number (2*i + 3)
	maxIdx := (n - 3) / 2
	composite := make([]bool, maxIdx+1)

	for i := int64(0); (2*i+3)*(2*i+3) <= n; i++ {
		if !composite[i] {
			p := 2*i + 3
			// Mark multiples of p starting from p*p
			startIdx := (p*p - 3) / 2
			for k := startIdx; k <= maxIdx; k += p {
				composite[k] = true
			}
		}
	}

	primes := make([]int64, 0, maxIdx/2)
	for i := int64(0); i <= maxIdx; i++ {
		if !composite[i] {
			primes = append(primes, 2*i+3)
		}
	}
	return primes
}

// binarySplittingProduct computes the balanced product of a slice of big.Ints
// using divide-and-conquer to maintain equal digit lengths during multiplication.
func binarySplittingProduct(nums []*big.Int) *big.Int {
	if len(nums) == 0 {
		return big.NewInt(1)
	}
	if len(nums) == 1 {
		return nums[0]
	}
	mid := len(nums) / 2
	left := binarySplittingProduct(nums[:mid])
	right := binarySplittingProduct(nums[mid:])
	if left.BitLen() >= 1024 && right.BitLen() >= 1024 {
		return NTTMultiply(left, right)
	}
	return new(big.Int).Mul(left, right)
}
