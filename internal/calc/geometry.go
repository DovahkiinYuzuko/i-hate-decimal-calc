package calc

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrParallelLines     = errors.New("lines are parallel and do not intersect")
	ErrCoincidentLines   = errors.New("lines are coincident (infinite intersections)")
	ErrCoincidentCircles = errors.New("circles are coincident (infinite intersections)")
	ErrDegenerateTriangle = errors.New("points are collinear; circumcenter/orthocenter/incenter undefined")
)

// Point2D represents an exact point in the 2D plane.
type Point2D struct {
	X Node
	Y Node
}

// Line2D represents an exact line in general form: Ax + By + C = 0.
type Line2D struct {
	A Node
	B Node
	C Node
}

// Circle2D represents an exact circle in standard form: (x - H)^2 + (y - K)^2 = R^2.
type Circle2D struct {
	Center   Point2D
	Radius   Node
	RadiusSq Node
}

// TriangleCentersResult stores the classical centers of a triangle.
type TriangleCentersResult struct {
	Centroid     Point2D
	Circumcenter Point2D
	Orthocenter  Point2D
	Incenter     Point2D
}

// Helper algebraic functions using the existing CAS evaluator.

func geoSub(a, b Node) (Node, error) {
	negOne := mustRational(-1, 1)
	expr := NewAdd([]Node{a, NewMul([]Node{negOne, b})})
	return Eval(expr)
}

func geoAdd(a, b Node) (Node, error) {
	expr := NewAdd([]Node{a, b})
	return Eval(expr)
}

func geoMul(a, b Node) (Node, error) {
	expr := NewMul([]Node{a, b})
	return Eval(expr)
}

func geoDiv(a, b Node) (Node, error) {
	negOne := mustRational(-1, 1)
	inv, err := NewPow(b, negOne)
	if err != nil {
		return nil, err
	}
	expr := NewMul([]Node{a, inv})
	return Eval(expr)
}

func isZeroExpr(n Node) bool {
	if n == nil {
		return true
	}
	simplified, err := Eval(n)
	if err != nil {
		return false
	}
	return isZeroNode(simplified)
}

// IsCollinear tests if three points lie on the same straight line using the 2D cross product.
// S = (x2 - x1)*(y3 - y1) - (y2 - y1)*(x3 - x1) == 0.
func IsCollinear(p1, p2, p3 Point2D) bool {
	dx21, err := geoSub(p2.X, p1.X)
	if err != nil {
		return false
	}
	dy31, err := geoSub(p3.Y, p1.Y)
	if err != nil {
		return false
	}
	dy21, err := geoSub(p2.Y, p1.Y)
	if err != nil {
		return false
	}
	dx31, err := geoSub(p3.X, p1.X)
	if err != nil {
		return false
	}

	prod1, err := geoMul(dx21, dy31)
	if err != nil {
		return false
	}
	prod2, err := geoMul(dy21, dx31)
	if err != nil {
		return false
	}

	diff, err := geoSub(prod1, prod2)
	if err != nil {
		return false
	}

	return isZeroExpr(diff)
}

// IsParallelLines checks if two lines have proportional direction vectors: A1*B2 - A2*B1 == 0.
func IsParallelLines(l1, l2 Line2D) bool {
	prod1, err := geoMul(l1.A, l2.B)
	if err != nil {
		return false
	}
	prod2, err := geoMul(l2.A, l1.B)
	if err != nil {
		return false
	}
	diff, err := geoSub(prod1, prod2)
	if err != nil {
		return false
	}
	return isZeroExpr(diff)
}

// IsCoincidentLines checks if two lines are identical (all coefficient ratios match).
func IsCoincidentLines(l1, l2 Line2D) bool {
	if !IsParallelLines(l1, l2) {
		return false
	}
	// Also check B1*C2 - B2*C1 == 0 and A1*C2 - A2*C1 == 0
	bc1, err := geoMul(l1.B, l2.C)
	if err != nil {
		return false
	}
	bc2, err := geoMul(l2.B, l1.C)
	if err != nil {
		return false
	}
	diffBC, err := geoSub(bc1, bc2)
	if err != nil || !isZeroExpr(diffBC) {
		return false
	}

	ac1, err := geoMul(l1.A, l2.C)
	if err != nil {
		return false
	}
	ac2, err := geoMul(l2.A, l1.C)
	if err != nil {
		return false
	}
	diffAC, err := geoSub(ac1, ac2)
	if err != nil || !isZeroExpr(diffAC) {
		return false
	}

	return true
}

