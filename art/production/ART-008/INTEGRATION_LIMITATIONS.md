# ART-008 — Honest Integration Scope, Verification Boundaries & Open Release Gates

In strict compliance with `AGENTS.md`, `09_QA_ACCEPTANCE.md`, and `10_IMPLEMENTATION_MATRIX.md`, this document separates **what ART-008 actually executed and verified** from **what remains gated on future engineering milestones or external human/hardware/legal sign-off**.

---

## 1. What Was Genuinely Executed & Verified in ART-008

- **Cross-Package Asset & Hash Verification:** Every file across `ART-003`..`ART-007` (`244` canonical assets + `6` atlases) was decoded from disk, inspected for dimensions/alpha/sample-rate/loop seams, and hash-verified (`0` missing files, `0` corrupted slices).
- **Real Isolated `Unity 6000.6.7f1` Hardware GPU Showcase & Benchmarks:** Executed in `-batchmode` **without `-nographics`** on `Windows` (`Intel(R) Core(TM) i5-14600K`, `32,452 MB` RAM, `Intel(R) UHD Graphics 770` `Direct3D12`), rendering real `1080x1920` camera frames, verifying all `7` class scales and transitions, verifying `BroadcastAudioDispatcher` (`12` voices) and `HudInPlacePresenter` (`0` scene reloads), and measuring `180`-frame load loops at `10`, `50`, `100`, and `150` pooled entities.
- **Offline FFmpeg H.264 + AAC (`48 kHz`) Multi-Resolution Previews:** Encoded and FFprobe-verified synchronized `.mp4` previews at `1080x1920`, `720x1280`, and `360x640`, plus decoded frame PSNR and BT.709 grayscale readability boards.
- **Automated 20-Positive + 20-Negative QA Suite:** Exercised against real contract/dispatcher/timeline validation logic (`40/40 PASS`).

---

## 2. What ART-008 Does NOT Claim (Explicit Limitations)

1. **Not Production `M06` Completion (`M06 = NOT_STARTED`):**
   - `unity/` production client scenes were intentionally not modified. Wiring `unity/` to live `M05` `/ws/v1/public` streams belongs to `M06`.
2. **Not Connected to Live `M05` WebSocket or `M04` PostgreSQL Runtime:**
   - `M04` backend durability is being developed concurrently on `main`, and `M05` WebSocket endpoints are not yet complete. All `ART-008` timeline and failure runs use deterministic synthetic fixtures explicitly marked `"synthetic_fixture": true`.
3. **Isolated Editor `RenderTexture` CPU Submission Loop != 15-Minute Standalone Player + OBS Soak (`GATE-OBS` Open):**
   - `qa/performance_report.json` measures CPU-side `Camera.Render()` submission into an offscreen `1080x1920` `Direct3D12` `RenderTexture` (`1222.02 FPS` avg, `653.98 FPS` true 1% low, `1.32 ms` p99 at `150` entities; `0.0 B/frame` cumulative thread managed GC via `GC.GetAllocatedBytesForCurrentThread()`, while Editor background housekeeping caused a `~507 ms` 1%-low hitch on the 50-entity batchmode tier). Full async GPU completion timing and `UnityStats.drawCalls` are unavailable in Editor batchmode (`drawCalls = -1`, `NOT_MEASURED`), and offline FFmpeg encoding does **not** replace standalone player execution (`standalone15MinSoakStatus = NOT_EXECUTED`) or `OBS 1080x1920 30fps` hardware capture (`obsHardwareCaptureStatus = NOT_EXECUTED`).
4. **No Physical Smartphone or Real TikTok LIVE Testing:**
   - `360x640` and `720x1280` readability and H.264/AAC compression were verified via actual downscaled/decoded frames on desktop, **not** on physical mobile handsets or live TikTok servers (`P1 LIVE` remains gated).
5. **External Human UX & Listening Study (`NOT EXECUTED`):**
   - Automated typography, contrast (`>= 11.4:1`), LUFS/dBTP, and sync checks passed, and a complete human review pack is provided, but an external 5-person naive viewer comprehension/listening test has **not** been executed.
6. **Legal & Music Rights (`rights_status = PENDING_REVIEW`, `GATE-LEGAL` Open):**
   - All `244` assets are first-party procedural/pixel/UI/DSP creations with zero third-party copyrighted content, but `rights_status` remains `PENDING_REVIEW` until formal human legal sign-off. `MUS01` and `MUS02` remain **muted by default (`-80.0 dB`, `mute = true`)**.
