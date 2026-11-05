# ART-004 — Unity `6000.6.7f1` Atlas, Scale & Integration Guide

**Target Engine:** Unity `6000.6.7f1` (`unity/Assets/_TTGQLDL/`)  
**Canonical UX/Art Spec:** [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md)  
**Pilot Integration Baseline:** [ART-003 Integration Guide](../ART-003/UNITY_INTEGRATION_GUIDE.md)

---

## 1. Inspected Repository Unity Layout & Boundary

Inspection of `unity/Assets/_TTGQLDL/` confirms the authoritative folders:
- `unity/Assets/_TTGQLDL/Art/` (currently `.gitkeep` while `M06 = NOT_STARTED`)
- `unity/Assets/_TTGQLDL/Editor/`
- `unity/Assets/_TTGQLDL/Prefabs/`
- `unity/Assets/_TTGQLDL/Resources/`
- `unity/Assets/_TTGQLDL/Runtime/`
- `unity/Assets/_TTGQLDL/Scenes/` (`00_Bootstrap.unity`, `01_BroadcastWorld.unity`, `02_OfflineRecoveryPreview.unity`)
- `unity/Assets/_TTGQLDL/Tests/`

No generic `unity/Assets/Art/` path is used. During `ART-004`, **zero files inside `unity/` were modified** so concurrent backend/Unity verification gates (`tooling/unitycheck`) remain undisturbed.

---

## 2. Orthographic Camera Math & Per-Class `VisualRoot` Scale Table

At reference portrait broadcast resolution `1080×1920` with fixed orthographic camera `orthographicSize = 5.0` (`10.0` world units vertical span) and `Pixels Per Unit (PPU) = 192`:

$$\text{Screen Pixels per World Unit} = \frac{1920}{2 \times 5.0} = 192.0\text{ px/unit}$$

Because `PPU = 192` equals `192.0` screen pixels per world unit at `1080×1920`, a sprite of source pixel height $H_{\text{src}}$ renders at $H_{\text{src}} \times s$ pixels when its `VisualRoot` child transform uses uniform local scale $s$:

| Class (`Package`) | Cell (`px`) | Pivot | Standing Body $H_{\text{src}}$ | `VisualRoot` Scale $s$ | `1080×1920` Height | `720×1280` Height | `360×640` Height |
|---|---|---|---:|---:|---:|---:|---:|
| `Builder` (`ART-003`) | `128×128` | `(0.50, 0.08)` | `86 px` | `0.55814` | `48.0 px` | `32.0 px` | `16.0 px` |
| `Guard` (`ART-004`) | `128×128` | `(0.50, 0.08)` | `87 px` | `0.55172` | `48.0 px` | `32.0 px` | `16.0 px` |
| `Fighter` (`ART-004`) | `128×128` | `(0.50, 0.08)` | `86 px` (`98 px` w/ plume) | `0.55814` | `48.0 px` body (`54.7 px` plume) | `32.0 px` (`36.5 px`) | `16.0 px` (`18.2 px`) |
| `Runner` (`ART-004`) | `128×128` | `(0.50, 0.08)` | `81 px` | `0.54321` | `44.0 px` | `29.3 px` | `14.7 px` |
| `Grunt` (`ART-003`) | `128×128` | `(0.50, 0.08)` | `86 px` | `0.55814` | `48.0 px` | `32.0 px` | `16.0 px` |
| `Brute` (`ART-004`) | `128×128` | `(0.50, 0.08)` | `96 px` | `0.66667` | `64.0 px` | `42.7 px` | `21.3 px` |
| `Boss` (`ART-004`) | `256×256` | `(0.50, 0.08)` | `176 px` | `0.57955` | `102.0 px` | `68.0 px` | `34.0 px` |

---

## 3. Master Sprite Atlas & Memory / Pooling Budget (`Phase 11`)

`art/production/ART-004/atlas/ART004_Master_Atlas.png` packs all `112` `ART-004` sprites into a single deterministic `2048×2048` texture without rotation, tight-mesh distortion, or resolution downscaling:

