package units

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// UnitDef contains the dimensional and scaling definition of a physical unit.
type UnitDef struct {
	Name   string
	Symbol string
	Scale  *big.Rat
	Dims   ast.DimensionVector
}

var (
	registryOnce sync.Once
	unitMap      map[string]UnitDef
	prefixMap    map[string]*big.Rat
)

func initRegistry() {
	unitMap = make(map[string]UnitDef)
	prefixMap = make(map[string]*big.Rat)

	rat := func(num, denom int64) *big.Rat {
		return big.NewRat(num, denom)
	}

	pow10 := func(exp int) *big.Rat {
		if exp >= 0 {
			num := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exp)), nil)
			return new(big.Rat).SetInt(num)
		}
		den := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-exp)), nil)
		return new(big.Rat).SetFrac(big.NewInt(1), den)
	}

	// ---------------------------------------------------------------------
	// Prefixes
	// ---------------------------------------------------------------------
	prefixMap["Y"] = pow10(24)
	prefixMap["Z"] = pow10(21)
	prefixMap["E"] = pow10(18)
	prefixMap["P"] = pow10(15)
	prefixMap["T"] = pow10(12)
	prefixMap["G"] = pow10(9)
	prefixMap["M"] = pow10(6)
	prefixMap["k"] = pow10(3)
	prefixMap["h"] = pow10(2)
	prefixMap["da"] = pow10(1)
	prefixMap["d"] = pow10(-1)
	prefixMap["c"] = pow10(-2)
	prefixMap["m"] = pow10(-3)
	prefixMap["u"] = pow10(-6)
	prefixMap["n"] = pow10(-9)
	prefixMap["p"] = pow10(-12)
	prefixMap["f"] = pow10(-15)
	prefixMap["a"] = pow10(-18)

	// ---------------------------------------------------------------------
	// Base SI Units
	// ---------------------------------------------------------------------
	addBase := func(sym, name string, dimIdx int) {
		d := ast.NewDimensionVector()
		d[dimIdx] = rat(1, 1)
		unitMap[sym] = UnitDef{
			Name:   name,
			Symbol: sym,
			Scale:  rat(1, 1),
			Dims:   d,
		}
	}

	addBase("m", "meter", ast.DimLength)
	addBase("kg", "kilogram", ast.DimMass)
	addBase("s", "second", ast.DimTime)
	addBase("A", "ampere", ast.DimCurrent)
	addBase("K", "kelvin", ast.DimTemperature)
	addBase("mol", "mole", ast.DimAmount)
	addBase("cd", "candela", ast.DimLuminosity)

	// ---------------------------------------------------------------------
	// Derived SI Units
	// ---------------------------------------------------------------------
	addDerived := func(sym, name string, scale *big.Rat, dims ast.DimensionVector) {
		unitMap[sym] = UnitDef{
			Name:   name,
			Symbol: sym,
			Scale:  new(big.Rat).Set(scale),
			Dims:   dims.Clone(),
		}
	}

	// Frequency: Hz = 1/s
	dHz := ast.NewDimensionVector()
	dHz[ast.DimTime] = rat(-1, 1)
	addDerived("Hz", "hertz", rat(1, 1), dHz)

	// Force: N = kg * m / s^2
	dN := ast.NewDimensionVector()
	dN[ast.DimMass] = rat(1, 1)
	dN[ast.DimLength] = rat(1, 1)
	dN[ast.DimTime] = rat(-2, 1)
	addDerived("N", "newton", rat(1, 1), dN)

	// Pressure: Pa = N / m^2 = kg / (m * s^2)
	dPa := ast.NewDimensionVector()
	dPa[ast.DimMass] = rat(1, 1)
	dPa[ast.DimLength] = rat(-1, 1)
	dPa[ast.DimTime] = rat(-2, 1)
	addDerived("Pa", "pascal", rat(1, 1), dPa)

	// Energy: J = N * m = kg * m^2 / s^2
	dJ := ast.NewDimensionVector()
	dJ[ast.DimMass] = rat(1, 1)
	dJ[ast.DimLength] = rat(2, 1)
	dJ[ast.DimTime] = rat(-2, 1)
	addDerived("J", "joule", rat(1, 1), dJ)

	// Power: W = J / s = kg * m^2 / s^3
	dW := ast.NewDimensionVector()
	dW[ast.DimMass] = rat(1, 1)
	dW[ast.DimLength] = rat(2, 1)
	dW[ast.DimTime] = rat(-3, 1)
	addDerived("W", "watt", rat(1, 1), dW)

	// Electric Charge: C = A * s
	dC := ast.NewDimensionVector()
	dC[ast.DimCurrent] = rat(1, 1)
	dC[ast.DimTime] = rat(1, 1)
	addDerived("C", "coulomb", rat(1, 1), dC)

	// Voltage: V = W / A = kg * m^2 / (A * s^3)
	dV := ast.NewDimensionVector()
	dV[ast.DimMass] = rat(1, 1)
	dV[ast.DimLength] = rat(2, 1)
	dV[ast.DimCurrent] = rat(-1, 1)
	dV[ast.DimTime] = rat(-3, 1)
	addDerived("V", "volt", rat(1, 1), dV)

	// Capacitance: F = C / V = A^2 * s^4 / (kg * m^2)
	dF := ast.NewDimensionVector()
	dF[ast.DimCurrent] = rat(2, 1)
	dF[ast.DimTime] = rat(4, 1)
	dF[ast.DimMass] = rat(-1, 1)
	dF[ast.DimLength] = rat(-2, 1)
	addDerived("F", "farad", rat(1, 1), dF)

	// Resistance: ohm = V / A = kg * m^2 / (A^2 * s^3)
	dOhm := ast.NewDimensionVector()
	dOhm[ast.DimMass] = rat(1, 1)
	dOhm[ast.DimLength] = rat(2, 1)
	dOhm[ast.DimCurrent] = rat(-2, 1)
	dOhm[ast.DimTime] = rat(-3, 1)
	addDerived("ohm", "ohm", rat(1, 1), dOhm)

	// Conductance: S = 1 / ohm
	dS := dOhm.ScalarMul(rat(-1, 1))
	addDerived("S", "siemens", rat(1, 1), dS)

	// Magnetic Flux: Wb = V * s = kg * m^2 / (A * s^2)
	dWb := ast.NewDimensionVector()
	dWb[ast.DimMass] = rat(1, 1)
	dWb[ast.DimLength] = rat(2, 1)
	dWb[ast.DimCurrent] = rat(-1, 1)
	dWb[ast.DimTime] = rat(-2, 1)
	addDerived("Wb", "weber", rat(1, 1), dWb)

	// Magnetic Flux Density: T = Wb / m^2 = kg / (A * s^2)
	dT := ast.NewDimensionVector()
	dT[ast.DimMass] = rat(1, 1)
	dT[ast.DimCurrent] = rat(-1, 1)
	dT[ast.DimTime] = rat(-2, 1)
	addDerived("T", "tesla", rat(1, 1), dT)

	// Inductance: H = Wb / A = kg * m^2 / (A^2 * s^2)
	dH := ast.NewDimensionVector()
	dH[ast.DimMass] = rat(1, 1)
	dH[ast.DimLength] = rat(2, 1)
	dH[ast.DimCurrent] = rat(-2, 1)
	dH[ast.DimTime] = rat(-2, 1)
	addDerived("H", "henry", rat(1, 1), dH)

	// ---------------------------------------------------------------------
	// Accepted Non-SI Units
	// ---------------------------------------------------------------------
	// Time: min (60 s), h (3600 s), d (86400 s)
	dTime := ast.NewDimensionVector()
	dTime[ast.DimTime] = rat(1, 1)
	addDerived("min", "minute", rat(60, 1), dTime)
	addDerived("h", "hour", rat(3600, 1), dTime)
	addDerived("hour", "hour", rat(3600, 1), dTime)
	addDerived("d", "day", rat(86400, 1), dTime)
	addDerived("day", "day", rat(86400, 1), dTime)

	// Mass: g (1/1000 kg), t (1000 kg)
	dMass := ast.NewDimensionVector()
	dMass[ast.DimMass] = rat(1, 1)
	addDerived("g", "gram", rat(1, 1000), dMass)
	addDerived("t", "tonne", rat(1000, 1), dMass)

	// Volume: L (1/1000 m^3)
	dVol := ast.NewDimensionVector()
	dVol[ast.DimLength] = rat(3, 1)
	addDerived("L", "liter", rat(1, 1000), dVol)
	addDerived("l", "liter", rat(1, 1000), dVol)
}

