package source

import (
	"context"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
)

func BenchmarkScenarioEvents1000(b *testing.B) {
	session := testSession(ModeMock)
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		events, err := ScenarioEvents(session, 1, ScenarioMultipleParticipants, 1000)
		if err != nil || len(events) != 1000 {
			b.Fatal(err)
		}
	}
}

func BenchmarkMockResume1000(b *testing.B) {
	config := MockConfig{Session: testSession(ModeMock), Seed: 1, Scenario: ScenarioMultipleParticipants, Count: 1000}
	mock, err := NewMock(config)
	if err != nil {
		b.Fatal(err)
	}
	noop := func(context.Context, Event) error { return nil }
	if err := mock.Start(context.Background(), config.Session, noop); err != nil {
		b.Fatal(err)
	}
	cursor := NewCursor(config.Session.ID, mock.SourceID(), 0)
	b.ResetTimer()
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if err := mock.Reconnect(context.Background(), cursor, noop); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReplayDecodeRecord(b *testing.B) {
	line := []byte(`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","provider_event_id":"event-0001","provider_viewer_id":"viewer-0001","alias":"Villager 0001","event_kind":"comment","message":"XÂY","received_at":"2026-01-01T00:00:01Z"}`)
	session := replaySession()
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if _, err := decodeReplayRecord(line, session, 1); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReplayStreamSix(b *testing.B) {
	replay, err := NewReplay(ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: "replay_valid.ndjson"})
	if err != nil {
		b.Fatal(err)
	}
	noop := func(context.Context, Event) error { return nil }
	if err := replay.Start(context.Background(), replaySession(), noop); err != nil {
		b.Fatal(err)
	}
	cursor := NewCursor(replaySession().ID, replay.SourceID(), 0)
	b.ResetTimer()
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if err := replay.Reconnect(context.Background(), cursor, noop); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateEvent(b *testing.B) {
	event := validEvent()
	session := testSession(ModeMock)
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if err := ValidateEvent(event, session, ModeMock); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParserBridge(b *testing.B) {
	event := validEvent()
	session := testSession(ModeMock)
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if err := ValidateEvent(event, session, ModeMock); err != nil {
			b.Fatal(err)
		}
		if _, err := commands.ParseComment(event.Message); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCursorParse(b *testing.B) {
	encoded := NewCursor("synthetic-session-001", "digest", 12345).Encode()
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if _, err := ParseCursor(encoded); err != nil {
			b.Fatal(err)
		}
	}
}
