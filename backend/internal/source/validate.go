package source

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

const (
	// MaxCommentBytes is the source-contract comment limit, matching the parser.
	MaxCommentBytes = 500
	// MaxIdentifierRunes bounds provider event/viewer identifiers.
	MaxIdentifierRunes = 256
	// MaxAliasGraphemes bounds display aliases.
	MaxAliasGraphemes = 64
	// MaxAliasBytes bounds alias processing before grapheme counting.
	MaxAliasBytes = 1024
	// MaxAliasMarks bounds combining marks in one alias.
	MaxAliasMarks = 16
)

// ValidateEvent applies the source trust boundary: session/provider binding, event
// kind, comment encoding/length, identifier bounds, alias bounds and control/bidi
// spoofing. Command grammar is deliberately NOT interpreted here; the parser owns
// that. The event is never echoed in error text.
func ValidateEvent(event Event, session Session, mode Mode) error {
	if !session.Valid() {
		return ErrInvalidSession
	}
	if mode != ModeMock && mode != ModeReplay {
		return fmt.Errorf("%w: %s", ErrModeUnsupported, mode)
	}
	if event.Provider != mode.Provider() {
		return ErrProviderMismatch
	}
	if event.SourceSession != session.ID {
		return ErrSessionMismatch
	}
	if !event.Kind.Valid() {
		return ErrEventKindInvalid
	}
	if err := validateMessage(event.Message); err != nil {
		return err
	}
	if err := validateIdentifier(event.ProviderEventID, "event"); err != nil {
		return err
	}
	if err := validateIdentifier(event.ProviderViewerID, "viewer"); err != nil {
		return err
	}
	if err := validateAlias(event.Alias); err != nil {
		return err
	}
	return nil
}

func validateMessage(message string) error {
	if len(message) > MaxCommentBytes {
		return fmt.Errorf("%w: exceeds %d bytes", ErrInvalidMessage, MaxCommentBytes)
	}
	if !utf8.ValidString(message) {
		return fmt.Errorf("%w: not valid UTF-8", ErrInvalidMessage)
	}
	for _, r := range message {
		if isAllowedTextWhitespace(r) {
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return fmt.Errorf("%w: forbidden control or format character", ErrInvalidMessage)
		}
	}
	return nil
}

func validateIdentifier(value, kind string) error {
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%w: %s identifier is not valid UTF-8", ErrInvalidMetadata, kind)
	}
	if utf8.RuneCountInString(value) > MaxIdentifierRunes {
		return fmt.Errorf("%w: %s identifier exceeds %d characters", ErrInvalidMetadata, kind, MaxIdentifierRunes)
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return fmt.Errorf("%w: %s identifier contains a control or format character", ErrInvalidMetadata, kind)
		}
	}
	return nil
}

func validateAlias(alias string) error {
	if alias == "" {
		return nil
	}
	if !utf8.ValidString(alias) {
		return fmt.Errorf("%w: not valid UTF-8", ErrInvalidAlias)
	}
	if len(alias) > MaxAliasBytes {
		return fmt.Errorf("%w: exceeds %d bytes", ErrInvalidAlias, MaxAliasBytes)
	}
	marks := 0
	atStart := true
	for _, r := range alias {
		if isAllowedTextWhitespace(r) {
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return fmt.Errorf("%w: forbidden control or format character", ErrInvalidAlias)
		}
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r) {
			if atStart {
				return fmt.Errorf("%w: leading combining mark", ErrInvalidAlias)
			}
			marks++
			if marks > MaxAliasMarks {
				return fmt.Errorf("%w: excessive combining marks", ErrInvalidAlias)
			}
			continue
		}
		atStart = false
	}
	if clusters := graphemeClusters(alias); clusters > MaxAliasGraphemes {
		return fmt.Errorf("%w: exceeds %d grapheme clusters", ErrInvalidAlias, MaxAliasGraphemes)
	}
	return nil
}

// graphemeClusters returns a conservative upper bound on extended grapheme
// clusters. Combining marks, zero-width joiners, variation selectors and emoji
// modifiers never start a new cluster, so the count can only overestimate; that is
// safe for a maximum-length check.
func graphemeClusters(value string) int {
	count := 0
	previousCR := false
	for _, r := range value {
		switch {
		case previousCR && r == '\n':
			previousCR = false
			continue
		case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r):
			continue
		case r == '\u200D': // zero-width joiner
			continue
		case isVariationSelector(r):
			continue
		case r >= 0x1F3FB && r <= 0x1F3FF: // emoji skin-tone modifiers
			continue
		}
		previousCR = r == '\r'
		count++
	}
	return count
}

func isVariationSelector(r rune) bool {
	return (r >= 0xFE00 && r <= 0xFE0F) || (r >= 0xE0100 && r <= 0xE01EF)
}

func isAllowedTextWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return unicode.Is(unicode.Zs, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r)
}
