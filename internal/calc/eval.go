package calc

import (
	"fmt"
	"math/big"
	"math/rand/v2"

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
		return nil, fmt.Errorf("cannot evaluate nil node")
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
		if v.Name == "dsolve" {
			return evalDSolveSpecial(v.Args, env)
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
		return NewRelOp(lhs, v.Op, rhs), nil

	default:
		return nil, fmt.Errorf("unknown node type for evaluation: %T", n)
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
	return EvalWithEnv(node, env)
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
	switch name {
	case "assume":
		if env == nil {
			return nil, fmt.Errorf("no environment available for assumptions")
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
		return nil, fmt.Errorf("assume requires 1 or 2 arguments")

	case "unassume":
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
		return nil, fmt.Errorf("unassume expects a variable name")

	case "assumptions":
		if env == nil {
			return NewList(nil), nil
		}
		list := env.Assumptions().ListAssumptions()
		var elems []Node
		for _, s := range list {
			elems = append(elems, &VarNode{Name: s})
		}
		return NewList(elems), nil

	case "clear_assumptions":
		if env != nil {
			env.Assumptions().ClearAll()
		}
		return &VarNode{Name: "ok"}, nil

	case "sin":
		arg := args[0]
		if r, ok := matchPiMultiple(arg); ok {
			// sin(pi * r)
			if val, ok := evalTrigPi("sin", r); ok {
				return val, nil
			}
		}
		if val, ok := evalTrigWithAssumptions("sin", arg, env); ok {
			return val, nil
		}
		if rat, ok := arg.(*RationalNode); ok && rat.Val.Sign() == 0 {
			return mustRational(0, 1), nil
		}
		return NewFunc(name, args)

	case "cos":
		arg := args[0]
		if r, ok := matchPiMultiple(arg); ok {
			if val, ok := evalTrigPi("cos", r); ok {
				return val, nil
			}
		}
		if val, ok := evalTrigWithAssumptions("cos", arg, env); ok {
			return val, nil
		}
		if rat, ok := arg.(*RationalNode); ok && rat.Val.Sign() == 0 {
			return mustRational(1, 1), nil
		}
		return NewFunc(name, args)

	case "tan":
		arg := args[0]
		if r, ok := matchPiMultiple(arg); ok {
			// Check for undefined values: pi/2 + k*pi (where r = 1/2, 3/2, etc.)
			// r - 1/2 is integer?
			rMinusHalf := new(big.Rat).Sub(r, big.NewRat(1, 2))
			if rMinusHalf.IsInt() {
				return nil, NewZeroDivisionError("math error: tan(%s) is undefined (division by zero)", arg.String())
			}
			if val, ok := evalTrigPi("tan", r); ok {
				return val, nil
			}
		}
		if val, ok := evalTrigWithAssumptions("tan", arg, env); ok {
			return val, nil
		}
		if rat, ok := arg.(*RationalNode); ok && rat.Val.Sign() == 0 {
			return mustRational(0, 1), nil
		}
		return NewFunc(name, args)

	case "asin", "acos", "atan":
		arg := args[0]
		if val, ok, err := evalInverseTrig(name, arg); err != nil {
			return nil, err
		} else if ok {
			return val, nil
		}
		return NewFunc(name, args)

	case "log":
		if len(args) == 1 {
			// log10(x)
			rat, ok := args[0].(*RationalNode)
			if !ok {
				return NewFunc(name, args)
			}
			if rat.Val.Sign() <= 0 {
				return nil, NewDomainError("errors.domain_log_arg", "log domain error: argument must be positive, got %s", rat.String())
			}
			if rat.Val.IsInt() {
				n := rat.Val.Num().Int64()
				power := 0
				for n > 1 && n%10 == 0 {
					n /= 10
					power++
				}
				if n == 1 {
					return mustRational(int64(power), 1), nil
				}
			}
			return NewFunc(name, args)
		}
		// log(base, x)
		baseRat, baseOk := args[0].(*RationalNode)
		xRat, xOk := args[1].(*RationalNode)
		if baseOk {
			if baseRat.Val.Sign() <= 0 {
				return nil, NewDomainError("errors.domain_log_base", "log base error: base must be positive, got %s", baseRat.String())
			}
			if baseRat.Val.Cmp(big.NewRat(1, 1)) == 0 {
				return nil, NewDomainError("errors.domain_log_base_one", "log base error: base cannot be 1")
			}
		}
		if xOk && xRat.Val.Sign() <= 0 {
			return nil, NewDomainError("errors.domain_log_arg", "log domain error: argument must be positive, got %s", xRat.String())
		}
		if baseOk && xOk && baseRat.Val.IsInt() && xRat.Val.IsInt() {
			b := baseRat.Val.Num().Int64()
			x := xRat.Val.Num().Int64()
			power := 0
			for x > 1 && x%b == 0 {
				x /= b
				power++
			}
			if x == 1 {
				return mustRational(int64(power), 1), nil
			}
		}
		return NewFunc(name, args)

	case "exp":
		if isZero(args[0]) {
			return mustRational(1, 1), nil
		}
		return NewFunc(name, args)

	case "ln":
		arg := args[0]
		if c, ok := arg.(*ConstNode); ok && c.Name == "e" {
			return mustRational(1, 1), nil
		}
		if pow, ok := arg.(*PowNode); ok {
			if c, ok := pow.Base.(*ConstNode); ok && c.Name == "e" {
				return pow.Exp, nil // ln(e^x) = x
			}
		}
		rat, ok := arg.(*RationalNode)
		if !ok {
			return NewFunc(name, args)
		}
		if rat.Val.Sign() <= 0 {
			return nil, NewDomainError("errors.domain_ln_arg", "ln domain error: argument must be positive, got %s", rat.String())
		}
		if rat.Val.Cmp(big.NewRat(1, 1)) == 0 {
			return mustRational(0, 1), nil
		}
		return NewFunc(name, args)

	case "abs":
		arg := args[0]
		switch v := arg.(type) {
		case *RationalNode:
			newRat := new(big.Rat).Abs(v.Val)
			return NewRationalFromBigRat(newRat), nil
		case *ComplexNode:
			// |a + bi| = sqrt(a^2 + b^2)
			aSq := &PowNode{Base: v.Real, Exp: mustRational(2, 1)}
			bSq := &PowNode{Base: v.Imag, Exp: mustRational(2, 1)}
			sumNode := NewAdd([]Node{aSq, bSq})
			evaledSum, err := Eval(sumNode)
			if err != nil {
				return nil, err
			}
			return simplifySqrt(evaledSum)
		case *UnaryOpNode:
			if v.Op == "-" {
				return simplifyFunc("abs", []Node{v.Expr})
			}
		case *SqrtNode:
			return v, nil
		case *MulNode:
			if len(v.Factors) > 0 {
				if r, ok := v.Factors[0].(*RationalNode); ok && r.Val.Sign() < 0 {
					posR := NewRationalFromBigRat(new(big.Rat).Abs(r.Val))
					newFactors := make([]Node, len(v.Factors))
					copy(newFactors, v.Factors)
					newFactors[0] = posR
					return simplifyMul(newFactors)
				}
			}
		}
		if isNegative(arg) {
			neg, err := simplifyUnaryOp("-", arg)
			if err == nil {
				return simplifyFunc("abs", []Node{neg})
			}
		}
		return NewFunc(name, args)

	case "cbrt":
		arg := args[0]
		if u, ok := arg.(*UnaryOpNode); ok && u.Op == "-" {
			sub, err := simplifyFunc("cbrt", []Node{u.Expr})
			if err != nil {
				return nil, err
			}
			return simplifyUnaryOp("-", sub)
		}
		if rat, ok := arg.(*RationalNode); ok {
			if rat.Val.Sign() < 0 {
				posRat := new(big.Rat).Abs(rat.Val)
				sub, err := simplifyFunc("cbrt", []Node{NewRationalFromBigRat(posRat)})
				if err != nil {
					return nil, err
				}
				return simplifyUnaryOp("-", sub)
			}
			if rat.Val.Sign() == 0 {
				return mustRational(0, 1), nil
			}
			if root, ok := isRatPerfectCube(rat.Val); ok {
				return NewRationalFromBigRat(root), nil
			}
			numOut, numRem := extractCubeFree(rat.Val.Num())
			denomOut, denomRem := extractCubeFree(rat.Val.Denom())
			one := big.NewInt(1)
			if numOut.Cmp(one) > 0 || denomOut.Cmp(one) > 0 {
				coeff := new(big.Rat).SetFrac(numOut, denomOut)
				rem := new(big.Rat).SetFrac(numRem, denomRem)
				cbrtRem, _ := NewFunc("cbrt", []Node{NewRationalFromBigRat(rem)})
				return simplifyMul([]Node{NewRationalFromBigRat(coeff), cbrtRem})
			}
		}
		return NewFunc(name, args)

	case "gcd":
		aRat, aOk := args[0].(*RationalNode)
		bRat, bOk := args[1].(*RationalNode)
		if aOk && bOk {
			if !aRat.Val.IsInt() || !bRat.Val.IsInt() {
				return nil, fmt.Errorf("gcd domain error: arguments must be integers, got %s and %s", aRat.String(), bRat.String())
			}
			g := new(big.Int).GCD(nil, nil, aRat.Val.Num(), bRat.Val.Num())
			return NewRationalFromBigRat(new(big.Rat).SetInt(g)), nil
		}
		return NewFunc(name, args)

	case "lcm":
		aRat, aOk := args[0].(*RationalNode)
		bRat, bOk := args[1].(*RationalNode)
		if aOk && bOk {
			if !aRat.Val.IsInt() || !bRat.Val.IsInt() {
				return nil, fmt.Errorf("lcm domain error: arguments must be integers, got %s and %s", aRat.String(), bRat.String())
			}
			if aRat.Val.Sign() == 0 || bRat.Val.Sign() == 0 {
				return mustRational(0, 1), nil
			}
			aInt := aRat.Val.Num()
			bInt := bRat.Val.Num()
			g := new(big.Int).GCD(nil, nil, aInt, bInt)
			div := new(big.Int).Div(aInt, g)
			l := new(big.Int).Mul(div, bInt)
			l.Abs(l)
			return NewRationalFromBigRat(new(big.Rat).SetInt(l)), nil
		}
		return NewFunc(name, args)

	case "mod":
		aRat, aOk := args[0].(*RationalNode)
		bRat, bOk := args[1].(*RationalNode)
		if aOk && bOk {
			if !aRat.Val.IsInt() || !bRat.Val.IsInt() {
				return nil, fmt.Errorf("mod domain error: arguments must be integers, got %s and %s", aRat.String(), bRat.String())
			}
			if bRat.Val.Sign() == 0 {
				return nil, NewZeroDivisionError("division by zero in mod")
			}
			m := new(big.Int).Mod(aRat.Val.Num(), bRat.Val.Num())
			return NewRationalFromBigRat(new(big.Rat).SetInt(m)), nil
		}
		return NewFunc(name, args)

	case "perm":
		nRat, nOk := args[0].(*RationalNode)
		rRat, rOk := args[1].(*RationalNode)
		if nOk && rOk {
			if !nRat.Val.IsInt() || !rRat.Val.IsInt() {
				return nil, fmt.Errorf("perm domain error: arguments must be integers, got %s and %s", nRat.String(), rRat.String())
			}
			if nRat.Val.Sign() < 0 || rRat.Val.Sign() < 0 {
				return nil, fmt.Errorf("perm domain error: arguments must be non-negative, got %s and %s", nRat.String(), rRat.String())
			}
			nVal := nRat.Val.Num()
			rVal := rRat.Val.Num()
			if rVal.Cmp(nVal) > 0 {
				return mustRational(0, 1), nil
			}
			res := big.NewInt(1)
			curr := new(big.Int).Set(nVal)
			count := rVal.Int64()
			one := big.NewInt(1)
			for i := int64(0); i < count; i++ {
				res.Mul(res, curr)
				curr.Sub(curr, one)
			}
			return NewRationalFromBigRat(new(big.Rat).SetInt(res)), nil
		}
		return NewFunc(name, args)

	case "comb":
		nRat, nOk := args[0].(*RationalNode)
		rRat, rOk := args[1].(*RationalNode)
		if nOk && rOk {
			if !nRat.Val.IsInt() || !rRat.Val.IsInt() {
				return nil, fmt.Errorf("comb domain error: arguments must be integers, got %s and %s", nRat.String(), rRat.String())
			}
			if nRat.Val.Sign() < 0 || rRat.Val.Sign() < 0 {
				return nil, fmt.Errorf("comb domain error: arguments must be non-negative, got %s and %s", nRat.String(), rRat.String())
			}
			nVal := nRat.Val.Num()
			rVal := rRat.Val.Num()
			if rVal.Cmp(nVal) > 0 {
				return mustRational(0, 1), nil
			}
			res := new(big.Int).Binomial(nVal.Int64(), rVal.Int64())
			return NewRationalFromBigRat(new(big.Rat).SetInt(res)), nil
		}
		return NewFunc(name, args)

	case "rand":
		if len(args) == 1 {
			maxRat, ok := args[0].(*RationalNode)
			if !ok || !maxRat.Val.IsInt() {
				return nil, fmt.Errorf("rand domain error: max must be an integer, got %s", args[0].String())
			}
			maxVal := maxRat.Val.Num().Int64()
			if maxVal < 0 {
				return nil, fmt.Errorf("rand domain error: max must be non-negative, got %d", maxVal)
			}
			if maxVal == 0 {
				return mustRational(0, 1), nil
			}
			val := rand.Int64N(maxVal + 1)
			return mustRational(val, 1), nil
		} else if len(args) == 2 {
			minRat, minOk := args[0].(*RationalNode)
			maxRat, maxOk := args[1].(*RationalNode)
			if !minOk || !maxOk || !minRat.Val.IsInt() || !maxRat.Val.IsInt() {
				return nil, fmt.Errorf("rand domain error: min and max must be integers, got %s and %s", args[0].String(), args[1].String())
			}
			minVal := minRat.Val.Num().Int64()
			maxVal := maxRat.Val.Num().Int64()
			if minVal > maxVal {
				return nil, fmt.Errorf("rand domain error: min cannot be greater than max, got %d > %d", minVal, maxVal)
			}
			delta := maxVal - minVal + 1
			val := minVal + rand.Int64N(delta)
			return mustRational(val, 1), nil
		} else if len(args) == 3 {
			seedRat, sOk := args[0].(*RationalNode)
			minRat, minOk := args[1].(*RationalNode)
			maxRat, maxOk := args[2].(*RationalNode)
			if !sOk || !minOk || !maxOk || !seedRat.Val.IsInt() || !minRat.Val.IsInt() || !maxRat.Val.IsInt() {
				return nil, fmt.Errorf("rand domain error: seed, min, and max must be integers")
			}
			seedVal := seedRat.Val.Num().Int64()
			minVal := minRat.Val.Num().Int64()
			maxVal := maxRat.Val.Num().Int64()
			if minVal > maxVal {
				return nil, fmt.Errorf("rand domain error: min cannot be greater than max, got %d > %d", minVal, maxVal)
			}
			delta := maxVal - minVal + 1
			rng := rand.New(rand.NewPCG(uint64(seedVal), 1))
			val := minVal + rng.Int64N(delta)
			return mustRational(val, 1), nil
		}
		return NewFunc(name, args)

	case "arg":
		return evalArg(args[0])

	case "polar":
		return evalPolar(args[0])

	case "polar_exp":
		return evalPolarExp(args[0])

	case "rect":
		return evalRect(args[0], args[1])

	case "factor":
		if len(args) == 1 {
			return Factor(args[0])
		}
		varName := "x"
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("factor error: second argument must be a variable name, got %s", args[1].String())
		}
		return Factor(args[0], varName)

	case "expand":
		return expandNode(args[0]), nil

	case "apart":
		if len(args) == 1 {
			return EvalApart(args[0])
		}
		varName := ""
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("apart error: second argument must be a variable name, got %s", args[1].String())
		}
		return EvalApart(args[0], varName)

	case "together":
		return EvalTogether(args[0])

	case "trig_expand":
		return EvalTrigExpand(args[0], env)

	case "trig_reduce":
		return EvalTrigReduce(args[0], env)

	case "integrate":
		varName := "x"
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("integrate error: second argument must be a variable name, got %s", args[1].String())
		}
		if len(args) == 2 {
			return evalIndefiniteIntegral(args[0], varName)
		}
		return evalDefiniteIntegral(args[0], varName, args[2], args[3])

	case "limit":
		var dirNode Node = nil
		if len(args) == 4 {
			dirNode = args[3]
		}
		return EvalLimit(args[0], args[1], args[2], dirNode)

	case "diff":
		varName := "x"
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("diff error: second argument must be a variable name, got %s", args[1].String())
		}
		if len(args) == 3 {
			r, ok := args[2].(*RationalNode)
			if !ok || !r.Val.IsInt() || r.Val.Sign() < 0 {
				return nil, fmt.Errorf("diff error: third argument must be a non-negative integer, got %s", args[2].String())
			}
			order := r.Val.Num().Int64()
			res := args[0]
			for k := int64(0); k < order; k++ {
				var err error
				res, err = differentiate(res, varName)
				if err != nil {
					return nil, err
				}
			}
			return res, nil
		}
		return differentiate(args[0], varName)

	case "dsolve":
		return evalDSolveSpecial(args, env)

	case "laplace":
		var tName, sName string
		if len(args) >= 2 {
			if v, ok := args[1].(*VarNode); ok {
				tName = v.Name
			} else {
				return nil, fmt.Errorf("laplace error: second argument must be a variable name, got %s", args[1].String())
			}
		}
		if len(args) >= 3 {
			if v, ok := args[2].(*VarNode); ok {
				sName = v.Name
			} else {
				return nil, fmt.Errorf("laplace error: third argument must be a variable name, got %s", args[2].String())
			}
		}
		return EvalLaplace(args[0], tName, sName, env)

	case "inv_laplace":
		var sName, tName string
		if len(args) >= 2 {
			if v, ok := args[1].(*VarNode); ok {
				sName = v.Name
			} else {
				return nil, fmt.Errorf("inv_laplace error: second argument must be a variable name, got %s", args[1].String())
			}
		}
		if len(args) >= 3 {
			if v, ok := args[2].(*VarNode); ok {
				tName = v.Name
			} else {
				return nil, fmt.Errorf("inv_laplace error: third argument must be a variable name, got %s", args[2].String())
			}
		}
		return EvalInvLaplace(args[0], sName, tName, env)

	case "solve":
		varName := "x"
		if v, ok := args[1].(*VarNode); ok {
			varName = v.Name
		} else {
			return nil, fmt.Errorf("solve error: second argument must be a variable name, got %s", args[1].String())
		}
		return solveEquation(args[0], varName)

	case "det":
		mat, ok := args[0].(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("det error: argument must be a matrix, got %s", args[0].String())
		}
		return evalDet(mat)

	case "inv":
		mat, ok := args[0].(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("inv error: argument must be a matrix, got %s", args[0].String())
		}
		return evalInv(mat)

	case "transpose":
		mat, ok := args[0].(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("transpose error: argument must be a matrix, got %s", args[0].String())
		}
		return evalTranspose(mat), nil

	case "rref":
		evaled, err := Eval(args[0])
		if err != nil {
			return nil, err
		}
		mat, ok := evaled.(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("rref error: argument must be a matrix, got %s", args[0].String())
		}
		return evalRREF(mat)

	case "rank":
		evaled, err := Eval(args[0])
		if err != nil {
			return nil, err
		}
		mat, ok := evaled.(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("rank error: argument must be a matrix, got %s", args[0].String())
		}
		return evalRank(mat)

	case "solve_linear", "linsolve":
		return evalSolveLinear(args[0], args[1])

	case "trace", "tr":
		evaled, err := Eval(args[0])
		if err != nil {
			return nil, err
		}
		mat, ok := evaled.(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("trace error: argument must be a matrix, got %s", args[0].String())
		}
		return EvalTrace(mat, env)

	case "eigenvals":
		evaled, err := Eval(args[0])
		if err != nil {
			return nil, err
		}
		mat, ok := evaled.(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("eigenvals error: argument must be a matrix, got %s", args[0].String())
		}
		return EvalEigenvals(mat, env)

	case "eigenvects":
		evaled, err := Eval(args[0])
		if err != nil {
			return nil, err
		}
		mat, ok := evaled.(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("eigenvects error: argument must be a matrix, got %s", args[0].String())
		}
		return EvalEigenvects(mat, env)

	case "poly_gcd":
		var vName string
		if len(args) >= 3 {
			if v, ok := args[2].(*VarNode); ok {
				vName = v.Name
			} else {
				return nil, fmt.Errorf("poly_gcd error: third argument must be a variable name, got %s", args[2].String())
			}
		}
		return EvalPolyGCD(args[0], args[1], vName, env)

	case "poly_lcm":
		var vName string
		if len(args) >= 3 {
			if v, ok := args[2].(*VarNode); ok {
				vName = v.Name
			} else {
				return nil, fmt.Errorf("poly_lcm error: third argument must be a variable name, got %s", args[2].String())
			}
		}
		return EvalPolyLCM(args[0], args[1], vName, env)

	case "resultant":
		var vName string
		if len(args) >= 3 {
			if v, ok := args[2].(*VarNode); ok {
				vName = v.Name
			} else {
				return nil, fmt.Errorf("resultant error: third argument must be a variable name, got %s", args[2].String())
			}
		}
		return EvalResultant(args[0], args[1], vName, env)

	case "fourier_series":
		return EvalFourierSeries(args, env)

	case "cfrac":


		return evalCFrac(args[0])

	case "from_cfrac":
		return evalFromCFrac(args[0])

	case "taylor":
		return evalTaylor(args[0], args[1], args[2], args[3])

	case "sum":
		return evalSum(args[0], args[1], args[2], args[3])

	case "dot":
		return evalDot(args[0], args[1])

	case "cross":
		return evalCross(args[0], args[1])

	case "norm":
		return evalNorm(args[0])

	case "grad":
		return evalGrad(args[0], args[1])

	case "div":
		return evalDiv(args[0], args[1])

	case "curl":
		return evalCurl(args[0], args[1])

	case "line_intersect":
		return evalLineIntersectFunc(args[0], args[1])

	case "circle_intersect":
		return evalCircleIntersectFunc(args[0], args[1], args[2], args[3])

	case "triangle_area":
		return evalTriangleAreaFunc(args[0], args[1], args[2])

	case "triangle_centers":
		return evalTriangleCentersFunc(args[0], args[1], args[2])

	case "binom":
		return EvalBinomPMF(args)

	case "hyper":
		return EvalHyperPMF(args)

	case "geom":
		return EvalGeomPMF(args)

	case "bayes":
		return EvalBayes(args)

	case "expect":
		return EvalExpect(args)

	case "variance":
		return EvalVariance(args)

	case "stddev":
		return EvalStdDev(args)

	case "plot":
		return EvalPlot(args)

	case "inv_mod":
		return EvalInvMod(args[0], args[1])

	case "crt":
		return EvalCRT(args)

	case "totient":
		return EvalTotient(args[0])

	case "is_prime":
		return EvalIsPrime(args[0])

	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

// evalDSolveSpecial extracts arguments for dsolve without evaluating the first argument eagerly.
func evalDSolveSpecial(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("dsolve requires 1 to 3 arguments, got %d", len(args))
	}
	var yName, xName string
	if len(args) >= 2 {
		if vy, ok := args[1].(*VarNode); ok {
			yName = vy.Name
		} else {
			return nil, fmt.Errorf("dsolve error: second argument must be a variable name, got %s", args[1].String())
		}
	}
	if len(args) >= 3 {
		if vx, ok := args[2].(*VarNode); ok {
			xName = vx.Name
		} else {
			return nil, fmt.Errorf("dsolve error: third argument must be a variable name, got %s", args[2].String())
		}
	}
	return EvalDSolve(args[0], yName, xName, env)
}

