package calc

import (
	"math/big"
	"testing"
)

// TestDeterministicPrimeBasic verifies basic prime and composite cases including edge cases.
func TestDeterministicPrimeBasic(t *testing.T) {
	// Negative and small numbers
	testCases := []struct {
		n    int64
		want bool
	}{
		{-10, false},
		{-1, false},
		{0, false},
		{1, false},
		{2, true},
		{3, true},
		{4, false},
		{5, true},
		{6, false},
		{7, true},
		{8, false},
		{9, false},
		{10, false},
		{11, true},
		{13, true},
		{17, true},
		{19, true},
		{23, true},
		{29, true},
		{31, true},
		{37, true},
		{41, true},
		{43, true},
		{47, true},
		{49, false}, // 7^2
		{97, true},
		{100, false},
		{10007, true},
		{10009, true},
		{1000000007, true},
		{1000000009, true},
		{2147483647, true}, // 2^31 - 1 (Mersenne 8)
		{2147483649, false},
	}

	for _, tc := range testCases {
		got := IsDeterministicPrime(big.NewInt(tc.n))
		if got != tc.want {
			t.Errorf("IsDeterministicPrime(%d) = %v; want %v", tc.n, got, tc.want)
		}
	}
}

// TestCarmichaelNumbers verifies that known Carmichael numbers (Fermat pseudoprimes)
// are correctly identified as COMPOSITE and never misclassified as primes.
func TestCarmichaelNumbers(t *testing.T) {
	carmichaels := []int64{
		561,    // 3 * 11 * 17
		1105,   // 5 * 13 * 17
		1729,   // 7 * 13 * 19 (Hardy-Ramanujan)
		2465,   // 5 * 17 * 29
		2821,   // 7 * 7 * 31 (7 * 13 * 31)
		6601,   // 7 * 23 * 41
		8911,   // 7 * 19 * 67
		10585,  // 5 * 29 * 73
		15841,  // 7 * 31 * 73
		29341,  // 13 * 37 * 61
		41041,  // 7 * 11 * 13 * 41
		46657,  // 13 * 37 * 97
		52633,  // 7 * 73 * 103
		62745,  // 3 * 5 * 47 * 89
		63973,  // 7 * 13 * 19 * 37
		75361,  // 11 * 13 * 17 * 31
		101101, // 7 * 11 * 13 * 101
		115921, // 13 * 37 * 241
		162401, // 17 * 41 * 233
		172081, // 7 * 13 * 31 * 61
		188461, // 7 * 13 * 19 * 97
		252601, // 41 * 61 * 101
		278545, // 5 * 17 * 29 * 113
		294409, // 37 * 73 * 109
		314821, // 13 * 61 * 397
		334153, // 19 * 43 * 409
		340561, // 13 * 17 * 23 * 67
		399001, // 31 * 61 * 211
		410041, // 41 * 73 * 137
		488881, // 37 * 73 * 181
		512461, // 31 * 61 * 271
	}

	for _, c := range carmichaels {
		if IsDeterministicPrime(big.NewInt(c)) {
			t.Errorf("Carmichael number %d was incorrectly classified as prime", c)
		}
		if IsDeterministicPrime64(uint64(c)) {
			t.Errorf("IsDeterministicPrime64(%d) incorrectly classified Carmichael as prime", c)
		}
	}
}

// TestLarge64BitPrimes verifies large 64-bit primes near uint64 limits.
func TestLarge64BitPrimes(t *testing.T) {
	largePrimes := []uint64{
		2305843009213693951, // 2^61 - 1 (Mersenne 9)
		18446744073709551557, // 2^64 - 59 (largest prime < 2^64)
		18446744073709551533, // 2^64 - 83
		18446744073709551521, // 2^64 - 95
	}

	for _, p := range largePrimes {
		if !IsDeterministicPrime64(p) {
			t.Errorf("Large prime %d failed IsDeterministicPrime64", p)
		}
		bigP := new(big.Int).SetUint64(p)
		if !IsDeterministicPrime(bigP) {
			t.Errorf("Large prime %d failed IsDeterministicPrime", p)
		}
	}

	largeComposites := []uint64{
		18446744073709551559, // 2^64 - 57
		18446744073709551615, // 2^64 - 1
		2305843009213693953, // 2^61 + 1 (composite: divisible by 3)
	}

	for _, c := range largeComposites {
		if IsDeterministicPrime64(c) {
			t.Errorf("Large composite %d incorrectly passed IsDeterministicPrime64", c)
		}
		bigC := new(big.Int).SetUint64(c)
		if IsDeterministicPrime(bigC) {
			t.Errorf("Large composite %d incorrectly passed IsDeterministicPrime", c)
		}
	}
}

// TestArbitraryPrecisionPrimes tests numbers exceeding 2^64 (Baillie-PSW path).
func TestArbitraryPrecisionPrimes(t *testing.T) {
	// 2^127 - 1 (Mersenne 12: 170141183460469231731687303715884105727)
	m127, _ := new(big.Int).SetString("170141183460469231731687303715884105727", 10)
	if !IsDeterministicPrime(m127) {
		t.Errorf("Mersenne 127 (%s) failed IsDeterministicPrime", m127.String())
	}

	// m127 - 1 is composite
	m127Minus1 := new(big.Int).Sub(m127, big.NewInt(1))
	if IsDeterministicPrime(m127Minus1) {
		t.Errorf("Mersenne 127 - 1 (%s) was incorrectly classified as prime", m127Minus1.String())
	}
}

// TestConsistencyWithSieve checks all numbers up to 10,000 against a simple Sieve of Eratosthenes.
func TestConsistencyWithSieve(t *testing.T) {
	const limit = 10000
	isPrime := make([]bool, limit+1)
	for i := 2; i <= limit; i++ {
		isPrime[i] = true
	}
	for p := 2; p*p <= limit; p++ {
		if isPrime[p] {
			for i := p * p; i <= limit; i += p {
				isPrime[i] = false
			}
		}
	}

	for i := 0; i <= limit; i++ {
		got64 := IsDeterministicPrime64(uint64(i))
		gotBig := IsDeterministicPrime(big.NewInt(int64(i)))

		if got64 != isPrime[i] {
			t.Fatalf("Mismatch at %d: IsDeterministicPrime64=%v, sieve=%v", i, got64, isPrime[i])
		}
		if gotBig != isPrime[i] {
			t.Fatalf("Mismatch at %d: IsDeterministicPrime=%v, sieve=%v", i, gotBig, isPrime[i])
		}
	}
}

// TestBasesEquivalence verifies that millerRabin7 and millerRabin12 produce identical results.
func TestBasesEquivalence(t *testing.T) {
	// Check all odd numbers up to 50,000
	for n := uint64(3); n < 50000; n += 2 {
		r7 := millerRabin7(n)
		r12 := millerRabin12(n)
		if r7 != r12 {
			t.Fatalf("Equivalence failed at n=%d: millerRabin7=%v, millerRabin12=%v", n, r7, r12)
		}
	}
}

// Benchmarks: 7-base vs 12-base
func BenchmarkMillerRabin7(b *testing.B) {
	primes := []uint64{
		1000000007,
		2147483647,
		2305843009213693951,
		18446744073709551557,
	}
	for b.Loop() {
		for _, p := range primes {
			_ = millerRabin7(p)
		}
	}
}

func BenchmarkMillerRabin12(b *testing.B) {
	primes := []uint64{
		1000000007,
		2147483647,
		2305843009213693951,
		18446744073709551557,
	}
	for b.Loop() {
		for _, p := range primes {
			_ = millerRabin12(p)
		}
	}
}

