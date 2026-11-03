# GAMEPLAY CONTRACT v1
MUST implement exactly unless an ADR changes this document + tests.

## World and wave rules
One persistent Village per broadcaster. Finite 1..50 village level, 1..20 wall tier. Resource counters are bounded, unsigned integer quantities internally (explicit max 1,000,000 per resource), persisted with CHECK >=0. Village starts wall_hp=1000, townhall_hp=500, wood=400, stone=250, food=300. Wall max HP = 1000 + 100*(tier-1). Defeat: retain progression and reset wall to 250 HP after results, town hall to 250, deduct at most floor(0.2*unspent resources) and grant recovery 100 wood. Never softlock.
States IDLE->PREPARATION(180s)->ASSAULT(90s)->FINALE(20s)->RESULTS(10s)->PREPARATION. Pause freezes sim time; restart resumes stored sim tick. Boss wave multiples of 5. Assault enemies start entering lanes exactly at assault start.

## Command contract
Normalize user message: Unicode NFKC; trim; lower-case; fold Vietnamese diacritics including đ->d; collapse whitespace. Only exact input XAY, THU, DANH, BUILD, DEFEND, ATTACK accepted for roles; do not match arbitrary chat fragments. A/B/C exact only during active votes. Join implicitly on first valid role. No public moderator commands.
ActorKey = HMAC-SHA256(secret, provider+":"+broadcaster+":"+stableProviderId), if stable ID exists and is authorized; otherwise session-scoped generated anonymous identifier with explicitly limited dedup guarantees. Never use display name as permanent key.
Window index = floor(server_simulation_second / 20). For each (session,actor,window) store only one final role, last valid role wins. At boundary apply ONE contribution independent of number of submitted comments. If viewer doesn't submit new command next window, previous role remains displayed but contributes zero until they send a new valid action. 10 XP per contributing window, capped 100 XP per 300-second wave and 300 XP per session. Latest valid vote per actor per vote ID wins and counts once.
Roles XAY=builders, THU=defenders, DANH=fighters. Aggregation at 20s boundaries, carried-over totals NEVER counted again.

## Deterministic integer balance
No floating point in authoritative simulation. Define isqrt(n) = integer floor square-root. In every window, aggregate N builders B, defenders G, fighters F; calculate:
repair_budget = floor(3*isqrt(B)*(10+tier)/10).
defense_points = 2*isqrt(G) + floor(G/100).
attack_points = 4*isqrt(F) + floor(F/75).
Wood cost repair = ceil(actual_repair/5); repair limited to affordable wood and missing HP. If wall is full, builders convert up to 25% of budget into construction work; construction points require 1 stone per 5 points rounded up, wall tier upgrade costs 100*tier points and 100*tier stone, consumes once at completion. Exactly one resource transaction per resolution boundary.
Enemy composition per wave n: enemy_count=25+7*n, min 20, cap 500; seeded weighted deterministic 60% grunts/25% runners/15% brutes rounded via integer remainder, split west/center/east seeded. Base grunt HP10 damage2, runner HP6 damage1, brute HP30 damage7, boss HP = 400+60*n, damage20. Incoming damage occurs every 20s during assault and finale, never during preparation; 1 point of defense blocks 1 point of incoming damage. Attack points convert to 1 HP of enemy damage each. Unit targeting: west->center->east; within lane runner->grunt->brute->boss; ties by stable ID. Boss appears at finale entry for waves divisible by 5; otherwise elite brute with 3x standard brute HP.
Resolve order: (1) builder repair/construction (2) fighter damage to enemies (3) enemy retaliation from survivors (4) guard damage block (5) wall HP reduction, overflow into town hall HP. Wall/town hall cannot become negative; on town hall zero mark wave lost. On zero surviving enemies during assault/FINALE, keep clock running until results but show early clear state. If loss, do not keep subtracting damage.
Enemy damage contribution for each surviving grunt/runner/brute/boss per resolution = archetype damage (not per-hit frame). Any defeated enemy gives no retaliation. Enemy spawns at assault start; no additional spawns within wave. No hidden difficulty manipulation.
A wave wins if town hall HP>0 and all enemies eliminated by start of RESULTS; otherwise loss, with defeat recovery. Reward on victory: wood min(cap,100+10*n), stone min(cap,60+5*n), food min(cap,50+5*n). Award points by actual effective participation, not raw comments.

## Voting
Every even wave: one decision opens at t=100s and closes t=140s, eligible valid participant identities can send A/B/C. A: wall fortification, consume 50 wood for +15% defense this wave; B: ambush, 50 food for +15% attack; C: evacuation, this wave loss stock penalty halved but defense -10%. Use integer rational calculations. Option invalid if insufficient resources; display disabled state. Tie: deterministic seeded PRNG stored in event log. No vote: choose A if affordable, else C. One vote per user, latest ballot effective.

## Presentation, recognition
Visual names sampled via fair deterministic rotation; opt-out/safety filtering applied. No promise every viewer's avatar appears. Group bars update at publication. Wave result panel 10s, awards repairs, defense saves, battle output. No pay-to-win and no rewards for excessive comments.

## Acceptance invariants
Duplicated 100 messages in one window => 1 contribution. Role update replaces. Never negative wall, resources, XP. Fixed seed+accepted event sequence+balance version => byte-identical canonical JSON state hash. No fake viewer activity when LIVE disconnects. Any supported commands remain playable during all wave phases. Load balancing uses aggregates, not N per-viewer combat objects.
