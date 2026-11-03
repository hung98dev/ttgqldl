# TTGQLDL — MASTER IMPLEMENTATION PLAN
**Only authoritative specification** · Updated 2026-11-03 · Branch main · Planning status (not implemented/tested).

## 0. GOALS, SCOPE, GOVERNANCE AND FIXED TECHNOLOGY
One document contains all mandatory design, gameplay, equations, persistence, API, Unity, TikTok integration, privacy, testing and execution requirements. No other document is a competing source. AGENTS.md is only a short instruction to read this file. Generated implementation files (code, OpenAPI, SQL, golden test fixtures and evidence) are required deliverables, not secondary plans. No separate docs/v2, forks, feature branches or PRs unless explicitly requested.

P0 = full offline mock/replay game with Go authoritative engine, PostgreSQL, Unity broadcast app, operator UI, verified original/right-cleared assets, test/load/recovery and CI; explicitly label simulation MOCK/REPLAY. P1 = separately gated authorized TikTok LIVE production integration with real credentials, permissions, broadcaster rights, preview, human review, moderation and real end-to-end rehearsal. AI must never claim external permission, legal compliance or real testing from an unverified document.

Locked stable baseline: Go 1.27.3, PostgreSQL 18.6, Unity Editor 6000.6.7f1. Windows 11 desktop Unity/OBS, Linux amd64 backend; Go net/http + pgx/v5 + sqlc + goose; React/TypeScript/Vite operator; Docker Compose; PostgreSQL is sole durable source of truth. Do not add Redis, Kafka, Kubernetes, microservices or an external paid service without measured evidence and explicit approved change. Pin exact third-party versions, Unity package versions, lockfiles, Docker image digests at first executable implementation. Verify vendor stable releases and license:
https://go.dev/dl/ ; https://www.postgresql.org/support/versioning/ ; https://unity.com/releases/editor/archive .

Product: 9:16 cooperative chibi village survival. Three input commands XAY/THU/DANH (and exact English aliases). One valid join per wave grants automatic 20s-boundary role contribution throughout the wave; repetitive comments do not add power or XP. 300s fixed wave, 10Hz authoritative tick, three lanes, boss every fifth wave. Logical population 10k; <=150 visual entities. Do not auto simulate empty-room losses. No pay-to-win, gambling, spam incentives, gifts influencing combat or falsified viewer counts.

Hard requirements say MUST/SHALL. If contradictory: block the affected requirement, repair **this file** and corresponding test fixtures with an explicit decision; independent tasks may continue. No unstated default or fabricated PASS. Work solely on main; run pre-commit tests, verify CI post-commit and revert/repair regressions. Evidence must contain code SHA, exact commands, OS/hardware, test inputs, expected/actual output, CI artifact, review provenance.

**Sections:** 1 Game Rules; 2 Tick Algorithm; 3 PostgreSQL/Recovery; 4 API/WebSocket; 5 LIVE/Security; 6 Unity/Art; 7 Operations; 8 Acceptance; 9 Ordered Milestones; 10 External Blockers; 11 Audit Findings.

---


---

# 1. GAME RULES — BINDING

## Session/village and progression
One broadcaster owns one village (broadcaster key stable, privacy-protected). Session has one active wave at a time. A village keeps level 1..50, wall tier 1..20, XP cosmetics, resources; starts level=1, wall tier=1, wall_hp=1000, townhall_hp=500, wood=400, stone=250, food=300, construction_work=0; resources capped at 1,000,000 each. Initial wave_number=1, consecutive_wins=0. All backend arithmetic signed int64 checked for overflow, never floats.
One wave is 300s at 10Hz (ticks 0..2999). Start only on first eligible role message after operator START; status WAITING before that. PHASE PREP [0,160s); ASSAULT [160,260s); FINALE [260,280s); RESULTS [280,300s). Boss on wave_number divisible by 5. Engine only advances simulation time while RUNNING; PAUSED, PROVIDER_DISCONNECTED, STORAGE_UNAVAILABLE freeze time. At RESULTS completion next wave returns WAITING until a new eligible role, preventing unattended loss loops.

## Exact player action model
Text exact role commands XAY/THU/DANH and BUILD/DEFEND/ATTACK; lower-case, trim, Unicode NFKC, fold Vietnamese diacritics incl. đ->d, collapse ASCII/Unicode spaces; only one canonical token accepted; punctuation and substrings are invalid. One role selection registers player for CURRENT wave; ONE role contribution at each 20s resolution boundary until RESULTS, even if user sends no more messages. Thus viewers never need to repeatedly comment. Last VALID role accepted before a boundary takes effect on the next boundary; at most one effective role-change per 60 seconds of server simulation time for the same identity (initial join excluded); additional changes are ignored/acknowledged as cooldown, NEVER modify previous role. Repeated identical messages are no-op. A player joining RESULTS is assigned to NEXT wave and receives immediate visible 'NEXT WAVE' receipt, without counting as present now; when next wave starts, registration transfers atomically and persists for that wave. A participant joining a paused wave receives queued role only if source is connected, applied on resume without advancing simulation clock. Player roles expire at wave rollover; previous wave XP/identity persist.
A player earns +10 XP once per wave with at least one eligible contributing interval; no XP per chat, repeated comment, vote or gift. XP cap per player per SESSION = 300 and level is cosmetics only. A newly joining viewer contributes equally in the same role as veterans. Display actual accepted count and source health, not fake viewers. IDs are scoped HMAC identities from authorized stable provider ID, otherwise session-only ephemeral with honest limitations.

## Roles and economy
At every tick multiple of 200 (20-second boundary), except results phase and tick 0, count active registered participants B=builders, G=guards, F=fighters. Population anchor P = max(1, unique registrations in the wave at tick 1600), capped at 10,000 for difficulty coefficients, while all valid viewers beyond 10k still counted in contributions with sqrt saturation cap=10,000 per role. New viewers after tick1600 increase effort but cannot increase that wave's enemy HP, encouraging newcomers.
Builder generation each interval: wood += 2*isqrt(B)+floor(B/50); stone += isqrt(B)+floor(B/500), cap overflow at 1,000,000. Repair capacity = 8*isqrt(B). First generate wood+stone, then repair min(repair_capacity, wall_max-wall_hp, 5*wood); wood cost ceil(actual_repair/5), rounded integer. When wall has no missing HP, leftover capacity becomes construction work. Otherwise leftover capacity also becomes construction work after repair. Construction work accumulates only if B>0, cap progress at threshold until upgrade. Upgrade automatically if construction_work >= 200*wall_tier AND stone >= 100*wall_tier AND tier<20: subtract both costs, increase tier by 1, new wall_max=1000+200*(tier-1), add 200 to wall_hp (cap to new max); at most ONE tier upgrade per boundary; leave excess construction_work for next boundary. When at tier20 work is ignored.
Guard block = 8*isqrt(G) each COMBAT boundary, cannot be banked. Fighter attack = 14*isqrt(F), cannot be banked. If 1<=P<=3, scripted NPC militia attack bonus 10*(4-P) per COMBAT boundary. No NPC appears in statistics as a user or earns XP. All isqrt integer floor.

