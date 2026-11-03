using System;
using System.Linq;
using NUnit.Framework;
using Ttgqldl.Bootstrap;
using Ttgqldl.Core;
using Ttgqldl.Networking;
using Ttgqldl.Presentation;
using Ttgqldl.UI;
using UnityEditor;
using UnityEngine;

namespace Ttgqldl.Tests.EditMode
{
    /// <summary>
    /// Composition-root behaviour, assembly boundaries and the locked project
    /// configuration. These are the checks that stop the client from growing an
    /// accidental second runtime or an editor dependency in runtime code.
    /// </summary>
    public sealed class RuntimeAndProjectTests
    {
        private sealed class FakeSource : IPublicStateSource
        {
            public event Action<PublicSnapshot> SnapshotPublished;
            public event Action<ConnectionState> ConnectionChanged;

            public ConnectionState Connection { get; private set; } = ConnectionState.Disconnected;
            public bool IsAvailable { get; private set; }

            public void Start()
            {
                IsAvailable = true;
            }

            public void Stop()
            {
                IsAvailable = false;
            }

            public void Publish(PublicSnapshot snapshot) => SnapshotPublished?.Invoke(snapshot);

            public void SetConnection(ConnectionState state)
            {
                Connection = state;
                ConnectionChanged?.Invoke(state);
            }
        }

        [SetUp]
        public void SetUp()
        {
            TtgqldlRuntime.ResetForTests();
        }

        [TearDown]
        public void TearDown()
        {
            TtgqldlRuntime.ResetForTests();
        }

        [Test]
        public void InitializeTwiceReturnsTheSameRuntime()
        {
            var first = TtgqldlRuntime.Initialize(new FakeSource());
            var second = TtgqldlRuntime.Initialize(new FakeSource());

            Assert.AreSame(first, second);
            Assert.AreEqual(1, TtgqldlRuntime.InitializeCount);
        }

        [Test]
        public void ValidSnapshotBecomesTheCurrentView()
        {
            var source = new FakeSource();
            var runtime = TtgqldlRuntime.Initialize(source);
            source.Publish(MockFixtures.CreateDefault());

            Assert.IsTrue(runtime.HasValidSnapshot);
            Assert.AreEqual("WAVE 3", runtime.CurrentView.WaveLabel);
            Assert.IsTrue(runtime.CurrentView.MockWatermarkVisible);
            StringAssert.Contains("XÂY 12", runtime.CurrentView.RoleCountsLabel);
        }

        [Test]
        public void InvalidSnapshotKeepsLastValidViewAndFlagsStale()
        {
            var source = new FakeSource();
            var runtime = TtgqldlRuntime.Initialize(source);
            source.Publish(MockFixtures.CreateDefault());
            Assert.IsTrue(runtime.HasValidSnapshot);

            // A snapshot with a negative resource must be refused, not displayed.
            var broken = new PublicSnapshot(1, "broken", 3, VillageLifecycle.Running, new string[0],
                Phase.Prep, 100, 1000, 1000, 500, 500, -1, 250, 300,
                new RoleCounts(1, 1, 1, 3),
                new[] { new LaneState(LaneId.West, 0, 0), new LaneState(LaneId.Center, 0, 0), new LaneState(LaneId.East, 0, 0) },
                new VoteState(false, new[] { "A", "B", "C" }, 0, 0, 0, null),
                SourceStatus.Mock, new SpotlightEntry[0], null, false);
            source.Publish(broken);

            Assert.IsTrue(runtime.HasValidSnapshot, "the last valid snapshot must be retained");
            Assert.AreEqual("WAVE 3", runtime.CurrentView.WaveLabel);
            Assert.IsTrue(runtime.CurrentView.StaleIndicatorVisible, "an invalid update must be visibly stale");
            Assert.AreEqual("STALE", runtime.CurrentView.StatusLabel);
            Assert.IsNotEmpty(runtime.Diagnostics.LastError);
        }

