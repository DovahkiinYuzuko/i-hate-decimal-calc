package calc

import (
	"fmt"
	"strings"
)

// -------------------------------------------------------------------------
// Symbolic Fourier Series Expansion
// (Euler-Fourier Formulas & Harmonic Analysis)
// -------------------------------------------------------------------------

// EvalFourierSeries computes the symbolic Fourier series expansion of f(t) up to order n with half-period L.
// Signature: fourier_series(f, [t, L, n])
//   - f: function to expand
//   - t: expansion variable (optional, defaults to detected variable or "t")
//   - L: half-period (optional, defaults to pi)
//   - n: expansion order/degree (optional, defaults to 3)
func EvalFourierSeries(args []Node, env *Env) (Node, error) {
	if len(args) < 1 || len(args) > 4 {
		return nil, fmt.Errorf("fourier_series: requires between 1 and 4 arguments: f(t), [var, L, n]")
	}

	f := args[0]
	if f == nil {
		return nil, fmt.Errorf("fourier_series: nil function argument")
	}

	// 1. Variable extraction
	var tVar string
	var err error
	if len(args) >= 2 && args[1] != nil {
		tVar, err = extractFourierVar(args[1], f)
		if err != nil {
			return nil, err
		}
	} else {
		tVar, err = extractFourierVar(nil, f)
		if err != nil {
			return nil, err
		}
	}

	// 2. Half-period L (default: pi)
	var L Node = &ConstNode{Name: "pi"}
	if len(args) >= 3 && args[2] != nil {
		L = args[2]
	}
	evalL, err := EvalWithEnv(L, env)
	if err == nil && evalL != nil {
		L = evalL
	}
	if isZero(L) {
		return nil, fmt.Errorf("fourier_series: half-period L cannot be zero")
	}

	// 3. Expansion order n (default: 3)
	var n int64 = 3
	if len(args) >= 4 && args[3] != nil {
		n, err = extractFourierOrder(args[3], 3)
		if err != nil {
			return nil, err
		}
	}

	tNode := &VarNode{Name: tVar}
	negL, err := simplifyUnaryOp("-", L)
	if err != nil {
		negL = &UnaryOpNode{Op: "-", Expr: L}
	}

	// 4. Compute DC component: a0 = (1 / (2L)) * int_{-L}^L f(t) dt
	intA0, err := evalDefiniteIntegral(f, tVar, negL, L)
	if err != nil {
		return nil, fmt.Errorf("fourier_series: failed to integrate DC term: %w", err)
	}

	twoL, err := simplifyMul([]Node{mustRational(2, 1), L})
	if err != nil {
		twoL = NewMul([]Node{mustRational(2, 1), L})
	}
	invTwoL, err := simplifyPow(twoL, mustRational(-1, 1))
	if err != nil {
		invTwoL = &PowNode{Base: twoL, Exp: mustRational(-1, 1)}
	}
	a0, err := simplifyMul([]Node{intA0, invTwoL})
	if err != nil {
		a0 = NewMul([]Node{intA0, invTwoL})
	}
	a0Evaled, err := EvalWithEnv(a0, env)
	if err == nil && a0Evaled != nil {
		a0 = a0Evaled
	}

	var terms []Node
	if !isZero(a0) {
		terms = append(terms, a0)
	}

	// 5. Precompute 1 / L
	invL, err := simplifyPow(L, mustRational(-1, 1))
	if err != nil {
		invL = &PowNode{Base: L, Exp: mustRational(-1, 1)}
	}

	// Check if L is pi (VarNode "pi" or ConstNode "pi")
	isPiPeriod := false
	if vn, ok := L.(*VarNode); ok && vn.Name == "pi" {
		isPiPeriod = true
	} else if cn, ok := L.(*ConstNode); ok && cn.Name == "pi" {
		isPiPeriod = true
	}

	// 6. Loop for k = 1 ... n
	for k := int64(1); k <= n; k++ {
		kNode := mustRational(k, 1)

		// Harmonic argument: omega_k * t = (k * pi * t) / L
		var omegaKt Node
		if isPiPeriod {
			// If L == pi, (k * pi * t) / pi = k * t
			omegaKt, err = simplifyMul([]Node{kNode, tNode})
			if err != nil {
				omegaKt = NewMul([]Node{kNode, tNode})
			}
		} else {
			piNode := &ConstNode{Name: "pi"}
			num, _ := simplifyMul([]Node{kNode, piNode, tNode})
			omegaKt, _ = simplifyMul([]Node{num, invL})
		}
		omegaKt, _ = EvalWithEnv(omegaKt, env)

		cosNode := &FuncNode{Name: "cos", Args: []Node{omegaKt}}
		sinNode := &FuncNode{Name: "sin", Args: []Node{omegaKt}}

		// Cosine harmonic: ak = (1 / L) * int_{-L}^L f(t) * cos(omega_k * t) dt
		integrandCos, err := simplifyMul([]Node{f, cosNode})
		if err != nil {
			integrandCos = NewMul([]Node{f, cosNode})
		}
		if reduced, rErr := EvalTrigReduce(integrandCos, env); rErr == nil && reduced != nil {
			integrandCos = reduced
		}

		intAk, err := evalDefiniteIntegral(integrandCos, tVar, negL, L)
		if err == nil && intAk != nil {
			ak, akErr := simplifyMul([]Node{intAk, invL})
			if akErr == nil {
				ak, _ = EvalWithEnv(ak, env)
				if !isZero(ak) {
					termA, tErr := simplifyMul([]Node{ak, cosNode})
					if tErr == nil {
						terms = append(terms, termA)
					} else {
						terms = append(terms, NewMul([]Node{ak, cosNode}))
					}
				}
			}
		} else if err != nil {
			return nil, fmt.Errorf("fourier_series: failed to integrate cos harmonic k=%d: %w", k, err)
		}

		// Sine harmonic: bk = (1 / L) * int_{-L}^L f(t) * sin(omega_k * t) dt
		integrandSin, err := simplifyMul([]Node{f, sinNode})
		if err != nil {
			integrandSin = NewMul([]Node{f, sinNode})
		}
		if reduced, rErr := EvalTrigReduce(integrandSin, env); rErr == nil && reduced != nil {
			integrandSin = reduced
		}

		intBk, err := evalDefiniteIntegral(integrandSin, tVar, negL, L)
		if err == nil && intBk != nil {
			bk, bkErr := simplifyMul([]Node{intBk, invL})
			if bkErr == nil {
				bk, _ = EvalWithEnv(bk, env)
				if !isZero(bk) {
					termB, tErr := simplifyMul([]Node{bk, sinNode})
					if tErr == nil {
						terms = append(terms, termB)
					} else {
						terms = append(terms, NewMul([]Node{bk, sinNode}))
					}
				}
			}
		} else if err != nil {
			return nil, fmt.Errorf("fourier_series: failed to integrate sin harmonic k=%d: %w", k, err)
		}
	}

	if len(terms) == 0 {
		return mustRational(0, 1), nil
	}
	if len(terms) == 1 {
		return terms[0], nil
	}

	res, err := simplifyAdd(terms)
	if err != nil {
		res = NewAdd(terms)
	}
	res, err = EvalWithEnv(res, env)
	if err == nil && res != nil {
		return res, nil
	}
	return Eval(res)
}

