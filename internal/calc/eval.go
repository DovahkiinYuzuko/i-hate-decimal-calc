package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

var (
	rationalIntCacheSmall [21]*RationalNode // -10 .. 10 (offset +10)
	rationalHalfPositive  = &RationalNode{Val: big.NewRat(1, 2)}
	rationalHalfNegative  = &RationalNode{Val: big.NewRat(-1, 2)}
)

func init() {
	for i := -10; i <= 10; i++ {
		rationalIntCacheSmall[i+10] = &RationalNode{Val: big.NewRat(int64(i), 1)}
	}
}

// mustRational creates a RationalNode directly for guaranteed non-zero denominators,
// reusing cached singletons for common small integers and fractions.
func mustRational(num, denom int64) *RationalNode {
	switch denom {
	case 1:
		if num >= -10 && num <= 10 {
			return rationalIntCacheSmall[num+10]
		}
	case 2:
		switch num {
		case 1:
			return rationalHalfPositive
		case -1:
			return rationalHalfNegative
		}
	}
	return &RationalNode{Val: big.NewRat(num, denom)}
}

// Eval evaluates and simplifies an AST node into its canonical exact form using an empty environment.
func Eval(n Node) (Node, error) {
	return EvalWithEnv(n, NewEnv())
}

// EvalWithEnv evaluates and simplifies an AST node in the context of the given environment.
func EvalWithEnv(n Node, env *Env) (Node, error) {
	if n == nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.err_cannot_evaluate_nil_node"))
	}

	switch v := n.(type) {
	case *RationalNode:
		return v, nil

	case *ConstNode:
		if v.Name == "deg" {
			return &MulNode{
				Factors: []Node{
					NewRationalFromBigRat(big.NewRat(1, 180)),
					&ConstNode{Name: "pi"},
				},
			}, nil
		}
		return v, nil

	case *VarNode:
		if env != nil {
			if bound, ok := env.Get(v.Name); ok {
				return EvalWithEnv(bound, env)
			}
		}
		return v, nil

	case *ComplexNode:
		r, err := EvalWithEnv(v.Real, env)
		if err != nil {
			return nil, err
		}
		im, err := EvalWithEnv(v.Imag, env)
		if err != nil {
			return nil, err
		}
		return NewComplex(r, im), nil

	case *SqrtNode:
		rad, err := EvalWithEnv(v.Radicand, env)
		if err != nil {
			return nil, err
		}
		return simplifySqrtWithEnv(rad, env)

	case *FuncNode:
		if spec, ok := LookupFunction(v.Name); ok && spec.LazyArgs {
			if spec.Handler != nil {
				return spec.Handler(v.Args, env)
			}
		}
		evaledArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			ea, err := EvalWithEnv(a, env)
			if err != nil {
				return nil, err
			}
			evaledArgs[i] = ea
		}
		return simplifyFuncWithEnv(v.Name, evaledArgs, env)

	case *UnaryOpNode:
		expr, err := EvalWithEnv(v.Expr, env)
		if err != nil {
			return nil, err
		}
		return simplifyUnaryOp(v.Op, expr)

	case *PowNode:
		base, err := EvalWithEnv(v.Base, env)
		if err != nil {
			return nil, err
		}
		exp, err := EvalWithEnv(v.Exp, env)
		if err != nil {
			return nil, err
		}
		return simplifyPow(base, exp)

	case *AddNode:
		evaledTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			et, err := EvalWithEnv(t, env)
			if err != nil {
				return nil, err
			}
			evaledTerms[i] = et
		}
		return simplifyAdd(evaledTerms)

	case *MulNode:
		evaledFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			ef, err := EvalWithEnv(f, env)
			if err != nil {
				return nil, err
			}
			evaledFactors[i] = ef
		}
		return simplifyMul(evaledFactors)

	case *ListNode:
		evaledElements := make([]Node, len(v.Elements))
		for i, e := range v.Elements {
			ee, err := EvalWithEnv(e, env)
			if err != nil {
				return nil, err
			}
			evaledElements[i] = ee
		}
		return NewList(evaledElements), nil

	case *MatrixNode:
		evaledData := make([][]Node, v.Rows)
		for r := 0; r < v.Rows; r++ {
			evaledData[r] = make([]Node, v.Cols)
			for c := 0; c < v.Cols; c++ {
				ec, err := EvalWithEnv(v.Data[r][c], env)
				if err != nil {
					return nil, err
				}
				evaledData[r][c] = ec
			}
		}
		return NewMatrix(v.Rows, v.Cols, evaledData)

	case *RelOpNode:
		lhs, err := EvalWithEnv(v.LHS, env)
		if err != nil {
			return nil, err
		}
		rhs, err := EvalWithEnv(v.RHS, env)
		if err != nil {
			return nil, err
		}
		evaluatedRel := NewRelOp(lhs, v.Op, rhs)

		// For inequalities (<, <=, >, >=), attempt rigorous rational interval evaluation for constant relations:
		if v.Op == "<" || v.Op == "<=" || v.Op == ">" || v.Op == ">=" {
			if res, decided := EvaluateRelOpWithInterval(evaluatedRel); decided {
				if res {
					return &VarNode{Name: "true"}, nil
				}
				return &VarNode{Name: "false"}, nil
			}
		}

		return evaluatedRel, nil

	case *QuantifierNode:
		return v, nil

	default:
		return nil, fmt.Errorf("%s", i18n.T("errors.err_unknown_node_type_for_evaluation", n))
	}
}

