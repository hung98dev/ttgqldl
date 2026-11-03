using System;
using System.Collections.Generic;
using System.Globalization;
using System.Text;
using Ttgqldl.Core;

namespace Ttgqldl.Presentation
{
    /// <summary>Render-only lane data.</summary>
    public sealed class LaneViewModel
    {
        public LaneViewModel(string label, int ordinaryHp, int bossHp, bool alive)
        {
            Label = label;
            OrdinaryHp = ordinaryHp;
            BossHp = bossHp;
            Alive = alive;
        }

        public string Label { get; }
        public int OrdinaryHp { get; }
        public int BossHp { get; }
        public bool Alive { get; }
    }

    /// <summary>
    /// Everything the HUD may draw. Produced only by
    /// <see cref="PresentationMapper"/> from a validated public snapshot; it
    /// carries no authoritative rules and no identity material.
    /// </summary>
    public sealed class HudViewModel
    {
        public string WaveLabel { get; internal set; } = string.Empty;
        public string TimerLabel { get; internal set; } = string.Empty;
        public string PhaseLabel { get; internal set; } = string.Empty;
        public string LifecycleLabel { get; internal set; } = string.Empty;
        public string StatusLabel { get; internal set; } = string.Empty;
        public string WallLabel { get; internal set; } = string.Empty;
        public string TownhallLabel { get; internal set; } = string.Empty;
        public string ResourcesLabel { get; internal set; } = string.Empty;
        public string RoleCountsLabel { get; internal set; } = string.Empty;
        public string VoteLabel { get; internal set; } = string.Empty;
        public string SpotlightLabel { get; internal set; } = string.Empty;
        public string BannerTextKey { get; internal set; } = string.Empty;
        public string ErrorText { get; internal set; } = string.Empty;

        public IReadOnlyList<LaneViewModel> Lanes { get; internal set; } = Array.Empty<LaneViewModel>();
        public ConnectionState Connection { get; internal set; } = ConnectionState.Disconnected;

        public bool MockWatermarkVisible { get; internal set; }
        public bool StaleIndicatorVisible { get; internal set; }
        public bool PausedBannerVisible { get; internal set; }
        public bool DisconnectedBannerVisible { get; internal set; }
        public bool StorageBannerVisible { get; internal set; }
        public bool ClearVisible { get; internal set; }
        public bool VictoryVisible { get; internal set; }
        public bool DefeatVisible { get; internal set; }
        public bool VoteVisible { get; internal set; }

        /// <summary>
        /// Stable textual projection used for determinism tests and logs. Two
        /// mappings of the same input must produce the same string.
        /// </summary>
        public string ToCanonicalString()
        {
            var sb = new StringBuilder(256);
            sb.Append(WaveLabel).Append('|')
                .Append(TimerLabel).Append('|')
                .Append(PhaseLabel).Append('|')
                .Append(LifecycleLabel).Append('|')
                .Append(StatusLabel).Append('|')
                .Append(WallLabel).Append('|')
                .Append(TownhallLabel).Append('|')
                .Append(ResourcesLabel).Append('|')
                .Append(RoleCountsLabel).Append('|')
                .Append(VoteLabel).Append('|')
                .Append(SpotlightLabel).Append('|')
                .Append(BannerTextKey).Append('|')
                .Append(ErrorText).Append('|')
                .Append(Connection.ToString()).Append('|')
                .Append(MockWatermarkVisible ? '1' : '0')
                .Append(StaleIndicatorVisible ? '1' : '0')
                .Append(PausedBannerVisible ? '1' : '0')
                .Append(DisconnectedBannerVisible ? '1' : '0')
                .Append(StorageBannerVisible ? '1' : '0')
                .Append(ClearVisible ? '1' : '0')
                .Append(VictoryVisible ? '1' : '0')
                .Append(DefeatVisible ? '1' : '0')
                .Append(VoteVisible ? '1' : '0');
            foreach (var lane in Lanes)
            {
                sb.Append('|').Append(lane.Label)
                    .Append(':').Append(lane.OrdinaryHp.ToString(CultureInfo.InvariantCulture))
                    .Append(':').Append(lane.BossHp.ToString(CultureInfo.InvariantCulture))
                    .Append(':').Append(lane.Alive ? '1' : '0');
            }

            return sb.ToString();
        }
    }
}
