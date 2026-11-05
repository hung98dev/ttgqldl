# ART-007 — Original SFX, Ambient & Music Production Package (`SFX001`)

**Status:** `PASS` (Production Audio Assets, Contracts, Isolated Unity `6000.6.7f1` Verification & Adversarial QA Complete)  
**Requirement Authority:** `SFX001` in `docs/plan/07_UNITY_ART_UX.md`  
**Visual/Audio Style:** Bastion Dawn v1 — Stylized Fantasy Village Defense  
**Target Engine:** Unity `6000.6.7f1`, Built-in 2D (`M06` remains `NOT_STARTED`; `unity/` untouched)  
**Rights & Stream Gates:** `rights_status = PENDING_REVIEW` (`GATE-LEGAL` & `GATE-OBS` remain open; **Music is MUTED by default at `-80.0 dB`**)

---

## 1. Deliverable Inventory (15 Playable Original Audio Masters + Runtime + Editable Sources)

All 15 master audio files are **100% original, deterministic `48,000 Hz` / `24-bit` PCM WAVs** synthesized from physical-modeling equations in [dsp_engine.py](tools/dsp_engine.py) and [build_audio.py](tools/build_audio.py) with zero external audio samples, zero SoundFonts, and zero clipped samples:

### A. 12 Canonical Game SFX (`sfx/wav/*.wav` + `sfx/runtime/*.ogg`)

| Cue ID | Master WAV | Runtime Ogg | Ch | Duration | True Peak | Integrated Loudness | Synchronized Event / Animation Frame |
|---|---|---|---|---|---|---|---|
| `SFX01` | `sfx/wav/sfx_role_join.wav` | `sfx/runtime/sfx_role_join.ogg` | Mono | `0.42s` | `-5.50 dBTP` | `-14.36 LUFS` | Durable `role_accepted` / burst `join_toast` (`2.0s` PREP / `3.5s` ASSAULT cooldown) |
| `SFX02` | `sfx/wav/sfx_builder_repair.wav` | `sfx/runtime/sfx_builder_repair.ogg` | Mono | `0.38s` | `-3.20 dBTP` | `-20.03 LUFS` | `Builder.repair` clip `f03` (`sheet 13`, `250.0 ms`) + `vfx_builder_repair_spark[f00]` |
| `SFX03` | `sfx/wav/sfx_guard_block.wav` | `sfx/runtime/sfx_guard_block.ogg` | Mono | `0.45s` | `-2.80 dBTP` | `-19.49 LUFS` | `Guard.shieldbrace` clip `f02` (`sheet 12`, `166.7 ms`) + `vfx_guard_shield_block[f00]` |
| `SFX04` | `sfx/wav/sfx_fighter_slash.wav` | `sfx/runtime/sfx_fighter_slash.ogg` | Mono | `0.36s` | `-3.00 dBTP` | `-19.74 LUFS` | `Fighter.swordslash` clip `f03` (`sheet 13`, `250.0 ms`) + `vfx_fighter_slash_arc[f01]` |
| `SFX05` | `sfx/wav/sfx_enemy_attack.wav` | `sfx/runtime/sfx_enemy_attack.ogg` | Mono | `0.32s` | `-4.80 dBTP` | `-21.64 LUFS` | `Grunt[f03]` (`0st`), `Runner[f02]` (`+2st`), `Brute[f03]` (`-3st`) single-asset variation |
| `SFX06` | `sfx/wav/sfx_wall_damage.wav` | `sfx/runtime/sfx_wall_damage.ogg` | Mono | `0.58s` | `-3.50 dBTP` | `-22.16 LUFS` | Committed `damage_applied` (`wall_loss > 0 && new_wall_hp > 0`) |
| `SFX07` | `sfx/wav/sfx_wall_breach.wav` | `sfx/runtime/sfx_wall_breach.ogg` | Stereo | `1.28s` | `-1.80 dBTP` | `-17.43 LUFS` | Committed `damage_applied` (`prev_wall_hp > 0 && new_wall_hp == 0`) |
| `SFX08` | `sfx/wav/sfx_boss_warning.wav` | `sfx/runtime/sfx_boss_warning.ogg` | Stereo | `1.86s` | `-2.80 dBTP` | `-12.47 LUFS` | Authoritative `boss_incoming` (`wave_no % 5 == 0`, pre-tick `2600`, once/wave) |
| `SFX09` | `sfx/wav/sfx_boss_slam.wav` | `sfx/runtime/sfx_boss_slam.ogg` | Stereo | `1.16s` | `-1.50 dBTP` | `-17.38 LUFS` | `Boss.cleaveslam` clip `f05` (`sheet 11`, `416.7 ms`) + `vfx_boss_slam_telegraph[f05]` |
| `SFX10` | `sfx/wav/sfx_vote_closed.wav` | `sfx/runtime/sfx_vote_closed.ogg` | Mono | `0.48s` | `-5.20 dBTP` | `-18.52 LUFS` | Authoritative `vote_closed` at tick `1200` (neutral fifth; never implies A/B/C early) |
| `SFX11` | `sfx/wav/sfx_wave_victory.wav` | `sfx/runtime/sfx_wave_victory.ogg` | Stereo | `2.45s` | `-2.00 dBTP` | `-11.49 LUFS` | Settled `wave_result == victory` at tick `>= 2800` (deduped once/wave) |
| `SFX12` | `sfx/wav/sfx_wave_defeat.wav` | `sfx/runtime/sfx_wave_defeat.ogg` | Stereo | `2.35s` | `-3.20 dBTP` | `-11.98 LUFS` | Settled `wave_result == loss` at tick `>= 2800` (dignified Dorian resolve) |

