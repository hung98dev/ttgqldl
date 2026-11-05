# ART-005 — Unity 6000.6.7f1 Environment Integration Guide (M06 Handoff)

## 1. Camera, World-Unit & PPU Contract

Matches `unity/Assets/_TTGQLDL/Editor/SceneFactory.cs` (`Camera.orthographicSize = 5.0` at `1080×1920`, `9:16`):
- **Pixels Per Unit (`PPU`):** `192` (`1` world unit = `192.0` screen pixels at `1080×1920`).
- **World Bounds:** `X ∈ [-2.8125, +2.8125]`, `Y ∈ [-5.0000, +5.0000]`.
- **Exact Screen-to-World Bijection:**
  $$\text{world\_x} = \frac{\text{screen\_x} - 540.0}{192.0}, \qquad \text{world\_y} = \frac{960.0 - \text{screen\_y}}{192.0}$$
- **Livestream Safe-Zone World Limits:**
  - Right TikTok engagement icon rail (`x <= 82%` = `885.6 px`): `world_x <= +1.8000`.
  - Bottom comment / join overlay (`y <= 88%` = `1689.6 px`): `world_y >= -3.8000`.

---

## 2. Canonical Six-Layer `SortingLayer` Stack (`TagManager.asset`)

Register in exact bottom-to-top order during `M06`:

| Index | `SortingLayer` | `sortingOrder` Rule | Assigned Assets |
|---:|---|---|---|
| 0 | `Ground` | `0..10` | `env_tile_01_meadow_main`, `env_tile_02_meadow_variation` (`Tilemap` or `SpriteDrawMode.Tiled`, `WrapMode.Repeat`) |
| 1 | `LanesAndDecals` | `0..30` | West/East ochre lanes (`env_tile_03`, `env_tile_04`, `trans_ochre_verge_strip`), Center Boss Causeway (`env_tile_05`, `trans_causeway_stone_curb`), `trans_lane_bastion_apron`, and dynamic `vfx_boss_slam_telegraph` (`order = 25`) |
| 2 | `BastionBack` | `-Mathf.RoundToInt(worldY * 100f)` | Inner courtyard (`env_tile_06`), North wall slice (`env_wall_state*_back`), `env_bldg_watchtower`, `env_castle_band1..4`, `env_house_01..06`, `env_prop_*`, and `NW/NE` scaffolds |
| 3 | `Units` | `-Mathf.RoundToInt(pivotWorldY * 100f)` | All 7 character classes (`Builder`, `Guard`, `Fighter`, `Runner`, `Grunt`, `Brute`, `Boss`; `<= 150` pooled `SpriteRenderer`s) |
| 4 | `BastionFront` | `0..20` | Low South wall parapet & Gate arch slice (`env_wall_state*_front`) and `SW/SE` repair scaffolds (`env_wall_repair_scaffold`) |
| 5 | `VFX` | `0..50` | Combat & repair bursts (`vfx_builder_repair_spark`, `vfx_fighter_slash_arc`, `vfx_guard_shield_block`, `vfx_unit_hit_puff`; `<= 20` pooled) |

---

## 3. Texture, Atlas & Preset Import Rules

Presets are provided in `atlas/unity_presets/`:
- `TTG_ART005_Environment_Point192.preset.yaml` — Standalone cutout sprites & packed atlases (`PPU = 192`, `FilterMode.Point`, `Mipmaps = false`, `WrapMode.Clamp`, `AlphaIsTransparency = true`, `MaxSize = 2048`).
- `TTG_ART005_SeamlessTile_Repeat192.preset.yaml` — Seamless `256×256` terrain tiles when used with `SpriteDrawMode.Tiled` or custom UV quad (`WrapMode.Repeat`, `FilterMode.Point`, `Mipmaps = false`).
- `ART005_Environment.spriteatlasv2.yaml` — `SpriteAtlas` v2 template (`padding = 4`, `enableRotation = 0`, `enableTightPacking = 0`, `maxTextureSize = 2048`).

Packed atlases verified in isolated `Unity 6000.6.7f1 -batchmode`:
- `atlas/environment_atlas_01_fortress.png` (`2048×2048` RGBA32, `20` fortress/village cutouts, `16.0 MiB` uncompressed, `0` bleed).
- `atlas/environment_atlas_02_layers_and_terrain.png` (`2048×2048` RGBA32, `20` wall depth slices + terrain/transition slices, `16.0 MiB` uncompressed, `0` bleed).

---

## 4. Runtime State & World Lighting Evaluation (`environment_state_mapping.json` & `lighting_palettes.json`)

1. **Townhall Cosmetic Swap (`0.0` world-unit drift):** Select `env_castle_band1..4` from `village_level` (`1..9`, `10..24`, `25..39`, `40..50`) on a single pooled `SpriteRenderer` at `world = (-0.44792, +0.85417)` (`scale = 0.50`). Never alter `townhall_hp = 500`.
2. **Split Wall Damage Swap (`0.0` world-unit drift):** Evaluate `environment_state_mapping.json` priority (`is_post_loss_recovery && wall_hp > 0` -> `state5`; `wall_hp <= 0` -> `state4`; `ratio < 0.25` -> `state3`; `ratio < 0.60` -> `state2`; else `state1`) and assign `*_back` + `*_front` simultaneously at `world = (-0.44792, -0.19792)`.
3. **Environment-Only World Lighting:** Apply `lighting_palettes.json` (`dawn`, `midday`, `dusk`, `boss_eclipse`) via `MaterialPropertyBlock` tint/saturation on environment `SortingLayers` (`0..2, 4`) only — never tinting the UI `Canvas`.
