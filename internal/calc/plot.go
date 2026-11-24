package calc

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// Braille dot bitmasks for a 2x4 cell:
// (col 0, row 0) -> 0x01   (col 1, row 0) -> 0x08
// (col 0, row 1) -> 0x02   (col 1, row 1) -> 0x10
// (col 0, row 2) -> 0x04   (col 1, row 2) -> 0x20
// (col 0, row 3) -> 0x40   (col 1, row 3) -> 0x80
var brailleDotMap = [4][2]rune{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// plotFeature represents a mathematically significant point detected by CAS.
type plotFeature struct {
	Kind     string // "root", "min", "max", "asymptote"
	XNode    Node
	YNode    Node
	XVal     float64
	YVal     float64
	LabelStr string
}

// BrailleCanvas manages a terminal 2D grid using Braille characters.
type BrailleCanvas struct {
	CharW    int // width in character cells (default: 50)
	CharH    int // height in character lines (default: 16)
	PixelW   int // CharW * 2
	PixelH   int // CharH * 4
	Dots     [][]bool
	XMin     float64
	XMax     float64
	YMin     float64
	YMax     float64
	XMinNode Node
	XMaxNode Node
	YMinNode Node
	YMaxNode Node
	Features []plotFeature
}

func newBrailleCanvas(charW, charH int, xMin, xMax, yMin, yMax float64, xMinNode, xMaxNode, yMinNode, yMaxNode Node) *BrailleCanvas {
	pW := charW * 2
	pH := charH * 4
	dots := make([][]bool, pH)
	for i := range dots {
		dots[i] = make([]bool, pW)
	}
	return &BrailleCanvas{
		CharW:    charW,
		CharH:    charH,
		PixelW:   pW,
		PixelH:   pH,
		Dots:     dots,
		XMin:     xMin,
		XMax:     xMax,
		YMin:     yMin,
		YMax:     yMax,
		XMinNode: xMinNode,
		XMaxNode: xMaxNode,
		YMinNode: yMinNode,
		YMaxNode: yMaxNode,
	}
}

func (c *BrailleCanvas) toPixelX(x float64) int {
	if c.XMax == c.XMin {
		return 0
	}
	frac := (x - c.XMin) / (c.XMax - c.XMin)
	px := int(math.Round(frac * float64(c.PixelW-1)))
	return px
}

func (c *BrailleCanvas) toPixelY(y float64) int {
	if c.YMax == c.YMin {
		return 0
	}
	frac := (c.YMax - y) / (c.YMax - c.YMin)
	py := int(math.Round(frac * float64(c.PixelH-1)))
	return py
}

// SetPixel sets the dot at (px, py) on the canvas. Out-of-bounds coordinates are clipped.
func (c *BrailleCanvas) SetPixel(px, py int) {
	if px >= 0 && px < c.PixelW && py >= 0 && py < c.PixelH {
		c.Dots[py][px] = true
	}
}

// drawLine draws a line between (x0, y0) and (x1, y1) using Bresenham's algorithm.
func (c *BrailleCanvas) drawLine(x0, y0, x1, y1 int) {
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

	currX, currY := x0, y0
	for {
		c.SetPixel(currX, currY)
		if currX == x1 && currY == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			currX += sx
		}
		if e2 <= dx {
			err += dx
			currY += sy
		}
	}
}

