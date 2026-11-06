# OWNER APPROVAL REQUIRED — ART-010 PHASE 9 GATE

**Status:** `BLOCKED_ON_OWNER_DIRECTION_CHOICE` (Intentional Phase A Hard Stop)  
**Completed Phase:** `ART-010 Phase A — Three Premium Gameplay Art Directions`  
**Blocked Next Phase:** `ART-010B — Production Asset Overhaul & Unity Built-in 2D Implementation`

---

## 1. Mandatory Stop Boundary (Enforced)

Per `ART-010` Phase 9 rules and `AGENTS.md`, execution **stops here** pending explicit selection by the repository owner. None of the following actions have been started or altered:

- [x] **No mass sprite sheet regeneration:** `ART-003` and `ART-004` character/VFX production sheets remain untouched.
- [x] **No master replacement:** `ART-005`..`ART-009` environment, UI, audio, and showcase masters remain intact and regression-verified.
- [x] **No Unity production scene rewrite:** `unity/Assets/ArtShowcase/` and production client plans remain unchanged.
- [x] **No backend, simulation, or SQL edits:** Concurrent `M04` work and `ttg-rules-v1` contracts are 100% preserved.
- [x] **No false milestone promotion:** `M06` remains gated; static Phase A mockups are never claimed as Unity runtime QA.

---

## 2. Decision Options for Repository Owner

Please inspect the three concept folders under `art/production/ART-010/concepts/` and the comparison boards under `art/production/ART-010/comparisons/`, then choose one option to unlock `ART-010B`:

### Option 1 — Pure `Concept A` (`Fantasy Diorama Kingdom`)
- **Best if you prioritize:** Warm golden-hour storybook charm, lush living medieval village details, and emotional attachment to protecting the kingdom.
- **Trade-off to accept:** Requires slightly tighter foreground contrast discipline so lush foliage/lanterns never compete with medium-scale (`~78–90 px`) combat silhouettes at `360×640`.

### Option 2 — Pure `Concept B` (`High-Impact Action Defense`)
- **Best if you prioritize:** Maximum visceral combat punch, largest unit legibility (`~110–126 px` at `1080p` / `37–42 px` at `360p`), bold sword/shield/magic VFX, and immediate Boss spectacle.
- **Trade-off to accept:** Tighter camera crop shows primarily the northern gatehouse and upper ramparts rather than the full cozy village interior.

### Option 3 — Pure `Concept C` (`Vertical Siege Battlefield`)
- **Best if you prioritize:** Native `9:16` portrait siege readability, clearest top-to-bottom 3-lane convergence, natural cliff-chasm clearance along TikTok's rightmost `18%` overlay rail, and clean cool-vs-warm lighting separation.
- **Trade-off to accept:** Slate-and-granite citadel atmosphere feels more tactical/military than cozy storybook unless warmed up in the inner courtyard.

### Option 4 — **Recommended Hybrid (`Concept C` Layout + `Concept B` Scale/VFX + `Concept A` Warm Village Materials)**
- **Spatial Architecture from `Concept C`:** Keep the native `9:16` tiered vertical siege framing, three converging stone bridges/lanes in the upper-middle field, cool moonlit canyon threat zone at top, and natural cliff-drop safe margin along the rightmost `18%` TikTok rail.
- **Character Scale & Combat VFX from `Concept B`:** Scale defenders/enemies to `~96–112 px` (`32–37 px` at `360×640`) with bold directional sword arcs, golden shield-block sparks, and arcane Boss ground telegraphs.
- **Sanctuary Material Warmth from `Concept A`:** Enrich the inner castle courtyard and timbered village rooftops (`y = 880..1420`) with warm golden-amber lantern glow, slate-and-oak hand-painted roofs, and rich royal banners so viewers immediately care about protecting the home sanctuary.

---

## 3. What Unlocks Immediately After Owner Sign-Off (`ART-010B`)

Once you reply with **A**, **B**, **C**, or **Hybrid**, `ART-010B` will produce a concrete production replacement plan and execute modular asset production:
1. Layered 2D battlefield slices (sky/canyon backdrop, 3-lane middle-ground terrain, modular Wall Lv1–Lv7 + damage states, Townhall Lv1–Lv5, foreground occlusion props, baked shadow/light maps) packed into `<= 2048×2048` Unity Built-in 2D atlases.
2. Upgraded `256×256` character & VFX frame sheets matching the approved world perspective and scale while preserving existing `12 FPS` animation state timing and `ART-007` audio sync hooks.
3. Sculpted 9-slice fantasy HUD atlas & beveled icon set plugged directly into existing `HudInPlacePresenter` safe-zone anchors.