// IntersectLines computes the exact intersection point of two lines using Cramer's rule.
// A1*x + B1*y = -C1
// A2*x + B2*y = -C2
// D = A1*B2 - A2*B1
// Dx = B1*C2 - B2*C1
// Dy = A2*C1 - A1*C2
func IntersectLines(l1, l2 Line2D) (Point2D, error) {
	prodA1B2, err := geoMul(l1.A, l2.B)
	if err != nil {
		return Point2D{}, err
	}
	prodA2B1, err := geoMul(l2.A, l1.B)
	if err != nil {
		return Point2D{}, err
	}
	D, err := geoSub(prodA1B2, prodA2B1)
	if err != nil {
		return Point2D{}, err
	}

	if isZeroExpr(D) {
		if IsCoincidentLines(l1, l2) {
			return Point2D{}, ErrCoincidentLines
		}
		return Point2D{}, ErrParallelLines
	}

	// Dx = B1*C2 - B2*C1
	prodB1C2, err := geoMul(l1.B, l2.C)
	if err != nil {
		return Point2D{}, err
	}
	prodB2C1, err := geoMul(l2.B, l1.C)
	if err != nil {
		return Point2D{}, err
	}
	Dx, err := geoSub(prodB1C2, prodB2C1)
	if err != nil {
		return Point2D{}, err
	}

	// Dy = A2*C1 - A1*C2
	prodA2C1, err := geoMul(l2.A, l1.C)
	if err != nil {
		return Point2D{}, err
	}
	prodA1C2, err := geoMul(l1.A, l2.C)
	if err != nil {
		return Point2D{}, err
	}
	Dy, err := geoSub(prodA2C1, prodA1C2)
	if err != nil {
		return Point2D{}, err
	}

	x, err := geoDiv(Dx, D)
	if err != nil {
		return Point2D{}, err
	}
	y, err := geoDiv(Dy, D)
	if err != nil {
		return Point2D{}, err
	}

	return Point2D{X: x, Y: y}, nil
}

func signNode(n Node) (int, error) {
	simplified, err := Eval(n)
	if err != nil {
		return 0, err
	}
	if isZeroNode(simplified) {
		return 0, nil
	}
	if r, ok := simplified.(*RationalNode); ok {
		return r.Val.Sign(), nil
	}
	c, err := evalComplex(simplified)
	if err != nil {
		return 0, err
	}
	if math.Abs(real(c)) < 1e-12 {
		return 0, nil
	}
	if real(c) > 0 {
		return 1, nil
	}
	return -1, nil
}

