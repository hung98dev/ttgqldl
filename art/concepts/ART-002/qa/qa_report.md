# ART-002 Deliverable I — Automated & Visual Quality Assurance Report

- **Work Package:** `ART-002`
- **Machine-Readable Metrics Output:** `qa/qa_metrics.json`
- **Diagnostic Visual Evidence Plates:**
  - `qa/qa_resolutions_sheet.png` (`1080×1920`, `720×1280`, `360×640` Color & Grayscale + effective typography audit)
  - `qa/qa_h264_compression_simulation.png` (YUV `4:2:0` subsampling + `8×8` DCT quantization at `Q=32` & bitrate-starved `Q=18` + Laplacian noise audit)
  - `qa/qa_grayscale_and_contrast_audit.png` (BT.709 grayscale + programmatic WCAG 2.1 contrast table + safe-area pixel gradient verification)
  - `qa/qa_48px_character_audit.png` (`7` classes at `48 px` base scale after YUV `4:2:0` compression in Color & Grayscale)

---

## 1. Honest Verification Boundaries (Automated vs Visual vs Human Testing)

In strict compliance with `AGENTS.md` and `ART-002 §13`, this report separates what was **programmatically measured**, what was **visually inspected by the Art Director**, and what **requires external human participants / hardware in later milestones**:

1. **Automated Programmatic Checks (`EXECUTED — PASS`):** Deterministic Python (`Pillow` + `NumPy`) inspection of image dimensions (`1080×1920`, `720×1280`, `360×640`), effective `360×640` font heights, exact WCAG 2.1 relative-luminance contrast ratios across all `12` HUD foreground/background pairs, pixel-level safe-zone leak checks (`x > 885 px`, `y > 1689 px`) across all `11` clean `1080×1920` HUD screens, YUV `4:2:0` + `8×8` DCT compression PSNR, and 2D Laplacian terrain high-frequency noise standard deviation.
2. **Structured Visual Inspection (`EXECUTED — PASS`):** Direct `1:1` and `2x–4x` nearest-neighbor inspection of all `7` four-view character turnarounds, weapon/point-filter detail sheets, `48 px` color/grayscale matrices, `12` village environment elements, and `16` visual states (catching and fixing two early defects during `ART-002`: `v1` Guard/Fighter `5.0`-head stretch and East-lane slope/HUD text overflow).
3. **Human Cold-Viewer Usability Testing & Hardware Capture (`NOT PERFORMED — GATED`):**
   - `QA-PSY-01` (`>= 5` cold external human viewers identifying village HP, incoming threat, and `Bình luận XÂY / THỦ / ĐÁNH` within `3.0` seconds) was **not** performed and is **not** claimed.
   - Real `OBS 32.1.x` hardware encoder capture (`GATE-OBS`) and authorized TikTok LIVE stream testing (`GATE-LIVE`) remain external operational gates.

---

## 2. Ten-Point Inspection Summary & Measured Threshold Compliance

| # | Required Inspection Item | Required Threshold | Measured `ART-002` Value | Method | Status |
|---:|---|---|---|---|---|
| 1 | **`1080 × 1920` Primary Reference** | Exact `9:16` portrait, zero clipping/seams | `1080 × 1920` verified across all `12` full-screen HUD plates (`0` bbox overflows) | Automated + Visual | **PASS** |
| 2 | **`720 × 1280` Mid-Tier Stream** | Clean downscale, crisp icons & unit edges | `720 × 1280` verified (`PSNR = 26.21 dB` under `4:2:0 Q=26` proxy) | Automated + Visual | **PASS** |
| 3 | **`360 × 640` Mobile Preview** | Primary status & single CTA readable at `1:1` | Verified on `ui/hud_mobile_360x640.png` & `qa/qa_resolutions_sheet.png` | Automated + Visual | **PASS** |
| 4 | **BT.709 Grayscale** | Zero color-only status or class ambiguity | Verified at `1080p`, `360p`, and `48 px` unit scale (segmented HP + distinct silhouettes) | Automated + Visual | **PASS** |
| 5 | **Simulated H.264 Compression** | Silhouettes & CTA survive `4:2:0` + `8×8` DCT | `1080p Q=32`: `27.69 dB`; harsh `Q=18`: `26.22 dB`; `360p Q=26`: `24.52 dB` | Automated + Visual | **PASS** |
| 6 | **Safe-Area Overlap** | `0` essential HUD in `x > 82%` or `y > 88%` | Active HUD envelope `x = 36..876 px` (`81.11%`), `y = 22..1686 px` (`87.81%`); `0.00` HUD step in safe zones | Automated Pixel Check | **PASS** |
| 7 | **Character Visibility (`48 px`)** | All `7` classes recognizable at `48 px` base | Runner `44 px`, Roles/Grunt `48 px`, Brute `64 px`, Boss `102 px` (`PSNR = 28.95 dB` @ `4:2:0 Q=24`) | Automated + Visual | **PASS** |
| 8 | **Text Readability (`360×640`)** | CTA `>=16 px`, HP/Timer `>=14 px`, Sec `>=12 px` | CTA **`16.0 px`** (`48 px`), Timer **`14.33 px`** (`43 px`), HP **`14.0 px`** (`42 px`), Sec **`12.0–12.67 px`** (`36–38 px`) | Automated Exact Check | **PASS** |
| 9 | **UI Contrast (WCAG 2.1)** | Essential text contrast **`>= 4.50 : 1`** | **`11.74 : 1` minimum** (`17.85 : 1` CTA, `17.91 : 1` HP, `12.57 : 1` Timer) across all `12` pairs | Automated Luminance | **PASS** |
| 10 | **Environment Noise & Budgets** | Low-frequency turf; `<=150` units; `<=20` VFX | Turf Laplacian SD reduced **`50.0%`** (`21.76 -> 10.87`); `150` entity & `20` VFX caps locked | Automated + Spec | **PASS** |

