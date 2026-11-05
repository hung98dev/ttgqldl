package source

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"
)

// Mode is the source mode bound to one session. authorized_live is a recognized
// mode value but no adapter implements it: GATE-LIVE remains blocked and the
// configuration gate rejects it.
type Mode uint8

const (
	ModeMock           Mode = 1
	ModeReplay         Mode = 2
	ModeAuthorizedLive Mode = 3
)

// Valid reports whether the mode is a defined value.
func (m Mode) Valid() bool { return m >= ModeMock && m <= ModeAuthorizedLive }

// String returns the stable mode name used in session configuration.
func (m Mode) String() string {
	switch m {
	case ModeMock:
		return "mock"
	case ModeReplay:
		return "replay"
	case ModeAuthorizedLive:
		return "authorized_live"
	default:
		return "invalid"
	}
}

// Provider returns the provider label carried by events of this mode.
func (m Mode) Provider() string { return m.String() }

// Session binds a source adapter to one logical source session.
type Session struct {
	ID   string
	Mode Mode
}

// Valid reports whether the session binding is well formed. The session id is a
// synthetic or provider-scoped identifier, never a raw provider viewer id; it is
// bounded to keep metadata processing cheap.
func (s Session) Valid() bool {
	if !s.Mode.Valid() {
		return false
	}
	if s.ID == "" || len(s.ID) > 128 || !utf8.ValidString(s.ID) {
		return false
	}
	for _, r := range s.ID {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}

// EventKind is the kind of a source event. Only KindComment may ever become a game
// command; every other kind is a valid, observable source event that must never
// grant a role, vote, XP or combat advantage.
type EventKind uint8

const (
	KindComment EventKind = 1
	KindGift    EventKind = 2
	KindLike    EventKind = 3
	KindShare   EventKind = 4
	KindFollow  EventKind = 5
	KindJoin    EventKind = 6
)

// Valid reports whether the kind is defined.
func (k EventKind) Valid() bool { return k >= KindComment && k <= KindJoin }

// String returns the stable event-kind name.
func (k EventKind) String() string {
	switch k {
	case KindComment:
		return "comment"
	case KindGift:
		return "gift"
	case KindLike:
		return "like"
	case KindShare:
		return "share"
	case KindFollow:
		return "follow"
	case KindJoin:
		return "join"
	default:
		return "invalid"
	}
}

// ParseEventKind maps the stable name used by replay fixtures to an EventKind.
func ParseEventKind(name string) (EventKind, error) {
	switch name {
	case "comment":
		return KindComment, nil
	case "gift":
		return KindGift, nil
	case "like":
		return KindLike, nil
	case "share":
		return KindShare, nil
	case "follow":
		return KindFollow, nil
	case "join":
		return KindJoin, nil
	default:
		return 0, fmt.Errorf("%w: unknown event kind", ErrInvalidEvent)
	}
}

// Event is one validated source event. Provider identifiers are untrusted input and
// never appear in public output; they exist only for later identity/durability work
// packages.
type Event struct {
	Provider         string
	SourceSession    string
	ProviderEventID  string
	ProviderViewerID string
	Alias            string
	Message          string
	Kind             EventKind
	ReceivedAt       time.Time
	UpstreamAt       *time.Time
}

// TimestampReliability describes how much a source's timestamps can be trusted.
type TimestampReliability uint8

const (
	TimestampUnknown TimestampReliability = iota
	TimestampSyntheticDeterministic
	TimestampRecorded
	TimestampUnreliable
)

// String returns a stable label.
func (t TimestampReliability) String() string {
	switch t {
	case TimestampSyntheticDeterministic:
		return "synthetic_deterministic"
	case TimestampRecorded:
		return "recorded"
	case TimestampUnreliable:
		return "unreliable"
	default:
		return "unknown"
	}
}

// RateLimits reports declared maximum rates; zero means unspecified.
type RateLimits struct {
	EventsPerSecond int
	Burst           int
}

// Capabilities is the truthful capability declaration of one adapter. Synthetic
// stable identifiers are declared as stable, never as authorized provider
// identities.
type Capabilities struct {
	StableViewerID       bool
	StableEventID        bool
	ReplayCursor         bool
	DeliveryOrder        bool
	TimestampReliability TimestampReliability
	MaximumRates         RateLimits
	DisplayName          bool
	ModerationEvents     bool
}

// Status is the adapter lifecycle status, separate from the game lifecycle.
type Status uint8

const (
	StatusNew Status = iota
	StatusRunning
	StatusStopped
	StatusEOF
	StatusDisconnected
	StatusFailed
)

// String returns a stable status label.
func (s Status) String() string {
	switch s {
	case StatusNew:
		return "new"
	case StatusRunning:
		return "running"
	case StatusStopped:
		return "stopped"
	case StatusEOF:
		return "eof"
	case StatusDisconnected:
		return "disconnected"
	case StatusFailed:
		return "failed"
	default:
		return "invalid"
	}
}

// Health is a concurrency-safe snapshot. LastError is a stable category, never raw
// input, identifiers or error text.
type Health struct {
	Status    Status
	Delivered uint64
	Cursor    Cursor
	LastError string
}

// Sink delivers one source event. It returns a non-nil error to signal that the
// event was not accepted (backpressure or a processing failure); the adapter then
// stops without advancing its cursor. The context is cancelled by Stop or caller
// cancellation so a blocked sink can abort promptly. A source never claims durable
// exactly-once ingestion: durable acknowledgement belongs to W0305.
type Sink func(ctx context.Context, event Event) error

// Source is the binding adapter contract: one active adapter per session, explicit
// Start/Stop lifecycle, truthful Health and Capabilities, and deterministic resume
// through a versioned cursor. Start and Reconnect deliver events synchronously on
// the caller's goroutine until EOF, cancellation, Stop or a delivery failure.
type Source interface {
	Start(ctx context.Context, session Session, sink Sink) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) Health
	Capabilities(ctx context.Context) Capabilities
	Reconnect(ctx context.Context, cursor Cursor, sink Sink) error
}

