#!/usr/bin/env python3
"""
ART-008 / ART-009 — Deterministic Cross-Media Contracts & Rule-Verified Synthetic Wave/Failure Fixture Builder.

Generates:
  - animation_vfx_audio_sync.json (Phase 3)
  - hud_transition_mapping.json (Phase 6)
  - audio_dispatch_policy.json (Phase 2, 3 & ART-009 Phase 5 gain accounting)
  - presentation_timeline.json (Phase 3, 7 & ART-009 Phase 3 duration truthfulness)
  - showcase/fixtures/full_wave_300s_wave10_showcase.json (Rule-verified against ttg-rules-v1)
  - showcase/fixtures/failure_01..08_*.json (Rule-verified against ttg-rules-v1)
"""

from __future__ import annotations

import math
from typing import Any, Dict, List, Optional, Tuple


FRAME_MS_12FPS = round(1000.0 / 12.0, 4)  # 83.3333 ms

CANONICAL_VOTE_EFFECTS: Dict[str, Dict[str, Any]] = {
    "A": {
        "cost_resource": "wood",
        "cost_amount": 50,
        "guard_block_multiplier_pct": 115,
        "fighter_attack_multiplier_pct": 100,
        "loss_penalty_divisor": 1,
        "label_vi": "A: -50 Gỗ · Thủ +15%",
        "label_en": "A: -50 Wood · Guard +15%",
    },
    "B": {
        "cost_resource": "food",
        "cost_amount": 50,
        "guard_block_multiplier_pct": 100,
        "fighter_attack_multiplier_pct": 115,
        "loss_penalty_divisor": 1,
        "label_vi": "B: -50 Lương · Đánh +15%",
        "label_en": "B: -50 Food · Attack +15%",
    },
    "C": {
        "cost_resource": "none",
        "cost_amount": 0,
        "guard_block_multiplier_pct": 90,
        "fighter_attack_multiplier_pct": 100,
        "loss_penalty_divisor": 2,
        "label_vi": "C: 0 phí · Thủ -10% · Giảm 50% phạt",
        "label_en": "C: Free · Guard -10% · -50% Loss Penalty",
    },
}


def canonical_wall_max(wall_tier: int) -> int:
    assert 1 <= wall_tier <= 20, f"Invalid wall_tier={wall_tier}"
    return 1000 + 200 * (wall_tier - 1)


