package ingest

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/identity"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

func benchmarkDeduper(b *testing.B) *Deduper {
	b.Helper()
	return benchmarkDeduperWith(b, nil)
}

func benchmarkDeduperWith(b *testing.B, mutate func(*Config)) *Deduper {
	b.Helper()
	config := DefaultConfig()
	config.Now = func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }
	config.MaxEntries = 1_000_000
	config.MaxActors = 1_000_000
	if mutate != nil {
		mutate(&config)
	}
	deduper, err := New(config)
	if err != nil {
		b.Fatal(err)
	}
	return deduper
}

func benchAction(b *testing.B, message string) commands.Action {
	b.Helper()
	action, err := commands.ParseComment(message)
	if err != nil {
		b.Fatal(err)
	}
	return action
}

func BenchmarkStableFirstSeen(b *testing.B) {
	deduper := benchmarkDeduper(b)
	actor := actorFor(1)
	action := benchAction(b, "XÂY")
	event := commentEvent("session-a", "mock", "")
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		event.ProviderEventID = fmt.Sprintf("first-%d", index)
		decision, reservation, err := deduper.Admit(event, actor, action)
		if err != nil || decision.Class != ClassAccepted {
			b.Fatalf("%+v/%v", decision, err)
		}
		if err := reservation.Release(); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(deduper.Stats().StableEntries), "entries")
}

