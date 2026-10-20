package provenance

import (
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// PathKind indicates the branch/subexpression targeted in an AST.
type PathKind string

const (
	PathRoot  PathKind = "root"
	PathArg   PathKind = "arg"
	PathNum   PathKind = "num"
	PathDen   PathKind = "den"
	PathLeft  PathKind = "left"
	PathRight PathKind = "right"
	PathBody  PathKind = "body"
)

// PathElem represents a single step in a typed structural path to a subexpression.
type PathElem struct {
	Kind  PathKind
	Index int
}

// RelationKind defines the mathematical relation established by a derivation step.
type RelationKind string

const (
	RelationEq                 RelationKind = "Eq"                 // =
	RelationIff                RelationKind = "Iff"                // ↔
	RelationImplies            RelationKind = "Implies"            // →
	RelationLe                 RelationKind = "Le"                 // ≤
	RelationLt                 RelationKind = "Lt"                 // <
	RelationEquivalentModuloC  RelationKind = "EquivalentModuloC"  // = mod C (Integration constant equivalence)
)

// TrustLevel defines the formal certification status of a derivation step.
type TrustLevel string

const (
	// TrustCertified indicates the step is fully verified by Lean 4 or kernel rules.
	TrustCertified TrustLevel = "Certified"

	// TrustAlgebraicallyVerified indicates verified by internal algebra checks, but without a full Lean tactic.
	TrustAlgebraicallyVerified TrustLevel = "AlgebraicallyVerified"

	// TrustHeuristic indicates an explanatory step without formal verification.
	TrustHeuristic TrustLevel = "Heuristic"
)

// Visibility controls how derivation steps are presented to human readers.
type Visibility string

const (
	// VisPresentable marks coarse, user-facing pedagogical milestones.
	VisPresentable Visibility = "Presentable"

	// VisInternal marks formal intermediate micro-steps that are folded by default.
	VisInternal Visibility = "Internal"

	// VisStructural marks algebraic normalization (associativity, commutativity, sign reordering).
	VisStructural Visibility = "Structural"
)

// SideCondition records an explicit mathematical requirement for a rewrite to remain sound.
type SideCondition struct {
	Kind        string   // e.g., "NonZero", "Positive", "NonNegative", "BranchCut", "Differentiable"
	Target      ast.Node // Expression subject to the condition
	Description string   // Semantic description or parameter
}

// DerivationNode is the atomic unit of the fine-grained derivation DAG / tree.
type DerivationNode struct {
	ID          int64
	RuleID      string
	Relation    RelationKind
	Path        []PathElem
	Before      ast.Node
	After       ast.Node
	Parameters  map[string]ast.Node
	Conditions  []SideCondition
	Trust       TrustLevel
	Vis         Visibility
	Certificate AlgebraicCertificate
	Children    []*DerivationNode
}
