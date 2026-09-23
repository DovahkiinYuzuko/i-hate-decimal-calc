package domain

// Domain represents a mathematical ring/field over elements of type E.
// It acts as the parent algebraic structure and provides all operations on elements,
// preventing duplicate metadata in element representations.
type Domain[E any] interface {
	// Name returns the identifier of the domain (e.g., "Q", "F_17").
	Name() string

	// Zero returns the additive identity element.
	Zero() E

	// One returns the multiplicative identity element.
	One() E

	// IsZero returns true if a is the additive identity.
	IsZero(a E) bool

	// IsOne returns true if a is the multiplicative identity.
	IsOne(a E) bool

	// Equals returns true if a and b are algebraically identical.
	Equals(a, b E) bool

	// Add returns a + b.
	Add(a, b E) E

	// Sub returns a - b.
	Sub(a, b E) E

	// Mul returns a * b.
	Mul(a, b E) E

	// Neg returns -a.
	Neg(a E) E

	// String returns a human-readable representation of a.
	String(a E) string

	// Clone returns an independent deep copy of a.
	Clone(a E) E
}

// FieldDomain represents a field where every non-zero element has a multiplicative inverse.
type FieldDomain[E any] interface {
	Domain[E]

	// Inv returns the multiplicative inverse a^(-1). Returns an error if a is zero.
	Inv(a E) (E, error)

	// Div returns a / b (equivalent to a * b^(-1)). Returns an error if b is zero.
	Div(a, b E) (E, error)
}
