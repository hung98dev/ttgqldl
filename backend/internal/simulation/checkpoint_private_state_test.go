package simulation

import (
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// collisionAssaultBase is a genuine wave-10 state at tick 2400: the population is
// locked, the vote is closed with two applied ballots, the boss has not spawned and
// ordinary pools remain. It is built through the real reducer, not hand-assigned.
func collisionAssaultBase(t *testing.T) domain.SessionState {
	t.Helper()
	state, err := domain.NewInitialState(villageID, sessionID, domain.SourceModeMock)
	if err != nil {
		t.Fatal(err)
	}
	state.Wave.Number = 10
	if err := state.Validate(); err != nil {
		t.Fatal(err)
	}
	first := roleCommand(1, uniqueActor(1), domain.RoleBuilder, 10, 0)
	accepted, queue, _ := acceptRoles(t, state, RoleQueue{}, first)
	state, queue, _, _, err = AdvanceWithRoles(accepted, queue, Input{
		Kind: KindStartWave, Registration: &Registration{ActorKey: first.ActorKey, Role: first.Role},
	})
	if err != nil {
		t.Fatal(err)
	}
	var commands []RoleCommand
	for index := 2; index <= 10; index++ {
		commands = append(commands, roleCommand(int64(index), uniqueActor(index), domain.RoleBuilder, 10, 1))
	}
	accepted, queue, _ = acceptRoles(t, state, queue, commands...)
	state, queue = advanceWithRoles(t, accepted, queue, 899)
	accepted, queue, _, err = AcceptVoteCommands(state, queue, []VoteCommand{
		{InputSeq: 20, ActorKey: uniqueActor(1), TargetWave: 10, ScheduledTick: 900, Option: domain.VoteOptionA, AcceptedTick: AcceptedTickUnspecified},
		{InputSeq: 21, ActorKey: uniqueActor(2), TargetWave: 10, ScheduledTick: 900, Option: domain.VoteOptionC, AcceptedTick: AcceptedTickUnspecified},
	})
	if err != nil {
		t.Fatal(err)
	}
	state, queue = advanceWithRoles(t, accepted, queue, 2400)
	if !state.Wave.Vote.Closed || len(state.Wave.Ballots) != 2 {
		t.Fatalf("assault base must carry an applied ballot set: %+v", state.Wave.Vote)
	}
	if state.Wave.BossIncoming || state.Wave.Boss.Exists {
		t.Fatalf("assault base boss state = %+v incoming=%t", state.Wave.Boss, state.Wave.BossIncoming)
	}
	if err := state.Validate(); err != nil {
		t.Fatal(err)
	}
	return state
}

// collisionSettledBase is the same wave settled at tick 2800.
func collisionSettledBase(t *testing.T) domain.SessionState {
	t.Helper()
	state := collisionAssaultBase(t)
	state, _ = advanceWithRoles(t, state, RoleQueue{}, domain.FinalCombatTick)
	if state.Wave.Result == domain.WaveResultPending {
		t.Fatal("settled base was not settled")
	}
	if err := state.Validate(); err != nil {
		t.Fatal(err)
	}
	return state
}

// lockedFixtureState builds a locked wave with one fighter and no enemy damage yet.
func lockedFixtureState(t *testing.T, waveNo int64, tick int, anchor, actual int, baseHP int64) domain.SessionState {
	t.Helper()
	state := newState(t)
	state.Wave.Number = waveNo
	state.Wave.Tick = tick
	phase, err := domain.PhaseForTick(tick)
	if err != nil {
		t.Fatal(err)
	}
	state.Wave.Phase = phase
	state.Lifecycle = domain.LifecycleRunning
	state.Members = []domain.WaveMember{{ActorKey: uniqueActor(1), Role: domain.RoleFighter, JoinedTick: 0}}
	pools, err := domain.BuildEnemyPools(baseHP)
	if err != nil {
		t.Fatal(err)
	}
	state.Wave.Pools = pools
	bossMax := int64(0)
	if domain.IsBossWave(waveNo) {
		bossMax, err = domain.BossMaxHP(baseHP)
		if err != nil {
			t.Fatal(err)
		}
	}
	state.Difficulty = domain.Difficulty{
		Level:            1,
		PopulationAnchor: anchor,
		Locked:           true,
		ActualPopulation: actual,
		RecoveryPercent:  100,
		BaseHP:           baseHP,
		MilitiaAttack:    domain.MilitiaAttack(anchor),
		BossMaxHP:        bossMax,
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("locked fixture invalid: %v", err)
	}
	return state
}

// collisionCase is one authoritative field that canonical schema v1 does not cover.
type collisionCase struct {
	name     string
	base     func(t *testing.T) domain.SessionState
	mutate   func(*domain.SessionState)
	behavior string
}

func collisionCases() []collisionCase {
	return []collisionCase{
		{
			name:   "input_sequence",
			base:   collisionAssaultBase,
			mutate: func(state *domain.SessionState) { state.InputSequence = 5 },
			behavior: "not consumed by the W0202-W0208 reducer; it is the accepted-input sequence " +
				"namespace that M03 ingest dedupe and durable replay ordering require",
		},
		{
			name:   "source_mode",
			base:   collisionAssaultBase,
			mutate: func(state *domain.SessionState) { state.Mode = domain.SourceModeReplay },
			behavior: "not consumed by the pure reducer; it gates source admission and audit provenance " +
				"above the reducer",
		},
		{
			name: "applied_ballot_identity",
			base: collisionAssaultBase,
			mutate: func(state *domain.SessionState) {
				// Same option and tick, different accepted input sequence: the tally
				// is unchanged, but the stored identity is what a same-tick re-vote
				// would be compared against by the latest-wins upsert.
				state.Wave.Ballots[0].InputSeq++
			},
			behavior: "the option tally is canonical, but the stored (actor, tick, input_seq) identity " +
				"drives the same-tick latest-wins comparison for a later re-vote",
		},
		{
			name: "militia_attack",
			base: func(t *testing.T) domain.SessionState {
				return lockedFixtureState(t, 7, 1800, 30, 30, 238)
			},
			mutate:   func(state *domain.SessionState) { state.Difficulty.MilitiaAttack = 5 },
			behavior: "combat adds Difficulty.MilitiaAttack to the fighter attack at every combat boundary",
		},
		{
			name: "boss_incoming",
			base: func(t *testing.T) domain.SessionState {
				state := lockedFixtureState(t, 5, 1800, 30, 30, 238)
				for index := range state.Wave.Pools {
					state.Wave.Pools[index].HP = 0
				}
				return state
			},
			mutate:   func(state *domain.SessionState) { state.Wave.BossIncoming = true },
			behavior: "the flag suppresses a duplicate boss_incoming event on every later pre-spawn boundary",
		},
		{
			name: "wave_result",
			base: collisionSettledBase,
			mutate: func(state *domain.SessionState) {
				state.Wave.Result = domain.WaveResultPending
			},
			behavior: "rollover fails closed with ErrUnsettledWave while the result is pending and " +
				"succeeds once it is settled",
		},
		{
			name: "village_level",
			base: collisionAssaultBase,
			mutate: func(state *domain.SessionState) {
				state.Village.Level = 7
			},
			behavior: "persistent progression value; not consumed by the pure reducer, and it must " +
				"still be restored faithfully",
		},
		{
			name: "actual_population",
			base: func(t *testing.T) domain.SessionState {
				// Above the coefficient cap, two truthful counts share the same
				// capped anchor, so the anchor is not a proxy for this field.
				return lockedFixtureState(t, 7, 1800, 10000, 10001, 3848)
			},
			mutate: func(state *domain.SessionState) { state.Difficulty.ActualPopulation = 10002 },
			behavior: "the truthful participant count is reported separately from the capped formula " +
				"anchor; it is not re-read after the lock",
		},
		{
			name: "recovery_percent",
			base: func(t *testing.T) domain.SessionState {
				state := lockedFixtureState(t, 7, 1800, 30, 30, 202)
				state.ConsecutiveLosses = 1
				state.Difficulty.RecoveryPercent = 85
				return state
			},
			mutate: func(state *domain.SessionState) {
				state.Difficulty.RecoveryPercent = 70
			},
			behavior: "derived recovery modifier displayed on the HUD; it is not re-read after the lock",
		},
		{
			name: "boss_max_hp",
			base: func(t *testing.T) domain.SessionState {
				return lockedFixtureState(t, 5, 2400, 30, 30, 238)
			},
			mutate: func(state *domain.SessionState) {
				state.Difficulty.BossMaxHP++
			},
			behavior: "the tick-2600 boss spawn creates exactly Difficulty.BossMaxHP hit points",
		},
	}
}

// TestPrivateStateCollisions reproduces, for every authoritative field that
// canonical v2 (A1 scope) omits, a pair of independently valid states that share the canonical
// hash. The complete private digest must separate them and verification must reject
// a mismatched restore. Only fields whose collision is actually reproduced are
// claimed here.
func TestPrivateStateCollisions(t *testing.T) {
	collisions, separated, rejected, canonicalCollisions := 0, 0, 0, 0
	for _, testCase := range collisionCases() {
		t.Run(testCase.name, func(t *testing.T) {
			base := testCase.base(t)
			mutated := base.Clone()
			testCase.mutate(&mutated)
			if err := base.Validate(); err != nil {
				t.Fatalf("base state invalid: %v", err)
			}
			if err := mutated.Validate(); err != nil {
				t.Fatalf("mutated state invalid: %v", err)
			}

			baseHash, err := canonical.Hash(base)
			if err != nil {
				t.Fatal(err)
			}
			mutatedHash, err := canonical.Hash(mutated)
			if err != nil {
				t.Fatal(err)
			}
			if baseHash != mutatedHash {
				t.Fatalf("the canonical hash already separates %s; the recorded gap is stale", testCase.name)
			}
			canonicalCollisions++

			baseEnvelope, err := BuildCheckpointEnvelope(base, RoleQueue{})
			if err != nil {
				t.Fatal(err)
			}
			mutatedEnvelope, err := BuildCheckpointEnvelope(mutated, RoleQueue{})
			if err != nil {
				t.Fatal(err)
			}
			if baseEnvelope.PrivateDigest == mutatedEnvelope.PrivateDigest {
				t.Fatalf("private digest does not separate %s", testCase.name)
			}
			separated++
			if err := VerifyCheckpointEnvelope(baseEnvelope, mutated, RoleQueue{}); err == nil {
				t.Fatalf("checkpoint for %s accepted a state it does not cover", testCase.name)
			}
			rejected++
			collisions++
			t.Logf("collision %-24s canonical_hash=equal private_digest=differs continuation=%s",
				testCase.name, testCase.behavior)
		})
	}
	if collisions != len(collisionCases()) || separated != collisions || rejected != collisions || canonicalCollisions != collisions {
		t.Fatalf("collision accounting: cases=%d canonical=%d separated=%d rejected=%d",
			len(collisionCases()), canonicalCollisions, separated, rejected)
	}
	t.Logf("private_integrity: collision_cases=%d canonical_collisions=%d private_digest_separations=%d rejected_restores=%d",
		collisions, canonicalCollisions, separated, rejected)
}

// TestPrivateStateCollisionsChangeContinuation proves that the omitted fields are
// not cosmetic: each subtest runs the same deterministic continuation from two
// states that share the canonical hash and shows a different result.
func TestPrivateStateCollisionsChangeContinuation(t *testing.T) {
	t.Run("militia_attack_changes_combat", func(t *testing.T) {
		base := lockedFixtureState(t, 7, 1800, 30, 30, 238)
		mutated := base.Clone()
		mutated.Difficulty.MilitiaAttack = 5
		baseAfter, _ := advanceWithRoles(t, base, RoleQueue{}, 2000)
		mutatedAfter, _ := advanceWithRoles(t, mutated, RoleQueue{}, 2000)
		if ordinaryHP(baseAfter) == ordinaryHP(mutatedAfter) {
			t.Fatalf("militia attack did not change combat: both %d", ordinaryHP(baseAfter))
		}
	})

	t.Run("boss_max_hp_changes_spawn", func(t *testing.T) {
		base := lockedFixtureState(t, 5, 2400, 30, 30, 238)
		mutated := base.Clone()
		mutated.Difficulty.BossMaxHP++
		baseAfter, _ := advanceWithRoles(t, base, RoleQueue{}, domain.BossSpawnTick)
		mutatedAfter, _ := advanceWithRoles(t, mutated, RoleQueue{}, domain.BossSpawnTick)
		if baseAfter.Wave.Boss.MaxHP == mutatedAfter.Wave.Boss.MaxHP {
			t.Fatalf("boss spawn did not change: both %d", baseAfter.Wave.Boss.MaxHP)
		}
	})

	t.Run("boss_incoming_suppresses_duplicate_event", func(t *testing.T) {
		base := lockedFixtureState(t, 5, 1800, 30, 30, 238)
		for index := range base.Wave.Pools {
			base.Wave.Pools[index].HP = 0
		}
		quiet := base.Clone()
		quiet.Wave.BossIncoming = true
		if got := countEvents(t, base, RoleQueue{}, EventBossIncoming); got != 1 {
			t.Fatalf("unflagged state emitted %d boss_incoming events, want 1", got)
		}
		if got := countEvents(t, quiet, RoleQueue{}, EventBossIncoming); got != 0 {
			t.Fatalf("flagged state emitted %d boss_incoming events, want 0", got)
		}
	})

	t.Run("pending_result_blocks_rollover", func(t *testing.T) {
		settled := collisionSettledBase(t)
		atBoundary, _ := advanceWithRoles(t, settled, RoleQueue{}, domain.RolloverTick)
		pending := atBoundary.Clone()
		pending.Wave.Result = domain.WaveResultPending
		if _, _, err := Next(atBoundary, Input{Kind: KindRollover}); err != nil {
			t.Fatalf("settled rollover failed: %v", err)
		}
		if _, _, err := Next(pending, Input{Kind: KindRollover}); err == nil {
			t.Fatal("rollover accepted a wave with a pending result")
		}
	})
}

func ordinaryHP(state domain.SessionState) int64 {
	var total int64
	for _, pool := range state.Wave.Pools {
		total += pool.HP
	}
	return total
}

func countEvents(t *testing.T, state domain.SessionState, queue RoleQueue, eventType EventType) int {
	t.Helper()
	_, _, events := advanceOneStep(t, state, queue)
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}
