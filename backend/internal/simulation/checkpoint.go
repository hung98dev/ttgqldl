package simulation

import (
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// CheckpointEnvelope is the private replay-restoration contract. The canonical
// game-state hash covers SessionState only, so the pending accepted input queues
// need their own deterministic digest: otherwise a tampered queue would resume
// into different gameplay while the stored state hash still matched.
type CheckpointEnvelope struct {
	SchemaVersion int                  `json:"schema_version"`
	RulesVersion  string               `json:"rules_version"`
	StateHash     string               `json:"state_hash"`
	RoleQueue     []RoleCommand        `json:"role_queue"`
	BallotQueue   []VoteCommand        `json:"ballot_queue"`
	SessionXP     []domain.PlayerXP    `json:"session_xp"`
	PauseReasons  []domain.PauseReason `json:"pause_reasons"`
	IntegrityHash string               `json:"integrity_hash"`
}

// checkpointEnvelopeVersion is the private restoration format version, separate
// from the canonical game-state schema. Version 2 adds the complete persisted
// session XP set to the payload and digest; version 1 envelopes cannot be treated
// as equivalent because they lack that coverage, so they are rejected.
const checkpointEnvelopeVersion = 3

// BuildCheckpointEnvelope computes the integrity digest over the full restoration
// payload: canonical state hash plus every queued role and ballot command.
func BuildCheckpointEnvelope(state domain.SessionState, queue RoleQueue) (CheckpointEnvelope, error) {
	stateHash, err := canonical.Hash(state)
	if err != nil {
		return CheckpointEnvelope{}, err
	}
	envelope := CheckpointEnvelope{
		SchemaVersion: checkpointEnvelopeVersion,
		RulesVersion:  state.Revision.RulesVersion,
		StateHash:     stateHash,
		RoleQueue:     append([]RoleCommand(nil), queue.Entries...),
		BallotQueue:   append([]VoteCommand(nil), queue.Ballots...),
		SessionXP:     append([]domain.PlayerXP(nil), state.XP...),
		PauseReasons:  append([]domain.PauseReason(nil), state.PauseReasons...),
	}
	digest, err := envelopeDigest(envelope)
	if err != nil {
		return CheckpointEnvelope{}, err
	}
	envelope.IntegrityHash = digest
	return envelope, nil
}

// VerifyCheckpointEnvelope recomputes the integrity digest and validates every
// queued command, rejecting tampering and duplicate input sequences.
func VerifyCheckpointEnvelope(envelope CheckpointEnvelope, state domain.SessionState, queue RoleQueue) error {
	if envelope.SchemaVersion != checkpointEnvelopeVersion {
		return fmt.Errorf("%w: unsupported checkpoint envelope version %d", ErrInvalidState, envelope.SchemaVersion)
	}
	if envelope.RulesVersion != state.Revision.RulesVersion || state.Revision.RulesVersion != domain.RulesVersion {
		return fmt.Errorf("%w: checkpoint rules version mismatch", ErrInvalidState)
	}
	stateHash, err := canonical.Hash(state)
	if err != nil {
		return err
	}
	if stateHash != envelope.StateHash {
		return fmt.Errorf("%w: checkpoint state hash mismatch", ErrInvalidState)
	}
	if len(envelope.RoleQueue) != queue.Len() || len(envelope.BallotQueue) != len(queue.Ballots) {
		return fmt.Errorf("%w: checkpoint queue does not match the restored queue", ErrInvalidState)
	}
	for index := range envelope.RoleQueue {
		if envelope.RoleQueue[index] != queue.Entries[index] {
			return fmt.Errorf("%w: checkpoint role queue was tampered", ErrInvalidState)
		}
	}
	for index := range envelope.BallotQueue {
		if envelope.BallotQueue[index] != queue.Ballots[index] {
			return fmt.Errorf("%w: checkpoint ballot queue was tampered", ErrInvalidState)
		}
	}
	// Persisted XP, including actors outside current-wave membership, is part of
	// the private restoration contract even though canonical v1 does not hash it.
	if len(envelope.SessionXP) != len(state.XP) {
		return fmt.Errorf("%w: checkpoint XP records do not match", ErrInvalidState)
	}
	for index := range envelope.SessionXP {
		if envelope.SessionXP[index] != state.XP[index] {
			return fmt.Errorf("%w: checkpoint XP record was tampered", ErrInvalidState)
		}
	}
	if len(envelope.PauseReasons) != len(state.PauseReasons) {
		return fmt.Errorf("%w: checkpoint pause reasons do not match", ErrInvalidState)
	}
	for index := range envelope.PauseReasons {
		if envelope.PauseReasons[index] != state.PauseReasons[index] {
			return fmt.Errorf("%w: checkpoint pause reasons were tampered", ErrInvalidState)
		}
	}
	seen := make(map[int64]bool, len(envelope.RoleQueue)+len(envelope.BallotQueue))
	for _, command := range envelope.RoleQueue {
		if seen[command.InputSeq] {
			return fmt.Errorf("%w: duplicate input sequence %d", ErrInvalidState, command.InputSeq)
		}
		seen[command.InputSeq] = true
		if !command.ActorKey.Valid() || !command.Role.Valid() || command.InputSeq <= 0 {
			return fmt.Errorf("%w: invalid queued role command", ErrInvalidState)
		}
	}
	for _, command := range envelope.BallotQueue {
		if seen[command.InputSeq] {
			return fmt.Errorf("%w: duplicate input sequence %d", ErrInvalidState, command.InputSeq)
		}
		seen[command.InputSeq] = true
		if !command.ActorKey.Valid() || !command.Option.Valid() || command.InputSeq <= 0 {
			return fmt.Errorf("%w: invalid queued ballot", ErrInvalidState)
		}
	}
	digest, err := envelopeDigest(envelope)
	if err != nil {
		return err
	}
	if digest != envelope.IntegrityHash {
		return fmt.Errorf("%w: checkpoint integrity hash mismatch", ErrInvalidState)
	}
	return nil
}

func envelopeDigest(envelope CheckpointEnvelope) (string, error) {
	roles := make([]any, 0, len(envelope.RoleQueue))
	for _, command := range envelope.RoleQueue {
		roles = append(roles, map[string]any{
			"input_seq":      command.InputSeq,
			"actor_key":      string(command.ActorKey),
			"role":           command.Role.String(),
			"target_wave":    command.TargetWave,
			"scheduled_tick": int64(command.ScheduledTick),
			"accepted_tick":  int64(command.AcceptedTick),
		})
	}
	ballots := make([]any, 0, len(envelope.BallotQueue))
	for _, command := range envelope.BallotQueue {
		ballots = append(ballots, map[string]any{
			"input_seq":      command.InputSeq,
			"actor_key":      string(command.ActorKey),
			"option":         command.Option.String(),
			"target_wave":    command.TargetWave,
			"scheduled_tick": int64(command.ScheduledTick),
			"accepted_tick":  int64(command.AcceptedTick),
		})
	}
	xp := make([]any, 0, len(envelope.SessionXP))
	for _, row := range envelope.SessionXP {
		xp = append(xp, map[string]any{
			"actor_key":  string(row.ActorKey),
			"session_xp": row.XP,
		})
	}
	pauseReasons := make([]any, 0, len(envelope.PauseReasons))
	for _, reason := range envelope.PauseReasons {
		pauseReasons = append(pauseReasons, reason.String())
	}
	payload := map[string]any{
		"schema_version": int64(envelope.SchemaVersion),
		"rules_version":  envelope.RulesVersion,
		"state_hash":     envelope.StateHash,
		"role_queue":     roles,
		"ballot_queue":   ballots,
		"session_xp":     xp,
		"pause_reasons":  pauseReasons,
	}
	encoded, err := canonical.MarshalJCS(payload)
	if err != nil {
		return "", err
	}
	return canonical.SHA256Hex(encoded), nil
}
