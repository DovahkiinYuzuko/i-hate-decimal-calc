package lp

import (
	"fmt"
	"math/big"
)

// SanitizeAndVerifyCertificate verifies that the certificate satisfies exact mathematical invariants
// using independent matrix-vector rational arithmetic.
func SanitizeAndVerifyCertificate(prob *Problem, cert *Certificate) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}

	n := len(prob.VarNames)
	m := len(prob.Constraints)

	// Case 1: StatusOptimal
	if cert.PrimalPoint != nil && cert.DualVector != nil {
		x := cert.PrimalPoint
		y := cert.DualVector

		if len(x) != n {
			return fmt.Errorf("primal point dimension (%d) mismatch with variable count (%d)", len(x), n)
		}
		if len(y) != m {
			return fmt.Errorf("dual vector dimension (%d) mismatch with constraint count (%d)", len(y), m)
		}

		// 1. Primal Non-negativity: x_j >= 0
		for j := 0; j < n; j++ {
			if x[j].Sign() < 0 {
				return fmt.Errorf("primal variable %s = %s is negative", prob.VarNames[j], x[j].String())
			}
		}

		// 2. Primal Feasibility: A x <= b (or >=, ==)
		ax := make([]*big.Rat, m)
		for i, c := range prob.Constraints {
			sum := new(big.Rat)
			for j := 0; j < n; j++ {
				term := new(big.Rat).Mul(c.Coeffs[j], x[j])
				sum.Add(sum, term)
			}
			ax[i] = sum

			cmp := sum.Cmp(c.RHS)
			switch c.Sense {
			case RelLe:
				if cmp > 0 {
					return fmt.Errorf("primal constraint %d (%s) violated: %s > %s", i+1, c.Name, sum.String(), c.RHS.String())
				}
			case RelGe:
				if cmp < 0 {
					return fmt.Errorf("primal constraint %d (%s) violated: %s < %s", i+1, c.Name, sum.String(), c.RHS.String())
				}
			case RelEq:
				if cmp != 0 {
					return fmt.Errorf("primal equality constraint %d (%s) violated: %s != %s", i+1, c.Name, sum.String(), c.RHS.String())
				}
			}
		}

		// 3. Objective value equality: c^T x == b^T y
		cx := new(big.Rat)
		for j := 0; j < n; j++ {
			cx.Add(cx, new(big.Rat).Mul(prob.Objective[j], x[j]))
		}
		if prob.ObjConstant != nil {
			cx.Add(cx, prob.ObjConstant)
		}

		by := new(big.Rat)
		for i, c := range prob.Constraints {
			by.Add(by, new(big.Rat).Mul(c.RHS, y[i]))
		}
		if prob.ObjConstant != nil {
			by.Add(by, prob.ObjConstant)
		}

		if cx.Cmp(by) != 0 {
			return fmt.Errorf("strong duality gap violated: c^T x (%s) != b^T y (%s)", cx.String(), by.String())
		}

		// 4. Complementary Slackness: y_i * (b_i - A_i x) == 0
		for i, c := range prob.Constraints {
			slack := new(big.Rat).Sub(c.RHS, ax[i])
			prod := new(big.Rat).Mul(y[i], slack)
			if prod.Sign() != 0 {
				return fmt.Errorf("complementary slackness violated on constraint %d (%s): y_i=%s, slack=%s, prod=%s",
					i+1, c.Name, y[i].String(), slack.String(), prod.String())
			}
		}
		return nil
	}

	// Case 2: StatusInfeasible (Farkas Certificate)
	if cert.FarkasRay != nil {
		y := cert.FarkasRay
		if len(y) != m {
			return fmt.Errorf("farkas vector dimension (%d) mismatch with constraint count (%d)", len(y), m)
		}

		// Check y^T A >= 0 (for maximization) and y^T b < 0
		yA := make([]*big.Rat, n)
		for j := 0; j < n; j++ {
			sum := new(big.Rat)
			for i, c := range prob.Constraints {
				term := new(big.Rat).Mul(y[i], c.Coeffs[j])
				sum.Add(sum, term)
			}
			yA[j] = sum
			if sum.Sign() < 0 {
				return fmt.Errorf("farkas condition violated: (y^T A)_%d = %s < 0", j, sum.String())
			}
		}

		yb := new(big.Rat)
		for i, c := range prob.Constraints {
			yb.Add(yb, new(big.Rat).Mul(y[i], c.RHS))
		}
		if yb.Sign() >= 0 {
			return fmt.Errorf("farkas condition violated: y^T b = %s >= 0 (expected < 0)", yb.String())
		}
		return nil
	}

	// Case 3: StatusUnbounded (Recession Ray d)
	if cert.RecessionRay != nil && cert.BasePoint != nil {
		x0 := cert.BasePoint
		d := cert.RecessionRay

		if len(d) != n || len(x0) != n {
			return fmt.Errorf("recession ray or base point dimension mismatch with variable count (%d)", n)
		}

		// d >= 0 and not all zero
		dZero := true
		for j := 0; j < n; j++ {
			if d[j].Sign() < 0 {
				return fmt.Errorf("recession ray component %d is negative: %s", j, d[j].String())
			}
			if d[j].Sign() > 0 {
				dZero = false
			}
		}
		if dZero {
			return fmt.Errorf("recession ray is entirely zero")
		}

		// A d <= 0 (for <= constraints)
		for i, c := range prob.Constraints {
			sum := new(big.Rat)
			for j := 0; j < n; j++ {
				sum.Add(sum, new(big.Rat).Mul(c.Coeffs[j], d[j]))
			}
			switch c.Sense {
			case RelLe:
				if sum.Sign() > 0 {
					return fmt.Errorf("recession ray violates A_i d <= 0 on constraint %d: %s > 0", i+1, sum.String())
				}
			case RelGe:
				if sum.Sign() < 0 {
					return fmt.Errorf("recession ray violates A_i d >= 0 on constraint %d: %s < 0", i+1, sum.String())
				}
			case RelEq:
				if sum.Sign() != 0 {
					return fmt.Errorf("recession ray violates A_i d == 0 on constraint %d: %s != 0", i+1, sum.String())
				}
			}
		}

		// c^T d > 0 for maximization, or < 0 for minimization
		cd := new(big.Rat)
		for j := 0; j < n; j++ {
			cd.Add(cd, new(big.Rat).Mul(prob.Objective[j], d[j]))
		}
		if prob.Sense == OptMaximize && cd.Sign() <= 0 {
			return fmt.Errorf("recession ray does not improve objective: c^T d = %s <= 0", cd.String())
		} else if prob.Sense == OptMinimize && cd.Sign() >= 0 {
			return fmt.Errorf("recession ray does not improve objective: c^T d = %s >= 0", cd.String())
		}

		return nil
	}

	return fmt.Errorf("incomplete certificate data")
}
