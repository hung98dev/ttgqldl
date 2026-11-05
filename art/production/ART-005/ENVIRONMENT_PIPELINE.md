# ART-005 — Deterministic 2D Environment & Terrain Production Pipeline

## 1. Supersampled Architectural Cutout Pipeline (`art005_env_lib.py`)

All 20 structural and prop cutout sprites (`4` Townhall bands, `5` Curtain Wall states, `1` Builder Scaffold, `1` Watchtower, `6` Courtyard Cottages, `3` Courtyard Props) plus the `10` split wall depth slices (`BastionBack` + `BastionFront`) are generated deterministically via `4×` supersampled vector-cel rasterization (`2048×2048` -> `512×512`, `1024×1024` -> `256×256`, `768×768` -> `192×192`, `512×512` -> `128×128`):

1. **Fixed `30°` / `3/4` Pseudo-Isometric Projection:** Every roof pitch, ashlar course, timber stud, and plinth bevel shares the exact `iso_pt(cx, base_y, u, v, h)` projection and `45°` top-left sunlight model from `ART-003` and `ART-004`.
2. **Zero-Jump Footprint Anchoring:**
   - All 4 Townhall progression sprites (`env_castle_band1..4`) share the exact `cx = 1024, base_y = 1844` (`(256, 461)` in `512×512`, `pivot = (0.50, 0.10)`) stone plinth footprint so level-up swaps exhibit `0.0 px` positional jump.
   - All 5 Curtain Wall states (`env_wall_state1..5`) share the exact `477 px` outer ring span (`44.17%` of `1080p`) and `pivot = (0.50, 0.14)` with transparent exterior and transparent inner courtyard (`alpha == 0` at center `(256, 256)`).
3. **Split Depth Masking (`BastionBack` + `BastionFront`):**
   - Each wall state exports both a canonical `512×512` sprite (`wall/states/`) and exact upper/lower split slices (`wall/layers/*_back.png` `512×320` on `BastionBack`, `wall/layers/*_front.png` `512×256` on `BastionFront`) with smooth angular rampart height `wall_h(theta) = 160 - 52 * sin(theta)` so East/West flanks join seamlessly while units inside the courtyard or South Gate sort cleanly between `BastionBack` and `BastionFront`.
4. **Binary Alpha & `#11151C` Outer Contour Lock:** Downsampling applies binary coverage thresholding (`alpha >= 128 -> 255`, else `0`) followed by morphological `2 px` `#11151C` outer silhouette ink locking (`0` semi-transparent halo pixels across all 20 cutout sprites and 10 wall slices).

---

## 2. Toroidal H.264-Safe Terrain & Material Transition Pipeline

All 6 canonical `256×256` terrain tiles (`terrain/tiles/env_tile_01..06_*.png`) enforce:
- **Zero `#11151C` Black Ink:** Ground and road surfaces never use dark structural ink, keeping `48 px` units and the `102 px` Boss visually dominant.
- **Strict Calm Saturation (`HSV S <= 48%`) & Low Laplacian Noise (`SD < 16.0`):** Eliminates `libx264 yuv420p` mosquito noise at `2.5 Mbps` (`PSNR = 38.4 dB` verified on `1080×1920` 3-lane combat frames).
- **Exact Toroidal Wrap & Cross-Tile Compatibility:**
  - `env_tile_01_meadow_main.png` uses `2π`-periodic trigonometric harmonics quantized into smooth gouache-cel tonal bands.
  - `env_tile_02_meadow_variation.png` applies a `sin²(πx/N)sin²(πy/N)` interior window over `tile_01` so its perimeter matches `tile_01` `1:1` on all four edges for arbitrary `Tilemap` cell mixing.
  - `env_tile_04_ochre_road_border.png` matches `tile_01` on its left edge (`x = 0`), transitions across a sculpted earth verge, and matches `env_tile_03_ochre_road_body.png` on its right edge (`x = 255`).
  - `env_tile_05_limestone_causeway.png` provides the high-value pale limestone Center Boss Causeway (`#B8C0C8`–`#D8DEE4`) with **zero** baked Boss telegraph markings (dynamic `vfx_boss_slam_telegraph` from `ART-004` renders on `LanesAndDecals` only during Boss slam windup).
