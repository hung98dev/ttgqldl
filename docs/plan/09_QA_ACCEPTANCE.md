# TESTS, SLO, ACCEPTANCE

## Core principles
All test cases MUST be executable and reproducible, or explicitly identified as a manual human test. A spec is never evidence that code works. Every PASS requires SHA + command + fixture + time + environment + log artifact and the active `ttg-rules-v1` rules identifier; human-required tests use real human signature. A missing license/device/credential = BLOCKED, not PASS. The required baseline host is specified in 08_OPERATIONS.md, and any other tested hardware recorded verbatim.

## Mandatory test IDs
QA-GOV-01 docs/plan/00_INDEX.md and its topic-owned modules are the only normative plan, main-only branch policy, no hidden PR dependency.
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
QA-SIM-05 defeat freezes economy/combat and prevents any post-defeat XP; loss penalty, 250 HP recovery, level retention and no overflow/negative.
QA-BAL-01 simulated P=1,3,4,5,10,30,100,1000,10000 at 0/20/25/40/50/60/75/100% fighter mixes, d=1,5,10 and boss/nonboss; battle must end within 300s, no divide by zero, no mathematically unwinnable recurrence after losses with at least one fighter or low-P NPC militia. Every case must be classified as feasible or arithmetic-only: consecutive_wins>0 together with consecutive_losses>0 cannot occur in a real session because a win resets consecutive_losses, so those pairings are stress tests and MUST NOT be reported as player-experience evidence.
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
For each Mxx-Wxxxx: docs/evidence/Mxx-Wxxxx.md starts with a machine-readable block so the plan validator can prove status honesty:

    <!-- evidence
    wp: M00-W0001
    milestone: M00
    status: PASS
    commit: WORKTREE@<40-hex-head>   (or a 40-hex sha)
    head: <40-hex commit the tests were executed on>
    utc: 2026-11-03T12:00:00Z
    environment: <OS, cpu, tool versions>
    requirements: REQ-02-ENEMY,REQ-03-ORDER
    tests: QA-SIM-02,QA-BAL-03
    reviewer: <automated tooling name, or the real human reviewer>
    reviewer_kind: automated|human
    -->

and then records, in prose or tables: files inspected and modified, an exact command list, expected versus actual output, metric summary, log path, outstanding blockers and rerun instructions. `docs/traceability.csv` has exactly these columns: requirement_id,document_anchor,owner_module,work_package,test_id,priority,verification_method,spec_status,impl_status,evidence_uri; document_anchor is `<module>.md#<github-heading-slug>`, test_id lists QA ids, spec_status is VERIFIED|FIXED|BLOCKED, impl_status uses the matrix vocabulary and evidence_uri is a repository path that exists. `tooling/planvalidate` (run from `tooling/` as `go run ./planvalidate -root ..`) fails if any MUST is unmapped, any QA id is missing or duplicated, any anchor does not resolve, or any PASS lacks a matching evidence record. Weekly or on milestones review drift in doc/fixture versions and lint links.

## Required CI stages
Go test ./... + -race, fuzz seeds, sqlc/generation check, goose migrate up/down PG18.6, determinism golden, OpenAPI/JSON schema contract tests, security SAST/secret/SBOM/license, Unity EditMode/PlayMode with a licensed editor (local licensed execution is accepted for M01; CI execution is optional - see "Acceptance-policy revision"), TS UI tests, k6/Go load harness on dedicated runner. Manual OBS, privacy, streamer and legal gates are never spoofed by headless CI.

