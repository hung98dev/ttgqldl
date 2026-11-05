# ART-006 — Authoritative 16-State HUD & Compositional Priority Mapping

Machine-readable contract: [`ui_state_mapping.json`](ui_state_mapping.json).

---

## 1. Correction of `ART-002` Tick Prose vs Normative Gameplay Rules

> [!WARNING]
> **Normative Correction Recorded (`Phase 5`):**
> `art/concepts/ART-002/04_ux_ui_and_states.md` labeled ordinary `PREP` in prose as ticks `0..999` and treated `PREP_60_WARN` / `PREP_15_WARN` / `VOTE_OPEN` as if they replaced the phase.
> Per [`docs/plan/02_GAME_RULES.md`](../../../docs/plan/02_GAME_RULES.md) and [`docs/plan/03_SIMULATION.md`](../../../docs/plan/03_SIMULATION.md), the **authoritative 10 Hz (`3000`-tick) wave phases** are strictly:
> - `WAITING`: `wave_tick = -1` (`P = 0`, clock frozen at `05:00`, zero unattended waves or fake bots)
> - `PREP`: **`wave_tick = 0..1599`** (`[0s, 160s)`)
> - `ASSAULT`: **`wave_tick = 1600..2599`** (`[160s, 260s)`)
> - `FINALE`: **`wave_tick = 2600..2799`** (`[260s, 280s)`)
> - `RESULTS`: **`wave_tick = 2800..2999`** (`[280s, 300s)`)
> - `ROLLOVER CONTROL`: **`wave_tick = 3000`** (control transition; never an extra combat tick)
>
> Accordingly, in `ART-006`:
> - `PREP_60_WARN` (`tick 1000..1449`) and `PREP_15_WARN` (`tick 1450..1599`) are **visual countdown substates within authoritative `phase == "prep"`**.
> - `VOTE_OPEN` (`wave_no % 2 == 0` and `800 <= wave_tick <= 1199`, i.e., `[80s, 120s)`) is an **orthogonal subordinate strip overlay** that coexists with `PREP` and `PREP_60_WARN` while keeping `Bình luận XÂY / THỦ / ĐÁNH` dominant below it.

---

## 2. Eight-Slot Compositional HUD Model (Non-Exclusive Priority System)

Rather than forcing all 16 presentation states into a brittle single enum that destroys valid combinations, `ART-006` resolves HUD presentation across **8 orthogonal slots**:

| Slot # | Slot ID | Priority / Resolution Rule |
|---:|---|---|
| **1** | `session_gate` | `MAINTENANCE` when `lifecycle == "stopped"`; `ATTRACT` when pre-session attract mode (`P=0`, `MOCK` watermark mandatory); otherwise `ACTIVE_SESSION`. |
| **2** | `freeze_blockers` | `pause_reasons` is a coexisting set (`QA-OPS-04`). Simulation clock stays frozen until **all** blockers clear. Primary Threat Banner displays highest-severity blocker (`storage_unavailable > source_disconnected > operator_paused`), while `secondary_diag_pill` surfaces coexisting blockers or active Boss truth. |
| **3** | `base_gameplay_phase` | `WAITING_FOR_VIEWER` (`tick -1`), `PREP` (`0..1599`), `ASSAULT` (`1600..2599`), `FINALE` (`2600..2799`), `RESULTS_WIN` / `RESULTS_LOSS` (`2800..2999`, settled at tick `2800`). |
| **4** | `prep_warning_substate` | Active only when `phase == "prep"` and unfrozen: `PREP_60_WARN` (`1000..1449`), `PREP_15_WARN` (`1450..1599`). |
| **5** | `boss_presentation_mode` | Only on `wave_no % 5 == 0`. Before `tick 2600`: `boss_exists == false` and `boss_hp == 0` mandatory; `BOSS_INCOMING` warning shown at `tick >= 2450` or when ordinary lanes hit `0` early (**`CLEAR` banner strictly forbidden!**). At `tick >= 2600` (`finale`): `BOSS_ACTIVE` spawns Boss sprite and renders embedded Boss HP bar. |
| **6** | `subordinate_strip` | `VOTE_OPEN` iff `wave_no % 2 == 0` and `800 <= wave_tick <= 1199`; `NEXT_WAVE_QUEUE` during `RESULTS` when `pending_next_wave_count > 0`; `VOTE_LOCKED` when buff chosen; otherwise wave objective/paused hint. Primary role CTA always stays visible below. |
| **7** | `ingress_degradation` | **`W0306` Discipline:** `DEGRADED_DROPS` (`"QUÁ TẢI · LỆNH HỢP LỆ RƠI"`) is shown **only** when `metrics_available == true` AND `confirmed_valid_drops > 0`. High queue fill (`>=75%`) or retryable rejections with `confirmed_valid_drops == 0` show `BACKPRESSURE_NO_LOSS` (`"0 lệnh rơi"`). `metrics_available == false` shows `METRICS_UNKNOWN_STALE` without inventing `0`. |
| **8** | `honest_disclosures` | Mandatory `MOCK / REPLAY` footer whenever `source_status != "connected"`; `System Militia (P≤10)` badge when `1 <= P <= 10` (never counted in viewer `P`); `-15%` / `-30%` post-loss recovery badge when `is_post_loss_recovery == true`. |

---

## 3. Canonical 16 HUD States (`hud/states/`)

