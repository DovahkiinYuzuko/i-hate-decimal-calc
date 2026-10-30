package calc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/units"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterHandler("convert_unit", handleConvertUnit)
	RegisterHandler("unit", handleUnitFunc)
}

// normalizeDimensionless unboxes a dimensionless quantity into a simplified numeric node.
func normalizeDimensionless(val Node, scale *big.Rat) (Node, error) {
	if scale == nil || scale.Cmp(big.NewRat(1, 1)) == 0 {
		return val, nil
	}
	return simplifyMul([]Node{val, ast.NewRationalFromBigRat(scale)})
}

// EvalQuantityBinary evaluates binary operations (+, -, *, /, ^) involving at least one QuantityNode.
func EvalQuantityBinary(op string, left, right Node) (Node, error) {
	switch op {
	case "+":
		q1, ok1 := left.(*ast.QuantityNode)
		q2, ok2 := right.(*ast.QuantityNode)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("%s", i18n.T("errors.dimension_mismatch", left.String(), right.String()))
		}
		return evalQuantityAdd(q1, q2)

	case "-":
		q1, ok1 := left.(*ast.QuantityNode)
		q2, ok2 := right.(*ast.QuantityNode)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("%s", i18n.T("errors.dimension_mismatch", left.String(), right.String()))
		}
		return evalQuantitySub(q1, q2)

	case "*":
		return evalQuantityMul(left, right)

	case "/":
		return evalQuantityDiv(left, right)

	case "^":
		qLeft, ok := left.(*ast.QuantityNode)
		if !ok {
			return nil, fmt.Errorf("base must be a quantity")
		}
		return evalQuantityPow(qLeft, right)

	default:
		return nil, fmt.Errorf("unsupported binary operator for quantities: %s", op)
	}
}

// evalQuantityAdd adds two quantities with identical dimensions.
func evalQuantityAdd(q1, q2 *ast.QuantityNode) (Node, error) {
	if !q1.Dimensions.Equal(q2.Dimensions) {
		return nil, fmt.Errorf("%s", i18n.T("errors.dimension_mismatch", q1.String(), q2.String()))
	}

	ratio := new(big.Rat).Quo(q2.Scale, q1.Scale)
	ratioNode := ast.NewRationalFromBigRat(ratio)

	scaledV2, err := simplifyMul([]Node{q2.Value, ratioNode})
	if err != nil {
		return nil, err
	}

	sumVal, err := simplifyAdd([]Node{q1.Value, scaledV2})
	if err != nil {
		return nil, err
	}

	return ast.NewQuantityNode(sumVal, q1.Scale, q1.Dimensions, q1.DisplayUnit), nil
}

// evalQuantitySub subtracts two quantities with identical dimensions.
func evalQuantitySub(q1, q2 *ast.QuantityNode) (Node, error) {
	if !q1.Dimensions.Equal(q2.Dimensions) {
		return nil, fmt.Errorf("%s", i18n.T("errors.dimension_mismatch", q1.String(), q2.String()))
	}

	ratio := new(big.Rat).Quo(q2.Scale, q1.Scale)
	ratioNode := ast.NewRationalFromBigRat(ratio)

	scaledV2, err := simplifyMul([]Node{q2.Value, ratioNode})
	if err != nil {
		return nil, err
	}

	negScaledV2, err := simplifyUnaryOp("-", scaledV2)
	if err != nil {
		return nil, err
	}

	diffVal, err := simplifyAdd([]Node{q1.Value, negScaledV2})
	if err != nil {
		return nil, err
	}

	return ast.NewQuantityNode(diffVal, q1.Scale, q1.Dimensions, q1.DisplayUnit), nil
}

// evalQuantityMul multiplies quantities or a quantity and a scalar.
func evalQuantityMul(left, right Node) (Node, error) {
	qLeft, isQLeft := left.(*ast.QuantityNode)
	qRight, isQRight := right.(*ast.QuantityNode)

	if !isQLeft && !isQRight {
		return simplifyMul([]Node{left, right})
	}

	if isQLeft && !isQRight {
		newVal, err := simplifyMul([]Node{qLeft.Value, right})
		if err != nil {
			return nil, err
		}
		return ast.NewQuantityNode(newVal, qLeft.Scale, qLeft.Dimensions, qLeft.DisplayUnit), nil
	}

	if !isQLeft && isQRight {
		newVal, err := simplifyMul([]Node{left, qRight.Value})
		if err != nil {
			return nil, err
		}
		return ast.NewQuantityNode(newVal, qRight.Scale, qRight.Dimensions, qRight.DisplayUnit), nil
	}

	newDims := qLeft.Dimensions.Add(qRight.Dimensions)
	newScale := new(big.Rat).Mul(qLeft.Scale, qRight.Scale)

	newVal, err := simplifyMul([]Node{qLeft.Value, qRight.Value})
	if err != nil {
		return nil, err
	}

	if newDims.IsZero() {
		return normalizeDimensionless(newVal, newScale)
	}

	return ast.NewQuantityNode(newVal, newScale, newDims, ""), nil
}

