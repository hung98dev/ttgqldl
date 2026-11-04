package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// CaseInputs is the documented, fully explicit input of one reference scenario.
// Every field is typed: there is no []any and no permissive blend of registrations
// and ballots. A case either names its explicit joins and role/ballot events or it
// describes a synthetic population by role counts.
type CaseInputs struct {
	WaveNo            int64 `json:"wave_no"`
	ConsecutiveWins   int   `json:"consecutive_wins"`
	ConsecutiveLosses int   `json:"consecutive_losses"`
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
	// Ballot is the legacy shorthand: every synthetic participant casts this option
	// at tick 900. Explicit ballots use BallotEvents instead.
	Ballot string `json:"ballot_option,omitempty"`
	// Joins are explicit registrations with their effective tick. The first entry
	// must be effective at tick 0 because it is the registration that starts the
	// wave; every other entry is accepted in advance and applied at its own tick.
	Joins []Join `json:"explicit_joins,omitempty"`
	// BallotEvents are explicit ballots with their actor, option and effective
	// tick. At least one actor votes A, B and C in the genuine tie case.
	BallotEvents []BallotEvent `json:"ballot_events,omitempty"`
	// RoleEvents are explicit role selections (including cooldown attempts) for
	// already registered participants.
	RoleEvents []RoleEventInput `json:"role_events,omitempty"`
}

// Join is one scheduled registration.
type Join struct {
	Actor string `json:"actor_key"`
	Role  string `json:"role"`
	Tick  int    `json:"scheduled_tick"`
}

// BallotEvent is one scheduled ballot.
type BallotEvent struct {
	Actor  string `json:"actor_key"`
	Option string `json:"option"`
	Tick   int    `json:"scheduled_tick"`
}

// RoleEventInput is one scheduled role selection for an existing participant.
type RoleEventInput struct {
	Actor string `json:"actor_key"`
	Role  string `json:"role"`
	Tick  int    `json:"scheduled_tick"`
}

// CaseOutcome is the authoritative end state of a reference scenario, captured
// after the tick-2800 settlement and before the tick-3000 rollover.
type CaseOutcome struct {
	Result            string `json:"result"`
	WaveNo            int64  `json:"wave_no"`
	Tick              int    `json:"tick"`
	Lifecycle         string `json:"lifecycle"`
	Defeated          bool   `json:"defeated"`
	AnchorPopulation  int64  `json:"anchor_population"`
	Difficulty        int    `json:"difficulty"`
	BaseHP            int64  `json:"base_hp"`
	WallTier          int    `json:"wall_tier"`
	WallHP            int64  `json:"wall_hp"`
	WallMax           int64  `json:"wall_max"`
	TownhallHP        int64  `json:"townhall_hp"`
	Wood              int64  `json:"wood"`
	Stone             int64  `json:"stone"`
	Food              int64  `json:"food"`
	ConstructionWork  int64  `json:"construction_work"`
	TotalWins         int64  `json:"total_wins"`
	ConsecutiveWins   int    `json:"consecutive_wins"`
	ConsecutiveLosses int    `json:"consecutive_losses"`
	VoteChosen        string `json:"vote_chosen"`
	SessionXPTotal    int64  `json:"session_xp_total"`
	BossHPAt2600      int64  `json:"boss_hp_at_2600"`
	OrdinaryHPAt2600  int64  `json:"ordinary_hp_at_2600"`
	StateDigest       string `json:"state_digest"`
	// VoteCounts is the applied-ballot tally; it proves a tie is a real 1/1/1 tie
	// rather than a label. It is emitted only for even (voting) waves.
	VoteCounts *VoteTally `json:"vote_counts,omitempty"`
	// MemberState is the complete participant state. It is emitted for cases with
	// explicit registrations, so registration and cooldown bookkeeping is compared
	// field by field instead of being inferred from aggregates.
	MemberState []MemberSnapshot `json:"member_state,omitempty"`
	// RoleEvents is the independent model's role decision log, emitted only for
	// cases that script explicit role selections.
	RoleEvents []RoleLogRow `json:"role_events,omitempty"`
}

// GoldenCase is one curated, reproducible reference vector.
type GoldenCase struct {
	ID      string      `json:"id"`
	Note    string      `json:"note"`
	Inputs  CaseInputs  `json:"inputs"`
	Outcome CaseOutcome `json:"outcome"`
	Trace   []TraceRow  `json:"boundary_trace"`
}

// SweepAggregate summarises the population/role sweep.
type SweepAggregate struct {
	Population   int64 `json:"population"`
	FighterShare int   `json:"fighter_share_percent"`
	Wins         int   `json:"wins"`
	Runs         int   `json:"runs"`
	MinFighters  int   `json:"min_fighters_to_win_boss_d1"`
}

// GoldenFile is the machine-consumable artifact M02 must consume.
type GoldenFile struct {
	GeneratedBy            string           `json:"generated_by"`
	FixtureSchema          int              `json:"fixture_schema"`
	RulesVersion           string           `json:"rules_version"`
	CanonicalSchemaVersion int              `json:"canonical_schema_version"`
	SweepNote              string           `json:"sweep_note"`
	SweepRuns              int              `json:"sweep_runs"`
	SweepDigest            string           `json:"sweep_digest"`
	Aggregates             []SweepAggregate `json:"sweep_aggregates"`
	Cases                  []GoldenCase     `json:"cases"`
}

// GoldenManifest pins the generated fixture so a production test can enforce its
// count, identity, versions and file digest instead of trusting a prose claim.
type GoldenManifest struct {
	Schema                 int      `json:"schema"`
	FixtureSchema          int      `json:"fixture_schema"`
	RulesVersion           string   `json:"rules_version"`
	CanonicalSchemaVersion int      `json:"canonical_schema_version"`
	Source                 string   `json:"source"`
	SourceDigestSHA256     string   `json:"source_digest_sha256"`
	FixtureDigestSHA256    string   `json:"fixture_digest_sha256"`
	CaseCount              int      `json:"case_count"`
	CaseIDs                []string `json:"case_ids"`
	Provenance             string   `json:"provenance"`
	Regeneration           string   `json:"regeneration"`
}

// Case is a scenario definition with a convenient literal form.
type Case struct {
	ID     string
	Note   string
	Inputs CaseInputs
}

func defaultInputs() CaseInputs {
	return CaseInputs{
		WaveNo:     7,
		WallTier:   1,
		WallHP:     BaseWallMax,
		TownhallHP: TownhallMaxDmg,
		Wood:       400,
		Stone:      250,
		Food:       300,
	}
}

// inputEvent is one typed explicit input with its effective tick.
type inputEvent struct {
	Tick   int
	Kind   ScheduleKind
	Actor  string
	Role   Role
	Option string
}