- **Top Band (`y = 0..1055`):** `26` cells of `256×256` (`Boss` `16f` + `vfx_boss_slam_telegraph` `10f`) with `4 px` padding per cell side (`264×264` pitch = `8 px` inter-sprite gutter).
- **Bottom Band (`y = 1056..2007`):** `86` cells of `128×128` (`Guard 18f`, `Fighter 18f`, `Runner 18f`, `Brute 18f`, `vfx_fighter_slash_arc 4f`, `vfx_guard_shield_block 4f`, `vfx_unit_hit_puff 6f`) with `4 px` padding per cell side (`136×136` pitch = `8 px` inter-sprite gutter).
- **Import Settings:** `FilterMode.Point`, `mipmapEnabled = false`, `alphaIsTransparency = true`, `Uncompressed RGBA32`, `SpriteMeshType.FullRect`, `enableRotation = false`, `enableTightPacking = false`.

| Atlas Texture | Dimensions | Packed Sprites | Disk PNG Size | Raw `RGBA32` GPU Memory |
|---|---|---:|---:|---:|
| `ART-003` `Characters_Pilot_Atlas.png` | `1024×1024` | `42` | `~95 KiB` | `4.00 MiB` (`4,194,304 B`) |
| `ART-004` `ART004_Master_Atlas.png` | `2048×2048` | `112` | `~335 KiB` | `16.00 MiB` (`16,777,216 B`) |
| **Combined Roster (`7` Classes + `5` VFX)** | **`<= 2048×2048` cap** | **`154`** | **`~430 KiB`** | **`20.00 MiB` total** |

### Pooling & Sorting Layer Contract (`M06` Ready)
- **Entity & VFX Pooling (`ART004` normative rule):** Pre-warm `<= 150` `SpriteRenderer` unit instances and `<= 20` pooled VFX burst `SpriteRenderer` instances at scene bootstrap; swap sprite references from pre-loaded arrays at `12 Hz` (`0` per-frame `Resources.Load` or `Instantiate` GC allocations).
- **Sorting Layers:**
  1. `10_GroundTelegraphsAndShadows`: Ground contact shadows and `vfx_boss_slam_telegraph` (`pivot = (0.50, 0.43)`, always rendered underneath character feet; supports reduced-motion mode by holding static warning frame `f02` + single impact frame `f06`).
  2. `20_CastleAndUnits_YSorted`: All `7` character classes using `TransparencySortMode.CustomAxis (0, 1, 0)` anchored at foot pivot `(0.50, 0.08)`; `-X` lane facing uses `SpriteRenderer.flipX = true` (`0.0000` world-unit pivot drift).
  3. `30_CombatAndRepairVFX`: `vfx_builder_repair_spark`, `vfx_fighter_slash_arc`, `vfx_guard_shield_block`, and `vfx_unit_hit_puff`.
- **Authoritative Boss Timing Alignment:** `vfx_boss_slam_telegraph` is decoupled from `chr_boss_warlord` frames so the server-driven Boss warning banner (before spawn tick `2600` on every `5th` wave) and combat slam telegraphs (before final combat tick `2800`) never imply premature damage before the authoritative server impact tick.

---

## 4. Isolated Scratch-Project Unity `6000.6.7f1` Verification Proof

Executed via installed `Unity 6000.6.7f1` (`-batchmode -nographics -quit`) in an isolated scratch project outside the repository, recorded in `art/production/ART-004/qa/unity_isolated_verification.json`:
- Verified all `112` `ART-004` individual sprites + `42` `ART-003` sprites (`154` total) and `ART004_Master_Atlas.png` (`2048×2048`, `112` slices, `16.00 MiB` runtime memory via `Profiler.GetRuntimeMemorySizeLong`).
- Verified `0` gutter bleed pixels across all `112` atlas slices, exact pivots, `0.0000` world-unit `flipX` drift, `12 FPS` `Animator` + `AnimationClip` sampling across every frame of all `7` classes, and sorting-order separation in a multi-class mock scene.
- Note: Static batchmode import and `AnimationClip` sampling do **not** substitute for `M06` live runtime `30 FPS` / `24 FPS 1% low` profiling (`QA-VIS-01`) or hardware `OBS` capture (`GATE-OBS`).
