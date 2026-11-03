using System;
using Ttgqldl.Core;

namespace Ttgqldl.Networking
{
    /// <summary>
    /// Supplies public snapshots to the client. Implementations are adapters:
    /// the offline fixture used in M01, and the real WebSocket stream owned by
    /// M05. The client never talks to TikTok directly.
    /// </summary>
    public interface IPublicStateSource
    {
        /// <summary>Raised for each accepted snapshot, in publish order.</summary>
        event Action<PublicSnapshot> SnapshotPublished;

        /// <summary>Raised when the presentation-level connection state changes.</summary>
        event Action<ConnectionState> ConnectionChanged;

        ConnectionState Connection { get; }

        /// <summary>False when the source cannot emit any snapshot.</summary>
        bool IsAvailable { get; }

        void Start();

        void Stop();
    }

    /// <summary>
    /// Reserved boundary for the M05 WebSocket client.
    ///
    /// No implementation exists in M01 and no endpoint may be invented here: the
    /// adapter will be written against the versioned contract in
    /// docs/plan/05_API_WEBSOCKET.md (Sec-WebSocket-Protocol token, snapshot then
    /// diffs, base_revision, resync). The unfinished dependency is recorded in
    /// docs/evidence/M01-W0104.md.
    /// </summary>
    public interface IPublicStateStreamAdapter
    {
        bool IsConnected { get; }

        /// <summary>Raw JSON envelope exactly as received; M05 owns decoding.</summary>
        event Action<string> MessageReceived;

        event Action<ConnectionState> ConnectionChanged;

        void Connect(string websocketUrl, string shortLivedToken);

        void Disconnect();
    }
}
