package identity

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

const (
	// EncodingVersion identifies the length-prefixed identity tuple encoding.
	EncodingVersion = 1
	// MinSecretBytes is the minimum accepted HMAC secret length (256-bit key).
	MinSecretBytes = 32
	// MaxSecretBytes bounds accepted secret material so an accidental file or
	// certificate cannot be loaded as a key.
	MaxSecretBytes = 4096
	// MaxKeyVersionBytes matches the database key_version CHECK constraint.
	MaxKeyVersionBytes = 32
	// MaxProviderBytes bounds the provider discriminator.
	MaxProviderBytes = 32
	// MaxProviderVersionBytes bounds the provider identity version.
	MaxProviderVersionBytes = 32
	// MaxBroadcasterBytes bounds the broadcaster/village scope.
	MaxBroadcasterBytes = 128
	// MaxViewerIDRunes bounds a stable provider viewer id (source contract 256).
	MaxViewerIDRunes = 256
	// MaxViewerIDBytes bounds the encoded viewer id.
	MaxViewerIDBytes = 1024
)

// KeyVersion names one HMAC secret generation. The shape matches the
// actor_identity_aliases.key_version CHECK constraint so every derived result can be
// persisted without re-encoding the label.
type KeyVersion string

// ParseKeyVersion validates a key version label. Labels are separated from secret
// material; they are not secret and may appear in database rows and diagnostics.
func ParseKeyVersion(raw string) (KeyVersion, error) {
	if raw == "" || len(raw) > MaxKeyVersionBytes {
		return "", fmt.Errorf("%w: length", ErrInvalidKeyVersion)
	}
	for index := 0; index < len(raw); index++ {
		character := raw[index]
		valid := (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '.' || character == '_' || character == ':' || character == '-'
		if !valid {
			return "", fmt.Errorf("%w: shape", ErrInvalidKeyVersion)
		}
	}
	return KeyVersion(raw), nil
}

// Valid reports whether the version can be stored in the database key_version column.
func (v KeyVersion) Valid() bool {
	_, err := ParseKeyVersion(string(v))
	return err == nil
}

// String returns the non-secret version label.
func (v KeyVersion) String() string { return string(v) }

// Input is the complete identity tuple. Display alias, comment text, timestamps,
// event sequence, gift counts, client IP, fingerprints and per-message randomness are
// deliberately absent: none may affect ActorKey derivation.
type Input struct {
	Provider        string
	ProviderVersion string
	BroadcasterKey  string
	ViewerID        string
}

// Validate applies the identity-input boundary. Values are used exactly as supplied
// (no NFC normalization, trimming or case folding), so distinct provider identifiers
// cannot silently collapse into one ActorKey.
func (i Input) Validate() error {
	if !validProvider(i.Provider) {
		return fmt.Errorf("%w: provider", ErrInvalidIdentityInput)
	}
	if !validProviderVersion(i.ProviderVersion) {
		return fmt.Errorf("%w: provider version", ErrInvalidIdentityInput)
	}
	if !validScopedText(i.BroadcasterKey, MaxBroadcasterBytes, MaxBroadcasterBytes) {
		return fmt.Errorf("%w: broadcaster scope", ErrInvalidIdentityInput)
	}
	if i.ViewerID == "" || strings.TrimSpace(i.ViewerID) == "" {
		return ErrMissingViewerID
	}
	if !validViewerID(i.ViewerID) {
		return fmt.Errorf("%w: viewer id", ErrInvalidIdentityInput)
	}
	return nil
}

func validViewerID(value string) bool {
	return value != "" && strings.TrimSpace(value) != "" &&
		validScopedText(value, MaxViewerIDBytes, MaxViewerIDRunes)
}

func validProvider(value string) bool {
	if value == "" || len(value) > MaxProviderBytes {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		valid := (character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '_' || character == '-'
		if !valid {
			return false
		}
	}
	return true
}

func validProviderVersion(value string) bool {
	if value == "" || len(value) > MaxProviderVersionBytes {
		return false
	}
	for index := 0; index < len(value); index++ {
		character := value[index]
		valid := (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '.' || character == '_' || character == ':' || character == '-'
		if !valid {
			return false
		}
	}
	return true
}

// validScopedText checks UTF-8 validity, byte/character bounds and rejects controls,
// format characters (bidi overrides, zero-width marks) and whitespace-only values.
// It intentionally does not normalize Unicode.
func validScopedText(value string, maxBytes, maxRunes int) bool {
	if value == "" || len(value) > maxBytes || !utf8.ValidString(value) {
		return false
	}
	runes := utf8.RuneCountInString(value)
	if runes == 0 || runes > maxRunes {
		return false
	}
	allSpace := true
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
		if !unicode.IsSpace(r) {
			allSpace = false
		}
	}
	return !allSpace
}

// Result is one scoped pseudonymous identity. ActorKey is private; it must never be
// written to the public stream or a user-visible log. Ephemeral marks a
// session-scoped synthetic identity that is not provider-backed and must not be
// persisted as an actor_identity_aliases row.
type Result struct {
	ActorKey   domain.ActorKey
	KeyVersion KeyVersion
	Ephemeral  bool
}

// String redacts the private ActorKey from casual formatting.
func (r Result) String() string {
	return "identity.Result{actor_key=<redacted>, key_version=" + r.KeyVersion.String() + ", ephemeral=" + boolString(r.Ephemeral) + "}"
}

// GoString redacts the private ActorKey for %#v formatting.
func (r Result) GoString() string { return r.String() }

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
