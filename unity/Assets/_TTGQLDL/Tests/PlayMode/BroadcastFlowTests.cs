using System.Collections;
using NUnit.Framework;
using Ttgqldl.Bootstrap;
using Ttgqldl.Core;
using Ttgqldl.Networking;
using Ttgqldl.UI;
using UnityEngine;
using UnityEngine.SceneManagement;
using UnityEngine.TestTools;

namespace Ttgqldl.Tests.PlayMode
{
    /// <summary>
    /// End-to-end scene behaviour with the offline fixture only. No network, no
    /// TikTok and no production endpoint is touched.
    /// </summary>
    public sealed class BroadcastFlowTests
    {
        private const string BootstrapScene = "00_Bootstrap";
        private const string BroadcastScene = "01_BroadcastWorld";

        [SetUp]
        public void SetUp()
        {
            TtgqldlRuntime.ResetForTests();
            SceneRouter.ResetForTests();
        }

        [TearDown]
        public void TearDown()
        {
            TtgqldlRuntime.ResetForTests();
            SceneRouter.ResetForTests();
        }

        [UnityTest]
        public IEnumerator BootstrapLoadsBroadcastSceneWithFixtureData()
        {
            SceneManager.LoadScene(BootstrapScene);
            yield return null;
            yield return null;
            yield return null;

            Assert.IsNotNull(TtgqldlRuntime.Instance, "bootstrap must create the runtime");
            Assert.IsTrue(TtgqldlRuntime.Instance.HasValidSnapshot, "the offline fixture must publish a snapshot");
            Assert.AreEqual(BroadcastScene, SceneManager.GetActiveScene().name,
                "bootstrap must hand over to the long-lived broadcast scene");

            var hud = Object.FindAnyObjectByType<HudView>();
            Assert.IsNotNull(hud, "broadcast scene must contain the HUD");
            Assert.IsTrue(hud.HasCompleteBindings(), "every HUD reference must be wired");
            Assert.IsNotNull(hud.Current, "the HUD must have received a view model");
            Assert.AreEqual("MOCK", hud.Current.StatusLabel);
            Assert.IsTrue(hud.Current.MockWatermarkVisible, "the watermark must be visible for fixture data");
        }

        [UnityTest]
        public IEnumerator BroadcastSceneHasNoMissingScriptReferences()
        {
            SceneManager.LoadScene(BroadcastScene);
            yield return null;
            yield return null;

            var roots = SceneManager.GetActiveScene().GetRootGameObjects();
            Assert.IsNotEmpty(roots);
            foreach (var root in roots)
            {
                var components = root.GetComponentsInChildren<Component>(true);
                foreach (var component in components)
                {
                    Assert.IsNotNull(component,
                        "missing script reference under " + root.name);
                }
            }
        }

        [UnityTest]
        public IEnumerator ConnectionStateChangesDoNotReloadTheScene()
        {
            SceneManager.LoadScene(BroadcastScene);
            yield return null;
            yield return null;

            var runtime = TtgqldlRuntime.Instance;
            Assert.IsNotNull(runtime);
            var source = runtime.Source as OfflineFixtureSource;
            Assert.IsNotNull(source, "the M01 runtime must be driven by the offline fixture");
            var loadsBefore = SceneRouter.LoadCount;
            var sceneBefore = SceneManager.GetActiveScene().name;

            source.SetConnection(ConnectionState.Stale);
            yield return null;
            var hud = Object.FindAnyObjectByType<HudView>();
            Assert.IsNotNull(hud);
            Assert.AreEqual("STALE", hud.Current.StatusLabel);

            source.SetConnection(ConnectionState.Disconnected);
            yield return null;
            Assert.AreEqual("DISCONNECTED", hud.Current.StatusLabel);
            Assert.IsTrue(hud.Current.DisconnectedBannerVisible);

            source.SetConnection(ConnectionState.Mock);
            yield return null;
            Assert.AreEqual("MOCK", hud.Current.StatusLabel);

            Assert.AreEqual(loadsBefore, SceneRouter.LoadCount,
                "connection state changes must never reload the broadcast scene");
            Assert.AreEqual(sceneBefore, SceneManager.GetActiveScene().name);
        }

        [UnityTest]
        public IEnumerator PausedLifecycleIsABannerNotASceneReload()
        {
            SceneManager.LoadScene(BroadcastScene);
            yield return null;
            yield return null;

            var runtime = TtgqldlRuntime.Instance;
            var loadsBefore = SceneRouter.LoadCount;

            // Recompute with the same fixture; the lifecycle banner is driven by the
            // published snapshot, and reloading must never be the response.
            runtime.Recompute();
            yield return null;

            Assert.AreEqual(loadsBefore, SceneRouter.LoadCount);
            Assert.IsNotNull(Object.FindAnyObjectByType<HudView>());
        }

        [UnityTest]
        public IEnumerator SecondInitializeDoesNotDuplicateTheRuntime()
        {
            SceneManager.LoadScene(BootstrapScene);
            yield return null;
            yield return null;
            yield return null;

            var existing = TtgqldlRuntime.Instance;
            var again = TtgqldlRuntime.Initialize(new OfflineFixtureSource());

            Assert.AreSame(existing, again);
            Assert.AreEqual(1, TtgqldlRuntime.InitializeCount);
        }

        [UnityTest]
        public IEnumerator DestroyingTheControllerReleasesTheRuntimeSubscription()
        {
            SceneManager.LoadScene(BroadcastScene);
            yield return null;
            yield return null;

            var controller = Object.FindAnyObjectByType<BroadcastSceneController>();
            Assert.IsNotNull(controller);
            var runtime = TtgqldlRuntime.Instance;
            Assert.Greater(runtime.SubscriberCount, 0);

            Object.Destroy(controller.gameObject);
            yield return null;

            Assert.AreEqual(0, runtime.SubscriberCount,
                "the controller must unsubscribe when destroyed");
        }

        [UnityTest]
        public IEnumerator RecoveryPreviewOpensAndRendersDiagnostics()
        {
            SceneManager.LoadScene("02_OfflineRecoveryPreview");
            yield return null;
            yield return null;

            var controller = Object.FindAnyObjectByType<OfflineRecoveryPreviewController>();
            Assert.IsNotNull(controller);
            StringAssert.Contains("OFFLINE RECOVERY PREVIEW", controller.LastRendered);
            StringAssert.Contains("diagnostic", controller.LastRendered.ToLowerInvariant());
        }
    }
}
