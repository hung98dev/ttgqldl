# MULTIDISCIPLINARY FINAL DESIGN AUDIT — 30 SIMULATED PERSPECTIVES
Date: 2026-11-03. Scope: TTGQLDL docs/plan/00–11 on main. This is 30 *reasoning lenses*, NOT 30 actual hired experts, 10,000 real surveyed viewers, or a measured gameplay/UX study. The author inspected contracts and performed a separate exploratory integer-combat simulation. Audit findings are non-normative; **the binding fixes live only in each topic-owned module**. Never interpret "resolved in specification" as tested implementation.

## Executive finding
Previous plans contained first-tick state ambiguity, future-wave event loss under global input cursors, uninitialized admin/village setup, premature victory before boss spawn, role-switch/20-second timing conflicts, confusing cooldowns, absence of provider stable identity requirements, small-audience boss imbalance, untested Unity/OBS UX, and unprovable claims of unconditional automation. These issues have been addressed as design decisions and test IDs. M00 round 2 (below) adds the issues found by actually building the executable validators. No executable production backend/Unity/CI tests exist yet. M01–M09 are NOT_STARTED and P1 requires real account/provider/legal/human proof.

## Ten senior Tech Lead perspectives
| ID | Specialty | Severity | Gap found | Binding design disposition | Proof not yet obtained |
|---|---|---|---|---|---|
| TL01 | System architecture | CRITICAL | No precise single-owner contract for game state | Go authoritative reducer; Unity projected/visual only (01/03/05) | QA-DET-01, QA-API-02 |
| TL02 | PostgreSQL DBA | CRITICAL | Future-wave seq50 can be skipped after current-wave seq51 | Per-event applied_at_tick and wave_no; exact transaction order (04) | QA-DB-06 |
| TL03 | Distributed systems | CRITICAL | Crash and stale writer can double rewards or mutate after takeover | DB-clock writer_epoch fencing, tick-state+outbox atomicity (04) | QA-DB-05/07, QA-OPS-03 |
| TL04 | API architect | HIGH | Start needs village_id but new install had no create-village route | Admin village create/list, bootstrap secret, idempotency and negative cases (05) | QA-API-03 |
| TL05 | Application security | CRITICAL | No stable provider ID permits fake new viewers each comment | Real LIVE fails closed without stable identity/authorized source (06) | QA-SEC-07 |
| TL06 | Privacy engineer | HIGH | Restore may resurrect erased players from event log | New privacy checkpoint, protected deletion tombstones and restore verification (04/06) | QA-DB-08, QA-SEC-03 |
| TL07 | SRE | HIGH | Source outage and manual pause collide, resumption unclear | Sorted independent blockers; no wallclock catchup, operator resume (03/08) | QA-OPS-04 |
| TL08 | Performance engineer | HIGH | 1k/s serialized DB row lock and giant 10k-member snapshots each tick | <=256/50ms ingress batches, compact tick state and bounded public view (04) | QA-LOAD-01 |
| TL09 | CI/Release | HIGH | A written SLO or design was treated like a PASS | Evidence/CI and BLOCKED external gates; M00 IN_PROGRESS (09/10) | QA-GOV-01/02/03 |
| TL10 | Protocol/network | HIGH | No production handshake details and stale diff rules | Per-session signed scoped WS subprotocol, base_revision, heartbeat <=2s (05) | QA-API-04 |

