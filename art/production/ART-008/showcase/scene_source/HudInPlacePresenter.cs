using System;
using UnityEngine;
using UnityEngine.SceneManagement;
using UnityEngine.UI;

namespace TTGQLDL.Showcase.UI
{
    /// <summary>
    /// Presentation-only data transfer view derived strictly from a single validated
    /// synthetic/replay public_snapshot revision. Never computes gameplay authority.
    /// </summary>
    [Serializable]
    public sealed class SyntheticCommittedView
    {
        public string stateId;
        public string fixtureId;
        public string visualStateId;
        public string lifecycle;
        public string sourceStatus;
        public string[] pauseReasons;
        public int waveNo;
        public int waveTick;
        public int remainingSeconds;
        public string phase;
        public int villageLevel;
        public int wallTier;
        public int wallHp;
        public int wallMax;
        public int townhallHp;
        public int townhallMax;
        public int wood;
        public int stone;
        public int food;
        public int builderCount;
        public int guardCount;
        public int fighterCount;
        public int uniqueJoined;
        public int westHp;
        public int centerHp;
        public int eastHp;
        public bool bossIncoming;
        public bool bossExists;
        public int bossHp;
        public int bossMaxHp;
        public bool voteOpen;
        public string voteSelected;
        public int voteA;
        public int voteB;
        public int voteC;
        public string waveResult;
        public bool isPostLossRecovery;
        public int pendingNextWaveCount;
        public bool militiaActive;
        public int militiaAttackBonus;
        public string connectionState;
        public bool metricsAvailable;
        public int queueFillPct;
        public int retryableRejectedCount;
        public int confirmedValidDrops;
        public bool showJoinToast;
        public string joinToastAlias;
        public string joinToastRole;
        public string outputPng1080p;
        public string outputPng720p;
        public string outputPng360p;
    }

    /// <summary>
    /// In-place 16-state HUD controller for the isolated ART-008 / ART-009 vertical slice.
    /// Updates 100% of visible counters, HP bars, canonical Vote A/B/C descriptions, banners,
    /// diagnostic pills, and synthetic watermarks from a single <see cref="SyntheticCommittedView"/>
    /// revision WITHOUT reloading the Unity scene.
    /// </summary>
    public sealed class HudInPlacePresenter : MonoBehaviour
    {
        public Canvas rootCanvas;
        public CanvasGroup threatBannerGroup;
        public RectTransform threatBannerRect;
        public Image threatBannerSkin;
        public Image sourceStatusSkin;
        public Image wallHpFill;
        public Image castleHpFill;
        public GameObject bossHpBarRoot;
        public Image bossHpFill;
        public CanvasGroup secondaryDiagGroup;
        public Image secondaryDiagSkin;
        public Image voteStripSkin;
        public Image ctaBannerSkin;
        public CanvasGroup joinToastGroup;
        public CanvasGroup watermarkGroup;

        public Sprite skinNavy;
        public Sprite skinOk;
        public Sprite skinWarn;
        public Sprite skinErr;
        public Sprite skinVoteOpen;
        public Sprite skinVoteLocked;
        public Sprite skinHpHealthy;
        public Sprite skinHpWarn;
        public Sprite skinHpCrit;

        public Text wallHpLabel;
        public Text castleHpLabel;
        public Text waveLabel;
        public Text timerLabel;
        public Text sourceStatusLabel;
        public Text woodLabel;
        public Text stoneLabel;
        public Text foodLabel;
        public Text threatBannerLabel;
        public Text secondaryDiagLabel;
        public Text joinToastLabel;
        public Text voteStripLabel;
        public Text primaryCtaLabel;
        public Text roleBuilderLabel;
        public Text roleGuardLabel;
        public Text roleFighterLabel;
        public Text watermarkLabel;

        private Scene _activeSceneBefore;
        private float _displayedWallRatio = 1.0f;
        private float _targetWallRatio = 1.0f;
        private float _bannerTransitionT = 1.0f;
        private Vector2 _bannerRestAnchoredPos;

