# ART-006 — Unity 6000.6.7f1 UI Prefab, 9-Slice & Atlas Integration Guide

## 1. Target Engine & Governance Boundary

- **Target Unity Version:** `Unity 6000.6.7f1` (`Built-in 2D`, `com.unity.ugui: 2.6.0` / `TextMeshPro`).
- **Actual Repository Art Root:** `unity/Assets/_TTGQLDL/Art/` (specifically `unity/Assets/_TTGQLDL/Art/UI/` during `M06` integration — **never** create a duplicate `unity/Assets/Art/` tree).
- **Milestone Discipline:** `M04` is in parallel backend development, `M05` is not yet started, and **`M06` remains `NOT_STARTED`**. `ART-006` delivers verified production assets, presets, JSON contracts, synthetic fixtures, and an Editor verification script inside `art/production/ART-006/` without modifying production files in `unity/` prematurely.

---

## 2. Recommended `M06` Target File Mapping Under `unity/Assets/_TTGQLDL/`

When `M06` unblocks, copy or import `ART-006` assets into:

```text
unity/Assets/_TTGQLDL/
  Art/
    UI/
      Icons/                 <- art/production/ART-006/icons/ui_icon_*.png (20 sprites)
      Components/            <- art/production/ART-006/components/ui_comp_*.png (15 9-slice skins)
      Atlas/                 <- ui_atlas_01_icons.{png,json}, ui_atlas_02_components.{png,json}
      Presets/               <- TTG_ART006_UIIcon_128.preset.yaml, TTG_ART006_UISkin_NineSlice.preset.yaml, ART006_HUD_UI.spriteatlasv2.yaml
      Localization/          <- localization_vi.json, localization_en.json
      Contracts/             <- ui_component_manifest.json, ui_layout_tokens.json, ui_state_mapping.json
  Prefabs/
    UI/
      BroadcastHudCanvas.prefab
```

---

## 3. TextureImporter, Filtering & 9-Slice Border Settings

Preset templates provided in [`unity_presets/`](unity_presets/):

1. **Icons (`TTG_ART006_UIIcon_128.preset.yaml`):**
   - `Texture Type = Sprite (2D and UI)`, `Sprite Mode = Single`, `Pixels Per Unit = 100`, `Mesh Type = FullRect`.
   - `Generate Mip Maps = Off` (`m_EnableMipMap: 0`), `Alpha Is Transparency = On`, `Wrap Mode = Clamp`.
   - `Filter Mode = Bilinear` (all 20 icons have hard binary `0/255` alpha with `>=10 px` transparent padding, so `Bilinear` downscales `128->38 px` smoothly with zero halo; `Point` is also clean).
   - `Compression = None (RGBA32)` (`1.31 MiB` uncompressed across 20 standalone `128×128` icons, or `4.00 MiB` in `1024×1024` `ui_atlas_01_icons.png`).
2. **9-Slice Component Skins (`TTG_ART006_UISkin_NineSlice.preset.yaml`):**
   - `Texture Type = Sprite (2D and UI)`, `Sprite Mode = Single`, `Pixels Per Unit = 100`, `Mesh Type = FullRect`.
   - `Generate Mip Maps = Off`, `Alpha Is Transparency = On`, `Wrap Mode = Clamp`.
   - `Filter Mode = Point` (preserves crisp `1 px` cel bevels and exact `8 px` tile periods on `warn`/`crit` HP fills).
   - **CRITICAL — Border Component Ordering:** Unity `TextureImporter.spriteBorder` expects **`Vector4(left, bottom, right, top)` (`L, B, R, T`)**. Read `unity_sprite_border_lbrt` directly from `ui_component_manifest.json`.
   - **HP Fill Setup:** Place `WallHpFill` / `CastleHpFill` inside a child `RectTransform` with **`RectMask2D`**. Use `Image.Type.Sliced` for `ui_comp_hp_bar_fill_healthy`, and `Image.Type.Tiled` for `ui_comp_hp_bar_fill_warn` and `ui_comp_hp_bar_fill_crit` so the `8 px` colorblind dots/hatching never stretch as HP shrinks from `100%` to `0%`.

