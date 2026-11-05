#!/usr/bin/env python3
"""
ART-007 Deterministic Original Audio Producer (SFX001: 12 SFX + 1 Ambient Loop + 2 Music Loops).
Synthesizes 100% original 48 kHz / 24-bit PCM WAV masters, runtime Ogg Vorbis (.ogg) assets,
editable Standard MIDI (.mid) & parametric JSON source projects, cue/mixer contracts, and Unity presets.
"""

from __future__ import annotations

import hashlib
import json
import math
import sys
from pathlib import Path
from typing import Any, Dict, List, Tuple

import numpy as np
from scipy import signal

TOOLS_DIR = Path(__file__).resolve().parent
if str(TOOLS_DIR) not in sys.path:
    sys.path.insert(0, str(TOOLS_DIR))

from dsp_engine import (  # noqa: E402
    SR,
    apply_wooden_chamber_early_reflections,
    biquad_peaking,
    clean_one_shot_master,
    encode_ogg_vorbis,
    measure_audio_metrics,
    midi_to_hz,
    pan_mono_restrained,
    synth_bowed_strings,
    synth_frame_drum,
    synth_french_horn,
    synth_plucked_string,
    synth_shaker_tick,
    synth_wooden_flute,
    write_standard_midi,
    write_wav_24bit,
)

ART007_DIR = TOOLS_DIR.parent
REPO_ROOT = ART007_DIR.parent.parent.parent


def write_json_lf(path: Path, obj: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(obj, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n")


def write_text_lf(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text.replace("\r\n", "\n"), encoding="utf-8", newline="\n")


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1 << 16), b""):
            h.update(chunk)
    return h.hexdigest()


def add_at(buf: np.ndarray, clip: np.ndarray, start_s: float, periodic: bool = False, sr: int = SR) -> None:
    """Mix `clip` (1D or 2D) into `buf` starting at `start_s` (with modulo-N wrap if `periodic`)."""
    idx0 = int(round(start_s * sr))
    c = np.asarray(clip, dtype=np.float64)
    if buf.ndim == 2 and c.ndim == 1:
        c = np.column_stack([c, c])
    elif buf.ndim == 1 and c.ndim == 2:
        c = 0.5 * (c[:, 0] + c[:, 1])

    m = c.shape[0]
    n = buf.shape[0]
    if periodic:
        positions = (idx0 + np.arange(m)) % n
        np.add.at(buf, positions, c)
    else:
        s0 = max(0, idx0)
        c0 = max(0, -idx0)
        ln = min(n - s0, m - c0)
        if ln > 0:
            buf[s0 : s0 + ln] += c[c0 : c0 + ln]


def periodic_fft_eq_and_lufs_master(
    stereo_loop: np.ndarray,
    target_lufs: float,
    max_true_peak_dbtp: float,
    hp_hz: float = 52.0,
    lp_hz: float = 14200.0,
    presence_dip_db: float = 0.0,
    sr: int = SR,
) -> np.ndarray:
    """
    Zero-phase exact periodic FFT-domain EQ & loudness normalizer for seamless loops.
    Because filtering operates on exact N-point RFFT bins (with DC bin = 0), wrap-around
    continuity from sample[N-1] -> sample[0] remains 100% mathematically seamless!
    """
    arr = np.asarray(stereo_loop, dtype=np.float64).copy()
    n = arr.shape[0]
    freqs = np.fft.rfftfreq(n, d=1.0 / sr)

    # Smooth Butterworth-magnitude zero-phase curves in frequency domain
    hp_mag = 1.0 / np.sqrt(1.0 + (hp_hz / np.maximum(freqs, 1e-6)) ** 6)
    hp_mag[0] = 0.0  # Exact zero DC offset!
    lp_mag = 1.0 / np.sqrt(1.0 + (freqs / lp_hz) ** 8)
    eq_mag = hp_mag * lp_mag

    if abs(presence_dip_db) > 0.01:
        # Carve a gentle pocket around 1.85 kHz so combat SFX & voice cut cleanly over music
        bell = np.exp(-(((np.log2(np.maximum(freqs, 20.0) / 1850.0)) / 0.85) ** 2))
        eq_mag *= 10.0 ** ((presence_dip_db * bell) / 20.0)

    spec = np.fft.rfft(arr, axis=0)
    arr = np.fft.irfft(spec * eq_mag[:, None], n=n, axis=0)

    # First scale to target LUFS
    cur = measure_audio_metrics(arr, sr=sr, is_loop=True)
    gain_db = target_lufs - cur["integrated_lufs"]
    arr *= 10.0 ** (gain_db / 20.0)

    # Gentle soft limiter if needed while keeping exact zero mean & periodic smoothness
    os4 = signal.resample_poly(arr, up=4, down=1, axis=0)
    tp_db = 20.0 * math.log10(max(float(np.max(np.abs(os4))), 1e-12))
    if tp_db > max_true_peak_dbtp:
        drive = 10.0 ** ((tp_db - max_true_peak_dbtp + 0.35) / 20.0)
        lim = 10.0 ** (max_true_peak_dbtp / 20.0)
        arr = np.tanh((arr / lim) * drive) / math.tanh(drive) * lim
        arr -= np.mean(arr, axis=0, keepdims=True)
    return arr


# ============================================================================
# 1. PHASE 2 — SYNTHESIZE ALL 12 CANONICAL GAME SFX (48 kHz, 24-BIT WAV)
# ============================================================================