        public bool ReducedMotion { get; set; }
        public string CurrentVisualState { get; private set; } = "WAITING_FOR_VIEWER";
        public string CurrentFixtureId { get; private set; } = "";
        public int InPlaceStateTransitionsCount { get; private set; }
        public int SceneReloadCount { get; private set; }

        public void Initialize()
        {
            _activeSceneBefore = SceneManager.GetActiveScene();
            SceneManager.sceneLoaded -= OnSceneLoaded;
            SceneManager.sceneLoaded += OnSceneLoaded;
            if (threatBannerRect != null)
            {
                _bannerRestAnchoredPos = threatBannerRect.anchoredPosition;
            }
        }

        private void OnDestroy()
        {
            SceneManager.sceneLoaded -= OnSceneLoaded;
        }

        private void OnSceneLoaded(Scene scene, LoadSceneMode mode)
        {
            SceneReloadCount++;
        }

        private static bool HasReason(string[] reasons, string target)
        {
            if (reasons == null) return false;
            for (int i = 0; i < reasons.Length; i++)
            {
                if (string.Equals(reasons[i], target, StringComparison.Ordinal)) return true;
            }
            return false;
        }

        private Sprite PickHpFillSprite(float ratio)
        {
            if (ratio < 0.25f && skinHpCrit != null) return skinHpCrit;
            if (ratio < 0.60f && skinHpWarn != null) return skinHpWarn;
            return skinHpHealthy != null ? skinHpHealthy : (wallHpFill != null ? wallHpFill.sprite : null);
        }

