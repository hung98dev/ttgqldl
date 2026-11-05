package ingest_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/identity"
	"github.com/hung98dev/ttgqldl/backend/internal/ingest"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

type observation struct {
	Class    ingest.Classification
	ActorKey domain.ActorKey
	Action   commands.Action
	EventID  string
}

type pipeline struct {
	keys         *identity.KeySet
	deduper      *ingest.Deduper
	broadcaster  string
	observations []observation
	classes      map[ingest.Classification]int
	identityErr  int
	parseErr     int
	calls        int
	failAfter    int
}

func newPipeline(t *testing.T, failAfter int) *pipeline {
	t.Helper()
	key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	deduper, err := ingest.New(ingest.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	return &pipeline{
		keys:        keys,
		deduper:     deduper,
		broadcaster: "village-a",
		classes:     make(map[ingest.Classification]int),
		failAfter:   failAfter,
	}
}

func (p *pipeline) handle(_ context.Context, event source.Event) error {
	p.calls++
	if p.failAfter > 0 && p.calls > p.failAfter {
		return errors.New("downstream failure")
	}
	if event.Kind != source.KindComment {
		return nil
	}
	resolved, err := p.keys.Derive(identity.Input{
		Provider:        event.Provider,
		ProviderVersion: "v1",
		BroadcasterKey:  p.broadcaster,
		ViewerID:        event.ProviderViewerID,
	})
	if err != nil {
		p.identityErr++
		return nil
	}
	action, err := commands.ParseComment(event.Message)
	if err != nil {
		p.parseErr++
		return nil
	}
	decision, reservation, err := p.deduper.Admit(event, resolved.ActorKey, action)
	if err != nil && !errors.Is(err, ingest.ErrCacheFull) {
		return err
	}
	p.classes[decision.Class]++
	if decision.Class != ingest.ClassAccepted {
		return nil
	}
	if err := reservation.Finalize(); err != nil {
		return err
	}
	p.observations = append(p.observations, observation{
		Class:    decision.Class,
		ActorKey: resolved.ActorKey,
		Action:   action,
		EventID:  event.ProviderEventID,
	})
	return nil
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

func TestUniqueParticipants(t *testing.T) {
	p := newPipeline(t, 0)
	mock := mockAdapter(t, source.ScenarioMultipleParticipants, 100, 1)
	if err := mock.Start(context.Background(), mockSession(), p.handle); err != nil {
		t.Fatal(err)
	}
	if len(p.observations) != 100 || p.classes[ingest.ClassAccepted] != 100 {
		t.Fatalf("accepted = %d", len(p.observations))
	}
	unique := make(map[domain.ActorKey]bool)
	for _, item := range p.observations {
		unique[item.ActorKey] = true
	}
	if len(unique) != 100 {
		t.Fatalf("unique actors = %d", len(unique))
	}
}

func TestStableIDDuplicates(t *testing.T) {
	p := newPipeline(t, 0)
	stamp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	events := make([]source.Event, 0, 100)
	for index := 0; index < 100; index++ {
		events = append(events, source.Event{
			Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "duplicate-1",
			ProviderViewerID: "viewer-1", Alias: "Villager", Message: "XÂY",
			Kind: source.KindComment, ReceivedAt: stamp.Add(time.Duration(index) * time.Second),
		})
	}
	mock, err := source.NewMockWithEvents(mockSession(), events)
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.Start(context.Background(), mockSession(), p.handle); err != nil {
		t.Fatal(err)
	}
	if len(p.observations) != 1 || p.classes[ingest.ClassAccepted] != 1 {
		t.Fatalf("accepted = %d", len(p.observations))
	}
	if p.classes[ingest.ClassDuplicateProviderID] != 99 {
		t.Fatalf("duplicates = %d", p.classes[ingest.ClassDuplicateProviderID])
	}
}

func TestReconnectRedelivery(t *testing.T) {
	p := newPipeline(t, 2)
	mock := mockAdapter(t, source.ScenarioFiniteScript, 5, 2)
	err := mock.Start(context.Background(), mockSession(), p.handle)
	if !errors.Is(err, source.ErrDeliveryRejected) {
		t.Fatalf("interrupted start = %v", err)
	}
	health := mock.Health(context.Background())
	if health.Status != source.StatusDisconnected || health.Cursor.NextEvent != 2 {
		t.Fatalf("health after interruption = %+v", health)
	}
	if len(p.observations) != 2 {
		t.Fatalf("observations after interruption = %d", len(p.observations))
	}
	p.failAfter = 0
	if err := mock.Reconnect(context.Background(), health.Cursor, p.handle); err != nil {
		t.Fatal(err)
	}
	if len(p.observations) != 5 {
		t.Fatalf("observations after reconnect = %d", len(p.observations))
	}
	// Deliberate redelivery from the beginning must not create candidates.
	before := len(p.observations)
	if err := mock.Reconnect(context.Background(), source.NewCursor(mockSession().ID, mock.SourceID(), 0), p.handle); err != nil {
		t.Fatal(err)
	}
	if len(p.observations) != before {
		t.Fatalf("redelivery created %d extra candidates", len(p.observations)-before)
	}
	if p.classes[ingest.ClassDuplicateProviderID] != 5 {
		t.Fatalf("redelivery duplicates = %d", p.classes[ingest.ClassDuplicateProviderID])
	}
}

func TestConflictingIDsInPipeline(t *testing.T) {
	p := newPipeline(t, 0)
	events := []source.Event{
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "conflict-1", ProviderViewerID: "viewer-a", Message: "XÂY", Kind: source.KindComment},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "conflict-1", ProviderViewerID: "viewer-a", Message: "ATTACK", Kind: source.KindComment},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "conflict-1", ProviderViewerID: "viewer-b", Message: "XÂY", Kind: source.KindComment},
	}
	mock, err := source.NewMockWithEvents(mockSession(), events)
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.Start(context.Background(), mockSession(), p.handle); err != nil {
		t.Fatal(err)
	}
	if len(p.observations) != 1 || p.classes[ingest.ClassConflictingProviderID] != 2 {
		t.Fatalf("accepted=%d conflicts=%d", len(p.observations), p.classes[ingest.ClassConflictingProviderID])
	}
}

