package provenance

import (
	"fmt"
	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/calc/ast"
)

// AlgebraicCertificate represents a high-level, checkable mathematical witness.
type AlgebraicCertificate interface {
	CertificateKind() string
	Summary() string
}

// AntiderivativeCertificate certifies an integration step by asserting F' = f.
type AntiderivativeCertificate struct {
	Integrand      ast.Node // f(x)
	Antiderivative ast.Node // F(x)
	Variable       string   // x
	Domain         string   // e.g. "R \ {0}", "interval [0, 1]"
}

func (c *AntiderivativeCertificate) CertificateKind() string {
	return "Antiderivative"
}

func (c *AntiderivativeCertificate) Summary() string {
	return fmt.Sprintf("d/d%s (%v) = %v", c.Variable, c.Antiderivative, c.Integrand)
}

// NullstellensatzCertificate certifies polynomial ideal membership or inconsistency.
type NullstellensatzCertificate struct {
	Target       ast.Node   // p(x)
	Generators   []ast.Node // f_i(x)
	Cofactors    []ast.Node // q_i(x) such that Target = sum(q_i * f_i)
	Inconsistent bool       // true if Target == 1
}

func (c *NullstellensatzCertificate) CertificateKind() string {
	return "Nullstellensatz"
}

func (c *NullstellensatzCertificate) Summary() string {
	if c.Inconsistent {
		return "1 = sum(q_i * f_i) [Inconsistent ideal certificate]"
	}
	return "p = sum(q_i * f_i) [Ideal membership certificate]"
}

// LimitCertificate certifies limit evaluation via indeterminate form and derivative ratio bounds.
type LimitCertificate struct {
	Expression      ast.Node // f(x)
	Variable        string   // x
	Point           ast.Node // a
	Direction       string   // "both", "left", "right"
	LHopitalOrder   int      // number of L'Hopital rule applications
	DerivativeRatio ast.Node // f'(x)/g'(x)
}

func (c *LimitCertificate) CertificateKind() string {
	return "Limit"
}

func (c *LimitCertificate) Summary() string {
	if c.LHopitalOrder > 0 {
		return fmt.Sprintf("L'Hopital order %d, ratio: %v", c.LHopitalOrder, c.DerivativeRatio)
	}
	return "Direct algebraic limit certificate"
}