// Render compiles the canvas, axes, pinned labels, and metadata into terminal text.
func (c *BrailleCanvas) Render() string {
	// Build character grid
	grid := make([][]rune, c.CharH)
	for cy := 0; cy < c.CharH; cy++ {
		grid[cy] = make([]rune, c.CharW)
		for cx := 0; cx < c.CharW; cx++ {
			var mask rune
			for dr := 0; dr < 4; dr++ {
				py := cy*4 + dr
				for dc := 0; dc < 2; dc++ {
					px := cx*2 + dc
					if c.Dots[py][px] {
						mask |= brailleDotMap[dr][dc]
					}
				}
			}
			if mask == 0 {
				grid[cy][cx] = ' '
			} else {
				grid[cy][cx] = rune(0x2800) + mask
			}
		}
	}

	// Draw vertical asymptote dashed lines ('┆')
	for _, feat := range c.Features {
		if feat.Kind == "asymptote" {
			px := c.toPixelX(feat.XVal)
			cx := px / 2
			if cx >= 0 && cx < c.CharW {
				for cy := 0; cy < c.CharH; cy++ {
					if grid[cy][cx] == ' ' {
						grid[cy][cx] = '┆'
					}
				}
			}
		}
	}

	// Draw X axis if in range
	if c.YMin <= 0 && c.YMax >= 0 {
		pyAxis := c.toPixelY(0)
		cyAxis := pyAxis / 4
		if cyAxis >= 0 && cyAxis < c.CharH {
			for cx := 0; cx < c.CharW; cx++ {
				if grid[cyAxis][cx] == ' ' {
					grid[cyAxis][cx] = '─'
				}
			}
		}
	}

	// Draw Y axis if in range
	if c.XMin <= 0 && c.XMax >= 0 {
		pxAxis := c.toPixelX(0)
		cxAxis := pxAxis / 2
		if cxAxis >= 0 && cxAxis < c.CharW {
			for cy := 0; cy < c.CharH; cy++ {
				switch grid[cy][cxAxis] {
				case ' ':
					grid[cy][cxAxis] = '│'
				case '─':
					grid[cy][cxAxis] = '┼'
				}
			}
		}
	}

	// Place feature markers (◆ for roots, ● for extrema, ○ for removable holes) and pin labels
	for _, feat := range c.Features {
		if feat.Kind == "asymptote" || feat.Kind == "horiz_asymptote" || feat.Kind == "oblique_asymptote" {
			continue
		}
		px := c.toPixelX(feat.XVal)
		py := c.toPixelY(feat.YVal)
		cx := px / 2
		cy := py / 4

		if cx >= 0 && cx < c.CharW && cy >= 0 && cy < c.CharH {
			marker := '◆'
			if feat.Kind == "min" || feat.Kind == "max" {
				marker = '●'
			} else if feat.Kind == "hole" {
				marker = '○'
			}
			grid[cy][cx] = marker

			// Pin label string adjacent to marker (prefer right or top)
			label := feat.LabelStr
			labelRunes := []rune(label)
			labelLen := len(labelRunes)

			// Try placing to the right
			startCol := cx + 2
			targetRow := cy
			if startCol+labelLen < c.CharW {
				for idx, r := range labelRunes {
					grid[targetRow][startCol+idx] = r
				}
			} else if cx-labelLen-2 >= 0 {
				// Try placing to the left
				startCol = cx - labelLen - 1
				for idx, r := range labelRunes {
					grid[targetRow][startCol+idx] = r
				}
			}
		}
	}

	// Format lines with left axis border and ticks
	yMaxStr := Format(c.YMaxNode)
	yMinStr := Format(c.YMinNode)
	marginW := max(len(yMaxStr), len(yMinStr)) + 1
	if marginW < 5 {
		marginW = 5
	}

	var sb strings.Builder
	sb.WriteString("\n")

	for cy := 0; cy < c.CharH; cy++ {
		var leftLabel string
		tickChar := "│"
		if cy == 0 {
			leftLabel = yMaxStr
			tickChar = "┼"
		} else if cy == c.CharH-1 {
			leftLabel = yMinStr
			tickChar = "┼"
		} else if cy == c.CharH/2 && (c.YMin > 0 || c.YMax < 0) {
			// Midpoint tick if 0 is not crossed
			leftLabel = ""
			tickChar = "┼"
		}

		fmtStr := fmt.Sprintf("%%%ds %%s ", marginW)
		sb.WriteString(fmt.Sprintf(fmtStr, leftLabel, tickChar))
		sb.WriteString(string(grid[cy]))
		sb.WriteString("\n")
	}

	// Bottom axis line
	sb.WriteString(strings.Repeat(" ", marginW+1))
	sb.WriteString("└")
	sb.WriteString(strings.Repeat("─", c.CharW+2))
	sb.WriteString(" x\n")

	// Bottom axis labels (XMin, XMid, XMax)
	xMinStr := Format(c.XMinNode)
	xMaxStr := Format(c.XMaxNode)
	sb.WriteString(strings.Repeat(" ", marginW+2))
	sb.WriteString(xMinStr)
	spacing := c.CharW - len(xMinStr) - len(xMaxStr)
	if spacing > 0 {
		sb.WriteString(strings.Repeat(" ", spacing))
	}
	sb.WriteString(xMaxStr)
	sb.WriteString("\n\n")

	// Metadata section: Detected CAS features
	sb.WriteString(i18n.T("plot.features_header"))
	hasFeature := false
	for _, feat := range c.Features {
		hasFeature = true
		switch feat.Kind {
		case "root":
			sb.WriteString(i18n.T("plot.zero", Format(feat.XNode)))
		case "min":
			sb.WriteString(i18n.T("plot.local_min", Format(feat.XNode), Format(feat.YNode)))
		case "max":
			sb.WriteString(i18n.T("plot.local_max", Format(feat.XNode), Format(feat.YNode)))
		case "asymptote":
			sb.WriteString(i18n.T("plot.asymptote", Format(feat.XNode)))
		case "hole":
			sb.WriteString(i18n.T("plot.hole", Format(feat.XNode), Format(feat.YNode)))
		case "horiz_asymptote":
			sb.WriteString(i18n.T("plot.horiz_asymptote", Format(feat.YNode)))
		case "oblique_asymptote":
			sb.WriteString(i18n.T("plot.oblique_asymptote", Format(feat.YNode)))
		}
	}
	if !hasFeature {
		sb.WriteString(i18n.T("plot.no_features"))
	}
	sb.WriteString(i18n.T("plot.domain",
		Format(c.XMinNode), Format(c.XMaxNode), Format(c.YMinNode), Format(c.YMaxNode)))

	return sb.String()
}