func TestMissingEventIDsBestEffort(t *testing.T) {
	p := newPipeline(t, 0)
	events := []source.Event{
		{Provider: "mock", SourceSession: mockSession().ID, ProviderViewerID: "viewer-a", Message: "XÂY", Kind: source.KindComment},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderViewerID: "viewer-a", Message: "BUILD", Kind: source.KindComment},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderViewerID: "viewer-a", Message: "THỦ", Kind: source.KindComment},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderViewerID: "viewer-a", Message: "A", Kind: source.KindComment},
	}
	mock, err := source.NewMockWithEvents(mockSession(), events)
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.Start(context.Background(), mockSession(), p.handle); err != nil {
		t.Fatal(err)
	}
	if len(p.observations) != 3 {
		t.Fatalf("observations = %d", len(p.observations))
	}
	if p.classes[ingest.ClassRepeatNoop] != 1 {
		t.Fatalf("repeat = %d", p.classes[ingest.ClassRepeatNoop])
	}
}

func TestThousandCommentSpamIsBounded(t *testing.T) {
	p := newPipeline(t, 0)
	mock := mockAdapter(t, source.ScenarioRepeatedComments, 1000, 3)
	if err := mock.Start(context.Background(), mockSession(), p.handle); err != nil {
		t.Fatal(err)
	}
	if len(p.observations) != 10 {
		t.Fatalf("accepted = %d, want burst 10", len(p.observations))
	}
	if p.classes[ingest.ClassRateLimited] != 990 {
		t.Fatalf("rate limited = %d", p.classes[ingest.ClassRateLimited])
	}
	stats := p.deduper.Stats()
	if stats.StableEntries != 10 || stats.Buckets != 1 {
		t.Fatalf("bounded stats = %+v", stats)
	}
}

