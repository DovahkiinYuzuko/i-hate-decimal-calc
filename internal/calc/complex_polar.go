package calc

import (
	"fmt"
	"math/big"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// extractRealImag extracts the exact real and imaginary components from an AST node.
// Returns realPart and imagPart such that z = realPart + i * imagPart.
func extractRealImag(n Node) (Node, Node, error) {
	if n == nil {
		return mustRational(0, 1), mustRational(0, 1), nil
	}

	evaled, err := Eval(n)
	if err != nil {
		evaled = n
	}

	switch v := evaled.(type) {
	case *ComplexNode:
		return v.Real, v.Imag, nil

	case *RationalNode:
		return v, mustRational(0, 1), nil

	case *VarNode:
		if v.Name == "i" {
			return mustRational(0, 1), mustRational(1, 1), nil
		}
		return v, mustRational(0, 1), nil

	case *ConstNode:
		if v.Name == "i" {
			return mustRational(0, 1), mustRational(1, 1), nil
		}
		return v, mustRational(0, 1), nil

	case *UnaryOpNode:
		if v.Op == "-" {
			re, im, err := extractRealImag(v.Expr)
			if err != nil {
				return nil, nil, err
			}
			negRe, _ := simplifyUnaryOp("-", re)
			negIm, _ := simplifyUnaryOp("-", im)
			return negRe, negIm, nil
		}
		return v, mustRational(0, 1), nil

	case *MulNode:
		// Check if "i" is a factor
		iIdx := -1
		for idx, f := range v.Factors {
			if isImaginaryUnit(f) {
				iIdx = idx
				break
			}
		}
		if iIdx >= 0 {
			otherFactors := make([]Node, 0, len(v.Factors)-1)
			for idx, f := range v.Factors {
				if idx != iIdx {
					otherFactors = append(otherFactors, f)
				}
			}
			if len(otherFactors) == 0 {
				return mustRational(0, 1), mustRational(1, 1), nil
			}
			imPart, err := simplifyMul(otherFactors)
			if err != nil {
				return nil, nil, err
			}
			return mustRational(0, 1), imPart, nil
		}
		return v, mustRational(0, 1), nil

	case *AddNode:
		var realTerms []Node
		var imagTerms []Node
		for _, term := range v.Terms {
			re, im, err := extractRealImag(term)
			if err != nil {
				return nil, nil, err
			}
			if !isZero(re) {
				realTerms = append(realTerms, re)
			}
			if !isZero(im) {
				imagTerms = append(imagTerms, im)
			}
		}
		var realPart Node = mustRational(0, 1)
		var imagPart Node = mustRational(0, 1)
		if len(realTerms) == 1 {
			realPart = realTerms[0]
		} else if len(realTerms) > 1 {
			realPart, _ = Eval(NewAdd(realTerms))
		}
		if len(imagTerms) == 1 {
			imagPart = imagTerms[0]
		} else if len(imagTerms) > 1 {
			imagPart, _ = Eval(NewAdd(imagTerms))
		}
		return realPart, imagPart, nil

	default:
		return v, mustRational(0, 1), nil
	}
}

func isImaginaryUnit(n Node) bool {
	if c, ok := n.(*ConstNode); ok && c.Name == "i" {
		return true
	}
	if v, ok := n.(*VarNode); ok && v.Name == "i" {
		return true
	}
	return false
}

// evalArg calculates the exact principal argument Arg(z) in (-pi, pi].
func evalArg(z Node) (Node, error) {
	evaled, err := Eval(z)
	if err != nil {
		return nil, err
	}

	re, im, err := extractRealImag(evaled)
	if err != nil {
		return nil, err
	}

	sRe, errRe := signNode(re)
	sIm, errIm := signNode(im)
	if errRe != nil || errIm != nil {
		return nil, fmt.Errorf("cannot determine sign of complex components (re=%s, im=%s)", re.String(), im.String())
	}

	// Origin: undefined
	if sRe == 0 && sIm == 0 {
		return nil, fmt.Errorf("%s", i18n.T("errors.arg_zero_undefined"))
	}

	// Real axis:
	if sIm == 0 {
		if sRe > 0 {
			return mustRational(0, 1), nil // 0
		}
		return &ConstNode{Name: "pi"}, nil // pi
	}

	// Imaginary axis:
	if sRe == 0 {
		if sIm > 0 {
			return piMultiple(1, 2), nil // pi / 2
		}
		return piMultiple(-1, 2), nil // -pi / 2
	}

	// General quadrants: test special angles via |im| / |re|
	posRe := re
	if sRe < 0 {
		posRe, _ = simplifyUnaryOp("-", re)
	}
	posIm := im
	if sIm < 0 {
		posIm, _ = simplifyUnaryOp("-", im)
	}

	ratioNode := NewMul([]Node{posIm, &PowNode{Base: posRe, Exp: mustRational(-1, 1)}})
	evalRatio, err := Eval(ratioNode)
	if err != nil {
		evalRatio = ratioNode
	}

	vType, _ := classifyTrigVal(evalRatio)

	var baseThetaNum, baseThetaDenom int64
	switch vType {
	case "1":
		baseThetaNum, baseThetaDenom = 1, 4 // pi / 4
	case "sqrt(3)":
		baseThetaNum, baseThetaDenom = 1, 3 // pi / 3
	case "sqrt(3)/3":
		baseThetaNum, baseThetaDenom = 1, 6 // pi / 6
	}

	if baseThetaDenom > 0 {
		// Quadrant mapping:
		// Q1 (re > 0, im > 0): theta = theta0
		// Q2 (re < 0, im > 0): theta = pi - theta0 = (denom - num) / denom * pi
		// Q3 (re < 0, im < 0): theta = -pi + theta0 = -(denom - num) / denom * pi
		// Q4 (re > 0, im < 0): theta = -theta0
		if sRe > 0 && sIm > 0 {
			return piMultiple(baseThetaNum, baseThetaDenom), nil
		} else if sRe < 0 && sIm > 0 {
			return piMultiple(baseThetaDenom-baseThetaNum, baseThetaDenom), nil
		} else if sRe < 0 && sIm < 0 {
			return piMultiple(-(baseThetaDenom - baseThetaNum), baseThetaDenom), nil
		} else {
			return piMultiple(-baseThetaNum, baseThetaDenom), nil
		}
	}

	// Fallback to symbolic atan
	atanNode := &FuncNode{Name: "atan", Args: []Node{evalRatio}}
	piNode := &ConstNode{Name: "pi"}
	if sRe > 0 && sIm > 0 {
		return atanNode, nil
	} else if sRe < 0 && sIm > 0 {
		return NewAdd([]Node{piNode, &UnaryOpNode{Op: "-", Expr: atanNode}}), nil
	} else if sRe < 0 && sIm < 0 {
		negPi := &UnaryOpNode{Op: "-", Expr: piNode}
		return NewAdd([]Node{negPi, atanNode}), nil
	} else {
		return &UnaryOpNode{Op: "-", Expr: atanNode}, nil
	}
}

// evalPolar converts z into polar form: r * (cos(theta) + i * sin(theta)).
func evalPolar(z Node) (Node, error) {
	evaled, err := Eval(z)
	if err != nil {
		return nil, err
	}

	re, im, err := extractRealImag(evaled)
	if err != nil {
		return nil, err
	}

	// Modulus r = sqrt(re^2 + im^2)
	aSq := &PowNode{Base: re, Exp: mustRational(2, 1)}
	bSq := &PowNode{Base: im, Exp: mustRational(2, 1)}
	sumNode := NewAdd([]Node{aSq, bSq})
	evaledSum, err := Eval(sumNode)
	if err != nil {
		return nil, err
	}
	rNode, err := simplifySqrt(evaledSum)
	if err != nil {
		return nil, err
	}
	rNode, err = Eval(rNode)
	if err != nil {
		return nil, err
	}

	// If r = 0, return 0
	if isZero(rNode) {
		return mustRational(0, 1), nil
	}

	thetaNode, err := evalArg(evaled)
	if err != nil {
		return nil, err
	}

	iNode := &ConstNode{Name: "i"}
	cosNode := &FuncNode{Name: "cos", Args: []Node{thetaNode}}
	sinNode := &FuncNode{Name: "sin", Args: []Node{thetaNode}}
	isinTheta := NewMul([]Node{iNode, sinNode})
	cisNode := NewAdd([]Node{cosNode, isinTheta})

	if rat, ok := rNode.(*RationalNode); ok && rat.Val.Cmp(big.NewRat(1, 1)) == 0 {
		return cisNode, nil
	}

	return NewMul([]Node{rNode, cisNode}), nil
}

// evalPolarExp converts z into Euler exponential form: r * e^(i * theta).
func evalPolarExp(z Node) (Node, error) {
	evaled, err := Eval(z)
	if err != nil {
		return nil, err
	}

	re, im, err := extractRealImag(evaled)
	if err != nil {
		return nil, err
	}

	aSq := &PowNode{Base: re, Exp: mustRational(2, 1)}
	bSq := &PowNode{Base: im, Exp: mustRational(2, 1)}
	sumNode := NewAdd([]Node{aSq, bSq})
	evaledSum, err := Eval(sumNode)
	if err != nil {
		return nil, err
	}
	rNode, err := simplifySqrt(evaledSum)
	if err != nil {
		return nil, err
	}
	rNode, err = Eval(rNode)
	if err != nil {
		return nil, err
	}

	if isZero(rNode) {
		return mustRational(0, 1), nil
	}

	thetaNode, err := evalArg(evaled)
	if err != nil {
		return nil, err
	}

	eNode := &ConstNode{Name: "e"}
	iNode := &ConstNode{Name: "i"}
	expNode, _ := NewPow(eNode, NewMul([]Node{iNode, thetaNode}))

	if rat, ok := rNode.(*RationalNode); ok && rat.Val.Cmp(big.NewRat(1, 1)) == 0 {
		return expNode, nil
	}

	return NewMul([]Node{rNode, expNode}), nil
}

// evalRect converts polar form (r, theta) into rectangular complex form: r * cos(theta) + i * (r * sin(theta)).
func evalRect(rNode, thetaNode Node) (Node, error) {
	evalR, err := Eval(rNode)
	if err != nil {
		return nil, err
	}
	evalTheta, err := Eval(thetaNode)
	if err != nil {
		return nil, err
	}

	cosNode := &FuncNode{Name: "cos", Args: []Node{evalTheta}}
	sinNode := &FuncNode{Name: "sin", Args: []Node{evalTheta}}

	evalCos, err := Eval(cosNode)
	if err != nil {
		return nil, err
	}
	evalSin, err := Eval(sinNode)
	if err != nil {
		return nil, err
	}

	rePart, err := Eval(NewMul([]Node{evalR, evalCos}))
	if err != nil {
		return nil, err
	}
	imPart, err := Eval(NewMul([]Node{evalR, evalSin}))
	if err != nil {
		return nil, err
	}

	if isZero(imPart) {
		return rePart, nil
	}

	iNode := &ConstNode{Name: "i"}
	return Eval(NewAdd([]Node{rePart, NewMul([]Node{iNode, imPart})}))
}