## Enemies / waves / boss
At tick 1600 snapshot P, difficulty d=min(10,1+consecutive_wins); initial ordinary HP base_hp = max(1, floor((40+8*d+38*isqrt(P))*(100-15*min(consecutive_losses,3))/100)), difficulty reduction is transparent 'recovery modifier' shown on HUD. Enemy pools: west=floor(base_hp*34/100), center=floor(base_hp*32/100), east=base_hp-west-center. Each lane subpools runner=25%, grunt=55%, brute=remainder (integer floors); each subpool has only HP counter and visual archetype, not 10,000 enemy entities. Normal units spawn at tick 1600; deterministic visual counts are for presentation, not hit points.
Boss wave: at tick 2600, spawn boss pool with HP ceil(base_hp/5); non-boss wave: no new enemy, but FINALE warning/visual climax. Boss is in center lane and is attacked after all ordinary subpools in that lane. One initial fixed enemyHP pool for each wave; no hidden rubber-banding after tick1600.
At COMBAT boundaries t=180,200,220,240,260,280s (tick 1800..2800), in this exact order: (1) role/economy contribution and repair; (2) boss spawn if tick 2600 and boss wave; (3) apply fighter+militia attack, vote B modifier 115/100 if active, integer floor; (4) calculate enemy retaliation from surviving ordinary HP and boss (5) apply guard block with vote A 115/100 or vote C 90/100, floors; (6) subtract damage from wall, overflow into townhall, stopping at zero; (7) emit canonical state and visual cues. For determinism, distribute damage to lanes in rotating order of starting lane = (combat_index mod 3), split attacks floor(amount/3) and distribute remainder in lane order; reallocate unused attack damage to other living subpools in same deterministic order. Within lane damage target priority runner, grunt, brute, boss. If boss pool remains but ordinary center subpools have HP, boss receives damage last.
Ordinary enemy retaliation before guard block: raw = ceil((10+3*d+9*isqrt(P))*ordinary_remaining/base_hp) when ordinary_remaining>0 else 0; boss surviving after spawn adds ceil((10+3*d+9*isqrt(P))/5). Apply guard block once to TOTAL and max(0,raw-guard_block). If current wall is zero, subsequent damage goes to townhall. If townhall becomes 0 wave is lost immediately, no further resource deductions/damage; simulation still transitions to results at fixed time. If all pools empty before results, show CLEAR and no further enemy attack. Attack/retaliation do not happen during PREP or RESULTS.
At RESULTS enter (t=280) resolve once: victory iff townhall_hp>0 and all enemy pools zero, else loss. Victory reward wood min(cap,wood+100), stone min(cap,stone+60), food min(cap,food+50), consecutive_wins++, consecutive_losses=0, village level increases by 1 every 3 wins to max50 (from persistent total_wins); loss consecutive_wins=0, consecutive_losses++ capped at3, wood penalty min(wood,floor(wood/5)), stone penalty floor(stone/5), food penalty floor(food/5); vote C halves each penalty with integer floor. For defeat, wall_hp=max(250,wall_hp), townhall_hp=max(250,townhall_hp) but never above max; ensure at least 100 wood via max(100,wood) and cap. Results idempotent one-time event. At rollover, wave_number++, reset registered roles, vote and enemies, preserve persistent village.
Vote every EVEN wave, available [t=80,t=120), one latest ballot per player, exact A/B/C. Only current-wave participants can vote. A if wood>=50: deduct 50 wood at close, +15% guard block for wave; B if food>=50: deduct 50 food, +15% attack; C free, -10% guard block but reduces stock penalties on loss by 50%. A/B unavailable if insufficient resource at close: choose valid option with next highest votes, otherwise C. Tie: lexicographic A->B->C (deterministic, no RNG); zero votes: choose A if affordable else C. Vote decision applied exactly at tick1200, once, and displayed. No vote/more comments required to continue playing.

## Bounds and anti-spam
Role counts B/G/F can exceed population anchor, but effective counts for formula each capped at 10,000. Never repeat role contribution based on number of comments. Each ID is one player per wave, independent of comment burst. Send feedback only for accepted role/new choices and meaningful events; no invitation to flood or game stats based on comments count. Store alias opt-out and safety filtering. No paid combat advantage.

## Explicit fixtures / initial acceptance cases
One player XAY: NPC militia makes solo P=1 wave possible; 100 duplicate role comments create only one registration, not 100 workers. FIRST role change after initial join is allowed immediately; later changes require >=600 sim ticks after preceding change. At most one active role per player and boundary. Boss appears tick2600 before combat boundary, so two resolution opportunities at tick2600/2800. At tick2800 combat resolves BEFORE entering RESULTS. No viewer wave triggers WAITING not endless losses. No game clock catches up while provider is down. A role action at t=299.9s is assigned NEXT wave; replay must preserve acceptance tick exactly.


---

# 2. DETERMINISTIC SIMULATION — BINDING

## Integer/time model
10 fixed logical ticks per second, ticks int64 and no floating arithmetic or OS wall clock in game reducer. Global wave tick local range 0..2999. Every tick scheduled in strict ascending sequence. Phase is derived only from local tick: [0,1600)=PREP, [1600,2600)=ASSAULT, [2600,2800)=FINALE, [2800,3000)=RESULTS. A combat resolution at boundary tick 2800 executes BEFORE the phase becomes RESULTS and result is then evaluated on the same tick; do not process economy twice. Tick 3000 closes RESULTS and moves to WAITING for next wave. On restart use committed tick and accepted input log, not elapsed wall time.

## Total order of events at every tick boundary
1. WAITING accepts the first authorized role and starts a wave at tick0; PAUSED/SOURCE_DISCONNECTED/STORAGE_UNAVAILABLE freeze ticks. STOPPED rejects new game actions; operator/source controls remain usable.
2. Consume durable input_events ordered by input_seq whose scheduled_tick<=current boundary and not previously applied. Ingest schedules at last committed wave_tick+1, so no input arriving during a tick can affect that already committing tick. During RESULTS assign valid join to NEXT wave tick0.
3. If t=1200, close vote and choose option.
4. If t=1600, lock P,d,base_hp and spawn ordinary pools.
5. If t is positive multiple 200 <=2800 and not RESULTS: aggregate roles and perform builder economy+construction once; if t>=1800, apply fighter phase (boss spawn if t=2600) and enemy response.
6. If t=2800, compute final wave outcome once and write result.
7. Emit state hash, outbox entries, committed sequence and tick in ONE transaction. Publish committed state only.
8. If t=3000, carry progression and NEXT-wave pending registrations, reset wave and WAITING until valid role; do not auto-start an empty wave.

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
- t=2999 role action belongs NEXT wave, no XP in finished current wave.
- Tick is int64; cap duration/sequence; schema validates overflow.
- If zero participants and WAITING, no simulation tick advances or wave loss occurs.
- Out-of-order upstream source event timestamps do not rewrite previously committed game state. Server accept sequence/time are authoritative. Capture both when allowed for diagnostics.
- If source disconnects, freeze wave time by default. Queued previously committed inputs are replayed; no fake viewer events.

## Event sourcing and idempotency invariants
A valid command event stores accept_tick, sequence, event_id, actor_key, wave number, canonical payload. Same unique provider event ID is idempotent. For missing stable provider event IDs, dedupe best-effort by source session+actor+normalized command+short interval; do NOT assert exactly-once delivery across crashes; one-player role semantics still prevent multiple contribution. Duplicate role command never gives extra XP. Snapshot includes all role membership, accepted window/cooldown changes, pending next-wave actions, vote, pools, resource counters, wave_tick, game rules version and output sequence.
A tick transaction owns: accepted commands for tick, state change, emitted domain events and last_processed_sequence; commit before WebSocket publish. If DB unavailable, simulation MUST pause; never send uncommitted successes to clients. Recovery replays accepted inputs in deterministic order, exactly once by sequence, verifies SHA256 canonical state.

## Canonical hashing
Define state hash as SHA256 of RFC 8785 JSON Canonicalization Scheme encoding of a versioned **pure game state**; omit transport timestamps, random visual positions and network counters. Integer-only numbers safe within JSON precision 2^53-1; encode any larger sequence/tick as decimal strings. Include sorted participant roles by actor_key, sorted enemy pools by (lane,archetype), all resources and vote, tick and relevant metadata; exact field list in 03_PERSISTENCE.md. Every commit affecting canonical serialization requires regenerated and independently reviewed golden fixtures, and master version bump if incompatible.


---

# 3. POSTGRESQL 18.6 DATA, TRANSACTIONS AND RECOVERY

## Ownership, isolation and transactional boundaries
Only PostgreSQL is authoritative. One Go simulation leader per village holds an exclusive database-backed lease with fencing epoch and heartbeat. An expired lease does not grant a previous owner authority: each write compares the fencing epoch; stale owner writes fail. Source ingestion and simulation reducer are separate transactions but **never** acknowledge an authoritative mutation before its durable commit. Use serializable or appropriately row-locked transactions for sequence/ownership; handle SQLSTATE 40001/40P01 retries with jitter, max three retries then explicit 503/degraded. No connection pool unlimited growth. No schema migration on hot production startup. PostgreSQL 18.6 exact docker digest pinned in M01.

