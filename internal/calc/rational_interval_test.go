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

func TestRationalInterval_SinCosExtremaAndRangeReduction(t *testing.T) {
	// sin([0, 4]) crosses pi/2 (~1.57) and 7pi/6 (~3.66), so max must reach 1 (not endpoint max = 0).
	sinNode := &FuncNode{Name: "sin", Args: []Node{NewIntervalNode(big.NewRat(0, 1), big.NewRat(4, 1))}}
	ivSin, err := EvalNodeInterval(sinNode, 12)
	if err != nil {
		t.Fatalf("EvalNodeInterval(sin([0,4])) failed: %v", err)
	}
	if ivSin.High.Cmp(big.NewRat(1, 1)) < 0 {
		t.Errorf("sin([0,4]) upper bound = %s, must enclose maximum 1", ivSin.High.RatString())
	}
	if ivSin.Low.Cmp(big.NewRat(-1, 1)) < 0 || ivSin.High.Cmp(big.NewRat(1, 1)) > 0 {
		t.Errorf("sin([0,4]) out of [-1, 1]: %s", ivSin.String())
	}

	// cos([0, 7]) spans more than a full 2*pi period, so must enclose both -1 and 1.
	cosNode := &FuncNode{Name: "cos", Args: []Node{NewIntervalNode(big.NewRat(0, 1), big.NewRat(7, 1))}}
	ivCos, err := EvalNodeInterval(cosNode, 12)
	if err != nil {
		t.Fatalf("EvalNodeInterval(cos([0,7])) failed: %v", err)
	}
	if ivCos.Low.Cmp(big.NewRat(-1, 1)) > 0 || ivCos.High.Cmp(big.NewRat(1, 1)) < 0 {
		t.Errorf("cos([0,7]) = %s, must enclose [-1, 1]", ivCos.String())
	}

	// cos([1, 4]) crosses pi (~3.14159), so min must reach -1.
	cosPiNode := &FuncNode{Name: "cos", Args: []Node{NewIntervalNode(big.NewRat(1, 1), big.NewRat(4, 1))}}
	ivCosPi, err := EvalNodeInterval(cosPiNode, 12)
	if err != nil {
		t.Fatalf("EvalNodeInterval(cos([1,4])) failed: %v", err)
	}
	if ivCosPi.Low.Cmp(big.NewRat(-1, 1)) > 0 {
		t.Errorf("cos([1,4]) lower bound = %s, must enclose minimum -1", ivCosPi.Low.RatString())
	}

	// Large rational argument beyond int64: 10^25 must not overflow int64 or hang.
	hugeX, _ := new(big.Int).SetString("10000000000000000000000000", 10)
	sinHuge := RationalBoundSinCos(new(big.Rat).SetInt(hugeX), true, 12)
	if sinHuge.Low.Cmp(big.NewRat(-1, 1)) < 0 || sinHuge.High.Cmp(big.NewRat(1, 1)) > 0 {
		t.Errorf("sin(10^25) out of [-1, 1]: %s", sinHuge.String())
	}
}

