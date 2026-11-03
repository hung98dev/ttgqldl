namespace Ttgqldl.Core
{
    /// <summary>Wave phase, derived by the server from the local tick only.</summary>
    public enum Phase
    {
        Waiting = 0,
        Prep = 1,
        Assault = 2,
        Finale = 3,
        Results = 4,
    }

    /// <summary>Session lifecycle as published by the backend.</summary>
    public enum VillageLifecycle
    {
        Waiting = 0,
        Running = 1,
        Paused = 2,
        SourceDisconnected = 3,
        StorageUnavailable = 4,
        Stopped = 5,
    }

    /// <summary>The three approach lanes, ordered west, center, east.</summary>
    public enum LaneId
    {
        West = 0,
        Center = 1,
        East = 2,
    }

    /// <summary>Where the public state came from.</summary>
    public enum SourceStatus
    {
        Connected = 0,
        Disconnected = 1,
        Mock = 2,
        Replay = 3,
    }

    /// <summary>Client-side connection presentation state (never authoritative).</summary>
    public enum ConnectionState
    {
        Live = 0,
        Mock = 1,
        Replay = 2,
        Stale = 3,
        Disconnected = 4,
        Error = 5,
    }
}