// inputEvents validates the typed case inputs and returns them ordered by
// effective tick, preserving declaration order inside one tick. Invalid inputs
// abort generation: a case must never silently drop a registration or a ballot.
func inputEvents(in CaseInputs) []inputEvent {
	var events []inputEvent
	registered := map[string]bool{}
	if len(in.Joins) > 0 {
		for _, join := range in.Joins {
			role, ok := ParseRole(join.Role)
			if !ok {
				panic(fmt.Sprintf("case input: unknown role %q", join.Role))
			}
			if registered[join.Actor] {
				panic(fmt.Sprintf("case input: duplicate join actor %q", join.Actor))
			}
			if join.Tick < 0 || join.Tick > FinalCombatTick {
				panic(fmt.Sprintf("case input: join tick %d outside 0..2800", join.Tick))
			}
			registered[join.Actor] = true
			events = append(events, inputEvent{Tick: join.Tick, Kind: ScheduleJoin, Actor: join.Actor, Role: role})
		}
		if events[0].Tick != 0 {
			panic("case input: the first explicit join must be effective at tick 0")
		}
		for _, join := range in.Joins[1:] {
			if join.Tick == 0 {
				panic("case input: only the starting registration can be effective at tick 0")
			}
		}
	} else {
		for index, actor := range syntheticActors(in) {
			// The production contract starts a wave with exactly one accepted
			// registration at tick 0; every later registration is accepted at tick 0
			// and becomes effective at tick 1. The reference model uses the same
			// shape so participant state is directly comparable.
			tick := 1
			if index == 0 {
				tick = 0
			}
			registered[actor.Actor] = true
			events = append(events, inputEvent{Tick: tick, Kind: ScheduleJoin, Actor: actor.Actor, Role: actor.Role})
		}
	}
	for _, event := range in.RoleEvents {
		role, ok := ParseRole(event.Role)
		if !ok {
			panic(fmt.Sprintf("case input: unknown role %q", event.Role))
		}
		if !registered[event.Actor] {
			panic(fmt.Sprintf("case input: role event for unregistered actor %q", event.Actor))
		}
		if event.Tick < 1 || event.Tick > FinalCombatTick {
			panic(fmt.Sprintf("case input: role event tick %d outside 1..2800", event.Tick))
		}
		events = append(events, inputEvent{Tick: event.Tick, Kind: ScheduleRoleChange, Actor: event.Actor, Role: role})
	}
	if in.Ballot != "" {
		if !validOption(in.Ballot) {
			panic(fmt.Sprintf("case input: unknown ballot option %q", in.Ballot))
		}
		for _, actor := range in.actorNames() {
			events = append(events, inputEvent{Tick: 900, Kind: ScheduleBallot, Actor: actor, Option: in.Ballot})
		}
	}
	for _, ballot := range in.BallotEvents {
		if !validOption(ballot.Option) {
			panic(fmt.Sprintf("case input: unknown ballot option %q", ballot.Option))
		}
		if ballot.Tick < VoteWindowStart || ballot.Tick > VoteWindowEnd {
			panic(fmt.Sprintf("case input: ballot tick %d outside [800,1199]", ballot.Tick))
		}
		if !registered[ballot.Actor] {
			panic(fmt.Sprintf("case input: ballot for unregistered actor %q", ballot.Actor))
		}
		events = append(events, inputEvent{Tick: ballot.Tick, Kind: ScheduleBallot, Actor: ballot.Actor, Option: ballot.Option})
	}
	sort.SliceStable(events, func(i, j int) bool { return events[i].Tick < events[j].Tick })
	return events
}

func validOption(option string) bool {
	return option == "A" || option == "B" || option == "C"
}

// actorNames lists the registered actor names in declaration order.
func (in CaseInputs) actorNames() []string {
	if len(in.Joins) > 0 {
		names := make([]string, 0, len(in.Joins))
		for _, join := range in.Joins {
			names = append(names, join.Actor)
		}
		return names
	}
	synthetic := syntheticActors(in)
	names := make([]string, 0, len(synthetic))
	for _, actor := range synthetic {
		names = append(names, actor.Actor)
	}
	return names
}

func buildSim(in CaseInputs) *Sim {
	s := NewSim()
	s.WaveNo = in.WaveNo
	s.ConsecutiveWins = in.ConsecutiveWins
	s.ConsecutiveLosses = in.ConsecutiveLosses
	s.WallTier = in.WallTier
	s.WallHP = in.WallHP
	s.TownhallHP = in.TownhallHP
	s.Wood = in.Wood
	s.Stone = in.Stone
	s.Food = in.Food
	s.ConstructionWork = in.ConstructionWork
	s.Lifecycle = "running"
	s.Tick = 0

	for index, event := range inputEvents(in) {
		s.Pending = append(s.Pending, scheduledEvent{
			Tick:   event.Tick,
			Seq:    index,
			Kind:   event.Kind,
			Actor:  event.Actor,
			Role:   event.Role,
			Option: event.Option,
		})
	}
	// Tick-0 inputs belong to the wave start and are effective before the first
	// gameplay tick.
	s.applyDue()
	return s
}

type actorRole struct {
	Actor string
	Role  Role
}

func syntheticActors(in CaseInputs) []actorRole {
	var out []actorRole
	for i := 0; i < in.Builders; i++ {
		out = append(out, actorRole{fmt.Sprintf("b%05d", i), RoleBuilder})
	}
	for i := 0; i < in.Guards; i++ {
		out = append(out, actorRole{fmt.Sprintf("g%05d", i), RoleGuard})
	}
	for i := 0; i < in.Fighters; i++ {
		out = append(out, actorRole{fmt.Sprintf("f%05d", i), RoleFighter})
	}
	return out
}

// RunTo advances the simulation until the given wave tick (inclusive).
func (s *Sim) RunTo(tick int) {
	for s.Lifecycle == "running" && s.Tick < tick {
		s.Tick++
		s.ProcessTick()
	}
}

// CloseWave performs the single tick-3000 control transition.
func (s *Sim) CloseWave() {
	if s.Lifecycle != "running" {
		return
	}
	s.Tick = WaveTicks
	s.rollover()
}

// RunWave advances exactly one wave to rollover.
func (s *Sim) RunWave() {
	s.RunTo(WaveTicks - 1)
	s.CloseWave()
}

func outcomeOf(s *Sim, inputs CaseInputs) CaseOutcome {
	var boss2600, ord2600 int64
	for _, row := range s.Trace {
		if row.Tick == 2600 {
			boss2600 = row.BossHP
			ord2600 = row.OrdinaryHP
		}
	}
	out := CaseOutcome{
		Result:            s.Result,
		WaveNo:            s.WaveNo,
		Tick:              s.Tick,
		Lifecycle:         s.Lifecycle,
		Defeated:          s.Defeated,
		AnchorPopulation:  s.AnchorPopulation,
		Difficulty:        s.Difficulty,
		BaseHP:            s.BaseHP,
		WallTier:          s.WallTier,
		WallHP:            s.WallHP,
		WallMax:           s.wallMax(),
		TownhallHP:        s.TownhallHP,
		Wood:              s.Wood,
		Stone:             s.Stone,
		Food:              s.Food,
		ConstructionWork:  s.ConstructionWork,
		TotalWins:         s.TotalWins,
		ConsecutiveWins:   s.ConsecutiveWins,
		ConsecutiveLosses: s.ConsecutiveLosses,
		VoteChosen:        s.VoteChosen,
		SessionXPTotal:    s.totalXP(),
		BossHPAt2600:      boss2600,
		OrdinaryHPAt2600:  ord2600,
		StateDigest:       s.StateDigest(),
	}
	if VoteAvailableForFixture(s.WaveNo) {
		a, b, c := s.voteScores()
		out.VoteCounts = &VoteTally{A: a, B: b, C: c}
	}
	if len(inputs.Joins) > 0 {
		out.MemberState = s.memberSnapshots()
	}
	if len(inputs.RoleEvents) > 0 {
		out.RoleEvents = append([]RoleLogRow(nil), s.RoleLog...)
	}
	return out
}

// VoteAvailableForFixture mirrors the documented even-wave voting rule.
func VoteAvailableForFixture(waveNo int64) bool { return waveNo > 0 && waveNo%2 == 0 }

