#!/usr/bin/env python3
"""
ART-007 Deterministic Physical-Modeling DSP, Standard MIDI Writer & ITU-R BS.1770-4 Analyzer.
100% original procedural synthesis in Python/NumPy/SciPy — zero external audio samples,
zero unlicensed soundfonts, zero copyrighted melodies.
"""

from __future__ import annotations

import hashlib
import math
import struct
import subprocess
import wave
from pathlib import Path
from typing import Any, Dict, List, Sequence, Tuple

import numpy as np
from scipy import signal

SR = 48000


# ============================================================================
# 1. DETERMINISTIC 24-BIT / 16-BIT PCM WAV & OGG VORBIS I/O + MIDI WRITER
# ============================================================================

def write_wav_24bit(path: Path, audio: np.ndarray, sr: int = SR) -> None:
    """Write float64 audio in [-1.0, 1.0] as deterministic 48 kHz 24-bit PCM WAV (pcm_s24le)."""
    path.parent.mkdir(parents=True, exist_ok=True)
    arr = np.asarray(audio, dtype=np.float64)
    if arr.ndim == 1:
        arr = arr[:, None]
    n_samples, n_ch = arr.shape
    clipped = np.clip(arr, -0.999999, 0.999999)
    scaled = np.rint(clipped * 8388607.0).astype(np.int32)

    # Pack little-endian 24-bit signed integers (3 bytes per sample)
    flat = scaled.reshape(-1)
    b0 = (flat & 0xFF).astype(np.uint8)
    b1 = ((flat >> 8) & 0xFF).astype(np.uint8)
    b2 = ((flat >> 16) & 0xFF).astype(np.uint8)
    packed = np.column_stack([b0, b1, b2]).tobytes()

    with wave.open(str(path), "wb") as wf:
        wf.setnchannels(n_ch)
        wf.setsampwidth(3)
        wf.setframerate(sr)
        wf.writeframes(packed)


def read_wav_float64(path: Path) -> Tuple[np.ndarray, int, int, int]:
    """Read 16-bit, 24-bit, or 32-bit PCM WAV and return (audio_float64 [N, C], sr, channels, bit_depth)."""
    with wave.open(str(path), "rb") as wf:
        n_ch = wf.getnchannels()
        sw = wf.getsampwidth()
        sr = wf.getframerate()
        n_frames = wf.getnframes()
        comp = wf.getcomptype()
        if comp != "NONE":
            raise ValueError(f"Unsupported compressed WAV comptype={comp} in {path}")
        raw = wf.readframes(n_frames)

    if len(raw) != n_frames * n_ch * sw:
        raise ValueError(f"Truncated WAV payload in {path}: expected {n_frames * n_ch * sw}, got {len(raw)}")

    if sw == 2:
        vals = np.frombuffer(raw, dtype="<i2").astype(np.float64) / 32768.0
    elif sw == 3:
        u8 = np.frombuffer(raw, dtype=np.uint8).reshape(-1, 3)
        i32 = (
            u8[:, 0].astype(np.int32)
            | (u8[:, 1].astype(np.int32) << 8)
            | (u8[:, 2].astype(np.int32) << 16)
        )
        neg = (i32 & 0x800000) != 0
        i32[neg] -= 1 << 24
        vals = i32.astype(np.float64) / 8388608.0
    elif sw == 4:
        vals = np.frombuffer(raw, dtype="<i4").astype(np.float64) / 2147483648.0
    else:
        raise ValueError(f"Unsupported sample width {sw} bytes in {path}")

    return vals.reshape(n_frames, n_ch), sr, n_ch, sw * 8


def encode_ogg_vorbis(wav_path: Path, ogg_path: Path, quality: float = 6.0) -> None:
    """Encode a WAV master to runtime Ogg Vorbis (.ogg) via FFmpeg libvorbis."""
    ogg_path.parent.mkdir(parents=True, exist_ok=True)
    cmd = [
        "ffmpeg", "-y", "-hide_banner", "-loglevel", "error",
        "-i", str(wav_path),
        "-c:a", "libvorbis", "-qscale:a", f"{quality:.1f}",
        "-map_metadata", "-1",
        "-fflags", "+bitexact", "-flags:a", "+bitexact",
        str(ogg_path),
    ]
    subprocess.run(cmd, check=True)


def _midi_varlen(val: int) -> bytes:
    val = max(0, int(val))
    buf = [val & 0x7F]
    val >>= 7
    while val > 0:
        buf.append((val & 0x7F) | 0x80)
        val >>= 7
    return bytes(reversed(buf))


