# ART-001 — Deliverable G: Unity 6000.6.7f1 Handoff Specification

**Project:** TTGQLDL (`hung98dev/ttgqldl`)  
**Pinned Engine & Packages (`QA-CLI-01`):** Unity Editor **`6000.6.7f1`**, Built-in 2D Pipeline (no URP/HDRP), `com.unity.ugui` `2.6.0`, `com.unity.test-framework` `1.8.0`, Force Text serialization, Visible Meta Files  
**Handoff Scope:** Concept-to-Engine specification for future `M06` implementation (**`M06` remains `NOT_STARTED`**; `unity/` was inspected read-only and was not modified during `ART-001`)

---

## 1. Machine-Readable Asset Manifest Schema (`QA-ASSET-01`)

Every concept deliverable in [`manifest.json`](manifest.json)—and every future production sprite, tile, icon, VFX sheet, font, and audio asset in `M06`—uses the following mandatory `16`-field JSON object contract:

```json
{
  "asset_id": "ART001-CONCEPT-DIR-A-1080",
  "category": "concept_gameplay_mockup",
  "description": "Direction A (Bright Stylized Fantasy - Bastion Dawn) 1080x1920 full-screen gameplay & anchored HUD concept mockup",
  "style_version": "bastion-dawn-v0.1-proposed",
  "source_path": "art/concepts/ART-001/images/source/world_direction_a_raw.jpg",
  "export_path": "art/concepts/ART-001/images/direction_a_1080x1920.png",
  "dimensions": "1080x1920",
  "format": "PNG (RGB 8-bit)",
  "pivot": "n/a (full-screen concept)",
  "pixels_per_unit": null,
  "frames": 1,
  "fps": 0,
  "atlas": "none (concept_only)",
  "license": "Proprietary-Original / Repository-Owned Concept (Pending GATE-LEGAL Human Review)",
  "provenance": "AI-assisted original 2D concept world via Google Antigravity image-generator + deterministic Python/Pillow anchored HUD compositor; zero third-party IP",
  "review_status": "PROPOSED_PENDING_OWNER_APPROVAL"
}
```

### Field Rules & Honesty Contract
- **Existing Deliverables Only (`ART-001`):** [`manifest.json`](manifest.json) lists **only** the real concept image deliverables actually created and verified on disk in `art/concepts/ART-001/images/`. Zero uncreated `M06` sprite sheets, UI atlases, or audio files (`SFX001`) are listed as delivered.
- **`review_status` Allowed Vocabulary:** `PROPOSED_PENDING_OWNER_APPROVAL` | `APPROVED_CONCEPT` | `PRODUCTION_READY_PENDING_LEGAL` | `CLEARED` | `BLOCKED`.

---

## 2. Future M06 Production Import & Runtime Targets (Locked Architecture)

Once `ART-002` locks the visual direction, `M06` (`W0601–W0608`) must configure Unity `6000.6.7f1` `AssetPostprocessor` / Preset imports to these exact parameters:

### 2.1 Character & Enemy Sprites (`ART001`, `ART002`)
- **Source Frame Dimensions:** **`128 × 128` RGBA PNG** (`8`-bit per channel, straight alpha, outer contour alpha thresholded `>= 224` opaque for clean point sampling).
  - *Boss Sizing Note (`ART-002` Confirmation Item):* To achieve `2.1×` displayed height (`~96–104 px` on `1080 × 1920`) without pixel blockiness, recommend **`256 × 256` RGBA PNG** cells at the same `PPU = 180` inside the `<= 2048 × 2048` Boss/Enemy atlas (16 frames = `1024 × 1024` sheet, well under `2048 × 2048`). If uniform `128 × 128` cells are preferred, import Boss frames at `PPU = 90` (`0.5×` standard `PPU`).
- **Orthographic Camera & Pixels Per Unit (`PPU`) Math:**
  - Existing `SceneFactory.cs` sets `Camera.orthographicSize = 5.0f` (`10.0` world units vertical height = `1920 px` → **`192 px` per world unit**).
  - At `Camera.orthographicSize = 5.0f` and **`Pixels Per Unit = 180`** (or `192` for exact `1:1` texel mapping when `86 px` character height in `128 × 128` is scaled by `0.56×` world scale, or authoring standing hero height at `48–54 px` inside `128 × 128` at **`PPU = 192`** for true `1:1` integer `FilterMode.Point` texel crispness!):
    - **Recommended `PPU = 192` (`1` texel = `1` pixel at `1080 × 1920`)** when sprites are exported at final broadcast display resolution (`~48–54 px` body inside `128 × 128` generous swing-margin cell), **OR** `128 × 128` full-cell hero (`~86 px` tall) downsampled offline via stepped-cel palette quantization to a `64 × 64` / `128 × 128` `PPU = 192` runtime sprite so `FilterMode.Point` never suffers fractional non-integer texel shimmer!
