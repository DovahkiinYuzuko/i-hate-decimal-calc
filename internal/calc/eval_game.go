package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/lp"
)

// EvalMex computes the Minimum EXcluded value (mex) of a set of non-negative integers.
// By the Pigeonhole Principle, for any collection of N elements, mex <= N.
// Any element > N cannot affect the mex and can be safely ignored.
// Time Complexity: O(N), Space Complexity: O(N).
// Supported calls:
// - mex([0, 1, 2, 4]) -> 3
// - mex(0, 1, 3) -> 2
// - mex([]) -> 0
func EvalMex(args []Node) (Node, error) {
	var elements []Node
	if len(args) == 1 {
		if list, ok := args[0].(*ListNode); ok {
			elements = list.Elements
		} else {
			elements = args
		}
	} else {
		elements = args
	}

	n := len(elements)
	seen := make([]bool, n+1)
	for _, elem := range elements {
		val, err := extractNonNegativeInt(elem, "mex")
		if err != nil {
			return nil, fmt.Errorf("%s", i18n.T("game.err_mex_non_negative_int", elem.String()))
		}
		if val.IsInt64() {
			v := val.Int64()
			if v >= 0 && v <= int64(n) {
				seen[v] = true
			}
		}
	}

	for m := 0; m <= n; m++ {
		if !seen[m] {
			return mustRational(int64(m), 1), nil
		}
	}

	return mustRational(int64(n+1), 1), nil
}

// EvalNimSum computes the bitwise XOR sum (Nim-sum) of non-negative integers.
// Supported calls:
// - nim_sum(1, 2, 3) -> 0
// - nim_sum([3, 4, 5]) -> 2
// - nim_sum(0) -> 0
func EvalNimSum(args []Node) (Node, error) {
	var elements []Node
	if len(args) == 1 {
		if list, ok := args[0].(*ListNode); ok {
			elements = list.Elements
		} else {
			elements = args
		}
	} else {
		elements = args
	}

	res := big.NewInt(0)
	for _, elem := range elements {
		val, err := extractNonNegativeInt(elem, "nim_sum")
		if err != nil {
			return nil, fmt.Errorf("%s", i18n.T("game.err_nim_sum_non_negative_int", elem.String()))
		}
		res.Xor(res, val)
	}

	return NewRationalFromBigRat(new(big.Rat).SetInt(res)), nil
}