---

## 4. Prefab-Ready Hierarchy (`BroadcastHudCanvas`)

Aligns directly with existing `HudLayoutConfig.cs`, `SafeAreaPanel.cs`, `PresentationMapper.cs`, and `HudView.cs` in `unity/Assets/_TTGQLDL/Scripts/Presentation/`:

```text
BroadcastHudCanvas (Canvas: ScreenSpaceOverlay | CanvasScaler: ScaleWithScreenSize, 1080x1920, Match=0.5 | GraphicRaycaster)
└── SafeAreaContainer (RectTransform: anchorMin=(0.00, 0.12), anchorMax=(0.82, 1.00) — tunable via SafeAreaPanel)
    ├── TopStatusPanel (Image.Sliced: ui_comp_resource_bar)
    │   ├── WallHpBar (Image.Sliced: ui_comp_hp_bar_frame)
    │   │   ├── FillClipMask (RectMask2D) -> WallHpFill (Image.Sliced/Tiled: healthy/warn/crit)
    │   │   └── WallHpLabel (TextMeshProUGUI: >=43px, "TƯỜNG 1120/1600")
    │   ├── TimerCapsule (Image.Sliced: ui_comp_resource_bar + ui_icon_timer + TMP >=43px)
    │   ├── CastleHpBar (Image.Sliced: ui_comp_hp_bar_frame + FillClipMask + TMP >=37px)
    │   ├── WavePlaque (Image.Sliced: ui_comp_resource_bar + TMP >=37px)
    │   ├── SourceConnectionStatus (Image.Sliced: status_pill_ok/warn/err + icon + TMP >=36px)
    │   └── ResourceStrip (HorizontalLayoutGroup: WoodCell, StoneCell, FoodCell — TMP >=36px)
    ├── ThreatBanner (Image.Sliced: status_pill_err/warn/ok or resource_bar + contextual icon + TMP >=37px + BossHpSubBar)
    ├── ErrorOrPauseDiagnosticOverlay (Image.Sliced: status_pill_warn/err — Militia / Backpressure / Coexisting Blockers)
    ├── SpotlightRibbon (Image.Sliced: ui_comp_resource_bar — bounded <=12 truncated public aliases)
    ├── JoinToast (Image.Sliced: ui_comp_toast_join — single slot, 2.0s hold, burst aggregation)
    └── LowerCommandDock (Scrim Image + Vertical Stack)
        ├── VoteOverlay (Image.Sliced: ui_comp_vote_card_open / locked — subordinate strip)
        ├── PrimaryCommandCTA (Image.Sliced: ui_comp_cta_banner + TMP 48px "Bình luận XÂY / THỦ / ĐÁNH")
        ├── RoleCounterRow (HorizontalLayoutGroup, spacing=9: RoleBuilder, RoleGuard, RoleFighter)
        └── MockReplayWatermark (Image.Sliced: ui_comp_status_pill_warn — active whenever source_status != connected)
```

---

## 5. Reproducible Isolated Unity Batchmode Verification

Reference C# Editor script: [`unity_presets/ART006HudPrefabVerifier.cs.txt`](unity_presets/ART006HudPrefabVerifier.cs.txt).  
Measured output record: [`qa/unity_isolated_verification.json`](qa/unity_isolated_verification.json).

 Note on batching/draw calls: Both `1024×1024` RGBA32 atlases (`4.00 MiB` each, `8.00 MiB` combined uncompressed) are verified in Unity Editor via `Profiler.GetRuntimeMemorySizeLong`. Production runtime frame-time and batch-count claims remain deferred until `M06` scene integration and `GATE-OBS` hardware profiling.
