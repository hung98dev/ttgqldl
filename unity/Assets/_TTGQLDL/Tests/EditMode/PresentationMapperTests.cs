using NUnit.Framework;
using Ttgqldl.Core;
using Ttgqldl.Networking;
using Ttgqldl.Presentation;

namespace Ttgqldl.Tests.EditMode
{
    /// <summary>
    /// The mapper may only restate server truth. These tests pin the
    /// "never show CLEAR while the boss is pending" and "victory only at
    /// RESULTS" rules plus deterministic formatting.
    /// </summary>
    public sealed class PresentationMapperTests
    {
        [Test]
        public void DefaultFixtureProducesReadableLabels()
        {
            var view = PresentationMapper.Map(MockFixtures.CreateDefault(), ConnectionState.Mock, 0f);

            Assert.AreEqual("WAVE 3", view.WaveLabel);
            Assert.AreEqual("02:41", view.TimerLabel);
            Assert.AreEqual("CHUẨN BỊ", view.PhaseLabel);
            Assert.AreEqual("XÂY 12   THỦ 8   ĐÁNH 20", view.RoleCountsLabel);
            Assert.AreEqual("GỖ 400   ĐÁ 250   LƯƠNG 300", view.ResourcesLabel);
            Assert.AreEqual("TƯỜNG 1000/1000", view.WallLabel);
            Assert.AreEqual("NHÀ CHÍNH 500/500", view.TownhallLabel);
            Assert.AreEqual(3, view.Lanes.Count);
            Assert.AreEqual("TÂY", view.Lanes[0].Label);
            Assert.AreEqual("GIỮA", view.Lanes[1].Label);
            Assert.AreEqual("ĐÔNG", view.Lanes[2].Label);
        }

        [Test]
        public void MockSourceShowsWatermarkAndMockStatus()
        {
            var view = PresentationMapper.Map(MockFixtures.CreateDefault(), ConnectionState.Mock, 0f);
            Assert.IsTrue(view.MockWatermarkVisible);
            Assert.AreEqual("MOCK", view.StatusLabel);
        }

        [Test]
        public void LiveSourceHidesWatermark()
        {
            var snapshot = MockFixtures.CreateDefault();
            var live = new PublicSnapshot(
                snapshot.SchemaVersion, snapshot.Revision, snapshot.WaveNo, snapshot.Lifecycle,
                snapshot.PauseReasons, snapshot.Phase, snapshot.RemainingSeconds, snapshot.WallHp,
                snapshot.WallMax, snapshot.TownhallHp, snapshot.TownhallMax, snapshot.Wood,
                snapshot.Stone, snapshot.Food, snapshot.Counts, snapshot.Lanes, snapshot.Vote,
                SourceStatus.Connected, snapshot.Spotlights, snapshot.Banner, snapshot.BossIncoming);

            var view = PresentationMapper.Map(live, ConnectionState.Live, 0f);
            Assert.IsFalse(view.MockWatermarkVisible);
            Assert.AreEqual("LIVE", view.StatusLabel);
        }

        [Test]
        public void BossIncomingNeverShowsClearOrVictory()
        {
            var view = PresentationMapper.Map(MockFixtures.CreateBossIncoming(), ConnectionState.Mock, 0f);
            Assert.IsFalse(view.ClearVisible, "CLEAR must not appear while a boss spawn is pending");
            Assert.IsFalse(view.VictoryVisible);
            Assert.AreEqual("ui.banner.boss_incoming", view.BannerTextKey);
            Assert.AreEqual("QUYẾT CHIẾN", view.PhaseLabel);
        }

        [Test]
        public void EmptyPoolsWithoutBossIncomingMayShowClear()
        {
            var snapshot = WithLanes(Phase.Assault, new LaneState(LaneId.West, 0, 0),
                new LaneState(LaneId.Center, 0, 0), new LaneState(LaneId.East, 0, 0), bossIncoming: false);
            var view = PresentationMapper.Map(snapshot, ConnectionState.Mock, 0f);
            Assert.IsTrue(view.ClearVisible);
            Assert.IsFalse(view.VictoryVisible);
        }

        [Test]
        public void VictoryOnlyAtResults()
        {
            var atResults = PresentationMapper.Map(MockFixtures.CreateVictory(), ConnectionState.Mock, 0f);
            Assert.IsTrue(atResults.VictoryVisible);
            Assert.IsFalse(atResults.ClearVisible);

            var midWave = WithLanes(Phase.Assault, new LaneState(LaneId.West, 0, 0),
                new LaneState(LaneId.Center, 0, 0), new LaneState(LaneId.East, 0, 0), bossIncoming: false);
            Assert.IsFalse(PresentationMapper.Map(midWave, ConnectionState.Mock, 0f).VictoryVisible);
        }