## Ten senior Game Programmer perspectives
| ID | Specialty | Severity | Gap found | Binding design disposition | Proof not yet obtained |
|---|---|---|---|---|---|
| GP01 | Game loop | CRITICAL | WAITING had no valid initial sim tick; 0 and -1 mismatch | Initial StartWave(-1,0), 3000 rollover control (02/03/04) | QA-SIM-06 |
| GP02 | Combat | CRITICAL | Tick2800 phase transition could skip final boss hit | Final combat first, settlement once, RESULTS 20s (02/03) | QA-SIM-03 |
| GP03 | Boss system | CRITICAL | Early CLEAR before scheduled boss | BOSS_INCOMING until spawn2600, result at2800 (02/03/07) | QA-SIM-08 |
| GP04 | Balancing | HIGH | 10/30-person balanced team sometimes loses boss mathematically | Fighter attack 18*isqrt(F); integer normal/base boss pools (02/03) | QA-BAL-03/04 |
| GP05 | New-player fairness | HIGH | Militia P<=3 cliff when 4th player joins | Published assist for P<=10 with linear falloff, no fake viewers (02) | QA-BAL-01 |
| GP06 | Progression | HIGH | Consecutive win count could exceed DB check; total wins not incremented | cap at10, total_wins++ and level formula (02/04) | QA-ECO-01 |
| GP07 | Input/game command | HIGH | Role change cooldown was described inconsistently | First switch immediate, next >=600 ticks, one membership (02/03/09) | QA-CMD-04 |
| GP08 | Event/replay | CRITICAL | Role entered at RESULTS might require a second comment to start next wave | Auto-promote next-wave membership at rollover (02/03/04) | QA-SIM-04/07 |
| GP09 | Rendering | HIGH | 10k game objects vs aggregate game state unclear | <=150 pooled representative sprites, one world snapshot, no per-viewer combat object (07) | QA-VIS-01 |
| GP10 | Content pipeline | HIGH | Placeholder character art could be accepted as final | Original/licensed full role/boss animation manifest + 360px capture (07) | QA-ASSET-01 |

## Ten psychology and LIVE audience-experience perspectives
| ID | Lens | Severity | Gap found | Binding design disposition | Proof not yet obtained |
|---|---|---|---|---|---|
| PSY01 | Attention | HIGH | Casual viewer cannot decode a cluttered scene in 3 seconds | One castle HP / threat / role CTA hierarchy (07) | QA-PSY-01 |
| PSY02 | Cognitive load | HIGH | Multiple commands and votes shown with equal weight | One primary join CTA; A/B/C only during vote window (02/07) | QA-UX-01 |
| PSY03 | Agency | HIGH | A single comment might not visibly matter | Counts update at next committed projection, no fabricated feedback (05/07) | QA-UX-03 |
| PSY04 | Belonging | MEDIUM | Not everyone can get a visible named avatar at 10k | Fair bounded spotlight rotation, aggregate participation display (07) | QA-VIS-01 |
| PSY05 | Fairness | HIGH | Repeating comments or gifts could dominate | One join per wave, one XP award, votes per stable actor, no paid combat (02/06) | QA-CMD-02, QA-PSY-02 |
| PSY06 | Curiosity | HIGH | 160 seconds prep may look inactive | Truthful 20/30s progress, 15s threat warning, scheduled boss suspense (07) | QA-UX-03 |
| PSY07 | Return motivation | MEDIUM | Losing could erase everything and stop return | Persistent cosmetics/village, bounded recovery without humiliation (02) | QA-SIM-05 |
| PSY08 | Late arrivals | HIGH | RESULTS join counted too late and forgotten | Next-wave queue auto-start, immediate truthful NEXT-WAVE feedback (02/03/07) | QA-SIM-07 |
| PSY09 | Social safety | HIGH | On-video nicknames enable harassment or identity exposure | Sanitization, alias opt-out, privacy hide and actor lookup (05/06/07) | QA-SEC-06 |
| PSY10 | Ethical engagement | CRITICAL | Excessive compulsion/spam/false crowd feedback | No forced repeat comments, no fake numbers, no paid strength, respectful CTA (01/02/07) | QA-PSY-03 |

## Exploratory numeric sanity check (NOT a validated game build)
A separate integer approximation using wave5 boss, difficulty d=1, no prior losses and starting full village state observed that for 25% builders /25% guards /50% fighters (rounded to integer actor counts), P=10 and P=30 would leave ordinary+boss HP 3 and 34 respectively under old fighter coefficient14, but leave 0 under coefficient18. Candidate formula baseline is therefore 18*isqrt(F) in authoritative 02/03, and QA-BAL-03 now requires expected ordinary/boss HP vectors: P10 162/33; P30 238/48. This exercise detects a mathematical risk, not measured player sentiment, production balance certification or a Go acceptance PASS. Recompute using the actual Go reducer and independently implemented test oracle; balance may still change ONLY with rules_version and golden fixtures updated together.

