package calc

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/parser"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterHandler("clifford", HandleClifford)
	RegisterHandler("clifford_wedge", HandleCliffordWedge)
	RegisterHandler("clifford_contract", HandleCliffordContract)
	RegisterHandler("clifford_dual", HandleCliffordDual)
	RegisterHandler("quantum_equiv", HandleQuantumEquiv)
	RegisterHandler("quantum_eval", HandleQuantumEval)
}

// parseMetricArgs parses metric signature arguments starting at index startIdx.
func parseMetricArgs(args []Node, startIdx int) (*Metric, error) {
	if len(args) < startIdx+2 {
		return nil, fmt.Errorf("%s", i18n.T("errors.clifford_metric_int"))
	}

	toInt := func(n Node) (int, error) {
		if r, ok := n.(*RationalNode); ok && r.Val.IsInt() {
			val := r.Val.Num().Int64()
			if val >= 0 && val <= 64 {
				return int(val), nil
			}
		}
		return -1, fmt.Errorf("%s", i18n.T("errors.clifford_metric_int"))
	}

	p, err := toInt(args[startIdx])
	if err != nil {
		return nil, err
	}
	q, err := toInt(args[startIdx+1])
	if err != nil {
		return nil, err
	}
	r := 0
	if len(args) > startIdx+2 {
		r, err = toInt(args[startIdx+2])
		if err != nil {
			return nil, err
		}
	}

	if p+q+r > 64 {
		return nil, fmt.Errorf("%s", i18n.T("errors.clifford_dim_limit", p+q+r))
	}

	return NewMetric(p, q, r)
}

// evalCliffordExpr recursively evaluates an AST expression in Clifford algebra Cl(p, q, r).
func evalCliffordExpr(n Node, m *Metric, env *Env) (*Multivector, error) {
	if n == nil {
		return NewMultivector(m, nil), nil
	}

	switch node := n.(type) {
	case *RationalNode:
		return NewScalarMultivector(m, node), nil

	case *VarNode:
		// Check for basis vector e0, e1, ...
		if strings.HasPrefix(node.Name, "e") && len(node.Name) > 1 {
			idx, err := strconv.Atoi(node.Name[1:])
			if err == nil && idx >= 0 && idx < m.Dim {
				return NewBasisBlade(m, idx), nil
			}
		}
		// Try lookup in env
		if env != nil {
			if val, ok := env.Get(node.Name); ok {
				return evalCliffordExpr(val, m, env)
			}
		}
		// Treat as a scalar symbolic variable
		return NewScalarMultivector(m, node), nil

	case *StringNode:
		// Parse string expression as AST
		parsed, err := parser.Parse(node.Value)
		if err != nil {
			return nil, err
		}
		return evalCliffordExpr(parsed, m, env)

	case *AddNode:
		res := NewMultivector(m, nil)
		for _, term := range node.Terms {
			mvTerm, err := evalCliffordExpr(term, m, env)
			if err != nil {
				return nil, err
			}
			res, err = res.Add(mvTerm)
			if err != nil {
				return nil, err
			}
		}
		return res, nil

	case *MulNode:
		res := NewScalarMultivector(m, ast.NewRationalFromBigRat(big.NewRat(1, 1)))
		for _, factor := range node.Factors {
			mvFactor, err := evalCliffordExpr(factor, m, env)
			if err != nil {
				return nil, err
			}
			var errProd error
			res, errProd = res.GeometricProduct(mvFactor)
			if errProd != nil {
				return nil, errProd
			}
		}
		return res, nil

	case *UnaryOpNode:
		if node.Op == "-" {
			mv, err := evalCliffordExpr(node.Expr, m, env)
			if err != nil {
				return nil, err
			}
			return mv.Scale(ast.NewRationalFromBigRat(big.NewRat(-1, 1))), nil
		}
		return evalCliffordExpr(node.Expr, m, env)

	case *PowNode:
		baseMv, err := evalCliffordExpr(node.Base, m, env)
		if err != nil {
			return nil, err
		}
		// Case 1: Base ^ Exp where Exp is integer power
		if ratExp, ok := node.Exp.(*RationalNode); ok && ratExp.Val.IsInt() {
			expVal := ratExp.Val.Num().Int64()
			if expVal == 0 {
				return NewScalarMultivector(m, ast.NewRationalFromBigRat(big.NewRat(1, 1))), nil
			}
			if expVal == -1 {
				return baseMv.TryInverse()
			}
			if expVal > 0 && expVal <= 16 {
				curr := baseMv
				for i := int64(1); i < expVal; i++ {
					curr, err = curr.GeometricProduct(baseMv)
					if err != nil {
						return nil, err
					}
				}
				return curr, nil
			}
		}
		// Case 2: Outer product (e0 ^ e1 parsed as PowNode)
		expMv, errExp := evalCliffordExpr(node.Exp, m, env)
		if errExp == nil {
			return baseMv.Wedge(expMv)
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.clifford_unsupported_power", node.String()))

	case *FuncNode:
		switch node.Name {
		case "wedge":
			if len(node.Args) != 2 {
				return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "wedge", 2, 2, len(node.Args)))
			}
			a, err := evalCliffordExpr(node.Args[0], m, env)
			if err != nil {
				return nil, err
			}
			b, err := evalCliffordExpr(node.Args[1], m, env)
			if err != nil {
				return nil, err
			}
			return a.Wedge(b)
		case "inv":
			if len(node.Args) != 1 {
				return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "inv", 1, 1, len(node.Args)))
			}
			a, err := evalCliffordExpr(node.Args[0], m, env)
			if err != nil {
				return nil, err
			}
			return a.TryInverse()
		case "rev":
			if len(node.Args) != 1 {
				return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "rev", 1, 1, len(node.Args)))
			}
			a, err := evalCliffordExpr(node.Args[0], m, env)
			if err != nil {
				return nil, err
			}
			return a.Reverse(), nil
		case "dual":
			if len(node.Args) != 1 {
				return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "dual", 1, 1, len(node.Args)))
			}
			a, err := evalCliffordExpr(node.Args[0], m, env)
			if err != nil {
				return nil, err
			}
			return a.PoincareDual(), nil
		default:
			return NewScalarMultivector(m, node), nil
		}

	default:
		return NewScalarMultivector(m, n), nil
	}
}

