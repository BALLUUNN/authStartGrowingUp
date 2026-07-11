// Package errs provides a structured error handling system for the application.
// It defines a hierarchy of error codes, supports error wrapping, and allows
// adding contextual fields (field-value pairs) for validation and logging.
//
// The package is built around the AppError type, which implements the standard
// error interface and supports unwrapping. It also provides a ValidationErrors
// type for aggregating multiple validation errors into a single error.
package errs

import (
	"errors"
	"fmt"
	"strings"
)

// Code represents a machine-readable error code used for categorizing errors.
// These codes are useful for clients to programmatically handle different
// error types (e.g., map to HTTP status codes or gRPC codes).
type Code string

// Error codes for common error scenarios in the application.
const (
	CodeInternal      Code = "internal_error"   // Unexpected system errors
	CodeValidation    Code = "validation_error" // Input validation failures
	CodeWrongPassword Code = "wrong_password"   // Authentication: incorrect password
	CodeInvalidToken  Code = "invalid_token"    // Token is malformed, revoked, or not found
	CodeExpiredToken  Code = "expired_token"    // Token has exceeded its validity period
)

// Sentinel errors are exported for use with errors.Is() to check error types.
// They are pre-created AppError instances with default messages.
var (
	ErrInternal      = &AppError{code: CodeInternal, message: "internal error"}
	ErrValidation    = &AppError{code: CodeValidation, message: "validation error"}
	ErrWrongPassword = &AppError{code: CodeWrongPassword, message: "wrong password"}
	ErrInvalidToken  = &AppError{code: CodeInvalidToken, message: "invalid token"}
	ErrExpiredToken  = &AppError{code: CodeExpiredToken, message: "token expired"}
)

// AppError is the core error type for the application.
// It carries a code, a human-readable message, an optional wrapped error,
// and optional field-value metadata (useful for validation errors).
type AppError struct {
	code    Code   // Machine-readable error code
	message string // Human-readable message (safe for clients)
	err     error  // Wrapped underlying error (for logging/debugging)
	field   string // Field name (e.g., "email") for validation errors
	value   any    // Associated value (e.g., invalid input) for that field
}

// ValidationErrors is a collection of AppError instances, specifically for
// aggregating multiple validation failures. It implements error and supports
// unwrapping into a slice of errors.
type ValidationErrors []*AppError

// NewInternalError creates a new AppError with CodeInternal and a generic
// message "internal server error". The provided err is wrapped for logging.
func NewInternalError(err error) *AppError {
	return &AppError{
		code:    CodeInternal,
		message: "internal server error",
		err:     err,
	}
}

// NewValidationError creates a new AppError with CodeValidation.
// It includes a custom message and an optional wrapped error.
func NewValidationError(message string, err error) *AppError {
	return &AppError{
		code:    CodeValidation,
		message: message,
		err:     err,
	}
}

// NewWrongPasswordError creates an error for incorrect password attempts.
func NewWrongPasswordError(message string, err error) *AppError {
	return &AppError{
		code:    CodeWrongPassword,
		message: message,
		err:     err,
	}
}

// NewInvalidTokenError creates an error for invalid tokens (not found, revoked, etc.).
func NewInvalidTokenError(message string, err error) *AppError {
	return &AppError{
		code:    CodeInvalidToken,
		message: message,
		err:     err,
	}
}

// NewExpiredTokenError creates an error for expired tokens.
func NewExpiredTokenError(message string, err error) *AppError {
	return &AppError{
		code:    CodeExpiredToken,
		message: message,
		err:     err,
	}
}

