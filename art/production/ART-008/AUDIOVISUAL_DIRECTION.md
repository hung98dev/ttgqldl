# ART-008 — Bastion Dawn v1 Unified Audiovisual Direction

## 1. Visual Hierarchy & 3-Second Mobile Viewer Comprehension

Every composed `1080x1920` frame (and its `720x1280` / `360x640` portrait stream downscales) enforces a strict 3-tier reading order so a viewer scrolling onto the livestream understands the game within 3 seconds — even with phone audio completely muted:

1. **Tier 1 — Fortress & Wall Condition (Top Status + Center Citadel):**
   - High-contrast 9-slice Wall HP (`TƯỜNG`) and Townhall HP (`THÀNH`) bars paired with the physical `ART-005` Bastion Wall (`state1_healthy` -> `state2_damaged` -> `state3_critical` -> `state4_breached` -> `state5_recovery`) and 4 Townhall progression bands (`Lv 1–9`, `10–24`, `25–39`, `40–50`).
2. **Tier 2 — Active Threat & Countdown Timer (Top-Right Capsule + Threat Banner + 3 Lanes):**
   - `MM:SS` countdown (`>= 14.33px` at `360p`) + color/icon-coded Threat Banner (`navy` PREP, `amber` 60s warn / Assault / Pause / Reconnect, `crimson` 15s warn / Boss / Storage Error / Confirmed Drops, `emerald` Victory) + West, Center, and East tactical lanes converging on the gate.
3. **Tier 3 — Single Dominant Viewer Action (Unified Lower Command Dock):**
   - Exactly ONE primary call-to-action plaque (`48px` at `1080p` = `16.0px` at `360p`):
     `Bình luận XÂY / THỦ / ĐÁNH`
   - Zero competing primary CTAs, zero fake viewer touch buttons, and strict clearance of the right `18%` (`x <= 885.6px`) and bottom `12%` (`y <= 1689.6px`) platform chrome rails.

---

## 2. Character Scale, Perspective & Sorting Harmony

All world assets share a locked isometric/top-down `3/4` Bastion Dawn perspective (`Orthographic Size = 5.0`, `PPU = 192`, `FilterMode.Point`, zero camera pan/zoom/shake):

| Class / Layer | Source Cell | Approved `visual_root_scale` | Displayed Tight Body @ `1080p` | Sorting Layer |
|---|---|---|---|---|
| **Builder** (`ART-003`) | `128x128` | `0.55814` | `48.0 px` | `Units` (`-round(worldY*100)`) |
| **Guard** (`ART-004`) | `128x128` | `0.55172` | `48.0 px` | `Units` (`-round(worldY*100)`) |
| **Fighter** (`ART-004`) | `128x128` | `0.55814` | `48.0 px` | `Units` (`-round(worldY*100)`) |
| **Grunt** (`ART-003`) | `128x128` | `0.55814` | `48.0 px` | `Units` (`-round(worldY*100)`) |
| **Runner** (`ART-004`) | `128x128` | `0.54321` | `44.0 px` | `Units` (`-round(worldY*100)`) |
| **Brute** (`ART-004`) | `128x128` | `0.66667` | `64.0 px` | `Units` (`-round(worldY*100)`) |
| **Boss Warlord** (`ART-004`) | `256x256` | `0.57955` | `102.0 px` | `Units` (`-round(worldY*100)`) |
| **Boss Slam Telegraph** (`ART-004`) | `256x256` | `0.68000` | Ground rune under feet | `LanesAndDecals` (`order = 24`) |
| **Combat / Repair VFX** (`ART-003/004`) | `128x128` | `0.54–0.58` | Impact point attachment | `VFX` (`auto-clear after final frame`) |

---

## 3. Temporal Polish, Accessibility & Acoustic Balance

- **Animation Cadence:** Locked at `12 FPS` (`83.333 ms` per sprite step) with deterministic non-interpolated frame stepping (`Idle -> Move -> Action -> Recovery -> Idle` and `Idle -> Hit -> Idle`).
- **HUD Micro-Animations & `ReducedMotion`:** Standard transitions use `<= 220 ms` subtle alpha crossfades, `<= 12 px` vertical banner settle, `180 ms` HP bar interpolation, and a single-slot `2.0 s` Join Toast. Every transition has a verified `ReducedMotion` equivalent (`0 px` slide, `1.0` scale, instant/fade-only update).
- **Stream & Photosensitivity Safety:** `0 Hz` strobe (`<= 3 Hz` ceiling), `0 px` camera shake, BT.709 grayscale luminance separation, and complete state legibility when muted.
- **Acoustic Mix Hierarchy:** Priority `Alerts` (`0.0 dB` bus) > Combat `SFX` (`-1.5 dB` bus, with `SFX01` attenuated an additional `-5.5 dB` and throttled to `3,500 ms` during `ASSAULT`/`FINALE`) > `Ambient` (`-3.0 dB` bus, seamless `20.0 s` `AMB01` loop) > `Music` (**locked at `-80.0 dB`, `mute = true` by default**).