// HandleClifford simplifies a multivector expression in Cl(p, q, [r]).
func HandleClifford(args []Node, env *Env) (Node, error) {
	if len(args) < 3 || len(args) > 4 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "clifford", 3, 4, len(args)))
	}
	m, err := parseMetricArgs(args, 1)
	if err != nil {
		return nil, err
	}
	mv, err := evalCliffordExpr(args[0], m, env)
	if err != nil {
		return nil, err
	}
	return ast.NewStringNode(mv.String()), nil
}

// HandleCliffordWedge computes the exterior product A ^ B.
func HandleCliffordWedge(args []Node, env *Env) (Node, error) {
	if len(args) < 4 || len(args) > 5 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "clifford_wedge", 4, 5, len(args)))
	}
	m, err := parseMetricArgs(args, 2)
	if err != nil {
		return nil, err
	}
	mvA, err := evalCliffordExpr(args[0], m, env)
	if err != nil {
		return nil, err
	}
	mvB, err := evalCliffordExpr(args[1], m, env)
	if err != nil {
		return nil, err
	}
	res, err := mvA.Wedge(mvB)
	if err != nil {
		return nil, err
	}
	return ast.NewStringNode(res.String()), nil
}

// HandleCliffordContract computes contraction (left, right, inner).
func HandleCliffordContract(args []Node, env *Env) (Node, error) {
	if len(args) < 5 || len(args) > 6 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "clifford_contract", 5, 6, len(args)))
	}
	mode := "left"
	if str, ok := args[2].(*StringNode); ok {
		mode = strings.ToLower(str.Value)
	} else if v, ok := args[2].(*VarNode); ok {
		mode = strings.ToLower(v.Name)
	}

	m, err := parseMetricArgs(args, 3)
	if err != nil {
		return nil, err
	}
	mvA, err := evalCliffordExpr(args[0], m, env)
	if err != nil {
		return nil, err
	}
	mvB, err := evalCliffordExpr(args[1], m, env)
	if err != nil {
		return nil, err
	}

	var res *Multivector
	switch mode {
	case "left":
		res, err = mvA.LeftContract(mvB)
	case "right":
		res, err = mvA.RightContract(mvB)
	case "inner":
		res, err = mvA.InnerProduct(mvB)
	default:
		return nil, fmt.Errorf("%s", i18n.T("errors.clifford_contraction_mode", mode))
	}

	if err != nil {
		return nil, err
	}
	return ast.NewStringNode(res.String()), nil
}

