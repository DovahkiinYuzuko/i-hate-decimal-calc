package calc

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// NodeType identifies the specific kind of AST node.
type NodeType int

const (
	NodeRational NodeType = iota
	NodeSqrt
	NodeConst
	NodeFunc
	NodeComplex
	NodeAdd
	NodeMul
	NodePow
	NodeUnaryOp
	NodeVar
	NodeList
	NodeMatrix
	NodePlot
	NodeRelOp
)

// Node represents any node in the mathematical expression tree.
type Node interface {
	Type() NodeType
	String() string
	Equal(other Node) bool
}

// -------------------------------------------------------------------------
// RationalNode (Fraction)
// -------------------------------------------------------------------------

// RationalNode represents an exact rational number using big.Rat.
type RationalNode struct {
	Val *big.Rat
}

// NewRational creates a new RationalNode with the given numerator and denominator.
// Automatically reduces the fraction. Returns an error if denominator is zero.
func NewRational(num, denom int64) (*RationalNode, error) {
	if denom == 0 {
		return nil, NewZeroDivisionError("division by zero: denominator cannot be zero")
	}
	r := new(big.Rat).SetFrac64(num, denom)
	return &RationalNode{Val: r}, nil
}

// NewRationalFromBigRat wraps an existing big.Rat into a RationalNode.
func NewRationalFromBigRat(r *big.Rat) *RationalNode {
	copied := new(big.Rat).Set(r)
	return &RationalNode{Val: copied}
}

func (n *RationalNode) Type() NodeType { return NodeRational }

func (n *RationalNode) String() string {
	if n.Val.IsInt() {
		return n.Val.Num().String()
	}
	return n.Val.String()
}

func (n *RationalNode) Equal(other Node) bool {
	o, ok := other.(*RationalNode)
	if !ok {
		return false
	}
	return n.Val.Cmp(o.Val) == 0
}

// -------------------------------------------------------------------------
// SqrtNode (Square Root)
// -------------------------------------------------------------------------

// SqrtNode represents a square root symbol: sqrt(Radicand).
type SqrtNode struct {
	Radicand Node
}

// NewSqrt creates a new SqrtNode.
func NewSqrt(radicand Node) *SqrtNode {
	return &SqrtNode{Radicand: radicand}
}

func (n *SqrtNode) Type() NodeType { return NodeSqrt }

func (n *SqrtNode) String() string {
	return fmt.Sprintf("sqrt(%s)", n.Radicand.String())
}

func (n *SqrtNode) Equal(other Node) bool {
	o, ok := other.(*SqrtNode)
	if !ok {
		return false
	}
	return n.Radicand.Equal(o.Radicand)
}

// -------------------------------------------------------------------------
// ConstNode (Constants: pi, e)
// -------------------------------------------------------------------------

// ConstNode represents a mathematical constant symbol like pi or e.
type ConstNode struct {
	Name string
}

// NewConst creates a new ConstNode. Allowed names: "pi", "e", "deg".
func NewConst(name string) (*ConstNode, error) {
	switch name {
	case "pi", "e", "deg":
		return &ConstNode{Name: name}, nil
	default:
		return nil, fmt.Errorf("unknown constant: %s", name)
	}
}

func (n *ConstNode) Type() NodeType { return NodeConst }

func (n *ConstNode) String() string { return n.Name }

func (n *ConstNode) Equal(other Node) bool {
	o, ok := other.(*ConstNode)
	if !ok {
		return false
	}
	return n.Name == o.Name
}

// -------------------------------------------------------------------------
// FuncNode (Functions: sin, cos, tan, log, ln)
// -------------------------------------------------------------------------

// FuncNode represents a function call with one or more arguments.
type FuncNode struct {
	Name string
	Args []Node
}

// NewFunc creates and validates a function node.
func NewFunc(name string, args []Node) (*FuncNode, error) {
	if err := ValidateFuncArgs(name, args); err != nil {
		return nil, err
	}
	return &FuncNode{Name: name, Args: args}, nil
}

