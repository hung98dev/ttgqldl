package identity

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// AliasMaxBytes is the database backstop for a display alias column.
	AliasMaxBytes = 256
	// AliasMaxGraphemes bounds public display aliases.
	AliasMaxGraphemes = 64
	// AliasMaxMarks bounds combining marks in one alias.
	AliasMaxMarks = 16
	// DefaultAliasFallback is the safe alias default from the security plan.
	DefaultAliasFallback = "Villager"
)

// AliasVisibility carries the owner's presentation state. Hidden, erased and opted-out
// states always suppress display, regardless of the raw alias.
type AliasVisibility uint8

const (
	AliasVisible AliasVisibility = iota
	AliasHidden
	AliasErased
	AliasOptedOut
)

// DisplayAlias is the safe presentation result. Fallback is true when an unusable raw
// alias was replaced; Visible is false when policy suppresses display entirely.
type DisplayAlias struct {
	Text     string
	Visible  bool
	Fallback bool
}

// SanitizeDisplayAlias is a structural publication boundary, not a profanity
// moderator. It enforces UTF-8, the 64-grapheme/256-byte limits, control/format/mark
// safety, URL-like and markup rejection, and the fallback/opt-out policy. Lexical
// moderation and audit remain M05/M08 gates.
func SanitizeDisplayAlias(raw string, visibility AliasVisibility, fallback string) DisplayAlias {
	safeFallback := DefaultAliasFallback
	if fallback != "" && structurallySafeAlias(fallback) && !unsafeAliasContent(fallback) {
		safeFallback = fallback
	}
	if visibility != AliasVisible {
		return DisplayAlias{}
	}
	if structurallySafeAlias(raw) && !unsafeAliasContent(raw) {
		return DisplayAlias{Text: raw, Visible: true}
	}
	return DisplayAlias{Text: safeFallback, Visible: true, Fallback: true}
}

func structurallySafeAlias(value string) bool {
	if value == "" || len(value) > AliasMaxBytes || !utf8.ValidString(value) {
		return false
	}
	if strings.TrimSpace(value) == "" {
		return false
	}
	marks := 0
	atStart := true
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r) {
			if atStart {
				return false
			}
			marks++
			if marks > AliasMaxMarks {
				return false
			}
			continue
		}
		atStart = false
	}
	return graphemeClustersAlias(value) <= AliasMaxGraphemes
}

func unsafeAliasContent(value string) bool {
	lower := strings.ToLower(value)
	if strings.ContainsAny(value, "<>&") {
		return true
	}
	return strings.Contains(lower, "://") ||
		strings.Contains(lower, "www.") ||
		strings.Contains(lower, "tiktok.com") ||
		strings.Contains(lower, "javascript:")
}

// graphemeClustersAlias is a conservative upper bound on extended grapheme clusters:
// marks, ZWJ, variation selectors and emoji modifiers never start a cluster, so the
// count can only overestimate (safe for a maximum check).
func graphemeClustersAlias(value string) int {
	count := 0
	previousCR := false
	for _, r := range value {
		switch {
		case previousCR && r == '\n':
			previousCR = false
			continue
		case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r):
			continue
		case r == '\u200D':
			continue
		case (r >= 0xFE00 && r <= 0xFE0F) || (r >= 0xE0100 && r <= 0xE01EF):
			continue
		case r >= 0x1F3FB && r <= 0x1F3FF:
			continue
		}
		previousCR = r == '\r'
		count++
	}
	return count
}
