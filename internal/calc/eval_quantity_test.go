package calc_test

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/parser"
)

func TestEvalQuantity_ParsingAndConstruction(t *testing.T) {
	// 10 [m]
	parsed, err := parser.Parse("10 [m]")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	evaled, err := calc.Eval(parsed)
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}

	q, ok := evaled.(*ast.QuantityNode)
	if !ok {
		t.Fatalf("expected QuantityNode, got %T: %s", evaled, evaled.String())
	}

	if q.Value.String() != "10" {
		t.Fatalf("expected value 10, got %s", q.Value.String())
	}
	if q.Scale.Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("expected scale 1, got %s", q.Scale)
	}
	if q.Dimensions[ast.DimLength].Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("expected length dimension 1, got %s", q.Dimensions)
	}
	if calc.Format(q) != "10 [m]" {
		t.Fatalf("expected format '10 [m]', got '%s'", calc.Format(q))
	}
}

func TestEvalQuantity_PrefixParsing(t *testing.T) {
	// 5 [km]
	parsed, err := parser.Parse("5 [km]")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	evaled, err := calc.Eval(parsed)
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}

	q, ok := evaled.(*ast.QuantityNode)
	if !ok {
		t.Fatalf("expected QuantityNode, got %T: %s", evaled, evaled.String())
	}

	if q.Scale.Cmp(big.NewRat(1000, 1)) != 0 {
		t.Fatalf("expected scale 1000 for km, got %s", q.Scale)
	}
	if q.Dimensions[ast.DimLength].Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("expected length dimension 1 for km, got %s", q.Dimensions)
	}
}

func TestEvalQuantity_CompoundUnit(t *testing.T) {
	// 20 [m/s]
	parsed, err := parser.Parse("20 [m/s]")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	evaled, err := calc.Eval(parsed)
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}

	q, ok := evaled.(*ast.QuantityNode)
	if !ok {
		t.Fatalf("expected QuantityNode, got %T: %s", evaled, evaled.String())
	}

	if q.Dimensions[ast.DimLength].Cmp(big.NewRat(1, 1)) != 0 ||
		q.Dimensions[ast.DimTime].Cmp(big.NewRat(-1, 1)) != 0 {
		t.Fatalf("expected [m/s] dimensions, got %s", q.Dimensions)
	}
}
