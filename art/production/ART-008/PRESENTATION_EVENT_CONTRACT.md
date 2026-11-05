# ART-008 — Non-Authoritative Presentation Event & Deduplication Contract

## 1. Strict Authority Boundary

```text
Committed Backend State / Outbox Event (M03/M04/M05)
   └──► Verified Client State-Diff & Dedupe Adapter (ttg-audio-dedupe-v2)
           └──► Visual State / 12 FPS Animation / VFX Burst / 48 kHz Audio Cue
```

- **Presentation NEVER computes gameplay authority:** The art/presentation layer never calculates HP damage, wave timers, vote winners, Boss spawns, or participant acceptance.
- **No speculative feedback:** Raw TikTok comments or unverified client timers never trigger registration toasts (`SFX01`), Boss appearances (`< tick 2600`), or `RESULTS_WIN` / `RESULTS_LOSS` (`< tick 2800`).

---

## 2. Two-Tier Presentation Deduplication (`ttg-audio-dedupe-v2`)

Reconnect resyncs and outbox redeliveries can replay already-delivered events, while ordinary waves legitimately emit many independent events of the same type (e.g., 3 separate wall hits, repeated Fighter slashes, repeated Builder repairs, multiple Boss slams).

1. **Tier 1 — Exact Committed Delivery Identity (`All Event-Driven Cues`):**
   - Key: `{session_id}:{stream_seq}` stored in a `256`-entry FIFO ring buffer plus per-session monotonic `max_seen_stream_seq` floor (`stream_seq <= max_seen_stream_seq - 256` rejected as stale replay).
   - **Result:** Exact duplicate deliveries of `(session_id, stream_seq)` play `1` time total; two distinct committed events `(sess, 800)` and `(sess, 801)` of type `damage_applied` or `attack_resolved` are **never** treated as duplicates and both remain playable subject only to acoustic cooldown (`150–800 ms`).
2. **Tier 2 — Genuine Once-Per-Wave Milestones (`Restricted Set Only`):**
   - Key: `{session_id}:{wave_no}:{canonical_milestone}` restricted strictly to:
     - `vote_closed` (`SFX10`)
     - `boss_incoming` (`SFX08`)
     - `boss_spawned` (`SFX09` initial entrance; subsequent Boss combat slams use `boss_combat_slam` on Tier 1 only!)
     - `wave_result` (`SFX11` Victory / `SFX12` Defeat)
   - **Result:** Even if a reconnect snapshot re-projects `wave_result` under a higher `stream_seq`, Victory/Defeat stings never double-fire in the same `(session_id, wave_no)`.

---

## 3. Payload Sufficiency Classification (`simulation.Event` & `05_API_WEBSOCKET.md`)

| Cue ID | Presentation Event | Data Source Classification | M05 / M06 Adapter Requirement |
|---|---|---|---|
| `SFX01` | Role Join Chime + Single-Slot Toast | `1. Committed simulation.Event` (`role_accepted`) | Project sanitized `display_alias <= 12 chars` + `role` + `stream_seq` (never `ActorKey`). Throttle `2000ms` PREP / `3500ms @ -5.5 dB` ASSAULT; suppress during active `Alerts`. |
| `SFX02` | Builder Repair + `vfx_builder_repair_spark` | `2+4. Committed Event + Client Diff` | `role_resolution` / snapshot diff where `counts.builder > 0` and `wall_hp` increases (or active repair tick); align VFX+SFX to `chr_role_builder_repair` frame `3` (`+250ms`). |
| `SFX03` | Guard Block + `vfx_guard_shield_block` | `2+4. Committed Event + Client Diff` | `damage_applied` / `attack_resolved` while `counts.guard > 0` and guard mitigation applies; align to `chr_role_guard_shieldbrace` frame `2` (`+166.7ms`). |
| `SFX04` | Fighter Slash + `vfx_fighter_slash_arc` | `1+3. Committed Event + 12 FPS Local Sync` | `attack_resolved` while `counts.fighter > 0`; align VFX+SFX to `chr_role_fighter_swordslash` frame `3` (`+250ms`). |
| `SFX05` | Enemy Attack + `vfx_unit_hit_puff` | `1+3. Committed Event + 12 FPS Local Sync` | `damage_applied` while lane `ordinary_hp > 0`; align to enemy action impact frame (`f02` Runner, `f03` Grunt/Brute). |
| `SFX06` | Wall Damage (`state 1->2->3`) | `2+4. Snapshot Diff Adapter` | Client state-diff tracker detects `curr.wall_hp < prev.wall_hp` and `curr.wall_hp > 0`. (`maxConcurrentInstances = 2`, `cooldown = 350ms`). |
| `SFX07` | Wall Breach (`state 4`) | `2+4. Snapshot Zero-Crossing Adapter` | Client state-diff tracker detects `prev.wall_hp > 0 && curr.wall_hp == 0`. |
| `SFX08` | Boss Incoming Warning (`tick 2450`) | `2. Public Snapshot` (`boss.boss_incoming`) | Trigger once per Boss wave when `wave_no % 5 == 0`, `2450 <= tick < 2600`, `boss_incoming == true` (ground telegraph only; zero Boss sprite). |
| `SFX09` | Boss Spawn (`tick 2600`) & Boss CleaveSlam | `2+3. Snapshot + Finale Combat Beat` | Spawn entrance at `tick >= 2600` (`boss_exists == true`), plus repeatable Finale Boss combat beats aligned to `cleaveslam` frame `5` (`+416.7ms`). |
| `SFX10` | Even-Wave Vote Closed (`tick 1200`) | `1+2. Committed Event / Vote Lock` | Trigger once per even wave when `vote.open` transitions `true -> false` with `vote.selected in {A,B,C}` at `tick 1200`. |
| `SFX11` | Wave Victory (`RESULTS_WIN`) | `2+4. Committed Settlement Projection` | Trigger once per wave at `tick >= 2800` when `settlement.wave_result == "win"` (`QA-SIM-12` public projection required in `M05`). |
| `SFX12` | Wave Defeat (`RESULTS_LOSS`) | `2+4. Committed Settlement Projection` | Trigger once per wave when `settlement.wave_result == "loss"` (`tick >= 2800` or authoritative `townhall_hp == 0`). |
| `AMB01` | Village Dawn Ambient Loop (`20.0s`) | `3. Scene Lifecycle` | Start seamlessly on showcase/session load (`-3.0 dB` Ambient bus); respect global mute. |
| `MUS01` / `MUS02` | Prep / Assault Music Loops | `Gated by GATE-LEGAL` | **Keep muted (`-80.0 dB`, `mute = true`) by default in all builds until `GATE-LEGAL` written clearance.** |