// FindUnit searches the registry for an exact unit match.
func FindUnit(symbol string) (UnitDef, bool) {
	registryOnce.Do(initRegistry)
	u, ok := unitMap[symbol]
	return u, ok
}

// FindPrefix searches the prefix registry.
func FindPrefix(symbol string) (*big.Rat, bool) {
	registryOnce.Do(initRegistry)
	p, ok := prefixMap[symbol]
	return p, ok
}

// ResolveUnitSymbol resolves a unit string with optional SI prefix into an exact scale and dimension vector.
func ResolveUnitSymbol(raw string) (*big.Rat, ast.DimensionVector, error) {
	registryOnce.Do(initRegistry)

	// 1. Direct match (e.g. "min", "m", "kg", "N")
	if def, ok := unitMap[raw]; ok {
		return new(big.Rat).Set(def.Scale), def.Dims.Clone(), nil
	}

	// 2. Check 2-char prefix ("da")
	if len(raw) > 2 && raw[:2] == "da" {
		base := raw[2:]
		if def, ok := unitMap[base]; ok {
			scale := new(big.Rat).Mul(prefixMap["da"], def.Scale)
			return scale, def.Dims.Clone(), nil
		}
	}

	// 3. Check 1-char prefix
	if len(raw) > 1 {
		pfx := raw[:1]
		base := raw[1:]
		if pfxScale, hasPfx := prefixMap[pfx]; hasPfx {
			if def, ok := unitMap[base]; ok {
				scale := new(big.Rat).Mul(pfxScale, def.Scale)
				return scale, def.Dims.Clone(), nil
			}
		}
	}

	return nil, ast.NewDimensionVector(), fmt.Errorf("unknown physical unit '%s'", raw)
}
