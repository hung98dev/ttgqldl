package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// Domain separators make an ActorKey from one derivation purpose useless as a key for
// another purpose, even if the same secret bytes were ever reused by mistake.
const (
	actorKeyDomain  = "ttg-actor-key"
	ephemeralDomain = "ttg-ephemeral-actor-key"
	// EphemeralKeyVersion labels synthetic session-scoped identities. It is not a
	// provider key version and must never be written to actor_identity_aliases.
	EphemeralKeyVersion KeyVersion = "ephemeral-v1"
)

// deriveResult computes the lowercase-hex HMAC-SHA256 ActorKey for one identity tuple:
//
//	HMAC-SHA256(secret,
//	    lp(domain) || u64be(EncodingVersion) || lp(key_version) ||
//	    lp(field_1) || ... || lp(field_n))
//
// where lp(s) = u64be(len(s)) || s. Field order is fixed, every variable-length field
// is length-prefixed, and the encoding version and key version are bound into the
// message. The result is validated against the domain ActorKey contract; a derivation
// that somehow produced a malformed key fails closed.
func deriveResult(version KeyVersion, secret []byte, input Input) (Result, error) {
	key, err := deriveActorKey(actorKeyDomain, version, secret,
		input.Provider, input.ProviderVersion, input.BroadcasterKey, input.ViewerID)
	if err != nil {
		return Result{}, err
	}
	return Result{ActorKey: key, KeyVersion: version}, nil
}

func deriveEphemeral(sessionID string, viewerID string, secret []byte) (domain.ActorKey, error) {
	return deriveActorKey(ephemeralDomain, EphemeralKeyVersion, secret, sessionID, viewerID)
}

func deriveActorKey(domainName string, version KeyVersion, secret []byte, fields ...string) (domain.ActorKey, error) {
	mac := hmac.New(sha256.New, secret)
	writeLPString(mac, domainName)
	writeUint64(mac, EncodingVersion)
	writeLPString(mac, version.String())
	for _, field := range fields {
		writeLPString(mac, field)
	}
	sum := mac.Sum(nil)
	actorKey := domain.ActorKey(hex.EncodeToString(sum))
	if !actorKey.Valid() {
		return "", ErrDerivation
	}
	return actorKey, nil
}

func writeUint64(hash hash.Hash, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = hash.Write(encoded[:])
}

func writeLPString(hash hash.Hash, value string) {
	writeUint64(hash, uint64(len(value)))
	_, _ = hash.Write([]byte(value))
}
