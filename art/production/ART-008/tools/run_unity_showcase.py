#!/usr/bin/env python3
"""
ART-008 — Isolated Unity 6000.6.7f1 Vertical-Slice Showcase & Performance Runner.

Executes inside an isolated temporary Unity 6000.6.7f1 project outside `unity/`
(preserving M06 = NOT_STARTED), using real Direct3D11 GPU rendering (NO -nographics)
to import all 229 canonical sprites + 15 canonical WAVs, verify all 7 character
classes & transitions, verify BroadcastAudioDispatcher, capture 3 real 1080x1920
Direct3D11 camera frames, and benchmark 10 / 50 / 100 / 150 pooled entities.
"""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any, Dict, List


REPO_ROOT = Path(__file__).resolve().parents[4]
ART006_DIR = REPO_ROOT / "art" / "production" / "ART-006"
ART008_DIR = REPO_ROOT / "art" / "production" / "ART-008"
SCENE_SRC_DIR = ART008_DIR / "showcase" / "scene_source"
ISOLATED_OUT_DIR = ART008_DIR / "showcase" / "isolated_unity"
QA_OUT_DIR = ART008_DIR / "qa"

UNITY_EXE = Path(r"C:\Program Files\Unity\Hub\Editor\6000.6.7f1\Editor\Unity.exe")


def verify_csharp_sprite_stems(valid_stems: set[str]) -> None:
    cs_text = (SCENE_SRC_DIR / "ART008UnityShowcaseVerifier.cs").read_text(encoding="utf-8")
    direct_refs = set(re.findall(r'"((?:env_|trans_|ui_comp_|vfx_boss_slam_telegraph_f05)[a-z0-9_]*)"', cs_text))
    missing = sorted(r for r in direct_refs if r not in valid_stems)
    if missing:
        raise RuntimeError(f"C# verifier references missing sprite stems: {missing}")


def run_isolated_unity_showcase() -> Dict[str, Any]:
    if not UNITY_EXE.exists():
        raise FileNotFoundError(f"Unity 6000.6.7f1 executable not found at {UNITY_EXE}")

    ISOLATED_OUT_DIR.mkdir(parents=True, exist_ok=True)
    QA_OUT_DIR.mkdir(parents=True, exist_ok=True)

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

    out_verif = QA_OUT_DIR / "unity_isolated_verification.json"
    out_perf = QA_OUT_DIR / "performance_report.json"
    out_scene_sum = ISOLATED_OUT_DIR / "scene_hierarchy_manifest.json"
    out_png_assault = ISOLATED_OUT_DIR / "unity_d3d11_assault_7classes_1080p.png"
    out_png_boss = ISOLATED_OUT_DIR / "unity_d3d11_boss_slam_finale_1080p.png"
    out_png_reconn = ISOLATED_OUT_DIR / "unity_d3d11_reconnect_inplace_hud_1080p.png"
    log_copy = ISOLATED_OUT_DIR / "unity_batchmode_execution_excerpt.log"

    with tempfile.TemporaryDirectory(prefix="ttg_art008_u6_", ignore_cleanup_errors=True) as tmp_str:
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

        # Stage C# runtime scripts and Editor verifier
        rt_dir = proj_dir / "Assets" / "_TTGQLDL_Showcase" / "Runtime"
        ed_dir = proj_dir / "Assets" / "Editor"
        rt_dir.mkdir(parents=True, exist_ok=True)
        ed_dir.mkdir(parents=True, exist_ok=True)

        for cs_name in ("BroadcastAudioDispatcher.cs", "SpriteTransitionController.cs", "HudInPlacePresenter.cs"):
            shutil.copy2(SCENE_SRC_DIR / cs_name, rt_dir / cs_name)
        shutil.copy2(SCENE_SRC_DIR / "ART008UnityShowcaseVerifier.cs", ed_dir / "ART008UnityShowcaseVerifier.cs")

        # Stage all 229 PNG sprites & 15 WAV masters into isolated Assets tree
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

        # IMPORTANT: Do NOT pass -nographics so Direct3D11 initializes and renders genuine 1080x1920 GPU frames!
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
        print(f"[ART-008] Launching isolated Unity 6000.6.7f1 Direct3D11 showcase ({len(sprite_rules)} sprites + 15 WAVs)...")
        proc = subprocess.run(cmd, env=env, capture_output=True, text=True)

        if unity_log.exists():
            lines = unity_log.read_text(encoding="utf-8", errors="replace").splitlines()
            excerpt = []
            for ln in lines:
                if any(k in ln for k in ("Unity Version", "GfxDevice", "Direct3D", "Compiler", "error CS", "Exception", "ART008", "Exiting batchmode")):
                    clean_ln = ln.replace(tmp_str, "<TEMP>").replace(tmp_str.replace("\\", "/"), "<TEMP>")
                    clean_ln = re.sub(r"[A-Za-z]:[\\/][Uu]sers[\\/][^\\/\s]+", "<USER_HOME>", clean_ln)
                    excerpt.append(clean_ln)
            log_copy.write_text("\n".join(excerpt[:220]) + "\n", encoding="utf-8")

        if proc.returncode != 0:
            tail = "\n".join(lines[-80:]) if unity_log.exists() else proc.stderr
            raise RuntimeError(f"Unity exited with code {proc.returncode}:\n{tail}")

        # Copy generated .unity scene YAML & BroadcastAudioDispatcher.prefab for handoff inspection
        gen_scene = proj_dir / "Assets" / "_TTGQLDL_Showcase" / "Scenes" / "BastionDawn_ART008_VerticalSlice.unity"
        gen_prefab = proj_dir / "Assets" / "_TTGQLDL_Showcase" / "Prefabs" / "BroadcastAudioDispatcher.prefab"
        if gen_scene.exists():
            shutil.copy2(gen_scene, ISOLATED_OUT_DIR / "BastionDawn_ART008_VerticalSlice.unity")
        if gen_prefab.exists():
            shutil.copy2(gen_prefab, ISOLATED_OUT_DIR / "BroadcastAudioDispatcher.prefab")

    # Normalize renderedGpuFrames paths to repo-relative POSIX strings so committed JSON contains zero temp/host-user paths
    verif_doc = json.loads(out_verif.read_text(encoding="utf-8"))
    verif_doc["renderedGpuFrames"] = [
        Path(p).resolve().relative_to(REPO_ROOT).as_posix()
        for p in verif_doc.get("renderedGpuFrames", [])
    ]
    out_verif.write_text(json.dumps(verif_doc, indent=2) + "\n", encoding="utf-8")

    perf_doc = json.loads(out_perf.read_text(encoding="utf-8"))
    out_perf.write_text(json.dumps(perf_doc, indent=2) + "\n", encoding="utf-8")

    print(
        f"[UNITY SHOWCASE {verif_doc['status']}] "
        f"GPU={verif_doc['graphicsDeviceName']} ({verif_doc['graphicsDeviceType']}) | "
        f"Sprites={verif_doc['importedSpriteCount']} | Clips={verif_doc['importedAudioClipCount']} | "
        f"Classes={len(verif_doc['classMeasurements'])} | LoadTiers={len(perf_doc['loadTiers'])}"
    )
    return {"verification": verif_doc, "performance": perf_doc}


if __name__ == "__main__":
    run_isolated_unity_showcase()
