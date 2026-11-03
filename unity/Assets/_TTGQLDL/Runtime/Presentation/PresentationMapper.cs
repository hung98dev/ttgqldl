using System;
using System.Globalization;
using Ttgqldl.Core;

namespace Ttgqldl.Presentation
{
    /// <summary>
    /// Pure mapping from a validated public snapshot to a render-only view model.
    ///
    /// It never computes resources, combat, XP or victory: the victory and clear
    /// flags only restate what the server already published (all pools empty, no
    /// pending boss, RESULTS phase). Identical inputs and identical elapsed time
    /// always produce an identical model.
    /// </summary>
    public static class PresentationMapper
    {
        /// <summary>After this many seconds without a snapshot the view is stale.</summary>
        public const int StaleThresholdSeconds = 3;

        /// <summary>A snapshot that failed validation is not displayed at all.</summary>
        public static HudViewModel Rejected(string reason, ConnectionState connection)
        {
            return new HudViewModel
            {
                WaveLabel = "WAVE --",
                TimerLabel = "--:--",
                PhaseLabel = "NO DATA",
                LifecycleLabel = "UNKNOWN",
                StatusLabel = StatusLabelFor(connection),
                WallLabel = "TƯỜNG --/--",
                TownhallLabel = "NHÀ CHÍNH --/--",
                ResourcesLabel = "GỖ --   ĐÁ --   LƯƠNG --",
                RoleCountsLabel = "XÂY --   THỦ --   ĐÁNH --",
                VoteLabel = string.Empty,
                SpotlightLabel = string.Empty,
                BannerTextKey = string.Empty,
                ErrorText = reason ?? string.Empty,
                Lanes = Array.Empty<LaneViewModel>(),
                Connection = connection,
                DisconnectedBannerVisible = connection == ConnectionState.Disconnected,
                StaleIndicatorVisible = connection == ConnectionState.Stale,
            };
        }

        public static HudViewModel Map(PublicSnapshot snapshot, ConnectionState connection, float secondsSinceUpdate)
        {
            if (snapshot == null)
            {
                return Rejected("no snapshot", connection);
            }

            var stale = connection == ConnectionState.Stale || secondsSinceUpdate > StaleThresholdSeconds;
            var ordinary = TotalOrdinaryHp(snapshot);
            var boss = TotalBossHp(snapshot);
            var cleared = ordinary == 0 && boss == 0;
            var atResults = snapshot.Phase == Phase.Results;

            return new HudViewModel
            {
                WaveLabel = "WAVE " + snapshot.WaveNo.ToString(CultureInfo.InvariantCulture),
                TimerLabel = FormatTimer(snapshot.RemainingSeconds),
                PhaseLabel = PhaseLabelFor(snapshot.Phase),
                LifecycleLabel = LifecycleLabelFor(snapshot.Lifecycle),
                StatusLabel = StatusLabelFor(connection),
                WallLabel = "TƯỜNG " + snapshot.WallHp.ToString(CultureInfo.InvariantCulture) + "/" +
                            snapshot.WallMax.ToString(CultureInfo.InvariantCulture),
                TownhallLabel = "NHÀ CHÍNH " + snapshot.TownhallHp.ToString(CultureInfo.InvariantCulture) + "/" +
                                snapshot.TownhallMax.ToString(CultureInfo.InvariantCulture),
                ResourcesLabel = "GỖ " + snapshot.Wood.ToString(CultureInfo.InvariantCulture) +
                                 "   ĐÁ " + snapshot.Stone.ToString(CultureInfo.InvariantCulture) +
                                 "   LƯƠNG " + snapshot.Food.ToString(CultureInfo.InvariantCulture),
                RoleCountsLabel = "XÂY " + snapshot.Counts.Builder.ToString(CultureInfo.InvariantCulture) +
                                  "   THỦ " + snapshot.Counts.Guard.ToString(CultureInfo.InvariantCulture) +
                                  "   ĐÁNH " + snapshot.Counts.Fighter.ToString(CultureInfo.InvariantCulture),
                VoteLabel = VoteLabelFor(snapshot.Vote),
                SpotlightLabel = SpotlightLabelFor(snapshot),
                BannerTextKey = snapshot.Banner?.TextKey ?? string.Empty,
                ErrorText = string.Empty,
                Lanes = MapLanes(snapshot),
                Connection = connection,
                MockWatermarkVisible = snapshot.IsMockSource,
                StaleIndicatorVisible = stale,
                PausedBannerVisible = snapshot.Lifecycle == VillageLifecycle.Paused,
                DisconnectedBannerVisible = connection == ConnectionState.Disconnected ||
                                            snapshot.Lifecycle == VillageLifecycle.SourceDisconnected,
                StorageBannerVisible = snapshot.Lifecycle == VillageLifecycle.StorageUnavailable,
                // CLEAR may never be shown while a boss spawn is still pending.
                ClearVisible = cleared && !snapshot.BossIncoming && !atResults,
                // Victory is only ever a restatement of the server's RESULTS state.
                VictoryVisible = atResults && cleared && !snapshot.BossIncoming && snapshot.TownhallHp > 0,
                DefeatVisible = atResults && snapshot.TownhallHp == 0,
                VoteVisible = snapshot.Vote != null && snapshot.Vote.Open,
            };
        }