// Stable rejection categories. Errors never contain raw comments, aliases or
// provider identifiers.
var (
	ErrInvalidSession    = errors.New("source: invalid session")
	ErrModeUnsupported   = errors.New("source: source mode is not implemented")
	ErrInvalidEvent      = errors.New("source: invalid event")
	ErrInvalidMessage    = errors.New("source: invalid message")
	ErrInvalidAlias      = errors.New("source: invalid alias")
	ErrInvalidMetadata   = errors.New("source: invalid event metadata")
	ErrSessionMismatch   = errors.New("source: event session does not match the adapter session")
	ErrProviderMismatch  = errors.New("source: event provider does not match the adapter mode")
	ErrEventKindInvalid  = errors.New("source: unknown event kind")
	ErrInvalidCursor     = errors.New("source: invalid cursor")
	ErrCursorMismatch    = errors.New("source: cursor belongs to another session or source")
	ErrAlreadyStarted    = errors.New("source: adapter was already started")
	ErrNotReconnectable  = errors.New("source: adapter cannot reconnect in its current status")
	ErrDeliveryRejected  = errors.New("source: delivery callback rejected the event")
	ErrSourceStopped     = errors.New("source: adapter was stopped")
	ErrInvalidFixture    = errors.New("source: invalid replay fixture")
	ErrFixtureDigest     = errors.New("source: replay fixture digest mismatch")
	ErrMalformedRecord   = errors.New("source: malformed replay record")
	ErrSchemaUnsupported = errors.New("source: unsupported replay schema version")
	ErrRecordTooLong     = errors.New("source: replay record exceeds the size limit")
	ErrRecordEncoding    = errors.New("source: replay record is not valid UTF-8")
	ErrSnapshotIO        = errors.New("source: verified snapshot I/O failure")
)