        [Test]
        public void NoSnapshotYetShowsNoDataInsteadOfFakeState()
        {
            var runtime = TtgqldlRuntime.Initialize(new FakeSource());
            Assert.IsFalse(runtime.HasValidSnapshot);
            Assert.AreEqual("NO DATA", runtime.CurrentView.PhaseLabel);
            Assert.IsFalse(runtime.CurrentView.VictoryVisible);
        }

        [Test]
        public void SubscribersAreTrackedAndRemoved()
        {
            var runtime = TtgqldlRuntime.Initialize(new FakeSource());
            Action<HudViewModel> handler = _ => { };

            runtime.ViewChanged += handler;
            runtime.ViewChanged += handler;
            Assert.AreEqual(2, runtime.SubscriberCount);

            runtime.ViewChanged -= handler;
            Assert.AreEqual(1, runtime.SubscriberCount);
        }

        [Test]
        public void ShutdownStopsDeliveringEvents()
        {
            var source = new FakeSource();
            var runtime = TtgqldlRuntime.Initialize(source);
            var received = 0;
            runtime.ViewChanged += _ => received++;

            runtime.Shutdown();
            var before = received;
            source.Publish(MockFixtures.CreateDefault());

            Assert.AreEqual(before, received);
        }

        [Test]
        public void CoreAssemblyHasNoEngineOrPresentationDependency()
        {
            var references = typeof(PublicSnapshot).Assembly.GetReferencedAssemblies()
                .Select(name => name.Name ?? string.Empty).ToArray();

            Assert.IsFalse(references.Any(name => name.StartsWith("UnityEngine")),
                "Core must not reference UnityEngine: " + string.Join(",", references));
            Assert.IsFalse(references.Any(name => name.StartsWith("Ttgqldl.")),
                "Core must not reference another Ttgqldl assembly: " + string.Join(",", references));
        }

        [Test]
        public void PresentationAssemblyDoesNotReferenceUI()
        {
            var references = typeof(PresentationMapper).Assembly.GetReferencedAssemblies()
                .Select(name => name.Name ?? string.Empty).ToArray();

            Assert.IsFalse(references.Any(name => name == "Ttgqldl.UI"), "Presentation must not reference UI");
            Assert.IsFalse(references.Any(name => name.StartsWith("UnityEngine")),
                "Presentation must stay engine-free: " + string.Join(",", references));
        }

        [Test]
        public void BuildSettingsContainTheThreeApprovedScenes()
        {
            var scenes = EditorBuildSettings.scenes.Where(scene => scene.enabled).Select(scene => scene.path).ToArray();
            Assert.AreEqual(3, scenes.Length);
            CollectionAssert.Contains(scenes, "Assets/_TTGQLDL/Scenes/00_Bootstrap.unity");
            CollectionAssert.Contains(scenes, "Assets/_TTGQLDL/Scenes/01_BroadcastWorld.unity");
            CollectionAssert.Contains(scenes, "Assets/_TTGQLDL/Scenes/02_OfflineRecoveryPreview.unity");
        }

        [Test]
        public void ProjectIsLockedToPortraitGammaForceText()
        {
            Assert.AreEqual("TTGQLDL", PlayerSettings.productName);
            Assert.AreEqual(ColorSpace.Gamma, PlayerSettings.colorSpace);
            Assert.AreEqual(UIOrientation.Portrait, PlayerSettings.defaultInterfaceOrientation);
            Assert.AreEqual(SerializationMode.ForceText, EditorSettings.serializationMode);
        }

        [Test]
        public void SceneAssetsExistOnDisk()
        {
            foreach (var path in new[]
                     {
                         "Assets/_TTGQLDL/Scenes/00_Bootstrap.unity",
                         "Assets/_TTGQLDL/Scenes/01_BroadcastWorld.unity",
                         "Assets/_TTGQLDL/Scenes/02_OfflineRecoveryPreview.unity",
                     })
            {
                Assert.IsNotNull(AssetDatabase.LoadAssetAtPath<SceneAsset>(path), "missing scene: " + path);
                Assert.IsTrue(System.IO.File.Exists(path + ".meta"), "scene has no .meta: " + path);
            }
        }
    }
}
