package commands

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// FuzzParseComment asserts the parser's safety properties for arbitrary bytes:
// no panic, deterministic output, a typed allowlisted action on success, a zero
// action on every rejection, stable normalization and enforced input bounds.
func FuzzParseComment(f *testing.F) {
	seeds := []string{
		"XAY", "XÂY", "xây", "THU", "THỦ", "DANH", "ĐÁNH", "BUILD", "DEFEND", "ATTACK",
		"A", "B", "C", "a", " XÂY ", "\tTHỦ\n", "\u00A0ĐÁNH\u00A0",
		"XAY DANH", "XAY!", "!XAY", "XAY?", "/XAY", "#XAY", "123XAY", "XAY123",
		"X Â Y", "B U I L D", "ATTACK NOW", "ХАУ", "Α", "Ａ", "１",
		"XAY\u200B", "\u202EXAY", "\u2066XAY\u2069", "XAY\uFEFF", "\u0301XAY",
		"XAY\u034F", "A\uFE0F", "THU\u180B", "XAY\uFE00", "XAY\U000E0100", "XAY\U000E01EF",
		"XAY" + "\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301\u0301",
		"\xff", "XAY\xff", "\x80", "\xed\xa0\x80",
		strings.Repeat("X", 500), strings.Repeat("X", 501),
		"\uFF38\uFF21\uFF39", "XÂY\u0301", "X\u200DY", "XÂY\x00", "XÂY THỦ", "XÂY\nTHỦ",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		action, err := ParseComment(raw)

		switch {
		case len(raw) == 0:
			if !errors.Is(err, ErrEmpty) {
				t.Fatalf("empty input error = %v, want ErrEmpty", err)
			}
		case len(raw) > MaxCommentBytes:
			if !errors.Is(err, ErrTooLong) {
				t.Fatalf("oversized (%d bytes) error = %v, want ErrTooLong", len(raw), err)
			}
		case !utf8.ValidString(raw):
			if !errors.Is(err, ErrInvalidUTF8) {
				t.Fatalf("invalid UTF-8 error = %v, want ErrInvalidUTF8", err)
			}
		}
		if err != nil {
			if action != (Action{}) {
				t.Fatalf("rejected input returned action %+v", action)
			}
			if len(raw) <= MaxCommentBytes && utf8.ValidString(raw) && len(raw) != 0 {
				// Deterministic rejection category for the same bytes.
				againAction, againErr := ParseComment(raw)
				if againAction != (Action{}) || !errors.Is(againErr, err) {
					t.Fatalf("non-deterministic rejection: %v then %v", err, againErr)
				}
			}
			return
		}

		if len(raw) > MaxCommentBytes {
			t.Fatalf("accepted input above the byte limit: %d", len(raw))
		}
		if !action.Valid() {
			t.Fatalf("accepted input produced an invalid action: %+v", action)
		}
		token := actionToken(action)
		if token == "" {
			t.Fatalf("accepted action has no canonical token: %+v", action)
		}
		normalized, nerr := normalizeComment(token)
		if nerr != nil || normalized != token {
			t.Fatalf("canonical token %q is not stable: %q, %v", token, normalized, nerr)
		}
		againAction, againErr := ParseComment(raw)
		if againErr != nil || againAction != action {
			t.Fatalf("non-deterministic acceptance: %+v/%v then %+v/%v", action, err, againAction, againErr)
		}
		reparsed, rerr := ParseComment(token)
		if rerr != nil || reparsed != action {
			t.Fatalf("canonical token %q does not re-parse to the same action", token)
		}
	})
}

// actionToken returns the canonical token for an action, or "" when the action is
// not a recognized command. It is test-only and deliberately exhaustive.
func actionToken(action Action) string {
	switch action.Kind {
	case KindRole:
		switch action.Role {
		case domain.RoleBuilder:
			return "xay"
		case domain.RoleGuard:
			return "thu"
		case domain.RoleFighter:
			return "danh"
		}
	case KindBallot:
		switch action.Ballot {
		case domain.VoteOptionA:
			return "a"
		case domain.VoteOptionB:
			return "b"
		case domain.VoteOptionC:
			return "c"
		}
	}
	return ""
}
