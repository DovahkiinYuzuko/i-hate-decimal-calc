package calc

import (
	"math/big"
	"math/bits"
)

// extGCD computes the Extended Euclidean Algorithm on a and b,
// returning gcd, x, and y such that a*x + b*y = gcd.
func extGCD(a, b *big.Int) (*big.Int, *big.Int, *big.Int) {
	if b.Sign() == 0 {
		return new(big.Int).Set(a), big.NewInt(1), big.NewInt(0)
	}
	g, x1, y1 := extGCD(b, new(big.Int).Mod(a, b))
	// x = y1, y = x1 - (a / b) * y1
	q := new(big.Int).Div(a, b)
	x := new(big.Int).Set(y1)
	y := new(big.Int).Sub(x1, new(big.Int).Mul(q, y1))
	return g, x, y
}

// EvalInvMod computes the modular inverse of a modulo m: a * x = 1 (mod m).
// Both a and m must be integers with m >= 2.
func EvalInvMod(aNode, mNode Node) (Node, error) {
	aRat, okA := aNode.(*RationalNode)
	mRat, okM := mNode.(*RationalNode)
	if !okA || !aRat.Val.IsInt() {
		return nil, NewDomainError("", "inv_mod: first argument must be an integer, got %s", aNode.String())
	}
	if !okM || !mRat.Val.IsInt() {
		return nil, NewDomainError("", "inv_mod: second argument (modulus) must be an integer, got %s", mNode.String())
	}

	m := mRat.Val.Num()
	if m.Cmp(big.NewInt(2)) < 0 {
		return nil, NewDomainError("", "inv_mod: modulus must be an integer >= 2, got %s", m.String())
	}

	a := new(big.Int).Mod(aRat.Val.Num(), m)
	if a.Sign() < 0 {
		a.Add(a, m)
	}

	g, x, _ := extGCD(a, m)
	if g.Cmp(big.NewInt(1)) != 0 {
		return nil, NewDomainError("", "inv_mod: %s and %s are not coprime (gcd = %s), no modular inverse exists", aRat.Val.Num().String(), m.String(), g.String())
	}

	x.Mod(x, m)
	if x.Sign() < 0 {
		x.Add(x, m)
	}

	return &RationalNode{Val: new(big.Rat).SetInt(x)}, nil
}

// solvePairCRT merges two congruences x = r1 (mod m1) and x = r2 (mod m2)
// into a single congruence x = R (mod M) where M = lcm(m1, m2).
// Returns (R, M, error). If inconsistent, returns an error.
func solvePairCRT(r1, m1, r2, m2 *big.Int) (*big.Int, *big.Int, error) {
	g, p, _ := extGCD(m1, m2)
	diff := new(big.Int).Sub(r2, r1)
	rem := new(big.Int).Mod(diff, g)
	if rem.Sign() != 0 {
		return nil, nil, NewDomainError("", "crt: system of congruences has no solution (gcd(%s, %s) = %s does not divide %s - %s)", m1.String(), m2.String(), g.String(), r2.String(), r1.String())
	}

	// M = (m1 / g) * m2
	m1DivG := new(big.Int).Div(m1, g)
	M := new(big.Int).Mul(m1DivG, m2)

	// k = (diff / g) * p mod (m2 / g)
	m2DivG := new(big.Int).Div(m2, g)
	diffDivG := new(big.Int).Div(diff, g)
	k := new(big.Int).Mul(diffDivG, p)
	k.Mod(k, m2DivG)
	if k.Sign() < 0 {
		k.Add(k, m2DivG)
	}

	// R = (r1 + m1 * k) mod M
	R := new(big.Int).Mul(m1, k)
	R.Add(R, r1)
	R.Mod(R, M)
	if R.Sign() < 0 {
		R.Add(R, M)
	}

	return R, M, nil
}