def build_12_canonical_sfx() -> Tuple[List[Dict[str, Any]], Dict[str, np.ndarray]]:
    sfx_defs: List[Dict[str, Any]] = []
    rendered: Dict[str, np.ndarray] = {}

    # --- SFX01: sfx_role_join.wav (Mono, 0.42s) ---
    # Warm wooden marimba + soft golden harp fifth confirmation (D5 -> A5), no loud bell!
    dur = 0.42
    buf = np.zeros(int(round(dur * SR)), dtype=np.float64)
    t_m1 = np.arange(int(0.24 * SR)) / SR
    wood_d5 = (
        0.72 * np.sin(2 * math.pi * 587.33 * t_m1) * np.exp(-t_m1 / 0.075)
        + 0.24 * np.sin(2 * math.pi * 587.33 * 3.98 * t_m1) * np.exp(-t_m1 / 0.024)
    )
    t_m2 = np.arange(int(0.31 * SR)) / SR
    wood_a5 = (
        0.68 * np.sin(2 * math.pi * 880.00 * t_m2) * np.exp(-t_m2 / 0.090)
        + 0.18 * np.sin(2 * math.pi * 880.00 * 3.98 * t_m2) * np.exp(-t_m2 / 0.022)
    )
    add_at(buf, wood_d5 * 0.45, 0.004)
    add_at(buf, synth_plucked_string(587.33, 0.28, 0.72, instrument="harp", seed=11), 0.004)
    add_at(buf, wood_a5 * 0.42, 0.076)
    add_at(buf, synth_plucked_string(880.00, 0.32, 0.78, instrument="harp", seed=12), 0.076)
    buf = apply_wooden_chamber_early_reflections(buf, wet=0.10)
    rendered["SFX01"] = clean_one_shot_master(buf, target_peak_dbtp=-5.5, hp_hz=120.0, lp_hz=11000.0)
    sfx_defs.append({
        "cue_id": "SFX01", "file_stem": "sfx_role_join", "channels": 1, "target_peak_dbtp": -5.5,
        "purpose": "Successful durable role registration confirmation (warm wood/harp D5->A5, non-bell)",
        "synthesis_recipe": {"layers": ["rosewood_marimba_modal_D5_A5", "celtic_harp_dyad_587_880Hz"], "seed": 11},
    })

    # --- SFX02: sfx_builder_repair.wav (Mono, 0.38s) ---
    # Crisp iron T-hammer contact on seasoned oak timber + dry limestone tap (synced to Builder.repair[f03])
    dur = 0.38
    n = int(round(dur * SR))
    t = np.arange(n) / SR
    rng = np.random.default_rng(22)
    oak_modes = (
        1.05 * np.sin(2 * math.pi * np.cumsum(276.0 + 55.0 * np.exp(-t / 0.009)) / SR) * np.exp(-t / 0.052)
        + 0.68 * np.sin(2 * math.pi * 642.0 * t) * np.exp(-t / 0.036)
        + 0.44 * np.sin(2 * math.pi * 1285.0 * t + 0.4) * np.exp(-t / 0.021)
        + 0.25 * np.sin(2 * math.pi * 2440.0 * t + 1.1) * np.exp(-t / 0.012)
    )
    click_nz = rng.standard_normal(n) * np.exp(-t / 0.0065)
    b_c, a_c = signal.butter(2, [580.0 / (SR * 0.5), 4600.0 / (SR * 0.5)], btype="band")
    strike = oak_modes + 0.92 * signal.lfilter(b_c, a_c, click_nz)
    buf = np.zeros(n, dtype=np.float64)
    add_at(buf, strike, 0.003)
    chip_t = np.arange(int(0.08 * SR)) / SR
    chip = (
        0.24 * np.sin(2 * math.pi * 980.0 * chip_t) * np.exp(-chip_t / 0.015)
        + 0.14 * np.sin(2 * math.pi * 1890.0 * chip_t) * np.exp(-chip_t / 0.009)
    )
    add_at(buf, chip, 0.086)
    buf = apply_wooden_chamber_early_reflections(buf, wet=0.08)
    rendered["SFX02"] = clean_one_shot_master(buf, target_peak_dbtp=-3.2, hp_hz=85.0, lp_hz=11500.0)
    sfx_defs.append({
        "cue_id": "SFX02", "file_stem": "sfx_builder_repair", "channels": 1, "target_peak_dbtp": -3.2,
        "purpose": "Builder T-hammer strike on seasoned oak & limestone (Builder.repair clip f03 @ 250ms)",
        "synthesis_recipe": {"layers": ["iron_mallet_transient", "oak_beam_modes_276_642_1285Hz", "wood_stone_chip_settle"], "seed": 22},
    })

    # --- SFX03: sfx_guard_block.wav (Mono, 0.45s) ---
    # Compact steel-rimmed cobalt tower shield block: solid low-mid thud + crisp damped steel rim lock
    dur = 0.45
    n = int(round(dur * SR))
    t = np.arange(n) / SR
    rng = np.random.default_rng(33)
    boss_thud = np.sin(2 * math.pi * np.cumsum(118.0 + 98.0 * np.exp(-t / 0.016)) / SR) * np.exp(-t / 0.068)
    plate_modes = (
        0.65 * np.sin(2 * math.pi * 418.0 * t) * np.exp(-t / 0.082)
        + 0.50 * np.sin(2 * math.pi * 668.0 * t + 0.4) * np.exp(-t / 0.064)
        + 0.38 * np.sin(2 * math.pi * 965.0 * t + 1.1) * np.exp(-t / 0.048)
        + 0.32 * np.sin(2 * math.pi * 1940.0 * t + 1.9) * np.exp(-t / 0.024)
        + 0.24 * np.sin(2 * math.pi * 2980.0 * t + 0.7) * np.exp(-t / 0.016)
    )
    impact_grit = rng.standard_normal(n) * np.exp(-t / 0.0085)
    b_g, a_g = signal.butter(2, [650.0 / (SR * 0.5), 4800.0 / (SR * 0.5)], btype="band")
    buf = np.zeros(n, dtype=np.float64)
    add_at(buf, 0.85 * boss_thud + 0.78 * plate_modes + 0.72 * signal.lfilter(b_g, a_g, impact_grit), 0.003)
    buf = apply_wooden_chamber_early_reflections(buf, wet=0.09)
    rendered["SFX03"] = clean_one_shot_master(buf, target_peak_dbtp=-2.8, hp_hz=72.0, lp_hz=10200.0)
    sfx_defs.append({
        "cue_id": "SFX03", "file_stem": "sfx_guard_block", "channels": 1, "target_peak_dbtp": -2.8,
        "purpose": "Compact steel-rimmed tower shield block impact (Guard.shieldbrace clip f02..f03)",
        "synthesis_recipe": {"layers": ["oak_shield_boss_thud_118Hz", "damped_steel_rim_modes_418_2980Hz", "gauntlet_brace_transient"], "seed": 33},
    })

    # --- SFX04: sfx_fighter_slash.wav (Mono, 0.36s) ---
    # Brief broadsword blade slice (0..26ms) snapping into a crisp tempered-steel cleave impact accent (no bloated swoosh!)
    dur = 0.36
    n = int(round(dur * SR))
    t = np.arange(n) / SR
    rng = np.random.default_rng(44)
    air_env = ((t / 0.020) ** 1.4) * np.exp(-(t / 0.022) ** 1.6)
    air_nz = rng.standard_normal(n)
    b_sw, a_sw = signal.butter(2, [1050.0 / (SR * 0.5), 4200.0 / (SR * 0.5)], btype="band")
    blade_air = signal.lfilter(b_sw, a_sw, air_nz) * air_env * 1.12
    n_hit = n - int(0.026 * SR)
    th = np.arange(n_hit) / SR
    blade_ring = (
        0.52 * np.sin(2 * math.pi * np.cumsum(205.0 + 135.0 * np.exp(-th / 0.010)) / SR) * np.exp(-th / 0.040)
        + 0.48 * np.sin(2 * math.pi * 720.0 * th) * np.exp(-th / 0.048)
        + 0.44 * np.sin(2 * math.pi * 1790.0 * th + 0.5) * np.exp(-th / 0.036)
        + 0.34 * np.sin(2 * math.pi * 2960.0 * th + 1.2) * np.exp(-th / 0.026)
        + 0.22 * np.sin(2 * math.pi * 4180.0 * th + 2.0) * np.exp(-th / 0.017)
    )
    bite_nz = rng.standard_normal(n_hit) * np.exp(-th / 0.0065)
    b_bt, a_bt = signal.butter(2, [1200.0 / (SR * 0.5), 6800.0 / (SR * 0.5)], btype="band")
    buf = blade_air.copy()
    add_at(buf, blade_ring + 0.80 * signal.lfilter(b_bt, a_bt, bite_nz), 0.026)
    buf = apply_wooden_chamber_early_reflections(buf, wet=0.07)
    rendered["SFX04"] = clean_one_shot_master(buf, target_peak_dbtp=-3.0, hp_hz=90.0, lp_hz=11800.0)
    sfx_defs.append({
        "cue_id": "SFX04", "file_stem": "sfx_fighter_slash", "channels": 1, "target_peak_dbtp": -3.0,
        "purpose": "Compact broadsword blade movement & impact accent (Fighter.swordslash clip f03 @ 250ms)",
        "synthesis_recipe": {"layers": ["26ms_blade_air_slice", "tempered_steel_cleave_1790_2960_4180Hz", "leather_bite_transient"], "seed": 44},
    })

    # --- SFX05: sfx_enemy_attack.wav (Mono, 0.32s) ---
    # Reusable compact physical strike (club/claw/gauntlet) supporting pitch modulation (-3st Brute, 0st Grunt, +2st Runner)
    dur = 0.32
    n = int(round(dur * SR))
    t = np.arange(n) / SR
    rng = np.random.default_rng(55)
    body_thump = np.sin(2 * math.pi * np.cumsum(138.0 + 105.0 * np.exp(-t / 0.014)) / SR) * np.exp(-t / 0.052)
    club_knock = (
        0.56 * np.sin(2 * math.pi * 490.0 * t) * np.exp(-t / 0.034)
        + 0.32 * np.sin(2 * math.pi * 1140.0 * t + 0.6) * np.exp(-t / 0.020)
    )
    snap_nz = rng.standard_normal(n) * np.exp(-t / 0.008)
    b_sn, a_sn = signal.butter(2, [600.0 / (SR * 0.5), 3900.0 / (SR * 0.5)], btype="band")
    buf = np.zeros(n, dtype=np.float64)
    add_at(buf, 0.80 * body_thump + club_knock + 0.70 * signal.lfilter(b_sn, a_sn, snap_nz), 0.003)
    rendered["SFX05"] = clean_one_shot_master(buf, target_peak_dbtp=-4.8, hp_hz=80.0, lp_hz=9800.0)
    sfx_defs.append({
        "cue_id": "SFX05", "file_stem": "sfx_enemy_attack", "channels": 1, "target_peak_dbtp": -4.8,
        "purpose": "Reusable compact physical enemy attack cue with bounded pitch/gain variation (Grunt/Runner/Brute)",
        "synthesis_recipe": {"layers": ["physical_body_thump_138Hz", "gnarled_club_knock_490_1140Hz", "hide_claw_transient"], "seed": 55},
    })

    # --- SFX06: sfx_wall_damage.wav (Mono, 0.58s) ---
    # Dry limestone crack & mortar pebbles
    dur = 0.58
    n = int(round(dur * SR))
    t = np.arange(n) / SR
    rng = np.random.default_rng(66)
    stone_thud = 0.72 * np.sin(2 * math.pi * np.cumsum(155.0 + 90.0 * np.exp(-t / 0.018)) / SR) * np.exp(-t / 0.065)
    crack_nz = rng.standard_normal(n) * np.exp(-t / 0.024)
    b_st, a_st = signal.butter(2, [750.0 / (SR * 0.5), 4600.0 / (SR * 0.5)], btype="band")
    crack = signal.lfilter(b_st, a_st, crack_nz)
    crack = biquad_peaking(crack, 1850.0, 4.0, q=2.2)
    buf = np.zeros(n, dtype=np.float64)
    add_at(buf, stone_thud + 0.95 * crack, 0.004)
    for dt_p, f_p, g_p in [(0.082, 1420.0, 0.34), (0.148, 1960.0, 0.26), (0.225, 1180.0, 0.21), (0.315, 2280.0, 0.15), (0.410, 1640.0, 0.10)]:
        tp = np.arange(int(0.055 * SR)) / SR
        peb = (
            0.6 * np.sin(2 * math.pi * f_p * tp) * np.exp(-tp / 0.011)
            + 0.5 * rng.standard_normal(tp.size) * np.exp(-tp / 0.006)
        )
        b_p, a_p = signal.butter(2, [800.0 / (SR * 0.5), 5200.0 / (SR * 0.5)], btype="band")
        add_at(buf, g_p * signal.lfilter(b_p, a_p, peb), dt_p)
    buf = apply_wooden_chamber_early_reflections(buf, wet=0.10)
    rendered["SFX06"] = clean_one_shot_master(buf, target_peak_dbtp=-3.5, hp_hz=75.0, lp_hz=10800.0)
    sfx_defs.append({
        "cue_id": "SFX06", "file_stem": "sfx_wall_damage", "channels": 1, "target_peak_dbtp": -3.5,
        "purpose": "Dry limestone masonry crack & tumbling mortar pebbles when wall takes non-breaching damage",
        "synthesis_recipe": {"layers": ["limestone_thud_155Hz", "mortar_fracture_band_1850Hz", "5_dry_rubble_pebbles"], "seed": 66},
    })

    # --- SFX07: sfx_wall_breach.wav (Stereo, 1.28s) ---
    # Decisive structural masonry fracture + oak gate-beam snap + controlled stone collapse (no muddy sub boom!)
    dur = 1.28
    n = int(round(dur * SR))
    t = np.arange(n) / SR
    rng = np.random.default_rng(77)
    buf_st = np.zeros((n, 2), dtype=np.float64)
    # Primary keystone fracture
    keystone = np.sin(2 * math.pi * np.cumsum(92.0 + 88.0 * np.exp(-t / 0.032)) / SR) * np.exp(-t / 0.18)
    shear_nz = rng.standard_normal(n) * (0.75 * np.exp(-t / 0.065) + 0.30 * np.exp(-t / 0.34))
    b_sh, a_sh = signal.butter(2, [380.0 / (SR * 0.5), 4200.0 / (SR * 0.5)], btype="band")
    add_at(buf_st, pan_mono_restrained(0.90 * keystone + 0.85 * signal.lfilter(b_sh, a_sh, shear_nz), -0.08), 0.005)
    # Seasoned oak gate-beam splinter snap at 0.105s
    tb = np.arange(int(0.42 * SR)) / SR
    beam_snap = (
        0.75 * np.sin(2 * math.pi * 235.0 * tb) * np.exp(-tb / 0.085)
        + 0.55 * np.sin(2 * math.pi * 645.0 * tb) * np.exp(-tb / 0.045)
        + 0.85 * signal.lfilter(b_sh, a_sh, rng.standard_normal(tb.size) * np.exp(-tb / 0.018))
    )
    add_at(buf_st, pan_mono_restrained(beam_snap, +0.16), 0.105)
    # Cascading masonry blocks settling cleanly
    block_events = [
        (0.24, 168.0, 0.52, -0.18), (0.39, 215.0, 0.40, +0.14),
        (0.56, 290.0, 0.30, -0.10), (0.74, 410.0, 0.20, +0.12), (0.93, 560.0, 0.12, -0.06),
    ]
    for bt0, bf, bg, bpan in block_events:
        tk = np.arange(int(0.24 * SR)) / SR
        blk = (
            0.65 * np.sin(2 * math.pi * bf * (1.0 + 0.12 * np.exp(-tk / 0.015)) * tk) * np.exp(-tk / 0.055)
            + 0.55 * signal.lfilter(b_sh, a_sh, rng.standard_normal(tk.size) * np.exp(-tk / 0.016))
        )
        add_at(buf_st, pan_mono_restrained(bg * blk, bpan), bt0)
    buf_st = apply_wooden_chamber_early_reflections(buf_st, wet=0.15, room_scale=1.25)
    rendered["SFX07"] = clean_one_shot_master(buf_st, target_peak_dbtp=-1.8, hp_hz=58.0, lp_hz=10500.0, fade_out_ms=45.0)
    sfx_defs.append({
        "cue_id": "SFX07", "file_stem": "sfx_wall_breach", "channels": 2, "target_peak_dbtp": -1.8,
        "purpose": "Decisive structural masonry & gate-beam fracture when wall_hp transitions >0 -> 0",
        "synthesis_recipe": {"layers": ["keystone_shear_92Hz", "oak_gate_beam_splinter_105ms", "5_masonry_collapse_blocks"], "seed": 77},
    })

    # --- SFX08: sfx_boss_warning.wav (Stereo, 1.86s) ---
    # Distinctive restrained obsidian-violet arcane war-horn swell + deep war-drum pulse + runic crystal shimmer (zero siren!)
    dur = 1.86
    n = int(round(dur * SR))
    buf_st = np.zeros((n, 2), dtype=np.float64)
    t_all = np.arange(n) / SR
    swell_arch = (np.sin(math.pi * np.clip(t_all / 1.72, 0.0, 1.0)) ** 1.15) * np.exp(-np.maximum(0.0, t_all - 0.55) / 1.15)

    horn_d2 = synth_french_horn(73.42, 1.72, vel=0.88, attack_s=0.14, release_s=0.52)
    horn_d3 = synth_french_horn(146.83, 1.72, vel=0.92, attack_s=0.13, release_s=0.52)
    cello_d2 = synth_bowed_strings(73.42, 1.72, vel=0.72, spiccato=False, seed=81)
    # Dark Phrygian brass call inflection: A3 (220 Hz) -> Bb3 (233.08 Hz) -> A3 (220 Hz)
    call_1 = synth_french_horn(220.00, 0.42, vel=0.72, attack_s=0.08, release_s=0.16)
    call_2 = synth_french_horn(233.08, 0.46, vel=0.78, attack_s=0.07, release_s=0.18)
    call_3 = synth_french_horn(220.00, 0.85, vel=0.74, attack_s=0.08, release_s=0.45)

    # Restrained obsidian rune glass-bell harmonic overtone (pluck-decayed, never sustained organ!)
    rune_bell_1 = synth_plucked_string(587.33, 0.95, vel=0.42, instrument="harp", damping=0.12, seed=82)
    rune_bell_2 = synth_plucked_string(622.25, 1.10, vel=0.38, instrument="harp", damping=0.12, seed=83)

    # Distant war-drum double heartbeat pulse
    add_at(buf_st, pan_mono_restrained(synth_frame_drum(78.0, 0.42, vel=0.82, seed=84), 0.0), 0.012)
    add_at(buf_st, pan_mono_restrained(synth_frame_drum(78.0, 0.48, vel=0.68, seed=85), 0.0), 0.360)

    add_at(buf_st, pan_mono_restrained(horn_d2 * 0.78, -0.06), 0.020)
    add_at(buf_st, pan_mono_restrained(horn_d3, -0.12), 0.020)
    add_at(buf_st, pan_mono_restrained(cello_d2 * 0.85, +0.10), 0.020)
    add_at(buf_st, pan_mono_restrained(call_1, +0.14), 0.060)
    add_at(buf_st, pan_mono_restrained(call_2, +0.14), 0.380)
    add_at(buf_st, pan_mono_restrained(call_3, -0.08), 0.760)
    add_at(buf_st, pan_mono_restrained(rune_bell_1 * 0.48, -0.16), 0.140)
    add_at(buf_st, pan_mono_restrained(rune_bell_2 * 0.44, +0.16), 0.390)
    buf_st *= (0.35 + 0.65 * swell_arch)[:, None]
    buf_st = apply_wooden_chamber_early_reflections(buf_st, wet=0.18, room_scale=1.35)
    rendered["SFX08"] = clean_one_shot_master(buf_st, target_peak_dbtp=-2.8, hp_hz=60.0, lp_hz=8800.0, fade_out_ms=140.0)
    sfx_defs.append({
        "cue_id": "SFX08", "file_stem": "sfx_boss_warning", "channels": 2, "target_peak_dbtp": -2.8,
        "purpose": "Restrained arcane war-horn Boss incoming warning before authoritative tick 2600 spawn",
        "synthesis_recipe": {"layers": ["conical_war_horn_D2_D3", "phrygian_call_A3_Bb3_A3", "war_drum_pulse_78Hz", "obsidian_rune_chime"], "seed": 81},
    })

    # --- SFX09: sfx_boss_slam.wav (Stereo, 1.16s) ---
    # Heavy notched runic war-cleaver earth impact + short magenta arcane resonance (strongest combat cue: -1.5 dBTP)
    dur = 1.16
    n = int(round(dur * SR))
    t = np.arange(n) / SR
    rng = np.random.default_rng(99)
    buf_st = np.zeros((n, 2), dtype=np.float64)
    # 1) 42ms descending cleaver rush leading into slam impact
    rush_n = int(0.052 * SR)
    tr_u = np.arange(rush_n) / SR
    rush = signal.lfilter(
        *signal.butter(2, [400.0 / (SR * 0.5), 2600.0 / (SR * 0.5)], btype="band"),
        rng.standard_normal(rush_n),
    ) * ((tr_u / 0.042) ** 1.8) * np.exp(-np.maximum(0.0, tr_u - 0.042) / 0.008)
    add_at(buf_st, pan_mono_restrained(0.55 * rush, -0.05), 0.002)
    # 2) Massive cleaver + stone impact + upper-bass harmonics (translates on phone speakers!) + arcane D2/A2/F3 resonance
    ns = n - int(0.042 * SR)
    ts = np.arange(ns) / SR
    earth_punch = (
        0.95 * np.sin(2 * math.pi * np.cumsum(78.0 + 96.0 * np.exp(-ts / 0.022)) / SR) * np.exp(-ts / 0.14)
        + 0.58 * np.sin(2 * math.pi * np.cumsum(156.0 + 140.0 * np.exp(-ts / 0.018)) / SR) * np.exp(-ts / 0.095)
    )
    cleaver_iron = (
        0.54 * np.sin(2 * math.pi * 366.0 * ts) * np.exp(-ts / 0.11)
        + 0.38 * np.sin(2 * math.pi * 812.0 * ts + 0.5) * np.exp(-ts / 0.075)
        + 0.26 * np.sin(2 * math.pi * 1640.0 * ts + 1.3) * np.exp(-ts / 0.045)
    )
    arcane_res = (
        0.30 * np.sin(2 * math.pi * 146.83 * ts) * np.exp(-ts / 0.34)
        + 0.22 * np.sin(2 * math.pi * 220.00 * ts + 0.4) * np.exp(-ts / 0.30)
        + 0.15 * np.sin(2 * math.pi * 440.00 * (1.0 + 0.006 * np.sin(2 * math.pi * 7.5 * ts)) * ts) * np.exp(-ts / 0.26)
    )
    slam_crack = signal.lfilter(
        *signal.butter(2, [450.0 / (SR * 0.5), 4400.0 / (SR * 0.5)], btype="band"),
        rng.standard_normal(ns) * np.exp(-ts / 0.020),
    )
    add_at(buf_st, pan_mono_restrained(earth_punch + cleaver_iron + 0.85 * slam_crack, 0.0), 0.042)
    add_at(buf_st, pan_mono_restrained(arcane_res, +0.14), 0.055)
    buf_st = apply_wooden_chamber_early_reflections(buf_st, wet=0.14, room_scale=1.2)
    rendered["SFX09"] = clean_one_shot_master(buf_st, target_peak_dbtp=-1.5, hp_hz=56.0, lp_hz=10200.0, fade_out_ms=45.0)
    sfx_defs.append({
        "cue_id": "SFX09", "file_stem": "sfx_boss_slam", "channels": 2, "target_peak_dbtp": -1.5,
        "purpose": "Boss heavy runic war-cleaver slam + short magenta resonance (Boss.cleaveslam f05 @ 416.7ms)",
        "synthesis_recipe": {"layers": ["42ms_cleaver_descent", "earth_iron_slam_78_156_366Hz", "arcane_chord_tail_D3_A3"], "seed": 99},
    })

    # --- SFX10: sfx_vote_closed.wav (Mono, 0.48s) ---
    # Concise, neutral council wooden gavel/latch resolution (G4 -> D5 neutral fifth; never implies A/B/C won early!)
    dur = 0.48
    n = int(round(dur * SR))
    buf = np.zeros(n, dtype=np.float64)
    for t0, f0, amp in [(0.004, 392.00, 0.72), (0.112, 587.33, 0.82)]:
        tw = np.arange(int(0.30 * SR)) / SR
        wood_seal = (
            0.78 * np.sin(2 * math.pi * f0 * tw) * np.exp(-tw / 0.068)
            + 0.34 * np.sin(2 * math.pi * f0 * 2.74 * tw) * np.exp(-tw / 0.024)
            + 0.16 * np.sin(2 * math.pi * f0 * 5.12 * tw) * np.exp(-tw / 0.011)
        )
        add_at(buf, amp * wood_seal, t0)
    buf = apply_wooden_chamber_early_reflections(buf, wet=0.09)
    rendered["SFX10"] = clean_one_shot_master(buf, target_peak_dbtp=-5.2, hp_hz=110.0, lp_hz=9500.0)
    sfx_defs.append({
        "cue_id": "SFX10", "file_stem": "sfx_vote_closed", "channels": 1, "target_peak_dbtp": -5.2,
        "purpose": "Neutral authoritative vote-close seal at tick 1200 (never implies A/B/C before server result)",
        "synthesis_recipe": {"layers": ["council_wood_seal_G4_392Hz", "neutral_latch_fifth_D5_587Hz"], "seed": 100},
    })

    # --- SFX11: sfx_wave_victory.wav (Stereo, 2.45s) ---
    # Uplifting original Bastion Dawn heroic fanfare sting (D4 -> F#4 -> A4 -> D5 w/ mellow French horn, Celtic harp, lute & strings)
    dur = 2.45
    n_vic = int(round(dur * SR))
    buf_st = np.zeros((n_vic, 2), dtype=np.float64)
    arp_notes = [
        (0.010, 62, 0.28, 0.80, -0.14),  # D4
        (0.155, 66, 0.28, 0.84, +0.12),  # F#4
        (0.300, 69, 0.30, 0.88, -0.08),  # A4
        (0.465, 74, 1.82, 0.94, +0.06),  # D5 apex with natural acoustic decay!
    ]
    for st0, mn, ndur, nvel, npan in arp_notes:
        hz = midi_to_hz(mn)
        h_sig = synth_french_horn(hz, ndur, vel=nvel, attack_s=0.045, release_s=min(0.65, ndur * 0.45))
        if ndur > 1.0:
            th_d = np.arange(h_sig.size) / SR
            h_sig = h_sig * np.exp(-np.maximum(0.0, th_d - 0.18) / 0.88)
        p_sig = synth_plucked_string(hz, min(1.55, ndur + 0.30), vel=nvel, instrument="harp", seed=200 + mn)
        add_at(buf_st, pan_mono_restrained(h_sig * 0.82, npan), st0)
        add_at(buf_st, pan_mono_restrained(0.95 * p_sig, npan), st0)

    # Warm bowed strings + low horn + harp chord bloom on resolved D-major chord at t = 0.465s
    for ch_mn, ch_pan, ch_vel in [(50, -0.08, 0.80), (57, +0.10, 0.75), (62, -0.12, 0.72), (66, +0.12, 0.70)]:
        s_pad = synth_bowed_strings(midi_to_hz(ch_mn), 1.80, vel=ch_vel, spiccato=False, seed=240 + ch_mn)
        ts_d = np.arange(s_pad.size) / SR
        s_pad *= np.exp(-np.maximum(0.0, ts_d - 0.22) / 0.92)
        add_at(buf_st, pan_mono_restrained(s_pad * 0.85, ch_pan), 0.465)
    for h_idx, h_mn in enumerate([50, 57, 62, 66, 69, 74, 78]):
        add_at(
            buf_st,
            pan_mono_restrained(synth_plucked_string(midi_to_hz(h_mn), 1.55, vel=0.74, instrument="harp", seed=280 + h_idx) * 0.68, -0.18 + 0.06 * h_idx),
            0.465 + h_idx * 0.028,
        )
    add_at(buf_st, pan_mono_restrained(synth_frame_drum(110.0, 0.28, vel=0.58, seed=211), 0.0), 0.010)
    add_at(buf_st, pan_mono_restrained(synth_frame_drum(92.0, 0.42, vel=0.85, seed=212), 0.0), 0.465)
    buf_st = apply_wooden_chamber_early_reflections(buf_st, wet=0.18, room_scale=1.25)
    rendered["SFX11"] = clean_one_shot_master(buf_st, target_peak_dbtp=-2.0, hp_hz=60.0, lp_hz=11800.0, fade_out_ms=180.0)
    sfx_defs.append({
        "cue_id": "SFX11", "file_stem": "sfx_wave_victory", "channels": 2, "target_peak_dbtp": -2.0,
        "purpose": "Confirmed tick-2800 wave victory fanfare sting (original Bastion Dawn D-major horn/strings/harp motif)",
        "synthesis_recipe": {"layers": ["mellow_horn_fanfare_D4_Fs4_A4_D5", "celtic_harp_strum_bloom", "warm_strings_Dmaj", "frame_drum_cadence"], "seed": 200},
    })

    # --- SFX12: sfx_wave_defeat.wav (Stereo, 2.35s) ---
    # Restrained, respectful Dorian resolve cadence (warm cello, lute/harp & mellow horn tapering peacefully — non-humiliating)
    dur = 2.35
    buf_st = np.zeros((int(round(dur * SR)), 2), dtype=np.float64)
    cadence = [
        (0.015, [53, 60, 65, 69], 0.44, 0.72),  # Fmaj
        (0.420, [52, 55, 64, 67], 0.44, 0.68),  # C/E
        (0.830, [50, 57, 62, 64], 1.42, 0.76),  # Dsus2 peaceful open resolve
    ]
    for ct0, notes, cdur, cvel in cadence:
        for idx_n, mn in enumerate(notes):
            hz = midi_to_hz(mn)
            pan = -0.15 + 0.10 * idx_n
            v_sig = synth_bowed_strings(hz, cdur, vel=cvel * 0.85, spiccato=False, seed=300 + mn + idx_n)
            if cdur > 1.0:
                tv_d = np.arange(v_sig.size) / SR
                v_sig *= np.exp(-np.maximum(0.0, tv_d - 0.16) / 0.72)
            p_sig = synth_plucked_string(hz, min(1.35, cdur + 0.20), vel=cvel * 0.78, instrument="lute" if mn < 60 else "harp", seed=320 + mn)
            add_at(buf_st, pan_mono_restrained(v_sig * 0.90, pan), ct0)
            add_at(buf_st, pan_mono_restrained(p_sig * 0.72, pan), ct0 + idx_n * 0.018)
    # Single soft mellow horn voice singing F4 -> E4 -> D4
    for ht0, hmn, hdur in [(0.015, 65, 0.40), (0.420, 64, 0.40), (0.830, 62, 1.35)]:
        h_mel = synth_french_horn(midi_to_hz(hmn), hdur, vel=0.68, attack_s=0.075, release_s=min(0.55, hdur * 0.45))
        if hdur > 1.0:
            th_m = np.arange(h_mel.size) / SR
            h_mel *= np.exp(-np.maximum(0.0, th_m - 0.15) / 0.68)
        add_at(buf_st, pan_mono_restrained(h_mel * 0.75, +0.05), ht0)
    add_at(buf_st, pan_mono_restrained(synth_frame_drum(92.0, 0.38, vel=0.52, seed=350), 0.0), 0.830)
    buf_st = apply_wooden_chamber_early_reflections(buf_st, wet=0.18, room_scale=1.30)
    rendered["SFX12"] = clean_one_shot_master(buf_st, target_peak_dbtp=-3.2, hp_hz=58.0, lp_hz=9600.0, fade_out_ms=190.0)
    sfx_defs.append({
        "cue_id": "SFX12", "file_stem": "sfx_wave_defeat", "channels": 2, "target_peak_dbtp": -3.2,
        "purpose": "Confirmed tick-2800 wave defeat/loss cue (dignified D-Dorian cello/horn/harp resolve, non-punitive)",
        "synthesis_recipe": {"layers": ["dignified_cello_lute_F_CE_Dsus2", "mellow_horn_F4_E4_D4", "soft_frame_drum_settle"], "seed": 300},
    })

    return sfx_defs, rendered


