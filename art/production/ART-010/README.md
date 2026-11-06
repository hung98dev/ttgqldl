# ART-010 — Visual Overhaul (Phase A: Three Premium Gameplay Art Directions for Approval)

**Package:** `ART-010`  
**Phase:** `Phase A — Visual Exploration & Art Direction Approval` (HARD STOP before `ART-010B`)  
**Target Platform & Aspect:** TikTok LIVE Portrait `9:16` (`1080×1920` master / `360×640` mobile verification)  
**Target Engine:** Unity `6000.6.7f1` — Built-in 2D / uGUI + TextMeshPro  
**Canonical Gameplay Contract:** `ttg-rules-v1` (3 lanes West/Center/East, 7 unit classes, Wall + Townhall HP, Wood/Stone/Food, Wave 10 `3000` ticks, single CTA `Bình luận XÂY / THỦ / ĐÁNH`)

---

## 1. Mission & Scope Boundary

`ART-003` through `ART-009` proved deterministic pipeline mechanics, canonical simulation playback, 15 original audio cues, and HUD state synchronization. However, an honest visual inspection of `ART-009` (`1080×1920` and `360×640`) confirms that the current prototype **is not visually compelling enough to ship or stop a TikTok LIVE scroller**: over `58%` of the frame is flat olive-green void, lane tiles form blocky staircases, the ring castle occupies `<9%` of the canvas, and `48 px` characters shrink to `15–16 px` specks at `360×640`.

`ART-010` Phase A delivers **three genuinely distinct, commercially polished 2D portrait gameplay art directions** so the repository owner can approve the exact visual target before mass asset replacement begins in `ART-010B`:

1. **Concept A — Fantasy Diorama Kingdom (`concepts/A_fantasy_diorama/`)**: Warm golden-hour storybook kingdom with lush medieval architecture, timbered houses, lantern-lit stone lanes, and strong emotional attachment to the village.
2. **Concept B — High-Impact Action Defense (`concepts/B_action_defense/`)**: Tighter `1.6×` close-up action framing with `~2.4×` larger unit silhouettes (`37–42 px` at `360p`), dramatic twilight rim lighting, bold sword/shield/magic VFX, and visceral Boss dread.
3. **Concept C — Vertical Siege Battlefield (`concepts/C_vertical_siege/`)**: Native `9:16` four-tier vertical siege architecture with three stone bridges converging top-to-bottom, cool-threat vs warm-sanctuary lighting separation, and natural cliff-chasm clearance along the rightmost `18%` TikTok UI rail.

> **IMPORTANT — PHASE 9 APPROVAL GATE:**  
> `ART-010` Phase A is **DESIGN EXPLORATION ONLY**. All `ART-003`..`ART-009` production masters, Unity showcase scripts, backend services, SQL schemas, and milestone statuses (`M06`) remain untouched. Mass production (`ART-010B`) is strictly gated on owner sign-off in [`OWNER_APPROVAL_REQUIRED.md`](OWNER_APPROVAL_REQUIRED.md).

---

## 2. Deliverable Inventory (`art/production/ART-010/`)

```text
art/production/ART-010/
  README.md                         # Package overview, scope boundary & directory map
  ART_DIRECTION_REVIEW.md           # Honest critique of ART-009 + detailed design breakdown of Concepts A, B, C
  COMPARISON_REPORT.md              # 1-10 expert heuristic matrix vs ART-009 + recommended direction
  ASSET_REUSE_ANALYSIS.md           # Package-by-package (ART-003..ART-009) reuse scope & Unity Built-in 2D plan
  OWNER_APPROVAL_REQUIRED.md        # Explicit Phase 9 STOP gate & owner decision checklist

  concepts/
    A_fantasy_diorama/
      gameplay_1080p.png            # 1080x1920 active ASSAULT beauty mockup + integrated diegetic HUD
      boss_1080p.png                # 1080x1920 Wave 10 FINALE Boss encounter + integrated diegetic HUD
      gameplay_360p.png             # True 360x640 direct LANCZOS downscale of gameplay_1080p.png
      boss_360p.png                 # True 360x640 direct LANCZOS downscale of boss_1080p.png
      safe_area.png                 # 1080x1920 TikTok 82%x88% safe-area, focal castle & combat overlay
      environment_clean.png         # 1080x1920 HUD-free clean world composition plate

    B_action_defense/
      gameplay_1080p.png
      boss_1080p.png
      gameplay_360p.png
      boss_360p.png
      safe_area.png
      environment_clean.png

    C_vertical_siege/
      gameplay_1080p.png
      boss_1080p.png
      gameplay_360p.png
      boss_360p.png
      safe_area.png
      environment_clean.png

  comparisons/
    current_vs_A.png                # Side-by-side Before (ART-009) vs After (Concept A) ASSAULT & BOSS
    current_vs_B.png                # Side-by-side Before (ART-009) vs After (Concept B) ASSAULT & BOSS
    current_vs_C.png                # Side-by-side Before (ART-009) vs After (Concept C) ASSAULT & BOSS
    all_three_360p.png              # 1:1 unscaled 360x640 matrix: Old ART-009 vs A vs B vs C

  qa/
    visual_review.md                # Independent visual & gameplay-contract QA audit of all 22 PNGs
    composition_metrics.json        # Machine-measured resolution, contrast, flat-void ratio, PSNR & safe-area bounds
    asset_provenance.json           # Honest provenance, SHA-256 hashes & GATE-LEGAL pending status
```

---

## 3. Non-Negotiable Truthfulness Disclosures

- **Static Design Mockups Only:** The 18 concept PNGs and 4 comparison boards are static Phase A visual target mockups, **not** captures from a newly built Unity runtime scene.
- **Expert Heuristic Scoring Only:** Scores in [`COMPARISON_REPORT.md`](COMPARISON_REPORT.md) are transparent professional art-direction judgments (`1–10`), **not** claims of external human viewer or usability studies.
- **Rights & Provenance:** Zero third-party proprietary game screenshots, ripped textures, or stock UI kits were used. Commercial release clearance remains gated on `GATE-LEGAL = PENDING_REVIEW` (`qa/asset_provenance.json`).
