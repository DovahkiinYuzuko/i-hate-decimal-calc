package calc

import (
	"fmt"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterHandler("laplace", handleLaplace)
	RegisterHandler("inv_laplace", handleInvLaplace)
	RegisterHandler("fourier_series", handleFourierSeries)
}

func handleLaplace(args []Node, env *Env) (Node, error) {
	var tName, sName string
	if len(args) >= 2 {
		if v, ok := args[1].(*VarNode); ok {
			tName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "laplace", "second", args[1].String()))
		}
	}
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			sName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "laplace", "third", args[2].String()))
		}
	}
	return EvalLaplace(args[0], tName, sName, env)
}

func handleInvLaplace(args []Node, env *Env) (Node, error) {
	var sName, tName string
	if len(args) >= 2 {
		if v, ok := args[1].(*VarNode); ok {
			sName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "inv_laplace", "second", args[1].String()))
		}
	}
	if len(args) >= 3 {
		if v, ok := args[2].(*VarNode); ok {
			tName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "inv_laplace", "third", args[2].String()))
		}
	}
	return EvalInvLaplace(args[0], sName, tName, env)
}

func handleFourierSeries(args []Node, env *Env) (Node, error) {
	return EvalFourierSeries(args, env)
}
