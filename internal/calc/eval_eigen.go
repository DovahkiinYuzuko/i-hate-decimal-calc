package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Exact Matrix Trace, Eigenvalues, and Eigenvectors (Faddeev-LeVerrier + RREF)
// -------------------------------------------------------------------------

// EvalTrace calculates the exact matrix trace: tr(A) = sum_{i=1}^n A_{i,i}.
func EvalTrace(mat *MatrixNode, env *Env) (Node, error) {
	if mat == nil {
		return nil, fmt.Errorf("trace: nil matrix")
	}
	if mat.Rows != mat.Cols {
		return nil, fmt.Errorf("%s: trace requires square matrix, got %dx%d",
			i18n.T("errors.matrix_dim_error"), mat.Rows, mat.Cols)
	}

	n := mat.Rows
	if n == 0 {
		return mustRational(0, 1), nil
	}

	diagTerms := make([]Node, n)
	for i := 0; i < n; i++ {
		diagTerms[i] = mat.Data[i][i]
	}

	sum, err := simplifyAdd(diagTerms)
	if err != nil {
		return nil, err
	}
	return Eval(sum)
}

// faddeevLeVerrier computes the characteristic polynomial coefficients of square matrix mat:
// P(lambda) = det(lambda * I - A) = lambda^n + c_1 * lambda^{n-1} + ... + c_n.
// Returns slice of coefficients in ascending degree order: poly[i] is coefficient of lambda^i (poly[n] = 1).
func faddeevLeVerrier(mat *MatrixNode) ([]Node, error) {
	n := mat.Rows
	if n != mat.Cols {
		return nil, fmt.Errorf("%s: faddeevLeVerrier requires square matrix, got %dx%d",
			i18n.T("errors.matrix_dim_error"), mat.Rows, mat.Cols)
	}

	// Create identity matrix I_n
	identityData := make([][]Node, n)
	for r := 0; r < n; r++ {
		identityData[r] = make([]Node, n)
		for c := 0; c < n; c++ {
			if r == c {
				identityData[r][c] = mustRational(1, 1)
			} else {
				identityData[r][c] = mustRational(0, 1)
			}
		}
	}
	identityMat, err := NewMatrix(n, n, identityData)
	if err != nil {
		return nil, err
	}

	// c[k] for k = 1..n where P(lambda) = lambda^n + c_1 lambda^{n-1} + ... + c_n
	cVals := make([]Node, n+1) // 1-indexed for c_1..c_n
	var B Node = identityMat

	for k := 1; k <= n; k++ {
		// A_k = A * B_{k-1}
		AkNode, err := mulMatrixOrScalar(mat, B)
		if err != nil {
			return nil, err
		}
		AkMat, ok := AkNode.(*MatrixNode)
		if !ok {
			return nil, fmt.Errorf("faddeevLeVerrier: intermediate matrix multiplication failed")
		}

		// tr_k = tr(A_k)
		trK, err := EvalTrace(AkMat, nil)
		if err != nil {
			return nil, err
		}

		// c_k = -(1/k) * tr_k
		invK := mustRational(1, int64(k))
		negTr, err := simplifyUnaryOp("-", trK)
		if err != nil {
			return nil, err
		}
		ckExpr, err := simplifyMul([]Node{negTr, invK})
		if err != nil {
			return nil, err
		}
		ck, err := Eval(ckExpr)
		if err != nil {
			return nil, err
		}
		cVals[k] = ck

		if k < n {
			// B_k = A_k + c_k * I_n
			ckINode, err := mulMatrixOrScalar(ck, identityMat)
			if err != nil {
				return nil, err
			}
			ckIMat := ckINode.(*MatrixNode)

			nextBData := make([][]Node, n)
			for r := 0; r < n; r++ {
				nextBData[r] = make([]Node, n)
				for c := 0; c < n; c++ {
					sumElem, err := simplifyAdd([]Node{AkMat.Data[r][c], ckIMat.Data[r][c]})
					if err != nil {
						return nil, err
					}
					evaledElem, err := Eval(sumElem)
					if err != nil {
						evaledElem = sumElem
					}
					nextBData[r][c] = evaledElem
				}
			}
			B, err = NewMatrix(n, n, nextBData)
			if err != nil {
				return nil, err
			}
		}
	}

	// Convert to ascending degree polynomial:
	// P(lambda) = lambda^n + c_1 lambda^{n-1} + ... + c_n
	// poly[i] is coefficient of lambda^i:
	// poly[n] = 1
	// poly[n-k] = c_k for k=1..n (so poly[0] = c_n)
	poly := make([]Node, n+1)
	poly[n] = mustRational(1, 1)
	for k := 1; k <= n; k++ {
		poly[n-k] = cVals[k]
	}

	return poly, nil
}