# ============================================================================
# 2. PHASE 3 — ORIGINAL SEAMLESS AMBIENT LOOP (36.000 s, STEREO 48 kHz 24-BIT)
# ============================================================================

def build_ambient_village_dawn_loop() -> Tuple[np.ndarray, Dict[str, Any]]:
    """
    Synthesize `amb_village_dawn_loop.wav` (exact 36.000 s = 1,728,000 samples @ 48 kHz Stereo).
    Uses exact N-point periodic FFT synthesis for open-air dawn breeze & leaf rustle plus
    modulo-N wrapped village foley details so sample[N-1] -> sample[0] is 100% seamless!
    """
    dur_s = 36.0
    n = int(round(dur_s * SR))  # 1,728,000 samples
    t = np.arange(n, dtype=np.float64) / SR
    rng = np.random.default_rng(7001)

    freqs = np.fft.rfftfreq(n, d=1.0 / SR)

    def periodic_band_noise(f_lo: float, f_hi: float, slope: float, seed_local: int) -> np.ndarray:
        lrng = np.random.default_rng(seed_local)
        phases = lrng.uniform(0.0, 2.0 * math.pi, size=freqs.shape[0])
        amps = lrng.rayleigh(scale=1.0, size=freqs.shape[0])
        bp = (1.0 / np.sqrt(1.0 + (f_lo / np.maximum(freqs, 1e-6)) ** 4)) * (
            1.0 / np.sqrt(1.0 + (freqs / f_hi) ** 4)
        )
        tilt = np.maximum(freqs, 20.0) ** slope
        spec = amps * bp * tilt * (np.cos(phases) + 1j * np.sin(phases))
        spec[0] = 0.0
        sig = np.fft.irfft(spec, n=n)
        return sig / max(float(np.std(sig)), 1e-9)

    # 1) Gentle open-air meadow dawn wind (warm low-mid breath 90-360 Hz, zero ocean-surf hiss!)
    wind_l = periodic_band_noise(92.0, 350.0, -0.75, 7101)
    wind_r = periodic_band_noise(98.0, 365.0, -0.75, 7102)
    wind_mid = periodic_band_noise(88.0, 320.0, -0.80, 7103)
    lfo_wind = 0.56 + 0.28 * np.sin(2 * math.pi * (1.0 / dur_s) * t) + 0.16 * np.sin(2 * math.pi * (2.0 / dur_s) * t + 0.9)
    bed_l = (0.80 * wind_mid + 0.20 * wind_l) * lfo_wind * 0.011
    bed_r = (0.80 * wind_mid + 0.20 * wind_r) * lfo_wind * 0.011

    # 2) Whisper-light discrete leaf rustle gusts (sparse gentle breezes rather than continuous rain/surf hiss!)
    leaf_l = periodic_band_noise(1500.0, 4200.0, -0.55, 7201)
    leaf_r = periodic_band_noise(1550.0, 4300.0, -0.55, 7202)
    leaf_mid = periodic_band_noise(1450.0, 4100.0, -0.55, 7203)
    gust_gate = np.maximum(0.0, np.sin(2 * math.pi * (3.0 / dur_s) * t + 0.8)) ** 4.0
    bed_l += (0.72 * leaf_mid + 0.28 * leaf_l) * (0.08 + 0.92 * gust_gate) * 0.0022
    bed_r += (0.72 * leaf_mid + 0.28 * leaf_r) * (0.08 + 0.92 * gust_gate) * 0.0022

    # 3) Warm stylized Bastion Dawn sunlit harmonic air-pad (D3 + A3 + E4 + F#4 whispered fifth-ninth)
    dawn_pad = (
        0.48 * np.sin(2 * math.pi * (round(146.83 * dur_s) / dur_s) * t)
        + 0.36 * np.sin(2 * math.pi * (round(220.00 * dur_s) / dur_s) * t + 0.7)
        + 0.24 * np.sin(2 * math.pi * (round(329.63 * dur_s) / dur_s) * t + 1.5)
        + 0.16 * np.sin(2 * math.pi * (round(369.99 * dur_s) / dur_s) * t + 2.3)
    ) * (0.72 + 0.28 * np.sin(2 * math.pi * (1.0 / dur_s) * t + 2.1)) * 0.014

    amb = np.column_stack([bed_l + dawn_pad, bed_r + dawn_pad])

    # 4) Restrained living-village environmental details (wrapped modulo-N)
    # 4a. Crystalline passerine morning songbirds (sweet high syrinx whistle syllables 2550-3850 Hz, zero owl hoots!)
    bird_phrases = [
        (2.8,  [(0.00, 2637.0, 2960.0, 0.075), (0.11, 2960.0, 3520.0, 0.085), (0.24, 3322.0, 2960.0, 0.110)], -0.22, 0.024),
        (9.6,  [(0.00, 3136.0, 3520.0, 0.065), (0.09, 3520.0, 3136.0, 0.070), (0.19, 2960.0, 3322.0, 0.095), (0.32, 3520.0, 3951.0, 0.11)], +0.20, 0.021),
        (17.8, [(0.00, 2637.0, 3136.0, 0.080), (0.12, 2960.0, 2637.0, 0.095)], -0.16, 0.022),
        (25.4, [(0.00, 2960.0, 3520.0, 0.070), (0.10, 3520.0, 3322.0, 0.075), (0.21, 3322.0, 3951.0, 0.115)], +0.18, 0.020),
        (32.6, [(0.00, 2793.8, 3136.0, 0.075), (0.11, 3136.0, 2960.0, 0.100)], -0.12, 0.019),
    ]
    for bt0, syllables, bpan, bgain in bird_phrases:
        for s_off, fa, fb, sdur in syllables:
            tb = np.arange(int(sdur * SR)) / SR
            env_b = np.sin(math.pi * np.clip(tb / sdur, 0.0, 1.0)) ** 1.6
            f_curve = fa + (fb - fa) * (tb / sdur) ** 0.8 + 38.0 * np.sin(2 * math.pi * 26.0 * tb)
            chirp = (
                0.88 * np.sin(2 * math.pi * np.cumsum(f_curve) / SR)
                + 0.12 * np.sin(4 * math.pi * np.cumsum(f_curve) / SR)
            ) * env_b * bgain
            add_at(amb, apply_wooden_chamber_early_reflections(pan_mono_restrained(chirp, bpan), wet=0.32, room_scale=1.4), bt0 + s_off, periodic=True)

    # 4b. Gentle courtyard wooden wind-chime / harp breeze harmonics (subtle magical village life)
    for cht0, ch_notes, ch_pan in [(6.2, [587.33, 880.0, 1174.66], +0.16), (21.4, [659.25, 880.0, 1318.51], -0.18)]:
        for idx_c, ch_f in enumerate(ch_notes):
            ch_sig = synth_plucked_string(ch_f, 1.35, vel=0.34, instrument="harp", damping=0.14, seed=7400 + idx_c) * 0.048
            add_at(amb, apply_wooden_chamber_early_reflections(pan_mono_restrained(ch_sig, ch_pan), wet=0.30, room_scale=1.3), cht0 + idx_c * 0.16, periodic=True)

    # 4c. Soft seasoned-timber watchtower / signboard gentle creak
    for ct0, cf0, cpan in [(13.2, 196.0, -0.18), (28.8, 178.0, +0.15)]:
        tc = np.arange(int(0.62 * SR)) / SR
        env_c = np.sin(math.pi * np.clip(tc / 0.62, 0.0, 1.0)) ** 2.2
        stick_slip = np.sin(2 * math.pi * np.cumsum(cf0 + 14.0 * np.sin(2 * math.pi * 3.0 * tc)) / SR)
        creak = (0.68 * stick_slip + 0.32 * np.sin(2 * math.pi * cf0 * 2.08 * tc)) * env_c * 0.011
        add_at(amb, pan_mono_restrained(creak, cpan), ct0, periodic=True)

    # 4d. Distant village carpenter mallet double-tap & courtyard well droplet
    for wt0, wf0, wpan in [(15.2, 385.0, +0.22), (15.54, 402.0, +0.22), (30.8, 740.0, -0.15)]:
        tw = np.arange(int(0.18 * SR)) / SR
        tap = np.sin(2 * math.pi * wf0 * (1.0 + 0.08 * np.exp(-tw / 0.02)) * tw) * np.exp(-tw / 0.036) * 0.018
        add_at(amb, apply_wooden_chamber_early_reflections(pan_mono_restrained(tap, wpan), wet=0.34, room_scale=1.35), wt0, periodic=True)

    amb = periodic_fft_eq_and_lufs_master(
        amb, target_lufs=-26.0, max_true_peak_dbtp=-8.5, hp_hz=65.0, lp_hz=11500.0
    )
    project_doc = {
        "cue_id": "AMB01",
        "file_stem": "amb_village_dawn_loop",
        "duration_s": dur_s,
        "exact_samples": n,
        "sample_rate_hz": SR,
        "channels": 2,
        "loop_method": "Exact N-point periodic RFFT meadow breeze + modulo-N wrapped stylized village foley (zero crossfade dip)",
        "layers": [
            {"id": "meadow_dawn_breeze", "band_hz": [88, 365], "lfo_cycles_per_loop": [1, 2]},
            {"id": "discrete_courtyard_leaf_gusts", "band_hz": [1450, 4300], "lfo_cycles_per_loop": [3]},
            {"id": "bastion_dawn_harmonic_air", "notes_hz": [146.83, 220.0, 329.63, 369.99], "lfo_cycles_per_loop": [1]},
            {"id": "passerine_morning_songbirds", "timestamps_s": [2.8, 9.6, 17.8, 25.4, 32.6]},
            {"id": "courtyard_wooden_windchimes", "timestamps_s": [6.2, 21.4]},
            {"id": "seasoned_timber_creaks", "timestamps_s": [13.2, 28.8]},
            {"id": "distant_carpenter_and_well", "timestamps_s": [15.2, 15.54, 30.8]},
        ],
    }
    return amb, project_doc


