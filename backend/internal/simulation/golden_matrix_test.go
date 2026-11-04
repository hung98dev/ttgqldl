package simulation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// referenceCase mirrors the independently generated golden vector fields that this
// differential test consumes. Unknown fields (votes, explicit joins) are ignored so
// unsupported cases can be classified rather than silently coerced.
type referenceCase struct {
	ID     string `json:"id"`
	Note   string `json:"note"`
	Inputs struct {
		WaveNo            int64 `json:"wave_no"`
		ConsecutiveWins   int64 `json:"consecutive_wins"`
		ConsecutiveLosses int64 `json:"consecutive_losses"`
		WallTier          int   `json:"wall_tier"`
		WallHP            int64 `json:"wall_hp"`
		TownhallHP        int64 `json:"townhall_hp"`
		Wood              int64 `json:"wood"`
		Stone             int64 `json:"stone"`
		Food              int64 `json:"food"`
		ConstructionWork  int64 `json:"construction_work"`
		Builders          int   `json:"builders"`
		Guards            int   `json:"guards"`
		Fighters          int   `json:"fighters"`
		ExplicitJoins     []any `json:"explicit_joins"`
	} `json:"inputs"`
	Outcome struct {
		Result            string `json:"result"`
		BaseHP            int64  `json:"base_hp"`
		WallTier          int    `json:"wall_tier"`
		WallHP            int64  `json:"wall_hp"`
		TownhallHP        int64  `json:"townhall_hp"`
		Wood              int64  `json:"wood"`
		Stone             int64  `json:"stone"`
		Food              int64  `json:"food"`
		ConstructionWork  int64  `json:"construction_work"`
		TotalWins         int64  `json:"total_wins"`
		ConsecutiveWins   int64  `json:"consecutive_wins"`
		ConsecutiveLosses int64  `json:"consecutive_losses"`
		SessionXPTotal    int64  `json:"session_xp_total"`
		Defeated          bool   `json:"defeated"`
		VoteChosen        string `json:"vote_chosen"`
		BossHPAt2600      int64  `json:"boss_hp_at_2600"`
		OrdinaryHPAt2600  int64  `json:"ordinary_hp_at_2600"`
	} `json:"outcome"`
}

type referenceMatrix struct {
	Cases []referenceCase `json:"cases"`
}

func TestGoldenMatrixDifferential(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "golden", "matrix.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read reference matrix: %v", err)
	}
	var matrix referenceMatrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatalf("parse reference matrix: %v", err)
	}
	if len(matrix.Cases) == 0 {
		t.Fatal("reference matrix is empty")
	}
	executed, skipped := 0, 0
	for _, ref := range matrix.Cases {
		participants := ref.Inputs.Builders + ref.Inputs.Guards + ref.Inputs.Fighters
		if participants == 0 || len(ref.Inputs.ExplicitJoins) > 0 || ref.Outcome.VoteChosen != "" {
			skipped++
			continue
		}
		if ref.Inputs.WaveNo%2 == 0 {
			// Production opens a real vote on even waves; reference cases without an
			// explicit vote selection are not directly comparable.
			skipped++
			continue
		}
		executed++
		state := goldenMatrixState(t, ref)
		ordinaryAt2600, bossAt2600 := int64(-1), int64(-1)
		frozenAt2600 := false
		for state.Tick() < domain.FinalCombatTick {
			next, _, _, _, err := AdvanceWithRoles(state, RoleQueue{}, Input{Kind: KindAdvanceTick})
			if err != nil {
				t.Fatalf("%s: advance at %d: %v", ref.ID, state.Tick(), err)
			}
			state = next
			if state.Tick() == domain.BossSpawnTick {
				frozenAt2600 = state.Wave.Defeated
				ordinaryAt2600 = 0
				for _, pool := range state.Wave.Pools {
					ordinaryAt2600 += pool.HP
				}
				bossAt2600 = state.Wave.Boss.HP
			}
		}
		var xpTotal int64
		for _, row := range state.XP {
			xpTotal += row.XP
		}
		checks := []struct {
			field    string
			expected string
			actual   string
		}{
			{"base_hp", fmt.Sprint(ref.Outcome.BaseHP), fmt.Sprint(state.Difficulty.BaseHP)},
			{"wall_tier", fmt.Sprint(ref.Outcome.WallTier), fmt.Sprint(state.Village.WallTier)},
			{"wall_hp", fmt.Sprint(ref.Outcome.WallHP), fmt.Sprint(state.Village.WallHP)},
			{"townhall_hp", fmt.Sprint(ref.Outcome.TownhallHP), fmt.Sprint(state.Village.TownhallHP)},
			{"wood", fmt.Sprint(ref.Outcome.Wood), fmt.Sprint(state.Village.Resources.Wood)},
			{"stone", fmt.Sprint(ref.Outcome.Stone), fmt.Sprint(state.Village.Resources.Stone)},
			{"food", fmt.Sprint(ref.Outcome.Food), fmt.Sprint(state.Village.Resources.Food)},
			{"construction_work", fmt.Sprint(ref.Outcome.ConstructionWork), fmt.Sprint(state.Village.ConstructionWork)},
			{"total_wins", fmt.Sprint(ref.Outcome.TotalWins), fmt.Sprint(state.Village.TotalWins)},
			{"consecutive_wins", fmt.Sprint(ref.Outcome.ConsecutiveWins), fmt.Sprint(state.ConsecutiveWins)},
			{"consecutive_losses", fmt.Sprint(ref.Outcome.ConsecutiveLosses), fmt.Sprint(state.ConsecutiveLosses)},
			{"session_xp_total", fmt.Sprint(ref.Outcome.SessionXPTotal), fmt.Sprint(xpTotal)},
			{"defeated", fmt.Sprint(ref.Outcome.Defeated), fmt.Sprint(state.Wave.Defeated)},
			{"result", normalizeReferenceResult(ref.Outcome.Result), state.Wave.Result.String()},
		}
		if !frozenAt2600 && !ref.Outcome.Defeated {
			checks = append(checks,
				struct {
					field    string
					expected string
					actual   string
				}{"ordinary_hp_at_2600", fmt.Sprint(ref.Outcome.OrdinaryHPAt2600), fmt.Sprint(ordinaryAt2600)},
				struct {
					field    string
					expected string
					actual   string
				}{"boss_hp_at_2600", fmt.Sprint(ref.Outcome.BossHPAt2600), fmt.Sprint(bossAt2600)},
			)
		}
		for _, check := range checks {
			if check.expected != check.actual {
				t.Fatalf("case_id=%s wave_no=%d population=%d tick=%d field=%s expected=%s actual=%s",
					ref.ID, ref.Inputs.WaveNo, participants, state.Tick(), check.field, check.expected, check.actual)
			}
		}
	}
	if executed == 0 {
		t.Fatal("no reference cases were executable")
	}
	t.Logf("golden matrix: %d cases executed, %d skipped (unsupported by deterministic mapping)", executed, skipped)
}