// evalNodeFloat evaluates an AST node to a real float64 using interval enclosure.
func evalNodeFloat(n Node) (float64, error) {
	interval, err := EvalNodeInterval(n, 20)
	if err != nil {
		return 0, err
	}
	mid := new(big.Rat).Add(interval.Low, interval.High)
	mid.Mul(mid, big.NewRat(1, 2))
	r, _ := mid.Float64()
	if math.IsNaN(r) || math.IsInf(r, 0) {
		return 0, fmt.Errorf("%s", i18n.T("errors.plot_nan_or_inf"))
	}
	return r, nil
}

// substituteVar replaces occurrences of varName with valNode in an expression tree.
func substituteVar(n Node, varName string, valNode Node) Node {
	return Substitute(n, varName, valNode)
}

// findFreeVariable finds the variable in the expression.
func findFreeVariable(n Node) string {
	vars := ExtractFreeVariables(n)
	if len(vars) > 0 {
		return vars[0]
	}
	return "x"
}

// parseDomain extracts [lower, upper] nodes and their float values.
func parseDomain(n Node) (Node, Node, float64, float64, error) {
	var lowerNode, upperNode Node

	switch v := n.(type) {
	case *ListNode:
		if len(v.Elements) != 2 {
			return nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_domain_elements", len(v.Elements)))
		}
		lowerNode = v.Elements[0]
		upperNode = v.Elements[1]
	case *MatrixNode:
		if v.Rows == 1 && v.Cols == 2 {
			lowerNode = v.Data[0][0]
			upperNode = v.Data[0][1]
		} else {
			return nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_matrix_domain"))
		}
	default:
		return nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_domain_list", n.String()))
	}

	lowerVal, err := evalNodeFloat(lowerNode)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_eval_min", err))
	}
	upperVal, err := evalNodeFloat(upperNode)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_eval_max", err))
	}

	if lowerVal >= upperVal {
		return nil, nil, 0, 0, fmt.Errorf("%s", i18n.T("errors.plot_invalid_domain"))
	}

	return lowerNode, upperNode, lowerVal, upperVal, nil
}

// PlotPoint represents a point in continuous mathematical coordinates.
type PlotPoint struct {
	X float64
	Y float64
}

// PlotBranch represents a continuous curve segment without singularities or viewport jumps.
type PlotBranch struct {
	Points []PlotPoint
}

// PlotAsymptote represents a vertical, horizontal, or oblique asymptote in mathematical coordinates.
type PlotAsymptote struct {
	Kind      string  // "vertical", "horizontal", "oblique"
	XVal      float64 // vertical asymptote x
	YVal      float64 // horizontal asymptote y
	Slope     float64 // oblique asymptote m
	Intercept float64 // oblique asymptote b
	ExactNode Node
}

// PlotScene is the renderer-independent intermediate representation in mathematical coordinates.
type PlotScene struct {
	Expr       Node
	VarName    string
	XMin, XMax float64
	YMin, YMax float64
	XMinNode   Node
	XMaxNode   Node
	YMinNode   Node
	YMaxNode   Node
	Branches   []PlotBranch
	Features   []plotFeature
	Asymptotes []PlotAsymptote
}

// findAsymptotes identifies vertical asymptotes and removable singularities where denominator is 0.
func findAsymptotes(expr Node, varName string, xMin, xMax float64) []plotFeature {
	var features []plotFeature
	var denominators []Node

	var walk func(n Node)
	walk = func(n Node) {
		if n == nil {
			return
		}
		if pow, ok := n.(*PowNode); ok {
			if r, ok := pow.Exp.(*RationalNode); ok && r.Val.Sign() < 0 {
				denominators = append(denominators, pow.Base)
			}
		}
		switch v := n.(type) {
		case *AddNode:
			for _, t := range v.Terms {
				walk(t)
			}
		case *MulNode:
			for _, f := range v.Factors {
				walk(f)
			}
		case *FuncNode:
			for _, a := range v.Args {
				walk(a)
			}
		case *SqrtNode:
			walk(v.Radicand)
		case *UnaryOpNode:
			walk(v.Expr)
		}
	}
	walk(expr)

	seenX := make(map[string]bool)
	for _, denom := range denominators {
		solList, err := solveEquation(denom, varName)
		if err == nil {
			if list, ok := solList.(*ListNode); ok {
				for _, sol := range list.Elements {
					val, err := evalNodeFloat(sol)
					if err == nil && val >= xMin && val <= xMax {
						key := Format(sol)
						if seenX[key] {
							continue
						}
						seenX[key] = true

						// Distinguish removable singularity (finite limit exists) from vertical asymptote
						limRes, limErr := EvalLimit(expr, &VarNode{Name: varName}, sol, nil)
						if limErr == nil && limRes != nil {
							if yLim, yErr := evalNodeFloat(limRes); yErr == nil && !math.IsNaN(yLim) && !math.IsInf(yLim, 0) {
								features = append(features, plotFeature{
									Kind:     "hole",
									XNode:    sol,
									YNode:    limRes,
									XVal:     val,
									YVal:     yLim,
									LabelStr: fmt.Sprintf("[%s, %s]", Format(sol), Format(limRes)),
								})
								continue
							}
						}

						features = append(features, plotFeature{
							Kind:     "asymptote",
							XNode:    sol,
							XVal:     val,
							LabelStr: fmt.Sprintf("[%s, asymp]", Format(sol)),
						})
					}
				}
			}
		}
	}

	return features
}