func runCase(c Case) GoldenCase {
	s := buildSim(c.Inputs)
	s.RunTo(WaveTicks - 1) // settlement at 2800, before the rollover reset
	return GoldenCase{ID: c.ID, Note: c.Note, Inputs: c.Inputs, Outcome: outcomeOf(s, c.Inputs), Trace: s.Trace}
}

// curatedCases returns the reference vectors. Every expectation asserted in
// assertions() was derived by hand from the documented formulas first and then
// compared with this independent implementation.
func curatedCases() []Case {
	bal03a := defaultInputs()
	bal03a.WaveNo = 5
	bal03a.Builders, bal03a.Guards, bal03a.Fighters = 2, 2, 6

	bal03b := defaultInputs()
	bal03b.WaveNo = 5
	bal03b.Builders, bal03b.Guards, bal03b.Fighters = 8, 8, 14

	soloFighter := defaultInputs()
	soloFighter.WaveNo = 5
	soloFighter.Fighters = 1

	soloBuilder := defaultInputs()
	soloBuilder.WaveNo = 5
	soloBuilder.Builders = 1

	voteA := defaultInputs()
	voteA.WaveNo = 4
	voteA.Builders, voteA.Guards, voteA.Fighters = 2, 2, 6
	voteA.Ballot = "A"

	voteB := defaultInputs()
	voteB.WaveNo = 4
	voteB.Builders, voteB.Guards, voteB.Fighters = 2, 2, 6
	voteB.Ballot = "B"

	voteC := defaultInputs()
	voteC.WaveNo = 4
	voteC.Builders, voteC.Guards, voteC.Fighters = 6, 2, 2
	voteC.Ballot = "C"

	zeroVoteAffordable := defaultInputs()
	zeroVoteAffordable.WaveNo = 4
	zeroVoteAffordable.Builders = 2

	zeroVoteUnaffordable := defaultInputs()
	zeroVoteUnaffordable.WaveNo = 4
	zeroVoteUnaffordable.Wood = 40
	zeroVoteUnaffordable.Fighters = 2

	// C010: the previous fixture only registered three builders and never cast a
	// ballot, so the label "tie" was never executed. This fixture casts one A, one
	// B and one C ballot at a legal tick, producing a genuine 1/1/1 tie.
	tieVote := defaultInputs()
	tieVote.WaveNo = 4
	tieVote.Joins = []Join{
		{Actor: "a1", Role: "builder", Tick: 0},
		{Actor: "a2", Role: "builder", Tick: 1},
		{Actor: "a3", Role: "builder", Tick: 1},
	}
	tieVote.BallotEvents = []BallotEvent{
		{Actor: "a1", Option: "A", Tick: 900},
		{Actor: "a2", Option: "B", Tick: 900},
		{Actor: "a3", Option: "C", Tick: 900},
	}

	// C024: the same 1/1/1 tie with an unaffordable A. The documented selection
	// must skip A and choose B, proving the affordability filter runs inside the
	// tie-break order rather than only for the zero-ballot default.
	tieUnaffordableA := defaultInputs()
	tieUnaffordableA.WaveNo = 4
	tieUnaffordableA.Wood = 40
	tieUnaffordableA.Joins = []Join{
		{Actor: "u1", Role: "fighter", Tick: 0},
		{Actor: "u2", Role: "fighter", Tick: 1},
		{Actor: "u3", Role: "fighter", Tick: 1},
	}
	tieUnaffordableA.BallotEvents = []BallotEvent{
		{Actor: "u1", Option: "A", Tick: 900},
		{Actor: "u2", Option: "B", Tick: 900},
		{Actor: "u3", Option: "C", Tick: 900},
	}

	// C025: one latest ballot per player wins. u1 first votes A and later changes
	// to C at a legal later tick, so the applied tally must be A=1, C=2.
	revote := defaultInputs()
	revote.WaveNo = 4
	revote.Joins = []Join{
		{Actor: "v1", Role: "fighter", Tick: 0},
		{Actor: "v2", Role: "fighter", Tick: 1},
		{Actor: "v3", Role: "fighter", Tick: 1},
	}
	revote.BallotEvents = []BallotEvent{
		{Actor: "v1", Option: "A", Tick: 900},
		{Actor: "v2", Option: "A", Tick: 900},
		{Actor: "v3", Option: "C", Tick: 900},
		{Actor: "v1", Option: "C", Tick: 1000},
	}

	defeat := defaultInputs()
	defeat.WaveNo = 5
	defeat.WallHP = 10
	defeat.TownhallHP = 10
	defeat.Guards = 30

	upgrade := defaultInputs()
	upgrade.WaveNo = 7
	upgrade.Builders = 36
	upgrade.Fighters = 30

	repair := defaultInputs()
	repair.WaveNo = 7
	repair.WallHP = 900
	repair.Builders = 36
	repair.Fighters = 30

	capWood := defaultInputs()
	capWood.WaveNo = 7
	capWood.Wood = 999900
	capWood.Builders = 20
	capWood.Fighters = 20

	huge := defaultInputs()
	huge.WaveNo = 5
	huge.Fighters = 10000

	recovery := defaultInputs()
	recovery.WaveNo = 5
	recovery.ConsecutiveLosses = 3
	recovery.Fighters = 30

	hardWin := defaultInputs()
	hardWin.WaveNo = 5
	hardWin.ConsecutiveWins = 9
	hardWin.Fighters = 30

	lossNoDefeat := defaultInputs()
	lossNoDefeat.WaveNo = 7
	lossNoDefeat.Builders = 10

	cliffP4 := defaultInputs()
	cliffP4.WaveNo = 5
	cliffP4.Builders = 4

	cliffP5 := defaultInputs()
	cliffP5.WaveNo = 5
	cliffP5.Builders = 5

	bossIncoming := defaultInputs()
	bossIncoming.WaveNo = 5
	bossIncoming.Fighters = 30

	lateJoin := defaultInputs()
	lateJoin.WaveNo = 7
	lateJoin.Joins = []Join{
		{Actor: "early", Role: "fighter", Tick: 0},
		{Actor: "anchor_edge", Role: "fighter", Tick: 1599},
		{Actor: "post_anchor", Role: "fighter", Tick: 1600},
		{Actor: "late", Role: "fighter", Tick: 1999},
	}

	// C023: the previous fixture registered one builder and never selected another
	// role, so the advertised cooldown path was never executed. This script covers
	// the free first switch, a rejected attempt 599 ticks later, and the accepted
	// switch at the exact 600-tick boundary.
	cooldown := defaultInputs()
	cooldown.WaveNo = 7
	cooldown.Joins = []Join{{Actor: "solo", Role: "builder", Tick: 0}}
	cooldown.RoleEvents = []RoleEventInput{
		{Actor: "solo", Role: "fighter", Tick: 1}, // first switch: permitted immediately
		{Actor: "solo", Role: "guard", Tick: 600}, // 599 ticks since the last change: rejected
		{Actor: "solo", Role: "guard", Tick: 601}, // exactly 600 ticks: accepted
	}

	return []Case{
		{ID: "C001", Note: "QA-BAL-03 P=10 boss B2/G2/F6; boss dies at 2600, victory settled at 2800", Inputs: bal03a},
		{ID: "C002", Note: "QA-BAL-03 P=30 boss B8/G8/F14; boss survives 2600 with 16 HP and dies at 2800 (tick-2800 combat is outcome-critical)", Inputs: bal03b},
		{ID: "C003", Note: "Solo fighter boss wave: militia 40 applies, no builder income", Inputs: soloFighter},
		{ID: "C004", Note: "Solo builder boss wave: militia makes P=1 viable", Inputs: soloBuilder},
		{ID: "C005", Note: "Vote A: 50 wood at close, guard block 115% (8->9)", Inputs: voteA},
		{ID: "C006", Note: "Vote B: attack 115% (40->46) at every combat boundary", Inputs: voteB},
		{ID: "C007", Note: "Vote C: guard block 90% and halved loss penalties", Inputs: voteC},
		{ID: "C008", Note: "Zero ballots, A affordable: deterministic default A is charged", Inputs: zeroVoteAffordable},
		{ID: "C009", Note: "Zero ballots, wood<50: fallback C, no deduction", Inputs: zeroVoteUnaffordable},
		{ID: "C010", Note: "Genuine A/B/C tie: three ballots at tick 900, tally 1/1/1, deterministic priority selects A", Inputs: tieVote},
		{ID: "C011", Note: "DEFEATED at 2200: later economy/XP skipped, exactly one settlement, recovery floors", Inputs: defeat},
		{ID: "C012", Note: "Eight builders: construction work reaches 200 and one upgrade is applied at 2600", Inputs: upgrade},
		{ID: "C013", Note: "Damaged wall repaired from capacity, wood cost ceil(repair/5)", Inputs: repair},
		{ID: "C014", Note: "Resource generation clamps at 1,000,000", Inputs: capWood},
		{ID: "C015", Note: "P=10000 all fighters: sqrt saturation cap and int64 bounds", Inputs: huge},
		{ID: "C016", Note: "Three consecutive losses: 55% recovery modifier", Inputs: recovery},
		{ID: "C017", Note: "Difficulty cap d=10 after nine consecutive wins", Inputs: hardWin},
		{ID: "C018", Note: "Loss without defeat: penalties only, no 250 HP floors", Inputs: lossNoDefeat},
		{ID: "C019", Note: "Militia cliff: P=4 all-builder wins", Inputs: cliffP4},
		{ID: "C020", Note: "Militia cliff: P=5 all-builder loses", Inputs: cliffP5},
		{ID: "C021", Note: "Boss incoming: ordinary cleared at 2200, boss spawn 2600, victory 2800", Inputs: bossIncoming},
		{ID: "C022", Note: "Anchor P counts joined_tick<=1599 only", Inputs: lateJoin},
		{ID: "C023", Note: "Role change: first switch immediate, 599-tick attempt rejected, exact 600-tick switch accepted", Inputs: cooldown},
		{ID: "C024", Note: "Genuine 1/1/1 tie with unaffordable A: affordability filter selects B", Inputs: tieUnaffordableA},
		{ID: "C025", Note: "Re-vote latest-wins: v1 A->C at tick 1000, applied tally A=1 C=2, C selected", Inputs: revote},
	}
}

