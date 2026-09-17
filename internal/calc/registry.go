package calc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// FuncHandler defines the evaluation logic for a function with environment context.
type FuncHandler func(args []Node, env *Env) (Node, error)

// FunctionSpec defines the specification, validation rules, and evaluation hook for a built-in function.
type FunctionSpec struct {
	Name             string
	MinArgs          int
	MaxArgs          int
	AllowBareSymbol  bool // If true, can appear without parens as a distribution symbol (e.g. expect(binom, ...))
	LazyArgs         bool // If true, arguments are not evaluated eagerly before calling Handler (e.g. dsolve)
	Validate         func(args []Node) error
	Evaluate         func(args []Node) (Node, error)
	Handler          FuncHandler
}

var builtInFunctions = map[string]FunctionSpec{}

// RegisterFunction registers a built-in function specification.
func RegisterFunction(spec FunctionSpec) {
	if existing, ok := builtInFunctions[spec.Name]; ok {
		if spec.Handler == nil && existing.Handler != nil {
			spec.Handler = existing.Handler
		}
		if !spec.LazyArgs && existing.LazyArgs {
			spec.LazyArgs = existing.LazyArgs
		}
	}
	builtInFunctions[spec.Name] = spec
}

// RegisterHandler attaches an evaluation handler to an existing function or creates a minimal spec.
func RegisterHandler(name string, handler FuncHandler) {
	if spec, ok := builtInFunctions[name]; ok {
		spec.Handler = handler
		builtInFunctions[name] = spec
	} else {
		builtInFunctions[name] = FunctionSpec{
			Name:    name,
			Handler: handler,
		}
	}
}

// RegisterLazyHandler attaches an evaluation handler whose arguments are not eagerly evaluated.
func RegisterLazyHandler(name string, handler FuncHandler) {
	if spec, ok := builtInFunctions[name]; ok {
		spec.Handler = handler
		spec.LazyArgs = true
		builtInFunctions[name] = spec
	} else {
		builtInFunctions[name] = FunctionSpec{
			Name:     name,
			LazyArgs: true,
			Handler:  handler,
		}
	}
}

// LookupFunction retrieves a FunctionSpec by function name, supporting case-insensitive fallback.
func LookupFunction(name string) (FunctionSpec, bool) {
	if spec, ok := builtInFunctions[name]; ok {
		return spec, true
	}
	spec, ok := builtInFunctions[strings.ToLower(name)]
	return spec, ok
}

// IsReservedFunc reports whether the given identifier name is a reserved built-in function name, case-insensitively.
func IsReservedFunc(name string) bool {
	if _, ok := builtInFunctions[name]; ok {
		return true
	}
	_, ok := builtInFunctions[strings.ToLower(name)]
	return ok
}

// IsBareSymbolAllowed reports whether the function name can appear as an isolated symbol, case-insensitively.
func IsBareSymbolAllowed(name string) bool {
	if spec, ok := builtInFunctions[name]; ok {
		return spec.AllowBareSymbol
	}
	spec, ok := builtInFunctions[strings.ToLower(name)]
	return ok && spec.AllowBareSymbol
}

// ValidateFuncArgs checks argument count and specific domain constraints for a function call.
func ValidateFuncArgs(name string, args []Node) error {
	spec, ok := LookupFunction(name)
	if !ok {
		return fmt.Errorf("%s", i18n.T("errors.unknown_function", name))
	}

	argCount := len(args)
	if spec.MinArgs == spec.MaxArgs {
		if argCount != spec.MinArgs {
			return fmt.Errorf("%s", i18n.T("errors.invalid_args_count", name, spec.MinArgs, argCount))
		}
	} else {
		if argCount < spec.MinArgs || (spec.MaxArgs != -1 && argCount > spec.MaxArgs) {
			if spec.MaxArgs == -1 {
				return fmt.Errorf("%s", i18n.T("errors.func_args_at_least", name, spec.MinArgs, argCount))
			}
			return fmt.Errorf("%s", i18n.T("errors.func_args_between", name, spec.MinArgs, spec.MaxArgs, argCount))
		}
	}

	if spec.Validate != nil {
		if err := spec.Validate(args); err != nil {
			return err
		}
	}

	return nil
}

// EvaluateFunction invokes the registered evaluation hook for the named function.
func EvaluateFunction(name string, args []Node) (Node, error) {
	spec, ok := LookupFunction(name)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.unknown_function", name))
	}
	if spec.Evaluate == nil {
		return nil, fmt.Errorf("function %s has no evaluation handler registered", name)
	}
	return spec.Evaluate(args)
}