def write_standard_midi(
    path: Path,
    bpm: float,
    time_sig: Tuple[int, int],
    tracks: List[Dict[str, Any]],
    ppq: int = 480,
) -> None:
    """
    Write an editable Format-1 Standard MIDI File (.mid) retaining the exact original composition.
    Each track dict has: {"name": str, "channel": int (0..15), "program": int (0..127),
    "notes": [{"beat": float, "dur_beats": float, "midi": int, "vel": int}, ...]}.
    """
    path.parent.mkdir(parents=True, exist_ok=True)
    raw_tracks: List[bytes] = []

    # Track 0: Tempo & Time Signature conductor track
    t0_events: List[Tuple[int, bytes]] = []
    us_per_beat = int(round(60_000_000.0 / bpm))
    num, den = time_sig
    den_pow = int(round(math.log2(den)))
    t0_events.append((0, b"\xFF\x03" + _midi_varlen(len(b"Bastion Dawn Conductor")) + b"Bastion Dawn Conductor"))
    t0_events.append((0, b"\xFF\x58\x04" + bytes([num, den_pow, 24, 8])))
    t0_events.append((0, b"\xFF\x51\x03" + struct.pack(">I", us_per_beat)[1:]))

    def pack_track(abs_events: List[Tuple[int, bytes]], end_tick: int = 0) -> bytes:
        # Sort note-offs before note-ons at identical ticks
        abs_events_sorted = sorted(abs_events, key=lambda x: (x[0], 0 if (x[1][0] & 0xF0) == 0x80 else 1))
        out = bytearray()
        prev_tick = 0
        for tick, ev in abs_events_sorted:
            dt = max(0, tick - prev_tick)
            out.extend(_midi_varlen(dt))
            out.extend(ev)
            prev_tick = max(prev_tick, tick)
        final_dt = max(0, end_tick - prev_tick)
        out.extend(_midi_varlen(final_dt))
        out.extend(b"\xFF\x2F\x00")
        return b"MTrk" + struct.pack(">I", len(out)) + bytes(out)

    max_end_tick = max(
        [int(round((n["beat"] + n["dur_beats"]) * ppq)) for tr in tracks for n in tr["notes"]] + [ppq * 4]
    )
    raw_tracks.append(pack_track(t0_events, max_end_tick))

    for tr in tracks:
        ch = int(tr.get("channel", 0)) & 0x0F
        prog = int(tr.get("program", 0)) & 0x7F
        name_b = tr["name"].encode("utf-8")
        evs: List[Tuple[int, bytes]] = [
            (0, b"\xFF\x03" + _midi_varlen(len(name_b)) + name_b),
            (0, bytes([0xC0 | ch, prog])),
        ]
        for n in tr["notes"]:
            st = int(round(float(n["beat"]) * ppq))
            en = max(st + 12, int(round((float(n["beat"]) + float(n["dur_beats"])) * ppq)))
            pitch = max(0, min(127, int(n["midi"])))
            vel = max(1, min(127, int(n.get("vel", 90))))
            evs.append((st, bytes([0x90 | ch, pitch, vel])))
            evs.append((en, bytes([0x80 | ch, pitch, 0])))
        raw_tracks.append(pack_track(evs, max_end_tick))

    header = b"MThd" + struct.pack(">IHHH", 6, 1, len(raw_tracks), ppq)
    path.write_bytes(header + b"".join(raw_tracks))


# ============================================================================
# 2. ITU-R BS.1770-4 / EBU R128 LOUDNESS, TRUE PEAK & SEAM QA ANALYZER
# ============================================================================

def _bs1770_k_weight(audio_2d: np.ndarray, sr: int = SR) -> np.ndarray:
    """Apply exact 48 kHz ITU-R BS.1770-4 Stage-1 high-shelf + Stage-2 RLB high-pass filters."""
    if sr == 48000:
        b1 = np.array([1.53512485958697, -2.69169618940638, 1.19839281085285], dtype=np.float64)
        a1 = np.array([1.0, -1.69065929318241, 0.73248077421585], dtype=np.float64)
        b2 = np.array([1.0, -2.0, 1.0], dtype=np.float64)
        a2 = np.array([1.0, -1.99004745483398, 0.99007225036621], dtype=np.float64)
    else:
        b1, a1 = signal.butter(2, 1500.0 / (sr * 0.5), btype="high")
        b2, a2 = signal.butter(2, 38.0 / (sr * 0.5), btype="high")
    y = signal.lfilter(b1, a1, audio_2d, axis=0)
    y = signal.lfilter(b2, a2, y, axis=0)
    return y