## Schema contract — M01 bootstrap migrations
Required tables and constraints (actual executable, forward-tested SQL is M01 deliverable, not prose):
- villages: id UUID PK, broadcaster_key CHAR(64) UNIQUE (scoped pseudonym), level SMALLINT CHECK 1..50 DEFAULT1, wall_tier SMALLINT CHECK1..20 DEFAULT1, wall_hp BIGINT CHECK>=0 DEFAULT1000, townhall_hp BIGINT CHECK0..500 DEFAULT500, wood/stone/food BIGINT CHECK0..1000000 DEFAULT 400/250/300, construction_work BIGINT CHECK>=0 DEFAULT0, total_wins BIGINT CHECK>=0 DEFAULT0, consecutive_wins SMALLINT CHECK0..10 DEFAULT0, consecutive_losses SMALLINT CHECK0..3 DEFAULT0, created_at/updated_at TIMESTAMPTZ; enforce wall_hp<=1000+200*(wall_tier-1).
- sessions: id UUID PK, village_id FK, mode TEXT CHECK mock|replay|authorized_live, lifecycle TEXT CHECK waiting|running|paused|source_disconnected|storage_unavailable|stopped, wave_no BIGINT >=1, wave_tick INT 0..3000, rules_version TEXT NOT NULL, writer_epoch BIGINT DEFAULT0, input_sequence BIGINT DEFAULT0, stream_sequence BIGINT DEFAULT0, last_applied_input_sequence BIGINT DEFAULT0, started_at, stopped_at, last_heartbeat_at. Unique partial index for one active session per village when lifecycle!=stopped.
- players: id UUID PK, village_id FK, actor_key CHAR(64) NOT NULL, ephemeral BOOL NOT NULL, alias TEXT length <=128 bytes, hide_alias BOOL DEFAULT FALSE, lifetime_xp BIGINT>=0, created_at, last_seen_at, UNIQUE(village_id,actor_key). Ephemeral actors are deleted when session expires; do not link session anonymous user to another live.
- session_player_progress: session_id FK, actor_key CHAR(64), session_xp SMALLINT CHECK0..300 DEFAULT0, PRIMARY KEY(session_id,actor_key).
- wave_members: session_id FK, wave_no BIGINT>=1, actor_key CHAR(64), role TEXT CHECK builder|guard|fighter, joined_tick INT, last_change_tick INT, last_update_seq BIGINT, xp_awarded BOOL DEFAULT FALSE, PRIMARY KEY(session_id,wave_no,actor_key). Pending-next-wave joins use wave_no=current+1.
- ballots: session_id FK, wave_no BIGINT, actor_key CHAR(64), option CHAR(1) CHECK A|B|C, input_seq BIGINT, PRIMARY KEY(session_id,wave_no,actor_key).
- input_receipts: session_id FK, provider_source TEXT, provider_event_id TEXT, inserted_at TIMESTAMPTZ, UNIQUE(session_id,provider_source,provider_event_id); NEVER insert NULL-ID as empty; optionally separate best-effort TTL cache. Index TTL cleanup. Only accepted game-affecting events must be durably deduped; repeated same-role chat can be rejected by bounded cache and metric before DB.
- input_events: session_id FK, input_seq BIGINT, accepted_tick INT, scheduled_tick INT, wave_no BIGINT, actor_key CHAR(64), kind TEXT, canonical_payload JSONB, rules_version TEXT, inserted_at TIMESTAMPTZ, PRIMARY KEY(session_id,input_seq). Index (session_id,wave_no,accepted_tick,input_seq).
- checkpoints: session_id FK, wave_no BIGINT, wave_tick INT, input_seq BIGINT, revision BIGINT, state_json JSONB, state_hash CHAR(64), rules_version TEXT, created_at TIMESTAMPTZ, PRIMARY KEY(session_id,revision). Index newest per session.
- domain_events: session_id FK, revision BIGINT, event_index INT, event_type TEXT, payload JSONB, created_at TIMESTAMPTZ, PRIMARY KEY(session_id,revision,event_index). Must not store unmoderated raw text.
- stream_outbox: session_id FK, stream_seq BIGINT, revision BIGINT, type TEXT, payload JSONB, created_at TIMESTAMPTZ, PRIMARY KEY(session_id,stream_seq), checkpoint_ref optional.
- operator_accounts: id UUID PK, username CITEXT or text unique lower index, password_hash TEXT with Argon2id, role admin|operator|viewer, disabled_at, created_at.
- operator_sessions: id UUID PK, account_id FK, secret_hash CHAR(64), expires_at, revoked_at; secrets hashed not stored plain.
- operator_audit: id BIGINT identity PK, actor_id UUID, action, target, reason, created_at, ip_hash optional; minimum privilege for audit reads.
- writer_leases: village_id PK FK, writer_epoch BIGINT, owner_instance TEXT, leased_until TIMESTAMPTZ.
All FK cascade policy explicit: session game logs cascade only by audited retention purge; player deletion does not accidentally cascade authoritative wave outcomes. Every bigint checked for safe SQL/JSON protocol ranges. No schema in public PostgreSQL search_path for API role; grants strictly least-privilege.

## Transaction protocols
Ingest: validate adapter provenance, scope, ID and grammar; in transaction lock session row, allocate input_seq, determine target_wave and scheduled_tick = committed_tick+1, insert receipt for stable event ID, append immutable validated input_event; do NOT mutate authoritative wave_members or ballots in ingest. Reducer applies accepted commands and cooldown exactly once. Duplicate provider ID returns prior receipt with no second event. During RESULTS target next wave tick0, during WAITING initial valid role starts wave at tick0. Acknowledge only after committed append.
Reducer: read session row FOR UPDATE and verify writer epoch; pull unprocessed input_events ordered by input_seq with target_wave=current and scheduled_tick<=boundary tick; apply all once; advance logical tick and player roles, settle economic state, checkpoints and game event/outbox. Write updated village+session and outbox in same DB transaction. If commit fails, publish NOTHING. Snapshots every 600 ticks and wave end; tick/role commands since last full snapshot are recoverable by reading ordered immutable input_events. If simulation cannot persist a tick, freeze until DB back.
Recovery: acquire new epoch, load latest valid checkpoint, verify hash, replay accepted inputs up to last persisted boundary, never advance beyond committed tick using wall clock. Outbox might contain committed messages not published: replay over WebSocket from stream_seq if retained; otherwise send full projected snapshot and new cursor. Checkpoint hash mismatch => fail closed/incident, no auto-repair. Crash tests kill process before/after commit and verify no duplicate XP, resources, votes or rewards.
Database partition/retention: input_receipts 24h minimum after session or provider replay horizon (choose longer verified), input_events until a verified checkpoint covers them and 30-day diagnostics expiry, stream_outbox 7 days or snapshot-resync always available, operator audit 180 days (actual privacy/legal policy may require different retention; privacy owner gate). Raw comment body never stored outside bounded redacted debug traces.

## Privacy erasure versus replay (non-negotiable)
Individual erasure on active wave: suppress alias immediately; revoke future contribution at next boundary and queue deletion. At next committed wave boundary, create an anonymized aggregate checkpoint, revoke personal mapping, delete personally linkable player, membership, pending votes and pre-checkpoint per-player input history once safe, update privacy_epoch; only replay since new checkpoint is promised. Past settled aggregates and irreversible resource totals remain as nonidentifying numeric totals, not actor mappings. Refresh the canonical hash at privacy boundary and record verifiable privacy_epoch audit without username. No user identity may be resurrected by restoring old backups: restore pipeline reapplies deletion tombstone ledger kept separately with retained anonymous cryptographic markers. Delete tests include live, snapshot, backup restore and cross-session consistency.

