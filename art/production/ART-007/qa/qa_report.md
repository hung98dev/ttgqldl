# ART-007 — Reproducible Audio QA, Adversarial Validation & Listening Review Report

**Package:** `ART-007` (`SFX001`)  
**QA Verdict:** `PASS` (`18/18` Automated Checks Passed, `12/12` Adversarial Negative Cases Rejected, `15/15` Unity `6000.6.7f1` Clips Verified)  
**Machine-Readable Evidence:** [qa_metrics.json](qa_metrics.json) | [adversarial_validator_results.json](adversarial_validator_results.json) | [stream_encode_metrics.json](stream_encode_metrics.json) | [unity_isolated_verification.json](unity_isolated_verification.json)

---

## 1. Automated 18-Check QA Matrix ([validate_audio.py](../tools/validate_audio.py))

All 18 mandatory Phase-8 checks pass with `0` errors:

1. **`01_twelve_canonical_sfx_exist` (`PASS`):** Exactly 12 canonical `48 kHz / 24-bit` WAVs in `sfx/wav/` and 12 runtime `.ogg` files in `sfx/runtime/`.
2. **`02_one_ambient_loop_exists` (`PASS`):** `amb_village_dawn_loop.wav` + `.ogg` + `3×` seamless repeat preview `amb_village_dawn_x3_seam_check.ogg`.
3. **`03_two_music_loops_exist` (`PASS`):** `music_village_prep_loop.wav` & `music_village_assault_loop.wav` + `.ogg` + Format-1 Standard MIDI `.mid` files.
4. **`04_files_decode_correctly` (`PASS`):** All 15 WAVs, 15 Ogg Vorbis streams (`ffprobe`), and 2 Standard MIDI (`MThd`) files decode without error.
5. **`05_sample_rate_48khz_24bit` (`PASS`):** `48,000 Hz` / `24-bit` (`pcm_s24le`) across all 15 masters; `48,000 Hz` Vorbis across all 15 runtime assets.
6. **`06_channel_count_valid` (`PASS`):** Mono (`1ch`) on `SFX01..SFX06, SFX10`; restrained Stereo (`2ch`) on `SFX07..SFX09, SFX11..SFX12, AMB01, MUS01, MUS02`.
7. **`07_no_clipping` (`PASS`):** `0` clipped samples across all 15 masters and gameplay mixes; max True Peak is `-1.50 dBTP` (`SFX09`).
8. **`08_no_unexpected_dc_offset` (`PASS`):** `|DC offset| <= 0.000001` across all 15 masters.
9. **`09_no_silent_placeholder` (`PASS`):** Every asset contains active synthesized audio (`RMS > -29 dBFS`, `Peak >= -13.1 dBFS`).
10. **`10_no_corrupted_or_zero_byte_audio` (`PASS`):** Valid `RIFF/WAVE`, `OggS`, and `MThd` headers and non-trivial payloads on every file.
11. **`11_no_attack_or_tail_truncation` (`PASS`):** Clean zero-crossing cosine boundaries (`first_sample_abs = 0.0`, `last_sample_abs = 0.0`) on all 12 one-shot SFX.
12. **`12_loop_seam_continuity` (`PASS`):** Sample-accurate periodic RFFT + modulo-`N` wrap across `AMB01` (`seam_step_ratio = 0.024×`), `MUS01` (`0.103×`), and `MUS02` (`0.023×`) with zero click or crossfade volume dip.
13. **`13_asset_hashes_match_manifest` (`PASS`):** 100% of WAV and OGG SHA-256 hashes match [../asset_manifest.json](../asset_manifest.json).
14. **`14_every_cue_mapped_and_domain_safe` (`PASS`):** All 15 cues uniquely mapped in [../audio_cue_mapping.json](../audio_cue_mapping.json); exact `ART-003/004` sheet indices (`Builder.repair[13]`, `Guard.shieldbrace[12]`, `Fighter.swordslash[13]`, `Boss.cleaveslam[11]`) and `SFX11` tick-`2800` `wave_result == victory` guard verified.
15. **`15_complete_provenance_pending_review` (`PASS`):** Complete Phase-11 provenance fields and `rights_status = "PENDING_REVIEW"` verified on every asset.
16. **`16_music_muted_by_default` (`PASS`):** `music_muted_by_default = true` and `MusicVolumeDb = -80.0 dB` verified across mixer, cue contract, and Unity prefab.
17. **`17_deterministic_rebuild_reproducibility` (`PASS`):** Re-synthesizing from scratch produces bit-identical SHA-256 WAV outputs (`max_diff = 0.0`).
18. **`18_no_unlicensed_sources_bundled` (`PASS`):** Zero `.sf2`, `.sf3`, `.dls`, `.mp3`, `.flac`, `.aiff` or external sample files exist anywhere in the package.

---

## 2. Adversarial Negative QA Suite (12 Deliberate Defects Rejected)

Executed in isolated temporary sandboxes ([adversarial_validator_results.json](adversarial_validator_results.json)); all **12 / 12** defects were caught and rejected (`0` escapes):

