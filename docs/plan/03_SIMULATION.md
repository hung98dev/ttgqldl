# DETERMINISTIC SIMULATION — BINDING

**Time/order contract for `ttg-rules-v1`, canonical `schema_version`=1.** Any change to tick ordering, rounding, hashing fields or boundary handling requires a rules_version bump and same-commit update of 02_GAME_RULES.md, 04_DATABASE.md, 09_QA_ACCEPTANCE.md and the regenerated `tooling/refeval` golden vectors.

## Integer/time model
10 fixed logical ticks per second; signed wave_tick ranges [-1,3000], -1 means WAITING, cumulative counters checked int64; no floating-point or wall clock in reducer. Global wave tick local range 0..2999. Every tick scheduled in strict ascending sequence. Phase is derived only from local tick: [0,1600)=PREP, [1600,2600)=ASSAULT, [2600,2800)=FINALE, [2800,3000)=RESULTS. A combat resolution at boundary tick 2800 executes BEFORE the phase becomes RESULTS and result is then evaluated on the same tick; do not process economy twice. Tick3000 closes RESULTS: queued roles start next wave at tick0 automatically, otherwise WAITING at wave_tick=-1. On restart use committed tick and accepted input log, not elapsed wall time.

## Total order of events at every tick boundary
1. WAITING accepts the first authorized role and starts a wave at tick0; PAUSED/SOURCE_DISCONNECTED/STORAGE_UNAVAILABLE freeze ticks. STOPPED rejects new game actions; operator/source controls remain usable.
2. Consume durable input_events ordered by input_seq whose scheduled_tick<=current boundary and not previously applied. Ingest schedules at last committed wave_tick+1, so no input arriving during a tick can affect that already committing tick. During RESULTS assign valid join to NEXT wave tick0.
3. If t=1200 and the wave is EVEN, close the ballot and choose the option; odd waves have no vote.
4. If t=1600, lock P,d,base_hp and spawn ordinary pools. The boss pool is NOT created here; only its derivable maximum ceil(base_hp/5) is known.
5. If t is a positive multiple of 200 and t<=2800, with t2800 resolved BEFORE entering RESULTS, and only if not already DEFEATED: aggregate roles for members with joined_tick<=t; award one-time +10 session XP (cap 300) to each active member with xp_awarded=false and mark it awarded; then perform builder generation, repair and construction/upgrade once; if t>=1800, then create the boss pool at t=2600, apply fighter+militia attack, apply enemy retaliation and guard mitigation, and apply the resulting damage.
6. If t=2800, compute the final wave outcome once and write the result.
7. Emit state hash, outbox entries, committed sequence and tick in ONE transaction. Publish committed state only.
8. At tick3000 promote NEXT-wave actors; each promoted membership keeps role but is persisted with joined_tick=0 so it is inside the next anchor window. Start the next wave at tick0 if any exist, else WAITING at wave_tick=-1; no second comment.

## Pseudocode — reducer owns no I/O
    Next(state, ordered_events, tick):
       assert tick == state.last_tick + 1
       apply_exact_valid_commands(ordered_events)
       if tick == 1200 and state.wave_no % 2 == 0: close_ballot()
       if tick == 1600: seed_enemy_pools_and_lock_population()
       if tick % 200 == 0 and 0 < tick <= 2800 and not state.defeated:
          aggregate_roles(joined_tick <= tick)
          award_one_time_xp_and_mark_awarded()
          apply_builder_generation_repair_upgrade()
          if tick >= 1800:
             if tick == 2600 and boss_wave: create_boss_pool(boss_max_hp)
             attack = 18*isqrt(min(F,10000)) + (max(0,40-4*(P-1)) if 1<=P<=10 else 0)
             if vote == B: attack = floor(115*attack/100)
             apply_attack_to_subpools_deterministically(attack)
             ordinary = sum(ordinary_subpool_hp)
             raw = ceil(base_raw_damage * ordinary/base_hp) if ordinary>0 else 0
             if boss_hp>0: raw += ceil(base_raw_damage/5)
             defense = 8*isqrt(min(G,10000))
             if vote == A: defense = floor(115*defense/100)
             if vote == C: defense = floor(90*defense/100)
             damage = max(0,raw-defense)
             wall_loss=min(state.wall_hp,damage)
             state.wall_hp -= wall_loss
             state.townhall_hp = max(0,state.townhall_hp-(damage-wall_loss))
       if tick == 2800: settle_wave_once()
       return state, canonical_events

## Seed and randomness
Balance simulation itself requires NO RNG: attacks and damage use fixed integer math, seed only for cosmetic sampling. No cosmetic PRNG implementation is mandated in Go. For deterministic purely visual choices, Unity MAY choose index = uint64_le(first8(SHA256(session_id + ':' + wave_no + ':' + visual_event_seq))) % option_count when option_count>0; the cosmetic hash and selection are never written into authoritative game-state hash. The gameplay outcome MUST never depend on cosmetic randomness. Tie votes deterministic A>B>C; no unspecified seeded randomness or iteration over Go maps. All aggregate iteration sorted by ActorKey when emitting acknowledgments; lane indices west=0,center=1,east=2 and archetype indices runner=0,grunt=1,brute=2,boss=3.