## Additional binding QA cases for audit gaps
QA-SIM-06 First join on WAITING=-1 initializes tick0 only; 10Hz execution never calls a tick outside [0,2999] for gameplay; rollover tick3000 is a control boundary only.
QA-SIM-07 P=0 WAITING never creates fake viewers, unattended losses or background enemy damage; queued RESULTS players auto-start wave on rollover.
QA-SIM-08 At boss wave all ordinary enemies can die before tick2600, but BOSS_INCOMING remains; boss spawn/combat at2600 then2800, victory banner only after t2800.
QA-SIM-09 DEFEATED before t2800 skips ALL later builder/economy/XP/combat effects; settlement once.
QA-CMD-04 First role switch after joining immediately allowed; next role switch rejected until last successful switch+600 ticks; duplicate messages never reset cooldown.
QA-BAL-03 With d=1, no recovery modifier, wave_no=5 boss, population P=10 and builders2,guards2,fighters6: ordinary base_hp=162, boss_hp=33; fighter attack per combat interval=40 (18*isqrt(6)+militia4); the boss dies at tick2600 and victory is settled at tick2800 with wall_hp=1000, wood=518, stone=324, food=350, session XP 100. For P=30 with builders8,guards8,fighters14: base_hp=238, boss_hp=48, attack=54; the boss SURVIVES tick2600 with 16 HP and is killed only by the tick2800 combat, so victory is settled at tick2800 with wall_hp=1000, wood=544, stone=338, session XP 300. These are EXPECTED mathematical vectors, not completed Go tests; they MUST become golden Go fixtures asserting exact tick-by-tick outcomes, and the P=30 vector is the mandatory regression that proves QA-SIM-10 (excluding tick2800 would flip the result to a loss).
QA-BAL-04 Cross-check reference combat implementation against an independently implemented evaluator for each population and role mix; if any case contradicts spec, mark M02 FAIL and correct math before app/UI.
QA-DB-06 Queue NEXT-wave input_seq50 and later current-wave input_seq51; process current event once, preserve future event for next wave and apply once at tick0 after rollover. Reject use of one global applied-sequence cursor.
QA-DB-07 Writer lease fencing and batch order: source 1k/s through batch<=256/50ms, commit ACK only after persistence; mixed operator pause + source outage + DB outage does not advance tick.
QA-DB-08 Restore after privacy deletion must not recreate alias, player mapping, pending votes or next-wave memberships; verify tombstone ledger and privacy_epoch across snapshot/replay.
QA-API-03 Clean DB bootstrap creates admin then village via POST /admin/v1/villages, lists it, starts session; negative unauthorized/duplicate tests. No imaginary manual database seed.
QA-API-04 Unity/WebSocket handshake accepts Sec-WebSocket-Protocol scoped token, rejects leaked query token, supports diff base_revision and heartbeat stale state; client never displays victory when boss_incoming=true.
QA-SEC-07 Missing stable provider viewer identity makes authorized_live non-playable and P1 BLOCKED; no spoofing based on display name or per-comment synthetic identity.
QA-OPS-04 Operator pause + source loss + reconnect must retain pause; resume only after all distinct blockers cleared, no wallclock catch-up.
QA-UX-03 Every actual 30s section has truthful information, one visible role CTA, no empty/noisy 160s intro, no manufactured activity and no misleading crowd count.
QA-PSY-03 Run P0 heuristic evaluation with 5 uncoached viewers; report observed (not invented) completion for single-comment onboarding and 3-second comprehension. Pilot retention after P1 only if lawful and consented; never present role-played 10,000 audience opinions as real evidence.
QA-GOV-03 Module ownership, cross-links, implementation matrix IDs and QA references resolve. Review file is explicitly non-normative. No additional plans/branches.
QA-GATE-03 Separate completion states: engineering-code-built, automated QA PASS, visual-UX MANUAL PASS, and authorized TikTok LIVE P1 PASS; code completion alone never equals production launch.