| # | Canonical State ID | Fixture | Rendered Output (`1080×1920`) | Authoritative Semantics |
|---:|---|---|---|---|
| 01 | `ATTRACT` | `state_01_attract.json` | `hud/states/01_attract.png` | `P=0`, `tick=-1`, calm village showcase, prominent `MOCK` watermark, zero fake combat. |
| 02 | `WAITING_FOR_VIEWER` | `state_02_waiting_for_viewer.json` | `hud/states/02_waiting_for_viewer.png` | `P=0`, `tick=-1`, clock frozen at `05:00`, zero fake bots or unattended loss. |
| 03 | `PREP` | `state_03_prep.json` | `hud/states/03_prep.png` | Wave 3, `tick=600` (`0..1599`), Builders repairing scaffolds, `P=8` (`System Militia` disclosed), single Join Toast. |
| 04 | `PREP_60_WARN` | `state_04_prep_60_warn.json` | `hud/states/04_prep_60_warn.png` | Wave 3, `tick=1000` (visual substate inside `phase="prep"`), 60s amber assault countdown. |
| 05 | `PREP_15_WARN` | `state_05_prep_15_warn.json` | `hud/states/05_prep_15_warn.png` | Wave 4, `tick=1450` (visual substate inside `phase="prep"`), urgent 15s crimson warning + locked Vote `[B]`. |
| 06 | `VOTE_OPEN` | `state_06_vote_open.json` | `hud/states/06_vote_open.png` | Even Wave 4, `tick=950` (`800..1199`), subordinate `A/B/C` tally strip above dominant `XÂY/THỦ/ĐÁNH` CTA. |
| 07 | `ASSAULT` | `state_07_assault.json` | `hud/states/07_assault.png` | Wave 7, `tick=1960`, simultaneous West/Center/East lane combat + live HP & Join Toast. |
| 08 | `FINALE` | `state_08_finale.json` | `hud/states/08_finale.png` | Non-Boss Wave 8, `tick=2660` (`2600..2799`), dusk finale clash before tick-2800 settlement. |
| 09 | `BOSS` | `state_09_boss.json` | `hud/states/09_boss.png` | Wave 10, `tick=2620` (`>=2600`), Center Causeway Boss (`88/88 HP`), embedded Boss HP bar, burst Join Toast (`+12`). |
| 10 | `RESULTS_WIN` | `state_10_results_win.json` | `hud/states/10_results_win.png` | Wave 10 settled win (`tick=2860`), `+100G +60Đ +50L` reward + `14` queued next-wave registrations. |
| 11 | `RESULTS_LOSS` | `state_11_results_loss.json` | `hud/states/11_results_loss.png` | Wave 5 breach (`tick=2880`), `250 HP` floor + `-15%` next-wave enemy HP recovery relief disclosed. |
| 12 | `SOURCE_RECONNECTING` | `state_12_source_reconnecting.json` | `hud/states/12_source_reconnecting.png` | `pause_reasons=["source_disconnected"]`, clock frozen, stale indicator, zero false state wipe. |
| 13 | `PAUSED` | `state_13_paused.json` | `hud/states/13_paused.png` | `pause_reasons=["operator_paused"]`, simulation frozen cleanly by operator. |
| 14 | `STORAGE_UNAVAILABLE` | `state_14_storage_unavailable.json` | `hud/states/14_storage_unavailable.png` | `pause_reasons=["storage_unavailable"]`, highest-priority crimson DB safe-halt banner. |
| 15 | `DEGRADED_DROPS` | `state_15_degraded_drops.json` | `hud/states/15_degraded_drops.png` | `metrics_available=true` & `confirmed_valid_drops=17` (>0) — truthful confirmed-loss alert. |
| 16 | `MAINTENANCE` | `state_16_maintenance.json` | `hud/states/16_maintenance.png` | `lifecycle="stopped"`, `P=0`, timer `--:--`, clean session-stopped maintenance screen. |

---

## 4. Eight Verified Overlay Combinations (`hud/overlays/`)

1. **`combo_01_prep_60_plus_vote_open`:** Even Wave 6 at `tick=1050` — `PREP_60_WARN` banner + `VOTE_OPEN` strip + dominant primary CTA simultaneously.
2. **`combo_02_prep_plus_operator_paused`:** Even Wave 6 paused during voting (`tick=920`) — `PAUSED` banner freezes clock & vote window cleanly.
3. **`combo_03_assault_plus_source_disconnected`:** Wave 9 `ASSAULT` frozen by `source_disconnected` — preserves defender counts and lane HP truth while frozen.
4. **`combo_04_boss_incoming_pre2600_lanes_empty`:** Wave 10 at `tick=2480` with all ordinary lanes `0 HP` — shows `BOSS_INCOMING` warning, **does not spawn Boss sprite early**, and **forbids premature `CLEAR` banner**.
5. **`combo_05_boss_plus_storage_unavailable`:** Wave 15 active Boss (`62/104 HP`) interrupted by `storage_unavailable` — primary banner shows DB safe halt while secondary diagnostic pill preserves Boss HP truth.
6. **`combo_06_results_plus_next_wave_registration`:** `RESULTS_WIN` (`tick=2910`) with `23` queued next-wave comments + `+1 ĐỢT SAU` Join Toast without inflating current-wave `P=106`.
7. **`combo_07_backpressure_zero_valid_drops`:** `84%` queue fill and `62` retryable rejections with `confirmed_valid_drops=0` — shows amber `HÀNG ĐỢI 84% · Điều tiết (0 lệnh rơi)`, **never** falsely claiming `"LỆNH RƠI"`.
8. **`combo_08_source_reconnected_operator_pause_retained` (`QA-OPS-04`):** Source reconnected (`replay`), yet `operator_paused` remains in `pause_reasons` — simulation remains `PAUSED` until operator explicitly resumes.
