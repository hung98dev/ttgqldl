package identity_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/identity"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

const syntheticMaterialHex = "8f1e2d3c4b5a69788796a5b4c3d2e1f00112233445566778899aabbccddeeff0"

type observation struct {
	ActorKey   domain.ActorKey
	KeyVersion identity.KeyVersion
	Ephemeral  bool
	Kind       commands.Kind
	Role       domain.Role
	Ballot     domain.VoteOption
	EventID    string
}

func syntheticKeys(t *testing.T) *identity.KeySet {
	t.Helper()
	secret, err := hex.DecodeString(syntheticMaterialHex)
	if err != nil {
		t.Fatal(err)
	}
	key, err := identity.NewKey("v1", secret)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	return keys
}

func mockSession() source.Session {
	return source.Session{ID: "synthetic-session-001", Mode: source.ModeMock}
}

func replaySession() source.Session {
	return source.Session{ID: "synthetic-session-001", Mode: source.ModeReplay}
}

func mockAdapter(t *testing.T, scenario source.Scenario, count int, seed uint64) *source.Mock {
	t.Helper()
	mock, err := source.NewMock(source.MockConfig{Session: mockSession(), Seed: seed, Scenario: scenario, Count: count})
	if err != nil {
		t.Fatal(err)
	}
	return mock
}

func replayAdapter(t *testing.T, fixture string) *source.Replay {
	t.Helper()
	replay, err := source.NewReplay(source.ReplayConfig{
		Session:   replaySession(),
		Directory: filepath.Join("..", "source", "testdata"),
		Fixture:   fixture,
	})
	if err != nil {
		t.Fatal(err)
	}
	return replay
}

func resolveProvider(keys *identity.KeySet, broadcaster string) func(source.Event) (identity.Result, error) {
	return func(event source.Event) (identity.Result, error) {
		return keys.Derive(identity.Input{
			Provider:        event.Provider,
			ProviderVersion: "v1",
			BroadcasterKey:  broadcaster,
			ViewerID:        event.ProviderViewerID,
		})
	}
}

func runPipeline(t *testing.T, adapter source.Source, session source.Session, resolve func(source.Event) (identity.Result, error)) ([]observation, []error, []error) {
	t.Helper()
	var observations []observation
	var identityErrors []error
	var parseErrors []error
	if err := adapter.Start(context.Background(), session, func(_ context.Context, event source.Event) error {
		if event.Kind != source.KindComment {
			return nil
		}
		resolved, err := resolve(event)
		if err != nil {
			identityErrors = append(identityErrors, err)
			return nil
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			parseErrors = append(parseErrors, err)
			return nil
		}
		observations = append(observations, observation{
			ActorKey:   resolved.ActorKey,
			KeyVersion: resolved.KeyVersion,
			Ephemeral:  resolved.Ephemeral,
			Kind:       action.Kind,
			Role:       action.Role,
			Ballot:     action.Ballot,
			EventID:    event.ProviderEventID,
		})
		return nil
	}); err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	return observations, identityErrors, parseErrors
}

func TestRepeatedViewerResolvesToOneActorKey(t *testing.T) {
	keys := syntheticKeys(t)
	mock := mockAdapter(t, source.ScenarioRepeatedComments, 5, 1)
	observations, identityErrors, parseErrors := runPipeline(t, mock, mockSession(), resolveProvider(keys, "village-a"))
	if len(identityErrors) != 0 || len(parseErrors) != 0 {
		t.Fatalf("unexpected errors: identity=%v parse=%v", identityErrors, parseErrors)
	}
	if len(observations) != 5 {
		t.Fatalf("observations = %d", len(observations))
	}
	for _, item := range observations[1:] {
		if item.ActorKey != observations[0].ActorKey {
			t.Fatal("repeated viewer resolved to more than one actor key")
		}
		if item.Kind != commands.KindRole || item.Role != domain.RoleBuilder {
			t.Fatalf("unexpected action: %+v", item)
		}
	}
}

func TestDistinctViewersResolveToDistinctActorKeys(t *testing.T) {
	keys := syntheticKeys(t)
	mock := mockAdapter(t, source.ScenarioMultipleParticipants, 6, 2)
	observations, _, _ := runPipeline(t, mock, mockSession(), resolveProvider(keys, "village-a"))
	if len(observations) != 6 {
		t.Fatalf("observations = %d", len(observations))
	}
	unique := make(map[domain.ActorKey]bool)
	for _, item := range observations {
		unique[item.ActorKey] = true
	}
	if len(unique) != 6 {
		t.Fatalf("distinct viewers produced %d actor keys", len(unique))
	}
}

func TestBroadcasterIsolation(t *testing.T) {
	keys := syntheticKeys(t)
	first := mockAdapter(t, source.ScenarioMultipleParticipants, 4, 3)
	second := mockAdapter(t, source.ScenarioMultipleParticipants, 4, 3)
	left, _, _ := runPipeline(t, first, mockSession(), resolveProvider(keys, "village-a"))
	right, _, _ := runPipeline(t, second, mockSession(), resolveProvider(keys, "village-b"))
	if len(left) != len(right) {
		t.Fatal("pipeline lengths differ")
	}
	for index := range left {
		if left[index].ActorKey == right[index].ActorKey {
			t.Fatalf("broadcaster isolation failed for event %d", index)
		}
	}
}

