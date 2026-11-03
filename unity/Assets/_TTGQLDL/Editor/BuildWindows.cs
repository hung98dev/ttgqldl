using System;
using System.IO;
using System.Linq;
using UnityEditor;
using UnityEditor.Build.Reporting;
using UnityEngine;

namespace Ttgqldl.Editor
{
    /// <summary>
    /// Windows x86_64 standalone build entry point.
    ///
    /// Batch usage (tooling/unity-build.ps1|sh):
    ///   Unity -batchmode -quit -projectPath unity -executeMethod
    ///   Ttgqldl.Editor.BuildWindows.PerformBuild -logFile Build/unity-build.log
    ///
    /// The process exit code is 0 only when Unity reports BuildResult.Succeeded.
    /// </summary>
    public static class BuildWindows
    {
        public const string OutputDirectory = "Build/Windows";
        public const string ExecutableName = "TTGQLDL.exe";
        public const int ExpectedSceneCount = 3;

        [MenuItem("TTGQLDL/Build Windows Player")]
        public static void PerformBuild()
        {
            try
            {
                var scenes = EditorBuildSettings.scenes
                    .Where(scene => scene.enabled)
                    .Select(scene => scene.path)
                    .ToArray();

                if (scenes.Length != ExpectedSceneCount)
                {
                    throw new InvalidOperationException(
                        "expected " + ExpectedSceneCount + " enabled build scenes, found " + scenes.Length);
                }

                var outputPath = Path.Combine(OutputDirectory, ExecutableName);
                Directory.CreateDirectory(OutputDirectory);

                Debug.Log("[TTGQLDL] build start: unity=" + Application.unityVersion +
                          " target=StandaloneWindows64 scenes=" + string.Join(",", scenes) +
                          " output=" + Path.GetFullPath(outputPath));

                var options = new BuildPlayerOptions
                {
                    scenes = scenes,
                    locationPathName = outputPath,
                    target = BuildTarget.StandaloneWindows64,
                    targetGroup = BuildTargetGroup.Standalone,
                    options = BuildOptions.None,
                };

                var report = BuildPipeline.BuildPlayer(options);
                var summary = report.summary;

                Debug.Log("[TTGQLDL] build result=" + summary.result +
                          " errors=" + summary.totalErrors +
                          " warnings=" + summary.totalWarnings +
                          " sizeBytes=" + summary.totalSize +
                          " durationSeconds=" + summary.totalTime.TotalSeconds.ToString("0.0"));

                if (summary.result != BuildResult.Succeeded)
                {
                    throw new InvalidOperationException("build did not succeed: " + summary.result);
                }

                if (!File.Exists(outputPath))
                {
                    throw new InvalidOperationException("build reported success but the player is missing: " + outputPath);
                }

                Debug.Log("[TTGQLDL] build succeeded: " + Path.GetFullPath(outputPath));
                ExitBatch(0);
            }
            catch (Exception exception)
            {
                Debug.LogError("[TTGQLDL] build failed: " + exception);
                ExitBatch(1);
                throw;
            }
        }

        private static void ExitBatch(int exitCode)
        {
            if (Application.isBatchMode)
            {
                EditorApplication.Exit(exitCode);
            }
        }
    }
}
