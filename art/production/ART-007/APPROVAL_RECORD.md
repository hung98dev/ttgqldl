# ART-007 — Provenance, Originality & Rights Approval Record

**Package:** `ART-007` (`SFX001`)  
**Rights Status:** `PENDING_REVIEW`  
**Open Release Gates:** `GATE-LEGAL` (`PENDING_REVIEW`) | `GATE-OBS` (`OPEN`)  
**Mandatory Production Music Default:** **MUTED BY DEFAULT (`-80.0 dB`, `mute = true`)**

---

## 1. Provenance & First-Party Original Synthesis Declaration

Every single waveform in `ART-007` (`12` canonical SFX + `1` seamless `36.0s` village dawn ambient loop + `2` original seamless music compositions) was synthesized **100% from mathematical first principles** inside this repository:

- **Synthesis Code & Editable Projects:**
  - Physical-modeling DSP engine & Standard MIDI writer: [dsp_engine.py](tools/dsp_engine.py)
  - Deterministic asset builder & score definitions: [build_audio.py](tools/build_audio.py)
  - Editable per-SFX synthesis patches: `source/sfx/*_patch.json` (`12` files)
  - Editable ambient scene project: [amb_village_dawn_loop_project.json](source/ambient/amb_village_dawn_loop_project.json)
  - Editable Format-1 Standard MIDI Files & JSON scores: `source/music/music_village_prep_loop.mid`, [music_village_prep_loop_score.json](source/music/music_village_prep_loop_score.json), `source/music/music_village_assault_loop.mid`, [music_village_assault_loop_score.json](source/music/music_village_assault_loop_score.json)
- **Toolchain & Versions Recorded:**
  - `Python 3.15.0` (`BSD-style PSF License`)
  - `NumPy 2.5.4` (`BSD-3-Clause`) & `SciPy 1.18.1` (`BSD-3-Clause`)
  - `FFmpeg 9.0.2-full_build` (`libvorbis`, `aac`, `libx264` — used solely as format encoder/verifier on our first-party PCM outputs)
- **Zero Third-Party Audio Content:**
  - External audio samples used: **`NONE` (`0` files)**
  - SoundFonts (`.sf2` / `.sf3` / `.dls`) or commercial sample libraries used: **`NONE`**
  - Generative cloud music/SFX services used: **`NONE`**
  - Copyrighted melodies, quotations, or game soundtrack adaptations: **`NONE`**

Complete per-asset SHA-256 digests and provenance records are stored in [asset_manifest.json](asset_manifest.json).

---

## 2. Legal & Operational Gate Boundaries (`GATE-LEGAL` & `GATE-OBS`)

1. **Procedural Originality Does NOT Auto-Close `GATE-LEGAL`:**
   - Although 100% first-party procedural synthesis eliminates third-party sample-library licensing dependencies, `rights_status` is strictly maintained as **`PENDING_REVIEW`** until formal human legal/compliance sign-off (`GATE-LEGAL`).
2. **Music Remains Muted By Default:**
   - `music_village_prep_loop` (`MUS01`) and `music_village_assault_loop` (`MUS02`) are configured with `muted_by_default = true` (`-80.0 dB` in [audio_mixer_config.json](audio_mixer_config.json), `mute = true` in `BroadcastAudioManager.prefab`) so music is never broadcast accidentally before stream rights are formally cleared.
3. **Offline Broadcast Proxy Does NOT Close `GATE-OBS`:**
   - Offline FFmpeg `H.264 + AAC-LC` (`128 kbps` & `96 kbps`) stream proxy encoding (`qa/stream_encode_metrics.json`) verifies codec headroom and intelligibility, while real OBS hardware capture and 30-minute physical stream soak testing remain gated under `GATE-OBS` / `M07`.
