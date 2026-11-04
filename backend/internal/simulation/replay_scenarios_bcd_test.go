package simulation

import (
	"errors"
	"testing"

	"github.com/hung98dev/ttgqldl/backend/internal/domain"
)

// replayStartWave builds a genuine wave through the accepted-registration API:
// one start registration at tick 0, then every other registration accepted at
// tick 0 and effective at tick 1.
func replayStartWave(t *testing.T, waveNo int64, configure func(*domain.SessionState), roles []domain.Role) domain.SessionState {
	t.Helper()
	state, err := domain.NewInitialState(villageID, sessionID, domain.SourceModeMock)
	if err != nil {
		t.Fatal(err)
	}
	state.Wave.Number = waveNo
	if configure != nil {
		configure(&state)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("replay fixture invalid: %v", err)
	}
	first := roleCommand(1, uniqueActor(1), roles[0], waveNo, 0)
	accepted, queue, _ := acceptRoles(t, state, RoleQueue{}, first)
	state, queue, _, _, err = AdvanceWithRoles(accepted, queue, Input{
		Kind: KindStartWave, Registration: &Registration{ActorKey: first.ActorKey, Role: first.Role},
	})
	if err != nil {
		t.Fatalf("start replay wave: %v", err)
	}
	var commands []RoleCommand
	for index, role := range roles[1:] {
		commands = append(commands, roleCommand(int64(index+2), uniqueActor(index+2), role, waveNo, 1))
	}
	accepted, queue, _ = acceptRoles(t, state, queue, commands...)
	state, _ = advanceWithRoles(t, accepted, queue, 1)
	return state
}

func repeatedRoles(count int, role domain.Role) []domain.Role {
	roles := make([]domain.Role, 0, count)
	for index := 0; index < count; index++ {
		roles = append(roles, role)
	}
	return roles
}

func countEventType(events []Event, eventType EventType) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

// --- Scenario B: real defeat, settlement, recovery and rollover ---