### B. 1 Original Seamless Ambient Loop & 2 Original Music Loops (**Music Muted By Default**)

| Cue ID | Master WAV (`48 kHz / 24-bit` Stereo) | Runtime Ogg | Editable Source(s) | Specs & Default State |
|---|---|---|---|---|
| `AMB01` | `ambient/wav/amb_village_dawn_loop.wav` | `ambient/runtime/amb_village_dawn_loop.ogg` | [amb_village_dawn_loop_project.json](source/ambient/amb_village_dawn_loop_project.json) | `36.00s` (`-26.00 LUFS`, `-12.64 dBTP`, seam step ratio `0.024×`); `3×` repeat in `previews/loops/amb_village_dawn_x3_seam_check.ogg` |
| `MUS01` | `music/wav/music_village_prep_loop.wav` | `music/runtime/music_village_prep_loop.ogg` | `source/music/music_village_prep_loop.mid`, [music_village_prep_loop_score.json](source/music/music_village_prep_loop_score.json) | `100 BPM`, `4/4`, `16 Bars = 38.40s` (`-19.50 LUFS`, D Mixolydian) — **MUTED BY DEFAULT (`-80 dB`)** |
| `MUS02` | `music/wav/music_village_assault_loop.wav` | `music/runtime/music_village_assault_loop.ogg` | `source/music/music_village_assault_loop.mid`, [music_village_assault_loop_score.json](source/music/music_village_assault_loop_score.json) | `128 BPM`, `4/4`, `20 Bars = 37.50s` (`-17.50 LUFS`, D Dorian) — **MUTED BY DEFAULT (`-80 dB`)** |

---

## 2. Package Architecture & Documentation Index

- [AUDIO_STYLE_GUIDE.md](AUDIO_STYLE_GUIDE.md) — Bastion Dawn v1 acoustic material language, musical architecture, and prohibitions.
- [AUDIO_MIXING_POLICY.md](AUDIO_MIXING_POLICY.md) — 12-voice budget, 10k-viewer burst anti-spam rules, sidechain ducking, and loudness targets.
- [UNITY_AUDIO_HANDOFF.md](UNITY_AUDIO_HANDOFF.md) — Differentiated `AudioImporter` YAML presets, `AudioMixer` bus hierarchy, and isolated `Unity 6000.6.7f1` results.
- [APPROVAL_RECORD.md](APPROVAL_RECORD.md) — First-party procedural provenance, `PENDING_REVIEW` status, `GATE-LEGAL` & `GATE-OBS` boundaries.
- [qa/qa_report.md](qa/qa_report.md) — 18 automated QA checks, 12 adversarial negative cases, multimodal listening review, and AAC stream proxy evidence.
- Machine-readable contracts & evidence:
  - [asset_manifest.json](asset_manifest.json)
  - [audio_cue_mapping.json](audio_cue_mapping.json)
  - [audio_mixer_config.json](audio_mixer_config.json)
  - [qa/qa_metrics.json](qa/qa_metrics.json)
  - [qa/adversarial_validator_results.json](qa/adversarial_validator_results.json)
  - [qa/stream_encode_metrics.json](qa/stream_encode_metrics.json)
  - [qa/unity_isolated_verification.json](qa/unity_isolated_verification.json)

---

## 3. Deterministic Reproduction Commands

```powershell
python -B art/production/ART-007/tools/build_audio.py
python -B art/production/ART-007/tools/render_audio_previews.py
python -B art/production/ART-007/tools/validate_audio.py --write-manifest
python -B art/production/ART-007/tools/validate_audio.py
```
