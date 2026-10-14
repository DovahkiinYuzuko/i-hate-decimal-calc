package calc

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// TableParallelCutoff is the minimum number of iteration steps to justify spawning worker goroutines.
const TableParallelCutoff = 16

func init() {
	RegisterLazyHandler("table", handleTable)
	RegisterLazyHandler("product", handleProduct)
	RegisterLazyHandler("for", handleFor)
}

func parseIterationBounds(startNode, endNode, stepNode Node, env *Env, fnName string) (*big.Rat, *big.Rat, *big.Rat, error) {
	evalStart, err := EvalWithEnv(startNode, env)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, err))
	}
	rStart, okStart := evalStart.(*RationalNode)
	if !okStart {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, "start bound must evaluate to rational, got "+evalStart.String()))
	}

	evalEnd, err := EvalWithEnv(endNode, env)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, err))
	}
	rEnd, okEnd := evalEnd.(*RationalNode)
	if !okEnd {
		return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, "end bound must evaluate to rational, got "+evalEnd.String()))
	}

	step := big.NewRat(1, 1)
	if stepNode != nil {
		evalStep, err := EvalWithEnv(stepNode, env)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, err))
		}
		rStep, okStep := evalStep.(*RationalNode)
		if !okStep {
			return nil, nil, nil, fmt.Errorf("%s", i18n.T("errors.iteration_bounds_eval", fnName, "step must evaluate to rational, got "+evalStep.String()))
		}
		step = new(big.Rat).Set(rStep.Val)
	}

	return new(big.Rat).Set(rStart.Val), new(big.Rat).Set(rEnd.Val), step, nil
}

func isPureExpression(node Node) bool {
	if node == nil {
		return true
	}
	isPure := true
	ast.Walk(node, func(n ast.Node) bool {
		if rel, ok := n.(*ast.RelOpNode); ok && rel.Op == "=" {
			isPure = false
			return false
		}
		if fn, ok := n.(*ast.FuncNode); ok {
			switch fn.Name {
			case "assume", "unassume", "clear_assumptions", "for":
				isPure = false
				return false
			}
		}
		return true
	})
	return isPure
}

func computeStepCount(start, end, step *big.Rat) *big.Int {
	if step.Sign() == 0 {
		return big.NewInt(0)
	}
	diff := new(big.Rat).Sub(end, start)
	if (step.Sign() > 0 && diff.Sign() < 0) || (step.Sign() < 0 && diff.Sign() > 0) {
		return big.NewInt(0)
	}
	quot := new(big.Rat).Quo(diff, step)
	intDiv := new(big.Int).Quo(quot.Num(), quot.Denom())
	return new(big.Int).Add(intDiv, big.NewInt(1))
}

func generateStepValues(start, step *big.Rat, count int) []*big.Rat {
	values := make([]*big.Rat, count)
	if start.IsInt() && step.IsInt() && start.Num().IsInt64() && step.Num().IsInt64() {
		start64 := start.Num().Int64()
		step64 := step.Num().Int64()
		for i := 0; i < count; i++ {
			values[i] = big.NewRat(start64+int64(i)*step64, 1)
		}
		return values
	}

	curr := new(big.Rat).Set(start)
	for i := 0; i < count; i++ {
		values[i] = new(big.Rat).Set(curr)
		curr.Add(curr, step)
	}
	return values
}

