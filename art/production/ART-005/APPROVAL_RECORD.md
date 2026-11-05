# ART-005 — Governance, Scope & Approval Record

## 1. Upstream Design & Pilot Lineage

1. **Blueprint Baseline:** [ART-002 README](../../concepts/ART-002/README.md) and [03_environment_design.md](../../concepts/ART-002/03_environment_design.md) established the `Bastion Dawn v1` circular bastion (`~44%` of `1080p` width), three asymmetric-origin tactical lanes (`West`, `Center Boss Causeway`, `East`), 4 Townhall cosmetic bands, 5 wall damage states, non-blocking courtyard cottages, and calm H.264-safe terrain rules.
2. **Character & VFX Compatibility Baseline:** [ART-003 README](../ART-003/README.md) (`Builder`, `Grunt`, `vfx_builder_repair_spark`) and [ART-004 README](../ART-004/README.md) (`Guard`, `Fighter`, `Runner`, `Brute`, `Boss`, and 4 combat/telegraph VFX) locked `PPU = 192`, `45°` top-left warm sunlight, `#11151C` structural dark-ink silhouettes, and `VisualRoot` gameplay pixel budgets (`44–102 px` at `1080×1920`).
3. **Owner Execution Authorization:** User explicitly commissioned `ART-005` in-session to produce the complete modular production environment package on `main`. Note: formal human sign-off checklists in [ART-002 OWNER_APPROVAL.md](../../concepts/ART-002/OWNER_APPROVAL.md) are preserved untouched and never silently checked.

---

## 2. Authoritative Rule & Milestone Boundaries

- **Cosmetic Townhall Invariant ([02_GAME_RULES.md](../../../docs/plan/02_GAME_RULES.md)):** All four Townhall sprites (`Lv 1–9`, `Lv 10–24`, `Lv 25–39`, `Lv 40–50`) are strictly cosmetic progression visuals. Authoritative `townhall_hp` remains `500` at every village level.
- **Wall Damage & Recovery Invariant ([02_GAME_RULES.md](../../../docs/plan/02_GAME_RULES.md), [03_SIMULATION.md](../../../docs/plan/03_SIMULATION.md)):** Wall damage visuals map strictly to normalized `wall_hp / wall_max` across tiers `1..20` (`wall_max = 1000 + 200 * (wall_tier - 1)`), with `env_wall_state5_recovery.png` prioritized when `post_loss_recovery` (`250 HP` floor) is active and `wall_hp > 0`.
- **Watchtower & Courtyard Cottages:** Strictly cosmetic landmarks; zero hidden turrets, ranged attacks, or pathfinding collision changes.
- **Milestone Integrity ([10_IMPLEMENTATION_MATRIX.md](../../../docs/plan/10_IMPLEMENTATION_MATRIX.md)):**
  - `M06` Unity runtime scene wiring remains `NOT_STARTED`.
  - `GATE-LEGAL` (`rights_status = "PENDING_REVIEW"`), `GATE-OBS`, `QA-VIS-01` (`>= 60 FPS` live Unity runtime soak), and `QA-PSY-01` (human first-time viewer comprehension) remain honestly open and unfabricated.
