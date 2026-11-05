# ART-008 — Final Audiovisual Integration, Polish & Production Handoff

- **Package ID:** `ART-008`
- **Art Direction:** `Bastion Dawn v1` (Stylized Fantasy Village Defense)
- **Target Engine:** `Unity 6000.6.7f1` — Built-in 2D (`1080x1920` portrait reference, `Orthographic Size = 5.0`, `PPU = 192`)
- **Scope:** Cross-package audiovisual vertical-slice integration, contract alignment, synchronization proofs, isolated Unity GPU showcase, bounded performance verification, and definitive `M06` production handoff.
- **Rights Status:** `PENDING_REVIEW` across all `244` canonical assets (`Music` strictly **MUTED BY DEFAULT** at `-80.0 dB`, `mute = true` until `GATE-LEGAL` clearance).

---

## 1. What ART-008 Proves

`ART-008` unites the five completed production packages (`ART-003`, `ART-004`, `ART-005`, `ART-006`, `ART-007`) into a single verified, reproducible audiovisual presentation vertical slice without duplicating upstream master trees or touching authoritative backend/M06 production directories:

1. **Zero-Duplication Cross-Package Asset Audit (`244` Canonical Assets + `6` Packed Atlases):**
   - `ART-003` (`18` Builder frames, `18` Grunt frames, `6` Repair Spark VFX frames, `1` `1024x1024` Pilot Atlas).
   - `ART-004` (`88` Guard/Fighter/Runner/Brute/Boss frames, `24` combat VFX frames, `1` `2048x2048` Master Atlas).
   - `ART-005` (`26` canonical environment sprites + `14` parapet/transition companion slices, `4` Townhall bands, `5` wall states, `4` lighting palettes, `2` `2048x2048` Atlases).
   - `ART-006` (`20` text-free UI icons, `15` nine-slice skins, `16` canonical states + `8` combined overlays, `59` `vi-VN`/`en-US` strings, `2` `1024x1024` UI Atlases).
   - `ART-007` (`12` original short SFX, `1` seamless ambient loop, `2` seamless music loops in `48 kHz / 24-bit` WAV + runtime `.ogg`, repaired `ttg-audio-dedupe-v2` cue contract, `15/15` bit-identical deterministic WAV rebuild).
2. **Repaired Audio Integration Contracts (`Phase 2` Findings A–E):**
   - Parameterized `SFX01` (`Role Join`) anti-spam test verifying `PREP = 2,000 ms` (`5` plays / `10s`), `ASSAULT/FINALE = 3,500 ms` at `-5.5 dB` (`3` plays / `10s`), and `0` plays while priority `Alerts` (`SFX07..12`) are active.
   - Formalized **Two-Tier Presentation Deduplication (`ttg-audio-dedupe-v2`)**: Tier 1 (`256`-entry `{session_id}:{stream_seq}` ring + monotonic cursor) suppresses exact reconnect/outbox redeliveries while allowing all independent repeated combat/damage/repair/slam events; Tier 2 (`{session_id}:{wave_no}:{milestone}`) applies strictly to genuine once-per-wave milestones (`vote_closed`, `boss_incoming`, `boss_spawned`, `wave_result`).
   - Full `15/15` WAV deterministic rebuild verification (`0` byte differences across `12` SFX + `AMB01` + `MUS01` + `MUS02`).
3. **Isolated `Unity 6000.6.7f1` Hardware GPU Vertical Slice & Bounded Load Benchmark:**
   - Verified in a clean temporary `Unity 6000.6.7f1` project outside `unity/` with real hardware `Direct3D12` rendering on `Intel(R) UHD Graphics 770` + `Intel(R) Core(TM) i5-14600K`.
   - Verified all `7` character classes at exact `1080p` target heights (`Builder 48px`, `Guard 48px`, `Fighter 48px`, `Grunt 48px`, `Runner 44px`, `Brute 64px`, `Boss 102px`), `6` world sorting layers (`Ground`, `LanesAndDecals`, `BastionBack`, `Units`, `BastionFront`, `VFX`) + `ScreenSpaceCamera` 9-slice HUD, `0` scene reloads across state transitions, `12` preallocated pooled audio voices, and `10 / 50 / 100 / 150` pooled entity benchmarks (`261.66 FPS` avg at `150` entities, `0 B` managed GC allocation per frame).
