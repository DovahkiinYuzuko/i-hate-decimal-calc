package calc

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// ProofStep represents a single atomic rewriting or deductive step in an equational reasoning chain.
type ProofStep struct {
	Before          ast.Node
	After           ast.Node
	Rule            string
	RuleDescription string
	Residual        ast.Node
	IsSelfVerified  bool
}

// String returns a human-readable representation of the proof step.
func (s ProofStep) String() string {
	var b strings.Builder
	if s.Before != nil && s.After != nil {
		fmt.Fprintf(&b, "%s  ──[%s]──>  %s", Format(s.Before), s.Rule, Format(s.After))
	} else if s.After != nil {
		fmt.Fprintf(&b, "= %s (%s)", Format(s.After), s.Rule)
	}
	if s.IsSelfVerified {
		b.WriteString(" [VERIFIED]")
	}
	return b.String()
}

// ProofTrace holds the entire sequence of rewriting steps from the original expression to the final result.
// It serves as the single source of truth for --explain, --verify, and --lean.
type ProofTrace struct {
	Original ast.Node
	Result   ast.Node
	Steps    []ProofStep
	Metadata map[string]string
}

// NewProofTrace initializes a new ProofTrace with original and result nodes.
func NewProofTrace(original, result ast.Node) *ProofTrace {
	return &ProofTrace{
		Original: original,
		Result:   result,
		Steps:    make([]ProofStep, 0),
		Metadata: make(map[string]string),
	}
}

// AddStep appends a new proof step to the trace.
func (t *ProofTrace) AddStep(step ProofStep) {
	if t.Steps == nil {
		t.Steps = make([]ProofStep, 0)
	}
	t.Steps = append(t.Steps, step)
}

// StepCount returns the total number of steps in the trace.
func (t *ProofTrace) StepCount() int {
	return len(t.Steps)
}

// LastStep returns the most recent proof step, or nil if the trace is empty.
func (t *ProofTrace) LastStep() *ProofStep {
	if len(t.Steps) == 0 {
		return nil
	}
	return &t.Steps[len(t.Steps)-1]
}

