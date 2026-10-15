package calc_test

import (
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/parser"
)

func evalExpr(t *testing.T, input string) (ast.Node, error) {
	t.Helper()
	parsed, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse error on '%s': %v", input, err)
	}
	return calc.Eval(parsed)
}

func TestEvalQuantity_ArithmeticOperations(t *testing.T) {
	// 加算 (同一次元・単位一致): 10 [m] + 5 [m] == 15 [m]
	res, err := evalExpr(t, "10 [m] + 5 [m]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calc.Format(res) != "15 [m]" {
		t.Fatalf("expected '15 [m]', got '%s'", calc.Format(res))
	}

	// 減算 (単位換算を伴う同一次元): 1 [km] - 200 [m] == 800 [m] (or 4/5 [km])
	res, err = evalExpr(t, "1 [km] - 200 [m]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// どちらの表示単位でも値と次元が整合していればよい (800 [m] または 4/5 [km])
	q, ok := res.(*ast.QuantityNode)
	if !ok {
		t.Fatalf("expected QuantityNode, got %T: %s", res, res.String())
	}
	if q.Dimensions[ast.DimLength].Cmp(ast.NewDimensionVector()[ast.DimLength]) == 0 {
		t.Fatalf("expected length dimension, got %s", q.Dimensions)
	}

	// 乗算: 5 [kg] * 2 [m/s^2]
	res, err = evalExpr(t, "5 [kg] * 2 [m/s^2]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q, ok = res.(*ast.QuantityNode)
	if !ok {
		t.Fatalf("expected QuantityNode, got %T", res)
	}
	if q.Dimensions[ast.DimMass].Sign() != 1 || q.Dimensions[ast.DimLength].Sign() != 1 || q.Dimensions[ast.DimTime].Sign() != -1 {
		t.Fatalf("expected force dimensions for 5 [kg] * 2 [m/s^2], got %s", q.Dimensions)
	}

	// 除算: 10 [m] / 2 [s] == 5 [m/s]
	res, err = evalExpr(t, "10 [m] / 2 [s]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calc.Format(res) != "5 [m/s]" {
		t.Fatalf("expected '5 [m/s]', got '%s'", calc.Format(res))
	}

	// 無次元縮約: 10 [m] / 2 [m] == 5 (通常数値ノードへ縮約)
	res, err = evalExpr(t, "10 [m] / 2 [m]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, isQ := res.(*ast.QuantityNode); isQ {
		t.Fatalf("expected dimensionless reduction to numeric node, got QuantityNode: %s", res.String())
	}
	if calc.Format(res) != "5" {
		t.Fatalf("expected '5', got '%s'", calc.Format(res))
	}
}

func TestEvalQuantity_DimensionMismatchError(t *testing.T) {
	// 10 [m] + 5 [kg] は次元不整合エラー
	_, err := evalExpr(t, "10 [m] + 5 [kg]")
	if err == nil {
		t.Fatalf("expected dimension mismatch error for 10 [m] + 5 [kg], got nil")
	}
}

func TestEvalQuantity_TranscendentalGuard(t *testing.T) {
	// 初等超越関数に有次元の引数を渡すとエラー
	_, err := evalExpr(t, "sin(10 [m])")
	if err == nil {
		t.Fatalf("expected transcendental error for sin(10 [m]), got nil")
	}

	_, err = evalExpr(t, "exp(5 [s])")
	if err == nil {
		t.Fatalf("expected transcendental error for exp(5 [s]), got nil")
	}
}

func TestEvalQuantity_RelationalOperators(t *testing.T) {
	// 1000 [m] == 1 [km] -> 1 (true)
	res, err := evalExpr(t, "1000 [m] == 1 [km]")
	if err != nil {
		t.Fatalf("unexpected error on 1000 [m] == 1 [km]: %v", err)
	}
	if calc.Format(res) != "1" {
		t.Fatalf("expected 1 (true), got %s", calc.Format(res))
	}

	// 1 [h] == 60 [min] -> 1 (true)
	res, err = evalExpr(t, "1 [h] == 60 [min]")
	if err != nil {
		t.Fatalf("unexpected error on 1 [h] == 60 [min]: %v", err)
	}
	if calc.Format(res) != "1" {
		t.Fatalf("expected 1 (true), got %s", calc.Format(res))
	}

	// 1 [m] == 1 [kg] -> 0 (false)
	res, err = evalExpr(t, "1 [m] == 1 [kg]")
	if err != nil {
		t.Fatalf("unexpected error on 1 [m] == 1 [kg]: %v", err)
	}
	if calc.Format(res) != "0" {
		t.Fatalf("expected 0 (false), got %s", calc.Format(res))
	}
}

func TestEvalQuantity_ConvertUnit(t *testing.T) {
	// convert_unit(100 [km/h], "m/s") -> 250/9 [m/s]
	res, err := evalExpr(t, "convert_unit(100 [km/h], m/s)")
	if err != nil {
		t.Fatalf("unexpected error on convert_unit: %v", err)
	}
	if calc.Format(res) != "250/9 [m/s]" {
		t.Fatalf("expected '250/9 [m/s]', got '%s'", calc.Format(res))
	}

	// convert_unit((1/4) [h], min) -> 15 [min] (ユズコの例)
	res, err = evalExpr(t, "convert_unit((1/4) [h], min)")
	if err != nil {
		t.Fatalf("unexpected error on convert_unit((1/4) [h], min): %v", err)
	}
	if calc.Format(res) != "15 [min]" {
		t.Fatalf("expected '15 [min]', got '%s'", calc.Format(res))
	}

	// 不適合な単位換算: convert_unit(10 [m], kg) -> エラー
	_, err = evalExpr(t, "convert_unit(10 [m], kg)")
	if err == nil {
		t.Fatalf("expected incompatible unit conversion error, got nil")
	}
}

func TestEvalQuantity_Verify(t *testing.T) {
	expr, err := parser.Parse("convert_unit(100 [km/h], m/s)")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	res, err := calc.Eval(expr)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	cert, err := calc.VerifyComputation(expr, res, calc.NewEnv())
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if !cert.IsVerified {
		t.Fatalf("expected verified certificate, got unverified: %s", cert.Details)
	}
	if cert.Domain != calc.DomainUnits {
		t.Fatalf("expected domain units, got %s", cert.Domain)
	}
}