## Two distinct snapshot types
- PRIVATE canonical reducer state: complete sorted wave members, roles, votes, HP subpools, resources, counters, tick, pending roles, input sequence, privacy_epoch, rules_version; may be >1MiB and never sent to broadcaster video.
- PUBLIC projected view: aggregate counts, wave stage, lane statuses, wall/townhall, resource rounded progress, selected top 12 moderated alias samples, vote counts, contribution highlights, source health; target <=32KiB and HARD max 128KiB. Stream never exposes ActorKey/HMAC/real provider IDs.
Canonical JSON hash via RFC8785 for state, actor list sorted by actor_key bytes, enemy lanes 0..2 archetypes0..3. Domain serialization version must be part of hash. Input_events/telemetry/wall timestamps are omitted.


---

# 4. BACKEND API AND WEBSOCKET PROTOCOL

## Conventions and error model
All endpoints are /admin/v1 unless stated. JSON UTF-8, Content-Type application/json, no floats where integer required. UUID lowercase, UTC RFC3339 timestamps. Accept maximum HTTP request 16KiB (mock events aggregate max 64KiB). Success is 200/201/204, validation 400 code=INVALID_ARGUMENT, unauthenticated 401 UNAUTHENTICATED, forbidden 403 FORBIDDEN, missing 404 NOT_FOUND, wrong lifecycle/duplicate conflicting session 409 FAILED_PRECONDITION, throttle 429 RESOURCE_EXHAUSTED with Retry-After, DB unavailable 503 UNAVAILABLE. Every error JSON: {"error":{"code":"ENUM","message":"safe_text","request_id":"uuid","retryable":false}}. Never include secrets/provider identifiers or SQL errors. Idempotent mutations require Idempotency-Key UUID stored per actor/action for 24h; replay returns original response.
All admin write requests: same-site Secure HttpOnly session cookie, CSRF X-CSRF-Token linked to server session and Origin allowlist; RBAC roles viewer(read only), operator(start/pause/resume/mute), admin(accounts/data deletion/settings). Login protected with per-user and global rate limits. Unity broadcast client accesses separate view-only short-lived stream token via operator panel; token lifetime 300s and rotate without losing connection.

## Routes (exact request/response behavior)
- GET /healthz -> 200 {"status":"alive"}; no DB check.
- GET /readyz -> 200 {"status":"ready","writer_epoch":integer} or 503; protected metrics not exposed.
- GET /metrics -> Prometheus exposition, internal network only.
- POST /admin/v1/auth/login {username,password} ->200 {user:{id,role},csrf_token}; Set-Cookie secure session. Wrong creds ->401 generic; 5 attempts/5min ->429.
- POST /admin/v1/auth/logout {} ->204 revoke cookie.
- GET /admin/v1/me ->200 {id,role,csrf_token}; 401 anonymous.
- GET /admin/v1/sessions ->200 {items:[SessionSummary],next_cursor?}; bounded page size <=50.
- POST /admin/v1/sessions/start {village_id:UUID,source:"mock"|"replay"|"authorized_live"} ->201 {session_id,status:"waiting",wave_no:1}; 409 if already active; authorized_live requires verified gate flag and adapter auth, else403 PLATFORM_NOT_AUTHORIZED.
- POST /admin/v1/sessions/{id}/pause {reason:string<=256} ->200 SessionSummary; waiting/paused idempotent; forbidden status returns409.
- POST /admin/v1/sessions/{id}/resume {} ->200 SessionSummary; source_disconnected or DB unhealthy ->409.
- POST /admin/v1/sessions/{id}/stop {reason:string<=256} ->200 SessionSummary with stopped=true; idempotent if already stopped.
- GET /admin/v1/sessions/{id}/state ->200 PublicState plus internal safe operator stats but no raw stable provider IDs.
- GET /admin/v1/sessions/{id}/metrics ->200 counts, p50/p95 backend latency, ingress dropped breakdown, queue, ws.
- POST /admin/v1/sessions/{id}/stream-token {} ->200 {token,expires_at,websocket_url}; viewed only by operator/admin.
- POST /admin/v1/sessions/{id}/moderation {actor_public_ref:string,action:"hide"|"unhide"|"block",reason:string<=256} ->200 {applied:true}; never expose ActorKey via public wire.
- POST /admin/v1/sessions/{id}/privacy/erase {actor_public_ref:string,reason:string<=256} ->202 {status:"queued",request_id:UUID}; admin only, append auditable erasure workflow.
- POST /dev/v1/events {session_id:UUID,events:[{event_id:string,viewer_id:string,alias:string,comment:string}]} ->202 {accepted:int,ignored:int,invalid:int}; dev mode only, route not registered prod/stage.
- POST /admin/v1/sessions/{id}/settings {low_vfx:bool,mute_audio:bool} ->200 settings; operator permitted; changes do not alter combat/official gameplay.
Every request/response schema must be materialized as machine-readable OpenAPI 3.1 and verified via schema-conformance tests in M03; these exact routes are normative, not examples.

## WebSocket
GET /stream/v1/sessions/{session_id} with a short-lived token validated from Sec-WebSocket-Protocol; query-string tokens are forbidden in all environments to avoid leaks. TLS/WSS for remote. Upon connect send full public_snapshot within 1s, then public_state_diff at most 5Hz. Every envelope JSON:
{"schema_version":1,"type":"public_snapshot|public_state_diff|visual_event|connection_status|error","session_id":"uuid","stream_seq":"decimal-string","revision":"decimal-string","sent_at":"RFC3339","payload":{}}
stream_seq monotonic per session only; no gaps silently ignored. On gap, client requests {"schema_version":1,"type":"resync","after_stream_seq":"decimal-string"}; server sends public_snapshot with latest stream_seq (NOT unbounded delta backfill). Client discards older revisions and any unexpected schema version; present STALE/RECONNECTING when >3s without snapshot or heartbeat. Server ping every10s; idle timeout30s. max single outbound frame 128KiB, stream outbox max bounded 20 messages/client; drop cosmetic events first; if still full disconnect slow subscriber with 1013 and require fresh snapshot.
public_snapshot payload fixed fields: {wave_no:int,phase:enum,remaining_seconds:int,wall_hp:int,wall_max:int,townhall_hp:int,townhall_max:500,wood:int,stone:int,food:int,counts:{builder:int,guard:int,fighter:int,unique_joined:int},lanes:[{id:"west|center|east",ordinary_hp:int,boss_hp:int}],vote:{open:bool,options:[A,B,C],counts:{A:int,B:int,C:int},selected?:char},source_status:"connected|disconnected|mock|replay",spotlights:[{public_ref,display_alias,role}],event_banner:{kind,text_key,expiry_ms}}. Stream diff uses RFC6902 JSON Patch only if payload<=128KiB and base_revision matches; otherwise server sends full snapshot. Fixed text keys map to client localization catalog, not arbitrary HTML or user input.

## Time/latency
Internal latency from backend event received to committed publish, p95 <=1.0s at measured target; external platform-to-backend not controllable and must be separately shown as UNKNOWN/MEASURED. UI acknowledgement can appear at next publish once valid command committed; domain effect occurs at 20s boundary. Max concurrent WebSocket clients initially 4 trusted devices, not 10k player sockets; viewers watch TikTok video, they are not websocket clients.


---

# 5. TIKTOK CONNECTOR, PRIVACY, SECURITY

## TikTok factual gate
TikTok's public developer documentation describing LIVE history and Mini Games does NOT prove a universal, public real-time LIVE-comment subscription for desktop OBS broadcast. The TikTok-approved real-time event acquisition mechanism, permitted scopes, account consent, rate limits, identity stability, policy and allowed monetization MUST be documented and tested before production adapter activates. No undocumented scraping, cookie theft, unofficial session bypass, reverse-engineered endpoint or falsified counts. The system must remain useful in mock/replay without real LIVE data. Official reference: https://developers.tiktok.com/docs/en/our-guidelines-developer-guidelines ; https://developers.tiktok.com/doc/search?q=live .
GATE EXTERNAL-LIVE: owner must attach provider contract, approved app permissions, consent proof, sample redacted lawful event payload, source event-ID/reconnect semantics, service terms and region restrictions. No agent self-approval. If not met, leave LIVE disabled and mark P1 BLOCKED while P0 passes independently.

