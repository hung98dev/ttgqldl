package reducer

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/simulation"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
)

var ErrPrivacyActor = errors.New("reducer: invalid privacy actor")

// Anonymize commits a privacy boundary without advancing the wave tick. The erased
// actor(s) are removed from the complete private continuation state, all older
// checkpoints are retired, and a new checkpoint at revision+1 becomes the minimum
// trusted replay boundary. Inputs for the erased actors are removed only in the same
// fenced transaction after the new checkpoint has been written.
func (r *Reducer) Anonymize(ctx context.Context, lease writer.Lease, sessionID string, actorKeys []string) (Result, error) {
	if len(actorKeys) == 0 {
		return Result{}, ErrPrivacyActor
	}
	actors := make(map[string]struct{}, len(actorKeys))
	for _, key := range actorKeys {
		if !domain.ActorKey(key).Valid() {
			return Result{}, ErrPrivacyActor
		}
		actors[key] = struct{}{}
	}
	var result Result
	var expectedRevision int64
	var expectedHash string
	err := r.fence.WithFence(ctx, lease, sessionID, func(ctx context.Context, tx writer.FencedTx) error {
		session, err := loadSession(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if session.WriterEpoch != lease.Epoch {
			return writer.ErrStaleEpoch
		}
		if session.RulesVersion != domain.RulesVersion {
			return ErrInconsistentState
		}
		village, err := loadVillage(ctx, tx, session.VillageID)
		if err != nil {
			return err
		}
		state, queue, queuedNext, err := loadRuntimeState(session)
		if err != nil {
			return err
		}
		if err := verifySessionState(session, state); err != nil {
			return err
		}
		oldState := state.Clone()
		state = filterActorState(state, actors)
		queue = filterActorQueue(queue, actors)
		if len(queuedNext) > 0 {
			rows, err := tx.Query(ctx, `SELECT input_seq FROM ttg.input_events WHERE session_id = $1 AND actor_key = ANY($2)`, session.ID, actorKeys)
			if err != nil {
				return err
			}
			remove := make(map[int64]struct{})
			for rows.Next() {
				var seq int64
				if err := rows.Scan(&seq); err != nil {
					rows.Close()
					return err
				}
				remove[seq] = struct{}{}
			}
			rows.Close()
			filtered := queuedNext[:0]
			for _, seq := range queuedNext {
				if _, drop := remove[seq]; !drop {
					filtered = append(filtered, seq)
				}
			}
			queuedNext = normalizeSequences(filtered)
		}
		state.PrivacyEpoch++
		if err := state.Validate(); err != nil {
			return fmt.Errorf("%w: anonymized state: %v", ErrInconsistentState, err)
		}
		revision, err := nextRevision(session.Revision)
		if err != nil {
			return err
		}
		state.InputSequence = session.InputSequence
		state.PrivacyEpoch = session.PrivacyEpoch + 1
		stateHash, err := canonical.Hash(state)
		if err != nil {
			return err
		}
		runtimeJSON, err := encodeRuntime(state, queue, queuedNext)
		if err != nil {
			return err
		}
		streamSequence := session.StreamSequence
		if err := r.persist(ctx, tx, session, village, oldState, state, queue, runtimeJSON, nil, nil, revision, streamSequence, stateHash, state.Tick()); err != nil {
			return err
		}
		if err := r.recordRevision(ctx, tx, session, revision, "privacy", stateHash, runtimeJSON, state, streamSequence); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM ttg.checkpoints WHERE session_id = $1 AND revision < $2`, session.ID, revision); err != nil {
			return err
		}
		for _, actor := range actorKeys {
			if _, err := tx.Exec(ctx, `DELETE FROM ttg.input_receipts WHERE session_id = $1 AND input_seq IN (SELECT input_seq FROM ttg.input_events WHERE session_id = $1 AND actor_key = $2)`, session.ID, actor); err != nil {
				return err
			}
			for _, statement := range []string{
				`DELETE FROM ttg.input_events WHERE session_id = $1 AND actor_key = $2`,
				`DELETE FROM ttg.session_player_progress WHERE session_id = $1 AND actor_key = $2`,
				`DELETE FROM ttg.wave_members WHERE session_id = $1 AND actor_key = $2`,
				`DELETE FROM ttg.ballots WHERE session_id = $1 AND actor_key = $2`,
			} {
				if _, err := tx.Exec(ctx, statement, session.ID, actor); err != nil {
					return err
				}
			}
			for _, statement := range []string{
				`DELETE FROM ttg.actor_identity_aliases WHERE village_id = $1 AND actor_key = $2`,
				`DELETE FROM ttg.players WHERE village_id = $1 AND actor_key = $2`,
			} {
				if _, err := tx.Exec(ctx, statement, session.VillageID, actor); err != nil {
					return err
				}
			}
		}
		expectedRevision = revision
		expectedHash = stateHash
		result = Result{Revision: revision, WaveNo: state.Wave.Number, Tick: state.Tick(), StateHash: stateHash}
		return nil
	})
	if err != nil {
		if errors.Is(err, writer.ErrCommitUnknown) {
			return r.reconcile(ctx, sessionID, expectedRevision, expectedHash, true, result)
		}
		return Result{}, err
	}
	return result, nil
}

func filterActorState(state domain.SessionState, actors map[string]struct{}) domain.SessionState {
	members := state.Members[:0]
	for _, member := range state.Members {
		if _, drop := actors[string(member.ActorKey)]; !drop {
			members = append(members, member)
		}
	}
	state.Members = members
	pending := state.Pending[:0]
	for _, member := range state.Pending {
		if _, drop := actors[string(member.ActorKey)]; !drop {
			pending = append(pending, member)
		}
	}
	state.Pending = pending
	xp := state.XP[:0]
	for _, row := range state.XP {
		if _, drop := actors[string(row.ActorKey)]; !drop {
			xp = append(xp, row)
		}
	}
	state.XP = xp
	ballots := state.Wave.Ballots[:0]
	for _, ballot := range state.Wave.Ballots {
		if _, drop := actors[string(ballot.ActorKey)]; !drop {
			ballots = append(ballots, ballot)
		}
	}
	state.Wave.Ballots = ballots
	counts := domain.VoteCounts{}
	for _, ballot := range ballots {
		switch ballot.Option {
		case domain.VoteOptionA:
			counts.A++
		case domain.VoteOptionB:
			counts.B++
		case domain.VoteOptionC:
			counts.C++
		}
	}
	state.Wave.Vote.Counts = counts
	domain.SortWaveMembers(state.Members)
	domain.SortPendingMembers(state.Pending)
	domain.SortPlayerXP(state.XP)
	domain.SortBallots(state.Wave.Ballots)
	return state
}

func filterActorQueue(queue simulation.RoleQueue, actors map[string]struct{}) simulation.RoleQueue {
	entries := queue.Entries[:0]
	for _, entry := range queue.Entries {
		if _, drop := actors[string(entry.ActorKey)]; !drop {
			entries = append(entries, entry)
		}
	}
	queue.Entries = entries
	ballots := queue.Ballots[:0]
	for _, ballot := range queue.Ballots {
		if _, drop := actors[string(ballot.ActorKey)]; !drop {
			ballots = append(ballots, ballot)
		}
	}
	queue.Ballots = ballots
	return queue
}

func sortedActorKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
