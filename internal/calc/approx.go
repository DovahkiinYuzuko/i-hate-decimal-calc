package calc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// Approx evaluates an AST node into an approximate decimal string representation
// with a default certified precision of 30 digits.
func Approx(n Node) (string, error) {
	return ApproxWithDigits(n, 30)
}

// ApproxWithDigits evaluates an AST node into an approximate decimal string representation
// with the specified number of digits, accompanied by a certified outward-rounded error bound (± radius).
func ApproxWithDigits(n Node, digits int) (string, error) {
	if n == nil {
		return "", fmt.Errorf("%s", i18n.T("approx.err_cannot_approximate_nil_node"))
	}
	if digits < 1 {
		digits = 30
	}
	if digits > 500 {
		digits = 500
	}

	// 1. Structural traversal for ListNode
	if list, ok := n.(*ListNode); ok {
		appStrs := make([]string, len(list.Elements))
		for i, e := range list.Elements {
			str, err := ApproxWithDigits(e, digits)
			if err != nil {
				return "", err
			}
			appStrs[i] = str
		}
		return fmt.Sprintf("[%s]", strings.Join(appStrs, ", ")), nil
	}

	// 2. Structural traversal for MatrixNode
	if mat, ok := n.(*MatrixNode); ok {
		rowStrs := make([]string, mat.Rows)
		for r := 0; r < mat.Rows; r++ {
			elemStrs := make([]string, mat.Cols)
			for c := 0; c < mat.Cols; c++ {
				str, err := ApproxWithDigits(mat.Data[r][c], digits)
				if err != nil {
					return "", err
				}
				elemStrs[c] = str
			}
			rowStrs[r] = fmt.Sprintf("[%s]", strings.Join(elemStrs, ", "))
		}
		return fmt.Sprintf("[%s]", strings.Join(rowStrs, ", ")), nil
	}

	// 3. Structural traversal for RelOpNode (e.g. solve solutions like x == sqrt(2))
	if rel, ok := n.(*RelOpNode); ok {
		leftStr := rel.LHS.String()
		rightStr, err := ApproxWithDigits(rel.RHS, digits)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s %s", leftStr, rel.Op, rightStr), nil
	}

	// 4. Structural traversal for QuantityNode (e.g. physical units 100 [km/h])
	if q, ok := n.(*QuantityNode); ok {
		valStr, err := ApproxWithDigits(q.Value, digits)
		if err != nil {
			return "", err
		}
		if q.DisplayUnit != "" {
			return fmt.Sprintf("%s [%s]", valStr, q.DisplayUnit), nil
		}
		return valStr, nil
	}

	// 5. Complex numbers: ComplexNode
	if c, ok := n.(*ComplexNode); ok {
		rStr, errR := ApproxWithDigits(c.Real, digits)
		if errR != nil {
			return "", errR
		}
		imStr, errI := ApproxWithDigits(c.Imag, digits)
		if errI != nil {
			return "", errI
		}
		if imStr == "0" {
			return rStr, nil
		}
		if rStr == "0" {
			if imStr == "1" {
				return "i", nil
			}
			if imStr == "-1" {
				return "-i", nil
			}
			return fmt.Sprintf("%si", imStr), nil
		}
		if strings.HasPrefix(imStr, "-") {
			return fmt.Sprintf("%s - %si", rStr, strings.TrimPrefix(imStr, "-")), nil
		}
		return fmt.Sprintf("%s + %si", rStr, imStr), nil
	}

	// 6. Scalar / Real expressions: Adaptive interval refinement
	// Target tolerance: eps = 10^-(digits + 2)
	ten := big.NewInt(10)
	denom := new(big.Int).Exp(ten, big.NewInt(int64(digits+2)), nil)
	eps := new(big.Rat).SetFrac(big.NewInt(1), denom)

	maxSteps := digits * 3
	if maxSteps < 60 {
		maxSteps = 60
	}
	if maxSteps > 200 {
		maxSteps = 200
	}

	interval, err := AdaptiveRefineInterval(n, eps, maxSteps)
	if err != nil {
		return "", err
	}

	return formatApproxInterval(interval, digits), nil
}