        public static string PhaseLabelFor(Phase phase)
        {
            switch (phase)
            {
                case Phase.Waiting: return "CHỜ NGƯỜI CHƠI";
                case Phase.Prep: return "CHUẨN BỊ";
                case Phase.Assault: return "TẤN CÔNG";
                case Phase.Finale: return "QUYẾT CHIẾN";
                case Phase.Results: return "KẾT QUẢ";
                default: return "KHÔNG RÕ";
            }
        }

        public static string LifecycleLabelFor(VillageLifecycle lifecycle)
        {
            switch (lifecycle)
            {
                case VillageLifecycle.Waiting: return "WAITING";
                case VillageLifecycle.Running: return "RUNNING";
                case VillageLifecycle.Paused: return "PAUSED";
                case VillageLifecycle.SourceDisconnected: return "SOURCE_DISCONNECTED";
                case VillageLifecycle.StorageUnavailable: return "STORAGE_UNAVAILABLE";
                case VillageLifecycle.Stopped: return "STOPPED";
                default: return "UNKNOWN";
            }
        }

        public static string StatusLabelFor(ConnectionState state)
        {
            switch (state)
            {
                case ConnectionState.Live: return "LIVE";
                case ConnectionState.Mock: return "MOCK";
                case ConnectionState.Replay: return "REPLAY";
                case ConnectionState.Stale: return "STALE";
                case ConnectionState.Disconnected: return "DISCONNECTED";
                case ConnectionState.Error: return "ERROR";
                default: return "UNKNOWN";
            }
        }

        public static string FormatTimer(int remainingSeconds)
        {
            var clamped = remainingSeconds;
            if (clamped < 0)
            {
                clamped = 0;
            }

            if (clamped > PublicStateValidator.MaxWallSeconds)
            {
                clamped = PublicStateValidator.MaxWallSeconds;
            }

            var minutes = clamped / 60;
            var seconds = clamped % 60;
            return minutes.ToString("00", CultureInfo.InvariantCulture) + ":" +
                   seconds.ToString("00", CultureInfo.InvariantCulture);
        }

        private static LaneViewModel[] MapLanes(PublicSnapshot snapshot)
        {
            var lanes = new LaneViewModel[snapshot.Lanes.Count];
            for (var i = 0; i < snapshot.Lanes.Count; i++)
            {
                var lane = snapshot.Lanes[i];
                lanes[i] = new LaneViewModel(
                    LaneLabelFor(lane.Id),
                    lane.OrdinaryHp,
                    lane.BossHp,
                    lane.OrdinaryHp > 0 || lane.BossHp > 0);
            }

            return lanes;
        }

        public static string LaneLabelFor(LaneId lane)
        {
            switch (lane)
            {
                case LaneId.West: return "TÂY";
                case LaneId.Center: return "GIỮA";
                case LaneId.East: return "ĐÔNG";
                default: return "?";
            }
        }

        private static string VoteLabelFor(VoteState vote)
        {
            if (vote == null)
            {
                return string.Empty;
            }

            var text = "A " + vote.CountA.ToString(CultureInfo.InvariantCulture) +
                       "   B " + vote.CountB.ToString(CultureInfo.InvariantCulture) +
                       "   C " + vote.CountC.ToString(CultureInfo.InvariantCulture);
            if (!vote.Open && !string.IsNullOrEmpty(vote.Selected))
            {
                text += "   → " + vote.Selected;
            }

            return text;
        }

        private static string SpotlightLabelFor(PublicSnapshot snapshot)
        {
            if (snapshot.Spotlights == null || snapshot.Spotlights.Count == 0)
            {
                return string.Empty;
            }

            var max = snapshot.Spotlights.Count < 3 ? snapshot.Spotlights.Count : 3;
            var parts = new string[max];
            for (var i = 0; i < max; i++)
            {
                var alias = snapshot.Spotlights[i].DisplayAlias;
                if (string.IsNullOrEmpty(alias))
                {
                    parts[i] = string.Empty;
                    continue;
                }

                parts[i] = alias.Length <= 12 ? alias : alias.Substring(0, 12) + "…";
            }

            return string.Join("  ", parts);
        }

        private static int TotalOrdinaryHp(PublicSnapshot snapshot)
        {
            var total = 0;
            for (var i = 0; i < snapshot.Lanes.Count; i++)
            {
                total += snapshot.Lanes[i].OrdinaryHp;
            }

            return total;
        }

        private static int TotalBossHp(PublicSnapshot snapshot)
        {
            var total = 0;
            for (var i = 0; i < snapshot.Lanes.Count; i++)
            {
                total += snapshot.Lanes[i].BossHp;
            }

            return total;
        }
    }
}
