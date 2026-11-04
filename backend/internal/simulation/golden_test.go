package simulation

import (
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// goldenWaveFixture builds a documented synthetic wave-5 fixture: a fresh village
// with the given role mix joined at tick 0. It is a mathematical fixture, not a
// recorded playthrough history.
func goldenWaveFixture(t *testing.T, builders, guards, fighters int) domain.SessionState {
	t.Helper()
	state, _ := startWithRole(t, 'a', domain.RoleBuilder)
	state.Wave.Number = 5
	state.Difficulty = domain.Difficulty{Level: 1}
	members := make([]domain.WaveMember, 0, builders+guards+fighters)
	index := 1
	appendRole := func(count int, role domain.Role) {
		for i := 0; i < count; i++ {
			members = append(members, domain.WaveMember{ActorKey: uniqueActor(index), Role: role, JoinedTick: 0})
			index++
		}
	}
	appendRole(builders, domain.RoleBuilder)
	appendRole(guards, domain.RoleGuard)
	appendRole(fighters, domain.RoleFighter)
	domain.SortWaveMembers(members)
	state.Members = members
	state.XP = []domain.PlayerXP{}
	return state
}

func totalXP(state domain.SessionState) int64 {
	total := int64(0)
	for _, row := range state.XP {
		total += row.XP
	}
	return total
}

func TestGoldenBossWavePopulation10(t *testing.T) {
	state := goldenWaveFixture(t, 2, 2, 6)
	at1600, _ := advanceWithRoles(t, state, RoleQueue{}, 1600)
	if at1600.Difficulty.BaseHP != 162 || at1600.Difficulty.BossMaxHP != 33 {
		t.Fatalf("base/boss = %d/%d, want 162/33", at1600.Difficulty.BaseHP, at1600.Difficulty.BossMaxHP)
	}
	at2600, _ := advanceWithRoles(t, at1600, RoleQueue{}, 2600)
	if !at2600.Wave.Boss.Exists || at2600.Wave.Boss.HP != 0 {
		t.Fatalf("boss at 2600 = %+v, want spawned and killed", at2600.Wave.Boss)
	}
	final, _ := advanceWithRoles(t, at2600, RoleQueue{}, 3000)
	if final.Wave.Result != domain.WaveResultVictory {
		t.Fatalf("result = %s", final.Wave.Result)
	}
	if final.Village.WallHP != 1000 {
		t.Fatalf("wall = %d, want 1000", final.Village.WallHP)
	}
	if final.Village.Resources.Wood != 518 || final.Village.Resources.Stone != 324 || final.Village.Resources.Food != 350 {
		t.Fatalf("resources = %+v, want 518/324/350", final.Village.Resources)
	}
	if got := totalXP(final); got != 100 {
		t.Fatalf("total XP = %d, want 100", got)
	}
}

func TestGoldenBossWavePopulation30(t *testing.T) {
	state := goldenWaveFixture(t, 8, 8, 14)
	at1600, _ := advanceWithRoles(t, state, RoleQueue{}, 1600)
	if at1600.Difficulty.BaseHP != 238 || at1600.Difficulty.BossMaxHP != 48 {
		t.Fatalf("base/boss = %d/%d, want 238/48", at1600.Difficulty.BaseHP, at1600.Difficulty.BossMaxHP)
	}
	at2600, _ := advanceWithRoles(t, at1600, RoleQueue{}, 2600)
	if !at2600.Wave.Boss.Exists || at2600.Wave.Boss.HP != 16 {
		t.Fatalf("boss at 2600 = %+v, want HP 16", at2600.Wave.Boss)
	}
	at2800, _ := advanceWithRoles(t, at2600, RoleQueue{}, 2800)
	if at2800.Wave.Boss.HP != 0 {
		t.Fatalf("boss at 2800 = %d, want 0", at2800.Wave.Boss.HP)
	}
	final, _ := advanceWithRoles(t, at2800, RoleQueue{}, 3000)
	if final.Wave.Result != domain.WaveResultVictory {
		t.Fatalf("result = %s", final.Wave.Result)
	}
	if final.Village.WallHP != 1000 {
		t.Fatalf("wall = %d, want 1000", final.Village.WallHP)
	}
	if final.Village.Resources.Wood != 544 || final.Village.Resources.Stone != 338 {
		t.Fatalf("resources = %+v, want wood 544 stone 338", final.Village.Resources)
	}
	if got := totalXP(final); got != domain.CapSessionXP {
		t.Fatalf("total XP = %d, want %d", got, domain.CapSessionXP)
	}
}
