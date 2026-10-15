package units

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// EvalUnitExpr evaluates an AST node representing a unit expression (e.g. m, m/s, kg*m/s^2, km/h).
// It returns the combined rational scale factor and the resulting Q^7 dimension vector.
func EvalUnitExpr(node ast.Node) (*big.Rat, ast.DimensionVector, error) {
	if node == nil {
		return nil, ast.NewDimensionVector(), fmt.Errorf("nil unit expression")
	}

	switch v := node.(type) {
	case *ast.VarNode:
		return ResolveUnitSymbol(v.Name)

	case *ast.RationalNode:
		// Scalar coefficient inside unit expression (e.g., 1 in 1/s)
		return new(big.Rat).Set(v.Val), ast.NewDimensionVector(), nil

	case *ast.MulNode:
		totalScale := big.NewRat(1, 1)
		totalDims := ast.NewDimensionVector()
		for _, factor := range v.Factors {
			s, d, err := EvalUnitExpr(factor)
			if err != nil {
				return nil, ast.NewDimensionVector(), err
			}
			totalScale.Mul(totalScale, s)
			totalDims = totalDims.Add(d)
		}
		return totalScale, totalDims, nil

	case *ast.PowNode:
		baseScale, baseDims, err := EvalUnitExpr(v.Base)
		if err != nil {
			return nil, ast.NewDimensionVector(), err
		}

		expRat, ok := v.Exp.(*ast.RationalNode)
		if !ok {
			return nil, ast.NewDimensionVector(), fmt.Errorf("unit exponent must be an exact rational, got %s", v.Exp.String())
		}

		// Compute baseScale ^ expRat
		// For integer exponents:
		if !expRat.Val.IsInt() {
			return nil, ast.NewDimensionVector(), fmt.Errorf("non-integer unit scaling factor not supported for compound units: %s", expRat.Val.String())
		}
		expInt := expRat.Val.Num().Int64()
		scaled := ratPowInt(baseScale, expInt)

		// Dimension vector is scaled by exact rational (fractional dimensions are fully supported)
		scaledDims := baseDims.ScalarMul(expRat.Val)
		return scaled, scaledDims, nil

	case *ast.UnaryOpNode:
		if v.Op == "-" {
			// e.g. -1 as exponent
			s, d, err := EvalUnitExpr(v.Expr)
			if err != nil {
				return nil, ast.NewDimensionVector(), err
			}
			s.Neg(s)
			return s, d, nil
		}
		return nil, ast.NewDimensionVector(), fmt.Errorf("unsupported unary operator in unit expression: %s", v.Op)

	default:
		return nil, ast.NewDimensionVector(), fmt.Errorf("unsupported node in unit expression: %T (%s)", node, node.String())
	}
}

// ratPowInt computes r^exp for integer exp.
func ratPowInt(r *big.Rat, exp int64) *big.Rat {
	if exp == 0 {
		return big.NewRat(1, 1)
	}
	if exp < 0 {
		inv := new(big.Rat).Inv(r)
		return ratPowInt(inv, -exp)
	}
	num := new(big.Int).Exp(r.Num(), big.NewInt(exp), nil)
	den := new(big.Int).Exp(r.Denom(), big.NewInt(exp), nil)
	return new(big.Rat).SetFrac(num, den)
}
