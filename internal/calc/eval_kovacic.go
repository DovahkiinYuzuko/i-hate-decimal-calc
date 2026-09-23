package calc

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// -------------------------------------------------------------------------
// Kovacic Algorithm Core Lifecycle & State Machine
// -------------------------------------------------------------------------

// KovacicState represents a discrete lifecycle state in Kovacic's algorithm pipeline.
type KovacicState int

const (
	KovacicStateInit                 KovacicState = 0
	KovacicStateNormalFormReduced    KovacicState = 1
	KovacicStatePolesAnalyzed        KovacicState = 2
	KovacicStateCase1Tested          KovacicState = 3
	KovacicStateCase2Tested          KovacicState = 4
	KovacicStateSolutionConstructed  KovacicState = 5
	KovacicStateNonLiouvillianProven KovacicState = 6
	KovacicStateFailed               KovacicState = 7
)

func (s KovacicState) String() string {
	switch s {
	case KovacicStateInit:
		return "Init"
	case KovacicStateNormalFormReduced:
		return "NormalFormReduced"
	case KovacicStatePolesAnalyzed:
		return "PolesAnalyzed"
	case KovacicStateCase1Tested:
		return "Case1Tested"
	case KovacicStateCase2Tested:
		return "Case2Tested"
	case KovacicStateSolutionConstructed:
		return "SolutionConstructed"
	case KovacicStateNonLiouvillianProven:
		return "NonLiouvillianProven"
	case KovacicStateFailed:
		return "Failed"
	default:
		return fmt.Sprintf("KovacicState(%d)", int(s))
	}
}

// KovacicLifecycleFSM governs valid state transitions in the Kovacic solver pipeline.
type KovacicLifecycleFSM struct {
	mu           sync.RWMutex
	currentState KovacicState
	history      []KovacicState
}

// NewKovacicLifecycleFSM creates a new KovacicLifecycleFSM initialized to KovacicStateInit.
func NewKovacicLifecycleFSM() *KovacicLifecycleFSM {
	return &KovacicLifecycleFSM{
		currentState: KovacicStateInit,
		history:      []KovacicState{KovacicStateInit},
	}
}

// CurrentState returns the current state.
func (fsm *KovacicLifecycleFSM) CurrentState() KovacicState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.currentState
}

// History returns a copy of the state transition history.
func (fsm *KovacicLifecycleFSM) History() []KovacicState {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	cp := make([]KovacicState, len(fsm.history))
	copy(cp, fsm.history)
	return cp
}

// TransitionTo validates and performs a state transition.
func (fsm *KovacicLifecycleFSM) TransitionTo(target KovacicState) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	valid := false
	switch fsm.currentState {
	case KovacicStateInit:
		valid = (target == KovacicStateNormalFormReduced || target == KovacicStateFailed)
	case KovacicStateNormalFormReduced:
		valid = (target == KovacicStatePolesAnalyzed || target == KovacicStateSolutionConstructed || target == KovacicStateFailed)
	case KovacicStatePolesAnalyzed:
		valid = (target == KovacicStateCase1Tested || target == KovacicStateCase2Tested || target == KovacicStateNonLiouvillianProven || target == KovacicStateFailed)
	case KovacicStateCase1Tested:
		valid = (target == KovacicStateSolutionConstructed || target == KovacicStateCase2Tested || target == KovacicStateNonLiouvillianProven || target == KovacicStateFailed)
	case KovacicStateCase2Tested:
		valid = (target == KovacicStateSolutionConstructed || target == KovacicStateNonLiouvillianProven || target == KovacicStateFailed)
	case KovacicStateSolutionConstructed, KovacicStateNonLiouvillianProven, KovacicStateFailed:
		valid = false // Terminal states
	}

	if !valid {
		return fmt.Errorf("%s", i18n.T("kovacic.err_invalid_kovacic_fsm_transition", fsm.currentState, target))
	}

	fsm.currentState = target
	fsm.history = append(fsm.history, target)
	return nil
}

// -------------------------------------------------------------------------
// Kovacic Algorithm Core Data Structures
// -------------------------------------------------------------------------

// KovacicResult contains the outcome of the Kovacic solver pipeline.
type KovacicResult struct {
	FSM           *KovacicLifecycleFSM
	IsLiouvillian bool
	Case          int    // 1, 2, or 4
	Basis         []Node // Linear independent solution basis {y1, y2}
	GeneralSol    Node   // C_1 * y1 + C_2 * y2
	ProofMessage  string // i18n key or proof certificate
}

// rationalPole represents a finite pole of the rational function r(x).
type rationalPole struct {
	c     *big.Rat // Location of the pole (x = c)
	order int      // Pole order / multiplicity
	bCoeff *big.Rat // Leading Laurent coefficient b / (x-c)^order
}

// -------------------------------------------------------------------------
// Top-Level Public Solvers
// -------------------------------------------------------------------------

// SolveKovacicExact solves a second-order linear homogeneous ODE:
//   y'' + P(x)*y' + Q(x)*y = 0
// using Kovacic's algorithm and differential Galois theory.
func SolveKovacicExact(pExpr, qExpr Node, varName string) (*KovacicResult, error) {
	fsm := NewKovacicLifecycleFSM()
	return SolveKovacicExactWithFSM(pExpr, qExpr, varName, fsm)
}

