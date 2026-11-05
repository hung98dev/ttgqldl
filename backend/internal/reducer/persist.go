package reducer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/simulation"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
)

func (r *Reducer) persist(
	ctx context.Context,
	tx writer.FencedTx,
	session sessionRow,
	village villageRow,
	oldState, newState domain.SessionState,
	queue simulation.RoleQueue,
	runtimeJSON string,
	events []simulation.Event,
	applied []int64,
	revision int64,
	streamSequence int64,
	stateHash string,
	appliedTick int,
) error {
	_ = village
	pauseJSON, err := json.Marshal(pauseReasonStrings(newState.PauseReasons))
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE ttg.sessions
		SET wave_no = $2, wave_tick = $3, lifecycle = $4, privacy_epoch = $5,
		    last_reduced_tick = $3, last_state_hash = $6, runtime_state = $7::jsonb,
		    reducer_revision = $8, stream_sequence = $9, pause_reasons = $10::jsonb
		WHERE id = $1 AND writer_epoch = $11`,
		session.ID, newState.Wave.Number, newState.Tick(), newState.Lifecycle.String(),
		newState.PrivacyEpoch, stateHash, runtimeJSON, revision, streamSequence, string(pauseJSON), session.WriterEpoch)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return writer.ErrStaleEpoch
	}
	if err := r.check("after_session"); err != nil {
		return err
	}

	tag, err = tx.Exec(ctx, `UPDATE ttg.villages
		SET level = $2, wall_tier = $3, wall_hp = $4, townhall_hp = $5, wood = $6,
		    stone = $7, food = $8, construction_work = $9, total_wins = $10,
		    consecutive_wins = $11, consecutive_losses = $12, updated_at = clock_timestamp()
		WHERE id = $1`,
		session.VillageID, newState.Village.Level, newState.Village.WallTier, newState.Village.WallHP,
		newState.Village.TownhallHP, newState.Village.Resources.Wood, newState.Village.Resources.Stone,
		newState.Village.Resources.Food, newState.Village.ConstructionWork, newState.Village.TotalWins,
		newState.ConsecutiveWins, newState.ConsecutiveLosses)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return writer.ErrVillageNotFound
	}
	if err := r.check("after_village"); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM ttg.wave_members WHERE session_id = $1 AND wave_no = $2`, session.ID, newState.Wave.Number); err != nil {
		return err
	}
	for _, member := range newState.Members {
		if _, err := tx.Exec(ctx, `INSERT INTO ttg.wave_members
			(session_id, wave_no, actor_key, role, joined_tick, last_change_tick, first_switch_used, xp_awarded)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (session_id, wave_no, actor_key) DO UPDATE
			SET role = EXCLUDED.role, joined_tick = EXCLUDED.joined_tick,
			    last_change_tick = EXCLUDED.last_change_tick,
			    first_switch_used = EXCLUDED.first_switch_used,
			    xp_awarded = EXCLUDED.xp_awarded`,
			session.ID, newState.Wave.Number, string(member.ActorKey), member.Role.String(),
			member.JoinedTick, member.LastChangeTick, member.FirstSwitchUsed, member.XPAwarded); err != nil {
			return err
		}
	}
	if err := r.check("after_members"); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM ttg.session_player_progress WHERE session_id = $1`, session.ID); err != nil {
		return err
	}
	for _, progress := range newState.XP {
		if _, err := tx.Exec(ctx, `INSERT INTO ttg.session_player_progress (session_id, actor_key, session_xp)
			VALUES ($1, $2, $3) ON CONFLICT (session_id, actor_key) DO UPDATE SET session_xp = EXCLUDED.session_xp`,
			session.ID, string(progress.ActorKey), progress.XP); err != nil {
			return err
		}
	}
	if err := r.check("after_xp"); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM ttg.ballots WHERE session_id = $1 AND wave_no = $2`, session.ID, newState.Wave.Number); err != nil {
		return err
	}
	for _, ballot := range newState.Wave.Ballots {
		if _, err := tx.Exec(ctx, `INSERT INTO ttg.ballots (session_id, wave_no, actor_key, option, input_seq)
			VALUES ($1, $2, $3, $4, $5)`,
			session.ID, newState.Wave.Number, string(ballot.ActorKey), ballot.Option.String(), ballot.InputSeq); err != nil {
			return err
		}
	}
	if err := r.check("after_ballots"); err != nil {
		return err
	}

	for index, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO ttg.domain_events (session_id, revision, event_index, event_type, payload)
			VALUES ($1, $2, $3, $4, $5::jsonb)`,
			session.ID, revision, index, string(event.Type), string(payload)); err != nil {
			return err
		}
	}
	if err := r.check("after_domain_events"); err != nil {
		return err
	}

	for index, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO ttg.stream_outbox (session_id, stream_seq, revision, type, payload)
			VALUES ($1, $2, $3, $4, $5::jsonb)`,
			session.ID, session.StreamSequence+int64(index)+1, revision, string(event.Type), string(payload)); err != nil {
			return err
		}
	}
	if err := r.check("after_outbox"); err != nil {
		return err
	}

	if len(applied) > 0 {
		tag, err := tx.Exec(ctx, `UPDATE ttg.input_events SET applied_at_tick = $3
			WHERE session_id = $1 AND input_seq = ANY($2) AND applied_at_tick IS NULL`,
			session.ID, applied, appliedTick)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != int64(len(applied)) {
			return fmt.Errorf("%w: applied input count %d, expected %d", ErrInconsistentState, tag.RowsAffected(), len(applied))
		}
	}
	if err := r.check("after_applied"); err != nil {
		return err
	}
	return nil
}

func pauseReasonStrings(reasons []domain.PauseReason) []string {
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, reason.String())
	}
	return out
}
