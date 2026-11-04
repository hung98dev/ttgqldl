package source_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/source"
)

type observation struct {
	Kind    commands.Kind
	Role    domain.Role
	Ballot  domain.VoteOption
	EventID string
}

func mockSession() source.Session {
	return source.Session{ID: "synthetic-session-001", Mode: source.ModeMock}
}

func replaySession() source.Session {
	return source.Session{ID: "synthetic-session-001", Mode: source.ModeReplay}
}

// bridge is the test-only pipeline: validate at the source boundary, parse exactly
// one typed action, and observe it. It never touches SessionState, XP, identity,
// durable events or the reducer.
func bridge(t *testing.T, adapter source.Source, session source.Session) ([]observation, []error) {
	t.Helper()
	var observations []observation
	var parseErrors []error
	if err := adapter.Start(context.Background(), session, func(_ context.Context, event source.Event) error {
		if event.Kind != source.KindComment {
			return nil
		}
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			parseErrors = append(parseErrors, err)
			return nil
		}
		observations = append(observations, observation{
			Kind:    action.Kind,
			Role:    action.Role,
			Ballot:  action.Ballot,
			EventID: event.ProviderEventID,
		})
		return nil
	}); err != nil {
		t.Fatalf("pipeline start: %v", err)
	}
	return observations, parseErrors
}

func TestMockParserBridgeProducesBuilder(t *testing.T) {
	config := source.MockConfig{Session: mockSession(), Seed: 1, Scenario: source.ScenarioSingleRegistration, Count: 1}
	mock, err := source.NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	observations, parseErrors := bridge(t, mock, config.Session)
	if len(parseErrors) != 0 {
		t.Fatalf("unexpected parser errors: %v", parseErrors)
	}
	if len(observations) != 1 || observations[0].Role != domain.RoleBuilder || observations[0].Kind != commands.KindRole {
		t.Fatalf("observations = %+v", observations)
	}
}

func TestReplayParserBridgeProducesGuardFighterAndBallot(t *testing.T) {
	replay, err := source.NewReplay(source.ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: "replay_valid.ndjson"})
	if err != nil {
		t.Fatal(err)
	}
	observations, parseErrors := bridge(t, replay, replaySession())
	if len(parseErrors) != 0 {
		t.Fatalf("unexpected parser errors: %v", parseErrors)
	}
	// Records: XÂY, THỦ, ĐÁNH, B, gift, join. Only comments become actions.
	if len(observations) != 4 {
		t.Fatalf("observations = %+v", observations)
	}
	if observations[0].Role != domain.RoleBuilder || observations[1].Role != domain.RoleGuard || observations[2].Role != domain.RoleFighter {
		t.Fatalf("role observations = %+v", observations)
	}
	if observations[3].Kind != commands.KindBallot || observations[3].Ballot != domain.VoteOptionB {
		t.Fatalf("ballot observation = %+v", observations[3])
	}
}

func TestEnglishAliasesMapIdentically(t *testing.T) {
	events := []source.Event{
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e1", Kind: source.KindComment, Message: "BUILD"},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e2", Kind: source.KindComment, Message: "DEFEND"},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e3", Kind: source.KindComment, Message: "ATTACK"},
	}
	mock, err := source.NewMockWithEvents(mockSession(), events)
	if err != nil {
		t.Fatal(err)
	}
	observations, _ := bridge(t, mock, mockSession())
	want := []domain.Role{domain.RoleBuilder, domain.RoleGuard, domain.RoleFighter}
	for index, role := range want {
		if observations[index].Role != role {
			t.Fatalf("alias %d role = %s, want %s", index, observations[index].Role, role)
		}
	}
}

func TestBallotsRemainDistinctActions(t *testing.T) {
	events := []source.Event{
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e1", Kind: source.KindComment, Message: "A"},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e2", Kind: source.KindComment, Message: "B"},
		{Provider: "mock", SourceSession: mockSession().ID, ProviderEventID: "e3", Kind: source.KindComment, Message: "C"},
	}
	mock, err := source.NewMockWithEvents(mockSession(), events)
	if err != nil {
		t.Fatal(err)
	}
	observations, _ := bridge(t, mock, mockSession())
	if len(observations) != 3 {
		t.Fatalf("observations = %+v", observations)
	}
	for index, want := range []domain.VoteOption{domain.VoteOptionA, domain.VoteOptionB, domain.VoteOptionC} {
		if observations[index].Kind != commands.KindBallot || observations[index].Ballot != want {
			t.Fatalf("ballot %d = %+v", index, observations[index])
		}
	}
}

