package reducer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/commands"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
	"github.com/hung98dev/ttgqldl/backend/internal/ingest"
	"github.com/hung98dev/ttgqldl/backend/internal/simulation"
	"github.com/hung98dev/ttgqldl/backend/internal/storage"
	"github.com/hung98dev/ttgqldl/backend/internal/writer"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func testPool(t *testing.T) *storage.Pool {
	t.Helper()
	dsn := os.Getenv("TTG_TEST_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		if os.Getenv("TTG_REQUIRE_DATABASE") == "1" {
			t.Fatal("TTG_TEST_DATABASE_URL is required but not set")
		}
		t.Skip("TTG_TEST_DATABASE_URL is not set")
	}
	pool, err := storage.Open(context.Background(), storage.Options{DSN: dsn, MaxConns: 8, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type testSession struct {
	VillageID string
	SessionID string
	Manager   *writer.Manager
	Lease     writer.Lease
}

func newTestSession(t *testing.T, pool *storage.Pool, lifecycle string, leaseDuration time.Duration) testSession {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var villageID, sessionID string
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.villages (id, broadcaster_key)
		VALUES (gen_random_uuid(), md5(random()::text) || md5(random()::text)) RETURNING id::text`).Scan(&villageID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO ttg.sessions (id, village_id, mode, lifecycle, rules_version)
		VALUES (gen_random_uuid(), $1, 'mock', $2, 'ttg-rules-v1') RETURNING id::text`, villageID, lifecycle).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		tx, err := pool.Begin(cleanupCtx)
		if err != nil {
			return
		}
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.writer_leases WHERE village_id = $1`, villageID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.stream_outbox WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.domain_events WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.checkpoints WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.reducer_transitions WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.input_receipts WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.input_events WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.wave_members WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.ballots WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.session_player_progress WHERE session_id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.sessions WHERE id = $1`, sessionID)
		_, _ = tx.Exec(cleanupCtx, `DELETE FROM ttg.villages WHERE id = $1`, villageID)
		_ = tx.Commit(cleanupCtx)
	})
	manager, err := writer.New(pool, "reducer-instance", writer.Config{LeaseDuration: leaseDuration})
	if err != nil {
		t.Fatal(err)
	}
	lease, err := manager.Acquire(ctx, villageID)
	if err != nil {
		t.Fatal(err)
	}
	return testSession{VillageID: villageID, SessionID: sessionID, Manager: manager, Lease: lease}
}

func insertInput(t *testing.T, pool *storage.Pool, sessionID string, seq int64, acceptedTick, scheduledTick int32, waveNo int64, actor, kind, payload string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO ttg.input_events
		(session_id, input_seq, accepted_tick, scheduled_tick, wave_no, actor_key, kind, canonical_payload, rules_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, 'ttg-rules-v1')`,
		sessionID, seq, acceptedTick, scheduledTick, waveNo, actor, kind, payload); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE ttg.sessions SET input_sequence = GREATEST(input_sequence, $2) WHERE id = $1`, sessionID, seq); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func insertRoleInput(t *testing.T, pool *storage.Pool, sessionID string, seq int64, acceptedTick, scheduledTick int32, waveNo int64, actor string, role domain.Role) {
	t.Helper()
	payload, err := json.Marshal(struct {
		Kind string `json:"kind"`
		Role string `json:"role"`
	}{Kind: "role", Role: role.String()})
	if err != nil {
		t.Fatal(err)
	}
	insertInput(t, pool, sessionID, seq, acceptedTick, scheduledTick, waveNo, actor, "role", string(payload))
}

func insertVoteInput(t *testing.T, pool *storage.Pool, sessionID string, seq int64, acceptedTick, scheduledTick int32, waveNo int64, actor string, option domain.VoteOption) {
	t.Helper()
	payload, err := json.Marshal(struct {
		Kind   string `json:"kind"`
		Ballot string `json:"ballot"`
	}{Kind: "vote", Ballot: option.String()})
	if err != nil {
		t.Fatal(err)
	}
	insertInput(t, pool, sessionID, seq, acceptedTick, scheduledTick, waveNo, actor, "vote", string(payload))
}