func TestEphemeralSessionIsolation(t *testing.T) {
	secret, err := hex.DecodeString(syntheticMaterialHex)
	if err != nil {
		t.Fatal(err)
	}
	expiry := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return expiry.Add(-time.Hour) }
	first, err := identity.NewEphemeral(identity.EphemeralConfig{SessionID: "session-a", Secret: secret, ExpiresAt: expiry, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	second, err := identity.NewEphemeral(identity.EphemeralConfig{SessionID: "session-b", Secret: secret, ExpiresAt: expiry, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	left, err := first.Derive("viewer-0001")
	if err != nil {
		t.Fatal(err)
	}
	right, err := second.Derive("viewer-0001")
	if err != nil {
		t.Fatal(err)
	}
	if left.ActorKey == right.ActorKey {
		t.Fatal("ephemeral identities correlate across sessions")
	}
	again, err := first.Derive("viewer-0001")
	if err != nil {
		t.Fatal(err)
	}
	if again.ActorKey != left.ActorKey || !left.Ephemeral {
		t.Fatal("ephemeral identity is not stable/labelled within its session")
	}
}

func TestResumePreservesIdentity(t *testing.T) {
	keys := syntheticKeys(t)
	mock := mockAdapter(t, source.ScenarioMultipleParticipants, 6, 4)
	baseline, _, _ := runPipeline(t, mock, mockSession(), resolveProvider(keys, "village-a"))
	if len(baseline) != 6 {
		t.Fatalf("baseline = %d", len(baseline))
	}

	var resumed []observation
	if err := mock.Reconnect(context.Background(), source.NewCursor(mockSession().ID, mock.SourceID(), 2), func(_ context.Context, event source.Event) error {
		resolved, err := resolveProvider(keys, "village-a")(event)
		if err != nil {
			t.Fatalf("resume identity: %v", err)
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			t.Fatalf("resume parse: %v", err)
		}
		resumed = append(resumed, observation{ActorKey: resolved.ActorKey, KeyVersion: resolved.KeyVersion, Role: action.Role, Ballot: action.Ballot, Kind: action.Kind, EventID: event.ProviderEventID})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resumed, baseline[2:]) {
		t.Fatalf("resumed identity/actions differ:\n%+v\n%+v", resumed, baseline[2:])
	}
}

func TestReplayIdentityIsDeterministic(t *testing.T) {
	run := func() []observation {
		keys := syntheticKeys(t)
		replay := replayAdapter(t, "replay_unicode.ndjson")
		observations, _, _ := runPipeline(t, replay, replaySession(), resolveProvider(keys, "village-a"))
		return observations
	}
	if !reflect.DeepEqual(run(), run()) {
		t.Fatal("replay identity/action observations are not deterministic")
	}
}

func TestAliasesDoNotAffectActorKey(t *testing.T) {
	keys := syntheticKeys(t)
	stamp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []source.Event{
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e1", ProviderViewerID: "stable-viewer", Alias: "Alpha", Message: "XÂY", Kind: source.KindComment, ReceivedAt: stamp},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e2", ProviderViewerID: "stable-viewer", Alias: "Completely Different", Message: "THỦ", Kind: source.KindComment, ReceivedAt: stamp},
	}
	mock, err := source.NewMockWithEvents(mockSession(), events)
	if err != nil {
		t.Fatal(err)
	}
	observations, _, _ := runPipeline(t, mock, mockSession(), resolveProvider(keys, "village-a"))
	if len(observations) != 2 {
		t.Fatalf("observations = %d", len(observations))
	}
	if observations[0].ActorKey != observations[1].ActorKey {
		t.Fatal("display alias affected actor key derivation")
	}
	if observations[0].Role == observations[1].Role {
		t.Fatal("fixture roles did not change; test is vacuous")
	}
}

func TestDuplicateProviderEventIDsRemainObservable(t *testing.T) {
	keys := syntheticKeys(t)
	replay := replayAdapter(t, "replay_duplicates.ndjson")
	observations, _, _ := runPipeline(t, replay, replaySession(), resolveProvider(keys, "village-a"))
	if len(observations) != 4 {
		t.Fatalf("observations = %d", len(observations))
	}
	if observations[0].EventID != observations[1].EventID || observations[2].EventID != observations[3].EventID {
		t.Fatalf("duplicate event ids lost: %+v", observations)
	}
	if observations[0].ActorKey != observations[1].ActorKey || observations[2].ActorKey != observations[3].ActorKey {
		t.Fatal("same viewer did not resolve to the same actor key across duplicate records")
	}
}

func TestMissingIdentityFailsClosed(t *testing.T) {
	keys := syntheticKeys(t)
	stamp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []source.Event{{
		Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e1",
		Alias: "Anonymous", Message: "XÂY", Kind: source.KindComment, ReceivedAt: stamp,
	}}
	mock, err := source.NewMockWithEvents(mockSession(), events)
	if err != nil {
		t.Fatal(err)
	}
	observations, identityErrors, parseErrors := runPipeline(t, mock, mockSession(), resolveProvider(keys, "village-a"))
	if len(observations) != 0 || len(parseErrors) != 0 {
		t.Fatalf("missing identity produced observations/actions: %d/%d", len(observations), len(parseErrors))
	}
	if len(identityErrors) != 1 || !errors.Is(identityErrors[0], identity.ErrMissingViewerID) {
		t.Fatalf("identity error = %v", identityErrors)
	}
	// No ephemeral fallback exists in this provider-backed resolver.
	if strings.Contains(identityErrors[0].Error(), "ephemeral") {
		t.Fatal("missing stable identity was silently replaced by an ephemeral identity")
	}
}

func TestNonCommentAndInvalidGrammarProduceNoAction(t *testing.T) {
	keys := syntheticKeys(t)
	nonComment := mockAdapter(t, source.ScenarioNonCommentEvents, 8, 5)
	observations, _, parseErrors := runPipeline(t, nonComment, mockSession(), resolveProvider(keys, "village-a"))
	if len(observations) != 0 || len(parseErrors) != 0 {
		t.Fatalf("non-comment events produced actions: %d/%d", len(observations), len(parseErrors))
	}

	invalid := replayAdapter(t, "replay_invalid_grammar.ndjson")
	observations, identityErrors, parseErrors := runPipeline(t, invalid, replaySession(), resolveProvider(keys, "village-a"))
	if len(identityErrors) != 0 {
		t.Fatalf("identity errors: %v", identityErrors)
	}
	if len(observations) != 1 || observations[0].Ballot != domain.VoteOptionA {
		t.Fatalf("invalid grammar observations = %+v", observations)
	}
	if len(parseErrors) != 4 {
		t.Fatalf("parse errors = %d, want 4", len(parseErrors))
	}
}

func TestNoRawIdentifiersInPublicOutput(t *testing.T) {
	replay := replayAdapter(t, "replay_malformed.ndjson")
	err := replay.Start(context.Background(), replaySession(), func(context.Context, source.Event) error { return nil })
	if err == nil {
		t.Fatal("malformed fixture accepted")
	}
	health := replay.Health(context.Background())
	public := err.Error() + " " + health.LastError
	for _, forbidden := range []string{"viewer-0001", "event-0001", "synthetic-session-001"} {
		if strings.Contains(public, forbidden) {
			t.Fatalf("public source output leaked %q: %s", forbidden, public)
		}
	}

	keys := syntheticKeys(t)
	_, deriveErr := keys.Derive(identity.Input{Provider: "mock", ProviderVersion: "v1", BroadcasterKey: "village-a"})
	if deriveErr == nil {
		t.Fatal("missing viewer unexpectedly derived")
	}
	if strings.Contains(deriveErr.Error(), "viewer-") || strings.Contains(deriveErr.Error(), syntheticMaterialHex) {
		t.Fatalf("identity error leaked private material: %v", deriveErr)
	}

	keysetText := fmt.Sprintf("%v %#v", keys, keys)
	if strings.Contains(keysetText, syntheticMaterialHex) {
		t.Fatalf("keyset formatting leaked synthetic key material: %s", keysetText)
	}
}

func TestSourceScriptTamperResistanceThroughPipeline(t *testing.T) {
	keys := syntheticKeys(t)
	events := []source.Event{
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e1", ProviderViewerID: "viewer-a", Alias: "A", Message: "XÂY", Kind: source.KindComment},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e2", ProviderViewerID: "viewer-b", Alias: "B", Message: "THỦ", Kind: source.KindComment},
	}
	mock, err := source.NewMockWithEvents(mockSession(), events)
	if err != nil {
		t.Fatal(err)
	}
	sourceID := mock.SourceID()
	before, _, _ := runPipeline(t, mock, mockSession(), resolveProvider(keys, "village-a"))

	events[0].Message = "ATTACK"
	events[0].ProviderViewerID = "mutated"
	if mock.SourceID() != sourceID {
		t.Fatal("source identity changed after caller mutation")
	}
	var after []observation
	if err := mock.Reconnect(context.Background(), source.NewCursor(mockSession().ID, sourceID, 0), func(_ context.Context, event source.Event) error {
		resolved, err := resolveProvider(keys, "village-a")(event)
		if err != nil {
			t.Fatalf("identity: %v", err)
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		after = append(after, observation{ActorKey: resolved.ActorKey, KeyVersion: resolved.KeyVersion, Kind: action.Kind, Role: action.Role, Ballot: action.Ballot, EventID: event.ProviderEventID})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("caller mutation changed pipeline output:\n%+v\n%+v", before, after)
	}
}