## Hard verification policy for an autonomous coding agent
- A work package has a three-part verification checklist: (a) source/deliverables exist and compile, (b) its referenced executable tests pass on pinned toolchain, (c) applicable manual or provider-human gates have attached real evidence. If (c) not available, report CODE_COMPLETE or BLOCKED but never production PASS.
- For M00 the agent MUST cross-check all explicit normative numerical formulas and boundaries with independently computed reference vectors; after M01 it must run automated tests. Review of a plan alone is not evidence of implemented features.
- Release blocker check must include: P0 runnable from clean Windows/Linux checkout without TikTok, correct UI 9:16, no raw identity on video, 10k load within measured SLO, all game simulations deterministic; P1 needs real approved source and observed stream.
- No artificial test success: simulated benchmarks, hypothetical user testing and written assumptions are evidence of design exploration only, NOT measured production FPS, live input throughput or verified audience retention.

## Baseline sanity evidence classification
Exploratory numerical projections in 12_REVIEW_FINDINGS.md are NOT executable game tests and never satisfy QA-BAL-03/04. The first implementation must produce independent Go golden fixtures and deterministic seed/replay tests. The documentation audit has not executed Unity/Golang/PostgreSQL tests yet; matrix status remains IN_PROGRESS for M00 and NOT_STARTED for M01–M09.

QA-SEC-08 Erase participant after full PostgreSQL backup; export tombstone to independent ledger, restore old DB snapshot, apply newer ledger, assert no private account/alias/queued event resurrected; missing external ledger => startup FAILS CLOSED.
QA-VIS-02 Unity 6000.6.7f1 continuous 2h streaming build survives repeated pause/source disconnect/recovery without SceneManager.LoadSceneAsync hang; logs/frame metrics recorded.


QA-DB-09 Schema requires nullable last_change_tick, explicit first_switch_used and sessions.privacy_epoch; first role change immediately permitted, later >=600 ticks. SQL and reducer/golden hash must agree on these persisted fields.
QA-SIM-10 At tick2800 mandatory final combat/repair precedes RESULTS, including 20s role resolution, but if DEFEATED economy/combat is skipped; avoid excluding tick2800 just because its resulting phase is RESULTS.

QA-SIM-11 XP ordering and idempotency: at each 200-tick boundary the reducer awards at most +10 session XP to each active member with xp_awarded=false, awards BEFORE that boundary's combat/defeat effect, never awards after a defeat, never awards twice on replay, and caps at 300; a crash replay must reproduce identical session_xp values in the canonical hash.
QA-DB-10 NEXT-wave membership normalization: a registration accepted during RESULTS is persisted for wave_no+1 with joined_tick=0, last_change_tick NULL and first_switch_used=false, promoted at tick3000 without a second comment, and counted inside the next wave's anchor window (joined_tick<=1599); storing the RESULTS tick as joined_tick is a FAIL.
QA-BAL-05 Required-fighter-share curve must match the independent reference evaluator: boss wave, d=1, no prior losses, no vote, minimum fighters to win = 0 for P=1,3,4; 1 for P=5; 4 for P=10; 9 for P=30; 25 for P=100; 196 for P=1000; 1849 for P=10000. M02 golden fixtures reproduce these exactly; any change requires a rules_version bump and regenerated `tooling/refeval` vectors.
QA-API-05 Admin audit and privacy-erasure status endpoints: audit read is admin-only and paginated <=50 without secrets, actor_key or provider ids; erase status is operator/admin readable, returns queued|applied|blocked|failed, and unauthorized or cross-village access returns 403/404 without leaking existence.
QA-GOV-04 Plan validator: `tooling/planvalidate` exits 0 on the canonical plan and exits non-zero when seeded with any single defect — broken link, duplicate QA id, undefined QA reference, milestone cycle, PASS without evidence, constant drift, or traceability row pointing at a missing evidence file.

