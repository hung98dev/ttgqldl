#!/usr/bin/env python3
"""
ART-009 — Canonical Gameplay Fixture, HUD Truthfulness & Performance Metric Validator.

Enforces `ttg-rules-v1` (docs/plan/02_GAME_RULES.md, 03_SIMULATION.md, 04_DATABASE.md,
05_API_WEBSOCKET.md, 07_UNITY_ART_UX.md, 09_QA_ACCEPTANCE.md and tooling/refeval/model.go)
across all ART-008 showcase fixtures, timeline metadata, Unity isolated HUD verification
states, C# presenters, and performance telemetry.

Pure Python 3 standard library — safe for fast Linux CI (`--ci-fast`) and local runs.
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import math
import re
import sys
from pathlib import Path
from typing import Any, Callable, Dict, List, Optional, Tuple

TOOLS_DIR = Path(__file__).resolve().parent
ART009_DIR = TOOLS_DIR.parent
REPO_ROOT = ART009_DIR.parents[2]
ART008_DIR = REPO_ROOT / "art" / "production" / "ART-008"
ART007_DIR = REPO_ROOT / "art" / "production" / "ART-007"
ART006_DIR = REPO_ROOT / "art" / "production" / "ART-006"
GOLDEN_VECTORS_PATH = REPO_ROOT / "tooling" / "refeval" / "testdata" / "golden_vectors.json"

if str(ART008_DIR / "tools") not in sys.path:
    sys.path.insert(0, str(ART008_DIR / "tools"))

from showcase_contracts import (  # noqa: E402
    compute_prep_seconds_until_assault,
    format_prep_warning_banner,
    make_snapshot,
    resolve_authoritative_hud_banner,
)


# ============================================================================
# Canonical `ttg-rules-v1` Integer Reference Formulas (matching model.go)
# ============================================================================

def canonical_wall_max(wall_tier: int) -> int:
    return 1000 + 200 * (int(wall_tier) - 1)


def canonical_phase_for_tick(lifecycle: str, wave_tick: int) -> str:
    if lifecycle == "waiting" or wave_tick == -1:
        return "waiting"
    if 0 <= wave_tick < 1600:
        return "prep"
    if 1600 <= wave_tick < 2600:
        return "assault"
    if 2600 <= wave_tick < 2800:
        return "finale"
    if 2800 <= wave_tick < 3000:
        return "results"
    return "invalid_tick"


def canonical_difficulty(consecutive_wins: int) -> int:
    return min(10, 1 + max(0, int(consecutive_wins)))


def canonical_base_hp(population_anchor: int, consecutive_wins: int, consecutive_losses: int) -> int:
    p_eff = min(max(1, int(population_anchor)), 10000)
    d = canonical_difficulty(consecutive_wins)
    raw_hp = 40 + 8 * d + 38 * math.isqrt(p_eff)
    loss_mod = 100 - 15 * min(max(0, int(consecutive_losses)), 3)
    return max(1, (raw_hp * loss_mod) // 100)


def canonical_lane_split(base_hp: int) -> Tuple[int, int, int]:
    b = int(base_hp)
    west = (b * 34) // 100
    center = (b * 32) // 100
    east = b - west - center
    return west, center, east


def canonical_subpool_split(lane_hp: int) -> Tuple[int, int, int]:
    lh = int(lane_hp)
    runner = (lh * 25) // 100
    grunt = (lh * 55) // 100
    brute = lh - runner - grunt
    return runner, grunt, brute


def canonical_boss_max_hp(wave_no: int, base_hp: int) -> int:
    if int(wave_no) % 5 != 0 or int(base_hp) <= 0:
        return 0
    return (int(base_hp) + 4) // 5


def canonical_militia_bonus(unique_joined: int) -> int:
    p = int(unique_joined)
    if 1 <= p <= 10:
        return max(0, 40 - 4 * (p - 1))
    return 0


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_json_lf(path: Path, payload: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    text = json.dumps(payload, indent=2, ensure_ascii=False) + "\n"
    path.write_bytes(text.encode("utf-8"))


# ============================================================================
# Core Bundle Loader & 25-Invariant Canonical Evaluator
# ============================================================================

def load_bundle_from_repo() -> Dict[str, Any]:
    fixtures_dir = ART008_DIR / "showcase" / "fixtures"
    full_wave_path = fixtures_dir / "full_wave_300s_wave10_showcase.json"
    full_wave = json.loads(full_wave_path.read_text(encoding="utf-8"))

    failure_fixtures: Dict[str, Any] = {}
    for fp in sorted(fixtures_dir.glob("failure_*.json")):
        failure_fixtures[fp.stem] = json.loads(fp.read_text(encoding="utf-8"))

    timeline_path = ART008_DIR / "presentation_timeline.json"
    timeline = json.loads(timeline_path.read_text(encoding="utf-8"))

    unity_ver_path = ART008_DIR / "qa" / "unity_isolated_verification.json"
    unity_ver = json.loads(unity_ver_path.read_text(encoding="utf-8"))

    perf_path = ART008_DIR / "qa" / "performance_report.json"
    perf = json.loads(perf_path.read_text(encoding="utf-8"))

    audio_policy_path = ART008_DIR / "audio_dispatch_policy.json"
    audio_policy = json.loads(audio_policy_path.read_text(encoding="utf-8"))

    rights_path = ART008_DIR / "qa" / "rights_audit.json"
    rights = json.loads(rights_path.read_text(encoding="utf-8"))

    readme_text = (ART008_DIR / "README.md").read_text(encoding="utf-8")
    qa_report_text = (ART008_DIR / "qa" / "qa_report.md").read_text(encoding="utf-8")
    cs_verifier_text = (
        ART008_DIR / "showcase" / "scene_source" / "ART008UnityShowcaseVerifier.cs"
    ).read_text(encoding="utf-8")
    cs_hud_text = (
        ART008_DIR / "showcase" / "scene_source" / "HudInPlacePresenter.cs"
    ).read_text(encoding="utf-8")

    vi_strings = json.loads((ART006_DIR / "localization_vi.json").read_text(encoding="utf-8"))["strings"]
    en_strings = json.loads((ART006_DIR / "localization_en.json").read_text(encoding="utf-8"))["strings"]
    art007_manifest = json.loads((ART007_DIR / "asset_manifest.json").read_text(encoding="utf-8"))
    art009_handoff_doc = (ART009_DIR / "UNITY_HANDOFF_READINESS.md").read_text(encoding="utf-8")
    art008_event_contract_doc = (ART008_DIR / "PRESENTATION_EVENT_CONTRACT.md").read_text(encoding="utf-8")
    art008_direction_doc = (ART008_DIR / "AUDIOVISUAL_DIRECTION.md").read_text(encoding="utf-8")

    golden_cases: List[Dict[str, Any]] = []
    if GOLDEN_VECTORS_PATH.exists():
        golden_cases = json.loads(GOLDEN_VECTORS_PATH.read_text(encoding="utf-8")).get("cases", [])

    return {
        "full_wave": full_wave,
        "full_wave_sha256": sha256_file(full_wave_path),
        "failure_fixtures": failure_fixtures,
        "timeline": timeline,
        "unity_verification": unity_ver,
        "performance_report": perf,
        "audio_policy": audio_policy,
        "rights_audit": rights,
        "art008_readme": readme_text,
        "art008_qa_report": qa_report_text,
        "cs_verifier_text": cs_verifier_text,
        "cs_hud_text": cs_hud_text,
        "vi_strings": vi_strings,
        "en_strings": en_strings,
        "art007_manifest": art007_manifest,
        "art009_handoff_doc": art009_handoff_doc,
        "art008_event_contract_doc": art008_event_contract_doc,
        "art008_direction_doc": art008_direction_doc,
        "golden_cases": golden_cases,
    }


def collect_all_snapshots(bundle: Dict[str, Any]) -> List[Tuple[str, Dict[str, Any], Optional[Dict[str, Any]]]]:
    """Returns list of (context_id, public_snapshot, parent_beat_or_step)."""
    items: List[Tuple[str, Dict[str, Any], Optional[Dict[str, Any]]]] = []
    fw = bundle["full_wave"]
    for beat in fw.get("beats", []):
        bid = f"full_wave.{beat.get('beat_id', 'unknown')}"
        snap = beat.get("fixture", {}).get("public_snapshot")
        if isinstance(snap, dict):
            items.append((bid, snap, beat))
    extra_defeat = fw.get("supplementary_states", {})
    if isinstance(extra_defeat, dict):
        for sid, sdata in sorted(extra_defeat.items()):
            snap = sdata.get("fixture", {}).get("public_snapshot") if isinstance(sdata, dict) else None
            if isinstance(snap, dict):
                items.append((f"supplementary.{sid}", snap, sdata))
    for fname, fdoc in sorted(bundle["failure_fixtures"].items()):
        snap_main = fdoc.get("fixture", {}).get("public_snapshot")
        if isinstance(snap_main, dict):
            items.append((f"{fname}.fixture", snap_main, fdoc))
        stale_step = fdoc.get("stale_snapshot_rejected_step")
        if isinstance(stale_step, dict):
            snap_stale = stale_step.get("fixture", {}).get("public_snapshot") or stale_step.get("public_snapshot")
            if isinstance(snap_stale, dict):
                items.append((f"{fname}.stale_snapshot_rejected_step", snap_stale, stale_step))
        for step in fdoc.get("steps", []):
            sid = f"{fname}.{step.get('step_id', 'step')}"
            snap = step.get("fixture", {}).get("public_snapshot")
            if isinstance(snap, dict):
                items.append((sid, snap, step))
    return items


def evaluate_bundle(bundle: Dict[str, Any]) -> Dict[str, Any]:
    """
    Evaluates all 25 canonical `ttg-rules-v1` & truthfulness invariants on a loaded bundle.
    Returns structured per-rule results and overall pass status.
    """
    snaps = collect_all_snapshots(bundle)
    fw = bundle["full_wave"]
    fw_beats = fw.get("beats", [])
    unity_ver = bundle["unity_verification"]
    perf = bundle["performance_report"]
    timeline = bundle["timeline"]
    cs_ver = bundle["cs_verifier_text"]
    cs_hud = bundle["cs_hud_text"]
    readme_txt = bundle["art008_readme"]
    vi_s = bundle["vi_strings"]
    en_s = bundle["en_strings"]

    rules: List[Dict[str, Any]] = []

    def record(rule_id: str, title: str, failures: List[str], evidence: Optional[Dict[str, Any]] = None) -> None:
        rules.append({
            "rule_id": rule_id,
            "title": title,
            "passed": len(failures) == 0,
            "failure_count": len(failures),
            "failures": failures[:12],
            "evidence": evidence or {},
        })

    # Verify our reference helper equations against M02 golden vectors read-only first
    golden_mismatches: List[str] = []
    for gc in bundle.get("golden_cases", []):
        gid = gc["id"]
        inp = gc["inputs"]
        out = gc["outcome"]
        exp_wall_max = canonical_wall_max(out["wall_tier"])
        if out["wall_max"] != exp_wall_max:
            golden_mismatches.append(f"{gid}: golden wall_max={out['wall_max']} != formula {exp_wall_max}")
        exp_base = canonical_base_hp(out["anchor_population"], inp["consecutive_wins"], inp["consecutive_losses"])
        if out["base_hp"] != exp_base:
            golden_mismatches.append(f"{gid}: golden base_hp={out['base_hp']} != formula {exp_base}")

    # 1. R01_TOWNHALL_HP_BOUNDS: 0 <= townhall_hp <= 500
    f_r01: List[str] = []
    for cid, s, _ in snaps:
        th = s.get("townhall_hp")
        if not isinstance(th, int) or not (0 <= th <= 500):
            f_r01.append(f"{cid}: townhall_hp={th} violates 0 <= townhall_hp <= 500")
    if re.search(r"1850\s*/\s*2000", cs_ver):
        f_r01.append("ART008UnityShowcaseVerifier.cs hardcodes invalid '1850/2000' Townhall HP")
    record("R01_TOWNHALL_HP_BOUNDS", "Townhall current HP is always within 0..500", f_r01)

    # 2. R02_TOWNHALL_MAX_500: townhall_max == 500 everywhere
    f_r02: List[str] = []
    for cid, s, _ in snaps:
        tmax = s.get("townhall_max")
        if tmax != 500:
            f_r02.append(f"{cid}: townhall_max={tmax} != canonical 500")
    for st in unity_ver.get("hud_verification_states", []):
        chp = str(st.get("castle_hp_text", ""))
        if not chp.endswith("/500"):
            f_r02.append(f"unity state {st.get('state_id')}: castle_hp_text='{chp}' does not end with '/500'")
    if len(unity_ver.get("hud_verification_states", [])) == 0:
        f_r02.append("unity_isolated_verification.json missing hud_verification_states proof")
    record("R02_TOWNHALL_MAX_500", "Townhall maximum HP is always 500 across fixtures and Unity HUD", f_r02)

    # 3. R03_WALL_MAX_FROM_TIER: wall_max == 1000 + 200 * (wall_tier - 1)
    f_r03: List[str] = list(golden_mismatches)
    for cid, s, _ in snaps:
        wt = s.get("wall_tier")
        wm = s.get("wall_max")
        if not isinstance(wt, int) or not isinstance(wm, int):
            f_r03.append(f"{cid}: non-integer wall_tier={wt} or wall_max={wm}")
            continue
        expected_wm = canonical_wall_max(wt)
        if wm != expected_wm:
            f_r03.append(f"{cid}: wall_tier={wt} has wall_max={wm}, expected {expected_wm}")
    record("R03_WALL_MAX_FROM_TIER", "wall_max strictly equals 1000 + 200 * (wall_tier - 1)", f_r03)

    # 4. R04_WALL_HP_BOUNDS: 0 <= wall_hp <= wall_max
    f_r04: List[str] = []
    for cid, s, _ in snaps:
        wh = s.get("wall_hp")
        wm = s.get("wall_max")
        if not isinstance(wh, int) or not isinstance(wm, int) or not (0 <= wh <= wm):
            f_r04.append(f"{cid}: wall_hp={wh} out of bounds 0..{wm}")
    record("R04_WALL_HP_BOUNDS", "Wall current HP is always within 0..wall_max", f_r04)

    # 5. R05_VILLAGE_LEVEL_BOUNDS: 1 <= village_level <= 50
    f_r05: List[str] = []
    for cid, s, _ in snaps:
        vl = s.get("village_level")
        if not isinstance(vl, int) or not (1 <= vl <= 50):
            f_r05.append(f"{cid}: village_level={vl} outside 1..50")
        tw = s.get("total_wins")
        if isinstance(tw, int):
            exp_vl = min(50, 1 + tw // 3)
            if vl != exp_vl:
                f_r05.append(f"{cid}: village_level={vl} != min(50, 1 + {tw}//3)={exp_vl}")
    record("R05_VILLAGE_LEVEL_BOUNDS", "Village level is within 1..50 and matches total_wins formula", f_r05)

    # 6. R06_WALL_TIER_BOUNDS: 1 <= wall_tier <= 20
    f_r06: List[str] = []
    for cid, s, _ in snaps:
        wt = s.get("wall_tier")
        if not isinstance(wt, int) or not (1 <= wt <= 20):
            f_r06.append(f"{cid}: wall_tier={wt} outside 1..20")
    record("R06_WALL_TIER_BOUNDS", "Wall tier is always within 1..20", f_r06)

    # 7. R07_RESOURCE_BOUNDS: 0 <= wood, stone, food <= 1,000,000
    f_r07: List[str] = []
    for cid, s, _ in snaps:
        for rk in ("wood", "stone", "food"):
            rv = s.get(rk)
            if not isinstance(rv, int) or not (0 <= rv <= 1_000_000):
                f_r07.append(f"{cid}: {rk}={rv} outside 0..1_000_000")
    record("R07_RESOURCE_BOUNDS", "Resources (wood, stone, food) remain within 0..1,000,000", f_r07)

    # 8. R08_WAITING_TICK_MINUS_ONE: WAITING always uses wave_tick == -1 and P == 0
    f_r08: List[str] = []
    for cid, s, beat in snaps:
        life = s.get("lifecycle")
        ph = s.get("phase")
        wtick = s.get("wave_tick")
        if life == "waiting" or ph == "waiting":
            if wtick != -1:
                f_r08.append(f"{cid}: WAITING snapshot has wave_tick={wtick} instead of -1")
            if beat and isinstance(beat, dict) and "wave_tick" in beat and beat["wave_tick"] != -1:
                f_r08.append(f"{cid}: WAITING beat metadata has wave_tick={beat['wave_tick']} instead of -1")
            cnts = s.get("counts", {})
            if cnts.get("unique_joined") != 0 or cnts.get("builder") != 0 or cnts.get("guard") != 0 or cnts.get("fighter") != 0:
                f_r08.append(f"{cid}: WAITING snapshot must have P=0 and 0 role counts, got {cnts}")
    record("R08_WAITING_TICK_MINUS_ONE", "WAITING state consistently uses wave_tick = -1 and P = 0", f_r08)

    # 9. R09_PHASE_AT_EVERY_TICK: phase strictly matches canonical tick boundaries
    f_r09: List[str] = []
    for cid, s, beat in snaps:
        life = str(s.get("lifecycle", ""))
        wtick = s.get("wave_tick")
        ph = s.get("phase")
        if not isinstance(wtick, int):
            f_r09.append(f"{cid}: non-int wave_tick={wtick}")
            continue
        if wtick < -1 or wtick >= 3000:
            f_r09.append(f"{cid}: wave_tick={wtick} outside legal [-1..2999] snapshot domain")
            continue
        if life not in ("stopped",):
            exp_ph = canonical_phase_for_tick("waiting" if life == "waiting" else "running", wtick)
            if ph != exp_ph:
                f_r09.append(f"{cid}: wave_tick={wtick} (lifecycle={life}) has phase='{ph}', expected '{exp_ph}'")
        if beat and isinstance(beat, dict) and "wave_tick" in beat:
            if int(beat["wave_tick"]) != wtick:
                f_r09.append(f"{cid}: beat wave_tick={beat['wave_tick']} disagrees with snapshot wave_tick={wtick}")
    record("R09_PHASE_AT_EVERY_TICK", "Phase derivation matches [-1=waiting, 0..1599=prep, 1600..2599=assault, 2600..2799=finale, 2800..2999=results]", f_r09)

    # 10. R10_EVEN_WAVE_ONLY_VOTE: even waves only, open only in [800..1199]
    f_r10: List[str] = []
    for cid, s, _ in snaps:
        wno = int(s.get("wave_no", 0))
        wtick = int(s.get("wave_tick", 0))
        vote = s.get("vote", {})
        v_open = bool(vote.get("open", False))
        v_sel = vote.get("selected")
        if wno % 2 == 1:
            if v_open or v_sel not in (None, ""):
                f_r10.append(f"{cid}: odd wave {wno} illegally has vote open={v_open}, selected={v_sel}")
        else:
            if v_open and not (800 <= wtick <= 1199):
                f_r10.append(f"{cid}: even wave {wno} has vote.open=True outside [800..1199] at tick={wtick}")
            if wtick < 1200 and v_sel not in (None, ""):
                f_r10.append(f"{cid}: even wave {wno} locked vote.selected={v_sel} before tick 1200 (tick={wtick})")
    record("R10_EVEN_WAVE_ONLY_VOTE", "Voting opens only on even waves during ticks 800..1199 and locks at 1200", f_r10)

    # 11. R11_VOTE_ABC_COSTS_AND_EFFECTS: canonical Vote A/B/C definitions everywhere
    f_r11: List[str] = []
    bad_vote_patterns = [
        r"SỬA\s*TƯỜNG\s*\+25%",
        r"Option\s*B\s*wall\s*reinforcement",
        r"Option\s*B.*wall\s*repair",
        r"B:.*repair",
    ]
    for pat in bad_vote_patterns:
        if re.search(pat, cs_ver, flags=re.IGNORECASE):
            f_r11.append(f"ART008UnityShowcaseVerifier.cs contains forbidden Vote B description matching /{pat}/")
        if re.search(pat, cs_hud, flags=re.IGNORECASE):
            f_r11.append(f"HudInPlacePresenter.cs contains forbidden Vote B description matching /{pat}/")
        for b in fw_beats:
            note = (
                str(b.get("narrative", ""))
                + " "
                + str(b.get("meaningful_visual_change", ""))
                + " "
                + json.dumps(b.get("fixture", {}), ensure_ascii=False)
            )
            if re.search(pat, note, flags=re.IGNORECASE):
                f_r11.append(f"full_wave.{b.get('beat_id')} contains forbidden Vote B text matching /{pat}/")
    if "(-50G)" not in vi_s.get("hud.vote.open", "") or "(-50L)" not in vi_s.get("hud.vote.open", ""):
        f_r11.append("VI localization hud.vote.open missing canonical A(-50G) / B(-50L) costs")
    if "(-50W)" not in en_s.get("hud.vote.open", "") or "(-50F)" not in en_s.get("hud.vote.open", ""):
        f_r11.append("EN localization hud.vote.open missing canonical A(-50W) / B(-50F) costs")
    # Check Vote B stability & canonical effect metadata across Wave 10 beats >= 1200
    selected_seen: Optional[str] = None
    for b in fw_beats:
        s = b.get("fixture", {}).get("public_snapshot", {})
        if int(s.get("wave_no", 0)) == 10 and int(s.get("wave_tick", -1)) >= 1200:
            v = s.get("vote", {})
            sel = v.get("selected")
            if sel not in ("A", "B", "C"):
                f_r11.append(f"full_wave.{b.get('beat_id')}: Wave 10 tick>={s.get('wave_tick')} missing vote.selected")
            elif selected_seen is None:
                selected_seen = sel
            elif sel != selected_seen:
                f_r11.append(f"full_wave.{b.get('beat_id')}: vote.selected changed mid-wave {selected_seen} -> {sel}")
            if sel == "B":
                eff = v.get("canonical_effect", {})
                if eff.get("cost_resource") != "food" or eff.get("cost_amount") != 50 or eff.get("fighter_attack_multiplier_pct") != 115:
                    f_r11.append(f"full_wave.{b.get('beat_id')}: Vote B missing canonical_effect (food=50, fighter_attack_multiplier_pct=115)")
    # Verify Vote B food deduction at tick 1200 in Wave 10 boundary trace
    b_trace = fw.get("boundary_trace", [])
    if not isinstance(b_trace, list) or len(b_trace) == 0:
        f_r11.append("full_wave_300s_wave10_showcase.json missing authoritative boundary_trace")
    else:
        t1200 = [r for r in b_trace if r.get("tick") == 1200]
        if not t1200 or t1200[0].get("vote_cost_charged") != {"resource": "food", "amount": 50}:
            f_r11.append("Wave 10 boundary_trace at tick 1200 does not record one-time 50 Food charge for Vote B")
    record("R11_VOTE_ABC_COSTS_AND_EFFECTS", "Vote A/B/C costs, effects, localization, and one-time tick-1200 deduction match ttg-rules-v1", f_r11)

    # 12. R12_CANONICAL_LANE_SPLIT_AT_SPAWN: 34% / 32% / remainder + runner/grunt/brute subpools
    f_r12: List[str] = []
    # Explicit reference assertion for base_hp = 1420
    if canonical_lane_split(1420) != (482, 454, 484):
        f_r12.append("Internal canonical_lane_split(1420) failed expected (482, 454, 484)")
    spawn_beats = [
        b for b in fw_beats
        if int(b.get("fixture", {}).get("public_snapshot", {}).get("wave_no", 0)) == 10
        and int(b.get("fixture", {}).get("public_snapshot", {}).get("wave_tick", -1)) == 1600
    ]
    if not spawn_beats:
        f_r12.append("Wave 10 fixture missing tick 1600 population-lock & lane-spawn beat")
    for sb in spawn_beats:
        s = sb["fixture"]["public_snapshot"]
        base_hp = s.get("base_hp")
        p_anc = s.get("population_anchor")
        cw = s.get("consecutive_wins_before_wave")
        cl = s.get("settlement", {}).get("consecutive_losses", 0)
        if not isinstance(base_hp, int) or base_hp <= 0:
            f_r12.append(f"tick 1600 beat missing positive integer base_hp (got {base_hp})")
            continue
        if isinstance(p_anc, int) and isinstance(cw, int):
            exp_bhp = canonical_base_hp(p_anc, cw, cl)
            if base_hp != exp_bhp:
                f_r12.append(f"tick 1600 base_hp={base_hp} != canonical_base_hp(P={p_anc}, cw={cw}, cl={cl})={exp_bhp}")
        exp_w, exp_c, exp_e = canonical_lane_split(base_hp)
        lanes = {ln["id"]: ln for ln in s.get("lanes", [])}
        got_tuple = (
            lanes.get("west", {}).get("ordinary_hp"),
            lanes.get("center", {}).get("ordinary_hp"),
            lanes.get("east", {}).get("ordinary_hp"),
        )
        if got_tuple != (exp_w, exp_c, exp_e):
            f_r12.append(f"tick 1600 lanes={got_tuple} != canonical 34%/32%/rem ({exp_w}, {exp_c}, {exp_e}) for base_hp={base_hp}")
        for lid, exp_lhp in (("west", exp_w), ("center", exp_c), ("east", exp_e)):
            ln = lanes.get(lid, {})
            exp_r, exp_g, exp_br = canonical_subpool_split(exp_lhp)
            got_sub = (ln.get("runner_hp"), ln.get("grunt_hp"), ln.get("brute_hp"))
            if got_sub != (exp_r, exp_g, exp_br):
                f_r12.append(f"tick 1600 lane '{lid}' subpools={got_sub} != canonical ({exp_r}, {exp_g}, {exp_br})")
    # Also verify subpool sum on every snapshot that has lanes
    for cid, s, _ in snaps:
        wtick = int(s.get("wave_tick", -1))
        for ln in s.get("lanes", []):
            ohp = ln.get("ordinary_hp", 0)
            if wtick < 1600 and ohp != 0:
                f_r12.append(f"{cid}: pre-1600 tick={wtick} has non-zero lane '{ln.get('id')}' ordinary_hp={ohp}")
            if "runner_hp" in ln or "grunt_hp" in ln or "brute_hp" in ln:
                r_hp = int(ln.get("runner_hp", -1))
                g_hp = int(ln.get("grunt_hp", -1))
                b_hp = int(ln.get("brute_hp", -1))
                if r_hp < 0 or g_hp < 0 or b_hp < 0 or (r_hp + g_hp + b_hp) != ohp:
                    f_r12.append(f"{cid}: lane '{ln.get('id')}' subpools ({r_hp},{g_hp},{b_hp}) do not sum to ordinary_hp={ohp}")
            else:
                f_r12.append(f"{cid}: lane '{ln.get('id')}' missing runner_hp/grunt_hp/brute_hp subpool breakdown")
    record("R12_CANONICAL_LANE_SPLIT_AT_SPAWN", "Initial three-lane HP (34%/32%/rem) and runner/grunt/brute subpools (25%/55%/rem) follow ttg-rules-v1", f_r12)

    # 13. R13_BOSS_MAX_FROM_BASE_HP: boss_max_hp == ceil(base_hp / 5)
    f_r13: List[str] = []
    if canonical_boss_max_hp(10, 1420) != 284:
        f_r13.append("Internal canonical_boss_max_hp(10, 1420) failed expected 284")
    for cid, s, _ in snaps:
        wno = int(s.get("wave_no", 0))
        wtick = int(s.get("wave_tick", -1))
        boss = s.get("boss", {})
        b_hp = int(boss.get("boss_hp", 0))
        b_max = int(boss.get("boss_max_hp", 0))
        base_hp = int(s.get("base_hp", 0))
        if b_hp < 0 or b_max < 0 or b_hp > b_max:
            f_r13.append(f"{cid}: invalid boss_hp={b_hp} / boss_max_hp={b_max}")
        if wno % 5 == 0 and wtick >= 1600:
            if base_hp <= 0:
                f_r13.append(f"{cid}: Boss wave {wno} at tick={wtick} missing base_hp to verify boss_max_hp={b_max}")
            else:
                exp_bmax = canonical_boss_max_hp(wno, base_hp)
                if b_max != exp_bmax:
                    f_r13.append(f"{cid}: boss_max_hp={b_max} != ceil({base_hp}/5)={exp_bmax}")
    record("R13_BOSS_MAX_FROM_BASE_HP", "Boss max HP strictly equals ceil(base_hp / 5) and never uses cinematic 1800 pools", f_r13)

    # 14. R14_BOSS_ABSENT_BEFORE_2600: boss_exists == False and boss_hp == 0 before tick 2600
    f_r14: List[str] = []
    for cid, s, _ in snaps:
        wtick = int(s.get("wave_tick", -1))
        boss = s.get("boss", {})
        if wtick < 2600:
            if bool(boss.get("boss_exists", False)) or int(boss.get("boss_hp", 0)) != 0:
                f_r14.append(f"{cid}: Boss prematurely active before tick 2600 (tick={wtick}, boss={boss})")
    record("R14_BOSS_ABSENT_BEFORE_2600", "Boss does not exist and has 0 authoritative HP prior to tick 2600", f_r14)

    # 15. R15_BOSS_EVERY_FIFTH_WAVE_ONLY: wave_no % 5 != 0 -> zero Boss state
    f_r15: List[str] = []
    for cid, s, _ in snaps:
        wno = int(s.get("wave_no", 0))
        boss = s.get("boss", {})
        if wno % 5 != 0:
            if bool(boss.get("boss_exists")) or bool(boss.get("boss_incoming")) or int(boss.get("boss_hp", 0)) != 0 or int(boss.get("boss_max_hp", 0)) != 0:
                f_r15.append(f"{cid}: non-Boss wave {wno} carries Boss state {boss}")
    record("R15_BOSS_EVERY_FIFTH_WAVE_ONLY", "Boss spawns/incoming occurs exclusively on every fifth wave (wave_no % 5 == 0)", f_r15)

    # 16. R16_NO_PREMATURE_RESULTS: wave_result is None before tick 2800
    f_r16: List[str] = []
    for cid, s, _ in snaps:
        wtick = int(s.get("wave_tick", -1))
        wres = s.get("settlement", {}).get("wave_result")
        if wtick < 2800 and wres is not None:
            f_r16.append(f"{cid}: premature settlement wave_result='{wres}' at tick={wtick} (< 2800)")
        if 2800 <= wtick < 3000 and wres not in ("win", "loss"):
            f_r16.append(f"{cid}: RESULTS snapshot at tick={wtick} missing wave_result in ('win','loss'), got {wres}")
    record("R16_NO_PREMATURE_RESULTS", "Wave settlement result is never emitted before tick 2800 and always present in 2800..2999", f_r16)

    # 17. R17_MONOTONIC_ENEMY_HP_AND_DECORATIVE_UNITS: no enemy HP increases after spawn
    f_r17: List[str] = []
    prev_wave = None
    prev_lanes: Optional[Dict[str, int]] = None
    prev_boss_hp: Optional[int] = None
    for b in fw_beats:
        bid = b.get("beat_id", "beat")
        s = b.get("fixture", {}).get("public_snapshot", {})
        wno = int(s.get("wave_no", 0))
        wtick = int(s.get("wave_tick", -1))
        cur_lanes = {ln["id"]: int(ln["ordinary_hp"]) for ln in s.get("lanes", [])}
        cur_boss = int(s.get("boss", {}).get("boss_hp", 0))
        if wno != prev_wave:
            prev_wave = wno
            prev_lanes = cur_lanes if wtick >= 1600 else None
            prev_boss_hp = cur_boss if wtick >= 2600 else None
            continue
        if wtick >= 1600:
            if prev_lanes is not None:
                for lid in ("west", "center", "east"):
                    if cur_lanes.get(lid, 0) > prev_lanes.get(lid, 0):
                        f_r17.append(f"full_wave.{bid} (tick={wtick}): lane '{lid}' HP increased {prev_lanes[lid]} -> {cur_lanes[lid]}")
            prev_lanes = cur_lanes
        if wtick >= 2600:
            if prev_boss_hp is not None and cur_boss > prev_boss_hp:
                f_r17.append(f"full_wave.{bid} (tick={wtick}): boss_hp increased {prev_boss_hp} -> {cur_boss}")
            prev_boss_hp = cur_boss
    if fw.get("visual_units_authority_mode") != "decorative_aggregate_pool_proxies":
        f_r17.append("full_wave fixture must declare visual_units_authority_mode='decorative_aggregate_pool_proxies'")
    record("R17_MONOTONIC_ENEMY_HP_AND_DECORATIVE_UNITS", "Ordinary and Boss HP decrease monotonically after spawn; visual units are decorative proxies", f_r17)

    # 18. R18_SINGLE_SETTLEMENT_AND_REWARDS: one-time settlement at 2800, exact rewards, no repair in RESULTS
    f_r18: List[str] = []
    settle_events = [
        b for b in fw_beats
        if any(
            ev.get("event_type") in ("wave_result", "wave_victory", "wave_defeat")
            for ev in (b.get("audio_events", []) + b.get("events", []))
        )
    ]
    if len(settle_events) != 1:
        f_r18.append(f"Wave 10 showcase must emit exactly 1 settlement event beat, found {len(settle_events)}")
    # Check no wall_hp or resource changes during RESULTS (2800 < tick < 3000)
    results_snaps = [
        (b.get("beat_id"), b["fixture"]["public_snapshot"])
        for b in fw_beats
        if int(b.get("fixture", {}).get("public_snapshot", {}).get("wave_no", 0)) == 10
        and 2800 <= int(b.get("fixture", {}).get("public_snapshot", {}).get("wave_tick", -1)) < 3000
    ]
    if len(results_snaps) >= 2:
        base_id, s0 = results_snaps[0]
        for bid, s_later in results_snaps[1:]:
            for k in ("wall_hp", "townhall_hp", "wood", "stone", "food", "wall_tier"):
                if s_later.get(k) != s0.get(k):
                    f_r18.append(f"Illegal state mutation during RESULTS between {base_id} and {bid}: {k} {s0.get(k)} -> {s_later.get(k)}")
    # Check victory reward in boundary_trace at tick 2800
    if isinstance(b_trace, list) and len(b_trace) > 0:
        t2800 = [r for r in b_trace if r.get("tick") == 2800]
        if len(t2800) != 1:
            f_r18.append(f"boundary_trace must contain exactly 1 settlement entry at tick 2800, got {len(t2800)}")
        else:
            rew = t2800[0].get("settlement_reward")
            if t2800[0].get("settlement_result") == "win" and rew != {"wood": 100, "stone": 60, "food": 50}:
                f_r18.append(f"tick 2800 win settlement_reward={rew} != canonical +100W/+60S/+50F")
    record("R18_SINGLE_SETTLEMENT_AND_REWARDS", "Settlement executes once at tick 2800 with exact +100W/+60S/+50F rewards and zero RESULTS repair", f_r18)

    # 19. R19_ROLE_COUNTS_AND_MILITIA_MATH: B + G + F == unique_joined & exact militia formula
    f_r19: List[str] = []
    for cid, s, _ in snaps:
        cnts = s.get("counts", {})
        uj = cnts.get("unique_joined")
        b_c, g_c, f_c = cnts.get("builder"), cnts.get("guard"), cnts.get("fighter")
        if not all(isinstance(x, int) and x >= 0 for x in (uj, b_c, g_c, f_c)):
            f_r19.append(f"{cid}: invalid role counts {cnts}")
            continue
        if (b_c + g_c + f_c) != uj:
            f_r19.append(f"{cid}: builder({b_c})+guard({g_c})+fighter({f_c}) != unique_joined({uj})")
        assist = s.get("system_assist", {})
        # Militia is evaluated against population anchor P once locked (tick >= 1600), or current P during prep
        wtick = int(s.get("wave_tick", -1))
        p_ref = int(s.get("population_anchor", uj)) if wtick >= 1600 else uj
        exp_m_active = 1 <= p_ref <= 10
        exp_m_bonus = canonical_militia_bonus(p_ref) if exp_m_active else 0
        if bool(assist.get("militia_active", False)) != exp_m_active:
            f_r19.append(f"{cid}: militia_active={assist.get('militia_active')} != expected {exp_m_active} for P={p_ref}")
        if int(assist.get("militia_attack_bonus", -1)) != exp_m_bonus:
            f_r19.append(f"{cid}: militia_attack_bonus={assist.get('militia_attack_bonus')} != canonical {exp_m_bonus} for P={p_ref}")
    record("R19_ROLE_COUNTS_AND_MILITIA_MATH", "Role counts sum to unique_joined and militia bonus equals max(0, 40 - 4*(P-1)) for 1<=P<=10", f_r19)

    # 20. R20_RESULTS_QUEUED_NEXT_WAVE_ISOLATION: RESULTS joins queue for wave_no + 1 only
    f_r20: List[str] = []
    if len(results_snaps) >= 2:
        _, s_res0 = results_snaps[0]
        _, s_res1 = results_snaps[-1]
        if s_res1["counts"] != s_res0["counts"]:
            f_r20.append(f"RESULTS registration illegally mutated current-wave counts: {s_res0['counts']} -> {s_res1['counts']}")
        q_next = int(s_res1.get("settlement", {}).get("pending_next_wave_count", 0))
        rollover_beats = [
            b for b in fw_beats
            if int(b.get("fixture", {}).get("public_snapshot", {}).get("wave_no", 0)) == 11
        ]
        if not rollover_beats:
            f_r20.append("Missing Wave 11 rollover beat")
        else:
            s_w11 = rollover_beats[0]["fixture"]["public_snapshot"]
            if q_next > 0:
                if s_w11.get("wave_tick") != 0 or s_w11.get("phase") != "prep":
                    f_r20.append(f"Wave 11 rollover with pending_next_wave={q_next} must enter tick=0 prep, got tick={s_w11.get('wave_tick')}")
                if s_w11["counts"]["unique_joined"] != q_next:
                    f_r20.append(f"Wave 11 promoted unique_joined={s_w11['counts']['unique_joined']} != queued {q_next}")
    record("R20_RESULTS_QUEUED_NEXT_WAVE_ISOLATION", "Registrations during RESULTS stay queued for Wave+1 and promote cleanly at tick 3000 rollover", f_r20)

    # 21. R21_WALL_UPGRADE_RULES: tier increases <= +1 per 200-tick boundary with work & stone costs
    f_r21: List[str] = []
    prev_wt: Optional[int] = None
    prev_wno: Optional[int] = None
    for b in fw_beats:
        bid = b.get("beat_id", "beat")
        s = b.get("fixture", {}).get("public_snapshot", {})
        wno = int(s.get("wave_no", 0))
        wtick = int(s.get("wave_tick", -1))
        wt = int(s.get("wall_tier", 1))
        if prev_wno == wno and prev_wt is not None:
            if wt < prev_wt:
                f_r21.append(f"full_wave.{bid}: wall_tier regressed {prev_wt} -> {wt}")
            elif wt > prev_wt:
                if wt - prev_wt > 1:
                    f_r21.append(f"full_wave.{bid}: wall_tier jumped >1 in single step ({prev_wt} -> {wt})")
                if wtick <= 0 or wtick > 2800:
                    f_r21.append(f"full_wave.{bid}: illegal wall_tier upgrade at tick={wtick}")
        prev_wno = wno
        prev_wt = wt
    if isinstance(b_trace, list) and len(b_trace) > 0:
        for row in b_trace:
            if row.get("upgraded_wall"):
                old_t = int(row.get("wall_tier_before", 0))
                work_b = int(row.get("construction_work_before_upgrade", 0))
                stone_b = int(row.get("stone_before_upgrade", 0))
                if work_b < 200 * old_t or stone_b < 100 * old_t:
                    f_r21.append(f"tick {row.get('tick')} upgrade lacked required work({work_b}<{200*old_t}) or stone({stone_b}<{100*old_t})")
    else:
        f_r21.append("Missing boundary_trace proof of wall construction_work and stone upgrade thresholds")
    record("R21_WALL_UPGRADE_RULES", "Wall tier upgrades obey 200*tier work and 100*tier stone thresholds at 200-tick boundaries", f_r21)

    # 22. R22_TRUTHFUL_STATUS_AND_WATERMARK_TEXT: drops vs backpressure, watermark, music mute
    f_r22: List[str] = []
    # Check failure_05 (backpressure with 0 drops) vs failure_06 (confirmed drops > 0)
    f05 = bundle["failure_fixtures"].get("failure_05_backpressure_no_loss", {})
    f05_diag = f05.get("fixture", {}).get("public_snapshot", {}).get("diagnostics", {})
    if int(f05_diag.get("confirmed_valid_drops", -1)) != 0 or int(f05_diag.get("queue_fill_pct", 0)) < 75:
        f_r22.append("failure_05_backpressure_no_loss must have confirmed_valid_drops == 0 and queue_fill_pct >= 75")
    f06 = bundle["failure_fixtures"].get("failure_06_confirmed_valid_loss", {})
    f06_diag = f06.get("fixture", {}).get("public_snapshot", {}).get("diagnostics", {})
    if int(f06_diag.get("confirmed_valid_drops", 0)) <= 0:
        f_r22.append("failure_06_confirmed_valid_loss must have confirmed_valid_drops > 0")
    for cid, s, _ in snaps:
        src = s.get("source_status")
        if src not in ("mock", "replay", "connected", "disconnected"):
            f_r22.append(f"{cid}: unknown source_status='{src}'")
    for st in unity_ver.get("hud_verification_states", []):
        src = st.get("fixture_source_status")
        wm_vis = bool(st.get("watermark_visible", False))
        if src != "connected" and not wm_vis:
            f_r22.append(f"Unity state {st.get('state_id')} has source_status='{src}' but watermark_visible=False")
    # Music rights & mute check
    aud_pol = bundle["audio_policy"]
    rights = bundle["rights_audit"]
    if rights.get("global_rights_status") != "PENDING_REVIEW":
        f_r22.append(f"rights_audit.global_rights_status='{rights.get('global_rights_status')}' != 'PENDING_REVIEW'")
    if "APPROVED" in str(rights.get("gate_legal_status", "")).upper() or not bool(rights.get("music_muted_by_default", False)):
        f_r22.append("rights_audit illegally marks GATE-LEGAL approved or unmutes music")
    m_pol = aud_pol.get("music_safety_lock", {})
    if not bool(m_pol.get("muted_by_default", False)) or float(m_pol.get("default_volume_db", 0.0)) > -80.0 or not bool(m_pol.get("audio_source_mute_flag", False)):
        f_r22.append(f"Music safety gate violated in audio_dispatch_policy.json: {m_pol}")
    if not bool(unity_ver.get("musicMutedByDefaultVerified", False)):
        f_r22.append("Unity verification musicMutedByDefaultVerified is False")
    record("R22_TRUTHFUL_STATUS_AND_WATERMARK_TEXT", "Backpressure vs drops, synthetic watermarks, and Music PENDING_REVIEW mute (-80 dB) remain truthful", f_r22)

    # 23. R23_HUD_MATCHES_UNDERLYING_FIXTURE: SyntheticCommittedView HUD parity across all 12 states
    f_r23: List[str] = []
    if "SyntheticCommittedView" not in cs_hud and "SyntheticCommittedView" not in cs_ver:
        f_r23.append("C# showcase missing presentation-only SyntheticCommittedView adapter")
    hardcoded_literals = ["TƯỜNG 680 / 1000", "THÀNH 1850/2000", "GỖ 420", "ĐÁ 310", "LƯƠNG 540", "XÂY: 48", "THỦ: 64", "ĐÁNH: 92"]
    for lit in hardcoded_literals:
        if lit in cs_ver or lit in cs_hud:
            f_r23.append(f"Hardcoded HUD literal '{lit}' still present in Unity C# showcase source")
    required_12_states = [
        "01_waiting",
        "02_prep",
        "03_vote_open",
        "04_vote_b_locked",
        "05_assault",
        "06_boss_incoming",
        "07_boss_active",
        "08_victory",
        "09_defeat",
        "10_source_disconnected",
        "11_storage_unavailable",
        "12_wave_rollover",
    ]
    hud_states = {st.get("state_id"): st for st in unity_ver.get("hud_verification_states", [])}
    for req_id in required_12_states:
        if req_id not in hud_states:
            f_r23.append(f"unity_isolated_verification.json missing required GPU-rendered HUD state '{req_id}'")
            continue
        st = hud_states[req_id]
        f_snap = st.get("fixture_summary", {})
        if f"{f_snap.get('wall_hp')}/{f_snap.get('wall_max')}" not in str(st.get("wall_hp_text", "")).replace(" ", ""):
            f_r23.append(f"{req_id}: wall_hp_text='{st.get('wall_hp_text')}' mismatches fixture {f_snap.get('wall_hp')}/{f_snap.get('wall_max')}")
        if f"{f_snap.get('townhall_hp')}/{f_snap.get('townhall_max')}" not in str(st.get("castle_hp_text", "")).replace(" ", ""):
            f_r23.append(f"{req_id}: castle_hp_text='{st.get('castle_hp_text')}' mismatches fixture {f_snap.get('townhall_hp')}/{f_snap.get('townhall_max')}")
        if str(f_snap.get("wood")) not in str(st.get("wood_text", "")) or str(f_snap.get("stone")) not in str(st.get("stone_text", "")) or str(f_snap.get("food")) not in str(st.get("food_text", "")):
            f_r23.append(f"{req_id}: resource text ({st.get('wood_text')}, {st.get('stone_text')}, {st.get('food_text')}) mismatches fixture {f_snap}")
        if str(f_snap.get("builder")) not in str(st.get("builder_text", "")) or str(f_snap.get("guard")) not in str(st.get("guard_text", "")) or str(f_snap.get("fighter")) not in str(st.get("fighter_text", "")):
            f_r23.append(f"{req_id}: role counter text mismatches fixture {f_snap}")
        if req_id == "04_vote_b_locked":
            vtxt = str(st.get("vote_strip_text", ""))
            if "-50" not in vtxt or "+15%" not in vtxt:
                f_r23.append(f"04_vote_b_locked vote_strip_text='{vtxt}' does not display canonical -50 Food & +15% Fighter ATK")
    record("R23_HUD_MATCHES_UNDERLYING_FIXTURE", "All 12 GPU-rendered Unity HUD states derive 100% of counters from SyntheticCommittedView fixtures", f_r23)

    # 24. R24_REBUILD_HASHES_AND_PREVIEW_PROVENANCE: timeline durations, hashes & performance truthfulness
    f_r24: List[str] = []
    if timeline.get("logical_wave_duration_s") != 300:
        f_r24.append(f"presentation_timeline.json missing logical_wave_duration_s=300 (got {timeline.get('logical_wave_duration_s')})")
    if not isinstance(timeline.get("encoded_montage_duration_s"), (int, float)) or not (15.0 <= float(timeline["encoded_montage_duration_s"]) <= 35.0):
        f_r24.append(f"presentation_timeline.json missing accurate encoded_montage_duration_s (~20-30s), got {timeline.get('encoded_montage_duration_s')}")
    if timeline.get("canonical_wave10_fixture_sha256") != bundle["full_wave_sha256"]:
        f_r24.append("presentation_timeline.json canonical_wave10_fixture_sha256 does not match full_wave_300s_wave10_showcase.json")
    # Performance definitions audit (Phase 4 Findings A, B, C, D)
    g_api = str(perf.get("graphicsApi", ""))
    if "Direct3D12" in g_api:
        if "Direct3D11" in readme_txt or "D3D11" in str(unity_ver.get("honestyNote", "")):
            f_r24.append("ART-008 docs/verification mislabel Direct3D12 batchmode execution as Direct3D11/D3D11")
    tiers = perf.get("loadTiers", [])
    if not tiers:
        f_r24.append("performance_report.json missing loadTiers")
    for tr in tiers:
        tname = f"entities_{tr.get('visibleEntityCount', '?')}"
        if tr.get("onePercentLowDefinition") != "1000.0 / mean(slowest ceil(N * 0.01) frameTimesMs)":
            f_r24.append(f"{tname}: missing canonical 1% low FPS definition over slowest ceil(N*0.01) frames")
        if "p99FrameTimeMs" not in tr or "slowestOnePercentMeanMs" not in tr:
            f_r24.append(f"{tname}: must report both p99FrameTimeMs and slowestOnePercentMeanMs separately")
        else:
            sm = float(tr["slowestOnePercentMeanMs"])
            quant_tol = max(0.08, (1000.0 * 0.0006) / max(0.01, sm * sm) + 0.02)
            if sm > 0 and abs(float(tr.get("onePercentLowFps", 0.0)) - (1000.0 / sm)) > quant_tol:
                f_r24.append(f"{tname}: onePercentLowFps={tr.get('onePercentLowFps')} != 1000.0 / slowestOnePercentMeanMs({sm})")
            if float(tr.get("onePercentLowFps", 0.0)) > float(tr.get("averageFps", 0.0)) + 0.05:
                f_r24.append(f"{tname}: impossible onePercentLowFps ({tr.get('onePercentLowFps')}) > averageFps ({tr.get('averageFps')})")
        if tr.get("timingScope") != "cpu_camera_render_submission_editor_batchmode":
            f_r24.append(f"{tname}: timingScope must explicitly state cpu_camera_render_submission_editor_batchmode")
        if tr.get("standalone15MinFpsStatus") != "NOT_EXECUTED" or tr.get("obsNvencCaptureFpsStatus") != "NOT_EXECUTED":
            f_r24.append(f"{tname}: standalone/OBS FPS statuses must be explicitly NOT_EXECUTED")
        dc_status = tr.get("drawCallsStatus")
        if tr.get("drawCalls") == 0 and dc_status != "NOT_MEASURED":
            f_r24.append(f"{tname}: reports drawCalls=0 as workload instead of NOT_MEASURED")
        gc_method = tr.get("managedGcMeasurementMethod", "")
        if gc_method not in (
            "ProfilerRecorder_GC_Allocated_In_Frame",
            "GC_GetAllocatedBytesForCurrentThread_Cumulative",
            "NOT_MEASURED",
        ):
            f_r24.append(f"{tname}: invalid managed GC measurement method '{gc_method}' (net heap delta is not per-frame allocation)")
    # Check README consistency with 150-entity tier FPS
    t150 = [t for t in tiers if t.get("visibleEntityCount") == 150]
    if t150:
        avg150 = f"{float(t150[0].get('averageFps', 0.0)):.2f}"
        if avg150 not in readme_txt:
            f_r24.append(f"ART-008 README does not cite single-source 150-entity average FPS ({avg150}) from performance_report.json")
    record("R24_REBUILD_HASHES_AND_PREVIEW_PROVENANCE", "Timeline durations, SHA-256 provenance, 1% low FPS math, GC profiler counters, and README figures agree", f_r24)

    # 25. R25_NO_UNSUPPORTED_M05_WS_SCHEMA_CLAIM: explicit synthetic contract & XP <= 300 cap
    f_r25: List[str] = []
    if fw.get("m05_ws_contract_status") != "SYNTHETIC_PRESENTATION_FIXTURE_M05_NOT_STARTED":
        f_r25.append("full_wave fixture must declare m05_ws_contract_status='SYNTHETIC_PRESENTATION_FIXTURE_M05_NOT_STARTED'")
    if fw.get("is_executable_m02_replay") is not False:
        f_r25.append("full_wave fixture must explicitly set is_executable_m02_replay=false")
    for cid, s, _ in snaps:
        max_xp = s.get("max_viewer_session_xp", 0)
        if not isinstance(max_xp, int) or not (0 <= max_xp <= 300):
            f_r25.append(f"{cid}: max_viewer_session_xp={max_xp} violates 0..300 session XP cap")
    if isinstance(b_trace, list):
        for row in b_trace:
            mxp = int(row.get("max_viewer_session_xp", 0))
            if mxp < 0 or mxp > 300:
                f_r25.append(f"boundary_trace tick {row.get('tick')}: max_viewer_session_xp={mxp} exceeds 300 XP cap")
    record("R25_NO_UNSUPPORTED_M05_WS_SCHEMA_CLAIM", "Fixtures explicitly mark M05 WS schema as NOT_STARTED synthetic views and enforce <= 300 session XP cap", f_r25)

    passed_count = sum(1 for r in rules if r["passed"])
    return {
        "rules_version": "ttg-rules-v1",
        "total_rules": len(rules),
        "passed_rules": passed_count,
        "failed_rules": len(rules) - passed_count,
        "overall_passed": passed_count == len(rules),
        "golden_vectors_cross_checked": len(bundle.get("golden_cases", [])),
        "snapshots_inspected": len(snaps),
        "rules": rules,
    }


# ============================================================================
# 15 Deliberate Negative Mutation Tests (Phase 7)
# ============================================================================

def run_negative_mutation_suite(base_bundle: Dict[str, Any]) -> Dict[str, Any]:
    cases: List[Tuple[str, str, str, Callable[[Dict[str, Any]], None]]] = []

    def m01(b: Dict[str, Any]) -> None:
        b["full_wave"]["beats"][0]["fixture"]["public_snapshot"]["townhall_max"] = 2000
        b["full_wave"]["beats"][0]["fixture"]["public_snapshot"]["townhall_hp"] = 1850

    def m02(b: Dict[str, Any]) -> None:
        s = b["full_wave"]["beats"][1]["fixture"]["public_snapshot"]
        s["wall_tier"] = 8
        s["wall_max"] = 1200

    def m03(b: Dict[str, Any]) -> None:
        for beat in b["full_wave"]["beats"]:
            s = beat["fixture"]["public_snapshot"]
            if s.get("wave_no") == 10 and s.get("wave_tick") == 1600:
                s["base_hp"] = 1420
                s["lanes"] = [
                    {"id": "west", "ordinary_hp": 420, "runner_hp": 105, "grunt_hp": 231, "brute_hp": 84},
                    {"id": "center", "ordinary_hp": 560, "runner_hp": 140, "grunt_hp": 308, "brute_hp": 112},
                    {"id": "east", "ordinary_hp": 440, "runner_hp": 110, "grunt_hp": 242, "brute_hp": 88},
                ]
                break

    def m04(b: Dict[str, Any]) -> None:
        for beat in b["full_wave"]["beats"]:
            s = beat["fixture"]["public_snapshot"]
            if s.get("wave_no") == 10 and int(s.get("wave_tick", -1)) >= 1600:
                s["base_hp"] = 1420
                s["boss"]["boss_max_hp"] = 1800
                break

    def m05(b: Dict[str, Any]) -> None:
        b["cs_hud_text"] += "\n// CHỐT BẦU CHỌN [B: SỬA TƯỜNG +25%]\n"

    def m06(b: Dict[str, Any]) -> None:
        s = b["full_wave"]["beats"][0]["fixture"]["public_snapshot"]
        s["lifecycle"] = "waiting"
        s["phase"] = "waiting"
        s["wave_tick"] = 0

    def m07(b: Dict[str, Any]) -> None:
        b["full_wave"]["beats"][-1].setdefault("audio_events", []).append({"cue_id": "SFX11", "event_type": "wave_result", "mock_stream_seq": 99999})

    def m08(b: Dict[str, Any]) -> None:
        for beat in b["full_wave"]["beats"]:
            s = beat["fixture"]["public_snapshot"]
            if 1600 <= int(s.get("wave_tick", -1)) < 2600:
                s["boss"]["boss_exists"] = True
                s["boss"]["boss_hp"] = 120
                break

    def m09(b: Dict[str, Any]) -> None:
        s = b["full_wave"]["beats"][1]["fixture"]["public_snapshot"]
        s["wave_tick"] = 1500
        s["phase"] = "assault"

    def m10(b: Dict[str, Any]) -> None:
        if b["full_wave"].get("boundary_trace"):
            for r in b["full_wave"]["boundary_trace"]:
                if r.get("tick") == 2800:
                    r["settlement_reward"] = {"wood": 250, "stone": 60, "food": 50}
        else:
            b["full_wave"]["boundary_trace"] = [{"tick": 2800, "settlement_result": "win", "settlement_reward": {"wood": 250, "stone": 60, "food": 50}}]

    def m11(b: Dict[str, Any]) -> None:
        b["full_wave"]["beats"][1]["fixture"]["public_snapshot"]["max_viewer_session_xp"] = 340

    def m12(b: Dict[str, Any]) -> None:
        if b["unity_verification"].get("hud_verification_states"):
            b["unity_verification"]["hud_verification_states"][0]["castle_hp_text"] = "CHÍNH 999/500"
        else:
            b["unity_verification"]["hud_verification_states"] = [{"state_id": "01_waiting", "castle_hp_text": "CHÍNH 999/500", "fixture_summary": {"townhall_hp": 500, "townhall_max": 500}}]

    def m13(b: Dict[str, Any]) -> None:
        if b["unity_verification"].get("hud_verification_states"):
            b["unity_verification"]["hud_verification_states"][0]["fixture_source_status"] = "mock"
            b["unity_verification"]["hud_verification_states"][0]["watermark_visible"] = False
        else:
            b["unity_verification"]["hud_verification_states"] = [{"state_id": "01_waiting", "fixture_source_status": "mock", "watermark_visible": False}]

    def m14(b: Dict[str, Any]) -> None:
        b["rights_audit"]["global_rights_status"] = "APPROVED"
        b["audio_policy"]["music_safety_lock"]["muted_by_default"] = False

    def m15(b: Dict[str, Any]) -> None:
        if b["performance_report"].get("loadTiers"):
            b["performance_report"]["loadTiers"][0]["drawCalls"] = 0
            b["performance_report"]["loadTiers"][0]["drawCallsStatus"] = "MEASURED"
            b["performance_report"]["loadTiers"][0]["onePercentLowDefinition"] = "1000 / p99FrameTimeMs"

    cases = [
        ("N01_TOWNHALL_MAX_2000", "Inject townhall_max = 2000 & townhall_hp = 1850", "R02_TOWNHALL_MAX_500", m01),
        ("N02_TIER8_WALL_MAX_1200", "Inject Tier 8 with wall_max = 1200 (expected 2400)", "R03_WALL_MAX_FROM_TIER", m02),
        ("N03_INVALID_LANE_420_560_440", "Inject 420/560/440 lane split for base_hp = 1420", "R12_CANONICAL_LANE_SPLIT_AT_SPAWN", m03),
        ("N04_BOSS_MAX_1800", "Inject boss_max_hp = 1800 for base_hp = 1420 (expected 284)", "R13_BOSS_MAX_FROM_BASE_HP", m04),
        ("N05_WRONG_VOTE_B_TEXT", "Inject Vote B wall-repair text ('SỬA TƯỜNG +25%')", "R11_VOTE_ABC_COSTS_AND_EFFECTS", m05),
        ("N06_WAITING_TICK_ZERO", "Inject WAITING snapshot with wave_tick = 0", "R08_WAITING_TICK_MINUS_ONE", m06),
        ("N07_DUPLICATE_VICTORY", "Inject duplicate wave_victory settlement event", "R18_SINGLE_SETTLEMENT_AND_REWARDS", m07),
        ("N08_BOSS_BEFORE_2600", "Inject active Boss before tick 2600", "R14_BOSS_ABSENT_BEFORE_2600", m08),
        ("N09_UNSUPPORTED_PHASE_BOUNDARY", "Inject phase='assault' at tick 1500 (< 1600)", "R09_PHASE_AT_EVERY_TICK", m09),
        ("N10_INVALID_RESOURCE_AWARD", "Inject invalid +250 Wood victory settlement reward", "R18_SINGLE_SETTLEMENT_AND_REWARDS", m10),
        ("N11_INCORRECT_XP_CAP", "Inject session XP = 340 (> 300 cap)", "R25_NO_UNSUPPORTED_M05_WS_SCHEMA_CLAIM", m11),
        ("N12_STALE_UI_COUNTER", "Inject stale HUD Castle HP text disagreeing with fixture", "R23_HUD_MATCHES_UNDERLYING_FIXTURE", m12),
        ("N13_MISSING_SYNTHETIC_WATERMARK", "Inject mock source_status with watermark_visible = False", "R22_TRUTHFUL_STATUS_AND_WATERMARK_TEXT", m13),
        ("N14_WRONG_MUSIC_RIGHTS_DEFAULT", "Inject gate_legal_status=APPROVED and unmuted music", "R22_TRUTHFUL_STATUS_AND_WATERMARK_TEXT", m14),
        ("N15_FALSE_PERFORMANCE_METRIC_STATUS", "Inject 1000/p99 1%-low definition & draw_calls=0 MEASURED", "R24_REBUILD_HASHES_AND_PREVIEW_PROVENANCE", m15),
    ]

    results: List[Dict[str, Any]] = []
    for nid, desc, expected_rule, mut_fn in cases:
        mutated = copy.deepcopy(base_bundle)
        mut_fn(mutated)
        ev = evaluate_bundle(mutated)
        failed_map = {r["rule_id"]: r for r in ev["rules"] if not r["passed"]}
        caught = expected_rule in failed_map
        results.append({
            "test_id": nid,
            "description": desc,
            "expected_catching_rule": expected_rule,
            "rejected_as_expected": caught,
            "sample_failure_reason": failed_map[expected_rule]["failures"][0] if caught else "NOT_REJECTED",
        })

    rejected_count = sum(1 for r in results if r["rejected_as_expected"])
    return {
        "total_negative_tests": len(results),
        "rejected_count": rejected_count,
        "all_rejected": rejected_count == len(results),
        "results": results,
    }


# ============================================================================
# ART-009 Hotfix Separately Reported Countdown & Audio-Handoff Regression Suite
# ============================================================================

EXPECTED_HOTFIX_BOUNDARIES: List[Tuple[int, str, int, str, str]] = [
    (999, "prep", 61, "ĐỢT CHẴN BẦU CHỌN · GÕ A / B / C", "EVEN WAVE VOTE · TYPE A / B / C"),
    (1000, "prep", 60, "CẢNH BÁO: CÒN 60 GIÂY QUÁI TỚI!", "WARNING: 60s UNTIL ENEMY ASSAULT!"),
    (1199, "prep", 41, "CẢNH BÁO: CÒN 41 GIÂY QUÁI TỚI!", "WARNING: 41s UNTIL ENEMY ASSAULT!"),
    (1200, "prep", 40, "CẢNH BÁO: CÒN 40 GIÂY QUÁI TỚI!", "WARNING: 40s UNTIL ENEMY ASSAULT!"),
    (1449, "prep", 16, "CẢNH BÁO: CÒN 16 GIÂY QUÁI TỚI!", "WARNING: 16s UNTIL ENEMY ASSAULT!"),
    (1450, "prep", 15, "KHẨN CẤP: QUÁI TỚI TRONG 15 GIÂY!", "URGENT: ENEMIES IN 15 SECONDS!"),
    (1500, "prep", 10, "KHẨN CẤP: QUÁI TỚI TRONG 10 GIÂY!", "URGENT: ENEMIES IN 10 SECONDS!"),
    (1599, "prep", 1, "KHẨN CẤP: QUÁI TỚI TRONG 1 GIÂY!", "URGENT: ENEMIES IN 1 SECOND!"),
    (1600, "assault", 0, "GIAO TRANH · T:210 · G:198 · Đ:212", "ASSAULT · W:210 · C:198 · E:212"),
]


def _make_boundary_test_fixture(tick: int, phase: str) -> Dict[str, Any]:
    if phase == "assault":
        return make_snapshot(
            stream_seq=5000 + tick, wave_no=10, wave_tick=tick, phase="assault",
            total_wins=39, consecutive_wins_before_wave=5, wall_tier=7, wall_hp=2200, townhall_hp=500,
            wood=592, stone=256, food=490, builder=48, guard=92, fighter=64,
            population_anchor=204, west_hp=210, center_hp=198, east_hp=212, boss_max_hp=124,
            vote_selected="B", vote_a=34, vote_b=68, vote_c=25,
        )
    vote_open = 800 <= tick <= 1199
    return make_snapshot(
        stream_seq=5000 + tick, wave_no=10, wave_tick=tick, phase="prep",
        total_wins=39, consecutive_wins_before_wave=5, wall_tier=6 if tick < 1400 else 7,
        wall_hp=1944 if tick < 1400 else 2200, townhall_hp=500,
        wood=560, stone=820 if tick < 1400 else 250, food=540 if vote_open else 490,
        builder=121 if tick < 1450 else 64, guard=31 if tick < 1450 else 80, fighter=52 if tick < 1450 else 60,
        vote_open=vote_open, vote_selected=None if vote_open else "B", vote_a=34, vote_b=68, vote_c=25,
    )


def evaluate_hotfix_regressions(bundle: Dict[str, Any]) -> Dict[str, Any]:
    checks: List[Dict[str, Any]] = []

    def rec(cid: str, title: str, failures: List[str]) -> None:
        checks.append({
            "check_id": cid,
            "title": title,
            "passed": len(failures) == 0,
            "failure_count": len(failures),
            "failures": failures[:12],
        })

    uver = bundle["unity_verification"]
    rb_by_tick = {int(r["wave_tick"]): r for r in uver.get("prep_countdown_readbacks", [])}
    hud_by_id = {s["state_id"]: s for s in uver.get("hud_verification_states", [])}
    extra_by_id = {s["state_id"]: s for s in uver.get("prep_warning_gpu_renders", [])}

    # H01: Dynamic PREP warning countdown across ticks 999..1600 in BOTH Vietnamese & English
    f_h01: List[str] = []
    for tick, exp_phase, exp_sec, exp_vi, exp_en in EXPECTED_HOTFIX_BOUNDARIES:
        fx = _make_boundary_test_fixture(tick, exp_phase)
        py_vi = resolve_authoritative_hud_banner(fx, locale="vi-VN")
        py_en = resolve_authoritative_hud_banner(fx, locale="en-US")
        if py_vi["seconds_until_assault"] != exp_sec:
            f_h01.append(f"shared_formatter tick={tick}: seconds_until_assault={py_vi['seconds_until_assault']} != {exp_sec}")
        if py_vi["banner_text"] != exp_vi:
            f_h01.append(f"shared_formatter tick={tick} vi-VN: '{py_vi['banner_text']}' != '{exp_vi}'")
        if py_en["banner_text"] != exp_en:
            f_h01.append(f"shared_formatter tick={tick} en-US: '{py_en['banner_text']}' != '{exp_en}'")
        if tick in (999, 1000, 1199):
            exp_v_rem = (1200 - tick + 9) // 10
            if f"VOTE {exp_v_rem}s" not in py_vi["vote_strip_text"] or f"VOTE {exp_v_rem}s" not in py_en["vote_strip_text"]:
                f_h01.append(f"shared_formatter tick={tick}: subordinate even-wave vote strip missing VOTE {exp_v_rem}s")

        rb = rb_by_tick.get(tick)
        if rb is None:
            f_h01.append(f"unity_readback missing boundary tick={tick}")
            continue
        if rb.get("phase") != exp_phase:
            f_h01.append(f"unity_readback tick={tick}: phase={rb.get('phase')} != {exp_phase}")
        if int(rb.get("seconds_until_assault", -999)) != exp_sec:
            f_h01.append(f"unity_readback tick={tick}: seconds_until_assault={rb.get('seconds_until_assault')} != {exp_sec}")
        if rb.get("vi_banner_text") != exp_vi:
            f_h01.append(f"unity_readback tick={tick} vi-VN: '{rb.get('vi_banner_text')}' != '{exp_vi}'")
        if rb.get("en_banner_text") != exp_en:
            f_h01.append(f"unity_readback tick={tick} en-US: '{rb.get('en_banner_text')}' != '{exp_en}'")
        if not bool(rb.get("text_fits_without_overflow")) or float(rb.get("vi_preferred_width_px", 9999.0)) > 816.0 or float(rb.get("en_preferred_width_px", 9999.0)) > 816.0:
            f_h01.append(f"unity_readback tick={tick}: banner text overflows 816px inner box (vi={rb.get('vi_preferred_width_px')}, en={rb.get('en_preferred_width_px')})")
        if tick in (999, 1000, 1199):
            exp_v_rem = (1200 - tick + 9) // 10
            if f"VOTE {exp_v_rem}s" not in str(rb.get("vi_vote_strip_text", "")) or f"VOTE {exp_v_rem}s" not in str(rb.get("en_vote_strip_text", "")):
                f_h01.append(f"unity_readback tick={tick}: even-wave vote strip not preserved ({rb.get('vi_vote_strip_text')})")

    st04 = hud_by_id.get("04_vote_b_locked")
    if not st04 or st04.get("threat_banner_text") != "CẢNH BÁO: CÒN 40 GIÂY QUÁI TỚI!":
        f_h01.append(f"04_vote_b_locked threat_banner_text='{st04.get('threat_banner_text') if st04 else None}' != 'CẢNH BÁO: CÒN 40 GIÂY QUÁI TỚI!'")
    en04 = uver.get("english_vote_b_proof", {})
    if en04.get("threat_banner_text") != "WARNING: 40s UNTIL ENEMY ASSAULT!":
        f_h01.append(f"04b_vote_b_locked_en threat_banner_text='{en04.get('threat_banner_text')}' != 'WARNING: 40s UNTIL ENEMY ASSAULT!'")
    st05 = hud_by_id.get("05_assault")
    if st05 and any(tok in str(st05.get("threat_banner_text", "")) for tok in ("CẢNH BÁO: CÒN", "KHẨN CẤP: QUÁI TỚI", "UNTIL ENEMY ASSAULT", "URGENT: ENEMIES IN")):
        f_h01.append(f"05_assault displays PREP warning during ASSAULT: '{st05.get('threat_banner_text')}'")
    if extra_by_id.get("03b_prep_60_warn", {}).get("threat_banner_text") != "CẢNH BÁO: CÒN 60 GIÂY QUÁI TỚI!":
        f_h01.append("03b_prep_60_warn GPU render state missing or not displaying 60s warning")
    if extra_by_id.get("04c_prep_15_warn", {}).get("threat_banner_text") != "KHẨN CẤP: QUÁI TỚI TRONG 15 GIÂY!":
        f_h01.append("04c_prep_15_warn GPU render state missing or not displaying 15s urgent warning")
    rec(
        "H01_DYNAMIC_PREP_COUNTDOWN_BOUNDARIES_VI_EN",
        "Dynamic PREP warning countdown verified across ticks 999..1600 in VI & EN (shared formatter + Unity GPU read-backs + 360p fit)",
        f_h01,
    )

    # H02: Paused state never advances via wall-clock catch-up + blocker priority preserved
    f_h02: List[str] = []
    pf = uver.get("paused_wallclock_freeze_proof")
    if not isinstance(pf, dict):
        f_h02.append("Missing paused_wallclock_freeze_proof in unity_isolated_verification.json")
    else:
        if bool(pf.get("advanced_from_wallclock")):
            f_h02.append("paused_wallclock_freeze_proof reports advanced_from_wallclock == True")
        if not bool(pf.get("priority_storage_gt_source_gt_pause_verified")):
            f_h02.append("Blocker priority storage_unavailable > source_disconnected > operator_paused failed")
        if pf.get("paused_vi_banner_before") != "TẠM DỪNG ĐIỀU HÀNH · ĐỒNG HỒ DỪNG" or pf.get("paused_vi_banner_after") != pf.get("paused_vi_banner_before"):
            f_h02.append(f"Paused banner drifted across 25s wall-clock step: '{pf.get('paused_vi_banner_before')}' -> '{pf.get('paused_vi_banner_after')}'")
        if pf.get("paused_vi_timer_before") != pf.get("paused_vi_timer_after"):
            f_h02.append(f"Paused timer drifted across 25s wall-clock step: '{pf.get('paused_vi_timer_before')}' -> '{pf.get('paused_vi_timer_after')}'")
        if pf.get("running_no_snapshot_vi_banner_before") != "CẢNH BÁO: CÒN 40 GIÂY QUÁI TỚI!" or pf.get("running_no_snapshot_vi_banner_after") != "CẢNH BÁO: CÒN 40 GIÂY QUÁI TỚI!":
            f_h02.append(f"Un-stepped snapshot drifted from wall-clock: '{pf.get('running_no_snapshot_vi_banner_before')}' -> '{pf.get('running_no_snapshot_vi_banner_after')}'")

    paused_fx = make_snapshot(
        stream_seq=5200, wave_no=10, wave_tick=1200, phase="prep", lifecycle="paused",
        pause_reasons=["operator_paused"], total_wins=39, wall_tier=6, wall_hp=1944, townhall_hp=500,
    )
    if resolve_authoritative_hud_banner(paused_fx, "vi-VN", simulated_wallclock_elapsed_ms=0) != resolve_authoritative_hud_banner(paused_fx, "vi-VN", simulated_wallclock_elapsed_ms=25000):
        f_h02.append("Shared formatter changed output when simulated_wallclock_elapsed_ms=25000")
    rec(
        "H02_PAUSED_NO_WALLCLOCK_CATCHUP_AND_PRIORITY",
        "Zero wall-clock countdown catch-up while paused/awaiting snapshot & STORAGE_UNAVAILABLE > SOURCE_RECONNECTING > PAUSED priority",
        f_h02,
    )

    # H03: Canonical 36.00s ambient loop duration matches ART-007 manifest across ART-008 & ART-009 handoff docs
    f_h03: List[str] = []
    m7 = bundle["art007_manifest"]
    amb_assets = [a for a in m7.get("assets", []) if a.get("cue_id") == "AMB01"]
    if (
        m7.get("sfx_count") != 12
        or m7.get("ambient_count") != 1
        or m7.get("music_count") != 2
        or m7.get("music_muted_by_default") is not True
        or m7.get("rights_status") != "PENDING_REVIEW"
        or len(amb_assets) != 1
    ):
        f_h03.append("ART-007 canonical manifest counts/mute/rights invariants unexpected")
    else:
        amb_dur = float(amb_assets[0].get("duration_s", 0.0))
        if abs(amb_dur - 36.0) > 0.01:
            f_h03.append(f"ART-007 AMB01 duration_s={amb_dur} != 36.0")
        handoff_txt = bundle["art009_handoff_doc"]
        ev_contract_txt = bundle["art008_event_contract_doc"]
        dir_doc_txt = bundle.get("art008_direction_doc", "")
        if "1 original seamless 36.00s ambient loop" not in handoff_txt:
            f_h03.append("ART-009 UNITY_HANDOFF_READINESS.md missing exact '1 original seamless 36.00s ambient loop'")
        if "36.00s" not in ev_contract_txt:
            f_h03.append("ART-008 PRESENTATION_EVENT_CONTRACT.md missing '36.00s' ambient loop duration")
        if dir_doc_txt and "36.00 s" not in dir_doc_txt and "36.00s" not in dir_doc_txt:
            f_h03.append("ART-008 AUDIOVISUAL_DIRECTION.md missing '36.00s' AMB01 loop duration")
        wrong_20s_pat = re.compile(r"(?:20(?:\.0+)?\s*s[^\n]{0,40}(?:ambient|amb01)|(?:ambient|amb01)[^\n]{0,40}20(?:\.0+)?\s*s)", re.IGNORECASE)
        if wrong_20s_pat.search(handoff_txt):
            f_h03.append("ART-009 UNITY_HANDOFF_READINESS.md contains stale 20s ambient loop claim")
        if wrong_20s_pat.search(ev_contract_txt):
            f_h03.append("ART-008 PRESENTATION_EVENT_CONTRACT.md contains stale 20s ambient loop claim")
        if dir_doc_txt and wrong_20s_pat.search(dir_doc_txt):
            f_h03.append("ART-008 AUDIOVISUAL_DIRECTION.md contains stale 20s AMB01 loop claim")
    rec(
        "H03_AMBIENT_LOOP_36S_HANDOFF_PARITY",
        "ART-009 & ART-008 handoff docs state exact 36.00s ambient loop matching ART-007 manifest (12 SFX, 1 ambient, 2 music muted, PENDING_REVIEW)",
        f_h03,
    )

    passed_checks = sum(1 for c in checks if c["passed"])
    return {
        "suite_id": "art009-hotfix-countdown-and-audio-regression-v1",
        "total_checks": len(checks),
        "passed_checks": passed_checks,
        "failed_checks": len(checks) - passed_checks,
        "overall_passed": passed_checks == len(checks),
        "boundary_readbacks": uver.get("prep_countdown_readbacks", []),
        "paused_wallclock_freeze_proof": uver.get("paused_wallclock_freeze_proof", {}),
        "checks": checks,
    }


def run_hotfix_negative_mutations(base_bundle: Dict[str, Any]) -> Dict[str, Any]:
    def hn01(b: Dict[str, Any]) -> None:
        for r in b["unity_verification"].get("prep_countdown_readbacks", []):
            if int(r.get("wave_tick", -1)) == 1200:
                r["vi_banner_text"] = "CẢNH BÁO: CÒN 60 GIÂY QUÁI TỚI!"
                r["en_banner_text"] = "WARNING: 60s UNTIL ENEMY ASSAULT!"

    def hn02(b: Dict[str, Any]) -> None:
        for r in b["unity_verification"].get("prep_countdown_readbacks", []):
            if int(r.get("wave_tick", -1)) == 1500:
                r["vi_banner_text"] = "KHẨN CẤP: QUÁI TỚI TRONG 15 GIÂY!"
                r["en_banner_text"] = "URGENT: ENEMIES IN 15 SECONDS!"

    def hn03(b: Dict[str, Any]) -> None:
        for r in b["unity_verification"].get("prep_countdown_readbacks", []):
            if int(r.get("wave_tick", -1)) == 1600:
                r["vi_banner_text"] = "KHẨN CẤP: QUÁI TỚI TRONG 0 GIÂY!"
                r["en_banner_text"] = "URGENT: ENEMIES IN 0 SECONDS!"
        for s in b["unity_verification"].get("hud_verification_states", []):
            if s.get("state_id") == "05_assault":
                s["threat_banner_text"] = "KHẨN CẤP: QUÁI TỚI TRONG 1 GIÂY!"

    def hn04(b: Dict[str, Any]) -> None:
        for r in b["unity_verification"].get("prep_countdown_readbacks", []):
            if int(r.get("wave_tick", -1)) == 1599:
                r["seconds_until_assault"] = 0
                r["vi_banner_text"] = "KHẨN CẤP: QUÁI TỚI TRONG 0 GIÂY!"
                r["en_banner_text"] = "URGENT: ENEMIES IN 0 SECONDS!"

    def hn05(b: Dict[str, Any]) -> None:
        pf = b["unity_verification"].setdefault("paused_wallclock_freeze_proof", {})
        pf["paused_vi_banner_after"] = "KHẨN CẤP: QUÁI TỚI TRONG 15 GIÂY!"
        pf["paused_vi_timer_after"] = "GIỜ 02:35"
        pf["advanced_from_wallclock"] = True

    def hn06(b: Dict[str, Any]) -> None:
        b["art009_handoff_doc"] = b["art009_handoff_doc"].replace(
            "1 original seamless 36.00s ambient loop",
            "1 seamless 20.0s ambient loop",
        )

    cases: List[Tuple[str, str, str, Callable[[Dict[str, Any]], None]]] = [
        ("HN01_HARDCODED_60S_AT_TICK_1200", "Inject hardcoded 60s PREP warning at tick 1200", "H01_DYNAMIC_PREP_COUNTDOWN_BOUNDARIES_VI_EN", hn01),
        ("HN02_HARDCODED_15S_AT_TICK_1500", "Inject hardcoded 15s urgent warning at tick 1500", "H01_DYNAMIC_PREP_COUNTDOWN_BOUNDARIES_VI_EN", hn02),
        ("HN03_PREP_WARNING_VISIBLE_DURING_ASSAULT", "Inject PREP warning banner visible during ASSAULT at tick 1600", "H01_DYNAMIC_PREP_COUNTDOWN_BOUNDARIES_VI_EN", hn03),
        ("HN04_INCORRECT_ROUNDING_AT_TICK_1599", "Inject floor-division 0s countdown rounding at tick 1599", "H01_DYNAMIC_PREP_COUNTDOWN_BOUNDARIES_VI_EN", hn04),
        ("HN05_PAUSED_WALLCLOCK_CATCHUP", "Inject paused-state presentation advancing via 25s wall-clock catch-up", "H02_PAUSED_NO_WALLCLOCK_CATCHUP_AND_PRIORITY", hn05),
        ("HN06_WRONG_20S_AMBIENT_DURATION_IN_HANDOFF", "Inject wrong 20.0s ambient loop duration into ART-009 handoff doc", "H03_AMBIENT_LOOP_36S_HANDOFF_PARITY", hn06),
    ]

    results: List[Dict[str, Any]] = []
    for hnid, desc, exp_check, mut_fn in cases:
        mutated = copy.deepcopy(base_bundle)
        mut_fn(mutated)
        ev = evaluate_hotfix_regressions(mutated)
        failed_map = {c["check_id"]: c for c in ev["checks"] if not c["passed"]}
        caught = exp_check in failed_map
        results.append({
            "test_id": hnid,
            "description": desc,
            "expected_catching_check": exp_check,
            "rejected_as_expected": caught,
            "sample_failure_reason": failed_map[exp_check]["failures"][0] if caught else "NOT_REJECTED",
        })

    rej_cnt = sum(1 for r in results if r["rejected_as_expected"])
    return {
        "total_hotfix_negative_tests": len(results),
        "rejected_count": rej_cnt,
        "all_rejected": rej_cnt == len(results),
        "results": results,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate ART-008/009 canonical gameplay fixtures against ttg-rules-v1.")
    parser.add_argument("--ci-fast", action="store_true", help="Fast deterministic CI mode (pure Python stdlib).")
    parser.add_argument("--write-reports", action="store_true", help="Write canonical regression & negative test JSONs to ART-009/qa/.")
    parser.add_argument("--save-baseline", type=str, default="", help="Optional path to save pre-remediation failing baseline JSON.")
    args = parser.parse_args()

    bundle = load_bundle_from_repo()
    pos = evaluate_bundle(bundle)
    neg = run_negative_mutation_suite(bundle)
    hotfix_pos = evaluate_hotfix_regressions(bundle)
    hotfix_neg = run_hotfix_negative_mutations(bundle)

    if args.save_baseline:
        write_json_lf(Path(args.save_baseline), {"positive_audit": pos, "negative_suite": neg})

    if args.write_reports:
        qa_dir = ART009_DIR / "qa"
        write_json_lf(qa_dir / "canonical_rules_regression.json", pos)
        write_json_lf(qa_dir / "negative_test_results.json", neg)
        write_json_lf(
            qa_dir / "hotfix_countdown_and_audio_regression.json",
            {"positive_checks": hotfix_pos, "negative_mutations": hotfix_neg},
        )

    print(
        f"[ART-009 Canonical Validator] Rules: {pos['passed_rules']}/{pos['total_rules']} PASS | "
        f"Negative Mutations Rejected: {neg['rejected_count']}/{neg['total_negative_tests']} | "
        f"Hotfix Regressions: {hotfix_pos['passed_checks']}/{hotfix_pos['total_checks']} PASS | "
        f"Hotfix Negatives Rejected: {hotfix_neg['rejected_count']}/{hotfix_neg['total_hotfix_negative_tests']} | "
        f"Snapshots Inspected: {pos['snapshots_inspected']} | Golden Cases Cross-Checked: {pos['golden_vectors_cross_checked']}"
    )
    for r in pos["rules"]:
        status = "PASS" if r["passed"] else "FAIL"
        detail = f" -> {r['failures'][0]}" if r["failures"] else ""
        print(f"  [{status}] {r['rule_id']}: {r['title']}{detail}")

    print("[ART-009 Hotfix Regression Suite — Reported Separately]")
    for hc in hotfix_pos["checks"]:
        status = "PASS" if hc["passed"] else "FAIL"
        detail = f" -> {hc['failures'][0]}" if hc["failures"] else ""
        print(f"  [{status}] {hc['check_id']}: {hc['title']}{detail}")
    for hnr in hotfix_neg["results"]:
        status = "PASS" if hnr["rejected_as_expected"] else "FAIL"
        print(f"  [HOTFIX NEG {status}] {hnr['test_id']} -> {hnr['sample_failure_reason']}")

    if not neg["all_rejected"]:
        for nr in neg["results"]:
            if not nr["rejected_as_expected"]:
                print(f"  [NEG FAIL] {nr['test_id']} was NOT rejected by {nr['expected_catching_rule']}")

    if not pos["overall_passed"] or not neg["all_rejected"] or not hotfix_pos["overall_passed"] or not hotfix_neg["all_rejected"]:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
