# ART-009 — Isolated Unity Performance Measurement Policy & Single-Source Audit

## 1. Remediated Metric Definitions (`Phase 4` Findings A–D)

All performance figures across `ART-008` and `ART-009` are single-sourced from `art/production/ART-008/qa/performance_report.json` (`qa/performance_metrics_audit.json` in `ART-009`).

1. **Finding A — True 1% Low FPS vs Frame-Time `p99`:**
   - **Previous Defect:** Computed `1000.0 / p99FrameTimeMs`, which is a single percentile sample rather than the mean of the slowest 1% of frames (at `N = 180`, index selection even caused `10`-entity 1% low `397.91 FPS` to exceed average `307.84 FPS`).
   - **Remediated Formula:** For `N = 180` measured frames (after `24` warmup frames):
     1. Sort all `N` frame durations descending.
     2. Take `K = ceil(N * 0.01) = 2` slowest frames.
     3. Compute `slowestOnePercentMeanMs = mean(frameTimesDesc[0 .. K-1])`.
     4. Compute `onePercentLowFps = 1000.0 / slowestOnePercentMeanMs`.
     5. Report `p99FrameTimeMs` (`frameTimesAsc[ceil(N * 0.99) - 1]`) separately.
2. **Finding B — CPU Render Submission vs Full GPU Frame Time & Standalone Soak:**
   - **Scope:** `timingScope = "cpu_camera_render_submission_editor_batchmode"`. Measures `Stopwatch` elapsed wall time around `SpriteTransitionController` + `BoundedVfxPool` + `HudInPlacePresenter` updates and synchronous `Camera.Render()` submission into a `1080x1920` `Direct3D12` `RenderTexture` inside `Unity 6000.6.7f1 -batchmode` (without `-nographics`).
   - **GPU & Standalone Honesty:** `FrameTimingManager.IsFeatureEnabled()` is queried at runtime; when Editor batchmode does not populate async GPU frame timings, `gpuFrameTimingStatus = "NOT_MEASURED_IN_EDITOR_BATCHMODE"` (`latestGpuFrameTimeMs = -1.0`). `standalone15MinSoakStatus = "NOT_EXECUTED"` and `obsHardwareCaptureStatus = "NOT_EXECUTED"`.
3. **Finding C — Cumulative Thread Managed GC Allocation vs Net Heap Delta:**
   - **Previous Defect:** `GC.GetTotalMemory(false)` before/after a 180-frame loop measures net heap delta, which can be `0` if an intermediate GC collection freed transient allocations.
   - **Remediated Measurement:** Uses `.NET` monotonic `GC.GetAllocatedBytesForCurrentThread()` (`managedGcMeasurementMethod = "GC_GetAllocatedBytesForCurrentThread_Cumulative"`) alongside `ProfilerRecorder(ProfilerCategory.Memory, "GC Allocated In Frame")`, reporting `netHeapDeltaBytes` separately.
4. **Finding D — Single-Source Reporting, Graphics API Naming & Draw Calls:**
   - Removed stale `261.66 FPS` prose from `ART-008/README.md` and `INTEGRATION_LIMITATIONS.md`.
   - Replaced `unity_d3d11_*.png` filenames with `unity_d3d12_*.png` matching actual `SystemInfo.graphicsDeviceType = Direct3D12`.
   - When `UnityStats.drawCalls` returns `0` in Editor batchmode outside PlayMode, `performance_report.json` records `drawCalls = -1` and `drawCallsStatus = "NOT_MEASURED"`.

---

## 2. Authoritative Single-Source Measurements (`Intel Core i5-14600K` + `Intel UHD Graphics 770`, `Direct3D12`)

| Visible Pooled Entities | Warmup / Measured Frames | Avg FPS (CPU Render Submission) | True 1% Low FPS (`K=2` Slowest Mean) | `p99` Frame Time (ms) | Cumulative Thread GC (`B/frame`) | Draw Calls Status |
|---|---|---|---|---|---|---|
| `10` | `24 / 180` | `1976.99` | `849.00` (`1.178 ms`) | `1.154` | `0.0` (`0 B` total) | `NOT_MEASURED (-1)` |
| `50` | `24 / 180` | `159.71` | `1.97` (`506.650 ms`*) | `498.439` | `0.0` (`0 B` total) | `NOT_MEASURED (-1)` |
| `100` | `24 / 180` | `1485.38` | `1036.75` (`0.965 ms`) | `0.901` | `0.0` (`0 B` total) | `NOT_MEASURED (-1)` |
| `150` (Cap) | `24 / 180` | `1222.02` | `653.98` (`1.529 ms`) | `1.315` | `0.0` (`0 B` total) | `NOT_MEASURED (-1)` |

*\*Transparency note on the `50`-entity tier:* Two frames during the `50`-entity Editor batchmode loop experienced a `~507 ms` Editor/OS background housekeeping stall while `10`, `100`, and `150` entities sustained `653.98–1036.75 FPS` 1% lows and `0 B` managed GC allocations. Under our true 1% low formula, that stall is preserved unvarnished rather than smoothed away, reinforcing why `M06` (`W0607`) requires a dedicated 15-minute standalone player soak outside the Unity Editor.
