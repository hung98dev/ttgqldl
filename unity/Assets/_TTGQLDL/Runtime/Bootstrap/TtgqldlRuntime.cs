using System;
using Ttgqldl.Core;
using Ttgqldl.Diagnostics;
using Ttgqldl.Networking;
using Ttgqldl.Presentation;

namespace Ttgqldl.Bootstrap
{
    /// <summary>
    /// Composition root of the Unity client.
    ///
    /// It owns the source adapter, validates every published snapshot, keeps the
    /// last accepted model and exposes a render-only view. It is deliberately a
    /// plain class: no scene dependency, no per-frame Unity API, so EditMode
    /// tests can drive it directly in both modes.
    /// </summary>
    public sealed class TtgqldlRuntime
    {
        private PublicSnapshot _lastValid;
        private bool _hasValid;
        private string _error = string.Empty;
        private float _secondsSinceUpdate;
        private bool _started;

        private TtgqldlRuntime(IPublicStateSource source)
        {
            Source = source ?? throw new ArgumentNullException(nameof(source));
            Diagnostics = new RuntimeDiagnostics();
        }

        /// <summary>The single live runtime, or null before bootstrap.</summary>
        public static TtgqldlRuntime Instance { get; private set; }

        /// <summary>Number of successful initialisations (duplicate detection).</summary>
        public static int InitializeCount { get; private set; }

        public IPublicStateSource Source { get; }
        public RuntimeDiagnostics Diagnostics { get; }
        public ConnectionState Connection { get; private set; } = ConnectionState.Disconnected;
        public HudViewModel CurrentView { get; private set; }
        public PublicSnapshot CurrentSnapshot => _lastValid;
        public bool HasValidSnapshot => _hasValid;
        public bool IsStarted => _started;
        public float SecondsSinceUpdate => _secondsSinceUpdate;
        public string LastError => _error;

        public event Action<HudViewModel> ViewChanged;

        public int SubscriberCount => ViewChanged == null ? 0 : ViewChanged.GetInvocationList().Length;

        /// <summary>
        /// Creates the runtime once. A second call returns the existing instance
        /// so two bootstrap objects cannot produce two runtimes.
        /// </summary>
        public static TtgqldlRuntime Initialize(IPublicStateSource source)
        {
            if (source == null)
            {
                throw new ArgumentNullException(nameof(source));
            }

            if (Instance != null)
            {
                return Instance;
            }

            var runtime = new TtgqldlRuntime(source);
            Instance = runtime;
            InitializeCount++;
            runtime.StartSource();
            return runtime;
        }

        private void StartSource()
        {
            Source.SnapshotPublished += OnSnapshot;
            Source.ConnectionChanged += OnConnectionChanged;
            Source.Start();
            Connection = Source.Connection;
            _started = true;
            Recompute();
        }

        private void OnSnapshot(PublicSnapshot snapshot)
        {
            var result = PublicStateValidator.Validate(snapshot);
            if (!result.IsValid)
            {
                _error = string.Join("; ", result.Errors);
                Diagnostics.RecordError(_error);
                // Never display an invalid snapshot: keep the last accepted model
                // and mark it stale so the operator sees the truth.
                Recompute();
                return;
            }

            _lastValid = snapshot;
            _hasValid = true;
            _secondsSinceUpdate = 0f;
            _error = string.Empty;
            if (Connection == ConnectionState.Disconnected || Connection == ConnectionState.Error)
            {
                Connection = Source.Connection;
            }

            Recompute();
        }

        private void OnConnectionChanged(ConnectionState state)
        {
            Connection = state;
            Diagnostics.RecordConnection(state);
            Recompute();
        }

        /// <summary>Advances the stale clock. Call once per frame from a controller.</summary>
        public void Tick(float deltaSeconds)
        {
            if (deltaSeconds < 0f)
            {
                deltaSeconds = 0f;
            }

            var previous = _secondsSinceUpdate;
            _secondsSinceUpdate += deltaSeconds;
            var crossed = previous <= PresentationMapper.StaleThresholdSeconds &&
                          _secondsSinceUpdate > PresentationMapper.StaleThresholdSeconds;
            if (crossed)
            {
                Recompute();
            }
        }

        /// <summary>Rebuilds and publishes the view model. Always deterministic.</summary>
        public void Recompute()
        {
            var effective = Connection;
            if (_hasValid && _secondsSinceUpdate > PresentationMapper.StaleThresholdSeconds &&
                effective != ConnectionState.Disconnected && effective != ConnectionState.Error)
            {
                effective = ConnectionState.Stale;
            }

            if (!_hasValid)
            {
                CurrentView = PresentationMapper.Rejected(
                    _error.Length > 0 ? _error : "waiting for the first snapshot", effective);
            }
            else if (_error.Length > 0)
            {
                // Last validated data, explicitly marked stale.
                CurrentView = PresentationMapper.Map(_lastValid, ConnectionState.Stale, _secondsSinceUpdate);
            }
            else
            {
                CurrentView = PresentationMapper.Map(_lastValid, effective, _secondsSinceUpdate);
            }

            ViewChanged?.Invoke(CurrentView);
        }

        /// <summary>Stops the source and detaches from it.</summary>
        public void Shutdown()
        {
            if (!_started)
            {
                return;
            }

            Source.SnapshotPublished -= OnSnapshot;
            Source.ConnectionChanged -= OnConnectionChanged;
            Source.Stop();
            _started = false;
        }

        /// <summary>Test hook: tears the singleton down completely.</summary>
        public static void ResetForTests()
        {
            if (Instance != null)
            {
                Instance.Shutdown();
                Instance = null;
            }

            InitializeCount = 0;
        }
    }
}
