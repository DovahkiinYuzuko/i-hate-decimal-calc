package calc

import (
	"fmt"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterHandler("poly_gcd", handlePolyGCD)
	RegisterHandler("poly_lcm", handlePolyLCM)
	RegisterHandler("resultant", handleResultant)
	RegisterHandler("groebner", handleGroebner)
	RegisterHandler("sturm", handleSturm)
	RegisterHandler("root_count", handleRootCount)
	RegisterHandler("isolate_roots", handleIsolateRoots)
	RegisterHandler("to_poly", handleToPoly)
	RegisterHandler("to_alg", handleToAlg)
	RegisterHandler("alg_inv", handleAlgInv)
	RegisterHandler("min_poly", handleMinPoly)
	RegisterHandler("cad", handleCAD)
}

func handlePolyGCD(args []Node, env *Env) (Node, error) {
	var vName string
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			vName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "poly_gcd", "third", args[2].String()))
		}
	}
	return EvalPolyGCD(args[0], args[1], vName, env)
}

func handlePolyLCM(args []Node, env *Env) (Node, error) {
	var vName string
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			vName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "poly_lcm", "third", args[2].String()))
		}
	}
	return EvalPolyLCM(args[0], args[1], vName, env)
}

func handleResultant(args []Node, env *Env) (Node, error) {
	var vName string
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			vName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "resultant", "third", args[2].String()))
		}
	}
	return EvalResultant(args[0], args[1], vName, env)
}

func handleGroebner(args []Node, env *Env) (Node, error) {
	var orderOpt Node
	if len(args) >= 3 {
		orderOpt = args[2]
	}
	return EvalGroebner(args[0], args[1], orderOpt, env)
}

func handleSturm(args []Node, env *Env) (Node, error) {
	switch len(args) {
	case 1:
		return EvalSturm(args[0], "", env)
	case 2:
		if v, ok := args[1].(*VarNode); ok {
			return EvalSturm(args[0], v.Name, env)
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_invalid_variable", "sturm", args[1].String()))
	default:
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "sturm", 1, 2, len(args)))
	}
}

func handleRootCount(args []Node, env *Env) (Node, error) {
	switch len(args) {
	case 1:
		return EvalRootCount(args[0], "", nil, nil, env)
	case 2:
		if v, ok := args[1].(*VarNode); ok {
			return EvalRootCount(args[0], v.Name, nil, nil, env)
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_invalid_variable", "root_count", args[1].String()))
	case 3:
		return EvalRootCount(args[0], "", args[1], args[2], env)
	case 4:
		if v, ok := args[1].(*VarNode); ok {
			return EvalRootCount(args[0], v.Name, args[2], args[3], env)
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_invalid_variable", "root_count", args[1].String()))
	default:
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "root_count", 1, 4, len(args)))
	}
}

func handleIsolateRoots(args []Node, env *Env) (Node, error) {
	switch len(args) {
	case 1:
		return EvalIsolateRoots(args[0], "", nil, nil, env)
	case 2:
		if v, ok := args[1].(*VarNode); ok {
			return EvalIsolateRoots(args[0], v.Name, nil, nil, env)
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_invalid_variable", "isolate_roots", args[1].String()))
	case 3:
		return EvalIsolateRoots(args[0], "", args[1], args[2], env)
	case 4:
		if v, ok := args[1].(*VarNode); ok {
			return EvalIsolateRoots(args[0], v.Name, args[2], args[3], env)
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.sturm_invalid_variable", "isolate_roots", args[1].String()))
	default:
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "isolate_roots", 1, 4, len(args)))
	}
}

