package ingest

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

// scopeBufferBytes bounds the stack encoding buffer. The validated ingress limits are:
// session <=128 bytes, provider <=64 bytes, event id <=1024 bytes (256 runes), actor 64,
// domain/prefix <32 and fixed 8-byte length prefixes; 4096 is comfortably above the
// maximum.
const scopeBufferBytes = 4096

// stableScope hashes the durable uniqueness scope (session, provider, event id) with a
// domain-separated length-prefixed encoding. No ambiguous separator is used and no raw
// provider id is retained in the cache.
func stableScope(session, provider, eventID string) [32]byte {
	var buffer [scopeBufferBytes]byte
	offset := 0
	offset = appendScopeString(buffer[:], offset, "ttg-ingest-stable-v1")
	offset = appendScopeString(buffer[:], offset, session)
	offset = appendScopeString(buffer[:], offset, provider)
	offset = appendScopeString(buffer[:], offset, eventID)
	return sha256.Sum256(buffer[:offset])
}

// windowScope hashes the best-effort scope (session, scoped actor key, event kind,
// canonical action). Alias, comment text, provider timestamps and per-message randomness
// are deliberately absent.
func windowScope(session string, actor domain.ActorKey, kind source.EventKind, action commands.Action) [32]byte {
	var buffer [scopeBufferBytes]byte
	offset := 0
	offset = appendScopeString(buffer[:], offset, "ttg-ingest-window-v1")
	offset = appendScopeString(buffer[:], offset, session)
	offset = appendScopeString(buffer[:], offset, string(actor))
	offset = appendScopeUint64(buffer[:], offset, uint64(kind))
	actionKind, actionValue := actionTuple(action)
	offset = appendScopeUint64(buffer[:], offset, uint64(actionKind))
	offset = appendScopeUint64(buffer[:], offset, uint64(actionValue))
	return sha256.Sum256(buffer[:offset])
}

// payloadFingerprint is a compact comparison digest of the verified identity and typed
// action. It distinguishes conflicting reuse of one provider event id without retaining a
// raw comment, alias or provider viewer id.
func payloadFingerprint(actor domain.ActorKey, kind source.EventKind, action commands.Action) [32]byte {
	var buffer [scopeBufferBytes]byte
	offset := 0
	offset = appendScopeString(buffer[:], offset, "ttg-ingest-fingerprint-v1")
	offset = appendScopeString(buffer[:], offset, string(actor))
	offset = appendScopeUint64(buffer[:], offset, uint64(kind))
	actionKind, actionValue := actionTuple(action)
	offset = appendScopeUint64(buffer[:], offset, uint64(actionKind))
	offset = appendScopeUint64(buffer[:], offset, uint64(actionValue))
	return sha256.Sum256(buffer[:offset])
}

// actionTuple returns a stable numeric action code. Role and ballot families cannot
// collide, and distinct roles/ballots stay distinct.
func actionTuple(action commands.Action) (uint8, uint8) {
	switch action.Kind {
	case commands.KindRole:
		// #nosec G115 -- callers validate action.Valid(); role values are 1..3 and the
		// numeric code is only a discriminator, never a signed quantity.
		return 1, uint8(action.Role)
	case commands.KindBallot:
		// #nosec G115 -- callers validate action.Valid(); ballot values are 1..3.
		return 2, uint8(action.Ballot)
	default:
		return 0, 0
	}
}

func appendScopeString(destination []byte, offset int, value string) int {
	binary.BigEndian.PutUint64(destination[offset:offset+8], uint64(len(value)))
	offset += 8
	copy(destination[offset:], value)
	return offset + len(value)
}

func appendScopeUint64(destination []byte, offset int, value uint64) int {
	binary.BigEndian.PutUint64(destination[offset:offset+8], value)
	return offset + 8
}
