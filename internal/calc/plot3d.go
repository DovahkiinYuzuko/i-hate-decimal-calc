package calc

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// Camera3D holds viewing and projection parameters for 3D plotting.
type Camera3D struct {
	AzimuthDeg   float64
	ElevationDeg float64
	Perspective  bool
	Distance     float64
	Aspect       float64
}

// ProjectedPoint2D represents a projected point in normalized [-1, 1] viewport coordinates with camera depth.
type ProjectedPoint2D struct {
	X     float64
	Y     float64
	Depth float64 // smaller means closer to the camera
	ZVal  float64 // original mathematical z value
}

// Stroke2D is the shared hidden-line-removed 2D polyline intermediate representation.
type Stroke2D struct {
	Points []ProjectedPoint2D
	Kind   string // "surface", "curve", "axis", "box"
}

// Plot3DFeature represents a CAS-detected exact critical point on a 3D surface.
type Plot3DFeature struct {
	Kind      string // "min", "max", "saddle", "critical"
	XNode     Node
	YNode     Node
	ZNode     Node
	XVal      float64
	YVal      float64
	ZVal      float64
	Projected ProjectedPoint2D
	Visible   bool
	LabelStr  string
}

// Plot3DAxisTick holds the projected endpoint, outward 2D direction, and exact symbolic value for a 3D axis tick.
type Plot3DAxisTick struct {
	Pos       ProjectedPoint2D
	OutDirX   float64
	OutDirY   float64
	ExactNode Node
	Val       float64
	Valid     bool
}

// Plot3DAxisAnnotation holds projected label and min/max tick metadata for one spatial axis.
type Plot3DAxisAnnotation struct {
	AxisName string // surface: VarX, VarY, "z"; curve: "x", "y", "z"
	LabelPos ProjectedPoint2D
	OutDirX  float64
	OutDirY  float64
	MinTick  Plot3DAxisTick
	MaxTick  Plot3DAxisTick
	Valid    bool
}

// Plot3DScene is the renderer-independent 3D plot scene containing HLR-processed 2D strokes, CAS features, and 3D axes.
type Plot3DScene struct {
	Kind                    string // "surface" or "curve"
	Expr                    Node
	VarX                    string
	VarY                    string
	XMin, XMax              float64
	YMin, YMax              float64
	ZMin, ZMax              float64
	XMinNode, XMaxNode      Node
	YMinNode, YMaxNode      Node
	ZMinNode, ZMaxNode      Node
	SpaceXMin, SpaceXMax    float64
	SpaceXMinNode           Node
	SpaceXMaxNode           Node
	Camera                  Camera3D
	Strokes                 []Stroke2D
	Features                []Plot3DFeature
	Axes                    []Plot3DAxisAnnotation
	HasNaNHoles             bool
	UnoccludedSurfaceLen    float64
	MonoColor               bool
}

const nearPlaneW = 0.2

func isFiniteFloat(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

type vec3 struct {
	x, y, z float64
}

type camVert struct {
	px, py, pz float64
	zVal       float64
	valid      bool
}

type projVert struct {
	sx, sy float64
	depth  float64
	invW   float64
	zVal   float64
	valid  bool
}

type tri2D struct {
	v0, v1, v2  projVert
	minX, maxX  float64
	minY, maxY  float64
	area2       float64
	cellID      int
	perspective bool
}

type seg3DInput struct {
	a, b       projVert
	kind       string
	ownerCellA int
	ownerCellB int
}

func evalNodeWithEnvFloat(n Node, env *Env) (Node, float64, error) {
	evalTarget := n
	if env != nil {
		ev, err := EvalWithEnv(n, env)
		if err != nil {
			return nil, 0, err
		}
		if ev != nil {
			evalTarget = ev
		}
	}
	val, err := evalNodeFloat(evalTarget)
	if err != nil {
		return nil, 0, err
	}
	return evalTarget, val, nil
}

func filterUnboundVars(vars []string, env *Env) []string {
	if env == nil {
		return vars
	}
	var unbound []string
	for _, v := range vars {
		if _, bound := env.Get(v); !bound {
			unbound = append(unbound, v)
		}
	}
	return unbound
}

func resolvePlotExprWithEnv(expr Node, env *Env, plotVars ...string) Node {
	if env == nil || expr == nil {
		return expr
	}
	exprEnv := env.Clone()
	for _, pv := range plotVars {
		if pv != "" {
			exprEnv.Delete(pv)
		}
	}
	if ev, err := EvalWithEnv(expr, exprEnv); err == nil && ev != nil {
		return ev
	}
	return expr
}

// parseDomain3DWithEnv parses either [min, max] or [var, min, max] into variable name, exact nodes, and float bounds.
func parseDomain3DWithEnv(n Node, env *Env) (string, Node, Node, float64, float64, error) {
	target := n
	switch n.(type) {
	case *ListNode, *MatrixNode:
	default:
		if env != nil {
			if ev, err := EvalWithEnv(n, env); err == nil && ev != nil {
				target = ev
			}
		}
	}

	switch v := target.(type) {
	case *ListNode:
		if len(v.Elements) == 2 {
			loNode, loVal, err := evalNodeWithEnvFloat(v.Elements[0], env)
			if err != nil {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_eval_min", err))
			}
			hiNode, hiVal, err := evalNodeWithEnvFloat(v.Elements[1], env)
			if err != nil {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_eval_max", err))
			}
			if loVal >= hiVal {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_invalid_domain"))
			}
			return "", loNode, hiNode, loVal, hiVal, nil
		}
		if len(v.Elements) == 3 {
			varNode, okV := v.Elements[0].(*VarNode)
			if !okV || strings.TrimSpace(varNode.Name) == "" {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot3d_domain_var", v.Elements[0].String()))
			}
			loNode, loVal, err := evalNodeWithEnvFloat(v.Elements[1], env)
			if err != nil {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_eval_min", err))
			}
			hiNode, hiVal, err := evalNodeWithEnvFloat(v.Elements[2], env)
			if err != nil {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_eval_max", err))
			}
			if loVal >= hiVal {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_invalid_domain"))
			}
			return varNode.Name, loNode, hiNode, loVal, hiVal, nil
		}
		return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot3d_domain_elements", len(v.Elements)))
	case *MatrixNode:
		if v.Rows == 1 && v.Cols == 2 {
			loNode, loVal, err := evalNodeWithEnvFloat(v.Data[0][0], env)
			if err != nil {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_eval_min", err))
			}
			hiNode, hiVal, err := evalNodeWithEnvFloat(v.Data[0][1], env)
			if err != nil {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_eval_max", err))
			}
			if loVal >= hiVal {
				return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_invalid_domain"))
			}
			return "", loNode, hiNode, loVal, hiVal, nil
		}
		return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_matrix_domain"))
	default:
		return "", nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_domain_list", target.String()))
	}
}

func isPlot3DStringOption(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "perspective", "persp", "ortho", "orthographic", "mono", "no_color":
		return true
	default:
		return false
	}
}

