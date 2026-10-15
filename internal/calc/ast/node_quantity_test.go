package ast_test

import (
	"math/big"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

func TestDimensionVector_Arithmetic(t *testing.T) {
	// 長さ: [1, 0, 0, 0, 0, 0, 0]
	d1 := ast.NewDimensionVector()
	d1[ast.DimLength] = big.NewRat(1, 1)

	// 時間: [0, 0, 1, 0, 0, 0, 0]
	d2 := ast.NewDimensionVector()
	d2[ast.DimTime] = big.NewRat(1, 1)

	// 速度: 長さ - 時間 = [1, 0, -1, 0, 0, 0, 0]
	velocity := d1.Sub(d2)
	if velocity[ast.DimLength].Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("expected DimLength == 1, got %s", velocity[ast.DimLength])
	}
	if velocity[ast.DimTime].Cmp(big.NewRat(-1, 1)) != 0 {
		t.Fatalf("expected DimTime == -1, got %s", velocity[ast.DimTime])
	}

	// 加速度: 速度 - 時間 = [1, 0, -2, 0, 0, 0, 0]
	accel := velocity.Sub(d2)
	if accel[ast.DimTime].Cmp(big.NewRat(-2, 1)) != 0 {
		t.Fatalf("expected DimTime == -2, got %s", accel[ast.DimTime])
	}

	// 累乗 (平方根): [2, 0, 0, 0, 0, 0, 0] * (1/2) = [1, 0, 0, 0, 0, 0, 0]
	area := ast.NewDimensionVector()
	area[ast.DimLength] = big.NewRat(2, 1)
	sqrtArea := area.ScalarMul(big.NewRat(1, 2))
	if !sqrtArea.Equal(d1) {
		t.Fatalf("expected sqrt(area) == length, got %s", sqrtArea)
	}

	// 無次元判定
	zeroDim := ast.NewDimensionVector()
	if !zeroDim.IsZero() {
		t.Fatalf("expected zeroDim.IsZero() == true")
	}
	if d1.IsZero() {
		t.Fatalf("expected d1.IsZero() == false")
	}
}

func TestQuantityNode_Basics(t *testing.T) {
	val := ast.NewRationalFromBigRat(big.NewRat(10, 1))
	scale := big.NewRat(1, 1)
	dims := ast.NewDimensionVector()
	dims[ast.DimLength] = big.NewRat(1, 1)

	q1 := ast.NewQuantityNode(val, scale, dims, "m")
	if q1.Type() != ast.NodeQuantity {
		t.Fatalf("expected NodeQuantity, got %v", q1.Type())
	}
	if q1.String() != "10 [m]" {
		t.Fatalf("expected '10 [m]', got '%s'", q1.String())
	}

	// 同値比較
	q2 := ast.NewQuantityNode(ast.NewRationalFromBigRat(big.NewRat(10, 1)), big.NewRat(1, 1), dims, "m")
	if !q1.Equal(q2) {
		t.Fatalf("expected q1.Equal(q2) == true")
	}

	// 単位違い比較
	timeDims := ast.NewDimensionVector()
	timeDims[ast.DimTime] = big.NewRat(1, 1)
	q3 := ast.NewQuantityNode(val, scale, timeDims, "s")
	if q1.Equal(q3) {
		t.Fatalf("expected q1.Equal(q3) == false")
	}
}
