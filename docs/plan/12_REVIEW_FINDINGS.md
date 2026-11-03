# REVIEW HISTORY (NON-NORMATIVE): TECH LEAD, GAME AND PSYCHOLOGY AUDIT
Historical findings do not override current topic-owned specifications. This is a structured simulated 30-perspective review, NOT interviews with 30 actual experts. Perspectives: 10 Tech Leads (architecture, storage, distributed systems, SRE, security, database, networking, testing, CI, API), 10 senior game programmers (simulation, pacing, economy, balance, rendering, UX, animations, content production, live operations, fairness), 10 psychology specialists (attention, cognitive load, autonomy, belonging, fairness, social proof, curiosity, motivation, social dynamics, return habit). The actual evidence is independent document inspection and model reasoning, not a clinical/consumer study.

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