- **TextureImporter Settings:**
  - `textureType = Sprite (2D and UI)`
  - `spriteImportMode = Single` (or `Multiple` sliced grid `128 × 128`)
  - `spritePivot = (0.50, 0.08)` (`Custom` alignment)
  - `filterMode = FilterMode.Point` (no bilinear blur, per `07_UNITY_ART_UX.md` line 23)
  - `mipmapEnabled = false`
  - `alphaIsTransparency = true`
  - `textureCompression = Uncompressed` (RGBA32 inside `<= 2048 × 2048` atlases: one `2048 × 2048` RGBA32 atlas = `16 MiB` VRAM; `3` atlases total = `<= 48 MiB` VRAM, zero DXT/BC7 color-bleed on `3 px` cel outlines!)

### 2.2 Sprite Atlas Budget (`<= 2048 × 2048` Max)
1. `atlas_characters_roles_enemies` (`2048 × 2048` max):
   - `3` Roles × `18` frames (`54`) + `1` Militia × `18` (`18`) + `3` Normal Enemies × `18` (`54`) = `126` frames of `128 × 128` (`11 × 12` grid = `1408 × 1536 px`) + `16` Boss frames (`256 × 256` = `4 × 4` block = `1024 × 1024` in Atlas 2 or packed cleanly in `2048 × 2048`!). Total character + Boss frames fit inside **two `2048 × 2048` atlases** (`padding = 4 px`, `enableRotation = false`, `enableTightPacking = false` so `FilterMode.Point` UV grids stay axis-aligned).
2. `atlas_environment_castle_lanes` (`2048 × 2048` max):
   - Townhall, Watchtower, `5` Wall damage/tier states, `6` houses, `3` decorations, `3` lane modular pieces, `6` terrain tiles.
3. `atlas_ui_and_vfx` (`1024 × 1024` max):
   - `9`-sliced HUD cards, `14` vector-crisp status/resource/role icons (`UI001`), and `5` VFX flipbooks (`ART005`).

### 2.3 Pooling, Scene Stability & Assembly Boundaries
- **Assembly Boundaries (`QA-CLI-01`):** Keep `Ttgqldl.Core` and `Ttgqldl.Presentation` engine-free (`no UnityEngine reference`). Sprite pools, `Animator`/`SpriteRenderer` flipbook drivers, and Canvas bindings live in `Ttgqldl.UI` / `Ttgqldl.Bootstrap` (or a dedicated `Ttgqldl.WorldView` runtime assembly approved in M06).
- **Entity Pool Cap (`QA-VIS-01`):** Pre-warm `150` pooled `SpriteRenderer` GameObjects at `01_BroadcastWorld` startup (`40` Builder, `40` Guard, `40` Fighter, `30` Enemy + `1` dedicated Boss slot; dynamic threat quota never exceeds `150` active; `0` runtime `Instantiate`/`Destroy` allocations during `15`-minute profiler capture).
- **VFX Pool Cap:** Pre-warm `20` pooled VFX flipbook instances; drop/recycle oldest cosmetic spark if `20` are active.
- **No `LoadSceneAsync` Churn (`QA-VIS-02`):** Keep `01_BroadcastWorld` loaded continuously; drive all `16` states (`WAITING`..`STORAGE_UNAVAILABLE`) via `HudViewModel` flags on `HudView` without reloading scenes.
- **Pre-M06 Code Fixes Flagged (Not Edited Here):** Remediate the three M01 skeleton layout items documented in [`04_hud_ux_system.md §2.2`](04_hud_ux_system.md) (`HudLayoutConfig` bottom-up Y conversion, `SafeArea` child double-scaling, and `>= 36 px / 42 px / 48 px` reference Canvas font sizes + `HudLayoutTests.cs` updates) during `M06-W0601` / `W0604`.

### 2.4 Windows Standalone & OBS Capture Output (`07_UNITY_ART_UX.md` line 38)
- **Target Build:** Windows `x86_64` Standalone (`1080 × 1920` portrait windowed/borderless, target `>= 30 FPS` average and `>= 24 FPS` `1%` low on reference capture host).
- **OBS Studio Preset:** Base & Output Canvas `1080 × 1920`, `30` or `60` fps, `Game Capture` / `Window Capture` targeting `TTGQLDL.exe`, Rec.709 color space, H.264 stream preview verified alongside `360 × 640` downscale test (`QA-OBS-01`, `QA-UX-01`).