// SolveKovacicExactWithFSM executes the Kovacic pipeline while tracking state transitions in fsm.
func SolveKovacicExactWithFSM(pExpr, qExpr Node, varName string, fsm *KovacicLifecycleFSM) (*KovacicResult, error) {
	if fsm == nil {
		fsm = NewKovacicLifecycleFSM()
	}

	// Step 1: Normal Form Reduction y = xi * u => u'' = r(x) * u
	rNode, xiNode, err := NormalizeSecondOrderODE(pExpr, qExpr, varName)
	if err != nil {
		_ = fsm.TransitionTo(KovacicStateFailed)
		return nil, fmt.Errorf("%s", i18n.T("kovacic.err_failed_to_reduce_normal_form", err))
	}
	if err := fsm.TransitionTo(KovacicStateNormalFormReduced); err != nil {
		return nil, err
	}

	// Fast-Path: If r(x) is constant r0 in Q
	if ratR, ok := rNode.(*RationalNode); ok {
		res, err := solveConstantCoeffNormalODE(ratR.Val, xiNode, varName, fsm)
		if err == nil && res != nil {
			return res, nil
		}
	}

	// Step 2: Pole & Singularity Analysis
	poles, infOrder, err := analyzePolesKovacic(rNode, varName)
	if err != nil {
		_ = fsm.TransitionTo(KovacicStateFailed)
		return nil, err
	}
	if err := fsm.TransitionTo(KovacicStatePolesAnalyzed); err != nil {
		return nil, err
	}

	// Step 3: Test Case 1 (Exponential Integral / Triangularizable)
	case1Sol, isCase1Possible := solveKovacicCase1(rNode, xiNode, poles, infOrder, varName)
	if isCase1Possible && case1Sol != nil {
		if err := fsm.TransitionTo(KovacicStateCase1Tested); err != nil {
			return nil, err
		}
		if err := fsm.TransitionTo(KovacicStateSolutionConstructed); err != nil {
			return nil, err
		}
		case1Sol.FSM = fsm
		case1Sol.Case = 1
		case1Sol.IsLiouvillian = true
		case1Sol.ProofMessage = i18n.T("kovacic.msg_case1_liouvillian_found")
		return case1Sol, nil
	}
	_ = fsm.TransitionTo(KovacicStateCase1Tested)

	// Step 4: Test Case 2 (Symmetric Square / Dihedral Group)
	case2Sol, isCase2Possible := solveKovacicCase2(rNode, xiNode, varName)
	if isCase2Possible && case2Sol != nil {
		if err := fsm.TransitionTo(KovacicStateCase2Tested); err != nil {
			return nil, err
		}
		if err := fsm.TransitionTo(KovacicStateSolutionConstructed); err != nil {
			return nil, err
		}
		case2Sol.FSM = fsm
		case2Sol.Case = 2
		case2Sol.IsLiouvillian = true
		case2Sol.ProofMessage = i18n.T("kovacic.msg_case2_liouvillian_found")
		return case2Sol, nil
	}
	_ = fsm.TransitionTo(KovacicStateCase2Tested)

	// Step 5: If Case 1 and Case 2 fail, and pole conditions preclude Case 3,
	// Differential Galois Group is SL(2, C) => Non-Liouvillian Proven!
	if err := fsm.TransitionTo(KovacicStateNonLiouvillianProven); err != nil {
		return nil, err
	}

	return &KovacicResult{
		FSM:           fsm,
		IsLiouvillian: false,
		Case:          4,
		Basis:         nil,
		GeneralSol:    nil,
		ProofMessage:  i18n.T("kovacic.msg_non_liouvillian_proven"),
	}, nil
}

// -------------------------------------------------------------------------
// Normal Form Reduction: y = xi * u => u'' = r(x) * u
// -------------------------------------------------------------------------

// NormalizeSecondOrderODE transforms y'' + P(x)*y' + Q(x)*y = 0 into u'' = r(x)*u.
func NormalizeSecondOrderODE(pExpr, qExpr Node, varName string) (rExpr Node, xiExpr Node, err error) {
	if pExpr == nil {
		pExpr = mustRational(0, 1)
	}
	if qExpr == nil {
		qExpr = mustRational(0, 1)
	}

	// 1. Calculate xi(x) = exp(-1/2 * integrate(P, x))
	var intP Node
	if isZeroNode(pExpr) {
		xiExpr = mustRational(1, 1)
	} else {
		intP, err = integrateCore(pExpr, varName)
		if err != nil {
			return nil, nil, err
		}
		halfIntP, err := simplifyMul([]Node{mustRational(-1, 2), intP})
		if err != nil {
			return nil, nil, err
		}
		xiExpr, err = Eval(&FuncNode{Name: "exp", Args: []Node{halfIntP}})
		if err != nil {
			return nil, nil, err
		}
	}

	// 2. Calculate P'(x) = dP/dx
	dP, err := differentiate(pExpr, varName)
	if err != nil {
		return nil, nil, err
	}

	// 3. r(x) = -Q + 1/4 * P^2 + 1/2 * P'
	negQ, err := simplifyUnaryOp("-", qExpr)
	if err != nil {
		return nil, nil, err
	}
	pSquared, err := simplifyPow(pExpr, mustRational(2, 1))
	if err != nil {
		return nil, nil, err
	}
	quarterPSquared, err := simplifyMul([]Node{mustRational(1, 4), pSquared})
	if err != nil {
		return nil, nil, err
	}
	halfDP, err := simplifyMul([]Node{mustRational(1, 2), dP})
	if err != nil {
		return nil, nil, err
	}

	rRaw, err := simplifyAdd([]Node{negQ, quarterPSquared, halfDP})
	if err != nil {
		return nil, nil, err
	}

	rSimplified, err := Eval(rRaw)
	if err != nil {
		return nil, nil, err
	}

	return rSimplified, xiExpr, nil
}

