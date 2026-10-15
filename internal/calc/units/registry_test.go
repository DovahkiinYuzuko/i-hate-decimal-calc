package units_test

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/units"
)

func TestUnitsRegistry_BaseUnits(t *testing.T) {
	cases := []struct {
		sym    string
		dimIdx int
	}{
		{"m", ast.DimLength},
		{"kg", ast.DimMass},
		{"s", ast.DimTime},
		{"A", ast.DimCurrent},
		{"K", ast.DimTemperature},
		{"mol", ast.DimAmount},
		{"cd", ast.DimLuminosity},
	}

	for _, tc := range cases {
		def, ok := units.FindUnit(tc.sym)
		if !ok {
			t.Fatalf("unit %s not found", tc.sym)
		}
		if def.Scale.Cmp(big.NewRat(1, 1)) != 0 {
			t.Fatalf("expected scale 1 for base unit %s, got %s", tc.sym, def.Scale)
		}
		if def.Dims[tc.dimIdx].Cmp(big.NewRat(1, 1)) != 0 {
			t.Fatalf("expected dim[%d] == 1 for %s, got %s", tc.dimIdx, tc.sym, def.Dims[tc.dimIdx])
		}
	}
}

func TestUnitsRegistry_DerivedUnits(t *testing.T) {
	// N = kg * m / s^2
	nDef, ok := units.FindUnit("N")
	if !ok {
		t.Fatalf("unit N not found")
	}
	if nDef.Dims[ast.DimMass].Cmp(big.NewRat(1, 1)) != 0 ||
		nDef.Dims[ast.DimLength].Cmp(big.NewRat(1, 1)) != 0 ||
		nDef.Dims[ast.DimTime].Cmp(big.NewRat(-2, 1)) != 0 {
		t.Fatalf("unexpected dimensions for N: %s", nDef.Dims)
	}

	// J = N * m = kg * m^2 / s^2
	jDef, ok := units.FindUnit("J")
	if !ok {
		t.Fatalf("unit J not found")
	}
	if jDef.Dims[ast.DimLength].Cmp(big.NewRat(2, 1)) != 0 {
		t.Fatalf("expected length^2 for J, got %s", jDef.Dims)
	}

	// W = J / s = kg * m^2 / s^3
	wDef, ok := units.FindUnit("W")
	if !ok {
		t.Fatalf("unit W not found")
	}
	if wDef.Dims[ast.DimTime].Cmp(big.NewRat(-3, 1)) != 0 {
		t.Fatalf("expected time^-3 for W, got %s", wDef.Dims)
	}
}

func TestUnitsRegistry_AcceptedNonSI(t *testing.T) {
	// min = 60 s
	minDef, ok := units.FindUnit("min")
	if !ok {
		t.Fatalf("unit min not found")
	}
	if minDef.Scale.Cmp(big.NewRat(60, 1)) != 0 {
		t.Fatalf("expected scale 60 for min, got %s", minDef.Scale)
	}
	if minDef.Dims[ast.DimTime].Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("expected time dimension for min, got %s", minDef.Dims)
	}

	// h = 3600 s
	hDef, ok := units.FindUnit("h")
	if !ok {
		t.Fatalf("unit h not found")
	}
	if hDef.Scale.Cmp(big.NewRat(3600, 1)) != 0 {
		t.Fatalf("expected scale 3600 for h, got %s", hDef.Scale)
	}
}

func TestUnitsRegistry_ResolvePrefix(t *testing.T) {
	// km -> scale = 1000, length = 1
	scale, dims, err := units.ResolveUnitSymbol("km")
	if err != nil {
		t.Fatalf("unexpected error resolving km: %v", err)
	}
	if scale.Cmp(big.NewRat(1000, 1)) != 0 {
		t.Fatalf("expected scale 1000 for km, got %s", scale)
	}
	if dims[ast.DimLength].Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("expected length 1 for km, got %s", dims)
	}

	// ms -> scale = 1/1000, time = 1
	scaleMs, dimsMs, err := units.ResolveUnitSymbol("ms")
	if err != nil {
		t.Fatalf("unexpected error resolving ms: %v", err)
	}
	if scaleMs.Cmp(big.NewRat(1, 1000)) != 0 {
		t.Fatalf("expected scale 1/1000 for ms, got %s", scaleMs)
	}
	if dimsMs[ast.DimTime].Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("expected time 1 for ms, got %s", dimsMs)
	}

	// kN -> scale = 1000, dimension of N
	scaleKN, dimsKN, err := units.ResolveUnitSymbol("kN")
	if err != nil {
		t.Fatalf("unexpected error resolving kN: %v", err)
	}
	if scaleKN.Cmp(big.NewRat(1000, 1)) != 0 {
		t.Fatalf("expected scale 1000 for kN, got %s", scaleKN)
	}
	if dimsKN[ast.DimMass].Cmp(big.NewRat(1, 1)) != 0 || dimsKN[ast.DimTime].Cmp(big.NewRat(-2, 1)) != 0 {
		t.Fatalf("unexpected dimensions for kN: %s", dimsKN)
	}
}
