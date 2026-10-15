package calc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/units"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	RegisterHandler("range", handleRange)
	RegisterLazyHandler("map", handleMap)
	RegisterLazyHandler("filter", handleFilter)
	RegisterLazyHandler("lambda", handleLambda)
}

// evalInteger evaluates a node and extracts an int64 integer. Returns error if non-integer.
func evalInteger(node Node, env *Env) (int64, error) {
	evaled, err := EvalWithEnv(node, env)
	if err != nil {
		return 0, err
	}
	rat, ok := evaled.(*RationalNode)
	if !ok || !rat.Val.IsInt() || !rat.Val.Num().IsInt64() {
		return 0, fmt.Errorf("%s", i18n.T("errors.index_not_integer", evaled.String()))
	}
	return rat.Val.Num().Int64(), nil
}

// sliceGetIndices implements CPython PySlice_GetIndicesEx exact normalization semantics.
func sliceGetIndices(length int64, startNode, stopNode, stepNode Node, env *Env) (start, stop, step int64, sliceLen int64, err error) {
	step = 1
	if stepNode != nil {
		s, err := evalInteger(stepNode, env)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if s == 0 {
			return 0, 0, 0, 0, fmt.Errorf("%s", i18n.T("errors.slice_step_zero"))
		}
		step = s
	}

	if step > 0 {
		// Forward slicing
		if startNode == nil {
			start = 0
		} else {
			st, err := evalInteger(startNode, env)
			if err != nil {
				return 0, 0, 0, 0, err
			}
			if st < 0 {
				start = st + length
				if start < 0 {
					start = 0
				}
			} else {
				start = st
				if start > length {
					start = length
				}
			}
		}

		if stopNode == nil {
			stop = length
		} else {
			sp, err := evalInteger(stopNode, env)
			if err != nil {
				return 0, 0, 0, 0, err
			}
			if sp < 0 {
				stop = sp + length
				if stop < 0 {
					stop = 0
				}
			} else {
				stop = sp
				if stop > length {
					stop = length
				}
			}
		}

		if start >= stop {
			sliceLen = 0
		} else {
			sliceLen = (stop - start - 1) / step + 1
		}
	} else {
		// Backward slicing (step < 0)
		if startNode == nil {
			start = length - 1
		} else {
			st, err := evalInteger(startNode, env)
			if err != nil {
				return 0, 0, 0, 0, err
			}
			if st < 0 {
				start = st + length
				if start < -1 {
					start = -1
				}
			} else {
				start = st
				if start >= length {
					start = length - 1
				}
			}
		}

		if stopNode == nil {
			stop = -1
		} else {
			sp, err := evalInteger(stopNode, env)
			if err != nil {
				return 0, 0, 0, 0, err
			}
			if sp < 0 {
				stop = sp + length
				if stop < -1 {
					stop = -1
				}
			} else {
				stop = sp
				if stop >= length {
					stop = length - 1
				}
			}
		}

		if start <= stop {
			sliceLen = 0
		} else {
			sliceLen = (start - stop - 1) / (-step) + 1
		}
	}

	return start, stop, step, sliceLen, nil
}

// evalSlice extracts a subsequence from a ListNode or MatrixNode according to CPython slice semantics.
func evalSlice(target Node, startNode, stopNode, stepNode Node, env *Env) (Node, error) {
	evalTarget, err := EvalWithEnv(target, env)
	if err != nil {
		return nil, err
	}

	switch col := evalTarget.(type) {
	case *ListNode:
		n := int64(len(col.Elements))
		start, _, step, sliceLen, err := sliceGetIndices(n, startNode, stopNode, stepNode, env)
		if err != nil {
			return nil, err
		}

		results := make([]Node, 0, sliceLen)
		idxRecord := make([]string, 0, sliceLen)
		curr := start
		for i := int64(0); i < sliceLen; i++ {
			results = append(results, col.Elements[curr])
			idxRecord = append(idxRecord, fmt.Sprintf("%d", curr))
			curr += step
		}

		outList := ast.NewList(results)

		// Record --explain step
		startStr := "nil"
		if startNode != nil {
			startStr = startNode.String()
		}
		stopStr := "nil"
		if stopNode != nil {
			stopStr = stopNode.String()
		}
		stepStr := "1"
		if stepNode != nil {
			stepStr = stepNode.String()
		}
		note := i18n.T("explain.slice_step_detail", n, startStr, stopStr, stepStr, strings.Join(idxRecord, ", "))
		RecordTraceRewrite(RuleSlice, &SliceNode{Target: evalTarget, Start: startNode, Stop: stopNode, Step: stepNode}, outList, note)

		return outList, nil

	case *MatrixNode:
		// Row slice of a 2D matrix
		n := int64(col.Rows)
		start, _, step, sliceLen, err := sliceGetIndices(n, startNode, stopNode, stepNode, env)
		if err != nil {
			return nil, err
		}
		if sliceLen == 0 {
			return ast.NewList([]Node{}), nil
		}
		newRows := make([][]Node, sliceLen)
		curr := start
		for i := int64(0); i < sliceLen; i++ {
			rowCopy := make([]Node, col.Cols)
			copy(rowCopy, col.Data[curr])
			newRows[i] = rowCopy
			curr += step
		}
		return ast.NewMatrix(int(sliceLen), col.Cols, newRows)

	default:
		return nil, fmt.Errorf("%s", i18n.T("errors.slice_not_applicable", evalTarget.String()))
	}
}

