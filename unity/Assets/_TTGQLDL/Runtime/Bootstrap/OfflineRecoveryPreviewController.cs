using System.Text;
using Ttgqldl.Diagnostics;
using UnityEngine;
using UnityEngine.UI;

namespace Ttgqldl.Bootstrap
{
    /// <summary>
    /// Operator-facing diagnostic scene. It shows the last real connection state
    /// and error from <see cref="RuntimeDiagnostics"/> instead of pretending the
    /// broadcast is healthy, and it can return to the broadcast scene.
    /// </summary>
    public sealed class OfflineRecoveryPreviewController : MonoBehaviour
    {
        [SerializeField] private Text diagnosticsText;

        public string LastRendered { get; private set; } = string.Empty;

#if UNITY_EDITOR
        /// <summary>Editor-only wiring used by the scene factory.</summary>
        public void EditorBind(Text diagnostics) => diagnosticsText = diagnostics;
#endif

        private void Start()
        {
            Refresh();
        }

        public void Refresh()
        {
            var builder = new StringBuilder(256);
            builder.AppendLine("OFFLINE RECOVERY PREVIEW");
            builder.AppendLine("This screen is a diagnostic state, not a live broadcast.");

            var runtime = TtgqldlRuntime.Instance;
            if (runtime == null)
            {
                builder.AppendLine("runtime: not initialised");
            }
            else
            {
                builder.Append("connection: ").AppendLine(runtime.Connection.ToString());
                builder.Append("valid snapshot: ").AppendLine(runtime.HasValidSnapshot ? "yes" : "no");
                var diagnostics = runtime.Diagnostics;
                builder.Append("last error: ")
                    .AppendLine(string.IsNullOrEmpty(diagnostics.LastError) ? "none" : diagnostics.LastError);
                var entries = diagnostics.Entries;
                var shown = 0;
                foreach (var entry in entries)
                {
                    if (shown >= 8)
                    {
                        break;
                    }

                    builder.Append("  - ").AppendLine(entry);
                    shown++;
                }
            }

            LastRendered = builder.ToString();
            if (diagnosticsText != null)
            {
                diagnosticsText.text = LastRendered;
            }

            Debug.Log("[TTGQLDL] recovery preview refreshed: " + (runtime == null ? "no runtime" : runtime.Connection.ToString()));
        }

        public void ReturnToBroadcast()
        {
            SceneRouter.Load(SceneRouter.BroadcastScene);
        }
    }
}
