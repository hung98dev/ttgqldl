// Package main implements the TTGQLDL independent reference evaluator for the
// documented gameplay and simulation equations (02_GAME_RULES.md,
// 03_SIMULATION.md).
//
// This is NOT production game code. It is a second, independent implementation
// used to cross-check the authoritative formulas, to produce executable golden
// vectors for M02 (QA-BAL-04), and to detect impossible outcomes, difficulty
// cliffs, overflow and non-determinism. It must never be imported by the Go
// backend: M02 owns the authoritative reducer.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Rules and format identifiers are part of the authoritative contract
// (01_PRODUCT_SCOPE.md, 02_GAME_RULES.md, 04_DATABASE.md).
const (
	RulesVersion           = "ttg-rules-v1"
	CanonicalSchemaVersion = 1
	// FixtureSchemaVersion versions the independently generated fixture input
	// model. It is deliberately separate from the gameplay rules identifier, the
	// canonical game-state schema and the production private checkpoint format.
	// A change to the fixture shape bumps this value so a production test fails
	// until the new shape is reviewed.
	FixtureSchemaVersion = 2
)

// Documented integer bounds.
const (
	CapResources            int64 = 1_000_000
	CapPopulation           int64 = 10_000
	CapSessionXP            int64 = 300
	XPPerWave               int64 = 10
	RoleChangeCooldownTicks       = 600
	MaxWallTier                   = 20
	MaxVillageLevel               = 50
	MaxConsecutiveWins            = 10
	MaxConsecutiveLosses          = 3
	WaveTicks                     = 3000
	TicksPerSecond                = 10
	TownhallMaxDmg          int64 = 500
	BaseWallMax             int64 = 1000
	WallMaxStep             int64 = 200
	UpgradeWorkPerTier      int64 = 200
	UpgradeStonePerTier     int64 = 100
	BossHPOrder                   = 5
	LaneWestShare                 = 34
	LaneCenterShare               = 32
	SubpoolRunnerShare            = 25
	SubpoolGruntShare             = 55
	AllianceRepairPerIsqrt  int64 = 8
	GuardBlockPerIsqrt      int64 = 8
	FighterAttackPerIsqrt   int64 = 18
	BuilderWoodPerIsqrt     int64 = 2
	EnemyScalePerIsqrt      int64 = 38
	EnemyBase               int64 = 40
	EnemyPerDifficulty      int64 = 8
	RecoveryLossStep        int64 = 15
	RecoveryPercent         int64 = 100
	MilitiaMax              int64 = 40
	MilitiaStep             int64 = 4
	MilitiaMaxPopulation    int64 = 10
	RepairWoodDivisor       int64 = 5
	VictoryRewardWood       int64 = 100
	VictoryRewardStone      int64 = 60
	VictoryRewardFood       int64 = 50
	DefeatFloorHP           int64 = 250
	DefeatFloorWood         int64 = 100
	VoteResourceCost        int64 = 50
	VoteWindowStart               = 800
	VoteWindowEnd                 = 1199
	VoteCloseTick                 = 1200
	PopulationLockTick            = 1600
	BossSpawnTick                 = 2600
	FinalCombatTick               = 2800
	RolloverTick                  = 3000
	CombatStartTick               = 1800
	FirstSwitchFree               = true
)

// Role is the canonical participant role.
type Role int8

const (
	RoleNone    Role = 0
	RoleBuilder Role = 1
	RoleGuard   Role = 2
	RoleFighter Role = 3
)

func (r Role) String() string {
	switch r {
	case RoleBuilder:
		return "builder"
	case RoleGuard:
		return "guard"
	case RoleFighter:
		return "fighter"
	}
	return "none"
}

// ParseRole accepts only the canonical English role names used by the model.
func ParseRole(s string) (Role, bool) {
	switch s {
	case "builder":
		return RoleBuilder, true
	case "guard":
		return RoleGuard, true
	case "fighter":
		return RoleFighter, true
	}
	return RoleNone, false
}