// NewValidationErrors aggregates multiple errors into a single error.
// It filters out nil errors and collects only errors that are of type
// CodeValidation. If any non-validation error is encountered, it is
// returned immediately (no aggregation). Returns nil if no validation errors.
// If exactly one validation error exists, it returns that error directly.
// Otherwise, it returns a ValidationErrors collection.
func NewValidationErrors(errs ...error) error {
	var validationErrs ValidationErrors

	for _, err := range errs {
		if err == nil {
			continue
		}

		switch typed := err.(type) {
		case ValidationErrors:
			// Unpack nested collections and append their items.
			validationErrs = append(validationErrs, typed.Items()...)
		case *AppError:
			if typed == nil {
				continue
			}
			// If it's not a validation error, return it immediately.
			if !errors.Is(typed, ErrValidation) {
				return typed
			}
			validationErrs = append(validationErrs, typed)
		default:
			// Try to extract an AppError via errors.As to handle wrapped errors.
			var appErr *AppError
			if errors.As(err, &appErr) && appErr != nil && errors.Is(appErr, ErrValidation) {
				validationErrs = append(validationErrs, appErr)
				continue
			}
			// Unknown error type → return as is.
			return err
		}
	}

	switch len(validationErrs) {
	case 0:
		return nil
	case 1:
		return validationErrs[0]
	default:
		return validationErrs
	}
}

// Error implements the error interface for AppError.
// It combines the message and the wrapped error (if any) into a single string.
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.message
}

// Unwrap returns the wrapped error, enabling errors.Is/As to traverse the chain.
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// Is implements custom equality logic for errors.Is.
// Two AppError instances are considered equal if they have the same Code.
// This allows sentinel errors (like ErrValidation) to match any error with
// that code, even if the messages differ.
func (e *AppError) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}

	var targetAppErr *AppError
	if !errors.As(target, &targetAppErr) {
		return false
	}

	return e.code != "" && e.code == targetAppErr.code
}

// Code returns the error code.
func (e *AppError) Code() Code {
	if e == nil {
		return ""
	}
	return e.code
}

// Message returns the human-readable error message.
func (e *AppError) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// Field returns the field name associated with the error (for validation).
func (e *AppError) Field() string {
	if e == nil {
		return ""
	}
	return e.field
}

// Value returns the value associated with the field (for validation).
func (e *AppError) Value() any {
	if e == nil {
		return nil
	}
	return e.value
}

// WithField returns a clone of the error with the field set.
// It preserves the original error's code, message, and wrapped error.
func (e *AppError) WithField(field string) *AppError {
	if e == nil {
		return nil
	}
	clone := *e
	clone.field = field
	return &clone
}

// WithValue returns a clone of the error with the value set.
func (e *AppError) WithValue(value any) *AppError {
	if e == nil {
		return nil
	}
	clone := *e
	clone.value = value
	return &clone
}

// WithFieldValue returns a clone with both field and value set.
func (e *AppError) WithFieldValue(field string, value any) *AppError {
	if e == nil {
		return nil
	}
	clone := *e
	clone.field = field
	clone.value = value
	return &clone
}

// Error implements the error interface for ValidationErrors.
// It concatenates the Error() strings of all contained errors, separated by "; ".
// If the collection is empty, it returns the default message from ErrValidation.
func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ErrValidation.Message()
	}

	parts := make([]string, 0, len(v))
	for _, err := range v {
		if err == nil {
			continue
		}
		parts = append(parts, err.Error())
	}
	if len(parts) == 0 {
		return ErrValidation.Message()
	}
	return strings.Join(parts, "; ")
}

// Unwrap returns the contained errors as a slice of error.
// This enables errors.Is/As to iterate over the collection.
func (v ValidationErrors) Unwrap() []error {
	if len(v) == 0 {
		return nil
	}

	errs := make([]error, 0, len(v))
	for _, err := range v {
		if err == nil {
			continue
		}
		errs = append(errs, err)
	}
	return errs
}

// Is checks whether the target error matches any error in the collection.
// It returns true if the target is ErrValidation, or if any contained error
// matches the target via errors.Is.
func (v ValidationErrors) Is(target error) bool {
	if len(v) == 0 || target == nil {
		return false
	}

	// The collection itself is considered a validation error.
	if errors.Is(ErrValidation, target) {
		return true
	}

	for _, err := range v {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// Items returns a slice of non-nil AppError pointers from the collection.
// Useful for iterating over the actual error objects.
func (v ValidationErrors) Items() []*AppError {
	if len(v) == 0 {
		return nil
	}

	items := make([]*AppError, 0, len(v))
	for _, err := range v {
		if err == nil {
			continue
		}
		items = append(items, err)
	}
	return items
}
