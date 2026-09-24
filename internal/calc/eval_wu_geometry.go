package calc

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// WuProofState represents the lifecycle state of a geometric proof.
type WuProofState string

const (
	WuStateInit               WuProofState = "Initialized"
	WuStateLinearSubstituted  WuProofState = "LinearSubstituted"
	WuStateTriangularized     WuProofState = "Triangularized"
	WuStateNDGClassified      WuProofState = "NDGClassified"
	WuStateConclusionReduced  WuProofState = "ConclusionReduced"
	WuStateCertified          WuProofState = "Certified"
	WuStateInconclusive       WuProofState = "Inconclusive"
)

// WuProofLifecycleFSM enforces valid transitions in geometric automated theorem proving.
type WuProofLifecycleFSM struct {
	CurrentState WuProofState
}

// NewWuProofLifecycleFSM creates a new FSM in initialized state.
func NewWuProofLifecycleFSM() *WuProofLifecycleFSM {
	return &WuProofLifecycleFSM{CurrentState: WuStateInit}
}

// TransitionTo validates and performs state transition.
func (fsm *WuProofLifecycleFSM) TransitionTo(next WuProofState) error {
	switch fsm.CurrentState {
	case WuStateInit:
		if next == WuStateLinearSubstituted || next == WuStateTriangularized {
			fsm.CurrentState = next
			return nil
		}
	case WuStateLinearSubstituted:
		if next == WuStateTriangularized {
			fsm.CurrentState = next
			return nil
		}
	case WuStateTriangularized:
		if next == WuStateNDGClassified || next == WuStateConclusionReduced {
			fsm.CurrentState = next
			return nil
		}
	case WuStateNDGClassified:
		if next == WuStateConclusionReduced {
			fsm.CurrentState = next
			return nil
		}
	case WuStateConclusionReduced:
		if next == WuStateCertified || next == WuStateInconclusive {
			fsm.CurrentState = next
			return nil
		}
	}
	return fmt.Errorf("invalid Wu proof lifecycle transition: %s -> %s", fsm.CurrentState, next)
}

func init() {
	RegisterFunction(FunctionSpec{
		Name:    "geo_prove",
		MinArgs: 2,
		MaxArgs: 3,
		Handler: func(args []Node, env *Env) (Node, error) {
			return EvalGeoProve(args, env)
		},
	})
}

// EvalGeoProve evaluates the Wu's method geometric automated theorem proving engine.
// Syntax: geo_prove(hypotheses_list, conclusion [, variable_order_list])
func EvalGeoProve(args []Node, env *Env) (Node, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("geo_prove requires 2 or 3 arguments: geo_prove(hypotheses, conclusion [, var_order])")
	}

	fsm := NewWuProofLifecycleFSM()

	// 1. Parse hypotheses
	var rawHypotheses []Node
	if listNode, ok := args[0].(*ListNode); ok {
		rawHypotheses = listNode.Elements
	} else {
		rawHypotheses = []Node{args[0]}
	}

	var hypPolys []Node
	var constNDGs []Node

	for _, h := range rawHypotheses {
		trans, err := TranslateGeometricPredicate(h)
		if err != nil {
			return nil, err
		}
		for _, eq := range trans.Equations {
			evalEq, err := Eval(expandNode(eq))
			if err != nil {
				evalEq = eq
			}
			hypPolys = append(hypPolys, evalEq)
		}
		constNDGs = append(constNDGs, trans.ConstructionNDGs...)
	}

	// 2. Parse conclusion
	conclTrans, err := TranslateGeometricPredicate(args[1])
	if err != nil {
		return nil, err
	}
	if len(conclTrans.Equations) == 0 {
		return nil, fmt.Errorf("conclusion translated into empty equations")
	}
	conclPoly := conclTrans.Equations[0]
	evalConcl, err := Eval(expandNode(conclPoly))
	if err == nil {
		conclPoly = evalConcl
	}
	constNDGs = append(constNDGs, conclTrans.ConstructionNDGs...)

	// 3. Determine variable ordering
	var order []string
	if len(args) == 3 {
		if listNode, ok := args[2].(*ListNode); ok {
			for _, elem := range listNode.Elements {
				order = append(order, elem.String())
			}
		}
	}

	if len(order) == 0 {
		// Automatically collect variables and prioritize parameters u_ before construction variables x_
		allVarsMap := make(map[string]bool)
		for _, p := range hypPolys {
			for _, v := range collectVariables(p) {
				allVarsMap[v] = true
			}
		}
		for _, v := range collectVariables(conclPoly) {
			allVarsMap[v] = true
		}

		var params []string
		var coords []string
		for v := range allVarsMap {
			if strings.HasPrefix(v, "u") {
				params = append(params, v)
			} else {
				coords = append(coords, v)
			}
		}
		// Sort deterministically
		sort.Strings(params)
		sort.Strings(coords)
		order = append(params, coords...)
	}

	// 4. Build Ascending Chain
	_ = fsm.TransitionTo(WuStateLinearSubstituted)
	_ = fsm.TransitionTo(WuStateTriangularized)

	tSet, saturationInitials, err := BuildAscendingChain(hypPolys, order)
	if err != nil {
		return nil, err
	}

	// 5. Successive Pseudo-division of conclusion
	_ = fsm.TransitionTo(WuStateNDGClassified)
	_ = fsm.TransitionTo(WuStateConclusionReduced)

	chainElements := tSet.Elements
	rem, reductionInitials, _, err := SuccessiveReduce(conclPoly, chainElements, order)
	if err != nil {
		return nil, err
	}

	allInitials := append(saturationInitials, reductionInitials...)
	var uniqueInitials []Node
	seenInit := make(map[string]bool)
	for _, initNode := range allInitials {
		s := initNode.String()
		if !seenInit[s] && !isOne(initNode) && !isZero(initNode) {
			seenInit[s] = true
			uniqueInitials = append(uniqueInitials, initNode)
		}
	}

	isCertified := isZero(rem)
	if isCertified {
		_ = fsm.TransitionTo(WuStateCertified)
	} else {
		_ = fsm.TransitionTo(WuStateInconclusive)
	}

	details := ""
	state := VerifyStateCertified
	if isCertified {
		details = i18n.T("verify.geo_holds")
	} else {
		details = fmt.Sprintf(i18n.T("verify.geo_fails"), rem.String())
		state = VerifyStateRefuted
	}

	cert := NewGeometricCertificate(
		hypPolys,
		conclPoly,
		chainElements,
		nil,
		constNDGs,
		uniqueInitials,
		rem,
		isCertified,
		details,
	)

	// Save certificate to environment if available for CLI --verify and --lean flags
	if env != nil {
		env.LastCert = &VerificationCertificate{
			Domain:     DomainGeometry,
			Equation:   fmt.Sprintf("prem(%s, CS) == %s", conclPoly.String(), rem.String()),
			Residual:   rem,
			IsVerified: isCertified,
			Details:    details,
			State:      state,
			CertIR:     cert,
		}
	}

	if isCertified {
		return &ConstNode{Name: "true"}, nil
	}
	return &ConstNode{Name: "false"}, nil
}
