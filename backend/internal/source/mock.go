package source

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"time"
)

// Scenario names a deterministic mock fixture generator.
type Scenario string

const (
	ScenarioSingleRegistration   Scenario = "single_registration"
	ScenarioMultipleParticipants Scenario = "multiple_participants"
	ScenarioRepeatedComments     Scenario = "repeated_comments"
	ScenarioRoleChanges          Scenario = "role_changes"
	ScenarioVotes                Scenario = "votes"
	ScenarioInvalidComments      Scenario = "invalid_comments"
	ScenarioUnicodeVariants      Scenario = "unicode_variants"
	ScenarioDuplicateEventIDs    Scenario = "duplicate_event_ids"
	ScenarioOutOfOrderUpstream   Scenario = "out_of_order_upstream"
	ScenarioNonCommentEvents     Scenario = "non_comment_events"
	ScenarioEmpty                Scenario = "empty"
	ScenarioFiniteScript         Scenario = "finite_script"
)

// MaxMockEvents bounds one mock script so memory stays predictable. The 10,000-user
// integrated population test belongs to W0307.
const MaxMockEvents = 1_000_000

// mockEpoch is a fixed synthetic timestamp base. Deterministic sources never read
// the wall clock, so identical configuration always yields identical events.
var mockEpoch = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// MockConfig is the explicit deterministic configuration of a mock source.
type MockConfig struct {
	Session  Session
	Seed     uint64
	Scenario Scenario
	Count    int
}

// Mock is a deterministic, seeded, network-free source. It never fabricates real
// viewer counts: every identity and event id is explicitly synthetic.
type Mock struct {
	session  Session
	events   []Event
	sourceID string
	lc       lifecycle
}

// NewMock builds a mock source from a scenario and seed.
func NewMock(config MockConfig) (*Mock, error) {
	if config.Session.Mode != ModeMock {
		return nil, fmt.Errorf("%w: mock requires mock mode", ErrInvalidSession)
	}
	events, err := ScenarioEvents(config.Session, config.Seed, config.Scenario, config.Count)
	if err != nil {
		return nil, err
	}
	return newMockWithEvents(config.Session, events)
}

// NewMockWithEvents builds a mock source from an explicit deterministic script.
func NewMockWithEvents(session Session, events []Event) (*Mock, error) {
	if session.Mode != ModeMock {
		return nil, fmt.Errorf("%w: mock requires mock mode", ErrInvalidSession)
	}
	return newMockWithEvents(session, events)
}

func newMockWithEvents(session Session, events []Event) (*Mock, error) {
	if !session.Valid() {
		return nil, ErrInvalidSession
	}
	if len(events) > MaxMockEvents {
		return nil, fmt.Errorf("%w: %d events exceed the mock bound", ErrInvalidEvent, len(events))
	}
	// The adapter owns an immutable deep copy: caller mutation after construction must
	// never change what is delivered under the accepted source identity.
	stored := copyEvents(events)
	for index := range stored {
		if err := ValidateEvent(stored[index], session, ModeMock); err != nil {
			return nil, fmt.Errorf("mock event %d: %w", index, err)
		}
	}
	return &Mock{
		session:  session,
		events:   stored,
		sourceID: mockSourceID(session, stored),
	}, nil
}

// mockScriptIdentityVersion is the source-script identity encoding version. Version
// 1 was separator-based and ambiguous; version 2 is domain-separated and
// length-prefixed. Old cursors carry a v1 digest and therefore fail closed with
// ErrCursorMismatch instead of being silently reinterpreted.
const mockScriptIdentityVersion = 2