func scenarioDefeatRecovery(t *testing.T) replayScenario {
	return replayScenario{
		ID: "B",
		Initial: func(t *testing.T) (domain.SessionState, RoleQueue) {
			// Documented fixture: a village that has already lost its wall and town
			// hall HP to earlier off-screen damage (wall 10, townhall 10), defended by
			// 30 builders. Every transition from here is real: the first combat
			// boundary destroys the town hall through enemy retaliation.
			state := replayStartWave(t, 1, func(state *domain.SessionState) {
				state.Village.WallHP = 10
				state.Village.TownhallHP = 10
			}, repeatedRoles(30, domain.RoleFighter))
			return state, RoleQueue{}
		},
		Actions: []replayAction{
			{
				name: "queue-next-wave-after-defeat", wave: 1, tick: domain.FinalCombatTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					var commands []RoleCommand
					for index := 60; index < 65; index++ {
						commands = append(commands, RoleCommand{
							InputSeq: int64(index), ActorKey: uniqueActor(index), Role: domain.RoleFighter,
							TargetWave: state.Wave.Number + 1, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
						})
					}
					next, nextQueue, _, err := AcceptRoleCommands(state, queue, commands)
					if err != nil {
						t.Fatalf("queue next-wave registrations: %v", err)
					}
					return next, nextQueue, nil
				},
			},
		},
		Done: func(state domain.SessionState) bool {
			return state.Wave.Number == 2 && state.Tick() == domain.CombatStartTick
		},
		Limit:           6000,
		CheckpointEvery: 40,
		Critical: map[int64]map[int]bool{
			1: {1799: true, 1800: true, 1801: true, 2799: true, 2800: true, 2801: true, 2999: true, 3000: true},
			2: {0: true, 1599: true, 1600: true, 1799: true, 1800: true},
		},
		Verify: func(t *testing.T, steps []replayStep) {
			t.Helper()
			lethal := findReplayStep(t, steps, 1, domain.CombatStartTick)
			if !lethal.State.Wave.Defeated {
				t.Fatal("scenario B must reach a real DEFEATED state through combat")
			}
			if lethal.State.Village.TownhallHP != 0 {
				t.Fatalf("townhall HP at defeat = %d, want 0", lethal.State.Village.TownhallHP)
			}
			if !containsEvent(lethal.Events, EventDamageApplied) {
				t.Fatal("lethal boundary emitted no damage_applied event")
			}

			settled := findReplayStep(t, steps, 1, domain.FinalCombatTick)
			if settled.State.Wave.Result != domain.WaveResultDefeat {
				t.Fatalf("settlement result = %s, want defeat", settled.State.Wave.Result)
			}
			if countEventType(settled.Events, EventWaveResult) != 1 {
				t.Fatalf("wave_result events at settlement = %d, want exactly 1", countEventType(settled.Events, EventWaveResult))
			}
			if settled.State.ConsecutiveLosses != 1 || settled.State.ConsecutiveWins != 0 {
				t.Fatalf("streaks after defeat = %d/%d", settled.State.ConsecutiveLosses, settled.State.ConsecutiveWins)
			}
			if settled.State.Village.WallHP != 250 || settled.State.Village.TownhallHP != 250 {
				t.Fatalf("recovery floors = wall %d townhall %d, want 250/250",
					settled.State.Village.WallHP, settled.State.Village.TownhallHP)
			}
			if settled.State.Village.Resources.Wood < 100 {
				t.Fatalf("recovery wood floor = %d, want >= 100", settled.State.Village.Resources.Wood)
			}
			if totalXP(settled.State) != 300 {
				t.Fatalf("retained session XP = %d, want 300 (30 participants, one wave each)", totalXP(settled.State))
			}

			rolled := findReplayStep(t, steps, 2, 0)
			if len(rolled.State.Members) != 5 {
				t.Fatalf("promoted wave-2 members = %d, want 5", len(rolled.State.Members))
			}
			if rolled.State.Wave.Defeated || rolled.State.Wave.Result != domain.WaveResultPending {
				t.Fatalf("wave 2 did not reset defeat/result: %+v", rolled.State.Wave)
			}
			if totalXP(rolled.State) != 300 {
				t.Fatalf("session XP after rollover = %d, want 300", totalXP(rolled.State))
			}

			locked := findReplayStep(t, steps, 2, domain.PopulationLockTick)
			if locked.State.Difficulty.PopulationAnchor != 5 || locked.State.Difficulty.ActualPopulation != 5 {
				t.Fatalf("wave-2 anchor = %d/%d, want 5/5",
					locked.State.Difficulty.PopulationAnchor, locked.State.Difficulty.ActualPopulation)
			}
			// (40 + 8*1 + 38*isqrt(5)) * 85 / 100 = floor(124*85/100) = 105.
			if locked.State.Difficulty.BaseHP != 105 || locked.State.Difficulty.RecoveryPercent != 85 {
				t.Fatalf("recovery modifier = %d%% base_hp %d, want 85%%/105",
					locked.State.Difficulty.RecoveryPercent, locked.State.Difficulty.BaseHP)
			}

			waveTwoCombat := findReplayStep(t, steps, 2, domain.CombatStartTick)
			attacks := eventsOfType(waveTwoCombat.Events, EventAttackResolved)
			if len(attacks) != 1 || attacks[0].Amount != 60 {
				t.Fatalf("wave-2 attack events = %+v, want one attack of 60 (5 fighters + militia 24)", attacks)
			}
		},
	}
}

// --- Scenario C: boss waves and rollover ---

func scenarioBoss(waveNumber int64, builders, guards, fighters int, id string, bossAt2600, wood, stone int64) replayScenario {
	var roles []domain.Role
	roles = append(roles, repeatedRoles(builders, domain.RoleBuilder)...)
	roles = append(roles, repeatedRoles(guards, domain.RoleGuard)...)
	roles = append(roles, repeatedRoles(fighters, domain.RoleFighter)...)
	xp := int64(builders+guards+fighters) * domain.XPPerWave
	if xp > domain.CapSessionXP {
		xp = domain.CapSessionXP
	}
	return replayScenario{
		ID: id,
		Initial: func(t *testing.T) (domain.SessionState, RoleQueue) {
			// Documented synthetic wave-5 setup: waves 1-4 are not simulated, but every
			// transition of wave 5 and its rollover is the real reducer.
			return replayStartWave(t, waveNumber, nil, roles), RoleQueue{}
		},
		Actions: []replayAction{
			{
				name: "next-wave-registrations", wave: waveNumber, tick: domain.FinalCombatTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					var commands []RoleCommand
					for index := 70; index < 73; index++ {
						commands = append(commands, RoleCommand{
							InputSeq: int64(index), ActorKey: uniqueActor(index), Role: domain.RoleFighter,
							TargetWave: state.Wave.Number + 1, ScheduledTick: 0, AcceptedTick: AcceptedTickUnspecified,
						})
					}
					next, nextQueue, _, err := AcceptRoleCommands(state, queue, commands)
					if err != nil {
						t.Fatalf("queue next-wave registrations: %v", err)
					}
					return next, nextQueue, nil
				},
			},
		},
		Done: func(state domain.SessionState) bool {
			return state.Wave.Number == waveNumber+1 && state.Tick() == 1
		},
		Limit:           4000,
		CheckpointEvery: 40,
		Critical: map[int64]map[int]bool{
			waveNumber: {1599: true, 1600: true, 1799: true, 1800: true, 2599: true, 2600: true,
				2799: true, 2800: true, 2999: true, 3000: true},
			waveNumber + 1: {0: true, 1: true},
		},
		Verify: func(t *testing.T, steps []replayStep) {
			verifyBossScenario(t, steps, waveNumber, bossAt2600, xp, wood, stone)
		},
	}
}