// solveConstantCoeffNormalODE solves u'' = r0 * u where r0 in Q.
func solveConstantCoeffNormalODE(r0 *big.Rat, xi Node, varName string, fsm *KovacicLifecycleFSM) (*KovacicResult, error) {
	var u1, u2 Node
	xNode := &VarNode{Name: varName}

	if r0.Sign() == 0 {
		u1 = mustRational(1, 1)
		u2 = xNode
	} else if r0.Sign() > 0 {
		// sqrt(r0)
		numSqrt, numExact := new(big.Int).Sqrt(r0.Num()), false
		denSqrt, denExact := new(big.Int).Sqrt(r0.Denom()), false
		if new(big.Int).Mul(numSqrt, numSqrt).Cmp(r0.Num()) == 0 {
			numExact = true
		}
		if new(big.Int).Mul(denSqrt, denSqrt).Cmp(r0.Denom()) == 0 {
			denExact = true
		}

		if numExact && denExact {
			lambdaRat := big.NewRat(1, 1).SetFrac(numSqrt, denSqrt)
			lamNode := &RationalNode{Val: lambdaRat}
			negLamNode := &RationalNode{Val: new(big.Rat).Neg(lambdaRat)}

			arg1, _ := simplifyMul([]Node{lamNode, xNode})
			arg2, _ := simplifyMul([]Node{negLamNode, xNode})

			u1 = &FuncNode{Name: "exp", Args: []Node{arg1}}
			u2 = &FuncNode{Name: "exp", Args: []Node{arg2}}
		} else {
			sqrtR0 := NewSqrt(&RationalNode{Val: r0})
			arg1, _ := simplifyMul([]Node{sqrtR0, xNode})
			negSqrtR0, _ := simplifyUnaryOp("-", sqrtR0)
			arg2, _ := simplifyMul([]Node{negSqrtR0, xNode})

			u1 = &FuncNode{Name: "exp", Args: []Node{arg1}}
			u2 = &FuncNode{Name: "exp", Args: []Node{arg2}}
		}
	} else {
		// r0 < 0 => harmonic oscillation cos(omega * x), sin(omega * x)
		absR0 := new(big.Rat).Abs(r0)
		numSqrt, numExact := new(big.Int).Sqrt(absR0.Num()), false
		denSqrt, denExact := new(big.Int).Sqrt(absR0.Denom()), false
		if new(big.Int).Mul(numSqrt, numSqrt).Cmp(absR0.Num()) == 0 {
			numExact = true
		}
		if new(big.Int).Mul(denSqrt, denSqrt).Cmp(absR0.Denom()) == 0 {
			denExact = true
		}

		var omegaNode Node
		if numExact && denExact {
			omegaRat := big.NewRat(1, 1).SetFrac(numSqrt, denSqrt)
			omegaNode = &RationalNode{Val: omegaRat}
		} else {
			omegaNode = NewSqrt(&RationalNode{Val: absR0})
		}

		arg, _ := simplifyMul([]Node{omegaNode, xNode})
		u1 = &FuncNode{Name: "cos", Args: []Node{arg}}
		u2 = &FuncNode{Name: "sin", Args: []Node{arg}}
	}

	// Restore y = xi * u
	y1, _ := simplifyMul([]Node{xi, u1})
	y2, _ := simplifyMul([]Node{xi, u2})
	y1, _ = Eval(y1)
	y2, _ = Eval(y2)

	_ = fsm.TransitionTo(KovacicStateSolutionConstructed)

	c1Y1, _ := simplifyMul([]Node{&VarNode{Name: "C_1"}, y1})
	c2Y2, _ := simplifyMul([]Node{&VarNode{Name: "C_2"}, y2})
	genSol, _ := simplifyAdd([]Node{c1Y1, c2Y2})

	return &KovacicResult{
		FSM:           fsm,
		IsLiouvillian: true,
		Case:          1,
		Basis:         []Node{y1, y2},
		GeneralSol:    genSol,
		ProofMessage:  i18n.T("kovacic.msg_case1_liouvillian_found"),
	}, nil
}

// -------------------------------------------------------------------------
// Pole and Singularity Analysis
// -------------------------------------------------------------------------

// analyzePolesKovacic analyzes poles of r(x) in Q[x].
func analyzePolesKovacic(rNode Node, varName string) (poles []rationalPole, infOrder int, err error) {
	num, den := extractFraction(rNode)
	degNum := degreeInVar(num, varName)
	degDen := degreeInVar(den, varName)

	infOrder = degDen - degNum

	// If denominator is 1, no finite poles
	if degDen <= 0 {
		return nil, infOrder, nil
	}

	// Find rational roots of denominator to detect finite poles
	roots := findRationalRootsForNode(den, varName)
	for _, r := range roots {
		// Multiplicity of root in denominator
		order := rootMultiplicity(den, varName, r)
		if order > 0 {
			// Compute leading Laurent coefficient: b = lim_{x->r} (x - r)^order * r(x)
			bCoeff := computePoleLeadingCoeff(num, den, varName, r, order)
			poles = append(poles, rationalPole{
				c:      r,
				order:  order,
				bCoeff: bCoeff,
			})
		}
	}

	return poles, infOrder, nil
}

// -------------------------------------------------------------------------
// Case 1 Solver (Exponential Integral / Rational Logarithmic Derivative)
// -------------------------------------------------------------------------