func (n *FuncNode) Type() NodeType { return NodeFunc }

func (n *FuncNode) String() string {
	argStrs := make([]string, len(n.Args))
	for i, a := range n.Args {
		argStrs[i] = a.String()
	}
	return fmt.Sprintf("%s(%s)", n.Name, strings.Join(argStrs, ", "))
}

func (n *FuncNode) Equal(other Node) bool {
	o, ok := other.(*FuncNode)
	if !ok || n.Name != o.Name || len(n.Args) != len(o.Args) {
		return false
	}
	for i := range n.Args {
		if !n.Args[i].Equal(o.Args[i]) {
			return false
		}
	}
	return true
}

// -------------------------------------------------------------------------
// ComplexNode (Complex Number: a + b*i)
// -------------------------------------------------------------------------

// ComplexNode represents a complex number with real and imaginary parts.
type ComplexNode struct {
	Real Node
	Imag Node
}

// NewComplex creates a new ComplexNode.
func NewComplex(real, imag Node) *ComplexNode {
	return &ComplexNode{Real: real, Imag: imag}
}

func (n *ComplexNode) Type() NodeType { return NodeComplex }

func (n *ComplexNode) String() string {
	return fmt.Sprintf("%s + %s*i", n.Real.String(), n.Imag.String())
}

func (n *ComplexNode) Equal(other Node) bool {
	o, ok := other.(*ComplexNode)
	if !ok {
		return false
	}
	return n.Real.Equal(o.Real) && n.Imag.Equal(o.Imag)
}

// -------------------------------------------------------------------------
// AddNode (N-ary Addition)
// -------------------------------------------------------------------------

// AddNode represents an addition of multiple terms (N-ary flat tree).
type AddNode struct {
	Terms []Node
}

// NewAdd creates an AddNode and flattens nested additions according to associative law.
func NewAdd(terms []Node) *AddNode {
	var flattened []Node
	for _, t := range terms {
		if add, ok := t.(*AddNode); ok {
			flattened = append(flattened, add.Terms...)
		} else {
			flattened = append(flattened, t)
		}
	}
	return &AddNode{Terms: flattened}
}

func (n *AddNode) Type() NodeType { return NodeAdd }

func (n *AddNode) String() string {
	strs := make([]string, len(n.Terms))
	for i, t := range n.Terms {
		strs[i] = t.String()
	}
	return strings.Join(strs, " + ")
}

func (n *AddNode) Equal(other Node) bool {
	o, ok := other.(*AddNode)
	if !ok || len(n.Terms) != len(o.Terms) {
		return false
	}
	for i := range n.Terms {
		if !n.Terms[i].Equal(o.Terms[i]) {
			return false
		}
	}
	return true
}

// -------------------------------------------------------------------------
// MulNode (N-ary Multiplication)
// -------------------------------------------------------------------------

// MulNode represents a multiplication of multiple factors (N-ary flat tree).
type MulNode struct {
	Factors []Node
}

// NewMul creates a MulNode and flattens nested multiplications according to associative law.
func NewMul(factors []Node) *MulNode {
	var flattened []Node
	for _, f := range factors {
		if mul, ok := f.(*MulNode); ok {
			flattened = append(flattened, mul.Factors...)
		} else {
			flattened = append(flattened, f)
		}
	}
	return &MulNode{Factors: flattened}
}

func (n *MulNode) Type() NodeType { return NodeMul }

