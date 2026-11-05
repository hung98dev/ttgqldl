# ART-003 Unity 6000.6.7f1 Atlas & Integration Readiness Guide

## 1. Inspected Repository Unity Structure & Ownership Boundary

Inspection of the live Unity project under `unity/` confirms:

- **Unity Editor Version:** `6000.6.7f1` (`unity/ProjectSettings/ProjectVersion.txt`)
- **Actual Project Art Root:** `unity/Assets/_TTGQLDL/Art/` (NOT the generic `unity/Assets/Art/` placeholder)
- **Scene & Camera Setup (`unity/Assets/_TTGQLDL/Editor/SceneFactory.cs`):**
  - `Main Camera` is orthographic (`orthographic = true`) with **`orthographicSize = 5.0f`**.
  - Vertical world height = `2 * 5.0 = 10.0` world units.
  - On the primary `1080×1920` portrait reference (`1920 px` tall), **`1.0` world unit = `192.0` screen pixels** (`5.625` world units wide × `10.0` world units tall).
  - On `720×1280`, `1.0` world unit = `128.0` screen pixels; on `360×640`, `1.0` world unit = `64.0` screen pixels.

Per `ART-003 §6` and [10_IMPLEMENTATION_MATRIX.md](../../../docs/plan/10_IMPLEMENTATION_MATRIX.md), all `ART-003` assets are kept strictly isolated inside `art/production/ART-003/` so `M01–M05` runtime/editor scripts and parallel agent tasks remain untouched until `M06` begins.

---

## 2. Exact On-Screen Size Math (`PPU = 192` vs `48 px` Display Target)

Rather than assuming `PPU = 192` alone dictates on-screen pixel height, we verified exact world-to-screen math in Unity `6000.6.7f1`:

- Inside each `128×128` sprite cell, the standing character body spans **`86 px`** (`y = 32..117`), leaving **`32 px`** overhead (`y = 0..31`) for raised `T-hammer` and `ClubSmash` weapon swings without cell clipping.
- At **`PPU = 192`** and `orthographicSize = 5.0f`:
  - Full `128×128` cell = `128 / 192 = 0.66667` world units (`128.0 px` on `1080×1920`).
  - Unscaled (`localScale = 1.0`) standing character height = `86 / 192 * 192 = 86.0 px` on `1080×1920`.
- To render the exact **`48.0 px` (`1080×1920`) / `32.0 px` (`720×1280`) / `16.0 px` (`360×640`)** gameplay standing target (`0.25` world units) required by [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md):
  - **Recommended `M06` Prefab Option A (Preserves `PPU = 192` Canonical Token):** Import sprites at `PPU = 192` and set the child `VisualRoot.localScale` on unit prefabs (`BuilderVisual`, `GruntVisual`, `RepairSparkVFX`) to **`(0.55814f, 0.55814f, 1.0f)`** (`48 / 86`), while keeping simulation/collider root scale at `(1, 1, 1)`. Cell world bounds become `0.37209 × 0.37209` units and standing height is **`48.00 px`**.
  - **Alternative Option B (`localScale = 1.0`):** Import `128×128` character cells at `PPU = 344` (`86 / 0.25 = 344`), which directly yields `0.25` world units (`48.00 px` at `1080×1920`) at `localScale = (1, 1, 1)`.

---

## 3. Pilot Atlas & Unity Import Preset Specifications

Located under `art/production/ART-003/atlas/`:

- **Packed Texture:** `art/production/ART-003/atlas/Characters_Pilot_Atlas.png`
  - **Dimensions:** `1024×1024 px` RGBA32 (`65.6%` occupancy across `42` cells; well inside the `<= 2048×2048` maximum budget).
  - **Grid & Padding:** `7 columns × 6 rows`, `4 px` transparent gutter on all four sides of every `128×128` cell (`8 px` inter-sprite separation, `pitch = 136 px`) — zero texel bleed.
