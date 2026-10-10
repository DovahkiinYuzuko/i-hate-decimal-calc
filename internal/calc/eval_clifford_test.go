package calc

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// naiveSwapCount counts the exact number of adjacent swaps needed to bring
// the basis vectors of blade u followed by blade v into canonical ascending order.
func naiveSwapCount(u, v uint64) int {
	var seq []int
	for i := 0; i < 64; i++ {
		if (u & (1 << i)) != 0 {
			seq = append(seq, i)
		}
	}
	for i := 0; i < 64; i++ {
		if (v & (1 << i)) != 0 {
			seq = append(seq, i)
		}
	}

	swaps := 0
	n := len(seq)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if seq[j] > seq[j+1] {
				seq[j], seq[j+1] = seq[j+1], seq[j]
				swaps++
			}
		}
	}
	return swaps
}

// TestClifford_SwapParity_PropertyTest exhaustively tests all blade pairs in d <= 6
// against the ground-truth naive bubble-sort reordering count.
func TestClifford_SwapParity_PropertyTest(t *testing.T) {
	d := 6
	limit := 1 << d

	for u := 0; u < limit; u++ {
		for v := 0; v < limit; v++ {
			u64 := uint64(u)
			v64 := uint64(v)

			// If they share basis vectors, let's test only disjoint or distinct pairs for pure swap
			expectedSwaps := naiveSwapCount(u64, v64)
			expectedSign := 1
			if expectedSwaps%2 != 0 {
				expectedSign = -1
			}

			actualSign := computeSwapSign(u64, v64)
			if actualSign != expectedSign {
				t.Fatalf("Swap sign mismatch for u=%b, v=%b: expected=%d, actual=%d (expected swaps=%d)",
					u, v, expectedSign, actualSign, expectedSwaps)
			}
		}
	}
}

func TestClifford_DefiningRelations_Cl30(t *testing.T) {
	// Cl(3, 0, 0): Euclidean 3D
	m, err := NewMetric(3, 0, 0)
	if err != nil {
		t.Fatalf("NewMetric failed: %v", err)
	}

	e1 := NewBasisBlade(m, 0) // mask = 1 (bit 0)
	e2 := NewBasisBlade(m, 1) // mask = 2 (bit 1)
	e3 := NewBasisBlade(m, 2) // mask = 4 (bit 2)

	one := NewScalarMultivector(m, ast.NewRationalFromBigRat(big.NewRat(1, 1)))
	negOne := NewScalarMultivector(m, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))

	// e1^2 == 1, e2^2 == 1, e3^2 == 1
	e1sq, _ := e1.GeometricProduct(e1)
	if !e1sq.Equals(one) {
		t.Fatalf("e1^2 should equal 1, got: %s", e1sq)
	}

	// e1 * e2 == - e2 * e1
	e1e2, _ := e1.GeometricProduct(e2)
	e2e1, _ := e2.GeometricProduct(e1)
	negE2e1, _ := negOne.GeometricProduct(e2e1)

	if !e1e2.Equals(negE2e1) {
		t.Fatalf("e1*e2 should equal -e2*e1, got e1e2=%s, e2e1=%s", e1e2, e2e1)
	}

	// e1 * e2 * e3 is pseudoscalar I
	e2e3, _ := e2.GeometricProduct(e3)
	iBlade, _ := e1.GeometricProduct(e2e3)

	// I^2 = (e1 e2 e3)^2 = -1 in Cl(3,0)
	iSq, _ := iBlade.GeometricProduct(iBlade)
	if !iSq.Equals(negOne) {
		t.Fatalf("I^2 in Cl(3,0) should equal -1, got: %s", iSq)
	}
}

