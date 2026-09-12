package calc

import (
	"fmt"
	"math/big"
)

// FunctionSpec defines the specification, validation rules, and evaluation hook for a built-in function.
type FunctionSpec struct {
	Name             string
	MinArgs          int
	MaxArgs          int
	AllowBareSymbol  bool // If true, can appear without parens as a distribution symbol (e.g. expect(binom, ...))
	Validate         func(args []Node) error
	Evaluate         func(args []Node) (Node, error)
}

var builtInFunctions = map[string]FunctionSpec{}

// RegisterFunction registers a built-in function specification.
func RegisterFunction(spec FunctionSpec) {
	builtInFunctions[spec.Name] = spec
}

// LookupFunction retrieves a FunctionSpec by function name.
func LookupFunction(name string) (FunctionSpec, bool) {
	spec, ok := builtInFunctions[name]
	return spec, ok
}

// IsReservedFunc reports whether the given identifier name is a reserved built-in function name.
func IsReservedFunc(name string) bool {
	_, ok := builtInFunctions[name]
	return ok
}

// IsBareSymbolAllowed reports whether the function name can appear as an isolated symbol.
func IsBareSymbolAllowed(name string) bool {
	spec, ok := builtInFunctions[name]
	return ok && spec.AllowBareSymbol
}

// ValidateFuncArgs checks argument count and specific domain constraints for a function call.
func ValidateFuncArgs(name string, args []Node) error {
	spec, ok := builtInFunctions[name]
	if !ok {
		return fmt.Errorf("unknown function: %s", name)
	}

	argCount := len(args)
	if spec.MinArgs == spec.MaxArgs {
		if argCount != spec.MinArgs {
			return fmt.Errorf("%s requires exactly %d argument(s), got %d", name, spec.MinArgs, argCount)
		}
	} else {
		if argCount < spec.MinArgs || (spec.MaxArgs != -1 && argCount > spec.MaxArgs) {
			if spec.MaxArgs == -1 {
				return fmt.Errorf("%s requires at least %d argument(s), got %d", name, spec.MinArgs, argCount)
			}
			return fmt.Errorf("%s requires %d to %d arguments, got %d", name, spec.MinArgs, spec.MaxArgs, argCount)
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
	spec, ok := builtInFunctions[name]
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", name)
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
					return fmt.Errorf("asin domain error: argument must be in [-1, 1], got %s", rat.String())
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
					return fmt.Errorf("acos domain error: argument must be in [-1, 1], got %s", rat.String())
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
				return fmt.Errorf("ln domain error: argument must be positive, got %s", rat.String())
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
					return fmt.Errorf("log domain error: argument must be positive, got %s", rat.String())
				}
			} else if len(args) == 2 {
				base := args[0]
				arg := args[1]
				if ratBase, ok := base.(*RationalNode); ok {
					if ratBase.Val.Sign() <= 0 {
						return fmt.Errorf("log base error: base must be positive, got %s", ratBase.String())
					}
					one := big.NewRat(1, 1)
					if ratBase.Val.Cmp(one) == 0 {
						return fmt.Errorf("log base error: base cannot be 1")
					}
				}
				if ratArg, ok := arg.(*RationalNode); ok && ratArg.Val.Sign() <= 0 {
					return fmt.Errorf("log domain error: argument must be positive, got %s", ratArg.String())
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
		Name:    "diff",
		MinArgs: 2,
		MaxArgs: 2,
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
}