func TestRationalInterval_LargeExpAndBoundedCbrt(t *testing.T) {
	// exp(40) via binary scaling without O(x) loop
	exp40 := RationalBoundExp(big.NewRat(40, 1), 12)
	// exp(40) ~ 2.35385e17
	if exp40.Low.Cmp(big.NewRat(235000000000000000, 1)) < 0 || exp40.High.Cmp(big.NewRat(236000000000000000, 1)) > 0 {
		t.Errorf("exp(40) enclosure unexpected: %s", exp40.String())
	}

	// cbrt(2) at steps=20: must enclose 2^(1/3) and keep bit length O(k) (no 3^iterSteps explosion)
	cbrt2, err := RationalBoundCbrt(big.NewRat(2, 1), 20)
	if err != nil {
		t.Fatalf("RationalBoundCbrt(2, 20) failed: %v", err)
	}
	lowCube := new(big.Rat).Mul(cbrt2.Low, new(big.Rat).Mul(cbrt2.Low, cbrt2.Low))
	highCube := new(big.Rat).Mul(cbrt2.High, new(big.Rat).Mul(cbrt2.High, cbrt2.High))
	two := big.NewRat(2, 1)
	if lowCube.Cmp(two) > 0 || highCube.Cmp(two) < 0 {
		t.Errorf("cbrt(2) failed enclosure: low^3=%s, high^3=%s", lowCube.RatString(), highCube.RatString())
	}
	if cbrt2.Low.Num().BitLen() > 256 || cbrt2.Low.Denom().BitLen() > 256 {
		t.Errorf("cbrt(2) bit length exploded: num=%d bits, den=%d bits", cbrt2.Low.Num().BitLen(), cbrt2.Low.Denom().BitLen())
	}

	// Negative exact cube: cbrt(-8) must enclose -2
	cbrtNeg8, err := RationalBoundCbrt(big.NewRat(-8, 1), 16)
	if err != nil {
		t.Fatalf("RationalBoundCbrt(-8, 16) failed: %v", err)
	}
	negTwo := big.NewRat(-2, 1)
	if cbrtNeg8.Low.Cmp(negTwo) > 0 || cbrtNeg8.High.Cmp(negTwo) < 0 {
		t.Errorf("cbrt(-8) = %s, must enclose -2", cbrtNeg8.String())
	}
}

func TestRationalInterval_LogBaseAndArbitraryBase(t *testing.T) {
	// Deductive: log(2) is base-10 (~0.30102999566...), NOT ln(2) (~0.693147...)
	log2Node, err := EvalString("log(2)")
	if err != nil {
		t.Fatalf("EvalString(log(2)) failed: %v", err)
	}
	ivLog2, err := EvalNodeInterval(log2Node, 20)
	if err != nil {
		t.Fatalf("EvalNodeInterval(log(2)) failed: %v", err)
	}
	loExpect := big.NewRat(301029, 1000000)
	hiExpect := big.NewRat(301031, 1000000)
	if ivLog2.Low.Cmp(loExpect) < 0 || ivLog2.High.Cmp(hiExpect) > 0 {
		t.Errorf("EvalNodeInterval(log(2)) = %s, expected near 0.301030 (base-10)", ivLog2.String())
	}

	// Deductive: 2-arg log(2, 3) (~1.5849625...)
	log23Node, err := EvalString("log(2, 3)")
	if err != nil {
		t.Fatalf("EvalString(log(2, 3)) failed: %v", err)
	}
	ivLog23, err := EvalNodeInterval(log23Node, 20)
	if err != nil {
		t.Fatalf("EvalNodeInterval(log(2, 3)) failed: %v", err)
	}
	if ivLog23.Low.Cmp(big.NewRat(158496, 100000)) < 0 || ivLog23.High.Cmp(big.NewRat(158497, 100000)) > 0 {
		t.Errorf("EvalNodeInterval(log(2, 3)) = %s, expected near 1.5849625", ivLog23.String())
	}

	// Deductive: base < 1 -> log(1/2, 4) == -2
	logHalf4Node, err := EvalString("log(1/2, 4)")
	if err != nil {
		t.Fatalf("EvalString(log(1/2, 4)) failed: %v", err)
	}
	ivLogHalf4, err := EvalNodeInterval(logHalf4Node, 20)
	if err != nil {
		t.Fatalf("EvalNodeInterval(log(1/2, 4)) failed: %v", err)
	}
	negTwo := big.NewRat(-2, 1)
	if ivLogHalf4.Low.Cmp(negTwo) > 0 || ivLogHalf4.High.Cmp(negTwo) < 0 {
		t.Errorf("EvalNodeInterval(log(1/2, 4)) = %s, must enclose -2", ivLogHalf4.String())
	}

	// Contrapositive: base interval spanning 1 ([1/2, 2]) must error because ln(base) contains 0
	badBaseNode := &FuncNode{
		Name: "log",
		Args: []Node{
			NewIntervalNode(big.NewRat(1, 2), big.NewRat(2, 1)),
			mustRational(4, 1),
		},
	}
	if iv, err := EvalNodeInterval(badBaseNode, 12); err == nil {
		t.Errorf("expected error for log([1/2, 2], 4), got %s", iv.String())
	}

	// Contradiction: log(1, 5) and log(-2, 5) must fail
	if _, err := EvalNodeInterval(&FuncNode{Name: "log", Args: []Node{mustRational(1, 1), mustRational(5, 1)}}, 12); err == nil {
		t.Errorf("expected error for log(1, 5)")
	}
	if _, err := EvalNodeInterval(&FuncNode{Name: "log", Args: []Node{mustRational(-2, 1), mustRational(5, 1)}}, 12); err == nil {
		t.Errorf("expected error for log(-2, 5)")
	}
}