        [Test]
        public void DefeatShownWhenHallIsGoneAtResults()
        {
            var snapshot = new PublicSnapshot(
                1, "rev", 4, VillageLifecycle.Running, new string[0], Phase.Results, 5,
                0, 1000, 0, 500, 10, 10, 10, new RoleCounts(0, 0, 0, 0),
                new[] { new LaneState(LaneId.West, 0, 0), new LaneState(LaneId.Center, 0, 0), new LaneState(LaneId.East, 0, 0) },
                new VoteState(false, new[] { "A", "B", "C" }, 0, 0, 0, null),
                SourceStatus.Mock, new SpotlightEntry[0], null, false);

            var view = PresentationMapper.Map(snapshot, ConnectionState.Mock, 0f);
            Assert.IsTrue(view.DefeatVisible);
            Assert.IsFalse(view.VictoryVisible);
        }

        [Test]
        public void StaleAfterThreshold()
        {
            var view = PresentationMapper.Map(MockFixtures.CreateDefault(), ConnectionState.Mock,
                PresentationMapper.StaleThresholdSeconds + 1f);
            Assert.IsTrue(view.StaleIndicatorVisible);
        }

        [Test]
        public void StaleConnectionIsLabelled()
        {
            var view = PresentationMapper.Map(MockFixtures.CreateDefault(), ConnectionState.Stale, 0f);
            Assert.AreEqual("STALE", view.StatusLabel);
            Assert.IsTrue(view.StaleIndicatorVisible);
        }

        [Test]
        public void DisconnectedIsVisible()
        {
            var view = PresentationMapper.Map(MockFixtures.CreateDefault(), ConnectionState.Disconnected, 0f);
            Assert.AreEqual("DISCONNECTED", view.StatusLabel);
            Assert.IsTrue(view.DisconnectedBannerVisible);
        }

        [Test]
        public void PauseAndStorageStatesBanner()
        {
            var paused = WithLifecycle(VillageLifecycle.Paused);
            Assert.IsTrue(PresentationMapper.Map(paused, ConnectionState.Mock, 0f).PausedBannerVisible);

            var storage = WithLifecycle(VillageLifecycle.StorageUnavailable);
            Assert.IsTrue(PresentationMapper.Map(storage, ConnectionState.Mock, 0f).StorageBannerVisible);
        }

        [Test]
        public void MappingIsDeterministic()
        {
            var snapshot = MockFixtures.CreateDefault();
            var first = PresentationMapper.Map(snapshot, ConnectionState.Mock, 1.25f);
            var second = PresentationMapper.Map(snapshot, ConnectionState.Mock, 1.25f);
            Assert.AreEqual(first.ToCanonicalString(), second.ToCanonicalString());
        }

        [Test]
        public void NullSnapshotIsRejectedNotInvented()
        {
            var view = PresentationMapper.Rejected("no snapshot", ConnectionState.Disconnected);
            Assert.AreEqual("NO DATA", view.PhaseLabel);
            StringAssert.Contains("no snapshot", view.ErrorText);
            Assert.IsFalse(view.VictoryVisible);
            Assert.IsFalse(view.MockWatermarkVisible, "an absent snapshot must not claim to be MOCK data");
        }

        [TestCase(0, "00:00")]
        [TestCase(59, "00:59")]
        [TestCase(60, "01:00")]
        [TestCase(161, "02:41")]
        [TestCase(300, "05:00")]
        [TestCase(-5, "00:00")]
        [TestCase(999, "05:00")]
        public void TimerIsClampedAndFormatted(int seconds, string expected)
        {
            Assert.AreEqual(expected, PresentationMapper.FormatTimer(seconds));
        }

        private static PublicSnapshot WithLanes(Phase phase, LaneState west, LaneState center, LaneState east,
            bool bossIncoming)
        {
            return new PublicSnapshot(
                1, "rev", 3, VillageLifecycle.Running, new string[0], phase, 120,
                900, 1000, 500, 500, 100, 100, 100, new RoleCounts(1, 1, 1, 3),
                new[] { west, center, east },
                new VoteState(false, new[] { "A", "B", "C" }, 0, 0, 0, null),
                SourceStatus.Mock, new SpotlightEntry[0], null, bossIncoming);
        }

        private static PublicSnapshot WithLifecycle(VillageLifecycle lifecycle)
        {
            var snapshot = MockFixtures.CreateDefault();
            return new PublicSnapshot(
                snapshot.SchemaVersion, snapshot.Revision, snapshot.WaveNo, lifecycle,
                new[] { "paused" }, snapshot.Phase, snapshot.RemainingSeconds, snapshot.WallHp,
                snapshot.WallMax, snapshot.TownhallHp, snapshot.TownhallMax, snapshot.Wood,
                snapshot.Stone, snapshot.Food, snapshot.Counts, snapshot.Lanes, snapshot.Vote,
                snapshot.SourceStatus, snapshot.Spotlights, snapshot.Banner, snapshot.BossIncoming);
        }
    }
}
