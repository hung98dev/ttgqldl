package ingest

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/identity"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

func TestPopulationScalesDeterminismAndIdentity(t *testing.T) {
	key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	const seed = uint64(20261105)
	for _, count := range []int{10, 100, 1_000, 10_000} {
		count := count
		t.Run(fmt.Sprintf("population-%d", count), func(t *testing.T) {
			session := source.Session{ID: fmt.Sprintf("synthetic-population-%d", count), Mode: source.ModeMock}
			runtime.GC()
			var before runtime.MemStats
			runtime.ReadMemStats(&before)
			generationStart := time.Now()
			events, err := source.ScenarioEvents(session, seed, source.ScenarioMultipleParticipants, count)
			if err != nil {
				t.Fatal(err)
			}
			generationElapsed := time.Since(generationStart)
			if len(events) != count {
				t.Fatalf("events = %d, want %d", len(events), count)
			}
			again, err := source.ScenarioEvents(session, seed, source.ScenarioMultipleParticipants, count)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(events, again) {
				t.Fatal("generation is not deterministic")
			}

			metrics := NewMetrics()
			deduper, err := New(Config{
				Now:        func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) },
				Metrics:    metrics,
				MaxEntries: 20_000,
				MaxActors:  20_000,
			})
			if err != nil {
				t.Fatal(err)
			}
			viewers := make(map[string]struct{}, count)
			eventIDs := make(map[string]struct{}, count)
			actors := make(map[domain.ActorKey]struct{}, count)
			roles := map[domain.Role]int{}
			identityElapsed := time.Duration(0)
			parseElapsed := time.Duration(0)
			dedupeElapsed := time.Duration(0)
			for index, event := range events {
				metrics.ObserveReceived()
				if _, exists := viewers[event.ProviderViewerID]; exists {
					t.Fatalf("duplicate viewer at %d", index)
				}
				viewers[event.ProviderViewerID] = struct{}{}
				if _, exists := eventIDs[event.ProviderEventID]; exists {
					t.Fatalf("duplicate event id at %d", index)
				}
				eventIDs[event.ProviderEventID] = struct{}{}

				identityStart := time.Now()
				resolved, err := keys.Derive(identity.Input{
					Provider: event.Provider, ProviderVersion: "v1",
					BroadcasterKey: "village-w0307", ViewerID: event.ProviderViewerID,
				})
				identityElapsed += time.Since(identityStart)
				if err != nil {
					t.Fatalf("identity %d: %v", index, err)
				}
				if _, exists := actors[resolved.ActorKey]; exists {
					t.Fatalf("duplicate actor key at %d", index)
				}
				actors[resolved.ActorKey] = struct{}{}

				parseStart := time.Now()
				action, err := commands.ParseComment(event.Message)
				parseElapsed += time.Since(parseStart)
				if err != nil {
					t.Fatalf("parse %d: %v", index, err)
				}
				if action.Kind != commands.KindRole {
					t.Fatalf("event %d action = %+v", index, action)
				}
				roles[action.Role]++
				metrics.ObserveValidated()

				dedupeStart := time.Now()
				decision, reservation, err := deduper.Admit(event, resolved.ActorKey, action)
				dedupeElapsed += time.Since(dedupeStart)
				if err != nil || decision.Class != ClassAccepted {
					t.Fatalf("admit %d = %+v/%v", index, decision, err)
				}
				reservation.Finalize()
			}
			if len(viewers) != count || len(eventIDs) != count || len(actors) != count {
				t.Fatalf("unique viewers=%d events=%d actors=%d, want %d", len(viewers), len(eventIDs), len(actors), count)
			}
			wantBuilder := (count + 2) / 3
			wantGuard := (count + 1) / 3
			wantFighter := count / 3
			if roles[domain.RoleBuilder] != wantBuilder || roles[domain.RoleGuard] != wantGuard || roles[domain.RoleFighter] != wantFighter {
				t.Fatalf("roles = %+v, want builder=%d guard=%d fighter=%d", roles, wantBuilder, wantGuard, wantFighter)
			}
			stats := deduper.Stats()
			if stats.StableEntries != count || stats.Pending != 0 {
				t.Fatalf("dedupe stats = %+v", stats)
			}
			snapshot := metrics.Snapshot()
			if snapshot.InputReceived != int64(count) || snapshot.InputValidated != int64(count) ||
				snapshot.AcceptedCandidates != int64(count) {
				t.Fatalf("metrics = %+v", snapshot)
			}
			runtime.GC()
			var after runtime.MemStats
			runtime.ReadMemStats(&after)
			t.Logf("population=%d generation=%s identity=%s parse=%s dedupe=%s heap_after=%d sys_after=%d total_alloc=%d",
				count, generationElapsed.Round(time.Microsecond), identityElapsed.Round(time.Microsecond),
				parseElapsed.Round(time.Microsecond), dedupeElapsed.Round(time.Microsecond),
				after.HeapAlloc, after.Sys, after.TotalAlloc-before.TotalAlloc)
		})
	}
}

