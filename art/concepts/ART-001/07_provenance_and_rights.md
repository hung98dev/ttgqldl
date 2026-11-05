# ART-001 — Provenance, Original-Content Declaration & Rights Review

**Project:** TTGQLDL (`hung98dev/ttgqldl`)  
**Governance Authorities:** [`AGENTS.md`](../../../AGENTS.md), [`docs/plan/07_UNITY_ART_UX.md`](../../../docs/plan/07_UNITY_ART_UX.md), [`docs/plan/09_QA_ACCEPTANCE.md`](../../../docs/plan/09_QA_ACCEPTANCE.md) (`QA-ASSET-01`, `QA-GATE-01`), [`docs/plan/11_EXTERNAL_GATES.md`](../../../docs/plan/11_EXTERNAL_GATES.md) (`GATE-LEGAL`)

---

## 1. Original-Content & Non-Infringement Declaration (`ART-001` Concept Scope)

1. **Zero Third-Party Franchises or Trademarks:** All visual concepts in `art/concepts/ART-001/`—including the three world directions (*Bastion Dawn*, *Hearthguard Chibi*, *Tactical Crisp Bastion*), the seven character archetypes (`Builder`, `Guard`, `Fighter`, `Runner`, `Grunt`, `Brute`, `Boss`), and all vector HUD icons—were authored from original geometric, color, and fantasy-archetype briefs written specifically for `TTGQLDL`. No third-party copyrighted character names, franchise logos, ripped game sprites, or scraped web images were used as inputs or overlays.
2. **Exact Tool & Generation Provenance:**
   - **Raw Concept Illustrations (`images/source/*.jpg`):** Generated during conversation `10767eed-7bf5-4f62-b206-f942db75c4bc` on `2026-11-05` via the built-in Google Antigravity `image-generator` subagent from original text-only art-direction specifications.
   - **Anchored HUD Overlays, Vector Icons, `360 × 640` Previews, BT.709 Grayscale Proofs & Layout Boards (`images/*.png`):** Deterministically rendered via local Python `3.15.0` / Pillow `13.0.0` (`render_gameplay_and_hud.py`) using custom geometric primitives and local Windows system fonts (`Segoe UI` / `Bahnschrift` / `Consolas`) strictly for **local offline concept visualization**.
3. **Typography Licensing Note for Future M06 Shipping Builds:**
   - Windows system fonts (`segoeuib.ttf`, `bahnschrift.ttf`, `consolab.ttf`) were used **only to rasterize static concept mockup PNGs** on a licensed Windows 11 workstation and are **NOT** bundled, redistributed, or embedded as `.ttf`/`.otf` files in the repository.
   - For the distributable `M06` Unity standalone player (`unity/Assets/_TTGQLDL/Art/UI/Fonts/`), open-licensed SIL Open Font License `1.1` fonts (**Be Vietnam Pro** + **Barlow Semi Condensed**) MUST be committed alongside their `OFL.txt` license files so `QA-OPS-07` (CycloneDX SBOM) and `GATE-LEGAL` have clean redistribution rights.

---

## 2. Honest Gate Status (`GATE-LEGAL` Remains `BLOCKED` / Pending Human Sign-Off)

Per [`docs/plan/11_EXTERNAL_GATES.md`](../../../docs/plan/11_EXTERNAL_GATES.md):

> *"AI may create original assets; document license origin without asserting legal sign-off."*

- **Current IP / Rights Status of `ART-001` Files:** `PROPOSED_PENDING_OWNER_APPROVAL` (concept stage only; **not** legal sign-off, **not** `GATE-LEGAL` PASS, **not** `QA-ASSET-01` M06 production PASS).
- **Required Human Actions Before `M06` / `P0` / `P1` Release:**
  1. **Owner Visual Approval (`ART-002`):** Repository owner inspects `ART-001` mockups and character lineup, confirms originality satisfaction, and locks one visual direction.
  2. **Accountable Legal / Rights Reviewer (`GATE-LEGAL`):** Human operator/rights holder reviews AI-assisted concept/asset policy under applicable jurisdiction and platform terms (TikTok LIVE / commercial broadcast policy) and decides whether final `M06` sprites are hand-authored/retraced in Aseprite/Photoshop/Spine from the approved `ART-001` concept sheets or cleared under documented studio commercial terms, attaching signed external evidence before `P1`.

---

## 3. Provenance & Checksum Ledger Overview

Every file delivered in `art/concepts/ART-001/images/` is registered in [`manifest.json`](manifest.json) with its exact dimensions, format, source-to-export lineage, provenance string, and review status. No missing, placeholder, or uncreated assets are claimed as complete.
