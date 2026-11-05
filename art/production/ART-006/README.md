# ART-006 — Production UI/HUD, 16-State Screen System & Unity Integration Handoff

| Field | Value |
|---|---|
| **Work Package** | `ART-006` |
| **Art Direction** | `Bastion Dawn v1` (`#11151C` dark-ink outlines, 2-step `45°` top-left cel shading) |
| **Target Engine** | `Unity 6000.6.7f1` (`Built-in 2D` / `com.unity.ugui 2.6.0` / `TextMeshPro`) |
| **Target Unity Art Path** | `unity/Assets/_TTGQLDL/Art/UI/` (`M06` remains `NOT_STARTED`) |
| **35 Production UI Assets** | `20` `128×128` RGBA32 icons + `15` 9-slice/tiled RGBA32 skins (`31` reused byte-identically from `ART-002`, `4` surgically repaired) |
| **Production UI Atlases** | `2` deterministic `1024×1024` RGBA32 atlases (`4.00 MiB` each = `8.00 MiB` total uncompressed, `0` rotation, `0` alpha bleed) |
| **Localization** | `vi-VN` (default) + `en-US` (secondary), `59` keys each, `0` missing Vietnamese diacritics, `0` bundled proprietary font binaries |
| **HUD State Coverage** | `16` canonical states (`hud/states/`) + `8` compositional overlay combinations (`hud/overlays/`) + `16×3` multi-res proofs (`1080p`, `720p`, `360p`) |
| **Automated & Adversarial QA** | `PASS` (`0` text overflows, `>= 12.3:1` WCAG contrast, `16/16` adversarial negative cases rejected, isolated `Unity 6000.6.7f1 -batchmode` `PASS`) |
| **Rights Status** | `PENDING_REVIEW` (`GATE-LEGAL`, `GATE-OBS`, `M05`, `M06`, and human `QA-PSY-01` remain unfabricated and open) |

---

## 1. Package Architecture & Entry Points

- **Governance & Provenance:** [`APPROVAL_RECORD.md`](APPROVAL_RECORD.md)
- **Component Kit & 9-Slice Spec:** [`UI_DESIGN_SYSTEM.md`](UI_DESIGN_SYSTEM.md) & [`ui_component_manifest.json`](ui_component_manifest.json)
- **16-State & Priority Rules:** [`HUD_STATE_MAPPING.md`](HUD_STATE_MAPPING.md) & [`ui_state_mapping.json`](ui_state_mapping.json)
- **Unity 6000.6.7f1 Handoff:** [`UNITY_INTEGRATION_GUIDE.md`](UNITY_INTEGRATION_GUIDE.md) & [`unity_presets/`](unity_presets/)
- **Localization & Layout Tokens:** [`localization_vi.json`](localization_vi.json), [`localization_en.json`](localization_en.json), [`ui_layout_tokens.json`](ui_layout_tokens.json)
- **Synthetic Fixtures (24):** [`fixtures/`](fixtures/) (`16` canonical + `8` overlay combinations; zero `ActorKey`/private fields)
- **QA Evidence & SHA-256 Manifest:** [`qa/qa_report.md`](qa/qa_report.md), [`qa/qa_metrics.json`](qa/qa_metrics.json), [`qa/adversarial_validator_results.json`](qa/adversarial_validator_results.json), [`qa/unity_isolated_verification.json`](qa/unity_isolated_verification.json), [`asset_manifest.json`](asset_manifest.json)

---

## 2. Reproducible Build & Validation Commands

From the repository root (`ttgqldl/`):

```powershell
# 1. Audit/repair the 35 UI assets, pack 2x 1024x1024 atlases, write contracts, fixtures & 9-slice/typography stress proofs
python -B art/production/ART-006/tools/build_ui_assets.py

# 2. Render all 16 HUD states, 8 overlay combos, 1080p/720p/360p multi-res proofs, grayscale & real FFmpeg H.264 stream tests over ART-005
python -B art/production/ART-006/tools/build_hud_previews.py

# 3. Run independent mechanical/domain validator + all 16 adversarial negative cases (and refresh SHA-256 manifest if --write-manifest)
python -B art/production/ART-006/tools/validate_art006.py
```

---

## 3. Scope & Honesty Boundaries

- **No Premature Runtime Claims:** `M04` is in parallel backend development, `M05` is not started, and `M06` remains `NOT_STARTED` in [`docs/plan/10_IMPLEMENTATION_MATRIX.md`](../../../docs/plan/10_IMPLEMENTATION_MATRIX.md). No live WebSocket binding or production scene mutation was performed inside `unity/`.
- **Simulated Platform Overlays Are QA-Only:** Any right/bottom mobile chrome shown in `proof_16` or safe-zone boards is synthetic QA geometry (`[QA MOCK PLATFORM OVERLAY — NOT OFFICIAL TIKTOK UI]`) and is never part of runtime HUD assets.
- **No Unlicensed Third-Party Assets or Audio:** Zero font binaries and zero audio/SFX files are committed in `ART-006`.
