package source

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func collectSink(events *[]Event) Sink {
	return func(_ context.Context, event Event) error {
		*events = append(*events, event)
		return nil
	}
}

func mockConfig(t *testing.T, scenario Scenario, count int) MockConfig {
	t.Helper()
	return MockConfig{Session: testSession(ModeMock), Seed: 42, Scenario: scenario, Count: count}
}

func TestScenarioGeneratorIsDeterministic(t *testing.T) {
	session := testSession(ModeMock)
	scenarios := []Scenario{
		ScenarioSingleRegistration, ScenarioMultipleParticipants, ScenarioRepeatedComments,
		ScenarioRoleChanges, ScenarioVotes, ScenarioInvalidComments, ScenarioUnicodeVariants,
		ScenarioDuplicateEventIDs, ScenarioOutOfOrderUpstream, ScenarioNonCommentEvents,
		ScenarioEmpty, ScenarioFiniteScript,
	}
	for _, scenario := range scenarios {
		t.Run(string(scenario), func(t *testing.T) {
			first, err := ScenarioEvents(session, 7, scenario, 5)
			if err != nil {
				t.Fatal(err)
			}
			second, err := ScenarioEvents(session, 7, scenario, 5)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("identical configuration produced different events")
			}
			for index := range first {
				if err := ValidateEvent(first[index], session, ModeMock); err != nil {
					t.Fatalf("generated event %d invalid: %v", index, err)
				}
				if first[index].Provider != "mock" {
					t.Fatalf("event provider = %q", first[index].Provider)
				}
			}
		})
	}
	if _, err := ScenarioEvents(session, 1, "unknown", 1); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("unknown scenario error = %v", err)
	}
}

func TestMockSeedsProduceReproduciblyDifferentScripts(t *testing.T) {
	session := testSession(ModeMock)
	first, err := ScenarioEvents(session, 1, ScenarioMultipleParticipants, 4)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ScenarioEvents(session, 2, ScenarioMultipleParticipants, 4)
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(first, second) {
		t.Fatal("different seeds produced identical scripts")
	}
	again, err := ScenarioEvents(session, 2, ScenarioMultipleParticipants, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(second, again) {
		t.Fatal("seed 2 was not reproducible")
	}
}

func TestMockStartDeliversUntilEOF(t *testing.T) {
	config := mockConfig(t, ScenarioMultipleParticipants, 5)
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	var events []Event
	if err := mock.Start(context.Background(), config.Session, collectSink(&events)); err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("delivered %d events", len(events))
	}
	health := mock.Health(context.Background())
	if health.Status != StatusEOF || health.Delivered != 5 {
		t.Fatalf("health = %+v", health)
	}
	wantCursor := NewCursor(config.Session.ID, mock.SourceID(), 5)
	if health.Cursor != wantCursor {
		t.Fatalf("cursor = %+v, want %+v", health.Cursor, wantCursor)
	}
	if health.LastError != "" {
		t.Fatalf("unexpected last error %q", health.LastError)
	}
	capabilities := mock.Capabilities(context.Background())
	if !capabilities.StableViewerID || !capabilities.StableEventID || !capabilities.ReplayCursor || !capabilities.DeliveryOrder {
		t.Fatalf("mock capabilities = %+v", capabilities)
	}
	if capabilities.TimestampReliability != TimestampSyntheticDeterministic {
		t.Fatalf("timestamp reliability = %s", capabilities.TimestampReliability)
	}
}

func TestMockEmptyAndSingleScenarios(t *testing.T) {
	for _, testCase := range []struct {
		scenario Scenario
		count    int
		want     int
	}{
		{ScenarioEmpty, 5, 0},
		{ScenarioSingleRegistration, 5, 1},
		{ScenarioFiniteScript, 3, 3},
	} {
		config := mockConfig(t, testCase.scenario, testCase.count)
		mock, err := NewMock(config)
		if err != nil {
			t.Fatal(err)
		}
		var events []Event
		if err := mock.Start(context.Background(), config.Session, collectSink(&events)); err != nil {
			t.Fatal(err)
		}
		if len(events) != testCase.want {
			t.Fatalf("%s delivered %d, want %d", testCase.scenario, len(events), testCase.want)
		}
		if health := mock.Health(context.Background()); health.Status != StatusEOF {
			t.Fatalf("%s status = %s", testCase.scenario, health.Status)
		}
	}
}