// assertions returns hard failures for documented expectations. Every expected
// value was computed by hand from 02_GAME_RULES.md before being asserted here.
func assertions() []string {
	var errs []string
	add := func(cond bool, format string, args ...any) {
		if !cond {
			errs = append(errs, fmt.Sprintf(format, args...))
		}
	}

	// --- enemy HP and boss HP tables ---
	type hpRow struct {
		P, CW, CL int
		want      int64
	}
	for _, r := range []hpRow{
		{1, 0, 0, 86}, {3, 0, 0, 86}, {4, 0, 0, 124}, {5, 0, 0, 124},
		{10, 0, 0, 162}, {30, 0, 0, 238}, {100, 0, 0, 428},
		{1000, 0, 0, 1226}, {10000, 0, 0, 3848},
		{10, 9, 0, 234}, {30, 3, 0, 262}, {10000, 9, 0, 3920},
		{10, 0, 1, 137}, {10, 0, 3, 89}, {30, 0, 2, 166},
	} {
		s := NewSim()
		s.AnchorPopulation = int64(r.P)
		s.ConsecutiveWins = r.CW
		s.ConsecutiveLosses = r.CL
		s.Difficulty = 1 + r.CW
		if s.Difficulty > 10 {
			s.Difficulty = 10
		}
		got := s.baseHPValue()
		add(got == r.want, "base_hp(P=%d,cw=%d,cl=%d)=%d want %d", r.P, r.CW, r.CL, got, r.want)
	}
	for _, tc := range []struct{ hp, want int64 }{
		{86, 18}, {124, 25}, {162, 33}, {238, 48}, {428, 86},
	} {
		add(ceilDiv(tc.hp, BossHPOrder) == tc.want, "boss_hp(%d)=%d want %d", tc.hp, ceilDiv(tc.hp, BossHPOrder), tc.want)
	}
	for _, tc := range []struct{ n, want int64 }{
		{0, 0}, {1, 1}, {3, 1}, {4, 2}, {6, 2}, {9, 3}, {10, 3},
		{30, 5}, {1000, 31}, {10000, 100},
	} {
		add(ISqrt(tc.n) == tc.want, "isqrt(%d)=%d want %d", tc.n, ISqrt(tc.n), tc.want)
	}

	// --- QA-BAL-03 explicit attack vectors ---
	p10 := NewSim()
	p10.AnchorPopulation = 10
	add(FighterAttackPerIsqrt*ISqrt(6)+p10.militiaAttack() == 40, "QA-BAL-03 P=10 attack must be 40")
	p30 := NewSim()
	p30.AnchorPopulation = 30
	add(FighterAttackPerIsqrt*ISqrt(14)+p30.militiaAttack() == 54, "QA-BAL-03 P=30 attack must be 54")
	add(GuardBlockPerIsqrt*ISqrt(2) == 8, "guard block G=2 must be 8")
	add(GuardBlockPerIsqrt*ISqrt(8) == 16, "guard block G=8 must be 16")
	for _, tc := range []struct{ p, want int64 }{{1, 40}, {4, 28}, {10, 4}, {11, 0}, {30, 0}} {
		s := NewSim()
		s.AnchorPopulation = tc.p
		add(s.militiaAttack() == tc.want, "militia(P=%d)=%d want %d", tc.p, s.militiaAttack(), tc.want)
	}

	// --- builder / repair / upgrade unit vectors ---
	{
		s := NewSim()
		s.Lifecycle = "running"
		s.Tick = 100
		s.WallHP = 900
		s.Wood = 400
		s.builderPhase(10)
		// generation wood 400+2*3=406, stone 250+3=253; capacity 24; need 100;
		// repair 24, cost ceil(24/5)=5, wall 924, leftover 0, work 0.
		add(s.Wood == 401, "repair wood=%d want 401", s.Wood)
		add(s.WallHP == 924, "repair wall=%d want 924", s.WallHP)
		add(s.Stone == 253, "repair stone=%d want 253", s.Stone)
		add(s.ConstructionWork == 0, "repair work=%d want 0", s.ConstructionWork)
	}
	{
		s := NewSim()
		s.Lifecycle = "running"
		s.Tick = 100
		s.builderPhase(8)
		add(s.ConstructionWork == 16, "work=%d want 16", s.ConstructionWork)
		add(s.Wood == 404, "work wood=%d want 404", s.Wood)
		for i := 0; i < 11; i++ {
			s.builderPhase(8)
		}
		add(s.WallTier == 1, "tier=%d want 1 at 192 work", s.WallTier)
		s.builderPhase(8) // 208 -> clamp 200 -> upgrade
		add(s.WallTier == 2, "tier=%d want 2", s.WallTier)
		add(s.ConstructionWork == 0, "work after upgrade=%d want 0", s.ConstructionWork)
		add(s.WallHP == 1200, "wall after upgrade=%d want 1200", s.WallHP)
		beforeStone := s.Stone
		s.builderPhase(8)
		add(s.WallTier == 2, "at most one upgrade per boundary violated: tier=%d", s.WallTier)
		add(s.Stone == beforeStone+2, "stone after second boundary=%d want %d", s.Stone, beforeStone+2)
	}

	// --- attack distribution unit vectors ---
	{
		s := NewSim()
		s.BaseHP = 162
		s.Lanes = [3]Pool{{Runner: 13, Grunt: 30, Brute: 12}, {Runner: 12, Grunt: 28, Brute: 11}, {Runner: 14, Grunt: 30, Brute: 12}}
		wasted := s.applyAttack(40, 0)
		add(s.ordinary() == 122, "attack ordinary=%d want 122", s.ordinary())
		add(wasted == 0, "attack wasted=%d want 0", wasted)
		s2 := NewSim()
		s2.BaseHP = 162
		s2.Lanes = [3]Pool{{Runner: 13, Grunt: 30, Brute: 12}, {Runner: 12, Grunt: 28, Brute: 11}, {Runner: 14, Grunt: 30, Brute: 12}}
		s2.applyAttack(40, 1)
		add(s2.ordinary() == 122, "rotated attack ordinary=%d want 122", s2.ordinary())
	}

	// --- settlement penalty unit vectors ---
	{
		mk := func(vote string) int64 {
			s := NewSim()
			s.Lifecycle = "running"
			s.Tick = FinalCombatTick
			s.Wood, s.Stone, s.Food = 400, 300, 300
			s.Lanes = [3]Pool{{Runner: 1}}
			s.VoteChosen = vote
			s.settle()
			return s.Wood
		}
		add(mk("") == 320, "loss wood penalty=%d want 320", mk(""))
		add(mk("C") == 360, "vote C halved penalty=%d want 360", mk("C"))
	}

	// --- curated case matrix ---
	byID := map[string]GoldenCase{}
	for _, c := range curatedCases() {
		byID[c.ID] = runCase(c)
	}

	c1 := byID["C001"]
	add(c1.Outcome.Result == "victory", "C001 result=%s want victory", c1.Outcome.Result)
	add(c1.Outcome.BaseHP == 162, "C001 base_hp=%d want 162", c1.Outcome.BaseHP)
	add(c1.Outcome.WallHP == 1000, "C001 wall=%d want 1000", c1.Outcome.WallHP)
	add(c1.Outcome.Wood == 518, "C001 wood=%d want 518", c1.Outcome.Wood)
	add(c1.Outcome.Stone == 324, "C001 stone=%d want 324", c1.Outcome.Stone)
	add(c1.Outcome.Food == 350, "C001 food=%d want 350", c1.Outcome.Food)
	add(c1.Outcome.SessionXPTotal == 100, "C001 xp=%d want 100", c1.Outcome.SessionXPTotal)
	add(c1.Outcome.WallTier == 1, "C001 tier=%d want 1", c1.Outcome.WallTier)
	add(c1.Outcome.OrdinaryHPAt2600 == 0, "C001 ordinary@2600=%d want 0", c1.Outcome.OrdinaryHPAt2600)
	add(c1.Outcome.BossHPAt2600 == 0, "C001 boss@2600=%d want 0", c1.Outcome.BossHPAt2600)

	c2 := byID["C002"]
	add(c2.Outcome.Result == "victory", "C002 result=%s want victory", c2.Outcome.Result)
	add(c2.Outcome.BaseHP == 238, "C002 base_hp=%d want 238", c2.Outcome.BaseHP)
	add(c2.Outcome.WallTier == 1, "C002 tier=%d want 1", c2.Outcome.WallTier)
	add(c2.Outcome.WallHP == 1000, "C002 wall=%d want 1000", c2.Outcome.WallHP)
	add(c2.Outcome.Wood == 544, "C002 wood=%d want 544", c2.Outcome.Wood)
	add(c2.Outcome.Stone == 338, "C002 stone=%d want 338", c2.Outcome.Stone)
	add(c2.Outcome.SessionXPTotal == 300, "C002 xp=%d want 300", c2.Outcome.SessionXPTotal)
	add(c2.Outcome.BossHPAt2600 == 16, "C002 boss@2600=%d want 16", c2.Outcome.BossHPAt2600)
	// QA-SIM-10: the tick-2800 combat is what kills this boss; without it the
	// settlement would be a LOSS.
	add(c2.Outcome.BossHPAt2600 > 0 && c2.Outcome.Result == "victory",
		"C002 must prove that the 2800 combat is outcome-critical")

	c4 := byID["C004"]
	add(c4.Outcome.Result == "victory", "C004 result=%s want victory", c4.Outcome.Result)
	add(c4.Outcome.WallHP == 1000, "C004 wall=%d want 1000", c4.Outcome.WallHP)
	add(c4.Outcome.Wood == 524, "C004 wood=%d want 524", c4.Outcome.Wood)
	add(c4.Outcome.Stone == 324, "C004 stone=%d want 324", c4.Outcome.Stone)
	add(c4.Outcome.SessionXPTotal == 10, "C004 xp=%d want 10", c4.Outcome.SessionXPTotal)

	c3 := byID["C003"]
	add(c3.Outcome.WallHP == 992, "C003 wall=%d want 992", c3.Outcome.WallHP)
	add(c3.Outcome.Wood == 500, "C003 wood=%d want 500 (no builder income)", c3.Outcome.Wood)
	add(c3.Outcome.SessionXPTotal == 10, "C003 xp=%d want 10", c3.Outcome.SessionXPTotal)

	// Vote vectors.
	add(byID["C005"].Outcome.VoteChosen == "A", "C005 vote=%s want A", byID["C005"].Outcome.VoteChosen)
	add(byID["C005"].Outcome.Wood == 469, "C005 wood=%d want 469", byID["C005"].Outcome.Wood)
	add(byID["C006"].Outcome.VoteChosen == "B", "C006 vote=%s want B", byID["C006"].Outcome.VoteChosen)
	add(byID["C006"].Outcome.Wood == 520, "C006 wood=%d want 520", byID["C006"].Outcome.Wood)
	add(byID["C007"].Outcome.VoteChosen == "C", "C007 vote=%s want C", byID["C007"].Outcome.VoteChosen)
	add(byID["C007"].Outcome.Result == "loss", "C007 result=%s want loss", byID["C007"].Outcome.Result)
	add(byID["C008"].Outcome.VoteChosen == "A", "C008 zero-ballot default=%s want A", byID["C008"].Outcome.VoteChosen)
	c9 := byID["C009"]
	add(c9.Outcome.VoteChosen == "C", "C009 unaffordable default=%s want C", c9.Outcome.VoteChosen)
	add(c9.Outcome.Wood == 140, "C009 wood=%d want 140 (no 50 deduction)", c9.Outcome.Wood)
	// Genuine tie: three ballots, one per option, at a legal tick.
	c10 := byID["C010"]
	add(c10.Outcome.VoteChosen == "A", "C010 tie=%s want A", c10.Outcome.VoteChosen)
	if c10.Outcome.VoteCounts == nil {
		add(false, "C010 must publish its applied ballot tally")
	} else {
		add(c10.Outcome.VoteCounts.A == 1 && c10.Outcome.VoteCounts.B == 1 && c10.Outcome.VoteCounts.C == 1,
			"C010 tally=%+v want 1/1/1", *c10.Outcome.VoteCounts)
	}
	add(len(c10.Inputs.BallotEvents) == 3, "C010 must script three explicit ballots")
	if len(c10.Outcome.MemberState) == 3 {
		for _, member := range c10.Outcome.MemberState {
			add(member.XPAwarded, "C010 member %s must be an XP-awarded contributor", member.Actor)
		}
	} else {
		add(false, "C010 must publish three participant states")
	}

	// Affordability filter inside the tie-break order.
	c24 := byID["C024"]
	add(c24.Outcome.VoteChosen == "B", "C024 unaffordable-A tie=%s want B", c24.Outcome.VoteChosen)
	if c24.Outcome.VoteCounts != nil {
		add(c24.Outcome.VoteCounts.A == 1 && c24.Outcome.VoteCounts.B == 1 && c24.Outcome.VoteCounts.C == 1,
			"C024 tally=%+v want 1/1/1", *c24.Outcome.VoteCounts)
	} else {
		add(false, "C024 must publish its applied ballot tally")
	}
	add(c24.Outcome.Wood == 140, "C024 wood=%d want 140 (40 start + 100 victory, no A charge)", c24.Outcome.Wood)

	// Latest-ballot-per-player re-vote.
	c25 := byID["C025"]
	add(c25.Outcome.VoteChosen == "C", "C025 re-vote=%s want C", c25.Outcome.VoteChosen)
	if c25.Outcome.VoteCounts != nil {
		add(c25.Outcome.VoteCounts.A == 1 && c25.Outcome.VoteCounts.B == 0 && c25.Outcome.VoteCounts.C == 2,
			"C025 tally=%+v want A=1 B=0 C=2", *c25.Outcome.VoteCounts)
	} else {
		add(false, "C025 must publish its applied ballot tally")
	}

	// Defeat handling.
	d := byID["C011"]
	add(d.Outcome.Result == "loss", "C011 result=%s want loss", d.Outcome.Result)
	add(d.Outcome.Defeated, "C011 must be defeated")
	add(d.Outcome.WallHP == DefeatFloorHP, "C011 wall=%d want 250 floor", d.Outcome.WallHP)
	add(d.Outcome.TownhallHP == DefeatFloorHP, "C011 townhall=%d want 250 floor", d.Outcome.TownhallHP)
	add(d.Outcome.Wood == 320, "C011 wood=%d want 320 (no post-defeat income)", d.Outcome.Wood)
	add(d.Outcome.SessionXPTotal == 300, "C011 pre-defeat XP retained, got %d", d.Outcome.SessionXPTotal)
	add(d.Outcome.ConsecutiveLosses == 1, "C011 consecutive_losses=%d want 1", d.Outcome.ConsecutiveLosses)
	add(d.Outcome.TotalWins == 0, "C011 total_wins=%d want 0", d.Outcome.TotalWins)

	// Upgrade vector: construction work must reach 200*wall_tier and consume
	// exactly one upgrade (structural assertions; exact counters are pinned by
	// the golden file and re-verified by the M02 reducer).
	up := byID["C012"]
	add(up.Outcome.Result == "victory", "C012 result=%s want victory", up.Outcome.Result)
	add(up.Outcome.WallTier == 2, "C012 tier=%d want 2", up.Outcome.WallTier)
	add(up.Outcome.WallHP == 1200, "C012 wall=%d want 1200", up.Outcome.WallHP)
	add(up.Outcome.WallHP == up.Outcome.WallMax, "C012 wall must equal the tier-2 max")
	add(up.Outcome.SessionXPTotal == 660, "C012 xp=%d want 660", up.Outcome.SessionXPTotal)

	// Repair vector: a damaged wall is repaired from capacity and upgrades later.
	rp := byID["C013"]
	add(rp.Outcome.Result == "victory", "C013 result=%s want victory", rp.Outcome.Result)
	add(rp.Outcome.WallTier == 2, "C013 tier=%d want 2", rp.Outcome.WallTier)
	add(rp.Outcome.WallHP == 1200, "C013 wall=%d want 1200", rp.Outcome.WallHP)

	// Resource cap and scale.
	add(byID["C014"].Outcome.Wood == CapResources, "C014 wood=%d want cap %d", byID["C014"].Outcome.Wood, CapResources)
	c15 := byID["C015"]
	add(c15.Outcome.Result == "victory", "C015 result=%s", c15.Outcome.Result)
	add(c15.Outcome.SessionXPTotal == 100000, "C015 xp=%d want 100000", c15.Outcome.SessionXPTotal)

	// Difficulty and recovery modifiers.
	add(byID["C016"].Outcome.BaseHP == 130, "C016 base_hp=%d want 130", byID["C016"].Outcome.BaseHP)
	add(byID["C017"].Outcome.Difficulty == 10, "C017 difficulty=%d want 10", byID["C017"].Outcome.Difficulty)
	add(byID["C017"].Outcome.BaseHP == 310, "C017 base_hp=%d want 310", byID["C017"].Outcome.BaseHP)

	// Non-defeat loss keeps no recovery floor and no 250 HP grant.
	nd := byID["C018"]
	add(nd.Outcome.Result == "loss", "C018 result=%s want loss", nd.Outcome.Result)
	add(!nd.Outcome.Defeated, "C018 must not be defeated")
	add(nd.Outcome.TownhallHP == TownhallMaxDmg, "C018 townhall=%d must stay untouched", nd.Outcome.TownhallHP)

	// Militia cliff (accepted, documented balance characteristic).
	add(byID["C019"].Outcome.Result == "victory", "C019 P=4 all-builder must win, got %s", byID["C019"].Outcome.Result)
	add(byID["C020"].Outcome.Result == "loss", "C020 P=5 all-builder must lose, got %s", byID["C020"].Outcome.Result)

	// Boss-incoming precedence.
	bi := byID["C021"]
	sawIncoming := false
	for _, r := range bi.Trace {
		if r.Tick >= 1600 && r.Tick < 2600 && r.BossLife == "incoming" {
			sawIncoming = true
		}
	}
	add(sawIncoming, "C021 must report BOSS_INCOMING before 2600")
	add(bi.Outcome.Result == "victory", "C021 result=%s want victory", bi.Outcome.Result)

	// Anchor boundary.
	add(byID["C022"].Outcome.AnchorPopulation == 2, "C022 anchor=%d want 2", byID["C022"].Outcome.AnchorPopulation)

	// Role-change cooldown: the case must execute the free switch, a rejected
	// attempt 599 ticks later and the accepted switch exactly at 600 ticks.
	c23 := byID["C023"]
	add(len(c23.Inputs.RoleEvents) == 3, "C023 must script three explicit role selections")
	add(c23.Outcome.AnchorPopulation == 1, "C023 anchor=%d want 1", c23.Outcome.AnchorPopulation)
	if len(c23.Outcome.RoleEvents) != 4 {
		add(false, "C023 role log=%d rows want 4 (registration + three selections)", len(c23.Outcome.RoleEvents))
	} else {
		type wantEvent struct {
			tick     int
			role     string
			previous string
			outcome  string
		}
		for index, want := range []wantEvent{
			{tick: 0, role: "builder", outcome: "registered"},
			{tick: 1, role: "fighter", previous: "builder", outcome: "role_changed"},
			{tick: 600, role: "guard", outcome: "cooldown_rejected"},
			{tick: 601, role: "guard", previous: "fighter", outcome: "role_changed"},
		} {
			got := c23.Outcome.RoleEvents[index]
			add(got.Tick == want.tick && got.Role == want.role && got.PreviousRole == want.previous && got.Outcome == want.outcome,
				"C023 role log[%d]=%+v want %+v", index, got, want)
		}
	}
	if len(c23.Outcome.MemberState) != 1 {
		add(false, "C023 must publish the participant state")
	} else {
		member := c23.Outcome.MemberState[0]
		add(member.Role == "guard", "C023 final role=%s want guard", member.Role)
		add(member.JoinedTick == 0, "C023 joined_tick=%d want 0", member.JoinedTick)
		add(member.LastChangeTick != nil && *member.LastChangeTick == 601,
			"C023 last_change_tick=%v want 601", member.LastChangeTick)
		add(member.FirstSwitchUsed, "C023 first_switch_used must be true")
		add(member.XPAwarded, "C023 participant must be XP-awarded")
	}

	// Global invariants over every trace row.
	for id, c := range byID {
		var bossBefore int64
		for _, r := range c.Trace {
			if r.Tick < BossSpawnTick && r.BossLife == "active" {
				bossBefore++
			}
			add(r.Tick <= FinalCombatTick, "%s trace row after final combat: %d", id, r.Tick)
			add(r.WallHP >= 0 && r.WallHP <= r.WallMax, "%s wall out of bounds at %d", id, r.Tick)
			add(r.TownhallHP >= 0 && r.TownhallHP <= TownhallMaxDmg, "%s townhall out of bounds at %d", id, r.Tick)
			add(r.Wood >= 0 && r.Wood <= CapResources, "%s wood out of bounds at %d", id, r.Tick)
			add(r.Stone >= 0 && r.Stone <= CapResources, "%s stone out of bounds at %d", id, r.Tick)
			add(r.Food >= 0 && r.Food <= CapResources, "%s food out of bounds at %d", id, r.Tick)
			add(r.Work >= 0 && r.Work <= UpgradeWorkPerTier*int64(r.WallTier), "%s work above threshold at %d", id, r.Tick)
			add(r.BossHP >= 0, "%s negative boss hp at %d", id, r.Tick)
		}
		add(bossBefore == 0, "%s boss became attackable before 2600", id)
	}

	// Determinism: identical inputs must produce identical digests.
	for _, c := range curatedCases() {
		a, b := runCase(c), runCase(c)
		add(a.Outcome.StateDigest == b.Outcome.StateDigest, "%s nondeterministic digest", c.ID)
	}

	// Duplicate-role flood must not add strength.
	flood := defaultInputs()
	flood.WaveNo = 5
	s := buildSim(flood)
	for i := 0; i < 1000; i++ {
		got := s.Join("flood", RoleBuilder, 100)
		if i == 0 {
			add(got == JoinRegistered, "first flood join=%s", got)
		} else {
			add(got == JoinNoopSameRole, "flood join %d=%s want noop", i, got)
		}
	}
	bCount, _, _ := s.activeCounts(200)
	add(bCount == 1, "flood produced %d builders want 1", bCount)

	// Role-change cooldown sequence.
	s = buildSim(defaultInputs())
	add(s.Join("solo", RoleBuilder, 0) == JoinRegistered, "cooldown: initial join")
	add(s.Join("solo", RoleFighter, 100) == JoinRoleChanged, "cooldown: first switch must be immediate")
	add(s.Join("solo", RoleGuard, 200) == JoinCooldown, "cooldown: second switch must be rejected")
	add(s.Join("solo", RoleGuard, 699) == JoinCooldown, "cooldown: 599 ticks must be rejected")
	add(s.Join("solo", RoleGuard, 700) == JoinRoleChanged, "cooldown: 600 ticks must be accepted")

	// RESULTS registration auto-starts the next wave with no second comment, and
	// the promoted row must land inside the next wave's anchor window.
	nas := defaultInputs()
	nas.WaveNo = 7
	nas.Fighters = 2
	s = buildSim(nas)
	s.RunTo(FinalCombatTick + 100)
	add(s.Phase() == "results", "expecting results phase, got %s", s.Phase())
	add(s.Join("late1", RoleFighter, 2950) == JoinNextWave, "RESULTS join must queue the next wave")
	add(s.Join("late2", RoleBuilder, 2999) == JoinNextWave, "second RESULTS join must queue")
	s.RunTo(WaveTicks - 1)
	s.CloseWave()
	add(s.Lifecycle == "running", "next wave must auto-start, lifecycle=%s", s.Lifecycle)
	add(s.Tick == 0, "next wave must start at tick 0, got %d", s.Tick)
	add(len(s.Members) == 2, "promoted members=%d want 2", len(s.Members))
	for _, m := range s.Members {
		add(m.JoinedTick == 0, "promoted %s joined_tick=%d want 0", m.Actor, m.JoinedTick)
	}
	s.RunTo(PopulationLockTick)
	add(s.AnchorPopulation == 2, "promoted members must be inside the next anchor window, P=%d", s.AnchorPopulation)

	// A wave with no queued participants returns to WAITING, never losses.
	empty := defaultInputs()
	empty.WaveNo = 7
	empty.Fighters = 1
	s = buildSim(empty)
	s.RunWave()
	add(s.Lifecycle == "waiting", "rollover without queue must be waiting, got %s", s.Lifecycle)
	add(s.Tick == -1, "waiting tick=%d want -1", s.Tick)

	// Session XP is capped at 300 per actor over many waves.
	xp := defaultInputs()
	xp.WaveNo = 7
	xp.Fighters = 1
	s = buildSim(xp)
	for i := 0; i < 30 && s.Lifecycle == "running"; i++ {
		s.RunTo(WaveTicks - 1)
		if s.Lifecycle != "running" {
			break
		}
		s.CloseWave()
		s.WaveNo++
		if s.WaveNo%2 == 0 {
			s.WaveNo++ // keep odd waves so no ballot opens
		}
	}
	var maxXP int64
	for _, v := range s.SessionXP {
		if v > maxXP {
			maxXP = v
		}
	}
	add(maxXP <= CapSessionXP, "session xp=%d must not exceed %d", maxXP, CapSessionXP)

	return errs
}

