using System;
using UnityEngine;
using UnityEngine.SceneManagement;
using UnityEngine.UI;

namespace TTGQLDL.Showcase.UI
{
    /// <summary>
    /// In-place 16-state HUD controller for the isolated ART-008 vertical slice.
    /// Switches presentation state (PREP, VOTE_OPEN, ASSAULT, BOSS_INCOMING, BOSS_ACTIVE, RESULTS_WIN,
    /// SOURCE_RECONNECTING, PAUSED, STORAGE_UNAVAILABLE, MAINTENANCE) WITHOUT reloading the Unity scene,
    /// supporting both subtle standard micro-animations (<=220ms, 0 Hz strobe, 0 camera shake) and ReducedMotion.
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
        public Image voteStripSkin;
        public Image ctaBannerSkin;
        public CanvasGroup joinToastGroup;

        public Sprite skinNavy;
        public Sprite skinOk;
        public Sprite skinWarn;
        public Sprite skinErr;
        public Sprite skinVoteOpen;
        public Sprite skinVoteLocked;

        private Scene _activeSceneBefore;
        private float _displayedWallRatio = 1.0f;
        private float _targetWallRatio = 1.0f;
        private float _bannerTransitionT = 1.0f;
        private Vector2 _bannerRestAnchoredPos;

        public Text waveLabel;
        public Text timerLabel;
        public Text sourceStatusLabel;
        public Text threatBannerLabel;
        public Text voteStripLabel;
        public Text primaryCtaLabel;

        public bool ReducedMotion { get; set; }
        public string CurrentVisualState { get; private set; } = "WAITING_FOR_VIEWER";
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

        public void ApplyCommittedStateInPlace(
            string visualStateId,
            float wallHpRatio,
            float castleHpRatio,
            bool voteLocked,
            bool showJoinToast)
        {
            if (!SceneManager.GetActiveScene().Equals(_activeSceneBefore))
            {
                SceneReloadCount++;
            }

            bool stateChanged = !string.Equals(CurrentVisualState, visualStateId, StringComparison.Ordinal);
            CurrentVisualState = visualStateId;
            _targetWallRatio = Mathf.Clamp01(wallHpRatio);

            if (castleHpFill != null)
            {
                castleHpFill.rectTransform.anchorMax = new Vector2(Mathf.Clamp01(castleHpRatio), 1f);
            }

            if (threatBannerSkin != null)
            {
                switch (visualStateId)
                {
                    case "STORAGE_UNAVAILABLE":
                    case "MAINTENANCE":
                    case "PREP_15_WARN":
                    case "BOSS_INCOMING":
                    case "BOSS_ACTIVE":
                    case "RESULTS_LOSS":
                    case "DEGRADED_DROPS":
                        threatBannerSkin.sprite = skinErr;
                        break;
                    case "SOURCE_RECONNECTING":
                    case "PAUSED":
                    case "PREP_60_WARN":
                    case "ASSAULT":
                    case "FINALE":
                        threatBannerSkin.sprite = skinWarn;
                        break;
                    case "RESULTS_WIN":
                        threatBannerSkin.sprite = skinOk;
                        break;
                    default:
                        threatBannerSkin.sprite = skinNavy;
                        break;
                }
            }

            if (sourceStatusSkin != null)
            {
                if (visualStateId == "STORAGE_UNAVAILABLE" || visualStateId == "MAINTENANCE")
                    sourceStatusSkin.sprite = skinErr;
                else if (visualStateId == "SOURCE_RECONNECTING" || visualStateId == "PAUSED")
                    sourceStatusSkin.sprite = skinWarn;
                else
                    sourceStatusSkin.sprite = skinOk;
            }

            if (voteStripSkin != null)
            {
                voteStripSkin.sprite = voteLocked ? skinVoteLocked : skinVoteOpen;
            }

            if (threatBannerLabel != null)
            {
                threatBannerLabel.text = visualStateId switch
                {
                    "BOSS_ACTIVE" => "BOSS CỔ NGỮ TẤN CÔNG — THỦ CỔNG CHÍNH!",
                    "BOSS_INCOMING" => "CẢNH BÁO BOSS — CÒN 15 GIÂY!",
                    "SOURCE_RECONNECTING" => "MẤT KẾT NỐI NGUỒN LIVE — ĐANG KẾT NỐI LẠI",
                    "PAUSED" => "TẠM DỪNG BỞI NGƯỜI VẬN HÀNH",
                    "STORAGE_UNAVAILABLE" => "LỖI LƯU TRỮ — TẠM DỪNG AN TOÀN",
                    "MAINTENANCE" => "BẢO TRÌ HỆ THỐNG — PHIÊN TẠM DỪNG",
                    "RESULTS_WIN" => "CHIẾN THẮNG ĐỢT 10 — GIỮ VỮNG THÀNH!",
                    _ => "ĐỢT TẤN CÔNG ĐANG DIỄN RA — 3 LÀN ÁP SÁT!"
                };
            }
            if (sourceStatusLabel != null)
            {
                sourceStatusLabel.text = visualStateId == "SOURCE_RECONNECTING" ? "MẤT KẾT NỐI" :
                                         visualStateId == "PAUSED" ? "TẠM DỪNG" :
                                         visualStateId == "STORAGE_UNAVAILABLE" ? "LỖI DB" : "MÔ PHỎNG";
            }
            if (primaryCtaLabel != null)
            {
                primaryCtaLabel.text = visualStateId == "MAINTENANCE"
                    ? "Phiên đang bảo trì — vui lòng chờ"
                    : "Bình luận XÂY / THỦ / ĐÁNH";
            }

            if (joinToastGroup != null)
            {
                joinToastGroup.alpha = showJoinToast ? 1f : 0f;
            }

            if (stateChanged)
            {
                InPlaceStateTransitionsCount++;
                if (ReducedMotion)
                {
                    _bannerTransitionT = 1f;
                    _displayedWallRatio = _targetWallRatio;
                    ApplyMotionValues();
                }
                else
                {
                    _bannerTransitionT = 0f;
                    ApplyMotionValues();
                }
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
                // Gentle 200ms opacity + 10px slide-in & 220ms HP interpolation (0 Hz strobe, 0 camera shake)
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
                wallHpFill.rectTransform.anchorMax = new Vector2(_displayedWallRatio, 1f);
            }
        }
    }
}