func roundToSpanPrecision(v, span float64) *big.Rat {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return big.NewRat(0, 1)
	}
	if span <= 0 || math.IsNaN(span) || math.IsInf(span, 0) {
		span = 1
	}
	exp := math.Floor(math.Log10(span)) - 2
	step := math.Pow(10, exp)
	if step <= 0 || math.IsNaN(step) || math.IsInf(step, 0) {
		return new(big.Rat).SetFloat64(v)
	}
	rounded := math.Round(v/step) * step
	if math.Abs(rounded) < step*1e-6 {
		rounded = 0
	}
	r := new(big.Rat).SetFloat64(rounded)
	if r == nil {
		return big.NewRat(0, 1)
	}
	return r
}

type evalFloatState uint8

const (
	evalFloatOK evalFloatState = iota
	evalFloatDomainError
	evalFloatUnsupported
)

func evalSampleFloat(expr Node, varName string, x float64) (float64, bool) {
	return evalSampleFloatVars(expr, map[string]float64{varName: x})
}

func evalSampleFloatVars(expr Node, vars map[string]float64) (float64, bool) {
	for _, v := range vars {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
	}
	y, state := evalASTFloat64Vars(expr, vars)
	switch state {
	case evalFloatOK:
		if math.IsNaN(y) || math.IsInf(y, 0) {
			return 0, false
		}
		return y, true
	case evalFloatDomainError:
		return 0, false
	default:
		return evalSampleIntervalFallbackVars(expr, vars)
	}
}

func evalSampleIntervalFallbackVars(expr Node, vars map[string]float64) (float64, bool) {
	subExpr := expr
	for varName, x := range vars {
		rRat := new(big.Rat).SetFloat64(x)
		if rRat == nil {
			return 0, false
		}
		subExpr = substituteVar(subExpr, varName, NewRationalFromBigRat(rRat))
	}
	y, err := evalNodeFloat(subExpr)
	if err != nil || math.IsNaN(y) || math.IsInf(y, 0) {
		return 0, false
	}
	return y, true
}