// IntersectLineCircle finds the exact intersection points between a line and a circle.
// Line: Ax + By + C = 0
// Circle: (x - H)^2 + (y - K)^2 = R^2
func IntersectLineCircle(l Line2D, c Circle2D) ([]Point2D, error) {
	if isZeroExpr(l.A) && isZeroExpr(l.B) {
		return nil, fmt.Errorf("invalid line: A and B cannot both be zero")
	}

	two := mustRational(2, 1)
	four := mustRational(4, 1)

	// Case 1: Line is not vertical (B != 0)
	// y = (-A*x - C) / B = m*x + d
	if !isZeroExpr(l.B) {
		negA, err := geoMul(mustRational(-1, 1), l.A)
		if err != nil {
			return nil, err
		}
		m, err := geoDiv(negA, l.B)
		if err != nil {
			return nil, err
		}

		negC, err := geoMul(mustRational(-1, 1), l.C)
		if err != nil {
			return nil, err
		}
		d, err := geoDiv(negC, l.B)
		if err != nil {
			return nil, err
		}

		// Quadratic in x: alpha*x^2 + beta*x + gamma = 0
		// alpha = 1 + m^2
		mSq, err := geoMul(m, m)
		if err != nil {
			return nil, err
		}
		alpha, err := geoAdd(mustRational(1, 1), mSq)
		if err != nil {
			return nil, err
		}

		// dMinusK = d - K
		dMinusK, err := geoSub(d, c.Center.Y)
		if err != nil {
			return nil, err
		}

		// beta = 2*(m*dMinusK - H)
		mDminusK, err := geoMul(m, dMinusK)
		if err != nil {
			return nil, err
		}
		betaInner, err := geoSub(mDminusK, c.Center.X)
		if err != nil {
			return nil, err
		}
		beta, err := geoMul(two, betaInner)
		if err != nil {
			return nil, err
		}

		// gamma = H^2 + (d-K)^2 - R^2
		hSq, err := geoMul(c.Center.X, c.Center.X)
		if err != nil {
			return nil, err
		}
		dMinusKSq, err := geoMul(dMinusK, dMinusK)
		if err != nil {
			return nil, err
		}
		hSqPlusDsq, err := geoAdd(hSq, dMinusKSq)
		if err != nil {
			return nil, err
		}
		gamma, err := geoSub(hSqPlusDsq, c.RadiusSq)
		if err != nil {
			return nil, err
		}

		// Discriminant D = beta^2 - 4*alpha*gamma
		betaSq, err := geoMul(beta, beta)
		if err != nil {
			return nil, err
		}
		fourAlpha, err := geoMul(four, alpha)
		if err != nil {
			return nil, err
		}
		fourAlphaGamma, err := geoMul(fourAlpha, gamma)
		if err != nil {
			return nil, err
		}
		disc, err := geoSub(betaSq, fourAlphaGamma)
		if err != nil {
			return nil, err
		}

		s, err := signNode(disc)
		if err != nil {
			return nil, err
		}
		if s < 0 {
			return []Point2D{}, nil
		}

		twoAlpha, err := geoMul(two, alpha)
		if err != nil {
			return nil, err
		}
		negBeta, err := geoMul(mustRational(-1, 1), beta)
		if err != nil {
			return nil, err
		}

		if s == 0 {
			// One tangent point
			x, err := geoDiv(negBeta, twoAlpha)
			if err != nil {
				return nil, err
			}
			mx, err := geoMul(m, x)
			if err != nil {
				return nil, err
			}
			y, err := geoAdd(mx, d)
			if err != nil {
				return nil, err
			}
			return []Point2D{{X: x, Y: y}}, nil
		}

		// Two intersection points: x = (-beta +/- sqrt(D)) / (2*alpha)
		sqrtD := NewSqrt(disc)
		x1Num, err := geoAdd(negBeta, sqrtD)
		if err != nil {
			return nil, err
		}
		x1, err := geoDiv(x1Num, twoAlpha)
		if err != nil {
			return nil, err
		}
		mx1, err := geoMul(m, x1)
		if err != nil {
			return nil, err
		}
		y1, err := geoAdd(mx1, d)
		if err != nil {
			return nil, err
		}

		x2Num, err := geoSub(negBeta, sqrtD)
		if err != nil {
			return nil, err
		}
		x2, err := geoDiv(x2Num, twoAlpha)
		if err != nil {
			return nil, err
		}
		mx2, err := geoMul(m, x2)
		if err != nil {
			return nil, err
		}
		y2, err := geoAdd(mx2, d)
		if err != nil {
			return nil, err
		}

		return []Point2D{{X: x1, Y: y1}, {X: x2, Y: y2}}, nil
	}

	// Case 2: Line is vertical (B == 0) => x = -C / A
	negC, err := geoMul(mustRational(-1, 1), l.C)
	if err != nil {
		return nil, err
	}
	x0, err := geoDiv(negC, l.A)
	if err != nil {
		return nil, err
	}

	// (x0 - H)^2 + (y - K)^2 = R^2 => (y - K)^2 = R^2 - (x0 - H)^2
	dx, err := geoSub(x0, c.Center.X)
	if err != nil {
		return nil, err
	}
	dxSq, err := geoMul(dx, dx)
	if err != nil {
		return nil, err
	}
	discY, err := geoSub(c.RadiusSq, dxSq)
	if err != nil {
		return nil, err
	}

	s, err := signNode(discY)
	if err != nil {
		return nil, err
	}
	if s < 0 {
		return []Point2D{}, nil
	}
	if s == 0 {
		return []Point2D{{X: x0, Y: c.Center.Y}}, nil
	}

	sqrtDiscY := NewSqrt(discY)
	y1, err := geoAdd(c.Center.Y, sqrtDiscY)
	if err != nil {
		return nil, err
	}
	y2, err := geoSub(c.Center.Y, sqrtDiscY)
	if err != nil {
		return nil, err
	}

	return []Point2D{{X: x0, Y: y1}, {X: x0, Y: y2}}, nil
}