        public void ApplyCommittedViewInPlace(SyntheticCommittedView v, string locale = "vi-VN")
        {
            if (v == null) throw new ArgumentNullException(nameof(v));
            if (v.townhallMax != 500 || v.townhallHp < 0 || v.townhallHp > 500)
                throw new InvalidOperationException($"Invalid Townhall HP {v.townhallHp}/{v.townhallMax}; canonical bound is 0..500.");
            int expectedWallMax = 1000 + 200 * (v.wallTier - 1);
            if (v.wallTier < 1 || v.wallTier > 20 || v.wallMax != expectedWallMax || v.wallHp < 0 || v.wallHp > v.wallMax)
                throw new InvalidOperationException($"Invalid Wall HP {v.wallHp}/{v.wallMax} for tier {v.wallTier} (expected max {expectedWallMax}).");

            if (!SceneManager.GetActiveScene().Equals(_activeSceneBefore))
            {
                SceneReloadCount++;
            }

            bool en = string.Equals(locale, "en-US", StringComparison.OrdinalIgnoreCase);
            bool stateChanged = !string.Equals(CurrentVisualState, v.visualStateId, StringComparison.Ordinal) ||
                                !string.Equals(CurrentFixtureId, v.fixtureId, StringComparison.Ordinal);
            CurrentVisualState = v.visualStateId ?? v.phase ?? "PREP";
            CurrentFixtureId = v.fixtureId ?? "";

            float wallRatio = v.wallMax > 0 ? Mathf.Clamp01((float)v.wallHp / v.wallMax) : 0f;
            float castleRatio = Mathf.Clamp01((float)v.townhallHp / 500f);
            _targetWallRatio = wallRatio;

            if (wallHpFill != null)
            {
                wallHpFill.sprite = PickHpFillSprite(wallRatio);
            }
            if (castleHpFill != null)
            {
                castleHpFill.sprite = PickHpFillSprite(castleRatio);
                castleHpFill.rectTransform.anchorMax = new Vector2(Mathf.Max(0.02f, castleRatio * 0.98f), 0.85f);
            }

            // 1. Exact HP, Timer, Wave, Resource & Role Counters from this single fixture revision
            if (wallHpLabel != null)
                wallHpLabel.text = en ? $"WALL {v.wallHp}/{v.wallMax}" : $"TƯỜNG {v.wallHp}/{v.wallMax}";
            if (castleHpLabel != null)
                castleHpLabel.text = en ? $"KEEP {v.townhallHp}/{v.townhallMax}" : $"CHÍNH {v.townhallHp}/{v.townhallMax}";

            int rem = Mathf.Max(0, v.remainingSeconds);
            string mmss = $"{rem / 60:D2}:{rem % 60:D2}";
            if (v.lifecycle == "stopped")
            {
                if (waveLabel != null) waveLabel.text = en ? "MAINT." : "BẢO TRÌ";
                if (timerLabel != null) timerLabel.text = en ? "TIME --:--" : "GIỜ --:--";
            }
            else
            {
                if (waveLabel != null) waveLabel.text = en ? $"WAVE {v.waveNo}" : $"ĐỢT {v.waveNo}";
                if (timerLabel != null) timerLabel.text = en ? $"TIME {mmss}" : $"GIỜ {mmss}";
            }

            if (woodLabel != null) woodLabel.text = en ? $"WOOD {v.wood}" : $"GỖ {v.wood}";
            if (stoneLabel != null) stoneLabel.text = en ? $"STONE {v.stone}" : $"ĐÁ {v.stone}";
            if (foodLabel != null) foodLabel.text = en ? $"FOOD {v.food}" : $"LƯƠNG {v.food}";

            if (roleBuilderLabel != null) roleBuilderLabel.text = en ? $"BUILD {v.builderCount}" : $"XÂY {v.builderCount}";
            if (roleGuardLabel != null) roleGuardLabel.text = en ? $"DEFEND {v.guardCount}" : $"THỦ {v.guardCount}";
            if (roleFighterLabel != null) roleFighterLabel.text = en ? $"FIGHT {v.fighterCount}" : $"ĐÁNH {v.fighterCount}";

            // 2. Source Status Pill (strict blocker priority: storage_unavailable > source_disconnected > operator_paused)
            bool dbErr = HasReason(v.pauseReasons, "storage_unavailable") || v.lifecycle == "storage_unavailable";
            bool srcDisc = HasReason(v.pauseReasons, "source_disconnected") || v.sourceStatus == "disconnected";
            bool opPause = HasReason(v.pauseReasons, "operator_paused") || v.lifecycle == "paused";
            bool isFrozen = dbErr || srcDisc || opPause || v.lifecycle == "waiting" || v.lifecycle == "stopped";

            if (sourceStatusSkin != null && sourceStatusLabel != null)
            {
                if (dbErr)
                {
                    sourceStatusSkin.sprite = skinErr;
                    sourceStatusLabel.text = en ? "DB HALT" : "LỖI DB";
                }
                else if (srcDisc)
                {
                    sourceStatusSkin.sprite = skinWarn;
                    sourceStatusLabel.text = en ? "NO SOURCE" : "MẤT NGUỒN";
                }
                else if (opPause)
                {
                    sourceStatusSkin.sprite = skinWarn;
                    sourceStatusLabel.text = en ? "PAUSED" : "TẠM DỪNG";
                }
                else if (v.connectionState == "stale")
                {
                    sourceStatusSkin.sprite = skinWarn;
                    sourceStatusLabel.text = en ? "STALE >3S" : "TRỄ >3S";
                }
                else if (v.sourceStatus == "connected")
                {
                    sourceStatusSkin.sprite = skinOk;
                    sourceStatusLabel.text = en ? "SOURCE OK" : "NGUỒN OK";
                }
                else if (v.sourceStatus == "replay")
                {
                    sourceStatusSkin.sprite = skinWarn;
                    sourceStatusLabel.text = "REPLAY";
                }
                else
                {
                    sourceStatusSkin.sprite = skinWarn;
                    sourceStatusLabel.text = "MOCK";
                }
            }

            // 3. Primary Threat Banner & Embedded Boss HP Bar
            bool showBossBar = false;
            Sprite bannerSkin = skinNavy;
            string bannerTxt;
            if (v.lifecycle == "stopped")
            {
                bannerSkin = skinErr;
                bannerTxt = en ? "MAINTENANCE · SESSION STOPPED" : "PHIÊN DỪNG / BẢO TRÌ · HẸN GẶP LẠI";
            }
            else if (dbErr)
            {
                bannerSkin = skinErr;
                bannerTxt = en ? "STORAGE ERROR · SAFE HALT ACTIVE" : "LỖI LƯU TRỮ DB · TẠM DỪNG AN TOÀN";
            }
            else if (srcDisc)
            {
                bannerSkin = skinWarn;
                bannerTxt = en ? "SOURCE LOST · CLOCK SAFELY FROZEN" : "MẤT KẾT NỐI NGUỒN · ĐỒNG HỒ DỪNG";
            }
            else if (opPause)
            {
                bannerSkin = skinWarn;
                bannerTxt = en ? "OPERATOR PAUSED · CLOCK FROZEN" : "TẠM DỪNG ĐIỀU HÀNH · ĐỒNG HỒ DỪNG";
            }
            else if (v.metricsAvailable && v.confirmedValidDrops > 0)
            {
                bannerSkin = skinErr;
                bannerTxt = en ? $"OVERLOAD · VALID DROPS ({v.confirmedValidDrops})" : $"QUÁ TẢI · LỆNH HỢP LỆ RƠI ({v.confirmedValidDrops})";
            }
            else if (v.phase == "waiting" || v.waveTick < 0)
            {
                bannerSkin = skinNavy;
                bannerTxt = en ? "WAITING (P=0) · CLOCK FROZEN 05:00" : "CHỜ NGƯỜI CHƠI (P=0) · DỪNG 05:00";
            }
            else if (v.phase == "prep")
            {
                if (v.waveTick >= 1450)
                {
                    bannerSkin = skinErr;
                    bannerTxt = en ? "URGENT: ENEMIES IN 15 SECONDS!" : "KHẨN CẤP: QUÁI TỚI TRONG 15 GIÂY!";
                }
                else if (v.waveTick >= 1000)
                {
                    bannerSkin = skinWarn;
                    bannerTxt = en ? "WARNING: 60s UNTIL ENEMY ASSAULT!" : "CẢNH BÁO: CÒN 60 GIÂY QUÁI TỚI!";
                }
                else if (v.voteOpen && (v.waveNo % 2 == 0) && v.waveTick >= 800 && v.waveTick <= 1199)
                {
                    bannerSkin = skinNavy;
                    bannerTxt = en ? "EVEN WAVE VOTE · TYPE A / B / C" : "ĐỢT CHẴN BẦU CHỌN · GÕ A / B / C";
                }
                else
                {
                    bannerSkin = skinNavy;
                    bannerTxt = en ? "PREP 0..160s · REPAIR & RALLY" : "CHUẨN BỊ 0..160s · GIA CỐ & TẬP HỢP";
                }
            }
            else if (v.phase == "assault")
            {
                if (v.bossIncoming && v.waveTick < 2600)
                {
                    bannerSkin = skinErr;
                    bannerTxt = en ? "LANES CLEAR · BOSS AT 260s!" : "QUÁI HẠ · CẢNH BÁO BOSS GIÂY 260!";
                }
                else if ((v.westHp + v.centerHp + v.eastHp) == 0 && !v.bossIncoming)
                {
                    bannerSkin = skinOk;
                    bannerTxt = en ? "LANES CLEAR · AWAITING SETTLEMENT" : "SẠCH QUÁI THƯỜNG · CHỜ CHỐT ĐỢT";
                }
                else
                {
                    bannerSkin = skinWarn;
                    bannerTxt = en ? $"ASSAULT · W:{v.westHp} · C:{v.centerHp} · E:{v.eastHp}" : $"GIAO TRANH · T:{v.westHp} · G:{v.centerHp} · Đ:{v.eastHp}";
                }
            }
            else if (v.phase == "finale")
            {
                if (v.bossExists && v.waveTick >= 2600)
                {
                    bannerSkin = skinErr;
                    bannerTxt = en
                        ? $"BOSS {v.bossHp}/{v.bossMaxHp} · W:{v.westHp} · E:{v.eastHp}"
                        : $"BOSS {v.bossHp}/{v.bossMaxHp} · TÂY {v.westHp} · ĐÔNG {v.eastHp}";
                    showBossBar = true;
                }
                else
                {
                    bannerSkin = skinWarn;
                    bannerTxt = en ? $"FINALE · W:{v.westHp} · C:{v.centerHp} · E:{v.eastHp}" : $"CUỐI ĐỢT · T:{v.westHp} · G:{v.centerHp} · Đ:{v.eastHp}";
                }
            }
            else if (v.phase == "results")
            {
                if (v.waveResult == "win")
                {
                    bannerSkin = skinOk;
                    bannerTxt = en ? $"WAVE {v.waveNo} WIN! +100W +60S +50F" : $"THẮNG ĐỢT {v.waveNo}! +100G · +60Đ · +50L";
                }
                else
                {
                    bannerSkin = skinErr;
                    bannerTxt = v.isPostLossRecovery
                        ? (en ? "BREACHED · 250 HP & -15% Enemy HP" : "THẤT THỦ · Sàn 250 HP & -15% máu quái")
                        : (en ? "TIME UP · Penalty & -15% Enemy HP" : "HẾT GIỜ · Phạt kho & -15% máu quái");
                }
            }
            else
            {
                bannerTxt = en ? "PREP · REPAIR & RALLY" : "CHUẨN BỊ · GIA CỐ & TẬP HỢP";
            }

            if (threatBannerSkin != null) threatBannerSkin.sprite = bannerSkin;
            if (threatBannerLabel != null)
            {
                threatBannerLabel.text = bannerTxt;
                threatBannerLabel.rectTransform.offsetMin = new Vector2(14f, showBossBar ? 15f : 4f);
            }
            if (bossHpBarRoot != null)
            {
                bossHpBarRoot.SetActive(showBossBar);
                if (showBossBar && bossHpFill != null)
                {
                    float bRatio = v.bossMaxHp > 0 ? Mathf.Clamp01((float)v.bossHp / v.bossMaxHp) : 0f;
                    bossHpFill.rectTransform.anchorMax = new Vector2(Mathf.Max(0.01f, bRatio), 1f);
                }
            }

            // 4. Subordinate Vote / Objective Strip (canonical Vote A/B/C costs & effects!)
            bool voteLockedSkin = false;
            string subTxt;
            if (v.lifecycle == "stopped")
            {
                voteLockedSkin = true;
                subTxt = en ? "Clock held · Resumes when blockers clear" : "Giữ đội hình · Chạy lại khi hết cờ dừng";
            }
            else if (v.phase == "waiting" || v.waveTick < 0)
            {
                subTxt = en ? "P=0 active · No bots or unattended loss" : "P=0 người chơi · Không tự chạy hay tự thua";
            }
            else if (isFrozen)
            {
                subTxt = en ? "Clock held · Resumes when blockers clear" : "Giữ đội hình · Chạy lại khi hết cờ dừng";
            }
            else if (v.voteOpen && (v.waveNo % 2 == 0) && v.waveTick >= 800 && v.waveTick <= 1199)
            {
                int vRem = Mathf.Max(0, (1200 - v.waveTick) / 10);
                subTxt = en
                    ? $"VOTE {vRem}s · A:{v.voteA}(-50W,Def+15%) B:{v.voteB}(-50F,Atk+15%) C:{v.voteC}(0)"
                    : $"VOTE {vRem}s · A:{v.voteA}(-50G,Thủ+15%) B:{v.voteB}(-50L,Đánh+15%) C:{v.voteC}(0)";
            }
            else if (v.phase == "results" && v.pendingNextWaveCount > 0)
            {
                voteLockedSkin = true;
                subTxt = en
                    ? $"NEXT WAVE {v.waveNo + 1}: Queued for start ({v.pendingNextWaveCount})"
                    : $"ĐỢT SAU ({v.waveNo + 1}): Vào thẳng đầu đợt ({v.pendingNextWaveCount} chờ)";
            }
            else if (!string.IsNullOrEmpty(v.voteSelected) && (v.waveNo % 2 == 0))
            {
                voteLockedSkin = true;
                string effDesc;
                if (v.voteSelected == "A")
                    effDesc = en ? "A: -50 Wood, Guard+15%" : "A: -50 Gỗ, Thủ+15%";
                else if (v.voteSelected == "B")
                    effDesc = en ? "B: -50 Food, Atk+15%" : "B: -50 Lương, Đánh+15%";
                else
                    effDesc = en ? "C: Free, -10%Def, -50%Loss" : "C: 0 phí, Thủ-10%, -50%phạt";
                subTxt = en
                    ? $"LOCKED [{effDesc}] · A:{v.voteA} B:{v.voteB} C:{v.voteC}"
                    : $"CHỐT [{effDesc}] · A:{v.voteA} B:{v.voteB} C:{v.voteC}";
            }
            else if (v.waveNo % 2 == 0)
            {
                int nextTier = Mathf.Min(20, v.wallTier + 1);
                subTxt = en
                    ? $"EVEN WAVE {v.waveNo}: Vote A/B/C at 03:40 · Wall Tier {nextTier}"
                    : $"ĐỢT CHẴN {v.waveNo}: Bầu A/B/C lúc 03:40 · Tường Cấp {nextTier}";
            }
            else
            {
                int nextTier = Mathf.Min(20, v.wallTier + 1);
                subTxt = en
                    ? $"ODD WAVE {v.waveNo}: Wall Tier {nextTier} & balance roles"
                    : $"ĐỢT LẺ {v.waveNo}: Nâng Tường Cấp {nextTier} & đủ 3 vai";
            }
            if (voteStripSkin != null) voteStripSkin.sprite = voteLockedSkin ? skinVoteLocked : skinVoteOpen;
            if (voteStripLabel != null) voteStripLabel.text = subTxt;

            // 5. Secondary Diagnostic / Militia / Coexisting Boss / Recovery Pill
            string diagTxt = null;
            Sprite diagSp = skinWarn;
            if (!v.metricsAvailable)
            {
                diagTxt = en ? "QUEUE METRICS: UNKNOWN / STALE" : "HÀNG ĐỢI: CHƯA RÕ / TẠM TRỄ";
            }
            else if (isFrozen && v.bossExists && v.waveTick >= 2600)
            {
                diagSp = skinErr;
                diagTxt = en
                    ? $"BOSS {v.bossHp}/{v.bossMaxHp} · W:{v.westHp} · E:{v.eastHp}"
                    : $"BOSS {v.bossHp}/{v.bossMaxHp} · TÂY {v.westHp} · ĐÔNG {v.eastHp}";
            }
            else if (v.pauseReasons != null && v.pauseReasons.Length > 1)
            {
                diagSp = skinErr;
                diagTxt = (en ? "ACTIVE BLOCKERS: " : "CỜ DỪNG ĐỒNG THỜI: ") + string.Join(" + ", v.pauseReasons);
            }
            else if (v.confirmedValidDrops == 0 && (v.queueFillPct >= 75 || v.retryableRejectedCount > 0))
            {
                diagSp = skinWarn;
                diagTxt = en
                    ? $"QUEUE {v.queueFillPct}% · Throttling (0 valid drops)"
                    : $"HÀNG ĐỢI {v.queueFillPct}% · Điều tiết (0 lệnh rơi)";
            }
            else if (v.militiaActive && v.uniqueJoined >= 1 && v.uniqueJoined <= 10)
            {
                diagSp = skinWarn;
                diagTxt = en
                    ? $"System Militia (P<=10 · +{v.militiaAttackBonus} ATK)"
                    : $"Dân quân Hệ thống (P<=10 · +{v.militiaAttackBonus} công)";
            }
            else if (v.isPostLossRecovery)
            {
                diagSp = skinOk;
                diagTxt = en ? "RECOVERY RELIEF: -15% enemy HP" : "HỖ TRỢ PHỤC HỒI: -15% máu quái";
            }
            if (secondaryDiagGroup != null)
            {
                secondaryDiagGroup.alpha = string.IsNullOrEmpty(diagTxt) ? 0f : 1f;
                if (!string.IsNullOrEmpty(diagTxt))
                {
                    if (secondaryDiagSkin != null) secondaryDiagSkin.sprite = diagSp;
                    if (secondaryDiagLabel != null) secondaryDiagLabel.text = diagTxt;
                }
            }

            // 6. Primary CTA, Join Toast & Synthetic/Replay Watermark
            if (primaryCtaLabel != null)
            {
                primaryCtaLabel.text = v.lifecycle == "stopped"
                    ? (en ? "MAINTENANCE · BACK SOON" : "PHIÊN DỪNG · CHỜ MỞ LẠI")
                    : (en ? "Comment BUILD / DEFEND / FIGHT" : "Bình luận XÂY / THỦ / ĐÁNH");
            }

            if (joinToastGroup != null)
            {
                bool showToast = v.showJoinToast && v.lifecycle == "running" && !string.IsNullOrEmpty(v.joinToastAlias);
                joinToastGroup.alpha = showToast ? 1f : 0f;
                if (showToast && joinToastLabel != null)
                {
                    string rShort = v.joinToastRole == "builder" ? (en ? "BUILD" : "XÂY") :
                                    v.joinToastRole == "guard" ? (en ? "DEFEND" : "THỦ") : (en ? "FIGHT" : "ĐÁNH");
                    joinToastLabel.text = en
                        ? $"+1 {rShort} · {v.joinToastAlias} (Accepted)"
                        : $"+1 {rShort} · {v.joinToastAlias} (Đã ghi nhận)";
                }
            }

            if (watermarkGroup != null)
            {
                bool showWm = !string.Equals(v.sourceStatus, "connected", StringComparison.Ordinal);
                watermarkGroup.alpha = showWm ? 1f : 0f;
                if (showWm && watermarkLabel != null)
                {
                    watermarkLabel.text = string.Equals(v.sourceStatus, "replay", StringComparison.Ordinal)
                        ? (en ? "REPLAY · DETERMINISTIC (NOT LIVE)" : "REPLAY · PHÁT LẠI (CHƯA LIVE)")
                        : (en ? "MOCK / REPLAY · SYNTHETIC (NOT LIVE)" : "MOCK / REPLAY · MÔ PHỎNG (CHƯA LIVE)");
                }
            }

            if (stateChanged)
            {
                InPlaceStateTransitionsCount++;
                _bannerTransitionT = ReducedMotion ? 1f : 0f;
                if (ReducedMotion) _displayedWallRatio = _targetWallRatio;
                ApplyMotionValues();
            }
        }

        public void StepMotionMs(float deltaMs)
        {
            if (ReducedMotion)
            {
                _bannerTransitionT = 1f;
                _displayedWallRatio = _targetWallRatio;
            }
            else
            {
                _bannerTransitionT = Mathf.Min(1f, _bannerTransitionT + (deltaMs / 200f));
                _displayedWallRatio = Mathf.MoveTowards(_displayedWallRatio, _targetWallRatio, deltaMs / 220f);
            }
            ApplyMotionValues();
        }

        private void ApplyMotionValues()
        {
            if (threatBannerGroup != null)
            {
                threatBannerGroup.alpha = ReducedMotion ? 1f : Mathf.Lerp(0.72f, 1.0f, _bannerTransitionT);
            }
            if (threatBannerRect != null)
            {
                float yOffset = ReducedMotion ? 0f : Mathf.Lerp(10f, 0f, _bannerTransitionT);
                threatBannerRect.anchoredPosition = _bannerRestAnchoredPos + new Vector2(0f, yOffset);
            }
            if (wallHpFill != null)
            {
                wallHpFill.rectTransform.anchorMax = new Vector2(Mathf.Max(0.02f, _displayedWallRatio * 0.98f), 0.85f);
            }
        }
    }
}