func TestScenarioCBossRollover(t *testing.T) {
	cases := []struct {
		name                    string
		b, g, f                 int
		bossAt2600, wood, stone int64
	}{
		{name: "P10", b: 2, g: 2, f: 6, bossAt2600: 0, wood: 518, stone: 324},
		{name: "P30", b: 8, g: 8, f: 14, bossAt2600: 16, wood: 544, stone: 338},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			scenario := scenarioBoss(5, testCase.b, testCase.g, testCase.f,
				"C-"+testCase.name, testCase.bossAt2600, testCase.wood, testCase.stone)
			outcome := runReplayScenario(t, scenario)
			logReplayOutcome(t, "C-"+testCase.name, outcome)
		})
	}
}

func verifyBossScenario(t *testing.T, steps []replayStep, waveNumber, bossAt2600 int64, xpTotal, wood, stone int64) {
	t.Helper()
	spawn := findReplayStep(t, steps, waveNumber, domain.BossSpawnTick)
	if !spawn.State.Wave.Boss.Exists {
		t.Fatal("boss pool does not exist at tick 2600")
	}
	if spawn.State.Wave.Boss.HP != bossAt2600 {
		t.Fatalf("boss HP at 2600 = %d, want %d", spawn.State.Wave.Boss.HP, bossAt2600)
	}
	final := findReplayStep(t, steps, waveNumber, domain.FinalCombatTick)
	if final.State.Wave.Boss.HP != 0 {
		t.Fatalf("boss HP at 2800 = %d, want 0", final.State.Wave.Boss.HP)
	}
	if final.State.Wave.Result != domain.WaveResultVictory {
		t.Fatalf("wave %d result = %s, want victory", waveNumber, final.State.Wave.Result)
	}
	if final.State.Village.Resources.Wood != wood || final.State.Village.Resources.Stone != stone {
		t.Fatalf("settlement resources = %+v, want wood %d stone %d", final.State.Village.Resources, wood, stone)
	}
	if totalXP(final.State) != xpTotal {
		t.Fatalf("session XP = %d, want %d", totalXP(final.State), xpTotal)
	}
	rolled := findReplayStep(t, steps, waveNumber+1, 0)
	if len(rolled.State.Members) != 3 {
		t.Fatalf("promoted next-wave members = %d, want 3", len(rolled.State.Members))
	}
	for _, member := range rolled.State.Members {
		if member.JoinedTick != 0 {
			t.Fatalf("promoted member joined_tick = %d, want 0", member.JoinedTick)
		}
	}
	if rolled.State.Wave.Boss.Exists || len(rolled.State.Wave.Pools) != 0 {
		t.Fatalf("boss/ordinary enemy state did not reset at rollover: %+v %d pools",
			rolled.State.Wave.Boss, len(rolled.State.Wave.Pools))
	}
	if rolled.State.Wave.Number != waveNumber+1 || rolled.State.Wave.Tick != 0 {
		t.Fatalf("rollover state = wave %d tick %d", rolled.State.Wave.Number, rolled.State.Wave.Tick)
	}
	if totalXP(rolled.State) != xpTotal {
		t.Fatalf("village XP did not persist across rollover: %d want %d", totalXP(rolled.State), xpTotal)
	}
}

// --- Scenario D: multiple pause reasons and deferred inputs ---

