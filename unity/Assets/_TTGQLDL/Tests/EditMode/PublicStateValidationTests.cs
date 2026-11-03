using NUnit.Framework;
using Ttgqldl.Core;
using Ttgqldl.Networking;

namespace Ttgqldl.Tests.EditMode
{
    /// <summary>
    /// The public projection must be rejected before display when it is null,
    /// malformed or out of range. Validation never repairs data.
    /// </summary>
    public sealed class PublicStateValidationTests
    {
        [Test]
        public void ValidFixturePasses()
        {
            var result = PublicStateValidator.Validate(MockFixtures.CreateDefault());
            Assert.IsTrue(result.IsValid, string.Join("; ", result.Errors));
        }

        [Test]
        public void NullSnapshotIsRejected()
        {
            var result = PublicStateValidator.Validate(null);
            Assert.IsFalse(result.IsValid);
            StringAssert.Contains("null", result.Errors[0]);
        }

        [Test]
        public void UnsupportedSchemaVersionIsRejected()
        {
            var snapshot = SnapshotWith(schemaVersion: 2);
            var result = PublicStateValidator.Validate(snapshot);
            Assert.IsFalse(result.IsValid);
            StringAssert.Contains("schema_version", string.Join(";", result.Errors));
        }

        [Test]
        public void WallAboveMaximumIsRejected()
        {
            var snapshot = SnapshotWith(wallHp: 1200, wallMax: 1000);
            Assert.IsFalse(PublicStateValidator.Validate(snapshot).IsValid);
        }

        [Test]
        public void NegativeResourcesAreRejected()
        {
            Assert.IsFalse(PublicStateValidator.Validate(SnapshotWith(wood: -1)).IsValid);
            Assert.IsFalse(PublicStateValidator.Validate(SnapshotWith(stone: -1)).IsValid);
            Assert.IsFalse(PublicStateValidator.Validate(SnapshotWith(food: -1)).IsValid);
        }

        [Test]
        public void ResourcesAboveCapAreRejected()
        {
            Assert.IsFalse(PublicStateValidator.Validate(SnapshotWith(wood: 1_000_001)).IsValid);
        }

        [Test]
        public void TimerOutsideWaveIsRejected()
        {
            Assert.IsFalse(PublicStateValidator.Validate(SnapshotWith(remainingSeconds: -1)).IsValid);
            Assert.IsFalse(PublicStateValidator.Validate(SnapshotWith(remainingSeconds: 301)).IsValid);
        }

        [Test]
        public void WrongLaneCountOrOrderIsRejected()
        {
            var wrongCount = SnapshotWith(lanes: new[]
            {
                new LaneState(LaneId.West, 1, 0),
                new LaneState(LaneId.Center, 1, 0),
            });
            Assert.IsFalse(PublicStateValidator.Validate(wrongCount).IsValid);

            var wrongOrder = SnapshotWith(lanes: new[]
            {
                new LaneState(LaneId.Center, 1, 0),
                new LaneState(LaneId.West, 1, 0),
                new LaneState(LaneId.East, 1, 0),
            });
            Assert.IsFalse(PublicStateValidator.Validate(wrongOrder).IsValid);
        }

        [Test]
        public void NegativeLaneHitPointsAreRejected()
        {
            var snapshot = SnapshotWith(lanes: new[]
            {
                new LaneState(LaneId.West, -1, 0),
                new LaneState(LaneId.Center, 0, 0),
                new LaneState(LaneId.East, 0, 0),
            });
            Assert.IsFalse(PublicStateValidator.Validate(snapshot).IsValid);
        }

        [Test]
        public void TooManySpotlightsAreRejected()
        {
            var spotlights = new SpotlightEntry[13];
            for (var i = 0; i < spotlights.Length; i++)
            {
                spotlights[i] = new SpotlightEntry("ref-" + i, "Villager " + i, "guard");
            }

            Assert.IsFalse(PublicStateValidator.Validate(SnapshotWith(spotlights: spotlights)).IsValid);
        }

        [Test]
        public void ActorKeyShapedPublicRefIsRejected()
        {
            var snapshot = SnapshotWith(spotlights: new[]
            {
                new SpotlightEntry(new string('a', 64), "Villager 001", "builder"),
            });
            var result = PublicStateValidator.Validate(snapshot);
            Assert.IsFalse(result.IsValid);
            StringAssert.Contains("actor key", string.Join(";", result.Errors));
        }

        [Test]
        public void UnknownSpotlightRoleIsRejected()
        {
            var snapshot = SnapshotWith(spotlights: new[]
            {
                new SpotlightEntry("ref-1", "Villager 001", "wizard"),
            });
            Assert.IsFalse(PublicStateValidator.Validate(snapshot).IsValid);
        }

        [Test]
        public void OpenVoteWithoutAllOptionsIsRejected()
        {
            var snapshot = SnapshotWith(vote: new VoteState(true, new[] { "A", "B" }, 0, 0, 0, null));
            Assert.IsFalse(PublicStateValidator.Validate(snapshot).IsValid);
        }

        [Test]
        public void SelectedVoteOutsideAbcIsRejected()
        {
            var snapshot = SnapshotWith(vote: new VoteState(false, new[] { "A", "B", "C" }, 0, 0, 0, "D"));
            Assert.IsFalse(PublicStateValidator.Validate(snapshot).IsValid);
        }

        [Test]
        public void BossIncomingDuringResultsIsRejected()
        {
            var snapshot = SnapshotWith(phase: Phase.Results, bossIncoming: true);
            Assert.IsFalse(PublicStateValidator.Validate(snapshot).IsValid);
        }

        [Test]
        public void EmptyBannerKeysAreRejected()
        {
            var snapshot = SnapshotWith(banner: new EventBanner(string.Empty, string.Empty, 1000));
            Assert.IsFalse(PublicStateValidator.Validate(snapshot).IsValid);
        }

        [Test]
        public void BossIncomingFixtureIsValid()
        {
            var result = PublicStateValidator.Validate(MockFixtures.CreateBossIncoming());
            Assert.IsTrue(result.IsValid, string.Join("; ", result.Errors));
        }

        private static PublicSnapshot SnapshotWith(
            int schemaVersion = PublicSnapshot.SupportedSchemaVersion,
            int wallHp = 1000,
            int wallMax = 1000,
            int remainingSeconds = 100,
            int wood = 400,
            int stone = 250,
            int food = 300,
            Phase phase = Phase.Prep,
            LaneState[] lanes = null,
            VoteState vote = null,
            SpotlightEntry[] spotlights = null,
            EventBanner banner = null,
            bool bossIncoming = false)
        {
            return new PublicSnapshot(
                schemaVersion,
                "test-revision",
                3,
                VillageLifecycle.Running,
                new string[0],
                phase,
                remainingSeconds,
                wallHp,
                wallMax,
                500,
                500,
                wood,
                stone,
                food,
                new RoleCounts(1, 1, 1, 3),
                lanes ?? new[]
                {
                    new LaneState(LaneId.West, 10, 0),
                    new LaneState(LaneId.Center, 10, 0),
                    new LaneState(LaneId.East, 10, 0),
                },
                vote ?? new VoteState(false, new[] { "A", "B", "C" }, 0, 0, 0, null),
                SourceStatus.Mock,
                spotlights ?? new SpotlightEntry[0],
                banner,
                bossIncoming);
        }
    }
}
