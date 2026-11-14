package calc

import (
	"math/big"
	"testing"
)

func productFromFactors(primes, unfactored []primeFactor) *big.Int {
	prod := big.NewInt(1)
	tmp := new(big.Int)
	for _, pf := range primes {
		tmp.Exp(pf.prime, big.NewInt(int64(pf.exp)), nil)
		prod.Mul(prod, tmp)
	}
	for _, cf := range unfactored {
		tmp.Exp(cf.prime, big.NewInt(int64(cf.exp)), nil)
		prod.Mul(prod, tmp)
	}
	return prod
}

func TestFactorECMDeterministic(t *testing.T) {
	// Deductive: perfect powers n = m^k are decomposed immediately before ECM/rho
	t.Run("Deductive_PerfectPowerDetection", func(t *testing.T) {
		base, ok := new(big.Int).SetString("1000000007", 10)
		if !ok {
			t.Fatal("failed to parse prime")
		}
		n := new(big.Int).Exp(base, big.NewInt(5), nil) // ~150-bit exact 5th power

		root, exp := detectPerfectPower(n)
		if root == nil || root.Cmp(base) != 0 || exp != 5 {
			t.Fatalf("detectPerfectPower((10^9+7)^5) = (%v, %d), want (%v, 5)", root, exp, base)
		}

		primes, unfactored := factorBigIntWithConfig(n, defaultECMConfig())
		if len(unfactored) != 0 || len(primes) != 1 || primes[0].prime.Cmp(base) != 0 || primes[0].exp != 5 {
			t.Fatalf("got primes=%v unfactored=%v, want [(1000000007, 5)]", primes, unfactored)
		}
	})

	// Deductive + Metamorphic: 80-bit+ balanced semiprimes factor cleanly into sorted prime factors
	// without triggering fallback O(sqrt(N)) trial division.
	t.Run("Deductive_LargeSemiprimeFactorization", func(t *testing.T) {
		p1, _ := new(big.Int).SetString("10000000000037", 10) // 14-digit prime (> 10^13)
		p2, _ := new(big.Int).SetString("10000000000051", 10) // 14-digit prime (> 10^13)
		n := new(big.Int).Mul(p1, p2)                         // ~93-bit semiprime

		// Force ECM directly by disabling Brent rho and setting small trial limit
		cfg := defaultECMConfig()
		cfg.BrentMaxSteps = 0
		primes, unfactored := factorBigIntWithConfig(n, cfg)
		if len(unfactored) != 0 {
			t.Fatalf("expected full ECM factorization of %s, got unfactored %v", n.String(), unfactored)
		}
		if len(primes) != 2 || primes[0].prime.Cmp(p1) != 0 || primes[1].prime.Cmp(p2) != 0 {
			t.Fatalf("unexpected factors: %v, want [%s, %s]", primes, p1.String(), p2.String())
		}
		if productFromFactors(primes, unfactored).Cmp(n) != 0 {
			t.Errorf("product of factors does not reconstruct %s", n.String())
		}

		// Also verify end-to-end factor(...) through EvalString on a >64-bit semiprime
		res, err := EvalString("factor(" + n.String() + ")")
		if err != nil {
			t.Fatalf("EvalString factor failed: %v", err)
		}
		expected := p1.String() + " * " + p2.String()
		if res.String() != expected {
			t.Errorf("factor(%s) = %s, want %s", n.String(), res.String(), expected)
		}
	})

	// Contrapositive: tight ECM work budget exceeded returns valid partial factorization
	// with algebraic product preservation (known primes * remaining composite == N).
	t.Run("Contrapositive_TightBudgetPartialFactorization", func(t *testing.T) {
		// RSA-like 160-bit balanced semiprime constructed from two 80-bit primes,
		// multiplied by small prime factor 17.
		q1, _ := new(big.Int).SetString("1208925819614629174706189", 10) // nextprime(2^80)
		q2, _ := new(big.Int).SetString("1208925819614629174706261", 10) // nextprime(nextprime(2^80))
		if !IsDeterministicPrime(q1) || !IsDeterministicPrime(q2) {
			t.Fatalf("test primes failed primality check")
		}
		bigComp := new(big.Int).Mul(q1, q2)
		small := big.NewInt(17)
		n := new(big.Int).Mul(small, bigComp)

		tightCfg := ecmWorkConfig{
			SmallTrialLimit: 100,
			BrentMaxSteps:   5,
			Tiers: []ecmTier{
				{B1: 20, B2: 50, MaxCurves: 1},
			},
		}

		primes, unfactored := factorBigIntWithConfig(n, tightCfg)
		if len(primes) != 1 || primes[0].prime.Cmp(small) != 0 || primes[0].exp != 1 {
			t.Errorf("expected partial prime factor [17], got %v", primes)
		}
		if len(unfactored) != 1 || unfactored[0].prime.Cmp(bigComp) != 0 {
			t.Errorf("expected unfactored composite remainder %s, got %v", bigComp.String(), unfactored)
		}
		if productFromFactors(primes, unfactored).Cmp(n) != 0 {
			t.Errorf("partial factorization product does not equal input N")
		}
	})

	// Edge & Boundary: small primes, 1, 0, negative numbers
	t.Run("Boundary_EdgeIntegers", func(t *testing.T) {
		for _, tc := range []struct {
			in   string
			want string
		}{
			{"factor(0)", "0"},
			{"factor(1)", "1"},
			{"factor(-12)", "-1 * 2^2 * 3"},
			{"factor(1000000007)", "1000000007"},
		} {
			res, err := EvalString(tc.in)
			if err != nil {
				t.Fatalf("EvalString(%s) error: %v", tc.in, err)
			}
			if res.String() != tc.want {
				t.Errorf("%s = %s, want %s", tc.in, res.String(), tc.want)
			}
		}
	})
}