## Adapter specification
Source interface Start(ctx,session,emit callback), Stop(ctx), Health(ctx), Capabilities(ctx), Reconnect(ctx,cursor). Capabilities: stable_viewer_id,stable_event_id,replay_cursor,delivery_order,timestamp_reliability,maximum_rates,display_name,moderation_events. Event {provider,source_session,provider_event_id?,provider_viewer_id?,alias?,message,received_at,upstream_at?,event_kind:"comment"}. All inputs untrusted. Before parsing validate ACTIVE session, allowed source, UTF-8, <=500 bytes comment, alias <=64 Unicode graphemes, source ID <=256 chars, absence of control/bidi override spoofing. Single source adapter active per session. The mock source is seeded by explicit config and uses watermark; replay input is versioned NDJSON fixtures and watermark.
Stable viewer ID: actor_key=HMAC-SHA256(identity_secret, provider_version||broadcaster_key||provider_viewer_id) stored hex. No usable viewer ID: ephemeral per connection/session; persistent XP disabled, on-screen indicate participation counts may be approximate; do not treat display alias as identity. Stable event ID provides idempotence; missing IDs use best-effort TTL dedup by actor+normalized_command+arrival bucket and cannot promise exactly once. Server acceptance order wins, not provider wall clock. On reconnect freeze state, display CONNECTIVITY LOST, after adapter health and cursor restoration resume only on operator policy; no synthetic backfill beyond provider-supported replay.

## Abuse protection and capacity
One role selection starts current wave; role-change cooldown >=60 sim seconds, no repeated XP. Ballot last valid vote counts once. Event rejection reasons separately counted: invalid_grammar, oversized, duplicate_provider_id, repeat_noop, cooldown, blocked_user, queue_overflow, unauthorized_source. A global bounded queue (20k) and per-identity token bucket (5 messages/10sec burst 10, but gameplay still only 1 role) must never cause memory explosion. If overload, drop invalid/non-actionable first; when any valid command lost, mark DROPPED_GAME_EVENT on operator dashboard, count exactly and record overload alert; no silent falsification. This gate is monitored even in mock tests.
No gift/like/share/follow/coin data influences attacks, rewards, ranking or chance. No gambling/paid reward. Moderators can hide aliases or block identity with audits. No aggressive requests to comment repeatedly, spam, or manipulated engagement.

## Data/privacy threat boundaries
HMAC keys/CSRF/DB passwords/stream tokens stored out of repo in secret store or injected environment/volume; do not write them to logs. Rotate via versioned secrets with mapping for permitted stable identities; never re-HMAC raw provider IDs from impossible missing material. Account-level alias opt-out and privacy removal works and is tested in restored backup. Stream emits only ephemeral random public references and filtered aliases; no raw ActorKey, provider ID or private IP to Unity. Alias default safe fallback 'Villager ####'; reject URL-like handles, profanity, excessive marks and bidi tricks before publication.
Security measures: TLS, CSRF, same-site cookie, Argon2id admin secrets, RBAC per route, per-user login throttling, signed short-lived stream access, SQL parameterization, audit and WAF/proxy request size cap. No external admin public exposure by default. Dependency/CVE and secrets scan. GDPR/Vietnam privacy evaluation is separate explicit human legal review, not a claim of compliance from code.
Retention: raw diagnostic comments maximum 24h if needed; source event receipts at least horizon of replay cursor and <=7d unless documented; accepted events after privacy-safe checkpoints until 30-day pruning; operator audits 180d or local legal-approved retention; public nickname until deletion. Privacy erase algorithm is in 03_PERSISTENCE; backup restore must replay erasure ledger.

## Threat test matrix
Command replay, duplicate ID, missing stable ID, 1,000 comments from one viewer, 5,000 events/s burst, 1MB message, malformed UTF-8, SQL/XSS/bidi, impersonation, emergency stop, stale session ID, stolen bearer token, expired stream token, CSRF browser attack, reconnect storm, DB outage, disk full, leaked metric endpoint, moderation evasion, privacy deletion and backup resurrection. Critical/high threat unmitigated = production FAIL.


---

# 6. UNITY 9:16, ART, INTERACTION

## Locked visual product
Unity 6000.6.7f1, 2D pseudo-isometric village (not 3D). Vertical 1080x1920 reference, target 30 FPS minimum on reference capture host; fixed orthographic camera, no manual pan. Render <=150 representative pooled entities (NOT 10k player objects). Default quota 40 builders, 40 guards, 40 fighters, 30 enemies; quotas adapt to real on-screen threat, <=150 total, boss always visible. Unity receives only server truth; client visual pools and interpolated HP cannot change combat. Prefer Unity Built-in 2D pipeline, URP only after approved measured ADR.
World: centered castle+wall occupying central 44% width; WEST/CENTER/EAST approach lanes visibly distinct; enemy silhouettes at edges; builders repair, guards defend and fighters battle. UI responsive anchored Canvas, never baked essential text.

## Responsive safe areas
Reference 1080x1920, CanvasScaler matchWidthOrHeight=0.5. Top HUD y=5..14%, castle/world y=20..74%, commands x=8..75% y=76..86%. Reserve right x>82% and bottom y>88% for TikTok chrome until actual live OBS overlay verified; platform overlays vary by region/client and require screenshots.
At 360x640 equivalent, commands >=16px, timer and castle HP >=14px, secondary text>=12px. Text contrast>=4.5:1; white/light outlined text on dark scrim; no reliance on color alone. Max 12 simultaneous player alias labels, truncated to <=12 visible characters plus ellipsis.

## Mandatory visual states
ATTRACT, WAITING_FOR_VIEWER, PREP, PREP_60_WARN, PREP_15_WARN, ASSAULT, FINALE, BOSS, RESULTS_WIN, RESULTS_LOSS, VOTE_OPEN, SOURCE_RECONNECTING, PAUSED, STORAGE_UNAVAILABLE, DEGRADED_DROPS, MAINTENANCE. Mock/replay watermark always displayed outside real authorized live mode. Source disconnect and unsaved state are not disguised as valid gameplay.
Healthy valid join must update aggregate count on next stream publication (<=200ms publication cadence, backend SLO separate); show rotating spotlight of filtered alias if permitted, not a promise every avatar appears. Repeated identical comment silent; changing role affects next 20s tick. Vote shows A/B/C and integer counts/close timer; result lasts 20s. No dynamic camera movement obscures the village.

## Animation and content manifest (mandatory M06)
ART001 player builder yellow apron/hammer, guard blue shield/cape, fighter red weapon/helmet; easily distinguished in grayscale and 48px preview; 3/4 facing center.
ART002 grunt round green, runner slim purple, brute square orange, boss black-violet oversized high-contrast silhouette.
ART003 wall 5 visual damage/tier states mapped across 20 tiers; townhall, one watchtower, 6 houses, 3 decorations.
ART004 3 approach lanes, 6 environmental tile variants, 4 lighting palettes (visual only), atlas and background.
ART005 projectile, hit, repair spark, shield effect, boss telegraph, no more than 20 simultaneous VFX bursts.
UI001 command icons, resource icons x3, wall, hall, boss, danger, timer, vote, source health; Vietnamese default and English localization.
SFX001 12 original/cleared short effects, one original ambient loop, two short music loops with rights recorded; music mute by default until stream rights confirmed.
Every player role: idle4, walk6, action6, hit2 frames at 12fps; enemies idle4, walk6, attack6, hit2; boss idle6, attack8, hit2. Source frame 128x128 PNG RGBA with pivot=(0.5,0.08), consistent 3/4 direction; Unity atlas <=2048x2048, filter Point for pixel-silhouette art. Atlas packing and pooling must be profiled in capture build.
Each asset needs exact source, provenance, right-to-use, dimensions, animation import preset, atlas, screenshot and license metadata. Placeholder assets M01 only; M06 PASS requires production original/licensed visual package, no third-party copyrighted characters.
Animations and effects not tied to ticking game logic. Never manufacture fictitious active viewer indicators, names or history.

## Accessibility and comprehension
Reduced-motion disables camera shake/strobe. No flashes>3Hz. At 360x640 with actual chat overlay, five uncoached viewers watch 3 seconds with audio off; >=4 correctly identify threat and one comment command. Capture two muted 15min sessions at 720x1280 and 1080x1920 plus 1440x2560 crop; document real TikTok overlays. Test source disconnection and empty-room states as first-class UX.
This is a test requirement, not a claim the research already happened.


