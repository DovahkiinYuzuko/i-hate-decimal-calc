package calc

import (
	"fmt"
)

// -------------------------------------------------------------------------
// 2D Analytic Geometry Node Binding Functions
// -------------------------------------------------------------------------

func parseLineNode(n Node) (Line2D, error) {
	list, ok := n.(*ListNode)
	if !ok || len(list.Elements) != 3 {
		return Line2D{}, fmt.Errorf("line must be a 3-element list [A, B, C] representing Ax + By + C = 0, got %s", n.String())
	}
	return Line2D{
		A: list.Elements[0],
		B: list.Elements[1],
		C: list.Elements[2],
	}, nil
}

func parsePointNode(n Node) (Point2D, error) {
	list, ok := n.(*ListNode)
	if !ok || len(list.Elements) != 2 {
		return Point2D{}, fmt.Errorf("point must be a 2-element list [X, Y], got %s", n.String())
	}
	return Point2D{
		X: list.Elements[0],
		Y: list.Elements[1],
	}, nil
}

func pointToNode(p Point2D) Node {
	return &ListNode{Elements: []Node{p.X, p.Y}}
}

func evalLineIntersectFunc(line1Node, line2Node Node) (Node, error) {
	l1, err := parseLineNode(line1Node)
	if err != nil {
		return nil, fmt.Errorf("line_intersect error in line 1: %w", err)
	}
	l2, err := parseLineNode(line2Node)
	if err != nil {
		return nil, fmt.Errorf("line_intersect error in line 2: %w", err)
	}
	pt, err := IntersectLines(l1, l2)
	if err != nil {
		return nil, fmt.Errorf("line_intersect error: %w", err)
	}
	return pointToNode(pt), nil
}

func evalCircleIntersectFunc(center1Node, r1Node, center2Node, r2Node Node) (Node, error) {
	c1Center, err := parsePointNode(center1Node)
	if err != nil {
		return nil, fmt.Errorf("circle_intersect error in circle 1 center: %w", err)
	}
	c2Center, err := parsePointNode(center2Node)
	if err != nil {
		return nil, fmt.Errorf("circle_intersect error in circle 2 center: %w", err)
	}

	r1Sq, err := geoMul(r1Node, r1Node)
	if err != nil {
		return nil, fmt.Errorf("circle_intersect error in r1^2: %w", err)
	}
	r2Sq, err := geoMul(r2Node, r2Node)
	if err != nil {
		return nil, fmt.Errorf("circle_intersect error in r2^2: %w", err)
	}

	c1 := Circle2D{Center: c1Center, Radius: r1Node, RadiusSq: r1Sq}
	c2 := Circle2D{Center: c2Center, Radius: r2Node, RadiusSq: r2Sq}

	pts, err := IntersectCircles(c1, c2)
	if err != nil {
		return nil, fmt.Errorf("circle_intersect error: %w", err)
	}

	elemNodes := make([]Node, len(pts))
	for i, pt := range pts {
		elemNodes[i] = pointToNode(pt)
	}
	return &ListNode{Elements: elemNodes}, nil
}

func evalTriangleAreaFunc(p1Node, p2Node, p3Node Node) (Node, error) {
	p1, err := parsePointNode(p1Node)
	if err != nil {
		return nil, fmt.Errorf("triangle_area error in vertex 1: %w", err)
	}
	p2, err := parsePointNode(p2Node)
	if err != nil {
		return nil, fmt.Errorf("triangle_area error in vertex 2: %w", err)
	}
	p3, err := parsePointNode(p3Node)
	if err != nil {
		return nil, fmt.Errorf("triangle_area error in vertex 3: %w", err)
	}

	return TriangleArea(p1, p2, p3)
}

func evalTriangleCentersFunc(p1Node, p2Node, p3Node Node) (Node, error) {
	p1, err := parsePointNode(p1Node)
	if err != nil {
		return nil, fmt.Errorf("triangle_centers error in vertex 1: %w", err)
	}
	p2, err := parsePointNode(p2Node)
	if err != nil {
		return nil, fmt.Errorf("triangle_centers error in vertex 2: %w", err)
	}
	p3, err := parsePointNode(p3Node)
	if err != nil {
		return nil, fmt.Errorf("triangle_centers error in vertex 3: %w", err)
	}

	res, err := TriangleCenters(p1, p2, p3)
	if err != nil {
		return nil, fmt.Errorf("triangle_centers error: %w", err)
	}

	return &ListNode{
		Elements: []Node{
			pointToNode(res.Centroid),
			pointToNode(res.Circumcenter),
			pointToNode(res.Orthocenter),
			pointToNode(res.Incenter),
		},
	}, nil
}
