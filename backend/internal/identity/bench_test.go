package identity

import (
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

func benchmarkInput() Input {
	return Input{
		Provider:        "mock",
		ProviderVersion: "v1",
		BroadcasterKey:  "village-a",
		ViewerID:        "viewer-0001",
	}
}

func BenchmarkDeriveSingle(b *testing.B) {
	key, err := NewKey("v1", validSecret())
	if err != nil {
		b.Fatal(err)
	}
	keys, err := NewKeySet(key, nil)
	if err != nil {
		b.Fatal(err)
	}
	input := benchmarkInput()
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := keys.Derive(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeriveCandidatesThreeVersions(b *testing.B) {
	active, err := NewKey("v3", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		b.Fatal(err)
	}
	v2, err := NewKey("v2", []byte("fedcba9876543210fedcba9876543210"))
	if err != nil {
		b.Fatal(err)
	}
	v1, err := NewKey("v1", []byte("00112233445566778899aabbccddeeff"))
	if err != nil {
		b.Fatal(err)
	}
	keys, err := NewKeySet(active, []Key{v2, v1})
	if err != nil {
		b.Fatal(err)
	}
	input := benchmarkInput()
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := keys.Candidates(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEphemeralDerive(b *testing.B) {
	expiry := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ephemeral, err := NewEphemeral(EphemeralConfig{
		SessionID: "synthetic-session-001",
		Secret:    validSecret(),
		ExpiresAt: expiry,
		Now:       func() time.Time { return expiry.Add(-time.Hour) },
	})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := ephemeral.Derive("synthetic-viewer-01"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIdentitySourceParserBridge measures the trust boundary plus identity plus
// parser for one comment; it is a microbenchmark, not W0307 population or M08 load
// acceptance.
func BenchmarkIdentitySourceParserBridge(b *testing.B) {
	session := source.Session{ID: "synthetic-session-001", Mode: source.ModeMock}
	event := source.Event{
		Provider:         "mock",
		SourceSession:    session.ID,
		ProviderEventID:  "event-0001",
		ProviderViewerID: "viewer-0001",
		Alias:            "Villager 0001",
		Message:          "XÂY",
		Kind:             source.KindComment,
	}
	key, err := NewKey("v1", validSecret())
	if err != nil {
		b.Fatal(err)
	}
	keys, err := NewKeySet(key, nil)
	if err != nil {
		b.Fatal(err)
	}
	input := benchmarkInput()
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if err := source.ValidateEvent(event, session, source.ModeMock); err != nil {
			b.Fatal(err)
		}
		if _, err := keys.Derive(input); err != nil {
			b.Fatal(err)
		}
		if _, err := commands.ParseComment(event.Message); err != nil {
			b.Fatal(err)
		}
	}
}