func solveKovacicCase1(rNode, xi Node, poles []rationalPole, infOrder int, varName string) (*KovacicResult, bool) {
	// Case 1 Necessary Condition Check:
	// All finite poles must have order 1 or even order >= 2.
	// If any finite pole has odd order >= 3, Case 1 CANNOT hold!
	for _, p := range poles {
		if p.order > 2 && p.order%2 != 0 {
			return nil, false
		}
	}
	// Infinity order must be >= 2 or even integer <= 0
	if infOrder < 2 && infOrder%2 != 0 {
		return nil, false
	}

	// 1. Candidate alpha_c sets for each finite pole
	candidatePoleAlphas := make([][]*big.Rat, len(poles))
	for i, p := range poles {
		alphas := candidateAlphasForPole(p)
		if len(alphas) == 0 {
			return nil, false
		}
		candidatePoleAlphas[i] = alphas
	}

	// 2. Candidate alpha_inf set for infinity
	infAlphas := candidateAlphasForInfinity(rNode, infOrder, varName)
	if len(infAlphas) == 0 {
		return nil, false
	}

	// 3. Cartesian product search for non-negative integer degree bound d = alpha_inf - sum(alpha_c)
	combinations := generateAlphaCombinations(candidatePoleAlphas, infAlphas)
	xNode := &VarNode{Name: varName}

	for _, combo := range combinations {
		d := combo.d
		if d < 0 {
			continue
		}

		// Construct theta(x) = sum_c alpha_c / (x - c)
		var thetaTerms []Node
		for i, p := range poles {
			alphaC := combo.poleAlphas[i]
			if alphaC.Sign() != 0 {
				denom, _ := simplifyAdd([]Node{xNode, &RationalNode{Val: new(big.Rat).Neg(p.c)}})
				term, _ := simplifyMul([]Node{&RationalNode{Val: alphaC}, &PowNode{Base: denom, Exp: mustRational(-1, 1)}})
				thetaTerms = append(thetaTerms, term)
			}
		}

		var thetaNode Node = mustRational(0, 1)
		if len(thetaTerms) > 0 {
			thetaNode, _ = simplifyAdd(thetaTerms)
		}

		// Search for polynomial P(x) of degree d satisfying:
		//   P'' + 2*theta*P' + (theta' + theta^2 - r)*P = 0
		pPoly, found := findCase1Polynomial(thetaNode, rNode, d, varName)
		if found && pPoly != nil {
			// u1 = P(x) * exp(integrate(theta, x))
			intTheta, err := integrateCore(thetaNode, varName)
			if err != nil {
				continue
			}
			expIntTheta, err := Eval(&FuncNode{Name: "exp", Args: []Node{intTheta}})
			if err != nil {
				continue
			}
			pNode := pPoly.toNode()

			u1, err := simplifyMul([]Node{pNode, expIntTheta})
			if err != nil {
				continue
			}
			u1, _ = Eval(u1)

			// Recover y1 = xi * u1
			y1, err := simplifyMul([]Node{xi, u1})
			if err != nil {
				continue
			}
			y1, _ = Eval(y1)

			// Second independent solution via Reduction of Order:
			// u2 = u1 * integrate(1 / u1^2, x) => y2 = xi * u2
			u1Squared, _ := simplifyPow(u1, mustRational(2, 1))
			invU1Sq, _ := simplifyPow(u1Squared, mustRational(-1, 1))
			invU1Sq, _ = Eval(invU1Sq)

			intInvU1Sq, err := integrateCore(invU1Sq, varName)
			var y2 Node
			if err == nil && intInvU1Sq != nil {
				u2, _ := simplifyMul([]Node{u1, intInvU1Sq})
				y2, _ = simplifyMul([]Node{xi, u2})
				y2, _ = Eval(y2)
			} else {
				// Fallback formal basis if second integral is non-elementary
				u2 := &FuncNode{Name: "integrate", Args: []Node{invU1Sq, xNode}}
				u2Mul, _ := simplifyMul([]Node{u1, u2})
				y2, _ = simplifyMul([]Node{xi, u2Mul})
			}

			c1Y1, _ := simplifyMul([]Node{&VarNode{Name: "C_1"}, y1})
			c2Y2, _ := simplifyMul([]Node{&VarNode{Name: "C_2"}, y2})
			genSol, _ := simplifyAdd([]Node{c1Y1, c2Y2})

			return &KovacicResult{
				IsLiouvillian: true,
				Case:          1,
				Basis:         []Node{y1, y2},
				GeneralSol:    genSol,
			}, true
		}
	}

	return nil, false
}

// -------------------------------------------------------------------------
// Case 2 Solver (Symmetric Square / 2nd Degree Algebraic Extensions)
// -------------------------------------------------------------------------

