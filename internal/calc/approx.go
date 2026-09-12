package calc

import (
	"fmt"
	"math"
	"math/cmplx"
	"strings"
)

// Approx evaluates an AST node into an approximate decimal string representation.
func Approx(n Node) (string, error) {
	if n == nil {
		return "", fmt.Errorf("cannot approximate nil node")
	}
	if list, ok := n.(*ListNode); ok {
		appStrs := make([]string, len(list.Elements))
		for i, e := range list.Elements {
			str, err := Approx(e)
			if err != nil {
				return "", err
			}
			appStrs[i] = str
		}
		return fmt.Sprintf("[%s]", strings.Join(appStrs, ", ")), nil
	}
	if mat, ok := n.(*MatrixNode); ok {
		rowStrs := make([]string, mat.Rows)
		for r := 0; r < mat.Rows; r++ {
			elemStrs := make([]string, mat.Cols)
			for c := 0; c < mat.Cols; c++ {
				str, err := Approx(mat.Data[r][c])
				if err != nil {
					return "", err
				}
				elemStrs[c] = str
			}
			rowStrs[r] = fmt.Sprintf("[%s]", strings.Join(elemStrs, ", "))
		}
		return fmt.Sprintf("[%s]", strings.Join(rowStrs, ", ")), nil
	}
	c, err := evalComplex(n)
	if err != nil {
		return "", err
	}
	return formatApprox(c), nil
}

func formatApprox(c complex128) string {
	r := real(c)
	im := imag(c)

	// Suppress tiny floating-point residue
	if math.Abs(im) < 1e-14 {
		return fmt.Sprintf("%.16g", r)
	}
	if math.Abs(r) < 1e-14 {
		if im == 1 {
			return "i"
		}
		if im == -1 {
			return "-i"
		}
		return fmt.Sprintf("%.16gi", im)
	}

	if im < 0 {
		if im == -1 {
			return fmt.Sprintf("%.16g - i", r)
		}
		return fmt.Sprintf("%.16g - %.16gi", r, -im)
	}
	if im == 1 {
		return fmt.Sprintf("%.16g + i", r)
	}
	return fmt.Sprintf("%.16g + %.16gi", r, im)
}

func evalFloat(n Node) (float64, error) {
	c, err := evalComplex(n)
	if err != nil {
		return 0, err
	}
	if math.Abs(imag(c)) > 1e-14 {
		return 0, fmt.Errorf("result is complex, not a real number: %v", c)
	}
	return real(c), nil
}

func evalComplex(n Node) (complex128, error) {
	switch v := n.(type) {
	case *RationalNode:
		f, _ := v.Val.Float64()
		return complex(f, 0), nil

	case *ConstNode:
		switch v.Name {
		case "pi":
			return complex(math.Pi, 0), nil
		case "e":
			return complex(math.E, 0), nil
		case "deg":
			return complex(math.Pi/180.0, 0), nil
		case "i":
			return complex(0, 1), nil
		default:
			return 0, fmt.Errorf("unknown constant: %s", v.Name)
		}

	case *SqrtNode:
		rad, err := evalComplex(v.Radicand)
		if err != nil {
			return 0, err
		}
		return cmplx.Sqrt(rad), nil

	case *FuncNode:
		if len(v.Args) == 1 {
			arg, err := evalComplex(v.Args[0])
			if err != nil {
				return 0, err
			}
			switch v.Name {
			case "sin":
				return cmplx.Sin(arg), nil
			case "cos":
				return cmplx.Cos(arg), nil
			case "tan":
				return cmplx.Tan(arg), nil
			case "asin":
				return cmplx.Asin(arg), nil
			case "acos":
				return cmplx.Acos(arg), nil
			case "atan":
				return cmplx.Atan(arg), nil
			case "ln":
				return cmplx.Log(arg), nil
			case "log":
				// Single arg log(x) is log10(x)
				return cmplx.Log10(arg), nil
			case "abs":
				return complex(cmplx.Abs(arg), 0), nil
			case "cbrt":
				if math.Abs(imag(arg)) < 1e-14 {
					r := real(arg)
					if r >= 0 {
						return complex(math.Cbrt(r), 0), nil
					}
					return complex(-math.Cbrt(-r), 0), nil
				}
				return cmplx.Pow(arg, complex(1.0/3.0, 0)), nil
			default:
				return 0, fmt.Errorf("unknown function: %s", v.Name)
			}
		} else if len(v.Args) == 2 && v.Name == "log" {
			base, err := evalComplex(v.Args[0])
			if err != nil {
				return 0, err
			}
			arg, err := evalComplex(v.Args[1])
			if err != nil {
				return 0, err
			}
			// log_b(x) = ln(x) / ln(b)
			lnBase := cmplx.Log(base)
			if lnBase == 0 {
				return 0, fmt.Errorf("division by zero in log base")
			}
			return cmplx.Log(arg) / lnBase, nil
		}
		return 0, fmt.Errorf("invalid function call: %s", v.Name)

	case *ComplexNode:
		r, err := evalFloat(v.Real)
		if err != nil {
			return 0, err
		}
		im, err := evalFloat(v.Imag)
		if err != nil {
			return 0, err
		}
		return complex(r, im), nil

	case *UnaryOpNode:
		val, err := evalComplex(v.Expr)
		if err != nil {
			return 0, err
		}
		if v.Op == "-" {
			return -val, nil
		}
		if v.Op == "!" {
			if math.Abs(imag(val)) > 1e-14 || real(val) < 0 || math.Floor(real(val)) != real(val) {
				return 0, fmt.Errorf("factorial requires non-negative integer")
			}
			n := int64(real(val))
			res := 1.0
			for i := int64(2); i <= n; i++ {
				res *= float64(i)
			}
			return complex(res, 0), nil
		}
		return 0, fmt.Errorf("unknown unary operator: %s", v.Op)

	case *PowNode:
		base, err := evalComplex(v.Base)
		if err != nil {
			return 0, err
		}
		exp, err := evalComplex(v.Exp)
		if err != nil {
			return 0, err
		}
		return cmplx.Pow(base, exp), nil

	case *MulNode:
		prod := complex(1, 0)
		for _, f := range v.Factors {
			val, err := evalComplex(f)
			if err != nil {
				return 0, err
			}
			prod *= val
		}
		return prod, nil

	case *AddNode:
		sum := complex(0, 0)
		for _, t := range v.Terms {
			val, err := evalComplex(t)
			if err != nil {
				return 0, err
			}
			sum += val
		}
		return sum, nil

	default:
		return 0, fmt.Errorf("cannot approximate node type: %T", n)
	}
}
