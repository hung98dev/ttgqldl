package identity

import (
	"fmt"
	"time"
)

// EphemeralConfig configures a session-scoped synthetic identity scope. Secret and
// clock are injected explicitly: there is no generated default, no global clock and no
// provider-backed continuity claim.
type EphemeralConfig struct {
	SessionID string
	Secret    []byte
	ExpiresAt time.Time
	Now       func() time.Time
}

// Ephemeral derives stable synthetic identities that are valid only inside one
// session. The session id and a caller-injected synthetic secret are both bound into
// the HMAC, so the same individual is not correlatable across sessions even if a test
// accidentally reuses synthetic secret bytes.
type Ephemeral struct {
	sessionID string
	secret    []byte
	expiresAt time.Time
	now       func() time.Time
}

// NewEphemeral validates an explicit synthetic scope. Missing/short/degenerate key
// material, an invalid session, a zero expiry or a nil clock fail closed.
func NewEphemeral(config EphemeralConfig) (*Ephemeral, error) {
	if !validScopedText(config.SessionID, 128, 128) {
		return nil, ErrInvalidSession
	}
	if err := validateSecret(config.Secret); err != nil {
		return nil, err
	}
	if config.ExpiresAt.IsZero() {
		return nil, fmt.Errorf("%w: expiry is required", ErrInvalidEphemeralConf)
	}
	if config.Now == nil {
		return nil, fmt.Errorf("%w: clock is required", ErrInvalidEphemeralConf)
	}
	return &Ephemeral{
		sessionID: config.SessionID,
		secret:    append([]byte(nil), config.Secret...),
		expiresAt: config.ExpiresAt,
		now:       config.Now,
	}, nil
}

// String redacts synthetic key material.
func (e *Ephemeral) String() string {
	if e == nil {
		return "identity.Ephemeral<nil>"
	}
	return fmt.Sprintf("identity.Ephemeral{session=%s, expires_at=%s, secret=<redacted>}", e.sessionID, e.expiresAt.UTC().Format(time.RFC3339))
}

// GoString redacts synthetic key material for %#v formatting.
func (e *Ephemeral) GoString() string { return e.String() }

// SessionID returns the bound synthetic session id.
func (e *Ephemeral) SessionID() string { return e.sessionID }

// ExpiresAt returns the scope expiry.
func (e *Ephemeral) ExpiresAt() time.Time { return e.expiresAt }

// Derive resolves a binding-stable synthetic ActorKey at the configured clock.
func (e *Ephemeral) Derive(viewerID string) (Result, error) {
	if e == nil {
		return Result{}, ErrInvalidEphemeralConf
	}
	return e.DeriveAt(viewerID, e.now())
}

// DeriveAt resolves a synthetic ActorKey at an explicit instant, making expiry
// deterministic in tests without reading a wall clock.
func (e *Ephemeral) DeriveAt(viewerID string, at time.Time) (Result, error) {
	if e == nil || len(e.secret) == 0 {
		return Result{}, ErrInvalidEphemeralConf
	}
	if !validViewerID(viewerID) {
		return Result{}, fmt.Errorf("%w: viewer id", ErrInvalidIdentityInput)
	}
	if !at.Before(e.expiresAt) {
		return Result{}, ErrEphemeralExpired
	}
	actorKey, err := deriveEphemeral(e.sessionID, viewerID, e.secret)
	if err != nil {
		return Result{}, err
	}
	return Result{
		ActorKey:   actorKey,
		KeyVersion: EphemeralKeyVersion,
		Ephemeral:  true,
	}, nil
}
