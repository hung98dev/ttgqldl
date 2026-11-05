# ART-004 — Pre-Flight Inspection, ART-003 Audit & Owner Approval Record

**Package:** `ART-004` — Complete Character Animation & Combat VFX Production (`Bastion Dawn v1`)  
**Inspected Base Commit:** `24966d6a99dd868729ea5ee1ad99c4b7b1e89675` (`art(art003): add builder grunt animation and vfx vertical slice`)  
**Canonical Plan Entry:** [00_INDEX.md](../../../docs/plan/00_INDEX.md)  
**Primary Visual Authority:** [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md)  
**Owner Visual Sign-Off Reference:** [OWNER_APPROVAL.md](../../concepts/ART-002/OWNER_APPROVAL.md)

---

## 1. Repository Owner Authorization

Repository owner (`hungdev98`) explicitly authorized `ART-004` execution on `main`, including:

1. Narrowly scoped `ART-003` QA validator hardening (`0.75 <= seam_ratio <= 1.65`, per-foot trajectory cross-checking, and negative self-tests) plus `Idle` loop quadrature smoothing for `Builder` and `Grunt`.
2. Production of all **5 remaining character classes** (`Guard`, `Fighter`, `Runner`, `Brute`, `Boss` = `88` frames) and **4 remaining combat VFX archetypes** (`vfx_fighter_slash_arc`, `vfx_guard_shield_block`, `vfx_unit_hit_puff`, `vfx_boss_slam_telegraph` = `24` frames), totaling **`112` real production RGBA frames**.
3. Resolving the `ART-002` Fighter shoulder discrepancy in favor of the canonical silhouette constraint (`No steel shoulder pauldrons`): converting Fighter's grey steel shoulder caps into crimson `#E63946` cloth surcoat sleeves while preserving `45°` top-left cel shading and `#11151C` outer ink contours.

---

## 2. Normative Identifier Disambiguation (`ART004` vs `ART-004`)

- Normative requirement `ART004` in [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md) specifies the runtime visual entity & VFX pooling ceilings (`<= 150` pooled entities, `<= 20` active VFX bursts).
- Work package `ART-004` under `art/production/ART-004/` is the character and combat VFX asset production package designed to respect those exact `ART004` runtime pooling budgets without modifying `M06` runtime code.
- Milestone `M06` remains `NOT_STARTED` in [10_IMPLEMENTATION_MATRIX.md](../../../docs/plan/10_IMPLEMENTATION_MATRIX.md).

---

## 3. Summary of Narrow `ART-003` Pre-Flight Audit & Repairs

1. **Loop-Seam Discrepancy Repaired:** `ART-003` prose targeted `0.75 <= seam_ratio <= 1.65`, whereas `validate_art003_qa_and_manifest.py` originally allowed `0.25 <= seam_ratio <= 2.35` and both `Builder Idle` (`1.919`) and `Grunt Idle` (`1.958`) used symmetric `[0, 1, 2, 1]` oscillation (`wrap_diff ~ 2x` interior step). Both `Idle` clips were rebuilt with quadrature circular phase (`0°, 90°, 180°, 270°`), bringing `Builder Idle` to `0.962` and `Grunt Idle` to `0.901`, and tightening the validator bound to `0.75 <= seam_ratio <= 1.65`.
2. **Planted-Foot Validator Hardened:** Replaced the coarse `max(y)` bounding-box proxy with explicit per-foot sole tracking (`front_foot_xy`, `back_foot_xy`) cross-checked against opaque pixels in the rendered PNGs, plus `7/7` in-memory negative self-tests in `art/production/ART-003/tools/validate_art003_qa_and_manifest.py`.
3. **Documentation Wording Fixed:** Corrected `"4 cosmetic Townhall damage states"` to `"4 cosmetic Townhall level bands (Lv 1–9, 10–24, 25–39, 40–50)"` in [ART-003/APPROVAL_RECORD.md](../ART-003/APPROVAL_RECORD.md).

---

## 4. Rights & Acceptance Boundaries

- Every file in `art/production/ART-004/asset_manifest.json` carries `rights_status = "PENDING_REVIEW"` (`GATE-LEGAL`).
- Zero copyrighted third-party characters, ripped sprites, unlicensed textures, or audio files are included.
- No fabricated runtime `30 FPS` / `24 FPS 1% low` claims (`QA-VIS-01`), hardware `OBS` capture claims (`GATE-OBS`), or external cold-viewer claims (`QA-PSY-01`) are made.
