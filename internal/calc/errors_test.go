package calc

import (
	"errors"
	"testing"

	"github.com/DovahkiinYuzuko/i-hate-decimal-calc/internal/i18n"
)

func TestCalcError_Is(t *testing.T) {
	errZero := NewZeroDivisionError("division by zero: test")
	if !errors.Is(errZero, ErrZeroDivisionSentinel) {
		t.Errorf("expected errZero to match ErrZeroDivisionSentinel via errors.Is")
	}

	errDomain := NewDomainError("errors.domain_error", "domain error")
	if errors.Is(errDomain, ErrZeroDivisionSentinel) {
		t.Errorf("expected errDomain NOT to match ErrZeroDivisionSentinel")
	}
	if !errors.Is(errDomain, ErrDomainSentinel) {
		t.Errorf("expected errDomain to match ErrDomainSentinel")
	}

	errMat := NewSingularMatrixError("matrix is singular")
	if !errors.Is(errMat, ErrSingularMatrixSentinel) {
		t.Errorf("expected errMat to match ErrSingularMatrixSentinel")
	}
}

func TestCalcError_Localization(t *testing.T) {
	_ = i18n.Init("en")

	err := NewZeroDivisionError("division by zero: denominator cannot be zero")
	// In English locale
	if i18n.CurrentLocale() == "en" {
		if err.Error() != "division by zero: denominator cannot be zero" {
			t.Errorf("unexpected en error string: got %q", err.Error())
		}
	}

	// Switch to Japanese
	_ = i18n.SetLocale("ja")
	defer func() { _ = i18n.SetLocale("en") }()

	if err.Error() != "ゼロ除算: 分母をゼロにすることはできません" {
		t.Errorf("unexpected ja error string: got %q", err.Error())
	}
}

func TestCalcError_FromEvaluation(t *testing.T) {
	// 1. Zero division in NewRational
	_, errRational := NewRational(1, 0)
	if !errors.Is(errRational, ErrZeroDivisionSentinel) {
		t.Errorf("expected NewRational(1, 0) to be ErrZeroDivisionSentinel, got: %v", errRational)
	}

	// 2. Domain error in NewFunc log(-5)
	_, errLog := NewFunc("log", []Node{mustRational(-5, 1)})
	if !errors.Is(errLog, ErrDomainSentinel) {
		t.Errorf("expected log(-5) to be ErrDomainSentinel, got: %v", errLog)
	}

	// 3. Singular matrix error in evalInv
	singularMat, _ := NewMatrix(2, 2, [][]Node{
		{mustRational(1, 1), mustRational(2, 1)},
		{mustRational(2, 1), mustRational(4, 1)},
	})
	invNode, _ := NewFunc("inv", []Node{singularMat})
	_, errInv := Eval(invNode)
	if !errors.Is(errInv, ErrSingularMatrixSentinel) {
		t.Errorf("expected inv of singular matrix to be ErrSingularMatrixSentinel, got: %v", errInv)
	}

	// 4. Syntax error in Parse
	_, errSyntax := Parse("1 + +")
	if !errors.Is(errSyntax, ErrSyntaxSentinel) {
		t.Errorf("expected '1 + +' to be ErrSyntaxSentinel, got: %v", errSyntax)
	}
}