func TestClifford_DegeneratePGA_Cl301(t *testing.T) {
	// 3D PGA: Cl(3, 0, 1) -> 3 positive bases, 0 negative, 1 null base (e0^2 = 0)
	// Base ordering: positive first (e1, e2, e3), then negative, then null (e0)
	m, err := NewMetric(3, 0, 1)
	if err != nil {
		t.Fatalf("NewMetric failed: %v", err)
	}

	e1 := NewBasisBlade(m, 0)
	e2 := NewBasisBlade(m, 1)
	e3 := NewBasisBlade(m, 2)
	e0 := NewBasisBlade(m, 3) // null base

	// 1. e0^2 == 0
	e0sq, _ := e0.GeometricProduct(e0)
	if !e0sq.IsZero() {
		t.Fatalf("e0^2 in PGA should equal 0, got: %s", e0sq)
	}

	// 2. e1 * e0 == -e0 * e1
	e1e0, _ := e1.GeometricProduct(e0)
	e0e1, _ := e0.GeometricProduct(e1)
	negOne := NewScalarMultivector(m, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
	negE0e1, _ := negOne.GeometricProduct(e0e1)
	if !e1e0.Equals(negE0e1) {
		t.Fatalf("e1*e0 should equal -e0*e1, got e1e0=%s, e0e1=%s", e1e0, e0e1)
	}

	// 3. Pseudoscalar I = e1 ^ e2 ^ e3 ^ e0
	// I^2 == 0 because it contains e0
	iBlade, _ := e1.GeometricProduct(e2)
	iBlade, _ = iBlade.GeometricProduct(e3)
	iBlade, _ = iBlade.GeometricProduct(e0)

	iSq, _ := iBlade.GeometricProduct(iBlade)
	if !iSq.IsZero() {
		t.Fatalf("Pseudoscalar I^2 in PGA should equal 0, got: %s", iSq)
	}

	// 4. Poincaré Dual of e0: u ^ PoincareDual(u) == I
	dualE0 := e0.PoincareDual()
	wedgeFull, _ := e0.Wedge(dualE0)
	if !wedgeFull.Equals(iBlade) {
		t.Fatalf("PoincareDual must satisfy u ^ dual(u) == I, got u ^ dual(u) = %s, expected I = %s", wedgeFull, iBlade)
	}

	// 5. Inversion of zero divisor e0 should fail gracefully
	_, errInv := e0.TryInverse()
	if errInv == nil {
		t.Fatalf("TryInverse on null vector e0 should fail, but succeeded")
	}
}

func TestClifford_Operations_WedgeContraction(t *testing.T) {
	m, _ := NewMetric(3, 0, 0)
	e1 := NewBasisBlade(m, 0)
	e2 := NewBasisBlade(m, 1)

	// Wedge: e1 ^ e1 == 0, e1 ^ e2 == e1e2
	wZero, _ := e1.Wedge(e1)
	if !wZero.IsZero() {
		t.Fatalf("e1 ^ e1 should be 0")
	}

	w12, _ := e1.Wedge(e2)
	e1e2, _ := e1.GeometricProduct(e2)
	if !w12.Equals(e1e2) {
		t.Fatalf("e1 ^ e2 should equal e1*e2 for orthogonal vectors")
	}

	// Left contraction:
	// For grade 1 vectors e1, e2: e1 ⌟ e2 = e1 . e2 = delta_12
	lc11, _ := e1.LeftContract(e1)
	one := NewScalarMultivector(m, ast.NewRationalFromBigRat(big.NewRat(1, 1)))
	if !lc11.Equals(one) {
		t.Fatalf("e1 ⌟ e1 should equal 1, got: %s", lc11)
	}

	lc12, _ := e1.LeftContract(e2)
	if !lc12.IsZero() {
		t.Fatalf("e1 ⌟ e2 should equal 0, got: %s", lc12)
	}

	// Inner product: e1 . e1 == 1, e1 . e2 == 0
	dot11, _ := e1.InnerProduct(e1)
	if !dot11.Equals(one) {
		t.Fatalf("e1 . e1 should equal 1, got: %s", dot11)
	}

	dot12, _ := e1.InnerProduct(e2)
	if !dot12.IsZero() {
		t.Fatalf("e1 . e2 should equal 0, got: %s", dot12)
	}
}

func TestClifford_AssociativityAndInvolutions(t *testing.T) {
	m, _ := NewMetric(1, 3, 0) // Spacetime algebra Cl(1, 3)
	e0 := NewBasisBlade(m, 0) // positive
	e1 := NewBasisBlade(m, 1) // negative
	e2 := NewBasisBlade(m, 2) // negative

	// A = e0 + 2*e1
	// B = e1 - e2
	// C = 3*e0 + e1^e2
	a, _ := e0.Add(e1.Scale(ast.NewRationalFromBigRat(big.NewRat(2, 1))))
	b, _ := e1.Sub(e2)
	e1e2, _ := e1.GeometricProduct(e2)
	c, _ := e0.Scale(ast.NewRationalFromBigRat(big.NewRat(3, 1))).Add(e1e2)

	// (A * B) * C == A * (B * C)
	ab, _ := a.GeometricProduct(b)
	abc1, _ := ab.GeometricProduct(c)

	bc, _ := b.GeometricProduct(c)
	abc2, _ := a.GeometricProduct(bc)

	if !abc1.Equals(abc2) {
		t.Fatalf("Associativity failed in Cl(1, 3):\n(AB)C = %s\nA(BC) = %s", abc1, abc2)
	}

	// Involutions:
	// Reverse(e1 * e2) = e2 * e1 = - e1 * e2
	revE1E2 := e1e2.Reverse()
	negOne := NewScalarMultivector(m, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
	negE1E2, _ := negOne.GeometricProduct(e1e2)
	if !revE1E2.Equals(negE1E2) {
		t.Fatalf("Reverse(e1e2) should equal -e1e2, got: %s", revE1E2)
	}
}