func evalASTFloat64Vars(n Node, vars map[string]float64) (float64, evalFloatState) {
	if n == nil {
		return 0, evalFloatUnsupported
	}
	switch v := n.(type) {
	case *RationalNode:
		f, _ := v.Val.Float64()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, evalFloatDomainError
		}
		return f, evalFloatOK
	case *VarNode:
		if val, ok := vars[v.Name]; ok {
			return val, evalFloatOK
		}
		return 0, evalFloatUnsupported
	case *ConstNode:
		switch strings.ToLower(v.Name) {
		case "pi":
			return math.Pi, evalFloatOK
		case "e":
			return math.E, evalFloatOK
		case "phi":
			return math.Phi, evalFloatOK
		default:
			return 0, evalFloatUnsupported
		}
	case *UnaryOpNode:
		val, st := evalASTFloat64Vars(v.Expr, vars)
		if st != evalFloatOK {
			return 0, st
		}
		if v.Op == "-" {
			return -val, evalFloatOK
		}
		return val, evalFloatOK
	case *AddNode:
		sum := 0.0
		for _, t := range v.Terms {
			tv, st := evalASTFloat64Vars(t, vars)
			if st != evalFloatOK {
				return 0, st
			}
			sum += tv
		}
		if math.IsNaN(sum) || math.IsInf(sum, 0) {
			return 0, evalFloatDomainError
		}
		return sum, evalFloatOK
	case *MulNode:
		prod := 1.0
		for _, f := range v.Factors {
			fv, st := evalASTFloat64Vars(f, vars)
			if st != evalFloatOK {
				return 0, st
			}
			prod *= fv
		}
		if math.IsNaN(prod) || math.IsInf(prod, 0) {
			return 0, evalFloatDomainError
		}
		return prod, evalFloatOK
	case *PowNode:
		base, stB := evalASTFloat64Vars(v.Base, vars)
		if stB != evalFloatOK {
			return 0, stB
		}
		exp, stE := evalASTFloat64Vars(v.Exp, vars)
		if stE != evalFloatOK {
			return 0, stE
		}
		if base == 0 && exp <= 0 {
			return 0, evalFloatDomainError
		}
		if base < 0 {
			if r, okR := v.Exp.(*RationalNode); okR && r.Val.IsInt() {
				res := math.Pow(base, exp)
				if math.IsNaN(res) || math.IsInf(res, 0) {
					return 0, evalFloatDomainError
				}
				return res, evalFloatOK
			}
			if r, okR := v.Exp.(*RationalNode); okR && r.Val.Denom().Cmp(big.NewInt(3)) == 0 {
				num := float64(r.Val.Num().Int64())
				res := math.Pow(math.Cbrt(base), num)
				if math.IsNaN(res) || math.IsInf(res, 0) {
					return 0, evalFloatDomainError
				}
				return res, evalFloatOK
			}
			return 0, evalFloatDomainError
		}
		res := math.Pow(base, exp)
		if math.IsNaN(res) || math.IsInf(res, 0) {
			return 0, evalFloatDomainError
		}
		return res, evalFloatOK
	case *SqrtNode:
		rad, st := evalASTFloat64Vars(v.Radicand, vars)
		if st != evalFloatOK {
			return 0, st
		}
		if rad < 0 {
			return 0, evalFloatDomainError
		}
		return math.Sqrt(rad), evalFloatOK
	case *FuncNode:
		if len(v.Args) != 1 {
			return 0, evalFloatUnsupported
		}
		arg, st := evalASTFloat64Vars(v.Args[0], vars)
		if st != evalFloatOK {
			return 0, st
		}
		switch strings.ToLower(v.Name) {
		case "sin":
			return math.Sin(arg), evalFloatOK
		case "cos":
			return math.Cos(arg), evalFloatOK
		case "tan":
			c := math.Cos(arg)
			if math.Abs(c) < 1e-12 {
				return 0, evalFloatDomainError
			}
			return math.Tan(arg), evalFloatOK
		case "asin":
			if arg < -1 || arg > 1 {
				return 0, evalFloatDomainError
			}
			return math.Asin(arg), evalFloatOK
		case "acos":
			if arg < -1 || arg > 1 {
				return 0, evalFloatDomainError
			}
			return math.Acos(arg), evalFloatOK
		case "atan":
			return math.Atan(arg), evalFloatOK
		case "sinh":
			res := math.Sinh(arg)
			if math.IsInf(res, 0) {
				return 0, evalFloatDomainError
			}
			return res, evalFloatOK
		case "cosh":
			res := math.Cosh(arg)
			if math.IsInf(res, 0) {
				return 0, evalFloatDomainError
			}
			return res, evalFloatOK
		case "tanh":
			return math.Tanh(arg), evalFloatOK
		case "exp":
			res := math.Exp(arg)
			if math.IsInf(res, 0) {
				return 0, evalFloatDomainError
			}
			return res, evalFloatOK
		case "ln", "log":
			if arg <= 0 {
				return 0, evalFloatDomainError
			}
			return math.Log(arg), evalFloatOK
		case "abs":
			return math.Abs(arg), evalFloatOK
		case "sqrt":
			if arg < 0 {
				return 0, evalFloatDomainError
			}
			return math.Sqrt(arg), evalFloatOK
		case "cbrt":
			return math.Cbrt(arg), evalFloatOK
		case "floor":
			return math.Floor(arg), evalFloatOK
		case "ceil":
			return math.Ceil(arg), evalFloatOK
		default:
			return 0, evalFloatUnsupported
		}
	default:
		return 0, evalFloatUnsupported
	}
}

// clipSegmentLiangBarsky clips segment (x0,y0)-(x1,y1) against [xMin,xMax] x [yMin,yMax].
func clipSegmentLiangBarsky(x0, y0, x1, y1, xMin, xMax, yMin, yMax float64) (float64, float64, float64, float64, bool) {
	dx := x1 - x0
	dy := y1 - y0
	t0, t1 := 0.0, 1.0

	clip := func(p, q float64) bool {
		if p == 0 {
			return q >= 0
		}
		r := q / p
		if p < 0 {
			if r > t1 {
				return false
			}
			if r > t0 {
				t0 = r
			}
		} else {
			if r < t0 {
				return false
			}
			if r < t1 {
				t1 = r
			}
		}
		return true
	}

	if !clip(-dx, x0-xMin) || !clip(dx, xMax-x0) || !clip(-dy, y0-yMin) || !clip(dy, yMax-y0) {
		return 0, 0, 0, 0, false
	}

	cx0 := x0 + t0*dx
	cy0 := y0 + t0*dy
	cx1 := x0 + t1*dx
	cy1 := y0 + t1*dy
	return cx0, cy0, cx1, cy1, true
}