---

# 7. DEVELOPMENT, CI, DEPLOYMENT AND INCIDENTS

## Runtime reference and toolchains
Windows 11 host for Unity OBS with >=6 physical CPU cores, >=16 GiB RAM and discrete GPU approximately RTX3060-class as reference only; Linux amd64 backend reference dedicated 8vCPU, 16GiB RAM, NVMe SSD and PostgreSQL 18.6 over private loopback/network. These are NOT verified current user's equipment; tests record real CPU/GPU/storage before claiming numbers. Go 1.27.3, Unity 6000.6.7f1, Postgres18.6. Pin exact external libs, node LTS, Docker digests and Unity packages during M01; no moving latest tags. Windows PowerShell and Linux Bash entrypoints run from repo root.
Single-main workflow: pre-commit quality gates, then commit main, then verify GitHub Actions and revert/repair. No auto-merge, PRs or branches unless explicitly requested.

## Required configuration register
TTG_ENV=dev|stage|prod (default dev)
TTG_SOURCE=mock|replay|authorized_live (default mock; live fails closed until permission verification)
TTG_HTTP_ADDR=127.0.0.1:8080 (default local)
TTG_DATABASE_URL: secret, mandatory all modes
TTG_HMAC_KEY_VERSION: nonempty and TTG_HMAC_SECRET: secret mandatory
TTG_ADMIN_PASSWORD_HASH: Argon2id mandatory on stage/prod, seed dev creds only in local environment
TTG_STREAM_SIGNING_SECRET: random >=32 bytes, mandatory stage/prod
TTG_SOURCE_CREDENTIAL_REF: mandatory in authorized_live
TTG_LOG_LEVEL=info; TTG_METRICS_ADDR=127.0.0.1:9090; TTG_MAX_QUEUE=20000; TTG_STREAM_RATE_HZ=5; TTG_SNAPSHOT_TICKS=600
Invalid enum, weak secret, missing key or migration mismatch => fail startup. .env.example placeholders ONLY, real secrets gitignored and scanned.

## Mandatory commands/scripts
tooling/doctor, up, down, migrate-up, migrate-down, test-unit, test-race, test-integration, test-schema, test-golden, test-load, test-unity, test-operator, test-security, backup, restore-verify, capture-obs; both .ps1 and .sh wrappers or portable Go CLI where possible.
doctor prints versions and refuses mismatched stable pins; up starts PostgreSQL/backend/operator and mock source. down preserves persistent state unless --wipe-dev-data explicit with interactive warning. All scripts write JSON evidence UTC timestamp, SHA, host, versions, command, exit status. If a test cannot run due to missing device/licensing, mark BLOCKED not PASS.

## Deployment
Docker Compose local single-host baseline; Postgres binds 127.0.0.1/private only, public API behind TLS reverse proxy with auth and fixed limits; do not expose admin or DB on public IP. Production image pinned by digest, semver, rollback. Source must be explicit and accepted at startup; never downgrade real-source sessions silently to mock. Stage exercises mock/replay, real integration stage after authorization. Runtime CPU/memory/pid limits explicit in Compose and bench results.

## Observability/alarms
Counters input_received/accepted/duplicate/ignored/cooldown/invalid/dropped, actual distinct player registrations, source health, writer epoch, committed tick, queue depth, websocket sessions, stream lag, DB tx retries and errors, Unity fps/audio peak. Histogram backend receive-to-commit/publish p50/p95/p99. External TikTok-to-backend latency is separately UNKNOWN until measurable.
Alert: queue >75% for30s, ANY dropped valid game action, internal p95>1s sustained5min, missing checkpoint >90 simulated seconds, DB disk>80%, source unexpectedly offline >5s, unauthorized operator attempt spike. Operator must see prominent warning not falsified healthy status.

## Availability/recovery
Nightly encrypted PostgreSQL backups and continuous WAL archive; aim RPO<=5min, RTO<=30min measured on target hardware. Keep >=7 restore points, weekly restore verification with privacy-erasure tombstone reapply; no reliance on nightly backups alone for 5min RPO. Event log+checkpoints determine exact last committed logical tick; never resume based on elapsed real wall time. Store default snapshot every600 ticks and wave boundary; rollback restores image digest + compatible schema, forbid unsafe downgrade that loses data.
Provider disconnect: freeze ticks, show banner, retry backoff 1/2/4/8/16/30s, require healthy source and authenticated operator resume after sustained connectivity >=10s. DB outage: stop acknowledgements and tick advance, enter STORAGE_UNAVAILABLE; replay after recovery. Unity reconnect: discard stale view, fetch public snapshot with new stream sequence, resync. Queue overload: drop invalid/non-actionable first, flag any genuine game command lost as DEGRADED and pause for operator if sustained 30s. Emergency stop: operator action audited, no further simulated ticks.
Privacy erase: queue, immediate alias suppression, anonymized checkpoint at safe wave boundary; retain deletion tombstone ledger for future restore. Incident retrospective required with timeline, root cause and reproduction.

## Production LIVE human prerequisites
Confirm approved provider event source, owner account/region and streaming rights; verify full overlay and latency on LIVE preview; original audio/art IP rights, privacy notice and deletion mechanism, staffed moderation, emergency stop, 2h rehearsal, physical workstation and recovery drills. Engineer automation CANNOT grant platform privileges or human legal approvals. If not met, mark LIVE release BLOCKED, allow standalone simulator deliverable only.


---

# 8. TESTS, SLO, ACCEPTANCE

## Core principles
All test cases MUST be executable and reproducible, or explicitly identified as a manual human test. A spec is never evidence that code works. Every PASS requires SHA + command + fixture + time + environment + log artifact; human-required tests use real human signature. A missing license/device/credential = BLOCKED, not PASS. The required baseline host is specified in 07_OPERATIONS, and any other tested hardware recorded verbatim.

## Mandatory test IDs
QA-GOV-01 master is only normative source, main-only branch policy, no hidden PR dependency.
QA-GOV-02 pinned official versions, exact checksums/lockfiles, Docker image digest.
QA-MAT-01 implementation matrix dependencies are DAG; SQL migrations available before DB-dependent work.
QA-CMD-01 exact Vietnamese command normalization including diacritics, U+0111, NFKC, whitespace, invalid punctuation.
QA-CMD-02 join once/wave auto-contributes, 1,000 repeated comments never add points, XP, DB roles or damage.
QA-CMD-03 controlled role switch cooldown and boundary effects, RESULTS registers NEXT wave.
QA-PSY-01 5-person muted 3-second phone test >=4/5 understand role and threat, document participants and setting.
QA-PSY-02 no XP for chat repetitions/gifts; same effective role for new and veteran.
QA-SIM-01 exact logical phase ticks 0,1599,1600,1799,1800,2599,2600,2799,2800,2999,3000.
QA-SIM-02 mandatory deterministic reducer operation order, integer rounding, lane redistribution.
QA-SIM-03 boss spawns before combat at t2600; t2800 final combat before results.
QA-SIM-04 WAITING without players and pause/disconnect freeze.
QA-SIM-05 defeat inventory penalty and recovery, level retention; no overflow/negative.
QA-BAL-01 simulated P=1,3,10,100,1000,10000 at 0/20/40/60/100% fighter mixes, d=1,5,10 and boss/nonboss; battle must end within 300s, no divide by zero, no mathematically unwinnable recurrence after losses with at least one fighter or low-P NPC militia.
QA-BAL-02 1,000 generated session/wave fixtures, test transparent scaling, bounded XP, resource and load. Target 65-80% first-three-wave win rate is hypothesis, NOT mandatory or faked until viewer research.
QA-ECO-01 construction/repair, one upgrade max per boundary, resource caps, vote deductions and rewards exactly once.
QA-VOTE-01 close boundary t1200, ties A>B>C, insufficient-resource fallback and no invalid ballot.
QA-DET-01 same accepted inputs+rules_version -> RFC8785 hash equal on >=100 checkpoint/restarts/crash locations.
QA-DET-02 no iteration, RNG or unstable timestamps influence combat hash.
QA-DB-01 up/down migrations, FK/unique/check constraints, indexes and least-privileged credentials.
QA-DB-05 durable input and reducer-outbox atomic transactions; reject stale fencing epochs.
QA-API-01 OpenAPI3.1 all endpoints, error schemas, status codes, idempotency and negative tests.
QA-API-02 stream <=128KiB frame, gap/resync, reconnect, old/new schema rejection, slow subscriber drop.
QA-SEC-03 identity HMAC rotation, alias optout/privacy delete and restored-backup erasure.
QA-SEC-05 CSRF/RBAC/stream token expiry/admin brute force protection/emergency stop.
QA-SEC-06 XSS/unicode confusable/bidi/offensive alias sanitization and audit.
QA-LIVE-01 production source must remain blocked absent verified right. Watermark on mock/replay.
QA-LOAD-01 10k unique players, 1k events/s sustained 10min, burst5k/s for10s. Measure accepted valid action success >=99.9%, if valid loss occurs show DEGRADATION and release FAIL; backend ingress-to-committed-publish p95<=1s and p99<=3s; external transport measured separately.
QA-VIS-01 <=150 pooled on-screen representatives; measured >=30 average fps and >=24fps 1% low across 15-minute 1080x1920 capture on reference workstation with Unity profiler/GC data.
QA-UX-01 all mandatory screens, 360x640 legibility, TikTok-overlay capture with no essential occlusion.
QA-UX-02 every phase accessible; PREP actionable status/event within60s without faking player input.
QA-OBS-01 record OBS actual output and streamer human validation; capture screenshots & audio levels.
QA-OPS-03 backend SIGKILL, DB disconnect, source disconnect, provider duplicate, WS resync, backup restore, writer lease split-brain attempts.
QA-ASSET-01 art/audio manifest completed and rights verified; production placeholder=FAIL.
QA-GATE-01 real owner permissions, legal/privacy, licensed artifacts, broadcast approvals, physical hardware readiness require external evidence.
QA-GATE-02 review current policy and third-party app availability prior to P1 and on updates.

