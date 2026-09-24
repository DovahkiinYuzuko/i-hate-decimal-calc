package calc

import (
	"fmt"
	"strings"
)

// PointCoord returns the x and y variable nodes for a geometric point name.
func PointCoord(name string) (Node, Node) {
	name = strings.TrimSpace(name)
	return NewVar(fmt.Sprintf("%s_x", name)), NewVar(fmt.Sprintf("%s_y", name))
}

// GeometricPredicateTranslation holds the algebraic polynomial(s) and accompanying construction NDGs.
type GeometricPredicateTranslation struct {
	Equations        []Node
	ConstructionNDGs []Node
}

// TranslateGeometricPredicate converts a geometric predicate node (e.g. collinear(A, B, C))
// into minimal-degree algebraic polynomial equations and construction NDGs.
func TranslateGeometricPredicate(n Node) (*GeometricPredicateTranslation, error) {
	fn, ok := n.(*FuncNode)
	if !ok {
		// If it's already an algebraic expression/equation, return as a single equation
		if rel, okRel := n.(*RelOpNode); okRel && (rel.Op == "==" || rel.Op == "=") {
			negRHS, _ := simplifyUnaryOp("-", rel.RHS)
			diff, err := simplifyAdd([]Node{rel.LHS, negRHS})
			if err != nil {
				return nil, err
			}
			return &GeometricPredicateTranslation{
				Equations: []Node{diff},
			}, nil
		}
		return &GeometricPredicateTranslation{
			Equations: []Node{n},
		}, nil
	}

	name := strings.ToLower(fn.Name)
	args := fn.Args

	switch name {
	case "midpoint":
		if len(args) != 3 {
			return nil, fmt.Errorf("midpoint requires 3 arguments: midpoint(M, A, B)")
		}
		mX, mY := PointCoord(args[0].String())
		aX, aY := PointCoord(args[1].String())
		bX, bY := PointCoord(args[2].String())

		// 2 * mX - aX - bX = 0
		twoMX, _ := simplifyMul([]Node{mustRational(2, 1), mX})
		negAX, _ := simplifyUnaryOp("-", aX)
		negBX, _ := simplifyUnaryOp("-", bX)
		eqX, _ := simplifyAdd([]Node{twoMX, negAX, negBX})

		// 2 * mY - aY - bY = 0
		twoMY, _ := simplifyMul([]Node{mustRational(2, 1), mY})
		negAY, _ := simplifyUnaryOp("-", aY)
		negBY, _ := simplifyUnaryOp("-", bY)
		eqY, _ := simplifyAdd([]Node{twoMY, negAY, negBY})

		return &GeometricPredicateTranslation{
			Equations: []Node{eqX, eqY},
		}, nil

	case "collinear":
		if len(args) != 3 {
			return nil, fmt.Errorf("collinear requires 3 arguments: collinear(A, B, C)")
		}
		aX, aY := PointCoord(args[0].String())
		bX, bY := PointCoord(args[1].String())
		cX, cY := PointCoord(args[2].String())

		// (bX - aX) * (cY - aY) - (bY - aY) * (cX - aX) = 0
		negAX, _ := simplifyUnaryOp("-", aX)
		negAY, _ := simplifyUnaryOp("-", aY)
		dx1, _ := simplifyAdd([]Node{bX, negAX})
		dy1, _ := simplifyAdd([]Node{bY, negAY})
		dx2, _ := simplifyAdd([]Node{cX, negAX})
		dy2, _ := simplifyAdd([]Node{cY, negAY})

		term1, _ := simplifyMul([]Node{dx1, dy2})
		term2, _ := simplifyMul([]Node{dy1, dx2})
		negTerm2, _ := simplifyUnaryOp("-", term2)
		det, _ := simplifyAdd([]Node{term1, negTerm2})

		// Construction NDG: A != B
		distSq, _ := distSqExpr(aX, aY, bX, bY)

		return &GeometricPredicateTranslation{
			Equations:        []Node{det},
			ConstructionNDGs: []Node{distSq},
		}, nil

	case "parallel":
		if len(args) != 4 {
			return nil, fmt.Errorf("parallel requires 4 arguments: parallel(A, B, C, D)")
		}
		aX, aY := PointCoord(args[0].String())
		bX, bY := PointCoord(args[1].String())
		cX, cY := PointCoord(args[2].String())
		dX, dY := PointCoord(args[3].String())

		// (bX - aX) * (dY - cY) - (bY - aY) * (dX - cX) = 0
		negAX, _ := simplifyUnaryOp("-", aX)
		negAY, _ := simplifyUnaryOp("-", aY)
		negCX, _ := simplifyUnaryOp("-", cX)
		negCY, _ := simplifyUnaryOp("-", cY)

		dx1, _ := simplifyAdd([]Node{bX, negAX})
		dy1, _ := simplifyAdd([]Node{bY, negAY})
		dx2, _ := simplifyAdd([]Node{dX, negCX})
		dy2, _ := simplifyAdd([]Node{dY, negCY})

		term1, _ := simplifyMul([]Node{dx1, dy2})
		term2, _ := simplifyMul([]Node{dy1, dx2})
		negTerm2, _ := simplifyUnaryOp("-", term2)
		eq, _ := simplifyAdd([]Node{term1, negTerm2})

		distAB, _ := distSqExpr(aX, aY, bX, bY)
		distCD, _ := distSqExpr(cX, cY, dX, dY)

		return &GeometricPredicateTranslation{
			Equations:        []Node{eq},
			ConstructionNDGs: []Node{distAB, distCD},
		}, nil

	case "perpendicular":
		if len(args) != 4 {
			return nil, fmt.Errorf("perpendicular requires 4 arguments: perpendicular(A, B, C, D)")
		}
		aX, aY := PointCoord(args[0].String())
		bX, bY := PointCoord(args[1].String())
		cX, cY := PointCoord(args[2].String())
		dX, dY := PointCoord(args[3].String())

		// (bX - aX) * (dX - cX) + (bY - aY) * (dY - cY) = 0
		negAX, _ := simplifyUnaryOp("-", aX)
		negAY, _ := simplifyUnaryOp("-", aY)
		negCX, _ := simplifyUnaryOp("-", cX)
		negCY, _ := simplifyUnaryOp("-", cY)

		dx1, _ := simplifyAdd([]Node{bX, negAX})
		dy1, _ := simplifyAdd([]Node{bY, negAY})
		dx2, _ := simplifyAdd([]Node{dX, negCX})
		dy2, _ := simplifyAdd([]Node{dY, negCY})

		term1, _ := simplifyMul([]Node{dx1, dx2})
		term2, _ := simplifyMul([]Node{dy1, dy2})
		eq, _ := simplifyAdd([]Node{term1, term2})

		distAB, _ := distSqExpr(aX, aY, bX, bY)
		distCD, _ := distSqExpr(cX, cY, dX, dY)

		return &GeometricPredicateTranslation{
			Equations:        []Node{eq},
			ConstructionNDGs: []Node{distAB, distCD},
		}, nil

	case "equal_length_sq", "equal_length":
		if len(args) != 4 {
			return nil, fmt.Errorf("equal_length requires 4 arguments: equal_length(A, B, C, D)")
		}
		aX, aY := PointCoord(args[0].String())
		bX, bY := PointCoord(args[1].String())
		cX, cY := PointCoord(args[2].String())
		dX, dY := PointCoord(args[3].String())

		distAB, _ := distSqExpr(aX, aY, bX, bY)
		distCD, _ := distSqExpr(cX, cY, dX, dY)
		negCD, _ := simplifyUnaryOp("-", distCD)
		diff, _ := simplifyAdd([]Node{distAB, negCD})

		return &GeometricPredicateTranslation{
			Equations: []Node{diff},
		}, nil

	case "between", "inside", "outside", "convex":
		return nil, fmt.Errorf("predicate '%s' involves real ordering/inequalities which is outside algebraic Wu's method; use CAD instead", name)

	default:
		// Fallback as generic algebraic expression
		return &GeometricPredicateTranslation{
			Equations: []Node{n},
		}, nil
	}
}

// distSqExpr creates (x1 - x2)^2 + (y1 - y2)^2.
func distSqExpr(x1, y1, x2, y2 Node) (Node, error) {
	negX2, _ := simplifyUnaryOp("-", x2)
	negY2, _ := simplifyUnaryOp("-", y2)
	dx, _ := simplifyAdd([]Node{x1, negX2})
	dy, _ := simplifyAdd([]Node{y1, negY2})
	dxSq, _ := simplifyPow(dx, mustRational(2, 1))
	dySq, _ := simplifyPow(dy, mustRational(2, 1))
	return simplifyAdd([]Node{dxSq, dySq})
}
