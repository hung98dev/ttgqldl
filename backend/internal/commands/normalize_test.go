package commands

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestVietnameseFolding covers precomposed, decomposed and upper-case Vietnamese
// forms, including the đ/Đ mapping that combining-mark removal alone cannot do.
func TestVietnameseFolding(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"xây", "xay"},
		{"XÂY", "xay"},
		{"x" + "\u00e2" + "y", "xay"},  // precomposed â
		{"xa" + "\u0302" + "y", "xay"}, // decomposed circumflex
		{"thủ", "thu"},
		{"THỦ", "thu"},
		{"thu" + "\u0309", "thu"}, // decomposed hook above
		{"đánh", "danh"},
		{"ĐÁNH", "danh"},
		{"đa" + "\u0301" + "nh", "danh"}, // decomposed acute after đ
		{"Đa" + "\u0301" + "nh", "danh"},
	}
	for _, testCase := range cases {
		token, err := normalizeComment(testCase.raw)
		if err != nil {
			t.Fatalf("raw %q rejected: %v", testCase.raw, err)
		}
		if token != testCase.want {
			t.Fatalf("raw %q token = %q, want %q", testCase.raw, token, testCase.want)
		}
	}
}

// TestWhitespacePolicy accepts legitimate leading/trailing Unicode whitespace and
// rejects whitespace that would join multiple words into a command.
func TestWhitespacePolicy(t *testing.T) {
	accepted := []string{
		" XÂY ",
		"\tTHỦ\n",
		"\u00A0ĐÁNH\u00A0",
		"\u3000A\u3000",
		"\u2028B\u2029",
		"\r\nATTACK\r\n",
	}
	for _, raw := range accepted {
		if _, err := ParseComment(raw); err != nil {
			t.Fatalf("whitespace variant %q rejected: %v", raw, err)
		}
	}
	rejected := []string{
		"X Â Y",
		"B U I L D",
		"XAY\tTHU",
		"XAY\nTHU",
		"A B",
		" " + "XAY" + " " + "DANH" + " ",
	}
	for _, raw := range rejected {
		if _, err := ParseComment(raw); !errors.Is(err, ErrNotACommand) {
			t.Fatalf("multi-token variant %q error = %v, want ErrNotACommand", raw, err)
		}
	}
}

// TestNFKCCompatibility accepts compatibility equivalents that normalize exactly to
// an authorized token.
func TestNFKCCompatibility(t *testing.T) {
	cases := []string{
		"ＡＴＴＡＣＫ", // full-width Latin
		"ＢＵＩＬＤ",  // full-width Latin
		"Ａ",      // full-width ballot A
		"ＤＥＦＥＮＤ", // full-width Latin
	}
	for _, raw := range cases {
		if _, err := ParseComment(raw); err != nil {
			t.Fatalf("NFKC-equivalent %q rejected: %v", raw, err)
		}
	}
	// Compatibility characters that normalize to a non-command must stay rejected.
	for _, raw := range []string{"ﬁ", "Ⅹ", "①", "㎡"} {
		if _, err := ParseComment(raw); !errors.Is(err, ErrNotACommand) {
			t.Fatalf("non-command compatibility char %q error = %v", raw, err)
		}
	}
}

// TestForbiddenSpoofingCharacters rejects bidi controls, isolates, zero-width
// characters and format characters in any position.
func TestForbiddenSpoofingCharacters(t *testing.T) {
	forbidden := []rune{
		'\u200B', '\u200C', '\u200D', '\u200E', '\u200F',
		'\u202A', '\u202B', '\u202C', '\u202D', '\u202E',
		'\u2060', '\u2066', '\u2067', '\u2068', '\u2069',
		'\uFEFF', '\u061C',
	}
	for _, r := range forbidden {
		for _, raw := range []string{string(r) + "XAY", "XAY" + string(r), "X" + string(r) + "AY"} {
			if _, err := ParseComment(raw); !errors.Is(err, ErrForbiddenCharacter) {
				t.Fatalf("spoofing rune %U in %q error = %v, want ErrForbiddenCharacter", r, raw, err)
			}
		}
	}
}

// TestControlCharacterPolicy rejects controls except the legitimate whitespace
// controls, which are trimmed or cause a one-token grammar rejection.
func TestControlCharacterPolicy(t *testing.T) {
	for r := rune(0); r < 0x20; r++ {
		raw := "XAY" + string(r)
		_, err := ParseComment(raw)
		switch r {
		case '\t', '\n', '\v', '\f', '\r':
			if err != nil {
				t.Fatalf("allowed whitespace control %U rejected: %v", r, err)
			}
		default:
			if !errors.Is(err, ErrForbiddenCharacter) {
				t.Fatalf("control %U error = %v, want ErrForbiddenCharacter", r, err)
			}
		}
	}
	for _, r := range []rune{'\u007F', '\u0085', '\u009F'} {
		if _, err := ParseComment("XAY" + string(r)); !errors.Is(err, ErrForbiddenCharacter) {
			t.Fatalf("control %U was not rejected", r)
		}
	}
}

