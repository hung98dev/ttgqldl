# ART-007 — Audio Style Guide (Bastion Dawn v1)

**Package:** `ART-007` (`SFX001`)  
**Style Identity:** Bastion Dawn v1 — Stylized Fantasy Village Defense  
**Target Engine:** Unity `6000.6.7f1` (Built-in 2D)  
**Master Specification:** `48,000 Hz` / `24-bit` PCM WAV (`pcm_s24le`), Runtime Ogg Vorbis (`.ogg`), Editable Standard MIDI (`.mid`) & Parametric DSP JSON

---

## 1. Sonic Direction & Material Language

The Bastion Dawn v1 soundscape translates the warm sunlit limestone, seasoned oak timber, forged cobalt steel, and dark obsidian-violet Boss runes from `ART-001..006` into a cohesive, mobile-legible acoustic palette:

- **Preparation Warmth (`WAITING`, `PREP` 160s):** Acoustic plucked lute, Celtic harp fifths, warm breathy wooden recorder, soft bowed cello/viola cushions, and peaceful dawn meadow breeze with sweet crystalline passerine songbirds (`2.6–3.9 kHz`).
- **Tactile Defense Materials:**
  - **Builder (`SFX02`):** Crisp iron T-hammer transient striking seasoned oak beams (`276 / 642 / 1285 Hz`) with a subtle limestone chip settle.
  - **Guard (`SFX03`):** Grounded `118 Hz` oak shield-boss impact paired with damped cobalt-steel rim modes (`418–2980 Hz`) — unmistakably defensive rather than offensive.
  - **Fighter (`SFX04`):** Tight `26 ms` blade air-slice snapping into a tempered broadsword cleave (`1790 / 2960 / 4180 Hz`) without bloated cinematic trailer swooshes.
  - **Fortress Masonry (`SFX06`, `SFX07`):** Dry limestone mortar cracks and tumbling pebbles for ordinary damage (`SFX06`), escalating to a decisive keystone shear and splintering oak gate-beam for `wall_hp -> 0` breach (`SFX07`).
- **Restrained Arcane Boss Identity (`SFX08`, `SFX09`):** Deep `78 Hz` war-drum heartbeat and conical French horn Phrygian call (`A3 -> Bb3 -> A3`) with plucked obsidian-rune overtones (`SFX08`), followed by the heaviest individual combat cue (`-1.50 dBTP`) on `Boss.cleaveslam[f05]` (`SFX09`) with `156–366 Hz` upper-bass harmonics that translate clearly on small smartphone speakers.
- **Fair, Respectful Settlement (`SFX10`, `SFX11`, `SFX12`):** Neutral two-note council wood-latch fifth (`G4 -> D5`) on `vote_closed` (`SFX10`, never implying A/B/C early); uplifting D-major horn/harp/strings heroic fanfare on `victory` (`SFX11`); dignified D-Dorian descending cello/lute/horn resolve (`F -> C/E -> Dsus2`) on `loss` (`SFX12`) that respects viewer effort without humiliating failure buzzer effects.

---

## 2. Explicit Acoustic Prohibitions

1. **Zero Muddy Sub-Bass Booms:** All one-shot SFX are high-passed between `56 Hz` and `120 Hz` so energy is concentrated in `90 Hz – 4.5 kHz` where mobile phone speakers reproduce punch cleanly.
2. **Zero Harsh Sirens or Countdown Beeps:** Neither `PREP` countdown warnings nor `SFX08` (`sfx_boss_warning.wav`) use electronic alarms, klaxons, or anxiety-inducing countdown ticks.
3. **Zero Long Reverb Tails:** All spatial cues use short, damped 5-tap wooden chamber early reflections (`11–67 ms`) so 3-lane combat never turns into reverberant mush.
4. **Zero Excessive Stereo Widening:** Panning is clamped to `[-0.35, +0.35]` constant-power so stereo phase correlation stays `>= +0.96` across all stereo masters and `0.5*(L+R)` mono fold-down loses `< 0.1 dB`.
5. **Zero Unlicensed Third-Party Recordings:** 100% of waveforms are synthesized deterministically from mathematical physical-modeling equations in [dsp_engine.py](tools/dsp_engine.py) and [build_audio.py](tools/build_audio.py).

---

## 3. Musical Architecture & Tonal Continuity

| Loop ID | File | Key / Mode | Tempo & Meter | Length | Tonal Role & Default Production State |
|---|---|---|---|---|---|
| `AMB01` | `ambient/wav/amb_village_dawn_loop.wav` | D Sunlit Open Fifth (`D3-A3-E4-F#4`) | Unmetered Seamless | `36.00s` (`1,728,000` samples) | Continuous peaceful village bed (`-26.0 LUFS`); sample-accurate N-point periodic RFFT loop |
| `MUS01` | `music/wav/music_village_prep_loop.wav` | D Mixolydian | `100.0 BPM`, `4/4`, `16 Bars` | `38.40s` (`1,843,200` samples) | Low-density pastoral lute, harp, wooden flute & legato strings (`-19.5 LUFS`, `-2.2 dB` `1.85 kHz` pocket) — **MUTED BY DEFAULT (`-80 dB`)** |
| `MUS02` | `music/wav/music_village_assault_loop.wav` | D Dorian | `128.0 BPM`, `4/4`, `20 Bars` | `37.50s` (`1,800,000` samples) | Grounded war-bodhrán, spiccato strings, mid-low lute & heroic French horn motif (`-17.5 LUFS`, `-3.0 dB` `1.85 kHz` SFX pocket) — **MUTED BY DEFAULT (`-80 dB`)** |

Both music compositions share the tonal center **D** (`D Mixolydian` in Preparation transforming into `D Dorian` in Assault and resolving to `D Major` in `SFX11` or `Dsus2` in `SFX12`), ensuring every alert sting and transition sounds musically unified whenever music is enabled for offline QA or future rights-cleared streams.
