package identity

import "errors"

// Stable rejection categories. Errors never contain secret material, raw provider
// viewer ids, display aliases or message text.
var (
	ErrInvalidKeyVersion    = errors.New("identity: invalid key version")
	ErrWeakKey              = errors.New("identity: weak or missing key material")
	ErrInvalidKey           = errors.New("identity: invalid key")
	ErrDuplicateKeyVersion  = errors.New("identity: duplicate key version")
	ErrUnknownKeyVersion    = errors.New("identity: unknown key version")
	ErrNoKeyMaterial        = errors.New("identity: no key material")
	ErrInvalidIdentityInput = errors.New("identity: invalid identity input")
	ErrMissingViewerID      = errors.New("identity: stable viewer id is required; LIVE actions must fail closed")
	ErrDerivation           = errors.New("identity: derivation produced an invalid actor key")
	ErrInvalidSession       = errors.New("identity: invalid ephemeral session")
	ErrEphemeralExpired     = errors.New("identity: ephemeral identity scope has expired")
	ErrInvalidEphemeralConf = errors.New("identity: invalid ephemeral identity configuration")
)
