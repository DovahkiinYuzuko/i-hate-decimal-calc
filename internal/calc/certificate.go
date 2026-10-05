package calc

import (
	"fmt"
	"strings"
)

// CertificateDomain defines the mathematical domain of the verified operation.
type CertificateDomain string

const (
	CertDomainIdentity      CertificateDomain = "identity"
	CertDomainFactor        CertificateDomain = "factor"
	CertDomainCalculus      CertificateDomain = "calculus"
	CertDomainODE           CertificateDomain = "ode"
	CertDomainMatrix        CertificateDomain = "matrix"
	CertDomainRationalApart CertificateDomain = "rational"
	CertDomainWZ            CertificateDomain = "wz"
	CertDomainGeometry      CertificateDomain = "geometry"
	CertDomainImpossibility CertificateDomain = "impossibility"
	CertDomainEquivalence   CertificateDomain = "equivalence"
	CertDomainNumberTheory  CertificateDomain = "number_theory"
	CertDomainElliptic      CertificateDomain = "elliptic"
	CertDomainRoot          CertificateDomain = "root"
	CertDomainLattice       CertificateDomain = "lattice"
	CertDomainGeneral       CertificateDomain = "general"
)


// Certificate represents a certified mathematical proof of correctness.
// It serves as an immutable intermediate representation (IR) between the CAS solver,
// internal verification engine, and external formal provers (Lean 4, Coq, etc.).
type Certificate interface {
	Domain() CertificateDomain
	IsVerified() bool
	Residual() Node
	NonDegeneracyConditions() []Node
	Details() string
	EquationString() string
	String() string
}

// BaseCertificate holds standard fields common to all algebraic certificates.
type BaseCertificate struct {
	DomainVal       CertificateDomain
	Verified        bool
	ResidualNode    Node
	Conditions      []Node
	DetailsMsg      string
	EquationStr     string
}

// Domain returns the mathematical domain of the certificate.
func (b *BaseCertificate) Domain() CertificateDomain {
	return b.DomainVal
}

// IsVerified returns whether the verification check succeeded.
func (b *BaseCertificate) IsVerified() bool {
	return b.Verified
}

// Residual returns the zero-residual node (expected to evaluate to 0 for successful verification).
func (b *BaseCertificate) Residual() Node {
	return b.ResidualNode
}

// NonDegeneracyConditions returns required non-zero / non-empty hypotheses (e.g. det(M) != 0, denom != 0).
func (b *BaseCertificate) NonDegeneracyConditions() []Node {
	return b.Conditions
}

// Details returns supplementary diagnostic or explanation messages.
func (b *BaseCertificate) Details() string {
	return b.DetailsMsg
}

// EquationString returns the mathematical equation representing the certificate.
func (b *BaseCertificate) EquationString() string {
	return b.EquationStr
}

// String returns a human-readable CLI representation of the certificate.
func (b *BaseCertificate) String() string {
	if b == nil {
		return ""
	}
	if b.Verified {
		return fmt.Sprintf("[VERIFIED: %s]", b.EquationStr)
	}
	return fmt.Sprintf("[FAILED VERIFICATION: %s] %s", b.EquationStr, b.DetailsMsg)
}

// LinearSolveCertificate represents a certified solution to a linear system A * x = b.
type LinearSolveCertificate struct {
	BaseCertificate
	Matrix   Node
	Solution Node
	Target   Node
}

// NewLinearSolveCertificate creates a new verified LinearSolveCertificate.
func NewLinearSolveCertificate(matrix, solution, target, residual Node, isVerified bool, details string) *LinearSolveCertificate {
	eqStr := ""
	if matrix != nil && solution != nil && target != nil {
		eqStr = fmt.Sprintf("%s * %s == %s", matrix.String(), solution.String(), target.String())
	}
	return &LinearSolveCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainMatrix,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Matrix:   matrix,
		Solution: solution,
		Target:   target,
	}
}

// PellCertificate represents a certified integer solution to Pell's equation x^2 - d * y^2 = 1.
type PellCertificate struct {
	BaseCertificate
	D Node
	X Node
	Y Node
}