// simplifyBranchRDP applies Ramer-Douglas-Peucker simplification in normalized [0,1] viewport space.
func simplifyBranchRDP(pts []PlotPoint, xSpan, ySpan, eps float64) []PlotPoint {
	if len(pts) <= 2 || xSpan <= 0 || ySpan <= 0 {
		return pts
	}
	dmax := 0.0
	idx := 0
	p0 := pts[0]
	pn := pts[len(pts)-1]
	x1, y1 := p0.X/xSpan, p0.Y/ySpan
	x2, y2 := pn.X/xSpan, pn.Y/ySpan
	dx, dy := x2-x1, y2-y1
	segLenSq := dx*dx + dy*dy

	for i := 1; i < len(pts)-1; i++ {
		px, py := pts[i].X/xSpan, pts[i].Y/ySpan
		var dist float64
		if segLenSq == 0 {
			dist = math.Hypot(px-x1, py-y1)
		} else {
			dist = math.Abs(dy*px-dx*py+x2*y1-y2*x1) / math.Sqrt(segLenSq)
		}
		if dist > dmax {
			idx = i
			dmax = dist
		}
	}

	if dmax > eps {
		left := simplifyBranchRDP(pts[:idx+1], xSpan, ySpan, eps)
		right := simplifyBranchRDP(pts[idx:], xSpan, ySpan, eps)
		out := make([]PlotPoint, 0, len(left)+len(right)-1)
		out = append(out, left[:len(left)-1]...)
		out = append(out, right...)
		return out
	}
	return []PlotPoint{p0, pn}
}

