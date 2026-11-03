using System;
using Ttgqldl.Core;

namespace Ttgqldl.Networking
{
    /// <summary>
    /// Deterministic, clearly-labelled offline data. Every value here is a
    /// MOCK fixture for layout work: it is not a simulation, it is never sent
    /// anywhere and it can never be mistaken for a real audience.
    /// </summary>
    public static class MockFixtures
    {
        public const string DefaultRevision = "mock-1";

        public static PublicSnapshot CreateDefault()
        {
            return new PublicSnapshot(
                schemaVersion: PublicSnapshot.SupportedSchemaVersion,
                revision: DefaultRevision,
                waveNo: 3,
                lifecycle: VillageLifecycle.Running,
                pauseReasons: Array.Empty<string>(),
                phase: Phase.Prep,
                remainingSeconds: 161,
                wallHp: 1000,
                wallMax: 1000,
                townhallHp: 500,
                townhallMax: 500,
                wood: 400,
                stone: 250,
                food: 300,
                counts: new RoleCounts(builder: 12, guard: 8, fighter: 20, uniqueJoined: 40),
                lanes: new[]
                {
                    new LaneState(LaneId.West, ordinaryHp: 86, bossHp: 0),
                    new LaneState(LaneId.Center, ordinaryHp: 81, bossHp: 0),
                    new LaneState(LaneId.East, ordinaryHp: 89, bossHp: 0),
                },
                vote: new VoteState(open: false, options: new[] { "A", "B", "C" }, countA: 0, countB: 0, countC: 0, selected: null),
                sourceStatus: SourceStatus.Mock,
                spotlights: new[]
                {
                    new SpotlightEntry("ref-a1", "Villager 001", "builder"),
                    new SpotlightEntry("ref-b2", "Villager 002", "fighter"),
                    new SpotlightEntry("ref-c3", "Villager 003", "guard"),
                },
                banner: null,
                bossIncoming: false);
        }

        /// <summary>A boss wave with every ordinary pool empty but the boss pending.</summary>
        public static PublicSnapshot CreateBossIncoming()
        {
            return new PublicSnapshot(
                schemaVersion: PublicSnapshot.SupportedSchemaVersion,
                revision: "mock-boss",
                waveNo: 5,
                lifecycle: VillageLifecycle.Running,
                pauseReasons: Array.Empty<string>(),
                phase: Phase.Finale,
                remainingSeconds: 12,
                wallHp: 640,
                wallMax: 1000,
                townhallHp: 500,
                townhallMax: 500,
                wood: 120,
                stone: 80,
                food: 90,
                counts: new RoleCounts(builder: 2, guard: 2, fighter: 6, uniqueJoined: 10),
                lanes: new[]
                {
                    new LaneState(LaneId.West, ordinaryHp: 0, bossHp: 0),
                    new LaneState(LaneId.Center, ordinaryHp: 0, bossHp: 33),
                    new LaneState(LaneId.East, ordinaryHp: 0, bossHp: 0),
                },
                vote: new VoteState(open: false, options: new[] { "A", "B", "C" }, countA: 3, countB: 4, countC: 1, selected: "B"),
                sourceStatus: SourceStatus.Mock,
                spotlights: Array.Empty<SpotlightEntry>(),
                banner: new EventBanner("boss", "ui.banner.boss_incoming", 4000),
                bossIncoming: true);
        }

        /// <summary>A finished wave used to exercise the RESULTS presentation.</summary>
        public static PublicSnapshot CreateVictory()
        {
            return new PublicSnapshot(
                schemaVersion: PublicSnapshot.SupportedSchemaVersion,
                revision: "mock-win",
                waveNo: 6,
                lifecycle: VillageLifecycle.Running,
                pauseReasons: Array.Empty<string>(),
                phase: Phase.Results,
                remainingSeconds: 8,
                wallHp: 812,
                wallMax: 1000,
                townhallHp: 500,
                townhallMax: 500,
                wood: 512,
                stone: 310,
                food: 350,
                counts: new RoleCounts(builder: 4, guard: 4, fighter: 12, uniqueJoined: 20),
                lanes: new[]
                {
                    new LaneState(LaneId.West, ordinaryHp: 0, bossHp: 0),
                    new LaneState(LaneId.Center, ordinaryHp: 0, bossHp: 0),
                    new LaneState(LaneId.East, ordinaryHp: 0, bossHp: 0),
                },
                vote: new VoteState(open: false, options: new[] { "A", "B", "C" }, countA: 0, countB: 0, countC: 0, selected: "C"),
                sourceStatus: SourceStatus.Mock,
                spotlights: Array.Empty<SpotlightEntry>(),
                banner: new EventBanner("result", "ui.banner.victory", 20000),
                bossIncoming: false);
        }
    }

    /// <summary>
    /// Offline snapshot source used by the bootstrap scene and by tests. It is
    /// deterministic, publishes synchronously and never performs I/O.
    /// </summary>
    public sealed class OfflineFixtureSource : IPublicStateSource
    {
        private readonly Func<PublicSnapshot> _factory;
        private PublicSnapshot _current;

        public OfflineFixtureSource()
            : this(MockFixtures.CreateDefault)
        {
        }

        public OfflineFixtureSource(Func<PublicSnapshot> factory)
        {
            _factory = factory ?? throw new ArgumentNullException(nameof(factory));
        }

        public event Action<PublicSnapshot> SnapshotPublished;
        public event Action<ConnectionState> ConnectionChanged;

        public ConnectionState Connection { get; private set; } = ConnectionState.Disconnected;
        public bool IsAvailable { get; private set; }
        public int PublishCount { get; private set; }
        public PublicSnapshot Current => _current;

        public void Start()
        {
            IsAvailable = true;
            SetConnection(ConnectionState.Mock);
            Publish();
        }

        public void Stop()
        {
            IsAvailable = false;
            SetConnection(ConnectionState.Disconnected);
        }

        /// <summary>Publishes the fixture again (used to simulate an update).</summary>
        public void Publish()
        {
            var snapshot = _factory();
            _current = snapshot;
            PublishCount++;
            SnapshotPublished?.Invoke(snapshot);
        }

        /// <summary>Drives the presentation-level connection state in tests and previews.</summary>
        public void SetConnection(ConnectionState state)
        {
            if (Connection == state)
            {
                return;
            }

            Connection = state;
            ConnectionChanged?.Invoke(state);
        }
    }
}
