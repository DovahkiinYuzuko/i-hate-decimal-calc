package calc

import (
	"fmt"
	"math/big"
	"sort"
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
	Args             []string
	DescKey          string
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
		// Allow unreserved identifier to act as a free symbolic function (e.g. sequence a(n) in rsolve)
		return nil
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
		return nil, fmt.Errorf("%s", i18n.T("registry.err_function", name))
	}
	return spec.Evaluate(args)
}

// SignatureInfo holds metadata and parameter names for auto-completion and REPL signature guidance.
type SignatureInfo struct {
	Name    string
	Args    []string
	MinArgs int
	MaxArgs int
	DescKey string
}

// defaultFunctionArgs maps known function names to human-friendly parameter names.
var defaultFunctionArgs = map[string][]string{
	"sqrt":             {"x"},
	"cbrt":             {"x"},
	"exp":              {"x"},
	"ln":               {"x"},
	"log":              {"base", "x"},
	"sin":              {"x"},
	"cos":              {"x"},
	"tan":              {"x"},
	"asin":             {"x"},
	"acos":             {"x"},
	"atan":             {"x"},
	"atan2":            {"y", "x"},
	"sinh":             {"x"},
	"cosh":             {"x"},
	"tanh":             {"x"},
	"asinh":            {"x"},
	"acosh":            {"x"},
	"atanh":            {"x"},
	"abs":              {"x"},
	"floor":            {"x"},
	"ceil":             {"x"},
	"round":            {"x"},
	"sign":             {"x"},
	"diff":             {"expr", "var"},
	"integrate":        {"expr", "var", "from", "to"},
	"limit":            {"expr", "var", "target", "dir"},
	"taylor":           {"expr", "var", "point", "order"},
	"sum":              {"expr", "var", "from", "to"},
	"product":          {"expr", "var", "from", "to"},
	"solve":            {"eq", "var"},
	"linsolve":         {"eqs", "vars"},
	"solve_linear":     {"eqs", "vars"},
	"dsolve":           {"ode", "var"},
	"factor":           {"poly", "var"},
	"expand":           {"expr"},
	"simplify":         {"expr"},
	"gcd":              {"a", "b"},
	"lcm":              {"a", "b"},
	"mod":              {"a", "b"},
	"modpow":           {"base", "exp", "mod"},
	"modinv":           {"a", "mod"},
	"perm":             {"n", "k"},
	"comb":             {"n", "k"},
	"matrix":           {"rows"},
	"det":              {"matrix"},
	"inv":              {"matrix"},
	"transpose":        {"matrix"},
	"trace":            {"matrix"},
	"rank":             {"matrix"},
	"rref":             {"matrix"},
	"eigenvalues":      {"matrix"},
	"eigenvectors":     {"matrix"},
	"dot":              {"v1", "v2"},
	"cross":            {"v1", "v2"},
	"norm":             {"v"},
	"grad":             {"expr", "vars"},
	"div":              {"vector", "vars"},
	"curl":             {"vector", "vars"},
	"laplacian":        {"expr", "vars"},
	"line_intersect":   {"line1", "line2"},
	"circle_intersect": {"circle1", "circle2"},
	"triangle_area":    {"a", "b", "c"},
	"triangle_centers": {"a", "b", "c"},
	"plot":             {"expr", "range"},
	"cad":              {"formula", "vars"},
	"qe":               {"formula"},
	"clifford":         {"expr", "p", "q", "r"},
	"rsolve":           {"eq", "func"},
	"verify":           {"expr"},
	"approx":           {"expr"},
	"interval":         {"expr"},

	// Advanced Algebra & Simplification
	"apart":                 {"expr", "var"},
	"together":              {"expr"},
	"trig_expand":           {"expr"},
	"trig_reduce":           {"expr"},
	"alg_inv":               {"poly"},
	"to_alg":                {"expr", "minpoly"},
	"to_poly":               {"expr", "var"},
	"poly_gcd":              {"a", "b", "var"},
	"poly_lcm":              {"a", "b", "var"},
	"resultant":             {"p1", "p2", "var"},
	"groebner":              {"polys", "vars", "order"},
	"ideal_intersection":    {"i1", "i2", "vars"},
	"ideal_colon":           {"i1", "i2", "vars"},
	"ideal_saturation":      {"i1", "i2", "vars"},
	"ideal_radical":         {"ideal", "vars"},
	"ideal_dimension":       {"ideal", "vars"},
	"primary_decomposition": {"ideal", "vars"},
	"primitive_element":     {"alpha1", "alpha2"},
	"to_primitive_element":  {"expr", "prim"},
	"min_poly":              {"elem", "vars"},
	"minimal_polynomial":    {"elem", "var"},
	"find_min_poly":         {"elem", "degree"},
	"galois_group":          {"poly", "var"},
	"is_solvable_by_radicals": {"poly", "var"},
	"abelian_group_structure": {"matrix"},

	// Matrix Decompositions & Linear Algebra
	"eigenvals":         {"matrix"},
	"eigenvects":        {"matrix"},
	"lu":                {"matrix"},
	"qr":                {"matrix"},
	"cholesky":          {"matrix"},
	"ldlt":              {"matrix"},
	"pinv":              {"matrix"},
	"tr":                {"matrix"},
	"hnf":               {"matrix"},
	"hnf_transform":     {"matrix"},
	"snf":               {"matrix"},
	"snf_transform":     {"matrix"},
	"invariant_factors": {"matrix"},
	"lll":               {"matrix", "delta"},

	// Number Theory & Discrete Math
	"is_prime":          {"n"},
	"totient":           {"n"},
	"inv_mod":           {"a", "m"},
	"crt":               {"remainders", "moduli"},
	"bernoulli":         {"n"},
	"fib":               {"n"},
	"fibonacci":         {"n"},
	"digit_count":       {"n"},
	"cfrac":             {"x"},
	"from_cfrac":        {"terms"},
	"solve_diophantine": {"eq", "vars"},
	"pell":              {"d", "n"},
	"solve_pell":        {"d", "n"},
	"pell_solve":        {"d", "n"},
	"bin":               {"n"},
	"oct":               {"n"},
	"hex":               {"n"},
	"to_base":           {"n", "base"},
	"from_base":         {"str", "base"},

	// Complex Analysis & Special Functions
	"arg":       {"z"},
	"polar":     {"z"},
	"polar_exp": {"z"},
	"rect":      {"r", "theta"},
	"gamma":     {"x"},
	"beta":      {"x", "y"},
	"zeta":      {"s"},
	"erf":       {"x"},
	"lambert_w": {"x", "branch"},
	"residue":   {"expr", "var", "pole"},

	// Integral Transforms & Summation
	"laplace":         {"expr", "t", "s"},
	"inv_laplace":     {"expr", "s", "t"},
	"ztrans":          {"expr", "n", "z"},
	"iztrans":         {"expr", "z", "n"},
	"fourier_series":  {"expr", "var", "period", "terms"},
	"gosper_sum":      {"expr", "var"},
	"wz_cert":         {"expr", "n", "k"},
	"risch_integrate": {"expr", "var"},

	// Real Algebraic Roots & Intervals
	"sturm":         {"poly", "var"},
	"root_count":    {"poly", "var", "a", "b"},
	"isolate_roots": {"poly", "var", "a", "b"},

	// Elliptic Curves
	"ec_torsion":    {"a", "b"},
	"ec_rank_bound": {"a", "b"},
	"ec_generators": {"a", "b"},
	"ec_add":        {"p1", "p2", "a", "b"},
	"ec_mul":        {"k", "p", "a", "b"},
	"ec_p_order":    {"a", "b"},
	"ec_p_trace":    {"a", "b"},
	"ec_p_add":      {"p1", "p2", "a", "b"},
	"ec_p_mul":      {"k", "p", "a", "b"},

	// p-adic & Puiseux Series
	"padic_val":    {"x", "p"},
	"padic_norm":   {"x", "p"},
	"padic_expand": {"x", "p", "n"},
	"puiseux":      {"expr", "var", "point", "order"},

	// Clifford Algebra & Quantum
	"clifford_wedge":    {"a", "b", "p", "q", "r"},
	"clifford_contract": {"a", "b", "p", "q", "r"},
	"clifford_dual":     {"mv", "p", "q", "r"},
	"quantum_equiv":     {"c1", "c2", "tolerance"},
	"quantum_eval":      {"circuit", "state"},

	// Optimization & Linear Programming
	"lp_maximize": {"objective", "constraints", "vars", "method"},
	"lp_minimize": {"objective", "constraints", "vars", "method"},

	// Automated Theorem Proving & Geometry DSL
	"geo_prove":        {"conjecture", "hypotheses", "method"},
	"midpoint":         {"m", "a", "b"},
	"collinear":        {"a", "b", "c"},
	"parallel":         {"a", "b", "c", "d"},
	"perpendicular":    {"a", "b", "c", "d"},
	"equal_length":     {"a", "b", "c", "d"},
	"equal_length_sq":  {"a", "b", "c", "d"},
	"circle_concyclic": {"a", "b", "c", "d"},

	// Probability & Statistics
	"binom":    {"n", "p", "k"},
	"geom":     {"p", "k"},
	"hyper":    {"bigN", "bigK", "n", "k"},
	"bayes":    {"prior", "likelihood", "evidence"},
	"expect":   {"dist", "p1", "p2", "p3"},
	"variance": {"dist", "p1", "p2", "p3"},
	"stddev":   {"dist", "p1", "p2", "p3"},
	"rand":     {"min", "max"},

	// Programming Constructs & Collections
	"map":       {"func", "list"},
	"filter":    {"pred", "list"},
	"range":     {"start", "stop", "step"},
	"table":     {"expr", "var", "from", "to", "step"},
	"for":       {"init", "cond", "update", "body"},
	"lambda":    {"params", "body"},
	"piecewise": {"cases", "otherwise"},

	// Assumptions & Environment
	"assume":            {"var", "domain"},
	"unassume":          {"var"},
	"assumptions":       {},
	"clear_assumptions": {},

	// Logic & Quantifiers
	"forall": {"vars", "condition"},
	"exists": {"vars", "condition"},

	// Units & Physical Dimensions
	"convert_unit": {"expr", "target_unit"},
	"unit":         {"name"},
	"u":            {"name"},
	"delta":        {"expr"},

	// Algebraic Pattern Matching & Term Rewriting System (TRS)
	"rule":    {"lhs", "rhs"},
	"replace": {"expr", "rules"},
	"rewrite": {"expr", "rules", "max_steps"},
}