func parsePlot3DOptionsWithEnv(opts []Node, env *Env, cam *Camera3D, explicitZ *bool, zMin, zMax *float64, zMinNode, zMaxNode *Node, mono *bool) error {
	for _, opt := range opts {
		switch v := opt.(type) {
		case *StringNode:
			switch strings.ToLower(strings.TrimSpace(v.Value)) {
			case "perspective", "persp":
				cam.Perspective = true
			case "ortho", "orthographic":
				cam.Perspective = false
			case "mono", "no_color":
				*mono = true
			default:
				return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
			}
		case *ListNode:
			if len(v.Elements) < 2 {
				return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
			}
			key := ""
			if vn, ok := v.Elements[0].(*VarNode); ok {
				key = strings.ToLower(vn.Name)
			} else if sn, ok := v.Elements[0].(*StringNode); ok {
				key = strings.ToLower(strings.TrimSpace(sn.Value))
			} else {
				return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
			}
			switch key {
			case "az", "azimuth":
				if len(v.Elements) != 2 {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				_, val, err := evalNodeWithEnvFloat(v.Elements[1], env)
				if err != nil || !isFiniteFloat(val) {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				cam.AzimuthDeg = val
			case "el", "elevation":
				if len(v.Elements) != 2 {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				_, val, err := evalNodeWithEnvFloat(v.Elements[1], env)
				if err != nil || !isFiniteFloat(val) {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				if val < -90 {
					val = -90
				} else if val > 90 {
					val = 90
				}
				cam.ElevationDeg = val
			case "aspect":
				if len(v.Elements) != 2 {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				_, val, err := evalNodeWithEnvFloat(v.Elements[1], env)
				if err != nil || !isFiniteFloat(val) || val <= 0 {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				cam.Aspect = val
			case "dist", "distance":
				if len(v.Elements) != 2 {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				_, val, err := evalNodeWithEnvFloat(v.Elements[1], env)
				if err != nil || !isFiniteFloat(val) || val <= 1.8 {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				cam.Distance = val
			case "proj", "projection":
				if len(v.Elements) != 2 {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				valNode := v.Elements[1]
				if _, ok := valNode.(*StringNode); !ok && env != nil {
					if ev, err := EvalWithEnv(valNode, env); err == nil && ev != nil {
						valNode = ev
					}
				}
				if sn, ok := valNode.(*StringNode); ok {
					switch strings.ToLower(strings.TrimSpace(sn.Value)) {
					case "perspective", "persp":
						cam.Perspective = true
					case "ortho", "orthographic":
						cam.Perspective = false
					default:
						return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
					}
				} else {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
			case "z":
				if len(v.Elements) != 3 {
					return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
				}
				loNode, lo, err := evalNodeWithEnvFloat(v.Elements[1], env)
				if err != nil {
					return fmt.Errorf("%s", i18n.T("errors.plot_eval_min", err))
				}
				hiNode, hi, err := evalNodeWithEnvFloat(v.Elements[2], env)
				if err != nil {
					return fmt.Errorf("%s", i18n.T("errors.plot_eval_max", err))
				}
				if lo >= hi {
					return fmt.Errorf("%s", i18n.T("errors.plot_invalid_domain"))
				}
				*explicitZ = true
				*zMin = lo
				*zMax = hi
				*zMinNode = loNode
				*zMaxNode = hiNode
			default:
				return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", v.String()))
			}
		default:
			return fmt.Errorf("%s", i18n.T("errors.plot3d_invalid_option", opt.String()))
		}
	}
	return nil
}

func makeExactBoundNode(v float64) Node {
	if !isFiniteFloat(v) {
		return NewRationalFromBigRat(big.NewRat(0, 1))
	}
	if math.Abs(v-math.Round(v)) < 1e-9 && math.Abs(v) < 1e15 {
		return NewRationalFromBigRat(big.NewRat(int64(math.Round(v)), 1))
	}
	s := fmt.Sprintf("%.4g", v)
	if r, ok := new(big.Rat).SetString(s); ok {
		return NewRationalFromBigRat(r)
	}
	if r := new(big.Rat).SetFloat64(v); r != nil {
		return NewRationalFromBigRat(r)
	}
	return NewRationalFromBigRat(big.NewRat(0, 1))
}

func computeRobustRange(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return -1, 1
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)

	rawMin := sorted[0]
	rawMax := sorted[len(sorted)-1]
	rawSpan := rawMax - rawMin

	p05 := sorted[int(float64(len(sorted)-1)*0.05)]
	p95 := sorted[int(float64(len(sorted)-1)*0.95)]
	coreSpan := p95 - p05

	lo, hi := rawMin, rawMax
	if coreSpan > 1e-9 && rawSpan > coreSpan*12.0 {
		pad := coreSpan * 0.25
		lo = p05 - pad
		hi = p95 + pad
	} else if coreSpan <= 1e-9 && rawSpan > 50.0 {
		// Flat plane with isolated spike
		med := sorted[len(sorted)/2]
		lo = med - 1.0
		hi = med + 1.0
	}

	if hi-lo < 1e-9 {
		mid := 0.5 * (lo + hi)
		lo = mid - 1.0
		hi = mid + 1.0
	}
	return lo, hi
}

type cameraBasis struct {
	view, right, up vec3
	perspective     bool
	dist            float64
}

func newCameraBasis(cam Camera3D) cameraBasis {
	az := cam.AzimuthDeg * math.Pi / 180.0
	el := cam.ElevationDeg * math.Pi / 180.0
	cosEl, sinEl := math.Cos(el), math.Sin(el)
	cosAz, sinAz := math.Cos(az), math.Sin(az)

	view := vec3{x: cosEl * cosAz, y: cosEl * sinAz, z: sinEl}
	right := vec3{x: -sinAz, y: cosAz, z: 0}
	up := vec3{x: -sinEl * cosAz, y: -sinEl * sinAz, z: cosEl}
	dist := cam.Distance
	if dist <= 1.8 {
		dist = 4.0
	}
	return cameraBasis{
		view:        view,
		right:       right,
		up:          up,
		perspective: cam.Perspective,
		dist:        dist,
	}
}

func (cb cameraBasis) toCam(p vec3, origZ float64) camVert {
	if !isFiniteFloat(p.x) || !isFiniteFloat(p.y) || !isFiniteFloat(p.z) || !isFiniteFloat(origZ) {
		return camVert{valid: false}
	}
	px := p.x*cb.right.x + p.y*cb.right.y + p.z*cb.right.z
	py := p.x*cb.up.x + p.y*cb.up.y + p.z*cb.up.z
	pz := p.x*cb.view.x + p.y*cb.view.y + p.z*cb.view.z
	if !isFiniteFloat(px) || !isFiniteFloat(py) || !isFiniteFloat(pz) {
		return camVert{valid: false}
	}
	return camVert{
		px:    px,
		py:    py,
		pz:    pz,
		zVal:  origZ,
		valid: true,
	}
}

func (cb cameraBasis) projectCam(cv camVert) projVert {
	if !cv.valid {
		return projVert{valid: false}
	}
	// Fit unit cube [-1, 1]^3 bounding sphere (radius sqrt(3)) inside [-1, 1] viewport
	const fitScale = 0.92 / 1.7320508075688772
	if cb.perspective {
		w := cb.dist - cv.pz
		if w < nearPlaneW-1e-9 {
			return projVert{valid: false}
		}
		if w < nearPlaneW {
			w = nearPlaneW
		}
		perspScale := (cb.dist / w) * fitScale * 0.85
		sx := cv.px * perspScale
		sy := cv.py * perspScale
		if !isFiniteFloat(sx) || !isFiniteFloat(sy) {
			return projVert{valid: false}
		}
		return projVert{
			sx:    sx,
			sy:    sy,
			depth: w,
			invW:  1.0 / w,
			zVal:  cv.zVal,
			valid: true,
		}
	}
	sx := cv.px * fitScale
	sy := cv.py * fitScale
	if !isFiniteFloat(sx) || !isFiniteFloat(sy) {
		return projVert{valid: false}
	}
	// Smaller depth means closer to the camera
	return projVert{
		sx:    sx,
		sy:    sy,
		depth: -cv.pz,
		invW:  1.0,
		zVal:  cv.zVal,
		valid: true,
	}
}

func interpCamVert(a, b camVert, t float64) camVert {
	return camVert{
		px:    a.px + t*(b.px-a.px),
		py:    a.py + t*(b.py-a.py),
		pz:    a.pz + t*(b.pz-a.pz),
		zVal:  a.zVal + t*(b.zVal-a.zVal),
		valid: true,
	}
}

func clipSegmentNear3D(a, b camVert, dist, nearW float64) (camVert, camVert, bool) {
	if !a.valid || !b.valid {
		return camVert{}, camVert{}, false
	}
	wa := dist - a.pz
	wb := dist - b.pz
	inA := wa >= nearW
	inB := wb >= nearW
	if !inA && !inB {
		return camVert{}, camVert{}, false
	}
	if inA && inB {
		return a, b, true
	}
	denom := wa - wb
	if math.Abs(denom) < 1e-12 {
		return camVert{}, camVert{}, false
	}
	t := (wa - nearW) / denom
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	inter := interpCamVert(a, b, t)
	if !inA {
		return inter, b, true
	}
	return a, inter, true
}

func clipTriangleNear3D(v0, v1, v2 camVert, dist, nearW float64) [][3]camVert {
	if !v0.valid || !v1.valid || !v2.valid {
		return nil
	}
	in0 := (dist - v0.pz) >= nearW
	in1 := (dist - v1.pz) >= nearW
	in2 := (dist - v2.pz) >= nearW
	if in0 && in1 && in2 {
		return [][3]camVert{{v0, v1, v2}}
	}
	if !in0 && !in1 && !in2 {
		return nil
	}

	poly := []camVert{v0, v1, v2}
	var out []camVert
	prev := poly[len(poly)-1]
	prevW := dist - prev.pz
	prevIn := prevW >= nearW
	for _, cur := range poly {
		curW := dist - cur.pz
		curIn := curW >= nearW
		if curIn != prevIn {
			denom := prevW - curW
			if math.Abs(denom) >= 1e-12 {
				t := (prevW - nearW) / denom
				if t < 0 {
					t = 0
				} else if t > 1 {
					t = 1
				}
				out = append(out, interpCamVert(prev, cur, t))
			}
		}
		if curIn {
			out = append(out, cur)
		}
		prev = cur
		prevW = curW
		prevIn = curIn
	}

	switch len(out) {
	case 3:
		return [][3]camVert{{out[0], out[1], out[2]}}
	case 4:
		return [][3]camVert{
			{out[0], out[1], out[2]},
			{out[0], out[2], out[3]},
		}
	default:
		return nil
	}
}

func makeTri2D(v0, v1, v2 projVert, cellID int, perspective bool) (tri2D, bool) {
	if !v0.valid || !v1.valid || !v2.valid {
		return tri2D{}, false
	}
	area2 := (v1.sx-v0.sx)*(v2.sy-v0.sy) - (v2.sx-v0.sx)*(v1.sy-v0.sy)
	if !isFiniteFloat(area2) || math.Abs(area2) < 1e-10 {
		return tri2D{}, false
	}
	minX := math.Min(v0.sx, math.Min(v1.sx, v2.sx))
	maxX := math.Max(v0.sx, math.Max(v1.sx, v2.sx))
	minY := math.Min(v0.sy, math.Min(v1.sy, v2.sy))
	maxY := math.Max(v0.sy, math.Max(v1.sy, v2.sy))
	return tri2D{
		v0: v0, v1: v1, v2: v2,
		minX: minX, maxX: maxX, minY: minY, maxY: maxY,
		area2: area2, cellID: cellID, perspective: perspective,
	}, true
}

func pointInTriDepth(t tri2D, px, py float64) (float64, bool) {
	if px < t.minX || px > t.maxX || py < t.minY || py > t.maxY {
		return 0, false
	}
	w0 := ((t.v1.sx-px)*(t.v2.sy-py) - (t.v2.sx-px)*(t.v1.sy-py)) / t.area2
	w1 := ((t.v2.sx-px)*(t.v0.sy-py) - (t.v0.sx-px)*(t.v2.sy-py)) / t.area2
	w2 := 1.0 - w0 - w1
	const baryEps = -1e-6
	if w0 < baryEps || w1 < baryEps || w2 < baryEps {
		return 0, false
	}
	if t.perspective {
		invW := w0*t.v0.invW + w1*t.v1.invW + w2*t.v2.invW
		if !isFiniteFloat(invW) || invW <= 1e-12 {
			return 0, false
		}
		return 1.0 / invW, true
	}
	depth := w0*t.v0.depth + w1*t.v1.depth + w2*t.v2.depth
	return depth, true
}

type triGrid2D struct {
	n          int
	minX, maxX float64
	minY, maxY float64
	cells      [][]int
}

func (g *triGrid2D) toCellX(x float64) int {
	span := g.maxX - g.minX
	if span <= 0 {
		return 0
	}
	idx := int(math.Floor((x - g.minX) / span * float64(g.n)))
	if idx < 0 {
		return 0
	}
	if idx >= g.n {
		return g.n - 1
	}
	return idx
}

func (g *triGrid2D) toCellY(y float64) int {
	span := g.maxY - g.minY
	if span <= 0 {
		return 0
	}
	idx := int(math.Floor((y - g.minY) / span * float64(g.n)))
	if idx < 0 {
		return 0
	}
	if idx >= g.n {
		return g.n - 1
	}
	return idx
}

func buildTriGrid2D(tris []tri2D, n int) *triGrid2D {
	if len(tris) == 0 || n <= 0 {
		return &triGrid2D{n: n}
	}
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, tr := range tris {
		if tr.minX < minX {
			minX = tr.minX
		}
		if tr.maxX > maxX {
			maxX = tr.maxX
		}
		if tr.minY < minY {
			minY = tr.minY
		}
		if tr.maxY > maxY {
			maxY = tr.maxY
		}
	}
	if !isFiniteFloat(minX) || !isFiniteFloat(maxX) || !isFiniteFloat(minY) || !isFiniteFloat(maxY) {
		return &triGrid2D{n: n}
	}
	if maxX-minX < 1e-9 {
		minX -= 1e-4
		maxX += 1e-4
	}
	if maxY-minY < 1e-9 {
		minY -= 1e-4
		maxY += 1e-4
	}

	g := &triGrid2D{
		n:     n,
		minX:  minX,
		maxX:  maxX,
		minY:  minY,
		maxY:  maxY,
		cells: make([][]int, n*n),
	}
	for i, tr := range tris {
		gx0, gx1 := g.toCellX(tr.minX), g.toCellX(tr.maxX)
		gy0, gy1 := g.toCellY(tr.minY), g.toCellY(tr.maxY)
		for gy := gy0; gy <= gy1; gy++ {
			for gx := gx0; gx <= gx1; gx++ {
				idx := gy*n + gx
				g.cells[idx] = append(g.cells[idx], i)
			}
		}
	}
	return g
}

func (g *triGrid2D) cellIndices(px, py float64) []int {
	if len(g.cells) == 0 || px < g.minX || px > g.maxX || py < g.minY || py > g.maxY {
		return nil
	}
	gx := g.toCellX(px)
	gy := g.toCellY(py)
	return g.cells[gy*g.n+gx]
}

func isPointOccluded(px, py, pDepth float64, ownerA, ownerB int, tris []tri2D, grid *triGrid2D) bool {
	if len(tris) == 0 {
		return false
	}
	candidates := grid.cellIndices(px, py)
	const depthBias = 2e-4
	for _, ti := range candidates {
		tr := tris[ti]
		if tr.cellID >= 0 && (tr.cellID == ownerA || tr.cellID == ownerB) {
			continue
		}
		tz, inside := pointInTriDepth(tr, px, py)
		if inside && tz < pDepth-depthBias {
			return true
		}
	}
	return false
}

func interpSegPoint(a, b projVert, t float64, perspective bool) ProjectedPoint2D {
	x := a.sx + t*(b.sx-a.sx)
	y := a.sy + t*(b.sy-a.sy)
	if perspective {
		invW := (1.0-t)*a.invW + t*b.invW
		if isFiniteFloat(invW) && invW > 1e-12 {
			w := 1.0 / invW
			zVal := w * ((1.0-t)*(a.zVal*a.invW) + t*(b.zVal*b.invW))
			return ProjectedPoint2D{
				X:     x,
				Y:     y,
				Depth: w,
				ZVal:  zVal,
			}
		}
	}
	return ProjectedPoint2D{
		X:     x,
		Y:     y,
		Depth: a.depth + t*(b.depth-a.depth),
		ZVal:  a.zVal + t*(b.zVal-a.zVal),
	}
}

func effectiveAspect(cam Camera3D) float64 {
	if !isFiniteFloat(cam.Aspect) || cam.Aspect <= 0 {
		return 1.0
	}
	return cam.Aspect
}

func pointDist2D(a, b ProjectedPoint2D) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

func pointToSegmentDist2D(p, a, b ProjectedPoint2D) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	segLenSq := dx*dx + dy*dy
	if segLenSq <= 1e-18 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / segLenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	projX := a.X + t*dx
	projY := a.Y + t*dy
	return math.Hypot(p.X-projX, p.Y-projY)
}

func simplifyCollinearStrokePoints(pts []ProjectedPoint2D, eps float64) []ProjectedPoint2D {
	if len(pts) <= 2 {
		return pts
	}
	dmax := 0.0
	idx := 0
	p0 := pts[0]
	pn := pts[len(pts)-1]
	for i := 1; i < len(pts)-1; i++ {
		d := pointToSegmentDist2D(pts[i], p0, pn)
		if d > dmax {
			dmax = d
			idx = i
		}
	}
	if dmax > eps {
		left := simplifyCollinearStrokePoints(pts[:idx+1], eps)
		right := simplifyCollinearStrokePoints(pts[idx:], eps)
		res := make([]ProjectedPoint2D, 0, len(left)+len(right)-1)
		res = append(res, left[:len(left)-1]...)
		res = append(res, right...)
		return res
	}
	return []ProjectedPoint2D{p0, pn}
}

func reverseProjectedPoints(pts []ProjectedPoint2D) []ProjectedPoint2D {
	n := len(pts)
	out := make([]ProjectedPoint2D, n)
	for i := 0; i < n; i++ {
		out[i] = pts[n-1-i]
	}
	return out
}

func strokeDepthBounds(pts []ProjectedPoint2D) (float64, float64, bool) {
	minD, maxD := math.Inf(1), math.Inf(-1)
	found := false
	for _, pt := range pts {
		if !isFiniteFloat(pt.Depth) {
			continue
		}
		if pt.Depth < minD {
			minD = pt.Depth
		}
		if pt.Depth > maxD {
			maxD = pt.Depth
		}
		found = true
	}
	return minD, maxD, found
}

func mergeAdjacentStrokes(raw []Stroke2D, mergeTol float64) []Stroke2D {
	if len(raw) <= 1 {
		return raw
	}

	globalMinD, globalMaxD := math.Inf(1), math.Inf(-1)
	sMinD := make([]float64, len(raw))
	sMaxD := make([]float64, len(raw))
	sHasD := make([]bool, len(raw))
	for i, st := range raw {
		lo, hi, ok := strokeDepthBounds(st.Points)
		sMinD[i], sMaxD[i], sHasD[i] = lo, hi, ok
		if ok {
			if lo < globalMinD {
				globalMinD = lo
			}
			if hi > globalMaxD {
				globalMaxD = hi
			}
		}
	}
	depthSpan := 0.0
	if isFiniteFloat(globalMinD) && isFiniteFloat(globalMaxD) && globalMaxD > globalMinD {
		depthSpan = globalMaxD - globalMinD
	}
	maxStrokeDepthRange := depthSpan / 6.0

	used := make([]bool, len(raw))
	merged := make([]Stroke2D, 0, len(raw))

	for i := 0; i < len(raw); i++ {
		if used[i] || len(raw[i].Points) < 2 {
			continue
		}
		used[i] = true
		kind := raw[i].Kind
		cur := make([]ProjectedPoint2D, len(raw[i].Points))
		copy(cur, raw[i].Points)
		curMinD, curMaxD, curHasD := sMinD[i], sMaxD[i], sHasD[i]

		for {
			joined := false
			for j := 0; j < len(raw); j++ {
				if used[j] || raw[j].Kind != kind || len(raw[j].Points) < 2 {
					continue
				}
				cand := raw[j].Points
				newMinD, newMaxD := curMinD, curMaxD
				newHasD := curHasD
				if sHasD[j] {
					if !newHasD {
						newMinD, newMaxD, newHasD = sMinD[j], sMaxD[j], true
					} else {
						if sMinD[j] < newMinD {
							newMinD = sMinD[j]
						}
						if sMaxD[j] > newMaxD {
							newMaxD = sMaxD[j]
						}
					}
				}
				if depthSpan > 1e-9 && newHasD && (newMaxD-newMinD) > maxStrokeDepthRange {
					continue
				}

				curHead := cur[0]
				curTail := cur[len(cur)-1]
				candHead := cand[0]
				candTail := cand[len(cand)-1]

				switch {
				case pointDist2D(curTail, candHead) <= mergeTol:
					cur = append(cur, cand[1:]...)
					joined = true
				case pointDist2D(curTail, candTail) <= mergeTol:
					rev := reverseProjectedPoints(cand)
					cur = append(cur, rev[1:]...)
					joined = true
				case pointDist2D(curHead, candTail) <= mergeTol:
					next := make([]ProjectedPoint2D, 0, len(cand)+len(cur)-1)
					next = append(next, cand...)
					next = append(next, cur[1:]...)
					cur = next
					joined = true
				case pointDist2D(curHead, candHead) <= mergeTol:
					rev := reverseProjectedPoints(cand)
					next := make([]ProjectedPoint2D, 0, len(rev)+len(cur)-1)
					next = append(next, rev...)
					next = append(next, cur[1:]...)
					cur = next
					joined = true
				}
				if joined {
					used[j] = true
					curMinD, curMaxD, curHasD = newMinD, newMaxD, newHasD
					break
				}
			}
			if !joined {
				break
			}
		}

		merged = append(merged, Stroke2D{
			Points: cur,
			Kind:   kind,
		})
	}
	return merged
}

func clipSegmentsHLR(segments []seg3DInput, tris []tri2D, perspective bool) []Stroke2D {
	var grid *triGrid2D
	if len(tris) > 0 {
		grid = buildTriGrid2D(tris, 32)
	}
	return clipSegmentsHLRWithGrid(segments, tris, grid, perspective)
}

func clipSegmentsHLRWithGrid(segments []seg3DInput, tris []tri2D, grid *triGrid2D, perspective bool) []Stroke2D {
	var rawStrokes []Stroke2D

	for _, seg := range segments {
		if !seg.a.valid || !seg.b.valid {
			continue
		}
		dx := seg.b.sx - seg.a.sx
		dy := seg.b.sy - seg.a.sy
		segLen := math.Hypot(dx, dy)
		if !isFiniteFloat(segLen) || segLen < 1e-9 {
			continue
		}

		steps := int(math.Ceil(segLen / 0.008))
		if steps < 4 {
			steps = 4
		}
		if steps > 48 {
			steps = 48
		}
		if len(tris) == 0 {
			steps = 1
		}

		var cur []ProjectedPoint2D
		flush := func() {
			if len(cur) >= 2 {
				rawStrokes = append(rawStrokes, Stroke2D{
					Points: cur,
					Kind:   seg.kind,
				})
			}
			cur = nil
		}

		for s := 0; s < steps; s++ {
			t0 := float64(s) / float64(steps)
			t1 := float64(s+1) / float64(steps)
			tm := 0.5 * (t0 + t1)

			pm := interpSegPoint(seg.a, seg.b, tm, perspective)
			if isPointOccluded(pm.X, pm.Y, pm.Depth, seg.ownerCellA, seg.ownerCellB, tris, grid) {
				flush()
				continue
			}

			p0 := interpSegPoint(seg.a, seg.b, t0, perspective)
			p1 := interpSegPoint(seg.a, seg.b, t1, perspective)
			if len(cur) == 0 {
				cur = append(cur, p0)
			}
			cur = append(cur, p1)
		}
		flush()
	}

	merged := mergeAdjacentStrokes(rawStrokes, 1e-4)
	strokes := make([]Stroke2D, 0, len(merged))
	for _, st := range merged {
		pts := simplifyCollinearStrokePoints(st.Points, 1e-5)
		if len(pts) < 2 {
			continue
		}
		totalLen := 0.0
		for i := 0; i < len(pts)-1; i++ {
			totalLen += pointDist2D(pts[i], pts[i+1])
		}
		minKeep := 0.015
		if st.Kind == "box" || st.Kind == "axis" || len(tris) == 0 {
			minKeep = 1e-6
		}
		if totalLen >= minKeep {
			strokes = append(strokes, Stroke2D{
				Points: pts,
				Kind:   st.Kind,
			})
		}
	}
	return strokes
}

// selectAxisHighs determines which of each pair of opposite unit-cube faces (-1 vs +1)
// is closer to the camera, following Matplotlib mplot3d's parallel-plane depth comparison
// with deterministic edge-on tie-breaking.
func selectAxisHighs(cb cameraBasis) [3]bool {
	corners := [8]vec3{
		{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
		{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
	}
	var pz [8]float64
	for i, pt := range corners {
		cv := cb.toCam(pt, pt.z)
		pz[i] = cv.pz
	}
	// Parallel face vertex index pairs: (min-face, max-face) for X, Y, Z.
	// Larger camera pz means closer to the camera.
	facePairs := [3][2][4]int{
		{{0, 3, 4, 7}, {1, 2, 5, 6}}, // X = -1 vs X = +1
		{{0, 1, 4, 5}, {2, 3, 6, 7}}, // Y = -1 vs Y = +1
		{{0, 1, 2, 3}, {4, 5, 6, 7}}, // Z = -1 vs Z = +1
	}
	var means0, means1 [3]float64
	var highs [3]bool
	var equals [3]bool
	eqCount := 0
	for axis := 0; axis < 3; axis++ {
		m0, m1 := 0.0, 0.0
		for k := 0; k < 4; k++ {
			m0 += pz[facePairs[axis][0][k]]
			m1 += pz[facePairs[axis][1][k]]
		}
		m0 *= 0.25
		m1 *= 0.25
		means0[axis], means1[axis] = m0, m1
		highs[axis] = m0 < m1
		if math.Abs(m0-m1) <= 1e-9 {
			equals[axis] = true
			eqCount++
		}
	}
	// Edge-on view tie-breaking (mplot3d axis3d.py L321-330):
	if eqCount >= 2 {
		vertical := 2
		for axis := 0; axis < 3; axis++ {
			if !equals[axis] {
				vertical = axis
				break
			}
		}
		switch vertical {
		case 2: // looking directly at XY plane
			highs = [3]bool{true, true, highs[2]}
		case 1: // looking directly at XZ plane
			highs = [3]bool{true, highs[1], false}
		case 0: // looking directly at YZ plane
			highs = [3]bool{highs[0], false, false}
		}
	}
	return highs
}

func axisSign(high bool) (back, front float64) {
	if high {
		return -1.0, 1.0
	}
	return 1.0, -1.0
}

type axisEdgeEndpoints struct {
	axisIdx int // 0=X, 1=Y, 2=Z
	pMin    vec3
	pMax    vec3
}

// selectViewAxisEdges chooses 3 outer bounding-box edges (one per X/Y/Z axis) aligned
// with the back-facing panes for the current camera orientation.
func selectViewAxisEdges(cb cameraBasis) [3]axisEdgeEndpoints {
	highs := selectAxisHighs(cb)
	backX, frontX := axisSign(highs[0])
	_, frontY := axisSign(highs[1])
	backZ, _ := axisSign(highs[2])
	return [3]axisEdgeEndpoints{
		{axisIdx: 0, pMin: vec3{x: -1, y: frontY, z: backZ}, pMax: vec3{x: 1, y: frontY, z: backZ}},
		{axisIdx: 1, pMin: vec3{x: frontX, y: -1, z: backZ}, pMax: vec3{x: frontX, y: 1, z: backZ}},
		{axisIdx: 2, pMin: vec3{x: backX, y: frontY, z: -1}, pMax: vec3{x: backX, y: frontY, z: 1}},
	}
}

func sameVec3Approx(a, b vec3) bool {
	return math.Abs(a.x-b.x) < 1e-6 && math.Abs(a.y-b.y) < 1e-6 && math.Abs(a.z-b.z) < 1e-6
}

func isSelectedAxisEdge(a, b vec3, axisEdges [3]axisEdgeEndpoints) bool {
	for _, ae := range axisEdges {
		if (sameVec3Approx(a, ae.pMin) && sameVec3Approx(b, ae.pMax)) ||
			(sameVec3Approx(a, ae.pMax) && sameVec3Approx(b, ae.pMin)) {
			return true
		}
	}
	return false
}

func appendBoundingBoxSegments(segments []seg3DInput, cb cameraBasis) []seg3DInput {
	corners := [8]vec3{
		{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
		{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
	}
	axisEdges := selectViewAxisEdges(cb)
	var c [8]camVert
	for i, pt := range corners {
		c[i] = cb.toCam(pt, pt.z)
	}
	edges := [12][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0},
		{4, 5}, {5, 6}, {6, 7}, {7, 4},
		{0, 4}, {1, 5}, {2, 6}, {3, 7},
	}
	for _, e := range edges {
		ca, cbv := c[e[0]], c[e[1]]
		if cb.perspective {
			var ok bool
			ca, cbv, ok = clipSegmentNear3D(ca, cbv, cb.dist, nearPlaneW)
			if !ok {
				continue
			}
		}
		pa := cb.projectCam(ca)
		pb := cb.projectCam(cbv)
		if !pa.valid || !pb.valid {
			continue
		}
		kind := "box"
		if isSelectedAxisEdge(corners[e[0]], corners[e[1]], axisEdges) {
			kind = "axis"
		}
		segments = append(segments, seg3DInput{
			a: pa, b: pb, kind: kind, ownerCellA: -1, ownerCellB: -1,
		})
	}
	return segments
}

func computeOutwardDir2D(cb cameraBasis, pt3 vec3, axisIdx int, pt2 ProjectedPoint2D) (float64, float64) {
	// Push outward in 3D space from the box center along the two non-axis coordinates
	// (equivalent to Matplotlib mplot3d _move_from_center with _axmask).
	const push = 0.22
	out3 := pt3
	if axisIdx != 0 {
		if pt3.x >= 0 {
			out3.x += push
		} else {
			out3.x -= push
		}
	}
	if axisIdx != 1 {
		if pt3.y >= 0 {
			out3.y += push
		} else {
			out3.y -= push
		}
	}
	if axisIdx != 2 {
		if pt3.z >= 0 {
			out3.z += push
		} else {
			out3.z -= push
		}
	}
	if pOut := cb.projectCam(cb.toCam(out3, out3.z)); pOut.valid {
		dx := pOut.sx - pt2.X
		dy := pOut.sy - pt2.Y
		if n := math.Hypot(dx, dy); isFiniteFloat(n) && n > 1e-6 {
			return dx / n, dy / n
		}
	}
	// Fallback: radial direction from projected cube center (0, 0, 0) in 2D
	if cProj := cb.projectCam(cb.toCam(vec3{}, 0)); cProj.valid {
		dx := pt2.X - cProj.sx
		dy := pt2.Y - cProj.sy
		if n := math.Hypot(dx, dy); isFiniteFloat(n) && n > 1e-6 {
			return dx / n, dy / n
		}
	}
	if axisIdx == 2 {
		return 1.0, 0.0
	}
	return 0.0, -1.0
}

func buildAxisAnnotations3D(
	cb cameraBasis,
	names [3]string,
	minNodes [3]Node,
	maxNodes [3]Node,
	minVals [3]float64,
	maxVals [3]float64,
) []Plot3DAxisAnnotation {
	edges := selectViewAxisEdges(cb)
	out := make([]Plot3DAxisAnnotation, 3)
	for i := 0; i < 3; i++ {
		ae := edges[i]
		mid3 := vec3{
			x: 0.5 * (ae.pMin.x + ae.pMax.x),
			y: 0.5 * (ae.pMin.y + ae.pMax.y),
			z: 0.5 * (ae.pMin.z + ae.pMax.z),
		}
		ann := Plot3DAxisAnnotation{
			AxisName: names[i],
		}
		if pMid := cb.projectCam(cb.toCam(mid3, mid3.z)); pMid.valid {
			pos := ProjectedPoint2D{X: pMid.sx, Y: pMid.sy, Depth: pMid.depth, ZVal: mid3.z}
			ox, oy := computeOutwardDir2D(cb, mid3, ae.axisIdx, pos)
			ann.LabelPos = pos
			ann.OutDirX = ox
			ann.OutDirY = oy
			ann.Valid = true
		}
		if pLo := cb.projectCam(cb.toCam(ae.pMin, ae.pMin.z)); pLo.valid {
			pos := ProjectedPoint2D{X: pLo.sx, Y: pLo.sy, Depth: pLo.depth, ZVal: ae.pMin.z}
			ox, oy := computeOutwardDir2D(cb, ae.pMin, ae.axisIdx, pos)
			ann.MinTick = Plot3DAxisTick{
				Pos:       pos,
				OutDirX:   ox,
				OutDirY:   oy,
				ExactNode: minNodes[i],
				Val:       minVals[i],
				Valid:     true,
			}
		}
		if pHi := cb.projectCam(cb.toCam(ae.pMax, ae.pMax.z)); pHi.valid {
			pos := ProjectedPoint2D{X: pHi.sx, Y: pHi.sy, Depth: pHi.depth, ZVal: ae.pMax.z}
			ox, oy := computeOutwardDir2D(cb, ae.pMax, ae.axisIdx, pos)
			ann.MaxTick = Plot3DAxisTick{
				Pos:       pos,
				OutDirX:   ox,
				OutDirY:   oy,
				ExactNode: maxNodes[i],
				Val:       maxVals[i],
				Valid:     true,
			}
		}
		out[i] = ann
	}
	return out
}

func plot3dHasVarName(vars []string, target string) bool {
	for _, v := range vars {
		if v == target {
			return true
		}
	}
	return false
}

func onlyAllowedVars(vars []string, vx, vy string) bool {
	for _, v := range vars {
		if v != vx && v != vy {
			return false
		}
	}
	return true
}

func solveRealRoots1D(expr Node, varName string) []Node {
	if expr == nil || isZeroNode(expr) {
		return nil
	}
	solList, err := solveEquation(expr, varName)
	if err != nil || solList == nil {
		return nil
	}
	var elems []Node
	if list, ok := solList.(*ListNode); ok {
		elems = list.Elements
	} else {
		elems = []Node{solList}
	}
	var out []Node
	for _, r := range elems {
		if r == nil {
			continue
		}
		if ev, err := Eval(r); err == nil && ev != nil {
			r = ev
		}
		if val, err := evalNodeFloat(r); err == nil && isFiniteFloat(val) {
			out = append(out, r)
		}
	}
	return out
}

func evalTwoVarNode(expr Node, vx string, xNode Node, vy string, yNode Node) (Node, float64, bool) {
	if expr == nil {
		return nil, 0, false
	}
	sub := substituteVar(substituteVar(expr, vx, xNode), vy, yNode)
	ev, err := Eval(sub)
	if err != nil || ev == nil {
		return nil, 0, false
	}
	val, err := evalNodeFloat(ev)
	if err != nil || !isFiniteFloat(val) {
		return ev, 0, false
	}
	return ev, val, true
}

type critPairCandidate struct {
	xNode, yNode Node
	xVal, yVal   float64
}

func solveGradientSystem2D(fx, fy Node, vx, vy string) []critPairCandidate {
	if fx == nil || fy == nil || isZeroNode(fx) || isZeroNode(fy) {
		return nil
	}
	fxVars := ExtractFreeVariables(fx)
	fyVars := ExtractFreeVariables(fy)
	if !onlyAllowedVars(fxVars, vx, vy) || !onlyAllowedVars(fyVars, vx, vy) {
		return nil
	}
	hasVxInFx := plot3dHasVarName(fxVars, vx)
	hasVyInFx := plot3dHasVarName(fxVars, vy)
	hasVxInFy := plot3dHasVarName(fyVars, vx)
	hasVyInFy := plot3dHasVarName(fyVars, vy)

	// Non-zero constant partial derivative => no critical points.
	if (!hasVxInFx && !hasVyInFx) || (!hasVxInFy && !hasVyInFy) {
		return nil
	}

	const maxCandidates = 32
	var cands []critPairCandidate
	addCand := func(xn, yn Node) {
		if len(cands) >= maxCandidates {
			return
		}
		xv, errX := evalNodeFloat(xn)
		yv, errY := evalNodeFloat(yn)
		if errX != nil || errY != nil || !isFiniteFloat(xv) || !isFiniteFloat(yv) {
			return
		}
		cands = append(cands, critPairCandidate{xNode: xn, yNode: yn, xVal: xv, yVal: yv})
	}

	// Strategy 1: Separable system (fx depends only on vx, fy depends only on vy).
	if !hasVyInFx && !hasVxInFy {
		xs := solveRealRoots1D(fx, vx)
		ys := solveRealRoots1D(fy, vy)
		for _, xn := range xs {
			for _, yn := range ys {
				addCand(xn, yn)
			}
		}
		return cands
	}

	// Strategy 2: One equation is univariate; solve it first and back-substitute.
	if !hasVyInFx {
		for _, xn := range solveRealRoots1D(fx, vx) {
			subFy, err := Eval(substituteVar(fy, vx, xn))
			if err != nil || subFy == nil || isZeroNode(subFy) {
				continue // positive-dimensional branch or invalid
			}
			for _, yn := range solveRealRoots1D(subFy, vy) {
				addCand(xn, yn)
			}
		}
		return cands
	}
	if !hasVxInFy {
		for _, yn := range solveRealRoots1D(fy, vy) {
			subFx, err := Eval(substituteVar(fx, vy, yn))
			if err != nil || subFx == nil || isZeroNode(subFx) {
				continue // positive-dimensional branch or invalid
			}
			for _, xn := range solveRealRoots1D(subFx, vx) {
				addCand(xn, yn)
			}
		}
		return cands
	}

	// Strategy 3: Coupled polynomial system via lex Groebner basis elimination
	// with SymPy polysys.py zero-dimensionality guards.
	gbNode, err := EvalGroebner(
		NewList([]Node{fx, fy}),
		NewList([]Node{NewVar(vx), NewVar(vy)}),
		NewStringNode("lex"),
		nil,
	)
	if err != nil || gbNode == nil {
		return nil
	}
	gbList, ok := gbNode.(*ListNode)
	if !ok || len(gbList.Elements) == 0 {
		return nil
	}
	basis := gbList.Elements
	// Constant ground basis [1] => inconsistent system (no solutions).
	if len(basis) == 1 && len(ExtractFreeVariables(basis[0])) == 0 {
		return nil
	}
	// Fewer generators than variables => positive-dimensional (non-isolated critical locus).
	if len(basis) < 2 {
		return nil
	}

	// Reduced zero-dimensional lex Groebner basis must contain exactly one univariate polynomial in vy.
	var uniY Node
	uniYIdx := -1
	uniCount := 0
	for i, b := range basis {
		bVars := ExtractFreeVariables(b)
		if !plot3dHasVarName(bVars, vx) && plot3dHasVarName(bVars, vy) {
			uniY = b
			uniYIdx = i
			uniCount++
		}
	}
	if uniCount != 1 || uniY == nil {
		return nil
	}

	for _, yn := range solveRealRoots1D(uniY, vy) {
		for i, b := range basis {
			if i == uniYIdx {
				continue
			}
			subB, err := Eval(substituteVar(b, vy, yn))
			if err != nil || subB == nil || isZeroNode(subB) {
				continue
			}
			if !plot3dHasVarName(ExtractFreeVariables(subB), vx) {
				continue
			}
			for _, xn := range solveRealRoots1D(subB, vx) {
				addCand(xn, yn)
			}
			break
		}
	}
	return cands
}

func isFeatureOccluded(
	px, py, pDepth float64,
	nx, ny float64,
	gridN int,
	tris []tri2D,
	grid *triGrid2D,
) bool {
	if len(tris) == 0 || grid == nil {
		return false
	}
	candidates := grid.cellIndices(px, py)
	// Exclude immediate mesh cells adjacent to the critical point itself to avoid self-occlusion.
	ci := int(math.Floor((nx + 1.0) * 0.5 * float64(gridN)))
	cj := int(math.Floor((ny + 1.0) * 0.5 * float64(gridN)))
	const selfBias = 1.5e-2
	for _, ti := range candidates {
		tr := tris[ti]
		if tr.cellID >= 0 {
			tcI := tr.cellID / gridN
			tcJ := tr.cellID % gridN
			if tcI >= ci-1 && tcI <= ci+1 && tcJ >= cj-1 && tcJ <= cj+1 {
				continue
			}
		}
		tz, inside := pointInTriDepth(tr, px, py)
		if inside && tz < pDepth-selfBias {
			return true
		}
	}
	return false
}

// findSurfaceCriticalPoints3D detects exact isolated critical points of z = f(x, y) within [xMin, xMax] x [yMin, yMax]
// and classifies them using the second partial derivative test (Hessian determinant).
func findSurfaceCriticalPoints3D(
	expr Node,
	vx, vy string,
	xMin, xMax, yMin, yMax float64,
) []Plot3DFeature {
	fx, errX := differentiate(expr, vx)
	fy, errY := differentiate(expr, vy)
	if errX != nil || errY != nil || fx == nil || fy == nil {
		return nil
	}
	if ev, err := Eval(fx); err == nil && ev != nil {
		fx = ev
	}
	if ev, err := Eval(fy); err == nil && ev != nil {
		fy = ev
	}
	cands := solveGradientSystem2D(fx, fy, vx, vy)
	if len(cands) == 0 {
		return nil
	}

	fxx, errXX := differentiate(fx, vx)
	fyy, errYY := differentiate(fy, vy)
	fxy, errXY := differentiate(fx, vy)
	hasHessian := errXX == nil && errYY == nil && errXY == nil && fxx != nil && fyy != nil && fxy != nil
	if hasHessian {
		if ev, err := Eval(fxx); err == nil && ev != nil {
			fxx = ev
		}
		if ev, err := Eval(fyy); err == nil && ev != nil {
			fyy = ev
		}
		if ev, err := Eval(fxy); err == nil && ev != nil {
			fxy = ev
		}
	}

	const domEps = 1e-9
	const resEps = 1e-8
	const dupEps = 1e-6
	const maxFeatures = 16

	var features []Plot3DFeature
	for _, c := range cands {
		if len(features) >= maxFeatures {
			break
		}
		if c.xVal < xMin-domEps || c.xVal > xMax+domEps || c.yVal < yMin-domEps || c.yVal > yMax+domEps {
			continue
		}
		// Double-check gradient vanishing at candidate (x0, y0)
		fxAt, fxVal, okFx := evalTwoVarNode(fx, vx, c.xNode, vy, c.yNode)
		fyAt, fyVal, okFy := evalTwoVarNode(fy, vx, c.xNode, vy, c.yNode)
		if !okFx || !okFy {
			continue
		}
		if !(isZeroNode(fxAt) || math.Abs(fxVal) < resEps) || !(isZeroNode(fyAt) || math.Abs(fyVal) < resEps) {
			continue
		}

		// Deduplicate numerically identical solutions
		dup := false
		for _, prev := range features {
			if math.Abs(prev.XVal-c.xVal) < dupEps && math.Abs(prev.YVal-c.yVal) < dupEps {
				dup = true
				break
			}
		}
		if dup {
			continue
		}

		zNode, zVal, okZ := evalTwoVarNode(expr, vx, c.xNode, vy, c.yNode)
		if !okZ {
			continue
		}

		kind := "critical"
		if hasHessian {
			hxxNode, hxxVal, okXX := evalTwoVarNode(fxx, vx, c.xNode, vy, c.yNode)
			hyyNode, hyyVal, okYY := evalTwoVarNode(fyy, vx, c.xNode, vy, c.yNode)
			hxyNode, hxyVal, okXY := evalTwoVarNode(fxy, vx, c.xNode, vy, c.yNode)
			if okXX && okYY && okXY {
				detExpr := NewAdd([]Node{
					NewMul([]Node{hxxNode, hyyNode}),
					NewMul([]Node{mustRational(-1, 1), hxyNode, hxyNode}),
				})
				detNode, errDet := Eval(detExpr)
				detVal := hxxVal*hyyVal - hxyVal*hxyVal
				exactZeroDet := errDet == nil && detNode != nil && isZeroNode(detNode)
				if errDet == nil && detNode != nil {
					if v, errF := evalNodeFloat(detNode); errF == nil && isFiniteFloat(v) {
						detVal = v
					}
				}
				const hessEps = 1e-10
				if !exactZeroDet && math.Abs(detVal) >= hessEps {
					if detVal < 0 {
						kind = "saddle"
					} else if hxxVal > hessEps {
						kind = "min"
					} else if hxxVal < -hessEps {
						kind = "max"
					}
				}
			}
		}

		features = append(features, Plot3DFeature{
			Kind:     kind,
			XNode:    c.xNode,
			YNode:    c.yNode,
			ZNode:    zNode,
			XVal:     c.xVal,
			YVal:     c.yVal,
			ZVal:     zVal,
			LabelStr: fmt.Sprintf("[%s, %s, %s]", Format(c.xNode), Format(c.yNode), Format(zNode)),
		})
	}

	sort.SliceStable(features, func(i, j int) bool {
		if math.Abs(features[i].XVal-features[j].XVal) > dupEps {
			return features[i].XVal < features[j].XVal
		}
		if math.Abs(features[i].YVal-features[j].YVal) > dupEps {
			return features[i].YVal < features[j].YVal
		}
		return features[i].ZVal < features[j].ZVal
	})
	return features
}

// projectSurfaceFeatures3D filters critical points against [zMin, zMax], projects them to 2D, and runs HLR visibility checks.
func projectSurfaceFeatures3D(
	raw []Plot3DFeature,
	xMin, xMax, yMin, yMax, zMin, zMax float64,
	cb cameraBasis,
	gridN int,
	tris []tri2D,
	grid *triGrid2D,
) []Plot3DFeature {
	if len(raw) == 0 {
		return nil
	}
	xSpan := xMax - xMin
	ySpan := yMax - yMin
	zSpan := zMax - zMin
	const domEps = 1e-9
	out := make([]Plot3DFeature, 0, len(raw))
	for _, f := range raw {
		if f.ZVal < zMin-domEps || f.ZVal > zMax+domEps {
			continue
		}
		nx := 2.0*(f.XVal-xMin)/xSpan - 1.0
		ny := 2.0*(f.YVal-yMin)/ySpan - 1.0
		nz := 2.0*(f.ZVal-zMin)/zSpan - 1.0
		cv := cb.toCam(vec3{x: nx, y: ny, z: nz}, f.ZVal)
		pv := cb.projectCam(cv)
		if pv.valid {
			f.Projected = ProjectedPoint2D{X: pv.sx, Y: pv.sy, Depth: pv.depth, ZVal: f.ZVal}
			f.Visible = !isFeatureOccluded(pv.sx, pv.sy, pv.depth, nx, ny, gridN, tris, grid)
		}
		out = append(out, f)
	}
	return out
}

// BuildPlot3DScene parses plot3d arguments without an explicit environment and executes the single continuous 2D HLR pass.
func BuildPlot3DScene(args []Node) (*Plot3DScene, error) {
	return BuildPlot3DSceneWithEnv(args, nil)
}

// BuildPlot3DSceneWithEnv parses plot3d arguments with env evaluation and executes the single continuous 2D HLR pass.
func BuildPlot3DSceneWithEnv(args []Node, env *Env) (*Plot3DScene, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot3d_args_count", len(args)))
	}

	cam := Camera3D{
		AzimuthDeg:   35.0,
		ElevationDeg: 25.0,
		Perspective:  false,
		Distance:     4.0,
		Aspect:       1.0,
	}
	var explicitZ bool
	var explicitZMin, explicitZMax float64
	var explicitZMinNode, explicitZMaxNode Node
	var mono bool

	// Curve mode: first arg is a ListNode of 3 expressions [x(t), y(t), z(t)]
	if listNode, ok := args[0].(*ListNode); ok {
		if len(listNode.Elements) != 3 {
			return nil, fmt.Errorf("%s", i18n.T("errors.plot3d_curve_dim", len(listNode.Elements)))
		}
		tVar, tMinNode, tMaxNode, tMin, tMax, err := parseDomain3DWithEnv(args[1], env)
		if err != nil {
			return nil, err
		}
		if tVar == "" {
			var allCurveVars []string
			for _, elem := range listNode.Elements {
				allCurveVars = append(allCurveVars, ExtractFreeVariables(elem)...)
			}
			unbound := filterUnboundVars(allCurveVars, env)
			if len(unbound) > 0 {
				tVar = unbound[0]
			} else {
				tVar = findFreeVariable(listNode.Elements[0])
				if tVar == "x" {
					if v1 := ExtractFreeVariables(listNode.Elements[1]); len(v1) > 0 {
						tVar = v1[0]
					} else if v2 := ExtractFreeVariables(listNode.Elements[2]); len(v2) > 0 {
						tVar = v2[0]
					} else if v0 := ExtractFreeVariables(listNode.Elements[0]); len(v0) > 0 {
						tVar = v0[0]
					} else {
						tVar = "t"
					}
				}
			}
		}
		if err := parsePlot3DOptionsWithEnv(args[2:], env, &cam, &explicitZ, &explicitZMin, &explicitZMax, &explicitZMinNode, &explicitZMaxNode, &mono); err != nil {
			return nil, err
		}
		resolvedElems := make([]Node, 3)
		for i := 0; i < 3; i++ {
			resolvedElems[i] = resolvePlotExprWithEnv(listNode.Elements[i], env, tVar)
		}
		resolvedList := NewList(resolvedElems)
		return buildParametricCurve3DScene(resolvedList, tVar, tMinNode, tMaxNode, tMin, tMax, cam, explicitZ, explicitZMin, explicitZMax, explicitZMinNode, explicitZMaxNode, mono)
	}

	// Surface mode: plot3d(expr, x_range, y_range, [opts...])
	if len(args) < 3 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot3d_surface_args", len(args)))
	}
	expr := args[0]
	vx, xMinNode, xMaxNode, xMin, xMax, err := parseDomain3DWithEnv(args[1], env)
	if err != nil {
		return nil, err
	}
	vy, yMinNode, yMaxNode, yMin, yMax, err := parseDomain3DWithEnv(args[2], env)
	if err != nil {
		return nil, err
	}

	freeVars := ExtractFreeVariables(expr)
	candVars := filterUnboundVars(freeVars, env)
	if len(candVars) == 0 {
		candVars = freeVars
	}
	if vx == "" && vy == "" {
		switch len(candVars) {
		case 0:
			vx, vy = "x", "y"
		case 1:
			if candVars[0] == "y" {
				vx, vy = "x", "y"
			} else {
				vx, vy = candVars[0], "y"
			}
		default:
			vx, vy = candVars[0], candVars[1]
		}
	} else if vx == "" {
		vx = "x"
		if vx == vy {
			vx = "t"
		}
		for _, fv := range candVars {
			if fv != vy {
				vx = fv
				break
			}
		}
	} else if vy == "" {
		vy = "y"
		if vy == vx {
			vy = "t"
		}
		for _, fv := range candVars {
			if fv != vx {
				vy = fv
				break
			}
		}
	}

	if vx == vy {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot3d_duplicate_vars", vx))
	}

	if err := parsePlot3DOptionsWithEnv(args[3:], env, &cam, &explicitZ, &explicitZMin, &explicitZMax, &explicitZMinNode, &explicitZMaxNode, &mono); err != nil {
		return nil, err
	}
	resolvedExpr := resolvePlotExprWithEnv(expr, env, vx, vy)
	return buildSurface3DScene(resolvedExpr, vx, vy, xMinNode, xMaxNode, xMin, xMax, yMinNode, yMaxNode, yMin, yMax, cam, explicitZ, explicitZMin, explicitZMax, explicitZMinNode, explicitZMaxNode, mono)
}

func buildSurface3DScene(
	expr Node,
	vx, vy string,
	xMinNode, xMaxNode Node, xMin, xMax float64,
	yMinNode, yMaxNode Node, yMin, yMax float64,
	cam Camera3D,
	explicitZ bool, zMin, zMax float64, zMinNode, zMaxNode Node,
	mono bool,
) (*Plot3DScene, error) {
	const gridN = 30
	const isoStride = 3

	zGrid := make([][]float64, gridN+1)
	validGrid := make([][]bool, gridN+1)
	var finiteZ []float64
	hasHoles := false

	vars := map[string]float64{vx: 0, vy: 0}
	for i := 0; i <= gridN; i++ {
		zGrid[i] = make([]float64, gridN+1)
		validGrid[i] = make([]bool, gridN+1)
		x := xMin + (xMax-xMin)*float64(i)/float64(gridN)
		vars[vx] = x
		for j := 0; j <= gridN; j++ {
			y := yMin + (yMax-yMin)*float64(j)/float64(gridN)
			vars[vy] = y
			z, ok := evalSampleFloatVars(expr, vars)
			if !ok {
				hasHoles = true
				continue
			}
			zGrid[i][j] = z
			validGrid[i][j] = true
			finiteZ = append(finiteZ, z)
		}
	}

	if len(finiteZ) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot3d_no_valid_points"))
	}

	rawFeatures := findSurfaceCriticalPoints3D(expr, vx, vy, xMin, xMax, yMin, yMax)

	if !explicitZ {
		zMin, zMax = computeRobustRange(finiteZ)
		zMinNode = makeExactBoundNode(zMin)
		zMaxNode = makeExactBoundNode(zMax)
		// Ensure true CAS-detected interior extrema are not clipped by discrete grid sampling gaps.
		baseSpan := zMax - zMin
		if baseSpan < 1e-6 {
			baseSpan = 1.0
		}
		for _, f := range rawFeatures {
			if f.ZVal < zMin && (zMin-f.ZVal) <= baseSpan*2.0 {
				zMin = f.ZVal
				zMinNode = f.ZNode
			}
			if f.ZVal > zMax && (f.ZVal-zMax) <= baseSpan*2.0 {
				zMax = f.ZVal
				zMaxNode = f.ZNode
			}
		}
	}

	xSpan := xMax - xMin
	ySpan := yMax - yMin
	zSpan := zMax - zMin
	if !isFiniteFloat(xSpan) || xSpan <= 0 || !isFiniteFloat(ySpan) || ySpan <= 0 || !isFiniteFloat(zSpan) || zSpan <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot3d_no_valid_points"))
	}
	cb := newCameraBasis(cam)

	camGrid := make([][]camVert, gridN+1)
	inSlab := make([][]bool, gridN+1)
	for i := 0; i <= gridN; i++ {
		camGrid[i] = make([]camVert, gridN+1)
		inSlab[i] = make([]bool, gridN+1)
		nx := 2.0*float64(i)/float64(gridN) - 1.0
		for j := 0; j <= gridN; j++ {
			if !validGrid[i][j] {
				continue
			}
			z := zGrid[i][j]
			if z < zMin || z > zMax {
				hasHoles = true
				continue
			}
			ny := 2.0*float64(j)/float64(gridN) - 1.0
			nz := 2.0*(z-zMin)/zSpan - 1.0
			if !isFiniteFloat(nx) || !isFiniteFloat(ny) || !isFiniteFloat(nz) {
				hasHoles = true
				continue
			}
			cv := cb.toCam(vec3{x: nx, y: ny, z: nz}, z)
			if !cv.valid {
				hasHoles = true
				continue
			}
			inSlab[i][j] = true
			camGrid[i][j] = cv
		}
	}

	jumpThresh := zSpan * 0.85
	var tris []tri2D
	cellValid := make([][]bool, gridN)
	appendTri := func(a, b, c camVert, cid int) {
		if cb.perspective {
			clipped := clipTriangleNear3D(a, b, c, cb.dist, nearPlaneW)
			for _, ct := range clipped {
				pa := cb.projectCam(ct[0])
				pb := cb.projectCam(ct[1])
				pc := cb.projectCam(ct[2])
				if t, ok := makeTri2D(pa, pb, pc, cid, true); ok {
					tris = append(tris, t)
				}
			}
			return
		}
		pa := cb.projectCam(a)
		pb := cb.projectCam(b)
		pc := cb.projectCam(c)
		if t, ok := makeTri2D(pa, pb, pc, cid, false); ok {
			tris = append(tris, t)
		}
	}

	for i := 0; i < gridN; i++ {
		cellValid[i] = make([]bool, gridN)
		for j := 0; j < gridN; j++ {
			if !inSlab[i][j] || !inSlab[i+1][j] || !inSlab[i][j+1] || !inSlab[i+1][j+1] {
				continue
			}
			z00, z10, z01, z11 := zGrid[i][j], zGrid[i+1][j], zGrid[i][j+1], zGrid[i+1][j+1]
			if math.Abs(z10-z00) > jumpThresh || math.Abs(z01-z00) > jumpThresh ||
				math.Abs(z11-z10) > jumpThresh || math.Abs(z11-z01) > jumpThresh ||
				math.Abs(z11-z00) > jumpThresh || math.Abs(z10-z01) > jumpThresh {
				hasHoles = true
				continue
			}
			c00, c10, c01, c11 := camGrid[i][j], camGrid[i+1][j], camGrid[i][j+1], camGrid[i+1][j+1]
			if !c00.valid || !c10.valid || !c01.valid || !c11.valid {
				continue
			}
			cellValid[i][j] = true
			cid := i*gridN + j
			if (i+j)%2 == 0 {
				appendTri(c00, c10, c11, cid)
				appendTri(c00, c11, c01, cid)
			} else {
				appendTri(c00, c10, c01, cid)
				appendTri(c10, c11, c01, cid)
			}
		}
	}

	var segments []seg3DInput
	unoccludedLen := 0.0
	addSurfaceSeg := func(ca, cbv camVert, ownerA, ownerB int) {
		if cb.perspective {
			var ok bool
			ca, cbv, ok = clipSegmentNear3D(ca, cbv, cb.dist, nearPlaneW)
			if !ok {
				return
			}
		}
		a := cb.projectCam(ca)
		b := cb.projectCam(cbv)
		if !a.valid || !b.valid {
			return
		}
		segLen := math.Hypot(b.sx-a.sx, b.sy-a.sy)
		if !isFiniteFloat(segLen) {
			return
		}
		unoccludedLen += segLen
		segments = append(segments, seg3DInput{a: a, b: b, kind: "surface", ownerCellA: ownerA, ownerCellB: ownerB})
	}

	// Drawn X-constant isolines
	for i := 0; i <= gridN; i += isoStride {
		for j := 0; j < gridN; j++ {
			if !inSlab[i][j] || !inSlab[i][j+1] {
				continue
			}
			if math.Abs(zGrid[i][j+1]-zGrid[i][j]) > jumpThresh {
				continue
			}
			ownerA, ownerB := -1, -1
			if i > 0 && cellValid[i-1][j] {
				ownerA = (i-1)*gridN + j
			}
			if i < gridN && cellValid[i][j] {
				ownerB = i*gridN + j
			}
			addSurfaceSeg(camGrid[i][j], camGrid[i][j+1], ownerA, ownerB)
		}
	}

	// Drawn Y-constant isolines
	for j := 0; j <= gridN; j += isoStride {
		for i := 0; i < gridN; i++ {
			if !inSlab[i][j] || !inSlab[i+1][j] {
				continue
			}
			if math.Abs(zGrid[i+1][j]-zGrid[i][j]) > jumpThresh {
				continue
			}
			ownerA, ownerB := -1, -1
			if j > 0 && cellValid[i][j-1] {
				ownerA = i*gridN + (j - 1)
			}
			if j < gridN && cellValid[i][j] {
				ownerB = i*gridN + j
			}
			addSurfaceSeg(camGrid[i][j], camGrid[i+1][j], ownerA, ownerB)
		}
	}

	segments = appendBoundingBoxSegments(segments, cb)
	var hlrGrid *triGrid2D
	if len(tris) > 0 {
		hlrGrid = buildTriGrid2D(tris, 32)
	}
	strokes := clipSegmentsHLRWithGrid(segments, tris, hlrGrid, cb.perspective)
	features := projectSurfaceFeatures3D(rawFeatures, xMin, xMax, yMin, yMax, zMin, zMax, cb, gridN, tris, hlrGrid)
	axes := buildAxisAnnotations3D(
		cb,
		[3]string{vx, vy, "z"},
		[3]Node{xMinNode, yMinNode, zMinNode},
		[3]Node{xMaxNode, yMaxNode, zMaxNode},
		[3]float64{xMin, yMin, zMin},
		[3]float64{xMax, yMax, zMax},
	)

	return &Plot3DScene{
		Kind:                 "surface",
		Expr:                 expr,
		VarX:                 vx,
		VarY:                 vy,
		XMin:                 xMin,
		XMax:                 xMax,
		YMin:                 yMin,
		YMax:                 yMax,
		ZMin:                 zMin,
		ZMax:                 zMax,
		XMinNode:             xMinNode,
		XMaxNode:             xMaxNode,
		YMinNode:             yMinNode,
		YMaxNode:             yMaxNode,
		ZMinNode:             zMinNode,
		ZMaxNode:             zMaxNode,
		SpaceXMin:            xMin,
		SpaceXMax:            xMax,
		SpaceXMinNode:        xMinNode,
		SpaceXMaxNode:        xMaxNode,
		Camera:               cam,
		Strokes:              strokes,
		Features:             features,
		Axes:                 axes,
		HasNaNHoles:          hasHoles,
		UnoccludedSurfaceLen: unoccludedLen,
		MonoColor:            mono,
	}, nil
}

func buildParametricCurve3DScene(
	listNode *ListNode,
	tVar string,
	tMinNode, tMaxNode Node, tMin, tMax float64,
	cam Camera3D,
	explicitZ bool, zMinOpt, zMaxOpt float64, zMinOptNode, zMaxOptNode Node,
	mono bool,
) (*Plot3DScene, error) {
	const samples = 160
	xs := make([]float64, samples+1)
	ys := make([]float64, samples+1)
	zs := make([]float64, samples+1)
	valid := make([]bool, samples+1)
	var fx, fy, fz []float64
	hasHoles := false

	vars := map[string]float64{tVar: 0}
	for i := 0; i <= samples; i++ {
		t := tMin + (tMax-tMin)*float64(i)/float64(samples)
		vars[tVar] = t
		xv, okX := evalSampleFloatVars(listNode.Elements[0], vars)
		yv, okY := evalSampleFloatVars(listNode.Elements[1], vars)
		zv, okZ := evalSampleFloatVars(listNode.Elements[2], vars)
		if !okX || !okY || !okZ {
			hasHoles = true
			continue
		}
		xs[i], ys[i], zs[i] = xv, yv, zv
		valid[i] = true
		fx = append(fx, xv)
		fy = append(fy, yv)
		fz = append(fz, zv)
	}

	if len(fx) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot3d_no_valid_points"))
	}

	xMin, xMax := computeRobustRange(fx)
	yMin, yMax := computeRobustRange(fy)
	zMin, zMax := computeRobustRange(fz)
	zMinNode := makeExactBoundNode(zMin)
	zMaxNode := makeExactBoundNode(zMax)
	if explicitZ {
		zMin, zMax = zMinOpt, zMaxOpt
		zMinNode, zMaxNode = zMinOptNode, zMaxOptNode
	}

	xSpan := xMax - xMin
	ySpan := yMax - yMin
	zSpan := zMax - zMin
	if !isFiniteFloat(xSpan) || xSpan <= 0 || !isFiniteFloat(ySpan) || ySpan <= 0 || !isFiniteFloat(zSpan) || zSpan <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot3d_no_valid_points"))
	}

	cb := newCameraBasis(cam)
	var segments []seg3DInput
	var prevCam camVert
	var prevX, prevY, prevZ float64
	hasPrev := false
	jumpX := xSpan * 0.85
	jumpY := ySpan * 0.85
	jumpZ := zSpan * 0.85

	for i := 0; i <= samples; i++ {
		if !valid[i] || xs[i] < xMin || xs[i] > xMax || ys[i] < yMin || ys[i] > yMax || zs[i] < zMin || zs[i] > zMax {
			hasPrev = false
			continue
		}
		nx := 2.0*(xs[i]-xMin)/xSpan - 1.0
		ny := 2.0*(ys[i]-yMin)/ySpan - 1.0
		nz := 2.0*(zs[i]-zMin)/zSpan - 1.0
		if !isFiniteFloat(nx) || !isFiniteFloat(ny) || !isFiniteFloat(nz) {
			hasPrev = false
			continue
		}
		curCam := cb.toCam(vec3{x: nx, y: ny, z: nz}, zs[i])
		if !curCam.valid {
			hasPrev = false
			continue
		}
		if hasPrev && math.Abs(xs[i]-prevX) <= jumpX && math.Abs(ys[i]-prevY) <= jumpY && math.Abs(zs[i]-prevZ) <= jumpZ {
			ca, cbv := prevCam, curCam
			ok := true
			if cb.perspective {
				ca, cbv, ok = clipSegmentNear3D(ca, cbv, cb.dist, nearPlaneW)
			}
			if ok {
				pa := cb.projectCam(ca)
				pb := cb.projectCam(cbv)
				if pa.valid && pb.valid {
					segments = append(segments, seg3DInput{
						a: pa, b: pb, kind: "curve", ownerCellA: -1, ownerCellB: -1,
					})
				}
			}
		}
		prevCam = curCam
		prevX, prevY, prevZ = xs[i], ys[i], zs[i]
		hasPrev = true
	}

	segments = appendBoundingBoxSegments(segments, cb)
	strokes := clipSegmentsHLR(segments, nil, cb.perspective)
	spaceXMinNode := makeExactBoundNode(xMin)
	spaceXMaxNode := makeExactBoundNode(xMax)
	yMinNode := makeExactBoundNode(yMin)
	yMaxNode := makeExactBoundNode(yMax)
	axes := buildAxisAnnotations3D(
		cb,
		[3]string{"x", "y", "z"},
		[3]Node{spaceXMinNode, yMinNode, zMinNode},
		[3]Node{spaceXMaxNode, yMaxNode, zMaxNode},
		[3]float64{xMin, yMin, zMin},
		[3]float64{xMax, yMax, zMax},
	)

	return &Plot3DScene{
		Kind:          "curve",
		Expr:          listNode,
		VarX:          tVar,
		XMin:          tMin,
		XMax:          tMax,
		YMin:          yMin,
		YMax:          yMax,
		ZMin:          zMin,
		ZMax:          zMax,
		XMinNode:      tMinNode,
		XMaxNode:      tMaxNode,
		YMinNode:      yMinNode,
		YMaxNode:      yMaxNode,
		ZMinNode:      zMinNode,
		ZMaxNode:      zMaxNode,
		SpaceXMin:     xMin,
		SpaceXMax:     xMax,
		SpaceXMinNode: spaceXMinNode,
		SpaceXMaxNode: spaceXMaxNode,
		Camera:        cam,
		Strokes:       strokes,
		Axes:          axes,
		HasNaNHoles:   hasHoles,
		MonoColor:     mono,
	}, nil
}

// Mid-luminance chromatic depth ramp in the 256-color 6x6x6 cube (indices 16..231).
// Avoids ANSI 0..15 and extreme bright/dark values so lines stay legible on both light and dark terminals.
var ansiDepthRamp256 = []int{
	45,  // nearest: bright cyan-blue
	39,  // sky blue
	33,  // azure blue
	69,  // cornflower blue
	105, // periwinkle
	141, // lavender purple
	135, // violet
	98,  // farthest: muted purple
}

func clampIntRange(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// RenderBraille3D rasterizes Plot3DScene.Strokes onto a 2x4 Braille canvas and returns (mono, colored).
func RenderBraille3D(scene *Plot3DScene) (string, string) {
	if scene == nil {
		return "", ""
	}
	const charW, charH = 50, 16
	const pixelW, pixelH = charW * 2, charH * 4

	dots := make([][]bool, pixelH)
	for i := range dots {
		dots[i] = make([]bool, pixelW)
	}
	cellMinDepth := make([][]float64, charH)
	cellHasMain := make([][]bool, charH)
	cellHasBox := make([][]bool, charH)
	for cy := 0; cy < charH; cy++ {
		cellMinDepth[cy] = make([]float64, charW)
		cellHasMain[cy] = make([]bool, charW)
		cellHasBox[cy] = make([]bool, charW)
		for cx := 0; cx < charW; cx++ {
			cellMinDepth[cy][cx] = math.Inf(1)
		}
	}

	aspect := effectiveAspect(scene.Camera)
	scale := math.Min(float64(pixelW-2), float64(pixelH-2)*aspect) * 0.5
	centerX := float64(pixelW-1) * 0.5
	centerY := float64(pixelH-1) * 0.5

	toPxFloat := func(x, y float64) (float64, float64) {
		px := centerX + x*scale
		py := centerY - y*scale/aspect
		return px, py
	}

	minDepth, maxDepth := computeMainDepthBounds(scene.Strokes)

	plotDot := func(px, py int, d float64, isMain bool) {
		if px < 0 || px >= pixelW || py < 0 || py >= pixelH {
			return
		}
		dots[py][px] = true
		cx, cy := px/2, py/4
		if isMain {
			cellHasMain[cy][cx] = true
			if d < cellMinDepth[cy][cx] {
				cellMinDepth[cy][cx] = d
			}
		} else {
			cellHasBox[cy][cx] = true
		}
	}

	drawDepthLine := func(x0, y0 int, d0 float64, x1, y1 int, d1 float64, isMain bool) {
		dx := int(math.Abs(float64(x1 - x0)))
		dy := -int(math.Abs(float64(y1 - y0)))
		sx := 1
		if x0 >= x1 {
			sx = -1
		}
		sy := 1
		if y0 >= y1 {
			sy = -1
		}
		err := dx + dy
		totalSteps := max(dx, -dy)
		cx, cy := x0, y0
		step := 0
		for {
			t := 0.0
			if totalSteps > 0 {
				t = float64(step) / float64(totalSteps)
			}
			plotDot(cx, cy, d0+t*(d1-d0), isMain)
			if cx == x1 && cy == y1 {
				break
			}
			e2 := 2 * err
			if e2 >= dy {
				err += dy
				cx += sx
			}
			if e2 <= dx {
				err += dx
				cy += sy
			}
			step++
		}
	}

	// Draw box/axis first, then main surface/curve so min(Z) foreground priority applies cleanly
	maxPxX := float64(pixelW - 1)
	maxPxY := float64(pixelH - 1)
	for _, pass := range []bool{false, true} {
		for _, st := range scene.Strokes {
			isMain := st.Kind == "surface" || st.Kind == "curve"
			if isMain != pass || len(st.Points) == 0 {
				continue
			}
			if len(st.Points) == 1 {
				pt := st.Points[0]
				if !isFiniteFloat(pt.X) || !isFiniteFloat(pt.Y) || !isFiniteFloat(pt.Depth) {
					continue
				}
				fx, fy := toPxFloat(pt.X, pt.Y)
				if !isFiniteFloat(fx) || !isFiniteFloat(fy) || fx < -0.5 || fx > maxPxX+0.5 || fy < -0.5 || fy > maxPxY+0.5 {
					continue
				}
				px := clampIntRange(int(math.Round(fx)), 0, pixelW-1)
				py := clampIntRange(int(math.Round(fy)), 0, pixelH-1)
				plotDot(px, py, pt.Depth, isMain)
				continue
			}
			for i := 0; i < len(st.Points)-1; i++ {
				p0, p1 := st.Points[i], st.Points[i+1]
				if !isFiniteFloat(p0.X) || !isFiniteFloat(p0.Y) || !isFiniteFloat(p0.Depth) ||
					!isFiniteFloat(p1.X) || !isFiniteFloat(p1.Y) || !isFiniteFloat(p1.Depth) {
					continue
				}
				fx0, fy0 := toPxFloat(p0.X, p0.Y)
				fx1, fy1 := toPxFloat(p1.X, p1.Y)
				if !isFiniteFloat(fx0) || !isFiniteFloat(fy0) || !isFiniteFloat(fx1) || !isFiniteFloat(fy1) {
					continue
				}
				cx0, cy0, cx1, cy1, visible := clipSegmentLiangBarsky(fx0, fy0, fx1, fy1, 0, maxPxX, 0, maxPxY)
				if !visible || !isFiniteFloat(cx0) || !isFiniteFloat(cy0) || !isFiniteFloat(cx1) || !isFiniteFloat(cy1) {
					continue
				}
				t0, t1 := 0.0, 1.0
				dxPx, dyPx := fx1-fx0, fy1-fy0
				if math.Abs(dxPx) >= math.Abs(dyPx) && dxPx != 0 {
					t0 = (cx0 - fx0) / dxPx
					t1 = (cx1 - fx0) / dxPx
				} else if dyPx != 0 {
					t0 = (cy0 - fy0) / dyPx
					t1 = (cy1 - fy0) / dyPx
				}
				cd0 := p0.Depth + t0*(p1.Depth-p0.Depth)
				cd1 := p0.Depth + t1*(p1.Depth-p0.Depth)
				px0 := clampIntRange(int(math.Round(cx0)), 0, pixelW-1)
				py0 := clampIntRange(int(math.Round(cy0)), 0, pixelH-1)
				px1 := clampIntRange(int(math.Round(cx1)), 0, pixelW-1)
				py1 := clampIntRange(int(math.Round(cy1)), 0, pixelH-1)
				drawDepthLine(px0, py0, cd0, px1, py1, cd1, isMain)
			}
		}
	}

	buildOutput := func(withColor bool) string {
		gridRunes := make([][]rune, charH)
		gridColor := make([][]int, charH)
		depthSpan := maxDepth - minDepth
		for cy := 0; cy < charH; cy++ {
			gridRunes[cy] = make([]rune, charW)
			gridColor[cy] = make([]int, charW)
			for cx := 0; cx < charW; cx++ {
				var mask rune
				for dr := 0; dr < 4; dr++ {
					py := cy*4 + dr
					for dc := 0; dc < 2; dc++ {
						px := cx*2 + dc
						if dots[py][px] {
							mask |= brailleDotMap[dr][dc]
						}
					}
				}
				gridRunes[cy][cx] = rune(0x2800) + mask
				targetColor := -1
				if mask != 0 {
					if cellHasMain[cy][cx] {
						normD := (cellMinDepth[cy][cx] - minDepth) / depthSpan
						if normD < 0 {
							normD = 0
						} else if normD > 1 {
							normD = 1
						}
						idx := int(math.Round(normD * float64(len(ansiDepthRamp256)-1)))
						targetColor = ansiDepthRamp256[idx]
					} else if cellHasBox[cy][cx] {
						targetColor = 244 // neutral mid-gray for bounding box / axes
					}
				}
				gridColor[cy][cx] = targetColor
			}
		}

		// Place 3D axis name labels outside the selected outer axis edges
		for _, ax := range scene.Axes {
			if !ax.Valid || strings.TrimSpace(ax.AxisName) == "" {
				continue
			}
			labelRunes := []rune(strings.TrimSpace(ax.AxisName))
			if len(labelRunes) == 0 || len(labelRunes) > 6 {
				continue
			}
			lx := ax.LabelPos.X + ax.OutDirX*0.14
			ly := ax.LabelPos.Y + ax.OutDirY*0.14
			fx, fy := toPxFloat(lx, ly)
			if !isFiniteFloat(fx) || !isFiniteFloat(fy) {
				continue
			}
			baseCX := clampIntRange(int(math.Round(fx))/2, 0, charW-len(labelRunes))
			baseCY := clampIntRange(int(math.Round(fy))/4, 0, charH-1)

			// Prefer an empty cell adjacent to the ideal position to avoid overwriting Braille lines
			placeCX, placeCY := baseCX, baseCY
			foundEmpty := false
			offsets := [][2]int{{0, 0}, {0, 1}, {1, 0}, {-1, 0}, {0, -1}, {2, 0}, {-2, 0}}
			for _, off := range offsets {
				cx := clampIntRange(baseCX+off[0], 0, charW-len(labelRunes))
				cy := clampIntRange(baseCY+off[1], 0, charH-1)
				clear := true
				for k := range labelRunes {
					if gridRunes[cy][cx+k] != rune(0x2800) {
						clear = false
						break
					}
				}
				if clear {
					placeCX, placeCY = cx, cy
					foundEmpty = true
					break
				}
			}
			if !foundEmpty {
				// Avoid overwriting existing non-Braille labels or markers
				canWrite := true
				for k := range labelRunes {
					r := gridRunes[placeCY][placeCX+k]
					if r < 0x2800 || r > 0x28FF {
						canWrite = false
						break
					}
				}
				if !canWrite {
					continue
				}
			}
			for k, lr := range labelRunes {
				gridRunes[placeCY][placeCX+k] = lr
				gridColor[placeCY][placeCX+k] = 250 // crisp neutral label tone
			}
		}

		// Place single-cell markers for visible CAS-detected critical points
		for _, feat := range scene.Features {
			if !feat.Visible {
				continue
			}
			if !isFiniteFloat(feat.Projected.X) || !isFiniteFloat(feat.Projected.Y) {
				continue
			}
			fx, fy := toPxFloat(feat.Projected.X, feat.Projected.Y)
			if !isFiniteFloat(fx) || !isFiniteFloat(fy) || fx < 0 || fx >= float64(pixelW) || fy < 0 || fy >= float64(pixelH) {
				continue
			}
			cx := clampIntRange(int(math.Round(fx))/2, 0, charW-1)
			cy := clampIntRange(int(math.Round(fy))/4, 0, charH-1)
			marker := '◇'
			markerColor := 177 // soft violet for degenerate critical point
			switch feat.Kind {
			case "min", "max":
				marker = '●'
				markerColor = 42 // green for local extrema
			case "saddle":
				marker = '◆'
				markerColor = 214 // amber for saddle point
			}
			gridRunes[cy][cx] = marker
			gridColor[cy][cx] = markerColor
		}

		var sb strings.Builder
		sb.WriteString("\n")
		for cy := 0; cy < charH; cy++ {
			sb.WriteString("  ")
			lastColor := -1
			for cx := 0; cx < charW; cx++ {
				r := gridRunes[cy][cx]
				targetColor := gridColor[cy][cx]
				if withColor && r != rune(0x2800) {
					if targetColor != lastColor {
						if targetColor >= 0 {
							sb.WriteString(fmt.Sprintf("\x1b[38;5;%dm", targetColor))
						} else {
							sb.WriteString("\x1b[0m")
						}
						lastColor = targetColor
					}
				} else if withColor && r == rune(0x2800) && lastColor != -1 {
					sb.WriteString("\x1b[0m")
					lastColor = -1
				}
				sb.WriteRune(r)
			}
			if withColor && lastColor != -1 {
				sb.WriteString("\x1b[0m")
			}
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
		if scene.Kind == "curve" {
			sb.WriteString(i18n.T("plot3d.curve_domain",
				scene.VarX, Format(scene.XMinNode), Format(scene.XMaxNode),
				Format(scene.ZMinNode), Format(scene.ZMaxNode),
			))
		} else {
			sb.WriteString(i18n.T("plot3d.surface_domain",
				scene.VarX, Format(scene.XMinNode), Format(scene.XMaxNode),
				scene.VarY, Format(scene.YMinNode), Format(scene.YMaxNode),
				Format(scene.ZMinNode), Format(scene.ZMaxNode),
			))
		}
		sb.WriteString(i18n.T("plot3d.camera_info",
			fmtSVGNum(scene.Camera.AzimuthDeg),
			fmtSVGNum(scene.Camera.ElevationDeg),
		))
		for _, feat := range scene.Features {
			switch feat.Kind {
			case "min":
				sb.WriteString(i18n.T("plot3d.feature_min", feat.LabelStr))
			case "max":
				sb.WriteString(i18n.T("plot3d.feature_max", feat.LabelStr))
			case "saddle":
				sb.WriteString(i18n.T("plot3d.feature_saddle", feat.LabelStr))
			default:
				sb.WriteString(i18n.T("plot3d.feature_critical", feat.LabelStr))
			}
		}
		return sb.String()
	}

	mono := buildOutput(false)
	if scene.MonoColor {
		return mono, ""
	}
	colored := buildOutput(true)
	return mono, colored
}

// EvalPlot3D evaluates plot3d(...) without an explicit environment.
func EvalPlot3D(args []Node) (Node, error) {
	return EvalPlot3DWithEnv(args, nil)
}

// EvalPlot3DWithEnv evaluates plot3d(...) using env and returns a *PlotNode containing terminal Braille, SVG, and Typst.
func EvalPlot3DWithEnv(args []Node, env *Env) (Node, error) {
	scene, err := BuildPlot3DSceneWithEnv(args, env)
	if err != nil {
		return nil, err
	}
	mono, colored := RenderBraille3D(scene)
	svgStr := RenderPlot3DSVG(scene)
	typstStr := RenderPlot3DTypst(scene)
	return NewPlotNodeWithColorAndVectors(mono, colored, svgStr, typstStr), nil
}
