package poly

import (
	"math/big"
	"testing"
)

func TestHenselLiftRoot(t *testing.T) {
	// f(x) = x^2 - 2 = -2 + 0*x + 1*x^2
	fCoeffs := []*big.Int{big.NewInt(-2), big.NewInt(0), big.NewInt(1)}
	p := big.NewInt(7)
	r0 := big.NewInt(3) // 3^2 = 9 = 2 mod 7

	targetK := 3 // 7^3 = 343
	r, err := HenselLiftRoot(fCoeffs, p, r0, targetK)
	if err != nil {
		t.Fatalf("HenselLiftRoot failed: %v", err)
	}

	p3 := new(big.Int).Exp(p, big.NewInt(int64(targetK)), nil)
	// r^2 - 2 mod 343 must be 0
	val := evalPolyMod(fCoeffs, r, p3)
	if val.Sign() != 0 {
		t.Fatalf("expected f(r) = 0 mod 343, got %s for r=%s", val.String(), r.String())
	}
}

func TestHenselLiftFactors(t *testing.T) {
	// f(x) = x^4 + 3*x^2 + 2 = (x^2 + 1)(x^2 + 2)
	// f = [2, 0, 3, 0, 1]
	fCoeffs := []*big.Int{big.NewInt(2), big.NewInt(0), big.NewInt(3), big.NewInt(0), big.NewInt(1)}

	// mod 5:
	// g0 = x^2 + 1 = [1, 0, 1]
	// h0 = x^2 + 2 = [2, 0, 1]
	g0 := []*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(1)}
	h0 := []*big.Int{big.NewInt(2), big.NewInt(0), big.NewInt(1)}
	p := big.NewInt(5)

	targetK := 3 // 5^3 = 125
	g, h, err := HenselLiftFactors(fCoeffs, g0, h0, p, targetK)
	if err != nil {
		t.Fatalf("HenselLiftFactors failed: %v", err)
	}

	p3 := new(big.Int).Exp(p, big.NewInt(int64(targetK)), nil)

	// g * h mod 125 must equal f mod 125
	gh := polyMulMod(g, h, p3)
	fMod := polyMod(fCoeffs, p3)

	degGH := polyDegree(gh)
	degF := polyDegree(fMod)
	if degGH != degF {
		t.Fatalf("degree mismatch: deg(g*h)=%d, deg(f)=%d", degGH, degF)
	}
	for i := 0; i <= degGH; i++ {
		if gh[i].Cmp(fMod[i]) != 0 {
			t.Fatalf("coeff %d mismatch: got %s, expected %s", i, gh[i].String(), fMod[i].String())
		}
	}

	// Centered modulo should recover original factors [1, 0, 1] and [2, 0, 1]
	cg := CenteredModulo(g, p3)
	ch := CenteredModulo(h, p3)

	if polyDegree(cg) != 2 || cg[0].Int64() != 1 || cg[2].Int64() != 1 {
		t.Fatalf("recovered g does not match x^2+1: %v", cg)
	}
	if polyDegree(ch) != 2 || ch[0].Int64() != 2 || ch[2].Int64() != 1 {
		t.Fatalf("recovered h does not match x^2+2: %v", ch)
	}
}

func TestLandauMignotteBound(t *testing.T) {
	// f(x) = x^4 + 3*x^2 + 2
	fCoeffs := []*big.Int{big.NewInt(2), big.NewInt(0), big.NewInt(3), big.NewInt(0), big.NewInt(1)}
	bound := LandauMignotteBound(fCoeffs, 2)
	if bound.Cmp(big.NewInt(1)) <= 0 {
		t.Fatalf("expected bound > 1, got %s", bound.String())
	}
}

func TestFactorHenselDegree2(t *testing.T) {
	// f(x) = x^4 + 3*x^2 + 2 = (x^2 + 1)(x^2 + 2)
	fCoeffs := []*big.Int{big.NewInt(2), big.NewInt(0), big.NewInt(3), big.NewInt(0), big.NewInt(1)}
	g, h, ok := FactorHenselDegree2(fCoeffs)
	if !ok {
		t.Fatal("FactorHenselDegree2 failed to factor x^4 + 3*x^2 + 2")
	}

	if polyDegree(g) != 2 || polyDegree(h) != 2 {
		t.Fatalf("expected degrees 2 and 2, got %d and %d", polyDegree(g), polyDegree(h))
	}
}