func TestPopulationMixedInputsClassification(t *testing.T) {
	const sessionID = "synthetic-mixed-w0307"
	events := make([]source.Event, 0, 10_000)
	add := func(id, viewer, message string, kind source.EventKind) {
		events = append(events, source.Event{
			Provider: "mock", SourceSession: sessionID, ProviderEventID: id,
			ProviderViewerID: viewer, Message: message, Kind: kind,
			ReceivedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		})
	}
	for index := 0; index < 4_000; index++ {
		add(fmt.Sprintf("mix-role-%04d", index), fmt.Sprintf("mix-viewer-role-%04d", index), []string{"XÂY", "THỦ", "ĐÁNH"}[index%3], source.KindComment)
	}
	for index := 0; index < 3_000; index++ {
		add(fmt.Sprintf("mix-vote-%04d", index), fmt.Sprintf("mix-viewer-vote-%04d", index), []string{"A", "B", "C"}[index%3], source.KindComment)
	}
	for index := 0; index < 1_000; index++ {
		add(fmt.Sprintf("mix-role-%04d", index), fmt.Sprintf("mix-viewer-role-%04d", index), []string{"XÂY", "THỦ", "ĐÁNH"}[index%3], source.KindComment)
	}
	for index := 0; index < 1_000; index++ {
		add(fmt.Sprintf("mix-invalid-%04d", index), fmt.Sprintf("mix-viewer-invalid-%04d", index), "XAY DANH", source.KindComment)
	}
	for index := 0; index < 500; index++ {
		add(fmt.Sprintf("mix-gift-%04d", index), fmt.Sprintf("mix-viewer-gift-%04d", index), "XÂY", source.KindGift)
	}
	for index := 0; index < 500; index++ {
		add("", "mix-viewer-repeat", "XÂY", source.KindComment)
	}
	if len(events) != 10_000 {
		t.Fatalf("events = %d", len(events))
	}
	key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	metrics := NewMetrics()
	deduper, err := New(Config{Now: func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }, Metrics: metrics, MaxEntries: 20_000, MaxActors: 20_000})
	if err != nil {
		t.Fatal(err)
	}
	viewers := make(map[string]struct{}, 10_000)
	eventIDs := make(map[string]struct{}, 10_000)
	actors := make(map[domain.ActorKey]struct{}, 10_000)
	session := source.Session{ID: sessionID, Mode: source.ModeMock}
	for index, event := range events {
		metrics.ObserveReceived()
		if event.ProviderEventID != "" {
			if err := source.ValidateEvent(event, session, source.ModeMock); err != nil {
				t.Fatalf("validate %d: %v", index, err)
			}
			metrics.ObserveValidated()
		}
		if event.ProviderViewerID != "" {
			viewers[event.ProviderViewerID] = struct{}{}
		}
		if event.ProviderEventID != "" {
			eventIDs[event.ProviderEventID] = struct{}{}
		}
		if event.Kind != source.KindComment {
			_, _, _ = deduper.Admit(event, "", commands.Action{})
			continue
		}
		resolved, err := keys.Derive(identity.Input{Provider: event.Provider, ProviderVersion: "v1", BroadcasterKey: "village-w0307", ViewerID: event.ProviderViewerID})
		if err != nil {
			t.Fatalf("identity %d: %v", index, err)
		}
		actors[resolved.ActorKey] = struct{}{}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			metrics.ObserveInvalidGrammar()
			continue
		}
		decision, reservation, err := deduper.Admit(event, resolved.ActorKey, action)
		if err != nil {
			t.Fatalf("admit %d: %v", index, err)
		}
		if reservation != nil {
			reservation.Finalize()
		}
		_ = decision
	}
	snapshot := metrics.Snapshot()
	if snapshot.InputReceived != 10_000 || snapshot.InputValidated != 9_500 ||
		snapshot.AcceptedCandidates != 7_001 || snapshot.DuplicateProviderID != 1_000 ||
		snapshot.RepeatNoop != 499 || snapshot.InvalidGrammar != 1_000 ||
		snapshot.IgnoredNonCommand != 500 || snapshot.RateLimited != 0 ||
		snapshot.InvalidIdentity != 0 || snapshot.CapacityExceeded != 0 ||
		snapshot.ConflictingProviderID != 0 {
		t.Fatalf("mixed metrics = %+v", snapshot)
	}
	if len(viewers) != 8_501 || len(eventIDs) != 8_500 || len(actors) != 8_001 {
		t.Fatalf("unique viewers=%d eventIDs=%d actors=%d", len(viewers), len(eventIDs), len(actors))
	}
}

func TestPopulationRepeatedActorDoesNotInflateIdentity(t *testing.T) {
	actorViewer := "mock-viewer-repeat-actor-w0307"
	key, err := identity.NewKey("v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.NewKeySet(key, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := keys.Derive(identity.Input{Provider: "mock", ProviderVersion: "v1", BroadcasterKey: "village-w0307", ViewerID: actorViewer})
	if err != nil {
		t.Fatal(err)
	}
	metrics := NewMetrics()
	deduper, err := New(Config{Now: func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }, Metrics: metrics, MaxEntries: 2_000, MaxActors: 2_000})
	if err != nil {
		t.Fatal(err)
	}
	actors := map[domain.ActorKey]struct{}{}
	accepted := 0
	for index := 0; index < 1_000; index++ {
		event := source.Event{
			Provider: "mock", SourceSession: "synthetic-repeated-w0307",
			ProviderEventID: fmt.Sprintf("repeat-%04d", index), ProviderViewerID: actorViewer,
			Message: "XÂY", Kind: source.KindComment,
			ReceivedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			t.Fatal(err)
		}
		decision, reservation, err := deduper.Admit(event, resolved.ActorKey, action)
		if err != nil {
			t.Fatal(err)
		}
		actors[resolved.ActorKey] = struct{}{}
		if decision.Class == ClassAccepted {
			accepted++
			reservation.Finalize()
		}
	}
	snapshot := metrics.Snapshot()
	if len(actors) != 1 || accepted != 10 || snapshot.AcceptedCandidates != 10 || snapshot.RateLimited != 990 {
		t.Fatalf("repeated actor: actors=%d accepted=%d metrics=%+v", len(actors), accepted, snapshot)
	}
}