// EvalNashZeroSum computes the exact minimax value and optimal mixed strategies
// for a two-player zero-sum game with payoff matrix A (m x n) using rational Linear Programming.
// Player 1 maximizes payoff, Player 2 minimizes payoff.
// Output: [value, x_strategy, y_strategy] where x in Q^m, y in Q^n are mixed strategies.
func EvalNashZeroSum(args []Node) (Node, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s", i18n.T("game.err_nash_args", len(args)))
	}

	var mat *MatrixNode
	switch v := args[0].(type) {
	case *MatrixNode:
		mat = v
	default:
		return nil, fmt.Errorf("%s", i18n.T("game.err_nash_matrix_required", args[0].String()))
	}

	m := mat.Rows
	n := mat.Cols
	if m == 0 || n == 0 || len(mat.Data) != m {
		return nil, fmt.Errorf("%s", i18n.T("game.err_nash_empty_matrix"))
	}
	for r := 0; r < m; r++ {
		if len(mat.Data[r]) != n {
			return nil, fmt.Errorf("%s", i18n.T("game.err_nash_empty_matrix"))
		}
	}

	// Extract rational entries and find minimum entry to determine shift constant c
	A := make([][]*big.Rat, m)
	var minVal *big.Rat
	for r := 0; r < m; r++ {
		A[r] = make([]*big.Rat, n)
		for c := 0; c < n; c++ {
			evaled, err := Eval(mat.Data[r][c])
			if err != nil {
				return nil, err
			}
			rNode, ok := evaled.(*RationalNode)
			if !ok {
				return nil, fmt.Errorf("%s", i18n.T("game.err_nash_rational_required", r, c))
			}
			val := new(big.Rat).Set(rNode.Val)
			A[r][c] = val
			if minVal == nil || val.Cmp(minVal) < 0 {
				minVal = new(big.Rat).Set(val)
			}
		}
	}

	// Shift constant c: ensure all A'[r][c] = A[r][c] + c > 0
	// If minVal <= 0, c = -minVal + 1; else c = 1
	cShift := big.NewRat(1, 1)
	if minVal.Sign() <= 0 {
		cShift.Neg(minVal)
		cShift.Add(cShift, big.NewRat(1, 1))
	}

	Aprime := make([][]*big.Rat, m)
	for r := 0; r < m; r++ {
		Aprime[r] = make([]*big.Rat, n)
		for c := 0; c < n; c++ {
			Aprime[r][c] = new(big.Rat).Add(A[r][c], cShift)
		}
	}

	// LP for Player 1:
	// Variables: u_0, ..., u_{m-1} >= 0
	// Minimize sum_{r=0}^{m-1} u_r
	// Subject to: sum_{r=0}^{m-1} u_r * Aprime[r][c] >= 1   for each c = 0..n-1
	p1VarNames := make([]string, m)
	p1Obj := make([]*big.Rat, m)
	for r := 0; r < m; r++ {
		p1VarNames[r] = fmt.Sprintf("u_%d", r)
		p1Obj[r] = big.NewRat(1, 1)
	}

	p1Cons := make([]lp.Constraint, n)
	for c := 0; c < n; c++ {
		coeffs := make([]*big.Rat, m)
		for r := 0; r < m; r++ {
			coeffs[r] = new(big.Rat).Set(Aprime[r][c])
		}
		p1Cons[c] = lp.Constraint{
			Name:   fmt.Sprintf("con_c%d", c),
			Coeffs: coeffs,
			Sense:  lp.RelGe,
			RHS:    big.NewRat(1, 1),
		}
	}

	p1Prob := &lp.Problem{
		Sense:       lp.OptMinimize,
		Objective:   p1Obj,
		VarNames:    p1VarNames,
		Constraints: p1Cons,
	}

	p1Res, err := lp.Solve(p1Prob)
	if err != nil || p1Res == nil || p1Res.Status != lp.StatusOptimal || p1Res.OptimalValue == nil || p1Res.OptimalValue.Sign() <= 0 || p1Res.PrimalSolution == nil {
		return nil, fmt.Errorf("%s", i18n.T("game.err_nash_solve_failed", err))
	}

	// Optimal value sum(u_r) = 1 / v'
	sumU := p1Res.OptimalValue

	vPrime := new(big.Rat).Inv(sumU)
	vGame := new(big.Rat).Sub(vPrime, cShift)

	// Player 1 optimal strategy: x_r = u_r * v'
	stratP1Nodes := make([]Node, m)
	for r := 0; r < m; r++ {
		uVal := p1Res.PrimalSolution[p1VarNames[r]]
		if uVal == nil {
			uVal = big.NewRat(0, 1)
		}
		xRat := new(big.Rat).Mul(uVal, vPrime)
		stratP1Nodes[r] = NewRationalFromBigRat(xRat)
	}

	// LP for Player 2:
	// Variables: w_0, ..., w_{n-1} >= 0
	// Maximize sum_{c=0}^{n-1} w_c
	// Subject to: sum_{c=0}^{n-1} Aprime[r][c] * w_c <= 1   for each r = 0..m-1
	p2VarNames := make([]string, n)
	p2Obj := make([]*big.Rat, n)
	for c := 0; c < n; c++ {
		p2VarNames[c] = fmt.Sprintf("w_%d", c)
		p2Obj[c] = big.NewRat(1, 1)
	}

	p2Cons := make([]lp.Constraint, m)
	for r := 0; r < m; r++ {
		coeffs := make([]*big.Rat, n)
		for c := 0; c < n; c++ {
			coeffs[c] = new(big.Rat).Set(Aprime[r][c])
		}
		p2Cons[r] = lp.Constraint{
			Name:   fmt.Sprintf("con_r%d", r),
			Coeffs: coeffs,
			Sense:  lp.RelLe,
			RHS:    big.NewRat(1, 1),
		}
	}

	p2Prob := &lp.Problem{
		Sense:       lp.OptMaximize,
		Objective:   p2Obj,
		VarNames:    p2VarNames,
		Constraints: p2Cons,
	}

	p2Res, err := lp.Solve(p2Prob)
	if err != nil || p2Res == nil || p2Res.Status != lp.StatusOptimal || p2Res.OptimalValue == nil || p2Res.OptimalValue.Sign() <= 0 || p2Res.PrimalSolution == nil {
		return nil, fmt.Errorf("%s", i18n.T("game.err_nash_solve_failed", err))
	}

	sumW := p2Res.OptimalValue
	vPrime2 := new(big.Rat).Inv(sumW)

	// Player 2 optimal strategy: y_c = w_c * v'
	stratP2Nodes := make([]Node, n)
	for c := 0; c < n; c++ {
		wVal := p2Res.PrimalSolution[p2VarNames[c]]
		if wVal == nil {
			wVal = big.NewRat(0, 1)
		}
		yRat := new(big.Rat).Mul(wVal, vPrime2)
		stratP2Nodes[c] = NewRationalFromBigRat(yRat)
	}

	return &ListNode{
		Elements: []Node{
			NewRationalFromBigRat(vGame),
			&ListNode{Elements: stratP1Nodes},
			&ListNode{Elements: stratP2Nodes},
		},
	}, nil
}
