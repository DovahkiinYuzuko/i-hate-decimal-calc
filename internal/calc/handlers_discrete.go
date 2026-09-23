package calc

import (
	"fmt"
	"math/big"
	"math/rand/v2"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func init() {
	// Discrete math
	RegisterHandler("gcd", handleGCD)
	RegisterHandler("lcm", handleLCM)
	RegisterHandler("mod", handleMod)
	RegisterHandler("perm", handlePerm)
	RegisterHandler("comb", handleComb)
	RegisterHandler("rand", handleRand)
	RegisterHandler("inv_mod", handleInvMod)
	RegisterHandler("crt", handleCRT)
	RegisterHandler("totient", handleTotient)
	RegisterHandler("is_prime", handleIsPrime)

	// Complex representation
	RegisterHandler("arg", handleArg)
	RegisterHandler("polar", handlePolar)
	RegisterHandler("polar_exp", handlePolarExp)
	RegisterHandler("rect", handleRect)

	// Probability & Statistics
	RegisterHandler("binom", handleBinom)
	RegisterHandler("hyper", handleHyper)
	RegisterHandler("geom", handleGeom)
	RegisterHandler("bayes", handleBayes)
	RegisterHandler("expect", handleExpect)
	RegisterHandler("variance", handleVariance)
	RegisterHandler("stddev", handleStdDev)

	// Continued fractions
	RegisterHandler("cfrac", handleCFrac)
	RegisterHandler("from_cfrac", handleFromCFrac)

	// Geometry
	RegisterHandler("line_intersect", handleLineIntersect)
	RegisterHandler("circle_intersect", handleCircleIntersect)
	RegisterHandler("triangle_area", handleTriangleArea)
	RegisterHandler("triangle_centers", handleTriangleCenters)

	// Visualization
	RegisterHandler("plot", handlePlot)
}

func handleGCD(args []Node, env *Env) (Node, error) {
	aRat, aOk := args[0].(*RationalNode)
	bRat, bOk := args[1].(*RationalNode)
	if aOk && bOk {
		if !aRat.Val.IsInt() || !bRat.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_int_args", "gcd", aRat.String(), bRat.String()))
		}
		g := new(big.Int).GCD(nil, nil, aRat.Val.Num(), bRat.Val.Num())
		return NewRationalFromBigRat(new(big.Rat).SetInt(g)), nil
	}
	return NewFunc("gcd", args)
}

func handleLCM(args []Node, env *Env) (Node, error) {
	aRat, aOk := args[0].(*RationalNode)
	bRat, bOk := args[1].(*RationalNode)
	if aOk && bOk {
		if !aRat.Val.IsInt() || !bRat.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_int_args", "lcm", aRat.String(), bRat.String()))
		}
		if aRat.Val.Sign() == 0 || bRat.Val.Sign() == 0 {
			return mustRational(0, 1), nil
		}
		aInt := aRat.Val.Num()
		bInt := bRat.Val.Num()
		g := new(big.Int).GCD(nil, nil, aInt, bInt)
		div := new(big.Int).Div(aInt, g)
		l := new(big.Int).Mul(div, bInt)
		l.Abs(l)
		return NewRationalFromBigRat(new(big.Rat).SetInt(l)), nil
	}
	return NewFunc("lcm", args)
}

func handleMod(args []Node, env *Env) (Node, error) {
	aRat, aOk := args[0].(*RationalNode)
	bRat, bOk := args[1].(*RationalNode)
	if aOk && bOk {
		if !aRat.Val.IsInt() || !bRat.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_int_args", "mod", aRat.String(), bRat.String()))
		}
		if bRat.Val.Sign() == 0 {
			return nil, NewZeroDivisionError("division by zero in mod")
		}
		m := new(big.Int).Mod(aRat.Val.Num(), bRat.Val.Num())
		return NewRationalFromBigRat(new(big.Rat).SetInt(m)), nil
	}
	return NewFunc("mod", args)
}

func handlePerm(args []Node, env *Env) (Node, error) {
	nRat, nOk := args[0].(*RationalNode)
	rRat, rOk := args[1].(*RationalNode)
	if nOk && rOk {
		if !nRat.Val.IsInt() || !rRat.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_int_args", "perm", nRat.String(), rRat.String()))
		}
		if nRat.Val.Sign() < 0 || rRat.Val.Sign() < 0 {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_non_negative_args", "perm", nRat.String(), rRat.String()))
		}
		nVal := nRat.Val.Num()
		rVal := rRat.Val.Num()
		if rVal.Cmp(nVal) > 0 {
			return mustRational(0, 1), nil
		}
		res := big.NewInt(1)
		curr := new(big.Int).Set(nVal)
		count := rVal.Int64()
		one := big.NewInt(1)
		for i := int64(0); i < count; i++ {
			res.Mul(res, curr)
			curr.Sub(curr, one)
		}
		return NewRationalFromBigRat(new(big.Rat).SetInt(res)), nil
	}
	return NewFunc("perm", args)
}

