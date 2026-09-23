package ast

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"strings"
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