// IntersectCircles finds the exact intersection points of two circles using the Radical Axis reduction.
func IntersectCircles(c1, c2 Circle2D) ([]Point2D, error) {
	dxCenter, err := geoSub(c1.Center.X, c2.Center.X)
	if err != nil {
		return nil, err
	}
	dyCenter, err := geoSub(c1.Center.Y, c2.Center.Y)
	if err != nil {
		return nil, err
	}

	isSameCenter := isZeroExpr(dxCenter) && isZeroExpr(dyCenter)
	if isSameCenter {
		dRadiusSq, err := geoSub(c1.RadiusSq, c2.RadiusSq)
		if err != nil {
			return nil, err
		}
		if isZeroExpr(dRadiusSq) {
			return nil, ErrCoincidentCircles
		}
		// Concentric but different radii => disjoint
		return []Point2D{}, nil
	}

	// Radical axis equation:
	// Circle 1: x^2 + y^2 - 2*h1*x - 2*k1*y + (h1^2 + k1^2 - r1^2) = 0
	// Circle 2: x^2 + y^2 - 2*h2*x - 2*k2*y + (h2^2 + k2^2 - r2^2) = 0
	// Subtracting:
	// 2*(h2 - h1)*x + 2*(k2 - k1)*y + ((h1^2 + k1^2 - r1^2) - (h2^2 + k2^2 - r2^2)) = 0
	two := mustRational(2, 1)

	h2MinusH1, err := geoSub(c2.Center.X, c1.Center.X)
	if err != nil {
		return nil, err
	}
	A_rad, err := geoMul(two, h2MinusH1)
	if err != nil {
		return nil, err
	}

	k2MinusK1, err := geoSub(c2.Center.Y, c1.Center.Y)
	if err != nil {
		return nil, err
	}
	B_rad, err := geoMul(two, k2MinusK1)
	if err != nil {
		return nil, err
	}

	// h1^2 + k1^2 - r1^2
	h1Sq, err := geoMul(c1.Center.X, c1.Center.X)
	if err != nil {
		return nil, err
	}
	k1Sq, err := geoMul(c1.Center.Y, c1.Center.Y)
	if err != nil {
		return nil, err
	}
	const1, err := geoAdd(h1Sq, k1Sq)
	if err != nil {
		return nil, err
	}
	const1, err = geoSub(const1, c1.RadiusSq)
	if err != nil {
		return nil, err
	}

	// h2^2 + k2^2 - r2^2
	h2Sq, err := geoMul(c2.Center.X, c2.Center.X)
	if err != nil {
		return nil, err
	}
	k2Sq, err := geoMul(c2.Center.Y, c2.Center.Y)
	if err != nil {
		return nil, err
	}
	const2, err := geoAdd(h2Sq, k2Sq)
	if err != nil {
		return nil, err
	}
	const2, err = geoSub(const2, c2.RadiusSq)
	if err != nil {
		return nil, err
	}

	C_rad, err := geoSub(const1, const2)
	if err != nil {
		return nil, err
	}

	radAxis := Line2D{A: A_rad, B: B_rad, C: C_rad}
	return IntersectLineCircle(radAxis, c1)
}