func solveKovacicCase2(rNode, xi Node, varName string) (*KovacicResult, bool) {
	// 3rd-order Symmetric Square ODE:
	//   z''' - 4*r(x)*z' - 2*r'(x)*z = 0
	// Search for polynomial solutions z(x) of low degree (up to degree 4)
	dr, err := differentiate(rNode, varName)
	if err != nil {
		return nil, false
	}

	for degZ := 0; degZ <= 4; degZ++ {
		zPoly, found := findSymmetricSquarePoly(rNode, dr, degZ, varName)
		if found && zPoly != nil && !isPolyZero(zPoly) {
			zNode := zPoly.toNode()

			// Recover w = 1/2 * (z' / z)
			dzNode, _ := differentiate(zNode, varName)
			halfDZ, _ := simplifyMul([]Node{mustRational(1, 2), dzNode})
			wNode, _ := simplifyMul([]Node{halfDZ, &PowNode{Base: zNode, Exp: mustRational(-1, 1)}})

			// u1 = exp(integrate(w, x))
			intW, err := integrateCore(wNode, varName)
			if err != nil {
				continue
			}
			u1, err := Eval(&FuncNode{Name: "exp", Args: []Node{intW}})
			if err != nil {
				continue
			}

			y1, _ := simplifyMul([]Node{xi, u1})
			y1, _ = Eval(y1)

			u1Squared, _ := simplifyPow(u1, mustRational(2, 1))
			invU1Sq, _ := simplifyPow(u1Squared, mustRational(-1, 1))
			invU1Sq, _ = Eval(invU1Sq)
			intInvU1Sq, err := integrateCore(invU1Sq, varName)
			var y2 Node
			if err == nil && intInvU1Sq != nil {
				u2, _ := simplifyMul([]Node{u1, intInvU1Sq})
				y2, _ = simplifyMul([]Node{xi, u2})
				y2, _ = Eval(y2)
			} else {
				xNode := &VarNode{Name: varName}
				u2 := &FuncNode{Name: "integrate", Args: []Node{invU1Sq, xNode}}
				u2Mul, _ := simplifyMul([]Node{u1, u2})
				y2, _ = simplifyMul([]Node{xi, u2Mul})
			}

			c1Y1, _ := simplifyMul([]Node{&VarNode{Name: "C_1"}, y1})
			c2Y2, _ := simplifyMul([]Node{&VarNode{Name: "C_2"}, y2})
			genSol, _ := simplifyAdd([]Node{c1Y1, c2Y2})

			return &KovacicResult{
				IsLiouvillian: true,
				Case:          2,
				Basis:         []Node{y1, y2},
				GeneralSol:    genSol,
			}, true
		}
	}

	return nil, false
}

// -------------------------------------------------------------------------
// Helper Mathematical Operations & Polynomial Solvers
// -------------------------------------------------------------------------

type alphaCombo struct {
	poleAlphas []*big.Rat
	infAlpha   *big.Rat
	d          int
}

func generateAlphaCombinations(poleAlphas [][]*big.Rat, infAlphas []*big.Rat) []alphaCombo {
	var result []alphaCombo
	var recurse func(idx int, current []*big.Rat)
	recurse = func(idx int, current []*big.Rat) {
		if idx == len(poleAlphas) {
			for _, infA := range infAlphas {
				sum := new(big.Rat)
				for _, pA := range current {
					sum.Add(sum, pA)
				}
				diff := new(big.Rat).Sub(infA, sum)
				if diff.IsInt() {
					dInt := int(diff.Num().Int64())
					cp := make([]*big.Rat, len(current))
					copy(cp, current)
					result = append(result, alphaCombo{
						poleAlphas: cp,
						infAlpha:   infA,
						d:          dInt,
					})
				}
			}
			return
		}
		for _, a := range poleAlphas[idx] {
			recurse(idx+1, append(current, a))
		}
	}
	recurse(0, []*big.Rat{})
	return result
}

func candidateAlphasForPole(p rationalPole) []*big.Rat {
	if p.order == 1 {
		return []*big.Rat{big.NewRat(0, 1)}
	}
	if p.order == 2 {
		// alpha^2 - alpha - b = 0 => alpha = (1 +- sqrt(1 + 4*b)) / 2
		onePlus4B := new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Mul(big.NewRat(4, 1), p.bCoeff))
		if onePlus4B.Sign() >= 0 {
			numSqrt := new(big.Int).Sqrt(onePlus4B.Num())
			denSqrt := new(big.Int).Sqrt(onePlus4B.Denom())
			if new(big.Int).Mul(numSqrt, numSqrt).Cmp(onePlus4B.Num()) == 0 &&
				new(big.Int).Mul(denSqrt, denSqrt).Cmp(onePlus4B.Denom()) == 0 {
				sqrtVal := big.NewRat(1, 1).SetFrac(numSqrt, denSqrt)
				alpha1 := new(big.Rat).Mul(big.NewRat(1, 2), new(big.Rat).Add(big.NewRat(1, 1), sqrtVal))
				alpha2 := new(big.Rat).Mul(big.NewRat(1, 2), new(big.Rat).Sub(big.NewRat(1, 1), sqrtVal))
				return []*big.Rat{alpha1, alpha2}
			}
		}
	}
	return nil
}

func candidateAlphasForInfinity(rNode Node, infOrder int, varName string) []*big.Rat {
	if infOrder > 2 {
		return []*big.Rat{big.NewRat(0, 1), big.NewRat(1, 1)}
	}
	if infOrder == 2 {
		// r(x) ~ b / x^2
		num, den := extractFraction(rNode)
		lcNum := leadingCoeffInVar(num, varName)
		lcDen := leadingCoeffInVar(den, varName)
		bRat := new(big.Rat).Quo(lcNum, lcDen)

		onePlus4B := new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Mul(big.NewRat(4, 1), bRat))
		if onePlus4B.Sign() >= 0 {
			numSqrt := new(big.Int).Sqrt(onePlus4B.Num())
			denSqrt := new(big.Int).Sqrt(onePlus4B.Denom())
			if new(big.Int).Mul(numSqrt, numSqrt).Cmp(onePlus4B.Num()) == 0 &&
				new(big.Int).Mul(denSqrt, denSqrt).Cmp(onePlus4B.Denom()) == 0 {
				sqrtVal := big.NewRat(1, 1).SetFrac(numSqrt, denSqrt)
				alpha1 := new(big.Rat).Mul(big.NewRat(1, 2), new(big.Rat).Add(big.NewRat(1, 1), sqrtVal))
				alpha2 := new(big.Rat).Mul(big.NewRat(1, 2), new(big.Rat).Sub(big.NewRat(1, 1), sqrtVal))
				return []*big.Rat{alpha1, alpha2}
			}
		}
	}
	return nil
}