// EvalString parses and evaluates a math expression string using an empty environment.
func EvalString(input string) (Node, error) {
	return EvalStringWithEnv(input, NewEnv())
}

// EvalStringWithEnv parses and evaluates a math expression string using the given environment.
func EvalStringWithEnv(input string, env *Env) (Node, error) {
	node, err := Parse(input)
	if err != nil {
		return nil, err
	}
	res, err := EvalWithEnv(node, env)
	if err != nil {
		return nil, err
	}
	if rel, ok := res.(*RelOpNode); ok && (rel.Op == "==" || rel.Op == "!=") {
		return EvaluateRelOpEquality(rel, env)
	}
	return res, nil
}

// ApplyDegreeMode transforms trigonometric and inverse trigonometric function calls in the AST
// to work with degrees rather than radians.
// sin(x), cos(x), tan(x) -> sin(x * deg), cos(x * deg), tan(x * deg)
// asin(x), acos(x), atan(x) -> asin(x) / deg, acos(x) / deg, atan(x) / deg
func ApplyDegreeMode(n Node) Node {
	if n == nil {
		return nil
	}

	degConst := &ConstNode{Name: "deg"}

	switch v := n.(type) {
	case *FuncNode:
		newArgs := make([]Node, len(v.Args))
		for i, a := range v.Args {
			newArgs[i] = ApplyDegreeMode(a)
		}

		switch v.Name {
		case "sin", "cos", "tan":
			if len(newArgs) >= 1 {
				// wrap argument with * deg
				newArgs[0] = &MulNode{
					Factors: []Node{newArgs[0], degConst},
				}
			}
			return &FuncNode{Name: v.Name, Args: newArgs}

		case "asin", "acos", "atan":
			// wrap result with / deg (i.e. * deg^-1)
			degInv := &PowNode{
				Base: degConst,
				Exp:  mustRational(-1, 1),
			}
			return &MulNode{
				Factors: []Node{
					&FuncNode{Name: v.Name, Args: newArgs},
					degInv,
				},
			}

		default:
			return &FuncNode{Name: v.Name, Args: newArgs}
		}

	case *AddNode:
		newTerms := make([]Node, len(v.Terms))
		for i, t := range v.Terms {
			newTerms[i] = ApplyDegreeMode(t)
		}
		return &AddNode{Terms: newTerms}

	case *MulNode:
		newFactors := make([]Node, len(v.Factors))
		for i, f := range v.Factors {
			newFactors[i] = ApplyDegreeMode(f)
		}
		return &MulNode{Factors: newFactors}

	case *PowNode:
		return &PowNode{
			Base: ApplyDegreeMode(v.Base),
			Exp:  ApplyDegreeMode(v.Exp),
		}

	case *UnaryOpNode:
		return &UnaryOpNode{
			Op:   v.Op,
			Expr: ApplyDegreeMode(v.Expr),
		}

	case *SqrtNode:
		return &SqrtNode{
			Radicand: ApplyDegreeMode(v.Radicand),
		}

	case *ComplexNode:
		return &ComplexNode{
			Real: ApplyDegreeMode(v.Real),
			Imag: ApplyDegreeMode(v.Imag),
		}

	case *ListNode:
		newElements := make([]Node, len(v.Elements))
		for i, e := range v.Elements {
			newElements[i] = ApplyDegreeMode(e)
		}
		return &ListNode{Elements: newElements}

	case *MatrixNode:
		newData := make([][]Node, v.Rows)
		for r := 0; r < v.Rows; r++ {
			newData[r] = make([]Node, v.Cols)
			for c := 0; c < v.Cols; c++ {
				newData[r][c] = ApplyDegreeMode(v.Data[r][c])
			}
		}
		return &MatrixNode{Rows: v.Rows, Cols: v.Cols, Data: newData}

	default:
		return n
	}
}

