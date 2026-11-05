package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/identity"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

func TestMetricsSourcePipelineStages(t *testing.T) {
	metrics := NewMetrics()
	key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	deduper, err := New(Config{Now: func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }, Metrics: metrics})
	if err != nil {
		t.Fatal(err)
	}
	appender := &fakeAppender{}
	batcher, err := NewBatcher(appender, BatcherConfig{
		QueueCapacity: 64, MaxBatch: 8, FlushInterval: 10 * time.Millisecond,
		Metrics: metrics, Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = batcher.Close(ctx)
	}()
	session := source.Session{ID: "synthetic-session-001", Mode: source.ModeMock}
	stamp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	events := []source.Event{
		{Provider: "mock", SourceSession: session.ID, ProviderEventID: "p1", ProviderViewerID: "viewer-1", Message: "XÂY", Kind: source.KindComment, ReceivedAt: stamp},
		{Provider: "mock", SourceSession: session.ID, ProviderEventID: "p2", ProviderViewerID: "viewer-2", Message: "THỦ", Kind: source.KindComment, ReceivedAt: stamp},
		{Provider: "mock", SourceSession: session.ID, ProviderEventID: "p3", ProviderViewerID: "viewer-3", Message: "A", Kind: source.KindComment, ReceivedAt: stamp},
		{Provider: "mock", SourceSession: session.ID, ProviderEventID: "gift-1", Kind: source.KindGift, ReceivedAt: stamp},
		{Provider: "mock", SourceSession: session.ID, ProviderEventID: "bad-grammar", ProviderViewerID: "viewer-4", Message: "XAY\u034F", Kind: source.KindComment, ReceivedAt: stamp},
		{Provider: "mock", SourceSession: session.ID, ProviderEventID: "no-viewer", Message: "XÂY", Kind: source.KindComment, ReceivedAt: stamp},
	}
	mock, err := source.NewMockWithEvents(session, events)
	if err != nil {
		t.Fatal(err)
	}
	sink := func(ctx context.Context, event source.Event) error {
		metrics.ObserveReceived()
		if event.Kind != source.KindComment {
			_, _, _ = deduper.Admit(event, "", commands.Action{})
			return nil
		}
		resolved, err := keys.Derive(identity.Input{Provider: event.Provider, ProviderVersion: "v1", BroadcasterKey: "village-a", ViewerID: event.ProviderViewerID})
		if err != nil {
			metrics.ObserveInvalidIdentity()
			return nil
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			metrics.ObserveInvalidGrammar()
			return nil
		}
		metrics.ObserveValidated()
		decision, reservation, err := deduper.Admit(event, resolved.ActorKey, action)
		if err != nil || decision.Class != ClassAccepted {
			return nil
		}
		_, err = batcher.Submit(ctx, DurableCandidate{
			SessionID: event.SourceSession, Provider: event.Provider, ProviderEventID: event.ProviderEventID,
			Fingerprint: payloadFingerprint(resolved.ActorKey, event.Kind, action),
			ActorKey:    resolved.ActorKey, Action: action,
		}, reservation)
		return err
	}
	if err := mock.Start(context.Background(), session, sink); err != nil {
		t.Fatal(err)
	}
	snapshot := metrics.Snapshot()
	if snapshot.InputReceived != 6 || snapshot.InputValidated != 3 || snapshot.InvalidGrammar != 1 ||
		snapshot.InvalidIdentity != 1 || snapshot.IgnoredNonCommand != 1 || snapshot.AcceptedCandidates != 3 ||
		snapshot.Enqueued != 3 || snapshot.DurableCommitted != 3 || snapshot.Inputs != 3 {
		t.Fatalf("pipeline stages = %+v", snapshot)
	}
}
