# ART-007 — Unity 6000.6.7f1 Audio Handoff & Prefab Contract

**Package:** `ART-007` (`SFX001`)  
**Target Engine:** Unity `6000.6.7f1`, Built-in 2D  
**Production Boundary:** `unity/` remains untouched (`M06 = NOT_STARTED`); verified in an isolated `Unity 6000.6.7f1 -batchmode -nographics` project (`qa/unity_isolated_verification.json`).

---

## 1. Differentiated `AudioImporter` Settings (Three Asset Profiles)

Never apply a single blanket compression mode to all 15 audio files. Apply the three provided Unity 6 presets in `unity_presets/`:

| Profile & Preset File | Target Assets | `loadType` | `compressionFormat` | `sampleRateSetting` | `preloadAudioData` | Rationale |
|---|---|---|---|---|---|---|
| [TTG_ART007_SFX_Short_DecompressOnLoad.preset.yaml](unity_presets/TTG_ART007_SFX_Short_DecompressOnLoad.preset.yaml) | `SFX01..SFX12` (`0.32s – 2.45s`) | `DecompressOnLoad` (`0`) | `PCM` (`0`) | `PreserveSampleRate` (`48 kHz`) | `true` (`loadInBackground = false`) | Zero CPU Vorbis decode spike and zero onset latency on frame-exact combat/hammer hits (`~2.09 MB` total PCM RAM across all 12 clips) |
| [TTG_ART007_Ambient_CompressedInMemory.preset.yaml](unity_presets/TTG_ART007_Ambient_CompressedInMemory.preset.yaml) | `amb_village_dawn_loop.wav` (`36.00s` Stereo) | `CompressedInMemory` (`1`) | `Vorbis` (`quality = 0.70`) | `PreserveSampleRate` (`48 kHz`) | `true` (`loadInBackground = true`) | Saves `~93%` RAM vs raw 36s stereo PCM (`~0.61 MB` in memory) with zero disk I/O jitter across seamless loop wrap |
| [TTG_ART007_Music_Streaming.preset.yaml](unity_presets/TTG_ART007_Music_Streaming.preset.yaml) | `music_village_prep_loop.wav`, `music_village_assault_loop.wav` (`38.4s / 37.5s`) | `Streaming` (`2`) | `Vorbis` (`quality = 0.70`) | `PreserveSampleRate` (`48 kHz`) | `false` (`loadInBackground = true`) | Near-zero resident RAM (`~128 KB` ring buffer total), zero startup cost while Music remains **muted by default** |

Keep `m_ForceToMono = 0` (`false`): `SFX01..SFX06` and `SFX10` are already mastered as true 1-channel Mono WAVs, while `SFX07..SFX09`, `SFX11..SFX12`, `AMB01`, `MUS01`, and `MUS02` preserve restrained phase-coherent Stereo (`correlation >= +0.967`).

---

## 2. `AudioMixer` Topology & Default Music Mute Contract

Configure `BroadcastAudioMixer.mixer` according to [audio_mixer_config.json](audio_mixer_config.json):

```text
Master (Exposed: MasterVolumeDb = 0.0 dB, Soft Limiter ceiling -1.0 dBTP)
 ├── Alerts  (Exposed: AlertsVolumeDb  =   0.0 dB)  <- SFX07, SFX08, SFX09, SFX10, SFX11, SFX12
 ├── SFX     (Exposed: SfxVolumeDb     =  -1.5 dB)  <- SFX01, SFX02, SFX03, SFX04, SFX05, SFX06
 ├── Ambient (Exposed: AmbientVolumeDb =  -3.0 dB)  <- AMB01
 └── Music   (Exposed: MusicVolumeDb   = -80.0 dB)  <- MUS01, MUS02 [MANDATORY DEFAULT MUTE]
```

- **Mandatory Default Music State:** Both the `Music` mixer group (`MusicVolumeDb = -80.0 dB`) AND `MusicLoopVoice_1_MutedByDefault` (`AudioSource.mute = true`, `volume = 0.0f`, `playOnAwake = false`) MUST remain muted by default until formal `GATE-LEGAL` approval is recorded.

---

## 3. `BroadcastAudioManager` Prefab Ownership & Lifecycle Contract

During M06 integration, copy the audio files into `unity/Assets/_TTGQLDL/Art/Audio/{SFX,Ambient,Music}/` and wire `BroadcastAudioManager.prefab` (verified in [ART007AudioPipelineVerifier.cs.txt](unity_presets/ART007AudioPipelineVerifier.cs.txt)):

1. **Hierarchy:** Root `BroadcastAudioManager` -> `CombatSfxPool_8` (`8× AudioSource`), `HighPriorityAlertsPool_2` (`2× AudioSource`), `AmbientLoopVoice_1` (`1× AudioSource`), `MusicLoopVoice_1_MutedByDefault` (`1× AudioSource`). Total = **12 preallocated `AudioSource` components**.
2. **Zero Network / Zero Authoritative RNG Coupling:** Audio components subscribe strictly to client-side presentation events decoded from committed server snapshots/events ([audio_cue_mapping.json](audio_cue_mapping.json)). Cosmetic micro-pitch selection hashes `{session_id}:{wave_no}:{visual_event_seq}` locally and never touches reducer state or simulation RNG.
3. **Pause / Resume / Shutdown Safety:**
   - Operator `paused`: attenuate `Ambient` by `-6.0 dB`, pause active loop advancement cleanly without destroying pooled sources.
   - Session `stopped` / scene teardown: stop all 12 voices, clear the 256-slot deduplication ring buffer, and release references with **0 leaked `AudioSource` objects** (verified in [unity_isolated_verification.json](qa/unity_isolated_verification.json)).

---

## 4. Actual Isolated Unity `6000.6.7f1` Verification Results

Executed via `Unity.exe -batchmode -nographics` using [ART007AudioPipelineVerifier.cs.txt](unity_presets/ART007AudioPipelineVerifier.cs.txt) -> [unity_isolated_verification.json](qa/unity_isolated_verification.json):

- **Unity Version:** `6000.6.7f1` (`status = PASS`)
- **Imported Clips:** `15 / 15` (`12` SFX `DecompressOnLoad/PCM`, `1` Ambient `CompressedInMemory/Vorbis`, `2` Music `Streaming/Vorbis`), all verified at `48,000 Hz` and decoded non-silent.
- **Total Runtime Clip Memory:** `2,831,063 bytes` (`~2.70 MB` total across all 15 assets!).
- **Pooled Sources & Shutdown Leak Check:** `12` preallocated `AudioSource` components serialized cleanly to `BroadcastAudioManager.prefab`; `0` leaked `AudioSource` objects after `DestroyImmediate`.
- **Anti-Spam & Replay Dedupe Proof:** `1,000` rapid join events in `1.0s` bounded; `5×` redelivered `sess-01:wave-10:wave_result` outbox events played `sfx_wave_victory` exactly `1` time; `music_muted_by_default_verified = true`.
- **Honesty Scope:** Executed in `-batchmode -nographics`; real acoustic listening was performed separately on rendered WAV/OGG/MP4 outputs, while live Unity Editor hardware speaker monitoring remains part of M06 human QA.
