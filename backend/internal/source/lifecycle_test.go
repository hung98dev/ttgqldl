package source

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConcurrentStopAndHealth(t *testing.T) {
	config := mockConfig(t, ScenarioMultipleParticipants, 500)
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	var enteredOnce bool
	sink := func(ctx context.Context, _ Event) error {
		if !enteredOnce {
			enteredOnce = true
			close(entered)
		}
		<-ctx.Done()
		return ctx.Err()
	}
	done := make(chan error, 1)
	go func() { done <- mock.Start(context.Background(), config.Session, sink) }()
	<-entered

	stopDone := make(chan error, 1)
	go func() { stopDone <- mock.Stop(context.Background()) }()
	var health Health
	for index := 0; index < 100; index++ {
		health = mock.Health(context.Background())
	}
	if health.Status != StatusNew && health.Status != StatusRunning && health.Status != StatusStopped {
		t.Fatalf("unexpected concurrent health status %s", health.Status)
	}
	select {
	case err := <-stopDone:
		if err != nil {
			t.Fatalf("stop: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent stop did not complete")
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("start: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("start did not return after concurrent stop")
	}
	if health = mock.Health(context.Background()); health.Status != StatusStopped {
		t.Fatalf("final status = %s", health.Status)
	}
}

func TestReconnectAfterCancellation(t *testing.T) {
	config := mockConfig(t, ScenarioVotes, 4)
	mock, err := NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err = mock.Start(ctx, config.Session, func(sinkCtx context.Context, _ Event) error {
		calls++
		if calls == 2 {
			cancel()
			<-sinkCtx.Done()
			return sinkCtx.Err()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
	health := mock.Health(context.Background())
	if health.Status != StatusDisconnected || health.Delivered != 1 {
		t.Fatalf("health = %+v", health)
	}
	var resumed []Event
	if err := mock.Reconnect(context.Background(), health.Cursor, collectSink(&resumed)); err != nil {
		t.Fatalf("reconnect after cancellation: %v", err)
	}
	if len(resumed) != 3 {
		t.Fatalf("resumed %d events", len(resumed))
	}
}

func TestReplayRejectsDuplicateSequenceAndMissingSession(t *testing.T) {
	duplicate := tempFixture(t, []byte(
		`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","event_kind":"comment","message":"XÂY"}`+"\n"+
			`{"schema_version":1,"source_session":"synthetic-session-001","event_seq":1,"provider":"replay","event_kind":"comment","message":"THỦ"}`+"\n"), nil)
	var events []Event
	err := duplicate.Start(context.Background(), replaySession(), collectSink(&events))
	if !errors.Is(err, ErrMalformedRecord) {
		t.Fatalf("duplicate sequence error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("delivered %d events before duplicate", len(events))
	}

	missingSession := tempFixture(t, []byte(
		`{"schema_version":1,"event_seq":1,"provider":"replay","event_kind":"comment","message":"XÂY"}`+"\n"), nil)
	err = missingSession.Start(context.Background(), replaySession(), func(context.Context, Event) error { return nil })
	if !errors.Is(err, ErrSessionMismatch) {
		t.Fatalf("missing session error = %v", err)
	}
}