// TriangleArea calculates the exact area of a triangle given three points using the Shoelace formula.
func TriangleArea(p1, p2, p3 Point2D) (Node, error) {
	// 2*S = (x2 - x1)*(y3 - y1) - (x3 - x1)*(y2 - y1)
	dx21, err := geoSub(p2.X, p1.X)
	if err != nil {
		return nil, err
	}
	dy31, err := geoSub(p3.Y, p1.Y)
	if err != nil {
		return nil, err
	}
	dx31, err := geoSub(p3.X, p1.X)
	if err != nil {
		return nil, err
	}
	dy21, err := geoSub(p2.Y, p1.Y)
	if err != nil {
		return nil, err
	}

	prod1, err := geoMul(dx21, dy31)
	if err != nil {
		return nil, err
	}
	prod2, err := geoMul(dx31, dy21)
	if err != nil {
		return nil, err
	}

	cross, err := geoSub(prod1, prod2)
	if err != nil {
		return nil, err
	}

	s, err := signNode(cross)
	if err != nil {
		return nil, err
	}
	absCross := cross
	if s < 0 {
		absCross, err = geoMul(mustRational(-1, 1), cross)
		if err != nil {
			return nil, err
		}
	}

	half := mustRational(1, 2)
	return geoMul(half, absCross)
}

// TriangleHeron calculates the exact area of a triangle given three side lengths a, b, c using Heron's formula.
func TriangleHeron(a, b, c Node) (Node, error) {
	half := mustRational(1, 2)

	// s = (a + b + c) / 2
	ab, err := geoAdd(a, b)
	if err != nil {
		return nil, err
	}
	abc, err := geoAdd(ab, c)
	if err != nil {
		return nil, err
	}
	s, err := geoMul(half, abc)
	if err != nil {
		return nil, err
	}

	// s - a
	sMinusA, err := geoSub(s, a)
	if err != nil {
		return nil, err
	}
	// s - b
	sMinusB, err := geoSub(s, b)
	if err != nil {
		return nil, err
	}
	// s - c
	sMinusC, err := geoSub(s, c)
	if err != nil {
		return nil, err
	}

	p1, err := geoMul(s, sMinusA)
	if err != nil {
		return nil, err
	}
	p2, err := geoMul(sMinusB, sMinusC)
	if err != nil {
		return nil, err
	}
	product, err := geoMul(p1, p2)
	if err != nil {
		return nil, err
	}

	return Eval(NewSqrt(product))
}

