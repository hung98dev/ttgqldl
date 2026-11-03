using System;
using System.Collections.Generic;
using Ttgqldl.Core;

namespace Ttgqldl.Diagnostics
{
    /// <summary>
    /// Bounded in-memory diagnostic log. It exists so the recovery preview and
    /// the status banner can show the last real error instead of inventing a
    /// healthy state. It never writes to disk and never contains player data.
    /// </summary>
    public sealed class RuntimeDiagnostics
    {
        public const int DefaultCapacity = 50;

        private readonly Queue<string> _entries;

        public RuntimeDiagnostics(int capacity = DefaultCapacity)
        {
            if (capacity <= 0)
            {
                throw new ArgumentOutOfRangeException(nameof(capacity));
            }

            Capacity = capacity;
            _entries = new Queue<string>(capacity);
        }

        public int Capacity { get; }

        public string LastError { get; private set; } = string.Empty;

        public ConnectionState LastConnection { get; private set; } = ConnectionState.Disconnected;

        public int Count => _entries.Count;

        public IReadOnlyCollection<string> Entries => _entries.ToArray();

        public void Record(string message)
        {
            if (string.IsNullOrWhiteSpace(message))
            {
                return;
            }

            if (_entries.Count == Capacity)
            {
                _entries.Dequeue();
            }

            _entries.Enqueue(message);
        }

        public void RecordError(string message)
        {
            if (string.IsNullOrWhiteSpace(message))
            {
                return;
            }

            LastError = message;
            Record("error: " + message);
        }

        public void RecordConnection(ConnectionState state)
        {
            LastConnection = state;
            Record("connection: " + state);
        }

        public void Clear()
        {
            _entries.Clear();
            LastError = string.Empty;
            LastConnection = ConnectionState.Disconnected;
        }
    }
}