QA-OPS-05 Environment and tooling contract: `tooling/doctor` reports each dependency as correct version, missing, incorrect version, unusable install or permission/licence blocked, and exits non-zero on any required gap; `dev-up`, `dev-down`, `backend-run`, `backend-test` and `tooling-test` exist in both PowerShell and Bash, resolve the repository root themselves, propagate a non-zero exit code from every failing step, never delete data without an interactive confirmation, and never modify unrelated installed software.
QA-SEC-09 Configuration contract: startup fails with a field-level message naming every invalid or missing required variable from the 08 register; `TTG_SOURCE=authorized_live` is refused while GATE-LIVE is blocked; stage/prod refuse a plaintext PostgreSQL URL to a non-loopback host and require the privileged secrets; `.env` is gitignored while `.env.example` carries only development placeholders; no secret value is written to logs.
QA-DB-11 Identity alias contract: `actor_identity_aliases` accepts only a 64-character lowercase hex HMAC digest as `actor_key` and a bounded `key_version` token; a mapping must reference a player in the same village through the composite foreign key; duplicate `(village_id, actor_key, key_version)` rows, cross-village mappings and missing players are rejected; erasing a player removes its aliases; rotation appends a new `key_version` for the same player. `privacy_ledger_outbox` rejects an export timestamp without an acknowledgement reference and vice versa, a non-object payload, negative attempts and an unknown tombstone.
QA-DB-12 Migration lifecycle: a database at migration version 3 upgrades to the latest version without losing villages, players or session rows; re-applying migrations is a no-op; the 00004 down migration fails closed while identity mappings or unexported deletion events exist and succeeds on an empty disposable database; a fresh database migrates to the latest version.
QA-DB-13 Schema drift: the committed `backend/db/schema.fingerprint.json` (tables, columns with type/nullability/default/length/precision, constraints and indexes, excluding goose metadata) equals the fingerprint read from a freshly migrated database, `backend/db/schema.sql` lists exactly the migrated tables, and `sqlc diff` reports no generated-code drift.
QA-SEC-10 Least-privilege database CI: `ttg_owner` is a non-superuser that runs migrations and holds CREATE on the database; `ttg_app` is a non-superuser with row-level DML only; the application role is denied CREATE TABLE (in `ttg` and `public`), CREATE SCHEMA, ALTER TABLE, DROP TABLE, TRUNCATE, CREATE ROLE, ownership changes, CREATE INDEX and CREATE EXTENSION, and a GRANT issued by it cannot widen PUBLIC access; goose metadata lives in schema `ttg` on both fresh and existing databases.
QA-CLI-01 Unity project foundation: the project opens in the pinned Unity 6000.6.7f1 editor, uses the Built-in 2D pipeline with no URP/HDRP, keeps Force Text serialization and Visible Meta Files, commits a package manifest plus lockfile pinning `com.unity.ugui` 2.6.0 and `com.unity.test-framework` 1.8.0, has a Git-visible `.meta` file for every asset and folder, and defines assembly boundaries in which Core and Presentation reference no engine assembly at all and no runtime assembly references editor-only code. `tooling/unitycheck` validates all of this without a licence.
QA-CLI-02 Portrait HUD layout: the canvas uses a 1080x1920 reference with `CanvasScaler.matchWidthOrHeight = 0.5`; the right 18% and bottom 12% stay reserved for platform chrome; every essential HUD rect (top HUD, world, commands, watermark) is inside that safe rectangle; the same normalised contract holds at 360x640; and a persistent MOCK watermark is visible whenever the data comes from a fixture or replay instead of a live source.
QA-CLI-03 Public-state presentation: the client renders only a validated immutable public snapshot; null, malformed or out-of-range fields are refused and the last accepted view is shown with a visible STALE indicator; CLEAR is never shown while a boss spawn is pending; victory appears only in RESULTS with the boss resolved and the town hall standing; no actor key or provider identifier can reach the screen; mapping identical inputs produces an identical view model; and pause, source loss and storage loss are presentation states that never reload the broadcast scene.
QA-OPS-06 Operator console foundation: `operator/` installs from a committed lockfile on Node 24 LTS (`npm ci`), typechecks under `tsc --noEmit` strict mode, lints with ESLint without errors, passes the Vitest/Testing-Library suite and produces a Vite production build whose bundle contains no credentials, database DSNs or privileged endpoints. The console exposes the six operator sections behind a typed router, persists only the light/dark theme, collapses its navigation below 1024 px, distinguishes loading/empty/error/unavailable states, and calls only `GET /healthz` and `GET /readyz` until M05/M07 endpoints exist.
QA-SEC-11 Operator authorization boundary: no authentication is implemented in the client; the shipped adapter reports `AUTH_UNAVAILABLE` and never fabricates a session, token or cookie. Viewer/operator/admin permissions gate UI affordances only and are never presented as backend authorization. Login never persists credentials, the HTTP client is configured for HttpOnly session cookies plus CSRF headers, and every unimplemented action is marked UNAVAILABLE or BLOCKED in the capability matrix together with its owning milestone.
QA-UX-04 Operator console experience: Vietnamese is the initial language with typed message keys; demo data always carries a persistent DEMO/MOCK banner and is never described as real players, a real stream, a connected source or an authenticated operator; destructive session actions require an explicit confirmation dialog; untrusted display aliases render as text; and navigation, forms and dialogs are keyboard operable with a skip link, visible focus and focus restoration.
QA-CLI-04 Unity result evaluation: the Unity test wrappers reject a result file that reports zero total tests, or a suite where every test was skipped, and never report a platform as passing without executed tests.

