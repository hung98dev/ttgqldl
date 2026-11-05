# ART-003 — Bastion Dawn v1 Sprite Animation Vertical Slice

**Status:** `PASS` (Production Animation Pilot — `Builder` + `Grunt` + `vfx_builder_repair_spark` + `Characters_Pilot_Atlas`)  
**Style Lock:** `Bastion Dawn v1` (Owner-approved in [OWNER_APPROVAL.md](../../concepts/ART-002/OWNER_APPROVAL.md) & [APPROVAL_RECORD.md](APPROVAL_RECORD.md))  
**Milestone Boundary:** `M06` remains `NOT_STARTED` in [10_IMPLEMENTATION_MATRIX.md](../../../docs/plan/10_IMPLEMENTATION_MATRIX.md); `rights_status` remains `PENDING_REVIEW` (`GATE-LEGAL`).

---

## 1. Package Overview & Deliverable Counts

`ART-003` proves that the approved `Bastion Dawn v1` visual direction produces clean, consistent, loopable `12 FPS` 2D animations that survive `48 px` (`1080×1920`), `32 px` (`720×1280`), and `16 px` (`360×640`) gameplay scales, BT.709 grayscale conversion, and `libx264 yuv420p` livestream compression before scaling production to the remaining five character classes.

| Deliverable Category | Exact Count | Specification |
|---|---:|---|
| `Builder` Source Frames | `18` PNGs | `128×128` RGBA, pivot `(0.50, 0.08)`, `Idle(4) + Walk(6) + Repair(6) + Hit(2)` @ `12 FPS` |
| `Builder` Fixed-Grid Sheet | `1` PNG | `768×384` RGBA (`6×3` deterministic grid) |
| `Grunt` Source Frames | `18` PNGs | `128×128` RGBA, pivot `(0.50, 0.08)`, `Idle(4) + Walk(6) + ClubSmash(6) + Hit(2)` @ `12 FPS` |
| `Grunt` Fixed-Grid Sheet | `1` PNG | `768×384` RGBA (`6×3` deterministic grid) |
| `vfx_builder_repair_spark` Frames | `6` PNGs | `128×128` RGBA, golden `#FEF08A` 4-point sparks + oak chips, `0` end-frame bright px |
| `vfx_builder_repair_spark` Sheet | `1` PNG | `768×128` RGBA (`6×1` deterministic grid) |
| **Total Production Frames / Sheets** | **`42` Frames + `3` Sheets** | Binary alpha (`0` halo px), `0` border clipping, `<= 16` character colors |
| Unity Pilot Sprite Atlas | `1` PNG + `1` JSON + `3` Presets | `1024×1024` RGBA32 (`<= 2048×2048`), `4 px` padding, `FilterMode.Point`, `PPU = 192` |
| `12 FPS` Loop Previews | `11` GIFs + `4` H.264 MP4s | Every individual clip, synced `Builder + Spark`, and `1080p` / `720p` / `360p` gameplay |
| Contact Sheets & QA Diagnostic Boards | `3` Contact + `2` Rig + `5` QA PNGs | Evaluable directly without opening Unity |

---

## 2. Directory Structure

- [APPROVAL_RECORD.md](APPROVAL_RECORD.md) — Base SHA inspection, `8/8` owner approval record, `PENDING_REVIEW` rights note, and `M06` boundary.
- [ANIMATION_PIPELINE_AND_CLEANUP_RULES.md](ANIMATION_PIPELINE_AND_CLEANUP_RULES.md) — Controlled 2D cutout/FK rig methodology, locked `16`/`15`-color palettes, `#11151C` contour rules, and mass-production rejection gates.
- [UNITY_INTEGRATION_GUIDE.md](UNITY_INTEGRATION_GUIDE.md) — Inspected `unity/Assets/_TTGQLDL/Art/` structure, `orthographicSize = 5.0` vs `PPU = 192` screen-pixel math (`VisualRoot` scale `0.55814` -> exact `48.0 px`), sorting layers, `flipX`, and isolated Unity `6000.6.7f1` batchmode proof.
- [qa/qa_report.md](qa/qa_report.md) — Complete programmatic & visual QA report covering all `12` consistency checks and `10` mandatory failure tests.
- `clip_frame_metadata.json` — Per-frame bounding boxes, ground-contact `Y`, opaque pixel counts, and `12 FPS` clip definitions.
- `asset_manifest.json` — Verified SHA-256 hashes, byte sizes, pixel dimensions, canonical sources, and `PENDING_REVIEW` rights status for every file in `ART-003`.
- `characters/builder/` — `frames/chr_role_builder_*.png` (`18` files), `chr_role_builder_sheet.png` (`768×384`), `rig_debug_sheet.png`.
- `characters/grunt/` — `frames/chr_enemy_grunt_*.png` (`18` files), `chr_enemy_grunt_sheet.png` (`768×384`), `rig_debug_sheet.png`.
- `vfx/builder_repair_spark/` — `frames/vfx_builder_repair_spark_f00..05.png` (`6` files), `vfx_builder_repair_spark_sheet.png` (`768×128`).
- `atlas/` — `Characters_Pilot_Atlas.png` (`1024×1024`), `Characters_Pilot_Atlas.json`, and `unity_presets/` (`TTG_CharacterSprite_Point192.preset.yaml`, `Characters_Pilot.spriteatlasv2.yaml`, `PilotSpriteImportVerifier.cs.txt`).
- `previews/contact_sheets/` — `contact_builder_all_18f.png`, `contact_grunt_all_18f.png`, `contact_vfx_and_sync_board.png`.
- `previews/gifs/` — `11` animated `12 FPS` GIF previews (`Idle`, `Walk`, `Repair`, `ClubSmash`, `Hit`, `VFX`, `Repair + Spark Sync`, and `1080p` composite battle crop).
- `previews/mp4/` — `4` `12 FPS` `libx264 yuv420p` MP4 previews (`1080×1920`, `720×1280`, `360×640`, and all-clips `1280×720` showcase).
- `qa/` — `5` diagnostic PNG boards, `qa_metrics.json`, and `unity_isolated_verification.json`.
- `tools/` — Reproducible deterministic Python scripts (`build_art003_rigs_and_frames.py`, `build_art003_atlas_and_previews.py`, `validate_art003_qa_and_manifest.py`).

---

## 3. Reproducing the Entire Package from Source

From the repository root (`python >= 3.11`, `Pillow`, `numpy`, `scipy`, `ffmpeg`):

```powershell
python art/production/ART-003/tools/build_art003_rigs_and_frames.py
python art/production/ART-003/tools/build_art003_atlas_and_previews.py
python art/production/ART-003/tools/validate_art003_qa_and_manifest.py
```

---

## 4. Honest Scope & Remaining Gates

1. **`M06` Not Complete:** `ART-003` is a `2`-character + `1`-VFX vertical slice inside `art/production/ART-003/`. `Guard`, `Fighter`, `Runner`, `Brute`, `Boss` (`256×256`), Village/Castle production sprites, and runtime `unity/Assets/_TTGQLDL/` wiring belong to subsequent production packages and `M06`.
2. **Stream Compression Proxy vs `GATE-OBS`:** `qa_h264_compression_proof.png` and `previews/mp4/*.mp4` use real `ffmpeg libx264 yuv420p` encoding as a software proxy; hardware OBS streaming + real mobile viewer capture remain gated under `GATE-OBS`.
3. **Rights Clearance (`GATE-LEGAL`):** All files carry `rights_status = "PENDING_REVIEW"` pending formal repository-owner / legal sign-off before commercial `P1` release.
