package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestAppErrorSupportsErrorsIs(t *testing.T) {
	root := errors.New("root")
	err := NewInternalError(root)

	if !errors.Is(err, ErrInternal) {
		t.Fatal("expected errors.Is(err, ErrInternal) to be true")
	}
	if !errors.Is(err, root) {
		t.Fatal("expected wrapped root error to be discoverable")
	}
}

func TestNewValidationErrorsAggregatesValidationErrors(t *testing.T) {
	err := NewValidationErrors(
		NewValidationError("invalid email", nil).WithFieldValue("email", "bad"),
		NewValidationError("invalid password", nil).WithFieldValue("password", "[redacted]"),
	)
	if err == nil {
		t.Fatal("expected aggregated validation error, got nil")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatal("expected aggregated validation error to match ErrValidation")
	}

	var validationErrs ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(validationErrs.Items()) != 2 {
		t.Fatalf("expected 2 validation errors, got %d", len(validationErrs.Items()))
	}
}

func TestNewValidationErrorsReturnsFirstNonValidationError(t *testing.T) {
	internalErr := NewInternalError(errors.New("root"))

	err := NewValidationErrors(
		NewValidationError("invalid email", nil).WithFieldValue("email", "bad"),
		internalErr,
	)
	if err == nil {
		t.Fatal("expected non-validation error, got nil")
	}
	if !errors.Is(err, ErrInternal) {
		t.Fatal("expected internal error to be returned")
	}
	if errors.Is(err, ErrValidation) {
		t.Fatal("did not expect internal error to be classified as validation")
	}
}

func TestWithFieldValueClonesError(t *testing.T) {
	base := ErrValidation.WithFieldValue("email", "bad@example")

	if base == ErrValidation {
		t.Fatal("expected cloned error instance")
	}
	if base.Field() != "email" {
		t.Fatalf("expected field email, got %q", base.Field())
	}
	if base.Value() != "bad@example" {
		t.Fatalf("expected stored value, got %#v", base.Value())
	}
	if !errors.Is(base, ErrValidation) {
		t.Fatal("expected cloned error to match ErrValidation")
	}
}

func TestNewValidationErrorsIgnoresTypedNilAppError(t *testing.T) {
	var validationErr *AppError

	err := NewValidationErrors(validationErr)
	if err != nil {
		t.Fatalf("expected nil error, got %#v", err)
	}
}

func TestTokenErrorsSupportErrorsIs(t *testing.T) {
	invalidTokenErr := NewInvalidTokenError("refresh token is invalid", nil)
	expiredTokenErr := NewExpiredTokenError("refresh token expired", nil)

	if !errors.Is(invalidTokenErr, ErrInvalidToken) {
		t.Fatal("expected invalid token error to match ErrInvalidToken")
	}
	if !errors.Is(expiredTokenErr, ErrExpiredToken) {
		t.Fatal("expected expired token error to match ErrExpiredToken")
	}
}

func ExampleAppError_WithFieldValue() {
	err := NewValidationError("invalid email", nil).WithFieldValue("email", "bad@example.com")

	fmt.Println(err.Message())
	fmt.Println(err.Field())
	fmt.Println(err.Value())
	fmt.Println(errors.Is(err, ErrValidation))

	// Output:
	// invalid email
	// email
	// bad@example.com
	// true
}

func ExampleNewValidationErrors() {
	err := NewValidationErrors(
		NewValidationError("invalid email", nil).WithFieldValue("email", "bad@example.com"),
		NewValidationError("password is too short", nil).WithFieldValue("password", "[redacted]"),
	)

	fmt.Println(errors.Is(err, ErrValidation))

	var validationErrs ValidationErrors
	fmt.Println(errors.As(err, &validationErrs))

	for _, item := range validationErrs.Items() {
		fmt.Printf("%s: %s (%s=%v)\n", item.Code(), item.Message(), item.Field(), item.Value())
	}

	// Output:
	// true
	// true
	// validation_error: invalid email (email=bad@example.com)
	// validation_error: password is too short (password=[redacted])
}