1. `adv_01_missing_sfx` — Deleted `sfx_guard_block.wav` -> **REJECTED**
2. `adv_02_corrupted_wav_header` — Corrupted `RIFF` magic bytes on `sfx_role_join.wav` -> **REJECTED**
3. `adv_03_wrong_sample_rate` — Re-encoded `sfx_vote_closed.wav` at `44,100 Hz` -> **REJECTED**
4. `adv_04_clipped_pcm_samples` — Injected `0.0 dBFS` clipped samples into `sfx_boss_slam.wav` -> **REJECTED**
5. `adv_05_silent_placeholder` — Replaced `sfx_fighter_slash.wav` with digital silence -> **REJECTED**
6. `adv_06_incorrect_asset_hash` — Tampered SHA-256 digest in `asset_manifest.json` -> **REJECTED**
7. `adv_07_missing_cue_mapping` — Removed `SFX09` from `audio_cue_mapping.json` -> **REJECTED**
8. `adv_08_duplicate_cue_id` — Duplicated `SFX01` in `audio_cue_mapping.json` -> **REJECTED**
9. `adv_09_broken_loop_seam` — Injected boundary step discontinuity at sample `0` of `amb_village_dawn_loop.wav` -> **REJECTED**
10. `adv_10_missing_rights_metadata` — Stripped `tool_and_version` and falsified `rights_status` -> **REJECTED**
11. `adv_11_music_enabled_by_default` — Unmuted `Music` (`-6.0 dB`) by default in `audio_mixer_config.json` -> **REJECTED**
12. `adv_12_invalid_event_generating_victory_sound` — Bound `SFX11` Victory sting to pre-`2600` lane clear instead of tick-`2800` `wave_result` -> **REJECTED**

---

## 3. Multimodal Listening Review & Iterative Sound-Design Refinements

Every generated `.wav`, `.ogg`, seam-window clip, `10k`-viewer stress mix, mono fold-down, and `H.264 + AAC` montage video was inspected via direct multimodal audio/video perception during two iterative tuning passes:

- **Pass 1 Findings & Fixes Applied:**
  1. `synth_french_horn` initially exhibited a static upper-harmonic organ buzz on sustained tails (`SFX08`, `SFX11`, `SFX12`); refined with conical bell formants (`460 Hz`), brass attack bloom decaying into mellow sustain (`1/h^1.95`), subtle lip-scoop, and natural acoustic decay envelopes.
  2. `amb_village_dawn_loop` initially had continuous `1.2–4.5 kHz` broadband leaf hiss resembling surf and low `1.1 kHz` bird glides; replaced with warm `88–365 Hz` meadow breeze, sparse discrete leaf gusts, crystalline `2.6–3.9 kHz` passerine syrinx trills, wooden courtyard wind-chimes, and distant carpenter taps.
  3. `music_village_assault_loop` initially placed plucked lute an octave too high (`+12` semitones); moved lute into warm mid-low register (`MIDI 50–65`), deepened war-bodhrán (`82 Hz`), and added noble sustained cello/horn glue.
  4. `high_activity_10k_antispam_mix_proof.wav` initially allowed `SFX01` (`sfx_role_join`) every `450 ms` during heavy combat; aligned cooldown to `2000 ms` (`PREP`) / `3500 ms` at `-5.5 dB` (`ASSAULT`, suppressed during `Alerts`), eliminating repetitive chime fatigue completely (`98.8%` spam suppression, `0` clipped samples).
- **Honesty Distinction on Listening Hardware:** Multimodal AI audio perception verified actual decoded waveforms, pitch/timbre musicality, loop seams, mono fold-down, and AAC stream proxies. Physical human ear testing on real smartphone speakers, consumer headphones, and live OBS hardware remains required during human QA (`GATE-OBS` / `M07`).

---

## 4. Real FFmpeg AAC Stream Encode Proxy Metrics ([stream_encode_metrics.json](stream_encode_metrics.json))

- **24.0s Gameplay Montage (PCM Master w/ Music & Ducking):** `-16.43 LUFS` Integrated, `-1.91 dBTP` True Peak, `0` clipped samples, stereo correlation `0.984`, foreground SFX-to-ducked-bed margin `+12.23 dB`.
- **1080×1920 @ `128 kbps` AAC-LC (`gameplay_audiovisual_montage_1080p_128k_aac.mp4`):** `-16.46 LUFS` Integrated, `-1.75 dBTP` True Peak, `0` clipped samples, reconstruction SNR `30.8 dB`.
- **360×640 @ `96 kbps` AAC-LC (`gameplay_audiovisual_montage_360p_96k_aac.mp4`):** `-16.48 LUFS` Integrated, `-0.88 dBTP` True Peak, `0` clipped samples, clean tactical cue intelligibility.
- **Mono Downmix (`mono_downmix_battle_montage.wav`):** `-0.04 dB` RMS fold-down delta (`>= 0.98` phase coherence — zero comb filtering or vanishing cues).
