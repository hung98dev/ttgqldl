package identity

import (
	"fmt"
	"sort"
)

// Key is one versioned HMAC secret. It owns its secret bytes and never exposes them;
// String and GoString are redacted so accidental logging cannot disclose key material.
type Key struct {
	version KeyVersion
	secret  []byte
}

// NewKey validates and copies key material. Secrets shorter than 256 bits, oversized
// values and degenerate all-identical keys are rejected. There is no default,
// generated or persisted production key.
func NewKey(version string, secret []byte) (Key, error) {
	parsed, err := ParseKeyVersion(version)
	if err != nil {
		return Key{}, err
	}
	if err := validateSecret(secret); err != nil {
		return Key{}, err
	}
	copied := append([]byte(nil), secret...)
	return Key{version: parsed, secret: copied}, nil
}

func validateSecret(secret []byte) error {
	switch {
	case len(secret) < MinSecretBytes:
		return fmt.Errorf("%w: secret is shorter than %d bytes", ErrWeakKey, MinSecretBytes)
	case len(secret) > MaxSecretBytes:
		return fmt.Errorf("%w: secret exceeds %d bytes", ErrInvalidKey, MaxSecretBytes)
	case degenerateSecret(secret):
		return fmt.Errorf("%w: secret is degenerate", ErrWeakKey)
	default:
		return nil
	}
}

func degenerateSecret(secret []byte) bool {
	if len(secret) == 0 {
		return true
	}
	first := secret[0]
	for _, b := range secret[1:] {
		if b != first {
			return false
		}
	}
	return true
}

// Version returns the non-secret key version label.
func (k Key) Version() KeyVersion { return k.version }

// String redacts the secret.
func (k Key) String() string {
	return "identity.Key{version=" + k.version.String() + ", secret=<redacted>}"
}

// GoString redacts the secret for %#v formatting.
func (k Key) GoString() string { return k.String() }

// KeySet is an immutable, concurrency-safe set of one active key and zero or more
// retained keys used only for candidate resolution during rotation. Derivation never
// mutates the set and never consults global state.
type KeySet struct {
	active KeyVersion
	keys   map[KeyVersion][]byte
	order  []KeyVersion
}

// NewKeySet builds a key set. Retained versions are sorted deterministically by label
// so candidate ordering never depends on input order. Duplicate labels are rejected.
func NewKeySet(active Key, retained []Key) (*KeySet, error) {
	if len(active.secret) == 0 || !active.version.Valid() {
		return nil, ErrNoKeyMaterial
	}
	keys := make(map[KeyVersion][]byte, len(retained)+1)
	keys[active.version] = append([]byte(nil), active.secret...)
	retainedVersions := make([]KeyVersion, 0, len(retained))
	for index, key := range retained {
		if len(key.secret) == 0 || !key.version.Valid() {
			return nil, fmt.Errorf("%w: retained key %d", ErrInvalidKey, index)
		}
		if _, exists := keys[key.version]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateKeyVersion, key.version)
		}
		keys[key.version] = append([]byte(nil), key.secret...)
		retainedVersions = append(retainedVersions, key.version)
	}
	sort.Slice(retainedVersions, func(i, j int) bool {
		return retainedVersions[i] < retainedVersions[j]
	})
	order := make([]KeyVersion, 0, len(retainedVersions)+1)
	order = append(order, active.version)
	order = append(order, retainedVersions...)
	return &KeySet{active: active.version, keys: keys, order: order}, nil
}

// String lists versions without key material.
func (s *KeySet) String() string {
	if s == nil {
		return "identity.KeySet<nil>"
	}
	return fmt.Sprintf("identity.KeySet{active=%s, versions=%v, secrets=<redacted>}", s.active, s.order)
}

// GoString redacts secrets for %#v formatting.
func (s *KeySet) GoString() string { return s.String() }

// ActiveVersion returns the version used for normal derivation.
func (s *KeySet) ActiveVersion() KeyVersion {
	if s == nil {
		return ""
	}
	return s.active
}

// Versions returns the deterministic candidate order: active first, then retained
// versions sorted by label.
func (s *KeySet) Versions() []KeyVersion {
	if s == nil {
		return nil
	}
	copied := make([]KeyVersion, len(s.order))
	copy(copied, s.order)
	return copied
}

// Derive resolves the active-version ActorKey. This is the only path that may assign a
// production identity for a known stable viewer id; callers without a lawful stable
// viewer id must fail closed rather than substituting an ephemeral identity.
func (s *KeySet) Derive(input Input) (Result, error) {
	if s == nil || len(s.keys) == 0 {
		return Result{}, ErrNoKeyMaterial
	}
	if err := input.Validate(); err != nil {
		return Result{}, err
	}
	return deriveResult(s.active, s.keys[s.active], input)
}

// DeriveWithVersion resolves one explicitly named retained or active version.
func (s *KeySet) DeriveWithVersion(input Input, version KeyVersion) (Result, error) {
	if s == nil || len(s.keys) == 0 {
		return Result{}, ErrNoKeyMaterial
	}
	if err := input.Validate(); err != nil {
		return Result{}, err
	}
	secret, exists := s.keys[version]
	if !exists {
		return Result{}, fmt.Errorf("%w: %s", ErrUnknownKeyVersion, version)
	}
	return deriveResult(version, secret, input)
}

// Candidates derives the same viewer under every permitted key version, active first
// then retained labels in sorted order. The transient raw viewer id is required for
// every candidate: an old digest cannot be reversed into one when its secret changes.
func (s *KeySet) Candidates(input Input) ([]Result, error) {
	if s == nil || len(s.keys) == 0 {
		return nil, ErrNoKeyMaterial
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(s.order))
	for _, version := range s.order {
		result, err := deriveResult(version, s.keys[version], input)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}