// NewPellCertificate creates a new verified PellCertificate.
func NewPellCertificate(d, x, y, residual Node, isVerified bool, details string) *PellCertificate {
	eqStr := ""
	if d != nil && x != nil && y != nil {
		eqStr = fmt.Sprintf("%s^2 - %s * %s^2 == 1", x.String(), d.String(), y.String())
	}
	return &PellCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainNumberTheory,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		D: d,
		X: x,
		Y: y,
	}
}

// PolynomialRootCertificate represents a certified root alpha of an algebraic polynomial P(alpha) = 0.
type PolynomialRootCertificate struct {
	BaseCertificate
	Polynomial Node
	Variable   string
	Root       Node
}

// NewPolynomialRootCertificate creates a new verified PolynomialRootCertificate.
func NewPolynomialRootCertificate(poly Node, variable string, root, residual Node, isVerified bool, details string) *PolynomialRootCertificate {
	eqStr := ""
	if poly != nil && root != nil {
		eqStr = fmt.Sprintf("%s [%s = %s] == 0", poly.String(), variable, root.String())
	}
	return &PolynomialRootCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainRoot,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Polynomial: poly,
		Variable:   variable,
		Root:       root,
	}
}

// IntegerRelationCertificate represents a certified linear integer relation sum(c_i * alpha_i) = 0.
type IntegerRelationCertificate struct {
	BaseCertificate
	Elements     []Node
	Coefficients []Node
}

// NewIntegerRelationCertificate creates a new verified IntegerRelationCertificate.
func NewIntegerRelationCertificate(elements, coeffs []Node, residual Node, isVerified bool, details string) *IntegerRelationCertificate {
	var terms []string
	for i := 0; i < len(elements) && i < len(coeffs); i++ {
		terms = append(terms, fmt.Sprintf("%s * %s", coeffs[i].String(), elements[i].String()))
	}
	eqStr := fmt.Sprintf("%s == 0", strings.Join(terms, " + "))
	return &IntegerRelationCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainLattice,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Elements:     elements,
		Coefficients: coeffs,
	}
}

// EllipticPointCertificate represents a certified rational point addition P1 + P2 = Sum on y^2 = x^3 + a*x + b.
type EllipticPointCertificate struct {
	BaseCertificate
	CurveA Node
	CurveB Node
	P1     Node
	P2     Node
	Sum    Node
}

// NewEllipticPointCertificate creates a new verified EllipticPointCertificate.
func NewEllipticPointCertificate(a, b, p1, p2, sum, residual Node, isVerified bool, details string) *EllipticPointCertificate {
	eqStr := ""
	if p1 != nil && p2 != nil && sum != nil {
		eqStr = fmt.Sprintf("%s + %s == %s", p1.String(), p2.String(), sum.String())
	}
	return &EllipticPointCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainElliptic,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		CurveA: a,
		CurveB: b,
		P1:     p1,
		P2:     p2,
		Sum:    sum,
	}
}

// LinearDiophantineCertificate represents a certified general integer solution to a*x + b*y = c.
type LinearDiophantineCertificate struct {
	BaseCertificate
	A         Node
	B         Node
	C         Node
	XSolution Node
	YSolution Node
	ParamVar  string
}

// NewLinearDiophantineCertificate creates a new verified LinearDiophantineCertificate.
func NewLinearDiophantineCertificate(a, b, c, xSol, ySol Node, paramVar string, residual Node, isVerified bool, details string) *LinearDiophantineCertificate {
	eqStr := ""
	if a != nil && b != nil && c != nil && xSol != nil && ySol != nil {
		eqStr = fmt.Sprintf("%s * (%s) + %s * (%s) == %s", a.String(), xSol.String(), b.String(), ySol.String(), c.String())
	}
	return &LinearDiophantineCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainNumberTheory,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		A:         a,
		B:         b,
		C:         c,
		XSolution: xSol,
		YSolution: ySol,
		ParamVar:  paramVar,
	}
}