// ConvertCertificateToProofTrace adapts any existing Certificate into a unified ProofTrace.
// This preserves backward compatibility with all existing CAS domains while routing verification
// and Lean 4 proof generation through a single canonical intermediate representation.
func ConvertCertificateToProofTrace(cert Certificate) (*ProofTrace, error) {
	if cert == nil {
		return nil, fmt.Errorf("%s", i18n.T("proof_trace.err_nil_certificate"))
	}

	switch c := cert.(type) {
	case *IdentityCertificate:
		trace := NewProofTrace(c.LHS, c.RHS)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["kind"] = string(c.Kind)
		trace.Metadata["equation"] = c.EquationString()
		rule := "AlgebraicIdentity"
		switch c.Kind {
		case IdentityKindFactor:
			rule = "PolynomialFactorization"
		case IdentityKindApart:
			rule = "PartialFractionExpansion"
		case IdentityKindEquivalence:
			rule = "AlgebraicEquivalence"
		}
		trace.AddStep(ProofStep{
			Before:          c.LHS,
			After:           c.RHS,
			Rule:            rule,
			RuleDescription: c.Details(),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *DerivCertificate:
		trace := NewProofTrace(c.Integrand, c.Antiderivative)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["variable"] = c.Variable
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.Integrand,
			After:           c.Antiderivative,
			Rule:            "IndefiniteIntegration",
			RuleDescription: i18n.T("proof_trace.rule_indefinite_integration", Format(c.Integrand), c.Variable),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *InvertibilityCertificate:
		trace := NewProofTrace(c.Matrix, c.Inverse)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.Matrix,
			After:           c.Inverse,
			Rule:            "MatrixInversion",
			RuleDescription: i18n.T("proof_trace.rule_matrix_inversion"),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *DecompositionCertificate:
		trace := NewProofTrace(c.Matrix, nil)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		var factorStr []string
		for _, f := range c.Factors {
			factorStr = append(factorStr, Format(f))
		}
		trace.AddStep(ProofStep{
			Before:          c.Matrix,
			After:           nil,
			Rule:            "MatrixDecomposition",
			RuleDescription: i18n.T("proof_trace.rule_matrix_decomposition", strings.Join(factorStr, " * ")),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *LinearSolveCertificate:
		trace := NewProofTrace(c.Target, c.Solution)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.Target,
			After:           c.Solution,
			Rule:            "LinearSolve",
			RuleDescription: i18n.T("proof_trace.rule_linear_solve"),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *GeometricCertificate:
		conclusionNode := c.Conclusion
		trueNode := &ast.ConstNode{Name: "true"}
		trace := NewProofTrace(conclusionNode, trueNode)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          conclusionNode,
			After:           trueNode,
			Rule:            "WuGeometricDeduction",
			RuleDescription: i18n.T("proof_trace.rule_wu_geometric"),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *PellCertificate:
		trace := NewProofTrace(c.D, c.X)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.D,
			After:           c.X,
			Rule:            "PellEquationSolve",
			RuleDescription: i18n.T("proof_trace.rule_pell_solve", Format(c.D)),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *PolynomialRootCertificate:
		trace := NewProofTrace(c.Polynomial, c.Root)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["variable"] = c.Variable
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.Polynomial,
			After:           c.Root,
			Rule:            "PolynomialRootVerification",
			RuleDescription: i18n.T("proof_trace.rule_polynomial_root", Format(c.Root), c.Variable),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *LinearDiophantineCertificate:
		trace := NewProofTrace(c.C, c.XSolution)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.C,
			After:           c.XSolution,
			Rule:            "LinearDiophantineSolve",
			RuleDescription: i18n.T("proof_trace.rule_linear_diophantine"),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *SmithNormalFormCertificate:
		trace := NewProofTrace(c.A, c.D)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.A,
			After:           c.D,
			Rule:            "SmithNormalForm",
			RuleDescription: i18n.T("proof_trace.rule_smith_normal_form"),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *HermiteNormalFormCertificate:
		trace := NewProofTrace(c.A, c.H)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.A,
			After:           c.H,
			Rule:            "HermiteNormalForm",
			RuleDescription: i18n.T("proof_trace.rule_hermite_normal_form"),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *ODECertificate:
		trace := NewProofTrace(c.ODE, c.Solution)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["dependent"] = c.DependentVar
		trace.Metadata["independent"] = c.IndependentVar
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.ODE,
			After:           c.Solution,
			Rule:            "ODESolution",
			RuleDescription: c.Details(),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *WZCertificate:
		trace := NewProofTrace(nil, nil)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Rule:            "WZHypergeometricIdentity",
			RuleDescription: c.Details(),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *IntegerRelationCertificate:
		zeroNode := &ast.RationalNode{Val: big.NewRat(0, 1)}
		trace := NewProofTrace(nil, zeroNode)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          nil,
			After:           zeroNode,
			Rule:            "IntegerRelationLLL",
			RuleDescription: i18n.T("proof_trace.rule_integer_relation"),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *EllipticPointCertificate:
		trace := NewProofTrace(c.P1, c.Sum)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.P1,
			After:           c.Sum,
			Rule:            "EllipticCurveAddition",
			RuleDescription: i18n.T("proof_trace.rule_elliptic_addition"),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	case *PrimitiveElementCertificate:
		thetaVar := &ast.VarNode{Name: c.SymbolTheta}
		trace := NewProofTrace(c.MinPolyTheta, thetaVar)
		trace.Metadata["domain"] = string(c.Domain())
		trace.Metadata["symbol_theta"] = c.SymbolTheta
		trace.Metadata["c"] = fmt.Sprintf("%d", c.C)
		trace.Metadata["equation"] = c.EquationString()
		trace.AddStep(ProofStep{
			Before:          c.MinPolyTheta,
			After:           thetaVar,
			Rule:            "PrimitiveElementIsomorphism",
			RuleDescription: i18n.T("proof_trace.rule_primitive_element"),
			Residual:        c.Residual(),
			IsSelfVerified:  c.IsVerified(),
		})
		return trace, nil

	default:
		// Generic fallback for any Certificate
		trace := NewProofTrace(nil, nil)
		trace.Metadata["domain"] = string(cert.Domain())
		trace.Metadata["equation"] = cert.EquationString()
		trace.AddStep(ProofStep{
			Rule:            string(cert.Domain()),
			RuleDescription: cert.Details(),
			Residual:        cert.Residual(),
			IsSelfVerified:  cert.IsVerified(),
		})
		return trace, nil
	}
}

// ConvertVerificationCertificateToProofTrace converts an eval_verify.go VerificationCertificate to a ProofTrace.
func ConvertVerificationCertificateToProofTrace(vc *VerificationCertificate) (*ProofTrace, error) {
	if vc == nil {
		return nil, fmt.Errorf("%s", i18n.T("proof_trace.err_nil_verification_certificate"))
	}
	if vc.CertIR != nil {
		return ConvertCertificateToProofTrace(vc.CertIR)
	}
	trace := NewProofTrace(nil, nil)
	trace.Metadata["domain"] = string(vc.Domain)
	trace.Metadata["equation"] = vc.Equation
	trace.AddStep(ProofStep{
		Rule:            string(vc.Domain),
		RuleDescription: vc.Details,
		Residual:        vc.Residual,
		IsSelfVerified:  vc.IsVerified,
	})
	return trace, nil
}

// matchNodesAlgebraically checks if two AST nodes are structurally or algebraically identical.
func matchNodesAlgebraically(a, b Node, env *Env) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Equal(b) {
		return true
	}
	negB, _ := simplifyUnaryOp("-", b)
	diff := NewAdd([]Node{a, negB})
	return checkIsZeroAlgebraically(diff, env)
}

// VerifyProofTrace checks that every step in the ProofTrace is mathematically valid via native residual zero checks.
// It structurally enforces a complete Chain of Trust and rejects empty or tampered traces.
func VerifyProofTrace(trace *ProofTrace, env *Env) (bool, error) {
	if trace == nil {
		return false, fmt.Errorf("%s", i18n.T("proof_trace.err_nil_proof_trace"))
	}
	if len(trace.Steps) == 0 {
		return false, fmt.Errorf("%s", i18n.T("proof_trace.err_empty_steps"))
	}

	// 1. Chain of trust validation:
	// - If Original is specified, it must bind algebraically to Step[0].Before
	if trace.Original != nil {
		if trace.Steps[0].Before == nil || !matchNodesAlgebraically(trace.Original, trace.Steps[0].Before, env) {
			return false, fmt.Errorf("%s", i18n.T("proof_trace.err_origin_mismatch"))
		}
	}

	// - Adjacent steps must chain: Step[i].After == Step[i+1].Before
	for i := 0; i < len(trace.Steps)-1; i++ {
		currAfter := trace.Steps[i].After
		nextBefore := trace.Steps[i+1].Before
		if currAfter != nil && nextBefore != nil {
			if !matchNodesAlgebraically(currAfter, nextBefore, env) {
				return false, fmt.Errorf("%s", i18n.T("proof_trace.err_broken_chain", i+1, i+2))
			}
		} else if (currAfter == nil) != (nextBefore == nil) {
			return false, fmt.Errorf("%s", i18n.T("proof_trace.err_broken_chain", i+1, i+2))
		}
	}

	// - If Result is specified, it must bind algebraically to Step[last].After
	lastStep := trace.Steps[len(trace.Steps)-1]
	if trace.Result != nil {
		if lastStep.After == nil || !matchNodesAlgebraically(lastStep.After, trace.Result, env) {
			return false, fmt.Errorf("%s", i18n.T("proof_trace.err_result_mismatch"))
		}
	}

	// 2. Validate all steps from scratch (never trust serialized IsSelfVerified flags)
	for i := range trace.Steps {
		step := &trace.Steps[i]

		var residual Node
		if step.Residual != nil {
			residual = step.Residual
		} else if step.Before != nil && step.After != nil {
			negAfter, _ := simplifyUnaryOp("-", step.After)
			residual = NewAdd([]Node{step.Before, negAfter})
		}

		if residual == nil {
			return false, fmt.Errorf("%s", i18n.T("proof_trace.err_no_residual_or_endpoints", i+1, step.Rule))
		}

		if !checkIsZeroAlgebraically(residual, env) {
			return false, fmt.Errorf("%s", i18n.T("proof_trace.err_residual_does_not_vanish", i+1, step.Rule))
		}

		step.IsSelfVerified = true
	}

	return true, nil
}

// RenderExplain formats the ProofTrace into a human-readable 2D Unicode tree.
func (t *ProofTrace) RenderExplain(lang string) string {
	if t == nil {
		return ""
	}

	prevLocale := i18n.CurrentLocale()
	if lang != "" {
		i18n.SetLocale(lang)
		defer i18n.SetLocale(prevLocale)
	}

	var b strings.Builder
	origStr := ""
	if t.Original != nil {
		origStr = Format(t.Original)
	} else if eq, ok := t.Metadata["equation"]; ok {
		origStr = eq
	}
	b.WriteString(i18n.T("trace.expression_header", origStr))

	if len(t.Steps) == 0 {
		b.WriteString(fmt.Sprintf("├── [%s]\n", i18n.T("trace.step_0_title")))
		b.WriteString(fmt.Sprintf("│   %s\n", i18n.T("trace.step_0_desc")))
		resStr := ""
		if t.Result != nil {
			resStr = Format(t.Result)
		}
		b.WriteString(fmt.Sprintf("└── %s\n    = %s\n", i18n.T("trace.result_header"), resStr))
		return b.String()
	}

	for i, step := range t.Steps {
		stepNum := i + 1
		ruleTitle := step.Rule
		if step.RuleDescription != "" {
			ruleTitle = fmt.Sprintf("%s (%s)", step.Rule, step.RuleDescription)
		}

		statusMark := i18n.T("trace.unverified_mark")
		if step.IsSelfVerified {
			statusMark = i18n.T("trace.verified_mark")
		}

		b.WriteString(fmt.Sprintf("├── %s %s\n", i18n.T("trace.step_header", stepNum, ruleTitle), statusMark))
		if step.Before != nil && step.After != nil {
			b.WriteString(fmt.Sprintf("│   %s  ──>  %s\n", Format(step.Before), Format(step.After)))
		}
		if step.Residual != nil {
			b.WriteString(fmt.Sprintf("│   Residual: %s\n", Format(step.Residual)))
		}
		if i < len(t.Steps)-1 {
			b.WriteString("│\n")
		}
	}

	b.WriteString(fmt.Sprintf("└── %s\n", i18n.T("trace.result_header")))
	resStr := ""
	if t.Result != nil {
		resStr = Format(t.Result)
	} else if len(t.Steps) > 0 && t.Steps[len(t.Steps)-1].After != nil {
		resStr = Format(t.Steps[len(t.Steps)-1].After)
	}
	b.WriteString(fmt.Sprintf("    = %s\n", resStr))

	return b.String()
}

// ProofStepJSON represents a single rewritten step in standalone JSON schema.
type ProofStepJSON struct {
	StepNumber      int    `json:"step_number"`
	Before          string `json:"before"`
	After           string `json:"after"`
	Rule            string `json:"rule"`
	RuleDescription string `json:"rule_description,omitempty"`
	Residual        string `json:"residual,omitempty"`
	IsSelfVerified  bool   `json:"is_self_verified"`
}

// ProofTraceJSON represents the full serialized algebraic certificate.
type ProofTraceJSON struct {
	Version    string            `json:"version"`
	Original   string            `json:"original"`
	Result     string            `json:"result"`
	IsVerified bool              `json:"is_verified"`
	TotalSteps int               `json:"total_steps"`
	Steps      []ProofStepJSON   `json:"steps"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ToJSON serializes the ProofTrace into indented JSON bytes.
func (t *ProofTrace) ToJSON() ([]byte, error) {
	if t == nil {
		return nil, fmt.Errorf("%s", i18n.T("proof_trace.err_serialize_nil_proof_trace"))
	}

	origStr := ""
	if t.Original != nil {
		origStr = Format(t.Original)
	}
	resStr := ""
	if t.Result != nil {
		resStr = Format(t.Result)
	}

	allVerified := true
	if len(t.Steps) == 0 {
		allVerified = false
	}

	stepsJSON := make([]ProofStepJSON, 0, len(t.Steps))
	for i, s := range t.Steps {
		if !s.IsSelfVerified {
			allVerified = false
		}
		beforeStr := ""
		if s.Before != nil {
			beforeStr = Format(s.Before)
		}
		afterStr := ""
		if s.After != nil {
			afterStr = Format(s.After)
		}
		residualStr := ""
		if s.Residual != nil {
			residualStr = Format(s.Residual)
		}
		stepsJSON = append(stepsJSON, ProofStepJSON{
			StepNumber:      i + 1,
			Before:          beforeStr,
			After:           afterStr,
			Rule:            s.Rule,
			RuleDescription: s.RuleDescription,
			Residual:        residualStr,
			IsSelfVerified:  s.IsSelfVerified,
		})
	}

	ptJSON := ProofTraceJSON{
		Version:    "1.0",
		Original:   origStr,
		Result:     resStr,
		IsVerified: allVerified,
		TotalSteps: len(t.Steps),
		Steps:      stepsJSON,
		Metadata:   t.Metadata,
	}

	return json.MarshalIndent(ptJSON, "", "  ")
}

// ToCanonicalJSON serializes the ProofTrace into an RFC 8785 (JCS) canonical JSON byte slice.
func (t *ProofTrace) ToCanonicalJSON() ([]byte, error) {
	raw, err := t.ToJSON()
	if err != nil {
		return nil, err
	}
	return CanonicalizeJSON(raw)
}

// ComputeProofHash computes the deterministic SHA-256 hex digest of the canonical RFC 8785 certificate.
func (t *ProofTrace) ComputeProofHash() (string, error) {
	raw, err := t.ToJSON()
	if err != nil {
		return "", err
	}
	return CanonicalHashSHA256(raw)
}

// SaveProofTraceJSON writes the ProofTrace as a JSON file to the specified filePath.
func SaveProofTraceJSON(trace *ProofTrace, filePath string) error {
	data, err := trace.ToJSON()
	if err != nil {
		return err
	}
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(filePath, data, 0644)
}

// LoadProofTraceFromJSON loads a ProofTrace from a JSON certificate file.
func LoadProofTraceFromJSON(filePath string) (*ProofTrace, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var ptJSON ProofTraceJSON
	if err := json.Unmarshal(data, &ptJSON); err != nil {
		return nil, fmt.Errorf("%s", fmt.Sprintf(i18n.T("proof_trace.err_invalid_certificate_json"), err.Error()))
	}

	var origNode ast.Node
	if ptJSON.Original != "" {
		parsed, err := Parse(ptJSON.Original)
		if err == nil {
			origNode = parsed
		}
	}

	var resNode ast.Node
	if ptJSON.Result != "" {
		parsed, err := Parse(ptJSON.Result)
		if err == nil {
			resNode = parsed
		}
	}

	trace := NewProofTrace(origNode, resNode)
	if ptJSON.Metadata != nil {
		trace.Metadata = ptJSON.Metadata
	}

	for _, sj := range ptJSON.Steps {
		var beforeNode ast.Node
		if sj.Before != "" {
			if b, err := Parse(sj.Before); err == nil {
				beforeNode = b
			}
		}
		var afterNode ast.Node
		if sj.After != "" {
			if a, err := Parse(sj.After); err == nil {
				afterNode = a
			}
		}
		var residualNode ast.Node
		if sj.Residual != "" {
			if r, err := Parse(sj.Residual); err == nil {
				residualNode = r
			}
		}

		trace.AddStep(ProofStep{
			Before:          beforeNode,
			After:           afterNode,
			Rule:            sj.Rule,
			RuleDescription: sj.RuleDescription,
			Residual:        residualNode,
			IsSelfVerified:  sj.IsSelfVerified,
		})
	}

	return trace, nil
}
