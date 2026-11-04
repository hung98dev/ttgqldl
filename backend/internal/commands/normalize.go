package commands

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// MaxCommentBytes is the source-contract maximum comment length. It is enforced
// first, before any Unicode normalization work, so oversized input cannot force
// expensive allocations.
const MaxCommentBytes = 500

// maxCombiningMarks bounds combining marks in one comment. Legitimate Vietnamese
// text needs at most a few; a longer run is a spoofing/resource attack.
const maxCombiningMarks = 8

// normalizeComment validates raw input and returns its canonical allowlisted
// token. The returned token is always a key of tokenActions.
func normalizeComment(raw string) (string, error) {
	if len(raw) == 0 {
		return "", ErrEmpty
	}
	if len(raw) > MaxCommentBytes {
		return "", ErrTooLong
	}
	if !utf8.ValidString(raw) {
		return "", ErrInvalidUTF8
	}
	if err := scanForbidden(raw); err != nil {
		return "", err
	}

	// Fast path: on pure ASCII, NFKC and Vietnamese diacritic folding are identity
	// maps. Only lower-casing and whitespace trimming remain, which avoids several
	// allocations for the most common commands (BUILD/DEFEND/ATTACK/A/B/C/XAY).
	if isASCII(raw) {
		token := strings.ToLower(strings.TrimFunc(raw, isAllowedWhitespace))
		return finishToken(token)
	}

	normalized := norm.NFKC.String(raw)
	// NFKC can reorder or compose characters; re-check the output so a normalization
	// step can never introduce a forbidden character.
	if err := scanForbidden(normalized); err != nil {
		return "", err
	}
	lowered := strings.ToLower(normalized)
	folded := foldVietnamese(lowered)
	collapsed := normalizeWhitespace(folded)
	return finishToken(strings.TrimFunc(collapsed, isAllowedWhitespace))
}

// finishToken applies the strict one-token rule and the exact allowlist check.
// There is no prefix, substring or punctuation stripping.
func finishToken(token string) (string, error) {
	if token == "" {
		return "", ErrEmpty
	}
	if strings.IndexFunc(token, isAllowedWhitespace) >= 0 {
		return "", ErrNotACommand
	}
	if _, ok := tokenActions[token]; !ok {
		return "", ErrNotACommand
	}
	return token, nil
}

// scanForbidden rejects controls, Unicode format characters and mark floods. It
// runs before NFKC and again on its output. Allowed whitespace is exempt from the
// control rule because legitimate leading/trailing whitespace is accepted.
func scanForbidden(text string) error {
	marks := 0
	atStart := true
	for _, r := range text {
		if isAllowedWhitespace(r) {
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ErrForbiddenCharacter
		}
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r) {
			if atStart {
				// A combining mark with no base character is malformed input, not a
				// command variant.
				return ErrForbiddenCharacter
			}
			marks++
			if marks > maxCombiningMarks {
				return ErrExcessiveCombining
			}
			continue
		}
		atStart = false
	}
	return nil
}

// isAllowedWhitespace accepts ASCII whitespace controls and Unicode separator
// categories. Other controls (including U+0085 NEL) and all format characters are
// rejected.
func isAllowedWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return unicode.Is(unicode.Zs, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r)
}

// isASCII reports whether every byte is ASCII. Invalid UTF-8 was rejected before
// this point, so ASCII here means printable ASCII plus allowed whitespace controls.
func isASCII(text string) bool {
	for index := 0; index < len(text); index++ {
		if text[index] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// foldVietnamese removes Vietnamese diacritics after lower-casing. It decomposes
// with NFD, drops combining marks and maps the Vietnamese letters đ/Đ to d, which
// ordinary mark removal cannot handle.
func foldVietnamese(text string) string {
	decomposed := norm.NFD.String(text)
	var builder strings.Builder
	builder.Grow(len(decomposed))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == '\u0111' || r == '\u0110' { // đ / Đ
			builder.WriteByte('d')
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

// normalizeWhitespace maps every allowed Unicode whitespace rune to an ASCII
// space. This is deterministic and independent of locale or environment.
func normalizeWhitespace(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	for _, r := range text {
		if isAllowedWhitespace(r) {
			builder.WriteByte(' ')
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
