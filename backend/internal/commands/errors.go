package commands

import "errors"

// Stable rejection categories. Callers branch with errors.Is and never parse
// error prose. Errors never include the rejected comment, provider identifiers or
// any other untrusted content.
var (
	// ErrEmpty reports an empty or whitespace-only comment.
	ErrEmpty = errors.New("commands: empty comment")
	// ErrTooLong reports a comment above the source-contract 500-byte limit.
	ErrTooLong = errors.New("commands: comment exceeds the 500-byte limit")
	// ErrInvalidUTF8 reports bytes that are not valid UTF-8.
	ErrInvalidUTF8 = errors.New("commands: comment is not valid UTF-8")
	// ErrForbiddenCharacter reports controls, Unicode format characters (bidi
	// overrides/isolates, zero-width marks) or a malformed leading combining mark.
	ErrForbiddenCharacter = errors.New("commands: comment contains a forbidden control or spoofing character")
	// ErrExcessiveCombining reports a combining-mark flood used for spoofing or
	// resource abuse.
	ErrExcessiveCombining = errors.New("commands: comment contains excessive combining marks")
	// ErrNotACommand reports syntactically valid text that is not exactly one
	// authorized command token (unknown word, punctuation, substring or multiple
	// tokens).
	ErrNotACommand = errors.New("commands: comment is not a recognized command")
)