## Residual external blockers versus code completeness
- "Resolved by design": major technical contradictions are given explicit deterministic contracts, not implemented source.
- "Must prove by running": Go/DB/Unity implementation, load 10k, 1k/s, game determinism, FPS, security, privacy restoration and build reproducibility.
- "Cannot be self-proven by an agent": actual TikTok API/account rights, copyrights and privacy legal approval, real OBS device overlays, five independent UX study participants, staff moderation and public-release authorization.
- "Current audit status": M00 IN_PROGRESS, M01–M09 NOT_STARTED, M10–M11 BLOCKED. An agent can implement an excellent P0 mock product against the modular plan without reinventing the game, but no written plan can *guarantee* a human-free lawful P1 TikTok deployment.

## Audit governance
Future review adds rows or updates disposition IN THIS file only. Every newly discovered issue gets ID, severity, owning module, exact rule change and automated/manual QA ID; never create an alternative master/v3 plan. All 30 simulated reviewer roles are analytical perspectives, not identity claims or genuine external endorsements.

## Final incremental security/engine findings
TL06 extension: historical database backups predating a deletion cannot know later tombstones stored only inside the restored DB. The design now requires an independent durable encrypted deletion ledger and fail-closed restoration in 04/08, with QA-SEC-08. GP09 extension: Unity 6000.6.7f1 release notes list repeated LoadSceneAsync hang risk; the BroadcastWorld scene remains loaded and overlays display reconnect/recovery, with QA-VIS-02. These are design remediations requiring executable tests, not evidence of completed operations.

GP02 final audit: tick2800 now explicitly belongs to the final combat calculation before the RESULTS phase, rather than being excluded by the generic results-phase rule. GP07/TL02 final audit: first_switch_used and privacy_epoch are mandatory persisted columns, not inferred dynamically. Tests QA-SIM-10 and QA-DB-09 remain unexecuted until M01/M02.

## M00 round 2 — findings produced by building the validators (2026-11-03)

This round differs from the 30-lens review: it comes from executing `tooling/planvalidate` and `tooling/refeval` against the plan. It is still NON-NORMATIVE; the binding text lives in the owning modules. Severity is BLOCKER (cannot implement deterministically), HIGH (contradiction or unverifiable requirement), MEDIUM (ambiguity that would cause divergence), LOW (editorial).

