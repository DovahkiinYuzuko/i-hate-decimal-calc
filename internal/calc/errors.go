package calc

import (
	"fmt"
	"strings"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

// ErrCode represents machine-identifiable error codes for calc operations.
type ErrCode string

const (
	ErrZeroDivision      ErrCode = "zero_division"
	ErrDomain            ErrCode = "domain_error"
	ErrSyntax            ErrCode = "syntax_error"
	ErrSingularMatrix    ErrCode = "singular_matrix"
	ErrDimensionMismatch ErrCode = "dimension_mismatch"
	ErrUndefinedVar      ErrCode = "undefined_var"
	ErrUnknownFunction   ErrCode = "unknown_function"
	ErrInvalidArgs       ErrCode = "invalid_args"
)

// Sentinel error instances for use with errors.Is.
var (
	ErrZeroDivisionSentinel      = &CalcError{Code: ErrZeroDivision, Fallback: "division by zero"}
	ErrDomainSentinel            = &CalcError{Code: ErrDomain, Fallback: "domain error"}
	ErrSyntaxSentinel            = &CalcError{Code: ErrSyntax, Fallback: "syntax error"}
	ErrSingularMatrixSentinel    = &CalcError{Code: ErrSingularMatrix, Fallback: "matrix is singular (determinant is zero)"}
	ErrDimensionMismatchSentinel  = &CalcError{Code: ErrDimensionMismatch, Fallback: "dimension mismatch"}
	ErrUndefinedVarSentinel      = &CalcError{Code: ErrUndefinedVar, Fallback: "undefined variable"}
	ErrUnknownFunctionSentinel   = &CalcError{Code: ErrUnknownFunction, Fallback: "unknown function"}
	ErrInvalidArgsSentinel       = &CalcError{Code: ErrInvalidArgs, Fallback: "invalid arguments"}
)

// CalcError represents a structured, localizable mathematical or calculation error.
type CalcError struct {
	Code       ErrCode
	MessageKey string
	Args       []any
	Fallback   string
	Err        error
}

// Error formats the error string respecting the active i18n locale or fallback.
func (e *CalcError) Error() string {
	if e == nil {
		return ""
	}
	currentLoc := i18n.CurrentLocale()
	// If in a non-default locale (e.g. ja) and a message key is provided, try localized string
	if currentLoc != "" && currentLoc != i18n.DefaultLocale && e.MessageKey != "" {
		localized := i18n.T(e.MessageKey, e.Args...)
		if localized != e.MessageKey && localized != "" && !strings.Contains(localized, "%!(EXTRA") {
			return localized
		}
	}
	// Fallback to exact English message for backward compatibility
	if e.Fallback != "" {
		if len(e.Args) > 0 && strings.Contains(e.Fallback, "%") {
			return fmt.Sprintf(e.Fallback, e.Args...)
		}
		return e.Fallback
	}
	if e.MessageKey != "" {
		localized := i18n.T(e.MessageKey, e.Args...)
		if localized != e.MessageKey && localized != "" && !strings.Contains(localized, "%!(EXTRA") {
			return localized
		}
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

// Unwrap returns the underlying error if wrapped.
func (e *CalcError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Is supports errors.Is comparison by checking error code equivalence.
func (e *CalcError) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	if t, ok := target.(*CalcError); ok {
		return e.Code == t.Code
	}
	return false
}

// NewCalcError constructs a new *CalcError.
func NewCalcError(code ErrCode, msgKey, fallback string, args ...any) *CalcError {
	return &CalcError{
		Code:       code,
		MessageKey: msgKey,
		Args:       args,
		Fallback:   fallback,
	}
}

// NewZeroDivisionError constructs a zero division *CalcError.
func NewZeroDivisionError(fallback string, args ...any) *CalcError {
	msgKey := "errors.zero_division"
	if strings.Contains(fallback, "denominator cannot be zero") {
		msgKey = "errors.zero_division_denominator"
	} else if strings.Contains(fallback, "0^(negative number)") {
		msgKey = "errors.zero_division_pow"
	} else if strings.Contains(fallback, "in mod") {
		msgKey = "errors.zero_division_mod"
	} else if strings.Contains(fallback, "tan(") {
		msgKey = "errors.domain_tan"
	}
	return &CalcError{
		Code:       ErrZeroDivision,
		MessageKey: msgKey,
		Args:       args,
		Fallback:   fallback,
	}
}

// NewDomainError constructs a domain *CalcError.
func NewDomainError(msgKey, fallback string, args ...any) *CalcError {
	if msgKey == "" {
		msgKey = "errors.domain_error"
	}
	return &CalcError{
		Code:       ErrDomain,
		MessageKey: msgKey,
		Args:       args,
		Fallback:   fallback,
	}
}

// NewSingularMatrixError constructs a singular matrix *CalcError.
func NewSingularMatrixError(fallback string, args ...any) *CalcError {
	if fallback == "" {
		fallback = "matrix is singular (determinant is zero)"
	}
	return &CalcError{
		Code:       ErrSingularMatrix,
		MessageKey: "errors.singular_matrix",
		Args:       args,
		Fallback:   fallback,
	}
}

// NewDimensionMismatchError constructs a dimension mismatch *CalcError.
func NewDimensionMismatchError(msgKey, fallback string, args ...any) *CalcError {
	if msgKey == "" {
		msgKey = "errors.matrix_dim_mismatch"
	}
	return &CalcError{
		Code:       ErrDimensionMismatch,
		MessageKey: msgKey,
		Args:       args,
		Fallback:   fallback,
	}
}

// NewSyntaxError constructs a syntax *CalcError.
func NewSyntaxError(fallback string, args ...any) *CalcError {
	msgKey := "errors.syntax_error"
	if strings.Contains(fallback, "implicit multiplication") {
		msgKey = "errors.implicit_multiplication"
	}
	return &CalcError{
		Code:       ErrSyntax,
		MessageKey: msgKey,
		Args:       args,
		Fallback:   fallback,
	}
}