// BuildPlotScene constructs a PlotScene IR with CAS exact pre-splitting, adaptive sampling, and viewport clipping.
func BuildPlotScene(args []Node) (*PlotScene, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("%s", i18n.T("errors.plot_args_count", len(args)))
	}

	expr := args[0]
	xMinNode, xMaxNode, xMin, xMax, err := parseDomain(args[1])
	if err != nil {
		return nil, err
	}

	var yMinNode, yMaxNode Node
	var yMin, yMax float64
	hasExplicitY := false

	if len(args) == 3 {
		yMinNode, yMaxNode, yMin, yMax, err = parseDomain(args[2])
		if err != nil {
			return nil, err
		}
		hasExplicitY = true
	}

	varName := findFreeVariable(expr)

	// Detect exact CAS singularities (vertical asymptotes & removable holes) first for pre-splitting
	singularities := findAsymptotes(expr, varName, xMin, xMax)

	// Estimate y-range if not explicitly supplied, using golden-ratio offset to avoid aliasing
	if !hasExplicitY {
		const coarseCount = 72
		const phiOffset = 0.6180339887498949
		var validYs []float64
		xSpan := xMax - xMin
		for i := 0; i < coarseCount; i++ {
			frac := (float64(i) + phiOffset) / float64(coarseCount)
			if frac >= 1.0 {
				frac = 0.999
			}
			x := xMin + frac*xSpan
			if y, ok := evalSampleFloat(expr, varName, x); ok && math.Abs(y) < 1e6 {
				validYs = append(validYs, y)
			}
		}
		if len(validYs) == 0 {
			yMin, yMax = -2, 2
			yMinNode = mustRational(-2, 1)
			yMaxNode = mustRational(2, 1)
		} else {
			sort.Float64s(validYs)
			// Trim extreme outliers near undetected poles
			loIdx := 0
			hiIdx := len(validYs) - 1
			if len(validYs) >= 20 {
				loIdx = len(validYs) / 20
				hiIdx = len(validYs) - 1 - loIdx
			}
			minY, maxY := validYs[loIdx], validYs[hiIdx]
			span := maxY - minY
			if span < 1e-6 {
				span = 2
			}
			yMin = minY - span*0.1
			yMax = maxY + span*0.1
			yMinNode = NewRationalFromBigRat(roundToSpanPrecision(yMin, yMax-yMin))
			yMaxNode = NewRationalFromBigRat(roundToSpanPrecision(yMax, yMax-yMin))
		}
	}

	scene := &PlotScene{
		Expr:     expr,
		VarName:  varName,
		XMin:     xMin,
		XMax:     xMax,
		YMin:     yMin,
		YMax:     yMax,
		XMinNode: xMinNode,
		XMaxNode: xMaxNode,
		YMinNode: yMinNode,
		YMaxNode: yMaxNode,
	}

	// CAS Feature Detection: Roots, Extrema, Singularities, and Oblique/Horizontal Asymptotes
	solList, err := solveEquation(expr, varName)
	if err == nil {
		if list, ok := solList.(*ListNode); ok {
			for _, root := range list.Elements {
				rVal, err := evalNodeFloat(root)
				if err == nil && rVal >= xMin && rVal <= xMax {
					scene.Features = append(scene.Features, plotFeature{
						Kind:     "root",
						XNode:    root,
						YNode:    mustRational(0, 1),
						XVal:     rVal,
						YVal:     0,
						LabelStr: fmt.Sprintf("[%s, 0]", Format(root)),
					})
				}
			}
		}
	}

	deriv, err := differentiate(expr, varName)
	if err == nil {
		critList, err := solveEquation(deriv, varName)
		if err == nil {
			if list, ok := critList.(*ListNode); ok {
				for _, crit := range list.Elements {
					cVal, err := evalNodeFloat(crit)
					if err == nil && cVal >= xMin && cVal <= xMax {
						yCritNode, err := Eval(substituteVar(expr, varName, crit))
						if err == nil {
							yCritVal, err := evalNodeFloat(yCritNode)
							if err == nil && yCritVal >= yMin && yCritVal <= yMax {
								kind := "min"
								deriv2, err := differentiate(deriv, varName)
								if err == nil {
									d2ValNode, err := Eval(substituteVar(deriv2, varName, crit))
									if err == nil {
										d2Val, err := evalNodeFloat(d2ValNode)
										if err == nil && d2Val < 0 {
											kind = "max"
										}
									}
								}
								scene.Features = append(scene.Features, plotFeature{
									Kind:     kind,
									XNode:    crit,
									YNode:    yCritNode,
									XVal:     cVal,
									YVal:     yCritVal,
									LabelStr: fmt.Sprintf("[%s, %s]", Format(crit), Format(yCritNode)),
								})
							}
						}
					}
				}
			}
		}
	}

	scene.Features = append(scene.Features, singularities...)
	for _, s := range singularities {
		if s.Kind == "asymptote" {
			scene.Asymptotes = append(scene.Asymptotes, PlotAsymptote{
				Kind:      "vertical",
				XVal:      s.XVal,
				ExactNode: s.XNode,
			})
		}
	}

	// Horizontal / oblique asymptotes via polynomial quotient from EvalApart helper
	if quotNode, ok := ExtractPolyQuotient(expr, varName); ok {
		if qPoly, okP := extractPoly(quotNode, varName); okP && qPoly.degree() <= 1 {
			var bVal, mVal float64
			if len(qPoly.coeffs) > 0 {
				if v, err := evalNodeFloat(qPoly.coeffs[0]); err == nil {
					bVal = v
				}
			}
			if qPoly.degree() == 1 && len(qPoly.coeffs) > 1 {
				if v, err := evalNodeFloat(qPoly.coeffs[1]); err == nil {
					mVal = v
				}
			}
			if qPoly.degree() == 0 {
				scene.Asymptotes = append(scene.Asymptotes, PlotAsymptote{
					Kind:      "horizontal",
					YVal:      bVal,
					ExactNode: quotNode,
				})
				scene.Features = append(scene.Features, plotFeature{
					Kind:     "horiz_asymptote",
					YNode:    quotNode,
					YVal:     bVal,
					LabelStr: fmt.Sprintf("y = %s", Format(quotNode)),
				})
			} else if qPoly.degree() == 1 {
				scene.Asymptotes = append(scene.Asymptotes, PlotAsymptote{
					Kind:      "oblique",
					Slope:     mVal,
					Intercept: bVal,
					ExactNode: quotNode,
				})
				scene.Features = append(scene.Features, plotFeature{
					Kind:     "oblique_asymptote",
					YNode:    quotNode,
					LabelStr: fmt.Sprintf("y = %s", Format(quotNode)),
				})
			}
		}
	}

	// Partition [xMin, xMax] at exact singularities
	var splits []float64
	splits = append(splits, xMin)
	epsMargin := (xMax - xMin) * 1e-6
	var singXs []float64
	for _, s := range singularities {
		if s.XVal > xMin+epsMargin && s.XVal < xMax-epsMargin {
			singXs = append(singXs, s.XVal)
		}
	}
	sort.Float64s(singXs)
	for _, sx := range singXs {
		if len(splits) == 0 || sx-splits[len(splits)-1] > epsMargin*2 {
			splits = append(splits, sx)
		}
	}
	splits = append(splits, xMax)

	xSpan := xMax - xMin
	ySpan := yMax - yMin
	if ySpan <= 0 {
		ySpan = 1
	}

	// Adaptive subdivision within each continuous partition, with total evaluation budget
	const maxTotalPoints = 900
	const maxDepth = 6
	totalPoints := 0

	type samplePair struct {
		x, y  float64
		valid bool
	}

	for segIdx := 0; segIdx < len(splits)-1; segIdx++ {
		segStart := splits[segIdx]
		segEnd := splits[segIdx+1]
		if segIdx > 0 {
			segStart += epsMargin
		}
		if segIdx < len(splits)-2 {
			segEnd -= epsMargin
		}
		if segStart >= segEnd {
			continue
		}

		// Initial grid with slight irrational interior shift to avoid periodic aliasing
		initN := 24
		if len(splits) > 2 {
			initN = 18
		}
		grid := make([]samplePair, initN+1)
		for i := 0; i <= initN; i++ {
			var x float64
			if i == 0 {
				x = segStart
			} else if i == initN {
				x = segEnd
			} else {
				jitter := (float64(i) + 0.0314159) / float64(initN)
				if jitter >= 1.0 {
					jitter = float64(i) / float64(initN)
				}
				x = segStart + jitter*(segEnd-segStart)
			}
			y, ok := evalSampleFloat(expr, varName, x)
			grid[i] = samplePair{x: x, y: y, valid: ok}
			totalPoints++
		}

		var rawSamples []samplePair
		var subdivide func(a, b samplePair, depth int)
		subdivide = func(a, b samplePair, depth int) {
			if depth >= maxDepth || totalPoints >= maxTotalPoints {
				rawSamples = append(rawSamples, b)
				return
			}
			// Do not refine deep outside viewport unless crossing viewport
			bothOutHigh := a.valid && b.valid && a.y > yMax+ySpan && b.y > yMax+ySpan
			bothOutLow := a.valid && b.valid && a.y < yMin-ySpan && b.y < yMin-ySpan
			if bothOutHigh || bothOutLow {
				rawSamples = append(rawSamples, b)
				return
			}

			mx := 0.5 * (a.x + b.x)
			my, mok := evalSampleFloat(expr, varName, mx)
			totalPoints++
			m := samplePair{x: mx, y: my, valid: mok}

			needRefine := false
			if !a.valid || !b.valid || !m.valid {
				needRefine = (a.valid != b.valid) || (a.valid && b.valid && !m.valid)
			} else {
				interpY := 0.5 * (a.y + b.y)
				if math.Abs(m.y-interpY) > ySpan*0.015 {
					needRefine = true
				}
			}

			if needRefine {
				subdivide(a, m, depth+1)
				subdivide(m, b, depth+1)
			} else {
				rawSamples = append(rawSamples, b)
			}
		}

		rawSamples = append(rawSamples, grid[0])
		for i := 0; i < len(grid)-1; i++ {
			subdivide(grid[i], grid[i+1], 0)
		}

		// Split rawSamples into clipped continuous PlotBranches
		var curBranch []PlotPoint
		flushBranch := func() {
			if len(curBranch) >= 2 {
				simplified := simplifyBranchRDP(curBranch, xSpan, ySpan, 0.002)
				scene.Branches = append(scene.Branches, PlotBranch{Points: simplified})
			} else if len(curBranch) == 1 {
				scene.Branches = append(scene.Branches, PlotBranch{Points: curBranch})
			}
			curBranch = nil
		}

		for i := 0; i < len(rawSamples)-1; i++ {
			p1 := rawSamples[i]
			p2 := rawSamples[i+1]
			if !p1.valid || !p2.valid {
				flushBranch()
				continue
			}

			// Fallback discontinuity check for undetected periodic poles (e.g. 1/sin(x), tan(x)+10):
			// large vertical jump exceeding 80% of viewport height where at least one endpoint leaves viewport or crosses midpoint
			midY := 0.5 * (yMin + yMax)
			if math.Abs(p2.y-p1.y) > ySpan*0.8 && (p1.y < yMin || p1.y > yMax || p2.y < yMin || p2.y > yMax || (p1.y-midY)*(p2.y-midY) < 0) {
				flushBranch()
				continue
			}

			cx0, cy0, cx1, cy1, visible := clipSegmentLiangBarsky(p1.x, p1.y, p2.x, p2.y, xMin, xMax, yMin, yMax)
			if !visible {
				flushBranch()
				continue
			}

			if len(curBranch) == 0 {
				curBranch = append(curBranch, PlotPoint{X: cx0, Y: cy0})
			} else {
				last := curBranch[len(curBranch)-1]
				if math.Hypot((cx0-last.X)/xSpan, (cy0-last.Y)/ySpan) > 1e-4 {
					flushBranch()
					curBranch = append(curBranch, PlotPoint{X: cx0, Y: cy0})
				}
			}
			curBranch = append(curBranch, PlotPoint{X: cx1, Y: cy1})
		}
		flushBranch()
	}

	return scene, nil
}

