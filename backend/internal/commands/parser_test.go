package commands

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// grammarFixture is the frozen cross-platform contract: exact UTF-8 bytes mapped to
// the expected canonical token, typed action or stable error category.
type grammarFixture struct {
	RulesVersion           string `json:"rules_version"`
	CanonicalSchemaVersion int    `json:"canonical_schema_version"`
	Comment                string `json:"comment"`
	Positive               []struct {
		RawHex string `json:"raw_hex"`
		Token  string `json:"token"`
		Kind   string `json:"kind"`
		Role   string `json:"role"`
		Ballot string `json:"ballot"`
	} `json:"positive"`
	Negative []struct {
		RawHex string `json:"raw_hex"`
		Error  string `json:"error"`
	} `json:"negative"`
}

func loadGrammarFixture(t *testing.T) grammarFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/grammar_fixtures.json")
	if err != nil {
		t.Fatalf("read grammar fixtures: %v", err)
	}
	var fixture grammarFixture
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatalf("parse grammar fixtures: %v", err)
	}
	if fixture.RulesVersion != domain.RulesVersion {
		t.Fatalf("fixture rules_version = %q", fixture.RulesVersion)
	}
	if fixture.CanonicalSchemaVersion != domain.CanonicalSchemaVersion {
		t.Fatalf("fixture canonical_schema_version = %d", fixture.CanonicalSchemaVersion)
	}
	return fixture
}

func decodeRaw(t *testing.T, rawHex string) string {
	t.Helper()
	decoded, err := hex.DecodeString(rawHex)
	if err != nil {
		t.Fatalf("fixture raw_hex %q: %v", rawHex, err)
	}
	return string(decoded)
}

func fixtureError(name string) error {
	switch name {
	case "ErrEmpty":
		return ErrEmpty
	case "ErrTooLong":
		return ErrTooLong
	case "ErrInvalidUTF8":
		return ErrInvalidUTF8
	case "ErrForbiddenCharacter":
		return ErrForbiddenCharacter
	case "ErrExcessiveCombining":
		return ErrExcessiveCombining
	case "ErrNotACommand":
		return ErrNotACommand
	default:
		return nil
	}
}

func fixtureBallot(name string) domain.VoteOption {
	switch name {
	case "A":
		return domain.VoteOptionA
	case "B":
		return domain.VoteOptionB
	case "C":
		return domain.VoteOptionC
	default:
		return domain.VoteOptionNone
	}
}

// TestFrozenGrammarFixtures is the cross-platform determinism gate: the exact
// bytes, tokens, actions and error categories are frozen in testdata and must be
// identical on Windows and Linux.
func TestFrozenGrammarFixtures(t *testing.T) {
	fixture := loadGrammarFixture(t)
	if len(fixture.Positive) == 0 || len(fixture.Negative) == 0 {
		t.Fatal("grammar fixtures must contain positive and negative cases")
	}
	t.Logf("grammar fixtures: positive=%d negative=%d", len(fixture.Positive), len(fixture.Negative))

	for index, entry := range fixture.Positive {
		raw := decodeRaw(t, entry.RawHex)
		token, err := normalizeComment(raw)
		if err != nil {
			t.Fatalf("positive[%d] raw=%q rejected: %v", index, raw, err)
		}
		if token != entry.Token {
			t.Fatalf("positive[%d] token = %q, want %q", index, token, entry.Token)
		}
		action, err := ParseComment(raw)
		if err != nil {
			t.Fatalf("positive[%d] raw=%q parse: %v", index, raw, err)
		}
		if !action.Valid() {
			t.Fatalf("positive[%d] produced an invalid action: %+v", index, action)
		}
		switch entry.Kind {
		case "role":
			wantRole, err := domain.ParseRole(entry.Role)
			if err != nil {
				t.Fatalf("positive[%d] fixture role %q: %v", index, entry.Role, err)
			}
			if action.Kind != KindRole || action.Role != wantRole || action.Ballot != domain.VoteOptionNone {
				t.Fatalf("positive[%d] raw=%q action=%+v, want role %s", index, raw, action, wantRole)
			}
		case "ballot":
			wantBallot := fixtureBallot(entry.Ballot)
			if action.Kind != KindBallot || action.Ballot != wantBallot || action.Role != 0 {
				t.Fatalf("positive[%d] raw=%q action=%+v, want ballot %s", index, raw, action, wantBallot)
			}
		default:
			t.Fatalf("positive[%d] unknown kind %q", index, entry.Kind)
		}
	}

	for index, entry := range fixture.Negative {
		raw := decodeRaw(t, entry.RawHex)
		action, err := ParseComment(raw)
		wantError := fixtureError(entry.Error)
		if wantError == nil {
			t.Fatalf("negative[%d] unknown fixture error %q", index, entry.Error)
		}
		if !errors.Is(err, wantError) {
			t.Fatalf("negative[%d] raw=%q error = %v, want %v", index, raw, err, wantError)
		}
		if action != (Action{}) {
			t.Fatalf("negative[%d] raw=%q returned a non-zero action: %+v", index, raw, action)
		}
	}
}