def measure_audio_metrics(audio: np.ndarray, sr: int = SR, is_loop: bool = False) -> Dict[str, Any]:
    """Compute comprehensive acoustic, loudness, true-peak, stereo-phase, and loop-seam metrics."""
    arr = np.asarray(audio, dtype=np.float64)
    if arr.ndim == 1:
        arr = arr[:, None]
    n_samples, n_ch = arr.shape
    dur_s = n_samples / float(sr)

    sample_peak = float(np.max(np.abs(arr))) if n_samples else 0.0
    sample_peak_dbfs = 20.0 * math.log10(max(sample_peak, 1e-12))

    # 4x polyphase oversampled True Peak (ITU-R BS.1770-4)
    os4 = signal.resample_poly(arr, up=4, down=1, axis=0)
    true_peak = float(np.max(np.abs(os4))) if os4.size else 0.0
    true_peak_dbtp = 20.0 * math.log10(max(true_peak, 1e-12))

    dc_offset = float(np.max(np.abs(np.mean(arr, axis=0))))
    rms_lin = float(np.sqrt(np.mean(arr ** 2)))
    rms_dbfs = 20.0 * math.log10(max(rms_lin, 1e-12))
    clipped_samples = int(np.sum(np.abs(arr) >= 0.9995))

    # ITU-R BS.1770-4 K-weighted Loudness (gated 400ms blocks for >=0.4s, full-clip energy for short SFX)
    kw = _bs1770_k_weight(arr, sr)
    ch_energy = np.sum(kw ** 2, axis=1)  # Weight = 1.0 for L, R, Mono
    block_n = int(0.400 * sr)
    hop_n = int(0.100 * sr)
    if n_samples >= block_n:
        csum = np.concatenate([[0.0], np.cumsum(ch_energy)])
        starts = np.arange(0, n_samples - block_n + 1, hop_n)
        block_ms = (csum[starts + block_n] - csum[starts]) / float(block_n)
        momentary_max_lufs = -0.691 + 10.0 * math.log10(max(float(np.max(block_ms)), 1e-12))
        abs_gated = block_ms[block_ms > 10.0 ** ((-70.0 + 0.691) / 10.0)]
        if abs_gated.size > 0:
            rel_thresh = float(np.mean(abs_gated)) * (10.0 ** (-1.0))  # -10 LU relative gate
            rel_gated = abs_gated[abs_gated >= rel_thresh]
            mean_sq = float(np.mean(rel_gated)) if rel_gated.size > 0 else float(np.mean(abs_gated))
            integrated_lufs = -0.691 + 10.0 * math.log10(max(mean_sq, 1e-12))
        else:
            integrated_lufs = -70.0
    else:
        mean_sq = float(np.mean(ch_energy)) if n_samples else 0.0
        integrated_lufs = -0.691 + 10.0 * math.log10(max(mean_sq, 1e-12))
        momentary_max_lufs = integrated_lufs

    # First-sample boundary click / attack truncation check & onset time
    first_sample_abs = float(np.max(np.abs(arr[0]))) if n_samples else 0.0
    last_sample_abs = float(np.max(np.abs(arr[-1]))) if n_samples else 0.0
    mono_env = np.max(np.abs(arr), axis=1)
    above_thresh = np.where(mono_env >= 10.0 ** (-40.0 / 20.0))[0]
    onset_ms = float(above_thresh[0]) * 1000.0 / sr if above_thresh.size > 0 else 0.0

    # Stereo correlation
    if n_ch == 2:
        l_ch, r_ch = arr[:, 0], arr[:, 1]
        denom = float(np.sqrt(np.sum(l_ch ** 2) * np.sum(r_ch ** 2)))
        stereo_corr = float(np.sum(l_ch * r_ch) / denom) if denom > 1e-12 else 1.0
        mono_downmix_rms_diff_db = 20.0 * math.log10(
            max(float(np.sqrt(np.mean((0.5 * (l_ch + r_ch)) ** 2))), 1e-12)
        ) - rms_dbfs
    else:
        stereo_corr = 1.0
        mono_downmix_rms_diff_db = 0.0

    metrics: Dict[str, Any] = {
        "sample_rate_hz": sr,
        "channels": n_ch,
        "samples": n_samples,
        "duration_s": round(dur_s, 4),
        "sample_peak_dbfs": round(sample_peak_dbfs, 2),
        "true_peak_dbtp": round(true_peak_dbtp, 2),
        "rms_dbfs": round(rms_dbfs, 2),
        "integrated_lufs": round(integrated_lufs, 2),
        "momentary_max_lufs": round(momentary_max_lufs, 2),
        "dc_offset": round(dc_offset, 7),
        "clipped_samples": clipped_samples,
        "onset_ms": round(onset_ms, 2),
        "first_sample_abs": round(first_sample_abs, 6),
        "last_sample_abs": round(last_sample_abs, 6),
        "stereo_phase_correlation": round(stereo_corr, 4),
        "mono_downmix_loss_db": round(mono_downmix_rms_diff_db, 2),
    }

    if is_loop and n_samples > sr:
        seam_step = float(np.max(np.abs(arr[0] - arr[-1])))
        interior_steps = np.max(np.abs(np.diff(arr, axis=0)), axis=1)
        p99_step = float(np.percentile(interior_steps, 99.5))
        win = int(0.050 * sr)  # 50ms window across seam vs interior 50ms transitions
        rms_tail = float(np.sqrt(np.mean(arr[-win:] ** 2)))
        rms_head = float(np.sqrt(np.mean(arr[:win] ** 2)))
        seam_rms_jump_db = abs(20.0 * math.log10(max(rms_head, 1e-9) / max(rms_tail, 1e-9)))

        # Compare seam 50ms jump against natural P99 interior 50ms transitions (accounts for musical downbeats!)
        n_w50 = n_samples // win
        w50_rms = np.sqrt(np.mean(arr[: n_w50 * win].reshape(n_w50, win, n_ch) ** 2, axis=(1, 2)))
        interior_50ms_jumps_db = np.abs(20.0 * np.log10(np.maximum(w50_rms[1:], 1e-9) / np.maximum(w50_rms[:-1], 1e-9)))
        p99_50ms_jump_db = float(np.percentile(interior_50ms_jumps_db, 99.0)) if interior_50ms_jumps_db.size else 1.0

        metrics["loop_seam_step_abs"] = round(seam_step, 6)
        metrics["loop_interior_p995_step_abs"] = round(p99_step, 6)
        metrics["loop_seam_step_ratio"] = round(seam_step / max(p99_step, 1e-7), 4)
        metrics["loop_seam_50ms_rms_jump_db"] = round(seam_rms_jump_db, 3)
        metrics["loop_interior_p99_50ms_jump_db"] = round(p99_50ms_jump_db, 3)
        metrics["loop_seam_vs_interior_p99_jump_ratio"] = round(seam_rms_jump_db / max(p99_50ms_jump_db, 0.25), 4)

    return metrics


