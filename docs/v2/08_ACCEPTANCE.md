# QA / SLO / PROOF — NORMATIVE V2

## Core principles
All test cases MUST be executable and reproducible, or explicitly identified as a manual human test. A spec is never evidence that code works. Every PASS requires SHA + command + fixture + time + environment + log artifact; human-required tests use real human signature. A missing license/device/credential = BLOCKED, not PASS. The required baseline host is specified in 07_OPERATIONS, and any other tested hardware recorded verbatim.

## Mandatory test IDs
QA-GOV-01 V2 is only normative source, main-only branch policy, no hidden PR dependency.
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