| ID | Severity | Owning module | Finding | Disposition | Tests |
|---|---|---|---|---|---|
| M00-A01 | BLOCKER | 04/03 | NEXT-wave membership had no `joined_tick` rule, so a RESULTS join stored with the RESULTS tick would fall outside the next anchor window and undercount P | Pending rows persist `wave_no+1`, `joined_tick=0`, `last_change_tick NULL`; promotion keeps role and needs no second comment | QA-DB-10, QA-SIM-07 |
| M00-A02 | BLOCKER | 03/02 | The tick total order never awarded XP, so XP timing, defeat interaction and replay idempotency were undefined | Explicit award step inside the 200-tick boundary, before economy and before any defeat effect, once per member per wave, cap 300 | QA-SIM-11, QA-DET-01 |
| M00-A03 | BLOCKER | 01/04 | `rules_version` and canonical `schema_version` were mandatory in the schema and hash but were never given values, so implementers had to invent identifiers | `ttg-rules-v1` and `schema_version=1` defined in 01 and repeated in 02/03/04/09 | QA-GOV-02, QA-DET-01 |
| M00-A04 | BLOCKER | 02/03 | Attack remainder order ("lane order" vs cyclic) and whether the boss is a redistribution target were ambiguous; the P=30 boss vector's outcome depends on it | One exact four-part algorithm: cyclic lanes from combat_index%3, priority runner/grunt/brute/boss, round-2 redistribution over ALL living pools including the boss | QA-SIM-02, QA-BAL-03 |
| M00-A05 | BLOCKER | 04 | The canonical hash omitted session XP and the boss-spawn transition, so a duplicate-XP crash would hash identically and QA-DB-05 could not be proven | `session_xp` added per member and `boss_spawned` added to the hashed field list | QA-DET-01, QA-DB-05 |
| M00-A06 | HIGH | 04/06 | Alias limit was 128 bytes in the schema but 64 Unicode graphemes in the adapter; Vietnamese aliases of 64 graphemes can exceed 128 bytes, so accepted aliases could fail INSERT | Grapheme rule remains the admission rule; storage backstop is `octet_length(alias)<=256` | QA-SEC-06 |
| M00-A07 | HIGH | 02 | Vote fallback ("next highest votes, otherwise C") was ambiguous and the zero-ballot default's resource charge was unstated | Deterministic selection algorithm with explicit affordability, tie order and charged default | QA-VOTE-01 |
| M00-A08 | HIGH | 02 | Settlement order between penalties, vote-C halving, defeat floors and resource caps was unspecified, and non-defeat losses were unclear | Explicit victory and loss sequences; floors apply only on defeat, after penalties | QA-ECO-01, QA-SIM-05 |
| M00-A09 | HIGH | 04 | `input_receipts.provider_event_id` was nullable inside its unique key, making provider dedup decorative | Column NOT NULL; receipts only for stable ids; explicit bounded TTL cache otherwise | QA-CMD-02, QA-DB-01 |
| M00-A10 | HIGH | 04/03 | `input_events.kind` and domain event types were unenumerated, so migrations and the WS mapping could diverge | `kind IN ('role','vote')` plus a canonical domain-event list | QA-DB-01, QA-API-02 |
| M00-A11 | HIGH | 05 | No audit-read or erasure-status route existed, so M07 could not prove moderation or deletion | Added `GET /admin/v1/audit` and `GET .../privacy/erase/{request_id}` with RBAC and bounds | QA-API-05, QA-SEC-03 |
| M00-A12 | MEDIUM | 02/04 | Boss pool creation was described as "at 2600" but state hydration and hashing needed to know it does not exist before then | `boss_hp=0` and `boss_spawned=false` before tick2600; only `ceil(base_hp/5)` is derivable | QA-SIM-03, QA-SIM-08 |
| M00-A13 | MEDIUM | 09 | Cross-reference named `07_OPERATIONS`, a module that does not exist; the baseline host is in 08 | Corrected to `08_OPERATIONS.md`; the link checker now guards it | QA-GOV-03 |
| M00-A14 | MEDIUM | 10 | M10/M11 dependencies were prose ("LIVE permission", "legal/OBS/owner"), so the DAG could not be machine-checked | Dependencies are `GATE-LIVE`, `GATE-OBS`, `GATE-LEGAL` from 11_EXTERNAL_GATES | QA-MAT-01 |
| M00-A15 | MEDIUM | 02/09 | QA-BAL-03 said "win at tick2800" while the P=10 vector actually kills the boss at 2600; the P=30 vector kills it only at 2800 | Both vectors stated explicitly with wall/resource/XP values; P=30 is the QA-SIM-10 regression | QA-BAL-03, QA-SIM-10 |
| M00-A16 | MEDIUM | 02 | Measured balance characteristics (non-monotonic required fighter share) were not recorded anywhere, inviting an undocumented coefficient change | Recorded as accepted characteristics with the measured curve; no coefficient change made | QA-BAL-05 |
| M00-A17 | LOW | 12 | "UE/OBS UX" misnamed the engine | Corrected to Unity/OBS | - |
| M00-A18 | LOW | AGENTS.md | The instruction sentence was truncated ("... not PASS. in 10_IMPLEMENTATION_MATRIX.md") | Rewritten to a complete rule | QA-GOV-01 |

### Round 2 residual, not resolved by design
- Local toolchain is Go 1.27.1 while the baseline pins 1.27.3; no PostgreSQL 18.6 server is installed on this workstation; Unity was not launched. Those are environment gaps recorded as BLOCKED evidence items, not as PASS.
- `consecutive_wins>0` with `consecutive_losses>0` is unreachable in play; sweep rows that combine them are arithmetic stress tests only.
- Nothing in this round constitutes a LIVE permission, legal, asset-rights, OBS or human-UX approval.
