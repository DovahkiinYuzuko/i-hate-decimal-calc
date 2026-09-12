package calc

import (
	"fmt"
	"math"
	"math/big"
	"strings"
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

func (c *BrailleCanvas) setPixel(px, py int) {
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
		c.setPixel(currX, currY)
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
					if px < c.PixelW && py < c.PixelH && c.Dots[py][px] {
						mask |= brailleDotMap[dr][dc]
					}
				}
			}
			if mask != 0 {
				grid[cy][cx] = rune(0x2800 + mask)
			} else {
				grid[cy][cx] = ' '
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
				if grid[cy][cxAxis] == ' ' {
					grid[cy][cxAxis] = '│'
				} else if grid[cy][cxAxis] == '─' {
					grid[cy][cxAxis] = '┼'
				}
			}
		}
	}

	// Place feature markers (◆ for roots, ● for extrema) and pin labels
	for _, feat := range c.Features {
		if feat.Kind == "asymptote" {
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
	sb.WriteString("[CAS Features Detected]\n")
	hasFeature := false
	for _, feat := range c.Features {
		hasFeature = true
		switch feat.Kind {
		case "root":
			sb.WriteString(fmt.Sprintf("* 零点 (Zero):       x = %s\n", Format(feat.XNode)))
		case "min":
			sb.WriteString(fmt.Sprintf("* 極小値 (Local Min): (%s, %s)\n", Format(feat.XNode), Format(feat.YNode)))
		case "max":
			sb.WriteString(fmt.Sprintf("* 極大値 (Local Max): (%s, %s)\n", Format(feat.XNode), Format(feat.YNode)))
		case "asymptote":
			sb.WriteString(fmt.Sprintf("* 漸近線 (Asymptote): x = %s (不連続分離)\n", Format(feat.XNode)))
		}
	}
	if !hasFeature {
		sb.WriteString("* 特異点・極値なし（単調・滑らかな曲線）\n")
	}
	sb.WriteString(fmt.Sprintf("* 表示領域 (Domain):  x ∈ [%s, %s], y ∈ [%s, %s]\n",
		Format(c.XMinNode), Format(c.XMaxNode), Format(c.YMinNode), Format(c.YMaxNode)))

	return sb.String()
}

// evalNodeFloat evaluates an AST node to a real float64 using approx.go evalComplex.
func evalNodeFloat(n Node) (float64, error) {
	cVal, err := evalComplex(n)
	if err != nil {
		return 0, err
	}
	if math.Abs(imag(cVal)) > 1e-12 {
		return 0, fmt.Errorf("complex result %v not supported on real plot", cVal)
	}
	r := real(cVal)
	if math.IsNaN(r) || math.IsInf(r, 0) {
		return 0, fmt.Errorf("result is NaN or Inf")
	}
	return r, nil
}

// substituteVar replaces occurrences of varName with valNode in an expression tree.
func substituteVar(n Node, varName string, valNode Node) Node {
	if n == nil {
		return nil
	}
	switch v := n.(type) {
	case *VarNode:
		if v.Name == varName {
			return valNode
		}
		return v
	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = substituteVar(t, varName, valNode)
		}
		return &AddNode{Terms: newTerms}
	case *MulNode:
		newFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			newFactors[i] = substituteVar(f, varName, valNode)
		}
		return &MulNode{Factors: newFactors}
	case *PowNode:
		return &PowNode{
			Base: substituteVar(v.Base, varName, valNode),
			Exp:  substituteVar(v.Exp, varName, valNode),
		}
	case *SqrtNode:
		return &SqrtNode{Radicand: substituteVar(v.Radicand, varName, valNode)}
	case *FuncNode:
		newArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			newArgs[i] = substituteVar(a, varName, valNode)
		}
		return &FuncNode{Name: v.Name, Args: newArgs}
	case *UnaryOpNode:
		return &UnaryOpNode{Op: v.Op, Expr: substituteVar(v.Expr, varName, valNode)}
	default:
		return n
	}
}

// findFreeVariable finds the variable in the expression.
func findFreeVariable(n Node) string {
	vars := make(map[string]bool)
	var walk func(node Node)
	walk = func(node Node) {
		if node == nil {
			return
		}
		switch v := node.(type) {
		case *VarNode:
			vars[v.Name] = true
		case *AddNode:
			for _, t := range v.Terms {
				walk(t)
			}
		case *MulNode:
			for _, f := range v.Factors {
				walk(f)
			}
		case *PowNode:
			walk(v.Base)
			walk(v.Exp)
		case *SqrtNode:
			walk(v.Radicand)
		case *FuncNode:
			for _, a := range v.Args {
				walk(a)
			}
		case *UnaryOpNode:
			walk(v.Expr)
		}
	}
	walk(n)
	for v := range vars {
		return v
	}
	return "x"
}

