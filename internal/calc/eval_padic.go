package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/padic"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)


func init() {
	RegisterHandler("padic_val", func(args []Node, env *Env) (Node, error) {
		return EvalPadicVal(args)
	})
	RegisterHandler("padic_norm", func(args []Node, env *Env) (Node, error) {
		return EvalPadicNorm(args)
	})
	RegisterHandler("padic_expand", func(args []Node, env *Env) (Node, error) {
		return EvalPadicExpand(args)
	})
}

// EvalPadicVal computes the p-adic valuation v_p(x) of a rational number x.
func EvalPadicVal(args []Node) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_padic_val_args"))
	}

	xRat, okX := args[0].(*RationalNode)
	if !okX {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_rational", "padic_val", args[0].String()))
	}

	pRat, okP := args[1].(*RationalNode)
	if !okP || !pRat.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_integer", "padic_val", args[1].String()))
	}

	p := pRat.Val.Num()
	if xRat.Val.Sign() == 0 {
		if p.Cmp(big.NewInt(2)) < 0 {
			return nil, fmt.Errorf("%s", i18n.T("padic.err_prime_less_than_2"))
		}
		return &VarNode{Name: "infinity"}, nil
	}

	v, err := padic.Valuation(xRat.Val, p)
	if err != nil {
		return nil, err
	}

	return mustRational(int64(v), 1), nil
}

// EvalPadicNorm computes the p-adic norm |x|_p of a rational number x.
func EvalPadicNorm(args []Node) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_padic_norm_args"))
	}

	xRat, okX := args[0].(*RationalNode)
	if !okX {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_rational", "padic_norm", args[0].String()))
	}

	pRat, okP := args[1].(*RationalNode)
	if !okP || !pRat.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_integer", "padic_norm", args[1].String()))
	}

	p := pRat.Val.Num()
	norm, err := padic.Norm(xRat.Val, p)
	if err != nil {
		return nil, err
	}

	return &RationalNode{Val: norm}, nil
}

// EvalPadicExpand computes the truncated p-adic expansion of a rational number x.
func EvalPadicExpand(args []Node) (Node, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_padic_expand_args"))
	}

	xRat, okX := args[0].(*RationalNode)
	if !okX {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_rational", "padic_expand", args[0].String()))
	}

	pRat, okP := args[1].(*RationalNode)
	if !okP || !pRat.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_integer", "padic_expand", args[1].String()))
	}

	precRat, okPrec := args[2].(*RationalNode)
	if !okPrec || !precRat.Val.IsInt() {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_arg_must_be_integer", "padic_expand", args[2].String()))
	}

	prec := int(precRat.Val.Num().Int64())
	if prec <= 0 {
		return nil, fmt.Errorf("%s", i18n.T("padic.err_prec_positive"))
	}

	p := pRat.Val.Num()
	padicNum, err := padic.NewPadicFromRat(xRat.Val, p, prec)
	if err != nil {
		return nil, err
	}

	str := padic.ExpansionString(padicNum)
	return NewVar(str), nil
}