func (n *MulNode) String() string {
	if len(n.Factors) == 2 {
		if r, ok := n.Factors[0].(*RationalNode); ok && !r.Val.IsInt() {
			if pow, ok := n.Factors[1].(*PowNode); ok && isNegative(pow.Exp) {
				expRat, isRat := pow.Exp.(*RationalNode)
				if isRat && expRat.Val.Cmp(big.NewRat(-1, 1)) == 0 {
					dStr := r.Val.Denom().String()
					bStr := pow.Base.String()
					switch pow.Base.(type) {
					case *AddNode, *MulNode, *UnaryOpNode:
						bStr = fmt.Sprintf("(%s)", bStr)
					}
					if r.Val.Num().Cmp(big.NewInt(1)) == 0 {
						return fmt.Sprintf("1/(%s * %s)", dStr, bStr)
					} else if r.Val.Num().Cmp(big.NewInt(-1)) == 0 {
						return fmt.Sprintf("-1/(%s * %s)", dStr, bStr)
					}
				}
			} else if r.Val.Num().Cmp(big.NewInt(1)) == 0 {
				// 1/d * X -> X/d
				return fmt.Sprintf("%s/%s", n.Factors[1].String(), r.Val.Denom().String())
			} else if r.Val.Num().Cmp(big.NewInt(-1)) == 0 {
				// -1/d * X -> -X/d
				return fmt.Sprintf("-%s/%s", n.Factors[1].String(), r.Val.Denom().String())
			}
		}
	}
	strs := make([]string, len(n.Factors))
	for i, f := range n.Factors {
		strs[i] = f.String()
	}
	return strings.Join(strs, " * ")
}

func (n *MulNode) Equal(other Node) bool {
	o, ok := other.(*MulNode)
	if !ok || len(n.Factors) != len(o.Factors) {
		return false
	}
	for i := range n.Factors {
		if !n.Factors[i].Equal(o.Factors[i]) {
			return false
		}
	}
	return true
}

// -------------------------------------------------------------------------
// PowNode (Power / Exponentiation)
// -------------------------------------------------------------------------

// PowNode represents Base^Exp.
type PowNode struct {
	Base Node
	Exp  Node
}

// NewPow creates a PowNode. Validates that 0^(negative) returns a division by zero error.
func NewPow(base, exp Node) (*PowNode, error) {
	if ratBase, ok := base.(*RationalNode); ok && ratBase.Val.Sign() == 0 {
		if ratExp, ok := exp.(*RationalNode); ok && ratExp.Val.Sign() < 0 {
			return nil, NewZeroDivisionError("division by zero: 0^(negative number) is undefined")
		}
	}
	return &PowNode{Base: base, Exp: exp}, nil
}

func (n *PowNode) Type() NodeType { return NodePow }

func (n *PowNode) String() string {
	baseStr := n.Base.String()
	switch n.Base.(type) {
	case *AddNode, *MulNode, *UnaryOpNode, *ComplexNode:
		baseStr = fmt.Sprintf("(%s)", baseStr)
	}
	return fmt.Sprintf("%s^%s", baseStr, n.Exp.String())
}

func (n *PowNode) Equal(other Node) bool {
	o, ok := other.(*PowNode)
	if !ok {
		return false
	}
	return n.Base.Equal(o.Base) && n.Exp.Equal(o.Exp)
}

// -------------------------------------------------------------------------
// UnaryOpNode (Unary Operators: -, !)
// -------------------------------------------------------------------------

// UnaryOpNode represents a unary operation (negation '-' or factorial '!').
type UnaryOpNode struct {
	Op   string
	Expr Node
}

// NewUnaryOp creates a UnaryOpNode. Validates that factorial is applied only to non-negative integers.
func NewUnaryOp(op string, expr Node) (*UnaryOpNode, error) {
	switch op {
	case "-":
		return &UnaryOpNode{Op: op, Expr: expr}, nil

	case "!":
		if rat, ok := expr.(*RationalNode); ok {
			if !rat.Val.IsInt() {
				return nil, NewDomainError("errors.domain_factorial_non_int", "factorial domain error: factorial of non-integer rational (%s) is undefined", rat.String())
			}
			if rat.Val.Sign() < 0 {
				return nil, NewDomainError("errors.domain_factorial_negative", "factorial domain error: factorial of negative integer (%s) is undefined", rat.String())
			}
		}
		return &UnaryOpNode{Op: op, Expr: expr}, nil

	default:
		return nil, fmt.Errorf("unknown unary operator: %s", op)
	}
}

func (n *UnaryOpNode) Type() NodeType { return NodeUnaryOp }

