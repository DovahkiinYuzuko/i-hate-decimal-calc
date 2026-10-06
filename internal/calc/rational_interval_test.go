package calc

import (
	"math/big"
	"testing"
)

func TestRationalInterval_BasicArithmetic(t *testing.T) {
	// I1 = [1, 2], I2 = [3, 4]
	i1 := NewRationalInterval(big.NewRat(1, 1), big.NewRat(2, 1))
	i2 := NewRationalInterval(big.NewRat(3, 1), big.NewRat(4, 1))

	// Add: [4, 6]
	add := i1.Add(i2)
	if add.Low.Cmp(big.NewRat(4, 1)) != 0 || add.High.Cmp(big.NewRat(6, 1)) != 0 {
		t.Errorf("Add failed: expected [4, 6], got %s", add.String())
	}

	// Sub: [1-4, 2-3] = [-3, -1]
	sub := i1.Sub(i2)
	if sub.Low.Cmp(big.NewRat(-3, 1)) != 0 || sub.High.Cmp(big.NewRat(-1, 1)) != 0 {
		t.Errorf("Sub failed: expected [-3, -1], got %s", sub.String())
	}

	// Mul: [3, 8]
	mul := i1.Mul(i2)
	if mul.Low.Cmp(big.NewRat(3, 1)) != 0 || mul.High.Cmp(big.NewRat(8, 1)) != 0 {
		t.Errorf("Mul failed: expected [3, 8], got %s", mul.String())
	}

	// Inv: [1/4, 1/3]
	inv2, err := i2.Inv()
	if err != nil {
		t.Fatalf("Inv failed: %v", err)
	}
	if inv2.Low.Cmp(big.NewRat(1, 4)) != 0 || inv2.High.Cmp(big.NewRat(1, 3)) != 0 {
		t.Errorf("Inv failed: expected [1/4, 1/3], got %s", inv2.String())
	}

	// Zero division rejection
	zeroInterval := NewRationalInterval(big.NewRat(-1, 1), big.NewRat(1, 1))
	if _, err := zeroInterval.Inv(); err == nil {
		t.Errorf("expected error for interval containing zero, got nil")
	}

	// PowInt
	pow3 := i1.PowInt(3) // [1, 8]
	if pow3.Low.Cmp(big.NewRat(1, 1)) != 0 || pow3.High.Cmp(big.NewRat(8, 1)) != 0 {
		t.Errorf("PowInt failed: expected [1, 8], got %s", pow3.String())
	}

	powEvenZero := zeroInterval.PowInt(2) // [-1, 1]^2 => [0, 1]
	if powEvenZero.Low.Sign() != 0 || powEvenZero.High.Cmp(big.NewRat(1, 1)) != 0 {
		t.Errorf("PowInt even over zero failed: expected [0, 1], got %s", powEvenZero.String())
	}
}

func TestRationalInterval_Constants(t *testing.T) {
	// Pi: 3.14159265...
	piInterval := RationalBoundPi(10)
	lowerPi := big.NewRat(314159, 100000)
	upperPi := big.NewRat(314160, 100000)

	if piInterval.Low.Cmp(lowerPi) < 0 || piInterval.High.Cmp(upperPi) > 0 {
		t.Errorf("RationalBoundPi(10) enclosure loose: %s not in [%s, %s]",
			piInterval.String(), lowerPi.RatString(), upperPi.RatString())
	}

	// Archimedes inequality: pi < 22/7
	archimedesUpper := big.NewRat(22, 7)
	if piInterval.High.Cmp(archimedesUpper) >= 0 {
		t.Errorf("Pi upper bound >= 22/7: %s vs %s", piInterval.High.RatString(), archimedesUpper.RatString())
	}

	// Euler's e: 2.7182818...
	eInterval := RationalBoundE(12)
	lowerE := big.NewRat(271828, 100000)
	upperE := big.NewRat(271829, 100000)
	if eInterval.Low.Cmp(lowerE) < 0 || eInterval.High.Cmp(upperE) > 0 {
		t.Errorf("RationalBoundE(12) enclosure loose: %s not in [%s, %s]",
			eInterval.String(), lowerE.RatString(), upperE.RatString())
	}
}