func goldenMatrixState(t *testing.T, ref referenceCase) domain.SessionState {
	t.Helper()
	state, err := domain.NewInitialState(
		domain.VillageID("11111111-1111-4111-8111-111111111111"),
		domain.SessionID("22222222-2222-4222-8222-222222222222"),
		domain.SourceModeMock)
	if err != nil {
		t.Fatal(err)
	}
	members := make([]domain.WaveMember, 0, ref.Inputs.Builders+ref.Inputs.Guards+ref.Inputs.Fighters)
	index := 1
	appendRole := func(count int, role domain.Role) {
		for i := 0; i < count; i++ {
			members = append(members, domain.WaveMember{ActorKey: uniqueActor(index), Role: role, JoinedTick: 0})
			index++
		}
	}
	appendRole(ref.Inputs.Builders, domain.RoleBuilder)
	appendRole(ref.Inputs.Guards, domain.RoleGuard)
	appendRole(ref.Inputs.Fighters, domain.RoleFighter)
	domain.SortWaveMembers(members)
	state.Members = members
	state.Wave.Number = ref.Inputs.WaveNo
	state.Wave.Tick = 0
	state.Wave.Phase = domain.PhasePrep
	state.Lifecycle = domain.LifecycleRunning
	state.Village.WallTier = ref.Inputs.WallTier
	state.Village.WallHP = ref.Inputs.WallHP
	state.Village.TownhallHP = ref.Inputs.TownhallHP
	state.Village.Resources = domain.Resources{Wood: ref.Inputs.Wood, Stone: ref.Inputs.Stone, Food: ref.Inputs.Food}
	state.Village.ConstructionWork = ref.Inputs.ConstructionWork
	state.ConsecutiveWins = ref.Inputs.ConsecutiveWins
	state.ConsecutiveLosses = ref.Inputs.ConsecutiveLosses
	state.Difficulty = domain.Difficulty{Level: 1}
	if err := state.Validate(); err != nil {
		t.Fatalf("%s fixture invalid: %v", ref.ID, err)
	}
	return state
}

// normalizeReferenceResult maps the independent evaluator's outcome vocabulary to
// the production WaveResult names; the words differ, the semantics do not.
func normalizeReferenceResult(result string) string {
	switch result {
	case "loss":
		return "defeat"
	case "win":
		return "victory"
	default:
		return result
	}
}
