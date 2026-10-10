package calc

import (
	"math/big"
	"testing"
)

func TestCyclotomic8_RingAxioms(t *testing.T) {
	zero := Cyclotomic8Zero()
	one := Cyclotomic8One()
	omega := Cyclotomic8Omega()
	invSqrt2 := Cyclotomic8InvSqrt2()

	// 1. Zero and One identities
	if !zero.IsZero() {
		t.Fatalf("Cyclotomic8Zero() should be zero")
	}
	if !one.Mul(omega).Equals(omega) {
		t.Fatalf("1 * omega should equal omega")
	}
	if !zero.Add(omega).Equals(omega) {
		t.Fatalf("0 + omega should equal omega")
	}

	// 2. omega^2 == i, omega^4 == -1, omega^8 == 1
	omega2 := omega.Mul(omega)
	omega4 := omega2.Mul(omega2)
	omega8 := omega4.Mul(omega4)

	negOne := one.Neg()
	if !omega4.Equals(negOne) {
		t.Fatalf("omega^4 should equal -1, got: %v", omega4)
	}
	if !omega8.Equals(one) {
		t.Fatalf("omega^8 should equal 1, got: %v", omega8)
	}

	// 3. invSqrt2 * sqrt2 == 1
	// In D[omega], sqrt(2) = omega - omega^3 (since omega = (1+i)/sqrt(2), omega^3 = (-1+i)/sqrt(2), omega - omega^3 = 2/sqrt(2) = sqrt(2))
	sqrt2 := omega.Sub(omega.Mul(omega2)) // omega - omega^3
	prod := invSqrt2.Mul(sqrt2)
	if !prod.Equals(one) {
		t.Fatalf("invSqrt2 * sqrt2 should equal 1, got: %v", prod)
	}
}

func TestCyclotomic8_Conj(t *testing.T) {
	omega := Cyclotomic8Omega()
	conjOmega := omega.Conj()

	// conj(omega) = omega^7 = -omega^3
	omega2 := omega.Mul(omega)
	omega3 := omega2.Mul(omega)
	negOmega3 := omega3.Neg()

	if !conjOmega.Equals(negOmega3) {
		t.Fatalf("conj(omega) should equal -omega^3, got: %v, expected: %v", conjOmega, negOmega3)
	}

	// norm(omega) = omega * conj(omega) = 1
	norm := omega.Mul(conjOmega)
	if !norm.Equals(Cyclotomic8One()) {
		t.Fatalf("omega * conj(omega) should equal 1, got: %v", norm)
	}
}

func TestCyclotomic8_IsRootOfUnity(t *testing.T) {
	omega := Cyclotomic8Omega()
	curr := Cyclotomic8One()

	for k := 0; k < 8; k++ {
		power, ok := curr.IsRootOfUnity()
		if !ok || power != k {
			t.Fatalf("expected omega^%d to be root of unity with power %d, got ok=%v, power=%d", k, k, ok, power)
		}
		curr = curr.Mul(omega)
	}

	// 2 is not a root of unity
	two := NewCyclotomic8Int(2)
	if _, ok := two.IsRootOfUnity(); ok {
		t.Fatalf("2 should not be identified as an 8th root of unity")
	}
}

func TestCyclotomic8_SdeReduction(t *testing.T) {
	// sqrt(2) / sqrt(2) = 1
	// sqrt(2) = omega - omega^3 with k=0. Divided by sqrt(2) has k=1.
	// NewCyclotomic8 with a0=0, a1=1, a2=0, a3=-1, k=1 should reduce to 1 (a0=1, a1=0, a2=0, a3=0, k=0).
	c := NewCyclotomic8(big.NewInt(0), big.NewInt(1), big.NewInt(0), big.NewInt(-1), 1)
	one := Cyclotomic8One()
	if !c.Equals(one) {
		t.Fatalf("expected SDE reduced sqrt(2)/sqrt(2) to equal 1, got: %v", c)
	}
}

func TestCyclotomic8_ToASTNode(t *testing.T) {
	one := Cyclotomic8One()
	node1 := one.ToASTNode()
	if node1 == nil {
		t.Fatalf("ToASTNode() returned nil for 1")
	}

	omega := Cyclotomic8Omega()
	nodeOmega := omega.ToASTNode()
	if nodeOmega == nil {
		t.Fatalf("ToASTNode() returned nil for omega")
	}
}

func TestCyclotomic8_AssociativityAndDistributivity(t *testing.T) {
	// A = 1 + 2*omega - omega^2 / sqrt(2)^2
	// B = -1 + omega^3 / sqrt(2)
	// C = 3*omega + 2*omega^2 / sqrt(2)^3
	a := NewCyclotomic8(big.NewInt(1), big.NewInt(2), big.NewInt(-1), big.NewInt(0), 2)
	b := NewCyclotomic8(big.NewInt(-1), big.NewInt(0), big.NewInt(0), big.NewInt(1), 1)
	c := NewCyclotomic8(big.NewInt(0), big.NewInt(3), big.NewInt(2), big.NewInt(0), 3)

	// (A + B) * C == A * C + B * C
	leftDist := (a.Add(b)).Mul(c)
	rightDist := (a.Mul(c)).Add(b.Mul(c))
	if !leftDist.Equals(rightDist) {
		t.Fatalf("Distributivity failed: (A+B)*C != A*C + B*C")
	}

	// (A * B) * C == A * (B * C)
	leftAssoc := (a.Mul(b)).Mul(c)
	rightAssoc := a.Mul(b.Mul(c))
	if !leftAssoc.Equals(rightAssoc) {
		t.Fatalf("Associativity failed: (A*B)*C != A*(B*C)")
	}
}

