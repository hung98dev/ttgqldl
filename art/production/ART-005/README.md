# ART-005 — Production Environment, Castle, Wall & Three-Lane Battlefield

## 1. Package Scope & Normative Disambiguation

- **Work Package ID:** `ART-005` (`art/production/ART-005/`)
- **Normative Disambiguation:** Work package `ART-005` delivers the production 2D environment, Townhall progression bands, defensive curtain wall states, village buildings, courtyard props, seamless terrain tiles, world lighting palettes, and Unity 2D atlases. It is distinct from requirement ID `ART005` (*Combat, Hit & Boss Telegraph VFX*, already delivered in [ART-003 README](../ART-003/README.md) and [ART-004 README](../ART-004/README.md)) in [07_UNITY_ART_UX.md](../../../docs/plan/07_UNITY_ART_UX.md).
- **Visual Style Lock:** `Bastion Dawn v1` (`#11151C` structural outer ink silhouette on buildings/props, **zero `#11151C` black ink on ground/road tiles**, 2-step `45°` top-left warm cel shading, binary `0/255` alpha with `0` semi-transparent edge halos on cutouts, `HSV S <= 48%` H.264-safe calm meadow & lanes).
- **Engine Target:** `Unity 6000.6.7f1` — Built-in 2D (`Orthographic Size = 5.0`, `PPU = 192` -> `192.0 px/unit` at `1080×1920`).
- **Governance & Milestone Status:** Production `unity/`, `backend/`, `operator/`, and `docs/plan/` remain untouched. `M06` in [10_IMPLEMENTATION_MATRIX.md](../../../docs/plan/10_IMPLEMENTATION_MATRIX.md) remains `NOT_STARTED`. Rights status remains `PENDING_REVIEW` (`GATE-LEGAL` open).

---

## 2. Canonical Deliverable Summary (26 Canonical Sprites + 14 Companion Depth/Transition Assets)

| Group | Canonical Count | Companion Slices | Dimensions | Sorting Layer(s) | Path Pattern |
|---|---:|---:|---|---|---|
| Townhall / Castle Progression (`Bands 1–4`) | 4 | 0 | `512×512` RGBA | `BastionBack` | `castle/frames/env_castle_band*.png` |
| Curtain Wall & South Gate (`States 1–5`) | 5 | 10 (`5` back + `5` front) | `512×512` (`512×320` back, `512×256` front) | `BastionBack` + `BastionFront` | `wall/states/*.png`, `wall/layers/*.png` |
| Modular Builder Repair Scaffold | 1 | 0 | `192×192` RGBA | `BastionBack` / `BastionFront` | `wall/scaffold/env_wall_repair_scaffold.png` |
| Defensive Village Watchtower | 1 | 0 | `256×256` RGBA | `BastionBack` | `buildings/watchtower/env_bldg_watchtower.png` |
| Courtyard Village Cottages (`Houses 1–6`) | 6 | 0 | `256×256` RGBA | `BastionBack` | `buildings/houses/env_house_01..06_*.png` |
| Low-Profile Courtyard Props (`Props 1–3`) | 3 | 0 | `128×128` RGBA | `BastionBack` | `buildings/props/env_prop_*.png` |
| Seamless Terrain & Road Tiles (`Tiles 1–6`) | 6 | 4 transitions | `256×256` RGBA | `Ground` / `LanesAndDecals` / `BastionBack` | `terrain/tiles/env_tile_01..06_*.png`, `terrain/transitions/trans_*.png` |
| **Total** | **26** | **14** | **40 packed atlas slices** | **6 canonical layers** | `art/production/ART-005/` |

---

## 3. Deterministic Contracts & Atlases

- `environment_layout.json` — Exact `1080×1920` screen and Unity world-unit (`Orthographic Size = 5.0`, `PPU = 192`) coordinates for all 3 lanes (`West`, `Center Boss Causeway`, `East <= 82% safe X`), central `477 px` (`44.17%`) bastion, Townhall, Watchtower, 6 courtyard cottages, 3 props, 4 scaffold anchors (`SW/SE/NW/NE`), and 6 `SortingLayers`.
- `environment_state_mapping.json` — Deterministic cosmetic Townhall level-band partition (`Lv 1–9`, `Lv 10–24`, `Lv 25–39`, `Lv 40–50`; authoritative `townhall_hp = 500` invariant) and normalized Wall HP damage state priority table (`post_loss_recovery` -> `breached` `0%` -> `critical` `<25%` -> `damaged` `25–59%` -> `healthy` `60–100%`) across all 20 wall tiers (`1000..4800` max HP).
- `lighting_palettes.json` — Four deterministic environment-only world lighting palettes (`dawn`, `midday`, `dusk`, `boss_eclipse`) preserving HUD colors and faction silhouette contrast.
- `atlas/environment_atlas_01_fortress.png` + `.json` (`2048×2048` RGBA32, `20` non-tile canonical sprites at 100% native resolution, `>= 8 px` gutter, `0` bleed).
- `atlas/environment_atlas_02_layers_and_terrain.png` + `.json` (`2048×2048` RGBA32, `10` split wall depth slices + `6` canonical tiles + `4` transitions = `20` slices, `>= 8 px` gutter, `0` bleed).
- `asset_manifest.json` — Full SHA-256 and byte-size inventory of every tracked file in `ART-005`.

---

## 4. Reproducible Build & Validation Commands

```powershell
python -B art/production/ART-005/tools/build_environment.py
python -B art/production/ART-005/tools/build_atlas_and_previews.py
python -B art/production/ART-005/tools/validate_art005.py
```

See also:
- [APPROVAL_RECORD.md](APPROVAL_RECORD.md)
- [ENVIRONMENT_PIPELINE.md](ENVIRONMENT_PIPELINE.md)
- [UNITY_INTEGRATION_GUIDE.md](UNITY_INTEGRATION_GUIDE.md)
- [qa/qa_report.md](qa/qa_report.md)