func TestRationalInterval_NegativeBaseOddRoot(t *testing.T) {
	// Deductive: negative base with odd denominator rational exponents
	cases := []struct {
		expr   string
		target *big.Rat
	}{
		{"(-32)^(1/5)", big.NewRat(-2, 1)},
		{"(-32)^(2/5)", big.NewRat(4, 1)},
		{"(-32)^(-1/5)", big.NewRat(-1, 2)},
		{"(-8)^(2/3)", big.NewRat(4, 1)},
	}
	for _, tc := range cases {
		node, err := EvalString(tc.expr)
		if err != nil {
			t.Fatalf("EvalString(%q) failed: %v", tc.expr, err)
		}
		iv, err := EvalNodeInterval(node, 20)
		if err != nil {
			t.Fatalf("EvalNodeInterval(%q) failed: %v", tc.expr, err)
		}
		if iv.Low.Cmp(tc.target) > 0 || iv.High.Cmp(tc.target) < 0 {
			t.Errorf("EvalNodeInterval(%q) = %s, must enclose %s", tc.expr, iv.String(), tc.target.RatString())
		}
	}

	// Zero-crossing intervals with positive odd-denominator exponents
	oddSpan := &PowNode{
		Base: NewIntervalNode(big.NewRat(-32, 1), big.NewRat(32, 1)),
		Exp:  NewRationalFromBigRat(big.NewRat(1, 5)),
	}
	ivOddSpan, err := EvalNodeInterval(oddSpan, 20)
	if err != nil {
		t.Fatalf("EvalNodeInterval([-32, 32]^(1/5)) failed: %v", err)
	}
	if ivOddSpan.Low.Cmp(big.NewRat(-2, 1)) > 0 || ivOddSpan.High.Cmp(big.NewRat(2, 1)) < 0 {
		t.Errorf("[-32, 32]^(1/5) = %s, must enclose [-2, 2]", ivOddSpan.String())
	}

	evenSpan := &PowNode{
		Base: NewIntervalNode(big.NewRat(-32, 1), big.NewRat(32, 1)),
		Exp:  NewRationalFromBigRat(big.NewRat(2, 5)),
	}
	ivEvenSpan, err := EvalNodeInterval(evenSpan, 20)
	if err != nil {
		t.Fatalf("EvalNodeInterval([-32, 32]^(2/5)) failed: %v", err)
	}
	if ivEvenSpan.Low.Sign() != 0 || ivEvenSpan.High.Cmp(big.NewRat(4, 1)) < 0 {
		t.Errorf("[-32, 32]^(2/5) = %s, must enclose [0, 4] with Low == 0", ivEvenSpan.String())
	}

	// Contrapositive: zero-crossing interval with negative exponent must fail
	negSpan := &PowNode{
		Base: NewIntervalNode(big.NewRat(-1, 1), big.NewRat(1, 1)),
		Exp:  NewRationalFromBigRat(big.NewRat(-1, 5)),
	}
	if iv, err := EvalNodeInterval(negSpan, 12); err == nil {
		t.Errorf("expected error for [-1, 1]^(-1/5), got %s", iv.String())
	}

	// Contradiction: even denominator on negative base must fail
	evenDenom := &PowNode{
		Base: mustRational(-16, 1),
		Exp:  NewRationalFromBigRat(big.NewRat(3, 4)),
	}
	if iv, err := EvalNodeInterval(evenDenom, 12); err == nil {
		t.Errorf("expected error for (-16)^(3/4), got %s", iv.String())
	}
}

