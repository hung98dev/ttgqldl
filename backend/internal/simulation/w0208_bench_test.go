package simulation

import (
	"strconv"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/canonical"
	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

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

// BenchmarkTickAdvancesOneLogicalTick on a quiet (non-boundary) tick.
func BenchmarkTickAdvancesOneLogicalTick(b *testing.B) {
	state := benchState(b, 100)
	state.Wave.Tick = 500
	state.Wave.Phase = domain.PhasePrep
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := Next(state, Input{Kind: KindAdvanceTick}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkContributionBoundary measures one 200-tick economy boundary.
func BenchmarkContributionBoundary(b *testing.B) {
	state := benchState(b, 100)
	state.Wave.Tick = 199
	state.Wave.Phase = domain.PhasePrep
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := Next(state, Input{Kind: KindAdvanceTick}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCheckpointRoundTrip measures envelope creation plus verification.
func BenchmarkCheckpointRoundTrip(b *testing.B) {
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
		members = append(members, domain.WaveMember{
			ActorKey: uniqueActor(i + 1), Role: domain.RoleBuilder, JoinedTick: 0,
		})
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
	state.Wave.Pools = pools
	state.Wave.Number = 2
	state.Wave.Tick = 201
	state.Wave.Phase = domain.PhasePrep
	state.Lifecycle = domain.LifecycleRunning
	state.Difficulty = domain.Difficulty{
		Level: 1, PopulationAnchor: domain.PopulationForDifficulty(participants), Locked: true, ActualPopulation: participants,
		RecoveryPercent: 100, BaseHP: baseHP, BossMaxHP: 0,
	}
	if err := state.Validate(); err != nil {
		b.Fatalf("bench state: %v", err)
	}
	return state
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

// BenchmarkCheckpointByPopulation measures private envelope creation + verification.
func BenchmarkCheckpointByPopulation(b *testing.B) {
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