func BenchmarkStableDuplicate(b *testing.B) {
	deduper := benchmarkDeduper(b)
	actor := actorFor(2)
	action := benchAction(b, "XÂY")
	event := commentEvent("session-a", "mock", "duplicate")
	decision, reservation, err := deduper.Admit(event, actor, action)
	if err != nil || decision.Class != ClassAccepted {
		b.Fatal(err)
	}
	if err := reservation.Finalize(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		decision, reservation, err := deduper.Admit(event, actor, action)
		if err != nil || decision.Class != ClassDuplicateProviderID || reservation != nil {
			b.Fatalf("%+v/%v", decision, err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(deduper.Stats().StableEntries), "entries")
}

func BenchmarkStableConflict(b *testing.B) {
	deduper := benchmarkDeduper(b)
	actor := actorFor(3)
	first := benchAction(b, "XÂY")
	conflict := benchAction(b, "THỦ")
	event := commentEvent("session-a", "mock", "conflict")
	decision, reservation, err := deduper.Admit(event, actor, first)
	if err != nil || decision.Class != ClassAccepted {
		b.Fatal(err)
	}
	if err := reservation.Finalize(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		decision, reservation, err := deduper.Admit(event, actor, conflict)
		if err != nil || decision.Class != ClassConflictingProviderID || reservation != nil {
			b.Fatalf("%+v/%v", decision, err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(deduper.Stats().StableEntries), "entries")
}

func BenchmarkBestEffortDuplicate(b *testing.B) {
	deduper := benchmarkDeduper(b)
	actor := actorFor(4)
	action := benchAction(b, "XÂY")
	event := commentEvent("session-a", "mock", "")
	decision, reservation, err := deduper.Admit(event, actor, action)
	if err != nil || decision.Class != ClassAccepted {
		b.Fatal(err)
	}
	if err := reservation.Finalize(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		decision, reservation, err := deduper.Admit(event, actor, action)
		if err != nil || decision.Class != ClassRepeatNoop || reservation != nil {
			b.Fatalf("%+v/%v", decision, err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(deduper.Stats().WindowEntries), "entries")
}

func BenchmarkRateAllowed(b *testing.B) {
	deduper := benchmarkDeduper(b)
	actor := actorFor(5)
	action := benchAction(b, "XÂY")
	event := commentEvent("session-a", "mock", "")
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		event.ProviderEventID = fmt.Sprintf("rate-%d", index)
		decision, reservation, err := deduper.Admit(event, actor, action)
		if err != nil || decision.Class != ClassAccepted {
			b.Fatalf("%+v/%v", decision, err)
		}
		if err := reservation.Release(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRateRejected(b *testing.B) {
	deduper := benchmarkDeduper(b)
	actor := actorFor(6)
	action := benchAction(b, "XÂY")
	event := commentEvent("session-a", "mock", "")
	for index := 0; index < 10; index++ {
		event.ProviderEventID = fmt.Sprintf("prefill-%d", index)
		decision, reservation, err := deduper.Admit(event, actor, action)
		if err != nil || decision.Class != ClassAccepted {
			b.Fatal(err)
		}
		if err := reservation.Finalize(); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		event.ProviderEventID = fmt.Sprintf("reject-%d", index)
		decision, reservation, err := deduper.Admit(event, actor, action)
		if err != nil || decision.Class != ClassRateLimited || reservation != nil {
			b.Fatalf("%+v/%v", decision, err)
		}
	}
}

func BenchmarkThousandRepeatedComments(b *testing.B) {
	actor := actorFor(7)
	action := benchAction(b, "XÂY")
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		deduper := benchmarkDeduper(b)
		event := commentEvent("session-a", "mock", "")
		for index := 0; index < 1000; index++ {
			event.ProviderEventID = fmt.Sprintf("spam-%d-%d", iteration, index)
			decision, reservation, err := deduper.Admit(event, actor, action)
			if err != nil {
				b.Fatal(err)
			}
			if decision.Class == ClassAccepted {
				if err := reservation.Finalize(); err != nil {
					b.Fatal(err)
				}
			}
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*1000), "ns/event")
}

func BenchmarkThousandUniqueViewers(b *testing.B) {
	action := benchAction(b, "XÂY")
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		deduper := benchmarkDeduper(b)
		for index := 0; index < 1000; index++ {
			event := commentEvent("session-a", "mock", fmt.Sprintf("viewer-%d-%d", iteration, index))
			decision, reservation, err := deduper.Admit(event, actorFor(iteration*1000+index), action)
			if err != nil || decision.Class != ClassAccepted {
				b.Fatalf("%+v/%v", decision, err)
			}
			if err := reservation.Finalize(); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*1000), "ns/event")
}

func BenchmarkIdentitySourceParserIngestBridge(b *testing.B) {
	key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		b.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		b.Fatal(err)
	}
	deduper := benchmarkDeduper(b)
	session := source.Session{ID: "synthetic-session-001", Mode: source.ModeMock}
	event := source.Event{
		Provider: "mock", SourceSession: session.ID, ProviderViewerID: "viewer-1",
		Alias: "Villager", Message: "XÂY", Kind: source.KindComment,
	}
	action, err := commands.ParseComment(event.Message)
	if err != nil {
		b.Fatal(err)
	}
	input := identity.Input{Provider: "mock", ProviderVersion: "v1", BroadcasterKey: "village-a", ViewerID: "viewer-1"}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		event.ProviderEventID = fmt.Sprintf("bridge-%d", index)
		if err := source.ValidateEvent(event, session, source.ModeMock); err != nil {
			b.Fatal(err)
		}
		resolved, err := keys.Derive(input)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := commands.ParseComment(event.Message); err != nil {
			b.Fatal(err)
		}
		decision, reservation, err := deduper.Admit(event, resolved.ActorKey, action)
		if err != nil || decision.Class != ClassAccepted {
			b.Fatalf("%+v/%v", decision, err)
		}
		if err := reservation.Release(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallelAdmission(b *testing.B) {
	deduper := benchmarkDeduperWith(b, func(config *Config) {
		config.RateCapacity = 1e9
	})
	action := benchAction(b, "XÂY")
	var counter atomic.Uint64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index := counter.Add(1)
			event := commentEvent("session-a", "mock", fmt.Sprintf("parallel-%d", index))
			decision, reservation, err := deduper.Admit(event, actorFor(8), action)
			if err != nil || decision.Class != ClassAccepted {
				b.Errorf("%+v/%v", decision, err)
				return
			}
			if err := reservation.Release(); err != nil {
				b.Errorf("%v", err)
				return
			}
		}
	})
}
