package source

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// CursorVersion is the stable cursor format version.
const CursorVersion = 1

// cursorMagic prefixes every encoded cursor and distinguishes it from provider
// identifiers or wall-clock values.
const cursorMagic = "ttgc1"

// Cursor identifies the next logical event position without ambiguity. Session and
// SourceID bind it to one adapter identity, so a cursor from another session or
// fixture cannot silently select a position elsewhere.
type Cursor struct {
	Version   int
	Session   string
	SourceID  string
	NextEvent uint64
}

// NewCursor builds the initial cursor for a session and adapter identity.
func NewCursor(session, sourceID string, next uint64) Cursor {
	return Cursor{Version: CursorVersion, Session: session, SourceID: sourceID, NextEvent: next}
}

// Valid reports whether the cursor has a recognized version and bound identity.
func (c Cursor) Valid() bool {
	return c.Version == CursorVersion && c.Session != "" && c.SourceID != ""
}

// Encode returns the deterministic textual form. Components are base64url-encoded so
// no separator or control character can be smuggled into the cursor.
func (c Cursor) Encode() string {
	encode := base64.RawURLEncoding.EncodeToString
	return fmt.Sprintf("%s.%s.%s.%d", cursorMagic, encode([]byte(c.Session)), encode([]byte(c.SourceID)), c.NextEvent)
}

// ParseCursor parses the exact cursor format. Unknown versions, malformed base64,
// trailing components and non-numeric positions are rejected; no provider or
// wall-clock value is ever accepted as a cursor.
func ParseCursor(raw string) (Cursor, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 4 || parts[0] != cursorMagic {
		return Cursor{}, fmt.Errorf("%w: unrecognized format or version", ErrInvalidCursor)
	}
	session, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(session) == 0 {
		return Cursor{}, fmt.Errorf("%w: malformed session component", ErrInvalidCursor)
	}
	sourceID, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sourceID) == 0 {
		return Cursor{}, fmt.Errorf("%w: malformed source component", ErrInvalidCursor)
	}
	next, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: malformed position", ErrInvalidCursor)
	}
	return Cursor{Version: CursorVersion, Session: string(session), SourceID: string(sourceID), NextEvent: next}, nil
}

// validateCursorBinding checks that a cursor was issued for this session and adapter
// identity. It never trusts a numeric position alone.
func validateCursorBinding(cursor Cursor, session Session, sourceID string) error {
	if !cursor.Valid() {
		return fmt.Errorf("%w: missing version or binding", ErrInvalidCursor)
	}
	if cursor.Session != session.ID || cursor.SourceID != sourceID {
		return ErrCursorMismatch
	}
	return nil
}
