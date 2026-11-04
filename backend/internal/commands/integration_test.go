package commands_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/simulation"
)

// TestParsedRoleActionsMatchDomainEnums proves the parser returns the exact
// existing domain role values, with no duplicated game-rule definitions.
func TestParsedRoleActionsMatchDomainEnums(t *testing.T) {
	cases := []struct {
		raw  string
		want domain.Role
	}{
		{"XAY", domain.RoleBuilder},
		{"XÂY", domain.RoleBuilder},
		{"BUILD", domain.RoleBuilder},
		{"THU", domain.RoleGuard},
		{"THỦ", domain.RoleGuard},
		{"DEFEND", domain.RoleGuard},
		{"DANH", domain.RoleFighter},
		{"ĐÁNH", domain.RoleFighter},
		{"ATTACK", domain.RoleFighter},
	}
	for _, testCase := range cases {
		action, err := commands.ParseComment(testCase.raw)
		if err != nil {
			t.Fatalf("%q: %v", testCase.raw, err)
		}
		if action.Kind != commands.KindRole {
			t.Fatalf("%q kind = %s", testCase.raw, action.Kind)
		}
		if action.Role != testCase.want {
			t.Fatalf("%q role = %s, want %s", testCase.raw, action.Role, testCase.want)
		}
		// Type compatibility with the reducer command shape: the parsed value drops
		// straight into the existing field without conversion or reinterpretation.
		command := simulation.RoleCommand{Role: action.Role}
		if !command.Role.Valid() {
			t.Fatalf("%q produced an invalid simulation role", testCase.raw)
		}
	}
}

// TestParsedBallotActionsMatchDomainEnums proves A/B/C map to the existing vote
// options and are never classified as roles.
func TestParsedBallotActionsMatchDomainEnums(t *testing.T) {
	cases := []struct {
		raw  string
		want domain.VoteOption
	}{
		{"A", domain.VoteOptionA},
		{"a", domain.VoteOptionA},
		{"Ｂ", domain.VoteOptionB},
		{"c", domain.VoteOptionC},
	}
	for _, testCase := range cases {
		action, err := commands.ParseComment(testCase.raw)
		if err != nil {
			t.Fatalf("%q: %v", testCase.raw, err)
		}
		if action.Kind != commands.KindBallot || action.IsRole() {
			t.Fatalf("%q was not classified as a ballot: %+v", testCase.raw, action)
		}
		if action.Ballot != testCase.want {
			t.Fatalf("%q ballot = %s, want %s", testCase.raw, action.Ballot, testCase.want)
		}
		command := simulation.VoteCommand{Option: action.Ballot}
		if !command.Option.Valid() || command.Option == domain.VoteOptionNone {
			t.Fatalf("%q produced an unusable simulation ballot", testCase.raw)
		}
	}
}

// TestRejectedInputCannotBecomeACommand proves rejections yield the zero action,
// which cannot be mistaken for a role or ballot by the simulation types.
func TestRejectedInputCannotBecomeACommand(t *testing.T) {
	invalid := []string{
		"XAY DANH", "XAY!", "TOI MUON XAY", "ХАУ", "XAY\u200B", "\u202EXAY",
		"\xff", "", "ATTACK NOW", "AA", "123XAY",
	}
	for _, raw := range invalid {
		action, err := commands.ParseComment(raw)
		if err == nil {
			t.Fatalf("%q was accepted", raw)
		}
		if action.Kind != commands.KindNone || action.Valid() || action.IsRole() || action.IsBallot() {
			t.Fatalf("%q returned a usable action: %+v", raw, action)
		}
		if !errors.Is(err, commands.ErrNotACommand) &&
			!errors.Is(err, commands.ErrForbiddenCharacter) &&
			!errors.Is(err, commands.ErrInvalidUTF8) &&
			!errors.Is(err, commands.ErrEmpty) {
			t.Fatalf("%q returned an unclassified error: %v", raw, err)
		}
		command := simulation.RoleCommand{Role: action.Role}
		if command.Role.Valid() {
			t.Fatalf("%q produced a valid role through the zero action", raw)
		}
	}
}

// TestParserIsPureAndThreadSafe proves the parser has no shared mutable state:
// concurrent parses of the same inputs always produce the same action or the same
// rejection category as the serial baseline.
func TestParserIsPureAndThreadSafe(t *testing.T) {
	inputs := []string{"XÂY", "THỦ", "ĐÁNH", "BUILD", "a", "C", "XAY DANH", "XAY!", "\xff", "ХАУ"}
	type result struct {
		action commands.Action
		err    error
	}
	baseline := make([]result, len(inputs))
	for index, raw := range inputs {
		baseline[index].action, baseline[index].err = commands.ParseComment(raw)
	}
	const workers = 32
	failures := make(chan string, workers)
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index, raw := range inputs {
				action, err := commands.ParseComment(raw)
				if action != baseline[index].action || (err == nil) != (baseline[index].err == nil) {
					failures <- "concurrent parse diverged for " + raw
					return
				}
			}
		}()
	}
	wait.Wait()
	close(failures)
	for failure := range failures {
		t.Fatal(failure)
	}
}
