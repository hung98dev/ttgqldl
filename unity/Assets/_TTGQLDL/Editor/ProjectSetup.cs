using System;
using System.Linq;
using UnityEditor;
using UnityEditor.Build;
using UnityEngine;

namespace Ttgqldl.Editor
{
    /// <summary>
    /// Locks the project configuration required by docs/plan/07_UNITY_ART_UX.md
    /// and generates the three approved scenes.
    ///
    /// Batch entry point:
    ///   Unity -batchmode -quit -projectPath unity -executeMethod
    ///   Ttgqldl.Editor.ProjectSetup.Configure -logFile -
    /// </summary>
    public static class ProjectSetup
    {
        public const string ProductName = "TTGQLDL";
        public const string CompanyName = "TTGQLDL";
        public const string ApplicationIdentifier = "com.ttgqldl.broadcast";
        public const int ReferenceWidth = 1080;
        public const int ReferenceHeight = 1920;

        [MenuItem("TTGQLDL/Configure Project")]
        public static void Configure()
        {
            try
            {
                ConfigurePlayer();
                ConfigureEditor();
                SceneFactory.CreateAll();
                ConfigureBuildSettings();
                Verify();
                AssetDatabase.SaveAssets();
                AssetDatabase.Refresh();
                Debug.Log("[TTGQLDL] project configuration complete");
                ExitBatch(0);
            }
            catch (Exception exception)
            {
                Debug.LogError("[TTGQLDL] project configuration failed: " + exception);
                ExitBatch(1);
                throw;
            }
        }

        /// <summary>
        /// Gamma colour space is deliberate: the client renders flat 2D UI and
        /// sprite silhouettes with the Built-in pipeline, no lighting or PBR, and
        /// gamma keeps authored UI colours identical to what the operator checks.
        /// Switching to linear would require an art re-check and is not needed
        /// before M06.
        /// </summary>
        public static void ConfigurePlayer()
        {
            PlayerSettings.companyName = CompanyName;
            PlayerSettings.productName = ProductName;
            PlayerSettings.SetApplicationIdentifier(NamedBuildTarget.Standalone, ApplicationIdentifier);
            PlayerSettings.colorSpace = ColorSpace.Gamma;
            PlayerSettings.defaultInterfaceOrientation = UIOrientation.Portrait;
            PlayerSettings.fullScreenMode = FullScreenMode.Windowed;
            PlayerSettings.defaultScreenWidth = ReferenceWidth;
            PlayerSettings.defaultScreenHeight = ReferenceHeight;
            PlayerSettings.resizableWindow = false;
            PlayerSettings.runInBackground = true;
            PlayerSettings.visibleInBackground = true;
            PlayerSettings.allowFullscreenSwitch = false;

            // 9:16 broadcast window: do not letterbox the render into a 16:9 swap.
            PlayerSettings.usePlayerLog = true;
            Debug.Log($"[TTGQLDL] player settings applied ({ReferenceWidth}x{ReferenceHeight}, portrait, Gamma)");
        }

        public static void ConfigureEditor()
        {
            EditorSettings.serializationMode = SerializationMode.ForceText;
            EditorSettings.externalVersionControl = "Visible Meta Files";
            Debug.Log("[TTGQLDL] editor settings applied (ForceText, Visible Meta Files)");
        }

        public static void ConfigureBuildSettings()
        {
            var scenes = new[]
            {
                SceneFactory.BootstrapPath,
                SceneFactory.BroadcastPath,
                SceneFactory.RecoveryPath,
            };

            EditorBuildSettings.scenes = scenes
                .Select(path => new EditorBuildSettingsScene(path, true))
                .ToArray();

            Debug.Log("[TTGQLDL] build settings scenes: " + string.Join(", ", scenes));
        }

        /// <summary>Fails the batch run when a required scene or setting is missing.</summary>
        public static void Verify()
        {
            foreach (var path in new[]
                     {
                         SceneFactory.BootstrapPath, SceneFactory.BroadcastPath, SceneFactory.RecoveryPath,
                     })
            {
                if (AssetDatabase.LoadAssetAtPath<SceneAsset>(path) == null)
                {
                    throw new InvalidOperationException("missing scene asset: " + path);
                }
            }

            var enabled = EditorBuildSettings.scenes.Count(scene => scene.enabled);
            if (enabled != 3)
            {
                throw new InvalidOperationException("expected 3 enabled build scenes, found " + enabled);
            }

            if (PlayerSettings.colorSpace != ColorSpace.Gamma)
            {
                throw new InvalidOperationException("color space must be Gamma for the Built-in 2D pipeline");
            }

            if (PlayerSettings.defaultInterfaceOrientation != UIOrientation.Portrait)
            {
                throw new InvalidOperationException("default orientation must be portrait");
            }

            if (EditorSettings.serializationMode != SerializationMode.ForceText)
            {
                throw new InvalidOperationException("asset serialization must be Force Text");
            }

            Debug.Log($"[TTGQLDL] verified: 3 scenes, Gamma, portrait, ForceText, Unity {Application.unityVersion}");
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