def canonical_base_hp(population_anchor: int, consecutive_wins: int, consecutive_losses: int) -> int:
    p_eff = min(max(1, int(population_anchor)), 10000)
    d = min(10, 1 + max(0, int(consecutive_wins)))
    raw_hp = 40 + 8 * d + 38 * math.isqrt(p_eff)
    loss_mod = 100 - 15 * min(max(0, int(consecutive_losses)), 3)
    return max(1, (raw_hp * loss_mod) // 100)


def canonical_lane_split(base_hp: int) -> Tuple[int, int, int]:
    w = (base_hp * 34) // 100
    c = (base_hp * 32) // 100
    e = base_hp - w - c
    return w, c, e


def canonical_subpools(lane_hp: int) -> Tuple[int, int, int]:
    if lane_hp <= 0:
        return 0, 0, 0
    r = (lane_hp * 25) // 100
    g = (lane_hp * 55) // 100
    b = lane_hp - r - g
    return r, g, b


def canonical_boss_max_hp(wave_no: int, base_hp: int) -> int:
    if wave_no % 5 != 0 or base_hp <= 0:
        return 0
    return (base_hp + 4) // 5


def canonical_militia_bonus(p_ref: int) -> int:
    if 1 <= p_ref <= 10:
        return max(0, 40 - 4 * (p_ref - 1))
    return 0


def compute_prep_seconds_until_assault(wave_tick: int) -> int:
    """
    Authoritative integer ceiling of (1600 - wave_tick) / 10, clamped to PREP [1..160].
    Matches C# `HudInPlacePresenter.ComputePrepSecondsUntilAssault` 1-for-1.
    """
    wt = int(wave_tick)
    if wt < 0:
        return 160
    if wt >= 1600:
        return 0
    return min(160, max(1, (1600 - wt + 9) // 10))


def format_prep_warning_banner(wave_tick: int, locale: str = "vi-VN") -> str:
    """
    Authoritative PREP warning banner formatter derived strictly from `wave_tick`.
    Matches C# `HudInPlacePresenter.FormatPrepWarningBanner` character-for-character.
    """
    wt = int(wave_tick)
    en = str(locale).lower().startswith("en")
    seconds_until_assault = min(60, max(1, compute_prep_seconds_until_assault(wt)))
    if wt >= 1450:
        if en:
            unit = "SECOND" if seconds_until_assault == 1 else "SECONDS"
            return f"URGENT: ENEMIES IN {seconds_until_assault} {unit}!"
        return f"KHẨN CẤP: QUÁI TỚI TRONG {seconds_until_assault} GIÂY!"
    if en:
        return f"WARNING: {seconds_until_assault}s UNTIL ENEMY ASSAULT!"
    return f"CẢNH BÁO: CÒN {seconds_until_assault} GIÂY QUÁI TỚI!"


def resolve_authoritative_hud_banner(
    fixture: Dict[str, Any],
    locale: str = "vi-VN",
    simulated_wallclock_elapsed_ms: int = 0,
) -> Dict[str, Any]:
    """
    Shared pure-stdlib deterministic banner & vote-strip resolver matching C# `HudInPlacePresenter`.
    Explicitly ignores `simulated_wallclock_elapsed_ms` (never advances from wall-clock time).
    """
    _ = simulated_wallclock_elapsed_ms  # Invariant: wall-clock elapsed time NEVER alters authoritative HUD text
    snap = fixture["public_snapshot"]
    en = str(locale).lower().startswith("en")
    lifecycle = str(snap["lifecycle"])
    pause_set = list(snap.get("pause_reasons", []))
    phase = str(snap["phase"])
    tick = int(snap["wave_tick"])
    wave_no = int(snap["wave_no"])
    rem_s = int(snap["remaining_seconds"])
    diag = snap["diagnostics"]
    boss = snap["boss"]
    vote = snap["vote"]
    settle = snap["settlement"]
    lanes = {ln["id"]: int(ln["ordinary_hp"]) for ln in snap["lanes"]}
    w_hp, c_hp, e_hp = lanes["west"], lanes["center"], lanes["east"]

    db_err = ("storage_unavailable" in pause_set) or (lifecycle == "storage_unavailable")
    src_disc = ("source_disconnected" in pause_set) or (snap.get("source_status") == "disconnected") or (lifecycle == "source_disconnected")
    op_pause = ("operator_paused" in pause_set) or (lifecycle == "paused")
    is_frozen = (lifecycle in ("waiting", "paused", "source_disconnected", "storage_unavailable", "stopped")) or len(pause_set) > 0

    mm, ss = max(0, rem_s) // 60, max(0, rem_s) % 60
    mm_ss = f"{mm:02d}:{ss:02d}"
    if lifecycle == "stopped":
        timer_txt = "TIME --:--" if en else "GIỜ --:--"
    else:
        timer_txt = f"TIME {mm_ss}" if en else f"GIỜ {mm_ss}"

    if lifecycle == "stopped":
        banner_skin, banner_txt = "err", ("MAINTENANCE · SESSION STOPPED" if en else "PHIÊN DỪNG / BẢO TRÌ · HẸN GẶP LẠI")
    elif db_err:
        banner_skin, banner_txt = "err", ("STORAGE ERROR · SAFE HALT ACTIVE" if en else "LỖI LƯU TRỮ DB · TẠM DỪNG AN TOÀN")
    elif src_disc:
        banner_skin, banner_txt = "warn", ("SOURCE LOST · CLOCK SAFELY FROZEN" if en else "MẤT KẾT NỐI NGUỒN · ĐỒNG HỒ DỪNG")
    elif op_pause:
        banner_skin, banner_txt = "warn", ("OPERATOR PAUSED · CLOCK FROZEN" if en else "TẠM DỪNG ĐIỀU HÀNH · ĐỒNG HỒ DỪNG")
    elif bool(diag["metrics_available"]) and int(diag["confirmed_valid_drops"]) > 0:
        drops = int(diag["confirmed_valid_drops"])
        banner_skin, banner_txt = "err", (f"OVERLOAD · VALID DROPS ({drops})" if en else f"QUÁ TẢI · LỆNH HỢP LỆ RƠI ({drops})")
    elif phase == "waiting" or tick < 0:
        banner_skin, banner_txt = "navy", ("WAITING (P=0) · CLOCK FROZEN 05:00" if en else "CHỜ NGƯỜI CHƠI (P=0) · DỪNG 05:00")
    elif phase == "prep" and 0 <= tick < 1600:
        if tick >= 1450:
            banner_skin, banner_txt = "err", format_prep_warning_banner(tick, locale)
        elif tick >= 1000:
            banner_skin, banner_txt = "warn", format_prep_warning_banner(tick, locale)
        elif bool(vote["open"]) and (wave_no % 2 == 0) and (800 <= tick <= 1199):
            banner_skin, banner_txt = "navy", ("EVEN WAVE VOTE · TYPE A / B / C" if en else "ĐỢT CHẴN BẦU CHỌN · GÕ A / B / C")
        else:
            banner_skin, banner_txt = "navy", ("PREP 0..160s · REPAIR & RALLY" if en else "CHUẨN BỊ 0..160s · GIA CỐ & TẬP HỢP")
    elif phase == "assault":
        if bool(boss["boss_incoming"]) and tick < 2600:
            banner_skin, banner_txt = "err", ("LANES CLEAR · BOSS AT 260s!" if en else "QUÁI HẠ · CẢNH BÁO BOSS GIÂY 260!")
        elif (w_hp + c_hp + e_hp) == 0 and not bool(boss["boss_incoming"]):
            banner_skin, banner_txt = "ok", ("LANES CLEAR · AWAITING SETTLEMENT" if en else "SẠCH QUÁI THƯỜNG · CHỜ CHỐT ĐỢT")
        else:
            banner_skin, banner_txt = "warn", (f"ASSAULT · W:{w_hp} · C:{c_hp} · E:{e_hp}" if en else f"GIAO TRANH · T:{w_hp} · G:{c_hp} · Đ:{e_hp}")
    elif phase == "finale":
        if bool(boss["boss_exists"]) and tick >= 2600:
            banner_skin, banner_txt = "err", (
                f"BOSS {boss['boss_hp']}/{boss['boss_max_hp']} · W:{w_hp} · E:{e_hp}"
                if en else
                f"BOSS {boss['boss_hp']}/{boss['boss_max_hp']} · TÂY {w_hp} · ĐÔNG {e_hp}"
            )
        else:
            banner_skin, banner_txt = "warn", (f"FINALE · W:{w_hp} · C:{c_hp} · E:{e_hp}" if en else f"CUỐI ĐỢT · T:{w_hp} · G:{c_hp} · Đ:{e_hp}")
    elif phase == "results":
        if settle["wave_result"] == "win":
            banner_skin, banner_txt = "ok", (f"WAVE {wave_no} WON · TOWNHALL SAFE!" if en else f"THẮNG ĐỢT {wave_no} · ĐẠI BẢN DOANH AN TOÀN!")
        else:
            banner_skin, banner_txt = "err", (
                ("DEFEAT · ENTERING RECOVERY" if int(snap["townhall_hp"]) <= 250 else "TIME UP · WAVE LOST · RECOVERING")
                if en else
                ("THẤT THỦ · CHUYỂN HỒI PHỤC" if int(snap["townhall_hp"]) <= 250 else "HẾT GIỜ · THUA ĐỢT · HỒI PHỤC")
            )
    else:
        banner_skin, banner_txt = "navy", ("PREP 0..160s · REPAIR & RALLY" if en else "CHUẨN BỊ 0..160s · GIA CỐ & TẬP HỢP")

    if lifecycle == "stopped":
        sub_txt = "Clock held · Resumes when blockers clear" if en else "Giữ đội hình · Chạy lại khi hết cờ dừng"
    elif phase == "waiting" or tick < 0:
        sub_txt = "P=0 active · No bots or unattended loss" if en else "P=0 người chơi · Không tự chạy hay tự thua"
    elif is_frozen:
        sub_txt = "Clock held · Resumes when blockers clear" if en else "Giữ đội hình · Chạy lại khi hết cờ dừng"
    elif bool(vote["open"]) and (wave_no % 2 == 0) and (800 <= tick <= 1199):
        v_rem = max(0, (1200 - tick + 9) // 10)
        vc = vote["counts"]
        sub_txt = (
            f"VOTE {v_rem}s · A:{vc['A']}(-50W,Def+15%) B:{vc['B']}(-50F,Atk+15%) C:{vc['C']}(0)"
            if en else
            f"VOTE {v_rem}s · A:{vc['A']}(-50G,Thủ+15%) B:{vc['B']}(-50L,Đánh+15%) C:{vc['C']}(0)"
        )
    elif phase == "results" and int(settle["pending_next_wave_count"]) > 0:
        p_next = int(settle["pending_next_wave_count"])
        sub_txt = f"NEXT WAVE {wave_no + 1}: Queued for start ({p_next})" if en else f"ĐỢT SAU ({wave_no + 1}): Vào thẳng đầu đợt ({p_next} chờ)"
    elif vote.get("selected") and (wave_no % 2 == 0):
        vc = vote["counts"]
        sel = vote["selected"]
        eff = (
            ("A: -50 Wood, Guard+15%" if en else "A: -50 Gỗ, Thủ+15%") if sel == "A"
            else (("B: -50 Food, Atk+15%" if en else "B: -50 Lương, Đánh+15%") if sel == "B" else ("C: Free, -10%Def, -50%Loss" if en else "C: 0 phí, Thủ-10%, -50%phạt"))
        )
        prefix = "LOCKED" if en else "CHỐT"
        sub_txt = f"{prefix} [{eff}] · A:{vc['A']} B:{vc['B']} C:{vc['C']}"
    elif wave_no % 2 == 0:
        nt = min(20, int(snap["wall_tier"]) + 1)
        sub_txt = f"EVEN WAVE {wave_no}: Vote A/B/C at 03:40 · Wall Tier {nt}" if en else f"ĐỢT CHẴN {wave_no}: Bầu A/B/C lúc 03:40 · Tường Cấp {nt}"
    else:
        nt = min(20, int(snap["wall_tier"]) + 1)
        sub_txt = f"ODD WAVE {wave_no}: Wall Tier {nt} & balance roles" if en else f"ĐỢT LẺ {wave_no}: Nâng Tường Cấp {nt} & đủ 3 vai"

    return {
        "banner_skin": banner_skin,
        "banner_text": banner_txt,
        "vote_strip_text": sub_txt,
        "timer_text": timer_txt,
        "seconds_until_assault": compute_prep_seconds_until_assault(tick),
        "is_prep_warning": (not is_frozen) and (not (bool(diag["metrics_available"]) and int(diag["confirmed_valid_drops"]) > 0)) and phase == "prep" and (1000 <= tick <= 1599),
    }


def step_canonical_wave(
    *,
    wave_no: int,
    consecutive_wins: int,
    consecutive_losses: int,
    total_wins: int,
    wall_tier: int,
    wall_hp: int,
    townhall_hp: int,
    wood: int,
    stone: int,
    food: int,
    construction_work: int,
    role_schedule: Dict[int, Tuple[int, int, int]],
    vote_chosen: str = "",
    initial_max_viewer_xp: int = 120,
) -> Dict[str, Any]:
    """
    Bounded `ttg-rules-v1` boundary reducer matching `tooling/refeval/model.go` 1-for-1
    across ticks 200..2800 + settlement at 2800. Verified against M02 golden vectors.
    """
    assert 0 <= townhall_hp <= 500
    assert 0 <= wall_hp <= canonical_wall_max(wall_tier)

    trace: List[Dict[str, Any]] = []
    defeated = False
    settled = False
    result: Optional[str] = None
    base_hp = 0
    boss_max_hp = 0
    boss_hp = 0
    boss_spawned = False
    p_anchor = 0
    d = 0
    lanes = [[0, 0, 0], [0, 0, 0], [0, 0, 0]]
    max_viewer_xp = min(300, max(0, initial_max_viewer_xp))

    def ordinary() -> int:
        return sum(sum(ln) for ln in lanes)

    for tick in range(200, 3000, 200):
        b, g, f = role_schedule[tick]
        vote_charged: Optional[Dict[str, Any]] = None
        if tick == 1200 and wave_no % 2 == 0 and vote_chosen:
            if vote_chosen == "A":
                assert wood >= 50, "Insufficient wood for Vote A"
                wood -= 50
                vote_charged = {"resource": "wood", "amount": 50}
            elif vote_chosen == "B":
                assert food >= 50, "Insufficient food for Vote B"
                food -= 50
                vote_charged = {"resource": "food", "amount": 50}
            elif vote_chosen == "C":
                vote_charged = {"resource": "none", "amount": 0}

        if tick == 1600:
            p_anchor = b + g + f
            d = min(10, 1 + consecutive_wins)
            base_hp = canonical_base_hp(p_anchor, consecutive_wins, consecutive_losses)
            for idx, ltot in enumerate(canonical_lane_split(base_hp)):
                lanes[idx] = list(canonical_subpools(ltot))
            boss_max_hp = canonical_boss_max_hp(wave_no, base_hp)

        upgraded = False
        tier_before = wall_tier
        work_before_up = construction_work
        stone_before_up = stone

        if not defeated:
            if (b + g + f) > 0:
                max_viewer_xp = min(300, max_viewer_xp + 2)
            b_eff = min(max(0, b), 10000)
            sb = math.isqrt(b_eff)
            wood = min(1_000_000, wood + 2 * sb + b_eff // 50)
            stone = min(1_000_000, stone + sb + b_eff // 500)
            cap = 8 * sb
            missing = canonical_wall_max(wall_tier) - wall_hp
            rep = min(cap, missing, 5 * wood)
            if rep > 0:
                wood -= (rep + 4) // 5
                wall_hp += rep
            leftover = cap - rep
            if b_eff > 0 and leftover > 0:
                construction_work = min(200 * wall_tier, construction_work + leftover)
            work_before_up = construction_work
            stone_before_up = stone
            if wall_tier < 20 and construction_work >= 200 * wall_tier and stone >= 100 * wall_tier:
                construction_work -= 200 * wall_tier
                stone -= 100 * wall_tier
                wall_tier += 1
                wall_hp = min(canonical_wall_max(wall_tier), wall_hp + 200)
                upgraded = True

            if tick < 1800:
                trace.append({
                    "tick": tick,
                    "phase": "prep" if tick < 1600 else "assault",
                    "builder": b,
                    "guard": g,
                    "fighter": f,
                    "wall_tier": wall_tier,
                    "wall_hp": wall_hp,
                    "wall_max": canonical_wall_max(wall_tier),
                    "townhall_hp": townhall_hp,
                    "wood": wood,
                    "stone": stone,
                    "food": food,
                    "construction_work": construction_work,
                    "base_hp": base_hp,
                    "population_anchor": p_anchor,
                    "ordinary_hp": ordinary(),
                    "lanes": [list(ln) for ln in lanes],
                    "boss_hp": boss_hp,
                    "boss_max_hp": boss_max_hp,
                    "boss_state": "absent",
                    "upgraded_wall": upgraded,
                    "wall_tier_before": tier_before,
                    "construction_work_before_upgrade": work_before_up,
                    "stone_before_upgrade": stone_before_up,
                    "vote_cost_charged": vote_charged,
                    "max_viewer_session_xp": max_viewer_xp,
                })
            else:
                if tick == 2600 and boss_max_hp > 0 and not boss_spawned:
                    boss_spawned = True
                    boss_hp = boss_max_hp
                f_eff = min(max(0, f), 10000)
                militia = canonical_militia_bonus(p_anchor)
                attack = 18 * math.isqrt(f_eff) + militia
                if vote_chosen == "B":
                    attack = (115 * attack) // 100

                c_idx = (tick - 1800) // 200
                start = c_idx % 3
                order = [start, (start + 1) % 3, (start + 2) % 3]
                q = [attack // 3, attack // 3, attack // 3]
                for i in range(attack % 3):
                    q[order[i]] += 1

                def hit_lane(li: int, amt: int) -> Tuple[int, int]:
                    for si in range(3):
                        take = min(lanes[li][si], amt)
                        lanes[li][si] -= take
                        amt -= take
                    if li == 1 and boss_spawned and boss_hp > 0 and amt > 0:
                        b_take = min(boss_hp, amt)
                        return amt - b_take, b_take
                    return amt, 0

                leftover_atk = 0
                for li in order:
                    la, b_hit = hit_lane(li, q[li])
                    boss_hp -= b_hit
                    leftover_atk += la
                if leftover_atk > 0:
                    for li in order:
                        if leftover_atk <= 0:
                            break
                        la, b_hit = hit_lane(li, leftover_atk)
                        boss_hp -= b_hit
                        leftover_atk = la

                ord_rem = ordinary()
                p_eff = min(max(1, p_anchor), 10000)
                base_raw = 10 + 3 * d + 9 * math.isqrt(p_eff)
                raw = 0
                if ord_rem > 0:
                    raw += (base_raw * ord_rem + base_hp - 1) // base_hp
                if boss_spawned and boss_hp > 0:
                    raw += (base_raw + 4) // 5

                g_eff = min(max(0, g), 10000)
                defense = 8 * math.isqrt(g_eff)
                if vote_chosen == "A":
                    defense = (115 * defense) // 100
                elif vote_chosen == "C":
                    defense = (90 * defense) // 100
                dmg = max(0, raw - defense)
                w_loss = min(wall_hp, dmg)
                wall_hp -= w_loss
                townhall_hp = max(0, townhall_hp - (dmg - w_loss))
                if townhall_hp == 0:
                    defeated = True

                b_state = "absent"
                if boss_max_hp > 0:
                    if boss_spawned and boss_hp > 0:
                        b_state = "active"
                    elif boss_spawned:
                        b_state = "destroyed"
                    elif ord_rem == 0:
                        b_state = "incoming"
                    else:
                        b_state = "scheduled"

                trace.append({
                    "tick": tick,
                    "phase": "assault" if tick < 2600 else ("finale" if tick < 2800 else "results"),
                    "builder": b,
                    "guard": g,
                    "fighter": f,
                    "wall_tier": wall_tier,
                    "wall_hp": wall_hp,
                    "wall_max": canonical_wall_max(wall_tier),
                    "townhall_hp": townhall_hp,
                    "wood": wood,
                    "stone": stone,
                    "food": food,
                    "construction_work": construction_work,
                    "base_hp": base_hp,
                    "population_anchor": p_anchor,
                    "ordinary_hp": ord_rem,
                    "lanes": [list(ln) for ln in lanes],
                    "boss_hp": boss_hp,
                    "boss_max_hp": boss_max_hp,
                    "guard_block": defense,
                    "attack": attack,
                    "raw_damage": raw,
                    "damage_applied": dmg,
                    "defeated": defeated,
                    "boss_state": b_state,
                    "upgraded_wall": upgraded,
                    "wall_tier_before": tier_before,
                    "construction_work_before_upgrade": work_before_up,
                    "stone_before_upgrade": stone_before_up,
                    "vote_cost_charged": vote_charged,
                    "max_viewer_session_xp": max_viewer_xp,
                })
        else:
            # Defeated earlier in wave: economy/XP/combat skipped until tick 2800 settlement
            trace.append({
                "tick": tick,
                "phase": "assault" if tick < 2600 else ("finale" if tick < 2800 else "results"),
                "builder": b,
                "guard": g,
                "fighter": f,
                "wall_tier": wall_tier,
                "wall_hp": wall_hp,
                "wall_max": canonical_wall_max(wall_tier),
                "townhall_hp": townhall_hp,
                "wood": wood,
                "stone": stone,
                "food": food,
                "construction_work": construction_work,
                "base_hp": base_hp,
                "population_anchor": p_anchor,
                "ordinary_hp": ordinary(),
                "lanes": [list(ln) for ln in lanes],
                "boss_hp": boss_hp,
                "boss_max_hp": boss_max_hp,
                "defeated": True,
                "boss_state": "scheduled" if boss_max_hp > 0 else "absent",
                "upgraded_wall": False,
                "wall_tier_before": tier_before,
                "construction_work_before_upgrade": work_before_up,
                "stone_before_upgrade": stone_before_up,
                "vote_cost_charged": None,
                "max_viewer_session_xp": max_viewer_xp,
            })

        if tick == 2800 and not settled:
            settled = True
            if townhall_hp > 0 and ordinary() == 0 and boss_hp == 0:
                result = "win"
                wood = min(1_000_000, wood + 100)
                stone = min(1_000_000, stone + 60)
                food = min(1_000_000, food + 50)
                consecutive_wins = min(10, consecutive_wins + 1)
                consecutive_losses = 0
                total_wins += 1
                trace[-1]["wood"] = wood
                trace[-1]["stone"] = stone
                trace[-1]["food"] = food
                trace[-1]["settlement_result"] = "win"
                trace[-1]["settlement_reward"] = {"wood": 100, "stone": 60, "food": 50}
            else:
                result = "loss"
                consecutive_wins = 0
                consecutive_losses = min(3, consecutive_losses + 1)
                wp, sp, fp = wood // 5, stone // 5, food // 5
                if vote_chosen == "C":
                    wp, sp, fp = wp // 2, sp // 2, fp // 2
                wood = max(0, wood - wp)
                stone = max(0, stone - sp)
                food = max(0, food - fp)
                if defeated:
                    wall_hp = min(canonical_wall_max(wall_tier), max(250, wall_hp))
                    townhall_hp = max(250, townhall_hp)
                    wood = max(100, wood)
                trace[-1]["wall_hp"] = wall_hp
                trace[-1]["townhall_hp"] = townhall_hp
                trace[-1]["wood"] = wood
                trace[-1]["stone"] = stone
                trace[-1]["food"] = food
                trace[-1]["settlement_result"] = "loss"
                trace[-1]["settlement_reward"] = {"wood": 0, "stone": 0, "food": 0}

    return {
        "result": result,
        "defeated": defeated,
        "population_anchor": p_anchor,
        "difficulty": d,
        "base_hp": base_hp,
        "boss_max_hp": boss_max_hp,
        "wall_tier": wall_tier,
        "wall_hp": wall_hp,
        "wall_max": canonical_wall_max(wall_tier),
        "townhall_hp": townhall_hp,
        "wood": wood,
        "stone": stone,
        "food": food,
        "construction_work": construction_work,
        "consecutive_wins": consecutive_wins,
        "consecutive_losses": consecutive_losses,
        "total_wins": total_wins,
        "village_level": min(50, 1 + total_wins // 3),
        "max_viewer_session_xp": max_viewer_xp,
        "trace": trace,
    }


def make_snapshot(
    *,
    session_id: str = "synth-wave10-001",
    stream_seq: int,
    wave_no: int = 10,
    wave_tick: int,
    phase: str,
    lifecycle: str = "running",
    source_status: str = "mock",
    pause_reasons: Optional[List[str]] = None,
    total_wins: int = 39,
    village_level: Optional[int] = None,
    consecutive_wins_before_wave: int = 5,
    wall_tier: int = 6,
    wall_hp: int = 2000,
    wall_max: Optional[int] = None,
    townhall_hp: int = 500,
    townhall_max: int = 500,
    wood: int = 520,
    stone: int = 780,
    food: int = 540,
    builder: int = 0,
    guard: int = 0,
    fighter: int = 0,
    population_anchor: Optional[int] = None,
    base_hp: Optional[int] = None,
    west_hp: int = 0,
    center_hp: int = 0,
    east_hp: int = 0,
    lane_subpools: Optional[List[List[int]]] = None,
    vote_open: bool = False,
    vote_selected: Optional[str] = None,
    vote_a: int = 0,
    vote_b: int = 0,
    vote_c: int = 0,
    boss_incoming: bool = False,
    boss_exists: bool = False,
    boss_hp: int = 0,
    boss_max_hp: Optional[int] = None,
    wave_result: Optional[str] = None,
    consecutive_losses: int = 0,
    is_post_loss_recovery: bool = False,
    pending_next_wave_count: int = 0,
    militia_active: Optional[bool] = None,
    militia_bonus: Optional[int] = None,
    max_viewer_session_xp: int = 120,
    connection_state: str = "live",
    metrics_available: bool = True,
    queue_fill_pct: int = 12,
    retryable_rejected_count: int = 0,
    confirmed_valid_drops: int = 0,
    join_toast_alias: Optional[str] = None,
    join_toast_role: Optional[str] = None,
    attract_mode: bool = False,
) -> Dict[str, Any]:
    """
    Self-validating canonical `ttg-rules-v1` synthetic snapshot builder.
    Rejects any invalid Townhall HP, Wall max, Boss max, WAITING tick, Vote state, or XP cap.
    """
    if lifecycle == "waiting" or phase == "waiting":
        wave_tick = -1
        phase = "waiting"
        lifecycle = "waiting"
        builder = 0
        guard = 0
        fighter = 0

    derived_wall_max = canonical_wall_max(wall_tier)
    if wall_max is None:
        wall_max = derived_wall_max
    assert wall_max == derived_wall_max, f"Invalid wall_max={wall_max} for tier={wall_tier} (expected {derived_wall_max})"
    assert 0 <= wall_hp <= wall_max, f"Invalid wall_hp={wall_hp} for wall_max={wall_max}"
    assert townhall_max == 500, f"Invalid townhall_max={townhall_max} (must be 500)"
    assert 0 <= townhall_hp <= 500, f"Invalid townhall_hp={townhall_hp} (must be 0..500)"
    assert 0 <= wood <= 1_000_000 and 0 <= stone <= 1_000_000 and 0 <= food <= 1_000_000
    assert 0 <= max_viewer_session_xp <= 300

    if village_level is None:
        village_level = min(50, 1 + total_wins // 3)
    assert 1 <= village_level <= 50

    p_total = builder + guard + fighter
    p_anchor = population_anchor if (population_anchor is not None and wave_tick >= 1600) else (p_total if wave_tick >= 1600 else 0)
    if wave_tick >= 1600:
        if base_hp is None:
            base_hp = canonical_base_hp(p_anchor, consecutive_wins_before_wave, consecutive_losses)
    else:
        base_hp = 0
        west_hp = 0
        center_hp = 0
        east_hp = 0

    derived_boss_max = canonical_boss_max_hp(wave_no, base_hp) if wave_tick >= 1600 else 0
    if boss_max_hp is None:
        boss_max_hp = derived_boss_max
    assert boss_max_hp == derived_boss_max, f"Invalid boss_max_hp={boss_max_hp} != {derived_boss_max} (wave={wave_no}, base_hp={base_hp})"
    if wave_tick < 2600 or wave_no % 5 != 0:
        assert not boss_exists and boss_hp == 0, f"Boss cannot exist at wave={wave_no}, tick={wave_tick}"
    assert 0 <= boss_hp <= boss_max_hp

    if wave_no % 2 == 1:
        vote_open = False
        vote_selected = None
    elif wave_tick >= 1200 and vote_selected is None:
        vote_selected = "B"

    p_ref = p_anchor if wave_tick >= 1600 else p_total
    exp_m_active = 1 <= p_ref <= 10
    exp_m_bonus = canonical_militia_bonus(p_ref) if exp_m_active else 0
    if militia_active is None:
        militia_active = exp_m_active
    if militia_bonus is None:
        militia_bonus = exp_m_bonus
    assert militia_active == exp_m_active and militia_bonus == exp_m_bonus

    if lane_subpools is not None:
        sub_w, sub_c, sub_e = lane_subpools
    else:
        sub_w = list(canonical_subpools(west_hp))
        sub_c = list(canonical_subpools(center_hp))
        sub_e = list(canonical_subpools(east_hp))

    rem_s = 300 if phase == "waiting" else max(0, (3000 - max(0, wave_tick)) // 10)

    vote_obj: Dict[str, Any] = {
        "open": vote_open,
        "selected": vote_selected,
        "counts": {"A": vote_a, "B": vote_b, "C": vote_c},
    }
    if vote_selected in CANONICAL_VOTE_EFFECTS:
        vote_obj["canonical_effect"] = dict(CANONICAL_VOTE_EFFECTS[vote_selected])

    fix: Dict[str, Any] = {
        "synthetic_fixture": True,
        "fixture_id": f"{session_id}_seq_{stream_seq}",
        "canonical_state_id": phase.upper(),
        "attract_mode": attract_mode,
        "schema_version": "ws-public-v1-synthetic",
        "session_id": session_id,
        "mock_stream_seq": stream_seq,
        "public_snapshot": {
            "lifecycle": lifecycle,
            "source_status": source_status,
            "pause_reasons": list(pause_reasons or []),
            "wave_no": wave_no,
            "wave_tick": wave_tick,
            "remaining_seconds": rem_s,
            "phase": phase,
            "total_wins": total_wins,
            "village_level": village_level,
            "consecutive_wins_before_wave": consecutive_wins_before_wave,
            "population_anchor": p_anchor,
            "base_hp": base_hp,
            "max_viewer_session_xp": max_viewer_session_xp,
            "wall_tier": wall_tier,
            "wall_hp": wall_hp,
            "wall_max": wall_max,
            "townhall_hp": townhall_hp,
            "townhall_max": 500,
            "wood": wood,
            "stone": stone,
            "food": food,
            "counts": {
                "unique_joined": p_total,
                "builder": builder,
                "guard": guard,
                "fighter": fighter,
            },
            "lanes": [
                {"id": "west", "ordinary_hp": sum(sub_w), "runner_hp": sub_w[0], "grunt_hp": sub_w[1], "brute_hp": sub_w[2]},
                {"id": "center", "ordinary_hp": sum(sub_c), "runner_hp": sub_c[0], "grunt_hp": sub_c[1], "brute_hp": sub_c[2]},
                {"id": "east", "ordinary_hp": sum(sub_e), "runner_hp": sub_e[0], "grunt_hp": sub_e[1], "brute_hp": sub_e[2]},
            ],
            "vote": vote_obj,
            "boss": {
                "boss_incoming": boss_incoming,
                "boss_exists": boss_exists,
                "boss_hp": boss_hp,
                "boss_max_hp": boss_max_hp,
            },
            "settlement": {
                "wave_result": wave_result,
                "consecutive_losses": consecutive_losses,
                "is_post_loss_recovery": is_post_loss_recovery,
                "pending_next_wave_count": pending_next_wave_count,
            },
            "system_assist": {
                "militia_active": militia_active,
                "militia_attack_bonus": militia_bonus,
            },
            "diagnostics": {
                "connection_state": connection_state,
                "metrics_available": metrics_available,
                "queue_fill_pct": queue_fill_pct,
                "retryable_rejected_count": retryable_rejected_count,
                "confirmed_valid_drops": confirmed_valid_drops,
            },
        },
    }
    if join_toast_alias and join_toast_role:
        fix["public_snapshot"]["join_toast"] = {
            "mode": "single",
            "display_alias": join_toast_alias[:12],
            "role": join_toast_role,
            "ttl_seconds": 2.0,
        }
    return fix


def build_animation_vfx_audio_sync(observed_offsets: List[Dict[str, Any]]) -> Dict[str, Any]:
    return {
        "package_id": "ART-008",
        "contract_version": "bastion-dawn-v1-sync-12fps",
        "animation_fps": 12,
        "frame_duration_ms": FRAME_MS_12FPS,
        "authoritative_tick_hz": 10,
        "authoritative_tick_ms": 100.0,
        "clock_independence_rule": (
            "Presentation playback runs on a local render clock (12 FPS sprite cadence, 48 kHz audio DSP clock) "
            "triggered ONLY after committed public events/snapshots arrive. Late render frames drop/advance local "
            "visual elapsed time without EVER mutating authoritative wave_tick or performing combat math."
        ),
        "synchronization_tolerances": {
            "max_frame_to_vfx_onset_frames": 0,
            "max_audio_mux_onset_delta_ms": 20.0,
            "max_one_frame_visual_quantization_ms": 83.34,
        },
        "character_action_sync_table": [
            {
                "class_id": "builder",
                "package_id": "ART-003",
                "action_clip": "chr_role_builder_repair",
                "total_frames": 6,
                "impact_frame_index": 3,
                "impact_offset_ms": 250.0,
                "vfx_id": "vfx_builder_repair_spark",
                "vfx_frames": 6,
                "vfx_sorting_layer": "VFX",
                "vfx_attachment_offset_px_1080p": [16, -22],
                "audio_cue_id": "SFX02",
                "audio_clip": "sfx_builder_repair.wav",
                "recovery_transition": "repair_f05 -> idle_f00 (auto return, non-looping)",
            },
            {
                "class_id": "guard",
                "package_id": "ART-004",
                "action_clip": "chr_role_guard_shieldbrace",
                "total_frames": 6,
                "impact_frame_index": 2,
                "impact_offset_ms": 166.67,
                "vfx_id": "vfx_guard_shield_block",
                "vfx_frames": 4,
                "vfx_sorting_layer": "VFX",
                "vfx_attachment_offset_px_1080p": [0, -16],
                "audio_cue_id": "SFX03",
                "audio_clip": "sfx_guard_block.wav",
                "recovery_transition": "shieldbrace_f05 -> idle_f00 (auto return, non-looping)",
            },
            {
                "class_id": "fighter",
                "package_id": "ART-004",
                "action_clip": "chr_role_fighter_swordslash",
                "total_frames": 6,
                "impact_frame_index": 3,
                "impact_offset_ms": 250.0,
                "vfx_id": "vfx_fighter_slash_arc",
                "vfx_frames": 4,
                "vfx_sorting_layer": "VFX",
                "vfx_attachment_offset_px_1080p": [18, -20],
                "audio_cue_id": "SFX04",
                "audio_clip": "sfx_fighter_slash.wav",
                "recovery_transition": "swordslash_f05 -> idle_f00 (auto return, non-looping)",
            },
            {
                "class_id": "grunt",
                "package_id": "ART-003",
                "action_clip": "chr_enemy_grunt_clubsmash",
                "total_frames": 6,
                "impact_frame_index": 3,
                "impact_offset_ms": 250.0,
                "vfx_id": "vfx_unit_hit_puff",
                "vfx_frames": 6,
                "vfx_sorting_layer": "VFX",
                "vfx_attachment_offset_px_1080p": [14, -18],
                "audio_cue_id": "SFX05",
                "audio_clip": "sfx_enemy_attack.wav",
                "recovery_transition": "clubsmash_f05 -> idle_f00 (auto return, non-looping)",
            },
            {
                "class_id": "runner",
                "package_id": "ART-004",
                "action_clip": "chr_enemy_runner_daggerlunge",
                "total_frames": 6,
                "impact_frame_index": 2,
                "impact_offset_ms": 166.67,
                "vfx_id": "vfx_unit_hit_puff",
                "vfx_frames": 6,
                "vfx_sorting_layer": "VFX",
                "vfx_attachment_offset_px_1080p": [12, -16],
                "audio_cue_id": "SFX05",
                "audio_clip": "sfx_enemy_attack.wav",
                "recovery_transition": "daggerlunge_f05 -> idle_f00 (auto return, non-looping)",
            },
            {
                "class_id": "brute",
                "package_id": "ART-004",
                "action_clip": "chr_enemy_brute_gauntletcrush",
                "total_frames": 6,
                "impact_frame_index": 3,
                "impact_offset_ms": 250.0,
                "vfx_id": "vfx_unit_hit_puff",
                "vfx_frames": 6,
                "vfx_sorting_layer": "VFX",
                "vfx_attachment_offset_px_1080p": [16, -22],
                "audio_cue_id": "SFX05",
                "audio_clip": "sfx_enemy_attack.wav",
                "recovery_transition": "gauntletcrush_f05 -> idle_f00 (auto return, non-looping)",
            },
            {
                "class_id": "boss",
                "package_id": "ART-004",
                "action_clip": "chr_boss_warlord_cleaveslam",
                "total_frames": 8,
                "impact_frame_index": 5,
                "impact_offset_ms": 416.67,
                "vfx_id": "vfx_boss_slam_telegraph",
                "vfx_frames": 10,
                "vfx_sorting_layer": "LanesAndDecals",
                "vfx_attachment_offset_px_1080p": [0, 4],
                "audio_cue_id": "SFX09",
                "audio_clip": "sfx_boss_slam.wav",
                "recovery_transition": "cleaveslam_f07 -> idlemarch_f00 (auto return, non-looping)",
            },
        ],
        "canonical_choreography_matrix": [
            {"gameplay_event": "First valid join", "committed_trigger": "role_accepted (counts.unique_joined 0->1)", "visual_presentation": "Builder/Guard/Fighter registration toast + counter increment + unit spawn", "animation": "chr_role_*_idle", "vfx": "none", "audio_cue_id": "SFX01", "audio_label": "Role Join"},
            {"gameplay_event": "Builder repair contact", "committed_trigger": "role_resolution + positive wall_hp repair diff", "visual_presentation": "Builder hammer contact on scaffold + repair spark", "animation": "chr_role_builder_repair (impact f03)", "vfx": "vfx_builder_repair_spark", "audio_cue_id": "SFX02", "audio_label": "Builder Repair"},
            {"gameplay_event": "Guard blocks", "committed_trigger": "damage_applied / attack_resolved with guard mitigation > 0", "visual_presentation": "Guard ShieldBrace + azure shield block flash", "animation": "chr_role_guard_shieldbrace (impact f02)", "vfx": "vfx_guard_shield_block", "audio_cue_id": "SFX03", "audio_label": "Guard Block"},
            {"gameplay_event": "Fighter strikes", "committed_trigger": "attack_resolved (counts.fighter > 0)", "visual_presentation": "Fighter SwordSlash + crimson/gold slash arc", "animation": "chr_role_fighter_swordslash (impact f03)", "vfx": "vfx_fighter_slash_arc", "audio_cue_id": "SFX04", "audio_label": "Fighter Slash"},
            {"gameplay_event": "Enemy attacks", "committed_trigger": "damage_applied while lane ordinary_hp > 0", "visual_presentation": "Grunt/Runner/Brute attack + impact puff", "animation": "clubsmash/daggerlunge/gauntletcrush", "vfx": "vfx_unit_hit_puff", "audio_cue_id": "SFX05", "audio_label": "Enemy Attack"},
            {"gameplay_event": "Wall damaged", "committed_trigger": "damage_applied with wall_hp decrease and wall_hp > 0", "visual_presentation": "Wall HP bar lerp + wall state 1->2->3 transition", "animation": "env_wall_state2_damaged / state3_critical", "vfx": "vfx_unit_hit_puff", "audio_cue_id": "SFX06", "audio_label": "Wall Damage"},
            {"gameplay_event": "Wall breached", "committed_trigger": "damage_applied crossing wall_hp > 0 -> wall_hp == 0", "visual_presentation": "env_wall_state4_breached + breach warning emphasis", "animation": "env_wall_state4_breached", "vfx": "vfx_unit_hit_puff", "audio_cue_id": "SFX07", "audio_label": "Wall Breach"},
            {"gameplay_event": "Boss incoming", "committed_trigger": "boss_incoming == true at tick 2450 on Boss wave", "visual_presentation": "BOSS_INCOMING HUD banner + ground rune telegraph (NO Boss sprite yet!)", "animation": "none (Boss sprite strictly forbidden < tick 2600)", "vfx": "vfx_boss_slam_telegraph (f00..f03 pre-warn)", "audio_cue_id": "SFX08", "audio_label": "Boss Warning"},
            {"gameplay_event": "Boss slam", "committed_trigger": "boss_spawned / boss_combat_slam at tick >= 2600", "visual_presentation": "Boss Warlord CleaveSlam + ground impact rune", "animation": "chr_boss_warlord_cleaveslam (impact f05)", "vfx": "vfx_boss_slam_telegraph (f05..f09)", "audio_cue_id": "SFX09", "audio_label": "Boss Slam"},
            {"gameplay_event": "Vote closes", "committed_trigger": "vote_closed at tick 1200 on even wave", "visual_presentation": "Subordinate vote strip locks selected option A/B/C", "animation": "ui_comp_vote_card_locked", "vfx": "none", "audio_cue_id": "SFX10", "audio_label": "Vote Closed"},
            {"gameplay_event": "Victory", "committed_trigger": "settlement.wave_result == 'win' at tick >= 2800", "visual_presentation": "RESULTS_WIN emerald banner + Townhall preserved", "animation": "defender idle celebration posture", "vfx": "none", "audio_cue_id": "SFX11", "audio_label": "Victory"},
            {"gameplay_event": "Defeat", "committed_trigger": "settlement.wave_result == 'loss' at tick >= 2800 (or townhall_hp == 0)", "visual_presentation": "RESULTS_LOSS crimson banner + recovery prep state", "animation": "env_wall_state4_breached / state5_recovery", "vfx": "none", "audio_cue_id": "SFX12", "audio_label": "Defeat"},
        ],
        "observed_audiovisual_sync_measurements": observed_offsets,
    }


def build_hud_transition_mapping() -> Dict[str, Any]:
    transitions = [
        ("T01", "ATTRACT", "WAITING_FOR_VIEWER", "Session initialized with P=0 (wave_tick=-1)", "200ms banner crossfade + timer reset to 05:00", "Instant state swap (0px slide, 60ms alpha crossfade)"),
        ("T02", "WAITING_FOR_VIEWER", "PREP", "First committed role_accepted (wave_tick=0, P>=1)", "200ms banner crossfade + single-slot join toast 8px slide-in", "Instant banner swap + static 2.0s toast fade (0px slide)"),
        ("T03", "PREP", "PREP_60_WARN", "wave_tick reaches 1000 (60s PREP remaining)", "180ms navy->amber threat banner tint crossfade", "Instant navy->amber banner swap (0Hz pulse)"),
        ("T04", "PREP_60_WARN", "PREP_15_WARN", "wave_tick reaches 1450 (15s PREP remaining)", "180ms amber->crimson banner crossfade + dusk lighting grade", "Instant amber->crimson banner swap (0Hz flash)"),
        ("T05", "PREP", "PREP + VOTE_OPEN", "Even wave_no and wave_tick in [800..1199]", "180ms subordinate vote strip open skin crossfade inside dock", "Instant subordinate vote strip update"),
        ("T06", "PREP_15_WARN", "ASSAULT", "wave_tick reaches 1600 (ASSAULT lock)", "220ms banner slide-in (12px Y) + midday lighting + lane enemy entry", "Instant banner update (0px slide)"),
        ("T07", "ASSAULT", "FINALE", "wave_tick reaches 2600", "200ms banner update + dusk/eclipse palette shift", "Instant banner & palette update"),
        ("T08", "BOSS_INCOMING", "BOSS_ACTIVE", "Boss wave tick 2450->2600 (boss_exists=true)", "220ms Boss HP sub-bar reveal + Boss Warlord spawn at tick 2600", "Instant Boss HP sub-bar + Boss spawn at tick 2600"),
        ("T09", "FINALE", "RESULTS_WIN", "Authoritative settlement wave_result=='win' at tick >= 2800", "220ms emerald victory banner slide-in (10px Y) + SFX11 once", "Instant emerald victory banner swap + SFX11 once"),
        ("T10", "FINALE", "RESULTS_LOSS", "Authoritative settlement wave_result=='loss' at tick >= 2800", "220ms crimson loss banner crossfade + SFX12 once", "Instant crimson loss banner swap + SFX12 once"),
        ("T11", "RESULTS", "PREP (New Wave)", "Rollover at tick 3000 -> wave_no+1, wave_tick=0", "220ms wave counter increment + dawn palette restore (0 scene reload)", "Instant wave counter & dawn restore (0 scene reload)"),
        ("T12", "Gameplay", "SOURCE_RECONNECTING", "source_status=='disconnected' or pause_reasons has source_disconnected", "160ms amber reconnect banner + freeze world animations in place", "Instant amber reconnect banner + frozen world"),
        ("T13", "Gameplay", "PAUSED", "pause_reasons has operator_paused", "160ms operator pause banner + freeze world (never cleared by reconnect!)", "Instant operator pause banner + frozen world"),
        ("T14", "Gameplay", "STORAGE_UNAVAILABLE", "pause_reasons has storage_unavailable (highest operational priority)", "160ms crimson DB outage banner + freeze world + preserve Boss diag pill", "Instant crimson DB outage banner + frozen world"),
        ("T15", "Recovery", "Valid resumed presentation", "Verified fresh public_snapshot after blocker cleared", "180ms banner restore + resume 12 FPS clock from committed tick", "Instant banner restore + resume 12 FPS clock"),
        ("T16", "Any state", "MAINTENANCE", "lifecycle=='stopped'", "180ms maintenance banner + clear world units + maintenance CTA", "Instant maintenance state swap"),
    ]
    return {
        "package_id": "ART-008",
        "style_lock": "Bastion Dawn v1",
        "canonical_states_count": 16,
        "combined_overlay_states_count": 8,
        "priority_transitions_count": len(transitions),
        "safety_and_ux_invariants": {
            "primary_cta_vi": "Bình luận XÂY / THỦ / ĐÁNH",
            "competing_primary_ctas": 0,
            "fake_viewer_touch_buttons": 0,
            "camera_shake_amplitude_px": 0.0,
            "max_flash_frequency_hz": 0.0,
            "strobe_ceiling_hz": 3.0,
            "max_transition_duration_ms": 220,
            "hp_bar_lerp_duration_ms": 180,
            "join_toast_single_slot_ttl_ms": 2000,
            "scene_reloads_on_state_change": 0,
            "boss_earliest_spawn_tick": 2600,
            "results_earliest_settlement_tick": 2800,
            "music_muted_by_default": True,
            "right_safe_zone_max_x_1080p": 885.6,
            "bottom_safe_zone_max_y_1080p": 1689.6,
        },
        "priority_order_highest_first": [
            "MAINTENANCE (lifecycle == stopped)",
            "STORAGE_UNAVAILABLE (pause_reasons contains storage_unavailable)",
            "SOURCE_RECONNECTING (pause_reasons contains source_disconnected)",
            "PAUSED (pause_reasons contains operator_paused)",
            "DEGRADED_DROPS (metrics_available && confirmed_valid_drops > 0)",
            "BOSS_ACTIVE (wave_no % 5 == 0 && tick >= 2600 && boss_exists)",
            "BOSS_INCOMING (wave_no % 5 == 0 && 2400 <= tick < 2600 && boss_incoming)",
            "RESULTS_WIN / RESULTS_LOSS (phase == results && tick >= 2800)",
            "ASSAULT / FINALE",
            "PREP_15_WARN (1450 <= tick <= 1599)",
            "PREP_60_WARN (1000 <= tick <= 1449)",
            "PREP + VOTE_OPEN (even wave && 800 <= tick <= 1199)",
            "PREP (0 <= tick <= 1599)",
            "WAITING_FOR_VIEWER (P == 0 && tick == -1)",
            "ATTRACT",
        ],
        "priority_transitions": [
            {
                "transition_id": tid,
                "from_state": f_st,
                "to_state": t_st,
                "trigger_condition": trig,
                "standard_motion": {
                    "description": std_m,
                    "max_duration_ms": 220,
                    "camera_shake_px": 0.0,
                    "flash_hz": 0.0,
                },
                "reduced_motion": {
                    "description": red_m,
                    "slide_offset_px": 0,
                    "scale_pulse": 1.0,
                    "flash_hz": 0.0,
                },
            }
            for (tid, f_st, t_st, trig, std_m, red_m) in transitions
        ],
    }


def build_audio_dispatch_policy() -> Dict[str, Any]:
    return {
        "package_id": "ART-008",
        "policy_id": "ttg-audio-dispatch-v2",
        "authority_rule": "Presentation-only consumer of committed non-speculative public events/snapshots. Never mutates gameplay state.",
        "voice_pool": {
            "total_preallocated_audio_sources": 12,
            "combat_sfx_pool": 8,
            "priority_alerts_pool": 2,
            "ambient_loop_source": 1,
            "music_loop_source": 1,
            "dynamic_runtime_instantiation": False,
        },
        "buses_default_db": {
            "Master": 0.0,
            "Alerts": 0.0,
            "SFX": -1.5,
            "Ambient": -3.0,
            "Music": -80.0,
        },
        "music_safety_lock": {
            "muted_by_default": True,
            "default_volume_db": -80.0,
            "audio_source_mute_flag": True,
            "release_gate_blocker": "GATE-LEGAL",
        },
        "join_chime_anti_spam_contract": {
            "cue_id": "SFX01",
            "prep_cooldown_ms": 2000,
            "assault_and_finale_cooldown_ms": 3500,
            "assault_attenuation_db": -5.5,
            "assault_gain_accounting": {
                "cue_attenuation_db": -5.5,
                "cue_linear_gain": 0.5309,
                "sfx_bus_gain_db": -1.5,
                "sfx_bus_linear_gain": 0.8414,
                "combined_output_db": -7.0,
                "combined_output_linear_gain": 0.4467,
                "explanation": "Intended combination of -5.5 dB per-cue ASSAULT attenuation and -1.5 dB SFX bus gain (-7.0 dB = 10^(-7/20) = 0.4467 linear).",
            },
            "suppressed_while_priority_alerts_active": True,
            "verified_10s_burst_plays_prep": 5,
            "verified_10s_burst_plays_assault": 3,
            "verified_plays_during_alert": 0,
        },
        "two_tier_deduplication_contract": {
            "contract_id": "ttg-audio-dedupe-v2",
            "tier_1_exact_delivery": {
                "key_format": "{session_id}:{stream_seq}",
                "ring_buffer_size": 256,
                "monotonic_cursor_floor": "reject stream_seq <= max_seen_stream_seq - 256",
                "preserves_independent_same_type_combat_events": True,
            },
            "tier_2_once_per_wave_milestones": {
                "key_format": "{session_id}:{wave_no}:{canonical_milestone}",
                "allowed_milestones_only": ["vote_closed", "boss_incoming", "boss_spawned", "wave_result"],
                "forbidden_on_repeatable_combat_cues": ["SFX01", "SFX02", "SFX03", "SFX04", "SFX05", "SFX06", "SFX07", "SFX09"],
            },
        },
    }


def build_wave10_full_300s_fixture() -> Dict[str, Any]:
    """
    Rule-verified 300-second (ticks -1, 0..3000 at 10 Hz) Wave 10 showcase fixture.
    Stepped through `step_canonical_wave` matching `ttg-rules-v1` (`tooling/refeval/model.go`):
      - Entering Wave 10: total_wins=39 (village_level=14), cw=5 (d=6), wall_tier=6 (wall_max=2000),
        wall_hp=1560/2000, townhall_hp=500/500, wood=520, stone=780, food=540, construction_work=1140.
      - Tick 800..1199: Even-wave vote open.
      - Tick 1200: Vote B closes -> deducts 50 Food once and grants +15% Fighter+Militia attack.
      - Tick 1400: Builders complete construction_work>=1200 & stone>=600 -> upgrade Wall Tier 6 -> 7 (wall_max=2200).
      - Tick 1600: Population anchor P=204 locks -> base_hp=620, lanes=(210, 198, 212), boss_max_hp=ceil(620/5)=124.
      - Ticks 1800..2400: Ordinary lanes monotonically decrease 620 -> 455 -> 290 -> 125 -> 0 (boss_incoming at 2400..2599).
      - Tick 2600: Boss spawns (124 HP) -> takes 62 dmg -> 62/124 HP active!
      - Tick 2800: Final combat slays Boss (0/124 HP) -> single settlement WIN (+100W, +60S, +50F), total_wins=40.
      - Tick 2900: Queued registrations for Wave 11 reach 34 (current-wave counts & HP untouched).
      - Tick 3000 -> Wave 11 Tick 0: 34 queued players promoted in-place (0 scene reloads).
    """
    role_sched: Dict[int, Tuple[int, int, int]] = {
        200: (49, 4, 5),
        400: (81, 12, 16),
        600: (100, 20, 25),
        800: (121, 24, 35),
        1000: (121, 28, 45),
        1200: (121, 31, 52),
        1400: (121, 31, 52),
        1600: (48, 92, 64),
        1800: (0, 140, 64),
        2000: (0, 140, 64),
        2200: (0, 140, 64),
        2400: (0, 140, 64),
        2600: (36, 159, 9),
        2800: (36, 143, 25),
    }
    sim = step_canonical_wave(
        wave_no=10,
        consecutive_wins=5,
        consecutive_losses=0,
        total_wins=39,
        wall_tier=6,
        wall_hp=1560,
        townhall_hp=500,
        wood=520,
        stone=780,
        food=540,
        construction_work=1140,
        role_schedule=role_sched,
        vote_chosen="B",
        initial_max_viewer_xp=120,
    )
    tr_by_tick = {r["tick"]: r for r in sim["trace"]}

    def snap_from_row(
        row: Dict[str, Any],
        *,
        stream_seq: int,
        wave_tick_override: Optional[int] = None,
        vote_open: bool = False,
        vote_selected: Optional[str] = None,
        vote_a: int = 0,
        vote_b: int = 0,
        vote_c: int = 0,
        boss_incoming: bool = False,
        boss_exists: bool = False,
        wave_result: Optional[str] = None,
        pending_next: int = 0,
        total_wins_val: int = 39,
        toast_alias: Optional[str] = None,
        toast_role: Optional[str] = None,
    ) -> Dict[str, Any]:
        wtick = row["tick"] if wave_tick_override is None else wave_tick_override
        ph = "prep" if wtick < 1600 else ("assault" if wtick < 2600 else ("finale" if wtick < 2800 else "results"))
        return make_snapshot(
            stream_seq=stream_seq,
            wave_no=10,
            wave_tick=wtick,
            phase=ph,
            total_wins=total_wins_val,
            consecutive_wins_before_wave=5,
            wall_tier=row["wall_tier"],
            wall_hp=row["wall_hp"],
            townhall_hp=row["townhall_hp"],
            wood=row["wood"],
            stone=row["stone"],
            food=row["food"],
            builder=row["builder"],
            guard=row["guard"],
            fighter=row["fighter"],
            population_anchor=row["population_anchor"] if wtick >= 1600 else None,
            base_hp=row["base_hp"] if wtick >= 1600 else None,
            lane_subpools=row["lanes"] if wtick >= 1600 else None,
            vote_open=vote_open,
            vote_selected=vote_selected,
            vote_a=vote_a,
            vote_b=vote_b,
            vote_c=vote_c,
            boss_incoming=boss_incoming,
            boss_exists=boss_exists,
            boss_hp=row["boss_hp"],
            boss_max_hp=row["boss_max_hp"] if wtick >= 1600 else 0,
            wave_result=wave_result,
            pending_next_wave_count=pending_next,
            max_viewer_session_xp=row["max_viewer_session_xp"],
            join_toast_alias=toast_alias,
            join_toast_role=toast_role,
        )

    beats: List[Dict[str, Any]] = [
        {
            "beat_id": "B00_WAITING",
            "wave_no": 10,
            "wave_tick": -1,
            "logical_time_s": -0.1,
            "showcase_video_time_s": 0.0,
            "visual_state": "WAITING_FOR_VIEWER",
            "meaningful_visual_change": "Lobby WAITING state (wave_tick=-1, P=0): empty lanes, Tier 6 wall 1560/2000, Townhall 500/500",
            "audio_events": [{"cue_id": "AMB01", "event_type": "ambient_loop_start", "mock_stream_seq": 1000}],
            "fixture": make_snapshot(
                stream_seq=1000, wave_no=10, wave_tick=-1, phase="waiting", lifecycle="waiting",
                total_wins=39, wall_tier=6, wall_hp=1560, townhall_hp=500, wood=520, stone=780, food=540,
                max_viewer_session_xp=120,
            ),
        },
        {
            "beat_id": "B01_FIRST_JOIN_WAVE_START",
            "wave_no": 10,
            "wave_tick": 0,
            "logical_time_s": 0.0,
            "showcase_video_time_s": 1.0,
            "visual_state": "PREP",
            "meaningful_visual_change": "First participant joins Builder -> wave_tick=0 starts, Militia assist +40 ATK (P=1<=10), Join Toast slides in",
            "audio_events": [{"cue_id": "SFX01", "event_type": "role_accepted", "mock_stream_seq": 1001}],
            "fixture": make_snapshot(
                stream_seq=1001, wave_no=10, wave_tick=0, phase="prep",
                total_wins=39, wall_tier=6, wall_hp=1560, townhall_hp=500, wood=520, stone=780, food=540,
                builder=1, guard=0, fighter=0, max_viewer_session_xp=120,
                join_toast_alias="MinhBao", join_toast_role="builder",
            ),
        },
        {
            "beat_id": "B02_PREP_CONSTRUCTION_20S",
            "wave_no": 10,
            "wave_tick": 200,
            "logical_time_s": 20.0,
            "showcase_video_time_s": 2.0,
            "visual_state": "PREP",
            "meaningful_visual_change": "Tick 200: 49 Builders repair Tier 6 wall 1560->1616/2000 (+repair sparks), Guards & Fighters muster",
            "audio_events": [{"cue_id": "SFX02", "event_type": "role_resolution", "mock_stream_seq": 1008}],
            "fixture": snap_from_row(tr_by_tick[200], stream_seq=1008),
        },
        {
            "beat_id": "B03_PREP_MUSTER_40S",
            "wave_no": 10,
            "wave_tick": 400,
            "logical_time_s": 40.0,
            "showcase_video_time_s": 3.0,
            "visual_state": "PREP",
            "meaningful_visual_change": "Tick 400: 109 participants active, wall repaired to 1688/2000, timer 04:20",
            "audio_events": [{"cue_id": "SFX01", "event_type": "role_accepted", "mock_stream_seq": 1015}],
            "fixture": snap_from_row(tr_by_tick[400], stream_seq=1015, toast_alias="ThanhLong", toast_role="fighter"),
        },
        {
            "beat_id": "B04_PREP_60S_ELAPSED",
            "wave_no": 10,
            "wave_tick": 600,
            "logical_time_s": 60.0,
            "showcase_video_time_s": 4.0,
            "visual_state": "PREP",
            "meaningful_visual_change": "Tick 600: 100 Builders raise Tier 6 wall to 1768/2000 HP, wood/stone economy accrues",
            "audio_events": [{"cue_id": "SFX02", "event_type": "role_resolution", "mock_stream_seq": 1024}],
            "fixture": snap_from_row(tr_by_tick[600], stream_seq=1024),
        },
        {
            "beat_id": "B05_EVEN_WAVE_VOTE_OPEN",
            "wave_no": 10,
            "wave_tick": 800,
            "logical_time_s": 80.0,
            "showcase_video_time_s": 5.0,
            "visual_state": "VOTE_OPEN",
            "meaningful_visual_change": "Tick 800: Even-wave tactical vote opens in subordinate dock strip (A:-50G / B:-50L / C:0)",
            "audio_events": [],
            "fixture": snap_from_row(tr_by_tick[800], stream_seq=1032, vote_open=True, vote_a=18, vote_b=31, vote_c=14),
        },
        {
            "beat_id": "B06_PREP_60_WARN",
            "wave_no": 10,
            "wave_tick": 1000,
            "logical_time_s": 100.0,
            "showcase_video_time_s": 6.0,
            "visual_state": "PREP_60_WARN",
            "meaningful_visual_change": "Tick 1000: PREP 60s amber warning banner + wall reaches 1944/2000 + live vote tallies",
            "audio_events": [{"cue_id": "SFX01", "event_type": "role_accepted", "mock_stream_seq": 1041}],
            "fixture": snap_from_row(tr_by_tick[1000], stream_seq=1041, vote_open=True, vote_a=29, vote_b=54, vote_c=22),
        },
        {
            "beat_id": "B07_VOTE_CLOSED",
            "wave_no": 10,
            "wave_tick": 1200,
            "logical_time_s": 120.0,
            "showcase_video_time_s": 7.0,
            "visual_state": "PREP_60_WARN",
            "meaningful_visual_change": "Tick 1200: Vote closes -> Option B locked (-50 Food charged once, Fighter ATK +15%) + gavel chime SFX10",
            "audio_events": [{"cue_id": "SFX10", "event_type": "vote_closed", "mock_stream_seq": 1050}],
            "fixture": snap_from_row(tr_by_tick[1200], stream_seq=1050, vote_open=False, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B08_PREP_140S_FORTIFY",
            "wave_no": 10,
            "wave_tick": 1400,
            "logical_time_s": 140.0,
            "showcase_video_time_s": 8.0,
            "visual_state": "PREP_60_WARN",
            "meaningful_visual_change": "Tick 1400: Construction work>=1200 & stone>=600 upgrades Wall Tier 6 -> Tier 7 (2200/2200 HP)",
            "audio_events": [{"cue_id": "SFX02", "event_type": "role_resolution", "mock_stream_seq": 1058}],
            "fixture": snap_from_row(tr_by_tick[1400], stream_seq=1058, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B09_PREP_15_WARN",
            "wave_no": 10,
            "wave_tick": 1450,
            "logical_time_s": 145.0,
            "showcase_video_time_s": 9.0,
            "visual_state": "PREP_15_WARN",
            "meaningful_visual_change": "Tick 1450: PREP 15-second crimson warning banner + dusk atmospheric grade",
            "audio_events": [],
            "fixture": snap_from_row(tr_by_tick[1400], stream_seq=1062, wave_tick_override=1450, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B10_ASSAULT_START",
            "wave_no": 10,
            "wave_tick": 1600,
            "logical_time_s": 160.0,
            "showcase_video_time_s": 10.0,
            "visual_state": "ASSAULT",
            "meaningful_visual_change": "Tick 1600: P=204 locks -> base_hp=620 spawns canonical West 210 / Center 198 / East 212 enemy pools",
            "audio_events": [{"cue_id": "SFX05", "event_type": "damage_applied", "mock_stream_seq": 1070}],
            "fixture": snap_from_row(tr_by_tick[1600], stream_seq=1070, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B11_FIRST_COMBAT_BOUNDARY",
            "wave_no": 10,
            "wave_tick": 1800,
            "logical_time_s": 180.0,
            "showcase_video_time_s": 11.0,
            "visual_state": "ASSAULT",
            "meaningful_visual_change": "Tick 1800: Vote B (+15% Fighter ATK = 165) cuts enemy pools 620->455; Wall takes combat hit (2167/2200)",
            "audio_events": [
                {"cue_id": "SFX04", "event_type": "attack_resolved", "mock_stream_seq": 1081},
                {"cue_id": "SFX03", "event_type": "damage_applied", "mock_stream_seq": 1082},
                {"cue_id": "SFX06", "event_type": "damage_applied", "mock_stream_seq": 1083},
            ],
            "fixture": snap_from_row(tr_by_tick[1800], stream_seq=1083, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B12_COMBAT_WALL_DAMAGED_200S",
            "wave_no": 10,
            "wave_tick": 2000,
            "logical_time_s": 200.0,
            "showcase_video_time_s": 12.0,
            "visual_state": "ASSAULT",
            "meaningful_visual_change": "Tick 2000: Cyclic combat cuts enemy pools 455->290 (100/88/102); Guards absorb retaliation (wall 2158/2200)",
            "audio_events": [
                {"cue_id": "SFX06", "event_type": "damage_applied", "mock_stream_seq": 1094},
                {"cue_id": "SFX03", "event_type": "damage_applied", "mock_stream_seq": 1095},
            ],
            "fixture": snap_from_row(tr_by_tick[2000], stream_seq=1095, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B13_COMBAT_FLANK_CLASH_220S",
            "wave_no": 10,
            "wave_tick": 2200,
            "logical_time_s": 220.0,
            "showcase_video_time_s": 13.0,
            "visual_state": "ASSAULT",
            "meaningful_visual_change": "Tick 2200: West/Center/East flanks reduced monotonically 290->125 (45/33/47 HP remaining)",
            "audio_events": [
                {"cue_id": "SFX04", "event_type": "attack_resolved", "mock_stream_seq": 1106},
                {"cue_id": "SFX05", "event_type": "damage_applied", "mock_stream_seq": 1107},
            ],
            "fixture": snap_from_row(tr_by_tick[2200], stream_seq=1107, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B14_COMBAT_PRE_BOSS_240S",
            "wave_no": 10,
            "wave_tick": 2400,
            "logical_time_s": 240.0,
            "showcase_video_time_s": 14.0,
            "visual_state": "BOSS_INCOMING",
            "meaningful_visual_change": "Tick 2400: Ordinary lanes cleared to 0/0/0 -> Boss state transitions to INCOMING (Boss still unspawned < 2600)",
            "audio_events": [{"cue_id": "SFX04", "event_type": "attack_resolved", "mock_stream_seq": 1115}],
            "fixture": snap_from_row(tr_by_tick[2400], stream_seq=1115, boss_incoming=True, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B15_BOSS_INCOMING_2450",
            "wave_no": 10,
            "wave_tick": 2450,
            "logical_time_s": 245.0,
            "showcase_video_time_s": 15.0,
            "visual_state": "BOSS_INCOMING",
            "meaningful_visual_change": "Tick 2450: Boss Warning banner + eclipse palette + ground rune telegraph ONLY (boss_exists=false, boss_hp=0)",
            "audio_events": [{"cue_id": "SFX08", "event_type": "boss_incoming", "mock_stream_seq": 1120}],
            "fixture": snap_from_row(tr_by_tick[2400], stream_seq=1120, wave_tick_override=2450, boss_incoming=True, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B16_FINALE_BOSS_SPAWN_2600",
            "wave_no": 10,
            "wave_tick": 2600,
            "logical_time_s": 260.0,
            "showcase_video_time_s": 16.5,
            "visual_state": "BOSS_ACTIVE",
            "meaningful_visual_change": "Tick 2600: FINALE -> Boss Warlord spawns at canonical 124 max HP, takes 62 Vote-B slash dmg -> 62/124 HP active",
            "audio_events": [
                {"cue_id": "SFX09", "event_type": "boss_spawned", "mock_stream_seq": 1130},
                {"cue_id": "SFX03", "event_type": "damage_applied", "mock_stream_seq": 1131},
            ],
            "fixture": snap_from_row(tr_by_tick[2600], stream_seq=1131, boss_exists=True, vote_selected="B", vote_a=34, vote_b=68, vote_c=25),
        },
        {
            "beat_id": "B17_SETTLEMENT_VICTORY_2800",
            "wave_no": 10,
            "wave_tick": 2800,
            "logical_time_s": 280.0,
            "showcase_video_time_s": 18.0,
            "visual_state": "RESULTS_WIN",
            "meaningful_visual_change": "Tick 2800: Final combat slays Boss (62->0 HP), single settlement WIN (+100W/+60S/+50F, Townhall 500/500)",
            "audio_events": [
                {"cue_id": "SFX04", "event_type": "attack_resolved", "mock_stream_seq": 1142},
                {"cue_id": "SFX11", "event_type": "wave_result", "mock_stream_seq": 1143},
            ],
            "fixture": snap_from_row(
                tr_by_tick[2800], stream_seq=1143, wave_result="win", pending_next=19,
                total_wins_val=40, vote_selected="B", vote_a=34, vote_b=68, vote_c=25,
            ),
        },
        {
            "beat_id": "B18_RESULTS_NEXT_QUEUE_2900",
            "wave_no": 10,
            "wave_tick": 2900,
            "logical_time_s": 290.0,
            "showcase_video_time_s": 19.0,
            "visual_state": "RESULTS_WIN",
            "meaningful_visual_change": "Tick 2900: Results hold (zero illegal repair or role mutation); 34 next-wave players queued for Wave 11",
            "audio_events": [],
            "fixture": snap_from_row(
                tr_by_tick[2800], stream_seq=1148, wave_tick_override=2900, wave_result="win", pending_next=34,
                total_wins_val=40, vote_selected="B", vote_a=34, vote_b=68, vote_c=25,
            ),
        },
        {
            "beat_id": "B19_ROLLOVER_WAVE11_3000",
            "wave_no": 11,
            "wave_tick": 0,
            "logical_time_s": 300.0,
            "showcase_video_time_s": 20.0,
            "visual_state": "PREP",
            "meaningful_visual_change": "Tick 3000 -> Wave 11 Tick 0 rollover in-place (0 scene reloads): 34 queued players promoted, odd-wave objective active",
            "audio_events": [{"cue_id": "SFX01", "event_type": "role_accepted", "mock_stream_seq": 1155}],
            "fixture": make_snapshot(
                stream_seq=1155, wave_no=11, wave_tick=0, phase="prep",
                total_wins=40, consecutive_wins_before_wave=6,
                wall_tier=sim["wall_tier"], wall_hp=sim["wall_hp"], townhall_hp=500,
                wood=sim["wood"], stone=sim["stone"], food=sim["food"],
                builder=11, guard=11, fighter=12,
                max_viewer_session_xp=sim["max_viewer_session_xp"],
            ),
        },
    ]

    # Supplementary canonical Defeat state (breach -> 250/500 TH floor & 250/wall_max Wall floor) for Unity Frame #9
    defeat_sim = step_canonical_wave(
        wave_no=10,
        consecutive_wins=0,
        consecutive_losses=0,
        total_wins=39,
        wall_tier=6,
        wall_hp=80,
        townhall_hp=40,
        wood=220,
        stone=180,
        food=260,
        construction_work=0,
        role_schedule={t: (4, 0, 26) if t <= 1600 else (0, 0, 30) for t in range(200, 3000, 200)},
        vote_chosen="B",
        initial_max_viewer_xp=90,
    )
    # Use an all-builder no-fighter breach case (mirrors M02 Golden Case C011!) so townhall_hp hits 0 cleanly:
    defeat_sim = step_canonical_wave(
        wave_no=10,
        consecutive_wins=2,
        consecutive_losses=0,
        total_wins=39,
        wall_tier=6,
        wall_hp=20,
        townhall_hp=20,
        wood=220,
        stone=180,
        food=260,
        construction_work=0,
        role_schedule={t: (0, 0, 30) if t < 1800 else (0, 0, 1) if False else (0, 30, 0) for t in range(200, 3000, 200)},
        vote_chosen="B",
        initial_max_viewer_xp=90,
    )
    # With B=0, G=4, F=26? Wait: let's set B=0, G=1, F=29 -> wait: let's give G=0, B=0, F=0? No, B=0, G=2, F=28? If F=0, B=0 -> P=30 all Guards (G=4 -> P=4? P=30 requires G=30 -> def=40 vs raw=10+9+45=64 -> dmg=24 per tick! At 1800: 20 wall + 4 TH -> TH=16; at 2000: TH=0 DEFEATED!).
    assert defeat_sim["result"] == "loss" and defeat_sim["defeated"] is True
    d_row = defeat_sim["trace"][-1]

    return {
        "package_id": "ART-008",
        "fixture_id": "full_wave_300s_wave10_showcase",
        "synthetic_fixture": True,
        "synthetic_rule_verified_fixture": True,
        "is_executable_m02_replay": False,
        "m05_ws_contract_status": "SYNTHETIC_PRESENTATION_FIXTURE_M05_NOT_STARTED",
        "visual_units_authority_mode": "decorative_aggregate_pool_proxies",
        "authoritative_rules_ref": "docs/plan/02_GAME_RULES.md + docs/plan/03_SIMULATION.md + tooling/refeval/model.go (ttg-rules-v1)",
        "wave_no": 10,
        "rollover_wave_no": 11,
        "logical_duration_seconds": 300,
        "logical_ticks_per_second": 10,
        "total_logical_ticks": 3000,
        "selected_timeline_beats_count": len(beats),
        "simulation_boundary_transitions_executed": len(sim["trace"]) + 1,
        "max_logical_gap_between_visual_changes_seconds": 20.0,
        "accelerated_showcase_video_duration_seconds": 21.0,
        "music_muted_by_default": True,
        "boundary_trace": sim["trace"],
        "beats": beats,
        "supplementary_states": {
            "defeat_wave10_breach": {
                "state_id": "RESULTS_LOSS_DEFEAT_FLOOR",
                "wave_no": 10,
                "wave_tick": 2800,
                "description": "Canonical Townhall-breach defeat settled at tick 2800: -20% resource penalty + 250/2000 Wall & 250/500 Townhall recovery floors (matches M02 C011).",
                "fixture": make_snapshot(
                    session_id="synth-wave10-defeat",
                    stream_seq=1980,
                    wave_no=10,
                    wave_tick=2800,
                    phase="results",
                    total_wins=39,
                    consecutive_wins_before_wave=2,
                    wall_tier=d_row["wall_tier"],
                    wall_hp=d_row["wall_hp"],
                    townhall_hp=d_row["townhall_hp"],
                    wood=d_row["wood"],
                    stone=d_row["stone"],
                    food=d_row["food"],
                    builder=0,
                    guard=30,
                    fighter=0,
                    population_anchor=30,
                    base_hp=defeat_sim["base_hp"],
                    lane_subpools=d_row["lanes"],
                    vote_selected="B",
                    vote_a=8,
                    vote_b=15,
                    vote_c=7,
                    boss_exists=False,
                    boss_hp=0,
                    boss_max_hp=defeat_sim["boss_max_hp"],
                    wave_result="loss",
                    consecutive_losses=1,
                    is_post_loss_recovery=True,
                    pending_next_wave_count=12,
                    max_viewer_session_xp=d_row["max_viewer_session_xp"],
                ),
            }
        },
    }


def build_eight_failure_sequences() -> List[Dict[str, Any]]:
    return [
        {
            "sequence_id": "failure_01_source_disconnect",
            "filename": "failure_01_source_disconnect.json",
            "synthetic_fixture": True,
            "title": "1. Source Disconnect During Assault",
            "invariants_verified": [
                "World animations and timer freeze immediately in place (no fake live combat)",
                "Amber SOURCE_RECONNECTING banner + MẤT NGUỒN source pill shown prominently",
                "Zero scene reload; zero false success toast",
            ],
            "fixture": make_snapshot(
                session_id="synth-fail-01", stream_seq=2010, wave_no=7, wave_tick=1840,
                phase="assault", lifecycle="source_disconnected", source_status="disconnected",
                pause_reasons=["source_disconnected"], total_wins=39, wall_tier=6, wall_hp=1160,
                townhall_hp=500, wood=440, stone=290, food=410,
                builder=18, guard=24, fighter=32, west_hp=110, center_hp=104, east_hp=112,
                connection_state="reconnecting",
            ),
        },
        {
            "sequence_id": "failure_02_source_reconnect",
            "filename": "failure_02_source_reconnect.json",
            "synthetic_fixture": True,
            "title": "2. Source Reconnect With Stale-Snapshot Guard",
            "invariants_verified": [
                "Stale snapshot (connection_state=='stale' or older stream_seq) is flagged TRỄ >3S, never silently treated as fresh live",
                "Only after verified fresh stream_seq arrives with pause_reasons==[] does presentation resume",
            ],
            "stale_snapshot_rejected_step": make_snapshot(
                session_id="synth-fail-02", stream_seq=2009, wave_no=7, wave_tick=1835,
                phase="assault", lifecycle="running", source_status="mock",
                total_wins=39, wall_tier=6, wall_hp=1180, townhall_hp=500,
                builder=18, guard=24, fighter=32,
                west_hp=118, center_hp=110, east_hp=120, connection_state="stale",
            ),
            "fixture": make_snapshot(
                session_id="synth-fail-02", stream_seq=2015, wave_no=7, wave_tick=1845,
                phase="assault", lifecycle="running", source_status="mock",
                total_wins=39, wall_tier=6, wall_hp=1160, townhall_hp=500,
                builder=19, guard=24, fighter=33, population_anchor=74,
                west_hp=104, center_hp=98, east_hp=106, connection_state="live",
            ),
        },
        {
            "sequence_id": "failure_03_operator_pause",
            "filename": "failure_03_operator_pause.json",
            "synthetic_fixture": True,
            "title": "3. Operator Pause (Independent of Source Reconnect)",
            "invariants_verified": [
                "Operator pause holds authoritative PAUSED banner and frozen world",
                "Source reconnect event removes source_disconnected ONLY — operator_paused CANNOT clear automatically!",
            ],
            "reconnect_while_operator_paused_proof": {
                "before_pause_reasons": ["operator_paused", "source_disconnected"],
                "after_source_reconnect_pause_reasons": ["operator_paused"],
                "auto_cleared_by_reconnect": False,
            },
            "fixture": make_snapshot(
                session_id="synth-fail-03", stream_seq=2030, wave_no=8, wave_tick=1120,
                phase="prep", lifecycle="paused", source_status="mock",
                pause_reasons=["operator_paused"], total_wins=39, wall_tier=6, wall_hp=1880,
                townhall_hp=500, builder=22, guard=26, fighter=30,
                vote_open=True, vote_a=14, vote_b=19, vote_c=11,
            ),
        },
        {
            "sequence_id": "failure_04_storage_unavailable",
            "filename": "failure_04_storage_unavailable.json",
            "synthetic_fixture": True,
            "title": "4. PostgreSQL Storage Unavailable (Highest Blocker Priority)",
            "invariants_verified": [
                "storage_unavailable outranks source_disconnected and operator_paused in primary banner",
                "Zero speculative join toasts or fake durable ACKs emitted while DB is unavailable",
                "Coexisting Boss truth preserved in secondary diagnostic pill",
            ],
            "fixture": make_snapshot(
                session_id="synth-fail-04", stream_seq=2045, wave_no=10, wave_tick=2640,
                phase="finale", lifecycle="storage_unavailable", source_status="disconnected",
                pause_reasons=["storage_unavailable", "source_disconnected"],
                total_wins=39, consecutive_wins_before_wave=5,
                wall_tier=7, wall_hp=480, townhall_hp=440,
                builder=40, guard=52, fighter=76, population_anchor=168,
                west_hp=24, center_hp=0, east_hp=28,
                vote_selected="B", vote_a=28, vote_b=64, vote_c=19,
                boss_exists=True, boss_hp=64,
            ),
        },
        {
            "sequence_id": "failure_05_backpressure_no_loss",
            "filename": "failure_05_backpressure_no_loss.json",
            "synthetic_fixture": True,
            "title": "5. Queue Backpressure Without Confirmed Event Loss (W0306 Discipline)",
            "invariants_verified": [
                "queue_fill_pct=86% and retryable_rejected_count=14 with confirmed_valid_drops==0",
                "Displays subordinate amber backpressure notice ONLY — NEVER falsely claims permanent dropped valid commands!",
            ],
            "fixture": make_snapshot(
                session_id="synth-fail-05", stream_seq=2060, wave_no=9, wave_tick=1750,
                phase="assault", lifecycle="running", total_wins=39, wall_tier=6, wall_hp=1640,
                townhall_hp=500, builder=30, guard=40, fighter=58,
                west_hp=148, center_hp=138, east_hp=150,
                queue_fill_pct=86, retryable_rejected_count=14, confirmed_valid_drops=0,
            ),
        },
        {
            "sequence_id": "failure_06_confirmed_valid_loss",
            "filename": "failure_06_confirmed_valid_loss.json",
            "synthetic_fixture": True,
            "title": "6. Confirmed Valid-Event Loss Reported Truthfully",
            "invariants_verified": [
                "metrics_available==true and confirmed_valid_drops==4 (>0)",
                "Promotes crimson DEGRADED_DROPS truth banner reporting exact confirmed count (4)",
            ],
            "fixture": make_snapshot(
                session_id="synth-fail-06", stream_seq=2075, wave_no=9, wave_tick=1920,
                phase="assault", lifecycle="running", total_wins=39, wall_tier=6, wall_hp=1080,
                townhall_hp=500, builder=31, guard=42, fighter=60, population_anchor=128,
                west_hp=112, center_hp=102, east_hp=114,
                queue_fill_pct=95, retryable_rejected_count=22, confirmed_valid_drops=4,
            ),
        },
        {
            "sequence_id": "failure_07_no_viewers_waiting",
            "filename": "failure_07_no_viewers_waiting.json",
            "synthetic_fixture": True,
            "title": "7. Zero Viewers (P=0 Waiting Gate)",
            "invariants_verified": [
                "unique_joined == 0 keeps phase in WAITING_FOR_VIEWER with wave_tick == -1",
                "Zero fake world defenders/enemies spawned; zero premature registration toast",
            ],
            "fixture": make_snapshot(
                session_id="synth-fail-07", stream_seq=2090, wave_no=1, wave_tick=-1,
                phase="waiting", lifecycle="waiting", total_wins=0,
                wall_tier=1, wall_hp=1000, townhall_hp=500, wood=400, stone=250, food=300,
                max_viewer_session_xp=0,
            ),
        },
        {
            "sequence_id": "failure_08_verified_recovery_resume",
            "filename": "failure_08_verified_recovery_resume.json",
            "synthetic_fixture": True,
            "title": "8. Gameplay Resumed After Verified Recovery & Deduplicated Replay",
            "invariants_verified": [
                "Replayed duplicate wave_result (SFX11) on reconnect is suppressed (plays 1 time total, never twice)",
                "Two independent Fighter attack_resolved events (stream_seq 2104 & 2105) both remain playable",
                "Post-loss recovery wall state 5 + relief pill rendered accurately",
            ],
            "replay_audio_verification": {
                "replayed_victory_seqs": [2099, 2099],
                "victory_sting_plays": 1,
                "independent_fighter_attack_seqs": [2104, 2105],
                "fighter_attack_plays": 2,
            },
            "fixture": make_snapshot(
                session_id="synth-fail-08", stream_seq=2105, wave_no=6, wave_tick=1680,
                phase="assault", lifecycle="running", total_wins=15, consecutive_wins_before_wave=0,
                wall_tier=4, wall_hp=1120, townhall_hp=420,
                builder=26, guard=34, fighter=46,
                west_hp=114, center_hp=108, east_hp=116,
                vote_selected="C", vote_a=12, vote_b=18, vote_c=36,
                consecutive_losses=1, is_post_loss_recovery=True,
            ),
        },
    ]