func seedRuntime(t *testing.T, pool *storage.Pool, sessionID string, state domain.SessionState, queue simulation.RoleQueue, queuedNext []int64, revision int64) {
	t.Helper()
	runtimeJSON, err := encodeRuntime(state, queue, queuedNext)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := canonical.Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE ttg.sessions
		SET wave_no = $2, wave_tick = $3, lifecycle = $4, last_reduced_tick = $3,
		    runtime_state = $5::jsonb, last_state_hash = $6, reducer_revision = $7
		WHERE id = $1`, sessionID, state.Wave.Number, state.Tick(), state.Lifecycle.String(), runtimeJSON, hash, revision); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func actorKey(letter string) string {
	return strings.Repeat(letter, 64)
}

func startedState(t *testing.T, villageID, sessionID, actor string) (domain.SessionState, simulation.RoleQueue) {
	t.Helper()
	state, err := domain.NewInitialState(domain.VillageID(villageID), domain.SessionID(sessionID), domain.SourceModeMock)
	if err != nil {
		t.Fatal(err)
	}
	command := simulation.RoleCommand{
		InputSeq: 1, ActorKey: domain.ActorKey(actor), Role: domain.RoleBuilder,
		TargetWave: 1, ScheduledTick: 0, AcceptedTick: -1,
	}
	state, queue, _, err := simulation.AcceptRoleCommands(state, simulation.RoleQueue{}, []simulation.RoleCommand{command})
	if err != nil {
		t.Fatal(err)
	}
	next, nextQueue, _, _, err := simulation.AdvanceWithRoles(state, queue, simulation.Input{
		Kind:         simulation.KindStartWave,
		Registration: &simulation.Registration{ActorKey: command.ActorKey, Role: command.Role},
	})
	if err != nil {
		t.Fatal(err)
	}
	return next, nextQueue
}

func advanceState(t *testing.T, state domain.SessionState, queue simulation.RoleQueue, target int) (domain.SessionState, simulation.RoleQueue) {
	t.Helper()
	for state.Tick() < target {
		next, nextQueue, _, _, err := simulation.AdvanceWithRoles(state, queue, simulation.Input{Kind: simulation.KindAdvanceTick})
		if err != nil {
			t.Fatalf("advance to %d at %d: %v", target, state.Tick(), err)
		}
		state, queue = next, nextQueue
	}
	return state, queue
}

func countRows(t *testing.T, pool *storage.Pool, statement string, args ...any) int64 {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var value int64
	if err := tx.QueryRow(ctx, statement, args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func sessionValue[T any](t *testing.T, pool *storage.Pool, statement string, args ...any) T {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var value T
	if err := tx.QueryRow(ctx, statement, args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestFirstJoinIsAtomicAndStartsAtTickZero(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	actor := actorKey("a")
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actor, domain.RoleBuilder)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	result, err := reducer.Step(context.Background(), session.Lease, session.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Revision != 1 || result.Tick != 0 || result.WaveNo != 1 {
		t.Fatalf("result = %+v", result)
	}
	if len(result.Applied) != 1 || result.Applied[0] != 1 {
		t.Fatalf("applied = %v", result.Applied)
	}
	if got := sessionValue[int32](t, pool, `SELECT wave_tick FROM ttg.sessions WHERE id = $1`, session.SessionID); got != 0 {
		t.Fatalf("wave_tick = %d", got)
	}
	if got := sessionValue[int64](t, pool, `SELECT reducer_revision FROM ttg.sessions WHERE id = $1`, session.SessionID); got != 1 {
		t.Fatalf("revision = %d", got)
	}
	if got := sessionValue[int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 1`, session.SessionID); got != 0 {
		t.Fatalf("applied_at_tick = %d", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.wave_members WHERE session_id = $1 AND wave_no = 1`, session.SessionID); got != 1 {
		t.Fatalf("members = %d", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.domain_events WHERE session_id = $1 AND revision = 1`, session.SessionID); got == 0 {
		t.Fatal("no domain events")
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1 AND revision = 1`, session.SessionID); got == 0 {
		t.Fatal("no outbox rows")
	}
	if got := sessionValue[int64](t, pool, `SELECT stream_sequence FROM ttg.sessions WHERE id = $1`, session.SessionID); got != countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1`, session.SessionID) {
		t.Fatalf("stream_sequence = %d", got)
	}
	if got := sessionValue[string](t, pool, `SELECT last_state_hash FROM ttg.sessions WHERE id = $1`, session.SessionID); len(got) != 64 {
		t.Fatalf("state hash = %q", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1 AND payload::text LIKE $2`, session.SessionID, "%"+actor+"%"); got != 0 {
		t.Fatalf("outbox leaked actor key: %d", got)
	}
	var runtimeJSON []byte
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `SELECT runtime_state FROM ttg.sessions WHERE id = $1`, session.SessionID).Scan(&runtimeJSON); err != nil {
		t.Fatal(err)
	}
	state, _, _, err := decodeRuntime(runtimeJSON)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := canonical.Hash(state)
	if err != nil {
		t.Fatal(err)
	}
	if hash != sessionValue[string](t, pool, `SELECT last_state_hash FROM ttg.sessions WHERE id = $1`, session.SessionID) {
		t.Fatal("materialized hash does not match runtime state")
	}
}

func TestQuietTickAdvancesOnlyAfterCommit(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actorKey("a"), domain.RoleBuilder)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	result, err := reducer.Step(context.Background(), session.Lease, session.SessionID)
	if err != nil {
		t.Fatalf("quiet step: %v (unwrapped: %v)", err, errors.Unwrap(err))
	}
	if result.Tick != 1 || result.Revision != 2 || len(result.Applied) != 0 {
		t.Fatalf("quiet result = %+v", result)
	}
	if got := sessionValue[int32](t, pool, `SELECT wave_tick FROM ttg.sessions WHERE id = $1`, session.SessionID); got != 1 {
		t.Fatalf("wave_tick = %d", got)
	}
}

func TestRuntimeEnvelopeRoundTrip(t *testing.T) {
	state, queue := startedState(t, "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002", actorKey("a"))
	encoded, err := encodeRuntime(state, queue, nil)
	if err != nil {
		t.Fatal(err)
	}
	var env runtimeEnvelope
	if err := json.Unmarshal([]byte(encoded), &env); err != nil {
		t.Fatal(err)
	}
	digest, err := runtimeDigest(env)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("stored=%s recomputed=%s state=%s", env.Digest, digest, string(env.State[:min(80, len(env.State))]))
	if env.Digest != digest {
		t.Fatal("digest mismatch")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestRoleChangeAndCooldownAreDurable(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	actor := actorKey("a")
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actor, domain.RoleBuilder)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 10; tick++ {
		if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
			t.Fatal(err)
		}
	}
	insertRoleInput(t, pool, session.SessionID, 2, 10, 11, 1, actor, domain.RoleGuard)
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := sessionValue[string](t, pool, `SELECT role FROM ttg.wave_members WHERE session_id = $1 AND wave_no = 1 AND actor_key = $2`, session.SessionID, actor); got != "guard" {
		t.Fatalf("role after first switch = %q", got)
	}
	if got := sessionValue[int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 2`, session.SessionID); got != 11 {
		t.Fatalf("first switch applied = %d", got)
	}
	insertRoleInput(t, pool, session.SessionID, 3, 11, 12, 1, actor, domain.RoleFighter)
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := sessionValue[string](t, pool, `SELECT role FROM ttg.wave_members WHERE session_id = $1 AND wave_no = 1 AND actor_key = $2`, session.SessionID, actor); got != "guard" {
		t.Fatalf("cooldown switch changed role to %q", got)
	}
	if got := sessionValue[int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 3`, session.SessionID); got != 12 {
		t.Fatalf("cooldown input applied = %d", got)
	}
}

func voteStateAt799(t *testing.T, villageID, sessionID, actor string) domain.SessionState {
	t.Helper()
	state, _ := startedState(t, villageID, sessionID, actor)
	phase, err := domain.PhaseForTick(799)
	if err != nil {
		t.Fatal(err)
	}
	state.Wave.Number = 2
	state.Wave.Tick = 799
	state.Wave.Phase = phase
	state.Wave.Result = domain.WaveResultPending
	state.Wave.Vote = domain.VoteState{}
	state.Wave.Ballots = []domain.Ballot{}
	state.Wave.Pools = []domain.EnemyPool{}
	state.Wave.Boss = domain.BossPool{}
	state.Wave.BossIncoming = false
	state.Lifecycle = domain.LifecycleRunning
	state.Members = []domain.WaveMember{{ActorKey: domain.ActorKey(actor), Role: domain.RoleBuilder, JoinedTick: 0}}
	state.Pending = []domain.PendingMember{}
	state.XP = []domain.PlayerXP{{ActorKey: domain.ActorKey(actor), XP: 0}}
	state.InputSequence = 1
	if err := state.Validate(); err != nil {
		t.Fatalf("vote fixture state: %v", err)
	}
	return state
}

func TestVoteLatestBallotReplacesAndClosesOnce(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "running", 2*time.Second)
	actor := actorKey("a")
	state := voteStateAt799(t, session.VillageID, session.SessionID, actor)
	seedRuntime(t, pool, session.SessionID, state, simulation.RoleQueue{}, nil, 0)
	insertVoteInput(t, pool, session.SessionID, 2, 799, 800, 2, actor, domain.VoteOptionA)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.ballots WHERE session_id = $1 AND wave_no = 2 AND actor_key = $2 AND option = 'A'`, session.SessionID, actor); got != 1 {
		t.Fatalf("ballot A rows = %d", got)
	}
	if got := sessionValue[int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 2`, session.SessionID); got != 800 {
		t.Fatalf("ballot applied = %d", got)
	}
	insertVoteInput(t, pool, session.SessionID, 3, 800, 801, 2, actor, domain.VoteOptionB)
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := sessionValue[string](t, pool, `SELECT option FROM ttg.ballots WHERE session_id = $1 AND wave_no = 2 AND actor_key = $2`, session.SessionID, actor); got != "B" {
		t.Fatalf("latest ballot = %q", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.ballots WHERE session_id = $1 AND wave_no = 2`, session.SessionID); got != 1 {
		t.Fatalf("ballot count = %d", got)
	}
}

func TestFutureWaveInput50WaitsWhileInput51Applies(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "running", 30*time.Second)
	actorA, actorB := actorKey("a"), actorKey("b")
	state, queue := startedState(t, session.VillageID, session.SessionID, actorA)
	state, queue = advanceState(t, state, queue, 2799)
	seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
	// Input 50 has the lower sequence but is accepted at the tick-2800 RESULTS
	// boundary and targets the NEXT wave. Input 51 was accepted at 2799 and is
	// effective at 2800, so it is applied first without a global applied watermark.
	insertRoleInput(t, pool, session.SessionID, 50, 2800, 0, 2, actorB, domain.RoleGuard)
	insertRoleInput(t, pool, session.SessionID, 51, 2799, 2800, 1, actorA, domain.RoleGuard)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	result, err := reducer.Step(context.Background(), session.Lease, session.SessionID)
	if err != nil {
		t.Fatalf("future step: %v (unwrapped: %v)", err, errors.Unwrap(err))
	}
	if result.Tick != 2800 || len(result.Applied) != 1 || result.Applied[0] != 51 {
		t.Fatalf("first result = %+v", result)
	}
	if got := sessionValue[*int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 50`, session.SessionID); got != nil {
		t.Fatalf("input 50 applied too early: %v", got)
	}
	if got := sessionValue[*int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 51`, session.SessionID); got == nil || *got != 2800 {
		t.Fatalf("input 51 applied = %v", got)
	}
	for result.Tick < 2999 {
		result, err = reducer.Step(context.Background(), session.Lease, session.SessionID)
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err = reducer.Step(context.Background(), session.Lease, session.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tick != 3000 {
		t.Fatalf("expected tick 3000, got %d", result.Tick)
	}
	if got := sessionValue[*int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 50`, session.SessionID); got != nil {
		t.Fatalf("input 50 applied before rollover: %v", got)
	}
	rollover, err := reducer.Step(context.Background(), session.Lease, session.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if rollover.WaveNo != 2 || rollover.Tick != 0 {
		t.Fatalf("rollover = %+v", rollover)
	}
	if got := sessionValue[*int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 50`, session.SessionID); got == nil || *got != 3000 {
		t.Fatalf("input 50 applied at rollover = %v", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.wave_members WHERE session_id = $1 AND wave_no = 2 AND actor_key = $2`, session.SessionID, actorB); got != 1 {
		t.Fatalf("promoted member = %d", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.checkpoints WHERE session_id = $1 AND wave_no = 2 AND wave_tick = 0`, session.SessionID); got != 1 {
		t.Fatalf("wave-start checkpoint = %d", got)
	}
	replay, err := reducer.VerifyReplay(context.Background(), session.SessionID, 0)
	if err != nil {
		t.Fatalf("replay across pending input/rollover: %v", err)
	}
	if replay.ToRevision == 0 || replay.StateHash == "" {
		t.Fatalf("replay result = %+v", replay)
	}
}

func TestFailureInjectionRollsBackEveryStage(t *testing.T) {
	stages := []string{"after_load", "before_select", "after_select", "after_apply", "after_session", "after_village", "after_members", "after_xp", "after_ballots", "after_domain_events", "after_outbox", "after_applied", "after_checkpoint", "before_commit"}
	pool := testPool(t)
	for _, stage := range stages {
		t.Run(stage, func(t *testing.T) {
			session := newTestSession(t, pool, "waiting", 2*time.Second)
			insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actorKey("a"), domain.RoleBuilder)
			reducer, err := New(session.Manager, pool)
			if err != nil {
				t.Fatal(err)
			}
			reducer.fault = func(got string) error {
				if got == stage {
					return errors.New("injected failure")
				}
				return nil
			}
			if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err == nil {
				t.Fatalf("stage %s did not fail", stage)
			}
			if got := sessionValue[int32](t, pool, `SELECT wave_tick FROM ttg.sessions WHERE id = $1`, session.SessionID); got != -1 {
				t.Fatalf("wave_tick = %d", got)
			}
			if got := sessionValue[int64](t, pool, `SELECT reducer_revision FROM ttg.sessions WHERE id = $1`, session.SessionID); got != 0 {
				t.Fatalf("revision = %d", got)
			}
			if got := countRows(t, pool, `SELECT count(*) FROM ttg.wave_members WHERE session_id = $1`, session.SessionID); got != 0 {
				t.Fatalf("members = %d", got)
			}
			if got := countRows(t, pool, `SELECT count(*) FROM ttg.domain_events WHERE session_id = $1`, session.SessionID); got != 0 {
				t.Fatalf("domain events = %d", got)
			}
			if got := countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1`, session.SessionID); got != 0 {
				t.Fatalf("outbox = %d", got)
			}
			if got := sessionValue[*int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 1`, session.SessionID); got != nil {
				t.Fatalf("applied_at_tick = %v", got)
			}
		})
	}
}

func TestStaleWriterCannotAdvanceReducer(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", writer.MinLeaseDuration)
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actorKey("a"), domain.RoleBuilder)
	waitUntil := time.Now().Add(2 * time.Second)
	for time.Now().Before(waitUntil) {
		now := sessionValue[time.Time](t, pool, `SELECT clock_timestamp()`)
		expiry := sessionValue[time.Time](t, pool, `SELECT leased_until FROM ttg.writer_leases WHERE village_id = $1`, session.VillageID)
		if now.After(expiry) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	ownerB, err := writer.New(pool, "reducer-instance-b", writer.Config{LeaseDuration: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	leaseB, err := ownerB.Acquire(context.Background(), session.VillageID)
	if err != nil {
		t.Fatal(err)
	}
	if leaseB.Epoch <= session.Lease.Epoch {
		t.Fatalf("takeover epoch = %d", leaseB.Epoch)
	}
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err == nil {
		t.Fatal("stale writer advanced a tick")
	}
	if got := sessionValue[int32](t, pool, `SELECT wave_tick FROM ttg.sessions WHERE id = $1`, session.SessionID); got != -1 {
		t.Fatalf("stale writer changed tick to %d", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1`, session.SessionID); got != 0 {
		t.Fatalf("stale writer emitted outbox rows: %d", got)
	}
}

type rawFencedTx struct{ tx pgx.Tx }

func (f rawFencedTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return f.tx.Exec(ctx, sql, args...)
}
func (f rawFencedTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return f.tx.Query(ctx, sql, args...)
}
func (f rawFencedTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return f.tx.QueryRow(ctx, sql, args...)
}

type commitUnknownFencer struct {
	pool   *storage.Pool
	commit bool
}

func (f *commitUnknownFencer) WithFence(ctx context.Context, _ writer.Lease, _ string, fn func(context.Context, writer.FencedTx) error) error {
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return err
	}
	if err := fn(ctx, rawFencedTx{tx: tx}); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if !f.commit {
		_ = tx.Rollback(ctx)
		return writer.ErrCommitUnknown
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return writer.ErrCommitUnknown
}

func TestAmbiguousCommitReconcilesDurableResult(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actorKey("a"), domain.RoleBuilder)
	reducer, err := New(&commitUnknownFencer{pool: pool, commit: true}, pool)
	if err != nil {
		t.Fatal(err)
	}
	result, err := reducer.Step(context.Background(), session.Lease, session.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Reconciled || result.Revision != 1 || result.Tick != 0 {
		t.Fatalf("reconciled result = %+v", result)
	}
	if got := sessionValue[int32](t, pool, `SELECT wave_tick FROM ttg.sessions WHERE id = $1`, session.SessionID); got != 0 {
		t.Fatalf("committed tick = %d", got)
	}
}

func TestUncommittedUnknownOutcomeFailsClosed(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actorKey("a"), domain.RoleBuilder)
	reducer, err := New(&commitUnknownFencer{pool: pool, commit: false}, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err == nil {
		t.Fatal("uncommitted unknown outcome reported success")
	}
	if got := sessionValue[int32](t, pool, `SELECT wave_tick FROM ttg.sessions WHERE id = $1`, session.SessionID); got != -1 {
		t.Fatalf("uncommitted unknown changed tick to %d", got)
	}
}

func TestSettlementBoundaryPersistsStateAndEvents(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "running", 2*time.Second)
	actor := actorKey("a")
	state, queue := startedState(t, session.VillageID, session.SessionID, actor)
	state, queue = advanceState(t, state, queue, 2799)
	seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
	insertRoleInput(t, pool, session.SessionID, 2, 2799, 2800, 1, actor, domain.RoleGuard)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	result, err := reducer.Step(context.Background(), session.Lease, session.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tick != 2800 || result.Revision != 1 {
		t.Fatalf("settlement result = %+v", result)
	}
	var runtimeJSON []byte
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `SELECT runtime_state FROM ttg.sessions WHERE id = $1`, session.SessionID).Scan(&runtimeJSON); err != nil {
		t.Fatal(err)
	}
	persisted, _, _, err := decodeRuntime(runtimeJSON)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Wave.Result == domain.WaveResultPending {
		t.Fatal("settlement did not persist a result")
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.domain_events WHERE session_id = $1 AND revision = 1 AND event_type = 'wave_result'`, session.SessionID); got == 0 {
		t.Fatal("no wave_result domain event")
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1 AND revision = 1 AND type = 'wave_result'`, session.SessionID); got == 0 {
		t.Fatal("no wave_result outbox row")
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.checkpoints WHERE session_id = $1 AND revision = 1 AND wave_tick = 2800`, session.SessionID); got != 1 {
		t.Fatalf("settlement checkpoint = %d", got)
	}
}

func advanceToWave(t *testing.T, state domain.SessionState, queue simulation.RoleQueue, targetWave int64) (domain.SessionState, simulation.RoleQueue) {
	t.Helper()
	seq := int64(100)
	for state.Wave.Number < targetWave {
		state, queue = advanceState(t, state, queue, domain.FinalCombatTick)
		command := simulation.RoleCommand{
			InputSeq: seq, ActorKey: domain.ActorKey(actorKey("a")), Role: domain.RoleBuilder,
			TargetWave: state.Wave.Number + 1, ScheduledTick: 0, AcceptedTick: state.Tick(),
		}
		seq++
		next, nextQueue, _, err := simulation.AcceptRoleCommands(state, queue, []simulation.RoleCommand{command})
		if err != nil {
			t.Fatalf("queue next wave at %d: %v", state.Tick(), err)
		}
		state, queue = next, nextQueue
		state, queue = advanceState(t, state, queue, domain.RolloverTick)
		next, nextQueue, _, _, err = simulation.AdvanceWithRoles(state, queue, simulation.Input{Kind: simulation.KindRollover})
		if err != nil {
			t.Fatalf("rollover to wave %d: %v", targetWave, err)
		}
		state, queue = next, nextQueue
	}
	return state, queue
}

func TestBossWaveSpawnsAt2600(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "running", 2*time.Second)
	actor := actorKey("a")
	state, queue := startedState(t, session.VillageID, session.SessionID, actor)
	state, queue = advanceToWave(t, state, queue, 5)
	state, queue = advanceState(t, state, queue, 2599)
	seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	result, err := reducer.Step(context.Background(), session.Lease, session.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if result.WaveNo != 5 || result.Tick != 2600 {
		t.Fatalf("boss step = %+v", result)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.domain_events WHERE session_id = $1 AND revision = 1 AND event_type = 'boss_spawned'`, session.SessionID); got == 0 {
		t.Fatal("no boss_spawned domain event")
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1 AND revision = 1 AND type = 'boss_spawned'`, session.SessionID); got == 0 {
		t.Fatal("no boss_spawned outbox row")
	}
}

func insertRoleBatch(t *testing.T, pool *storage.Pool, sessionID string, startSeq int64, count int, acceptedTick, scheduledTick int32, waveNo int64, actor string, role domain.Role) {
	t.Helper()
	payload, err := json.Marshal(struct {
		Kind string `json:"kind"`
		Role string `json:"role"`
	}{Kind: "role", Role: role.String()})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for index := 0; index < count; index++ {
		if _, err := tx.Exec(ctx, `INSERT INTO ttg.input_events
			(session_id, input_seq, accepted_tick, scheduled_tick, wave_no, actor_key, kind, canonical_payload, rules_version)
			VALUES ($1, $2, $3, $4, $5, $6, 'role', $7::jsonb, 'ttg-rules-v1')`,
			sessionID, startSeq+int64(index), acceptedTick, scheduledTick, waveNo, actor, string(payload)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE ttg.sessions SET input_sequence = GREATEST(input_sequence, $2) WHERE id = $1`, sessionID, startSeq+int64(count)-1); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestThousandDuplicateRolesDoNotInflateMembersOrXP(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	actor := actorKey("a")
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actor, domain.RoleBuilder)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	insertRoleBatch(t, pool, session.SessionID, 2, 1000, 0, 1, 1, actor, domain.RoleBuilder)
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.wave_members WHERE session_id = $1 AND wave_no = 1`, session.SessionID); got != 1 {
		t.Fatalf("members = %d", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.session_player_progress WHERE session_id = $1`, session.SessionID); got > 1 {
		t.Fatalf("XP rows = %d", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.input_events WHERE session_id = $1 AND applied_at_tick = 1`, session.SessionID); got != 1000 {
		t.Fatalf("applied duplicate inputs = %d", got)
	}
}

func TestConcurrentIngestionAndReducerSerialized(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actorKey("a"), domain.RoleBuilder)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	reducer.fault = func(stage string) error {
		if stage == "after_select" {
			close(entered)
			<-release
		}
		return nil
	}
	stepResult := make(chan error, 1)
	go func() {
		_, err := reducer.Step(context.Background(), session.Lease, session.SessionID)
		stepResult <- err
	}()
	<-entered
	appender, err := ingest.NewPostgresAppender(pool)
	if err != nil {
		t.Fatal(err)
	}
	action, err := commands.ParseComment("XÂY")
	if err != nil {
		t.Fatal(err)
	}
	appendResult := make(chan error, 1)
	go func() {
		_, err := appender.AppendSession(context.Background(), session.SessionID, []ingest.DurableCandidate{{
			SessionID: session.SessionID, Provider: "mock", ProviderEventID: "concurrent-1",
			Fingerprint: [32]byte{1}, ActorKey: domain.ActorKey(actorKey("b")), Action: action,
		}})
		appendResult <- err
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		waiting := sessionValue[int64](t, pool, `SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock' AND query ILIKE '%FROM ttg.sessions%'`)
		if waiting > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(release)
	if err := <-stepResult; err != nil {
		t.Fatalf("reducer: %v", err)
	}
	if err := <-appendResult; err != nil {
		t.Fatalf("append: %v", err)
	}
	if got := sessionValue[int32](t, pool, `SELECT accepted_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 2`, session.SessionID); got != 0 {
		t.Fatalf("concurrent input accepted_tick = %d", got)
	}
	reducer.fault = nil
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatalf("second step: %v", err)
	}
	if got := sessionValue[*int32](t, pool, `SELECT applied_at_tick FROM ttg.input_events WHERE session_id = $1 AND input_seq = 2`, session.SessionID); got == nil || *got != 1 {
		t.Fatalf("concurrent input applied = %v", got)
	}
}

func TestMaterializedVillageAndXPPersistInSameCommit(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	actor := actorKey("a")
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actor, domain.RoleBuilder)
	state, queue := startedState(t, session.VillageID, session.SessionID, actor)
	state.Village.Resources.Wood = 123
	state.Village.Resources.Stone = 77
	state.XP = []domain.PlayerXP{{ActorKey: domain.ActorKey(actor), XP: 42}}
	seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
	// Keep the accepted input pending for this fixture; the state is at tick0.
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	// The input was accepted at -1 but the fixture is at tick0; remove it so this
	// transition is a quiet tick that materializes the seeded state.
	dbExec := func(statement string, args ...any) {
		ctx := context.Background()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, statement, args...); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	dbExec(`UPDATE ttg.input_events SET applied_at_tick = 0 WHERE session_id = $1 AND input_seq = 1`, session.SessionID)
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := sessionValue[int64](t, pool, `SELECT wood FROM ttg.villages WHERE id = $1`, session.VillageID); got != 123 {
		t.Fatalf("wood = %d", got)
	}
	if got := sessionValue[int64](t, pool, `SELECT session_xp FROM ttg.session_player_progress WHERE session_id = $1 AND actor_key = $2`, session.SessionID, actor); got != 42 {
		t.Fatalf("xp = %d", got)
	}
}

func TestPausedRuntimeDoesNotAdvanceOrCatchUp(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "running", 2*time.Second)
	actor := actorKey("a")
	state, queue := startedState(t, session.VillageID, session.SessionID, actor)
	paused, pausedQueue, _, _, err := simulation.AdvanceWithRoles(state, queue, simulation.Input{Kind: simulation.KindPause, Reason: domain.PauseOperatorPaused})
	if err != nil {
		t.Fatal(err)
	}
	seedRuntime(t, pool, session.SessionID, paused, pausedQueue, nil, 0)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); !errors.Is(err, ErrNoWork) {
		t.Fatalf("paused step = %v", err)
	}
	if got := sessionValue[int32](t, pool, `SELECT wave_tick FROM ttg.sessions WHERE id = $1`, session.SessionID); got != 0 {
		t.Fatalf("paused tick changed to %d", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1`, session.SessionID); got != 0 {
		t.Fatalf("paused step emitted outbox rows: %d", got)
	}
}

func TestCheckpointAtWaveStartAndReplayMatches(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actorKey("a"), domain.RoleBuilder)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.checkpoints WHERE session_id = $1 AND revision = 1`, session.SessionID); got != 1 {
		t.Fatalf("initial checkpoint rows = %d", got)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.reducer_transitions WHERE session_id = $1`, session.SessionID); got != 2 {
		t.Fatalf("journal rows = %d", got)
	}
	revisionBefore := sessionValue[int64](t, pool, `SELECT reducer_revision FROM ttg.sessions WHERE id = $1`, session.SessionID)
	outboxBefore := countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1`, session.SessionID)
	replay, err := reducer.VerifyReplay(context.Background(), session.SessionID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if replay.CheckpointRevision != 1 || replay.ToRevision != 2 || replay.Transitions != 1 {
		t.Fatalf("replay = %+v", replay)
	}
	hash := sessionValue[string](t, pool, `SELECT last_state_hash FROM ttg.sessions WHERE id = $1`, session.SessionID)
	if replay.StateHash != hash {
		t.Fatalf("replay hash = %s, session = %s", replay.StateHash, hash)
	}
	if got := sessionValue[int64](t, pool, `SELECT reducer_revision FROM ttg.sessions WHERE id = $1`, session.SessionID); got != revisionBefore {
		t.Fatal("replay mutated revision")
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.stream_outbox WHERE session_id = $1`, session.SessionID); got != outboxBefore {
		t.Fatal("replay mutated outbox")
	}
}

func TestPeriodicCheckpointAt600AndReplay(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "running", 30*time.Second)
	actor := actorKey("a")
	state, queue := startedState(t, session.VillageID, session.SessionID, actor)
	state, queue = advanceState(t, state, queue, 599)
	seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.checkpoints WHERE session_id = $1 AND revision = 1 AND wave_tick = 600`, session.SessionID); got != 1 {
		t.Fatalf("periodic checkpoint = %d", got)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	replay, err := reducer.VerifyReplay(context.Background(), session.SessionID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if replay.CheckpointRevision != 1 || replay.Transitions != 1 || replay.ToRevision != 2 {
		t.Fatalf("periodic replay = %+v", replay)
	}
}

func TestHundredDistinctReplayPositions(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "running", 60*time.Second)
	actor := actorKey("a")
	state, queue := startedState(t, session.VillageID, session.SessionID, actor)
	state, queue = advanceState(t, state, queue, 599)
	seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 100; index++ {
		if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
			t.Fatalf("step %d: %v", index, err)
		}
	}
	revisionBefore := sessionValue[int64](t, pool, `SELECT reducer_revision FROM ttg.sessions WHERE id = $1`, session.SessionID)
	verified := 0
	for target := int64(1); target <= revisionBefore; target++ {
		result, err := reducer.VerifyReplay(context.Background(), session.SessionID, target)
		if err != nil {
			t.Fatalf("target %d: %v", target, err)
		}
		if result.ToRevision != target {
			t.Fatalf("target %d result %+v", target, result)
		}
		verified++
	}
	if verified != 100 {
		t.Fatalf("verified positions = %d", verified)
	}
	if got := sessionValue[int64](t, pool, `SELECT reducer_revision FROM ttg.sessions WHERE id = $1`, session.SessionID); got != revisionBefore {
		t.Fatal("100-position verification mutated revision")
	}
}

func TestCheckpointScheduleReasonTable(t *testing.T) {
	state := domain.SessionState{Wave: domain.WaveState{Number: 1, Tick: 0}}
	cases := []struct {
		tick            int
		operation, want string
		ok              bool
	}{
		{0, OperationStart, "initial", true},
		{0, OperationRollover, "wave_start", true},
		{600, OperationAdvance, "periodic", true},
		{1200, OperationAdvance, "periodic", true},
		{1800, OperationAdvance, "periodic", true},
		{2400, OperationAdvance, "periodic", true},
		{2800, OperationAdvance, "settlement", true},
		{3000, OperationAdvance, "control", true},
		{1, OperationAdvance, "", false},
	}
	for _, test := range cases {
		state.Wave.Tick = test.tick
		reason, ok := checkpointReason(state, test.operation)
		if ok != test.ok || reason != test.want {
			t.Fatalf("tick %d op %s = %q/%v, want %q/%v", test.tick, test.operation, reason, ok, test.want, test.ok)
		}
	}
}

func dbExec(t *testing.T, pool *storage.Pool, statement string, args ...any) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, statement, args...); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func checkpointedSession(t *testing.T, pool *storage.Pool, withInput bool) testSession {
	t.Helper()
	session := newTestSession(t, pool, "running", 30*time.Second)
	actor := actorKey("a")
	state, queue := startedState(t, session.VillageID, session.SessionID, actor)
	state, queue = advanceState(t, state, queue, 599)
	seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if withInput {
		insertRoleInput(t, pool, session.SessionID, 2, 600, 601, 1, actor, domain.RoleGuard)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	return session
}

func TestCheckpointCorruptionFailsClosed(t *testing.T) {
	pool := testPool(t)
	cases := []struct {
		name   string
		mutate func(t *testing.T, session testSession)
	}{
		{"state_hash", func(t *testing.T, session testSession) {
			dbExec(t, pool, `UPDATE ttg.checkpoints SET state_hash = repeat('0', 64) WHERE session_id = $1 AND revision = 1`, session.SessionID)
		}},
		{"checkpoint_cursor", func(t *testing.T, session testSession) {
			dbExec(t, pool, `UPDATE ttg.checkpoints SET state_json = jsonb_set(state_json, '{cursor,input_seq}', '999') WHERE session_id = $1 AND revision = 1`, session.SessionID)
		}},
		{"checkpoint_format", func(t *testing.T, session testSession) {
			dbExec(t, pool, `UPDATE ttg.checkpoints SET state_json = jsonb_set(state_json, '{format}', '99') WHERE session_id = $1 AND revision = 1`, session.SessionID)
		}},
		{"checkpoint_rules", func(t *testing.T, session testSession) {
			dbExec(t, pool, `UPDATE ttg.checkpoints SET rules_version = 'bad-rules' WHERE session_id = $1 AND revision = 1`, session.SessionID)
		}},
		{"checkpoint_json", func(t *testing.T, session testSession) {
			dbExec(t, pool, `UPDATE ttg.checkpoints SET state_json = '{}'::jsonb WHERE session_id = $1 AND revision = 1`, session.SessionID)
		}},
		{"missing_journal", func(t *testing.T, session testSession) {
			dbExec(t, pool, `DELETE FROM ttg.reducer_transitions WHERE session_id = $1 AND revision = 2`, session.SessionID)
		}},
		{"journal_hash", func(t *testing.T, session testSession) {
			dbExec(t, pool, `UPDATE ttg.reducer_transitions SET state_hash = repeat('0', 64) WHERE session_id = $1 AND revision = 2`, session.SessionID)
		}},
		{"missing_checkpoint", func(t *testing.T, session testSession) {
			dbExec(t, pool, `DELETE FROM ttg.checkpoints WHERE session_id = $1`, session.SessionID)
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			session := checkpointedSession(t, pool, false)
			test.mutate(t, session)
			reducer, err := New(session.Manager, pool)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := reducer.VerifyReplay(context.Background(), session.SessionID, 2); err == nil {
				t.Fatalf("corruption %s was accepted", test.name)
			}
		})
	}
}

func TestReplayRejectsMissingOrMutatedInput(t *testing.T) {
	pool := testPool(t)
	for _, mutate := range []string{"delete", "payload"} {
		t.Run(mutate, func(t *testing.T) {
			session := checkpointedSession(t, pool, true)
			if mutate == "delete" {
				dbExec(t, pool, `DELETE FROM ttg.input_events WHERE session_id = $1 AND input_seq = 2`, session.SessionID)
			} else {
				dbExec(t, pool, `UPDATE ttg.input_events SET canonical_payload = '{"kind":"role","role":"fighter"}'::jsonb WHERE session_id = $1 AND input_seq = 2`, session.SessionID)
			}
			reducer, err := New(session.Manager, pool)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := reducer.VerifyReplay(context.Background(), session.SessionID, 2); err == nil {
				t.Fatal("inconsistent input history was accepted")
			}
		})
	}
}

func TestReplayFailsWithoutCheckpointOrDatabase(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "waiting", 2*time.Second)
	insertRoleInput(t, pool, session.SessionID, 1, -1, 0, 1, actorKey("a"), domain.RoleBuilder)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	dbExec(t, pool, `DELETE FROM ttg.checkpoints WHERE session_id = $1`, session.SessionID)
	if _, err := reducer.VerifyReplay(context.Background(), session.SessionID, 1); !errors.Is(err, ErrReplayNoCheckpoint) {
		t.Fatalf("no checkpoint error = %v", err)
	}
}

func TestCheckpointSizeAndReplayTiming(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "running", 30*time.Second)
	actor := actorKey("a")
	state, queue := startedState(t, session.VillageID, session.SessionID, actor)
	state, queue = advanceState(t, state, queue, 599)
	seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	stepDuration := time.Since(started)
	size := sessionValue[int64](t, pool, `SELECT octet_length(state_json::text) FROM ttg.checkpoints WHERE session_id = $1 AND revision = 1`, session.SessionID)
	started = time.Now()
	if _, err := reducer.VerifyReplay(context.Background(), session.SessionID, 1); err != nil {
		t.Fatal(err)
	}
	replayDuration := time.Since(started)
	t.Logf("checkpoint revision1 tick600: step=%s replay=%s jsonb_text_bytes=%d", stepDuration.Round(time.Microsecond), replayDuration.Round(time.Microsecond), size)
}

func TestPeriodicCheckpointBoundaries(t *testing.T) {
	pool := testPool(t)
	for _, boundary := range []int{600, 1200, 1800, 2400} {
		t.Run(fmt.Sprintf("tick-%d", boundary), func(t *testing.T) {
			session := newTestSession(t, pool, "running", 30*time.Second)
			actor := actorKey("a")
			state, queue := startedState(t, session.VillageID, session.SessionID, actor)
			state, queue = advanceState(t, state, queue, boundary-1)
			seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
			reducer, err := New(session.Manager, pool)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
				t.Fatal(err)
			}
			if got := countRows(t, pool, `SELECT count(*) FROM ttg.checkpoints WHERE session_id = $1 AND revision = 1 AND wave_tick = $2`, session.SessionID, boundary); got != 1 {
				t.Fatalf("checkpoint at %d = %d", boundary, got)
			}
		})
	}
}

func TestReplayAcrossBossAndSettlement(t *testing.T) {
	pool := testPool(t)
	session := newTestSession(t, pool, "running", 60*time.Second)
	actor := actorKey("a")
	state, queue := startedState(t, session.VillageID, session.SessionID, actor)
	state, queue = advanceToWave(t, state, queue, 5)
	state, queue = advanceState(t, state, queue, 2399)
	seedRuntime(t, pool, session.SessionID, state, queue, nil, 0)
	reducer, err := New(session.Manager, pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
		t.Fatal(err)
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.checkpoints WHERE session_id = $1 AND revision = 1 AND wave_tick = 2400`, session.SessionID); got != 1 {
		t.Fatalf("checkpoint2400 = %d", got)
	}
	for tick := 2400; tick < 2600; tick++ {
		if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
			t.Fatalf("step %d: %v", tick, err)
		}
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.domain_events WHERE session_id = $1 AND event_type = 'boss_spawned'`, session.SessionID); got == 0 {
		t.Fatal("boss_spawned missing before replay")
	}
	for tick := 2600; tick < 2800; tick++ {
		if _, err := reducer.Step(context.Background(), session.Lease, session.SessionID); err != nil {
			t.Fatalf("step %d: %v", tick, err)
		}
	}
	if got := countRows(t, pool, `SELECT count(*) FROM ttg.domain_events WHERE session_id = $1 AND event_type = 'wave_result'`, session.SessionID); got == 0 {
		t.Fatal("wave_result missing before replay")
	}
	replay, err := reducer.VerifyReplayFrom(context.Background(), session.SessionID, 1, 0)
	if err != nil {
		t.Fatalf("boss/settlement replay: %v", err)
	}
	if replay.CheckpointRevision != 1 || replay.Transitions != 400 {
		t.Fatalf("boss replay = %+v", replay)
	}
}