// evalQuantityDiv divides quantities or a quantity and a scalar.
func evalQuantityDiv(left, right Node) (Node, error) {
	qLeft, isQLeft := left.(*ast.QuantityNode)
	qRight, isQRight := right.(*ast.QuantityNode)

	if !isQLeft && !isQRight {
		invRight, err := simplifyPow(right, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
		if err != nil {
			return nil, err
		}
		return simplifyMul([]Node{left, invRight})
	}

	if isQLeft && !isQRight {
		invRight, err := simplifyPow(right, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
		if err != nil {
			return nil, err
		}
		newVal, err := simplifyMul([]Node{qLeft.Value, invRight})
		if err != nil {
			return nil, err
		}
		return ast.NewQuantityNode(newVal, qLeft.Scale, qLeft.Dimensions, qLeft.DisplayUnit), nil
	}

	if !isQLeft && isQRight {
		invVal, err := simplifyPow(qRight.Value, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
		if err != nil {
			return nil, err
		}
		newVal, err := simplifyMul([]Node{left, invVal})
		if err != nil {
			return nil, err
		}
		invScale := new(big.Rat).Inv(qRight.Scale)
		invDims := qRight.Dimensions.ScalarMul(big.NewRat(-1, 1))
		return ast.NewQuantityNode(newVal, invScale, invDims, ""), nil
	}

	newDims := qLeft.Dimensions.Sub(qRight.Dimensions)
	newScale := new(big.Rat).Quo(qLeft.Scale, qRight.Scale)

	invRightVal, err := simplifyPow(qRight.Value, ast.NewRationalFromBigRat(big.NewRat(-1, 1)))
	if err != nil {
		return nil, err
	}
	newVal, err := simplifyMul([]Node{qLeft.Value, invRightVal})
	if err != nil {
		return nil, err
	}

	if newDims.IsZero() {
		return normalizeDimensionless(newVal, newScale)
	}

	displayUnit := ""
	if qLeft.DisplayUnit != "" && qRight.DisplayUnit != "" {
		displayUnit = qLeft.DisplayUnit + "/" + qRight.DisplayUnit
	}
	return ast.NewQuantityNode(newVal, newScale, newDims, displayUnit), nil
}

// evalQuantityPow raises a quantity to a rational power.
func evalQuantityPow(q *ast.QuantityNode, exp Node) (Node, error) {
	ratExp, ok := exp.(*ast.RationalNode)
	if !ok {
		return nil, fmt.Errorf("quantity power exponent must be an exact rational, got %s", exp.String())
	}

	newVal, err := simplifyPow(q.Value, exp)
	if err != nil {
		return nil, err
	}

	newDims := q.Dimensions.ScalarMul(ratExp.Val)

	var newScale *big.Rat
	if ratExp.Val.IsInt() {
		newScale = quantityRatPowInt(q.Scale, ratExp.Val.Num().Int64())
	} else {
		if q.Scale.Cmp(big.NewRat(1, 1)) == 0 {
			newScale = big.NewRat(1, 1)
		} else {
			return nil, fmt.Errorf("fractional power on non-standard unit scale not supported: %s^%s", q.Scale.RatString(), ratExp.Val.RatString())
		}
	}

	if newDims.IsZero() {
		return normalizeDimensionless(newVal, newScale)
	}

	return ast.NewQuantityNode(newVal, newScale, newDims, ""), nil
}

// EvalQuantityRelOp evaluates relational operations between quantities.
func EvalQuantityRelOp(op string, left, right Node) (Node, error) {
	qLeft, isQLeft := left.(*ast.QuantityNode)
	qRight, isQRight := right.(*ast.QuantityNode)

	if !isQLeft || !isQRight {
		if op == "==" {
			return ast.NewRationalFromBigRat(big.NewRat(0, 1)), nil
		}
		if op == "!=" {
			return ast.NewRationalFromBigRat(big.NewRat(1, 1)), nil
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.dimension_mismatch", left.String(), right.String()))
	}

	if !qLeft.Dimensions.Equal(qRight.Dimensions) {
		if op == "==" {
			return ast.NewRationalFromBigRat(big.NewRat(0, 1)), nil
		}
		if op == "!=" {
			return ast.NewRationalFromBigRat(big.NewRat(1, 1)), nil
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.dimension_mismatch", qLeft.String(), qRight.String()))
	}

	// Dimensions match: normalize to base values (V * S) and compare
	v1Base, err := simplifyMul([]Node{qLeft.Value, ast.NewRationalFromBigRat(qLeft.Scale)})
	if err != nil {
		return nil, err
	}
	v2Base, err := simplifyMul([]Node{qRight.Value, ast.NewRationalFromBigRat(qRight.Scale)})
	if err != nil {
		return nil, err
	}

	rel := NewRelOp(v1Base, op, v2Base)
	if op == "==" || op == "!=" {
		res, err := EvaluateRelOpEquality(rel, nil)
		if err == nil {
			if v, ok := res.(*VarNode); ok {
				if v.Name == "true" {
					return ast.NewRationalFromBigRat(big.NewRat(1, 1)), nil
				}
				if v.Name == "false" {
					return ast.NewRationalFromBigRat(big.NewRat(0, 1)), nil
				}
			}
		}
	}

	// Inequalities
	if r1, ok1 := v1Base.(*RationalNode); ok1 {
		if r2, ok2 := v2Base.(*RationalNode); ok2 {
			cmp := r1.Val.Cmp(r2.Val)
			switch op {
			case "<":
				if cmp < 0 {
					return ast.NewRationalFromBigRat(big.NewRat(1, 1)), nil
				}
				return ast.NewRationalFromBigRat(big.NewRat(0, 1)), nil
			case "<=":
				if cmp <= 0 {
					return ast.NewRationalFromBigRat(big.NewRat(1, 1)), nil
				}
				return ast.NewRationalFromBigRat(big.NewRat(0, 1)), nil
			case ">":
				if cmp > 0 {
					return ast.NewRationalFromBigRat(big.NewRat(1, 1)), nil
				}
				return ast.NewRationalFromBigRat(big.NewRat(0, 1)), nil
			case ">=":
				if cmp >= 0 {
					return ast.NewRationalFromBigRat(big.NewRat(1, 1)), nil
				}
				return ast.NewRationalFromBigRat(big.NewRat(0, 1)), nil
			}
		}
	}

	return rel, nil
}

// evalQuantityAddChain evaluates a chain of additions/subtractions involving quantities.
func evalQuantityAddChain(terms []Node) (Node, error) {
	if len(terms) == 0 {
		return mustRational(0, 1), nil
	}

	var acc Node
	for i, t := range terms {
		var isNeg bool
		curr := t
		if uop, ok := curr.(*UnaryOpNode); ok && uop.Op == "-" {
			isNeg = true
			curr = uop.Expr
		}

		if i == 0 {
			if isNeg {
				q, ok := curr.(*ast.QuantityNode)
				if !ok {
					return nil, fmt.Errorf("%s", i18n.T("errors.dimension_mismatch", curr.String(), "0"))
				}
				negVal, err := simplifyUnaryOp("-", q.Value)
				if err != nil {
					return nil, err
				}
				acc = ast.NewQuantityNode(negVal, q.Scale, q.Dimensions, q.DisplayUnit)
			} else {
				acc = curr
			}
			continue
		}

		op := "+"
		if isNeg {
			op = "-"
		}
		res, err := EvalQuantityBinary(op, acc, curr)
		if err != nil {
			return nil, err
		}
		acc = res
	}
	return acc, nil
}

// evalQuantityMulChain evaluates a chain of multiplications/divisions involving quantities.
func evalQuantityMulChain(factors []Node) (Node, error) {
	if len(factors) == 0 {
		return mustRational(1, 1), nil
	}

	var acc Node = factors[0]
	for i := 1; i < len(factors); i++ {
		curr := factors[i]
		if pow, ok := curr.(*PowNode); ok && isNegative(pow.Exp) {
			expRat, isRat := pow.Exp.(*RationalNode)
			if isRat && expRat.Val.Cmp(big.NewRat(-1, 1)) == 0 {
				res, err := EvalQuantityBinary("/", acc, pow.Base)
				if err != nil {
					return nil, err
				}
				acc = res
				continue
			}
		}

		res, err := EvalQuantityBinary("*", acc, curr)
		if err != nil {
			return nil, err
		}
		acc = res
	}
	return acc, nil
}

// quantityRatPowInt computes r^exp for integer exp.
func quantityRatPowInt(r *big.Rat, exp int64) *big.Rat {
	if exp == 0 {
		return big.NewRat(1, 1)
	}
	if exp < 0 {
		inv := new(big.Rat).Inv(r)
		return quantityRatPowInt(inv, -exp)
	}
	num := new(big.Int).Exp(r.Num(), big.NewInt(exp), nil)
	den := new(big.Int).Exp(r.Denom(), big.NewInt(exp), nil)
	return new(big.Rat).SetFrac(num, den)
}

func formatUnitTarget(target Node) string {
	switch v := target.(type) {
	case *ast.VarNode:
		return v.Name
	case *ast.StringNode:
		return v.Value
	case *ast.MulNode:
		var numParts []string
		var denParts []string
		for _, f := range v.Factors {
			if pow, ok := f.(*ast.PowNode); ok {
				if r, okR := pow.Exp.(*ast.RationalNode); okR && r.Val.Sign() < 0 {
					absExp := new(big.Rat).Abs(r.Val)
					baseStr := formatUnitTarget(pow.Base)
					if absExp.Cmp(big.NewRat(1, 1)) == 0 {
						denParts = append(denParts, baseStr)
					} else {
						denParts = append(denParts, fmt.Sprintf("%s^%s", baseStr, absExp.RatString()))
					}
					continue
				}
			}
			numParts = append(numParts, formatUnitTarget(f))
		}
		numStr := strings.Join(numParts, "*")
		if len(numStr) == 0 {
			numStr = "1"
		}
		if len(denParts) == 0 {
			return numStr
		}
		denStr := strings.Join(denParts, "*")
		if len(denParts) > 1 {
			denStr = "(" + denStr + ")"
		}
		return numStr + "/" + denStr
	case *ast.PowNode:
		if r, okR := v.Exp.(*ast.RationalNode); okR && r.Val.Sign() < 0 {
			absExp := new(big.Rat).Abs(r.Val)
			baseStr := formatUnitTarget(v.Base)
			if absExp.Cmp(big.NewRat(1, 1)) == 0 {
				return "1/" + baseStr
			}
			return fmt.Sprintf("1/%s^%s", baseStr, absExp.RatString())
		}
		baseStr := formatUnitTarget(v.Base)
		return fmt.Sprintf("%s^%s", baseStr, v.Exp.String())
	default:
		return target.String()
	}
}

// ConvertUnit converts a quantity to the target unit representation.
func ConvertUnit(source Node, targetUnit Node) (Node, error) {
	qSrc, ok := source.(*ast.QuantityNode)
	if !ok {
		return nil, fmt.Errorf("first argument to convert_unit must be a quantity with units, got %s", source.String())
	}

	fsm := units.NewQuantityFSM()
	targetScale, targetDims, err := fsm.ResolveUnitExprWithFSM(targetUnit)
	if err != nil {
		return nil, err
	}

	if !qSrc.Dimensions.Equal(targetDims) {
		return nil, fmt.Errorf("%s", i18n.T("errors.incompatible_unit_conversion", qSrc.String(), targetUnit.String()))
	}

	scaleRatio := new(big.Rat).Quo(qSrc.Scale, targetScale)
	newVal, err := simplifyMul([]Node{qSrc.Value, ast.NewRationalFromBigRat(scaleRatio)})
	if err != nil {
		return nil, err
	}

	targetDisplay := formatUnitTarget(targetUnit)
	return ast.NewQuantityNode(newVal, targetScale, targetDims, targetDisplay), nil
}

func handleConvertUnit(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("errors.convert_unit_requires_2_arguments"))
	}

	srcNode, err := EvalWithEnv(args[0], env)
	if err != nil {
		return nil, err
	}

	return ConvertUnit(srcNode, args[1])
}

func handleUnitFunc(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("unit requires 1 argument")
	}
	fsm := units.NewQuantityFSM()
	scale, dims, err := fsm.ResolveUnitExprWithFSM(args[0])
	if err != nil {
		return nil, err
	}
	one := ast.NewRationalFromBigRat(big.NewRat(1, 1))
	return ast.NewQuantityNode(one, scale, dims, args[0].String()), nil
}
