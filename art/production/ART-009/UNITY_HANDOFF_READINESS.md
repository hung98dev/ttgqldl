# ART-009 — Final Art-to-M06 Unity 6000.6.7f1 Production Handoff Readiness

## 1. Ready Production Assets & Runtime Adapters (Zero Further Art Packages Required)

All `244` canonical production assets (`ART-003`..`ART-007`), `6` RGBA32 atlases, portable `Unity 6000.6.7f1` runtime scripts, and canonical `ttg-rules-v1` showcase fixtures (`ART-008` + `ART-009`) are verified and ready for `M06` integration:

1. **Characters & Combat VFX (`ART-003` + `ART-004`):**
   - `124` character sprite frames across `7` classes (`Builder 48px`, `Guard 48px`, `Fighter 48px`, `Grunt 48px`, `Runner 44px`, `Brute 64px`, `Boss 102px` at `1080p`, `PPU = 192`, `12 FPS` non-interpolated).
   - `30` combat/repair VFX frames across `5` archetypes (`vfx_builder_repair_spark`, `vfx_guard_shield_block`, `vfx_fighter_slash_arc`, `vfx_unit_hit_puff`, `vfx_boss_slam_telegraph`) with verified auto-clear pooling (`<= 20` active bursts).
2. **Modular 3-Lane Battlefield, Castle & Wall (`ART-005`):**
   - `26` canonical environment sprites + `14` parapet/transition companion slices (`2` `2048x2048` atlases), `4` Townhall bands (`Lv 1..9`, `10..24`, `25..39`, `40..50`), `5` wall damage/recovery states driven by exact `wall_hp / (1000 + 200*(wall_tier - 1))` ratio, and `6` verified sorting layers (`Ground`, `LanesAndDecals`, `BastionBack`, `Units`, `BastionFront`, `VFX`).
3. **16-State + 8-Overlay Localized HUD (`ART-006` + `HudInPlacePresenter.cs`):**
   - `20` text-free icons + `15` nine-slice skins (`PPU = 100`), `59` `vi-VN`/`en-US` strings, `82% x 88%` TikTok safe-rail compliance (`x <= 885.6`, `y <= 1689.6` at `1080x1920`), single primary CTA (`Bình luận XÂY / THỦ / ĐÁNH`), `0` scene reloads across transitions, and `SyntheticCommittedView` binding (`0` hardcoded gameplay counters).
4. **Original 48 kHz / 24-bit Audio & Pooled Dispatcher (`ART-007` + `BroadcastAudioDispatcher.cs`):**
   - `12` original SFX + `1 original seamless 36.00s ambient loop` (`amb_village_dawn_loop.wav`) + `2` seamless music loops (`38.40s` prep, `37.50s` assault; `12` preallocated `AudioSource`s, `PREP` join `2000 ms`, `ASSAULT` join `3500 ms` at `-5.5 dB` cue + `-1.5 dB` `SFX` bus = `-7.0 dB` / `0.4467` linear, `ttg-audio-dedupe-v2` Two-Tier dedupe, `Music` **muted by default** at `-80.0 dB` / `mute = true`, `rights_status = PENDING_REVIEW`).

---

## 2. Explicit Dependencies & Open Gates Before `M06` / `M11` Sign-Off

| Dependency / Gate | Current Status | Required Action & Owner |
|---|---|---|
| **`M04` Durable Persistence (`W0406`, `W0407`)** | `IN_PROGRESS` | Complete `W0406` backup-before-erasure restore + HMAC rotation and `W0407` retention (`DeepSeek Harness`). |
| **`M05` Public WebSocket Projection (`/ws/v1/public`)** | `NOT_STARTED` | Implement `ws-public-v1` snapshot/diff/event endpoints & C# client contract (`DeepSeek Harness`). |
| **`M06` Production `unity/` Client Integration** | `NOT_STARTED` | Port `ART-008/showcase/scene_source/*.cs` into `unity/`, wire live `M05` WebSocket client, and execute 15-min standalone `>= 30 FPS` soak (`W0601–W0608`). |
| **`GATE-LEGAL` IP & Music Clearance** | `PENDING_REVIEW` | Human legal sign-off required before unmuting `MUS01`/`MUS02` or commercial broadcast. |
| **`GATE-OBS` Real Hardware Stream Capture** | `NOT EXECUTED` | Real `OBS 1080x1920 30fps` NVENC capture + physical mobile handset verification (`M06`/`M08`/`M11`). |
| **Human UX & Listening Review** | `NOT EXECUTED` | 5-person naive viewer 3-second comprehension and listening study (`W0608`). |
