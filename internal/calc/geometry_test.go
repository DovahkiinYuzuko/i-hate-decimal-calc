package calc

import (
	"math/big"
	"testing"
)

func rat(n, d int64) *big.Rat {
	return big.NewRat(n, d)
}

func numNode(n, d int64) Node {
	r, err := NewRational(n, d)
	if err != nil {
		panic(err)
	}
	return r
}

func pt(xN, xD, yN, yD int64) Point2D {
	return Point2D{
		X: numNode(xN, xD),
		Y: numNode(yN, yD),
	}
}

func line(aN, aD, bN, bD, cN, cD int64) Line2D {
	return Line2D{
		A: numNode(aN, aD),
		B: numNode(bN, bD),
		C: numNode(cN, cD),
	}
}

func TestIsCollinear(t *testing.T) {
	// (0, 0), (1, 1), (2, 2) are collinear
	p1 := pt(0, 1, 0, 1)
	p2 := pt(1, 1, 1, 1)
	p3 := pt(2, 1, 2, 1)
	if !IsCollinear(p1, p2, p3) {
		t.Errorf("expected p1, p2, p3 to be collinear")
	}

	// (0, 0), (2, 0), (0, 3) form a triangle (non-collinear)
	p4 := pt(2, 1, 0, 1)
	p5 := pt(0, 1, 3, 1)
	if IsCollinear(p1, p4, p5) {
		t.Errorf("expected p1, p4, p5 to NOT be collinear")
	}
}

func TestLinePredicates(t *testing.T) {
	// l1: x + y - 1 = 0
	l1 := line(1, 1, 1, 1, -1, 1)
	// l2: x + y - 5 = 0 (parallel to l1)
	l2 := line(1, 1, 1, 1, -5, 1)
	// l3: 2x + 2y - 2 = 0 (coincident to l1)
	l3 := line(2, 1, 2, 1, -2, 1)
	// l4: x - y = 0 (intersecting l1)
	l4 := line(1, 1, -1, 1, 0, 1)

	if !IsParallelLines(l1, l2) {
		t.Errorf("expected l1 and l2 to be parallel")
	}
	if IsCoincidentLines(l1, l2) {
		t.Errorf("expected l1 and l2 to NOT be coincident")
	}

	if !IsParallelLines(l1, l3) {
		t.Errorf("expected l1 and l3 to be parallel")
	}
	if !IsCoincidentLines(l1, l3) {
		t.Errorf("expected l1 and l3 to be coincident")
	}

	if IsParallelLines(l1, l4) {
		t.Errorf("expected l1 and l4 to NOT be parallel")
	}
}

