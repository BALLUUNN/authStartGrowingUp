package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	errs "github.com/BALLUUNN/authStartGrowingUp/pkg"
)

func TestCreateUserReturnsMultipleValidationErrors(t *testing.T) {
	_, err := CreateUser("x", "weak", "bad-email", "123")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !errors.Is(err, errs.ErrValidation) {
		t.Fatal("expected validation error kind")
	}

	var validationErrs errs.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(validationErrs.Items()) != 4 {
		t.Fatalf("expected 4 validation errors, got %d", len(validationErrs.Items()))
	}
}

func TestCreateUserNormalizesAndHashesPassword(t *testing.T) {
	user, err := CreateUser(" valid_user ", "Password1", " Test@Example.com ", " +14155552671 ")
	if err != nil {
		t.Fatalf("expected user to be created, got %v", err)
	}

	if user.Nickname != "valid_user" {
		t.Fatalf("expected trimmed nickname, got %q", user.Nickname)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}
	if user.Phone != "+14155552671" {
		t.Fatalf("expected trimmed phone, got %q", user.Phone)
	}
	if user.Password == "Password1" {
		t.Fatal("expected hashed password, got plain text")
	}
}

func TestCreateUserRejectsPasswordLongerThanBcryptLimit(t *testing.T) {
	password := strings.Repeat("A", 71) + "a1"

	_, err := CreateUser("valid_user", password, "test@example.com", "+14155552671")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !errors.Is(err, errs.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}

	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Field() != "password" {
		t.Fatalf("expected password field, got %q", appErr.Field())
	}
}

func TestCheckPasswordReturnsWrongPassword(t *testing.T) {
	user, err := CreateUser("valid_user", "Password1", "test@example.com", "+14155552671")
	if err != nil {
		t.Fatalf("expected user to be created, got %v", err)
	}

	err = user.CheckPassword("WrongPassword1")
	if err == nil {
		t.Fatal("expected wrong password error, got nil")
	}
	if !errors.Is(err, errs.ErrWrongPassword) {
		t.Fatalf("expected wrong password error, got %v", err)
	}
}

func TestCheckPasswordReturnsInternalForInvalidHash(t *testing.T) {
	user := &User{Password: "not-a-bcrypt-hash"}

	err := user.CheckPassword("Password1")
	if err == nil {
		t.Fatal("expected internal error, got nil")
	}
	if !errors.Is(err, errs.ErrInternal) {
		t.Fatalf("expected internal error, got %v", err)
	}
}

func TestCheckPasswordReturnsInternalForEmptyStoredHash(t *testing.T) {
	user := &User{}

	err := user.CheckPassword("Password1")
	if err == nil {
		t.Fatal("expected internal error, got nil")
	}
	if !errors.Is(err, errs.ErrInternal) {
		t.Fatalf("expected internal error, got %v", err)
	}
}

func TestUserJSONOmitsPassword(t *testing.T) {
	user, err := CreateUser("valid_user", "Password1", "test@example.com", "+14155552671")
	if err != nil {
		t.Fatalf("expected user to be created, got %v", err)
	}

	payload, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("expected JSON marshal to succeed, got %v", err)
	}
	if strings.Contains(string(payload), "Password1") {
		t.Fatalf("expected JSON payload to omit password, got %s", string(payload))
	}
	if strings.Contains(string(payload), "\"password\"") {
		t.Fatalf("expected password field to be omitted, got %s", string(payload))
	}
}
