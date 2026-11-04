package commands

import (
	"errors"
	"testing"
	"unicode"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// TestInvisibleDefaultIgnorableMarksRejected is the regression for the confirmed
// pre-fix bypass: the parser used to erase every combining mark, so default-ignorable
// marks such as CGJ and variation selectors could be appended to a command without
// invalidating it. The pre-fix results were recorded as accepted for all six inputs;
// after the fix every one is rejected explicitly.
func TestInvisibleDefaultIgnorableMarksRejected(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		mark rune
	}{
		{"combining grapheme joiner", "XAY\u034F", '\u034F'},
		{"variation selector 16", "A\uFE0F", '\uFE0F'},
		{"mongolian free variation selector", "THU\u180B", '\u180B'},
		{"variation selector 1", "XAY\uFE00", '\uFE00'},
		{"supplementary variation selector 17", "XAY\U000E0100", '\U000E0100'},
		{"supplementary variation selector 256", "XAY\U000E01EF", '\U000E01EF'},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if !unicode.Is(unicode.Mn, testCase.mark) {
				t.Fatalf("%U is not classified Mn; the regression fixture is invalid", testCase.mark)
			}
			action, err := ParseComment(testCase.raw)
			if !errors.Is(err, ErrForbiddenCharacter) {
				t.Fatalf("invisible mark input was not rejected: action=%+v err=%v", action, err)
			}
			if action != (Action{}) {
				t.Fatalf("rejection returned a non-zero action: %+v", action)
			}
		})
	}

	// Legitimate decomposed Vietnamese marks remain accepted.
	for _, raw := range []string{
		"xa\u0302y",  // x + circumflex + y
		"thu\u0309",  // thu + hook above
		"đa\u0301nh", // đánh with decomposed acute
		"XÂY",        // precomposed
	} {
		action, err := ParseComment(raw)
		if err != nil {
			t.Fatalf("legitimate Vietnamese variant %q rejected: %v", raw, err)
		}
		if !action.Valid() {
			t.Fatalf("legitimate Vietnamese variant %q produced %+v", raw, action)
		}
	}
	// A legitimate horned letter is valid text, just not a command; it must fail the
	// grammar, not the spoofing policy.
	if _, err := ParseComment("o\u031B"); !errors.Is(err, ErrNotACommand) {
		t.Fatalf("bare horned o error = %v, want ErrNotACommand", err)
	}
}

// TestActionValidRejectsMalformedTaggedUnion pins the corrected tagged-union
// contract: a ballot action with a populated role was previously reported valid.
func TestActionValidRejectsMalformedTaggedUnion(t *testing.T) {
	cases := []struct {
		name   string
		action Action
		want   bool
	}{
		{"valid role", Action{Kind: KindRole, Role: domain.RoleBuilder}, true},
		{"valid guard", Action{Kind: KindRole, Role: domain.RoleGuard}, true},
		{"valid ballot", Action{Kind: KindBallot, Ballot: domain.VoteOptionA}, true},
		{"ballot with role populated", Action{Kind: KindBallot, Role: domain.RoleFighter, Ballot: domain.VoteOptionA}, false},
		{"role with ballot populated", Action{Kind: KindRole, Role: domain.RoleBuilder, Ballot: domain.VoteOptionA}, false},
		{"invalid role value", Action{Kind: KindRole, Role: domain.Role(9)}, false},
		{"invalid ballot value", Action{Kind: KindBallot, Ballot: domain.VoteOption(9)}, false},
		{"ballot none", Action{Kind: KindBallot, Ballot: domain.VoteOptionNone}, false},
		{"role kind without role", Action{Kind: KindRole}, false},
		{"unknown kind", Action{Kind: Kind(9), Role: domain.RoleBuilder}, false},
		{"zero action", Action{}, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.action.Valid(); got != testCase.want {
				t.Fatalf("Valid() = %t, want %t for %+v", got, testCase.want, testCase.action)
			}
		})
	}
}