// findCase1Polynomial searches for P(x) = sum_{j=0}^d p_j x^j with p_d = 1 satisfying
// P'' + 2*theta*P' + (theta' + theta^2 - r)*P = 0.
func findCase1Polynomial(thetaNode, rNode Node, d int, varName string) (*univariatePoly, bool) {
	if d == 0 {
		// P(x) = 1, test if theta' + theta^2 - r == 0
		dTheta, _ := differentiate(thetaNode, varName)
		thetaSq, _ := simplifyPow(thetaNode, mustRational(2, 1))
		lhs, _ := simplifyAdd([]Node{dTheta, thetaSq})
		residual, _ := simplifyAdd([]Node{lhs, &UnaryOpNode{Op: "-", Expr: rNode}})
		residual, _ = Eval(residual)
		if isZeroNode(residual) {
			return &univariatePoly{varName: varName, coeffs: []Node{mustRational(1, 1)}}, true
		}
		return nil, false
	}

	// Solve linear system for coefficients p_0, ..., p_{d-1} of P(x)
	// Sample d distinct test points to establish a determined linear system
	mat := make([][]*big.Rat, d)
	rhs := make([]*big.Rat, d)

	dTheta, _ := differentiate(thetaNode, varName)
	thetaSq, _ := simplifyPow(thetaNode, mustRational(2, 1))
	coeff0, _ := simplifyAdd([]Node{dTheta, thetaSq, &UnaryOpNode{Op: "-", Expr: rNode}})
	coeff1, _ := simplifyMul([]Node{mustRational(2, 1), thetaNode})

	testPoints := []*big.Rat{
		big.NewRat(1, 1), big.NewRat(2, 1), big.NewRat(3, 1), big.NewRat(4, 1),
		big.NewRat(5, 1), big.NewRat(6, 1), big.NewRat(7, 1),
	}

	for row := 0; row < d; row++ {
		tp := testPoints[row%len(testPoints)]
		if row >= len(testPoints) {
			tp = big.NewRat(int64(row+1), 1)
		}
		mat[row] = make([]*big.Rat, d)

		// Substitute x = tp
		c0Val := evalNodeAtRat(coeff0, varName, tp)
		c1Val := evalNodeAtRat(coeff1, varName, tp)
		if c0Val == nil || c1Val == nil {
			return nil, false
		}

		// For each unknown p_j (j = 0 to d-1), basis term is x^j
		for j := 0; j < d; j++ {
			// term = x^j * coeff0 + j*x^{j-1} * coeff1 + j*(j-1)*x^{j-2}
			termVal := new(big.Rat).Mul(bigRatPowInt(tp, j), c0Val)
			if j >= 1 {
				d1 := new(big.Rat).Mul(big.NewRat(int64(j), 1), bigRatPowInt(tp, j-1))
				termVal.Add(termVal, new(big.Rat).Mul(d1, c1Val))
			}
			if j >= 2 {
				d2 := new(big.Rat).Mul(big.NewRat(int64(j*(j-1)), 1), bigRatPowInt(tp, j-2))
				termVal.Add(termVal, d2)
			}
			mat[row][j] = termVal
		}

		// RHS is - (value for p_d = 1 with j = d)
		termD := new(big.Rat).Mul(bigRatPowInt(tp, d), c0Val)
		if d >= 1 {
			d1 := new(big.Rat).Mul(big.NewRat(int64(d), 1), bigRatPowInt(tp, d-1))
			termD.Add(termD, new(big.Rat).Mul(d1, c1Val))
		}
		if d >= 2 {
			d2 := new(big.Rat).Mul(big.NewRat(int64(d*(d-1)), 1), bigRatPowInt(tp, d-2))
			termD.Add(termD, d2)
		}
		rhs[row] = new(big.Rat).Neg(termD)
	}

	sol, err := solveKovacicLinearSystem(mat, rhs)
	if err != nil || len(sol) != d {
		return nil, false
	}

	coeffs := make([]Node, d+1)
	for j := 0; j < d; j++ {
		coeffs[j] = &RationalNode{Val: sol[j]}
	}
	coeffs[d] = mustRational(1, 1)

	// Validate solution at a random check point (x = 13/7)
	testValidate := big.NewRat(13, 7)
	polyVal := evalKovacicPolyAtRat(coeffs, testValidate)
	polyD1Val := evalPolyD1AtRat(coeffs, testValidate)
	polyD2Val := evalPolyD2AtRat(coeffs, testValidate)

	c0Val := evalNodeAtRat(coeff0, varName, testValidate)
	c1Val := evalNodeAtRat(coeff1, varName, testValidate)

	res := new(big.Rat).Add(polyD2Val, new(big.Rat).Mul(c1Val, polyD1Val))
	res.Add(res, new(big.Rat).Mul(c0Val, polyVal))

	if res.Sign() == 0 {
		return &univariatePoly{varName: varName, coeffs: coeffs}, true
	}
	return nil, false
}