// garnerCRT implements Garner's algorithm (Garner, 1959) for pairwise coprime moduli.
// It reconstructs x in mixed-radix form to avoid coefficient swell.
func garnerCRT(remainders, moduli []*big.Int) (*big.Int, *big.Int, error) {
	k := len(moduli)
	if k == 0 {
		return big.NewInt(0), big.NewInt(1), nil
	}
	if k == 1 {
		res := new(big.Int).Mod(remainders[0], moduli[0])
		if res.Sign() < 0 {
			res.Add(res, moduli[0])
		}
		return res, new(big.Int).Set(moduli[0]), nil
	}

	// Precompute modular inverses: C[i][j] = m_i^(-1) mod m_j for i < j
	C := make([][]*big.Int, k)
	for i := 0; i < k; i++ {
		C[i] = make([]*big.Int, k)
		for j := i + 1; j < k; j++ {
			inv, err := EvalInvMod(&RationalNode{Val: new(big.Rat).SetInt(moduli[i])}, &RationalNode{Val: new(big.Rat).SetInt(moduli[j])})
			if err != nil {
				return nil, nil, err
			}
			C[i][j] = inv.(*RationalNode).Val.Num()
		}
	}

	// Compute mixed-radix digits a[i]
	a := make([]*big.Int, k)
	for i := 0; i < k; i++ {
		a[i] = new(big.Int).Mod(remainders[i], moduli[i])
		if a[i].Sign() < 0 {
			a[i].Add(a[i], moduli[i])
		}
		for j := 0; j < i; j++ {
			diff := new(big.Int).Sub(a[i], a[j])
			a[i].Mul(diff, C[j][i])
			a[i].Mod(a[i], moduli[i])
			if a[i].Sign() < 0 {
				a[i].Add(a[i], moduli[i])
			}
		}
	}

	// Reconstruct x using Horner-like scheme:
	// x = a[0] + m[0]*(a[1] + m[1]*(a[2] + ...))
	x := new(big.Int).Set(a[k-1])
	M := new(big.Int).Set(moduli[0])
	for i := k - 2; i >= 0; i-- {
		x.Mul(x, moduli[i])
		x.Add(x, a[i])
	}
	for i := 1; i < k; i++ {
		M.Mul(M, moduli[i])
	}

	return x, M, nil
}

// EvalCRT solves a system of linear congruences x = r_i (mod m_i).
// Supports either crt([r1, r2, ...], [m1, m2, ...]) or crt(r1, m1, r2, m2, ...).
func EvalCRT(args []Node) (Node, error) {
	var remNodes, modNodes []Node

	if len(args) == 2 {
		rList, okR := args[0].(*ListNode)
		mList, okM := args[1].(*ListNode)
		if okR && okM {
			remNodes = rList.Elements
			modNodes = mList.Elements
		} else {
			return nil, NewCalcError(ErrInvalidArgs, "", "crt with 2 arguments expects two lists: crt([r1, r2, ...], [m1, m2, ...])")
		}
	} else if len(args) >= 2 && len(args)%2 == 0 {
		for i := 0; i < len(args); i += 2 {
			remNodes = append(remNodes, args[i])
			modNodes = append(modNodes, args[i+1])
		}
	} else {
		return nil, NewCalcError(ErrInvalidArgs, "", "crt requires either two lists of equal length or an even number of arguments (r1, m1, r2, m2, ...)")
	}

	if len(remNodes) != len(modNodes) {
		return nil, NewCalcError(ErrDimensionMismatch, "", "crt: number of remainders (%d) must match number of moduli (%d)", len(remNodes), len(modNodes))
	}
	if len(remNodes) == 0 {
		return nil, NewCalcError(ErrInvalidArgs, "", "crt: at least one congruence equation is required")
	}

	remBig := make([]*big.Int, len(remNodes))
	modBig := make([]*big.Int, len(modNodes))

	for i := range remNodes {
		rRat, okR := remNodes[i].(*RationalNode)
		mRat, okM := modNodes[i].(*RationalNode)
		if !okR || !rRat.Val.IsInt() {
			return nil, NewDomainError("", "crt: remainder at index %d must be an integer, got %s", i+1, remNodes[i].String())
		}
		if !okM || !mRat.Val.IsInt() {
			return nil, NewDomainError("", "crt: modulus at index %d must be an integer, got %s", i+1, modNodes[i].String())
		}

		m := mRat.Val.Num()
		if m.Cmp(big.NewInt(2)) < 0 {
			return nil, NewDomainError("", "crt: modulus at index %d must be >= 2, got %s", i+1, m.String())
		}

		r := new(big.Int).Mod(rRat.Val.Num(), m)
		if r.Sign() < 0 {
			r.Add(r, m)
		}

		remBig[i] = r
		modBig[i] = m
	}

	// Check if all moduli are pairwise coprime
	allCoprime := true
	for i := 0; i < len(modBig); i++ {
		for j := i + 1; j < len(modBig); j++ {
			g := new(big.Int).GCD(nil, nil, modBig[i], modBig[j])
			if g.Cmp(big.NewInt(1)) != 0 {
				allCoprime = false
				break
			}
		}
		if !allCoprime {
			break
		}
	}

	if allCoprime {
		// Fast and memory-efficient Garner's algorithm
		sol, _, err := garnerCRT(remBig, modBig)
		if err != nil {
			return nil, err
		}
		return &RationalNode{Val: new(big.Rat).SetInt(sol)}, nil
	}

	// General CRT with non-coprime moduli: iteratively merge pairs
	currR := new(big.Int).Set(remBig[0])
	currM := new(big.Int).Set(modBig[0])

	for i := 1; i < len(remBig); i++ {
		nextR, nextM, err := solvePairCRT(currR, currM, remBig[i], modBig[i])
		if err != nil {
			return nil, err
		}
		currR = nextR
		currM = nextM
	}

	return &RationalNode{Val: new(big.Rat).SetInt(currR)}, nil
}

