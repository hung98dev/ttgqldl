# DETERMINISTIC SIMULATION — BINDING

## Integer/time model
10 fixed logical ticks per second, ticks int64 and no floating arithmetic or OS wall clock in game reducer. Global wave tick local range 0..2999. Every tick scheduled in strict ascending sequence. Phase is derived only from local tick: [0,1600)=PREP, [1600,2600)=ASSAULT, [2600,2800)=FINALE, [2800,3000)=RESULTS. A combat resolution at boundary tick 2800 executes BEFORE the phase becomes RESULTS and result is then evaluated on the same tick; do not process economy twice. Tick 3000 closes RESULTS and moves to WAITING for next wave. On restart use committed tick and accepted input log, not elapsed wall time.

## Total order of events at every tick boundary
1. WAITING accepts the first authorized role and starts a wave at tick0; PAUSED/SOURCE_DISCONNECTED/STORAGE_UNAVAILABLE freeze ticks. STOPPED rejects new game actions; operator/source controls remain usable.
2. Consume durable input_events ordered by input_seq whose scheduled_tick<=current boundary and not previously applied. Ingest schedules at last committed wave_tick+1, so no input arriving during a tick can affect that already committing tick. During RESULTS assign valid join to NEXT wave tick0.
3. If t=1200, close vote and choose option.
4. If t=1600, lock P,d,base_hp and spawn ordinary pools.
5. If t is positive multiple 200 <=2800, with t2800 resolved BEFORE entering RESULTS: aggregate roles and perform builder economy+construction once; if t>=1800, apply fighter phase (boss spawn if t=2600) and enemy response.
6. If t=2800, compute final wave outcome once and write result.
7. Emit state hash, outbox entries, committed sequence and tick in ONE transaction. Publish committed state only.
8. At tick3000, advance wave and promote queued NEXT-wave actors; start tick0 when queued members exist; otherwise WAITING; never require a queued actor to comment twice.

## Pseudocode — reducer owns no I/O
    Next(state, ordered_events, tick):
       assert tick == state.last_tick + 1
       apply_exact_valid_commands(ordered_events)
       if tick == 1200: close_ballot()
       if tick == 1600: seed_enemy_pools_and_lock_population()
       if tick % 200 == 0 and 0 < tick <= 2800:
          apply_builder_generation_repair_upgrade()
          if tick >= 1800:
             if tick == 2600 and boss_wave: add_boss_pool()
             attack = 14*isqrt(min(F,10000)) + militia_bonus(P)
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
Balance simulation itself requires NO RNG: attacks and damage use fixed integer math, seed only for cosmetic sampling. Cosmetic RNG must be Xoshiro or PCG with specified algorithm/parameters in a future **non-gameplay** client fixture; simulation outcome MUST never depend on it. Tie votes deterministic A>B>C; no unspecified seeded randomness or iteration over Go maps. All aggregate iteration sorted by ActorKey when emitting acknowledgments; lane indices west=0,center=1,east=2 and archetype indices runner=0,grunt=1,brute=2,boss=3.

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
Define state hash as SHA256 of RFC 8785 JSON Canonicalization Scheme encoding of a versioned **pure game state**; omit transport timestamps, random visual positions and network counters. Integer-only numbers safe within JSON precision 2^53-1; encode any larger sequence/tick as decimal strings. Include sorted participant roles by actor_key, sorted enemy pools by (lane,archetype), all resources and vote, tick and relevant metadata; exact field list in 04_DATABASE.md. Every commit affecting canonical serialization requires regenerated and independently reviewed golden fixtures, and master version bump if incompatible.