// mockSourceID binds cursors to the exact immutable script using an unambiguous,
// versioned encoding: domain prefix, encoding version, session, event count, then for
// every event in order every field (including timestamp presence and value) as a
// length-prefixed or fixed-width value. No field separator exists, so identifiers
// containing punctuation cannot construct an alias script.
func mockSourceID(session Session, events []Event) string {
	hash := sha256.New()
	writeIdentityString(hash, "ttg-source-script")
	writeIdentityUint64(hash, mockScriptIdentityVersion)
	writeIdentityString(hash, session.ID)
	writeIdentityUint64(hash, uint64(len(events)))
	for _, event := range events {
		writeIdentityString(hash, event.Provider)
		writeIdentityString(hash, event.SourceSession)
		writeIdentityString(hash, event.ProviderEventID)
		writeIdentityString(hash, event.ProviderViewerID)
		writeIdentityString(hash, event.Alias)
		writeIdentityString(hash, event.Message)
		writeIdentityUint64(hash, uint64(event.Kind))
		writeIdentityString(hash, event.ReceivedAt.UTC().Format(time.RFC3339Nano))
		if event.UpstreamAt == nil {
			writeIdentityUint64(hash, 0)
			continue
		}
		writeIdentityUint64(hash, 1)
		writeIdentityString(hash, event.UpstreamAt.UTC().Format(time.RFC3339Nano))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func writeIdentityUint64(hash hash.Hash, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = hash.Write(encoded[:])
}

func writeIdentityString(hash hash.Hash, value string) {
	writeIdentityUint64(hash, uint64(len(value)))
	_, _ = hash.Write([]byte(value))
}

// copyEvent deep-copies the only mutable part of an Event: the upstream timestamp
// pointer. Strings are immutable in Go; the slice itself is copied separately.
func copyEvent(event Event) Event {
	if event.UpstreamAt != nil {
		stamp := *event.UpstreamAt
		event.UpstreamAt = &stamp
	}
	return event
}

func copyEvents(events []Event) []Event {
	copied := make([]Event, len(events))
	for index := range events {
		copied[index] = copyEvent(events[index])
	}
	return copied
}

// ScenarioEvents generates the reviewed deterministic mock scripts. Identical
// (session, seed, scenario, count) always yields identical events.
func ScenarioEvents(session Session, seed uint64, scenario Scenario, count int) ([]Event, error) {
	if !session.Valid() || session.Mode != ModeMock {
		return nil, ErrInvalidSession
	}
	if count < 0 || count > MaxMockEvents {
		return nil, fmt.Errorf("%w: event count %d", ErrInvalidEvent, count)
	}
	switch scenario {
	case ScenarioEmpty:
		return []Event{}, nil
	case ScenarioSingleRegistration, ScenarioMultipleParticipants, ScenarioRepeatedComments,
		ScenarioRoleChanges, ScenarioVotes, ScenarioInvalidComments, ScenarioUnicodeVariants,
		ScenarioDuplicateEventIDs, ScenarioOutOfOrderUpstream, ScenarioNonCommentEvents,
		ScenarioFiniteScript:
	default:
		return nil, fmt.Errorf("%w: unknown scenario", ErrInvalidEvent)
	}

	effective := count
	if effective == 0 {
		effective = 3
	}
	if scenario == ScenarioSingleRegistration && effective > 1 {
		effective = 1
	}

	events := make([]Event, 0, effective)
	for index := 0; index < effective; index++ {
		actor := index
		message := "XÂY"
		kind := KindComment
		eventID := fmt.Sprintf("mock-%s-%d-%06d", scenario, seed, index)
		var upstream *time.Time
		switch scenario {
		case ScenarioSingleRegistration:
			message = "XÂY"
		case ScenarioMultipleParticipants, ScenarioFiniteScript:
			message = []string{"XÂY", "THỦ", "ĐÁNH"}[index%3]
		case ScenarioRepeatedComments:
			actor = 0
			message = "XÂY"
		case ScenarioRoleChanges:
			actor = 0
			message = []string{"XÂY", "THỦ"}[index%2]
		case ScenarioVotes:
			message = []string{"A", "B", "C"}[index%3]
		case ScenarioInvalidComments:
			message = []string{"XAY DANH", "XAY!", "hello", "", "ХАУ"}[index%5]
		case ScenarioUnicodeVariants:
			message = []string{"XÂY", "xây", "THỦ", "ĐÁNH", "ＡＴＴＡＣＫ", "Ｂ"}[index%6]
		case ScenarioDuplicateEventIDs:
			eventID = fmt.Sprintf("mock-duplicate-%d", index/2)
			message = []string{"XÂY", "THỦ"}[index%2]
		case ScenarioOutOfOrderUpstream:
			message = []string{"XÂY", "THỦ", "ĐÁNH"}[index%3]
			stamp := mockEpoch.Add(time.Duration(effective-index) * time.Second)
			upstream = &stamp
		case ScenarioNonCommentEvents:
			kind = EventKind(KindGift + EventKind(index%5))
			message = "XÂY"
		}
		jitter := syntheticViewerID(seed, actor)
		events = append(events, Event{
			Provider:         ModeMock.Provider(),
			SourceSession:    session.ID,
			ProviderEventID:  eventID,
			ProviderViewerID: jitter,
			Alias:            fmt.Sprintf("Villager %04d", actor%10000),
			Message:          message,
			Kind:             kind,
			ReceivedAt:       mockEpoch.Add(time.Duration(index) * time.Second),
			UpstreamAt:       upstream,
		})
	}
	return events, nil
}

// syntheticViewerID derives a stable synthetic identity from the seed and actor.
// It is explicitly not an authorized provider identity and is never hashed into a
// production ActorKey here.
func syntheticViewerID(seed uint64, actor int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("ttg-mock-viewer:%d:%d", seed, actor)))
	return "mock-viewer-" + hex.EncodeToString(sum[:12])
}