func (n *UnaryOpNode) String() string {
	if n.Op == "!" {
		return fmt.Sprintf("%s!", n.Expr.String())
	}
	return fmt.Sprintf("%s%s", n.Op, n.Expr.String())
}

func (n *UnaryOpNode) Equal(other Node) bool {
	o, ok := other.(*UnaryOpNode)
	if !ok || n.Op != o.Op {
		return false
	}
	return n.Expr.Equal(o.Expr)
}

// -------------------------------------------------------------------------
// VarNode (Variable Symbol)
// -------------------------------------------------------------------------

// VarNode represents a variable symbol (e.g., "x", "ans").
type VarNode struct {
	Name string
}

// NewVar creates a new VarNode with the given name.
func NewVar(name string) *VarNode {
	return &VarNode{Name: name}
}

func (n *VarNode) Type() NodeType { return NodeVar }

func (n *VarNode) String() string {
	return n.Name
}

func (n *VarNode) Equal(other Node) bool {
	o, ok := other.(*VarNode)
	if !ok {
		return false
	}
	return n.Name == o.Name
}

// -------------------------------------------------------------------------
// AssignStmt (Variable Assignment Statement)
// -------------------------------------------------------------------------

// AssignStmt represents a variable assignment statement (e.g., "x = 1/2").
type AssignStmt struct {
	Name  string
	Value Node
}

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
		return nil, fmt.Errorf("matrix dimension error: rows and cols must be >= 1, got %dx%d", rows, cols)
	}
	if len(data) != rows {
		return nil, fmt.Errorf("matrix dimension error: expected %d rows, got %d", rows, len(data))
	}
	for r, row := range data {
		if len(row) != cols {
			return nil, fmt.Errorf("matrix dimension error: row %d has %d elements, expected %d", r, len(row), cols)
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
// RelOpNode (Relational Operation: >, <, >=, <=)
// -------------------------------------------------------------------------

// RelOpNode represents a binary relational operation such as x > 0 or n <= 5.
type RelOpNode struct {
	LHS Node
	Op  string
	RHS Node
}

// NewRelOp creates a new RelOpNode.
func NewRelOp(lhs Node, op string, rhs Node) *RelOpNode {
	return &RelOpNode{LHS: lhs, Op: op, RHS: rhs}
}

func (n *RelOpNode) Type() NodeType { return NodeRelOp }
func (n *RelOpNode) String() string {
	return fmt.Sprintf("%s %s %s", n.LHS.String(), n.Op, n.RHS.String())
}
func (n *RelOpNode) Equal(other Node) bool {
	o, ok := other.(*RelOpNode)
	if !ok {
		return false
	}
	return n.Op == o.Op && n.LHS.Equal(o.LHS) && n.RHS.Equal(o.RHS)
}

// -------------------------------------------------------------------------
// AST Traversal & Transformation (Walker Pattern)
// -------------------------------------------------------------------------

// Walk traverses an AST in depth-first pre-order.
// If visitor returns false, Walk does not visit node's children.
func Walk(node Node, visitor func(Node) bool) {
	if node == nil {
		return
	}
	if !visitor(node) {
		return
	}

	switch v := node.(type) {
	case *SqrtNode:
		Walk(v.Radicand, visitor)
	case *FuncNode:
		for _, arg := range v.Args {
			Walk(arg, visitor)
		}
	case *ComplexNode:
		Walk(v.Real, visitor)
		Walk(v.Imag, visitor)
	case *AddNode:
		for _, term := range v.Terms {
			Walk(term, visitor)
		}
	case *MulNode:
		for _, factor := range v.Factors {
			Walk(factor, visitor)
		}
	case *PowNode:
		Walk(v.Base, visitor)
		Walk(v.Exp, visitor)
	case *UnaryOpNode:
		Walk(v.Expr, visitor)
	case *ListNode:
		for _, elem := range v.Elements {
			Walk(elem, visitor)
		}
	case *MatrixNode:
		for _, row := range v.Data {
			for _, cell := range row {
				Walk(cell, visitor)
			}
		}
	case *RelOpNode:
		Walk(v.LHS, visitor)
		Walk(v.RHS, visitor)
	case *RationalNode, *ConstNode, *VarNode, *PlotNode:
		// Leaf nodes: no children
	}
}

// Inspect traverses an AST unconditionally in depth-first order.
func Inspect(node Node, visitor func(Node)) {
	Walk(node, func(n Node) bool {
		visitor(n)
		return true
	})
}

// Transform traverses an AST in post-order and transforms nodes bottom-up.
// Child nodes are transformed first, and then transformer is called on the newly constructed node.
func Transform(node Node, transformer func(Node) Node) Node {
	if node == nil {
		return nil
	}

	var transformedChild Node
	switch v := node.(type) {
	case *SqrtNode:
		transformedChild = &SqrtNode{
			Radicand: Transform(v.Radicand, transformer),
		}
	case *FuncNode:
		newArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			newArgs[i] = Transform(a, transformer)
		}
		transformedChild = &FuncNode{
			Name: v.Name,
			Args: newArgs,
		}
	case *ComplexNode:
		transformedChild = &ComplexNode{
			Real: Transform(v.Real, transformer),
			Imag: Transform(v.Imag, transformer),
		}
	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = Transform(t, transformer)
		}
		transformedChild = &AddNode{
			Terms: newTerms,
		}
	case *MulNode:
		newFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			newFactors[i] = Transform(f, transformer)
		}
		transformedChild = &MulNode{
			Factors: newFactors,
		}
	case *PowNode:
		transformedChild = &PowNode{
			Base: Transform(v.Base, transformer),
			Exp:  Transform(v.Exp, transformer),
		}
	case *UnaryOpNode:
		transformedChild = &UnaryOpNode{
			Op:   v.Op,
			Expr: Transform(v.Expr, transformer),
		}
	case *ListNode:
		newElems := make([]Node, len(v.Elements))
		for i, e := range v.Elements {
			newElems[i] = Transform(e, transformer)
		}
		transformedChild = &ListNode{
			Elements: newElems,
		}
	case *MatrixNode:
		newData := make([][]Node, v.Rows)
		for r := 0; r < v.Rows; r++ {
			newData[r] = make([]Node, v.Cols)
			for c := 0; c < v.Cols; c++ {
				newData[r][c] = Transform(v.Data[r][c], transformer)
			}
		}
		transformedChild = &MatrixNode{
			Rows: v.Rows,
			Cols: v.Cols,
			Data: newData,
		}
	case *RelOpNode:
		transformedChild = &RelOpNode{
			LHS: Transform(v.LHS, transformer),
			Op:  v.Op,
			RHS: Transform(v.RHS, transformer),
		}
	case *RationalNode, *ConstNode, *VarNode, *PlotNode:
		transformedChild = node
	default:
		transformedChild = node
	}

	return transformer(transformedChild)
}

// Substitute replaces occurrences of varName with valNode in an expression tree.
func Substitute(node Node, varName string, valNode Node) Node {
	return Transform(node, func(curr Node) Node {
		if v, ok := curr.(*VarNode); ok && v.Name == varName {
			return valNode
		}
		return curr
	})
}

// ContainsVar checks if an AST node contains the given variable name.
func ContainsVar(node Node, varName string) bool {
	found := false
	Walk(node, func(curr Node) bool {
		if found {
			return false
		}
		if v, ok := curr.(*VarNode); ok && v.Name == varName {
			found = true
			return false
		}
		return true
	})
	return found
}

// ExtractFreeVariables collects all distinct variable names occurring in the node, sorted alphabetically.
func ExtractFreeVariables(node Node) []string {
	varMap := make(map[string]bool)
	Walk(node, func(curr Node) bool {
		if v, ok := curr.(*VarNode); ok {
			varMap[v.Name] = true
		}
		return true
	})
	if len(varMap) == 0 {
		return nil
	}
	vars := make([]string, 0, len(varMap))
	for name := range varMap {
		vars = append(vars, name)
	}
	sort.Strings(vars)
	return vars
}