// HandleCliffordDual computes the Poincaré dual of A.
func HandleCliffordDual(args []Node, env *Env) (Node, error) {
	if len(args) < 3 || len(args) > 4 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "clifford_dual", 3, 4, len(args)))
	}
	m, err := parseMetricArgs(args, 1)
	if err != nil {
		return nil, err
	}
	mvA, err := evalCliffordExpr(args[0], m, env)
	if err != nil {
		return nil, err
	}
	res := mvA.PoincareDual()
	return ast.NewStringNode(res.String()), nil
}

// parseGate extracts a QuantumGate from a node element.
func parseGate(elem Node) (QuantumGate, int, error) {
	var gateName string
	var target, control int
	control = -1
	maxQ := 0

	extractInt := func(n Node) (int, error) {
		if r, ok := n.(*RationalNode); ok && r.Val.IsInt() {
			val := int(r.Val.Num().Int64())
			if val >= 0 {
				return val, nil
			}
		}
		return -1, fmt.Errorf("%s", i18n.T("errors.quantum_expected_qubit_int", n.String()))
	}

	switch item := elem.(type) {
	case *ListNode:
		if len(item.Elements) < 2 {
			return QuantumGate{}, 0, fmt.Errorf("%s", i18n.T("errors.quantum_gate_list_min_len", item.String()))
		}
		if str, ok := item.Elements[0].(*StringNode); ok {
			gateName = strings.ToUpper(str.Value)
		} else if v, ok := item.Elements[0].(*VarNode); ok {
			gateName = strings.ToUpper(v.Name)
		} else {
			return QuantumGate{}, 0, fmt.Errorf("%s", i18n.T("errors.quantum_invalid_gate_name", item.Elements[0].String()))
		}

		var err error
		target, err = extractInt(item.Elements[1])
		if err != nil {
			return QuantumGate{}, 0, err
		}
		if target > maxQ {
			maxQ = target
		}

		if len(item.Elements) > 2 {
			control, err = extractInt(item.Elements[2])
			if err != nil {
				return QuantumGate{}, 0, err
			}
			if control > maxQ {
				maxQ = control
			}
		}

	case *FuncNode:
		gateName = strings.ToUpper(item.Name)
		if len(item.Args) < 1 {
			return QuantumGate{}, 0, fmt.Errorf("%s", i18n.T("errors.quantum_gate_list_min_len", item.String()))
		}
		var err error
		target, err = extractInt(item.Args[0])
		if err != nil {
			return QuantumGate{}, 0, err
		}
		if target > maxQ {
			maxQ = target
		}
		if len(item.Args) > 1 {
			control, err = extractInt(item.Args[1])
			if err != nil {
				return QuantumGate{}, 0, err
			}
			if control > maxQ {
				maxQ = control
			}
		}

	case *StringNode:
		fields := strings.Fields(item.Value)
		if len(fields) < 2 {
			return QuantumGate{}, 0, fmt.Errorf("%s", i18n.T("errors.quantum_circuit_invalid", item.Value))
		}
		gateName = strings.ToUpper(fields[0])
		tVal, err := strconv.Atoi(fields[1])
		if err != nil || tVal < 0 {
			return QuantumGate{}, 0, fmt.Errorf("%s", i18n.T("errors.quantum_expected_qubit_int", fields[1]))
		}
		target = tVal
		if target > maxQ {
			maxQ = target
		}
		if len(fields) > 2 {
			cVal, err := strconv.Atoi(fields[2])
			if err != nil || cVal < 0 {
				return QuantumGate{}, 0, fmt.Errorf("%s", i18n.T("errors.quantum_expected_qubit_int", fields[2]))
			}
			control = cVal
			if control > maxQ {
				maxQ = control
			}
		}

	default:
		return QuantumGate{}, 0, fmt.Errorf("%s", i18n.T("errors.quantum_circuit_invalid", elem.String()))
	}

	var gt GateType
	switch gateName {
	case "H":
		gt = GateH
	case "X":
		gt = GateX
	case "Y":
		gt = GateY
	case "Z":
		gt = GateZ
	case "S":
		gt = GateS
	case "T":
		gt = GateT
	case "CNOT", "CX":
		gt = GateCNOT
		if control == -1 {
			return QuantumGate{}, 0, fmt.Errorf("%s", i18n.T("errors.quantum_cnot_missing_control"))
		}
	default:
		return QuantumGate{}, 0, fmt.Errorf("%s", i18n.T("errors.quantum_gate_unknown", gateName))
	}

	return QuantumGate{
		Type:    gt,
		Target:  target,
		Control: control,
	}, maxQ, nil
}