func init() {
	// sqrt (creates SqrtNode in parser, but reserved as function)
	RegisterFunction(FunctionSpec{
		Name:    "sqrt",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Exponential & Logarithmic
	RegisterFunction(FunctionSpec{
		Name:    "exp",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Trigonometric & Hyperbolic
	RegisterFunction(FunctionSpec{
		Name:    "sin",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "cos",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "tan",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "asin",
		MinArgs: 1,
		MaxArgs: 1,
		Validate: func(args []Node) error {
			if rat, ok := args[0].(*RationalNode); ok {
				one := big.NewRat(1, 1)
				negOne := big.NewRat(-1, 1)
				if rat.Val.Cmp(one) > 0 || rat.Val.Cmp(negOne) < 0 {
					return NewDomainError("errors.domain_error", "asin domain error: argument must be in [-1, 1], got %s", rat.String())
				}
			}
			return nil
		},
	})
	RegisterFunction(FunctionSpec{
		Name:    "acos",
		MinArgs: 1,
		MaxArgs: 1,
		Validate: func(args []Node) error {
			if rat, ok := args[0].(*RationalNode); ok {
				one := big.NewRat(1, 1)
				negOne := big.NewRat(-1, 1)
				if rat.Val.Cmp(one) > 0 || rat.Val.Cmp(negOne) < 0 {
					return NewDomainError("errors.domain_error", "acos domain error: argument must be in [-1, 1], got %s", rat.String())
				}
			}
			return nil
		},
	})
	RegisterFunction(FunctionSpec{
		Name:    "atan",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Complex / Polar Form
	RegisterFunction(FunctionSpec{
		Name:    "arg",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "polar",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "polar_exp",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "rect",
		MinArgs: 2,
		MaxArgs: 2,
	})

	// Logarithmic
	RegisterFunction(FunctionSpec{
		Name:    "ln",
		MinArgs: 1,
		MaxArgs: 1,
		Validate: func(args []Node) error {
			if rat, ok := args[0].(*RationalNode); ok && rat.Val.Sign() <= 0 {
				return NewDomainError("errors.domain_ln_arg", "ln domain error: argument must be positive, got %s", rat.String())
			}
			return nil
		},
	})
	RegisterFunction(FunctionSpec{
		Name:    "log",
		MinArgs: 1,
		MaxArgs: 2,
		Validate: func(args []Node) error {
			if len(args) == 1 {
				if rat, ok := args[0].(*RationalNode); ok && rat.Val.Sign() <= 0 {
					return NewDomainError("errors.domain_log_arg", "log domain error: argument must be positive, got %s", rat.String())
				}
			} else if len(args) == 2 {
				base := args[0]
				arg := args[1]
				if ratBase, ok := base.(*RationalNode); ok {
					if ratBase.Val.Sign() <= 0 {
						return NewDomainError("errors.domain_log_base", "log base error: base must be positive, got %s", ratBase.String())
					}
					one := big.NewRat(1, 1)
					if ratBase.Val.Cmp(one) == 0 {
						return NewDomainError("errors.domain_log_base_one", "log base error: base cannot be 1")
					}
				}
				if ratArg, ok := arg.(*RationalNode); ok && ratArg.Val.Sign() <= 0 {
					return NewDomainError("errors.domain_log_arg", "log domain error: argument must be positive, got %s", ratArg.String())
				}
			}
			return nil
		},
	})

	// Arithmetic / Special
	RegisterFunction(FunctionSpec{
		Name:    "abs",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "cbrt",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "gcd",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "lcm",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "mod",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "perm",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "comb",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "rand",
		MinArgs: 1,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "cfrac",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "from_cfrac",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// CAS / Calculus
	RegisterFunction(FunctionSpec{
		Name:    "factor",
		MinArgs: 1,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "expand",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "trig_expand",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "trig_reduce",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "integrate",
		MinArgs: 2,
		MaxArgs: 4,
		Validate: func(args []Node) error {
			if len(args) != 2 && len(args) != 4 {
				return fmt.Errorf("integrate requires 2 arguments (indefinite) or 4 arguments (definite), got %d", len(args))
			}
			if _, ok := args[1].(*VarNode); !ok {
				return fmt.Errorf("integrate error: second argument must be a variable name, got %s", args[1].String())
			}
			return nil
		},
	})
	RegisterFunction(FunctionSpec{
		Name:    "diff",
		MinArgs: 2,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "solve",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "taylor",
		MinArgs: 4,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "sum",
		MinArgs: 4,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "limit",
		MinArgs: 3,
		MaxArgs: 4,
		Validate: func(args []Node) error {
			if len(args) != 3 && len(args) != 4 {
				return fmt.Errorf("limit requires 3 arguments (expr, var, target) or 4 arguments (expr, var, target, direction), got %d", len(args))
			}
			if _, ok := args[1].(*VarNode); !ok {
				return fmt.Errorf("limit error: second argument must be a variable name, got %s", args[1].String())
			}
			return nil
		},
	})

	// Linear Algebra
	RegisterFunction(FunctionSpec{
		Name:    "det",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "inv",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "transpose",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "rref",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "rank",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "solve_linear",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "linsolve",
		MinArgs: 2,
		MaxArgs: 2,
	})

	// 3D Vector Calculus
	RegisterFunction(FunctionSpec{
		Name:    "dot",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "cross",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "norm",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "grad",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "div",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "curl",
		MinArgs: 2,
		MaxArgs: 2,
	})

	// 2D Analytic Geometry
	RegisterFunction(FunctionSpec{
		Name:    "line_intersect",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "circle_intersect",
		MinArgs: 4,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "triangle_area",
		MinArgs: 3,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "triangle_centers",
		MinArgs: 3,
		MaxArgs: 3,
	})

	// Probability & Statistics
	RegisterFunction(FunctionSpec{
		Name:            "binom",
		MinArgs:         3,
		MaxArgs:         3,
		AllowBareSymbol: true,
	})
	RegisterFunction(FunctionSpec{
		Name:            "hyper",
		MinArgs:         4,
		MaxArgs:         4,
		AllowBareSymbol: true,
	})
	RegisterFunction(FunctionSpec{
		Name:            "geom",
		MinArgs:         2,
		MaxArgs:         2,
		AllowBareSymbol: true,
	})
	RegisterFunction(FunctionSpec{
		Name:    "bayes",
		MinArgs: 3,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "expect",
		MinArgs: 1,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "variance",
		MinArgs: 1,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "stddev",
		MinArgs: 1,
		MaxArgs: 4,
	})

	// Terminal Plotter
	RegisterFunction(FunctionSpec{
		Name:    "plot",
		MinArgs: 2,
		MaxArgs: 3,
	})

	// Assumptions
	RegisterFunction(FunctionSpec{
		Name:    "assume",
		MinArgs: 1,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "unassume",
		MinArgs: 0,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "assumptions",
		MinArgs: 0,
		MaxArgs: 0,
	})
	RegisterFunction(FunctionSpec{
		Name:    "clear_assumptions",
		MinArgs: 0,
		MaxArgs: 0,
	})

	// Number Theory & Modular Arithmetic Pack
	RegisterFunction(FunctionSpec{
		Name:    "inv_mod",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "crt",
		MinArgs: 2,
		MaxArgs: -1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "totient",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "is_prime",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Rational Functions Pack
	RegisterFunction(FunctionSpec{
		Name:    "apart",
		MinArgs: 1,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "together",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Differential Equations Pack
	RegisterFunction(FunctionSpec{
		Name:    "dsolve",
		MinArgs: 1,
		MaxArgs: 3,
	})

	// Laplace & Operational Transforms Pack
	RegisterFunction(FunctionSpec{
		Name:    "laplace",
		MinArgs: 1,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "inv_laplace",
		MinArgs: 1,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "delta",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Eigenvalues, Eigenvectors & Trace Pack (issue-49)
	RegisterFunction(FunctionSpec{
		Name:    "eigenvals",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "eigenvects",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "trace",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "tr",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Multivariate Polynomial GCD & LCM Pack (issue-50)
	RegisterFunction(FunctionSpec{
		Name:    "poly_gcd",
		MinArgs: 2,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "poly_lcm",
		MinArgs: 2,
		MaxArgs: 3,
	})

	// Sylvester Resultant & Elimination Pack (issue-51)
	RegisterFunction(FunctionSpec{
		Name:    "resultant",
		MinArgs: 2,
		MaxArgs: 3,
	})

	// Symbolic Fourier Series Expansion Pack (issue-52)
	RegisterFunction(FunctionSpec{
		Name:    "fourier_series",
		MinArgs: 1,
		MaxArgs: 4,
	})

	// Gröbner Bases & Polynomial Ideals Pack (issue-55)
	RegisterFunction(FunctionSpec{
		Name:    "groebner",
		MinArgs: 2,
		MaxArgs: 3,
	})

	// Exact Matrix Decompositions Pack (issue-59)
	RegisterFunction(FunctionSpec{
		Name:    "lu",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "qr",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "cholesky",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ldlt",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "pinv",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Exact Real Root Isolation & Sturm's Theorem Pack (issue-61)
	RegisterFunction(FunctionSpec{
		Name:    "sturm",
		MinArgs: 1,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "root_count",
		MinArgs: 1,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "isolate_roots",
		MinArgs: 1,
		MaxArgs: 4,
	})

	// Complex Analysis & Symbolic Residue Calculus Pack (issue-60)
	RegisterFunction(FunctionSpec{
		Name:    "residue",
		MinArgs: 3,
		MaxArgs: 3,
	})

	// Elementary Special Functions & Combinatorics Pack (issue-62)
	RegisterFunction(FunctionSpec{
		Name:    "gamma",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "beta",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "bernoulli",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "zeta",
		MinArgs: 1,
		MaxArgs: 1,
	})
}


