package errs

import (
	"errors"
	"fmt"
	"strings"
)

type Code string

const (
	CodeInternal      Code = "internal_error"
	CodeValidation    Code = "validation_error"
	CodeWrongPassword Code = "wrong_password"
)

var (
	ErrInternal      = &AppError{code: CodeInternal, message: "internal error"}
	ErrValidation    = &AppError{code: CodeValidation, message: "validation error"}
	ErrWrongPassword = &AppError{code: CodeWrongPassword, message: "wrong password"}
)

type AppError struct {
	code    Code
	message string
	err     error
	field   string
	value   any
}

type ValidationErrors []*AppError

func NewInternalError(err error) *AppError {
	return &AppError{
		code:    CodeInternal,
		message: "internal server error",
		err:     err,
	}
}

func NewValidationError(message string, err error, field string, value any) *AppError {
	return &AppError{
		code:    CodeValidation,
		message: message,
		err:     err,
		field:   field,
		value:   value,
	}
}

func NewWrongPasswordError(message string) *AppError {
	return &AppError{
		code:    CodeWrongPassword,
		message: message,
	}
}

func NewValidationErrors(errs ...error) error {
	var validationErrs ValidationErrors

	for _, err := range errs {
		if err == nil {
			continue
		}

		switch typed := err.(type) {
		case ValidationErrors:
			validationErrs = append(validationErrs, typed.Items()...)
		case *AppError:
			if typed == nil {
				continue
			}
			if !errors.Is(typed, ErrValidation) {
				return typed
			}
			validationErrs = append(validationErrs, typed)
		default:
			var appErr *AppError
			if errors.As(err, &appErr) && appErr != nil && errors.Is(appErr, ErrValidation) {
				validationErrs = append(validationErrs, appErr)
				continue
			}
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

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

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

func (e *AppError) Code() Code {
	if e == nil {
		return ""
	}
	return e.code
}

func (e *AppError) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

func (e *AppError) Field() string {
	if e == nil {
		return ""
	}
	return e.field
}

func (e *AppError) Value() any {
	if e == nil {
		return nil
	}
	return e.value
}

func (e *AppError) WithField(field string) *AppError {
	if e == nil {
		return nil
	}
	clone := *e
	clone.field = field
	return &clone
}

func (e *AppError) WithValue(value any) *AppError {
	if e == nil {
		return nil
	}
	clone := *e
	clone.value = value
	return &clone
}

func (e *AppError) WithFieldValue(field string, value any) *AppError {
	if e == nil {
		return nil
	}
	clone := *e
	clone.field = field
	clone.value = value
	return &clone
}

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

func (v ValidationErrors) Is(target error) bool {
	if len(v) == 0 || target == nil {
		return false
	}

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