func TestInvalidGrammarProducesNoAction(t *testing.T) {
	replay, err := source.NewReplay(source.ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: "replay_invalid_grammar.ndjson"})
	if err != nil {
		t.Fatal(err)
	}
	observations, parseErrors := bridge(t, replay, replaySession())
	// Only the final "A" is a command; sentence, punctuation, Cyrillic lookalike and
	// the invisible-mark spoof produce no action.
	if len(observations) != 1 || observations[0].Ballot != domain.VoteOptionA {
		t.Fatalf("observations = %+v", observations)
	}
	if len(parseErrors) != 4 {
		t.Fatalf("parse errors = %d, want 4", len(parseErrors))
	}
	for _, err := range parseErrors {
		if !errors.Is(err, commands.ErrNotACommand) && !errors.Is(err, commands.ErrForbiddenCharacter) {
			t.Fatalf("unexpected parser error category: %v", err)
		}
	}
}

func TestSourceRejectsSpoofedAndOversizedCommentsBeforeParser(t *testing.T) {
	for _, fixture := range []string{"replay_control.ndjson", "replay_oversize_message.ndjson"} {
		replay, err := source.NewReplay(source.ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: fixture})
		if err != nil {
			t.Fatal(err)
		}
		calls := 0
		err = replay.Start(context.Background(), replaySession(), func(context.Context, source.Event) error {
			calls++
			return nil
		})
		if !errors.Is(err, source.ErrInvalidMessage) {
			t.Fatalf("%s error = %v, want ErrInvalidMessage", fixture, err)
		}
		if calls != 0 {
			t.Fatalf("%s reached the parser bridge %d times", fixture, calls)
		}
	}
}

func TestNoRawProviderIdentifiersInPublicOutput(t *testing.T) {
	replay, err := source.NewReplay(source.ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: "replay_malformed.ndjson"})
	if err != nil {
		t.Fatal(err)
	}
	err = replay.Start(context.Background(), replaySession(), func(context.Context, source.Event) error { return nil })
	if err == nil {
		t.Fatal("malformed fixture was accepted")
	}
	message := err.Error()
	for _, forbidden := range []string{"viewer-0001", "event-0001", "synthetic-session-001"} {
		if strings.Contains(message, forbidden) {
			t.Fatalf("public error leaked %q: %s", forbidden, message)
		}
	}
	health := replay.Health(context.Background())
	if strings.Contains(health.LastError, "viewer") || strings.Contains(health.LastError, "event-") {
		t.Fatalf("health leaked a provider identifier: %q", health.LastError)
	}
	if health.LastError != "malformed_record" {
		t.Fatalf("health category = %q", health.LastError)
	}
}

func TestReplayTwiceYieldsIdenticalObservations(t *testing.T) {
	run := func() []observation {
		replay, err := source.NewReplay(source.ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: "replay_unicode.ndjson"})
		if err != nil {
			t.Fatal(err)
		}
		observations, _ := bridge(t, replay, replaySession())
		return observations
	}
	first := run()
	second := run()
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("replay observations differ:\n%+v\n%+v", first, second)
	}
}

func TestReconnectResumesCorrectRemainingEvents(t *testing.T) {
	config := source.MockConfig{Session: mockSession(), Seed: 3, Scenario: source.ScenarioMultipleParticipants, Count: 6}
	all, err := source.ScenarioEvents(config.Session, config.Seed, config.Scenario, config.Count)
	if err != nil {
		t.Fatal(err)
	}
	mock, err := source.NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.Start(context.Background(), config.Session, func(context.Context, source.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	var observations []observation
	if err := mock.Reconnect(context.Background(), source.NewCursor(config.Session.ID, mock.SourceID(), 2), func(_ context.Context, event source.Event) error {
		action, err := commands.ParseComment(event.Message)
		if err != nil {
			t.Fatalf("resumed event rejected: %v", err)
		}
		observations = append(observations, observation{Kind: action.Kind, Role: action.Role, Ballot: action.Ballot})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(observations) != len(all)-2 {
		t.Fatalf("resumed %d observations, want %d", len(observations), len(all)-2)
	}
}

func TestDuplicateProviderEventIDsStayObservable(t *testing.T) {
	replay, err := source.NewReplay(source.ReplayConfig{Session: replaySession(), Directory: "testdata", Fixture: "replay_duplicates.ndjson"})
	if err != nil {
		t.Fatal(err)
	}
	observations, _ := bridge(t, replay, replaySession())
	if len(observations) != 4 {
		t.Fatalf("observations = %+v", observations)
	}
	if observations[0].EventID != observations[1].EventID || observations[2].EventID != observations[3].EventID {
		t.Fatalf("duplicate ids not observable: %+v", observations)
	}
}

func TestNonCommentEventsNeverBecomeActions(t *testing.T) {
	config := source.MockConfig{Session: mockSession(), Seed: 4, Scenario: source.ScenarioNonCommentEvents, Count: 10}
	mock, err := source.NewMock(config)
	if err != nil {
		t.Fatal(err)
	}
	observations, parseErrors := bridge(t, mock, config.Session)
	if len(observations) != 0 || len(parseErrors) != 0 {
		t.Fatalf("non-comment events produced observations=%d errors=%d", len(observations), len(parseErrors))
	}
}