func TestMockSessionMismatchAndConfigValidation(t *testing.T) {
	config := mockConfig(t, ScenarioSingleRegistration, 1)
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	other := Session{ID: "synthetic-session-002", Mode: ModeMock}
	if err := mock.Start(context.Background(), other, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("session mismatch error = %v", err)
	}
	if _, err := NewMock(MockConfig{Session: testSession(ModeReplay), Scenario: ScenarioEmpty}); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("wrong mode error = %v", err)
	}
	if _, err := NewMock(MockConfig{Session: testSession(ModeMock), Scenario: ScenarioEmpty, Count: -1}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("negative count error = %v", err)
	}
	if _, err := NewMock(MockConfig{Session: testSession(ModeMock), Scenario: ScenarioEmpty, Count: MaxMockEvents + 1}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("oversized count error = %v", err)
	}
	if _, err := NewMockWithEvents(testSession(ModeMock), []Event{{Provider: "mock", Kind: KindComment, Message: "\xff"}}); err == nil {
		t.Fatal("invalid scripted event accepted")
	}
}

func TestMockLifecycleStartOnceAndStopBeforeStart(t *testing.T) {
	config := mockConfig(t, ScenarioFiniteScript, 2)
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.Stop(context.Background()); err != nil {
		t.Fatalf("stop before start: %v", err)
	}
	if health := mock.Health(context.Background()); health.Status != StatusStopped {
		t.Fatalf("status after stop before start = %s", health.Status)
	}
	if err := mock.Start(context.Background(), config.Session, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("start after stop error = %v", err)
	}

	second, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Start(context.Background(), config.Session, func(context.Context, Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := second.Start(context.Background(), config.Session, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("repeated start error = %v", err)
	}
	if err := second.Stop(context.Background()); err != nil {
		t.Fatalf("repeated stop: %v", err)
	}
}

func TestMockReconnectResumesExactRemainingScript(t *testing.T) {
	config := mockConfig(t, ScenarioVotes, 6)
	all, err := ScenarioEvents(config.Session, config.Seed, config.Scenario, config.Count)
	if err != nil {
		t.Fatal(err)
	}
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	// Reconnect is only legal after a terminal state; run once to EOF first.
	if err := mock.Start(context.Background(), config.Session, func(context.Context, Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	var events []Event
	cursor := NewCursor(config.Session.ID, mock.SourceID(), 2)
	if err := mock.Reconnect(context.Background(), cursor, collectSink(&events)); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	if !reflect.DeepEqual(events, all[2:]) {
		t.Fatalf("resumed events = %d, want %d", len(events), len(all)-2)
	}
	if health := mock.Health(context.Background()); health.Delivered != 10 || health.Status != StatusEOF {
		t.Fatalf("health after reconnect = %+v", health)
	}

	var atEnd []Event
	if err := mock.Reconnect(context.Background(), NewCursor(config.Session.ID, mock.SourceID(), uint64(len(all))), collectSink(&atEnd)); err != nil {
		t.Fatalf("reconnect at EOF: %v", err)
	}
	if len(atEnd) != 0 {
		t.Fatalf("reconnect at EOF delivered %d events", len(atEnd))
	}

	if err := mock.Reconnect(context.Background(), Cursor{Version: 1, Session: config.Session.ID, SourceID: "foreign", NextEvent: 0}, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrCursorMismatch) {
		t.Fatalf("foreign cursor error = %v", err)
	}
	otherSession := NewCursor("synthetic-session-999", mock.SourceID(), 0)
	if err := mock.Reconnect(context.Background(), otherSession, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrCursorMismatch) {
		t.Fatalf("foreign session cursor error = %v", err)
	}
	badVersion := NewCursor(config.Session.ID, mock.SourceID(), 0)
	badVersion.Version = 9
	if err := mock.Reconnect(context.Background(), badVersion, func(context.Context, Event) error { return nil }); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("bad cursor version error = %v", err)
	}
	if err := mock.Reconnect(context.Background(), NewCursor(config.Session.ID, mock.SourceID(), uint64(len(all)+1)), func(context.Context, Event) error { return nil }); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("out-of-range cursor error = %v", err)
	}

	fresh, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := fresh.Reconnect(context.Background(), NewCursor(config.Session.ID, fresh.SourceID(), 0), func(context.Context, Event) error { return nil }); !errors.Is(err, ErrNotReconnectable) {
		t.Fatalf("reconnect before start error = %v", err)
	}
}

func TestMockCallbackFailureKeepsCursor(t *testing.T) {
	config := mockConfig(t, ScenarioMultipleParticipants, 5)
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	delivered := 0
	failing := func(_ context.Context, _ Event) error {
		delivered++
		if delivered == 3 {
			return errors.New("reject third event")
		}
		return nil
	}
	if err := mock.Start(context.Background(), config.Session, failing); !errors.Is(err, ErrDeliveryRejected) {
		t.Fatalf("delivery failure error = %v", err)
	}
	health := mock.Health(context.Background())
	if health.Status != StatusDisconnected || health.Delivered != 2 || health.Cursor.NextEvent != 2 {
		t.Fatalf("health after callback failure = %+v", health)
	}
	if health.LastError != "delivery_rejected" {
		t.Fatalf("health category = %q", health.LastError)
	}

	var resumed []Event
	if err := mock.Reconnect(context.Background(), health.Cursor, collectSink(&resumed)); err != nil {
		t.Fatalf("reconnect after callback failure: %v", err)
	}
	if len(resumed) != 3 {
		t.Fatalf("resumed %d events after failure, want 3", len(resumed))
	}
}

func TestMockCancellationDuringBlockedCallback(t *testing.T) {
	config := mockConfig(t, ScenarioFiniteScript, 4)
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	delivered := 0
	sink := func(sinkCtx context.Context, _ Event) error {
		delivered++
		cancel()
		<-sinkCtx.Done()
		return sinkCtx.Err()
	}
	err = mock.Start(ctx, config.Session, sink)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
	health := mock.Health(context.Background())
	if health.Status != StatusDisconnected || health.Delivered != 0 {
		t.Fatalf("health after cancellation = %+v", health)
	}
	if delivered != 1 {
		t.Fatalf("sink calls = %d", delivered)
	}
}

func TestMockStopDuringBlockedCallback(t *testing.T) {
	config := mockConfig(t, ScenarioFiniteScript, 10)
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	var once sync.Once
	var calls atomic.Int64
	sink := func(ctx context.Context, _ Event) error {
		calls.Add(1)
		once.Do(func() { close(entered) })
		<-ctx.Done()
		return ctx.Err()
	}
	done := make(chan error, 1)
	go func() { done <- mock.Start(context.Background(), config.Session, sink) }()
	<-entered
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := mock.Stop(stopCtx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("start after stop = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("start did not return after stop")
	}
	callsAfterStop := calls.Load()
	time.Sleep(10 * time.Millisecond) // allow any stray callback to surface
	if calls.Load() != callsAfterStop {
		t.Fatal("callback ran after confirmed stop")
	}
	health := mock.Health(context.Background())
	if health.Status != StatusStopped {
		t.Fatalf("status after stop = %s", health.Status)
	}
	if err := mock.Stop(context.Background()); err != nil {
		t.Fatalf("repeated stop: %v", err)
	}
}

func TestMockHealthIsConcurrencySafe(t *testing.T) {
	config := mockConfig(t, ScenarioMultipleParticipants, 200)
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- mock.Start(context.Background(), config.Session, func(context.Context, Event) error { return nil })
	}()
	var wait sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := 0; index < 100; index++ {
				_ = mock.Health(context.Background())
				_ = mock.Capabilities(context.Background())
			}
		}()
	}
	wait.Wait()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