func handleComb(args []Node, env *Env) (Node, error) {
	nRat, nOk := args[0].(*RationalNode)
	rRat, rOk := args[1].(*RationalNode)
	if nOk && rOk {
		if !nRat.Val.IsInt() || !rRat.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_int_args", "comb", nRat.String(), rRat.String()))
		}
		if nRat.Val.Sign() < 0 || rRat.Val.Sign() < 0 {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_non_negative_args", "comb", nRat.String(), rRat.String()))
		}
		nVal := nRat.Val.Num()
		rVal := rRat.Val.Num()
		if rVal.Cmp(nVal) > 0 {
			return mustRational(0, 1), nil
		}
		res := new(big.Int).Binomial(nVal.Int64(), rVal.Int64())
		return NewRationalFromBigRat(new(big.Rat).SetInt(res)), nil
	}
	return NewFunc("comb", args)
}

func handleRand(args []Node, env *Env) (Node, error) {
	if len(args) == 1 {
		maxRat, ok := args[0].(*RationalNode)
		if !ok || !maxRat.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_rand_max_int", args[0].String()))
		}
		maxVal := maxRat.Val.Num().Int64()
		if maxVal < 0 {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_rand_max_non_negative", maxVal))
		}
		if maxVal == 0 {
			return mustRational(0, 1), nil
		}
		val := rand.Int64N(maxVal + 1)
		return mustRational(val, 1), nil
	} else if len(args) == 2 {
		minRat, minOk := args[0].(*RationalNode)
		maxRat, maxOk := args[1].(*RationalNode)
		if !minOk || !maxOk || !minRat.Val.IsInt() || !maxRat.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_rand_min_max_int", args[0].String(), args[1].String()))
		}
		minVal := minRat.Val.Num().Int64()
		maxVal := maxRat.Val.Num().Int64()
		if minVal > maxVal {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_rand_min_gt_max", minVal, maxVal))
		}
		delta := maxVal - minVal + 1
		val := minVal + rand.Int64N(delta)
		return mustRational(val, 1), nil
	} else if len(args) == 3 {
		seedRat, sOk := args[0].(*RationalNode)
		minRat, minOk := args[1].(*RationalNode)
		maxRat, maxOk := args[2].(*RationalNode)
		if !sOk || !minOk || !maxOk || !seedRat.Val.IsInt() || !minRat.Val.IsInt() || !maxRat.Val.IsInt() {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_rand_seed_min_max_int"))
		}
		seedVal := seedRat.Val.Num().Int64()
		minVal := minRat.Val.Num().Int64()
		maxVal := maxRat.Val.Num().Int64()
		if minVal > maxVal {
			return nil, fmt.Errorf("%s", i18n.T("errors.domain_rand_min_gt_max", minVal, maxVal))
		}
		delta := maxVal - minVal + 1
		rng := rand.New(rand.NewPCG(uint64(seedVal), 1))
		val := minVal + rng.Int64N(delta)
		return mustRational(val, 1), nil
	}
	return NewFunc("rand", args)
}

func handleInvMod(args []Node, env *Env) (Node, error) {
	return EvalInvMod(args[0], args[1])
}

func handleCRT(args []Node, env *Env) (Node, error) {
	return EvalCRT(args)
}

func handleTotient(args []Node, env *Env) (Node, error) {
	return EvalTotient(args[0])
}

func handleIsPrime(args []Node, env *Env) (Node, error) {
	return EvalIsPrime(args[0])
}

func handleArg(args []Node, env *Env) (Node, error) {
	return evalArg(args[0])
}

func handlePolar(args []Node, env *Env) (Node, error) {
	return evalPolar(args[0])
}

func handlePolarExp(args []Node, env *Env) (Node, error) {
	return evalPolarExp(args[0])
}

func handleRect(args []Node, env *Env) (Node, error) {
	return evalRect(args[0], args[1])
}

func handleBinom(args []Node, env *Env) (Node, error) {
	return EvalBinomPMF(args)
}

func handleHyper(args []Node, env *Env) (Node, error) {
	return EvalHyperPMF(args)
}

func handleGeom(args []Node, env *Env) (Node, error) {
	return EvalGeomPMF(args)
}

func handleBayes(args []Node, env *Env) (Node, error) {
	return EvalBayes(args)
}

func handleExpect(args []Node, env *Env) (Node, error) {
	return EvalExpect(args)
}

func handleVariance(args []Node, env *Env) (Node, error) {
	return EvalVariance(args)
}

func handleStdDev(args []Node, env *Env) (Node, error) {
	return EvalStdDev(args)
}

func handleCFrac(args []Node, env *Env) (Node, error) {
	return evalCFrac(args[0])
}

func handleFromCFrac(args []Node, env *Env) (Node, error) {
	return evalFromCFrac(args[0])
}

func handleLineIntersect(args []Node, env *Env) (Node, error) {
	return evalLineIntersectFunc(args[0], args[1])
}

func handleCircleIntersect(args []Node, env *Env) (Node, error) {
	return evalCircleIntersectFunc(args[0], args[1], args[2], args[3])
}

func handleTriangleArea(args []Node, env *Env) (Node, error) {
	return evalTriangleAreaFunc(args[0], args[1], args[2])
}

func handleTriangleCenters(args []Node, env *Env) (Node, error) {
	return evalTriangleCentersFunc(args[0], args[1], args[2])
}

func handlePlot(args []Node, env *Env) (Node, error) {
	return EvalPlot(args)
}
