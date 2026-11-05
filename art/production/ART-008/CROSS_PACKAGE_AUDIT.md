# ART-008 — Cross-Package Production Asset Audit (`ART-003` .. `ART-007`)

## 1. Executive Summary

- **Audit Verdict:** `PASS` (`0` missing/corrupt/incompatible asset errors after Phase-2 `ART-007` contract repairs).
- **Canonical Asset Entries Verified by SHA-256:** `244` (`0` duplicated master files in `ART-008`).
- **Packed Atlases Verified:** `6` (`1` `1024×1024` in `ART-003`, `1` `2048×2048` in `ART-004`, `2` `2048×2048` in `ART-005`, `2` `1024×1024` in `ART-006`; `0` gutter bleed pixels).
- **Global Rights Status:** `PENDING_REVIEW` (`GATE-LEGAL` remains open; `MUS01` & `MUS02` remain **muted at `-80 dB` by default**).

---

## 2. Package-by-Package Verification Matrix

| Package | Scope Audited | Verified Counts | Key Compatibility & Repair Findings |
|---|---|---|---|
| `ART-003` | `Builder`, `Grunt`, `vfx_builder_repair_spark`, Pilot Atlas | `36` char frames + `6` VFX frames = `42` PNGs | `12 FPS`, `PPU = 192`, `Builder` (`48px` @ `0.55814`) & `Grunt` (`48px` @ `0.55814`) foot baselines stable (`<=2px` drift); hammer contact at `repair[f03]` (`250.0ms`). |
| `ART-004` | `Guard`, `Fighter`, `Runner`, `Brute`, `Boss`, 4 Combat VFX, Master Atlas | `88` char frames + `24` VFX frames = `112` PNGs | `12 FPS`, `PPU = 192`; approved non-uniform root scales verified (`Guard 0.55172 -> 48px`, `Fighter 0.55814 -> 48px`, `Runner 0.54321 -> 44px`, `Brute 0.66667 -> 64px`, `Boss 0.57955 -> 102px`). |
| `ART-005` | Modular 3-Lane Battlefield, 4 Castle Bands, 5 Wall States, 4 Scaffolds, 4 Palettes | `26` canonical + `14` companion = `40` PNGs + `2` `2048²` atlases | Exact 6 sorting layers (`Ground`, `LanesAndDecals`, `BastionBack`, `Units`, `BastionFront`, `VFX`), `0`px Castle/Wall swap drift, East lane max `x = 873px < 885.6px` (`82%` safe rail). |
| `ART-006` | 20 Icons, 15 Nine-Slice Skins, 16 States + 8 Overlays, VI/EN Localization | `35` UI sprites + `2` `1024²` atlases + `59` VI/EN keys | Single primary CTA `Bình luận XÂY / THỦ / ĐÁNH` (`48px` @ `1080p` -> `16.0px` @ `360p`), `HP/Timer = 43px -> 14.33px`, `Secondary >= 36px -> 12.0px`, `WCAG >= 12.3:1`. |
| `ART-007` | 12 SFX, 1 Ambient Loop, 2 Music Loops, Mixer, Cue Contract & Unity Verifier | `15` WAV (`48kHz/24-bit`) + `15` OGG + `2` MID | **Repaired 5 Phase-2 integration contract gaps** without altering any of the 15 bit-identical WAV masters (see Section 3). |

---

## 3. Confirmed Integration Discrepancies Found & Repaired (Phase 2)

1. **Finding A — Outdated `450ms` Join Cooldown in `ART007AudioPipelineVerifier.cs.txt`:**
   - **Discrepancy:** `ART007AudioPipelineVerifier.cs.txt` simulated 1,000 join events over `1.0s` using an outdated `450ms` cooldown (`3` plays), whereas `audio_cue_mapping.json` specifies `PREP = 2,000ms` (`0.0 dB`), `ASSAULT = 3,500ms` (`-5.5 dB` = `0.5309` linear gain), and complete suppression while any priority `Alerts` cue is active.
   - **Repair:** Replaced with `EvaluateParameterizedJoinBurst(...)` in `ART007AudioPipelineVerifier.cs.txt` and `BroadcastAudioDispatcher.cs`, verifying `PREP 10s = 5 plays`, `ASSAULT 10s = 3 plays @ -5.5 dB`, and `Alerts-active = 0 plays` inside real `Unity 6000.6.7f1`.
2. **Finding B — Reconnect Deduplication vs Legitimate Repeated Events:**
   - **Discrepancy:** Grouping `wall_breached_once_per_breach` or ordinary combat cues under `{session_id}:{wave_no}:{event_type}` risks suppressing legitimate second breaches (after mid-wave Builder repair at 200-tick boundaries per `02_GAME_RULES.md`) or repeated attacks/damages.
   - **Repair:** Formalized `ttg-audio-dedupe-v2` Two-Tier Deduplication:
     - **Tier 1 (`{session_id}:{stream_seq}`, 256-entry ring + monotonic cursor):** Suppresses redelivered outbox duplicates while allowing **all** distinct `stream_seq` combat, repair, damage, Boss slam, and post-repair breach events.
     - **Tier 2 (`{session_id}:{wave_no}:{milestone}`):** Restricted strictly to the 4 genuine once-per-wave invariants (`vote_closed`, `boss_incoming`, `boss_spawned`, `wave_result`).
3. **Finding C — Authoritative `simulation.Event` Payload Sufficiency:**
   - **Discrepancy:** Go `simulation.Event` (`backend/internal/simulation/events.go`) carries `{Type, WaveNo, Tick, Phase, Amount, Boundary, Reason, Stage}` — it does NOT carry `prev_wall_hp`, unit actor IDs, or zero-damage Guard block events.
   - **Repair:** Classified all 15 cues across (1) committed domain events, (2) committed public snapshots/diffs, (3) cosmetic `12 FPS` local scheduling, and (4) explicit `M05`/`M06` state-tracker dependencies in `audio_cue_mapping.json` and `PRESENTATION_EVENT_CONTRACT.md`.
4. **Finding D — Isolated Runtime Event-Driven Audio Dispatcher:**
   - **Repair:** Implemented `BroadcastAudioDispatcher.cs` (`12` preallocated `AudioSource`s = `8` SFX + `2` Alerts + `1` Ambient + `1` Music muted `-80 dB`) and verified it inside the isolated `Unity 6000.6.7f1` showcase scene.
5. **Finding E — Deterministic Rebuild Coverage for All 15 Masters:**
   - **Repair:** Upgraded `ART-007/tools/validate_audio.py` Check `17` from 12 SFX to **all 15 WAV masters** (`12` SFX + `AMB01` + `MUS01` + `MUS02`), proving **0 byte differences (100% bit-identical SHA-256)** on clean synthesis rebuilds.

---

## 4. Remaining Honest Production Dependencies

- **No asset-level blockers remain across `ART-003` .. `ART-008`.**
- Production integration still requires `M04` completion (`W0405–W0407`), `M05` WebSocket projection (`/ws/v1/public`), `M06` production Unity scene wiring (`unity/`), `M07` operator endpoints, `GATE-LEGAL` sign-off, `GATE-OBS` hardware capture test, and real human UX comprehension testing (`NOT EXECUTED`).