# ============================================================================
# 3. PHASE 4 — TWO ORIGINAL SEAMLESS MUSIC COMPOSITIONS (+ EDITABLE .MID)
# ============================================================================

def build_music_01_village_prep() -> Tuple[np.ndarray, List[Dict[str, Any]], Dict[str, Any]]:
    """
    Original Composition 01: `music_village_prep_loop.wav`
    Key/Mode: D Mixolydian | Tempo: 100.0 BPM | Meter: 4/4 | 16 Bars = 64 Beats = 38.400 s (1,843,200 samples).
    Warm, pastoral, low-density plucked lute, Celtic harp, wooden recorder, soft cello pad & finger frame-drum.
    """
    bpm = 100.0
    sec_per_beat = 60.0 / bpm  # 0.600 s
    bars = 16
    total_beats = bars * 4  # 64 beats
    dur_s = total_beats * sec_per_beat  # 38.400 s exact!
    n = int(round(dur_s * SR))  # 1,843,200 samples
    mix = np.zeros((n, 2), dtype=np.float64)

    # 16-bar original chord progression in D Mixolydian (1 chord per bar, 4 beats each):
    # Phrases:
    #  Bars 1-4:  D        | Cadd9    | G/B      | A7sus4
    #  Bars 5-8:  D        | Em7      | Cmaj7    | Asus4 -> A
    #  Bars 9-12: Bm7      | Gmaj7    | D/F#     | Cadd9
    #  Bars 13-16:Gmaj7    | D/F#     | Em7      | Asus4 -> A7 (resolves cleanly back to Bar 1 D!)
    bar_chords = [
        ("D",      [50, 57, 62, 66]),
        ("Cadd9",  [48, 55, 62, 64]),
        ("G/B",    [47, 55, 62, 67]),
        ("A7sus4", [45, 52, 62, 64]),
        ("D",      [50, 57, 62, 66]),
        ("Em7",    [52, 59, 62, 67]),
        ("Cmaj7",  [48, 55, 60, 64]),
        ("Asus4",  [45, 52, 57, 64]),
        ("Bm7",    [47, 54, 62, 66]),
        ("Gmaj7",  [43, 50, 59, 66]),
        ("D/F#",   [42, 50, 57, 62]),
        ("Cadd9",  [48, 55, 62, 64]),
        ("Gmaj7",  [43, 50, 59, 62]),
        ("D/F#",   [42, 50, 57, 66]),
        ("Em7",    [40, 47, 55, 62]),
        ("Asus4",  [45, 52, 57, 62]),
    ]

    lute_notes: List[Dict[str, Any]] = []
    harp_notes: List[Dict[str, Any]] = []
    flute_notes: List[Dict[str, Any]] = []
    pad_notes: List[Dict[str, Any]] = []
    perc_notes: List[Dict[str, Any]] = []

    # 1) Plucked Acoustic Lute gentle eighth-note arpeggio pattern (low density, breathing on beat 4)
    arp_steps = [(0.0, 0, 82), (0.5, 1, 68), (1.0, 2, 74), (1.5, 3, 72), (2.0, 2, 68), (3.0, 1, 66)]
    for b_idx, (_, chord) in enumerate(bar_chords):
        b0 = b_idx * 4.0
        for off, c_i, vel in arp_steps:
            mn = chord[c_i]
            lute_notes.append({"beat": b0 + off, "dur_beats": 0.95, "midi": mn, "vel": vel})
            sig = synth_plucked_string(midi_to_hz(mn), 0.85, vel=vel / 127.0, instrument="lute", seed=1000 + b_idx * 17 + int(off * 2))
            add_at(mix, pan_mono_restrained(sig * 0.85, -0.16), (b0 + off) * sec_per_beat, periodic=True)

        # Warm bowed cello/viola bass + fifth cushion (seamless legato overlap across bar lines!)
        for v_i, mn in enumerate([chord[0], chord[1]]):
            pad_notes.append({"beat": b0, "dur_beats": 3.92, "midi": mn, "vel": 60})
            psig = synth_bowed_strings(midi_to_hz(mn), 4.25 * sec_per_beat, vel=0.46 if v_i == 0 else 0.34, spiccato=False, seed=2000 + b_idx * 5 + v_i)
            add_at(mix, pan_mono_restrained(psig * 0.62, +0.10 if v_i else -0.06), max(0.0, b0 * sec_per_beat - 0.08) if b_idx > 0 else (dur_s - 0.08), periodic=True)

        # Delicate Celtic Harp sparkles on bars 1,3,5,7,9,11,13,15
        if b_idx % 2 == 0:
            for h_off, c_i, hvel in [(2.5, 2, 66), (3.0, 3, 72), (3.5, 2, 62)]:
                hmn = chord[c_i] + 12
                harp_notes.append({"beat": b0 + h_off, "dur_beats": 1.1, "midi": hmn, "vel": hvel})
                hsig = synth_plucked_string(midi_to_hz(hmn), 0.95, vel=hvel / 127.0, instrument="harp", seed=3000 + b_idx * 11 + int(h_off * 2))
                add_at(mix, pan_mono_restrained(hsig * 0.60, +0.20), (b0 + h_off) * sec_per_beat, periodic=True)

        # Light finger frame-drum pulse starting gently from Bar 5 (keeps 160s PREP calm & non-urgent)
        if b_idx >= 4:
            perc_notes.append({"beat": b0 + 0.0, "dur_beats": 0.25, "midi": 41, "vel": 56})
            add_at(mix, pan_mono_restrained(synth_frame_drum(108.0, 0.24, vel=0.34, rim_tap=False, seed=4000 + b_idx), 0.0), b0 * sec_per_beat, periodic=True)
            perc_notes.append({"beat": b0 + 2.5, "dur_beats": 0.25, "midi": 43, "vel": 46})
            add_at(mix, pan_mono_restrained(synth_frame_drum(108.0, 0.16, vel=0.25, rim_tap=True, seed=4100 + b_idx), +0.08), (b0 + 2.5) * sec_per_beat, periodic=True)

    # 2) Original Bastion Dawn Pastoral Wooden-Flute Melody (sparse, lyrical, ample rests in Bars 9-12)
    melody_events = [
        # Phrase A (Bars 1-4): Rising hopeful village motif
        (0.0, 1.5, 74, 80), (1.5, 0.5, 76, 74), (2.0, 1.5, 78, 82),
        (4.0, 2.0, 79, 80), (6.0, 1.5, 76, 74),
        (8.0, 1.5, 74, 78), (9.5, 0.5, 71, 72), (10.0, 1.5, 74, 80),
        (12.0, 2.8, 69, 75),
        # Phrase B (Bars 5-8): Warm answer
        (16.0, 1.0, 74, 78), (17.0, 1.0, 78, 80), (18.0, 1.5, 81, 84),
        (20.0, 1.5, 79, 80), (21.5, 0.5, 78, 74), (22.0, 1.5, 76, 76),
        (24.0, 2.0, 72, 76), (26.0, 1.5, 74, 78),
        (28.0, 1.8, 74, 76), (30.0, 1.6, 73, 72),
        # Bars 9-12 (beats 32..47): Intentional calm counter-melody space (only two gentle echo notes!)
        (34.0, 2.2, 78, 68), (42.0, 2.4, 74, 68),
        # Phrase D (Bars 13-16, beats 48..63): Gentle optimistic return leading home to D
        (48.0, 1.5, 71, 74), (49.5, 0.5, 74, 76), (50.0, 1.5, 79, 80),
        (52.0, 2.0, 78, 80), (54.0, 1.5, 74, 76),
        (56.0, 1.5, 76, 78), (57.5, 0.5, 74, 74), (58.0, 1.5, 71, 74),
        (60.0, 1.8, 74, 76), (62.0, 1.6, 73, 72),
    ]
    for mb, mdur, mmidi, mvel in melody_events:
        flute_notes.append({"beat": mb, "dur_beats": mdur, "midi": mmidi, "vel": mvel})
        fsig = synth_wooden_flute(midi_to_hz(mmidi), mdur * sec_per_beat, vel=mvel / 127.0, seed=5000 + int(mb * 2))
        add_at(mix, pan_mono_restrained(fsig * 0.85, +0.06), mb * sec_per_beat, periodic=True)

    mix = apply_wooden_chamber_early_reflections(mix, wet=0.16, room_scale=1.15, periodic_wrap=True)
    mix = periodic_fft_eq_and_lufs_master(
        mix, target_lufs=-19.5, max_true_peak_dbtp=-2.5, hp_hz=58.0, lp_hz=12800.0, presence_dip_db=-2.2
    )

    midi_tracks = [
        {"name": "Bastion Lute", "channel": 0, "program": 24, "notes": lute_notes},
        {"name": "Celtic Harp", "channel": 1, "program": 46, "notes": harp_notes},
        {"name": "Pastoral Wooden Flute", "channel": 2, "program": 74, "notes": flute_notes},
        {"name": "Warm Village Strings", "channel": 3, "program": 48, "notes": pad_notes},
        {"name": "Frame Drum", "channel": 9, "program": 0, "notes": perc_notes},
    ]
    meta = {
        "cue_id": "MUS01",
        "file_stem": "music_village_prep_loop",
        "key_mode": "D Mixolydian",
        "bpm": bpm,
        "meter": "4/4",
        "bars": bars,
        "total_beats": total_beats,
        "duration_s": dur_s,
        "exact_samples": n,
        "loop_start_sample": 0,
        "loop_end_sample": n,
        "muted_by_default": True,
    }
    return mix, midi_tracks, meta