// TestMixedScriptLookalikes rejects deceptive non-Latin lookalikes while accepting
// compatibility forms that normalize exactly to authorized ASCII/Latin tokens.
func TestMixedScriptLookalikes(t *testing.T) {
	rejected := []string{
		"ХАУ",    // Cyrillic Ha A U
		"Α",      // Greek capital alpha
		"ΑΤΤΑCΚ", // mixed Greek/Latin
	}
	for _, raw := range rejected {
		action, err := ParseComment(raw)
		if !errors.Is(err, ErrNotACommand) {
			t.Fatalf("lookalike %q error = %v, want ErrNotACommand", raw, err)
		}
		if action != (Action{}) {
			t.Fatalf("lookalike %q produced an action", raw)
		}
	}
	accepted := []string{
		"ⅩAY",   // Roman numeral ten NFKC-normalizes to X
		"𝐁𝐔𝐈𝐋𝐃", // mathematical bold NFKC-normalizes to BUILD
	}
	for _, raw := range accepted {
		if _, err := ParseComment(raw); err != nil {
			t.Fatalf("NFKC-equivalent form %q rejected: %v", raw, err)
		}
	}
}

// TestInvalidUTF8 proves UTF-8 validation cannot be bypassed and that no
// normalization is attempted on invalid bytes.
func TestInvalidUTF8(t *testing.T) {
	cases := []string{
		"\xff",
		"XAY\xff",
		"\xc3",
		"\x80",
		"\xed\xa0\x80", // UTF-8-encoded surrogate
		"XAY\xc0\xaf",
	}
	for _, raw := range cases {
		if utf8.ValidString(raw) {
			t.Fatalf("fixture %q is unexpectedly valid UTF-8", raw)
		}
		if _, err := ParseComment(raw); !errors.Is(err, ErrInvalidUTF8) {
			t.Fatalf("invalid UTF-8 error = %v, want ErrInvalidUTF8", err)
		}
	}
}

// TestExcessiveCombining rejects a mark flood and a leading mark.
func TestExcessiveCombining(t *testing.T) {
	if _, err := ParseComment("XAY" + strings.Repeat("\u0301", maxCombiningMarks)); err != nil {
		t.Fatalf("allowed number of combining marks rejected: %v", err)
	}
	if _, err := ParseComment("XAY" + strings.Repeat("\u0301", maxCombiningMarks+1)); !errors.Is(err, ErrExcessiveCombining) {
		t.Fatalf("combining flood error = %v, want ErrExcessiveCombining", err)
	}
	if _, err := ParseComment("\u0301XAY"); !errors.Is(err, ErrForbiddenCharacter) {
		t.Fatalf("leading combining mark error = %v, want ErrForbiddenCharacter", err)
	}
}

// TestNonASCIIDigitsAreNotCommands rejects any digit, including NFKC equivalents.
func TestNonASCIIDigitsAreNotCommands(t *testing.T) {
	for _, raw := range []string{"1", "１", "١", "XAY1", "XAY１"} {
		if _, err := ParseComment(raw); !errors.Is(err, ErrNotACommand) {
			t.Fatalf("digit input %q error = %v, want ErrNotACommand", raw, err)
		}
	}
}

// TestLocaleIndependence proves results do not depend on process locale variables.
func TestLocaleIndependence(t *testing.T) {
	samples := []string{"ATTACK", "XÂY", "THỦ", "ĐÁNH", "Ａ", "XAY!"}
	baseline := make([]struct {
		action Action
		err    error
	}, len(samples))
	for index, raw := range samples {
		baseline[index].action, baseline[index].err = ParseComment(raw)
	}
	for _, locale := range []string{"tr_TR.UTF-8", "ar_SA.UTF-8", "vi_VN.UTF-8"} {
		t.Setenv("LANG", locale)
		t.Setenv("LC_ALL", locale)
		for index, raw := range samples {
			action, err := ParseComment(raw)
			if action != baseline[index].action || (err == nil) != (baseline[index].err == nil) {
				t.Fatalf("locale %s changed the result for %q: %+v/%v", locale, raw, action, err)
			}
		}
	}
}