---

## 3. Complete WCAG 2.1 Contrast Table (`qa/qa_metrics.json`)

| HUD Element | Foreground RGB / HEX | Background RGB / HEX | Measured Contrast Ratio | Requirement | Verdict |
|---|---|---|---:|---:|---|
| Wall & Castle HP Readout | `#FFFFFF` `(255, 255, 255)` | `#101726` `(16, 23, 38)` | **`17.91 : 1`** | `>= 4.50 : 1` | `PASS` |
| Primary Command CTA (`Bình luận...`) | `#FFFFFF` `(255, 255, 255)` | `#0F172A` `(15, 23, 42)` | **`17.85 : 1`** | `>= 4.50 : 1` | `PASS` |
| Boss Encounter Threat Banner | `#FFFFFF` `(255, 255, 255)` | `#260C31` `(38, 12, 49)` | **`17.81 : 1`** | `>= 4.50 : 1` | `PASS` |
| Role Counter Labels (`XÂY/THỦ/ĐÁNH`) | `#FFFFFF` `(255, 255, 255)` | `#121A2B` `(18, 26, 43)` | **`17.38 : 1`** | `>= 4.50 : 1` | `PASS` |
| `STORAGE_UNAVAILABLE` Error Banner | `#FFFFFF` `(255, 255, 255)` | `#3A0C12` `(58, 12, 18)` | **`16.92 : 1`** | `>= 4.50 : 1` | `PASS` |
| Wood / Stone / Food Counters | `#F1F5F9` `(241, 245, 249)` | `#101726` `(16, 23, 38)` | **`16.35 : 1`** | `>= 4.50 : 1` | `PASS` |
| `PREP` / `VOTE_OPEN` Phase Banner | `#FFFFFF` `(255, 255, 255)` | `#102230` `(16, 34, 48)` | **`16.23 : 1`** | `>= 4.50 : 1` | `PASS` |
| Locked Even-Wave Buff Strip | `#FEF08A` `(254, 240, 138)` | `#1C170C` `(28, 23, 12)` | **`15.33 : 1`** | `>= 4.50 : 1` | `PASS` |
| Source Health `OK` Pill | `#DCFCE7` `(220, 252, 231)` | `#0E2A1C` `(14, 42, 28)` | **`13.99 : 1`** | `>= 4.50 : 1` | `PASS` |
| Source Health `STALE` Pill | `#FFEDD5` `(255, 237, 213)` | `#3A1C0C` `(58, 28, 12)` | **`13.56 : 1`** | `>= 4.50 : 1` | `PASS` |
| Wave & Countdown Timer (`GIỜ 00:38`) | `#FEF08A` `(254, 240, 138)` | `#1E293B` `(30, 41, 59)` | **`12.57 : 1`** | `>= 4.50 : 1` | `PASS` |
| Honest `MOCK / REPLAY` Watermark | `#FACC15` `(250, 204, 21)` | `#1C160A` `(28, 22, 10)` | **`11.74 : 1`** | `>= 4.50 : 1` | `PASS` |

---

## 4. Defect Log Caught & Resolved During `ART-002` QA

1. **Defect QA-D01 (`RESOLVED`) — Initial `Guard` & `Fighter` Turnaround Proportion Stretch (`5.0` heads) & Extra Weapons:** Caught during visual inspection of raw turnaround pass `v1`; regenerated as `v2` (`3.2` heads tall, shield-only Guard, single-sword Fighter) before assembling `guard_model_sheet.png` and `fighter_model_sheet.png`.
2. **Defect QA-D02 (`RESOLVED`) — Top HUD Row 1 Text Crowding & Banner Right-Edge Overflow:** Caught during visual inspection of first `hud_master.png` build; restructured Top HUD into 3 spacious sub-rows and added `draw_text_strict()` bounding-box assertions across all 16 states (`0` overflows).
3. **Defect QA-D03 (`RESOLVED`) — Village Plate Rectangular Turf Seam & East-Lane Angle Kink:** Caught on `environment_overview.png`; replaced uniform background fill with row/column edge-clamped Gaussian field continuation + `44 px` cosine feathering + `45°` slope-matched East lane extension.