// evalIndex retrieves a single element by 0-based or negative index from a collection.
func evalIndex(target Node, indexNode Node, env *Env) (Node, error) {
	evalTarget, err := EvalWithEnv(target, env)
	if err != nil {
		return nil, err
	}

	switch col := evalTarget.(type) {
	case *ListNode:
		idxVal, err := evalInteger(indexNode, env)
		if err != nil {
			return nil, err
		}
		n := int64(len(col.Elements))
		resolvedIdx := idxVal
		if resolvedIdx < 0 {
			resolvedIdx += n
		}
		if resolvedIdx < 0 || resolvedIdx >= n {
			return nil, fmt.Errorf("%s", i18n.T("errors.index_out_of_range", indexNode.String(), n))
		}
		elem := col.Elements[resolvedIdx]
		RecordTraceRewrite(RuleSlice, &IndexNode{Target: evalTarget, Index: indexNode}, elem, fmt.Sprintf("index %s -> %s", indexNode.String(), elem.String()))
		return elem, nil

	case *MatrixNode:
		idxVal, err := evalInteger(indexNode, env)
		if err != nil {
			return nil, err
		}
		// Matrix indexing M[r] returns row vector as ListNode
		n := int64(col.Rows)
		resolvedIdx := idxVal
		if resolvedIdx < 0 {
			resolvedIdx += n
		}
		if resolvedIdx < 0 || resolvedIdx >= n {
			return nil, fmt.Errorf("%s", i18n.T("errors.index_out_of_range", indexNode.String(), n))
		}
		rowCopy := make([]Node, col.Cols)
		copy(rowCopy, col.Data[resolvedIdx])
		return ast.NewList(rowCopy), nil

	default:
		// Not a collection: Resolve as physical unit assignment (e.g. 10 [m], 5 [kg/s])
		fsm := units.NewQuantityFSM()
		scale, dims, unitErr := fsm.ResolveUnitExprWithFSM(indexNode)
		if unitErr == nil {
			return ast.NewQuantityNode(evalTarget, scale, dims, indexNode.String()), nil
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.index_not_applicable", evalTarget.String()))
	}
}

// handleRange creates an arithmetic sequence [start, start+step, ... < stop) with exact rational numbers.
func handleRange(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("%s", i18n.T("func_args_between", "range", 1, 3, len(args)))
	}

	var startNode, stopNode, stepNode Node
	switch len(args) {
	case 1:
		startNode = mustRational(0, 1)
		stopNode = args[0]
		stepNode = mustRational(1, 1)
	case 2:
		startNode = args[0]
		stopNode = args[1]
		stepNode = mustRational(1, 1)
	case 3:
		startNode = args[0]
		stopNode = args[1]
		stepNode = args[2]
	}

	evalStart, err := EvalWithEnv(startNode, env)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.range_bounds_eval", err))
	}
	rStart, okStart := evalStart.(*RationalNode)
	if !okStart {
		return nil, fmt.Errorf("%s", i18n.T("errors.range_bound_not_rational", "start", evalStart.String()))
	}

	evalStop, err := EvalWithEnv(stopNode, env)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.range_bounds_eval", err))
	}
	rStop, okStop := evalStop.(*RationalNode)
	if !okStop {
		return nil, fmt.Errorf("%s", i18n.T("errors.range_bound_not_rational", "stop", evalStop.String()))
	}

	evalStep, err := EvalWithEnv(stepNode, env)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("errors.range_bounds_eval", err))
	}
	rStep, okStep := evalStep.(*RationalNode)
	if !okStep {
		return nil, fmt.Errorf("%s", i18n.T("errors.range_bound_not_rational", "step", evalStep.String()))
	}

	startRat := rStart.Val
	stopRat := rStop.Val
	stepRat := rStep.Val

	if stepRat.Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.range_step_zero"))
	}

	// Calculate sequence count
	// diff = stop - start
	diff := new(big.Rat).Sub(stopRat, startRat)
	var count int64
	if (stepRat.Sign() > 0 && diff.Sign() <= 0) || (stepRat.Sign() < 0 && diff.Sign() >= 0) {
		count = 0
	} else {
		// quotient = diff / step
		quot := new(big.Rat).Quo(diff, stepRat)
		// For half-open [start, stop):
		// count is ceil(quot) if not exact int, or if exact int, exactly ceil of integer part minus 1 if endpoint is excluded?
		// Note: if start=0, stop=10, step=2 => quot=5 => count=5 (0, 2, 4, 6, 8)
		// if start=0, stop=5, step=2 => quot=2.5 => count=3 (0, 2, 4)
		intPart := new(big.Int).Quo(quot.Num(), quot.Denom())
		rem := new(big.Int).Rem(quot.Num(), quot.Denom())
		if rem.Sign() != 0 {
			intPart.Add(intPart, big.NewInt(1))
		}
		if !intPart.IsInt64() {
			return nil, fmt.Errorf("%s", i18n.T("errors.iteration_max_exceeded", "range", intPart.String()))
		}
		count = intPart.Int64()
	}

	warnLimit, maxLimit, warnWriter := env.GetIterationLimits()
	stepBig := big.NewInt(count)
	if maxLimit != nil && maxLimit.Sign() > 0 && stepBig.Cmp(maxLimit) > 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.iteration_max_exceeded", "range", maxLimit.String()))
	}
	if warnLimit != nil && warnLimit.Sign() > 0 && stepBig.Cmp(warnLimit) > 0 {
		fmt.Fprintf(warnWriter, "[WARN] %s\n", i18n.T("warnings.iteration_limit_exceeded", "range", stepBig.String(), warnLimit.String()))
	}

	results := make([]Node, count)
	curr := new(big.Rat).Set(startRat)
	for i := int64(0); i < count; i++ {
		results[i] = &RationalNode{Val: new(big.Rat).Set(curr)}
		curr.Add(curr, stepRat)
	}

	outList := ast.NewList(results)
	note := i18n.T("explain.range_step_detail", startRat.RatString(), stopRat.RatString(), stepRat.RatString(), count)
	RecordTraceRewrite(RuleRange, &FuncNode{Name: "range", Args: args}, outList, note)

	return outList, nil
}

