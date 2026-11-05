# ART-008 — Production Art & Audiovisual Integration Approval Record

| Field | Value |
|---|---|
| **Work Package** | `ART-008` — Final Audiovisual Integration, Polish & Production Handoff |
| **Art Direction Lock** | `Bastion Dawn v1` |
| **Target Engine Verified** | `Unity 6000.6.7f1` (Built-in 2D, `1080x1920`, `Ortho Size = 5.0`, `PPU = 192`) |
| **Technical Package Status** | **`COMPLETED` (Audiovisual Asset Integration Complete; Canonical Fixture, HUD Truthfulness & Metric Definitions Remediated in `ART-009`)** |
| **Canonical Upstream Packages Audited** | `ART-003`, `ART-004`, `ART-005`, `ART-006`, `ART-007` (`244` assets + `6` atlases, `0` errors) |
| **QA Acceptance Suite** | `20/20` Positive Checks `PASS` • `20/20` Negative Mutation Tests `PASS` (+ `ART-009` `25/25` Canonical `ttg-rules-v1` & `15/15` Negative Checks) |
| **Rights & Provenance Status** | `PENDING_REVIEW` (Music **MUTED BY DEFAULT** at `-80.0 dB` / `mute = true`) |
| **M06 Production Milestone Status** | `NOT_STARTED` (awaiting `M04` completion + `M05` WebSocket API + `M06` scene wiring) |

---

## 1. Approved Technical Deliverables

1. **Cross-Package Asset Integrity:** All `124` character sprite frames, `30` VFX burst frames, `40` modular environment & parapet sprites, `35` UI icons & nine-slice skins, `15` `48 kHz / 24-bit` audio masters (+ `.ogg` runtime exports), and `6` packed RGBA32 atlases are hash-locked in `unified_asset_manifest.json` with zero asset tree duplication.
2. **ART-007 Contract Alignment:** Findings A–E are resolved and verified in both Python DSP/QA (`18/18` checks, `14/14` adversarial cases, `15/15` bit-identical WAV rebuilds) and isolated `Unity 6000.6.7f1` runtime tests.
3. **Synchronization & Motion Polish:** `12 FPS` (`83.33 ms/frame`) character/VFX impact frames align deterministically with `48,000 Hz` stereo SFX onsets (`<= 0.01 ms` PCM placement delta, `<= 20 ms` mux tolerance), clean non-looping action/hit recovery transitions, `0 Hz` strobe (`<= 3 Hz` ceiling), `0 px` camera shake, and full `ReducedMotion` equivalents.
4. **Canonical Fixture & HUD Remediation (`ART-009`):** Post-inspection `ART-008` synthetic fixture discrepancies (`townhall_max = 2000`, `wall_max` tier formula, 3-lane split, `boss_max_hp`, Vote B description, `WAITING` `wave_tick = -1`, hardcoded Unity HUD strings, and batchmode performance metric definitions) are remediated in place and audited in `art/production/ART-009/APPROVAL_RECORD.md` for final `M06` handoff acceptance.

---

## 2. Explicit Unresolved External Gates (Never Claimed as Passed Here)

| Gate / Milestone | Current Status | Owner / Prerequisite |
|---|---|---|
| **`M04` Durable Persistence & Recovery** | `IN_PROGRESS` (Backend Track) | Backend engineering (`PostgreSQL` lease/outbox recovery) |
| **`M05` Public WebSocket & Projection API** | `NOT_STARTED` | Backend engineering (`/ws/v1/public` live snapshot/diff/event projection) |
| **`M06` Production Unity Client Scene** | `NOT_STARTED` | Unity gameplay engineering (wiring `unity/` production client + 15-min soak) |
| **`M07` Operator Console Controls** | `NOT_STARTED` | Operator console + backend auth endpoints |
| **`GATE-LEGAL` Rights & Music Clearance** | `OPEN` (`PENDING_REVIEW`) | Legal/IP reviewer sign-off required before live commercial broadcast or unmuting `MUS01`/`MUS02` |
| **`GATE-OBS` Hardware Capture & Stream Soak** | `OPEN` (`NOT EXECUTED`) | Real OBS `1080x1920` hardware encode + physical smartphone playback verification |
| **External Human UX / Listening Study** | `OPEN` (`NOT EXECUTED`) | 5-person naive viewer 3-second comprehension & listening fatigue review |