## Quality thresholds
Safety: 0 known critical/high exploitable vulnerabilities, secret scan clean, signed asset provenance. Determinism: 0 hash mismatches in 100 replay/crash positions; no duplicate reward. Load: peak queue <=20k, no memory leak in 2hr soak, valid event loss 0 in normal baseline target (if loss, backlog DEGRADATION and FAIL until tuned), explicit ignored invalid/noops allowed, CPU/RAM and DB p95 reported. Performance never "PASS" without actual measured host specs. UX: >=4/5 3s comprehension; not a scientific randomized 10k-user survey. Recovery: RPO<=5min RTO<=30min measured in isolated restore rehearsal. Stream client <=128KiB messages and <=5Hz authoritative state publication with visual events <=20/s, FPS thresholds measured.
No dependency on monetization or gift use. Real-source platform readiness is separate gate; P0 mock pass does not imply live P1 pass.

## Evidence files and traceability
For each Mxx-Wxxxx: docs/evidence/Mxx-Wxxxx.md contains milestone, requirement IDs, source code commit, exact machine+OS, build and test commands, test IDs, expected vs actual, log path, metric summary, UTC, automated/human reviewer, status and rerun instructions. docs/traceability.csv has columns requirement_id,document_anchor,work_package,test_id,priority,status,evidence_uri; M01 creates script to fail if any MUST unmapped or PASS without valid evidence. Weekly or on milestones review drift in doc/fixture versions, and lint links.

## Required CI stages
Go test ./... + -race, fuzz seeds, sqlc/generation check, goose migrate up/down PG18.6, determinism golden, OpenAPI/JSON schema contract tests, security SAST/secret/SBOM/license, Unity EditMode/PlayMode with license, TS UI tests, k6/Go load harness on dedicated runner. Manual OBS, privacy, streamer and legal gates are never spoofed by headless CI.


---

# 9. ORDERED IMPLEMENTATION MATRIX AND MANDATORY WORK PACKAGES
Allowed state NOT_STARTED, IN_PROGRESS, BLOCKED, FAIL, PASS. All P0 milestones start NOT_STARTED until implemented/tested. M10 and M11 start BLOCKED on external rights. Each package requires docs/evidence/Mxx-Wxxxx.md with SHA, test IDs, exact command, UTC, environment/hardware, actual results, PASS/FAIL/BLOCKED and reviewer. Completion is NOT inferred from this document.

| ID | Dependencies | Binding deliverable | Initial state | Completion gate |
|---|---|---|---|---|
| M00 | none | resolve plan consistency, fixture plan, traceability | NOT_STARTED | no internal contradictory requirements |
| M01 | M00 | pinned Go/PG/Unity/TS skeleton; ALL SQL schema and CI | NOT_STARTED | clean Windows/Linux build and PG migrations |
| M02 | M01 | 10Hz pure deterministic domain/game/replay | NOT_STARTED | golden math/wave/boss/vote/XP tests |
| M03 | M01,M02 | mock/replay source, parser, identity, durable input log | NOT_STARTED | spam, dedupe, role and queue tests |
| M04 | M02,M03 | fenced single writer, atomic tick+outbox, snapshots/recovery | NOT_STARTED | crash/replay/privacy/restore tests |
| M05 | M04 | contract-checked API, RBAC, WebSocket, projected state | NOT_STARTED | contract, security, resync tests |
| M06 | M05 | Unity gameplay HUD, all assets, OBS capture | NOT_STARTED | 150 object/FPS/legibility/rights test |
| M07 | M05 | operator dashboard, moderation, deletion and emergency stop | NOT_STARTED | admin security and E2E |
| M08 | M06,M07 | 10k viewer load, chaos, profiling, SAST and UX QA | NOT_STARTED | measured SLOs and proofs |
| M09 | M08 | P0 standalone MOCK/REPLAY release and recovery | NOT_STARTED | all P0 acceptance PASS |
| M10 | M09,LIVE permission | authorized TikTok comment adapter | BLOCKED | provider contract and real integration |
| M11 | M10,legal/OBS/owner | LIVE production rehearsal and release | BLOCKED | real 2h moderated broadcast + approvals |

M00 W0001 cross-section contradiction audit; W0002 test-vector catalog; W0003 verify official version releases; W0004 dependency DAG; W0005 requirement-to-test traceability; W0006 baseline engineering approval.
M01 W0101 Go executable and tests; W0102 PG18.6 image and Compose; W0103 ALL schema/migrations/indexes/sqlc (before M03); W0104 Unity pinned editor and tests; W0105 TS operator lockfiles; W0106 GitHub Actions unit/race/integration/Unity/security; W0107 Windows+Linux tooling; W0108 secrets templates.
M02 W0201 integer domain/types; W0202 phase and tick state; W0203 per-wave role and cooldown; W0204 anchor and three-lane HP pools/boss; W0205 attack/guard/builder/upgrade; W0206 vote; W0207 loss/reward/XP; W0208 100 golden seed/crash/replay fixtures.
M03 W0301 Unicode exact grammar; W0302 mock/replay event adapter; W0303 scoped HMAC/ephemeral identity; W0304 provider ID duplicate handling; W0305 durable monotonic event input; W0306 valid/invalid queue/drop metrics; W0307 10k unique mock participants; W0308 event fuzz tests.
M04 W0401 writer epoch fencing; W0402 committed state+tick+outbox; W0403 canonical snapshots and replay; W0404 SQL crash restart; W0405 PITR/backup restore; W0406 privacy erasure/backup tombstones; W0407 event retention.
M05 W0501 OpenAPI 3.1 schemas; W0502 JSON WS schemas; W0503 admin auth, CSRF and RBAC; W0504 full/patch/public snapshot and resync; W0505 expiring token handling; W0506 TS/C# typed clients and negative tests.
M06 W0601 centered castle, wall and lanes; W0602 pooled <=150 entities; W0603 builder/guard/fighter, enemies and boss animations; W0604 all source/phase screens and HUD; W0605 production original/right-cleared sprite/audio manifest; W0606 OBS 9:16 platform overlay screenshots; W0607 15min FPS+GC; W0608 human 3sec understandability.
M07 W0701 secure operator UI; W0702 start/pause/resume/stop; W0703 health/lag/queue metrics and alerts; W0704 name filtering/block/hide; W0705 erasure and evidence audit; W0706 emergency stop/recovery and admin E2E.
M08 W0801 1k events/s 10min + 5k/s burst10s with10k actors; W0802 backend crash, PG outage, reconnect; W0803 fencing split-brain attempt; W0804 SAST/SBOM/secrets/license audit; W0805 Go/DB profiling; W0806 Unity OBS real capture; W0807 full acceptance audit.
M09 W0901 build pinned images; W0902 mock/replay release visibly labeled; W0903 backup/restore and rollback; W0904 100% P0 evidence audit; W0905 changelog/known limitations.
M10 W1001 approved account event access and permission proof; W1002 authorized adapter; W1003 real reconnect/ID/delivery semantics; W1004 owner-consented live E2E; W1005 policy/privacy verification.
M11 W1101 real OBS/LIVE preview; W1102 human moderation readiness; W1103 >=2h real broadcast rehearsal; W1104 disaster rollback drill; W1105 legal/operator release signoff.