// AllFunctionNames returns a sorted list of all registered built-in function names.
func AllFunctionNames() []string {
	names := make([]string, 0, len(builtInFunctions))
	for name := range builtInFunctions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// LookupSignature returns signature information for the named function.
func LookupSignature(name string) (SignatureInfo, bool) {
	spec, ok := LookupFunction(name)
	if !ok {
		return SignatureInfo{}, false
	}

	args := spec.Args
	if len(args) == 0 {
		if defArgs, found := defaultFunctionArgs[strings.ToLower(spec.Name)]; found {
			args = defArgs
		} else {
			count := spec.MinArgs
			if count <= 0 {
				count = 1
			}
			args = make([]string, count)
			for i := 0; i < count; i++ {
				args[i] = fmt.Sprintf("arg%d", i+1)
			}
		}
	}

	descKey := spec.DescKey
	if descKey == "" {
		descKey = "fn." + strings.ToLower(spec.Name)
	}

	return SignatureInfo{
		Name:    spec.Name,
		Args:    args,
		MinArgs: spec.MinArgs,
		MaxArgs: spec.MaxArgs,
		DescKey: descKey,
	}, true
}

// AllFunctionSignatures returns sorted signature info for all registered functions.
func AllFunctionSignatures() []SignatureInfo {
	names := AllFunctionNames()
	sigs := make([]SignatureInfo, 0, len(names))
	for _, name := range names {
		if sig, ok := LookupSignature(name); ok {
			sigs = append(sigs, sig)
		}
	}
	return sigs
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

	RegisterFunction(FunctionSpec{
		Name:    "lambert_w",
		MinArgs: 1,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "erf",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "interval",
		MinArgs: 1,
		MaxArgs: 2,
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
				return fmt.Errorf("%s", i18n.T("registry.err_integrate_requires_2_arguments_indefinite", len(args)))
			}
			if _, ok := args[1].(*VarNode); !ok {
				return fmt.Errorf("%s", i18n.T("registry.err_integrate_error_second_argument_must", args[1].String()))
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
		MinArgs: 1,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "cad",
		MinArgs: 1,
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
		Name:     "table",
		MinArgs:  4,
		MaxArgs:  5,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "product",
		MinArgs:  4,
		MaxArgs:  5,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "for",
		MinArgs:  4,
		MaxArgs:  5,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:    "range",
		MinArgs: 1,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:     "map",
		MinArgs:  2,
		MaxArgs:  2,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "filter",
		MinArgs:  2,
		MaxArgs:  2,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "lambda",
		MinArgs:  2,
		MaxArgs:  2,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:    "limit",
		MinArgs: 3,
		MaxArgs: 4,
		Validate: func(args []Node) error {
			if len(args) != 3 && len(args) != 4 {
				return fmt.Errorf("%s", i18n.T("registry.err_limit_requires_3_arguments_expr", len(args)))
			}
			if _, ok := args[1].(*VarNode); !ok {
				return fmt.Errorf("%s", i18n.T("registry.err_limit_error_second_argument_must", args[1].String()))
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
	RegisterFunction(FunctionSpec{
		Name:    "solve_pell",
		MinArgs: 1,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "pell",
		MinArgs: 1,
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
	RegisterFunction(FunctionSpec{
		Name:    "digit_count",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "fib",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "fibonacci",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "to_base",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "from_base",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "bin",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "oct",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "hex",
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

	RegisterFunction(FunctionSpec{
		Name:    "u",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ztrans",
		MinArgs: 1,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "iztrans",
		MinArgs: 1,
		MaxArgs: 3,
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

	// Primary Decomposition of Multivariate Polynomial Ideals (issue-107)
	RegisterFunction(FunctionSpec{
		Name:    "primary_decomposition",
		MinArgs: 2,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ideal_radical",
		MinArgs: 2,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ideal_dimension",
		MinArgs: 2,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ideal_intersection",
		MinArgs: 3,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ideal_colon",
		MinArgs: 3,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ideal_saturation",
		MinArgs: 3,
		MaxArgs: 4,
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

	// Canonical Polynomial Representation & SDMP (issue-63)
	RegisterFunction(FunctionSpec{
		Name:    "to_poly",
		MinArgs: 1,
		MaxArgs: 3,
	})

	// Algebraic Number Fields & Minimal Polynomials (issue-65)
	RegisterFunction(FunctionSpec{
		Name:    "to_alg",
		MinArgs: 2,
		MaxArgs: 3,
	})
	RegisterFunction(FunctionSpec{
		Name:    "alg_inv",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "min_poly",
		MinArgs: 2,
		MaxArgs: 3,
	})

	// Hypergeometric Summation & WZ Theory (issue-68)
	RegisterFunction(FunctionSpec{
		Name:    "gosper_sum",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "wz_cert",
		MinArgs: 3,
		MaxArgs: 3,
	})

	// Deterministic Risch Algorithm (issue-66)
	RegisterFunction(FunctionSpec{
		Name:    "risch_integrate",
		MinArgs: 1,
		MaxArgs: 2,
	})

	// Self-Verifying CAS (issue-67)
	RegisterFunction(FunctionSpec{
		Name:    "verify",
		MinArgs: 1,
		MaxArgs: 2,
	})

	// Recurrence Relation Solver (issue-77)
	RegisterFunction(FunctionSpec{
		Name:            "rsolve",
		MinArgs:         2,
		MaxArgs:         3,
		AllowBareSymbol: true,
		LazyArgs:        true,
	})

	// Quantifier Elimination & First-Order Logic (issue-74)
	RegisterFunction(FunctionSpec{
		Name:     "qe",
		MinArgs:  1,
		MaxArgs:  1,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "forall",
		MinArgs:  2,
		MaxArgs:  2,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "exists",
		MinArgs:  2,
		MaxArgs:  2,
		LazyArgs: true,
	})

	// Elliptic Curves over Rationals & Nagell-Lutz (issue-76)
	RegisterFunction(FunctionSpec{
		Name:            "ec_add",
		MinArgs:         4,
		MaxArgs:         6,
		AllowBareSymbol: true,
	})
	RegisterFunction(FunctionSpec{
		Name:            "ec_mul",
		MinArgs:         4,
		MaxArgs:         5,
		AllowBareSymbol: true,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ec_torsion",
		MinArgs: 2,
		MaxArgs: 2,
	})
	// Elliptic Curve 2-Descent & Mordell-Weil Rank (issue-109)
	RegisterFunction(FunctionSpec{
		Name:    "ec_rank_bound",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ec_generators",
		MinArgs: 2,
		MaxArgs: 2,
	})

	// LLL Lattice Basis Reduction & Integer Relation (issue-75)
	RegisterFunction(FunctionSpec{
		Name:    "lll",
		MinArgs: 1,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "find_min_poly",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "minimal_polynomial",
		MinArgs: 1,
		MaxArgs: 2,
	})

	// p-adic Arithmetic & Hensel Lifting (issue-83)
	RegisterFunction(FunctionSpec{
		Name:    "padic_val",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "padic_norm",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "padic_expand",
		MinArgs: 3,
		MaxArgs: 3,
	})

	// Newton-Puiseux Algebraic Series Expansion (issue-84)
	RegisterFunction(FunctionSpec{
		Name:    "puiseux",
		MinArgs: 3,
		MaxArgs: 4,
	})

	// Certified Exact Digit Count (issue-115)
	RegisterFunction(FunctionSpec{
		Name:    "digit_count",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Finite Field Elliptic Curves & Schoof's Algorithm (issue-88)
	RegisterFunction(FunctionSpec{
		Name:    "ec_p_add",
		MinArgs: 4,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ec_p_mul",
		MinArgs: 4,
		MaxArgs: 4,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ec_p_order",
		MinArgs: 2,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "ec_p_trace",
		MinArgs: 2,
		MaxArgs: 2,
	})

	// Piecewise Functions & CAD Domain Branching (issue-98)
	RegisterFunction(FunctionSpec{
		Name:     "piecewise",
		MinArgs:  1,
		MaxArgs:  2,
		LazyArgs: true,
		Handler:  EvalPiecewiseHandler,
	})

	// Integer Matrix Normal Forms & Abelian Groups (issue-101)
	RegisterFunction(FunctionSpec{
		Name:    "hnf",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "hnf_transform",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "snf",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "snf_transform",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "invariant_factors",
		MinArgs: 1,
		MaxArgs: 1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "abelian_group_structure",
		MinArgs: 1,
		MaxArgs: 1,
	})

	// Diophantine Equation Solvers (issue-102)
	RegisterFunction(FunctionSpec{
		Name:    "solve_diophantine",
		MinArgs: 1,
		MaxArgs: 2,
	})
	RegisterFunction(FunctionSpec{
		Name:    "pell_solve",
		MinArgs: 1,
		MaxArgs: 2,
	})

	// Clifford Geometric Algebra & Quantum Circuit Equivalence (issue-108)
	RegisterFunction(FunctionSpec{
		Name:     "clifford",
		MinArgs:  3,
		MaxArgs:  4,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "clifford_wedge",
		MinArgs:  4,
		MaxArgs:  5,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "clifford_contract",
		MinArgs:  5,
		MaxArgs:  6,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "clifford_dual",
		MinArgs:  3,
		MaxArgs:  4,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "quantum_equiv",
		MinArgs:  2,
		MaxArgs:  3,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "quantum_eval",
		MinArgs:  1,
		MaxArgs:  2,
		LazyArgs: true,
	})

	// Primitive Element Theorem & Simple Extension Isomorphism
	RegisterFunction(FunctionSpec{
		Name:    "to_primitive_element",
		MinArgs: 1,
		MaxArgs: -1,
	})
	RegisterFunction(FunctionSpec{
		Name:    "primitive_element",
		MinArgs: 1,
		MaxArgs: -1,
	})

	// Physical Units & Dimensional Analysis
	RegisterFunction(FunctionSpec{
		Name:     "convert_unit",
		MinArgs:  2,
		MaxArgs:  2,
		LazyArgs: true,
	})
	RegisterFunction(FunctionSpec{
		Name:     "unit",
		MinArgs:  1,
		MaxArgs:  1,
		LazyArgs: true,
	})

	// Algebraic Pattern Matching & Term Rewriting System (TRS)
	RegisterFunction(FunctionSpec{
		Name:     "rule",
		MinArgs:  2,
		MaxArgs:  2,
		LazyArgs: true,
		Handler:  EvalRule,
	})
	RegisterFunction(FunctionSpec{
		Name:     "replace",
		MinArgs:  2,
		MaxArgs:  2,
		LazyArgs: true,
		Handler:  EvalReplace,
	})
	RegisterFunction(FunctionSpec{
		Name:     "rewrite",
		MinArgs:  2,
		MaxArgs:  3,
		LazyArgs: true,
		Handler:  EvalRewrite,
	})
}