// solveQuadraticExact finds exact roots of a2 * x^2 + a1 * x + a0 = 0.
func solveQuadraticExact(a2, a1, a0 Node) ([]Node, error) {
	// D = a1^2 - 4 * a2 * a0
	a1Sq, err := simplifyPow(a1, mustRational(2, 1))
	if err != nil {
		return nil, err
	}
	fourA2A0, err := simplifyMul([]Node{mustRational(4, 1), a2, a0})
	if err != nil {
		return nil, err
	}
	negFourA2A0, err := simplifyUnaryOp("-", fourA2A0)
	if err != nil {
		return nil, err
	}
	dExpr, err := simplifyAdd([]Node{a1Sq, negFourA2A0})
	if err != nil {
		return nil, err
	}
	d, err := Eval(dExpr)
	if err != nil {
		d = dExpr
	}

	negA1, err := simplifyUnaryOp("-", a1)
	if err != nil {
		return nil, err
	}
	twoA2, err := simplifyMul([]Node{mustRational(2, 1), a2})
	if err != nil {
		return nil, err
	}
	invTwoA2, err := simplifyPow(twoA2, mustRational(-1, 1))
	if err != nil {
		return nil, err
	}

	if isZero(d) {
		// Single repeated root: x = -a1 / (2*a2)
		root, err := simplifyMul([]Node{negA1, invTwoA2})
		if err != nil {
			return nil, err
		}
		evaledRoot, err := Eval(root)
		if err != nil {
			evaledRoot = root
		}
		return []Node{evaledRoot, evaledRoot}, nil
	}

	// Check if d is a negative rational -> complex root
	if ratD, ok := d.(*RationalNode); ok && ratD.Val.Sign() < 0 {
		posVal := new(big.Rat).Abs(ratD.Val)
		sqrtPos, err := simplifySqrt(&RationalNode{Val: posVal})
		if err != nil {
			return nil, err
		}
		// sqrt(D) = i * sqrt(|D|)
		iUnit := &ConstNode{Name: "i"}
		iSqrtD, err := simplifyMul([]Node{iUnit, sqrtPos})
		if err != nil {
			return nil, err
		}
		negISqrtD, err := simplifyUnaryOp("-", iSqrtD)
		if err != nil {
			return nil, err
		}

		num1, err := simplifyAdd([]Node{negA1, negISqrtD})
		if err != nil {
			return nil, err
		}
		root1, err := simplifyMul([]Node{num1, invTwoA2})
		if err != nil {
			return nil, err
		}
		evaledRoot1, err := Eval(root1)
		if err != nil {
			evaledRoot1 = root1
		}

		num2, err := simplifyAdd([]Node{negA1, iSqrtD})
		if err != nil {
			return nil, err
		}
		root2, err := simplifyMul([]Node{num2, invTwoA2})
		if err != nil {
			return nil, err
		}
		evaledRoot2, err := Eval(root2)
		if err != nil {
			evaledRoot2 = root2
		}

		return []Node{evaledRoot1, evaledRoot2}, nil
	}

	sqrtD, err := simplifySqrt(d)
	if err != nil {
		return nil, err
	}
	negSqrtD, err := simplifyUnaryOp("-", sqrtD)
	if err != nil {
		return nil, err
	}

	num1, err := simplifyAdd([]Node{negA1, negSqrtD})
	if err != nil {
		return nil, err
	}
	root1, err := simplifyMul([]Node{num1, invTwoA2})
	if err != nil {
		return nil, err
	}
	evaledRoot1, err := Eval(root1)
	if err != nil {
		evaledRoot1 = root1
	}

	num2, err := simplifyAdd([]Node{negA1, sqrtD})
	if err != nil {
		return nil, err
	}
	root2, err := simplifyMul([]Node{num2, invTwoA2})
	if err != nil {
		return nil, err
	}
	evaledRoot2, err := Eval(root2)
	if err != nil {
		evaledRoot2 = root2
	}

	return []Node{evaledRoot1, evaledRoot2}, nil
}

