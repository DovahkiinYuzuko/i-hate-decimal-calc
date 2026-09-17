package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterHandler("assume", handleAssume)
	RegisterHandler("unassume", handleUnassume)
	RegisterHandler("assumptions", handleAssumptions)
	RegisterHandler("clear_assumptions", handleClearAssumptions)
}

func handleAssume(args []Node, env *Env) (Node, error) {
	if env == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.assume_no_env"))
	}
	if len(args) == 1 {
		// e.g. assume(x > 0), assume(x >= 0), assume(x < 0), assume(x <= 0)
		if rel, ok := args[0].(*RelOpNode); ok {
			v, okVar := rel.LHS.(*VarNode)
			r, okRat := rel.RHS.(*RationalNode)
			if okVar && okRat && r.Val.Sign() == 0 {
				var prop Property
				switch rel.Op {
				case ">":
					prop = PropPositive
				case ">=":
					prop = PropNonNegative
				case "<":
					prop = PropNegative
				case "<=":
					prop = PropNonPositive
				default:
					return nil, fmt.Errorf("%s", i18n.T("errors.assume_unsupported", rel.Op))
				}
				if err := env.Assumptions().Assume(v.Name, prop); err != nil {
					return nil, err
				}
				return &VarNode{Name: "ok"}, nil
			}
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.assume_invalid", args[0].String()))
	}
	if len(args) == 2 {
		// e.g. assume(x, "positive"), assume(n, integer)
		v, okVar := args[0].(*VarNode)
		if !okVar {
			return nil, fmt.Errorf("%s", i18n.T("errors.assume_invalid", args[0].String()))
		}
		propName := ""
		if vProp, okVProp := args[1].(*VarNode); okVProp {
			propName = vProp.Name
		} else {
			propName = args[1].String()
		}
		prop, err := ParseProperty(propName)
		if err != nil {
			return nil, err
		}
		if err := env.Assumptions().Assume(v.Name, prop); err != nil {
			return nil, err
		}
		return &VarNode{Name: "ok"}, nil
	}
	return nil, fmt.Errorf("%s", i18n.T("errors.assume_args_count"))
}

func handleUnassume(args []Node, env *Env) (Node, error) {
	if env == nil {
		return &VarNode{Name: "ok"}, nil
	}
	if len(args) == 0 {
		env.Assumptions().ClearAll()
		return &VarNode{Name: "ok"}, nil
	}
	if v, ok := args[0].(*VarNode); ok {
		env.Assumptions().Unassume(v.Name)
		return &VarNode{Name: "ok"}, nil
	}
	return nil, fmt.Errorf("%s", i18n.T("errors.unassume_expect_var"))
}

func handleAssumptions(args []Node, env *Env) (Node, error) {
	if env == nil {
		return NewList(nil), nil
	}
	list := env.Assumptions().ListAssumptions()
	var elems []Node
	for _, s := range list {
		elems = append(elems, &VarNode{Name: s})
	}
	return NewList(elems), nil
}

func handleClearAssumptions(args []Node, env *Env) (Node, error) {
	if env != nil {
		env.Assumptions().ClearAll()
	}
	return &VarNode{Name: "ok"}, nil
}