Execution loop: inspect latest main and CI; choose first unblocked NOT_STARTED/FAIL milestone with dependencies PASS; implement code, tests and evidence; do not change combat math without recorded revision; update progress only with proof; commit directly main after checks; confirm CI; fix/revert if failed. Never pretend a missing approval is PASS.
---
# 10. EXTERNAL GATES AND HONEST LIMITS
GATE-LIVE: proof of permitted official or expressly authorized TikTok LIVE comment access for actual account, event schema, stable ID, rate limits, reconnect/cursor, consent and current terms. No scraping workaround.
GATE-OBS: real streamer hardware and actual LIVE overlay/preview tests.
GATE-LEGAL: privacy basis, content/asset rights, provider terms and responsible operator.
GATE-HUMAN-UX: independent 5 participants, 3-second muted phone test, >=4/5 correct command+threat.
GATE-PERF: dedicated specified reference host/Unity workstation and measured FPS, latency, loss, RPO/RTO.
External gates require real accountable person and documented evidence. P0 may finish while LIVE gate BLOCKED, but human UX/asset rights/real device evidence cannot be forged. An AI can work through independent engineering tasks automatically; it cannot guarantee fully human-free TikTok LIVE production deployment.
---


---

# 11. TECH LEAD, GAME AND PSYCHOLOGY AUDIT
This is a structured simulated 30-perspective review, NOT interviews with 30 actual experts. Perspectives: 10 Tech Leads (architecture, storage, distributed systems, SRE, security, database, networking, testing, CI, API), 10 senior game programmers (simulation, pacing, economy, balance, rendering, UX, animations, content production, live operations, fairness), 10 psychology specialists (attention, cognitive load, autonomy, belonging, fairness, social proof, curiosity, motivation, social dynamics, return habit). The actual evidence is independent document inspection and model reasoning, not a clinical/consumer study.

## Findings and dispositions
| ID | Severity | Previous gap / contradiction | Binding master resolution | Validation |
|---|---|---|---|---|
| F01 | BLOCKER | Role contribution once per 20s only if commented again; rewarded spam / contradicted anti-spam | One valid join/wave, role auto-contributes each interval | QA-CMD-02 |
| F02 | BLOCKER | Assault 180..270, finale 270..290 versus 20s resolution allowed boss zero or ambiguous combat | Exact integer ticks, boss t2600 and 2 opportunities t2600/2800 | QA-SIM-03 |
| F03 | BLOCKER | Fixed enemy count independent of 1/10/10k participation can make tiny lives unwinnable | Transparent population-anchored integer HP pools, low-P militia, capped scaling | QA-BAL-01 |
| F04 | BLOCKER | M03 required SQL upserts before M04 created schema | Persistence schema and migrations built before ingest integration | QA-MAT-01 |
| F05 | BLOCKER | State log/outbox described but outbox missing table and tick atomicity unclear | Typed outbox and game checkpoint together in one transaction | QA-DB-05 |
| F06 | BLOCKER | Replay hash had no canonical field list/version/encoding | RFC8785 canonical game state and fixture | QA-DET-01 |
| F07 | BLOCKER | No clear source rights; API might not be authorized | External authorization gate, no scraping, mock complete | QA-LIVE-01 |
| F08 | HIGH | XP per 20-second action encourages repeat commands | 10 XP once/wave; cosmetics only | QA-PSY-02 |
| F09 | HIGH | Vote timing and invalid-options/ties undefined in edge cases | t800..1199, closing t1200, deterministic fallback | QA-VOTE-01 |
| F10 | HIGH | World progress level / tier / construction costs unclear | Fixed progression and cost/time formulas | QA-ECO-01 |
| F11 | HIGH | Missing loss recovery means low viewer permanent spiral | capped adaptive difficulty + defeat recovery | QA-BAL-02 |
| F12 | HIGH | Source connection loss vs pause vs return not defined | fail-closed source contract, frozen tick, recovery protocol | QA-OPS-03 |
| F13 | HIGH | No response/error schemas, compatible WebSocket replay | versioned protocol/authorization in 04_PROTOCOL | QA-API-01 |
| F14 | HIGH | SQL fields absent for resources, membership, checkpoint, event ordering | DDL/properties/spec in 03_PERSISTENCE | QA-DB-01 |
| F15 | HIGH | Missing HMAC key rotation/deletion and anonymous identity rules | 05_PLATFORM_SECURITY policy and data lifecycle | QA-SEC-03 |
| F16 | HIGH | 10k load benchmark targets without machine, queue semantics or no-loss requirements | 08_ACCEPTANCE hardware & SLO, drop/queue policy | QA-LOAD-01 |
| F17 | HIGH | Unity art and screen layouts not sufficiently implementable | concrete asset manifest/UI states/OBS crop in 06_UNITY_CONTENT | QA-UX-01 |
| F18 | HIGH | Main-only workflow conflicted with prior PR/branch instructions/README | one-branch AGENTS contract and updated README | QA-GOV-01 |
| F19 | HIGH | Human signoffs treated as AI capabilities | P0 engineering autonomous vs P1 external human gates | QA-GATE-01 |
| F20 | HIGH | No production-ready moderation/streamer recovery plan | 05 security + 07 operations manual workflows | QA-SEC-05 |
| F21 | MEDIUM | Visual unit counts mixed with real damage actors | lane subpools are authority; visual objects bounded | QA-VIS-01 |
| F22 | MEDIUM | Luck/seeded tie rules ambiguous for replay | no game RNG; lexicographic vote tie | QA-DET-02 |
| F23 | MEDIUM | 'latest' dependencies not pinned exact when scaffolding | fixed tested baseline + CI dependency lock | QA-GOV-02 |
| F24 | MEDIUM | Inconsistent metric accepted-to-published vs viewer-visible | separate transport latency SLO and OBS preview target | QA-OBS-01 |
| F25 | MEDIUM | No data rights/IP on generated sprites/audio | licensed or original-only asset manifest & provenance | QA-ASSET-01 |
| F26 | MEDIUM | 3-second comprehensibility merely prose | UX rubric, sample captures, human measurable approval | QA-PSY-01 |
| F27 | MEDIUM | Gameplay too long inactive in prep | countdown+signaled non-fabricated building goals and active HUD | QA-UX-02 |
| F28 | MEDIUM | No name abuse, self-harm language or harassment suppression | safe alias fallback and hide/ban moderation | QA-SEC-06 |
| F29 | MEDIUM | No clean simulation-stop when chat empty | WAITING state between waves | QA-SIM-07 |
| F30 | MEDIUM | Unspecified update process for external policy changes | periodic review/fail closed if expired | QA-GATE-02 |

## Important limits
These are *resolved in design* to the extent specified in master. They have NOT been implemented, tested against TikTok, profiled on production host, or approved by a real operator. All QA IDs remain NOT_STARTED until live evidence. The phrase '100% autonomous production TikTok release' is not a truthful guarantee; permissions, legal, deployment access, art ownership, audience evaluation, stream rights and moderation require human ownership. P0 mock engineering may be automated after baseline approval.