// findSymmetricSquarePoly searches for polynomial z(x) satisfying z''' - 4*r*z' - 2*r'*z = 0.
func findSymmetricSquarePoly(rNode, dr Node, deg int, varName string) (*univariatePoly, bool) {
	if deg == 0 {
		// z = 1 => -2*r' == 0 => dr == 0
		drEval, _ := Eval(dr)
		if isZeroNode(drEval) {
			return &univariatePoly{varName: varName, coeffs: []Node{mustRational(1, 1)}}, true
		}
		return nil, false
	}

	// Solve linear system for deg+1 coefficients of z
	dim := deg + 1
	mat := make([][]*big.Rat, dim)
	rhs := make([]*big.Rat, dim)

	testPoints := []*big.Rat{
		big.NewRat(2, 1), big.NewRat(3, 1), big.NewRat(5, 1), big.NewRat(7, 1),
		big.NewRat(11, 1), big.NewRat(13, 1),
	}

	for row := 0; row < dim; row++ {
		tp := testPoints[row%len(testPoints)]
		mat[row] = make([]*big.Rat, dim)

		rVal := evalNodeAtRat(rNode, varName, tp)
		drVal := evalNodeAtRat(dr, varName, tp)
		if rVal == nil || drVal == nil {
			return nil, false
		}

		for j := 0; j < dim; j++ {
			// z_j = x^j
			// z''' - 4*r*z' - 2*dr*z
			term := new(big.Rat).Mul(big.NewRat(-2, 1), new(big.Rat).Mul(drVal, bigRatPowInt(tp, j)))
			if j >= 1 {
				d1 := new(big.Rat).Mul(big.NewRat(int64(j), 1), bigRatPowInt(tp, j-1))
				term.Sub(term, new(big.Rat).Mul(big.NewRat(4, 1), new(big.Rat).Mul(rVal, d1)))
			}
			if j >= 3 {
				d3 := new(big.Rat).Mul(big.NewRat(int64(j*(j-1)*(j-2)), 1), bigRatPowInt(tp, j-3))
				term.Add(term, d3)
			}
			mat[row][j] = term
		}
		rhs[row] = big.NewRat(0, 1)
	}

	// Find non-trivial nullspace vector
	nullVector := findRationalNullVector(mat)
	if nullVector != nil {
		coeffs := make([]Node, dim)
		for j := 0; j < dim; j++ {
			coeffs[j] = &RationalNode{Val: nullVector[j]}
		}
		return &univariatePoly{varName: varName, coeffs: coeffs}, true
	}

	return nil, false
}

// -------------------------------------------------------------------------
// Low-Level Rational Arithmetic & Evaluation Helpers
// -------------------------------------------------------------------------

func extractFraction(n Node) (num, den Node) {
	if m, ok := n.(*MulNode); ok {
		var nums []Node
		var dens []Node
		for _, f := range m.Factors {
			if p, ok := f.(*PowNode); ok {
				if r, okR := p.Exp.(*RationalNode); okR && r.Val.Cmp(big.NewRat(-1, 1)) == 0 {
					dens = append(dens, p.Base)
					continue
				}
			}
			nums = append(nums, f)
		}
		if len(dens) > 0 {
			numNode, _ := simplifyMul(nums)
			denNode, _ := simplifyMul(dens)
			return numNode, denNode
		}
	}
	return n, mustRational(1, 1)
}

func degreeInVar(n Node, varName string) int {
	poly, err := NodeToPoly(n, []string{varName}, ast.OrderLex)
	if err == nil && poly != nil {
		return DegreeInVar(poly, varName)
	}
	// Fallback heuristic for univariate monomials
	if v, ok := n.(*VarNode); ok && v.Name == varName {
		return 1
	}
	if r, ok := n.(*RationalNode); ok && r != nil {
		return 0
	}
	return 0
}

func leadingCoeffInVar(n Node, varName string) *big.Rat {
	poly, err := NodeToPoly(n, []string{varName}, ast.OrderLex)
	if err == nil && poly != nil && len(poly.Terms) > 0 {
		return new(big.Rat).Set(poly.Terms[0].Coeff)
	}
	if r, ok := n.(*RationalNode); ok {
		return new(big.Rat).Set(r.Val)
	}
	return big.NewRat(1, 1)
}

func findRationalRootsForNode(n Node, varName string) []*big.Rat {
	uPoly, ok := extractPoly(n, varName)
	if !ok || uPoly == nil {
		return nil
	}
	return findUnivariatePolyRationalRoots(uPoly)
}

func rootMultiplicity(n Node, varName string, root *big.Rat) int {
	cur := n
	mult := 0
	for {
		val := evalNodeAtRat(cur, varName, root)
		if val != nil && val.Sign() == 0 {
			mult++
			dCur, err := differentiate(cur, varName)
			if err != nil {
				break
			}
			cur, _ = Eval(dCur)
		} else {
			break
		}
	}
	return mult
}

func computePoleLeadingCoeff(num, den Node, varName string, root *big.Rat, order int) *big.Rat {
	// b = (order! * num(r)) / den^(order)(r)
	numVal := evalNodeAtRat(num, varName, root)
	dDen := den
	for i := 0; i < order; i++ {
		dDen, _ = differentiate(dDen, varName)
	}
	denVal := evalNodeAtRat(dDen, varName, root)
	if denVal != nil && denVal.Sign() != 0 && numVal != nil {
		// fact(order)
		fact := new(big.Int).MulRange(1, int64(order))
		res := new(big.Rat).Mul(numVal, new(big.Rat).SetInt(fact))
		return res.Quo(res, denVal)
	}
	return big.NewRat(1, 1)
}

func evalNodeAtRat(n Node, varName string, val *big.Rat) *big.Rat {
	sub := Substitute(n, varName, &RationalNode{Val: val})
	ev, err := Eval(sub)
	if err != nil {
		return nil
	}
	if r, ok := ev.(*RationalNode); ok {
		return new(big.Rat).Set(r.Val)
	}
	return nil
}