// sweepCase pairs an input with the requested population/role metadata so the
// aggregate table is keyed by the requested share, not by floor rounding.
type sweepCase struct {
	in    CaseInputs
	pop   int64
	share int
}

// sweepInputs enumerates the required matrix: P=1,3,4,5,10,30,100,1000,10000 x
// fighter share 0/25/50/75/100% x ordinary and boss waves x consecutive wins
// 0/3/9 x consecutive losses 0/1/3, plus vote A/B/C variants at the clean
// baseline. cw>0 with cl>0 is an arithmetic stress pair, not a reachable state.
func sweepInputs() []sweepCase {
	pops := []int64{1, 3, 4, 5, 10, 30, 100, 1000, 10000}
	shares := []int{0, 25, 50, 75, 100}
	cws := []int{0, 3, 9}
	cls := []int{0, 1, 3}
	votes := []string{"", "A", "B", "C"}
	var out []sweepCase
	for _, p := range pops {
		for _, share := range shares {
			f := int((p * int64(share)) / 100)
			rest := int(p) - f
			g := rest / 2
			b := rest - g
			for _, boss := range []bool{false, true} {
				for _, cw := range cws {
					for _, cl := range cls {
						for _, vote := range votes {
							if vote != "" && (cw != 0 || cl != 0) {
								continue
							}
							in := defaultInputs()
							in.Builders, in.Guards, in.Fighters = b, g, f
							in.ConsecutiveWins, in.ConsecutiveLosses = cw, cl
							switch {
							case boss && vote == "":
								in.WaveNo = 5
							case boss:
								in.WaveNo = 10
								in.Ballot = vote
							case vote != "":
								in.WaveNo = 8
								in.Ballot = vote
							default:
								in.WaveNo = 7
							}
							out = append(out, sweepCase{in: in, pop: p, share: share})
						}
					}
				}
			}
		}
	}
	return out
}