// solvePolyRoots finds all algebraic roots of polynomial poly given in ascending degree order:
// poly[0] + poly[1]*x + ... + poly[deg]*x^deg = 0.
func solvePolyRoots(poly []Node) ([]Node, error) {
	deg := len(poly) - 1
	for deg >= 0 && isZero(poly[deg]) {
		deg--
	}
	if deg <= 0 {
		return nil, fmt.Errorf("cannot find roots of constant or zero polynomial")
	}

	if deg == 1 {
		// a1 * x + a0 = 0 => x = -a0 / a1
		negA0, err := simplifyUnaryOp("-", poly[0])
		if err != nil {
			return nil, err
		}
		invA1, err := simplifyPow(poly[1], mustRational(-1, 1))
		if err != nil {
			return nil, err
		}
		root, err := simplifyMul([]Node{negA0, invA1})
		if err != nil {
			return nil, err
		}
		evaled, err := Eval(root)
		if err != nil {
			evaled = root
		}
		return []Node{evaled}, nil
	}

	if deg == 2 {
		return solveQuadraticExact(poly[2], poly[1], poly[0])
	}

	// Degree >= 3:
	// Check if all coefficients are rational to apply Rational Root Theorem & synthetic division
	allRational := true
	for i := 0; i <= deg; i++ {
		if _, ok := poly[i].(*RationalNode); !ok {
			allRational = false
			break
		}
	}

	if allRational {
		// Convert to integer polynomial by clearing denominators
		commonLcm := big.NewInt(1)
		for i := 0; i <= deg; i++ {
			r := poly[i].(*RationalNode)
			commonLcm = lcmInt(commonLcm, r.Val.Denom())
		}

		intPoly := make([]*big.Int, deg+1)
		for i := 0; i <= deg; i++ {
			r := poly[i].(*RationalNode)
			mult := new(big.Int).Div(commonLcm, r.Val.Denom())
			intPoly[i] = new(big.Int).Mul(r.Val.Num(), mult)
		}

		// Handle root x = 0 (min degree > 0)
		if intPoly[0].Sign() == 0 {
			var roots []Node
			var reducedIntPoly []*big.Int
			// Count trailing zeros
			k := 0
			for k <= deg && intPoly[k].Sign() == 0 {
				roots = append(roots, mustRational(0, 1))
				k++
			}
			reducedIntPoly = intPoly[k:]
			reducedNodes := make([]Node, len(reducedIntPoly))
			for idx, c := range reducedIntPoly {
				reducedNodes[idx] = &RationalNode{Val: new(big.Rat).SetInt(c)}
			}
			remRoots, err := solvePolyRoots(reducedNodes)
			if err != nil {
				return nil, err
			}
			roots = append(roots, remRoots...)
			return roots, nil
		}

		// Find rational roots using factor.go's findRationalRoots
		candRoots := findRationalRoots(intPoly)
		if len(candRoots) > 0 {
			var roots []Node
			currPoly := intPoly

			for _, cr := range candRoots {
				// Divide as many times as possible by (cr.denom * x - cr.num)
				for {
					quot, ok := syntheticDivide(currPoly, cr.num, cr.denom)
					if !ok {
						break
					}
					rootRat := new(big.Rat).SetFrac(cr.num, cr.denom)
					roots = append(roots, &RationalNode{Val: rootRat})
					currPoly = quot
					if len(currPoly) <= 1 {
						break
					}
				}
				if len(currPoly) <= 1 {
					break
				}
			}

			if len(currPoly) > 1 {
				// Convert remaining quotient polynomial to Node slice
				remNodes := make([]Node, len(currPoly))
				for idx, c := range currPoly {
					remNodes[idx] = &RationalNode{Val: new(big.Rat).SetInt(c)}
				}
				remRoots, err := solvePolyRoots(remNodes)
				if err == nil {
					roots = append(roots, remRoots...)
					return roots, nil
				}
			} else {
				return roots, nil
			}
		}
	}

	return nil, fmt.Errorf("eigenvals: cannot find exact roots for degree %d characteristic polynomial", deg)
}