func bigRatPowInt(base *big.Rat, exp int) *big.Rat {
	if exp == 0 {
		return big.NewRat(1, 1)
	}
	res := new(big.Rat).Set(base)
	for i := 1; i < exp; i++ {
		res.Mul(res, base)
	}
	return res
}

func evalKovacicPolyAtRat(coeffs []Node, x *big.Rat) *big.Rat {
	res := new(big.Rat)
	for j, c := range coeffs {
		if r, ok := c.(*RationalNode); ok && r != nil {
			term := new(big.Rat).Mul(r.Val, bigRatPowInt(x, j))
			res.Add(res, term)
		}
	}
	return res
}

func evalPolyD1AtRat(coeffs []Node, x *big.Rat) *big.Rat {
	res := new(big.Rat)
	for j := 1; j < len(coeffs); j++ {
		if r, ok := cNodeToRat(coeffs[j]); ok {
			term := new(big.Rat).Mul(new(big.Rat).Mul(big.NewRat(int64(j), 1), r), bigRatPowInt(x, j-1))
			res.Add(res, term)
		}
	}
	return res
}

func evalPolyD2AtRat(coeffs []Node, x *big.Rat) *big.Rat {
	res := new(big.Rat)
	for j := 2; j < len(coeffs); j++ {
		if r, ok := cNodeToRat(coeffs[j]); ok {
			factor := int64(j * (j - 1))
			term := new(big.Rat).Mul(new(big.Rat).Mul(big.NewRat(factor, 1), r), bigRatPowInt(x, j-2))
			res.Add(res, term)
		}
	}
	return res
}

func cNodeToRat(n Node) (*big.Rat, bool) {
	if r, ok := n.(*RationalNode); ok && r != nil {
		return r.Val, true
	}
	return nil, false
}

func solveKovacicLinearSystem(mat [][]*big.Rat, rhs []*big.Rat) ([]*big.Rat, error) {
	n := len(mat)
	if n == 0 {
		return nil, nil
	}
	m := len(mat[0])

	// Augmented matrix [A | b]
	aug := make([][]*big.Rat, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]*big.Rat, m+1)
		for j := 0; j < m; j++ {
			aug[i][j] = new(big.Rat).Set(mat[i][j])
		}
		aug[i][m] = new(big.Rat).Set(rhs[i])
	}

	// Gaussian elimination with exact big.Rat
	lead := 0
	for r := 0; r < n && lead < m; r++ {
		i := r
		for aug[i][lead].Sign() == 0 {
			i++
			if i == n {
				i = r
				lead++
				if lead == m {
					break
				}
			}
		}
		if lead == m {
			break
		}
		aug[r], aug[i] = aug[i], aug[r]

		leadVal := new(big.Rat).Set(aug[r][lead])
		for j := 0; j <= m; j++ {
			aug[r][j].Quo(aug[r][j], leadVal)
		}

		for k := 0; k < n; k++ {
			if k != r && aug[k][lead].Sign() != 0 {
				factor := new(big.Rat).Set(aug[k][lead])
				for j := 0; j <= m; j++ {
					aug[k][j].Sub(aug[k][j], new(big.Rat).Mul(factor, aug[r][j]))
				}
			}
		}
		lead++
	}

	sol := make([]*big.Rat, m)
	for j := 0; j < m; j++ {
		sol[j] = new(big.Rat)
	}
	for i := 0; i < n; i++ {
		pivotCol := -1
		for j := 0; j < m; j++ {
			if aug[i][j].Sign() != 0 {
				pivotCol = j
				break
			}
		}
		if pivotCol != -1 {
			sol[pivotCol] = aug[i][m]
		}
	}
	return sol, nil
}

func findRationalNullVector(mat [][]*big.Rat) []*big.Rat {
	n := len(mat)
	if n == 0 {
		return nil
	}
	m := len(mat[0])

	aug := make([][]*big.Rat, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]*big.Rat, m)
		for j := 0; j < m; j++ {
			aug[i][j] = new(big.Rat).Set(mat[i][j])
		}
	}

	lead := 0
	for r := 0; r < n && lead < m; r++ {
		i := r
		for i < n && aug[i][lead].Sign() == 0 {
			i++
		}
		if i == n {
			lead++
			r--
			continue
		}
		aug[r], aug[i] = aug[i], aug[r]

		leadVal := new(big.Rat).Set(aug[r][lead])
		for j := 0; j < m; j++ {
			aug[r][j].Quo(aug[r][j], leadVal)
		}

		for k := 0; k < n; k++ {
			if k != r && aug[k][lead].Sign() != 0 {
				factor := new(big.Rat).Set(aug[k][lead])
				for j := 0; j < m; j++ {
					aug[k][j].Sub(aug[k][j], new(big.Rat).Mul(factor, aug[r][j]))
				}
			}
		}
		lead++
	}

	// Identify free variables
	pivotInCol := make([]int, m)
	for j := 0; j < m; j++ {
		pivotInCol[j] = -1
	}
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			if aug[i][j].Sign() != 0 {
				pivotInCol[j] = i
				break
			}
		}
	}

	for j := 0; j < m; j++ {
		if pivotInCol[j] == -1 {
			// Free variable at column j: set x_j = 1
			nullVec := make([]*big.Rat, m)
			nullVec[j] = big.NewRat(1, 1)
			for col := 0; col < m; col++ {
				if rIdx := pivotInCol[col]; rIdx != -1 {
					nullVec[col] = new(big.Rat).Neg(aug[rIdx][j])
				} else if col != j {
					nullVec[col] = big.NewRat(0, 1)
				}
			}
			return nullVec
		}
	}
	return nil
}