func sweepRowKey(in CaseInputs) string {
	return fmt.Sprintf("P=%d;B=%d;G=%d;F=%d;wave=%d;cw=%d;cl=%d;vote=%s",
		in.Builders+in.Guards+in.Fighters, in.Builders, in.Guards, in.Fighters,
		in.WaveNo, in.ConsecutiveWins, in.ConsecutiveLosses, in.Ballot)
}

// runSweep executes the matrix and returns a canonical digest plus aggregates.
func runSweep() (int, string, []SweepAggregate) {
	cases := sweepInputs()
	h := sha256.New()
	type agg struct{ wins, runs int }
	acc := map[string]*agg{}
	for _, sc := range cases {
		s := buildSim(sc.in)
		s.RunTo(WaveTicks - 1)
		line := fmt.Sprintf("%s;result=%s;wall=%d;tier=%d;wood=%d;stone=%d;boss2600=%d\n",
			sweepRowKey(sc.in), s.Result, s.WallHP, s.WallTier, s.Wood, s.Stone, s.BossHP)
		h.Write([]byte(line))
		k := fmt.Sprintf("%d|%d", sc.pop, sc.share)
		if acc[k] == nil {
			acc[k] = &agg{}
		}
		acc[k].runs++
		if s.Result == "victory" {
			acc[k].wins++
		}
	}
	keys := make([]string, 0, len(acc))
	for k := range acc {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		var pi, si, pj, sj int
		fmt.Sscanf(keys[i], "%d|%d", &pi, &si)
		fmt.Sscanf(keys[j], "%d|%d", &pj, &sj)
		if pi != pj {
			return pi < pj
		}
		return si < sj
	})
	minF := map[int64]int{}
	aggs := make([]SweepAggregate, 0, len(keys))
	for _, k := range keys {
		var p, share int
		fmt.Sscanf(k, "%d|%d", &p, &share)
		v := acc[k]
		if _, ok := minF[int64(p)]; !ok {
			minF[int64(p)] = minFightersToWin(int64(p))
		}
		aggs = append(aggs, SweepAggregate{
			Population:   int64(p),
			FighterShare: share,
			Wins:         v.wins,
			Runs:         v.runs,
			MinFighters:  minF[int64(p)],
		})
	}
	return len(cases), hex.EncodeToString(h.Sum(nil)), aggs
}

