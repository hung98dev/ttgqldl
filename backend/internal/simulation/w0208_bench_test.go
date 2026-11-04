package simulation

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// benchState builds a valid locked mid-wave state with the given participants.
func benchState(b *testing.B, participants int) domain.SessionState {
	b.Helper()
	state, err := domain.NewInitialState(
		domain.VillageID("11111111-1111-4111-8111-111111111111"),
		domain.SessionID("22222222-2222-4222-8222-222222222222"),
		domain.SourceModeMock)
	if err != nil {
		b.Fatal(err)
	}
	members := make([]domain.WaveMember, 0, participants)
	for i := 0; i < participants; i++ {
		role := domain.RoleBuilder
		if i%3 == 1 {
			role = domain.RoleGuard
		} else if i%3 == 2 {
			role = domain.RoleFighter
		}
		members = append(members, domain.WaveMember{ActorKey: uniqueActor(i + 1), Role: role, JoinedTick: 0})
	}
	domain.SortWaveMembers(members)
	state.Members = members
	baseHP, err := domain.BaseEnemyHP(1, participants, 0)
	if err != nil {
		b.Fatal(err)
	}
	pools, err := domain.BuildEnemyPools(baseHP)
	if err != nil {
		b.Fatal(err)
	}
	state.Wave = domain.WaveState{Number: 2, Tick: 201, Phase: domain.PhasePrep, Pools: pools}
	state.Lifecycle = domain.LifecycleRunning
	state.Difficulty = domain.Difficulty{
		Level: 1, PopulationAnchor: domain.PopulationForDifficulty(participants), Locked: true,
		ActualPopulation: participants, RecoveryPercent: 100, BaseHP: baseHP,
	}
	if err := state.Validate(); err != nil {
		b.Fatalf("bench state: %v", err)
	}
	return state
}

func benchStateAt(b *testing.B, participants, wave, tick int) domain.SessionState {
	b.Helper()
	state := benchState(b, participants)
	phase, err := domain.PhaseForTick(tick)
	if err != nil {
		b.Fatal(err)
	}
	state.Wave.Number = int64(wave)
	state.Wave.Tick = tick
	state.Wave.Phase = phase
	if domain.IsBossWave(int64(wave)) {
		state.Difficulty.BossMaxHP = (state.Difficulty.BaseHP + 4) / 5
	} else {
		state.Difficulty.BossMaxHP = 0
	}
	if err := state.Validate(); err != nil {
		b.Fatalf("bench state at %d/%d: %v", wave, tick, err)
	}
	return state
}

// BenchmarkCanonicalProjection measures the canonical projection.
func BenchmarkCanonicalProjection(b *testing.B) {
	state := benchState(b, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := canonical.Project(state); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCanonicalHash measures projection plus JCS serialization plus SHA-256.
func BenchmarkCanonicalHash(b *testing.B) {
	state := benchState(b, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := canonical.Hash(state); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkQuietTick measures one non-boundary authoritative transition.
func BenchmarkQuietTick(b *testing.B) {
	state := benchStateAt(b, 100, 2, 500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, _, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContributionBoundary measures the real tick-200 economy boundary.
func BenchmarkContributionBoundary(b *testing.B) {
	state := benchStateAt(b, 100, 2, 199)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, _, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCombatBoundary measures the real tick-1800 combat boundary.
func BenchmarkCombatBoundary(b *testing.B) {
	state := benchStateAt(b, 100, 2, 1799)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, _, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBossBoundary measures boss spawn plus combat at tick 2600.
func BenchmarkBossBoundary(b *testing.B) {
	state := benchStateAt(b, 100, 5, 2599)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, _, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFinalSettlement measures tick-2800 final combat plus settlement.
func BenchmarkFinalSettlement(b *testing.B) {
	state := benchStateAt(b, 100, 5, 2799)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, _, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEnvelopeCreateVerify measures envelope creation and digest verification
// only; it is not a serialized checkpoint round trip.
func BenchmarkEnvelopeCreateVerify(b *testing.B) {
	state := benchState(b, 100)
	queue := RoleQueue{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		envelope, err := BuildCheckpointEnvelope(state, queue)
		if err != nil {
			b.Fatal(err)
		}
		if err := VerifyCheckpointEnvelope(envelope, state, queue); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCheckpointSerialize measures full private checkpoint serialization and
// deserialization followed by integrity verification.
func BenchmarkCheckpointSerialize(b *testing.B) {
	state := benchState(b, 100)
	queue := RoleQueue{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		envelope, err := BuildCheckpointEnvelope(state, queue)
		if err != nil {
			b.Fatal(err)
		}
		payload, err := json.Marshal(checkpointPayload{State: state, Queue: queue, Envelope: envelope})
		if err != nil {
			b.Fatal(err)
		}
		var restored checkpointPayload
		if err := json.Unmarshal(payload, &restored); err != nil {
			b.Fatal(err)
		}
		if err := VerifyCheckpointEnvelope(restored.Envelope, restored.State, restored.Queue); err != nil {
			b.Fatal(err)
		}
	}
}

// checkpointPayload mirrors the private restoration payload.
type checkpointPayload struct {
	State    domain.SessionState `json:"state"`
	Queue    RoleQueue           `json:"queue"`
	Envelope CheckpointEnvelope  `json:"envelope"`
}

// BenchmarkCanonicalHashByPopulation measures projection, JCS and SHA-256 for the
// three acceptance populations.
func BenchmarkCanonicalHashByPopulation(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		b.Run(strconv.Itoa(size)+"_participants", func(b *testing.B) {
			state := benchState(b, size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := canonical.Hash(state); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCheckpointCreateVerifyByPopulation measures envelope costs only.
func BenchmarkCheckpointCreateVerifyByPopulation(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		b.Run(strconv.Itoa(size)+"_participants", func(b *testing.B) {
			state := benchState(b, size)
			queue := RoleQueue{}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				envelope, err := BuildCheckpointEnvelope(state, queue)
				if err != nil {
					b.Fatal(err)
				}
				if err := VerifyCheckpointEnvelope(envelope, state, queue); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
