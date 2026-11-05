# ART-007 — Audio Mixing, Mastering & 10k-Viewer Anti-Spam Policy

**Package:** `ART-007` (`SFX001`)  
**Machine-Readable Contracts:** [audio_mixer_config.json](audio_mixer_config.json) | [audio_cue_mapping.json](audio_cue_mapping.json)

---

## 1. Presentation-Only & Anti-Spam Rules for High-Concurrency Streams

TTGQLDL can receive thousands of viewer comments per minute alongside three-lane combat. The audio subsystem obeys strict deterministic presentation boundaries:

1. **Zero Per-Comment Chime Spam (`SFX01` Burst Aggregation):**
   - Duplicate same-role comments, cooldown-rejected role switches, invalid comments, and uncommitted local inputs **NEVER** trigger audio.
   - Even when hundreds of valid `role_accepted` events commit simultaneously, `SFX01` (`sfx_role_join.wav`) is burst-aggregated to match `ART-006`'s `join_toast` window: at most **1 gentle cue per `2000 ms`** during `PREP`, attenuated by **`-5.5 dB` with a `3500 ms` cooldown** during `ASSAULT`, and suppressed completely during active `Alerts` (`Boss Warning`, `Boss Slam`, `Wall Breach`, `Wave Result`).
2. **Single-Asset Bounded Variation (`SFX05` Enemy Attack):**
   - Rather than playing identical full-volume strikes on every enemy swing or creating untracked extra files, `sfx_enemy_attack.wav` applies deterministic cosmetic pitch/gain offsets derived from `SHA256(session_id:wave_no:visual_event_seq)` (never touching authoritative simulation RNG):
     - **Grunt (`clubsmash[f03]`):** `0.0 st`, `0.0 dB`
     - **Runner (`daggerlunge[f02]`):** `+2.0 st`, `-1.5 dB`
     - **Brute (`gauntletcrush[f03]`):** `-3.0 st`, `+1.0 dB`
     - **Micro-jitter:** `[-0.50, -0.25, 0.0, +0.25, +0.50] st` with `150–260 ms` cooldown gating.
3. **Reconnect & Outbox Replay Deduplication:**
   - `BroadcastAudioManager` maintains a session-scoped `256`-entry ring buffer keyed by `{session_id}:{wave_no}:{canonical_event_type}` plus monotonic `last_audio_stream_seq`.
   - Redelivered outbox events after WebSocket reconnect (`vote_closed`, `boss_incoming`, `wall_breached`, `wave_result`) are suppressed so result or warning stings never double-fire.

---

## 2. Preallocated 12-Voice Budget & Priority Hierarchy

Dynamic `AudioSource` instantiation (`AddComponent` / `Instantiate` per event) is **forbidden**. Exactly **12 pooled `AudioSource` voices** are preallocated at startup:

| Pool / Bus | Voices | Unity Priority (`0` = Highest) | Stealing & Ducking Policy |
|---|---|---|---|
| `Alerts` (`SFX07..SFX12`) | `2` | `8 – 40` | Never stolen; triggers automatic sidechain ducking (`-5.0 dB` on `Music`, `-3.0 dB` on `Ambient`, `35 ms` attack / `250 ms` hold / `420 ms` release) |
| `SFX` (`SFX01..SFX06`) | `8` | `64 – 120` | Bounded per-cue cooldowns + max concurrent instances (`1–3`); steals oldest lowest-priority combat voice only if all 8 slots are occupied |
| `Ambient` (`AMB01`) | `1` | `140` | Continuous seamless loop; attenuates `-6.0 dB` when operator pauses session; fades out over `400 ms` on `stopped` |
| `Music` (`MUS01`, `MUS02`) | `1` | `160` | **MUTED BY DEFAULT (`-80.0 dB`, `mute = true`)** until `GATE-LEGAL` stream music approval |

One-click global mute (`MasterVolumeDb = -80 dB` / `AudioListener.pause`) aligns directly with the operator control contract `POST /admin/v1/sessions/{id}/settings` (`mute_audio: bool`).

---

## 3. Internal Engineering Loudness & Headroom Targets

> **Important Honesty Note:** The LUFS and dBTP figures below are **internal TTGQLDL audio-engineering targets** chosen for clean mobile speaker translation, headroom under 10k-viewer overlap, and artifact-free AAC encoding. They are **not** claimed as official TikTok platform specifications.

| Signal Class | Target Integrated Loudness | Max True Peak (`4×` OS) | Measured ART-007 Result |
|---|---|---|---|
| Individual Short SFX (`SFX01..SFX12`) | `-22.5` to `-11.0 LUFS` | `<= -1.50 dBTP` | `-22.16` to `-11.49 LUFS`, `-5.50` to `-1.50 dBTP`, `0` clips |
| Village Dawn Ambient (`AMB01`) | `-26.0 LUFS` (`±2.0 LU`) | `<= -8.50 dBTP` | `-26.00 LUFS`, `-12.64 dBTP`, `0` clips |
| Prep & Assault Music (`MUS01`, `MUS02`) | `-19.5` / `-17.5 LUFS` | `<= -2.20 dBTP` | `-19.50` / `-17.50 LUFS`, `-7.64` / `-4.87 dBTP` (**Default Muted**) |
| Combined Gameplay Montage (`PCM`) | `-18.5` to `-14.0 LUFS` | `<= -1.00 dBTP` | `-16.43 LUFS`, `-1.91 dBTP`, `+12.23 dB` SFX-to-bed margin |
| 10,000-Viewer Stress Burst (`8,500` raw events) | Controlled, unclipped | `<= -1.00 dBTP` | `0` clipped samples (`>98%` spam events suppressed cleanly) |
| Offline AAC-LC Proxy (`128 kbps` & `96 kbps`) | Preserve loudness & peaks | `<= -0.50 dBTP` | `128k`: `-16.46 LUFS`, `-1.75 dBTP`, `0` clips (`96k`: `-0.88 dBTP`, `0` clips) |