// TriangleCenters computes the centroid, circumcenter, orthocenter, and incenter of a triangle.
func TriangleCenters(p1, p2, p3 Point2D) (TriangleCentersResult, error) {
	if IsCollinear(p1, p2, p3) {
		return TriangleCentersResult{}, ErrDegenerateTriangle
	}

	three := mustRational(3, 1)

	// 1. Centroid: G = (p1 + p2 + p3) / 3
	xSum12, err := geoAdd(p1.X, p2.X)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	xSum, err := geoAdd(xSum12, p3.X)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	centroidX, err := geoDiv(xSum, three)
	if err != nil {
		return TriangleCentersResult{}, err
	}

	ySum12, err := geoAdd(p1.Y, p2.Y)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	ySum, err := geoAdd(ySum12, p3.Y)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	centroidY, err := geoDiv(ySum, three)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	centroid := Point2D{X: centroidX, Y: centroidY}

	// 2. Circumcenter: Intersection of perpendicular bisectors of p1-p2 and p2-p3
	// Bisector of p1-p2: 2*(x2 - x1)*x + 2*(y2 - y1)*y + (x1^2 + y1^2 - (x2^2 + y2^2)) = 0
	two := mustRational(2, 1)

	makeBisector := func(pa, pb Point2D) (Line2D, error) {
		dx, err := geoSub(pb.X, pa.X)
		if err != nil {
			return Line2D{}, err
		}
		aCoeff, err := geoMul(two, dx)
		if err != nil {
			return Line2D{}, err
		}

		dy, err := geoSub(pb.Y, pa.Y)
		if err != nil {
			return Line2D{}, err
		}
		bCoeff, err := geoMul(two, dy)
		if err != nil {
			return Line2D{}, err
		}

		xaSq, err := geoMul(pa.X, pa.X)
		if err != nil {
			return Line2D{}, err
		}
		yaSq, err := geoMul(pa.Y, pa.Y)
		if err != nil {
			return Line2D{}, err
		}
		normASq, err := geoAdd(xaSq, yaSq)
		if err != nil {
			return Line2D{}, err
		}

		xbSq, err := geoMul(pb.X, pb.X)
		if err != nil {
			return Line2D{}, err
		}
		ybSq, err := geoMul(pb.Y, pb.Y)
		if err != nil {
			return Line2D{}, err
		}
		normBSq, err := geoAdd(xbSq, ybSq)
		if err != nil {
			return Line2D{}, err
		}

		cCoeff, err := geoSub(normASq, normBSq)
		if err != nil {
			return Line2D{}, err
		}

		return Line2D{A: aCoeff, B: bCoeff, C: cCoeff}, nil
	}

	bisector12, err := makeBisector(p1, p2)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	bisector23, err := makeBisector(p2, p3)
	if err != nil {
		return TriangleCentersResult{}, err
	}

	circumcenter, err := IntersectLines(bisector12, bisector23)
	if err != nil {
		return TriangleCentersResult{}, err
	}

	// 3. Orthocenter: H = 3*G - 2*O (Euler line formula)
	threeGX, err := geoMul(three, centroid.X)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	twoOX, err := geoMul(two, circumcenter.X)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	orthoX, err := geoSub(threeGX, twoOX)
	if err != nil {
		return TriangleCentersResult{}, err
	}

	threeGY, err := geoMul(three, centroid.Y)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	twoOY, err := geoMul(two, circumcenter.Y)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	orthoY, err := geoSub(threeGY, twoOY)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	orthocenter := Point2D{X: orthoX, Y: orthoY}

	// 4. Incenter: I = (a*A + b*B + c*C) / (a + b + c)
	// a is opposite p1 (length p2-p3)
	// b is opposite p2 (length p1-p3)
	// c is opposite p3 (length p1-p2)
	dist := func(pa, pb Point2D) (Node, error) {
		dx, err := geoSub(pa.X, pb.X)
		if err != nil {
			return nil, err
		}
		dy, err := geoSub(pa.Y, pb.Y)
		if err != nil {
			return nil, err
		}
		dxSq, err := geoMul(dx, dx)
		if err != nil {
			return nil, err
		}
		dySq, err := geoMul(dy, dy)
		if err != nil {
			return nil, err
		}
		sumSq, err := geoAdd(dxSq, dySq)
		if err != nil {
			return nil, err
		}
		return Eval(NewSqrt(sumSq))
	}

	sideA, err := dist(p2, p3)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	sideB, err := dist(p1, p3)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	sideC, err := dist(p1, p2)
	if err != nil {
		return TriangleCentersResult{}, err
	}

	// perimeter = a + b + c
	perimAB, err := geoAdd(sideA, sideB)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	perimeter, err := geoAdd(perimAB, sideC)
	if err != nil {
		return TriangleCentersResult{}, err
	}

	// Incenter X = (a*x1 + b*x2 + c*x3) / perimeter
	ax1, err := geoMul(sideA, p1.X)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	bx2, err := geoMul(sideB, p2.X)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	cx3, err := geoMul(sideC, p3.X)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	sumX12, err := geoAdd(ax1, bx2)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	sumX, err := geoAdd(sumX12, cx3)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	incenterX, err := geoDiv(sumX, perimeter)
	if err != nil {
		return TriangleCentersResult{}, err
	}

	// Incenter Y = (a*y1 + b*y2 + c*y3) / perimeter
	ay1, err := geoMul(sideA, p1.Y)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	by2, err := geoMul(sideB, p2.Y)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	cy3, err := geoMul(sideC, p3.Y)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	sumY12, err := geoAdd(ay1, by2)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	sumY, err := geoAdd(sumY12, cy3)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	incenterY, err := geoDiv(sumY, perimeter)
	if err != nil {
		return TriangleCentersResult{}, err
	}
	incenter := Point2D{X: incenterX, Y: incenterY}

	return TriangleCentersResult{
		Centroid:     centroid,
		Circumcenter: circumcenter,
		Orthocenter:  orthocenter,
		Incenter:     incenter,
	}, nil
}
