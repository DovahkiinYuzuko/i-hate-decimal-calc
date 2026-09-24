package galois

import (
	"math/big"
	"testing"
)

func TestIdentifyDegree3(t *testing.T) {
	// x^3 - 2 = x^3 + 0*x^2 + 0*x - 2 => S3
	c1 := []*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(-2)}
	res1, err := IdentifyGaloisGroupIntegerPoly(c1)
	if err != nil {
		t.Fatalf("unexpected error for x^3 - 2: %v", err)
	}
	if res1.Group.Name != "S3" {
		t.Errorf("expected S3 for x^3 - 2, got %s", res1.Group.Name)
	}
	if !res1.IsSolvable {
		t.Errorf("expected x^3 - 2 to be solvable")
	}

	// x^3 - 3*x + 1 = x^3 + 0*x^2 - 3*x + 1 => A3 (disc = 81 = 9^2)
	c2 := []*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(-3), big.NewInt(1)}
	res2, err := IdentifyGaloisGroupIntegerPoly(c2)
	if err != nil {
		t.Fatalf("unexpected error for x^3 - 3x + 1: %v", err)
	}
	if res2.Group.Name != "A3" {
		t.Errorf("expected A3 for x^3 - 3x + 1, got %s", res2.Group.Name)
	}
	if !res2.IsSolvable {
		t.Errorf("expected x^3 - 3x + 1 to be solvable")
	}
}

func TestIdentifyDegree4(t *testing.T) {
	// x^4 - 2 = x^4 + 0*x^3 + 0*x^2 + 0*x - 2 => D4
	c1 := []*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(-2)}
	res1, err := IdentifyGaloisGroupIntegerPoly(c1)
	if err != nil {
		t.Fatalf("unexpected error for x^4 - 2: %v", err)
	}
	if res1.Group.Name != "D4" {
		t.Errorf("expected D4 for x^4 - 2, got %s", res1.Group.Name)
	}
	if !res1.IsSolvable {
		t.Errorf("expected x^4 - 2 to be solvable")
	}

	// x^4 - 10*x^2 + 1 => V4
	c2 := []*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(-10), big.NewInt(0), big.NewInt(1)}
	res2, err := IdentifyGaloisGroupIntegerPoly(c2)
	if err != nil {
		t.Fatalf("unexpected error for x^4 - 10x^2 + 1: %v", err)
	}
	if res2.Group.Name != "V4" {
		t.Errorf("expected V4 for x^4 - 10x^2 + 1, got %s", res2.Group.Name)
	}
	if !res2.IsSolvable {
		t.Errorf("expected x^4 - 10x^2 + 1 to be solvable")
	}
}

func TestIdentifyDegree5(t *testing.T) {
	// x^5 - 4*x + 2: Classic Abel-Ruffini non-solvable quintic => S5
	// Eisenstein at 2 => irreducible.
	c1 := []*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(-4), big.NewInt(2)}
	res1, err := IdentifyGaloisGroupIntegerPoly(c1)
	if err != nil {
		t.Fatalf("unexpected error for x^5 - 4x + 2: %v", err)
	}
	if res1.Group.Name != "S5" {
		t.Errorf("expected S5 for x^5 - 4x + 2, got %s", res1.Group.Name)
	}
	if res1.IsSolvable {
		t.Errorf("expected x^5 - 4x + 2 to be non-solvable by Abel-Ruffini")
	}

	// x^5 - 2: Frobenius group F20 (solvable by radicals: sqrt[5]{2})
	c2 := []*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(-2)}
	res2, err := IdentifyGaloisGroupIntegerPoly(c2)
	if err != nil {
		t.Fatalf("unexpected error for x^5 - 2: %v", err)
	}
	if res2.Group.Name != "F20" {
		t.Errorf("expected F20 for x^5 - 2, got %s", res2.Group.Name)
	}
	if !res2.IsSolvable {
		t.Errorf("expected x^5 - 2 to be solvable by radicals")
	}

	// x^5 - 5*x + 12: D5 (Solvable quintic with square discriminant)
	c3 := []*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(-5), big.NewInt(12)}
	res3, err := IdentifyGaloisGroupIntegerPoly(c3)
	if err != nil {
		t.Fatalf("unexpected error for x^5 - 5x + 12: %v", err)
	}
	if res3.Group.Name != "D5" {
		t.Errorf("expected D5 for x^5 - 5x + 12, got %s", res3.Group.Name)
	}
	if !res3.IsSolvable {
		t.Errorf("expected x^5 - 5x + 12 to be solvable")
	}

	// x^5 + 20*x + 16: A5 (Classic Keith Conrad example, non-solvable with square discriminant)
	c4 := []*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(20), big.NewInt(16)}
	res4, err := IdentifyGaloisGroupIntegerPoly(c4)
	if err != nil {
		t.Fatalf("unexpected error for x^5 + 20x + 16: %v", err)
	}
	if res4.Group.Name != "A5" {
		t.Errorf("expected A5 for x^5 + 20x + 16, got %s", res4.Group.Name)
	}
	if res4.IsSolvable {
		t.Errorf("expected x^5 + 20x + 16 to be non-solvable by Abel-Ruffini")
	}
}