// ISqrt is the documented integer floor square root (no floating point).
func ISqrt(n int64) int64 {
	if n <= 0 {
		return 0
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func clamp(v, lo, hi int64) int64 {
	return min64(hi, max64(lo, v))
}

func ceilDiv(a, b int64) int64 {
	if a <= 0 {
		return 0
	}
	return (a + b - 1) / b
}

// Pool is one lane's ordinary enemy subpools; each value is an HP counter.
type Pool struct {
	Runner int64 `json:"runner_hp"`
	Grunt  int64 `json:"grunt_hp"`
	Brute  int64 `json:"brute_hp"`
}

func (p Pool) Total() int64 { return p.Runner + p.Grunt + p.Brute }

// Member is one registered participant of the current wave.
type Member struct {
	Actor          string `json:"actor_key"`
	Role           Role   `json:"role"`
	JoinedTick     int    `json:"joined_tick"`
	LastChangeTick int    `json:"last_change_tick"`
	FirstSwitch    bool   `json:"first_switch_used"`
	XPAwarded      bool   `json:"xp_awarded"`
}

// Ballot is one vote with its acceptance tick.
type Ballot struct {
	Actor  string
	Option string
	Tick   int
}

// ScheduleKind enumerates the explicit input events a curated case can schedule.
// They are applied at their effective tick, exactly like the production reducer
// applies an accepted command when the logical clock reaches it.
type ScheduleKind int8

const (
	// ScheduleJoin registers a new participant for the current wave.
	ScheduleJoin ScheduleKind = 1
	// ScheduleRoleChange is a role selection by an already registered participant;
	// it exercises the free-first-switch and 600-tick cooldown contract.
	ScheduleRoleChange ScheduleKind = 2
	// ScheduleBallot is one ballot effective inside the voting window.
	ScheduleBallot ScheduleKind = 3
)

// scheduledEvent is one typed, explicit input event with its effective tick.
type scheduledEvent struct {
	Tick   int
	Seq    int
	Kind   ScheduleKind
	Actor  string
	Role   Role
	Option string
}

// RoleLogRow records the independent model's outcome for one accepted role
// selection. It is the reference counterpart of the production role events.
type RoleLogRow struct {
	Tick         int    `json:"tick"`
	Actor        string `json:"actor_key"`
	Role         string `json:"role"`
	PreviousRole string `json:"previous_role,omitempty"`
	Outcome      string `json:"outcome"`
}

// MemberSnapshot is one participant's authoritative state, used to prove that a
// case genuinely exercises registration and role-change bookkeeping.
type MemberSnapshot struct {
	Actor           string `json:"actor_key"`
	Role            string `json:"role"`
	JoinedTick      int    `json:"joined_tick"`
	LastChangeTick  *int   `json:"last_change_tick"`
	FirstSwitchUsed bool   `json:"first_switch_used"`
	XPAwarded       bool   `json:"xp_awarded"`
	SessionXP       int64  `json:"session_xp"`
}

// TraceRow is one deterministic boundary observation.
type TraceRow struct {
	Tick       int    `json:"tick"`
	Phase      string `json:"phase"`
	WallHP     int64  `json:"wall_hp"`
	WallMax    int64  `json:"wall_max"`
	TownhallHP int64  `json:"townhall_hp"`
	Wood       int64  `json:"wood"`
	Stone      int64  `json:"stone"`
	Food       int64  `json:"food"`
	Work       int64  `json:"construction_work"`
	WallTier   int    `json:"wall_tier"`
	OrdinaryHP int64  `json:"ordinary_hp"`
	BossHP     int64  `json:"boss_hp"`
	Defense    int64  `json:"guard_block"`
	Attack     int64  `json:"attack"`
	RawDamage  int64  `json:"raw_damage"`
	Damage     int64  `json:"damage_applied"`
	Defeated   bool   `json:"defeated"`
	SessionXP  int64  `json:"session_xp_total"`
	BossLife   string `json:"boss_state"`
}

// Sim is the reference simulation of one village across one or more waves.
type Sim struct {
	RulesVersion  string
	SchemaVersion int

	VillageLevel      int
	WallTier          int
	WallHP            int64
	TownhallHP        int64
	Wood              int64
	Stone             int64
	Food              int64
	ConstructionWork  int64
	TotalWins         int64
	ConsecutiveWins   int
	ConsecutiveLosses int

	WaveNo    int64
	Tick      int
	Lifecycle string
	Defeated  bool
	Settled   bool
	Result    string // "", "victory", "loss"

	AnchorPopulation int64
	Difficulty       int
	BaseHP           int64
	Lanes            [3]Pool
	BossMaxHP        int64
	BossHP           int64
	BossSpawned      bool

	VoteChosen string
	Ballots    []Ballot

	Members   []*Member
	Queued    []*Member
	SessionXP map[string]int64

	// Pending holds explicit role/ballot events that become effective at a future
	// tick. They are applied when the clock reaches that tick, so a future input
	// can never mutate an already observed boundary.
	Pending []scheduledEvent
	// RoleLog records every role decision the model produced, in input order.
	RoleLog []RoleLogRow

	Trace   []TraceRow
	Log     []string
	Applied int

	lastBossState    string
	lastPreviousRole string
}

// NewSim returns a session in the documented initial state
// (02_GAME_RULES.md "Session/village and progression", 04_DATABASE.md).
func NewSim() *Sim {
	return &Sim{
		RulesVersion:     RulesVersion,
		SchemaVersion:    CanonicalSchemaVersion,
		VillageLevel:     1,
		WallTier:         1,
		WallHP:           BaseWallMax,
		TownhallHP:       TownhallMaxDmg,
		Wood:             400,
		Stone:            250,
		Food:             300,
		ConstructionWork: 0,
		WaveNo:           1,
		Tick:             -1,
		Lifecycle:        "waiting",
		SessionXP:        map[string]int64{},
	}
}

func (s *Sim) wallMax() int64 {
	return BaseWallMax + WallMaxStep*int64(s.WallTier-1)
}

func (s *Sim) ordinary() int64 {
	return s.Lanes[0].Total() + s.Lanes[1].Total() + s.Lanes[2].Total()
}

// Phase is derived only from the local wave tick (03_SIMULATION.md).
func (s *Sim) Phase() string {
	if s.Lifecycle == "waiting" || s.Tick < 0 {
		return "waiting"
	}
	switch t := s.Tick; {
	case t < 1600:
		return "prep"
	case t < 2600:
		return "assault"
	case t < 2800:
		return "finale"
	case t < 3000:
		return "results"
	default:
		return "rollover"
	}
}

func (s *Sim) findMember(actor string) *Member {
	for _, m := range s.Members {
		if m.Actor == actor {
			return m
		}
	}
	return nil
}

func (s *Sim) findQueued(actor string) *Member {
	for _, m := range s.Queued {
		if m.Actor == actor {
			return m
		}
	}
	return nil
}

func (s *Sim) totalXP() int64 {
	var sum int64
	for _, v := range s.SessionXP {
		sum += v
	}
	return sum
}

// JoinResult enumerates the documented registration outcomes.
type JoinResult string

const (
	JoinRegistered   JoinResult = "registered"
	JoinNextWave     JoinResult = "next_wave_queued"
	JoinRoleChanged  JoinResult = "role_changed"
	JoinNoopSameRole JoinResult = "noop_same_role"
	JoinCooldown     JoinResult = "cooldown_rejected"
	JoinDegraded     JoinResult = "degraded_rejected"
)

// Join applies one authorized role command at the given acceptance tick
// (02_GAME_RULES.md "Exact player action model", 03_SIMULATION.md).
func (s *Sim) Join(actor string, role Role, tick int) JoinResult {
	outcome := s.join(actor, role, tick)
	switch outcome {
	case JoinRegistered:
		s.RoleLog = append(s.RoleLog, RoleLogRow{Tick: tick, Actor: actor, Role: role.String(), Outcome: "registered"})
	case JoinRoleChanged:
		s.RoleLog = append(s.RoleLog, RoleLogRow{Tick: tick, Actor: actor, Role: role.String(), PreviousRole: s.lastPreviousRole, Outcome: "role_changed"})
	case JoinCooldown:
		s.RoleLog = append(s.RoleLog, RoleLogRow{Tick: tick, Actor: actor, Role: role.String(), Outcome: "cooldown_rejected"})
	}
	return outcome
}

func (s *Sim) join(actor string, role Role, tick int) JoinResult {
	if s.Lifecycle == "waiting" {
		// WAITING accepts the first authorized role and starts the wave at tick 0.
		s.Lifecycle = "running"
		s.Tick = 0
	}
	if s.Lifecycle == "stopped" {
		return JoinDegraded
	}
	if tick < 0 {
		tick = 0
	}
	// RESULTS registers the NEXT wave; the pending row is stored with
	// joined_tick=0 for wave_no+1 so it is inside the next anchor window.
	if tick >= FinalCombatTick || s.Phase() == "results" {
		if m := s.findQueued(actor); m != nil {
			return s.applyRoleChange(m, role, tick, true)
		}
		s.Queued = append(s.Queued, &Member{
			Actor:          actor,
			Role:           role,
			JoinedTick:     0,
			LastChangeTick: -1,
		})
		return JoinNextWave
	}
	if m := s.findMember(actor); m != nil {
		return s.applyRoleChange(m, role, tick, false)
	}
	s.Members = append(s.Members, &Member{
		Actor:          actor,
		Role:           role,
		JoinedTick:     tick,
		LastChangeTick: -1,
	})
	return JoinRegistered
}

func (s *Sim) applyRoleChange(m *Member, role Role, tick int, queued bool) JoinResult {
	if m.Role == role {
		return JoinNoopSameRole
	}
	if !m.FirstSwitch {
		s.lastPreviousRole = m.Role.String()
		m.Role = role
		m.FirstSwitch = true
		m.LastChangeTick = tick
		return JoinRoleChanged
	}
	if tick-m.LastChangeTick >= RoleChangeCooldownTicks {
		s.lastPreviousRole = m.Role.String()
		m.Role = role
		m.LastChangeTick = tick
		return JoinRoleChanged
	}
	return JoinCooldown
}

func sortMembers(list []*Member) {
	sort.Slice(list, func(i, j int) bool { return list[i].Actor < list[j].Actor })
}

// Vote records one ballot; returns false when it is not eligible.
func (s *Sim) Vote(actor, option string, tick int) bool {
	if s.WaveNo%2 != 0 {
		return false
	}
	if tick < VoteWindowStart || tick > VoteWindowEnd {
		return false
	}
	m := s.findMember(actor)
	if m == nil || m.JoinedTick > tick {
		return false
	}
	if option != "A" && option != "B" && option != "C" {
		return false
	}
	for i := range s.Ballots {
		if s.Ballots[i].Actor == actor {
			s.Ballots[i] = Ballot{Actor: actor, Option: option, Tick: tick}
			return true
		}
	}
	s.Ballots = append(s.Ballots, Ballot{Actor: actor, Option: option, Tick: tick})
	return true
}

func (s *Sim) voteScores() (int64, int64, int64) {
	var a, b, c int64
	seen := map[string]bool{}
	for _, bal := range s.Ballots {
		if seen[bal.Actor] {
			continue
		}
		m := s.findMember(bal.Actor)
		if m == nil || m.JoinedTick > VoteWindowEnd {
			continue
		}
		seen[bal.Actor] = true
		switch bal.Option {
		case "A":
			a++
		case "B":
			b++
		case "C":
			c++
		}
	}
	return a, b, c
}

func (s *Sim) affordable(option string) bool {
	switch option {
	case "A":
		return s.Wood >= VoteResourceCost
	case "B":
		return s.Food >= VoteResourceCost
	}
	return true
}

// closeBallot implements 02_GAME_RULES.md "Vote": one latest ballot per player,
// deterministic tie order A>B>C, resource fallback, and the zero-ballot default.
func (s *Sim) closeBallot() {
	if s.WaveNo%2 != 0 {
		return
	}
	a, b, c := s.voteScores()
	scores := map[string]int64{"A": a, "B": b, "C": c}
	total := a + b + c
	chosen := ""
	if total == 0 {
		if s.affordable("A") {
			chosen = "A"
		} else {
			chosen = "C"
		}
	} else {
		for _, opt := range []string{"A", "B", "C"} { // lexicographic order = tie break
			if !s.affordable(opt) {
				continue
			}
			if chosen == "" {
				chosen = opt
				continue
			}
			if scores[opt] > scores[chosen] {
				chosen = opt
			}
		}
		if chosen == "" {
			chosen = "C"
		}
	}
	s.VoteChosen = chosen
	switch chosen {
	case "A":
		s.Wood = clamp(s.Wood-VoteResourceCost, 0, CapResources)
	case "B":
		s.Food = clamp(s.Food-VoteResourceCost, 0, CapResources)
	}
}

func (s *Sim) lockPopulation() {
	var p int64
	for _, m := range s.Members {
		if m.JoinedTick <= PopulationLockTick-1 {
			p++
		}
	}
	if p < 1 {
		p = 1
	}
	s.AnchorPopulation = p
	s.Difficulty = 1 + s.ConsecutiveWins
	if s.Difficulty > 10 {
		s.Difficulty = 10
	}
}

func (s *Sim) baseHPValue() int64 {
	p := s.AnchorPopulation
	if p > CapPopulation {
		p = CapPopulation
	}
	raw := EnemyBase + EnemyPerDifficulty*int64(s.Difficulty) + EnemyScalePerIsqrt*ISqrt(p)
	lossMod := RecoveryPercent - RecoveryLossStep*int64(min64(int64(s.ConsecutiveLosses), MaxConsecutiveLosses))
	v := (raw * lossMod) / RecoveryPercent
	if v < 1 {
		v = 1
	}
	return v
}

func (s *Sim) seedPools() {
	s.BaseHP = s.baseHPValue()
	west := s.BaseHP * LaneWestShare / 100
	center := s.BaseHP * LaneCenterShare / 100
	east := s.BaseHP - west - center
	s.Lanes[0] = splitSubpools(west)
	s.Lanes[1] = splitSubpools(center)
	s.Lanes[2] = splitSubpools(east)
	// The boss pool is SIZED at the population lock but only CREATED at tick
	// 2600: boss_hp stays 0 in the canonical state until then.
	if s.WaveNo%5 == 0 {
		s.BossMaxHP = ceilDiv(s.BaseHP, BossHPOrder)
	} else {
		s.BossMaxHP = 0
	}
	s.BossHP = 0
	s.BossSpawned = false
}

func splitSubpools(laneHP int64) Pool {
	runner := laneHP * SubpoolRunnerShare / 100
	grunt := laneHP * SubpoolGruntShare / 100
	return Pool{Runner: runner, Grunt: grunt, Brute: laneHP - runner - grunt}
}

func (s *Sim) activeCounts(tick int) (int64, int64, int64) {
	var b, g, f int64
	for _, m := range s.Members {
		if m.JoinedTick > tick {
			continue
		}
		switch m.Role {
		case RoleBuilder:
			b++
		case RoleGuard:
			g++
		case RoleFighter:
			f++
		}
	}
	return min64(b, CapPopulation), min64(g, CapPopulation), min64(f, CapPopulation)
}

// awardXP grants +10 session XP once per wave to every active contributor
// (02_GAME_RULES.md "Exact player action model", cap 300 per session).
func (s *Sim) awardXP(tick int) {
	for _, m := range s.Members {
		if m.JoinedTick > tick || m.XPAwarded {
			continue
		}
		cur := s.SessionXP[m.Actor]
		s.SessionXP[m.Actor] = clamp(cur+XPPerWave, 0, CapSessionXP)
		m.XPAwarded = true
	}
}

func (s *Sim) militiaAttack() int64 {
	p := s.AnchorPopulation
	if p < 1 || p > MilitiaMaxPopulation {
		return 0
	}
	v := MilitiaMax - MilitiaStep*(p-1)
	if v < 0 {
		return 0
	}
	return v
}

// builderPhase implements generation, repair, construction work and at most one
// wall upgrade per boundary (02_GAME_RULES.md "Roles and economy").
func (s *Sim) builderPhase(b int64) {
	s.Wood = clamp(s.Wood+BuilderWoodPerIsqrt*ISqrt(b)+b/50, 0, CapResources)
	s.Stone = clamp(s.Stone+ISqrt(b)+b/500, 0, CapResources)

	capacity := AllianceRepairPerIsqrt * ISqrt(b)
	need := s.wallMax() - s.WallHP
	woodLimit := RepairWoodDivisor * s.Wood
	repair := clamp(capacity, 0, min64(need, woodLimit))
	if repair > 0 {
		cost := ceilDiv(repair, RepairWoodDivisor)
		s.Wood = clamp(s.Wood-cost, 0, CapResources)
		s.WallHP = min64(s.wallMax(), s.WallHP+repair)
	}
	leftover := capacity - repair
	if b > 0 && leftover > 0 {
		threshold := UpgradeWorkPerTier * int64(s.WallTier)
		s.ConstructionWork = min64(threshold, s.ConstructionWork+leftover)
	}
	threshold := UpgradeWorkPerTier * int64(s.WallTier)
	if s.WallTier < MaxWallTier && s.ConstructionWork >= threshold {
		stoneCost := UpgradeStonePerTier * int64(s.WallTier)
		if s.Stone >= stoneCost {
			s.ConstructionWork -= threshold
			s.Stone -= stoneCost
			s.WallTier++
			s.WallHP = min64(s.wallMax(), s.WallHP+WallMaxStep)
		}
	}
}

// applyAttack distributes attack damage with the documented deterministic lane
// rotation and cross-lane redistribution. Returns wasted damage.
func (s *Sim) applyAttack(amount, combatIndex int64) int64 {
	if amount <= 0 {
		return 0
	}
	start := int(combatIndex % 3)
	per := amount / 3
	rem := amount % 3
	leftover := int64(0)
	for i := 0; i < 3; i++ {
		lane := (start + i) % 3
		share := per
		if int64(i) < rem {
			share++
		}
		leftover += s.applyToLane(lane, share)
	}
	if leftover > 0 {
		for i := 0; i < 3 && leftover > 0; i++ {
			lane := (start + i) % 3
			leftover = s.applyToLane(lane, leftover)
		}
	}
	return leftover
}

// applyToLane consumes amt against one lane in documented priority
// runner, grunt, brute, boss (boss only in the centre lane).
func (s *Sim) applyToLane(lane int, amt int64) int64 {
	if amt <= 0 {
		return 0
	}
	p := &s.Lanes[lane]
	consume := func(hp *int64) {
		if amt <= 0 || *hp <= 0 {
			return
		}
		d := min64(*hp, amt)
		*hp -= d
		amt -= d
	}
	consume(&p.Runner)
	consume(&p.Grunt)
	consume(&p.Brute)
	if lane == 1 && s.BossSpawned && s.BossHP > 0 {
		d := min64(s.BossHP, amt)
		s.BossHP -= d
		amt -= d
	}
	return amt
}

// combatPhase runs step (1)..(6) of 02_GAME_RULES.md "Enemies / waves / boss".
func (s *Sim) combatPhase(tick int, b, g, f int64) {
	s.awardXP(tick)
	s.builderPhase(b)

	combatIndex := int64((tick - CombatStartTick) / 200)
	if tick == BossSpawnTick && s.WaveNo%5 == 0 {
		// The boss pool is created at 2600 and is attackable at the same boundary.
		s.BossHP = s.BossMaxHP
		s.BossSpawned = true
	}

	attack := FighterAttackPerIsqrt*ISqrt(f) + s.militiaAttack()
	if s.VoteChosen == "B" {
		attack = 115 * attack / 100
	}
	s.applyAttack(attack, combatIndex)

	ordinary := s.ordinary()
	p := s.AnchorPopulation
	if p > CapPopulation {
		p = CapPopulation
	}
	baseRaw := 10 + 3*int64(s.Difficulty) + 9*ISqrt(p)
	var raw int64
	if ordinary > 0 {
		raw = ceilDiv(baseRaw*ordinary, s.BaseHP)
	}
	if s.BossSpawned && s.BossHP > 0 {
		raw += ceilDiv(baseRaw, 5)
	}

	defense := GuardBlockPerIsqrt * ISqrt(g)
	if s.VoteChosen == "A" {
		defense = 115 * defense / 100
	}
	if s.VoteChosen == "C" {
		defense = 90 * defense / 100
	}
	damage := max64(0, raw-defense)

	wallLoss := min64(s.WallHP, damage)
	s.WallHP -= wallLoss
	s.TownhallHP = max64(0, s.TownhallHP-(damage-wallLoss))
	if s.TownhallHP == 0 {
		s.Defeated = true
	}

	bossState := "absent"
	if s.BossMaxHP > 0 {
		switch {
		case s.BossSpawned && s.BossHP > 0:
			bossState = "active"
		case s.BossSpawned:
			bossState = "destroyed"
		case ordinary == 0:
			bossState = "incoming"
		default:
			bossState = "scheduled"
		}
	}
	s.lastBossState = bossState

	s.Trace = append(s.Trace, TraceRow{
		Tick:       tick,
		Phase:      s.Phase(),
		WallHP:     s.WallHP,
		WallMax:    s.wallMax(),
		TownhallHP: s.TownhallHP,
		Wood:       s.Wood,
		Stone:      s.Stone,
		Food:       s.Food,
		Work:       s.ConstructionWork,
		WallTier:   s.WallTier,
		OrdinaryHP: ordinary,
		BossHP:     s.BossHP,
		Defense:    defense,
		Attack:     attack,
		RawDamage:  raw,
		Damage:     damage,
		Defeated:   s.Defeated,
		SessionXP:  s.totalXP(),
		BossLife:   bossState,
	})
}

// settle executes the one-time RESULTS settlement at tick 2800.
func (s *Sim) settle() {
	if s.Settled {
		return
	}
	s.Settled = true
	cleared := s.ordinary() == 0 && s.BossHP == 0
	victory := s.TownhallHP > 0 && cleared
	if victory {
		s.Result = "victory"
		s.Wood = clamp(s.Wood+VictoryRewardWood, 0, CapResources)
		s.Stone = clamp(s.Stone+VictoryRewardStone, 0, CapResources)
		s.Food = clamp(s.Food+VictoryRewardFood, 0, CapResources)
		s.ConsecutiveWins = int(min64(int64(s.ConsecutiveWins+1), MaxConsecutiveWins))
		s.ConsecutiveLosses = 0
		s.TotalWins++
		s.VillageLevel = int(min64(MaxVillageLevel, 1+s.TotalWins/3))
		return
	}
	s.Result = "loss"
	s.ConsecutiveWins = 0
	if s.ConsecutiveLosses < MaxConsecutiveLosses {
		s.ConsecutiveLosses++
	}
	woodPenalty := s.Wood / 5
	stonePenalty := s.Stone / 5
	foodPenalty := s.Food / 5
	if s.VoteChosen == "C" {
		woodPenalty /= 2
		stonePenalty /= 2
		foodPenalty /= 2
	}
	s.Wood = clamp(s.Wood-min64(woodPenalty, s.Wood), 0, CapResources)
	s.Stone = clamp(s.Stone-stonePenalty, 0, CapResources)
	s.Food = clamp(s.Food-foodPenalty, 0, CapResources)
	if s.Defeated {
		s.WallHP = min64(s.wallMax(), max64(DefeatFloorHP, s.WallHP))
		s.TownhallHP = clamp(max64(DefeatFloorHP, s.TownhallHP), 0, TownhallMaxDmg)
		s.Wood = clamp(max64(DefeatFloorWood, s.Wood), 0, CapResources)
	}
}

// rollover performs the single control transition at tick 3000.
func (s *Sim) rollover() {
	s.WaveNo++
	s.Tick = 0
	s.Defeated = false
	s.Settled = false
	s.Result = ""
	s.VoteChosen = ""
	s.Ballots = nil
	s.Lanes = [3]Pool{}
	s.BossMaxHP = 0
	s.BossHP = 0
	s.BossSpawned = false
	s.BaseHP = 0
	s.AnchorPopulation = 0
	s.Difficulty = 0
	s.Members = nil
	s.WallHP = min64(s.WallHP, s.wallMax())
	if len(s.Queued) > 0 {
		for _, m := range s.Queued {
			m.JoinedTick = 0
			m.LastChangeTick = -1
		}
		s.Members = append([]*Member{}, s.Queued...)
		s.Queued = nil
		sortMembers(s.Members)
		s.Lifecycle = "running"
		return
	}
	s.Tick = -1
	s.Lifecycle = "waiting"
}

// applyDue applies every explicit input whose effective tick has been reached, in
// (tick, declaration order). Role events are declared before ballot events, which
// matches the production order: due role commands, then due ballots, then the
// boundary resolution itself.
func (s *Sim) applyDue() {
	for len(s.Pending) > 0 && s.Pending[0].Tick <= s.Tick {
		event := s.Pending[0]
		s.Pending = s.Pending[1:]
		switch event.Kind {
		case ScheduleJoin, ScheduleRoleChange:
			s.Join(event.Actor, event.Role, event.Tick)
		case ScheduleBallot:
			s.Vote(event.Actor, event.Option, event.Tick)
		}
	}
}

// ProcessTick applies the documented total order for the current tick.
func (s *Sim) ProcessTick() {
	if s.Lifecycle != "running" {
		return
	}
	s.applyDue()
	t := s.Tick
	if t == VoteCloseTick {
		s.closeBallot()
	}
	if t == PopulationLockTick {
		s.lockPopulation()
		s.seedPools()
	}
	if t > 0 && t%200 == 0 && t <= FinalCombatTick && !s.Defeated {
		b, g, f := s.activeCounts(t)
		if t < CombatStartTick {
			// Builder-only boundary: XP + economy, no combat.
			s.awardXP(t)
			s.builderPhase(b)
			s.Trace = append(s.Trace, TraceRow{
				Tick:       t,
				Phase:      s.Phase(),
				WallHP:     s.WallHP,
				WallMax:    s.wallMax(),
				TownhallHP: s.TownhallHP,
				Wood:       s.Wood,
				Stone:      s.Stone,
				Food:       s.Food,
				Work:       s.ConstructionWork,
				WallTier:   s.WallTier,
				OrdinaryHP: s.ordinary(),
				BossHP:     s.BossHP,
				SessionXP:  s.totalXP(),
				BossLife:   "absent",
			})
		} else {
			s.combatPhase(t, b, g, f)
		}
	}
	if t == FinalCombatTick {
		s.settle()
	}
}

// memberSnapshots returns the authoritative participant state sorted by actor.
func (s *Sim) memberSnapshots() []MemberSnapshot {
	out := make([]MemberSnapshot, 0, len(s.Members))
	for _, member := range s.Members {
		var lastChange *int
		if member.FirstSwitch {
			tick := member.LastChangeTick
			lastChange = &tick
		}
		out = append(out, MemberSnapshot{
			Actor:           member.Actor,
			Role:            member.Role.String(),
			JoinedTick:      member.JoinedTick,
			LastChangeTick:  lastChange,
			FirstSwitchUsed: member.FirstSwitch,
			XPAwarded:       member.XPAwarded,
			SessionXP:       s.SessionXP[member.Actor],
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Actor < out[j].Actor })
	return out
}

// VoteTally is the applied-ballot tally of one even wave.
type VoteTally struct {
	A int64 `json:"a"`
	B int64 `json:"b"`
	C int64 `json:"c"`
}

// StateDigest is a stable SHA-256 over the authoritative pure state fields,
// modelled on 04_DATABASE.md "Canonical RFC8785 hash" (field list is the
// normative contract; this is a reference digest, not the production encoder).
func (s *Sim) StateDigest() string {
	var b strings.Builder
	fmt.Fprintf(&b, "schema=%d;rules=%s;wave=%d;tick=%d;life=%s;def=%t;",
		s.SchemaVersion, s.RulesVersion, s.WaveNo, s.Tick, s.Lifecycle, s.Defeated)
	fmt.Fprintf(&b, "tier=%d;wall=%d;hall=%d;wood=%d;stone=%d;food=%d;work=%d;",
		s.WallTier, s.WallHP, s.TownhallHP, s.Wood, s.Stone, s.Food, s.ConstructionWork)
	fmt.Fprintf(&b, "tw=%d;cw=%d;cl=%d;P=%d;d=%d;base=%d;vote=%s;boss=%d;spawned=%t;",
		s.TotalWins, s.ConsecutiveWins, s.ConsecutiveLosses, s.AnchorPopulation,
		s.Difficulty, s.BaseHP, s.VoteChosen, s.BossHP, s.BossSpawned)
	for i, p := range s.Lanes {
		fmt.Fprintf(&b, "lane%d=%d,%d,%d;", i, p.Runner, p.Grunt, p.Brute)
	}
	ms := append([]*Member{}, s.Members...)
	sortMembers(ms)
	for _, m := range ms {
		fmt.Fprintf(&b, "m=%s,%s,%d,%d,%t,%t,%d;", m.Actor, m.Role, m.JoinedTick,
			m.LastChangeTick, m.FirstSwitch, m.XPAwarded, s.SessionXP[m.Actor])
	}
	qs := append([]*Member{}, s.Queued...)
	sortMembers(qs)
	for _, m := range qs {
		fmt.Fprintf(&b, "q=%s,%s,%d;", m.Actor, m.Role, m.JoinedTick)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
