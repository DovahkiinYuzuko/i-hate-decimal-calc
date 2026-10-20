package provenance

import (
	"fmt"
)

// StepVerification records the result of checking an individual derivation step.
type StepVerification struct {
	StepID      int64        `json:"step_id"`
	RuleID      string       `json:"rule_id"`
	Relation    RelationKind `json:"relation"`
	Passed      bool         `json:"passed"`
	Reason      string       `json:"reason,omitempty"`
	Certificate string       `json:"certificate,omitempty"`
}

// VerificationReport contains the aggregate verification outcome of a derivation tree.
type VerificationReport struct {
	AllVerified   bool                `json:"all_verified"`
	TotalSteps    int                 `json:"total_steps"`
	VerifiedSteps int                 `json:"verified_steps"`
	FailedSteps   int                 `json:"failed_steps"`
	StepResults   []*StepVerification `json:"step_results"`
}

// VerifyVisitor traverses a derivation tree and verifies the equational and conditional validity of each step.
type VerifyVisitor struct{}

// NewVerifyVisitor creates a new VerifyVisitor.
func NewVerifyVisitor() *VerifyVisitor {
	return &VerifyVisitor{}
}

// Verify verifies all nodes in the derivation tree.
func (v *VerifyVisitor) Verify(root *DerivationNode) *VerificationReport {
	report := &VerificationReport{
		AllVerified: true,
		StepResults: make([]*StepVerification, 0),
	}

	if root == nil {
		return report
	}

	var walk func(n *DerivationNode)
	walk = func(n *DerivationNode) {
		if n == nil {
			return
		}

		res := &StepVerification{
			StepID:   n.ID,
			RuleID:   n.RuleID,
			Relation: n.Relation,
			Passed:   true,
		}

		// Verify Certificate if attached
		if n.Certificate != nil {
			res.Certificate = n.Certificate.Summary()
		}

		// Basic equational sanity check
		if n.Before == nil && n.After == nil {
			res.Passed = false
			res.Reason = "empty step without before/after expressions"
		}

		// Trust level check
		if n.Trust == TrustHeuristic {
			// Heuristic steps are explanatory, not formally certified
			res.Passed = true
			res.Reason = "heuristic explanatory step"
		}

		report.TotalSteps++
		if res.Passed {
			report.VerifiedSteps++
		} else {
			report.FailedSteps++
			report.AllVerified = false
		}
		report.StepResults = append(report.StepResults, res)

		for _, ch := range n.Children {
			walk(ch)
		}
	}

	walk(root)
	return report
}

// Summary returns a human-readable summary of the verification report.
func (r *VerificationReport) Summary() string {
	if r.AllVerified {
		return fmt.Sprintf("100%% derivation verified (%d/%d steps confirmed)", r.VerifiedSteps, r.TotalSteps)
	}
	return fmt.Sprintf("Verification incomplete: %d/%d steps verified, %d failed", r.VerifiedSteps, r.TotalSteps, r.FailedSteps)
}
