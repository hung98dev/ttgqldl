using System;
using Ttgqldl.Presentation;
using UnityEngine;
using UnityEngine.UI;

namespace Ttgqldl.UI
{
    /// <summary>
    /// Applies a render-only <see cref="HudViewModel"/> to the screen.
    ///
    /// This component contains no gameplay logic: every value it shows was
    /// produced by <see cref="PresentationMapper"/> from a validated public
    /// snapshot. Elements that are false are hidden, never replaced by a
    /// fabricated success state.
    /// </summary>
    public sealed class HudView : MonoBehaviour
    {
        [Header("Header")]
        [SerializeField] private Text waveText;
        [SerializeField] private Text timerText;
        [SerializeField] private Text phaseText;
        [SerializeField] private Text lifecycleText;
        [SerializeField] private Text statusText;

        [Header("Village")]
        [SerializeField] private Text wallText;
        [SerializeField] private Text townhallText;
        [SerializeField] private Text resourcesText;
        [SerializeField] private Text rolesText;

        [Header("Lanes")]
        [SerializeField] private Text[] laneTexts = Array.Empty<Text>();

        [Header("Vote and spotlights")]
        [SerializeField] private GameObject votePanel;
        [SerializeField] private Text voteText;
        [SerializeField] private Text spotlightText;

        [Header("Overlays")]
        [SerializeField] private Text bannerText;
        [SerializeField] private Text mockWatermark;
        [SerializeField] private Text staleIndicator;
        [SerializeField] private Text connectionBanner;
        [SerializeField] private GameObject clearBadge;
        [SerializeField] private GameObject victoryBadge;
        [SerializeField] private GameObject defeatBadge;

        /// <summary>Number of times a view model was applied (test/diagnostic hook).</summary>
        public int ApplyCount { get; private set; }

        /// <summary>Last applied model.</summary>
        public HudViewModel Current { get; private set; }

        public void Apply(HudViewModel model)
        {
            if (model == null)
            {
                throw new ArgumentNullException(nameof(model));
            }

            Current = model;
            ApplyCount++;

            SetText(waveText, model.WaveLabel);
            SetText(timerText, model.TimerLabel);
            SetText(phaseText, model.PhaseLabel);
            SetText(lifecycleText, model.LifecycleLabel);
            SetText(statusText, model.StatusLabel);
            SetText(wallText, model.WallLabel);
            SetText(townhallText, model.TownhallLabel);
            SetText(resourcesText, model.ResourcesLabel);
            SetText(rolesText, model.RoleCountsLabel);
            SetText(voteText, model.VoteLabel);
            SetText(spotlightText, model.SpotlightLabel);
            SetText(bannerText, model.BannerTextKey);

            for (var i = 0; i < laneTexts.Length; i++)
            {
                if (laneTexts[i] == null)
                {
                    continue;
                }

                if (i < model.Lanes.Count)
                {
                    var lane = model.Lanes[i];
                    var boss = lane.BossHp > 0 ? "  BOSS " + lane.BossHp : string.Empty;
                    SetText(laneTexts[i], lane.Label + "  " + lane.OrdinaryHp + boss);
                    laneTexts[i].gameObject.SetActive(true);
                }
                else
                {
                    laneTexts[i].gameObject.SetActive(false);
                }
            }

            SetActive(votePanel, model.VoteVisible);
            SetActive(clearBadge, model.ClearVisible);
            SetActive(victoryBadge, model.VictoryVisible);
            SetActive(defeatBadge, model.DefeatVisible);
            SetActive(mockWatermark != null ? mockWatermark.gameObject : null, model.MockWatermarkVisible);
            SetActive(staleIndicator != null ? staleIndicator.gameObject : null, model.StaleIndicatorVisible);

            var bannerVisible = model.PausedBannerVisible || model.DisconnectedBannerVisible || model.StorageBannerVisible;
            if (connectionBanner != null)
            {
                if (bannerVisible)
                {
                    connectionBanner.text = model.PausedBannerVisible
                        ? "PAUSED BY OPERATOR"
                        : model.StorageBannerVisible
                            ? "STORAGE UNAVAILABLE"
                            : "SOURCE DISCONNECTED";
                }

                connectionBanner.gameObject.SetActive(bannerVisible);
            }
        }

        private static void SetText(Text target, string value)
        {
            if (target != null)
            {
                target.text = value ?? string.Empty;
            }
        }

        private static void SetActive(GameObject target, bool active)
        {
            if (target != null && target.activeSelf != active)
            {
                target.SetActive(active);
            }
        }

#if UNITY_EDITOR
        /// <summary>
        /// Editor-only wiring used by the scene factory. Never compiled into a
        /// player build and never used at runtime.
        /// </summary>
        public void EditorBind(
            Text wave, Text timer, Text phase, Text lifecycle, Text status,
            Text wall, Text hall, Text resources, Text roles,
            Text[] lanes, GameObject votePanelRoot, Text vote, Text spotlight,
            Text banner, Text watermark, Text stale, Text connection,
            GameObject clear, GameObject victory, GameObject defeat)
        {
            waveText = wave;
            timerText = timer;
            phaseText = phase;
            lifecycleText = lifecycle;
            statusText = status;
            wallText = wall;
            townhallText = hall;
            resourcesText = resources;
            rolesText = roles;
            laneTexts = lanes ?? Array.Empty<Text>();
            votePanel = votePanelRoot;
            voteText = vote;
            spotlightText = spotlight;
            bannerText = banner;
            mockWatermark = watermark;
            staleIndicator = stale;
            connectionBanner = connection;
            clearBadge = clear;
            victoryBadge = victory;
            defeatBadge = defeat;
        }
#endif

        /// <summary>True when every serialized reference is wired (used by scene tests).</summary>
        public bool HasCompleteBindings()
        {
            if (waveText == null || timerText == null || phaseText == null || lifecycleText == null ||
                statusText == null || wallText == null || townhallText == null || resourcesText == null ||
                rolesText == null || voteText == null || spotlightText == null || bannerText == null ||
                mockWatermark == null || staleIndicator == null || connectionBanner == null ||
                clearBadge == null || victoryBadge == null || defeatBadge == null || laneTexts.Length != 3)
            {
                return false;
            }

            foreach (var lane in laneTexts)
            {
                if (lane == null)
                {
                    return false;
                }
            }

            return true;
        }
    }
}
