package calc

import (
	"math/big"
	"testing"
)

func TestEvalInvMod(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"inv_mod(3, 11)", "4"},
		{"inv_mod(10, 17)", "12"},
		{"inv_mod(1, 100)", "1"},
		{"inv_mod(-3, 11)", "7"},
		{"inv_mod(65537, 1000000007)", "743534192"},
	}

	for _, tc := range tests {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("EvalString(%q) unexpected error: %v", tc.input, err)
		}
		if res.String() != tc.expected {
			t.Errorf("EvalString(%q) = %s; want %s", tc.input, res.String(), tc.expected)
		}
	}

	// Error cases
	errorInputs := []string{
		"inv_mod(6, 9)",  // not coprime
		"inv_mod(2, 1)",  // m < 2
		"inv_mod(2, 0)",  // m = 0
		"inv_mod(1/2, 5)", // non-integer
	}
	for _, inp := range errorInputs {
		_, err := EvalString(inp)
		if err == nil {
			t.Errorf("EvalString(%q) expected error, got nil", inp)
		}
	}
}

func TestEvalCRT(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic coprime CRT with list arguments
		{"crt([2, 3, 2], [3, 5, 7])", "23"},
		// Paired flat arguments
		{"crt(2, 3, 3, 5, 2, 7)", "23"},
		// Single congruence
		{"crt([4], [7])", "4"},
		// High-precision / large prime coprime CRT (Garner's Algorithm verification)
		{"crt([123456, 654321], [1000000007, 1000000009])", "499734575498265460"},
		// Non-coprime compatible congruences:
		// x = 3 (mod 4) and x = 5 (mod 6) => x = 11 (mod 12)
		{"crt([3, 5], [4, 6])", "11"},
		// x = 2 (mod 4) and x = 4 (mod 6) => x = 10 (mod 12)
		{"crt([2, 4], [4, 6])", "10"},
		// 3 equations with non-coprime moduli:
		// x = 1 (mod 2), x = 1 (mod 4), x = 5 (mod 6) => x = 5 (mod 12)
		{"crt([1, 1, 5], [2, 4, 6])", "5"},
	}

	for _, tc := range tests {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("EvalString(%q) unexpected error: %v", tc.input, err)
		}
		if res.String() != tc.expected {
			t.Errorf("EvalString(%q) = %s; want %s", tc.input, res.String(), tc.expected)
		}
	}

	// Error cases: incompatible congruences and invalid args
	errorInputs := []string{
		"crt([1, 2], [4, 6])",     // gcd(4, 6)=2 does not divide 2-1 => no solution
		"crt([1, 2, 3], [3, 5])",  // dimension mismatch
		"crt([1], [1])",           // modulus < 2
		"crt()",                   // too few args
		"crt([1/2], [3])",         // non-integer remainder
	}
	for _, inp := range errorInputs {
		_, err := EvalString(inp)
		if err == nil {
			t.Errorf("EvalString(%q) expected error, got nil", inp)
		}
	}
}

func TestEvalTotient(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"totient(1)", "1"},
		{"totient(2)", "1"},
		{"totient(3)", "2"},
		{"totient(4)", "2"},
		{"totient(5)", "4"},
		{"totient(6)", "2"},
		{"totient(7)", "6"},
		{"totient(8)", "4"},
		{"totient(9)", "6"},
		{"totient(10)", "4"},
		{"totient(36)", "12"},
		{"totient(100)", "40"},
		{"totient(1000000007)", "1000000006"}, // prime
		{"totient(1073741824)", "536870912"},   // 2^30 -> 2^29
	}

	for _, tc := range tests {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("EvalString(%q) unexpected error: %v", tc.input, err)
		}
		if res.String() != tc.expected {
			t.Errorf("EvalString(%q) = %s; want %s", tc.input, res.String(), tc.expected)
		}
	}

	// Error cases
	errorInputs := []string{
		"totient(0)",
		"totient(-10)",
		"totient(1/2)",
	}
	for _, inp := range errorInputs {
		_, err := EvalString(inp)
		if err == nil {
			t.Errorf("EvalString(%q) expected error, got nil", inp)
		}
	}
}

func TestEvalIsPrime(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Non-primes < 2
		{"is_prime(-5)", "0"},
		{"is_prime(0)", "0"},
		{"is_prime(1)", "0"},
		// Small primes
		{"is_prime(2)", "1"},
		{"is_prime(3)", "1"},
		{"is_prime(5)", "1"},
		{"is_prime(7)", "1"},
		{"is_prime(11)", "1"},
		{"is_prime(13)", "1"},
		// Small composites
		{"is_prime(4)", "0"},
		{"is_prime(6)", "0"},
		{"is_prime(8)", "0"},
		{"is_prime(9)", "0"},
		{"is_prime(15)", "0"},
		{"is_prime(25)", "0"},
		// Carmichael numbers (strong pseudoprimes which trick Fermat test)
		{"is_prime(561)", "0"},  // 3 * 11 * 17
		{"is_prime(1105)", "0"}, // 5 * 13 * 17
		{"is_prime(1729)", "0"}, // 7 * 13 * 19 (Hardy-Ramanujan number)
		{"is_prime(2465)", "0"},
		{"is_prime(2821)", "0"},
		{"is_prime(6601)", "0"},
		{"is_prime(8911)", "0"},
		// Medium primes
		{"is_prime(1000000007)", "1"},
		{"is_prime(1000000009)", "1"},
		// 64-bit boundary primes & composites (Sorenson & Webster 2015 verification)
		{"is_prime(2305843009213693951)", "1"},  // Mersenne prime 2^61 - 1
		{"is_prime(18446744073709551557)", "1"}, // Largest prime < 2^64 (2^64 - 59)
		{"is_prime(18446744073709551559)", "0"}, // 2^64 - 57 (composite: 3 * 6148914691236517186 + 1 ...)
		// Large prime > 64-bit (Mersenne prime 2^127 - 1)
		{"is_prime(170141183460469231731687303715884105727)", "1"},
	}

	for _, tc := range tests {
		res, err := EvalString(tc.input)
		if err != nil {
			t.Fatalf("EvalString(%q) unexpected error: %v", tc.input, err)
		}
		if res.String() != tc.expected {
			t.Errorf("EvalString(%q) = %s; want %s", tc.input, res.String(), tc.expected)
		}
	}

	// Error cases
	errorInputs := []string{
		"is_prime(1/2)",
		"is_prime(3.14)",
	}
	for _, inp := range errorInputs {
		_, err := EvalString(inp)
		if err == nil {
			t.Errorf("EvalString(%q) expected error, got nil", inp)
		}
	}
}

func TestExtGCD(t *testing.T) {
	a := big.NewInt(240)
	b := big.NewInt(46)
	g, x, y := extGCD(a, b)
	// 240*x + 46*y = g
	term1 := new(big.Int).Mul(a, x)
	term2 := new(big.Int).Mul(b, y)
	sum := new(big.Int).Add(term1, term2)
	if sum.Cmp(g) != 0 {
		t.Errorf("extGCD identity failed: %s*%s + %s*%s = %s != %s", a, x, b, y, sum, g)
	}
	if g.Int64() != 2 {
		t.Errorf("gcd(240, 46) expected 2, got %s", g)
	}
}
