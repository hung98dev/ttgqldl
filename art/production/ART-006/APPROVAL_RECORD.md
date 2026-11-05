# ART-006 — Production Approval, Provenance & Rights Record

| Field | Value |
|---|---|
| **Package ID** | `ART-006` |
| **Title** | Production UI/HUD, 16-State Screen System & Unity Integration Handoff |
| **Art Direction** | `Bastion Dawn v1` (`#11151C` dark-ink outlines, 2-step `45°` top-left cel shading) |
| **Upstream Approval Authority** | [`art/concepts/ART-002/OWNER_APPROVAL.md`](../../concepts/ART-002/OWNER_APPROVAL.md) (`APPROVED` 2026-04-06) |
| **World & Character Dependencies** | [`ART-003`](../ART-003/README.md), [`ART-004`](../ART-004/README.md), [`ART-005`](../ART-005/README.md) |
| **Target Engine** | `Unity 6000.6.7f1` (`Built-in 2D` / `com.unity.ugui 2.6.0` / `TextMeshPro`) |
| **Target Unity Art Path** | `unity/Assets/_TTGQLDL/Art/UI/` (isolated handoff; `M06` remains `NOT_STARTED`) |
| **Rights & Provenance Status** | `PENDING_REVIEW` (`GATE-LEGAL` remains human-gated) |

---

## 1. Upstream Reuse & Surgical Repair Ledger (35 UI Assets)

Following Phase 1 governance (*"Do not blindly regenerate valid assets; repair only assets failing production QA"*), all 35 approved `ART-002` UI assets were pixel-audited against strict Unity `Image.Type.Sliced` / `Image.Type.Tiled` rules:

| Category | Count | Disposition | Technical Summary |
|---|---:|---|---|
| **UI Icons (`icons/ui_icon_*.png`)** | `20` | `REUSED_BYTE_IDENTICAL` | Exact SHA-256 match with `art/concepts/ART-002/ui/components/ui_icon_*.png`. `128×128` RGBA32, `0` semi-transparent border halo pixels, `>= 10 px` outer margin, `#11151C` outlines, `0` baked text. |
| **Unmodified 9-Slice Skins (`components/`)** | `11` | `REUSED_BYTE_IDENTICAL` | Exact SHA-256 match with `ART-002`: `resource_bar`, `hp_bar_frame`, `role_counter_{builder,guard,fighter}`, `vote_card_{open,locked}`, `status_pill_{ok,warn,err}`, `cta_banner`. Verified `0.0` stretch-strip variance at exact `05_ui_component_kit.md` border insets. |
| **Surgically Repaired Skins (`components/`)** | `4` | `REPAIRED_FROM_ART002` | Repaired strictly in-house via deterministic signed-distance Python geometry (`tools/build_ui_assets.py`) to fix 9-slice distortion and border alpha leaks while preserving `Bastion Dawn v1` palette and silhouettes. |
| **Newly Invented Unapproved Assets** | `0` | `NONE` | Zero unapproved icons, zero fake touch buttons, zero monetization/gift widgets, zero audio files. |

### Detailed Record of the 4 Surgically Repaired Component Skins

Visual before/after proof: `previews/nine_slice/component_repair_before_after.png`.

1. **`ui_comp_toast_join.png` (`520×56`, insets `L=28, R=28, T=22, B=22`):**
   - **Defect in `ART-002`:** Outer capsule corner radius (`r=28`) encroached into column `x=28` on rows `y=0..1` (`top_strip_diff = 239.0`), smearing the top border when stretched in `Image.Type.Sliced`.
   - **Surgical Fix:** Tightened capsule corner radius by `2 px` (`r=24`) so corner curvature finishes cleanly at `x <= 25`, achieving `0.0` strip variance across top, middle, and bottom zones.
2. **`ui_comp_hp_bar_fill_healthy.png` (`368×30`, insets `L=12, R=12, T=10, B=10`):**
   - **Defect in `ART-002`:** Contained 4 hardcoded vertical black segment lines at `x=73, 147, 220, 294` inside the stretchable center zone, which stretched into wide black blocks at larger widths and collided at low HP.
   - **Surgical Fix:** Removed baked segment lines, yielding `0.0` horizontal 9-slice variance and clean `RectMask2D` clipping down to `1 px`.
3. **`ui_comp_hp_bar_fill_warn.png` (`368×30`, insets `L=12, R=12, T=10, B=10`):**
   - **Defect in `ART-002`:** Contained non-periodic vertical bars and wide spaced ovals that smeared under `Sliced` scaling.
   - **Surgical Fix:** Removed vertical bars and aligned high-contrast umber/cream caution dots to an exact **`8 px` horizontal tile period** across `x = 12..355` (`344 = 43 × 8 px`) for seamless `Image.Type.Tiled` + `RectMask2D` clipping.
4. **`ui_comp_hp_bar_fill_crit.png` (`368×30`, insets `L=12, R=12, T=10, B=10`):**
   - **Defect in `ART-002`:** Outer border leaked semi-transparent pixels (`alpha = 210` on outer ring) and diagonal stripes were interrupted by vertical segment bars.
   - **Surgical Fix:** Sealed the `#11151C` outer border at hard `alpha = 255` and aligned `45°` dark-crimson (`#6F1218`) colorblind-redundant hatching to an exact **`8 px` tile period** (`344 = 43 × 8 px`).

---

## 2. Font, Platform Overlay & Rights Governance

- **Typography & Font Licensing:**
  - Zero `.ttf`, `.otf`, `.woff`, or commercial font binaries are bundled inside `art/production/ART-006/`.
  - Offline QA previews and isolated Unity `TMP_FontAsset` verification resolve **`Inter-SemiBold.ttf` / `Inter-Regular.ttf`** (`SIL Open Font License 1.1`) and **`RobotoMono-Bold.ttf`** (`Apache-2.0` / `OFL-1.1`) directly from the local `Unity 6000.6.7f1` Editor installation (`Editor/Data/Resources/Fonts/`), with `Be Vietnam Pro` / `Noto Sans` (`SIL OFL 1.1`) documented as production `M06` TMP asset options.
- **Simulated Platform Overlays (QA Layer 6 Only):**
  - Any right-rail (`x > 82%`) and bottom-rail (`y > 88%`) platform chrome shown in `hud/composites/hud_clean_vs_tiktok_overlay_safe_zone.png`, `hud/mobile/mobile_stream_overlay_360p_720p.png`, and `proof_16_mobile_stream_overlay.png` is Original Synthetic QA Geometry explicitly labeled `[QA MOCK PLATFORM OVERLAY — NOT OFFICIAL TIKTOK UI]`. No TikTok logos, trademarks, or proprietary UI screenshots are used.
- **Audio Exclusion:** Zero SFX or music assets are included in `ART-006`.
- **Rights Status:** Every file entry in `asset_manifest.json` and `ui_component_manifest.json` retains `"rights_status": "PENDING_REVIEW"` until formal human legal sign-off (`GATE-LEGAL`).