func TestNonCommentEventsNeverBecomeCandidates(t *testing.T) {
	p := newPipeline(t, 0)
	mock := mockAdapter(t, source.ScenarioNonCommentEvents, 20, 4)
	if err := mock.Start(context.Background(), mockSession(), p.handle); err != nil {
		t.Fatal(err)
	}
	if len(p.observations) != 0 || p.classes[ingest.ClassAccepted] != 0 {
		t.Fatalf("non-comment candidates = %d", len(p.observations))
	}
}

func TestReplayDuplicateFixture(t *testing.T) {
	p := newPipeline(t, 0)
	replay := replayAdapter(t, "replay_duplicates.ndjson")
	if err := replay.Start(context.Background(), replaySession(), p.handle); err != nil {
		t.Fatal(err)
	}
	if len(p.observations) != 2 {
		t.Fatalf("accepted = %d", len(p.observations))
	}
	if p.classes[ingest.ClassDuplicateProviderID] != 1 || p.classes[ingest.ClassConflictingProviderID] != 1 {
		t.Fatalf("classes = %v", p.classes)
	}
}

func TestSecurityFailuresFailClosed(t *testing.T) {
	p := newPipeline(t, 0)
	missingViewer := source.Event{
		Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "missing-viewer",
		Message: "XÂY", Kind: source.KindComment,
	}
	mock, err := source.NewMockWithEvents(mockSession(), []source.Event{missingViewer})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.Start(context.Background(), mockSession(), p.handle); err != nil {
		t.Fatal(err)
	}
	if p.identityErr != 1 || len(p.observations) != 0 {
		t.Fatalf("missing identity: err=%d obs=%d", p.identityErr, len(p.observations))
	}

	// Source-valid but grammar-invalid spoof: parser rejects, no ingest candidate.
	spoof := source.Event{
		Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "spoof",
		ProviderViewerID: "viewer-a", Message: "XAY\u034F", Kind: source.KindComment,
	}
	spoofMock, err := source.NewMockWithEvents(mockSession(), []source.Event{spoof})
	if err != nil {
		t.Fatal(err)
	}
	spoofPipeline := newPipeline(t, 0)
	if err := spoofMock.Start(context.Background(), mockSession(), spoofPipeline.handle); err != nil {
		t.Fatal(err)
	}
	if spoofPipeline.parseErr != 1 || len(spoofPipeline.observations) != 0 {
		t.Fatalf("spoof: parse=%d obs=%d", spoofPipeline.parseErr, len(spoofPipeline.observations))
	}

	if _, err := identity.NewKey("v1", []byte("short")); !errors.Is(err, identity.ErrWeakKey) {
		t.Fatalf("malformed HMAC config = %v", err)
	}
}

func TestCrossScopeEventIDIsolation(t *testing.T) {
	deduper, err := ingest.New(ingest.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	action, err := commands.ParseComment("XÂY")
	if err != nil {
		t.Fatal(err)
	}
	actorA := domain.ActorKey("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	actorB := domain.ActorKey("fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210")
	scopes := []source.Event{
		{Provider: "mock", SourceSession: "session-a", ProviderEventID: "shared", Kind: source.KindComment},
		{Provider: "mock", SourceSession: "session-b", ProviderEventID: "shared", Kind: source.KindComment},
		{Provider: "replay", SourceSession: "session-a", ProviderEventID: "shared", Kind: source.KindComment},
	}
	for index, event := range scopes {
		decision, reservation, err := deduper.Admit(event, actorA, action)
		if err != nil || decision.Class != ingest.ClassAccepted || reservation == nil {
			t.Fatalf("scope %d = %+v/%v", index, decision, err)
		}
		if err := reservation.Finalize(); err != nil {
			t.Fatal(err)
		}
	}
	decision, _, err := deduper.Admit(scopes[0], actorB, action)
	if err != nil || decision.Class != ingest.ClassConflictingProviderID {
		t.Fatalf("same scope different actor = %+v/%v", decision, err)
	}
}