func evaluateTableParallel(expr Node, varName string, currValues []*big.Rat, baseEnv *Env) ([]Node, error) {
	n := len(currValues)
	workers := GetOptimalWorkers()
	if workers <= 1 || n < TableParallelCutoff {
		results := make([]Node, n)
		workEnv := baseEnv.Clone()
		for i, curr := range currValues {
			workEnv.Set(varName, &RationalNode{Val: curr})
			val, err := EvalWithEnv(expr, workEnv)
			if err != nil {
				return nil, err
			}
			results[i] = val
		}
		return results, nil
	}

	results := make([]Node, n)
	var firstErr error
	var errOnce sync.Once
	var wg sync.WaitGroup

	chunkSize := (n + workers - 1) / workers

	for w := 0; w < workers; w++ {
		startIdx := w * chunkSize
		if startIdx >= n {
			break
		}
		endIdx := startIdx + chunkSize
		if endIdx > n {
			endIdx = n
		}

		wg.Add(1)
		go func(sIdx, eIdx int) {
			defer wg.Done()
			wEnv := baseEnv.Clone()
			for i := sIdx; i < eIdx; i++ {
				if firstErr != nil {
					return
				}
				wEnv.Set(varName, &RationalNode{Val: currValues[i]})
				val, err := EvalWithEnv(expr, wEnv)
				if err != nil {
					errOnce.Do(func() {
						firstErr = err
					})
					return
				}
				results[i] = val
			}
		}(startIdx, endIdx)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func handleTable(args []Node, env *Env) (Node, error) {
	if len(args) < 4 || len(args) > 5 {
		return nil, fmt.Errorf("table requires 4 or 5 arguments (expr, var, start, end, [step]), got %d", len(args))
	}

	expr := args[0]
	v, ok := args[1].(*VarNode)
	if !ok {
		if _, isComp := args[1].(*ComplexNode); isComp {
			return nil, fmt.Errorf("%s (note: 'i' is the imaginary unit, please use 'k', 'j', 'n', etc.)", i18n.T("errors.iteration_arg_must_be_var", "table", args[1].String()))
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.iteration_arg_must_be_var", "table", args[1].String()))
	}

	var stepNode Node
	if len(args) == 5 {
		stepNode = args[4]
	}

	start, end, step, err := parseIterationBounds(args[2], args[3], stepNode, env, "table")
	if err != nil {
		return nil, err
	}

	if step.Sign() == 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.iteration_step_zero", "table"))
	}

	stepCountBig := computeStepCount(start, end, step)
	if stepCountBig.Sign() == 0 {
		return NewList([]Node{}), nil
	}

	warnLimit, maxLimit, warnWriter := env.GetIterationLimits()
	if maxLimit != nil && maxLimit.Sign() > 0 && stepCountBig.Cmp(maxLimit) > 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.iteration_max_exceeded", "table", maxLimit.String()))
	}
	if warnLimit != nil && warnLimit.Sign() > 0 && stepCountBig.Cmp(warnLimit) > 0 && warnWriter != nil {
		fmt.Fprintf(warnWriter, "[WARN] %s\n", i18n.T("warnings.iteration_limit_exceeded", "table", stepCountBig.String(), warnLimit.String()))
	}

	// Check if expression is pure and candidate for parallel evaluation
	if stepCountBig.IsInt64() {
		count := int(stepCountBig.Int64())
		if count >= TableParallelCutoff && GetOptimalWorkers() > 1 && isPureExpression(expr) {
			values := generateStepValues(start, step, count)
			results, err := evaluateTableParallel(expr, v.Name, values, env)
			if err != nil {
				return nil, err
			}
			return NewList(results), nil
		}
	}

	elements := make([]Node, 0)
	fsm := NewIterationFSM(v.Name, start, end, step, env)
	fsm.SetLimits(warnLimit, maxLimit, warnWriter)
	err = fsm.Run(func(curr *big.Rat) (bool, error) {
		val, err := EvalWithEnv(expr, env)
		if err != nil {
			return false, err
		}
		elements = append(elements, val)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return NewList(elements), nil
}


func handleProduct(args []Node, env *Env) (Node, error) {
	if len(args) < 4 || len(args) > 5 {
		return nil, fmt.Errorf("product requires 4 or 5 arguments (expr, var, start, end, [step]), got %d", len(args))
	}

	expr := args[0]
	v, ok := args[1].(*VarNode)
	if !ok {
		if _, isComp := args[1].(*ComplexNode); isComp {
			return nil, fmt.Errorf("%s (note: 'i' is the imaginary unit, please use 'k', 'j', 'n', etc.)", i18n.T("errors.iteration_arg_must_be_var", "product", args[1].String()))
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.iteration_arg_must_be_var", "product", args[1].String()))
	}

	var stepNode Node
	if len(args) == 5 {
		stepNode = args[4]
	}

	start, end, step, err := parseIterationBounds(args[2], args[3], stepNode, env, "product")
	if err != nil {
		return nil, err
	}

	terms := make([]Node, 0)
	hasZero := false
	fsm := NewIterationFSM(v.Name, start, end, step, env)
	fsm.SetLimits(env.GetIterationLimits())
	err = fsm.Run(func(curr *big.Rat) (bool, error) {
		val, err := EvalWithEnv(expr, env)
		if err != nil {
			return false, err
		}

		if r, ok := val.(*RationalNode); ok && r.Val.Sign() == 0 {
			hasZero = true
			return true, nil
		}

		terms = append(terms, val)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	if hasZero {
		return mustRational(0, 1), nil
	}
	if len(terms) == 0 {
		return mustRational(1, 1), nil
	}

	return combineProductTerms(terms)
}

func binarySplittingRationalProduct(rats []*RationalNode) Node {
	if len(rats) == 0 {
		return mustRational(1, 1)
	}
	if len(rats) == 1 {
		return rats[0]
	}

	nums := make([]*big.Int, len(rats))
	dens := make([]*big.Int, len(rats))
	for i, r := range rats {
		nums[i] = new(big.Int).Set(r.Val.Num())
		dens[i] = new(big.Int).Set(r.Val.Denom())
	}

	numProd := binarySplittingProduct(nums)
	denProd := binarySplittingProduct(dens)

	res := new(big.Rat).SetFrac(numProd, denProd)
	return &RationalNode{Val: res}
}

func binarySplittingNodeProduct(nodes []Node) (Node, error) {
	if len(nodes) == 0 {
		return mustRational(1, 1), nil
	}
	if len(nodes) == 1 {
		return nodes[0], nil
	}
	mid := len(nodes) / 2
	left, err := binarySplittingNodeProduct(nodes[:mid])
	if err != nil {
		return nil, err
	}
	right, err := binarySplittingNodeProduct(nodes[mid:])
	if err != nil {
		return nil, err
	}
	res, err := simplifyMul([]Node{left, right})
	if err != nil {
		return NewMul([]Node{left, right}), nil
	}
	return res, nil
}

func combineProductTerms(terms []Node) (Node, error) {
	if len(terms) == 0 {
		return mustRational(1, 1), nil
	}

	var ratTerms []*RationalNode
	var otherTerms []Node

	for _, t := range terms {
		if r, ok := t.(*RationalNode); ok {
			ratTerms = append(ratTerms, r)
		} else {
			otherTerms = append(otherTerms, t)
		}
	}

	var ratNode Node
	if len(ratTerms) > 0 {
		ratNode = binarySplittingRationalProduct(ratTerms)
		if r, ok := ratNode.(*RationalNode); ok && r.Val.Sign() == 0 {
			return mustRational(0, 1), nil
		}
	}

	if len(otherTerms) == 0 {
		if ratNode != nil {
			return ratNode, nil
		}
		return mustRational(1, 1), nil
	}

	otherNode, err := binarySplittingNodeProduct(otherTerms)
	if err != nil {
		return nil, err
	}

	if ratNode == nil {
		return otherNode, nil
	}

	if r, ok := ratNode.(*RationalNode); ok && r.Val.Cmp(big.NewRat(1, 1)) == 0 {
		return otherNode, nil
	}

	res, err := simplifyMul([]Node{ratNode, otherNode})
	if err != nil {
		return NewMul([]Node{ratNode, otherNode}), nil
	}
	return res, nil
}


func handleFor(args []Node, env *Env) (Node, error) {
	if len(args) < 4 || len(args) > 5 {
		return nil, fmt.Errorf("for requires 4 or 5 arguments (var, start, end, [step], body), got %d", len(args))
	}

	v, ok := args[0].(*VarNode)
	if !ok {
		if _, isComp := args[0].(*ComplexNode); isComp {
			return nil, fmt.Errorf("%s (note: 'i' is the imaginary unit, please use 'k', 'j', 'n', etc.)", i18n.T("errors.iteration_arg_must_be_var", "for", args[0].String()))
		}
		return nil, fmt.Errorf("%s", i18n.T("errors.iteration_arg_must_be_var", "for", args[0].String()))
	}

	var stepNode Node
	var bodyNode Node
	if len(args) == 4 {
		stepNode = nil
		bodyNode = args[3]
	} else {
		stepNode = args[3]
		bodyNode = args[4]
	}

	start, end, step, err := parseIterationBounds(args[1], args[2], stepNode, env, "for")
	if err != nil {
		return nil, err
	}

	lastRes := Node(mustRational(0, 1))
	fsm := NewIterationFSM(v.Name, start, end, step, env)
	fsm.SetLimits(env.GetIterationLimits())
	err = fsm.Run(func(curr *big.Rat) (bool, error) {
		val, err := EvalWithEnv(bodyNode, env)
		if err != nil {
			return false, err
		}
		lastRes = val
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return lastRes, nil
}
