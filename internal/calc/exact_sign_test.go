package calc

import (
	"math/big"
	"testing"
)

func TestExactSign_Rational(t *testing.T) {
	tests := []struct {
		name     string
		node     Node
		expected Sign
	}{
		{"positive integer", mustRational(5, 1), SignPositive},
		{"positive fraction", mustRational(1, 3), SignPositive},
		{"negative integer", mustRational(-5, 1), SignNegative},
		{"negative fraction", mustRational(-1, 3), SignNegative},
		{"zero", mustRational(0, 1), SignZero},
		{"micro positive fraction (10^-15)", mustRational(1, 1000000000000000), SignPositive},
		{"micro negative fraction (-10^-15)", mustRational(-1, 1000000000000000), SignNegative},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ExactSign(tt.node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, s)
			}
		})
	}
}

func TestExactSign_Radical(t *testing.T) {
	sqrt2 := NewSqrt(mustRational(2, 1))
	negSqrt2 := NewMul([]Node{mustRational(-1, 1), sqrt2})

	s1, err := ExactSign(sqrt2)
	if err != nil || s1 != SignPositive {
		t.Errorf("expected sqrt(2) to be Positive, got %v (err: %v)", s1, err)
	}

	s2, err := ExactSign(negSqrt2)
	if err != nil || s2 != SignNegative {
		t.Errorf("expected -sqrt(2) to be Negative, got %v (err: %v)", s2, err)
	}

	s0, err := ExactSign(NewSqrt(mustRational(0, 1)))
	if err != nil || s0 != SignZero {
		t.Errorf("expected sqrt(0) to be Zero, got %v (err: %v)", s0, err)
	}
}

func TestExactSign_QuadraticIrrationals(t *testing.T) {
	// sqrt(2) - 1 > 0 (2 > 1)
	sqrt2Minus1 := NewAdd([]Node{
		NewSqrt(mustRational(2, 1)),
		mustRational(-1, 1),
	})
	s1, err := ExactSign(sqrt2Minus1)
	if err != nil || s1 != SignPositive {
		t.Errorf("expected sqrt(2) - 1 to be Positive, got %v (err: %v)", s1, err)
	}

	// 1 - sqrt(2) < 0 (1 < 2)
	oneMinusSqrt2 := NewAdd([]Node{
		mustRational(1, 1),
		NewMul([]Node{mustRational(-1, 1), NewSqrt(mustRational(2, 1))}),
	})
	s2, err := ExactSign(oneMinusSqrt2)
	if err != nil || s2 != SignNegative {
		t.Errorf("expected 1 - sqrt(2) to be Negative, got %v (err: %v)", s2, err)
	}

	// 2*sqrt(3) - 3*sqrt(2) < 0 (4*3 = 12 < 9*2 = 18)
	twoSqrt3MinusThreeSqrt2 := NewAdd([]Node{
		NewMul([]Node{mustRational(2, 1), NewSqrt(mustRational(3, 1))}),
		NewMul([]Node{mustRational(-3, 1), NewSqrt(mustRational(2, 1))}),
	})
	s3, err := ExactSign(twoSqrt3MinusThreeSqrt2)
	if err != nil || s3 != SignNegative {
		t.Errorf("expected 2*sqrt(3) - 3*sqrt(2) to be Negative, got %v (err: %v)", s3, err)
	}

	// 3*sqrt(2) - 2*sqrt(3) > 0 (18 > 12)
	threeSqrt2MinusTwoSqrt3 := NewAdd([]Node{
		NewMul([]Node{mustRational(3, 1), NewSqrt(mustRational(2, 1))}),
		NewMul([]Node{mustRational(-2, 1), NewSqrt(mustRational(3, 1))}),
	})
	s4, err := ExactSign(threeSqrt2MinusTwoSqrt3)
	if err != nil || s4 != SignPositive {
		t.Errorf("expected 3*sqrt(2) - 2*sqrt(3) to be Positive, got %v (err: %v)", s4, err)
	}

	// 5*sqrt(2) - 7 > 0 (50 > 49)
	fiveSqrt2MinusSeven := NewAdd([]Node{
		NewMul([]Node{mustRational(5, 1), NewSqrt(mustRational(2, 1))}),
		mustRational(-7, 1),
	})
	s5, err := ExactSign(fiveSqrt2MinusSeven)
	if err != nil || s5 != SignPositive {
		t.Errorf("expected 5*sqrt(2) - 7 to be Positive, got %v (err: %v)", s5, err)
	}

	// 7 - 5*sqrt(2) < 0 (49 < 50)
	sevenMinusFiveSqrt2 := NewAdd([]Node{
		mustRational(7, 1),
		NewMul([]Node{mustRational(-5, 1), NewSqrt(mustRational(2, 1))}),
	})
	s6, err := ExactSign(sevenMinusFiveSqrt2)
	if err != nil || s6 != SignNegative {
		t.Errorf("expected 7 - 5*sqrt(2) to be Negative, got %v (err: %v)", s6, err)
	}
}

func TestExactSign_Geometry_MicroEpsilonSecant(t *testing.T) {
	// Circle: x^2 + y^2 = 1 (Center (0, 0), Radius 1)
	c := circle(0, 1, 0, 1, 1, 1)

	// Tangent line: y = 1 => 0*x + 1*y - 1 = 0
	lTan := line(0, 1, 1, 1, -1, 1)
	ptsTan, err := IntersectLineCircle(lTan, c)
	if err != nil {
		t.Fatalf("unexpected error for tangent line: %v", err)
	}
	if len(ptsTan) != 1 {
		t.Fatalf("expected 1 tangent point, got %d", len(ptsTan))
	}

	// Secant line with micro offset: y = 1 - 10^-14
	// In the old float implementation with 1e-12 epsilon, this would falsely be classified as tangent!
	// With ExactSign, discriminant is exactly positive, yielding 2 intersection points.
	// y = 1 - 1/10^14 = (10^14 - 1) / 10^14
	tenPow14 := new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil)
	numVal := new(big.Int).Sub(tenPow14, big.NewInt(1))
	yConstRat := new(big.Rat).SetFrac(numVal, tenPow14)
	negYConst := new(big.Rat).Neg(yConstRat)

	lMicroSecant := Line2D{
		A: mustRational(0, 1),
		B: mustRational(1, 1),
		C: &RationalNode{Val: negYConst},
	}

	ptsMicroSecant, err := IntersectLineCircle(lMicroSecant, c)
	if err != nil {
		t.Fatalf("unexpected error for micro secant line: %v", err)
	}
	if len(ptsMicroSecant) != 2 {
		t.Fatalf("expected 2 intersection points for micro secant, got %d (exact sign must prevent false tangent)", len(ptsMicroSecant))
	}

	// Disjoint line with micro offset: y = 1 + 10^-14
	numDisVal := new(big.Int).Add(tenPow14, big.NewInt(1))
	yDisRat := new(big.Rat).SetFrac(numDisVal, tenPow14)
	negDisConst := new(big.Rat).Neg(yDisRat)

	lMicroDisjoint := Line2D{
		A: mustRational(0, 1),
		B: mustRational(1, 1),
		C: &RationalNode{Val: negDisConst},
	}

	ptsMicroDisjoint, err := IntersectLineCircle(lMicroDisjoint, c)
	if err != nil {
		t.Fatalf("unexpected error for micro disjoint line: %v", err)
	}
	if len(ptsMicroDisjoint) != 0 {
		t.Fatalf("expected 0 intersection points for micro disjoint, got %d", len(ptsMicroDisjoint))
	}
}