// extractFourierVar extracts or deduces the variable of interest.
func extractFourierVar(node Node, f Node) (string, error) {
	if node != nil {
		switch v := node.(type) {
		case *VarNode:
			return v.Name, nil
		case *ConstNode:
			return v.Name, nil
		default:
			str := strings.TrimSpace(node.String())
			if str != "" {
				return str, nil
			}
		}
	}

	// Deduce from f
	vars := collectVariables(f)
	var filtered []string
	for _, v := range vars {
		if v != "pi" && v != "e" && v != "I" {
			filtered = append(filtered, v)
		}
	}
	if len(filtered) > 0 {
		return selectMainVariable(filtered), nil
	}
	return "t", nil
}

// extractFourierOrder extracts a positive integer degree n.
func extractFourierOrder(node Node, defaultN int64) (int64, error) {
	if node == nil {
		return defaultN, nil
	}
	evaled, err := Eval(node)
	if err != nil {
		evaled = node
	}
	if rn, ok := evaled.(*RationalNode); ok && rn.Val.IsInt() {
		val := rn.Val.Num().Int64()
		if val <= 0 {
			return 0, fmt.Errorf("fourier_series: degree n must be a positive integer, got %d", val)
		}
		if val > 100 {
			return 0, fmt.Errorf("fourier_series: degree n exceeds maximum limit (100), got %d", val)
		}
		return val, nil
	}
	return 0, fmt.Errorf("fourier_series: invalid degree node %s", node.String())
}