## Temporal boundary cases
- A role whose scheduled_tick<=1599 is included in anchor P at tick1600; input arriving as tick1600 commits schedules tick1601 and is NOT in P.
- A role with scheduled_tick<=2000 enters tick2000 resolution, unless that tick has already committed; accepted events cannot retroactively mutate committed tick.
- t=1200 vote decision committed before contribution; a ballot timestamp exactly boundary is late.
- t=2600 ordinary plus boss attack/retaliation; boss must be spawn-visible before next stream publication.
- t=2800 resolves combat first, result second, then RESULTS begins, no game combat at t=3000.
- Role actions during RESULTS including t2999 join NEXT wave; pending participants auto-start next wave at rollover, with no XP in the finished wave.
- Tick is int64; cap duration/sequence; schema validates overflow.
- If zero participants and WAITING, no simulation tick advances or wave loss occurs.
- Out-of-order upstream source event timestamps do not rewrite previously committed game state. Server accept sequence/time are authoritative. Capture both when allowed for diagnostics.
- If source disconnects, freeze wave time by default. Queued previously committed inputs are replayed; no fake viewer events.

## Event sourcing and idempotency invariants
A valid command event stores accept_tick, sequence, event_id, actor_key, wave number, canonical payload. Same unique provider event ID is idempotent. For missing stable provider event IDs, dedupe best-effort by source session+actor+normalized command+short interval; do NOT assert exactly-once delivery across crashes; one-player role semantics still prevent multiple contribution. Duplicate role command never gives extra XP. Snapshot includes all role membership, accepted window/cooldown changes, pending next-wave actions, vote, pools, resource counters, wave_tick, game rules version and output sequence.
A tick transaction owns: accepted commands for tick, state change, emitted domain events and last_processed_sequence; commit before WebSocket publish. If DB unavailable, simulation MUST pause; never send uncommitted successes to clients. Recovery replays accepted inputs in deterministic order, exactly once by sequence, verifies SHA256 canonical state.

## Canonical hashing
Define state hash as SHA256 of RFC 8785 JSON Canonicalization Scheme encoding of a versioned **pure game state** (canonical `schema_version`=1, see the exact field list in 04_DATABASE.md); omit transport timestamps, random visual positions and network counters. `boss_hp` is 0 until the boss pool is created at tick2600; the derivable maximum ceil(base_hp/5) is never hashed. Integer-only numbers safe within JSON precision 2^53-1; encode any larger sequence/tick as decimal strings. Include sorted participant roles by actor_key, sorted enemy pools by (lane,archetype), all resources and vote, tick and relevant metadata; exact field list in 04_DATABASE.md. Every commit affecting canonical serialization requires regenerated and independently reviewed golden fixtures, and master version bump if incompatible.

## Mandatory deterministic implementation clarifications
- Session creation: wave_no=1,wave_tick=-1,lifecycle=waiting. First role input targets wave1 tick0; reducer calls StartWave(-1 -> 0) and later NextTick(t -> t+1). Rollover tick3000 is ONE control transition, not an extra combat tick. After rollover wave_tick=0 with queued roles or -1 WAITING.
- Ingest schedules current-wave event to committed_tick+1; a command received while boundary tick is committing cannot change that boundary. Votes eligible by scheduled_tick in [800,1199] only; scheduled_tick=1200 is late. Commands in RESULTS queue next wave at tick0.
- Anchor P at tick1600 includes registrations with scheduled_tick<=1599 only. Boss spawns at tick2600 BEFORE fighter attack. At tick2800 final combat executes then results settle, and DEFEATED skips further economy and damage.
- Combat_index=0 at t1800. Attack A split equally among lanes, A%3 remainder distributed cyclically starting combat_index%3. Within each lane target runner,grunt,brute,center boss. Unused lane damage gets redistributed cyclically to other living pools by min(poolHP,remaining), with zero wasted while HP remains. No combat logic relies on cosmetic RNG.
- Before boss spawn on boss waves, ordinary pools empty => BOSS_INCOMING, never CLEAR. A wave result must be emitted only at tick2800.
- Pure Reduce(state,orderedCommands,tick) MUST NOT do database/network/time/OS I/O. Hashing follows canonical schema in 04_DATABASE.md, no map-iteration-dependent order. The pause reasons for manual operator pause, provider disconnect and database unavailable coexist; resume only when all clear without wallclock catch-up.
- Canonical domain event types emitted per commit are exactly: `wave_started`, `role_accepted`, `role_changed`, `role_rejected_cooldown`, `next_wave_queued`, `vote_opened`, `vote_closed`, `phase_changed`, `boss_incoming`, `boss_spawned`, `attack_resolved`, `damage_applied`, `wall_upgraded`, `wave_result`, `wave_rollover`, `degraded_event_dropped`, `source_status_changed`, `privacy_erased`. Each carries tick, wave_no and a payload restricted to public-safe or operator-safe fields per 05_API_WEBSOCKET.md; no raw provider identifier or actor_key leaves the backend.