4. **Synchronized H.264 + AAC (`48 kHz`) Multi-Resolution Showcase & Truthful Wave/Failure Timelines:**
   - Complete `300s` (`ticks 0..3000`) **Wave 10** showcase covering `WAITING (P=0)` -> first join -> `PREP` construction -> even-wave `VOTE_OPEN [800..1199]` -> `PREP_60_WARN (1000)` -> `VOTE_CLOSED (1200)` -> `PREP_15_WARN (1450)` -> `ASSAULT (1600)` -> combat -> `BOSS_INCOMING (2450)` -> `FINALE + Boss spawn (2600)` -> `RESULTS_WIN settlement (2800)` -> `Wave 11 rollover (3000)` with visual changes every `<= 20s` (`< 30s` ceiling).
   - Eight separate deterministic failure, backpressure, and reconnect sequences (`failure_01..08`).

---

## 2. Package Directory Structure

- `README.md` — Package overview and reproducible commands.
- `APPROVAL_RECORD.md` — Technical integration sign-off vs open external release gates.
- `AUDIOVISUAL_DIRECTION.md` — Unified visual, motion, and acoustic direction lock.
- `CROSS_PACKAGE_AUDIT.md` — Automated audit summary across `ART-003..007`.
- `UNITY_PRODUCTION_HANDOFF.md` — Step-by-step `M06` import, sorting, pooling, and adapter blueprint.
- `PRESENTATION_EVENT_CONTRACT.md` — Exact mapping from `simulation.Event` & `05_API_WEBSOCKET.md` to presentation cues and open `M05`/`M06` dependencies.
- `INTEGRATION_LIMITATIONS.md` — Honest boundary statement (`M05`, `M06`, `GATE-LEGAL`, `GATE-OBS`, human UX study).
- `unified_asset_manifest.json` — `244` canonical SHA-256 asset entries + `6` packed atlases + dependency lock matrix.
- `presentation_timeline.json`, `animation_vfx_audio_sync.json`, `hud_transition_mapping.json`, `audio_dispatch_policy.json` — Machine-readable presentation contracts.
- `showcase/scene_source/` — Portable C# runtime components (`BroadcastAudioDispatcher.cs`, `SpriteTransitionController.cs`, `HudInPlacePresenter.cs`) and isolated Editor builder (`ART008UnityShowcaseVerifier.cs`).
- `showcase/isolated_unity/` — Serialized `BastionDawn_ART008_VerticalSlice.unity`, `BroadcastAudioDispatcher.prefab`, `scene_hierarchy_manifest.json`, and 3 hardware `1080x1920` GPU captures.
- `showcase/fixtures/` — `full_wave_300s_wave10_showcase.json` + `failure_01..08_*.json` (all marked `"synthetic_fixture": true`).
- `previews/` — Synchronized H.264/AAC `.mp4` clips and `.png` proof boards across `audiovisual/`, `animation_transitions/`, `hud_transitions/`, `full_wave/`, `mobile/`, `grayscale/`, and `compression/`.
- `qa/` — `qa_report.md`, `qa_metrics.json`, `negative_test_results.json`, `unity_isolated_verification.json`, `performance_report.json`, and `rights_audit.json`.
- `tools/` — `validate_cross_package.py`, `run_unity_showcase.py`, `showcase_contracts.py`, `build_showcase.py`, and `validate_art008.py`.

---

## 3. Reproducible Commands

```powershell
# 1. Cross-package SHA-256 & metadata audit (ART-003..ART-007)
python -B art/production/ART-008/tools/validate_cross_package.py

# 2. Rebuild isolated Unity 6000.6.7f1 GPU showcase & 10/50/100/150 entity benchmark
python -B art/production/ART-008/tools/run_unity_showcase.py

# 3. Rebuild all contracts, synthetic fixtures, proof boards, and H.264+AAC MP4 clips
python -B art/production/ART-008/tools/build_showcase.py

# 4. Run full 20-positive + 20-negative acceptance QA suite
python -B art/production/ART-008/tools/validate_art008.py --write-reports

# 5. Run zero-dependency CI-fast check (<3s, pure Python 3 stdlib)
python -B art/production/ART-008/tools/validate_art008.py --ci-fast
```
