package units_test

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/units"
)

func TestEvalUnitExpr(t *testing.T) {
	// 単一単位: [m]
	mNode := ast.NewVar("m")
	scaleM, dimsM, err := units.EvalUnitExpr(mNode)
	if err != nil {
		t.Fatalf("unexpected error on m: %v", err)
	}
	if scaleM.Cmp(big.NewRat(1, 1)) != 0 || dimsM[ast.DimLength].Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("unexpected result for m: scale=%s, dims=%s", scaleM, dimsM)
	}

	// 合成単位の除算: m / s
	sNode := ast.NewVar("s")
	powS, err := ast.NewPow(sNode, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
	if err != nil {
		t.Fatalf("failed to create PowNode: %v", err)
	}
	divNode := ast.NewMul([]ast.Node{mNode, powS})
	scaleVel, dimsVel, err := units.EvalUnitExpr(divNode)
	if err != nil {
		t.Fatalf("unexpected error on m/s: %v", err)
	}
	if scaleVel.Cmp(big.NewRat(1, 1)) != 0 ||
		dimsVel[ast.DimLength].Cmp(big.NewRat(1, 1)) != 0 ||
		dimsVel[ast.DimTime].Cmp(big.NewRat(-1, 1)) != 0 {
		t.Fatalf("unexpected result for m/s: scale=%s, dims=%s", scaleVel, dimsVel)
	}

	// 単位換算を含む式: km / h
	// scale should be 1000 / 3600 = 5 / 18
	kmNode := ast.NewVar("km")
	hNode := ast.NewVar("h")
	powH, err := ast.NewPow(hNode, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
	if err != nil {
		t.Fatalf("failed to create PowNode: %v", err)
	}
	kmHNode := ast.NewMul([]ast.Node{kmNode, powH})
	scaleKmH, dimsKmH, err := units.EvalUnitExpr(kmHNode)
	if err != nil {
		t.Fatalf("unexpected error on km/h: %v", err)
	}
	expectedScale := big.NewRat(5, 18)
	if scaleKmH.Cmp(expectedScale) != 0 {
		t.Fatalf("expected scale 5/18 for km/h, got %s", scaleKmH)
	}
	if dimsKmH[ast.DimLength].Cmp(big.NewRat(1, 1)) != 0 || dimsKmH[ast.DimTime].Cmp(big.NewRat(-1, 1)) != 0 {
		t.Fatalf("unexpected dims for km/h: %s", dimsKmH)
	}
}
