package calc

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
	"fmt"
	"math/big"
)

// -------------------------------------------------------------------------
// Addition Simplification & Like-term Collection (Coeff × Base Model)
// -------------------------------------------------------------------------

type termEntry struct {
	coeff *big.Rat
	base  string
	node  Node
}

func simplifyAdd(terms []Node) (Node, error) {
	// 1. Flatten nested AddNodes
	flatTerms := make([]Node, 0, len(terms))
	for _, t := range terms {
		if add, ok := t.(*AddNode); ok {
			flatTerms = append(flatTerms, add.Terms...)
		} else {
			flatTerms = append(flatTerms, t)
		}
	}

	// 1.5. Check if any term is a MatrixNode
	var firstMatrix *MatrixNode
	for _, t := range flatTerms {
		if m, ok := t.(*MatrixNode); ok {
			firstMatrix = m
			break
		}
	}
	if firstMatrix != nil {
		resData := make([][]Node, firstMatrix.Rows)
		for r := 0; r < firstMatrix.Rows; r++ {
			resData[r] = make([]Node, firstMatrix.Cols)
			for c := 0; c < firstMatrix.Cols; c++ {
				resData[r][c] = mustRational(0, 1)
			}
		}
		for _, t := range flatTerms {
			m, ok := t.(*MatrixNode)
			if !ok {
				return nil, fmt.Errorf("%s", i18n.T("simplify.err_matrix_dimension_error_cannot_add", t.String()))
			}
			if m.Rows != firstMatrix.Rows || m.Cols != firstMatrix.Cols {
				return nil, fmt.Errorf("%s", i18n.T("simplify.err_matrix_dimension_mismatch_cannot_add", firstMatrix.Rows, firstMatrix.Cols, m.Rows, m.Cols))
			}
			for r := 0; r < m.Rows; r++ {
				for c := 0; c < m.Cols; c++ {
					sum, err := simplifyAdd([]Node{resData[r][c], m.Data[r][c]})
					if err != nil {
						return nil, err
					}
					resData[r][c] = sum
				}
			}
		}
		return NewMatrix(firstMatrix.Rows, firstMatrix.Cols, resData)
	}

	// 2. Separate into Coeff × Base
	ratSum := big.NewRat(0, 1)
	imagSum := big.NewRat(0, 1)
	hasComplex := false
	termMap := make(map[string]*termEntry)
	var termOrder []string

	for _, t := range flatTerms {
		switch v := t.(type) {
		case *RationalNode:
			ratSum.Add(ratSum, v.Val)

		case *ComplexNode:
			hasComplex = true
			if rRat, ok := v.Real.(*RationalNode); ok {
				ratSum.Add(ratSum, rRat.Val)
			} else if !isZero(v.Real) {
				baseKey := v.Real.String()
				if entry, exists := termMap[baseKey]; exists {
					entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
				} else {
					termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v.Real}
					termOrder = append(termOrder, baseKey)
				}
			}
			if iRat, ok := v.Imag.(*RationalNode); ok {
				imagSum.Add(imagSum, iRat.Val)
			} else if !isZero(v.Imag) {
				baseKey := fmt.Sprintf("(%s)*i", v.Imag.String())
				if entry, exists := termMap[baseKey]; exists {
					entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
				} else {
					termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
					termOrder = append(termOrder, baseKey)
				}
			}

		case *SqrtNode:
			baseKey := v.String()
			if entry, exists := termMap[baseKey]; exists {
				entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
			} else {
				termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
				termOrder = append(termOrder, baseKey)
			}

		case *MulNode:
			// Check if v is Coeff * Base
			if len(v.Factors) >= 2 {
				if r, ok := v.Factors[0].(*RationalNode); ok {
					var rest []Node
					rest = append(rest, v.Factors[1:]...)
					var baseNode Node
					if len(rest) == 1 {
						baseNode = rest[0]
					} else {
						baseNode = NewMul(rest)
					}
					baseKey := baseNode.String()
					if entry, exists := termMap[baseKey]; exists {
						entry.coeff.Add(entry.coeff, r.Val)
					} else {
						termMap[baseKey] = &termEntry{coeff: new(big.Rat).Set(r.Val), base: baseKey, node: baseNode}
						termOrder = append(termOrder, baseKey)
					}
					continue
				}
			}
			baseKey := v.String()
			if entry, exists := termMap[baseKey]; exists {
				entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
			} else {
				termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
				termOrder = append(termOrder, baseKey)
			}

		default:
			baseKey := v.String()
			if entry, exists := termMap[baseKey]; exists {
				entry.coeff.Add(entry.coeff, big.NewRat(1, 1))
			} else {
				termMap[baseKey] = &termEntry{coeff: big.NewRat(1, 1), base: baseKey, node: v}
				termOrder = append(termOrder, baseKey)
			}
		}
	}

	// 3. Assemble combined terms
	var resultTerms []Node
	if ratSum.Sign() != 0 {
		resultTerms = append(resultTerms, &RationalNode{Val: ratSum})
	}

	for _, key := range termOrder {
		entry := termMap[key]
		if entry.coeff.Sign() == 0 {
			continue // collected to 0
		}
		one := big.NewRat(1, 1)
		if entry.coeff.Cmp(one) == 0 {
			resultTerms = append(resultTerms, entry.node)
		} else {
			resultTerms = append(resultTerms, NewMul([]Node{&RationalNode{Val: entry.coeff}, entry.node}))
		}
	}

	if hasComplex && imagSum.Sign() != 0 {
		var realPart Node
		if len(resultTerms) == 0 {
			realPart = mustRational(0, 1)
		} else if len(resultTerms) == 1 {
			realPart = resultTerms[0]
		} else {
			realPart = NewAdd(resultTerms)
		}
		return NewComplex(realPart, &RationalNode{Val: imagSum}), nil
	}

	if len(resultTerms) == 0 {
		return mustRational(0, 1), nil
	}
	if len(resultTerms) == 1 {
		return resultTerms[0], nil
	}

	return NewAdd(resultTerms), nil
}