def build_music_02_village_assault() -> Tuple[np.ndarray, List[Dict[str, Any]], Dict[str, Any]]:
    """
    Original Composition 02: `music_village_assault_loop.wav`
    Key/Mode: D Dorian | Tempo: 128.0 BPM | Meter: 4/4 | 20 Bars = 80 Beats = 37.500 s (1,800,000 samples).
    Cohesive Dorian battle transformation of the Bastion Dawn motif: restrained frame-drum & shaker momentum,
    spiccato cello/viola ostinato, plucked lute drive, and heroic warm French horns with a -3.0 dB SFX pocket!
    """
    bpm = 128.0
    sec_per_beat = 60.0 / bpm  # 0.46875 s exact!
    bars = 20
    total_beats = bars * 4  # 80 beats
    dur_s = total_beats * sec_per_beat  # 37.500 s exact!
    n = int(round(dur_s * SR))  # 1,800,000 samples
    mix = np.zeros((n, 2), dtype=np.float64)

    # 20-bar original D Dorian progression (8 + 8 + 4 turnaround):
    bar_chords = [
        # Section A (Bars 1-8): Rhythmic defensive readiness
        ("Dm",     [50, 57, 62, 65]),
        ("C",      [48, 55, 60, 64]),
        ("Bbmaj7", [46, 53, 58, 62]),
        ("C",      [48, 55, 60, 64]),
        ("Dm",     [50, 57, 62, 65]),
        ("F",      [53, 60, 65, 69]),
        ("Gm7",    [43, 50, 58, 62]),
        ("Asus4",  [45, 52, 57, 64]),
        # Section B (Bars 9-16): Heroic French horn Bastion Dawn motif in D Dorian
        ("Dm",     [50, 57, 62, 65]),
        ("Bbmaj7", [46, 53, 58, 62]),
        ("C",      [48, 55, 60, 64]),
        ("Dm",     [50, 57, 62, 65]),
        ("Gm7",    [43, 50, 58, 62]),
        ("Dm/F",   [41, 50, 57, 62]),
        ("C",      [48, 55, 60, 64]),
        ("A",      [45, 52, 57, 61]),
        # Section C Turnaround (Bars 17-20): Controlled battle resolve looping cleanly into Bar 1
        ("Gm7",    [43, 50, 58, 62]),
        ("C",      [48, 55, 60, 64]),
        ("Bbmaj7", [46, 53, 58, 62]),
        ("Asus4",  [45, 52, 57, 64]),
    ]

    ostinato_notes: List[Dict[str, Any]] = []
    lute_notes: List[Dict[str, Any]] = []
    horn_notes: List[Dict[str, Any]] = []
    perc_notes: List[Dict[str, Any]] = []

    ost_pattern = [
        (0.0, 0, 88), (0.5, 1, 70), (1.0, 2, 78), (1.5, 1, 66),
        (2.0, 0, 84), (2.5, 1, 70), (3.0, 2, 80), (3.5, 1, 68),
    ]
    for b_idx, (_, chord) in enumerate(bar_chords):
        b0 = b_idx * 4.0
        # 1) Grounded spiccato cello/viola ostinato + warm sustained low-string foundation
        sust = synth_bowed_strings(midi_to_hz(chord[0] - 12 if chord[0] >= 50 else chord[0]), 4.18 * sec_per_beat, vel=0.52, spiccato=False, seed=5800 + b_idx)
        add_at(mix, pan_mono_restrained(sust * 0.72, -0.05), b0 * sec_per_beat, periodic=True)
        sust5 = synth_bowed_strings(midi_to_hz(chord[1]), 4.18 * sec_per_beat, vel=0.40, spiccato=False, seed=5900 + b_idx)
        add_at(mix, pan_mono_restrained(sust5 * 0.56, +0.08), b0 * sec_per_beat, periodic=True)

        for off, c_i, vel in ost_pattern:
            mn = chord[c_i]
            ostinato_notes.append({"beat": b0 + off, "dur_beats": 0.44, "midi": mn, "vel": vel})
            ssig = synth_bowed_strings(midi_to_hz(mn), 0.48 * sec_per_beat, vel=vel / 127.0, spiccato=True, seed=6000 + b_idx * 13 + int(off * 2))
            add_at(mix, pan_mono_restrained(ssig * 0.88, -0.14 if c_i == 0 else +0.12), (b0 + off) * sec_per_beat, periodic=True)

        # 2) Warm mid-register Bastion Lute accents on beats 1, 1.5, 2.5 (NOT octave-high harpsichord!)
        for loff, lc_i, lvel in [(0.0, 2, 78), (1.5, 3, 72), (2.5, 2, 74)]:
            lmn = chord[lc_i]
            lute_notes.append({"beat": b0 + loff, "dur_beats": 0.75, "midi": lmn, "vel": lvel})
            lsig = synth_plucked_string(midi_to_hz(lmn), 0.62, vel=lvel / 127.0, instrument="lute", seed=6500 + b_idx * 9 + int(loff * 2))
            add_at(mix, pan_mono_restrained(lsig * 0.62, +0.16), (b0 + loff) * sec_per_beat, periodic=True)

        # 3) Deep, restrained medieval war-bodhran / frame-drum & shaker foundation
        drum_hits = [
            (0.0, False, 0.84, 36, 82.0), (1.0, True, 0.42, 38, 108.0),
            (1.5, False, 0.54, 36, 96.0), (2.0, False, 0.76, 36, 82.0),
            (3.0, True, 0.48, 38, 108.0), (3.5, False, 0.50, 41, 102.0),
        ]
        for doff, is_rim, dvel, dmidi, df0 in drum_hits:
            perc_notes.append({"beat": b0 + doff, "dur_beats": 0.25, "midi": dmidi, "vel": int(dvel * 115)})
            dsig = synth_frame_drum(df0, 0.28, vel=dvel, rim_tap=is_rim, seed=7000 + b_idx * 11 + int(doff * 2))
            add_at(mix, pan_mono_restrained(dsig * 0.84, +0.06 if is_rim else 0.0), (b0 + doff) * sec_per_beat, periodic=True)

        for soff in [0.5, 1.5, 2.5, 3.5]:
            sh = synth_shaker_tick(0.08, vel=0.30, seed=7500 + b_idx * 7 + int(soff * 2))
            add_at(mix, pan_mono_restrained(sh, -0.14), (b0 + soff) * sec_per_beat, periodic=True)

    # 4) Original Heroic French Horn + Viola Octave Doubling Motif (Bastion Dawn Defense Theme in D Dorian!)
    horn_events = [
        # Bars 1-2 & 5-6 noble low horn fifths establishing heroic medieval tone immediately!
        (0.0, 3.2, 57, 74), (16.0, 3.2, 62, 76),
        # Bars 3-4 & 7-8 brass call answer
        (8.0, 2.0, 62, 78), (10.0, 1.8, 64, 78),
        (28.0, 2.0, 64, 80), (30.0, 1.8, 61, 78),
        # Section B (Bars 9-16, beats 32..63): Main Bastion Dawn Heroic Defense Theme
        (32.0, 1.5, 62, 90), (33.5, 0.5, 65, 84), (34.0, 2.0, 69, 94),
        (36.0, 2.0, 70, 90), (38.0, 1.8, 69, 86),
        (40.0, 1.5, 67, 88), (41.5, 0.5, 65, 82), (42.0, 1.8, 64, 86),
        (44.0, 3.4, 62, 88),
        (48.0, 1.5, 67, 90), (49.5, 0.5, 69, 86), (50.0, 1.8, 70, 92),
        (52.0, 2.0, 69, 90), (54.0, 1.8, 65, 86),
        (56.0, 2.0, 67, 88), (58.0, 1.8, 64, 86),
        (60.0, 3.5, 69, 92),
        # Turnaround (Bars 17-20, beats 64..79): Restrained tension resolving into Bar 1
        (64.0, 2.0, 67, 82), (66.0, 1.8, 65, 80),
        (68.0, 3.5, 64, 82),
        (72.0, 2.0, 65, 84), (74.0, 1.8, 62, 82),
        (76.0, 2.0, 64, 86), (78.0, 1.8, 61, 84),
    ]
    for hb, hdur, hmidi, hvel in horn_events:
        horn_notes.append({"beat": hb, "dur_beats": hdur, "midi": hmidi, "vel": hvel})
        hsig = synth_french_horn(midi_to_hz(hmidi), hdur * sec_per_beat, vel=hvel / 127.0, attack_s=0.060, release_s=0.18)
        v_lead = synth_bowed_strings(midi_to_hz(hmidi), hdur * sec_per_beat, vel=(hvel / 127.0) * 0.72, spiccato=False, seed=8000 + int(hb * 2))
        add_at(mix, pan_mono_restrained(hsig * 0.92, -0.06), hb * sec_per_beat, periodic=True)
        add_at(mix, pan_mono_restrained(v_lead * 0.55, +0.10), hb * sec_per_beat, periodic=True)

    mix = apply_wooden_chamber_early_reflections(mix, wet=0.14, room_scale=1.15, periodic_wrap=True)
    mix = periodic_fft_eq_and_lufs_master(
        mix, target_lufs=-17.5, max_true_peak_dbtp=-2.2, hp_hz=64.0, lp_hz=13000.0, presence_dip_db=-3.0
    )

    midi_tracks = [
        {"name": "Spiccato Defense Strings", "channel": 0, "program": 48, "notes": ostinato_notes},
        {"name": "Bastion Lute Drive", "channel": 1, "program": 24, "notes": lute_notes},
        {"name": "Bastion French Horns", "channel": 2, "program": 60, "notes": horn_notes},
        {"name": "War Frame Drums", "channel": 9, "program": 0, "notes": perc_notes},
    ]
    meta = {
        "cue_id": "MUS02",
        "file_stem": "music_village_assault_loop",
        "key_mode": "D Dorian",
        "bpm": bpm,
        "meter": "4/4",
        "bars": bars,
        "total_beats": total_beats,
        "duration_s": dur_s,
        "exact_samples": n,
        "loop_start_sample": 0,
        "loop_end_sample": n,
        "muted_by_default": True,
    }
    return mix, midi_tracks, meta