func TestIntersectLines(t *testing.T) {
	// l1: x + y - 4 = 0, l2: x - y = 0 => (2, 2)
	l1 := line(1, 1, 1, 1, -4, 1)
	l2 := line(1, 1, -1, 1, 0, 1)

	res, err := IntersectLines(l1, l2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xNum, ok1 := res.X.(*RationalNode)
	yNum, ok2 := res.Y.(*RationalNode)
	if !ok1 || !ok2 {
		t.Fatalf("expected RationalNode coordinates, got X=%T, Y=%T", res.X, res.Y)
	}

	if xNum.Val.Cmp(rat(2, 1)) != 0 || yNum.Val.Cmp(rat(2, 1)) != 0 {
		t.Errorf("expected (2, 2), got (%s, %s)", xNum.Val.String(), yNum.Val.String())
	}

	// Rational fraction intersection:
	// 3x + 2y - 7 = 0
	// x - y - 1 = 0
	// 3(1+y) + 2y = 7 => 5y = 4 => y = 4/5, x = 9/5
	l3 := line(3, 1, 2, 1, -7, 1)
	l4 := line(1, 1, -1, 1, -1, 1)

	res2, err2 := IntersectLines(l3, l4)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	xNum2 := res2.X.(*RationalNode)
	yNum2 := res2.Y.(*RationalNode)
	if xNum2.Val.Cmp(rat(9, 5)) != 0 || yNum2.Val.Cmp(rat(4, 5)) != 0 {
		t.Errorf("expected (9/5, 4/5), got (%s, %s)", xNum2.Val.String(), yNum2.Val.String())
	}

	// Parallel lines error check:
	lParallel := line(1, 1, 1, 1, 10, 1)
	_, errParallel := IntersectLines(l1, lParallel)
	if errParallel == nil {
		t.Errorf("expected error for parallel lines, got nil")
	}
}

func circle(hN, hD, kN, kD, rN, rD int64) Circle2D {
	r := numNode(rN, rD)
	rSq := numNode(rN*rN, rD*rD)
	return Circle2D{
		Center:   pt(hN, hD, kN, kD),
		Radius:   r,
		RadiusSq: rSq,
	}
}

func TestIntersectLineCircle(t *testing.T) {
	c := circle(0, 1, 0, 1, 2, 1) // x^2 + y^2 = 4

	// Secant line: y = 0 => 0*x + 1*y + 0 = 0
	lSecant := line(0, 1, 1, 1, 0, 1)
	pts, err := IntersectLineCircle(lSecant, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pts) != 2 {
		t.Fatalf("expected 2 intersection points, got %d", len(pts))
	}
	// Check (-2, 0) and (2, 0)
	x0 := pts[0].X.String()
	x1 := pts[1].X.String()
	if !((x0 == "-2" && x1 == "2") || (x0 == "2" && x1 == "-2")) {
		t.Errorf("expected points at x=-2 and x=2, got %s and %s", x0, x1)
	}

	// Tangent line: y = 2 => 0*x + 1*y - 2 = 0
	lTangent := line(0, 1, 1, 1, -2, 1)
	ptsTan, err := IntersectLineCircle(lTangent, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ptsTan) != 1 {
		t.Fatalf("expected 1 tangent point, got %d", len(ptsTan))
	}
	if ptsTan[0].X.String() != "0" || ptsTan[0].Y.String() != "2" {
		t.Errorf("expected (0, 2), got (%s, %s)", ptsTan[0].X.String(), ptsTan[0].Y.String())
	}

	// Disjoint line: y = 3 => 0*x + 1*y - 3 = 0
	lDisjoint := line(0, 1, 1, 1, -3, 1)
	ptsDis, err := IntersectLineCircle(lDisjoint, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ptsDis) != 0 {
		t.Fatalf("expected 0 intersection points, got %d", len(ptsDis))
	}
}

func TestIntersectCircles(t *testing.T) {
	// c1: x^2 + y^2 = 4, c2: (x-2)^2 + y^2 = 4
	c1 := circle(0, 1, 0, 1, 2, 1)
	c2 := circle(2, 1, 0, 1, 2, 1)

	pts, err := IntersectCircles(c1, c2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pts) != 2 {
		t.Fatalf("expected 2 intersection points, got %d", len(pts))
	}

	// Radical axis is x = 1, y = +/- sqrt(3)
	for _, p := range pts {
		if p.X.String() != "1" {
			t.Errorf("expected X=1, got %s", p.X.String())
		}
		ySq, err := geoMul(p.Y, p.Y)
		if err != nil {
			t.Fatalf("failed to square Y: %v", err)
		}
		if ySq.String() != "3" {
			t.Errorf("expected Y^2=3, got %s (from Y=%s)", ySq.String(), p.Y.String())
		}
	}

	// Tangent circles: c1 (0,0) r=1, c3 (3,0) r=2 => touches at (1, 0)
	c3 := circle(3, 1, 0, 1, 2, 1)
	cSmall := circle(0, 1, 0, 1, 1, 1)
	ptsTan, err := IntersectCircles(cSmall, c3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ptsTan) != 1 {
		t.Fatalf("expected 1 tangent point, got %d", len(ptsTan))
	}
	if ptsTan[0].X.String() != "1" || ptsTan[0].Y.String() != "0" {
		t.Errorf("expected (1, 0), got (%s, %s)", ptsTan[0].X.String(), ptsTan[0].Y.String())
	}

	// Concentric circles: c1 (0,0) r=2, c4 (0,0) r=1 => 0 points
	c4 := circle(0, 1, 0, 1, 1, 1)
	ptsConc, err := IntersectCircles(c1, c4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ptsConc) != 0 {
		t.Errorf("expected 0 points for concentric unequal circles, got %d", len(ptsConc))
	}

	// Coincident circles: c1 and c1 => ErrCoincidentCircles
	_, errCoinc := IntersectCircles(c1, c1)
	if errCoinc == nil {
		t.Errorf("expected ErrCoincidentCircles for identical circles, got nil")
	}
}

func TestTriangleArea(t *testing.T) {
	// (0, 0), (4, 0), (0, 3) => Area = 6
	p1 := pt(0, 1, 0, 1)
	p2 := pt(4, 1, 0, 1)
	p3 := pt(0, 1, 3, 1)

	area, err := TriangleArea(p1, p2, p3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if area.String() != "6" {
		t.Errorf("expected area 6, got %s", area.String())
	}

	// Heron with sides 3, 4, 5 => Area = 6
	heronArea, err := TriangleHeron(numNode(3, 1), numNode(4, 1), numNode(5, 1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if heronArea.String() != "6" {
		t.Errorf("expected Heron area 6, got %s", heronArea.String())
	}

	// Collinear points => Area = 0
	pCollinear := pt(2, 1, 2, 1)
	collinearArea, err := TriangleArea(p1, pt(1, 1, 1, 1), pCollinear)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if collinearArea.String() != "0" {
		t.Errorf("expected area 0 for collinear points, got %s", collinearArea.String())
	}
}

func TestTriangleCenters(t *testing.T) {
	// Right triangle: (0, 0), (4, 0), (0, 3)
	p1 := pt(0, 1, 0, 1) // A
	p2 := pt(4, 1, 0, 1) // B
	p3 := pt(0, 1, 3, 1) // C

	centers, err := TriangleCenters(p1, p2, p3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Centroid = (4/3, 1)
	if centers.Centroid.X.String() != "4/3" || centers.Centroid.Y.String() != "1" {
		t.Errorf("expected Centroid (4/3, 1), got (%s, %s)", centers.Centroid.X.String(), centers.Centroid.Y.String())
	}

	// Circumcenter = (2, 3/2)
	if centers.Circumcenter.X.String() != "2" || centers.Circumcenter.Y.String() != "3/2" {
		t.Errorf("expected Circumcenter (2, 3/2), got (%s, %s)", centers.Circumcenter.X.String(), centers.Circumcenter.Y.String())
	}

	// Orthocenter = (0, 0)
	if centers.Orthocenter.X.String() != "0" || centers.Orthocenter.Y.String() != "0" {
		t.Errorf("expected Orthocenter (0, 0), got (%s, %s)", centers.Orthocenter.X.String(), centers.Orthocenter.Y.String())
	}

	// Incenter = (1, 1)
	if centers.Incenter.X.String() != "1" || centers.Incenter.Y.String() != "1" {
		t.Errorf("expected Incenter (1, 1), got (%s, %s)", centers.Incenter.X.String(), centers.Incenter.Y.String())
	}

	// Degenerate triangle check:
	_, errDegen := TriangleCenters(p1, pt(1, 1, 1, 1), pt(2, 1, 2, 1))
	if errDegen == nil {
		t.Errorf("expected ErrDegenerateTriangle for collinear vertices, got nil")
	}
}
