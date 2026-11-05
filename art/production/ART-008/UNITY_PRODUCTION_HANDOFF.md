# ART-008 — Definitive M06 Unity 6000.6.7f1 Production Handoff

> **Scope Notice:** `ART-008` validates all assets, sorting, scaling, transitions, and pooled dispatchers inside an isolated `Unity 6000.6.7f1` project (`showcase/scene_source/` + `showcase/isolated_unity/`). Production `unity/` remains untouched (`M06 = NOT_STARTED`) until `M05` WebSocket APIs land.

---

## 1. Canonical Asset Import Rules (Zero Duplication)

Import directly from the canonical package directories verified in `unified_asset_manifest.json`:

1. **World, Character & VFX Sprites (`ART-003`, `ART-004`, `ART-005`):**
   - `TextureType = Sprite (2D and UI)`, `PixelsPerUnit = 192`, `FilterMode = Point`, `MipMaps = false`, `sRGB = true`, `AlphaIsTransparency = true`, `isReadable = false` (saves 50% CPU RAM).
   - Apply exact `pivot_normalized` from `unified_asset_manifest.json` (`[0.5, 0.08]` for characters, `[0.5, 0.5]` for VFX, `[0.5, -0.376]` / `[0.5, 0.0925]` for `env_wall_*_back` / `_front` parapet split).
2. **UI Icons & 9-Slice Skins (`ART-006`):**
   - `TextureType = Sprite (2D and UI)`, `PixelsPerUnit = 100`, `FilterMode = Bilinear`, `MipMaps = false`, `sRGB = true`.
   - Apply exact `unity_sprite_border_lbrt` from `art/production/ART-006/ui_component_manifest.json` to all `15` reusable skins (`Image.Type.Sliced` for containers; `Image.Type.Tiled` clipped by `RectMask2D` / anchor width for `ui_comp_hp_bar_fill_*`).
3. **Audio Clips (`ART-007`):**
   - Prefer `.wav` (`48 kHz / 24-bit`) as Editor source with Unity `AudioImporter` settings:
     - `SFX01..12`: `DecompressOnLoad`, `PCM`, `PreserveSampleRate` (`48000 Hz`), `loadInBackground = false`.
     - `AMB01`: `CompressedInMemory`, `Vorbis` (`quality = 0.70`), `loop = true`.
     - `MUS01..02`: `Streaming`, `Vorbis` (`quality = 0.70`), `loop = true`, **keep `AudioSource.mute = true` and `Music_Volume = -80.0 dB` until `GATE-LEGAL` passes**.

---

## 2. Camera, Sorting Layers & Scene Lifecycle

1. **Camera Setup:** Single fixed Orthographic Camera at `(0, 0, -10)`, `orthographicSize = 5.0`, portrait `1080x1920` output (`192` screen pixels per `1.0` Unity world unit). Never pan, zoom, or shake the camera.
2. **ProjectSettings/TagManager.asset Sorting Layers (Bottom -> Top):**
   1. `Default`
   2. `Ground` (`256x256` seamless meadow tiles)
   3. `LanesAndDecals` (3 approach lanes, bastion aprons, and `vfx_boss_slam_telegraph` ground rune)
   4. `BastionBack` (courtyard plaza, `env_wall_state*_back`, Townhall `scale=0.50`, Watchtower `scale=0.54`, 6 Houses `scale=0.33..0.34`, 3 Props `scale=0.36..0.38`, NW/NE scaffolds `scale=0.44`)
   5. `Units` (pooled `<= 150` defenders, enemies, and Boss sorted dynamically via `sortingOrder = Mathf.RoundToInt(-transform.position.y * 100f)`)
   6. `BastionFront` (`env_wall_state*_front` southern parapet & gate arch + SW/SE scaffolds `scale=0.44`)
   7. `VFX` (pooled `<= 20` active bursts + `ScreenSpaceCamera` / `ScreenSpaceOverlay` `1080x1920` HUD Canvas)
3. **Zero-Reload Scene Lifecycle (`Unity 6000.6.7f1` Safety):**
   - Never call `SceneManager.LoadScene` / `LoadSceneAsync` to handle wave rollovers (`tick 3000 -> wave+1 tick 0`), `SOURCE_RECONNECTING`, `PAUSED`, `STORAGE_UNAVAILABLE`, or `MAINTENANCE`. Update all world sprites, pools, and HUD widgets **in place** (`HudInPlacePresenter` pattern).

---

## 3. Ready-to-Port Runtime Components (`showcase/scene_source/`)

| Component | Responsibility | Resource Bound |
|---|---|---|
| `BroadcastAudioDispatcher.cs` | Parameterized `SFX01` cooldowns (`2000ms` PREP / `3500ms @ -5.5 dB` ASSAULT), alert suppression, `ttg-audio-dedupe-v2` Two-Tier dedupe, independent buses, Music `-80 dB` lock | `12` preallocated `AudioSource`s (`0` runtime spawns) |
| `SpriteTransitionController.cs` + `BoundedVfxPool` | `12 FPS` non-interpolated sprite stepping, impact-frame VFX/SFX firing, auto return from non-looping action/hit to `idle`/`idlemarch`, auto-clear VFX | `<= 150` pooled units, `<= 20` pooled VFX bursts |
| `HudInPlacePresenter.cs` | In-place 16-state + 8-overlay HUD switcher with `<= 220 ms` micro-motion and `ReducedMotion` toggle inside `82% x 88%` safe area | `0` scene reloads, `1` single-slot Join Toast |

---

## 4. Remaining M05 / M06 Engineering Steps

1. Implement `/ws/v1/public` client transport in `M05`/`M06` consuming `public_snapshot`, `public_state_diff`, `visual_event`, and `connection_status`.
2. Add a lightweight `ClientCommittedStateDiffTracker` in `M06` comparing `(prev_committed_snapshot, current_committed_snapshot)` so `wall_hp` delta (`>0` repair `SFX02`, `<0` damage `SFX06`, `>0 -> 0` breach `SFX07`) and Boss active attacks (`SFX09`) trigger deterministically from committed public diffs without inventing backend domain events.
3. Run the mandatory `M06`/`M08` 15-minute standalone `1080x1920` `>= 30 FPS` soak and `GATE-OBS` hardware capture test.