# ============================================================================
# 3. MASTERING, FILTERING, SPATIALIZATION & PHYSICAL-MODELING SYNTHESIS
# ============================================================================

def midi_to_hz(midi_note: float) -> float:
    return 440.0 * (2.0 ** ((midi_note - 69.0) / 12.0))


def biquad_peaking(sig: np.ndarray, freq_hz: float, gain_db: float, q: float = 1.2, sr: int = SR) -> np.ndarray:
    A = 10.0 ** (gain_db / 40.0)
    w0 = 2.0 * math.pi * freq_hz / sr
    alpha = math.sin(w0) / (2.0 * q)
    cosw = math.cos(w0)
    b0 = 1.0 + alpha * A
    b1 = -2.0 * cosw
    b2 = 1.0 - alpha * A
    a0 = 1.0 + alpha / A
    a1 = -2.0 * cosw
    a2 = 1.0 - alpha / A
    b = np.array([b0, b1, b2], dtype=np.float64) / a0
    a = np.array([1.0, a1 / a0, a2 / a0], dtype=np.float64)
    return signal.lfilter(b, a, sig, axis=0)


def clean_one_shot_master(
    sig: np.ndarray,
    target_peak_dbtp: float,
    hp_hz: float = 50.0,
    lp_hz: float = 13500.0,
    fade_in_ms: float = 2.5,
    fade_out_ms: float = 24.0,
    sr: int = SR,
) -> np.ndarray:
    """Apply DC removal, mobile-friendly high/low-pass, soft saturation, clean boundary fades, and exact True-Peak target."""
    arr = np.asarray(sig, dtype=np.float64).copy()
    if arr.ndim == 1:
        arr = arr[:, None]
    b_hp, a_hp = signal.butter(2, hp_hz / (sr * 0.5), btype="high")
    b_lp, a_lp = signal.butter(3, lp_hz / (sr * 0.5), btype="low")
    arr = signal.lfilter(b_hp, a_hp, arr, axis=0)
    arr = signal.lfilter(b_lp, a_lp, arr, axis=0)
    arr -= np.mean(arr, axis=0, keepdims=True)

    # Gentle analog-style soft knee before normalization
    mx = float(np.max(np.abs(arr)))
    if mx > 1e-9:
        arr = arr / mx
        arr = np.tanh(arr * 1.18) / math.tanh(1.18)

    n = arr.shape[0]
    fi = min(n // 4, max(8, int(round(fade_in_ms * 0.001 * sr))))
    fo = min(n // 2, max(16, int(round(fade_out_ms * 0.001 * sr))))
    arr[:fi] *= (0.5 - 0.5 * np.cos(np.linspace(0.0, math.pi, fi)))[:, None]
    arr[-fo:] *= (0.5 + 0.5 * np.cos(np.linspace(0.0, math.pi, fo)))[:, None]
    arr[0] = 0.0
    arr[-1] = 0.0
    arr -= np.mean(arr, axis=0, keepdims=True)
    arr[0] = 0.0
    arr[-1] = 0.0

    # Normalize to exact 4x-oversampled True Peak target
    os4 = signal.resample_poly(arr, up=4, down=1, axis=0)
    tp = float(np.max(np.abs(os4)))
    target_lin = 10.0 ** (target_peak_dbtp / 20.0)
    if tp > 1e-9:
        arr *= target_lin / tp
    return arr if arr.shape[1] > 1 else arr[:, 0]


def apply_wooden_chamber_early_reflections(
    stereo_or_mono: np.ndarray,
    wet: float = 0.14,
    room_scale: float = 1.0,
    periodic_wrap: bool = False,
    sr: int = SR,
) -> np.ndarray:
    """Short, restrained wooden hall early reflections (zero long muddy reverb tail!)."""
    arr = np.asarray(stereo_or_mono, dtype=np.float64)
    was_1d = arr.ndim == 1
    if was_1d:
        arr = np.column_stack([arr, arr])
    n = arr.shape[0]
    taps_ms_gain_lr = [
        (11.3 * room_scale, 0.42, 0.34),
        (19.7 * room_scale, 0.31, 0.39),
        (31.1 * room_scale, 0.24, 0.21),
        (46.8 * room_scale, 0.16, 0.19),
        (67.4 * room_scale, 0.09, 0.08),
    ]
    b_damp, a_damp = signal.butter(1, 3800.0 / (sr * 0.5), btype="low")
    damped = signal.lfilter(b_damp, a_damp, arr, axis=0)
    refl = np.zeros_like(arr)
    for d_ms, gl, gr in taps_ms_gain_lr:
        d_samp = int(round(d_ms * 0.001 * sr))
        if periodic_wrap:
            refl[:, 0] += gl * np.roll(damped[:, 1], d_samp)
            refl[:, 1] += gr * np.roll(damped[:, 0], d_samp + 7)
        elif d_samp < n:
            refl[d_samp:, 0] += gl * damped[:-d_samp, 1]
            d2 = min(n - 1, d_samp + 7)
            refl[d2:, 1] += gr * damped[:-d2, 0]
    out = arr * (1.0 - 0.25 * wet) + refl * wet
    return 0.5 * (out[:, 0] + out[:, 1]) if was_1d else out


def pan_mono_restrained(mono: np.ndarray, pan: float = 0.0) -> np.ndarray:
    """Constant-power stereo pan clamped to [-0.35, +0.35] so mono correlation stays >= +0.85."""
    p = max(-0.35, min(0.35, float(pan)))
    ang = (p + 1.0) * (math.pi * 0.25)
    return np.column_stack([mono * math.cos(ang), mono * math.sin(ang)])


# ============================================================================
# 4. INSTRUMENT TIMBRES (PLUCKED LUTE/HARP, WOODEN FLUTE, HORN, STRINGS, DRUMS)
# ============================================================================

def synth_plucked_string(
    freq_hz: float,
    dur_s: float,
    vel: float = 0.85,
    pick_pos: float = 0.19,
    inharmonicity: float = 0.00011,
    damping: float = 0.22,
    instrument: str = "lute",
    seed: int = 101,
    sr: int = SR,
) -> np.ndarray:
    """Inharmonic waveguide/modal plucked string with spruce soundboard body resonances."""
    n = max(64, int(round(dur_s * sr)))
    t = np.arange(n, dtype=np.float64) / sr
    rng = np.random.default_rng(seed)

    max_k = min(28, int(16500.0 // max(freq_hz, 20.0)))
    out = np.zeros(n, dtype=np.float64)
    base_t60 = (1.55 if instrument == "harp" else 1.15) * (220.0 / max(freq_hz, 90.0)) ** 0.32

    for k in range(1, max_k + 1):
        fk = freq_hz * k * math.sqrt(1.0 + inharmonicity * (k ** 2))
        if fk >= sr * 0.44:
            break
        comb = abs(math.sin(math.pi * k * pick_pos)) + 0.08
        amp_k = (comb / (k ** (1.28 if instrument == "harp" else 1.12))) * (0.65 + 0.35 * vel)
        tau_k = base_t60 / (1.0 + damping * (k ** 1.65))
        phase = float(rng.uniform(0.0, 0.25)) if k > 1 else 0.0
        out += amp_k * np.sin(2.0 * math.pi * fk * t + phase) * np.exp(-t / max(tau_k, 0.008))

    # Crisp fingernail/plectrum bridge excitation transient (first 6 ms)
    n_tr = min(n, int(0.007 * sr))
    tr = rng.standard_normal(n_tr) * np.exp(-np.linspace(0.0, 5.5, n_tr))
    b_tr, a_tr = signal.butter(2, [450.0 / (sr * 0.5), 3600.0 / (sr * 0.5)], btype="band")
    out[:n_tr] += 0.16 * vel * signal.lfilter(b_tr, a_tr, tr)

    # Soundboard body resonances
    if instrument == "lute":
        out = biquad_peaking(out, 215.0, 3.2, q=1.8, sr=sr)
        out = biquad_peaking(out, 460.0, 2.4, q=2.0, sr=sr)
    else:
        out = biquad_peaking(out, 290.0, 2.5, q=1.6, sr=sr)
        out = biquad_peaking(out, 920.0, 1.8, q=1.8, sr=sr)

    att = min(n, int(0.0025 * sr))
    out[:att] *= np.linspace(0.0, 1.0, att)
    rel = min(n // 3, int(0.035 * sr))
    out[-rel:] *= 0.5 + 0.5 * np.cos(np.linspace(0.0, math.pi, rel))
    return out * vel * 0.42


def synth_wooden_flute(
    freq_hz: float,
    dur_s: float,
    vel: float = 0.80,
    seed: int = 202,
    sr: int = SR,
) -> np.ndarray:
    """Warm pastoral wooden recorder/flute with gentle breath chiff and delayed vibrato."""
    n = max(128, int(round(dur_s * sr)))
    t = np.arange(n, dtype=np.float64) / sr
    rng = np.random.default_rng(seed)

    vib_gate = np.clip((t - 0.14) / 0.16, 0.0, 1.0)
    vib_cents = 6.5 * vib_gate * np.sin(2.0 * math.pi * 4.85 * t)
    inst_f = freq_hz * (2.0 ** (vib_cents / 1200.0))
    phase = 2.0 * math.pi * np.cumsum(inst_f) / sr

    harmonic_weights = [(1, 1.00), (2, 0.34), (3, 0.19), (4, 0.07), (5, 0.03)]
    tone = np.zeros(n, dtype=np.float64)
    for h, w in harmonic_weights:
        if freq_hz * h < sr * 0.42:
            tone += w * np.sin(h * phase)

    # Breath chiff + soft air support
    air = rng.standard_normal(n)
    b_air, a_air = signal.butter(2, [1400.0 / (sr * 0.5), 3900.0 / (sr * 0.5)], btype="band")
    air_f = signal.lfilter(b_air, a_air, air)
    chiff_env = 0.16 * np.exp(-t / 0.032) + 0.022

    # Smooth breath envelope
    att_n = min(n // 3, max(32, int(0.048 * sr)))
    rel_n = min(n // 3, max(64, int(0.085 * sr)))
    env = np.ones(n, dtype=np.float64)
    env[:att_n] = 0.5 - 0.5 * np.cos(np.linspace(0.0, math.pi, att_n))
    env[-rel_n:] = 0.5 + 0.5 * np.cos(np.linspace(0.0, math.pi, rel_n))
    return (tone + air_f * chiff_env) * env * vel * 0.28


def synth_french_horn(
    freq_hz: float,
    dur_s: float,
    vel: float = 0.85,
    attack_s: float = 0.075,
    release_s: float = 0.22,
    sr: int = SR,
) -> np.ndarray:
    """Warm, mellow conical-bore French horn with brass attack bloom, natural dynamic taper, and zero organ buzz."""
    n = max(128, int(round(dur_s * sr)))
    t = np.arange(n, dtype=np.float64) / sr
    rng = np.random.default_rng(int(round(freq_hz * 10)) + n % 997)

    att_n = min(n // 2, max(32, int(attack_s * sr)))
    rel_n = min(n // 2, max(64, int(release_s * sr)))
    env = np.exp(-t / max(0.85, dur_s * 1.45)) * 0.82 + 0.18
    env[:att_n] *= 0.5 - 0.5 * np.cos(np.linspace(0.0, math.pi, att_n))
    env[-rel_n:] *= 0.5 + 0.5 * np.cos(np.linspace(0.0, math.pi, rel_n))

    # Brass brightness blooms right after attack and mellows smoothly into warm round horn sustain
    bloom = np.exp(-np.maximum(0.0, t - attack_s * 0.85) / 0.19) * np.clip(t / max(attack_s * 0.7, 0.01), 0.0, 1.0)

    vib = 4.5 * np.clip((t - 0.16) / 0.22, 0.0, 1.0) * np.sin(2.0 * math.pi * 4.9 * t)
    voices = [(-4.2, 0.30), (0.0, 0.48), (+4.5, 0.30)]
    sig = np.zeros(n, dtype=np.float64)
    for det, v_gain in voices:
        # Subtle initial lip pitch scoop (+12 cents settling in 22ms)
        lip_scoop = 11.0 * np.exp(-t / 0.020)
        f_v = freq_hz * (2.0 ** ((det + vib - lip_scoop) / 1200.0))
        ph = 2.0 * math.pi * np.cumsum(f_v) / sr + float(rng.uniform(0.0, 2.0 * math.pi))
        for h in range(1, 11):
            fh = freq_hz * h
            if fh >= sr * 0.36:
                break
            # Conical horn bell resonance centered around 340-780 Hz; steep roll-off on high harmonics!
            bell_formant = 1.15 * math.exp(-(((fh - 460.0) / 480.0) ** 2)) + 0.22 * math.exp(-(((fh - 1050.0) / 620.0) ** 2)) + 0.10
            if h <= 3:
                h_env = 1.0
                roll = h ** 1.15
            else:
                h_env = 0.22 + 0.78 * bloom * (vel ** 0.7)
                roll = h ** 1.95
            ph_h = float(rng.uniform(0.0, 2.0 * math.pi))
            sig += v_gain * (bell_formant / roll) * np.sin(h * ph + ph_h) * h_env

    # Subtle warm brass air embouchure transient
    air = rng.standard_normal(n) * (0.06 * np.exp(-t / 0.035) + 0.008) * env
    b_a, a_a = signal.butter(2, [350.0 / (sr * 0.5), 1800.0 / (sr * 0.5)], btype="band")
    sig += signal.lfilter(b_a, a_a, air)

    sig = biquad_peaking(sig, 410.0, 3.2, q=1.3, sr=sr)
    b_lp, a_lp = signal.butter(3, 2600.0 / (sr * 0.5), btype="low")
    sig = signal.lfilter(b_lp, a_lp, sig)
    return sig * env * vel * 0.20


def synth_bowed_strings(
    freq_hz: float,
    dur_s: float,
    vel: float = 0.75,
    spiccato: bool = False,
    seed: int = 303,
    sr: int = SR,
) -> np.ndarray:
    """Warm wooden-body cello/viola section — lush legato cushion or grounded spiccato ostinato."""
    n = max(128, int(round(dur_s * sr)))
    t = np.arange(n, dtype=np.float64) / sr
    rng = np.random.default_rng(seed)

    if spiccato:
        att_n = min(n // 3, max(24, int(0.026 * sr)))
        env = (0.78 * np.exp(-t / max(0.16, dur_s * 0.72)) + 0.22 * np.exp(-t / max(0.28, dur_s)))
        env[:att_n] *= 0.5 - 0.5 * np.cos(np.linspace(0.0, math.pi, att_n))
        rel_n = min(n // 3, max(32, int(0.055 * sr)))
        env[-rel_n:] *= 0.5 + 0.5 * np.cos(np.linspace(0.0, math.pi, rel_n))
    else:
        att_n = min(n // 3, max(96, int(0.20 * sr)))
        rel_n = min(n // 3, max(96, int(0.24 * sr)))
        env = np.ones(n, dtype=np.float64)
        env[:att_n] = 0.5 - 0.5 * np.cos(np.linspace(0.0, math.pi, att_n))
        env[-rel_n:] = 0.5 + 0.5 * np.cos(np.linspace(0.0, math.pi, rel_n))

    sig = np.zeros(n, dtype=np.float64)
    for det_c, ph0 in [(-6.5, 0.0), (-1.8, 0.8), (+2.1, 1.7), (+6.8, 2.5)]:
        vib = 5.5 * np.clip((t - 0.10) / 0.18, 0.0, 1.0) * np.sin(2.0 * math.pi * 5.2 * t + ph0)
        fv = freq_hz * (2.0 ** ((det_c + vib) / 1200.0))
        ph = 2.0 * math.pi * np.cumsum(fv) / sr + ph0
        for h in range(1, 11):
            if freq_hz * h >= sr * 0.34:
                break
            ph_h = float(rng.uniform(0.0, 2.0 * math.pi))
            sig += 0.75 * (1.0 / (h ** 1.62)) * np.sin(h * ph + ph_h)

    if spiccato:
        # Warm horsehair bow-on-gut articulation
        rosin = rng.standard_normal(n) * np.exp(-t / 0.022)
        b_r, a_r = signal.butter(2, [420.0 / (sr * 0.5), 2100.0 / (sr * 0.5)], btype="band")
        sig += 0.28 * signal.lfilter(b_r, a_r, rosin)

    sig = biquad_peaking(sig, 225.0, 3.4, q=1.4, sr=sr)
    sig = biquad_peaking(sig, 540.0, 2.2, q=1.5, sr=sr)
    b_lp, a_lp = signal.butter(3, 2750.0 / (sr * 0.5), btype="low")
    sig = signal.lfilter(b_lp, a_lp, sig)
    return sig * env * vel * 0.15


def synth_frame_drum(
    f_base: float = 112.0,
    dur_s: float = 0.32,
    vel: float = 0.85,
    rim_tap: bool = False,
    seed: int = 404,
    sr: int = SR,
) -> np.ndarray:
    """Earthy medieval wooden frame-drum / war-tom (high-passed >60 Hz, warm mobile punch)."""
    n = max(64, int(round(dur_s * sr)))
    t = np.arange(n, dtype=np.float64) / sr
    rng = np.random.default_rng(seed)

    if rim_tap:
        modes = [(1.0, 1.0, 0.045), (2.32, 0.55, 0.028), (3.86, 0.30, 0.018)]
        f0 = f_base * 2.4
    else:
        modes = [(1.0, 1.0, 0.13), (1.593, 0.48, 0.085), (2.136, 0.28, 0.055), (2.653, 0.16, 0.038)]
        f0 = f_base

    out = np.zeros(n, dtype=np.float64)
    pitch_env = 1.0 + (0.16 if not rim_tap else 0.06) * np.exp(-t / 0.018)
    for ratio, amp, tau in modes:
        ph = 2.0 * math.pi * np.cumsum(f0 * ratio * pitch_env) / sr
        out += amp * np.sin(ph) * np.exp(-t / tau)

    skin = rng.standard_normal(n) * np.exp(-t / (0.008 if rim_tap else 0.014))
    b_s, a_s = signal.butter(2, [350.0 / (sr * 0.5), 3400.0 / (sr * 0.5)], btype="band")
    out += (0.38 if rim_tap else 0.25) * signal.lfilter(b_s, a_s, skin)

    att = min(n, int(0.0018 * sr))
    out[:att] *= np.linspace(0.0, 1.0, att)
    rel = min(n // 2, int(0.030 * sr))
    out[-rel:] *= np.linspace(1.0, 0.0, rel)
    return out * vel * 0.45


def synth_shaker_tick(dur_s: float = 0.09, vel: float = 0.4, seed: int = 505, sr: int = SR) -> np.ndarray:
    n = max(32, int(round(dur_s * sr)))
    t = np.arange(n, dtype=np.float64) / sr
    rng = np.random.default_rng(seed)
    env = (t / 0.012) * np.exp(1.0 - t / 0.012)
    env = np.clip(env, 0.0, 1.0) * np.exp(-t / 0.028)
    nz = rng.standard_normal(n)
    b, a = signal.butter(2, [2800.0 / (sr * 0.5), 7600.0 / (sr * 0.5)], btype="band")
    return signal.lfilter(b, a, nz) * env * vel * 0.14