// RenderBraille renders a PlotScene onto a Unicode BrailleCanvas for terminal display.
func RenderBraille(scene *PlotScene) string {
	charW, charH := 50, 16
	canvas := newBrailleCanvas(charW, charH, scene.XMin, scene.XMax, scene.YMin, scene.YMax, scene.XMinNode, scene.XMaxNode, scene.YMinNode, scene.YMaxNode)
	canvas.Features = scene.Features

	for _, branch := range scene.Branches {
		if len(branch.Points) == 1 {
			px := canvas.toPixelX(branch.Points[0].X)
			py := canvas.toPixelY(branch.Points[0].Y)
			canvas.SetPixel(px, py)
			continue
		}
		for i := 0; i < len(branch.Points)-1; i++ {
			px0 := canvas.toPixelX(branch.Points[i].X)
			py0 := canvas.toPixelY(branch.Points[i].Y)
			px1 := canvas.toPixelX(branch.Points[i+1].X)
			py1 := canvas.toPixelY(branch.Points[i+1].Y)
			canvas.drawLine(px0, py0, px1, py1)
		}
	}

	return canvas.Render()
}

// EvalPlot parses arguments and renders a 2D exact terminal plot (with embedded SVG & Typst projections).
func EvalPlot(args []Node) (Node, error) {
	scene, err := BuildPlotScene(args)
	if err != nil {
		return nil, err
	}
	brailleStr := RenderBraille(scene)
	svgStr := RenderPlotSVG(scene)
	typstStr := RenderPlotTypst(scene)
	return NewPlotNodeWithVectors(brailleStr, svgStr, typstStr), nil
}