func TestRationalInterval_TranscendentalFunctions(t *testing.T) {
	// sin(1) approx 0.84147098...
	sin1 := RationalBoundSinCos(big.NewRat(1, 1), true, 10)
	if sin1.Low.Cmp(big.NewRat(84, 100)) < 0 || sin1.High.Cmp(big.NewRat(85, 100)) > 0 {
		t.Errorf("sin(1) enclosure out of bounds: %s", sin1.String())
	}

	// cos(1) approx 0.5403023...
	cos1 := RationalBoundSinCos(big.NewRat(1, 1), false, 10)
	if cos1.Low.Cmp(big.NewRat(54, 100)) < 0 || cos1.High.Cmp(big.NewRat(55, 100)) > 0 {
		t.Errorf("cos(1) enclosure out of bounds: %s", cos1.String())
	}

	// exp(2) approx 7.389056...
	exp2 := RationalBoundExp(big.NewRat(2, 1), 12)
	if exp2.Low.Cmp(big.NewRat(738, 100)) < 0 || exp2.High.Cmp(big.NewRat(740, 100)) > 0 {
		t.Errorf("exp(2) enclosure out of bounds: %s", exp2.String())
	}

	// ln(2) approx 0.693147...
	ln2, err := RationalBoundLn(big.NewRat(2, 1), 12)
	if err != nil {
		t.Fatalf("RationalBoundLn failed: %v", err)
	}
	if ln2.Low.Cmp(big.NewRat(693, 1000)) < 0 || ln2.High.Cmp(big.NewRat(694, 1000)) > 0 {
		t.Errorf("ln(2) enclosure out of bounds: %s", ln2.String())
	}
}

func TestRationalInterval_RelationalEvaluation(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"1 < 2", "true"},
		{"5 >= 10", "false"},
		{"pi < 22/7", "true"},
		{"pi > 3", "true"},
		{"sin(1) > 1/2", "true"},
		{"cos(1) < 1", "true"},
		{"exp(2) > 7", "true"},
		{"exp(pi) > pi^e", "true"}, // Gelfond's constant e^π > π^e
		{"solve(pi < 22/7)", "true"},
		{"solve(exp(2) > 7)", "true"},
	}

	for _, tc := range tests {
		res, err := EvalString(tc.expr)
		if err != nil {
			t.Errorf("EvalString(%q) error: %v", tc.expr, err)
			continue
		}
		if res.String() != tc.expected {
			t.Errorf("EvalString(%q) = %s; expected %s", tc.expr, res.String(), tc.expected)
		}
	}
}

func TestAdaptiveRefineInterval(t *testing.T) {
	// sqrt(2) with eps = 1/10^6
	node := &SqrtNode{Radicand: mustRational(2, 1)}

	eps := big.NewRat(1, 1000000)
	interval, err := AdaptiveRefineInterval(node, eps, 60)
	if err != nil {
		t.Fatalf("AdaptiveRefineInterval failed: %v", err)
	}

	width := interval.Width()
	if width.Cmp(eps) > 0 {
		t.Errorf("expected width <= %s, got %s", eps.RatString(), width.RatString())
	}

	// Verify enclosure: Low^2 <= 2 <= High^2
	two := big.NewRat(2, 1)
	lowSq := new(big.Rat).Mul(interval.Low, interval.Low)
	highSq := new(big.Rat).Mul(interval.High, interval.High)
	if lowSq.Cmp(two) > 0 {
		t.Errorf("low bound squared exceeds 2: %s", lowSq.RatString())
	}
	if highSq.Cmp(two) < 0 {
		t.Errorf("high bound squared is below 2: %s", highSq.RatString())
	}
}

func TestFormatDecimalEnclosure(t *testing.T) {
	// Test 1/3: true value ~0.333333333...
	// Directed outward rounding with 4 digits: [0.3333, 0.3334]
	inv3 := NewExactRationalInterval(big.NewRat(1, 3))
	s := FormatDecimalEnclosure(inv3, 4)
	if s != "[0.3333, 0.3334]" {
		t.Errorf("expected [0.3333, 0.3334], got %s", s)
	}

	// Test -1/3: true value ~ -0.33333333...
	// Lower bound floor: -0.3334, Upper bound ceil: -0.3333
	negInv3 := NewExactRationalInterval(big.NewRat(-1, 3))
	sNeg := FormatDecimalEnclosure(negInv3, 4)
	if sNeg != "[-0.3334, -0.3333]" {
		t.Errorf("expected [-0.3334, -0.3333], got %s", sNeg)
	}

	// Test exact fraction 1/2: [0.50, 0.50]
	half := NewExactRationalInterval(big.NewRat(1, 2))
	sHalf := FormatDecimalEnclosure(half, 2)
	if sHalf != "[0.50, 0.50]" {
		t.Errorf("expected [0.50, 0.50], got %s", sHalf)
	}
}