// SourceID returns the digest-bound adapter identity used by cursors.
func (m *Mock) SourceID() string { return m.sourceID }

// EventCount returns the script length.
func (m *Mock) EventCount() int { return len(m.events) }

// Start delivers the script from the beginning.
func (m *Mock) Start(ctx context.Context, session Session, sink Sink) error {
	return m.run(ctx, session, Cursor{}, sink, false)
}

// Stop cancels delivery and waits for the emission loop to exit.
func (m *Mock) Stop(ctx context.Context) error { return m.lc.stop(ctx) }

// Health returns the concurrency-safe adapter health.
func (m *Mock) Health(context.Context) Health { return m.lc.health() }

// Capabilities returns the truthful mock capability set.
func (m *Mock) Capabilities(context.Context) Capabilities {
	return Capabilities{
		StableViewerID:       true,
		StableEventID:        true,
		ReplayCursor:         true,
		DeliveryOrder:        true,
		TimestampReliability: TimestampSyntheticDeterministic,
		DisplayName:          true,
		ModerationEvents:     false,
	}
}

// Reconnect resumes the exact remaining script from a cursor bound to this source.
func (m *Mock) Reconnect(ctx context.Context, cursor Cursor, sink Sink) error {
	return m.run(ctx, m.session, cursor, sink, true)
}

func (m *Mock) run(ctx context.Context, session Session, cursor Cursor, sink Sink, reconnect bool) error {
	if session != m.session {
		return ErrSessionMismatch
	}
	start := uint64(0)
	if reconnect {
		if err := validateCursorBinding(cursor, session, m.sourceID); err != nil {
			return err
		}
		start = cursor.NextEvent
		if start > uint64(len(m.events)) {
			return fmt.Errorf("%w: position %d beyond %d events", ErrInvalidCursor, start, len(m.events))
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := m.lc.begin(runCtx, cancel, session, m.sourceID, start, reconnect); err != nil {
		return err
	}
	status, runErr := StatusEOF, error(nil)
	for index := start; index < uint64(len(m.events)); index++ {
		if m.lc.stopping() {
			status, runErr = StatusStopped, nil
			break
		}
		if err := runCtx.Err(); err != nil {
			status, runErr = StatusDisconnected, err
			break
		}
		event := copyEvent(m.events[index])
		if err := ValidateEvent(event, session, ModeMock); err != nil {
			status, runErr = StatusFailed, err
			break
		}
		if err := m.lc.deliver(runCtx, sink, event, index+1); err != nil {
			switch {
			case errors.Is(err, ErrSourceStopped):
				status, runErr = StatusStopped, nil
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				status, runErr = StatusDisconnected, err
			case errors.Is(err, ErrDeliveryRejected):
				// A rejected delivery is a retryable interruption, not a permanent
				// source failure: the cursor stays before the rejected event.
				status, runErr = StatusDisconnected, err
			default:
				status, runErr = StatusFailed, err
			}
			break
		}
	}
	m.lc.finish(status, runErr)
	return runErr
}
