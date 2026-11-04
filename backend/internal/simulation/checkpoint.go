package simulation

import (
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// CheckpointEnvelope is the private replay-restoration contract. It carries two
// independent integrity values:
//
//   - StateHash is the canonical schema-v1 game-state hash. It is kept unchanged
//     so canonical fixtures and replay traces remain comparable.
//   - PrivateDigest is a SHA-256 over the complete private restoration payload:
//     every authoritative SessionState field (including the fields canonical v1
//     intentionally does not publish) plus every accepted-but-not-yet-applied role
//     and ballot command. Canonical v1 alone cannot distinguish two valid states
//     that differ in, for example, InputSequence, Mode, applied ballots,
//     MilitiaAttack, BossIncoming or the wave Result, and those fields change the
//     continuation.
//
// This is an integrity check against accidental or partial corruption, not an
// authentication mechanism: an attacker who can rewrite both the payload and the
// digest can recompute the digest. Signing or an external trusted anchor is out of
// scope for M02.
type CheckpointEnvelope struct {
	SchemaVersion int           `json:"schema_version"`
	RulesVersion  string        `json:"rules_version"`
	StateHash     string        `json:"state_hash"`
	PrivateDigest string        `json:"private_digest"`
	RoleQueue     []RoleCommand `json:"role_queue"`
	BallotQueue   []VoteCommand `json:"ballot_queue"`
}

// checkpointEnvelopeVersion is the private restoration format version. It is
// deliberately separate from the canonical game-state schema version, the gameplay
// rules identifier and the independent reference fixture version.
//
//	1: canonical hash + pending queues
//	2: + complete persisted session XP
//	3: + ordered pause reasons
//	4: complete deterministic private-state payload (every authoritative field,
//	   including Mode, InputSequence, applied ballots, difficulty internals,
//	   BossIncoming and the wave Result); supersedes the incremental exception list
const checkpointEnvelopeVersion = 4

// BuildCheckpointEnvelope validates the state, computes the canonical v1 hash and
// the complete private restoration digest.
func BuildCheckpointEnvelope(state domain.SessionState, queue RoleQueue) (CheckpointEnvelope, error) {
	if err := state.Validate(); err != nil {
		return CheckpointEnvelope{}, fmt.Errorf("%w: checkpoint state is invalid: %v", ErrInvalidState, err)
	}
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
	}
	digest, err := privateRestorationDigest(state, queue)
	if err != nil {
		return CheckpointEnvelope{}, err
	}
	envelope.PrivateDigest = digest
	return envelope, nil
}

