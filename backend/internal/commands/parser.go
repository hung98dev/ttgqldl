package commands

import "github.com/hung98dev/ttgqldl/backend/internal/domain"

// Kind distinguishes the two authorized command families. The zero value means no
// command was recognized.
type Kind uint8

const (
	// KindNone is the zero value: no recognized command.
	KindNone Kind = iota
	// KindRole is a role selection action.
	KindRole
	// KindBallot is a vote ballot action.
	KindBallot
)

// String returns a stable diagnostic label.
func (k Kind) String() string {
	switch k {
	case KindRole:
		return "role"
	case KindBallot:
		return "ballot"
	default:
		return "none"
	}
}

// Action is one recognized player command. Exactly one arm is populated according
// to Kind: KindRole carries a valid domain.Role and leaves Ballot as
// VoteOptionNone; KindBallot carries a valid non-none domain.VoteOption and leaves
// Role zero. The zero Action is returned with every error, so a caller can never
// mistake a rejection for a command.
type Action struct {
	Kind   Kind
	Role   domain.Role
	Ballot domain.VoteOption
}

// Valid reports whether the action is internally consistent. Action is a tagged
// union: exactly one arm may be populated, so a ballot action with a role (or a
// role action with a ballot) is rejected.
func (a Action) Valid() bool {
	switch a.Kind {
	case KindRole:
		return a.Role.Valid() && a.Ballot == domain.VoteOptionNone
	case KindBallot:
		return a.Ballot.Valid() && a.Ballot != domain.VoteOptionNone && a.Role == 0
	default:
		return false
	}
}

// IsRole reports whether the action is a role selection.
func (a Action) IsRole() bool { return a.Kind == KindRole }

// IsBallot reports whether the action is a vote ballot.
func (a Action) IsBallot() bool { return a.Kind == KindBallot }

// ParseComment parses one untrusted comment into exactly one recognized action.
//
// It is pure, deterministic and thread-safe. On rejection it returns the zero
// Action and a sentinel error from errors.go; the comment text is never included
// in the error and nothing is logged.
func ParseComment(raw string) (Action, error) {
	token, err := normalizeComment(raw)
	if err != nil {
		return Action{}, err
	}
	action, ok := tokenActions[token]
	if !ok {
		// normalizeComment only returns allowlisted tokens; this is defensive.
		return Action{}, ErrNotACommand
	}
	return action, nil
}

// tokenActions is the complete authorized grammar. It is the single source of
// truth for both the allowlist check and the typed mapping, so the two can never
// disagree. Lookup is a map read, not iteration, so results are deterministic.
var tokenActions = map[string]Action{
	"xay":    {Kind: KindRole, Role: domain.RoleBuilder},
	"thu":    {Kind: KindRole, Role: domain.RoleGuard},
	"danh":   {Kind: KindRole, Role: domain.RoleFighter},
	"build":  {Kind: KindRole, Role: domain.RoleBuilder},
	"defend": {Kind: KindRole, Role: domain.RoleGuard},
	"attack": {Kind: KindRole, Role: domain.RoleFighter},
	"a":      {Kind: KindBallot, Ballot: domain.VoteOptionA},
	"b":      {Kind: KindBallot, Ballot: domain.VoteOptionB},
	"c":      {Kind: KindBallot, Ballot: domain.VoteOptionC},
}
