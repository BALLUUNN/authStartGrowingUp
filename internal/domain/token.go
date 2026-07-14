package domain

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	errs "github.com/BALLUUNN/authStartGrowingUp/pkg/errs"
	"github.com/dombox/uuidv7"
)

const (
	refreshTokenClockSkew = 5 * time.Second
	refreshTokenMinLength = 32
)

// RefreshToken represents a refresh token entity used for obtaining new access tokens.
// It contains the token string, associated user, and expiration timestamp.
type RefreshToken struct {
	ID        uuidv7.UUID // Unique identifier for the refresh token record
	UserID    uuidv7.UUID // ID of the user who owns this token
	Token     string      // The actual refresh token string (opaque or JWT)
	ExpiresAt int64       // Unix timestamp (seconds) when the token expires
}

// NewRefreshToken creates a new RefreshToken instance with a generated UUID.
// It validates all fields and returns an error if any field is invalid.
// Returns InternalError for system-level issues (UUID generation or validation failures).
func NewRefreshToken(userID uuidv7.UUID, token string, expiresAt int64) (*RefreshToken, error) {
	uuid, err := uuidv7.New()
	if err != nil {
		return nil, errs.NewInternalError(err)
	}

	rt := &RefreshToken{
		ID:        uuid,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}

	// Validate immediately after construction to ensure object consistency
	if err := rt.Validate(); err != nil {
		return nil, err
	}

	return rt, nil
}

// isExpired checks whether the token has expired, with a 5-second leeway.
// The leeway accounts for clock skew between servers and network latency.
// Returns true if current time is after (expiry - 5 seconds).
func (r *RefreshToken) isExpired() bool {
	return r.IsExpiredAt(time.Now().UTC())
}

// String returns a safe string representation that does not leak the token body.
func (r *RefreshToken) String() string {
	if r == nil {
		return "RefreshToken<nil>"
	}

	return fmt.Sprintf(
		"RefreshToken{ID: %s, UserID: %s, ExpiresAt: %d}",
		r.ID.String(),
		r.UserID.String(),
		r.ExpiresAt,
	)
}

// IsExpiredAt reports whether the token should be treated as expired at a given time.
func (r *RefreshToken) IsExpiredAt(now time.Time) bool {
	expiry := time.Unix(r.ExpiresAt, 0).UTC()
	return now.UTC().After(expiry.Add(-refreshTokenClockSkew))
}

// ExpiresAtTime returns the expiration instant in UTC.
func (r *RefreshToken) ExpiresAtTime() time.Time {
	if r == nil {
		return time.Unix(0, 0).UTC()
	}

	return time.Unix(r.ExpiresAt, 0).UTC()
}

// Validate performs internal consistency checks on the token fields.
// All errors are InternalError because they indicate data corruption or
// programming errors, not client-side issues.
func (r *RefreshToken) Validate() error {
	if r == nil {
		return errs.NewInternalError(errors.New("refresh token is nil"))
	}
	if r.ID == uuidv7.Nil() {
		return errs.NewInternalError(errors.New("invalid refresh token ID")).WithFieldValue("id", r.ID)
	}
	if r.UserID == uuidv7.Nil() {
		return errs.NewInternalError(errors.New("invalid refresh token user ID")).WithFieldValue("user_id", r.UserID)
	}
	if r.Token == "" {
		return errs.NewInternalError(errors.New("refresh token cannot be empty")).WithFieldValue("token", "[redacted]")
	}
	if strings.TrimSpace(r.Token) != r.Token {
		return errs.NewInternalError(errors.New("refresh token cannot contain leading or trailing spaces")).
			WithFieldValue("token", "[redacted]")
	}
	if len(r.Token) < refreshTokenMinLength {
		return errs.NewInternalError(errors.New("refresh token is too short")).
			WithFieldValue("token", "[redacted]")
	}
	if r.ExpiresAt <= 0 {
		return errs.NewInternalError(errors.New("invalid expiration time")).WithFieldValue("expires_at", r.ExpiresAt)
	}
	return nil
}

// Check performs a complete validation of the refresh token.
// 1. Validates internal consistency via validate()
// 2. Checks expiration via isExpired()
//
// Returns:
//   - nil if the token is valid
//   - InternalError if validation fails (data corruption)
//   - ExpiredTokenError if the token has expired (client should re-authenticate)
func (r *RefreshToken) Check() error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.isExpired() {
		return errs.NewExpiredTokenError("your session has expired", nil).
			WithFieldValue("expires_at", r.ExpiresAt)
	}
	return nil
}

// Verify checks that the candidate token matches the stored token and is still usable.
func (r *RefreshToken) Verify(candidate string) error {
	if err := r.Check(); err != nil {
		return err
	}
	if strings.TrimSpace(candidate) == "" {
		return errs.NewInvalidTokenError("invalid refresh token", nil).
			WithFieldValue("token", "[redacted]")
	}
	if subtle.ConstantTimeCompare([]byte(r.Token), []byte(candidate)) != 1 {
		return errs.NewInvalidTokenError("invalid refresh token", nil).
			WithFieldValue("token", "[redacted]")
	}
	return nil
}