// parseQuantumCircuit parses a quantum circuit specification from an AST Node.
func parseQuantumCircuit(n Node, explicitQubits int) (*QuantumCircuit, error) {
	var gateNodes []Node

	switch root := n.(type) {
	case *ListNode:
		gateNodes = root.Elements
	case *MatrixNode:
		for _, row := range root.Data {
			gateNodes = append(gateNodes, &ast.ListNode{Elements: row})
		}
	default:
		return nil, fmt.Errorf("%s", i18n.T("errors.quantum_circuit_invalid", n.String()))
	}

	var parsedGates []QuantumGate
	maxQ := -1

	for _, elem := range gateNodes {
		gate, qIndex, err := parseGate(elem)
		if err != nil {
			return nil, err
		}
		parsedGates = append(parsedGates, gate)
		if qIndex > maxQ {
			maxQ = qIndex
		}
	}

	numQubits := explicitQubits
	if numQubits <= 0 {
		numQubits = maxQ + 1
	}
	if numQubits < 1 {
		numQubits = 1
	}
	if numQubits > 6 {
		return nil, fmt.Errorf("%s", i18n.T("errors.quantum_qubits_limit", numQubits))
	}

	qc, err := NewQuantumCircuit(numQubits)
	if err != nil {
		return nil, err
	}

	for _, g := range parsedGates {
		if err := qc.AddGate(g); err != nil {
			return nil, err
		}
	}

	return qc, nil
}

// HandleQuantumEquiv checks exact Clifford+T circuit equivalence.
func HandleQuantumEquiv(args []Node, env *Env) (Node, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "quantum_equiv", 2, 3, len(args)))
	}

	strict := false
	if len(args) == 3 {
		if r, ok := args[2].(*RationalNode); ok {
			strict = r.Val.Sign() != 0
		} else if v, ok := args[2].(*VarNode); ok {
			strict = (v.Name == "true" || v.Name == "strict")
		}
	}

	qc1, err := parseQuantumCircuit(args[0], 0)
	if err != nil {
		return nil, err
	}
	qc2, err := parseQuantumCircuit(args[1], 0)
	if err != nil {
		return nil, err
	}

	// Match qubit counts to maximum of both circuits
	maxQubits := qc1.NumQubits
	if qc2.NumQubits > maxQubits {
		maxQubits = qc2.NumQubits
	}

	qc1Pad, _ := NewQuantumCircuit(maxQubits)
	for _, g := range qc1.Gates {
		_ = qc1Pad.AddGate(g)
	}

	qc2Pad, _ := NewQuantumCircuit(maxQubits)
	for _, g := range qc2.Gates {
		_ = qc2Pad.AddGate(g)
	}

	res, err := QuantumEquiv(qc1Pad, qc2Pad, strict)
	if err != nil {
		return nil, err
	}

	var equivInt int64
	if res.IsEquivalent {
		equivInt = 1
	}

	return ast.NewList([]Node{
		ast.NewRationalFromBigRat(big.NewRat(equivInt, 1)),
		ast.NewRationalFromBigRat(big.NewRat(int64(res.PhasePower), 1)),
		ast.NewRationalFromBigRat(big.NewRat(int64(res.CounterexampleState), 1)),
	}), nil
}

// HandleQuantumEval evaluates a quantum circuit into its exact dense unitary matrix.
func HandleQuantumEval(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "quantum_eval", 1, 2, len(args)))
	}

	explicitQubits := 0
	if len(args) == 2 {
		if r, ok := args[1].(*RationalNode); ok && r.Val.IsInt() {
			explicitQubits = int(r.Val.Num().Int64())
		}
	}

	qc, err := parseQuantumCircuit(args[0], explicitQubits)
	if err != nil {
		return nil, err
	}

	mat, err := qc.ToDenseUnitary()
	if err != nil {
		return nil, err
	}

	dim := 1 << qc.NumQubits
	data := make([][]Node, dim)
	for r := 0; r < dim; r++ {
		row := make([]Node, dim)
		for c := 0; c < dim; c++ {
			row[c] = mat[r][c].ToASTNode()
		}
		data[r] = row
	}

	return ast.NewMatrix(dim, dim, data)
}
