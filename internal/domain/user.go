package domain

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"

	errs "github.com/BALLUUNN/authStartGrowingUp/pkg"
	"github.com/dombox/uuidv7"
	"golang.org/x/crypto/bcrypt"
)

// Precompiled regular expressions for validation.
// These are global to avoid recompilation on each call.
var (
	// UsernameRegex validates nicknames: 3-30 alphanumeric characters or underscores.
	UsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)
	// PhoneRegex validates E.164 phone numbers (e.g., +14155552671).
	PhoneRegex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
)

const (
	minPasswordLength = 8
	maxPasswordBytes  = 72
)

// User represents the core domain entity for an authenticated user.
// It contains personal information, credentials (hashed password), and verification flags.
type User struct {
	ID              uuidv7.UUID // Unique identifier for the user
	Nickname        string      // Display name, must be unique and match UsernameRegex
	Password        string      // Hashed password (bcrypt), never exposed via JSON
	Email           string      // Normalized email address (lowercase)
	Phone           string      // Phone number in E.164 format
	IsVerifiedEmail bool        // Whether the email has been verified
	IsVerifiedPhone bool        // Whether the phone number has been verified
}

// CreateUser constructs a new User with validation, normalization, and password hashing.
// It generates a new UUID, normalizes fields, validates all constraints,
// and hashes the password using bcrypt.
// Returns a ValidationError if any field fails validation, or InternalError on system issues.
func CreateUser(nickname, password, email, phone string) (*User, error) {
	uuid, err := uuidv7.New()
	if err != nil {
		return nil, errs.NewInternalError(err)
	}

	user := &User{
		ID:       uuid,
		Nickname: nickname,
		Password: password,
		Email:    email,
		Phone:    phone,
	}

	// Normalize fields (trim spaces, lowercase email)
	user.Normalize()

	// Collect all validation errors
	validationErrs := errs.NewValidationErrors(
		user.validateNickname(),
		user.validateEmail(),
		user.validatePhone(),
		user.validatePasswordStrength(),
	)
	if validationErrs != nil {
		return nil, validationErrs
	}

	// Hash the password before storing
	if err := user.HashPassword(); err != nil {
		return nil, errs.NewInternalError(err)
	}

	return user, nil
}

// String returns a safe string representation of the user.
// It omits sensitive fields (email, phone, password) to avoid logging PII.
func (u *User) String() string {
	if u == nil {
		return "User<nil>"
	}

	return "User{" +
		"ID: " + u.ID.String() +
		", Nickname: " + u.Nickname +
		", IsVerifiedEmail: " + fmt.Sprintf("%v", u.IsVerifiedEmail) +
		", IsVerifiedPhone: " + fmt.Sprintf("%v", u.IsVerifiedPhone) +
		"}"
}

// Validate performs profile validation without password checks.
// This is intended for update operations where the password is not being changed.
// It normalizes fields and returns ValidationErrors for any invalid fields.
func (u *User) Validate() error {
	if u == nil {
		return errs.NewInternalError(errors.New("user is nil"))
	}

	u.Normalize()
	return errs.NewValidationErrors(
		u.validateNickname(),
		u.validateEmail(),
		u.validatePhone(),
	)
}

// Normalize trims spaces and converts email to lowercase for consistent storage.
func (u *User) Normalize() {
	if u == nil {
		return
	}

	u.Nickname = strings.TrimSpace(u.Nickname)
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	u.Phone = strings.TrimSpace(u.Phone)
}

// validateNickname ensures the nickname matches the allowed pattern.
func (u *User) validateNickname() *errs.AppError {
	if u == nil {
		return errs.NewInternalError(errors.New("user is nil"))
	}

	if !UsernameRegex.MatchString(u.Nickname) {
		return errs.NewValidationError(
			"invalid nickname: must be 3-30 characters long and contain only letters, numbers, and underscores",
			nil,
		).WithFieldValue("nickname", u.Nickname)
	}
	return nil
}

// validateEmail uses net/mail.ParseAddress to validate format and length.
// It also ensures the email does not exceed 254 characters.
func (u *User) validateEmail() *errs.AppError {
	if u == nil {
		return errs.NewInternalError(errors.New("user is nil"))
	}

	if len(u.Email) > 254 {
		return errs.NewValidationError("invalid email format", nil).WithFieldValue("email", u.Email)
	}

	addr, err := mail.ParseAddress(u.Email)
	if err != nil || addr.Address != u.Email {
		return errs.NewValidationError("invalid email format", nil).WithFieldValue("email", u.Email)
	}

	return nil
}

// validatePhone checks that the phone number complies with E.164 format.
func (u *User) validatePhone() *errs.AppError {
	if u == nil {
		return errs.NewInternalError(errors.New("user is nil"))
	}

	if !PhoneRegex.MatchString(u.Phone) {
		return errs.NewValidationError(
			"invalid phone number: must be in E.164 format, for example +14155552671",
			nil,
		).WithFieldValue("phone", u.Phone)
	}
	return nil
}

// validatePasswordStrength enforces minimum length and character diversity:
// at least 8 characters, containing uppercase, lowercase, and a number.
func (u *User) validatePasswordStrength() *errs.AppError {
	if u == nil {
		return errs.NewInternalError(errors.New("user is nil"))
	}

	hasUpper, hasLower, hasNumber := false, false, false

	if utf8.RuneCountInString(u.Password) < minPasswordLength {
		return errs.NewValidationError(
			"invalid password: must be at least 8 characters long",
			nil,
		).WithFieldValue("password", "[redacted]")
	}

	if len(u.Password) > maxPasswordBytes {
		return errs.NewValidationError(
			"invalid password: must be at most 72 bytes long",
			nil,
		).WithFieldValue("password", "[redacted]")
	}

	for _, char := range u.Password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasNumber = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber {
		return errs.NewValidationError(
			"invalid password: must contain uppercase, lowercase, and a number",
			nil,
		).WithFieldValue("password", "[redacted]")
	}

	return nil
}

// HashPassword hashes the plaintext password using bcrypt with default cost.
// The resulting hash is stored in the Password field.
func (u *User) HashPassword() error {
	if u == nil {
		return errs.NewInternalError(errors.New("user is nil"))
	}

	if err := u.validatePasswordStrength(); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return errs.NewInternalError(err)
	}

	u.Password = string(hash)
	return nil
}

// CheckPassword compares a plaintext password against the stored bcrypt hash.
// Returns nil if the password matches, WrongPasswordError on mismatch,
// or InternalError for any other bcrypt error (e.g., malformed hash).
func (u *User) CheckPassword(password string) error {
	if u == nil {
		return errs.NewInternalError(errors.New("user is nil"))
	}
	if u.Password == "" {
		return errs.NewInternalError(errors.New("stored password hash is empty")).
			WithFieldValue("password", "[redacted]")
	}

	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err == nil {
		return nil
	}

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return errs.NewWrongPasswordError("wrong password", nil)
	}

	return errs.NewInternalError(err)
}

// ComparePassword is a deprecated convenience wrapper around CheckPassword.
// It hides the error type and returns only a boolean.
// Deprecated: use CheckPassword instead to get detailed error information.
func (u *User) ComparePassword(password string) bool {
	return u.CheckPassword(password) == nil
}