// TestActionAccessors pins the typed result contract.
func TestActionAccessors(t *testing.T) {
	builder, err := ParseComment("XÂY")
	if err != nil {
		t.Fatal(err)
	}
	if builder.Kind != KindRole || !builder.IsRole() || builder.IsBallot() {
		t.Fatalf("builder action = %+v", builder)
	}
	if builder.Role != domain.RoleBuilder || builder.Ballot != domain.VoteOptionNone {
		t.Fatalf("builder action fields = %+v", builder)
	}

	ballot, err := ParseComment("B")
	if err != nil {
		t.Fatal(err)
	}
	if ballot.Kind != KindBallot || !ballot.IsBallot() || ballot.IsRole() {
		t.Fatalf("ballot action = %+v", ballot)
	}
	if ballot.Ballot != domain.VoteOptionB || ballot.Role != 0 {
		t.Fatalf("ballot action fields = %+v", ballot)
	}

	rejected, err := ParseComment("XAY!")
	if !errors.Is(err, ErrNotACommand) {
		t.Fatalf("rejection error = %v", err)
	}
	if rejected != (Action{}) || rejected.Valid() {
		t.Fatalf("rejection returned a usable action: %+v", rejected)
	}
}

// TestRejectionCategories pins the stable error taxonomy and its precedence.
func TestRejectionCategories(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want error
	}{
		{"empty", "", ErrEmpty},
		{"whitespace only", " \t\n", ErrEmpty},
		{"too long wins over invalid utf8", strings.Repeat("X", MaxCommentBytes+1) + "\xff", ErrTooLong},
		{"invalid utf8", "XAY\xff", ErrInvalidUTF8},
		{"lone continuation byte", "\x80", ErrInvalidUTF8},
		{"bidi override", "\u202EXAY", ErrForbiddenCharacter},
		{"zero width space", "XAY\u200B", ErrForbiddenCharacter},
		{"control character", "XAY\x07", ErrForbiddenCharacter},
		{"nul byte", "XAY\x00", ErrForbiddenCharacter},
		{"combining flood", "XAY" + strings.Repeat("\u0301", maxCombiningMarks+1), ErrExcessiveCombining},
		{"multi token", "XAY DANH", ErrNotACommand},
		{"punctuation", "XAY!", ErrNotACommand},
		{"unknown word", "HELLO", ErrNotACommand},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			action, err := ParseComment(testCase.raw)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %v", err, testCase.want)
			}
			if action != (Action{}) {
				t.Fatalf("non-zero action on rejection: %+v", action)
			}
		})
	}
}

// TestCommentSizeBoundary pins the 500-byte source contract and the allowed
// leading/trailing whitespace around a valid token.
func TestCommentSizeBoundary(t *testing.T) {
	padded := strings.Repeat(" ", MaxCommentBytes-3) + "XAY"
	if len(padded) != MaxCommentBytes {
		t.Fatalf("fixture length = %d", len(padded))
	}
	action, err := ParseComment(padded)
	if err != nil {
		t.Fatalf("comment of exactly %d bytes rejected: %v", MaxCommentBytes, err)
	}
	if action.Role != domain.RoleBuilder {
		t.Fatalf("padded command role = %s", action.Role)
	}
	if _, err := ParseComment(padded + " "); !errors.Is(err, ErrTooLong) {
		t.Fatalf("comment of %d bytes error = %v, want ErrTooLong", MaxCommentBytes+1, err)
	}
}

// TestNormalizationIdempotence proves Normalize(Normalize(x)) = Normalize(x) for
// every accepted token and for representative accepted variants.
func TestNormalizationIdempotence(t *testing.T) {
	tokens := make([]string, 0, len(tokenActions))
	for token := range tokenActions {
		tokens = append(tokens, token)
	}
	for _, token := range tokens {
		once, err := normalizeComment(token)
		if err != nil {
			t.Fatalf("canonical token %q rejected: %v", token, err)
		}
		twice, err := normalizeComment(once)
		if err != nil {
			t.Fatalf("normalized token %q rejected: %v", token, err)
		}
		if once != twice || once != token {
			t.Fatalf("idempotence failed for %q: %q -> %q", token, once, twice)
		}
	}
	variants := []string{"XÂY", "  xây\t", "THỦ\n", "\u00A0ĐÁNH", "ＡＴＴＡＣＫ", "Ｃ"}
	for _, variant := range variants {
		once, err := normalizeComment(variant)
		if err != nil {
			t.Fatalf("variant %q rejected: %v", variant, err)
		}
		twice, err := normalizeComment(once)
		if err != nil || twice != once {
			t.Fatalf("variant %q not idempotent: %q -> %q (%v)", variant, once, twice, err)
		}
	}
}