// nullSpaceBasis computes the exact null space basis vectors for matrix A (satisfying A * v = 0).
// Each basis vector is returned as a 1D ListNode containing normalized integer or exact scalar components.
func nullSpaceBasis(A *MatrixNode) ([]Node, error) {
	if A == nil {
		return nil, fmt.Errorf("nullSpaceBasis: nil matrix")
	}

	rrefMat, pivotCols, err := computeRREF(A)
	if err != nil {
		return nil, err
	}

	cols := A.Cols
	isPivot := make([]bool, cols)
	pivotRowForCol := make([]int, cols)
	for r, pc := range pivotCols {
		if pc < cols {
			isPivot[pc] = true
			pivotRowForCol[pc] = r
		}
	}

	var freeCols []int
	for c := 0; c < cols; c++ {
		if !isPivot[c] {
			freeCols = append(freeCols, c)
		}
	}

	if len(freeCols) == 0 {
		// Trivial null space {0}
		zeroVec := make([]Node, cols)
		for i := 0; i < cols; i++ {
			zeroVec[i] = mustRational(0, 1)
		}
		return []Node{NewList(zeroVec)}, nil
	}

	var basisList []Node
	for _, freeCol := range freeCols {
		vec := make([]Node, cols)
		// For freeCol, value is 1; for other free cols, value is 0
		vec[freeCol] = mustRational(1, 1)
		for _, otherFC := range freeCols {
			if otherFC != freeCol {
				vec[otherFC] = mustRational(0, 1)
			}
		}

		// For pivot variables: x_{pc} = - RREF[r][freeCol]
		for pc := 0; pc < cols; pc++ {
			if isPivot[pc] {
				r := pivotRowForCol[pc]
				elem := rrefMat.Data[r][freeCol]
				negElem, err := simplifyUnaryOp("-", elem)
				if err != nil {
					return nil, err
				}
				evaledElem, err := Eval(negElem)
				if err != nil {
					evaledElem = negElem
				}
				vec[pc] = evaledElem
			}
		}

		// Normalize basis vector: clear rational denominators to get clean integer ratios
		normVec := normalizeBasisVector(vec)
		basisList = append(basisList, NewList(normVec))
	}

	return basisList, nil
}

// normalizeBasisVector scales a vector of rational components to coprime integers.
func normalizeBasisVector(vec []Node) []Node {
	n := len(vec)
	allRational := true
	for _, elem := range vec {
		if _, ok := elem.(*RationalNode); !ok {
			allRational = false
			break
		}
	}

	if !allRational {
		return vec
	}

	// 1. Find common LCM of all denominators
	commonLcm := big.NewInt(1)
	for _, elem := range vec {
		r := elem.(*RationalNode)
		commonLcm = lcmInt(commonLcm, r.Val.Denom())
	}

	// 2. Scale elements to integers
	intComponents := make([]*big.Int, n)
	for i, elem := range vec {
		r := elem.(*RationalNode)
		mult := new(big.Int).Div(commonLcm, r.Val.Denom())
		intComponents[i] = new(big.Int).Mul(r.Val.Num(), mult)
	}

	// 3. Find GCD of all non-zero components
	var commonGcd *big.Int
	for _, c := range intComponents {
		if c.Sign() != 0 {
			if commonGcd == nil {
				commonGcd = new(big.Int).Abs(c)
			} else {
				commonGcd.GCD(nil, nil, commonGcd, c)
			}
		}
	}

	if commonGcd == nil || commonGcd.Sign() == 0 {
		return vec
	}

	// 4. Determine sign: prefer first non-zero component to be positive
	sign := int64(1)
	for _, c := range intComponents {
		if c.Sign() != 0 {
			if c.Sign() < 0 {
				sign = -1
			}
			break
		}
	}

	divisor := new(big.Int).Mul(commonGcd, big.NewInt(sign))
	result := make([]Node, n)
	for i, c := range intComponents {
		normVal := new(big.Int).Div(c, divisor)
		result[i] = &RationalNode{Val: new(big.Rat).SetInt(normVal)}
	}

	return result
}

