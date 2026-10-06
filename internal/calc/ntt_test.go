package calc

import (
	"math/big"
	"math/rand"
	"testing"
)

func TestNTTMultiply_Basic(t *testing.T) {
	cases := []struct {
		a, b int64
	}{
		{0, 100},
		{100, 0},
		{1, 1},
		{12345, 6789},
		{-12345, 6789},
		{12345, -6789},
		{-12345, -6789},
	}

	for _, c := range cases {
		a := big.NewInt(c.a)
		b := big.NewInt(c.b)
		expected := new(big.Int).Mul(a, b)
		got := NTTMultiply(a, b)
		if got.Cmp(expected) != 0 {
			t.Errorf("NTTMultiply(%v, %v) = %v; want %v", a, b, got, expected)
		}
	}
}

func TestNTTMultiply_LargeRandom(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	// Test with numbers that exceed the standard 1024-bit threshold
	for i := 0; i < 5; i++ {
		a := new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), 2048))
		b := new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), 2048))

		expected := new(big.Int).Mul(a, b)
		got := NTTMultiply(a, b)

		if got.Cmp(expected) != 0 {
			t.Fatalf("NTTMultiply failed for 2048-bit numbers at iteration %d", i)
		}
	}
}

func TestNTTMultiply_Huge50000Bits(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	a := new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), 50000))
	b := new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), 50000))

	expected := new(big.Int).Mul(a, b)
	got := NTTMultiply(a, b)

	if got.Cmp(expected) != 0 {
		t.Fatalf("NTTMultiply failed for 50000-bit multiplication")
	}
}

func TestNTTMultiply_AlgebraicProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	a := new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), 1500))
	b := new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), 1500))
	c := new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), 1500))

	// 1. Commutativity: a * b == b * a
	ab := NTTMultiply(a, b)
	ba := NTTMultiply(b, a)
	if ab.Cmp(ba) != 0 {
		t.Errorf("Commutativity failed")
	}

	// 2. Distributivity: a * (b + c) == a * b + a * c
	bPlusC := new(big.Int).Add(b, c)
	left := NTTMultiply(a, bPlusC)

	ac := NTTMultiply(a, c)
	right := new(big.Int).Add(ab, ac)

	if left.Cmp(right) != 0 {
		t.Errorf("Distributivity failed: got diff %v", new(big.Int).Sub(left, right))
	}
}