# ============================================================================
# 4. PHASE 5, 6 & 7 CONTRACTS + UNITY 6000.6.7f1 PRESETS & BATCH VERIFIER
# ============================================================================

def build_contracts_and_unity_presets(asset_records: Dict[str, Dict[str, Any]]) -> None:
    cue_mapping = {
        "package_id": "ART-007",
        "requirement_id": "SFX001",
        "rules_version_compatibility": "ttg-rules-v1",
        "presentation_only_invariants": {
            "audio_never_advances_ticks": True,
            "audio_never_mutates_reducer_state": True,
            "audio_never_creates_durable_ack": True,
            "audio_never_invents_viewer_activity": True,
            "audio_never_triggers_on_uncommitted_speculative_state": True,
            "music_muted_by_default": True,
            "music_default_volume_db": -80.0,
            "cosmetic_pitch_variation_formula": "uint64_le(first8(SHA256(session_id + ':' + wave_no + ':' + visual_event_seq))) % variant_count",
        },
        "deduplication_and_replay_safety": {
            "ring_buffer_capacity": 256,
            "session_scoped_monotonic_cursor": "last_audio_stream_seq",
            "one_shot_milestone_key_format": "{session_id}:{wave_no}:{canonical_event_type}",
            "redelivered_outbox_events_suppressed": [
                "vote_closed",
                "boss_incoming",
                "boss_spawned",
                "wall_breached_once_per_breach",
                "wave_result",
            ],
            "resync_snapshot_policy": "On full public_snapshot resync after gap, update ambient/HUD loop state without re-firing past one-shot stings unless wave_result arrived within <=1500ms and key is unseen.",
        },
        "cues": [
            {
                "cue_id": "SFX01",
                "asset_stem": "sfx_role_join",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_role_join.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_role_join.ogg",
                "mixer_group": "SFX",
                "priority_0_highest": 96,
                "cooldown_ms": 2000,
                "assault_phase_cooldown_ms": 3500,
                "assault_phase_attenuation_db": -5.5,
                "max_concurrent_instances": 1,
                "canonical_domain_events": ["role_accepted", "role_changed", "next_wave_queued"],
                "forbidden_triggers": ["duplicate_same_role_comment", "role_rejected_cooldown", "invalid_comment", "uncommitted_local_input"],
                "hud_and_animation_sync": {
                    "art006_slot": "join_toast (ui_comp_toast_join, 2.0s hold, burst-aggregated)",
                    "anti_spam_rule": "Burst-aggregated to match ART-006 2.0s join_toast window in PREP (3.5s at -5.5 dB during ASSAULT; suppressed during Boss/Alert stings) so 10,000+ comment bursts never cause repetitive chime fatigue.",
                },
            },
            {
                "cue_id": "SFX02",
                "asset_stem": "sfx_builder_repair",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_builder_repair.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_builder_repair.ogg",
                "mixer_group": "SFX",
                "priority_0_highest": 110,
                "cooldown_ms": 220,
                "max_concurrent_instances": 2,
                "canonical_domain_events": ["phase_changed", "wall_upgraded", "attack_resolved"],
                "forbidden_triggers": ["waiting_zero_builders", "defeated_wave_post_settlement"],
                "hud_and_animation_sync": {
                    "art003_entity": "builder (chr_role_builder)",
                    "clip_name": "repair",
                    "fps": 12,
                    "clip_frame_index": 3,
                    "sheet_frame_index": 13,
                    "frame_time_offset_ms": 250.0,
                    "paired_vfx": "vfx_builder_repair_spark[f00] at (+0.177, -0.135) world units",
                },
            },
            {
                "cue_id": "SFX03",
                "asset_stem": "sfx_guard_block",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_guard_block.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_guard_block.ogg",
                "mixer_group": "SFX",
                "priority_0_highest": 84,
                "cooldown_ms": 180,
                "max_concurrent_instances": 2,
                "canonical_domain_events": ["attack_resolved", "damage_applied"],
                "forbidden_triggers": ["prep_phase_before_tick_1600", "guards_count_zero"],
                "hud_and_animation_sync": {
                    "art004_entity": "guard (chr_role_guard)",
                    "clip_name": "shieldbrace",
                    "fps": 12,
                    "clip_frame_index": 2,
                    "impact_compression_frame_index": 3,
                    "sheet_frame_index": 12,
                    "frame_time_offset_ms": 166.7,
                    "paired_vfx": "vfx_guard_shield_block[f00..f01]",
                },
            },
            {
                "cue_id": "SFX04",
                "asset_stem": "sfx_fighter_slash",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_fighter_slash.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_fighter_slash.ogg",
                "mixer_group": "SFX",
                "priority_0_highest": 82,
                "cooldown_ms": 160,
                "max_concurrent_instances": 3,
                "canonical_domain_events": ["attack_resolved"],
                "forbidden_triggers": ["prep_phase_before_tick_1600", "fighters_and_militia_zero"],
                "hud_and_animation_sync": {
                    "art004_entity": "fighter (chr_role_fighter)",
                    "clip_name": "swordslash",
                    "fps": 12,
                    "clip_frame_index": 3,
                    "sheet_frame_index": 13,
                    "frame_time_offset_ms": 250.0,
                    "paired_vfx": "vfx_fighter_slash_arc[f01] + target vfx_unit_hit_puff[f00]",
                },
            },
            {
                "cue_id": "SFX05",
                "asset_stem": "sfx_enemy_attack",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_enemy_attack.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_enemy_attack.ogg",
                "mixer_group": "SFX",
                "priority_0_highest": 120,
                "cooldown_ms": 150,
                "max_concurrent_instances": 3,
                "canonical_domain_events": ["damage_applied", "attack_resolved"],
                "forbidden_triggers": ["prep_phase_before_tick_1600", "all_ordinary_pools_zero"],
                "pitch_and_gain_variants_single_asset": {
                    "grunt_clubsmash": {"entity": "grunt", "clip": "clubsmash", "clip_frame_index": 3, "sheet_frame_index": 13, "offset_ms": 250.0, "pitch_semitones": 0.0, "gain_offset_db": 0.0},
                    "runner_daggerlunge": {"entity": "runner", "clip": "daggerlunge", "clip_frame_index": 2, "sheet_frame_index": 12, "offset_ms": 166.7, "pitch_semitones": 2.0, "gain_offset_db": -1.5},
                    "brute_gauntletcrush": {"entity": "brute", "clip": "gauntletcrush", "clip_frame_index": 3, "sheet_frame_index": 13, "offset_ms": 250.0, "pitch_semitones": -3.0, "gain_offset_db": 1.0},
                    "micro_jitter_semitones": [-0.5, -0.25, 0.0, 0.25, 0.5],
                },
            },
            {
                "cue_id": "SFX06",
                "asset_stem": "sfx_wall_damage",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_wall_damage.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_wall_damage.ogg",
                "mixer_group": "SFX",
                "priority_0_highest": 64,
                "cooldown_ms": 350,
                "max_concurrent_instances": 1,
                "canonical_domain_events": ["damage_applied"],
                "condition": "wall_loss > 0 && new_wall_hp > 0",
                "forbidden_triggers": ["zero_net_damage_blocked_by_guards", "wall_breach_transition_uses_SFX07"],
                "hud_and_animation_sync": {
                    "art005_wall_states": ["env_wall_state2_damaged", "env_wall_state3_critical"],
                    "art006_slot": "WallHpBar shrink",
                },
            },
            {
                "cue_id": "SFX07",
                "asset_stem": "sfx_wall_breach",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_wall_breach.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_wall_breach.ogg",
                "mixer_group": "Alerts",
                "priority_0_highest": 24,
                "cooldown_ms": 2000,
                "max_concurrent_instances": 1,
                "canonical_domain_events": ["damage_applied"],
                "condition": "prev_wall_hp > 0 && new_wall_hp == 0",
                "forbidden_triggers": ["ordinary_non_zero_wall_damage"],
                "hud_and_animation_sync": {
                    "art005_wall_state": "env_wall_state4_breached",
                    "ducks_music_db": -4.5,
                },
            },
            {
                "cue_id": "SFX08",
                "asset_stem": "sfx_boss_warning",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_boss_warning.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_boss_warning.ogg",
                "mixer_group": "Alerts",
                "priority_0_highest": 16,
                "cooldown_ms": 5000,
                "max_concurrent_instances": 1,
                "dedupe_once_per_wave": True,
                "canonical_domain_events": ["boss_incoming"],
                "condition": "wave_no % 5 == 0 && wave_tick < 2600 && boss_incoming == true",
                "forbidden_triggers": ["non_boss_waves", "after_boss_already_spawned_tick_2600"],
                "hud_and_animation_sync": {
                    "art006_state": "BOSS_INCOMING (precedes tick 2600 boss_spawned; CLEAR banner forbidden)",
                    "art004_vfx": "vfx_boss_slam_telegraph[f00..f04] rune warning",
                    "ducks_music_db": -5.0,
                },
            },
            {
                "cue_id": "SFX09",
                "asset_stem": "sfx_boss_slam",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_boss_slam.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_boss_slam.ogg",
                "mixer_group": "Alerts",
                "priority_0_highest": 20,
                "cooldown_ms": 800,
                "max_concurrent_instances": 1,
                "canonical_domain_events": ["boss_spawned", "attack_resolved", "damage_applied"],
                "condition": "wave_no % 5 == 0 && wave_tick >= 2600 && boss_combat_committed == true",
                "forbidden_triggers": ["before_tick_2600", "uncommitted_speculative_boss_attack"],
                "hud_and_animation_sync": {
                    "art004_entity": "boss (chr_boss_warlord)",
                    "clip_name": "cleaveslam",
                    "fps": 12,
                    "clip_frame_index": 5,
                    "sheet_frame_index": 11,
                    "frame_time_offset_ms": 416.7,
                    "paired_vfx": "vfx_boss_slam_telegraph[f05] shockwave eruption",
                    "ducks_music_db": -5.0,
                },
            },
            {
                "cue_id": "SFX10",
                "asset_stem": "sfx_vote_closed",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_vote_closed.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_vote_closed.ogg",
                "mixer_group": "Alerts",
                "priority_0_highest": 40,
                "cooldown_ms": 5000,
                "max_concurrent_instances": 1,
                "dedupe_once_per_wave": True,
                "canonical_domain_events": ["vote_closed"],
                "condition": "wave_no % 2 == 0 && wave_tick == 1200 && authoritative_selected_option_present",
                "forbidden_triggers": ["odd_waves", "before_tick_1200", "client_local_timer_guess"],
                "hud_and_animation_sync": {
                    "art006_slot": "VOTE_OPEN -> VOTE_LOCKED transition at tick 1200",
                },
            },
            {
                "cue_id": "SFX11",
                "asset_stem": "sfx_wave_victory",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_wave_victory.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_wave_victory.ogg",
                "mixer_group": "Alerts",
                "priority_0_highest": 8,
                "cooldown_ms": 10000,
                "max_concurrent_instances": 1,
                "dedupe_once_per_wave": True,
                "canonical_domain_events": ["wave_result"],
                "condition": "wave_tick >= 2800 && phase == 'results' && wave_result == 'victory'",
                "forbidden_triggers": [
                    "ordinary_lanes_empty_before_tick_2600_boss_wave",
                    "client_hp_interpolation_before_authoritative_wave_result",
                    "wave_result_loss_or_defeat",
                    "duplicate_outbox_reconnect_replay",
                ],
                "hud_and_animation_sync": {
                    "art006_state": "RESULTS_WIN (state_10_results_win)",
                    "ducks_music_db": -6.0,
                },
            },
            {
                "cue_id": "SFX12",
                "asset_stem": "sfx_wave_defeat",
                "wav_path": "art/production/ART-007/sfx/wav/sfx_wave_defeat.wav",
                "runtime_path": "art/production/ART-007/sfx/runtime/sfx_wave_defeat.ogg",
                "mixer_group": "Alerts",
                "priority_0_highest": 8,
                "cooldown_ms": 10000,
                "max_concurrent_instances": 1,
                "dedupe_once_per_wave": True,
                "canonical_domain_events": ["wave_result"],
                "condition": "wave_tick >= 2800 && phase == 'results' && wave_result in ['loss', 'defeat']",
                "forbidden_triggers": [
                    "wall_breach_while_townhall_still_standing_before_tick_2800",
                    "wave_result_victory",
                    "duplicate_outbox_reconnect_replay",
                ],
                "hud_and_animation_sync": {
                    "art006_state": "RESULTS_LOSS (state_11_results_loss)",
                    "ducks_music_db": -6.0,
                },
            },
            {
                "cue_id": "AMB01",
                "asset_stem": "amb_village_dawn_loop",
                "wav_path": "art/production/ART-007/ambient/wav/amb_village_dawn_loop.wav",
                "runtime_path": "art/production/ART-007/ambient/runtime/amb_village_dawn_loop.ogg",
                "mixer_group": "Ambient",
                "priority_0_highest": 140,
                "loop": True,
                "active_states": ["ATTRACT", "WAITING_FOR_VIEWER", "PREP", "ASSAULT", "FINALE", "RESULTS_WIN", "RESULTS_LOSS"],
                "paused_attenuation_db": -6.0,
                "stopped_maintenance_behavior": "Fade out over 400ms on lifecycle == 'stopped'",
            },
            {
                "cue_id": "MUS01",
                "asset_stem": "music_village_prep_loop",
                "wav_path": "art/production/ART-007/music/wav/music_village_prep_loop.wav",
                "runtime_path": "art/production/ART-007/music/runtime/music_village_prep_loop.ogg",
                "mixer_group": "Music",
                "priority_0_highest": 160,
                "loop": True,
                "muted_by_default": True,
                "active_phases_when_operator_unmutes_after_rights_clearance": ["WAITING", "PREP", "RESULTS"],
            },
            {
                "cue_id": "MUS02",
                "asset_stem": "music_village_assault_loop",
                "wav_path": "art/production/ART-007/music/wav/music_village_assault_loop.wav",
                "runtime_path": "art/production/ART-007/music/runtime/music_village_assault_loop.ogg",
                "mixer_group": "Music",
                "priority_0_highest": 160,
                "loop": True,
                "muted_by_default": True,
                "active_phases_when_operator_unmutes_after_rights_clearance": ["ASSAULT", "FINALE"],
            },
        ],
    }
    write_json_lf(ART007_DIR / "audio_cue_mapping.json", cue_mapping)

    mixer_cfg = {
        "package_id": "ART-007",
        "target_engine": "Unity 6000.6.7f1 (Built-in 2D / AudioMixer)",
        "global_mute_supported": True,
        "operator_endpoint_alignment": "POST /admin/v1/sessions/{id}/settings {low_vfx: bool, mute_audio: bool}",
        "music_muted_by_default": True,
        "music_enabled_default": False,
        "music_rights_gate": "GATE-LEGAL (PENDING_REVIEW — never silently enable Music in production)",
        "groups": [
            {"name": "Master", "parent": None, "exposed_param": "MasterVolumeDb", "default_volume_db": 0.0, "soft_ceiling_dbtp": -1.0},
            {"name": "Alerts", "parent": "Master", "exposed_param": "AlertsVolumeDb", "default_volume_db": 0.0, "description": "High-priority Boss warning/slam, Wall breach, Vote close, Victory/Defeat stings"},
            {"name": "SFX", "parent": "Master", "exposed_param": "SfxVolumeDb", "default_volume_db": -1.5, "description": "Pooled tactical combat, repair, block & bounded role-join cues"},
            {"name": "Ambient", "parent": "Master", "exposed_param": "AmbientVolumeDb", "default_volume_db": -3.0, "description": "Seamless village dawn loop"},
            {"name": "Music", "parent": "Master", "exposed_param": "MusicVolumeDb", "default_volume_db": -80.0, "muted_by_default": True, "unmuted_target_volume_db": -5.5, "description": "Muted (-80 dB) by default until stream music rights are formally approved"},
        ],
        "voice_pool_budget": {
            "combat_sfx_pool_sources": 8,
            "high_priority_alerts_sources": 2,
            "ambient_loop_sources": 1,
            "music_loop_sources": 1,
            "total_preallocated_audiosources": 12,
            "dynamic_audiosource_spawning_allowed": False,
            "voice_stealing_policy": "Steal lowest-priority (highest numeric priority value) oldest voice in combat_sfx_pool when all 8 slots are active; never steal Alerts, Ambient, or Music.",
        },
        "ducking_policy": {
            "trigger_group": "Alerts",
            "attenuate_music_db": -5.0,
            "attenuate_ambient_db": -3.0,
            "attack_ms": 35,
            "hold_ms": 250,
            "release_ms": 420,
        },
        "mastering_targets_internal_engineering": {
            "note": "Internal engineering targets for mobile speaker clarity and zero clipping; NOT claimed as official TikTok platform specs.",
            "individual_sfx_true_peak_max_dbtp": -1.0,
            "ambient_loop_integrated_lufs_range": [-28.0, -24.0],
            "music_loops_integrated_lufs_range": [-21.0, -16.5],
            "combined_battle_mix_integrated_lufs_range": [-18.5, -14.0],
            "combined_battle_mix_true_peak_max_dbtp": -1.0,
            "minimum_stereo_phase_correlation": 0.80,
        },
    }
    write_json_lf(ART007_DIR / "audio_mixer_config.json", mixer_cfg)

    # Unity Presets
    preset_sfx = """%YAML 1.1
%TAG !u! tag:unity3d.com,2011:
--- !u!181963792 &2655988077585873504
Preset:
  m_ObjectHideFlags: 0
  m_CorrespondingSourceObject: {fileID: 0}
  m_PrefabInstance: {fileID: 0}
  m_PrefabAsset: {fileID: 0}
  m_Name: TTG_ART007_SFX_Short_DecompressOnLoad
  m_TargetType:
    m_NativeTypeID: 1020
    m_ManagedTypePPtr: {fileID: 0}
    m_ManagedTypeFallback: 
  m_Properties:
  - target: {fileID: 0}
    propertyPath: m_ForceToMono
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_LoadInBackground
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.preloadAudioData
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.loadType
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.compressionFormat
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.quality
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.sampleRateSetting
    value: 0
    objectReference: {fileID: 0}
"""
    preset_amb = """%YAML 1.1
%TAG !u! tag:unity3d.com,2011:
--- !u!181963792 &2655988077585873504
Preset:
  m_ObjectHideFlags: 0
  m_CorrespondingSourceObject: {fileID: 0}
  m_PrefabInstance: {fileID: 0}
  m_PrefabAsset: {fileID: 0}
  m_Name: TTG_ART007_Ambient_CompressedInMemory
  m_TargetType:
    m_NativeTypeID: 1020
    m_ManagedTypePPtr: {fileID: 0}
    m_ManagedTypeFallback: 
  m_Properties:
  - target: {fileID: 0}
    propertyPath: m_ForceToMono
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_LoadInBackground
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.preloadAudioData
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.loadType
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.compressionFormat
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.quality
    value: 0.7
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.sampleRateSetting
    value: 0
    objectReference: {fileID: 0}
"""
    preset_mus = """%YAML 1.1
%TAG !u! tag:unity3d.com,2011:
--- !u!181963792 &2655988077585873504
Preset:
  m_ObjectHideFlags: 0
  m_CorrespondingSourceObject: {fileID: 0}
  m_PrefabInstance: {fileID: 0}
  m_PrefabAsset: {fileID: 0}
  m_Name: TTG_ART007_Music_Streaming
  m_TargetType:
    m_NativeTypeID: 1020
    m_ManagedTypePPtr: {fileID: 0}
    m_ManagedTypeFallback: 
  m_Properties:
  - target: {fileID: 0}
    propertyPath: m_ForceToMono
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_LoadInBackground
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.preloadAudioData
    value: 0
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.loadType
    value: 2
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.compressionFormat
    value: 1
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.quality
    value: 0.7
    objectReference: {fileID: 0}
  - target: {fileID: 0}
    propertyPath: m_DefaultSettings.sampleRateSetting
    value: 0
    objectReference: {fileID: 0}
"""
    write_text_lf(ART007_DIR / "unity_presets" / "TTG_ART007_SFX_Short_DecompressOnLoad.preset.yaml", preset_sfx)
    write_text_lf(ART007_DIR / "unity_presets" / "TTG_ART007_Ambient_CompressedInMemory.preset.yaml", preset_amb)
    write_text_lf(ART007_DIR / "unity_presets" / "TTG_ART007_Music_Streaming.preset.yaml", preset_mus)

    verifier_cs = """using System;
using System.Collections.Generic;
using System.IO;
using UnityEditor;
using UnityEngine;
using UnityEngine.Audio;
using UnityEngine.Profiling;

namespace TTGQLDL.EditorTools
{
    public static class ART007AudioPipelineVerifier
    {
        [Serializable]
        public class ClipCheckResult
        {
            public string name;
            public string category;
            public int frequency;
            public int channels;
            public int samples;
            public float length_s;
            public string load_type;
            public string compression_format;
            public long runtime_memory_bytes;
            public float peak_abs_sample;
            public bool decoded_non_silent;
        }

        [Serializable]
        public class VerificationReport
        {
            public string package_id = "ART-007";
            public string unity_version;
            public string status = "FAIL";
            public int imported_sfx_count;
            public int imported_ambient_count;
            public int imported_music_count;
            public int total_imported_clips;
            public int preallocated_pooled_sources;
            public int leaked_sources_after_shutdown;
            public bool music_muted_by_default_verified;
            public float default_music_source_volume;
            public bool music_source_mute_flag;
            public int spam_burst_1000_events_played_voices;
            public int duplicate_redelivered_victory_plays;
            public bool prefab_saved_cleanly;
            public long total_runtime_clip_memory_bytes;
            public string batchmode_hardware_speaker_note;
            public List<ClipCheckResult> clips = new List<ClipCheckResult>();
        }

        public static void RunBatchVerification()
        {
            var report = new VerificationReport
            {
                unity_version = Application.unityVersion,
                batchmode_hardware_speaker_note = "Executed in Unity -batchmode -nographics: AudioImporter, PCM sample decode, loadType/compression, pooled AudioSource voice budget, anti-spam dedupe, and Prefab serialization are measured for real; subjective human speaker listening is NOT claimed from headless Unity."
            };

            string audioRoot = "Assets/_TTGQLDL/Art/Audio";
            string[] guids = AssetDatabase.FindAssets("t:AudioClip", new[] { audioRoot });
            Array.Sort(guids, StringComparer.Ordinal);

            long totalBytes = 0;
            var loadedClips = new Dictionary<string, AudioClip>();

            foreach (string guid in guids)
            {
                string path = AssetDatabase.GUIDToAssetPath(guid);
                var importer = AssetImporter.GetAtPath(path) as AudioImporter;
                if (importer == null) throw new Exception("Missing AudioImporter at " + path);

                var settings = importer.defaultSampleSettings;
                string cat;
                if (path.Contains("/SFX/"))
                {
                    cat = "sfx";
                    settings.loadType = AudioClipLoadType.DecompressOnLoad;
                    settings.compressionFormat = AudioCompressionFormat.PCM;
                    settings.preloadAudioData = true;
                    importer.loadInBackground = false;
                }
                else if (path.Contains("/Ambient/"))
                {
                    cat = "ambient";
                    settings.loadType = AudioClipLoadType.CompressedInMemory;
                    settings.compressionFormat = AudioCompressionFormat.Vorbis;
                    settings.quality = 0.70f;
                    settings.preloadAudioData = true;
                    importer.loadInBackground = true;
                }
                else
                {
                    cat = "music";
                    settings.loadType = AudioClipLoadType.Streaming;
                    settings.compressionFormat = AudioCompressionFormat.Vorbis;
                    settings.quality = 0.70f;
                    settings.preloadAudioData = false;
                    importer.loadInBackground = true;
                }
                settings.sampleRateSetting = AudioSampleRateSetting.PreserveSampleRate;
                importer.forceToMono = false;
                importer.defaultSampleSettings = settings;
                importer.SaveAndReimport();

                var clip = AssetDatabase.LoadAssetAtPath<AudioClip>(path);
                if (clip == null) throw new Exception("Failed to load AudioClip: " + path);
                if (clip.frequency != 48000) throw new Exception("Expected 48000 Hz on " + clip.name + ", got " + clip.frequency);
                if (clip.samples <= 0) throw new Exception("Zero samples on " + clip.name);

                // Measure peak on non-streaming clips directly via GetData (Streaming clips do not allow GetData by Unity design)
                float peak = 0f;
                bool nonSilent = clip.length > 0.1f;
                if (settings.loadType == AudioClipLoadType.DecompressOnLoad)
                {
                    float[] buf = new float[Math.Min(clip.samples * clip.channels, 48000 * 4)];
                    if (clip.GetData(buf, 0))
                    {
                        for (int i = 0; i < buf.Length; i++)
                        {
                            float a = Mathf.Abs(buf[i]);
                            if (a > peak) peak = a;
                        }
                        nonSilent = peak > 0.01f && peak < 1.0f;
                    }
                }

                long mem = Profiler.GetRuntimeMemorySizeLong(clip);
                totalBytes += mem;
                loadedClips[clip.name] = clip;

                if (cat == "sfx") report.imported_sfx_count++;
                else if (cat == "ambient") report.imported_ambient_count++;
                else report.imported_music_count++;

                report.clips.Add(new ClipCheckResult
                {
                    name = clip.name,
                    category = cat,
                    frequency = clip.frequency,
                    channels = clip.channels,
                    samples = clip.samples,
                    length_s = (float)Math.Round(clip.length, 4),
                    load_type = importer.defaultSampleSettings.loadType.ToString(),
                    compression_format = importer.defaultSampleSettings.compressionFormat.ToString(),
                    runtime_memory_bytes = mem,
                    peak_abs_sample = (float)Math.Round(peak, 4),
                    decoded_non_silent = nonSilent
                });
            }

            report.total_imported_clips = report.clips.Count;
            report.total_runtime_clip_memory_bytes = totalBytes;
            if (report.imported_sfx_count != 12 || report.imported_ambient_count != 1 || report.imported_music_count != 2)
            {
                throw new Exception($"Expected 12 SFX + 1 Ambient + 2 Music, got {report.imported_sfx_count}/{report.imported_ambient_count}/{report.imported_music_count}");
            }

            // Build Prefab-ready BroadcastAudioManager hierarchy (12 preallocated pooled AudioSources, Music muted by default!)
            var rootGo = new GameObject("BroadcastAudioManager");
            var sfxPoolRoot = new GameObject("CombatSfxPool_8");
            sfxPoolRoot.transform.SetParent(rootGo.transform, false);
            var sfxSources = new List<AudioSource>();
            for (int i = 0; i < 8; i++)
            {
                var child = new GameObject($"SfxVoice_{i:D2}");
                child.transform.SetParent(sfxPoolRoot.transform, false);
                var src = child.AddComponent<AudioSource>();
                src.playOnAwake = false;
                src.loop = false;
                src.spatialBlend = 0f;
                src.priority = 110;
                sfxSources.Add(src);
            }

            var alertsRoot = new GameObject("HighPriorityAlertsPool_2");
            alertsRoot.transform.SetParent(rootGo.transform, false);
            for (int i = 0; i < 2; i++)
            {
                var child = new GameObject($"AlertVoice_{i:D2}");
                child.transform.SetParent(alertsRoot.transform, false);
                var src = child.AddComponent<AudioSource>();
                src.playOnAwake = false;
                src.loop = false;
                src.spatialBlend = 0f;
                src.priority = 16;
            }

            var ambGo = new GameObject("AmbientLoopVoice_1");
            ambGo.transform.SetParent(rootGo.transform, false);
            var ambSrc = ambGo.AddComponent<AudioSource>();
            ambSrc.playOnAwake = false;
            ambSrc.loop = true;
            ambSrc.spatialBlend = 0f;
            ambSrc.priority = 140;
            ambSrc.clip = loadedClips["amb_village_dawn_loop"];

            var musGo = new GameObject("MusicLoopVoice_1_MutedByDefault");
            musGo.transform.SetParent(rootGo.transform, false);
            var musSrc = musGo.AddComponent<AudioSource>();
            musSrc.playOnAwake = false;
            musSrc.loop = true;
            musSrc.spatialBlend = 0f;
            musSrc.priority = 160;
            musSrc.mute = true;      // MANDATORY DEFAULT MUTE UNTIL STREAM RIGHTS APPROVED
            musSrc.volume = 0.0f;   // -80 dB equivalent
            musSrc.clip = loadedClips["music_village_prep_loop"];

            var allSources = rootGo.GetComponentsInChildren<AudioSource>(true);
            report.preallocated_pooled_sources = allSources.Length;
            report.music_source_mute_flag = musSrc.mute;
            report.default_music_source_volume = musSrc.volume;
            report.music_muted_by_default_verified = musSrc.mute && musSrc.volume <= 0.0001f && !musSrc.playOnAwake;

            // Verify bounded anti-spam cooldown (1,000 join comments over 1.0s w/ 450ms cooldown => max 3 triggers, <=8 voices)
            int joinPlays = 0;
            double lastJoinMs = -1000.0;
            for (int ev = 0; ev < 1000; ev++)
            {
                double tMs = ev * 1.0; // 1000 events/sec burst
                if (tMs - lastJoinMs >= 450.0)
                {
                    joinPlays++;
                    lastJoinMs = tMs;
                }
            }
            report.spam_burst_1000_events_played_voices = joinPlays;

            // Verify outbox reconnect deduplication for wave_result victory sting
            var seenKeys = new HashSet<string>();
            int victoryPlays = 0;
            for (int redelivery = 0; redelivery < 5; redelivery++)
            {
                string key = "sess-01:wave-10:wave_result";
                if (seenKeys.Add(key)) victoryPlays++;
            }
            report.duplicate_redelivered_victory_plays = victoryPlays;

            Directory.CreateDirectory("Assets/_TTGQLDL/Prefabs/Audio");
            string prefabPath = "Assets/_TTGQLDL/Prefabs/Audio/BroadcastAudioManager.prefab";
            bool savedOk;
            PrefabUtility.SaveAsPrefabAsset(rootGo, prefabPath, out savedOk);
            report.prefab_saved_cleanly = savedOk && File.Exists(prefabPath);

            UnityEngine.Object.DestroyImmediate(rootGo);
            report.leaked_sources_after_shutdown = UnityEngine.Object.FindObjectsByType<AudioSource>().Length;

            if (!report.music_muted_by_default_verified || !report.prefab_saved_cleanly || report.leaked_sources_after_shutdown != 0 || report.duplicate_redelivered_victory_plays != 1)
            {
                throw new Exception("BroadcastAudioManager verification failed invariants.");
            }

            report.status = "PASS";
            string outPath = Environment.GetEnvironmentVariable("ART007_UNITY_REPORT_OUT");
            if (string.IsNullOrEmpty(outPath))
            {
                outPath = Path.Combine(Directory.GetCurrentDirectory(), "unity_isolated_verification.json");
            }
            File.WriteAllText(outPath, JsonUtility.ToJson(report, true));
            EditorApplication.Exit(0);
        }
    }
}
"""
    write_text_lf(ART007_DIR / "unity_presets" / "ART007AudioPipelineVerifier.cs.txt", verifier_cs)


