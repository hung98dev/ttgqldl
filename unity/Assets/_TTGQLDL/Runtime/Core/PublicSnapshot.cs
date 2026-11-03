using System;
using System.Collections.Generic;

namespace Ttgqldl.Core
{
    /// <summary>Aggregate participant counts for the current wave.</summary>
    public sealed class RoleCounts
    {
        public RoleCounts(int builder, int guard, int fighter, int uniqueJoined)
        {
            Builder = builder;
            Guard = guard;
            Fighter = fighter;
            UniqueJoined = uniqueJoined;
        }

        public int Builder { get; }
        public int Guard { get; }
        public int Fighter { get; }
        public int UniqueJoined { get; }
    }

    /// <summary>One lane's remaining enemy hit points.</summary>
    public sealed class LaneState
    {
        public LaneState(LaneId id, int ordinaryHp, int bossHp)
        {
            Id = id;
            OrdinaryHp = ordinaryHp;
            BossHp = bossHp;
        }

        public LaneId Id { get; }
        public int OrdinaryHp { get; }
        public int BossHp { get; }
    }

    /// <summary>Vote window state. <see cref="Selected"/> is null while open.</summary>
    public sealed class VoteState
    {
        public VoteState(bool open, IReadOnlyList<string> options, int countA, int countB, int countC, string selected)
        {
            Open = open;
            Options = options ?? Array.Empty<string>();
            CountA = countA;
            CountB = countB;
            CountC = countC;
            Selected = selected;
        }

        public bool Open { get; }
        public IReadOnlyList<string> Options { get; }
        public int CountA { get; }
        public int CountB { get; }
        public int CountC { get; }
        public string Selected { get; }
    }

    /// <summary>A moderated alias sample. Never an actor key or provider id.</summary>
    public sealed class SpotlightEntry
    {
        public SpotlightEntry(string publicRef, string displayAlias, string role)
        {
            PublicRef = publicRef ?? string.Empty;
            DisplayAlias = displayAlias ?? string.Empty;
            Role = role ?? string.Empty;
        }

        public string PublicRef { get; }
        public string DisplayAlias { get; }
        public string Role { get; }
    }

    /// <summary>A short-lived banner described only by kind and localisation key.</summary>
    public sealed class EventBanner
    {
        public EventBanner(string kind, string textKey, int expiryMs)
        {
            Kind = kind ?? string.Empty;
            TextKey = textKey ?? string.Empty;
            ExpiryMs = expiryMs;
        }

        public string Kind { get; }
        public string TextKey { get; }
        public int ExpiryMs { get; }
    }

    /// <summary>
    /// Immutable public projection of one session, mirroring the normative
    /// `public_snapshot` payload in docs/plan/05_API_WEBSOCKET.md.
    ///
    /// The client never derives authoritative resources, combat, XP or victory
    /// from this type: it only renders what the server published.
    /// </summary>
    public sealed class PublicSnapshot
    {
        /// <summary>The only schema version this client understands.</summary>
        public const int SupportedSchemaVersion = 1;

        public PublicSnapshot(
            int schemaVersion,
            string revision,
            long waveNo,
            VillageLifecycle lifecycle,
            IReadOnlyList<string> pauseReasons,
            Phase phase,
            int remainingSeconds,
            int wallHp,
            int wallMax,
            int townhallHp,
            int townhallMax,
            int wood,
            int stone,
            int food,
            RoleCounts counts,
            IReadOnlyList<LaneState> lanes,
            VoteState vote,
            SourceStatus sourceStatus,
            IReadOnlyList<SpotlightEntry> spotlights,
            EventBanner banner,
            bool bossIncoming)
        {
            SchemaVersion = schemaVersion;
            Revision = revision ?? string.Empty;
            WaveNo = waveNo;
            Lifecycle = lifecycle;
            PauseReasons = pauseReasons ?? Array.Empty<string>();
            Phase = phase;
            RemainingSeconds = remainingSeconds;
            WallHp = wallHp;
            WallMax = wallMax;
            TownhallHp = townhallHp;
            TownhallMax = townhallMax;
            Wood = wood;
            Stone = stone;
            Food = food;
            Counts = counts ?? new RoleCounts(0, 0, 0, 0);
            Lanes = lanes ?? Array.Empty<LaneState>();
            Vote = vote ?? new VoteState(false, Array.Empty<string>(), 0, 0, 0, null);
            SourceStatus = sourceStatus;
            Spotlights = spotlights ?? Array.Empty<SpotlightEntry>();
            Banner = banner;
            BossIncoming = bossIncoming;
        }

        public int SchemaVersion { get; }
        public string Revision { get; }
        public long WaveNo { get; }
        public VillageLifecycle Lifecycle { get; }
        public IReadOnlyList<string> PauseReasons { get; }
        public Phase Phase { get; }
        public int RemainingSeconds { get; }
        public int WallHp { get; }
        public int WallMax { get; }
        public int TownhallHp { get; }
        public int TownhallMax { get; }
        public int Wood { get; }
        public int Stone { get; }
        public int Food { get; }
        public RoleCounts Counts { get; }
        public IReadOnlyList<LaneState> Lanes { get; }
        public VoteState Vote { get; }
        public SourceStatus SourceStatus { get; }
        public IReadOnlyList<SpotlightEntry> Spotlights { get; }
        public EventBanner Banner { get; }
        public bool BossIncoming { get; }

        /// <summary>True when the snapshot carries no server truth.</summary>
        public bool IsMockSource => SourceStatus == SourceStatus.Mock || SourceStatus == SourceStatus.Replay;
    }
}
