package ast

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// ListNode (Multiple Elements / Equation Roots)
// -------------------------------------------------------------------------

// ListNode represents a list of elements (e.g., roots of an equation "[2, 3]").
type ListNode struct {
	Elements []Node
}

// NewList creates a new ListNode with the given elements.
func NewList(elements []Node) *ListNode {
	return &ListNode{Elements: elements}
}

func (n *ListNode) Type() NodeType { return NodeList }

func (n *ListNode) String() string {
	strs := make([]string, len(n.Elements))
	for i, e := range n.Elements {
		strs[i] = e.String()
	}
	return fmt.Sprintf("[%s]", strings.Join(strs, ", "))
}

func (n *ListNode) Equal(other Node) bool {
	o, ok := other.(*ListNode)
	if !ok || len(n.Elements) != len(o.Elements) {
		return false
	}
	for i := range n.Elements {
		if !n.Elements[i].Equal(o.Elements[i]) {
			return false
		}
	}
	return true
}

// -------------------------------------------------------------------------
// MatrixNode (2D Mathematical Matrix)
// -------------------------------------------------------------------------

// MatrixNode represents an exact 2D matrix of mathematical expression nodes.
type MatrixNode struct {
	Rows int
	Cols int
	Data [][]Node
}

// NewMatrix creates a new MatrixNode and validates that all rows have identical column length.
func NewMatrix(rows, cols int, data [][]Node) (*MatrixNode, error) {
	if rows <= 0 || cols <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("ast.err_matrix_dimension_error_rows_and", rows, cols))
	}
	if len(data) != rows {
		return nil, fmt.Errorf("%s", i18n.T("ast.err_matrix_dimension_error_expected_got", rows, len(data)))
	}
	for r, row := range data {
		if len(row) != cols {
			return nil, fmt.Errorf("%s", i18n.T("ast.err_matrix_dimension_error_row_expected", r, len(row), cols))
		}
	}
	return &MatrixNode{Rows: rows, Cols: cols, Data: data}, nil
}

func (n *MatrixNode) Type() NodeType { return NodeMatrix }

func (n *MatrixNode) String() string {
	rowStrs := make([]string, n.Rows)
	for r := 0; r < n.Rows; r++ {
		elemStrs := make([]string, n.Cols)
		for c := 0; c < n.Cols; c++ {
			elemStrs[c] = n.Data[r][c].String()
		}
		rowStrs[r] = fmt.Sprintf("[%s]", strings.Join(elemStrs, ", "))
	}
	return fmt.Sprintf("[%s]", strings.Join(rowStrs, ", "))
}

func (n *MatrixNode) Equal(other Node) bool {
	o, ok := other.(*MatrixNode)
	if !ok || n.Rows != o.Rows || n.Cols != o.Cols {
		return false
	}
	for r := 0; r < n.Rows; r++ {
		for c := 0; c < n.Cols; c++ {
			if !n.Data[r][c].Equal(o.Data[r][c]) {
				return false
			}
		}
	}
	return true
}

// -------------------------------------------------------------------------
// PlotNode (Rendered 2D Terminal Plot)
// -------------------------------------------------------------------------

// PlotNode represents a rendered 2D terminal plot string.
type PlotNode struct {
	Content string
}

func NewPlotNode(content string) *PlotNode {
	return &PlotNode{Content: content}
}

func (n *PlotNode) Type() NodeType { return NodePlot }
func (n *PlotNode) String() string { return n.Content }
func (n *PlotNode) Equal(other Node) bool {
	if o, ok := other.(*PlotNode); ok {
		return n.Content == o.Content
	}
	return false
}

// -------------------------------------------------------------------------
// PiecewiseNode (Piecewise Algebraic Function)
// -------------------------------------------------------------------------

// PiecewiseCase represents a single branch (expression, condition) in a piecewise definition.
type PiecewiseCase struct {
	Expr      Node
	Condition Node
}