// VerifyCheckpointEnvelope recomputes both integrity values and validates every
// queued command. It fails closed: an invalid state, a foreign revision, a
// mismatched queue, a duplicate cross-kind input sequence, a malformed command or
// any digest mismatch rejects the restore before a single simulation step runs.
func VerifyCheckpointEnvelope(envelope CheckpointEnvelope, state domain.SessionState, queue RoleQueue) error {
	if envelope.SchemaVersion != checkpointEnvelopeVersion {
		return fmt.Errorf("%w: unsupported checkpoint envelope version %d", ErrInvalidState, envelope.SchemaVersion)
	}
	if envelope.RulesVersion != state.Revision.RulesVersion || state.Revision.RulesVersion != domain.RulesVersion {
		return fmt.Errorf("%w: checkpoint rules version mismatch", ErrInvalidState)
	}
	if err := state.Validate(); err != nil {
		return fmt.Errorf("%w: checkpoint state is invalid: %v", ErrInvalidState, err)
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
	if err := validateCheckpointQueue(state, envelope.RoleQueue, envelope.BallotQueue); err != nil {
		return err
	}
	digest, err := privateRestorationDigest(state, queue)
	if err != nil {
		return err
	}
	if digest != envelope.PrivateDigest {
		return fmt.Errorf("%w: checkpoint private digest mismatch", ErrInvalidState)
	}
	return nil
}

// validateCheckpointQueue enforces the accepted-command contract on the private
// payload itself, so a tampered queue with a recomputed digest still fails.
func validateCheckpointQueue(state domain.SessionState, roles []RoleCommand, ballots []VoteCommand) error {
	seen := make(map[int64]bool, len(roles)+len(ballots))
	for _, command := range roles {
		if command.InputSeq <= 0 || !command.ActorKey.Valid() || !command.Role.Valid() {
			return fmt.Errorf("%w: invalid queued role command", ErrInvalidState)
		}
		if seen[command.InputSeq] {
			return fmt.Errorf("%w: duplicate input sequence %d", ErrInvalidState, command.InputSeq)
		}
		seen[command.InputSeq] = true
		if command.TargetWave != state.Wave.Number && command.TargetWave != state.Wave.Number+1 {
			return fmt.Errorf("%w: queued role command targets an invalid wave %d", ErrInvalidState, command.TargetWave)
		}
		if command.ScheduledTick < 0 || command.ScheduledTick > domain.FinalCombatTick {
			return fmt.Errorf("%w: queued role command tick %d is out of range", ErrInvalidState, command.ScheduledTick)
		}
		if command.TargetWave == state.Wave.Number+1 {
			if command.ScheduledTick != 0 {
				return fmt.Errorf("%w: next-wave role command must be effective at tick 0", ErrInvalidState)
			}
			continue
		}
		// A current-wave command was accepted at an already committed tick and is
		// effective strictly later; a retroactive command is rejected.
		if command.AcceptedTick < 0 || command.AcceptedTick > domain.RolloverTick {
			return fmt.Errorf("%w: queued role command accepted tick %d is out of range", ErrInvalidState, command.AcceptedTick)
		}
		if command.ScheduledTick <= command.AcceptedTick {
			return fmt.Errorf("%w: queued role command is retroactive (scheduled %d, accepted %d)",
				ErrInvalidState, command.ScheduledTick, command.AcceptedTick)
		}
	}
	for _, command := range ballots {
		if command.InputSeq <= 0 || !command.ActorKey.Valid() {
			return fmt.Errorf("%w: invalid queued ballot", ErrInvalidState)
		}
		if !command.Option.Valid() || command.Option == domain.VoteOptionNone {
			return fmt.Errorf("%w: queued ballot has no valid option", ErrInvalidState)
		}
		if seen[command.InputSeq] {
			return fmt.Errorf("%w: duplicate input sequence %d", ErrInvalidState, command.InputSeq)
		}
		seen[command.InputSeq] = true
		if command.TargetWave != state.Wave.Number {
			return fmt.Errorf("%w: queued ballot targets wave %d while the session is on %d",
				ErrInvalidState, command.TargetWave, state.Wave.Number)
		}
		if !domain.BallotTickEligible(command.ScheduledTick) {
			return fmt.Errorf("%w: queued ballot tick %d is outside the voting window", ErrInvalidState, command.ScheduledTick)
		}
		if command.AcceptedTick < 0 || command.ScheduledTick != command.AcceptedTick+1 {
			return fmt.Errorf("%w: queued ballot is not scheduled at accepted+1 (%d/%d)",
				ErrInvalidState, command.ScheduledTick, command.AcceptedTick)
		}
	}
	return nil
}

// privateRestorationDigest computes SHA-256 over the RFC 8785 canonical encoding of
// the complete private restoration payload. All slices are already in their
// canonical order and the encoder sorts object keys, so encoding is stable and
// never depends on map iteration order.
func privateRestorationDigest(state domain.SessionState, queue RoleQueue) (string, error) {
	payload, err := privateRestorationPayload(state, queue)
	if err != nil {
		return "", err
	}
	encoded, err := canonical.MarshalJCS(payload)
	if err != nil {
		return "", err
	}
	return canonical.SHA256Hex(encoded), nil
}

// privateRestorationPayload projects every authoritative field that deterministic
// continuation could depend on. It is intentionally exhaustive: adding a field to
// SessionState requires adding it here as well, otherwise checkpoint integrity
// would silently stop covering it.
func privateRestorationPayload(state domain.SessionState, queue RoleQueue) (map[string]any, error) {
	resources := map[string]any{
		"wood":  state.Village.Resources.Wood,
		"stone": state.Village.Resources.Stone,
		"food":  state.Village.Resources.Food,
	}
	village := map[string]any{
		"id":                string(state.Village.ID),
		"level":             int64(state.Village.Level),
		"wall_tier":         int64(state.Village.WallTier),
		"wall_hp":           state.Village.WallHP,
		"townhall_hp":       state.Village.TownhallHP,
		"resources":         resources,
		"construction_work": state.Village.ConstructionWork,
		"total_wins":        state.Village.TotalWins,
	}
	pools := make([]any, 0, len(state.Wave.Pools))
	for _, pool := range state.Wave.Pools {
		pools = append(pools, map[string]any{
			"lane":      pool.Lane.String(),
			"archetype": pool.Archetype.String(),
			"hp":        pool.HP,
		})
	}
	ballots := make([]any, 0, len(state.Wave.Ballots))
	for _, ballot := range state.Wave.Ballots {
		ballots = append(ballots, ballotedPayload(ballot))
	}
	wave := map[string]any{
		"number": state.Wave.Number,
		"tick":   int64(state.Wave.Tick),
		"phase":  state.Wave.Phase.String(),
		"pools":  pools,
		"boss": map[string]any{
			"exists": state.Wave.Boss.Exists,
			"hp":     state.Wave.Boss.HP,
			"max_hp": state.Wave.Boss.MaxHP,
		},
		"vote": map[string]any{
			"open":     state.Wave.Vote.Open,
			"closed":   state.Wave.Vote.Closed,
			"selected": optionalVoteOption(state.Wave.Vote.Selected),
			"counts": map[string]any{
				"a": state.Wave.Vote.Counts.A,
				"b": state.Wave.Vote.Counts.B,
				"c": state.Wave.Vote.Counts.C,
			},
		},
		"ballots":       ballots,
		"defeated":      state.Wave.Defeated,
		"result":        state.Wave.Result.String(),
		"boss_incoming": state.Wave.BossIncoming,
	}
	members := make([]any, 0, len(state.Members))
	for _, member := range state.Members {
		members = append(members, map[string]any{
			"actor_key":         string(member.ActorKey),
			"role":              member.Role.String(),
			"joined_tick":       int64(member.JoinedTick),
			"last_change_tick":  optionalTick(member.LastChangeTick),
			"first_switch_used": member.FirstSwitchUsed,
			"xp_awarded":        member.XPAwarded,
		})
	}
	pending := make([]any, 0, len(state.Pending))
	for _, entry := range state.Pending {
		pending = append(pending, map[string]any{
			"actor_key":         string(entry.ActorKey),
			"role":              entry.Role.String(),
			"target_wave":       entry.TargetWave,
			"last_change_tick":  optionalTick(entry.LastChangeTick),
			"first_switch_used": entry.FirstSwitchUsed,
		})
	}
	xp := make([]any, 0, len(state.XP))
	for _, row := range state.XP {
		xp = append(xp, map[string]any{
			"actor_key":  string(row.ActorKey),
			"session_xp": row.XP,
		})
	}
	difficulty := map[string]any{
		"level":             int64(state.Difficulty.Level),
		"population_anchor": int64(state.Difficulty.PopulationAnchor),
		"locked":            state.Difficulty.Locked,
		"actual_population": int64(state.Difficulty.ActualPopulation),
		"recovery_percent":  int64(state.Difficulty.RecoveryPercent),
		"base_hp":           state.Difficulty.BaseHP,
		"militia_attack":    state.Difficulty.MilitiaAttack,
		"boss_max_hp":       state.Difficulty.BossMaxHP,
	}
	pauseReasons := make([]any, 0, len(state.PauseReasons))
	for _, reason := range state.PauseReasons {
		pauseReasons = append(pauseReasons, reason.String())
	}

	roleQueue := make([]any, 0, len(queue.Entries))
	for _, command := range queue.Entries {
		roleQueue = append(roleQueue, map[string]any{
			"input_seq":      command.InputSeq,
			"actor_key":      string(command.ActorKey),
			"role":           command.Role.String(),
			"target_wave":    command.TargetWave,
			"scheduled_tick": int64(command.ScheduledTick),
			"accepted_tick":  int64(command.AcceptedTick),
		})
	}
	ballotQueue := make([]any, 0, len(queue.Ballots))
	for _, command := range queue.Ballots {
		ballotQueue = append(ballotQueue, map[string]any{
			"input_seq":      command.InputSeq,
			"actor_key":      string(command.ActorKey),
			"option":         command.Option.String(),
			"target_wave":    command.TargetWave,
			"scheduled_tick": int64(command.ScheduledTick),
			"accepted_tick":  int64(command.AcceptedTick),
		})
	}

	return map[string]any{
		"schema_version": int64(checkpointEnvelopeVersion),
		"rules_version":  state.Revision.RulesVersion,
		"state": map[string]any{
			"revision": map[string]any{
				"schema_version": int64(state.Revision.SchemaVersion),
				"rules_version":  state.Revision.RulesVersion,
			},
			"session_id":         string(state.ID),
			"village_id":         string(state.VillageID),
			"mode":               state.Mode.String(),
			"lifecycle":          state.Lifecycle.String(),
			"pause_reasons":      pauseReasons,
			"village":            village,
			"wave":               wave,
			"members":            members,
			"pending":            pending,
			"xp":                 xp,
			"difficulty":         difficulty,
			"consecutive_wins":   state.ConsecutiveWins,
			"consecutive_losses": state.ConsecutiveLosses,
			"privacy_epoch":      state.PrivacyEpoch,
			"input_sequence":     state.InputSequence,
		},
		"role_queue":   roleQueue,
		"ballot_queue": ballotQueue,
	}, nil
}

func ballotedPayload(ballot domain.Ballot) map[string]any {
	return map[string]any{
		"actor_key": string(ballot.ActorKey),
		"option":    ballot.Option.String(),
		"tick":      int64(ballot.Tick),
		"input_seq": ballot.InputSeq,
	}
}

// optionalTick encodes the explicit nullable last_change_tick.
func optionalTick(tick *int) any {
	if tick == nil {
		return nil
	}
	return int64(*tick)
}

// optionalVoteOption encodes "no selection" as JSON null.
func optionalVoteOption(option domain.VoteOption) any {
	if option == domain.VoteOptionNone {
		return nil
	}
	return option.String()
}
