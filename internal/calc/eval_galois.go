package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/galois"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterFunction(FunctionSpec{
		Name:    "galois_group",
		MinArgs: 1,
		MaxArgs: 2,
		Handler: EvalGaloisGroup,
	})
	RegisterFunction(FunctionSpec{
		Name:    "is_solvable_by_radicals",
		MinArgs: 1,
		MaxArgs: 2,
		Handler: EvalIsSolvableByRadicals,
	})
}

// EvalGaloisGroup evaluates galois_group(poly [, x]) and returns the identified Galois group.
func EvalGaloisGroup(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s", i18n.T("galois.err_galois_args", "galois_group"))
	}
	varName := "x"
	if len(args) == 2 {
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "galois_group", "second", args[1].String()))
		}
	} else {
		vars := extractVariables(args[0])
		if len(vars) > 0 {
			varName = vars[0]
		}
	}

	res, err := computeGaloisResult(args[0], varName)
	if err != nil {
		return nil, err
	}

	if !res.IsSolvable && env != nil {
		certIR := NewImpossibilityCertificate(
			ImpossibilityAbelRuffini,
			args[0],
			&VarNode{Name: res.Group.Name},
			true,
			fmt.Sprintf(i18n.T("galois.proof_abel_ruffini"), res.Group.Name),
		)
		env.LastCert = &VerificationCertificate{
			Domain:     DomainImpossibility,
			Equation:   fmt.Sprintf("AbelRuffini(%s)", args[0].String()),
			IsVerified: true,
			Details:    certIR.DetailsMsg,
			CertIR:     certIR,
		}
	}

	return &VarNode{Name: res.Group.Name}, nil
}

// EvalIsSolvableByRadicals evaluates is_solvable_by_radicals(poly [, x]) and returns true/false.
func EvalIsSolvableByRadicals(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("%s", i18n.T("galois.err_galois_args", "is_solvable_by_radicals"))
	}
	varName := "x"
	if len(args) == 2 {
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("%s", i18n.T("errors.arg_must_be_var", "is_solvable_by_radicals", "second", args[1].String()))
		}
	} else {
		vars := extractVariables(args[0])
		if len(vars) > 0 {
			varName = vars[0]
		}
	}

	res, err := computeGaloisResult(args[0], varName)
	if err != nil {
		return nil, err
	}

	if !res.IsSolvable && env != nil {
		certIR := NewImpossibilityCertificate(
			ImpossibilityAbelRuffini,
			args[0],
			&VarNode{Name: res.Group.Name},
			true,
			fmt.Sprintf(i18n.T("galois.proof_abel_ruffini"), res.Group.Name),
		)
		env.LastCert = &VerificationCertificate{
			Domain:     DomainImpossibility,
			Equation:   fmt.Sprintf("AbelRuffini(%s)", args[0].String()),
			IsVerified: true,
			Details:    certIR.DetailsMsg,
			CertIR:     certIR,
		}
	}

	if res.IsSolvable {
		return &VarNode{Name: "true"}, nil
	}
	return &VarNode{Name: "false"}, nil
}

// computeGaloisResult converts an arbitrary AST polynomial into a monic integer polynomial and computes its Galois group.
func computeGaloisResult(node Node, varName string) (*galois.GaloisIdentificationResult, error) {
	polyNode, err := NodeToPoly(node, []string{varName}, ast.OrderLex)
	if err != nil {
		return nil, err
	}
	if len(polyNode.Terms) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("galois.err_degree_out_of_range", 0))
	}

	deg := polyNode.Terms[0].Exponents[0]
	if deg < 1 || deg > 5 {
		return nil, fmt.Errorf("%s", i18n.T("galois.err_degree_out_of_range", deg))
	}

	// Extract rational coefficients for degrees deg down to 0
	ratCoeffs := make([]*big.Rat, deg+1)
	for i := range ratCoeffs {
		ratCoeffs[i] = new(big.Rat)
	}
	for _, term := range polyNode.Terms {
		p := term.Exponents[0]
		ratCoeffs[deg-p].Set(term.Coeff)
	}

	lead := ratCoeffs[0]
	if lead.Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("galois.err_degree_out_of_range", 0))
	}

	// Make monic over Q: c'_i = c_i / lead
	monicRats := make([]*big.Rat, deg+1)
	for i, c := range ratCoeffs {
		monicRats[i] = new(big.Rat).Quo(c, lead)
	}

	// Find least common multiple L of all denominators of monicRats
	lcmDenom := big.NewInt(1)
	for _, r := range monicRats {
		d := r.Denom()
		gcd := new(big.Int).GCD(nil, nil, lcmDenom, d)
		lcmDenom.Mul(lcmDenom, new(big.Int).Div(d, gcd))
	}

	// Scale by transformation y = L*x:
	// P(x) = 0 <=> Q(y) = y^deg + sum_{k=1}^deg (c'_k * L^k) y^{deg-k} = 0
	// The coefficients of Q(y) are guaranteed to be integers, and Q(y) is monic.
	intCoeffs := make([]*big.Int, deg+1)
	intCoeffs[0] = big.NewInt(1)

	lPow := big.NewInt(1)
	for k := 1; k <= deg; k++ {
		lPow = new(big.Int).Mul(lPow, lcmDenom)
		// monicRats[k] * L^k
		prod := new(big.Rat).Mul(monicRats[k], new(big.Rat).SetInt(lPow))
		if !prod.IsInt() {
			// Mathematically unreachable because L is multiple of all denominators
			return nil, fmt.Errorf("%s", i18n.T("galois.err_resolvent_failed", "non-integer scaled coefficient"))
		}
		intCoeffs[k] = new(big.Int).Set(prod.Num())
	}

	return galois.IdentifyGaloisGroupIntegerPoly(intCoeffs)
}

func extractVariables(node Node) []string {
	var vars []string
	seen := make(map[string]bool)

	var walk func(n Node)
	walk = func(n Node) {
		if n == nil {
			return
		}
		if v, ok := n.(*VarNode); ok {
			if v.Name != "true" && v.Name != "false" && v.Name != "pi" && v.Name != "e" && v.Name != "i" {
				if !seen[v.Name] {
					seen[v.Name] = true
					vars = append(vars, v.Name)
				}
			}
			return
		}
		switch val := n.(type) {
		case *AddNode:
			for _, term := range val.Terms {
				walk(term)
			}
		case *MulNode:
			for _, factor := range val.Factors {
				walk(factor)
			}
		case *PowNode:
			walk(val.Base)
			walk(val.Exp)
		case *UnaryOpNode:
			walk(val.Expr)
		case *FuncNode:
			for _, arg := range val.Args {
				walk(arg)
			}
		}
	}

	walk(node)
	return vars
}