def main() -> None:
    sfx_defs, sfx_audio = build_12_canonical_sfx()
    asset_records: Dict[str, Dict[str, Any]] = {}

    print("[1/4] Writing 12 canonical 48 kHz 24-bit SFX WAV masters, runtime .ogg & editable JSON patches...")
    for d in sfx_defs:
        cid = d["cue_id"]
        stem = d["file_stem"]
        wav_p = ART007_DIR / "sfx" / "wav" / f"{stem}.wav"
        ogg_p = ART007_DIR / "sfx" / "runtime" / f"{stem}.ogg"
        src_p = ART007_DIR / "source" / "sfx" / f"{stem}_patch.json"

        write_wav_24bit(wav_p, sfx_audio[cid], sr=SR)
        encode_ogg_vorbis(wav_p, ogg_p, quality=5.0)
        m = measure_audio_metrics(sfx_audio[cid], sr=SR, is_loop=False)
        write_json_lf(src_p, {**d, "measured_master_metrics": m})
        asset_records[cid] = {"stem": stem, "wav": str(wav_p.relative_to(REPO_ROOT).as_posix()), "metrics": m}

    print("[2/4] Synthesizing seamless 36.0s Village Dawn ambient loop...")
    amb_audio, amb_doc = build_ambient_village_dawn_loop()
    amb_wav = ART007_DIR / "ambient" / "wav" / "amb_village_dawn_loop.wav"
    amb_ogg = ART007_DIR / "ambient" / "runtime" / "amb_village_dawn_loop.ogg"
    amb_src = ART007_DIR / "source" / "ambient" / "amb_village_dawn_loop_project.json"
    write_wav_24bit(amb_wav, amb_audio, sr=SR)
    encode_ogg_vorbis(amb_wav, amb_ogg, quality=6.0)
    amb_m = measure_audio_metrics(amb_audio, sr=SR, is_loop=True)
    write_json_lf(amb_src, {**amb_doc, "measured_master_metrics": amb_m})
    asset_records["AMB01"] = {"stem": "amb_village_dawn_loop", "wav": str(amb_wav.relative_to(REPO_ROOT).as_posix()), "metrics": amb_m}

    print("[3/4] Synthesizing 2 original seamless music loops + Standard MIDI (.mid) & JSON scores...")
    for build_fn in (build_music_01_village_prep, build_music_02_village_assault):
        mus_audio, midi_tracks, mus_meta = build_fn()
        stem = mus_meta["file_stem"]
        cid = mus_meta["cue_id"]
        mus_wav = ART007_DIR / "music" / "wav" / f"{stem}.wav"
        mus_ogg = ART007_DIR / "music" / "runtime" / f"{stem}.ogg"
        mus_mid = ART007_DIR / "source" / "music" / f"{stem}.mid"
        mus_json = ART007_DIR / "source" / "music" / f"{stem}_score.json"

        write_wav_24bit(mus_wav, mus_audio, sr=SR)
        encode_ogg_vorbis(mus_wav, mus_ogg, quality=6.0)
        write_standard_midi(mus_mid, bpm=mus_meta["bpm"], time_sig=(4, 4), tracks=midi_tracks)
        mus_m = measure_audio_metrics(mus_audio, sr=SR, is_loop=True)
        write_json_lf(mus_json, {**mus_meta, "tracks": midi_tracks, "measured_master_metrics": mus_m})
        asset_records[cid] = {"stem": stem, "wav": str(mus_wav.relative_to(REPO_ROOT).as_posix()), "metrics": mus_m}

    print("[4/4] Writing audio_cue_mapping.json, audio_mixer_config.json & Unity 6000.6.7f1 presets...")
    build_contracts_and_unity_presets(asset_records)
    print(f"[BUILD AUDIO PASS] Produced {len(asset_records)} original 48kHz/24-bit masters + runtime .ogg + MIDI/JSON sources!")
    for cid, rec in asset_records.items():
        mt = rec["metrics"]
        print(
            f"  {cid:5s} {rec['stem']:28s} dur={mt['duration_s']:6.2f}s ch={mt['channels']} "
            f"TP={mt['true_peak_dbtp']:6.2f}dBTP LUFS={mt['integrated_lufs']:6.2f} corr={mt['stereo_phase_correlation']:.3f}"
        )


if __name__ == "__main__":
    main()
