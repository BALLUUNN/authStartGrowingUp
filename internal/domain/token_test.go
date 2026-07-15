package domain

import (
	"errors"
	"testing"
	"time"

	errs "github.com/BALLUUNN/authStartGrowingUp/pkg/errs"
	"github.com/dombox/uuidv7"
)

func TestNewRefreshTokenSuccess(t *testing.T) {
	userID := mustUUID(t)
	expiresAt := time.Now().Add(time.Hour).UTC().Unix()

	token, err := NewRefreshToken(userID, validTokenValue(), expiresAt)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if token == nil {
		t.Fatal("expected refresh token, got nil")
	}
	if token.ID == uuidv7.Nil() {
		t.Fatal("expected generated token ID")
	}
	if token.UserID != userID {
		t.Fatalf("expected user ID %v, got %v", userID, token.UserID)
	}
	if token.Token != validTokenValue() {
		t.Fatalf("expected token to be preserved, got %q", token.Token)
	}
	if token.ExpiresAt != expiresAt {
		t.Fatalf("expected expires_at %d, got %d", expiresAt, token.ExpiresAt)
	}
}

func TestNewRefreshTokenValidationFailures(t *testing.T) {
	validUserID := mustUUID(t)
	expiresAt := time.Now().Add(time.Hour).UTC().Unix()

	tests := []struct {
		name         string
		userID       uuidv7.UUID
		token        string
		expiresAt    int64
		wantField    string
		wantValue    any
		wantSentinel error
		wantCode     errs.Code
	}{
		{
			name:         "rejects nil user id",
			userID:       uuidv7.Nil(),
			token:        validTokenValue(),
			expiresAt:    expiresAt,
			wantField:    "user_id",
			wantValue:    uuidv7.Nil(),
			wantSentinel: errs.ErrInternal,
			wantCode:     errs.CodeInternal,
		},
		{
			name:         "rejects empty token",
			userID:       validUserID,
			token:        "",
			expiresAt:    expiresAt,
			wantField:    "token",
			wantValue:    "[redacted]",
			wantSentinel: errs.ErrInternal,
			wantCode:     errs.CodeInternal,
		},
		{
			name:         "rejects token shorter than minimum length",
			userID:       validUserID,
			token:        "short-token",
			expiresAt:    expiresAt,
			wantField:    "token",
			wantValue:    "[redacted]",
			wantSentinel: errs.ErrInternal,
			wantCode:     errs.CodeInternal,
		},
		{
			name:         "rejects non-positive expiry",
			userID:       validUserID,
			token:        validTokenValue(),
			expiresAt:    0,
			wantField:    "expires_at",
			wantValue:    int64(0),
			wantSentinel: errs.ErrInternal,
			wantCode:     errs.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := NewRefreshToken(tt.userID, tt.token, tt.expiresAt)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if token != nil {
				t.Fatalf("expected nil token, got %#v", token)
			}
			if !errors.Is(err, tt.wantSentinel) {
				t.Fatalf("expected errors.Is(err, %v) to be true, got err=%v", tt.wantSentinel, err)
			}

			var appErr *errs.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected AppError, got %T", err)
			}
			if appErr.Code() != tt.wantCode {
				t.Fatalf("expected code %q, got %q", tt.wantCode, appErr.Code())
			}
			if appErr.Field() != tt.wantField {
				t.Fatalf("expected field %q, got %q", tt.wantField, appErr.Field())
			}
			if appErr.Value() != tt.wantValue {
				t.Fatalf("expected value %#v, got %#v", tt.wantValue, appErr.Value())
			}
		})
	}
}

func TestRefreshTokenCheckSuccess(t *testing.T) {
	rt := validRefreshToken(t, time.Now().Add(time.Minute).UTC().Unix())

	if err := rt.Check(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestRefreshTokenCheckReturnsExpiredToken(t *testing.T) {
	expiredAt := time.Now().Add(-time.Minute).UTC().Unix()
	rt := validRefreshToken(t, expiredAt)

	err := rt.Check()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errs.ErrExpiredToken) {
		t.Fatalf("expected expired token error, got %v", err)
	}

	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code() != errs.CodeExpiredToken {
		t.Fatalf("expected code %q, got %q", errs.CodeExpiredToken, appErr.Code())
	}
	if appErr.Field() != "expires_at" {
		t.Fatalf("expected field expires_at, got %q", appErr.Field())
	}
	if appErr.Value() != expiredAt {
		t.Fatalf("expected expires_at value %d, got %#v", expiredAt, appErr.Value())
	}
}

func TestRefreshTokenCheckReturnsInternalErrorForCorruptedEntity(t *testing.T) {
	rt := validRefreshToken(t, time.Now().Add(time.Hour).UTC().Unix())
	rt.Token = ""

	err := rt.Check()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errs.ErrInternal) {
		t.Fatalf("expected internal error, got %v", err)
	}

	var appErr *errs.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Field() != "token" {
		t.Fatalf("expected field token, got %q", appErr.Field())
	}
	if appErr.Value() != "[redacted]" {
		t.Fatalf("expected redacted token value, got %#v", appErr.Value())
	}
}

func TestRefreshTokenIsExpiredUsesClockSkew(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt int64
		want      bool
	}{
		{
			name:      "treats near-expiry token as expired",
			expiresAt: time.Now().Add(3 * time.Second).UTC().Unix(),
			want:      true,
		},
		{
			name:      "keeps comfortably valid token active",
			expiresAt: time.Now().Add(20 * time.Second).UTC().Unix(),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := validRefreshToken(t, tt.expiresAt)
			if got := rt.isExpired(); got != tt.want {
				t.Fatalf("expected isExpired=%t, got %t", tt.want, got)
			}
		})
	}
}

func TestRefreshTokenVerifyReturnsInvalidToken(t *testing.T) {
	rt := validRefreshToken(t, time.Now().Add(time.Hour).UTC().Unix())

	err := rt.Verify(validTokenValue() + "-wrong")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errs.ErrInvalidToken) {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}

func TestRefreshTokenVerifyReturnsExpiredToken(t *testing.T) {
	rt := validRefreshToken(t, time.Now().Add(-time.Minute).UTC().Unix())

	err := rt.Verify(validTokenValue())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errs.ErrExpiredToken) {
		t.Fatalf("expected expired token error, got %v", err)
	}
}

func TestRefreshTokenVerifySucceeds(t *testing.T) {
	rt := validRefreshToken(t, time.Now().Add(time.Hour).UTC().Unix())

	if err := rt.Verify(validTokenValue()); err != nil {
		t.Fatalf("expected verify to succeed, got %v", err)
	}
}

func validRefreshToken(t *testing.T, expiresAt int64) *RefreshToken {
	t.Helper()

	token, err := NewRefreshToken(mustUUID(t), validTokenValue(), expiresAt)
	if err != nil {
		t.Fatalf("failed to create valid refresh token: %v", err)
	}

	return token
}

func validTokenValue() string {
	return "0123456789abcdef0123456789abcdef"
}

func mustUUID(t *testing.T) uuidv7.UUID {
	t.Helper()

	id, err := uuidv7.New()
	if err != nil {
		t.Fatalf("failed to generate UUID: %v", err)
	}

	return id
}
