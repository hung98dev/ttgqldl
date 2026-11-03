# TESTS, SLO, ACCEPTANCE

## Core principles
All test cases MUST be executable and reproducible, or explicitly identified as a manual human test. A spec is never evidence that code works. Every PASS requires SHA + command + fixture + time + environment + log artifact; human-required tests use real human signature. A missing license/device/credential = BLOCKED, not PASS. The required baseline host is specified in 07_OPERATIONS, and any other tested hardware recorded verbatim.

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
QA-BAL-01 simulated P=1,3,4,5,10,30,100,1000,10000 at 0/20/40/60/100% fighter mixes, d=1,5,10 and boss/nonboss; battle must end within 300s, no divide by zero, no mathematically unwinnable recurrence after losses with at least one fighter or low-P NPC militia.
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

## Additional binding QA cases for audit gaps
QA-SIM-06 First join on WAITING=-1 initializes tick0 only; 10Hz execution never calls a tick outside [0,2999] for gameplay; rollover tick3000 is a control boundary only.
QA-SIM-07 P=0 WAITING never creates fake viewers, unattended losses or background enemy damage; queued RESULTS players auto-start wave on rollover.
QA-SIM-08 At boss wave all ordinary enemies can die before tick2600, but BOSS_INCOMING remains; boss spawn/combat at2600 then2800, victory banner only after t2800.
QA-SIM-09 DEFEATED before t2800 skips ALL later builder/economy/XP/combat effects; settlement once.
QA-CMD-04 First role switch after joining immediately allowed; next role switch rejected until last successful switch+600 ticks; duplicate messages never reset cooldown.
QA-BAL-03 With d=1, no recovery modifier, wave_no=5 boss, population P=10 and builders2,guards2,fighters6: ordinary base_hp=162, boss_hp=33; fighter attack per combat interval=40 (18*isqrt(6)+militia4). For P=30 with builders8,guards8,fighters14: base_hp=238, boss_hp=48, attack=54. These are EXPECTED mathematical vectors, not completed Go tests. They MUST become golden Go fixtures asserting exact tick-by-tick outcomes and win at tick2800 under clean starting wall/hall/resources.
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
