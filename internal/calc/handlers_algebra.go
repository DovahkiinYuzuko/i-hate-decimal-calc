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
					return nil, fmt.Errorf("variable list must contain only variable symbols, got %s", elem.String())
				}
			}
		case *VarNode:
			vars = []string{v.Name}
		default:
			return nil, fmt.Errorf("second argument to to_poly must be a variable or list of variables, got %s", args[1].String())
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
			return nil, fmt.Errorf("unsupported polynomial order %s, must be 'lex' or 'grevlex'", args[2].String())
		}
	}

	return NodeToPoly(expr, vars, order)
}


