#!/usr/bin/env python3
"""
ART-008 / ART-009 — Isolated Unity 6000.6.7f1 Vertical-Slice Showcase & Performance Runner.

Executes inside an isolated temporary Unity 6000.6.7f1 project outside `unity/`
(preserving M06 = NOT_STARTED), using real hardware GPU rendering (NO -nographics)
to import all 229 canonical sprites + 15 canonical WAVs, verify all 7 character
classes & transitions, verify BroadcastAudioDispatcher & gain accounting, render
all 12 canonical SyntheticCommittedView HUD states at 1080p / 720p / 360p (+ English
Vote B proof), and benchmark 10 / 50 / 100 / 150 pooled entities with truthful
1%-low FPS and cumulative thread GC allocation counters.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any, Dict, List


TOOLS_DIR = Path(__file__).resolve().parent
REPO_ROOT = TOOLS_DIR.parents[3]
ART006_DIR = REPO_ROOT / "art" / "production" / "ART-006"
ART008_DIR = REPO_ROOT / "art" / "production" / "ART-008"
ART009_DIR = REPO_ROOT / "art" / "production" / "ART-009"
SCENE_SRC_DIR = ART008_DIR / "showcase" / "scene_source"
FIXTURES_DIR = ART008_DIR / "showcase" / "fixtures"
ISOLATED_OUT_DIR = ART008_DIR / "showcase" / "isolated_unity"
QA_OUT_DIR = ART008_DIR / "qa"

UNITY_EXE = Path(r"C:\Program Files\Unity\Hub\Editor\6000.6.7f1\Editor\Unity.exe")

if str(TOOLS_DIR) not in sys.path:
    sys.path.insert(0, str(TOOLS_DIR))

from showcase_contracts import (  # noqa: E402
    build_eight_failure_sequences,
    build_wave10_full_300s_fixture,
    make_snapshot,
)


def verify_csharp_sprite_stems(valid_stems: set[str]) -> None:
    cs_text = (SCENE_SRC_DIR / "ART008UnityShowcaseVerifier.cs").read_text(encoding="utf-8")
    direct_refs = set(
        re.findall(r'"((?:env_|trans_|ui_comp_|ui_icon_|vfx_boss_slam_telegraph_f)[a-z0-9_]*)"', cs_text)
    )
    missing = sorted(r for r in direct_refs if r not in valid_stems)
    if missing:
        raise RuntimeError(f"C# verifier references missing sprite stems: {missing}")


def fixture_to_committed_view(
    state_id: str,
    visual_state_id: str,
    fixture_doc: Dict[str, Any],
    *,
    out_1080p: Path | None = None,
    out_720p: Path | None = None,
    out_360p: Path | None = None,
    skip_gpu_render: bool = False,
) -> Dict[str, Any]:
    s = fixture_doc["public_snapshot"]
    cnts = s["counts"]
    lanes = {ln["id"]: int(ln["ordinary_hp"]) for ln in s["lanes"]}
    boss = s["boss"]
    vote = s["vote"]
    vc = vote.get("counts", {})
    settle = s["settlement"]
    assist = s["system_assist"]
    diag = s["diagnostics"]
    toast = s.get("join_toast")

    return {
        "stateId": state_id,
        "fixtureId": str(fixture_doc.get("fixture_id", state_id)),
        "visualStateId": visual_state_id,
        "lifecycle": str(s["lifecycle"]),
        "sourceStatus": str(s["source_status"]),
        "pauseReasons": list(s.get("pause_reasons", [])),
        "waveNo": int(s["wave_no"]),
        "waveTick": int(s["wave_tick"]),
        "remainingSeconds": int(s["remaining_seconds"]),
        "phase": str(s["phase"]),
        "villageLevel": int(s["village_level"]),
        "wallTier": int(s["wall_tier"]),
        "wallHp": int(s["wall_hp"]),
        "wallMax": int(s["wall_max"]),
        "townhallHp": int(s["townhall_hp"]),
        "townhallMax": int(s["townhall_max"]),
        "wood": int(s["wood"]),
        "stone": int(s["stone"]),
        "food": int(s["food"]),
        "builderCount": int(cnts["builder"]),
        "guardCount": int(cnts["guard"]),
        "fighterCount": int(cnts["fighter"]),
        "uniqueJoined": int(cnts["unique_joined"]),
        "westHp": lanes.get("west", 0),
        "centerHp": lanes.get("center", 0),
        "eastHp": lanes.get("east", 0),
        "bossIncoming": bool(boss.get("boss_incoming", False)),
        "bossExists": bool(boss.get("boss_exists", False)),
        "bossHp": int(boss.get("boss_hp", 0)),
        "bossMaxHp": int(boss.get("boss_max_hp", 0)),
        "voteOpen": bool(vote.get("open", False)),
        "voteSelected": str(vote.get("selected") or ""),
        "voteA": int(vc.get("A", 0)),
        "voteB": int(vc.get("B", 0)),
        "voteC": int(vc.get("C", 0)),
        "waveResult": str(settle.get("wave_result") or ""),
        "isPostLossRecovery": bool(settle.get("is_post_loss_recovery", False)),
        "pendingNextWaveCount": int(settle.get("pending_next_wave_count", 0)),
        "militiaActive": bool(assist.get("militia_active", False)),
        "militiaAttackBonus": int(assist.get("militia_attack_bonus", 0)),
        "connectionState": str(diag.get("connection_state", "live")),
        "metricsAvailable": bool(diag.get("metrics_available", True)),
        "queueFillPct": int(diag.get("queue_fill_pct", 0)),
        "retryableRejectedCount": int(diag.get("retryable_rejected_count", 0)),
        "confirmedValidDrops": int(diag.get("confirmed_valid_drops", 0)),
        "showJoinToast": toast is not None,
        "joinToastAlias": str(toast.get("display_alias", "")) if toast else "",
        "joinToastRole": str(toast.get("role", "")) if toast else "",
        "outputPng1080p": str(out_1080p) if out_1080p else "",
        "outputPng720p": str(out_720p) if out_720p else "",
        "outputPng360p": str(out_360p) if out_360p else "",
        "skipGpuRender": bool(skip_gpu_render),
    }


def build_prep_boundary_views(beats: Dict[str, Dict[str, Any]]) -> List[Dict[str, Any]]:
    """
    Canonical Wave 10 snapshots across all 9 required PREP/ASSAULT boundary ticks:
    999, 1000, 1199, 1200, 1449, 1450, 1500, 1599, 1600.
    """
    specs = [
        (
            "prep_tick_0999",
            "VOTE_OPEN",
            make_snapshot(
                stream_seq=3099, wave_no=10, wave_tick=999, phase="prep",
                total_wins=39, consecutive_wins_before_wave=5, wall_tier=6, wall_hp=1856, townhall_hp=500,
                wood=550, stone=810, food=540, builder=121, guard=24, fighter=35,
                vote_open=True, vote_a=28, vote_b=53, vote_c=21,
            ),
        ),
        ("prep_tick_1000", "PREP_60_WARN", beats["B06_PREP_60_WARN"]["fixture"]),
        (
            "prep_tick_1199",
            "PREP_60_WARN",
            make_snapshot(
                stream_seq=3199, wave_no=10, wave_tick=1199, phase="prep",
                total_wins=39, consecutive_wins_before_wave=5, wall_tier=6, wall_hp=1944, townhall_hp=500,
                wood=560, stone=825, food=540, builder=121, guard=28, fighter=45,
                vote_open=True, vote_a=34, vote_b=68, vote_c=25,
            ),
        ),
        ("prep_tick_1200", "PREP_60_WARN", beats["B07_VOTE_CLOSED"]["fixture"]),
        (
            "prep_tick_1449",
            "PREP_60_WARN",
            make_snapshot(
                stream_seq=3449, wave_no=10, wave_tick=1449, phase="prep",
                total_wins=39, consecutive_wins_before_wave=5, wall_tier=7, wall_hp=2200, townhall_hp=500,
                wood=574, stone=248, food=490, builder=121, guard=31, fighter=52,
                vote_open=False, vote_selected="B", vote_a=34, vote_b=68, vote_c=25,
            ),
        ),
        ("prep_tick_1450", "PREP_15_WARN", beats["B09_PREP_15_WARN"]["fixture"]),
        (
            "prep_tick_1500",
            "PREP_15_WARN",
            make_snapshot(
                stream_seq=3500, wave_no=10, wave_tick=1500, phase="prep",
                total_wins=39, consecutive_wins_before_wave=5, wall_tier=7, wall_hp=2200, townhall_hp=500,
                wood=580, stone=250, food=490, builder=64, guard=80, fighter=60,
                vote_open=False, vote_selected="B", vote_a=34, vote_b=68, vote_c=25,
            ),
        ),
        (
            "prep_tick_1599",
            "PREP_15_WARN",
            make_snapshot(
                stream_seq=3599, wave_no=10, wave_tick=1599, phase="prep",
                total_wins=39, consecutive_wins_before_wave=5, wall_tier=7, wall_hp=2200, townhall_hp=500,
                wood=590, stone=255, food=490, builder=48, guard=92, fighter=64,
                vote_open=False, vote_selected="B", vote_a=34, vote_b=68, vote_c=25,
            ),
        ),
        ("prep_tick_1600", "ASSAULT", beats["B10_ASSAULT_START"]["fixture"]),
    ]
    return [fixture_to_committed_view(sid, vis, fx) for sid, vis, fx in specs]


def build_12_canonical_unity_states(
    only_prep_affected: bool = False,
) -> tuple[List[Dict[str, Any]], Dict[str, Any], List[Dict[str, Any]], List[Dict[str, Any]]]:
    FIXTURES_DIR.mkdir(parents=True, exist_ok=True)
    w10_file = FIXTURES_DIR / "full_wave_300s_wave10_showcase.json"
    if only_prep_affected and w10_file.exists():
        wave10_doc = json.loads(w10_file.read_text(encoding="utf-8"))
        failures = [
            json.loads((FIXTURES_DIR / seq["filename"]).read_text(encoding="utf-8"))
            for seq in build_eight_failure_sequences()
        ]
    else:
        wave10_doc = build_wave10_full_300s_fixture()
        w10_file.write_text(
            json.dumps(wave10_doc, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n"
        )
        failures = build_eight_failure_sequences()
        for seq in failures:
            (FIXTURES_DIR / seq["filename"]).write_text(
                json.dumps(seq, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n"
            )

    beats = {b["beat_id"]: b for b in wave10_doc["beats"]}
    fails = {f["sequence_id"]: f for f in failures}
    defeat_state = wave10_doc["supplementary_states"]["defeat_wave10_breach"]

    hud_dir = ART009_DIR / "previews" / "corrected_hud"
    mob_dir = ART009_DIR / "previews" / "mobile"
    hud_dir.mkdir(parents=True, exist_ok=True)
    mob_dir.mkdir(parents=True, exist_ok=True)

    state_specs = [
        ("01_waiting", "WAITING_FOR_VIEWER", wave10_doc["beats"][0]["fixture"]),
        ("02_prep", "PREP", beats["B02_PREP_CONSTRUCTION_20S"]["fixture"]),
        ("03_vote_open", "VOTE_OPEN", beats["B05_EVEN_WAVE_VOTE_OPEN"]["fixture"]),
        ("04_vote_b_locked", "PREP_60_WARN", beats["B07_VOTE_CLOSED"]["fixture"]),
        ("05_assault", "ASSAULT", beats["B10_ASSAULT_START"]["fixture"]),
        ("06_boss_incoming", "BOSS_INCOMING", beats["B15_BOSS_INCOMING_2450"]["fixture"]),
        ("07_boss_active", "BOSS_ACTIVE", beats["B16_FINALE_BOSS_SPAWN_2600"]["fixture"]),
        ("08_victory", "RESULTS_WIN", beats["B17_SETTLEMENT_VICTORY_2800"]["fixture"]),
        ("09_defeat", "RESULTS_LOSS", defeat_state["fixture"]),
        ("10_source_disconnected", "SOURCE_RECONNECTING", fails["failure_01_source_disconnect"]["fixture"]),
        ("11_storage_unavailable", "STORAGE_UNAVAILABLE", fails["failure_04_storage_unavailable"]["fixture"]),
        ("12_wave_rollover", "PREP_WAVE_11", beats["B19_ROLLOVER_WAVE11_3000"]["fixture"]),
    ]

    states_out: List[Dict[str, Any]] = []
    for sid, vis_id, fix in state_specs:
        skip_render = bool(only_prep_affected and sid != "04_vote_b_locked")
        states_out.append(
            fixture_to_committed_view(
                sid,
                vis_id,
                fix,
                out_1080p=hud_dir / f"{sid}_1080p.png",
                out_720p=mob_dir / f"{sid}_720p.png",
                out_360p=mob_dir / f"{sid}_360p.png",
                skip_gpu_render=skip_render,
            )
        )

    en_vote_state = fixture_to_committed_view(
        "04b_vote_b_locked_en",
        "PREP_60_WARN_EN",
        beats["B07_VOTE_CLOSED"]["fixture"],
        out_1080p=hud_dir / "04b_vote_b_locked_en_1080p.png",
        skip_gpu_render=False,
    )

    extra_prep_renders = [
        fixture_to_committed_view(
            "03b_prep_60_warn",
            "PREP_60_WARN",
            beats["B06_PREP_60_WARN"]["fixture"],
            out_1080p=hud_dir / "03b_prep_60_warn_1080p.png",
            out_720p=mob_dir / "03b_prep_60_warn_720p.png",
            out_360p=mob_dir / "03b_prep_60_warn_360p.png",
        ),
        fixture_to_committed_view(
            "04c_prep_15_warn",
            "PREP_15_WARN",
            beats["B09_PREP_15_WARN"]["fixture"],
            out_1080p=hud_dir / "04c_prep_15_warn_1080p.png",
            out_720p=mob_dir / "04c_prep_15_warn_720p.png",
            out_360p=mob_dir / "04c_prep_15_warn_360p.png",
        ),
    ]
    boundary_views = build_prep_boundary_views(beats)
    return states_out, en_vote_state, extra_prep_renders, boundary_views


def to_repo_posix(path_str: str) -> str:
    if not path_str:
        return ""
    return Path(path_str).resolve().relative_to(REPO_ROOT).as_posix()


def run_isolated_unity_showcase(only_prep_affected: bool = False) -> Dict[str, Any]:
    if not UNITY_EXE.exists():
        raise FileNotFoundError(f"Unity 6000.6.7f1 executable not found at {UNITY_EXE}")

    ISOLATED_OUT_DIR.mkdir(parents=True, exist_ok=True)
    QA_OUT_DIR.mkdir(parents=True, exist_ok=True)
    (ART009_DIR / "qa").mkdir(parents=True, exist_ok=True)

    # Remove mislabeled legacy d3d11 filenames so only accurate d3d12 outputs remain
    for old_png in ISOLATED_OUT_DIR.glob("unity_d3d11_*.png"):
        old_png.unlink()

    unified = json.loads((ART008_DIR / "unified_asset_manifest.json").read_text(encoding="utf-8"))
    ui_comp = json.loads((ART006_DIR / "ui_component_manifest.json").read_text(encoding="utf-8"))
    ui_borders: Dict[str, List[int]] = {}
    for sk in ui_comp.get("components", ui_comp.get("skins", [])):
        ui_borders[sk["id"]] = sk["unity_sprite_border_lbrt"]

    valid_stems = {
        Path(a["asset_path"]).stem
        for a in unified["assets"]
        if a["asset_path"].endswith(".png")
    }
    verify_csharp_sprite_stems(valid_stems)

    states_12, en_vote_state, extra_prep_renders, boundary_views = build_12_canonical_unity_states(
        only_prep_affected=only_prep_affected
    )

    out_verif = QA_OUT_DIR / "unity_isolated_verification.json"
    out_perf = QA_OUT_DIR / "performance_report.json"
    out_scene_sum = ISOLATED_OUT_DIR / "scene_hierarchy_manifest.json"
    out_png_assault = ISOLATED_OUT_DIR / "unity_d3d12_assault_7classes_1080p.png"
    out_png_boss = ISOLATED_OUT_DIR / "unity_d3d12_boss_slam_finale_1080p.png"
    out_png_reconn = ISOLATED_OUT_DIR / "unity_d3d12_reconnect_inplace_hud_1080p.png"
    log_copy = ISOLATED_OUT_DIR / "unity_batchmode_execution_excerpt.log"

    with tempfile.TemporaryDirectory(prefix="ttg_art009_u6_", ignore_cleanup_errors=True) as tmp_str:
        proj_dir = Path(tmp_str) / "ART008_Isolated_Unity6"
        (proj_dir / "Packages").mkdir(parents=True, exist_ok=True)
        (proj_dir / "ProjectSettings").mkdir(parents=True, exist_ok=True)
        shutil.copy2(REPO_ROOT / "unity" / "Packages" / "manifest.json", proj_dir / "Packages" / "manifest.json")
        lock_src = REPO_ROOT / "unity" / "Packages" / "packages-lock.json"
        if lock_src.exists():
            shutil.copy2(lock_src, proj_dir / "Packages" / "packages-lock.json")
        shutil.copy2(
            REPO_ROOT / "unity" / "ProjectSettings" / "ProjectVersion.txt",
            proj_dir / "ProjectSettings" / "ProjectVersion.txt",
        )

        rt_dir = proj_dir / "Assets" / "_TTGQLDL_Showcase" / "Runtime"
        ed_dir = proj_dir / "Assets" / "Editor"
        rt_dir.mkdir(parents=True, exist_ok=True)
        ed_dir.mkdir(parents=True, exist_ok=True)

        for cs_name in ("BroadcastAudioDispatcher.cs", "SpriteTransitionController.cs", "HudInPlacePresenter.cs"):
            shutil.copy2(SCENE_SRC_DIR / cs_name, rt_dir / cs_name)
        shutil.copy2(SCENE_SRC_DIR / "ART008UnityShowcaseVerifier.cs", ed_dir / "ART008UnityShowcaseVerifier.cs")

        sprite_rules: List[Dict[str, Any]] = []
        for entry in unified["assets"]:
            src_rel = entry["asset_path"]
            src_abs = REPO_ROOT / src_rel
            pkg = entry["package_id"]
            if src_rel.endswith(".png"):
                sub = "UI" if pkg == "ART-006" else ("Environment" if pkg == "ART-005" else "Sprites")
                unity_rel = f"Assets/_TTGQLDL_Showcase/{sub}/{Path(src_rel).name}"
                dst_abs = proj_dir / unity_rel
                dst_abs.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(src_abs, dst_abs)

                piv = entry.get("pivot_normalized") or [0.5, 0.5]
                stem = Path(src_rel).stem
                b_lbrt = ui_borders.get(stem, [0, 0, 0, 0])
                is_ui = pkg == "ART-006"
                sprite_rules.append({
                    "assetPath": unity_rel,
                    "ppu": 100 if is_ui else 192,
                    "pivotX": float(piv[0]),
                    "pivotY": float(piv[1]),
                    "pointFilter": not is_ui,
                    "borderL": int(b_lbrt[0]),
                    "borderB": int(b_lbrt[1]),
                    "borderR": int(b_lbrt[2]),
                    "borderT": int(b_lbrt[3]),
                })
            elif src_rel.endswith(".wav"):
                atype = entry["asset_type"]
                cat = "SFX" if atype == "audio_sfx" else ("Ambient" if atype == "audio_ambient_loop" else "Music")
                unity_rel = f"Assets/_TTGQLDL_Showcase/Audio/{cat}/{Path(src_rel).name}"
                dst_abs = proj_dir / unity_rel
                dst_abs.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(src_abs, dst_abs)

        cfg_payload = {
            "spriteRules": sprite_rules,
            "verificationStates": states_12,
            "englishVoteProofState": en_vote_state,
            "extraPrepWarningRenderStates": extra_prep_renders,
            "prepCountdownBoundaryStates": boundary_views,
            "skipPerformanceBenchmark": bool(only_prep_affected),
            "skipUnchangedIsolatedRenders": bool(only_prep_affected),
            "outputVerificationJson": str(out_verif),
            "outputPerformanceJson": str(out_perf),
            "outputSceneSummaryJson": str(out_scene_sum),
            "outputRenderAssaultPng": str(out_png_assault),
            "outputRenderBossPng": str(out_png_boss),
            "outputRenderReconnectPng": str(out_png_reconn),
        }
        cfg_file = Path(tmp_str) / "art008_showcase_config.json"
        cfg_file.write_text(json.dumps(cfg_payload, indent=2), encoding="utf-8")

        unity_log = Path(tmp_str) / "unity_art008.log"
        env = os.environ.copy()
        env["ART008_UNITY_CONFIG_JSON"] = str(cfg_file)

        # IMPORTANT: Do NOT pass -nographics so hardware GPU initializes and renders genuine 1080p/720p/360p frames!
        cmd = [
            str(UNITY_EXE),
            "-batchmode",
            "-quit",
            "-projectPath",
            str(proj_dir),
            "-executeMethod",
            "TTGQLDL.Showcase.EditorTools.ART008UnityShowcaseVerifier.RunShowcaseBuildAndVerification",
            "-logFile",
            str(unity_log),
        ]
        print(f"[ART-008/009] Launching isolated Unity 6000.6.7f1 hardware GPU showcase ({len(sprite_rules)} sprites, 15 WAVs, 12x3 HUD renders)...")
        proc = subprocess.run(cmd, env=env, capture_output=True, text=True)

        if unity_log.exists():
            lines = unity_log.read_text(encoding="utf-8", errors="replace").splitlines()
            excerpt = []
            for ln in lines:
                if any(k in ln for k in ("Unity Version", "GfxDevice", "Direct3D", "Compiler", "error CS", "Exception", "ART008", "Exiting batchmode")):
                    clean_ln = ln.replace(tmp_str, "<TEMP>").replace(tmp_str.replace("\\", "/"), "<TEMP>")
                    clean_ln = re.sub(r"[A-Za-z]:[\\/][Uu]sers[\\/][^\\/\s]+", "<USER_HOME>", clean_ln)
                    excerpt.append(clean_ln)
            log_copy.write_text("\n".join(excerpt[:220]) + "\n", encoding="utf-8", newline="\n")

        if proc.returncode != 0:
            tail = "\n".join(lines[-80:]) if unity_log.exists() else proc.stderr
            raise RuntimeError(f"Unity exited with code {proc.returncode}:\n{tail}")

        gen_scene = proj_dir / "Assets" / "_TTGQLDL_Showcase" / "Scenes" / "BastionDawn_ART008_VerticalSlice.unity"
        gen_prefab = proj_dir / "Assets" / "_TTGQLDL_Showcase" / "Prefabs" / "BroadcastAudioDispatcher.prefab"
        if gen_scene.exists():
            shutil.copy2(gen_scene, ISOLATED_OUT_DIR / "BastionDawn_ART008_VerticalSlice.unity")
        if gen_prefab.exists():
            shutil.copy2(gen_prefab, ISOLATED_OUT_DIR / "BroadcastAudioDispatcher.prefab")

    # Normalize all output paths to repo-relative POSIX strings (zero host-user path leaks)
    verif_doc = json.loads(out_verif.read_text(encoding="utf-8"))
    verif_doc["renderedGpuFrames"] = [to_repo_posix(p) for p in verif_doc.get("renderedGpuFrames", [])]
    verif_doc["regeneratedInHotfixGpuFrames"] = [to_repo_posix(p) for p in verif_doc.get("regeneratedInHotfixGpuFrames", [])]
    for st in verif_doc.get("hud_verification_states", []):
        st["render_1080p"] = to_repo_posix(st.get("render_1080p", ""))
        st["render_720p"] = to_repo_posix(st.get("render_720p", ""))
        st["render_360p"] = to_repo_posix(st.get("render_360p", ""))
    for pst in verif_doc.get("prep_warning_gpu_renders", []):
        pst["render_1080p"] = to_repo_posix(pst.get("render_1080p", ""))
        pst["render_720p"] = to_repo_posix(pst.get("render_720p", ""))
        pst["render_360p"] = to_repo_posix(pst.get("render_360p", ""))
    if isinstance(verif_doc.get("english_vote_b_proof"), dict) and verif_doc["english_vote_b_proof"].get("state_id"):
        ep = verif_doc["english_vote_b_proof"]
        ep["render_1080p"] = to_repo_posix(ep.get("render_1080p", ""))
        ep["render_720p"] = to_repo_posix(ep.get("render_720p", ""))
        ep["render_360p"] = to_repo_posix(ep.get("render_360p", ""))

    out_verif.write_text(json.dumps(verif_doc, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n")
    (ART009_DIR / "qa" / "unity_isolated_verification.json").write_text(
        json.dumps(verif_doc, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n"
    )

    perf_doc = json.loads(out_perf.read_text(encoding="utf-8"))
    if not only_prep_affected:
        out_perf.write_text(json.dumps(perf_doc, indent=2, ensure_ascii=False) + "\n", encoding="utf-8", newline="\n")

    print(
        f"[UNITY SHOWCASE {verif_doc['status']}] "
        f"GPU={verif_doc['graphicsDeviceName']} ({verif_doc['graphicsDeviceType']}) | "
        f"Sprites={verif_doc['importedSpriteCount']} | Clips={verif_doc['importedAudioClipCount']} | "
        f"HUD12={len(verif_doc['hud_verification_states'])} | "
        f"PrepTicks={len(verif_doc.get('prep_countdown_readbacks', []))} | "
        f"Regenerated={len(verif_doc.get('regeneratedInHotfixGpuFrames', []))}"
    )
    return {"verification": verif_doc, "performance": perf_doc}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Run isolated Unity 6000.6.7f1 showcase & HUD verifier.")
    parser.add_argument(
        "--only-prep-affected",
        action="store_true",
        help="Re-render only PREP-warning affected GPU frames and preserve existing performance_report.json.",
    )
    args = parser.parse_args()
    run_isolated_unity_showcase(only_prep_affected=args.only_prep_affected)
