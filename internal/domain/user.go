package domain

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	errs "github.com/BALLUUNN/authStartGrowingUp/pkg"
	"github.com/dombox/uuidv7"
	"golang.org/x/crypto/bcrypt"
)

var (
	UsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)
	PhoneRegex    = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
)

type User struct {
	ID              uuidv7.UUID
	Nickname        string
	Password        string
	Email           string
	Phone           string
	IsVerifiedEmail bool
	IsVerifiedPhone bool
}

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

	user.Normalize()

	validationErrs := errs.NewValidationErrors(
		user.validateNickname(),
		user.validateEmail(),
		user.validatePhone(),
		user.validatePasswordStrength(),
	)
	if validationErrs != nil {
		return nil, validationErrs
	}

	if err := user.HashPassword(); err != nil {
		return nil, errs.NewInternalError(err)
	}

	return user, nil
}

func (u *User) String() string {
	return "User{" +
		"ID: " + u.ID.String() +
		", Nickname: " + u.Nickname +
		", IsVerifiedEmail: " + fmt.Sprintf("%v", u.IsVerifiedEmail) +
		", IsVerifiedPhone: " + fmt.Sprintf("%v", u.IsVerifiedPhone) +
		"}"
}

func (u *User) Validate() error {
	u.Normalize()
	return errs.NewValidationErrors(
		u.validateNickname(),
		u.validateEmail(),
		u.validatePhone(),
	)
}

func (u *User) Normalize() {
	u.Nickname = strings.TrimSpace(u.Nickname)
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	u.Phone = strings.TrimSpace(u.Phone)
}

func (u *User) validateNickname() *errs.AppError {
	if !UsernameRegex.MatchString(u.Nickname) {
		return errs.NewValidationError(
			"invalid nickname: must be 3-30 characters long and contain only letters, numbers, and underscores",
			nil,
			"nickname",
			u.Nickname,
		)
	}
	return nil
}

func (u *User) validateEmail() *errs.AppError {
	if len(u.Email) > 254 {
		return errs.NewValidationError("invalid email format", nil, "email", u.Email)
	}

	addr, err := mail.ParseAddress(u.Email)
	if err != nil || addr.Address != u.Email {
		return errs.NewValidationError("invalid email format", err, "email", u.Email)
	}

	return nil
}

func (u *User) validatePhone() *errs.AppError {
	if !PhoneRegex.MatchString(u.Phone) {
		return errs.NewValidationError(
			"invalid phone number: must be in E.164 format, for example +14155552671",
			nil,
			"phone",
			u.Phone,
		)
	}
	return nil
}

func (u *User) validatePasswordStrength() *errs.AppError {
	hasUpper, hasLower, hasNumber := false, false, false

	if len(u.Password) < 8 {
		return errs.NewValidationError(
			"invalid password: must be at least 8 characters long",
			nil,
			"password",
			"[redacted]",
		)
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
			"password",
			"[redacted]",
		)
	}

	return nil
}

func (u *User) HashPassword() error {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return errs.NewInternalError(err)
	}

	u.Password = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err == nil {
		return nil
	}

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return errs.NewWrongPasswordError("wrong password")
	}

	return errs.NewInternalError(err)
}

func (u *User) ComparePassword(password string) bool {
	return u.CheckPassword(password) == nil
}