// getDistinctPrimeFactors extracts unique prime factors of n using wheel factorization.
func getDistinctPrimeFactors(n *big.Int) []*big.Int {
	val := new(big.Int).Abs(n)
	var primes []*big.Int
	two := big.NewInt(2)
	three := big.NewInt(3)
	rem := new(big.Int)

	if rem.Mod(val, two).Sign() == 0 {
		primes = append(primes, big.NewInt(2))
		for rem.Mod(val, two).Sign() == 0 {
			val.Div(val, two)
		}
	}

	if rem.Mod(val, three).Sign() == 0 {
		primes = append(primes, big.NewInt(3))
		for rem.Mod(val, three).Sign() == 0 {
			val.Div(val, three)
		}
	}

	if val.Cmp(big.NewInt(1)) > 0 && val.ProbablyPrime(20) {
		primes = append(primes, new(big.Int).Set(val))
		return primes
	}

	d := big.NewInt(5)
	step := big.NewInt(2)
	d2 := new(big.Int).Mul(d, d)

	for d2.Cmp(val) <= 0 {
		if val.ProbablyPrime(20) {
			break
		}
		if rem.Mod(val, d).Sign() == 0 {
			primes = append(primes, new(big.Int).Set(d))
			for rem.Mod(val, d).Sign() == 0 {
				val.Div(val, d)
			}
		}
		d.Add(d, step)
		if step.Cmp(two) == 0 {
			step.SetInt64(4)
		} else {
			step.SetInt64(2)
		}
		d2.Mul(d, d)
	}

	if val.Cmp(big.NewInt(1)) > 0 {
		primes = append(primes, new(big.Int).Set(val))
	}

	return primes
}

// EvalTotient computes Euler's totient function phi(n) = n * prod_{p|n} (1 - 1/p).
// n must be an integer >= 1.
func EvalTotient(nNode Node) (Node, error) {
	rat, ok := nNode.(*RationalNode)
	if !ok || !rat.Val.IsInt() {
		return nil, NewDomainError("", "totient: argument must be an integer, got %s", nNode.String())
	}

	n := rat.Val.Num()
	if n.Sign() <= 0 {
		return nil, NewDomainError("", "totient: argument must be a positive integer >= 1, got %s", n.String())
	}

	if n.Cmp(big.NewInt(1)) == 0 {
		return mustRational(1, 1), nil
	}

	primes := getDistinctPrimeFactors(n)
	phi := new(big.Int).Set(n)
	temp := new(big.Int)

	for _, p := range primes {
		phi.Div(phi, p)
		temp.Sub(p, big.NewInt(1))
		phi.Mul(phi, temp)
	}

	return &RationalNode{Val: new(big.Rat).SetInt(phi)}, nil
}

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

// isDeterministicPrime64 implements deterministic Miller-Rabin test for all n < 2^64
// using the 12 prime bases proved by Sorenson & Webster (2015, Math. Comp. 2017).
func isDeterministicPrime64(n uint64) bool {
	if n < 2 {
		return false
	}
	if n == 2 || n == 3 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}

	// Write n - 1 as 2^s * d
	d := n - 1
	s := 0
	for d%2 == 0 {
		d /= 2
		s++
	}

	// The 12 prime bases of Sorenson & Webster (2015)
	bases := []uint64{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37}
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

// EvalIsPrime determines whether n is prime.
// For n < 2^64, uses Sorenson & Webster (2015) deterministic 12-base Miller-Rabin test.
// For n >= 2^64, uses big.Int.ProbablyPrime(20) (Baillie-PSW).
func EvalIsPrime(nNode Node) (Node, error) {
	rat, ok := nNode.(*RationalNode)
	if !ok || !rat.Val.IsInt() {
		return nil, NewDomainError("", "is_prime: argument must be an integer, got %s", nNode.String())
	}

	n := rat.Val.Num()
	if n.Cmp(big.NewInt(2)) < 0 {
		return mustRational(0, 1), nil
	}
	if n.Cmp(big.NewInt(2)) == 0 || n.Cmp(big.NewInt(3)) == 0 {
		return mustRational(1, 1), nil
	}

	// Quick check for multiples of 2 and 3
	two := big.NewInt(2)
	three := big.NewInt(3)
	rem := new(big.Int)
	if rem.Mod(n, two).Sign() == 0 || rem.Mod(n, three).Sign() == 0 {
		return mustRational(0, 1), nil
	}

	if n.IsUint64() {
		if isDeterministicPrime64(n.Uint64()) {
			return mustRational(1, 1), nil
		}
		return mustRational(0, 1), nil
	}

	// Arbitrary precision prime test (Baillie-PSW + Miller-Rabin)
	if n.ProbablyPrime(20) {
		return mustRational(1, 1), nil
	}
	return mustRational(0, 1), nil
}
