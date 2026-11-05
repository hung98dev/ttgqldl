# ART-004 — Complete Character Animation & Combat VFX Production (`Bastion Dawn v1`)

**Status:** `PASS` (All `5` Remaining Character Classes `88f` + `4` Combat VFX Archetypes `24f` = `112` Production Frames + `2048×2048` Master Atlas + All-`7`-Class Previews & QA)  
**Style Lock:** `Bastion Dawn v1` (Owner-approved in [OWNER_APPROVAL.md](../../concepts/ART-002/OWNER_APPROVAL.md) & [APPROVAL_RECORD.md](APPROVAL_RECORD.md))  
**Milestone & Rights Boundary:** `M06` remains `NOT_STARTED` in [10_IMPLEMENTATION_MATRIX.md](../../../docs/plan/10_IMPLEMENTATION_MATRIX.md); `rights_status` remains `PENDING_REVIEW` (`GATE-LEGAL`). Normative requirement `ART004` in [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md) (`<= 150` pooled entities, `<= 20` VFX bursts) remains distinct from this `ART-004` asset production package.

---

## 1. Deliverable Summary (`112` New Production Frames)

Building on the `ART-003` pilot (`Builder 18f`, `Grunt 18f`, `vfx_builder_repair_spark 6f`), `ART-004` completes the entire `7`-class character roster and `5`-archetype combat/repair VFX suite:

| Asset / Class | Spec | Clips (`@ 12 FPS`) | Count |
|---|---|---|---:|
| `Guard` (`chr_role_guard`) | `128×128` RGBA, sheet `768×384`, `48 px` @ `1080p` | `Idle(4)`, `Walk(6)`, `ShieldBrace(6)`, `Hit(2)` | `18` frames |
| `Fighter` (`chr_role_fighter`) | `128×128` RGBA, sheet `768×384`, `48 px` body @ `1080p` | `Idle(4)`, `Walk(6)`, `SwordSlash(6)`, `Hit(2)` | `18` frames |
| `Runner` (`chr_enemy_runner`) | `128×128` RGBA, sheet `768×384`, `44 px` @ `1080p` | `Idle(4)`, `Run(6)`, `DaggerLunge(6)`, `Hit(2)` | `18` frames |
| `Brute` (`chr_enemy_brute`) | `128×128` RGBA, sheet `768×384`, `64 px` @ `1080p` | `Idle(4)`, `March(6)`, `GauntletCrush(6)`, `Hit(2)` | `18` frames |
| `Boss` (`chr_boss_warlord`) | `256×256` RGBA, sheet `1024×1024`, `102 px` @ `1080p` | `IdleMarch(6)`, `CleaveSlam(8)`, `Hit(2)` | `16` frames |
| `vfx_fighter_slash_arc` | `128×128` RGBA, sheet `512×128` | `Burst(4)` synced w/ `Fighter.SwordSlash` | `4` frames |
| `vfx_guard_shield_block` | `128×128` RGBA, sheet `512×128` | `Burst(4)` synced w/ `Guard.ShieldBrace` | `4` frames |
| `vfx_unit_hit_puff` | `128×128` RGBA, sheet `768×128` | `Burst(6)` synced w/ unit `Hit` | `6` frames |
| `vfx_boss_slam_telegraph` | `256×256` RGBA, sheet `1280×512` | `Burst(10)` (`5f` warn + `3f` impact + `2f` fade) | `10` frames |
| **Total New Production Assets** | **`9` Fixed-Grid Sheets + `1` `2048×2048` Master Atlas** | **`0` halo px, `0` border clip, `11/11` adversarial QA** | **`112` frames** |

---

## 2. Package Map

- [APPROVAL_RECORD.md](APPROVAL_RECORD.md) — Pre-flight audit, owner sign-off, `ART-003` QA hardening summary, and `ART004` vs `ART-004` disambiguation.
- [ANIMATION_PIPELINE.md](ANIMATION_PIPELINE.md) — Deterministic 2D cutout/FK rig architecture, Fighter crimson cloth shoulder conversion, Runner `+X` lock, palette/alpha rules, and clip definitions.
- [UNITY_INTEGRATION_GUIDE.md](UNITY_INTEGRATION_GUIDE.md) — Inspected `unity/Assets/_TTGQLDL/Art/` structure, per-class `VisualRoot` scale table, `2048×2048` `ART004_Master_Atlas` (`16.00 MiB` raw RGBA32), sorting layers, and isolated Unity `6000.6.7f1` proof.
- [qa/qa_report.md](qa/qa_report.md) — Independent mechanical validation (`15/15` checks), `11/11` adversarial negative tests, visual readability (`1080p` / `720p` / `360p`, BT.709 grayscale, H.264 `yuv420p`), and cross-package `7`-class audit.
- `clip_frame_metadata.json` & `asset_manifest.json` — Complete frame bounding boxes, sole coordinates, Unity scales, SHA-256 digests, and `PENDING_REVIEW` rights records.
- `characters/{guard,fighter,runner,brute,boss}/` — Individual `frames/*.png`, fixed-grid `*_sheet.png`, and `rig_debug_sheet.png`.
- `vfx/{fighter_slash_arc,guard_shield_block,unit_hit_puff,boss_slam_telegraph}/` — Individual `frames/*.png` and fixed-grid `*_sheet.png`.
- `atlas/` — `ART004_Master_Atlas.png` (`2048×2048`), `ART004_Master_Atlas.json`, and `unity_presets/`.
- `previews/` — `7` static contact sheets (`contact_sheets/`), `28` `12 FPS` GIF loops (`gifs/`), and `4` `12 FPS` `libx264 yuv420p` MP4 videos (`mp4/`).
- `qa/` — `5` visual QA diagnostic PNG boards, `qa_metrics.json`, `adversarial_validator_results.json`, and `unity_isolated_verification.json`.
- `tools/` — `art004_rig_lib.py`, `build_art004_characters_and_vfx.py`, `build_art004_atlas_and_previews.py`, `validate_art004_qa_and_manifest.py`.

---

## 3. Reproducible Build & Independent QA Commands

From repository root (`python >= 3.11`, `Pillow`, `numpy`, `scipy`, `ffmpeg`):

```powershell
python -B art/production/ART-004/tools/build_art004_characters_and_vfx.py
python -B art/production/ART-004/tools/build_art004_atlas_and_previews.py
python -B art/production/ART-004/tools/validate_art004_qa_and_manifest.py
```
