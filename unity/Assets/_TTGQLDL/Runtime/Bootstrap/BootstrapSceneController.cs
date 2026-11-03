using Ttgqldl.Networking;
using UnityEngine;

namespace Ttgqldl.Bootstrap
{
    /// <summary>
    /// Entry scene controller: builds the runtime from the offline fixture,
    /// proves the fixture is available and hands over to the broadcast scene
    /// exactly once.
    /// </summary>
    public sealed class BootstrapSceneController : MonoBehaviour
    {
        [SerializeField] private bool autoAdvance = true;

        public TtgqldlRuntime Runtime { get; private set; }
        public bool FixtureAvailable { get; private set; }

        private void Start()
        {
            var source = new OfflineFixtureSource();
            Runtime = TtgqldlRuntime.Initialize(source);
            FixtureAvailable = source.IsAvailable && Runtime.HasValidSnapshot;

            if (!FixtureAvailable)
            {
                Runtime.Diagnostics.RecordError("offline fixture did not publish a valid snapshot");
                return;
            }

            if (autoAdvance)
            {
                SceneRouter.Load(SceneRouter.BroadcastScene);
            }
        }
    }
}