// EvalEigenvals calculates all eigenvalues of square matrix mat.
func EvalEigenvals(mat *MatrixNode, env *Env) (Node, error) {
	if mat == nil {
		return nil, fmt.Errorf("eigenvals: nil matrix")
	}
	if mat.Rows != mat.Cols {
		return nil, fmt.Errorf("%s: eigenvals requires square matrix, got %dx%d",
			i18n.T("errors.matrix_dim_error"), mat.Rows, mat.Cols)
	}

	n := mat.Rows
	if n == 0 {
		return NewList([]Node{}), nil
	}
	if n == 1 {
		evaled, err := Eval(mat.Data[0][0])
		if err != nil {
			evaled = mat.Data[0][0]
		}
		return NewList([]Node{evaled}), nil
	}

	// 1. Characteristic polynomial via Faddeev-LeVerrier algorithm
	poly, err := faddeevLeVerrier(mat)
	if err != nil {
		return nil, err
	}

	// 2. Solve polynomial roots
	roots, err := solvePolyRoots(poly)
	if err != nil {
		return nil, err
	}

	return NewList(roots), nil
}

// EvalEigenvects calculates eigenvalues and their corresponding eigenvector bases:
// Returns [[lambda_1, [v_1, ...]], [lambda_2, [v_2, ...]], ...]
func EvalEigenvects(mat *MatrixNode, env *Env) (Node, error) {
	if mat == nil {
		return nil, fmt.Errorf("eigenvects: nil matrix")
	}
	if mat.Rows != mat.Cols {
		return nil, fmt.Errorf("%s: eigenvects requires square matrix, got %dx%d",
			i18n.T("errors.matrix_dim_error"), mat.Rows, mat.Cols)
	}

	n := mat.Rows
	if n == 0 {
		return NewList([]Node{}), nil
	}

	// 1. Get eigenvalues
	eigenvalsNode, err := EvalEigenvals(mat, env)
	if err != nil {
		return nil, err
	}
	eigenvalsList := eigenvalsNode.(*ListNode)

	// 2. Group unique eigenvalues while preserving multiplicities
	type valGroup struct {
		val  Node
		mult int
	}
	var uniqueVals []valGroup
	for _, ev := range eigenvalsList.Elements {
		found := false
		for i := range uniqueVals {
			// Compare expressions via Eval(ev - uniqueVals[i].val) == 0
			diffExpr, err := simplifyAdd([]Node{ev, mustNegate(uniqueVals[i].val)})
			if err == nil {
				evaledDiff, err := Eval(diffExpr)
				if err == nil && isZero(evaledDiff) {
					uniqueVals[i].mult++
					found = true
					break
				}
			}
		}
		if !found {
			uniqueVals = append(uniqueVals, valGroup{val: ev, mult: 1})
		}
	}

	// 3. For each eigenvalue lambda_i, build singular matrix (lambda_i * I - A)
	// and extract null space basis
	var resultList []Node
	for _, vg := range uniqueVals {
		lambda := vg.val

		// Construct M = lambda * I - A
		mData := make([][]Node, n)
		for r := 0; r < n; r++ {
			mData[r] = make([]Node, n)
			for c := 0; c < n; c++ {
				negArc, err := simplifyUnaryOp("-", mat.Data[r][c])
				if err != nil {
					return nil, err
				}
				var elem Node
				if r == c {
					elem, err = simplifyAdd([]Node{lambda, negArc})
				} else {
					elem = negArc
				}
				if err != nil {
					return nil, err
				}
				evaledElem, err := Eval(elem)
				if err != nil {
					evaledElem = elem
				}
				mData[r][c] = evaledElem
			}
		}

		singularMat, err := NewMatrix(n, n, mData)
		if err != nil {
			return nil, err
		}

		bases, err := nullSpaceBasis(singularMat)
		if err != nil {
			return nil, err
		}

		// Pair: [lambda, [bases...]]
		pair := NewList([]Node{lambda, NewList(bases)})
		resultList = append(resultList, pair)
	}

	return NewList(resultList), nil
}

func mustNegate(n Node) Node {
	neg, err := simplifyUnaryOp("-", n)
	if err != nil {
		return n
	}
	return neg
}
