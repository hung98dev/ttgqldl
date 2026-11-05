# ART-003 Approval Gate & Governance Record

## 1. Repository & Base Inspection

- **Work Package:** `ART-003` — Production-Ready Sprite Animation Vertical Slice (`Builder + Grunt + Repair Spark VFX + Pilot Atlas`)
- **Branch:** `main` (per [AGENTS.md](../../../AGENTS.md))
- **Inspected Base Commit SHA:** `32b963e47ee490979c3ff370cf6a5d3cddcdcc1e`
- **Inspected Base CI Status:** GitHub Actions CI run `41988386038` (`5/5` jobs `PASS`: `Secret Scan`, `Validate Plan & Governance`, `Backend Lint Test & Determinism`, `Operator Console Lint Test & Build`, `Unity Guardrails`)
- **Canonical Plan Authority:** [00_INDEX.md](../../../docs/plan/00_INDEX.md), [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md), [09_QA_ACCEPTANCE.md](../../../docs/plan/09_QA_ACCEPTANCE.md), and [10_IMPLEMENTATION_MATRIX.md](../../../docs/plan/10_IMPLEMENTATION_MATRIX.md)

## 2. ART-002 -> ART-003 Owner Approval Gate Resolution

At initial inspection of `HEAD` (`32b963e47ee490979c3ff370cf6a5d3cddcdcc1e`), all eight checklist items in [OWNER_APPROVAL.md](../../concepts/ART-002/OWNER_APPROVAL.md) were unchecked (`[ ]`). In strict compliance with `ART-003 §0` (*"Do not invent, silently grant, or mark owner approval"*), execution paused at the approval gate and queried the repository owner directly.

On **`2026-11-05T15:15:37+07:00`**, the repository owner (`hungdev98`) explicitly granted **`APPROVED`** across all `8/8` items in [OWNER_APPROVAL.md](../../concepts/ART-002/OWNER_APPROVAL.md):

1. `[x]` Direction A (`Bastion Dawn v1`) locked as sole visual direction.
2. `[x]` Character model sheets (`Builder`, `Guard`, `Fighter`, `Runner`, `Grunt`, `Brute`, `Boss`) approved for animation rigging.
3. `[x]` Environment layout (`44%` central bastion width, unmistakable West/Center/East lanes, low-noise sunlit meadow `#68A347`) approved.
4. `[x]` Vertical `9:16` HUD (`<= 22%` safe-area budget, single primary CTA `"Bình luận XÂY / THỦ / ĐÁNH"`, safe zones `x > 82%` and `y > 88%`) approved.
5. `[x]` `Boss` `256×256` source cell size (`PPU = 192`, `96–112 px` displayed gameplay height) approved.
6. `[x]` `4` cosmetic Townhall level bands (`Lv 1–9`, `10–24`, `25–39`, `40–50`; authoritative `townhall_hp = 500` unchanged) approved without altering [02_GAME_RULES.md](../../../docs/plan/02_GAME_RULES.md).
7. `[x]` `360×640` mobile downscale (`16 px` unit height), BT.709 grayscale separation, and H.264 stream compression targets approved.
8. `[x]` Full transition from `ART-002` blueprint to `ART-003` sprite animation vertical slice authorized for execution, commit, and push on `main`.

## 3. Rights Status & Milestone Ownership Boundaries

- **`rights_status`:** Kept at **`PENDING_REVIEW`** per explicit owner instruction. All `42` sprite/VFX frames, `3` sprite sheets, and `1` pilot atlas in `art/production/ART-003/` are original `Bastion Dawn v1` fantasy creations with per-file SHA-256 hashes and provenance recorded in `art/production/ART-003/asset_manifest.json`. Final commercial/IP release clearance remains gated under `GATE-LEGAL` in [09_QA_ACCEPTANCE.md](../../../docs/plan/09_QA_ACCEPTANCE.md).
- **Milestone `M06` Status:** Remains **`NOT_STARTED`** in [10_IMPLEMENTATION_MATRIX.md](../../../docs/plan/10_IMPLEMENTATION_MATRIX.md). `ART-003` proves the production animation pipeline on `Builder` + `Grunt` + `vfx_builder_repair_spark` + `Characters_Pilot_Atlas` inside `art/production/ART-003/` without prematurely modifying `unity/`, `backend/`, `operator/`, or `docs/plan/`.