- **Machine-Readable Slice Table:** `art/production/ART-003/atlas/Characters_Pilot_Atlas.json` (both top-down image rects and Unity bottom-up `rect` + normalized `uv_rect_unity`).
- **Unity `6000.6.7f1` Presets & Reference Script (`art/production/ART-003/atlas/unity_presets/`):**
  - `TTG_CharacterSprite_Point192.preset.yaml`: `TextureImporterType.Sprite`, `SpriteImportMode.Single`, `spritePixelsPerUnit = 192`, `FilterMode.Point` (`0`), `mipmapEnabled = false` (`0`), `alphaIsTransparency = true`, `Uncompressed` (`RGBA32`), `Custom Pivot = (0.50, 0.08)`, `SpriteMeshType.FullRect`.
  - `Characters_Pilot.spriteatlasv2.yaml`: `maxTextureSize = 2048`, `padding = 4`, `filterMode = Point`, `generateMipMaps = 0`, `enableRotation = 0`, `enableTightPacking = 0`, `textureCompression = Uncompressed`.
  - `PilotSpriteImportVerifier.cs.txt`: Drop-in Editor verification helper ready to copy into `unity/Assets/_TTGQLDL/Editor/` during `M06`.

---

## 4. Sorting Layers, Mirroring & VFX Hook Contract for `M06`

1. **Sorting Layer Order (Matches [06_unity_handoff.md](../../concepts/ART-001/06_unity_handoff.md) & `ART-002`):**
   - `00_TerrainBase` -> Meadow, Ochre Lanes, Limestone Causeway
   - `10_GroundTelegraphsAndShadows` -> Soft `#11151C` ground ellipse (`alpha ~ 37%`) anchored at `(0, 0)` of unit `VisualRoot`
   - `20_CastleAndUnits_YSorted` -> `Builder` and `Grunt` `SpriteRenderer`s (`TransparencySortAxis = (0, 1, 0)` or `sortingOrder = Mathf.RoundToInt(-transform.position.y * 100f)`)
   - `30_CombatAndRepairVFX` -> `vfx_builder_repair_spark` (`12 FPS`, non-looping 6-frame one-shot)
   - `50_ScreenSpaceHUD` -> Safe-zone anchored HUD Canvas
2. **Facing & Horizontal Mirroring (`SpriteRenderer.flipX`):**
   - Canonical `Builder` and `Grunt` frames face down-right (`+X` / `3/4` gameplay view).
   - Because normalized pivot X is centered (`0.50` -> `x = 64.0`), setting `spriteRenderer.flipX = true` for East-lane enemies or left-moving courtyard units mirrors cleanly with **`0.0 px` horizontal pivot pop**.
3. **Visual-Only VFX Trigger (`vfx_builder_repair_spark`):**
   - Triggered purely as a client visual callback on `Builder` `Repair` frame index `3` (`t = 250 ms` into the 6-frame `12 FPS` clip) at local offset `(+0.177f, -0.135f)` world units (`+34 px, +26 px` in unscaled `128×128` cell space, scaled by `VisualRoot.localScale`).
   - **Never** drives authoritative wall HP repair ticks, which remain `100%` server/simulation-owned.

---

## 5. Isolated Unity `6000.6.7f1` Verification Status

- **Isolated Scratch Project Batchmode Proof:** **`EXECUTED — PASS`** (`art/production/ART-003/qa/unity_isolated_verification.json`).
  - Executed `C:\Program Files\Unity\Hub\Editor\6000.6.7f1\Editor\Unity.exe -batchmode -nographics -quit` in an isolated scratch project outside the repository (`0` files modified under `unity/`).
  - Verified all **`42/42` frame PNGs** and **`Characters_Pilot_Atlas.png` (`1024×1024`)** import cleanly at `PPU = 192`, `FilterMode.Point`, `mipmapCount = 1` (no mipmaps), `pivot = (64.00, 10.24)` px, unscaled `Sprite.bounds.size = (0.6667, 0.6667)` world units, and `48.00 px` standing display height under an `orthographicSize = 5.0` camera.
- **In-Repo Production Scene (`unity/`) Runtime Integration:** **`NOT EXECUTED (Gated to M06)`**. Production `unity/` scenes and scripts were intentionally left untouched per `ART-003 §6` and `§10`.