// ApplyDegreeModeToStatement applies degree mode to expressions within statements or plain expression nodes.
func ApplyDegreeModeToStatement(stmt interface{}) interface{} {
	switch s := stmt.(type) {
	case *AssignStmt:
		return &AssignStmt{
			Name:  s.Name,
			Value: ApplyDegreeMode(s.Value),
		}
	case Node:
		return ApplyDegreeMode(s)
	default:
		return stmt
	}
}

// -------------------------------------------------------------------------
// Functions (sin, cos, tan, log, ln)
// -------------------------------------------------------------------------

func simplifyFunc(name string, args []Node) (Node, error) {
	return simplifyFuncWithEnv(name, args, nil)
}

func simplifyFuncWithEnv(name string, args []Node, env *Env) (Node, error) {
	spec, ok := LookupFunction(name)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.unknown_function", name))
	}
	if spec.Handler != nil {
		return spec.Handler(args, env)
	}
	if spec.Evaluate != nil {
		return spec.Evaluate(args)
	}
	return NewFunc(name, args)
}

// EvaluateRelOpEquality evaluates equality and inequality relations (==, !=) into boolean values (true/false)
// when decidable algebraically or numerically via interval arithmetic. If undecided (e.g. contains variables),
// it returns the original relation unchanged.
func EvaluateRelOpEquality(rel *RelOpNode, env *Env) (Node, error) {
	if rel == nil {
		return nil, nil
	}
	if rel.Op != "==" && rel.Op != "!=" {
		return rel, nil
	}

	lhs := rel.LHS
	rhs := rel.RHS

	// 1. 代数的完全一致 (Algebraic exact identity)
	if lhs.Equal(rhs) {
		if rel.Op == "==" {
			return &VarNode{Name: "true"}, nil
		}
		return &VarNode{Name: "false"}, nil
	}

	// 2. 単一有理数値どうしの厳密比較
	if lRat, okL := lhs.(*RationalNode); okL {
		if rRat, okR := rhs.(*RationalNode); okR {
			isEqual := lRat.Val.Cmp(rRat.Val) == 0
			if (rel.Op == "==" && isEqual) || (rel.Op == "!=" && !isEqual) {
				return &VarNode{Name: "true"}, nil
			}
			return &VarNode{Name: "false"}, nil
		}
	}

	// 3. 代数式全体の差の簡約: Simplify(lhs - rhs) == 0
	negR, errNeg := simplifyUnaryOp("-", rhs)
	if errNeg == nil {
		diff, errDiff := simplifyAdd([]Node{lhs, negR})
		if errDiff == nil && isZero(diff) {
			if rel.Op == "==" {
				return &VarNode{Name: "true"}, nil
			}
			return &VarNode{Name: "false"}, nil
		}
	}

	// 4. 代数的等価性検証エンジンによる証明 (Algebraic equivalence verification)
	if cert, err := VerifyAlgebraicEquivalence(lhs, rhs, env); err == nil && cert != nil && cert.IsVerified {
		if rel.Op == "==" {
			return &VarNode{Name: "true"}, nil
		}
		return &VarNode{Name: "false"}, nil
	}

	// 5. 有理区間解析による厳密分離判定 (Rigorous rational interval separation)
	if res, decided := EvaluateRelOpWithInterval(rel); decided {
		if res {
			return &VarNode{Name: "true"}, nil
		}
		return &VarNode{Name: "false"}, nil
	}

	// 6. 判定不能（未確定の変数を含む方程式: x == 1, y' == y など）は式のまま保持
	return rel, nil
}

