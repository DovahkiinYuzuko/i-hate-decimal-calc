package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
)

// -------------------------------------------------------------------------
// 2D Analytic Geometry Node Binding Functions
// -------------------------------------------------------------------------

func parseLineNode(n Node) (Line2D, error) {
	list, ok := n.(*ListNode)
	if !ok || len(list.Elements) != 3 {
		return Line2D{}, fmt.Errorf("%s", i18n.T("geometry.err_line_must_be_a_3", n.String()))
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
		return Point2D{}, fmt.Errorf("%s", i18n.T("geometry.err_point_must_be_a_2", n.String()))
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
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_line_intersect_error_in_line", err))
	}
	l2, err := parseLineNode(line2Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_line_intersect_error_in_line_1", err))
	}
	pt, err := IntersectLines(l1, l2)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_line_intersect_error", err))
	}
	return pointToNode(pt), nil
}

func evalCircleIntersectFunc(center1Node, r1Node, center2Node, r2Node Node) (Node, error) {
	c1Center, err := parsePointNode(center1Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_circle_intersect_error_in_circle", err))
	}
	c2Center, err := parsePointNode(center2Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_circle_intersect_error_in_circle_1", err))
	}

	r1Sq, err := geoMul(r1Node, r1Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_circle_intersect_error_in_r1", err))
	}
	r2Sq, err := geoMul(r2Node, r2Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_circle_intersect_error_in_r2", err))
	}

	c1 := Circle2D{Center: c1Center, Radius: r1Node, RadiusSq: r1Sq}
	c2 := Circle2D{Center: c2Center, Radius: r2Node, RadiusSq: r2Sq}

	pts, err := IntersectCircles(c1, c2)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_circle_intersect_error", err))
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
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_triangle_area_error_in_vertex", err))
	}
	p2, err := parsePointNode(p2Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_triangle_area_error_in_vertex_1", err))
	}
	p3, err := parsePointNode(p3Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_triangle_area_error_in_vertex_2", err))
	}

	return TriangleArea(p1, p2, p3)
}

func evalTriangleCentersFunc(p1Node, p2Node, p3Node Node) (Node, error) {
	p1, err := parsePointNode(p1Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_triangle_centers_error_in_vertex", err))
	}
	p2, err := parsePointNode(p2Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_triangle_centers_error_in_vertex_1", err))
	}
	p3, err := parsePointNode(p3Node)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_triangle_centers_error_in_vertex_2", err))
	}

	res, err := TriangleCenters(p1, p2, p3)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("geometry.err_triangle_centers_error", err))
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
