package reducer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/simulation"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
)

var ErrCheckpointConflict = errors.New("reducer: conflicting checkpoint at the same revision")

const checkpointFormat = 1

type replayCursor struct {
	Revision  int64  `json:"revision"`
	WaveNo    int64  `json:"wave_no"`
	WaveTick  int    `json:"wave_tick"`
	InputSeq  int64  `json:"input_seq"`
	StreamSeq int64  `json:"stream_seq"`
	Reason    string `json:"reason"`
}

type checkpointEnvelope struct {
	Format          int             `json:"format"`
	SessionID       string          `json:"session_id"`
	VillageID       string          `json:"village_id"`
	RulesVersion    string          `json:"rules_version"`
	CanonicalSchema int             `json:"canonical_schema"`
	Cursor          replayCursor    `json:"cursor"`
	RuntimeDigest   string          `json:"runtime_digest"`
	Runtime         json.RawMessage `json:"runtime"`
	Digest          string          `json:"digest"`
}

func buildCheckpointEnvelope(state domain.SessionState, session sessionRow, revision int64, streamSeq int64, reason, runtimeJSON string) (string, string, error) {
	var runtime runtimeEnvelope
	if err := json.Unmarshal([]byte(runtimeJSON), &runtime); err != nil || runtime.Digest == "" {
		return "", "", fmt.Errorf("%w: runtime envelope", ErrInconsistentState)
	}
	stateHash, err := canonical.Hash(state)
	if err != nil {
		return "", "", err
	}
	envelope := checkpointEnvelope{
		Format: checkpointFormat, SessionID: session.ID, VillageID: session.VillageID,
		RulesVersion: session.RulesVersion, CanonicalSchema: state.Revision.SchemaVersion,
		Cursor: replayCursor{
			Revision: revision, WaveNo: state.Wave.Number, WaveTick: state.Tick(),
			InputSeq: state.InputSequence, StreamSeq: streamSeq, Reason: reason,
		},
		RuntimeDigest: runtime.Digest, Runtime: json.RawMessage(runtimeJSON),
	}
	digest, err := checkpointDigest(envelope)
	if err != nil {
		return "", "", err
	}
	envelope.Digest = digest
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return "", "", err
	}
	return string(encoded), stateHash, nil
}

func checkpointDigest(envelope checkpointEnvelope) (string, error) {
	payload := struct {
		Format          int          `json:"format"`
		SessionID       string       `json:"session_id"`
		VillageID       string       `json:"village_id"`
		RulesVersion    string       `json:"rules_version"`
		CanonicalSchema int          `json:"canonical_schema"`
		Cursor          replayCursor `json:"cursor"`
		RuntimeDigest   string       `json:"runtime_digest"`
	}{
		Format: envelope.Format, SessionID: envelope.SessionID, VillageID: envelope.VillageID,
		RulesVersion: envelope.RulesVersion, CanonicalSchema: envelope.CanonicalSchema,
		Cursor: envelope.Cursor, RuntimeDigest: envelope.RuntimeDigest,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func verifyCheckpointEnvelope(raw []byte, sessionID, villageID, rules string, revision int64, stateHash string, waveNo int64, waveTick int, inputSeq int64) (domain.SessionState, simulation.RoleQueue, []int64, error) {
	var envelope checkpointEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: checkpoint JSON", ErrInconsistentState)
	}
	if envelope.Format != checkpointFormat || envelope.Digest == "" {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: unsupported checkpoint format", ErrInconsistentState)
	}
	digest, err := checkpointDigest(envelope)
	if err != nil || digest != envelope.Digest {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: checkpoint digest", ErrInconsistentState)
	}
	if envelope.SessionID != sessionID || envelope.VillageID != villageID || envelope.RulesVersion != rules {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: checkpoint identity", ErrInconsistentState)
	}
	if envelope.CanonicalSchema != domain.CanonicalSchemaVersion {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: canonical schema", ErrInconsistentState)
	}
	if envelope.Cursor.Revision != revision || envelope.Cursor.WaveNo != waveNo || envelope.Cursor.WaveTick != waveTick || envelope.Cursor.InputSeq != inputSeq {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: checkpoint cursor", ErrInconsistentState)
	}
	state, queue, queuedNext, err := decodeRuntime(envelope.Runtime)
	if err != nil {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, err
	}
	if state.Wave.Number != waveNo || state.Tick() != waveTick || state.InputSequence != inputSeq {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: checkpoint state cursor", ErrInconsistentState)
	}
	actualHash, err := canonical.Hash(state)
	if err != nil || actualHash != stateHash {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: checkpoint state hash", ErrInconsistentState)
	}
	var runtime runtimeEnvelope
	if err := json.Unmarshal(envelope.Runtime, &runtime); err != nil || runtime.Digest != envelope.RuntimeDigest {
		return domain.SessionState{}, simulation.RoleQueue{}, nil, fmt.Errorf("%w: checkpoint runtime digest", ErrInconsistentState)
	}
	return state, queue, queuedNext, nil
}

func (r *Reducer) recordRevision(ctx context.Context, tx writer.FencedTx, session sessionRow, revision int64, operation, stateHash, runtimeJSON string, state domain.SessionState, streamSeq int64) error {
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.reducer_transitions (session_id, revision, operation, wave_no, wave_tick, state_hash)
		VALUES ($1, $2, $3, $4, $5, $6)`, session.ID, revision, operation, state.Wave.Number, state.Tick(), stateHash); err != nil {
		return err
	}
	reason, ok := checkpointReason(state, operation)
	if !ok {
		return nil
	}
	checkpointJSON, stateHash, err := buildCheckpointEnvelope(state, session, revision, streamSeq, reason, runtimeJSON)
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO ttg.checkpoints (session_id, revision, wave_no, wave_tick, input_seq, state_json, state_hash, rules_version)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
		ON CONFLICT (session_id, revision) DO NOTHING`,
		session.ID, revision, state.Wave.Number, state.Tick(), state.InputSequence, checkpointJSON, stateHash, session.RulesVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var existingHash string
	if err := tx.QueryRow(ctx, `SELECT state_hash FROM ttg.checkpoints WHERE session_id = $1 AND revision = $2`, session.ID, revision).Scan(&existingHash); err != nil {
		return err
	}
	if existingHash != stateHash {
		return ErrCheckpointConflict
	}
	return nil
}