// PiecewiseNode represents a piecewise-defined mathematical function.
// e.g., piecewise([[expr1, cond1], [expr2, cond2]], otherwise)
type PiecewiseNode struct {
	Cases     []PiecewiseCase
	Otherwise Node // optional default expression, can be nil
}

// NewPiecewiseNode creates a new PiecewiseNode.
func NewPiecewiseNode(cases []PiecewiseCase, otherwise Node) *PiecewiseNode {
	return &PiecewiseNode{
		Cases:     cases,
		Otherwise: otherwise,
	}
}

func (n *PiecewiseNode) Type() NodeType { return NodePiecewise }

func (n *PiecewiseNode) String() string {
	caseStrs := make([]string, len(n.Cases))
	for i, c := range n.Cases {
		condStr := "true"
		if c.Condition != nil {
			condStr = c.Condition.String()
		}
		caseStrs[i] = fmt.Sprintf("[%s, %s]", c.Expr.String(), condStr)
	}
	casesBlock := fmt.Sprintf("[%s]", strings.Join(caseStrs, ", "))
	if n.Otherwise != nil {
		return fmt.Sprintf("piecewise(%s, %s)", casesBlock, n.Otherwise.String())
	}
	return fmt.Sprintf("piecewise(%s)", casesBlock)
}

func (n *PiecewiseNode) Equal(other Node) bool {
	o, ok := other.(*PiecewiseNode)
	if !ok || len(n.Cases) != len(o.Cases) {
		return false
	}
	for i := range n.Cases {
		if !n.Cases[i].Expr.Equal(o.Cases[i].Expr) {
			return false
		}
		c1 := n.Cases[i].Condition
		c2 := o.Cases[i].Condition
		if (c1 == nil && c2 != nil) || (c1 != nil && c2 == nil) {
			return false
		}
		if c1 != nil && c2 != nil && !c1.Equal(c2) {
			return false
		}
	}
	if (n.Otherwise == nil && o.Otherwise != nil) || (n.Otherwise != nil && o.Otherwise == nil) {
		return false
	}
	if n.Otherwise != nil && o.Otherwise != nil && !n.Otherwise.Equal(o.Otherwise) {
		return false
	}
	return true
}

// -------------------------------------------------------------------------
// IntervalNode (Certified Rational Interval Enclosure [Low, High])
// -------------------------------------------------------------------------

// IntervalNode represents a certified closed interval [Low, High] subset of Q
// with arbitrary-precision rational endpoints guaranteeing enclosure of the true value.
type IntervalNode struct {
	Low  *big.Rat
	High *big.Rat
}

// NewIntervalNode creates a new IntervalNode ensuring Low <= High.
func NewIntervalNode(low, high *big.Rat) *IntervalNode {
	l := new(big.Rat).Set(low)
	h := new(big.Rat).Set(high)
	if l.Cmp(h) > 0 {
		l, h = h, l
	}
	return &IntervalNode{Low: l, High: h}
}

func (n *IntervalNode) Type() NodeType { return NodeInterval }

func (n *IntervalNode) String() string {
	width := new(big.Rat).Sub(n.High, n.Low)
	return fmt.Sprintf("[%s, %s] (width: %s)", n.Low.RatString(), n.High.RatString(), width.RatString())
}

func (n *IntervalNode) Equal(other Node) bool {
	o, ok := other.(*IntervalNode)
	if !ok {
		return false
	}
	return n.Low.Cmp(o.Low) == 0 && n.High.Cmp(o.High) == 0
}

// -------------------------------------------------------------------------
// StringNode (Arbitrary Base String / Text Literal)
// -------------------------------------------------------------------------

// StringNode represents a string value such as a base conversion representation or string literal.
type StringNode struct {
	Value string
}

// NewStringNode creates a new StringNode.
func NewStringNode(val string) *StringNode {
	return &StringNode{Value: val}
}

func (n *StringNode) Type() NodeType { return NodeString }
func (n *StringNode) String() string { return n.Value }
func (n *StringNode) Equal(other Node) bool {
	if o, ok := other.(*StringNode); ok {
		return n.Value == o.Value
	}
	return false
}


