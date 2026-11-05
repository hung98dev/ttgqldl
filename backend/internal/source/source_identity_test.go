package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func scriptedEvents() []Event {
	stamp := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	return []Event{
		{
			Provider: "mock", SourceSession: "synthetic-session-001",
			ProviderEventID: "e1", ProviderViewerID: "v1", Alias: "Villager 1",
			Message: "XÂY", Kind: KindComment, ReceivedAt: stamp, UpstreamAt: &stamp,
		},
		{
			Provider: "mock", SourceSession: "synthetic-session-001",
			ProviderEventID: "e2", ProviderViewerID: "v2", Alias: "Villager 2",
			Message: "THỦ", Kind: KindComment, ReceivedAt: stamp.Add(time.Second),
		},
	}
}

func TestMockScriptIsImmutableAfterConstruction(t *testing.T) {
	session := testSession(ModeMock)
	events := scriptedEvents()
	want := copyEvents(events)
	wantStamp := *events[0].UpstreamAt
	mock, err := NewMockWithEvents(session, events)
	if err != nil {
		t.Fatal(err)
	}
	sourceID := mock.SourceID()

	// Caller mutates the original slice, an event field and a shared timestamp.
	events[0].Message = "THỦ"
	events[0].ProviderViewerID = "mutated"
	events[1].Kind = KindGift
	*events[0].UpstreamAt = wantStamp.Add(time.Hour)
	events[0] = Event{Provider: "mock", Kind: KindComment, Message: "ATTACK"}

	var delivered []Event
	if err := mock.Start(context.Background(), session, func(_ context.Context, event Event) error {
		delivered = append(delivered, copyEvent(event))
		// Mutating the delivered view must not affect the stored script either.
		if event.UpstreamAt != nil {
			*event.UpstreamAt = event.UpstreamAt.Add(48 * time.Hour)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(delivered, want) {
		t.Fatalf("delivered mutated script:\n got %+v\nwant %+v", delivered, want)
	}
	if mock.SourceID() != sourceID {
		t.Fatal("source identity changed after delivery")
	}

	var resumed []Event
	if err := mock.Reconnect(context.Background(), NewCursor(session.ID, sourceID, 0), collectSink(&resumed)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resumed, want) {
		t.Fatalf("stored script was mutated by a sink: %+v", resumed)
	}
	if resumed[0].UpstreamAt == nil || !resumed[0].UpstreamAt.Equal(wantStamp) {
		t.Fatalf("upstream timestamp was mutated: %+v", resumed[0].UpstreamAt)
	}
}

func TestMockSourceIDCoversEveryField(t *testing.T) {
	session := testSession(ModeMock)
	base := []Event{scriptedEvents()[0]}
	baseID := mockSourceID(session, base)
	otherSession := Session{ID: "synthetic-session-002", Mode: ModeMock}
	if mockSourceID(otherSession, base) == baseID {
		t.Fatal("session is not bound into the source identity")
	}

	mutations := []struct {
		name   string
		mutate func(*Event)
	}{
		{"provider", func(e *Event) { e.Provider = "replay" }},
		{"event session", func(e *Event) { e.SourceSession = "synthetic-session-002" }},
		{"provider event id", func(e *Event) { e.ProviderEventID = "other" }},
		{"provider viewer id", func(e *Event) { e.ProviderViewerID = "other" }},
		{"alias", func(e *Event) { e.Alias = "other" }},
		{"message", func(e *Event) { e.Message = "THỦ" }},
		{"kind", func(e *Event) { e.Kind = KindGift }},
		{"received at", func(e *Event) { e.ReceivedAt = e.ReceivedAt.Add(time.Nanosecond) }},
		{"upstream value", func(e *Event) { changed := e.UpstreamAt.Add(time.Nanosecond); e.UpstreamAt = &changed }},
		{"upstream presence", func(e *Event) { e.UpstreamAt = nil }},
	}
	for _, testCase := range mutations {
		t.Run(testCase.name, func(t *testing.T) {
			event := base[0]
			if event.UpstreamAt != nil {
				stamp := *event.UpstreamAt
				event.UpstreamAt = &stamp
			}
			testCase.mutate(&event)
			if mockSourceID(session, []Event{event}) == baseID {
				t.Fatalf("%s is not covered by the source identity", testCase.name)
			}
		})
	}

	// Event order and count are part of the script identity.
	two := scriptedEvents()
	if mockSourceID(session, two) == baseID {
		t.Fatal("event count is not covered by the source identity")
	}
	swapped := []Event{two[1], two[0]}
	if mockSourceID(session, swapped) == mockSourceID(session, two) {
		t.Fatal("event order is not covered by the source identity")
	}
}

func TestMockSourceIDRejectsDelimiterAmbiguity(t *testing.T) {
	session := testSession(ModeMock)
	base := Event{Provider: "mock", SourceSession: session.ID, Alias: "same", Message: "XÂY", Kind: KindComment}
	pairs := []struct {
		aEvent, bEvent   string
		aViewer, bViewer string
	}{
		{"e|v", "e", "x", "v|x"},
		{"a|b", "a", "c", "b|c"},
		{"", "|", "a|b", "a|b"},
		{"e\n", "e", "v", "\nv"},
	}
	for _, pair := range pairs {
		a, b := base, base
		a.ProviderEventID, a.ProviderViewerID = pair.aEvent, pair.aViewer
		b.ProviderEventID, b.ProviderViewerID = pair.bEvent, pair.bViewer
		if reflect.DeepEqual(a, b) {
			t.Fatalf("fixture is not distinct: %+v", pair)
		}
		if mockSourceID(session, []Event{a}) == mockSourceID(session, []Event{b}) {
			t.Fatalf("delimiter ambiguity for %+v", pair)
		}
	}
}

func TestMockCursorRejectsChangedScript(t *testing.T) {
	session := testSession(ModeMock)
	first := scriptedEvents()
	second := scriptedEvents()
	second[0].Message = "ATTACK"

	mockA, err := NewMockWithEvents(session, first)
	if err != nil {
		t.Fatal(err)
	}
	mockB, err := NewMockWithEvents(session, second)
	if err != nil {
		t.Fatal(err)
	}
	if mockA.SourceID() == mockB.SourceID() {
		t.Fatal("different scripts share a source identity")
	}
	if err := mockA.Start(context.Background(), session, func(context.Context, Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	cursor := NewCursor(session.ID, mockA.SourceID(), 1)
	if err := mockB.Reconnect(context.Background(), cursor, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrCursorMismatch) {
		t.Fatalf("stale script cursor error = %v, want ErrCursorMismatch", err)
	}
}

func TestMockSourceIDIsVersionedDigest(t *testing.T) {
	session := testSession(ModeMock)
	id := mockSourceID(session, scriptedEvents())
	if len(id) != 64 {
		t.Fatalf("source id length = %d", len(id))
	}
	if _, err := hex.DecodeString(id); err != nil {
		t.Fatalf("source id is not lowercase hex: %v", err)
	}
	if id != strings.ToLower(id) {
		t.Fatal("source id is not lowercase")
	}
	if mockScriptIdentityVersion != 2 {
		t.Fatalf("source-script identity version = %d", mockScriptIdentityVersion)
	}
}

func TestLegacyAmbiguousSourceCursorFailsClosed(t *testing.T) {
	session := testSession(ModeMock)
	events := scriptedEvents()
	// Reconstruct the version-1 separator-based digest that older cursors carry.
	legacy := sha256.New()
	fmt.Fprintf(legacy, "ttg-mock-v1\nsession=%s\ncount=%d\n", session.ID, len(events))
	for _, event := range events {
		fmt.Fprintf(legacy, "event=%s|%s|%s|%s|%d\n",
			event.ProviderEventID, event.ProviderViewerID, event.Alias, event.Message, event.Kind)
	}
	legacyID := hex.EncodeToString(legacy.Sum(nil))

	mock, err := NewMockWithEvents(session, events)
	if err != nil {
		t.Fatal(err)
	}
	if mock.SourceID() == legacyID {
		t.Fatal("source identity did not change to version 2")
	}
	if err := mock.Start(context.Background(), session, func(context.Context, Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	legacyCursor := NewCursor(session.ID, legacyID, 0)
	if err := mock.Reconnect(context.Background(), legacyCursor, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrCursorMismatch) {
		t.Fatalf("legacy cursor error = %v, want ErrCursorMismatch", err)
	}
}