// formatApproxInterval formats a rational interval [Low, High] into a center value
// rounded to 'digits' decimal places, plus a certified outward-rounded error bound (± radius) in scientific notation.
func formatApproxInterval(interval RationalInterval, digits int) string {
	low := interval.Low
	high := interval.High

	if low.Cmp(high) == 0 && low.IsInt() {
		return low.Num().String()
	}

	// Mid = (Low + High) / 2
	mid := new(big.Rat).Add(low, high)
	mid.Mul(mid, big.NewRat(1, 2))

	// Format Mid to 'digits' decimal places with round-half-to-even
	neg := mid.Sign() < 0
	absMid := new(big.Rat).Abs(mid)

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	p := new(big.Int).Mul(absMid.Num(), scale)
	q := absMid.Denom()

	quot := new(big.Int)
	rem := new(big.Int)
	quot.QuoRem(p, q, rem)

	// Round half to even:
	// 2 * rem vs q
	twoRem := new(big.Int).Mul(rem, big.NewInt(2))
	cmp := twoRem.Cmp(q)
	if cmp > 0 || (cmp == 0 && quot.Bit(0) == 1) {
		quot.Add(quot, big.NewInt(1))
	}

	// Split into integer and fractional parts
	quotStr := quot.String()
	var intPart, fracPart string
	if len(quotStr) <= digits {
		quotStr = strings.Repeat("0", digits-len(quotStr)+1) + quotStr
	}
	splitIdx := len(quotStr) - digits
	intPart = quotStr[:splitIdx]
	fracPart = quotStr[splitIdx:]

	midStr := intPart
	if digits > 0 {
		midStr += "." + fracPart
	}
	if neg && (intPart != "0" || strings.Trim(fracPart, "0") != "") {
		midStr = "-" + midStr
	}

	// Now compute the exact rational value of midStr
	mRat := new(big.Rat)
	mRat.SetString(midStr)

	// Certified radius bound = max(|mRat - Low|, |mRat - High|)
	dLow := new(big.Rat).Abs(new(big.Rat).Sub(mRat, low))
	dHigh := new(big.Rat).Abs(new(big.Rat).Sub(mRat, high))
	maxDist := dLow
	if dHigh.Cmp(maxDist) > 0 {
		maxDist = dHigh
	}

	if maxDist.Sign() == 0 {
		return midStr
	}

	boundStr := formatCeilScientific(maxDist)
	return fmt.Sprintf("%s (± %s)", midStr, boundStr)
}

func formatCeilScientific(r *big.Rat) string {
	if r.Sign() <= 0 {
		return "0"
	}
	num := r.Num()
	den := r.Denom()

	ten := big.NewInt(10)
	kEst := int(float64(num.BitLen()-den.BitLen()) * 0.3010299956639812)
	k := kEst - 1

	for {
		var pNum, pDen *big.Int
		if k >= 0 {
			pNum = new(big.Int).Exp(ten, big.NewInt(int64(k+1)), nil)
			pDen = big.NewInt(1)
		} else {
			pNum = big.NewInt(1)
			pDen = new(big.Int).Exp(ten, big.NewInt(int64(-k-1)), nil)
		}
		left := new(big.Int).Mul(num, pDen)
		right := new(big.Int).Mul(pNum, den)
		if left.Cmp(right) < 0 {
			var pNum0, pDen0 *big.Int
			if k >= 0 {
				pNum0 = new(big.Int).Exp(ten, big.NewInt(int64(k)), nil)
				pDen0 = big.NewInt(1)
			} else {
				pNum0 = big.NewInt(1)
				pDen0 = new(big.Int).Exp(ten, big.NewInt(int64(-k)), nil)
			}
			l0 := new(big.Int).Mul(num, pDen0)
			r0 := new(big.Int).Mul(pNum0, den)
			if l0.Cmp(r0) >= 0 {
				break
			}
			k--
		} else {
			k++
		}
	}

	// 2 significant digits for the bound: C = ceil(r * 10^(1-k))
	shift := 1 - k
	var sNum, sDen *big.Int
	if shift >= 0 {
		sNum = new(big.Int).Mul(num, new(big.Int).Exp(ten, big.NewInt(int64(shift)), nil))
		sDen = den
	} else {
		sNum = num
		sDen = new(big.Int).Mul(den, new(big.Int).Exp(ten, big.NewInt(int64(-shift)), nil))
	}
	cInt := new(big.Int).Add(sNum, sDen)
	cInt.Sub(cInt, big.NewInt(1))
	cInt.Quo(cInt, sDen)

	cVal := cInt.Int64()
	if cVal >= 100 {
		cVal = 10
		k++
	}

	d1 := cVal / 10
	d2 := cVal % 10
	return fmt.Sprintf("%d.%de%+d", d1, d2, k)
}
