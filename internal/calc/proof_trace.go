package calc

import (
	"fmt"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
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
		return nil, fmt.Errorf("cannot convert nil certificate")
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
			RuleDescription: fmt.Sprintf("Antiderivative of %s with respect to %s", Format(c.Integrand), c.Variable),
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
			RuleDescription: "Multiplicative inverse satisfying A * A^-1 = I",
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
			RuleDescription: fmt.Sprintf("Matrix decomposition into factors: %s", strings.Join(factorStr, " * ")),
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
			RuleDescription: "Vector solution satisfying A * x = b",
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
			RuleDescription: "Geometric proposition mechanically verified via characteristic set pseudo-division",
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
			RuleDescription: fmt.Sprintf("Fundamental solution (x, y) = (%s, %s) satisfying x^2 - %s*y^2 = 1", Format(c.X), Format(c.Y), Format(c.D)),
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
			RuleDescription: fmt.Sprintf("Root %s satisfying P(%s) = 0", Format(c.Root), c.Variable),
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
			RuleDescription: "Particular solution satisfying a*x + b*y = c",
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
			RuleDescription: "Diagonal invariant factor decomposition satisfying U * A * V = D",
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
			RuleDescription: "Row-echelon triangular canonical form satisfying U * A = H",
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
		return nil, fmt.Errorf("cannot convert nil verification certificate")
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
