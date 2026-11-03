using UnityEngine.SceneManagement;

namespace Ttgqldl.Bootstrap
{
    /// <summary>
    /// The only place allowed to change scenes.
    ///
    /// Pause, source loss and storage loss are visual states handled inside the
    /// broadcast scene; this router exists for the single bootstrap hand-off and
    /// for the offline recovery preview. Reloading the active scene is refused so
    /// a reconnect storm cannot restart the broadcast world.
    /// </summary>
    public static class SceneRouter
    {
        public const string BootstrapScene = "00_Bootstrap";
        public const string BroadcastScene = "01_BroadcastWorld";
        public const string RecoveryScene = "02_OfflineRecoveryPreview";

        /// <summary>Total scene transitions performed since the last reset.</summary>
        public static int LoadCount { get; private set; }

        /// <summary>Last scene requested through this router.</summary>
        public static string LastLoaded { get; private set; }

        /// <summary>
        /// Loads <paramref name="sceneName"/> unless it is already active.
        /// Returns true when a transition was performed.
        /// </summary>
        public static bool Load(string sceneName)
        {
            if (string.IsNullOrEmpty(sceneName))
            {
                return false;
            }

            if (SceneManager.GetActiveScene().name == sceneName)
            {
                return false;
            }

            LoadCount++;
            LastLoaded = sceneName;
            SceneManager.LoadScene(sceneName);
            return true;
        }

        /// <summary>Test hook: clears the counters without touching the scene.</summary>
        public static void ResetForTests()
        {
            LoadCount = 0;
            LastLoaded = null;
        }
    }
}