## Follow-ups recorded during M01-W0105

QA-SIM-12 (owner M05, consuming M01-W0104) Authoritative wave result: the public snapshot must carry an explicit server-authored wave result (victory/defeat/clear) so the client never infers a win from HP and phase. The Unity `PresentationMapper.VictoryVisible` currently derives victory from `phase == RESULTS`, empty lane pools and a standing town hall; that is a presentation restatement today and must be replaced by the authoritative field when M05 defines it. W0105 records the dependency and does not change the protocol.

QA-SEC-12 Security pipeline: CI runs pinned scanners against the real repository - `govulncheck` for Go advisories, `gosec` for Go static analysis, `npm audit` failing on high/critical runtime advisories, `gitleaks` with a reviewed config, and `actionlint` for workflows - and a missing scanner makes the gate BLOCKED rather than green. Every finding is classified by severity, location, reachability and disposition; critical or high exploitable findings block acceptance; no gate uses `|| true`, blanket ignores or unconditional `continue-on-error`.
QA-OPS-07 Dependency inventory: a deterministic CycloneDX SBOM covers Go modules, npm packages, Unity packages and pinned container images, reports missing licence metadata explicitly instead of inventing it, fails on a container image without an immutable `sha256` digest, and reproduces byte-for-byte for an unchanged checkout.

## Acceptance-policy revision (repository owner, 2026-11-04)

The repository owner approved one change to the M01 Unity acceptance contract:
**licensed Unity EditMode/PlayMode execution inside GitHub Actions is OPTIONAL for
M01.** It was previously mandatory and could not be satisfied because no licensed
runner or protected environment exists.

Mandatory for M01 instead:

1. The Unity project is pinned to Editor 6000.6.7f1.
2. Real licensed local Editor execution shows EditMode and PlayMode passing with zero failures.
3. A real local Windows x64 standalone player build succeeds.
4. The Unity static project validation passes in GitHub Actions (`tooling/unitycheck`).
5. Backend, PostgreSQL, frontend, security, tooling and M00 regression gates pass.

The optional workflow `.github/workflows/unity-editor-tests.yml` is retained
unchanged in intent and has **NOT EXECUTED**. Its truthful state is
`NOT_STARTED / OPTIONAL`; it must never be reported as a passing gate. Future
licensed Unity CI remains a desirable improvement, not an M01 condition.

Evidence rule: a local Unity run is accepted only when the evidence names the
Editor version, the exact tested revision, the executed counts and the build
result, and only when the Unity sources have not changed materially since that
run. A historical run must never be presented as a new run on the latest commit.