func handleToPoly(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "to_poly", 1, 3, len(args)))
	}

	expr := args[0]
	var vars []string

	if len(args) >= 2 {
		switch v := args[1].(type) {
		case *ListNode:
			for _, elem := range v.Elements {
				if vn, ok := elem.(*VarNode); ok {
					vars = append(vars, vn.Name)
				} else {
					return nil, fmt.Errorf("%s", i18n.T("handlers.err_variable_list_must_contain_only", elem.String()))
				}
			}
		case *VarNode:
			vars = []string{v.Name}
		default:
			return nil, fmt.Errorf("%s", i18n.T("handlers.err_second_argument_to_to_poly", args[1].String()))
		}
	} else {
		vars = ast.ExtractFreeVariables(expr)
		if len(vars) == 0 {
			vars = []string{"x"}
		}
	}

	order := ast.OrderLex
	if len(args) >= 3 {
		orderStr := strings.ToLower(args[2].String())
		orderStr = strings.Trim(orderStr, "\"'`")
		switch orderStr {
		case "lex", "lexicographic":
			order = ast.OrderLex
		case "grevlex", "degrevlex", "gradedreverselexicographic":
			order = ast.OrderGrevLex
		default:
			return nil, fmt.Errorf("%s", i18n.T("handlers.err_unsupported_polynomial_order_must_be", args[2].String()))
		}
	}

	return NodeToPoly(expr, vars, order)
}

func handleToAlg(args []Node, env *Env) (Node, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "to_alg", 2, 3, len(args)))
	}
	repExpr := args[0]
	minPolyExpr := args[1]

	// Extract variable from minPolyExpr
	vars := ast.ExtractFreeVariables(minPolyExpr)
	if len(vars) == 0 {
		vars = []string{"x"}
	}
	mainVar := vars[0]

	minPoly, err := NodeToPoly(minPolyExpr, []string{mainVar}, ast.OrderLex)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("handlers.err_failed_to_parse_minimal_polynomial", err))
	}

	repPoly, err := NodeToPoly(repExpr, []string{mainVar}, ast.OrderLex)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("handlers.err_failed_to_parse_representative_polynomial", err))
	}

	symbol := mainVar
	if len(args) >= 3 {
		if vn, ok := args[2].(*VarNode); ok {
			symbol = vn.Name
		} else {
			symbol = strings.Trim(args[2].String(), "\"'`")
		}
	}

	return NewAlgebraicNumber(minPoly, repPoly, symbol)
}

func handleAlgInv(args []Node, env *Env) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("errors.invalid_args_count", "alg_inv", 1, len(args)))
	}
	algNode, ok := args[0].(*AlgebraicNumberNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("handlers.err_argument_to_alg_inv_must", args[0]))
	}
	return InvAlg(algNode)
}

func handleMinPoly(args []Node, env *Env) (Node, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("%s", i18n.T("errors.func_args_between", "min_poly", 2, 3, len(args)))
	}

	vars1 := ast.ExtractFreeVariables(args[0])
	if len(vars1) == 0 {
		vars1 = []string{"x"}
	}
	m1, err := NodeToPoly(args[0], []string{vars1[0]}, ast.OrderLex)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("handlers.err_failed_to_parse_first_polynomial", err))
	}

	vars2 := ast.ExtractFreeVariables(args[1])
	if len(vars2) == 0 {
		vars2 = []string{"x"}
	}
	m2, err := NodeToPoly(args[1], []string{vars2[0]}, ast.OrderLex)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("handlers.err_failed_to_parse_second_polynomial", err))
	}

	varName := "y"
	if len(args) >= 3 {
		if vn, ok := args[2].(*VarNode); ok {
			varName = vn.Name
		} else {
			varName = strings.Trim(args[2].String(), "\"'`")
		}
	}

	return MinPolySum(m1, m2, varName)
}
func handleCAD(args []Node, env *Env) (Node, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("%s", i18n.T("cad.err_poly_required"))
	}
	var polys []Node
	if list, ok := args[0].(*ListNode); ok {
		polys = list.Elements
	} else {
		polys = []Node{args[0]}
	}

	var vars []string
	if len(args) >= 2 {
		if vList, ok := args[1].(*ListNode); ok {
			for _, elem := range vList.Elements {
				if vn, ok := elem.(*VarNode); ok {
					vars = append(vars, vn.Name)
				}
			}
		} else if vn, ok := args[1].(*VarNode); ok {
			vars = []string{vn.Name}
		}
	}
	if len(vars) == 0 {
		for _, p := range polys {
			for _, v := range ExtractFreeVariables(p) {
				found := false
				for _, ev := range vars {
					if ev == v {
						found = true
						break
					}
				}
				if !found {
					vars = append(vars, v)
				}
			}
		}
	}
	return CADDecompose(polys, vars, env)
}