// minFightersToWin reports the smallest fighter count that clears a boss wave at
// d=1 with no prior losses and no vote. Only isqrt plateaus can change the attack
// value, so the search visits exactly F = 0,1,4,9,... <= population.
func minFightersToWin(pop int64) int {
	var candidates []int
	for k := int64(0); k*k <= CapPopulation; k++ {
		f := k * k
		if f > pop {
			break
		}
		candidates = append(candidates, int(f))
	}
	for _, f := range candidates {
		rest := int(pop) - f
		in := defaultInputs()
		in.WaveNo = 5
		in.Fighters = f
		in.Guards = rest / 2
		in.Builders = rest - in.Guards
		s := buildSim(in)
		s.RunTo(WaveTicks - 1)
		if s.Result == "victory" {
			return f
		}
	}
	return -1
}

func buildGolden() GoldenFile {
	runs, digest, aggs := runSweep()
	var cases []GoldenCase
	for _, c := range curatedCases() {
		cases = append(cases, runCase(c))
	}
	return GoldenFile{
		GeneratedBy:            "tooling/refeval (independent reference evaluator)",
		FixtureSchema:          FixtureSchemaVersion,
		RulesVersion:           RulesVersion,
		CanonicalSchemaVersion: CanonicalSchemaVersion,
		SweepNote:              "P=1,3,4,5,10,30,100,1000,10000 x fighter share 0/25/50/75/100% x ordinary/boss x consecutive_wins 0/3/9 x consecutive_losses 0/1/3 + vote A/B/C at the clean baseline. cw>0 with cl>0 is an arithmetic stress pair, not a reachable game state (a win resets consecutive_losses).",
		SweepRuns:              runs,
		SweepDigest:            digest,
		Aggregates:             aggs,
		Cases:                  cases,
	}
}

