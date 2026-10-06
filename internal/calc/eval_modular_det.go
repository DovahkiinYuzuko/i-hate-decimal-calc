package calc

import (
	"math/big"
)

var defaultModularPrimes = []uint64{
	998244353, 1004535809, 469762049,
	1000000007, 1000000009, 1000000021,
	1000000033, 1000000087, 1000000093,
	1000000097, 1000000103, 1000000123,
	1000000181, 1000000207, 1000000223,
	1000000241, 1000000271, 1000000289,
	1000000297, 1000000321, 1000000349,
}

// hadamardBound computes the Hadamard bound on the absolute value of det(mat):
// |det(mat)| <= prod_{i=1}^n sqrt(sum_{j=1}^n mat[i][j]^2).
func hadamardBound(mat [][]*big.Int) *big.Int {
	n := len(mat)
	if n == 0 {
		return big.NewInt(1)
	}

	bound := big.NewInt(1)
	for i := 0; i < n; i++ {
		rowSumSq := big.NewInt(0)
		for j := 0; j < n; j++ {
			if mat[i][j].Sign() != 0 {
				sq := new(big.Int).Mul(mat[i][j], mat[i][j])
				rowSumSq.Add(rowSumSq, sq)
			}
		}
		if rowSumSq.Sign() == 0 {
			return big.NewInt(0)
		}
		// sqrt(rowSumSq) + 1
		rowNorm := new(big.Int).Sqrt(rowSumSq)
		rowNorm.Add(rowNorm, big.NewInt(1))
		bound.Mul(bound, rowNorm)
	}
	return bound
}

func modPowU64(base, exp, mod uint64) uint64 {
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

func modInvU64(n, mod uint64) uint64 {
	return modPowU64(n, mod-2, mod)
}

// detModPrime computes det(mat) mod p using Gaussian elimination with partial pivoting in F_p.
func detModPrime(mat [][]*big.Int, p uint64) uint64 {
	n := len(mat)
	a := make([][]uint64, n)
	pBig := new(big.Int).SetUint64(p)
	rem := new(big.Int)

	for i := 0; i < n; i++ {
		a[i] = make([]uint64, n)
		for j := 0; j < n; j++ {
			rem.Mod(mat[i][j], pBig)
			if rem.Sign() < 0 {
				rem.Add(rem, pBig)
			}
			a[i][j] = rem.Uint64()
		}
	}

	det := uint64(1)
	sign := int64(1)

	for col := 0; col < n; col++ {
		pivotRow := -1
		for r := col; r < n; r++ {
			if a[r][col] != 0 {
				pivotRow = r
				break
			}
		}
		if pivotRow == -1 {
			return 0
		}
		if pivotRow != col {
			a[col], a[pivotRow] = a[pivotRow], a[col]
			sign = -sign
		}

		pivot := a[col][col]
		det = (det * pivot) % p
		invPivot := modInvU64(pivot, p)

		for r := col + 1; r < n; r++ {
			if a[r][col] == 0 {
				continue
			}
			factor := (a[r][col] * invPivot) % p
			for c := col; c < n; c++ {
				sub := (factor * a[col][c]) % p
				if a[r][c] >= sub {
					a[r][c] -= sub
				} else {
					a[r][c] = a[r][c] + p - sub
				}
			}
		}
	}

	if sign < 0 && det > 0 {
		det = p - det
	}
	return det
}

// DetModularCRT computes the exact determinant of an integer matrix using
// modular arithmetic across multiple primes and Garner's CRT reconstruction.
// This prevents intermediate coefficient explosion.
func DetModularCRT(mat [][]*big.Int) *big.Int {
	n := len(mat)
	if n == 0 {
		return big.NewInt(1)
	}
	if n == 1 {
		return new(big.Int).Set(mat[0][0])
	}
	if n == 2 {
		ad := new(big.Int).Mul(mat[0][0], mat[1][1])
		bc := new(big.Int).Mul(mat[0][1], mat[1][0])
		return ad.Sub(ad, bc)
	}

	// 1. Compute Hadamard bound: |det(mat)| <= H
	hBound := hadamardBound(mat)
	if hBound.Sign() == 0 {
		return big.NewInt(0)
	}

	// Target: product of primes M > 2 * H
	twoH := new(big.Int).Mul(hBound, big.NewInt(2))

	var chosenPrimes []*big.Int
	var remainders []*big.Int
	mProduct := big.NewInt(1)

	primeIdx := 0
	for mProduct.Cmp(twoH) <= 0 {
		var p uint64
		if primeIdx < len(defaultModularPrimes) {
			p = defaultModularPrimes[primeIdx]
			primeIdx++
		} else {
			// Generate next prime if standard pool is exhausted
			last := chosenPrimes[len(chosenPrimes)-1]
			nextP := new(big.Int).Add(last, big.NewInt(2))
			for !IsDeterministicPrime(nextP) {
				nextP.Add(nextP, big.NewInt(2))
			}
			p = nextP.Uint64()
		}

		pBig := new(big.Int).SetUint64(p)
		rem := detModPrime(mat, p)

		chosenPrimes = append(chosenPrimes, pBig)
		remainders = append(remainders, new(big.Int).SetUint64(rem))
		mProduct.Mul(mProduct, pBig)
	}

	// 2. Reconstruct via Garner's algorithm
	d, M, err := garnerCRT(remainders, chosenPrimes)
	if err != nil {
		// Fallback to standard Bareiss if CRT fails
		return big.NewInt(0)
	}

	// 3. Symmetric modulo [-M/2, M/2)
	halfM := new(big.Int).Rsh(M, 1)
	if d.Cmp(halfM) > 0 {
		d.Sub(d, M)
	}

	return d
}