func scenarioPauseResume(t *testing.T) replayScenario {
	ballot := func() VoteCommand {
		return VoteCommand{
			InputSeq: 500, ActorKey: uniqueActor(1), TargetWave: 2,
			ScheduledTick: 801, Option: domain.VoteOptionA, AcceptedTick: AcceptedTickUnspecified,
		}
	}
	roleChange := func() RoleCommand {
		return RoleCommand{
			InputSeq: 501, ActorKey: uniqueActor(1), Role: domain.RoleGuard,
			TargetWave: 2, ScheduledTick: 810, AcceptedTick: AcceptedTickUnspecified,
		}
	}
	return replayScenario{
		ID: "D",
		Initial: func(t *testing.T) (domain.SessionState, RoleQueue) {
			roles := []domain.Role{domain.RoleBuilder, domain.RoleGuard, domain.RoleFighter, domain.RoleFighter}
			state := replayStartWave(t, 2, nil, roles)
			state, _ = advanceWithRoles(t, state, RoleQueue{}, domain.VoteOpenTick)
			return state, RoleQueue{}
		},
		Actions: []replayAction{
			pauseAction("pause-operator", domain.PauseOperatorPaused),
			pauseAction("pause-source", domain.PauseSourceDisconnected),
			pauseAction("pause-storage", domain.PauseStorageUnavailable),
			{
				name: "frozen-while-paused", wave: 2, tick: domain.VoteOpenTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					before := state.Tick()
					if _, _, _, _, err := AdvanceWithRoles(state, queue, Input{Kind: KindAdvanceTick}); !errors.Is(err, ErrTickFrozen) {
						t.Fatalf("advance while paused: %v", err)
					}
					if state.Tick() != before {
						t.Fatalf("logical tick moved while paused: %d -> %d", before, state.Tick())
					}
					return state, queue, nil
				},
			},
			{
				name: "ballot-deferred", wave: 2, tick: domain.VoteOpenTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					next, nextQueue, _, err := AcceptVoteCommands(state, queue, []VoteCommand{ballot()})
					var deferred *VoteDeferredError
					if !errors.As(err, &deferred) || len(deferred.Commands) != 1 {
						t.Fatalf("paused ballot was not deferred: %v", err)
					}
					return next, nextQueue, nil
				},
			},
			{
				name: "role-deferred", wave: 2, tick: domain.VoteOpenTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					next, nextQueue, decisions, err := AcceptRoleCommands(state, queue, []RoleCommand{roleChange()})
					if len(decisions) != 0 {
						t.Fatal("deferred role command produced an applied decision")
					}
					var deferred *DeferredError
					if !errors.As(err, &deferred) || len(deferred.Commands) != 1 {
						t.Fatalf("paused role command was not deferred: %v", err)
					}
					return next, nextQueue, nil
				},
			},
			{
				name: "resume-operator-only", wave: 2, tick: domain.VoteOpenTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					next, events, err := Next(state, Input{Kind: KindResume, Reason: domain.PauseOperatorPaused})
					if err != nil {
						t.Fatal(err)
					}
					if next.Lifecycle != domain.LifecycleSourceDisconnected {
						t.Fatalf("lifecycle after clearing operator pause = %s", next.Lifecycle)
					}
					if len(events) != 0 {
						t.Fatalf("clearing one pause reason emitted a resume event: %+v", events)
					}
					if _, _, _, _, err := AdvanceWithRoles(next, queue, Input{Kind: KindAdvanceTick}); !errors.Is(err, ErrTickFrozen) {
						t.Fatalf("advance with remaining pause reasons: %v", err)
					}
					return next, queue, events
				},
			},
			{
				name: "resume-source", wave: 2, tick: domain.VoteOpenTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					next, events, err := Next(state, Input{Kind: KindResume, Reason: domain.PauseSourceDisconnected})
					if err != nil {
						t.Fatal(err)
					}
					if next.Lifecycle != domain.LifecycleStorageUnavailable {
						t.Fatalf("lifecycle after clearing source pause = %s", next.Lifecycle)
					}
					if _, _, _, err := AcceptRoleCommands(next, queue, []RoleCommand{roleChange()}); !errors.Is(err, ErrStorageUnavailable) {
						t.Fatalf("role input during a storage outage: %v", err)
					}
					return next, queue, events
				},
			},
			{
				name: "resume-storage", wave: 2, tick: domain.VoteOpenTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					next, events, err := Next(state, Input{Kind: KindResume, Reason: domain.PauseStorageUnavailable})
					if err != nil {
						t.Fatal(err)
					}
					if next.Lifecycle != domain.LifecycleRunning || next.Tick() != domain.VoteOpenTick {
						t.Fatalf("final resume = lifecycle %s tick %d", next.Lifecycle, next.Tick())
					}
					if countEventType(events, EventResumed) != 1 {
						t.Fatalf("resume events = %+v, want exactly one resumed event", events)
					}
					return next, queue, events
				},
			},
			{
				name: "resubmit-ballot-once", wave: 2, tick: domain.VoteOpenTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					next, nextQueue, decisions, err := AcceptVoteCommands(state, queue, []VoteCommand{ballot()})
					if err != nil {
						t.Fatalf("resubmit deferred ballot: %v", err)
					}
					if len(decisions) != 1 || len(nextQueue.Ballots) != 1 {
						t.Fatalf("resubmitted ballot queued %d/%d", len(decisions), len(nextQueue.Ballots))
					}
					return next, nextQueue, nil
				},
			},
			{
				name: "resubmit-role-once", wave: 2, tick: domain.VoteOpenTick,
				apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
					next, nextQueue, decisions, err := AcceptRoleCommands(state, queue, []RoleCommand{roleChange()})
					if err != nil {
						t.Fatalf("resubmit deferred role command: %v", err)
					}
					if len(decisions) != 1 || len(nextQueue.Entries) != 1 {
						t.Fatalf("resubmitted role command queued %d/%d", len(decisions), len(nextQueue.Entries))
					}
					return next, nextQueue, nil
				},
			},
		},
		Done: func(state domain.SessionState) bool {
			return state.Wave.Number == 2 && state.Tick() == 820
		},
		Limit:           200,
		CheckpointEvery: 4,
		Critical: map[int64]map[int]bool{
			2: {799: true, 800: true, 801: true, 810: true, 820: true},
		},
		Verify: func(t *testing.T, steps []replayStep) {
			t.Helper()
			multiPaused := findReplayStep(t, steps, 2, domain.VoteOpenTick)
			// The last step at tick 800 is the resumed state; find a paused one.
			var paused replayStep
			found := false
			for _, step := range steps {
				if step.WaveNo == 2 && step.Tick == domain.VoteOpenTick && len(step.State.PauseReasons) == 3 {
					paused, found = step, true
				}
			}
			if !found {
				t.Fatalf("no checkpoint candidate with three simultaneous pause reasons (last 800 state: %v)",
					multiPaused.State.PauseReasons)
			}
			if paused.State.Lifecycle != domain.LifecyclePaused {
				t.Fatalf("lifecycle with operator+source+storage = %s, want paused", paused.State.Lifecycle)
			}
			if paused.State.Tick() != domain.VoteOpenTick {
				t.Fatalf("paused logical tick = %d, want 800", paused.State.Tick())
			}
			applied := findReplayStep(t, steps, 2, 801)
			if len(applied.State.Wave.Ballots) != 1 || applied.State.Wave.Ballots[0].Tick != 801 {
				t.Fatalf("deferred ballot was not applied exactly at 801: %+v", applied.State.Wave.Ballots)
			}
			changed := findReplayStep(t, steps, 2, 810)
			if len(changed.State.Members) != 4 {
				t.Fatalf("member count changed by the resubmitted command: %d", len(changed.State.Members))
			}
			member := changed.State.Members[0]
			if member.ActorKey != uniqueActor(1) || member.Role != domain.RoleGuard {
				t.Fatalf("resubmitted role change did not apply exactly once: %+v", member)
			}
			if member.LastChangeTick == nil || *member.LastChangeTick != 810 {
				t.Fatalf("role change tick = %v, want 810", member.LastChangeTick)
			}
			if final := findReplayStep(t, steps, 2, 820); final.State.Tick() != 820 {
				t.Fatalf("final logical tick = %d, want 820 (no wall-clock catch-up)", final.State.Tick())
			}
		},
	}
}

func pauseAction(name string, reason domain.PauseReason) replayAction {
	return replayAction{
		name: name, wave: 2, tick: domain.VoteOpenTick,
		apply: func(t *testing.T, state domain.SessionState, queue RoleQueue) (domain.SessionState, RoleQueue, []Event) {
			next, events, err := Next(state, Input{Kind: KindPause, Reason: reason})
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if countEventType(events, EventPaused) != 1 {
				t.Fatalf("%s emitted %+v", name, events)
			}
			return next, queue, events
		},
	}
}

func TestScenarioBDefeatRecovery(t *testing.T) {
	outcome := runReplayScenario(t, scenarioDefeatRecovery(t))
	logReplayOutcome(t, "B", outcome)
}

func TestScenarioDMultiplePauseReasons(t *testing.T) {
	outcome := runReplayScenario(t, scenarioPauseResume(t))
	logReplayOutcome(t, "D", outcome)
}

func containsEvent(events []Event, eventType EventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func eventsOfType(events []Event, eventType EventType) []Event {
	var matched []Event
	for _, event := range events {
		if event.Type == eventType {
			matched = append(matched, event)
		}
	}
	return matched
}
