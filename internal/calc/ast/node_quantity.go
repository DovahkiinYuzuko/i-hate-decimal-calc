package ast

import (
	"fmt"
	"math/big"
	"strings"
)

// Base SI Dimension Indices
const (
	DimLength      = 0 // Length: m (meter)
	DimMass        = 1 // Mass: kg (kilogram)
	DimTime        = 2 // Time: s (second)
	DimCurrent     = 3 // Electric Current: A (ampere)
	DimTemperature = 4 // Thermodynamic Temperature: K (kelvin)
	DimAmount      = 5 // Amount of Substance: mol (mole)
	DimLuminosity  = 6 // Luminous Intensity: cd (candela)
)

var BaseUnitSymbols = [7]string{
	"m",
	"kg",
	"s",
	"A",
	"K",
	"mol",
	"cd",
}

// DimensionVector represents a 7-dimensional rational exponent vector over Q^7.
type DimensionVector [7]*big.Rat

// NewDimensionVector creates a zero-initialized DimensionVector.
func NewDimensionVector() DimensionVector {
	var dv DimensionVector
	for i := 0; i < 7; i++ {
		dv[i] = new(big.Rat)
	}
	return dv
}

// Clone creates a deep copy of the DimensionVector.
func (dv DimensionVector) Clone() DimensionVector {
	var res DimensionVector
	for i := 0; i < 7; i++ {
		if dv[i] != nil {
			res[i] = new(big.Rat).Set(dv[i])
		} else {
			res[i] = new(big.Rat)
		}
	}
	return res
}

// Add computes dv + other (vector addition of exponents).
func (dv DimensionVector) Add(other DimensionVector) DimensionVector {
	res := NewDimensionVector()
	for i := 0; i < 7; i++ {
		v1 := dv[i]
		if v1 == nil {
			v1 = new(big.Rat)
		}
		v2 := other[i]
		if v2 == nil {
			v2 = new(big.Rat)
		}
		res[i].Add(v1, v2)
	}
	return res
}

// Sub computes dv - other (vector subtraction of exponents).
func (dv DimensionVector) Sub(other DimensionVector) DimensionVector {
	res := NewDimensionVector()
	for i := 0; i < 7; i++ {
		v1 := dv[i]
		if v1 == nil {
			v1 = new(big.Rat)
		}
		v2 := other[i]
		if v2 == nil {
			v2 = new(big.Rat)
		}
		res[i].Sub(v1, v2)
	}
	return res
}

// ScalarMul multiplies each exponent component by a rational scalar.
func (dv DimensionVector) ScalarMul(scalar *big.Rat) DimensionVector {
	res := NewDimensionVector()
	if scalar == nil {
		return res
	}
	for i := 0; i < 7; i++ {
		if dv[i] != nil {
			res[i].Mul(dv[i], scalar)
		}
	}
	return res
}

// Equal returns true if all 7 dimension exponents match exactly.
func (dv DimensionVector) Equal(other DimensionVector) bool {
	for i := 0; i < 7; i++ {
		v1 := dv[i]
		if v1 == nil {
			v1 = new(big.Rat)
		}
		v2 := other[i]
		if v2 == nil {
			v2 = new(big.Rat)
		}
		if v1.Cmp(v2) != 0 {
			return false
		}
	}
	return true
}

// IsZero returns true if all 7 exponents are 0 (dimensionless quantity).
func (dv DimensionVector) IsZero() bool {
	for i := 0; i < 7; i++ {
		if dv[i] != nil && dv[i].Sign() != 0 {
			return false
		}
	}
	return true
}

func (dv DimensionVector) String() string {
	if dv.IsZero() {
		return "1"
	}
	var posParts []string
	var negParts []string

	for i := 0; i < 7; i++ {
		exp := dv[i]
		if exp == nil || exp.Sign() == 0 {
			continue
		}
		unit := BaseUnitSymbols[i]
		if exp.Sign() > 0 {
			if exp.Cmp(big.NewRat(1, 1)) == 0 {
				posParts = append(posParts, unit)
			} else {
				posParts = append(posParts, fmt.Sprintf("%s^(%s)", unit, exp.RatString()))
			}
		} else {
			absExp := new(big.Rat).Abs(exp)
			if absExp.Cmp(big.NewRat(1, 1)) == 0 {
				negParts = append(negParts, unit)
			} else {
				negParts = append(negParts, fmt.Sprintf("%s^(%s)", unit, absExp.RatString()))
			}
		}
	}

	numStr := strings.Join(posParts, "*")
	if len(numStr) == 0 {
		numStr = "1"
	}
	if len(negParts) == 0 {
		return numStr
	}
	denStr := strings.Join(negParts, "*")
	if len(negParts) > 1 {
		denStr = "(" + denStr + ")"
	}
	return numStr + "/" + denStr
}

// -------------------------------------------------------------------------
// QuantityNode
// -------------------------------------------------------------------------

// QuantityNode represents a physical quantity with a magnitude, scaling factor, and dimension vector.
type QuantityNode struct {
	Value       Node
	Scale       *big.Rat
	Dimensions  DimensionVector
	DisplayUnit string
}

// NewQuantityNode creates a new QuantityNode.
func NewQuantityNode(val Node, scale *big.Rat, dims DimensionVector, displayUnit string) *QuantityNode {
	s := scale
	if s == nil {
		s = big.NewRat(1, 1)
	} else {
		s = new(big.Rat).Set(scale)
	}
	return &QuantityNode{
		Value:       val,
		Scale:       s,
		Dimensions:  dims.Clone(),
		DisplayUnit: displayUnit,
	}
}

func (q *QuantityNode) Type() NodeType { return NodeQuantity }

func (q *QuantityNode) String() string {
	unitStr := q.DisplayUnit
	if unitStr == "" {
		unitStr = q.Dimensions.String()
	}
	return fmt.Sprintf("%s [%s]", q.Value.String(), unitStr)
}

func (q *QuantityNode) Equal(other Node) bool {
	o, ok := other.(*QuantityNode)
	if !ok {
		return false
	}
	if !q.Dimensions.Equal(o.Dimensions) {
		return false
	}
	if q.Scale.Cmp(o.Scale) != 0 {
		return false
	}
	return q.Value.Equal(o.Value)
}