// applyCallable applies a function identifier, LambdaNode, or built-in to a single argument safely.
func applyCallable(fnNode Node, argVal Node, env *Env) (Node, error) {
	return applyCallableWithArgs(fnNode, []Node{argVal}, env)
}

// applyCallableWithArgs applies a callable (LambdaNode, VarNode, FuncNode) to multiple arguments.
func applyCallableWithArgs(fnNode Node, args []Node, env *Env) (Node, error) {
	switch f := fnNode.(type) {
	case *LambdaNode:
		if len(f.Params) != len(args) {
			return nil, fmt.Errorf("%s", i18n.T("errors.lambda_param_count_mismatch", len(f.Params), len(args)))
		}
		if len(f.Params) == 1 {
			var result Node
			err := env.WithScopedVar(f.Params[0], args[0], func(setVar func(Node)) error {
				res, e := EvalWithEnv(f.Body, env)
				if e != nil {
					return e
				}
				result = res
				return nil
			})
			if err != nil {
				return nil, err
			}
			return result, nil
		}
		workEnv := env.Clone()
		for pIdx, pName := range f.Params {
			workEnv.Set(pName, args[pIdx])
		}
		return EvalWithEnv(f.Body, workEnv)

	case *VarNode:
		if bound, ok := env.Get(f.Name); ok {
			if lambda, okL := bound.(*LambdaNode); okL {
				return applyCallableWithArgs(lambda, args, env)
			}
		}
		return simplifyFuncWithEnv(f.Name, args, env)

	case *FuncNode:
		if f.Name == "lambda" {
			lambda, err := handleLambda(f.Args, env)
			if err != nil {
				return nil, err
			}
			return applyCallableWithArgs(lambda, args, env)
		}
		return simplifyFuncWithEnv(f.Name, args, env)

	default:
		return nil, fmt.Errorf("%s", i18n.T("errors.callable_required", "apply", fnNode.String()))
	}
}

