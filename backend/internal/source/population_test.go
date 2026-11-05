package source

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestScenarioTenThousandParticipantsIsDeterministicAndUnique(t *testing.T) {
	session := Session{ID: "synthetic-session-w0307", Mode: ModeMock}
	const seed = uint64(20261105)
	events, err := ScenarioEvents(session, seed, ScenarioMultipleParticipants, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 10_000 {
		t.Fatalf("events = %d, want 10000", len(events))
	}
	viewers := make(map[string]struct{}, 10_000)
	eventIDs := make(map[string]struct{}, 10_000)
	roles := map[string]int{}
	for index, event := range events {
		if !strings.HasPrefix(event.ProviderViewerID, "mock-viewer-") {
			t.Fatalf("event %d viewer is not synthetic: %q", index, event.ProviderViewerID)
		}
		if _, exists := viewers[event.ProviderViewerID]; exists {
			t.Fatalf("duplicate viewer id at %d", index)
		}
		viewers[event.ProviderViewerID] = struct{}{}
		if _, exists := eventIDs[event.ProviderEventID]; exists {
			t.Fatalf("duplicate event id at %d", index)
		}
		eventIDs[event.ProviderEventID] = struct{}{}
		roles[event.Message]++
		if event.Provider != ModeMock.Provider() || event.SourceSession != session.ID || event.Kind != KindComment {
			t.Fatalf("event %d shape = %+v", index, event)
		}
	}
	if len(viewers) != 10_000 || len(eventIDs) != 10_000 {
		t.Fatalf("viewers=%d eventIDs=%d", len(viewers), len(eventIDs))
	}
	if roles["XÂY"] != 3334 || roles["THỦ"] != 3333 || roles["ĐÁNH"] != 3333 {
		t.Fatalf("role distribution = %+v", roles)
	}
	again, err := ScenarioEvents(session, seed, ScenarioMultipleParticipants, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(events, again) {
		t.Fatal("same (session, seed, scenario, count) produced a different script")
	}
	otherSeed, err := ScenarioEvents(session, seed+1, ScenarioMultipleParticipants, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if events[0].ProviderEventID == otherSeed[0].ProviderEventID || events[0].ProviderViewerID == otherSeed[0].ProviderViewerID {
		t.Fatal("seed did not change script identity")
	}
	otherSession, err := ScenarioEvents(Session{ID: "synthetic-session-w0307-other", Mode: ModeMock}, seed, ScenarioMultipleParticipants, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	if events[0].SourceSession == otherSession[0].SourceSession {
		t.Fatal("session did not change source scope")
	}
	if _, err := ScenarioEvents(session, seed, ScenarioMultipleParticipants, MaxMockEvents+1); err == nil {
		t.Fatal("expected the event bound to reject an oversized script")
	}
}

func TestScenarioTenThousandInterruptedCursor(t *testing.T) {
	session := Session{ID: "synthetic-session-interrupted-w0307", Mode: ModeMock}
	events, err := ScenarioEvents(session, 20261105, ScenarioMultipleParticipants, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	mock, err := NewMockWithEvents(session, events)
	if err != nil {
		t.Fatal(err)
	}
	delivered := 0
	failing := func(context.Context, Event) error {
		delivered++
		if delivered == 3_333 {
			return errors.New("injected boundary failure")
		}
		return nil
	}
	if err := mock.Start(context.Background(), session, failing); !errors.Is(err, ErrDeliveryRejected) {
		t.Fatalf("start error = %v", err)
	}
	health := mock.Health(context.Background())
	if health.Status != StatusDisconnected || health.Delivered != 3_332 || health.Cursor.NextEvent != 3_332 {
		t.Fatalf("cursor after failure = %+v", health)
	}
	resumed := 0
	var firstResumed Event
	if err := mock.Reconnect(context.Background(), health.Cursor, func(_ context.Context, event Event) error {
		if resumed == 0 {
			firstResumed = event
		}
		resumed++
		return nil
	}); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if resumed != 6_668 {
		t.Fatalf("resumed = %d, want 6668", resumed)
	}
	if firstResumed.ProviderEventID != events[3_332].ProviderEventID {
		t.Fatalf("first resumed = %q, want %q", firstResumed.ProviderEventID, events[3_332].ProviderEventID)
	}
	if final := mock.Health(context.Background()); final.Cursor.NextEvent != 10_000 || final.Delivered != 10_000 {
		t.Fatalf("final cursor = %+v", final)
	}
}

func TestScenarioTenThousandDeliversEveryValidEvent(t *testing.T) {
	session := Session{ID: "synthetic-delivery-w0307", Mode: ModeMock}
	events, err := ScenarioEvents(session, 20261105, ScenarioMultipleParticipants, 10_000)
	if err != nil {
		t.Fatal(err)
	}
	mock, err := NewMockWithEvents(session, events)
	if err != nil {
		t.Fatal(err)
	}
	delivered := 0
	viewers := make(map[string]struct{}, 10_000)
	deliveryStart := time.Now()
	if err := mock.Start(context.Background(), session, func(_ context.Context, event Event) error {
		if err := ValidateEvent(event, session, ModeMock); err != nil {
			return err
		}
		delivered++
		viewers[event.ProviderViewerID] = struct{}{}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if delivered != 10_000 || len(viewers) != 10_000 {
		t.Fatalf("delivered=%d viewers=%d", delivered, len(viewers))
	}
	health := mock.Health(context.Background())
	if health.Cursor.NextEvent != 10_000 || health.Delivered != 10_000 {
		t.Fatalf("health = %+v", health)
	}
	t.Logf("10k source delivery=%s", time.Since(deliveryStart).Round(time.Millisecond))
}
