package calc

import (
	"fmt"
	"strings"
)

// InvertibilityCertificate represents a certified matrix inversion:
// M * Inverse == I, with NonDegeneracy condition det(M) != 0.
type InvertibilityCertificate struct {
	BaseCertificate
	Matrix      Node
	Inverse     Node
	Determinant Node
}

// NewInvertibilityCertificate creates a new verified InvertibilityCertificate.
func NewInvertibilityCertificate(matrix, inverse, det, residual Node, isVerified bool, details string) *InvertibilityCertificate {
	eqStr := ""
	if matrix != nil && inverse != nil {
		eqStr = fmt.Sprintf("%s * %s == I", matrix.String(), inverse.String())
	}
	var conds []Node
	if det != nil {
		conds = append(conds, det)
	}

	return &InvertibilityCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainMatrix,
			Verified:     isVerified,
			ResidualNode: residual,
			Conditions:   conds,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Matrix:      matrix,
		Inverse:     inverse,
		Determinant: det,
	}
}

// DecompositionKind defines the type of matrix decomposition.
type DecompositionKind string

const (
	DecompLU       DecompositionKind = "lu"
	DecompQR       DecompositionKind = "qr"
	DecompCholesky DecompositionKind = "cholesky"
	DecompLDLT     DecompositionKind = "ldlt"
	DecompPinv     DecompositionKind = "pinv"
)

// DecompositionCertificate represents a certified matrix factor decomposition:
// e.g. A == P * L * U or A == Q * R.
type DecompositionCertificate struct {
	BaseCertificate
	Kind    DecompositionKind
	Matrix  Node
	Factors []Node
}

// NewDecompositionCertificate creates a new verified DecompositionCertificate.
func NewDecompositionCertificate(kind DecompositionKind, matrix Node, factors []Node, residual Node, isVerified bool, details string) *DecompositionCertificate {
	var factorStrs []string
	for _, f := range factors {
		if f != nil {
			factorStrs = append(factorStrs, f.String())
		}
	}
	eqStr := ""
	if matrix != nil && len(factorStrs) > 0 {
		eqStr = fmt.Sprintf("%s == %s", matrix.String(), strings.Join(factorStrs, " * "))
	}

	return &DecompositionCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainMatrix,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		Kind:    kind,
		Matrix:  matrix,
		Factors: factors,
	}
}

// SmithNormalFormCertificate represents a certified Smith Normal Form decomposition U * A * V = D.
type SmithNormalFormCertificate struct {
	BaseCertificate
	A                Node
	U                Node
	V                Node
	D                Node
	InvariantFactors []Node
}

// NewSmithNormalFormCertificate creates a new verified SmithNormalFormCertificate.
func NewSmithNormalFormCertificate(a, u, v, d Node, invFactors []Node, residual Node, isVerified bool, details string) *SmithNormalFormCertificate {
	eqStr := ""
	if a != nil && u != nil && v != nil && d != nil {
		eqStr = fmt.Sprintf("%s * %s * %s == %s", u.String(), a.String(), v.String(), d.String())
	}
	return &SmithNormalFormCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainMatrix,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		A:                a,
		U:                u,
		V:                v,
		D:                d,
		InvariantFactors: invFactors,
	}
}

// HermiteNormalFormCertificate represents a certified Hermite Normal Form decomposition U * A = H.
type HermiteNormalFormCertificate struct {
	BaseCertificate
	A Node
	U Node
	H Node
}

// NewHermiteNormalFormCertificate creates a new verified HermiteNormalFormCertificate.
func NewHermiteNormalFormCertificate(a, u, h, residual Node, isVerified bool, details string) *HermiteNormalFormCertificate {
	eqStr := ""
	if a != nil && u != nil && h != nil {
		eqStr = fmt.Sprintf("%s * %s == %s", u.String(), a.String(), h.String())
	}
	return &HermiteNormalFormCertificate{
		BaseCertificate: BaseCertificate{
			DomainVal:    CertDomainMatrix,
			Verified:     isVerified,
			ResidualNode: residual,
			DetailsMsg:   details,
			EquationStr:  eqStr,
		},
		A: a,
		U: u,
		H: h,
	}
}