// handleLambda creates or returns a first-class LambdaNode.
func handleLambda(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("errors.lambda_param_count_mismatch", 2, len(args)))
	}
	var params []string
	if list, ok := args[0].(*ListNode); ok {
		for _, elem := range list.Elements {
			if v, okV := elem.(*VarNode); okV {
				params = append(params, v.Name)
			} else {
				return nil, fmt.Errorf("%s", i18n.T("errors.lambda_param_must_be_ident", elem.String()))
			}
		}
	} else if v, okV := args[0].(*VarNode); okV {
		params = append(params, v.Name)
	} else {
		return nil, fmt.Errorf("%s", i18n.T("errors.lambda_param_must_be_ident", args[0].String()))
	}
	return ast.NewLambdaNode(params, args[1]), nil
}

// handleMap applies a function or lambda to every element of a list.
func handleMap(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("func_args_between", "map", 2, 2, len(args)))
	}

	evalList, err := EvalWithEnv(args[1], env)
	if err != nil {
		return nil, err
	}
	listNode, ok := evalList.(*ListNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.map_arg_not_list", evalList.String()))
	}

	results := make([]Node, len(listNode.Elements))
	for i, elem := range listNode.Elements {
		res, err := applyCallable(args[0], elem, env)
		if err != nil {
			return nil, err
		}
		results[i] = res

		// Record --explain per-element
		detail := i18n.T("explain.map_step_detail", i, args[0].String(), elem.String(), res.String())
		RecordTraceRewrite(RuleMap, elem, res, detail)
	}

	outList := ast.NewList(results)
	return outList, nil
}

// isTruthy determines the algebraic/boolean truthiness of an evaluated node.
func isTruthy(node Node, env *Env) (bool, error) {
	if node == nil {
		return false, nil
	}
	switch v := node.(type) {
	case *VarNode:
		if v.Name == "true" {
			return true, nil
		}
		if v.Name == "false" {
			return false, nil
		}
		// If variable name is not a boolean literal, check environment
		if bound, ok := env.Get(v.Name); ok {
			return isTruthy(bound, env)
		}
		return false, nil

	case *RationalNode:
		return v.Val.Sign() != 0, nil

	case *RelOpNode:
		if v.Op == "==" || v.Op == "!=" {
			eqRes, err := EvaluateRelOpEquality(v, env)
			if err != nil {
				return false, err
			}
			if varNode, ok := eqRes.(*VarNode); ok {
				return varNode.Name == "true", nil
			}
		}
		if v.Op == "<" || v.Op == "<=" || v.Op == ">" || v.Op == ">=" {
			if res, decided := EvaluateRelOpWithInterval(v); decided {
				return res, nil
			}
		}
		return false, nil

	default:
		return false, nil
	}
}

// handleFilter extracts elements from a list satisfying a predicate function or lambda.
func handleFilter(args []Node, env *Env) (Node, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("%s", i18n.T("func_args_between", "filter", 2, 2, len(args)))
	}

	evalList, err := EvalWithEnv(args[1], env)
	if err != nil {
		return nil, err
	}
	listNode, ok := evalList.(*ListNode)
	if !ok {
		return nil, fmt.Errorf("%s", i18n.T("errors.filter_arg_not_list", evalList.String()))
	}

	results := make([]Node, 0, len(listNode.Elements))
	for i, elem := range listNode.Elements {
		cond, err := applyCallable(args[0], elem, env)
		if err != nil {
			return nil, err
		}
		truthy, err := isTruthy(cond, env)
		if err != nil {
			return nil, err
		}

		statusStr := "excluded"
		if truthy {
			statusStr = "included"
			results = append(results, elem)
		}

		detail := i18n.T("explain.filter_step_detail", i, args[0].String(), elem.String(), cond.String(), statusStr)
		RecordTraceRewrite(RuleFilter, elem, cond, detail)
	}

	outList := ast.NewList(results)
	return outList, nil
}