// parseDomain extracts [lower, upper] nodes and their float values.
func parseDomain(n Node) (Node, Node, float64, float64, error) {
	var lowerNode, upperNode Node

	switch v := n.(type) {
	case *ListNode:
		if len(v.Elements) != 2 {
			return nil, nil, 0, 0, fmt.Errorf("domain must have 2 elements [min, max], got %d", len(v.Elements))
		}
		lowerNode = v.Elements[0]
		upperNode = v.Elements[1]
	case *MatrixNode:
		if v.Rows == 1 && v.Cols == 2 {
			lowerNode = v.Data[0][0]
			upperNode = v.Data[0][1]
		} else {
			return nil, nil, 0, 0, fmt.Errorf("matrix domain must be 1x2 [min, max]")
		}
	default:
		return nil, nil, 0, 0, fmt.Errorf("domain must be a list [min, max], got %s", n.String())
	}

	lowerVal, err := evalNodeFloat(lowerNode)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("evaluating domain min: %v", err)
	}
	upperVal, err := evalNodeFloat(upperNode)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("evaluating domain max: %v", err)
	}

	if lowerVal >= upperVal {
		return nil, nil, 0, 0, fmt.Errorf("%s", MsgErrPlotInvalidDomain)
	}

	return lowerNode, upperNode, lowerVal, upperVal, nil
}

// findAsymptotes identifies vertical asymptotes where denominator is 0.
func findAsymptotes(expr Node, varName string, xMin, xMax float64) []plotFeature {
	var asyms []plotFeature
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

	for _, denom := range denominators {
		solList, err := solveEquation(denom, varName)
		if err == nil {
			if list, ok := solList.(*ListNode); ok {
				for _, sol := range list.Elements {
					val, err := evalNodeFloat(sol)
					if err == nil && val >= xMin && val <= xMax {
						asyms = append(asyms, plotFeature{
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

	return asyms
}

// EvalPlot parses arguments and renders a 2D exact terminal plot.
func EvalPlot(args []Node) (Node, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("plot requires 2 or 3 arguments (expr, domainX, [domainY]), got %d", len(args))
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

	// Canvas dimensions (default: 50 cells wide x 16 lines high)
	charW, charH := 50, 16
	pixelW := charW * 2
	pixelH := charH * 4

	// Sample points to find range if y is not explicit
	type samplePoint struct {
		x     float64
		y     float64
		valid bool
	}
	samples := make([]samplePoint, pixelW)
	var validYs []float64

	for px := 0; px < pixelW; px++ {
		frac := float64(px) / float64(pixelW-1)
		x := xMin + frac*(xMax-xMin)

		// Substitute x into expression
		rNode := NewRationalFromBigRat(new(big.Rat).SetFloat64(x))
		subExpr := substituteVar(expr, varName, rNode)
		y, err := evalNodeFloat(subExpr)
		if err != nil || math.IsNaN(y) || math.IsInf(y, 0) {
			samples[px] = samplePoint{x: x, valid: false}
		} else {
			samples[px] = samplePoint{x: x, y: y, valid: true}
			validYs = append(validYs, y)
		}
	}

	if !hasExplicitY {
		if len(validYs) == 0 {
			yMin, yMax = -2, 2
			yMinNode = mustRational(-2, 1)
			yMaxNode = mustRational(2, 1)
		} else {
			minY, maxY := validYs[0], validYs[0]
			for _, y := range validYs {
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
			// Add 10% padding
			span := maxY - minY
			if span < 1e-6 {
				span = 2
			}
			yMin = minY - span*0.1
			yMax = maxY + span*0.1
			yMinNode = NewRationalFromBigRat(new(big.Rat).SetFloat64(math.Round(yMin)))
			yMaxNode = NewRationalFromBigRat(new(big.Rat).SetFloat64(math.Round(yMax)))
		}
	}

	canvas := newBrailleCanvas(charW, charH, xMin, xMax, yMin, yMax, xMinNode, xMaxNode, yMinNode, yMaxNode)

	// CAS Feature Detection:
	// 1. Zeros (Roots): solve(f(x) == 0, x)
	solList, err := solveEquation(expr, varName)
	if err == nil {
		if list, ok := solList.(*ListNode); ok {
			for _, root := range list.Elements {
				rVal, err := evalNodeFloat(root)
				if err == nil && rVal >= xMin && rVal <= xMax {
					canvas.Features = append(canvas.Features, plotFeature{
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

	// 2. Extrema: diff(f(x), x) == 0
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
								// Test second derivative for min/max
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
								canvas.Features = append(canvas.Features, plotFeature{
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

	// 3. Asymptotes: denominator == 0
	asyms := findAsymptotes(expr, varName, xMin, xMax)
	canvas.Features = append(canvas.Features, asyms...)

	// Rasterize curve with Tupper discontinuity isolation
	var prevPx, prevPy int
	var prevValid bool

	for px := 0; px < pixelW; px++ {
		sp := samples[px]
		if !sp.valid {
			prevValid = false
			continue
		}

		py := canvas.toPixelY(sp.y)

		// Check asymptote crossing
		hasAsymBetween := false
		for _, asym := range asyms {
			asymPx := canvas.toPixelX(asym.XVal)
			if (prevPx <= asymPx && asymPx <= px) || (px <= asymPx && asymPx <= prevPx) {
				hasAsymBetween = true
				break
			}
		}

		// Tupper discontinuity condition: jump > half height without a valid continuous slope
		isDiscontinuous := hasAsymBetween || (prevValid && math.Abs(float64(py-prevPy)) > float64(pixelH)/2)

		if prevValid && !isDiscontinuous {
			canvas.drawLine(prevPx, prevPy, px, py)
		} else {
			canvas.setPixel(px, py)
		}

		prevPx = px
		prevPy = py
		prevValid = true
	}

	return NewPlotNode(canvas.Render()), nil
}