// buildManifest pins the generated fixture. The backend differential test reads
// this file and refuses to run if the count, identity, versions or digests do not
// match, so a silently stale or hand-edited matrix cannot pass.
func buildManifest(g GoldenFile, data []byte) GoldenManifest {
	ids := make([]string, 0, len(g.Cases))
	for _, c := range g.Cases {
		ids = append(ids, c.ID)
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	return GoldenManifest{
		Schema:                 2,
		FixtureSchema:          g.FixtureSchema,
		RulesVersion:           g.RulesVersion,
		CanonicalSchemaVersion: g.CanonicalSchemaVersion,
		Source:                 "tooling/refeval/testdata/golden_vectors.json",
		SourceDigestSHA256:     digest,
		FixtureDigestSHA256:    digest,
		CaseCount:              len(g.Cases),
		CaseIDs:                ids,
		Provenance:             "Independent reference evaluator committed in this repository; the backend matrix is generated by `go run ./refeval -write`, never hand-edited.",
		Regeneration:           "Run `go run ./refeval -write` from tooling/; normal tests never rewrite these files.",
	}
}

func marshalJSON(value any) []byte {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(b, '\n')
}

func marshalGolden(g GoldenFile) []byte { return marshalJSON(g) }

func main() {
	write := flag.Bool("write", false, "regenerate the golden vector file and the backend fixture copy")
	check := flag.Bool("check", false, "verify the golden vectors, backend matrix copy and manifest are byte-identical")
	goldenPath := flag.String("golden", "refeval/testdata/golden_vectors.json", "golden vector path")
	matrixPath := flag.String("matrix", "../backend/testdata/golden/matrix.json", "backend fixture copy path")
	manifestPath := flag.String("manifest", "../backend/testdata/golden/manifest.json", "backend fixture manifest path")
	list := flag.Bool("sweep", false, "print the population/role sweep aggregate table")
	flag.Parse()

	errs := assertions()
	g := buildGolden()
	data := marshalGolden(g)
	manifestData := marshalJSON(buildManifest(g, data))

	writeFile := func(path string, content []byte) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", filepath.Dir(path), err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s (%d bytes)\n", path, len(content))
	}
	checkFile := func(path string, content []byte) {
		existing, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
			errs = append(errs, fmt.Sprintf("cannot read %s", path))
			return
		}
		if !bytes.Equal(existing, content) {
			errs = append(errs, fmt.Sprintf("fixture drift: %s differs from the reference evaluator output", path))
		}
	}

	if *write {
		writeFile(*goldenPath, data)
		writeFile(*matrixPath, data)
		writeFile(*manifestPath, manifestData)
	}
	if *check {
		checkFile(*goldenPath, data)
		checkFile(*matrixPath, data)
		checkFile(*manifestPath, manifestData)
	}
	if *list {
		fmt.Printf("%-10s %-14s %-6s %-6s %s\n", "population", "fighter_share", "wins", "runs", "min_fighters")
		for _, a := range g.Aggregates {
			fmt.Printf("%-10d %-14d %-6d %-6d %d\n", a.Population, a.FighterShare, a.Wins, a.Runs, a.MinFighters)
		}
	}

	fmt.Printf("refeval: fixture_schema=%d, %d curated cases, %d sweep runs, sweep_digest=%s\n",
		g.FixtureSchema, len(g.Cases), g.SweepRuns, g.SweepDigest[:16])
	if len(errs) > 0 {
		fmt.Printf("FAIL (%d assertion(s))\n", len(errs))
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		os.Exit(1)
	}
	fmt.Println("PASS: all reference assertions hold")
}
