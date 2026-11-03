using NUnit.Framework;
using Ttgqldl.Presentation;

namespace Ttgqldl.Tests.EditMode
{
    /// <summary>
    /// The layout contract from docs/plan/07_UNITY_ART_UX.md must hold at the
    /// reference resolution and at the 360x640 minimum, and no essential element
    /// may enter the TikTok chrome reserves.
    /// </summary>
    public sealed class HudLayoutTests
    {
        [Test]
        public void ReferenceResolutionIsPortraitNineBySixteen()
        {
            Assert.AreEqual(1080f, HudLayoutConfig.ReferenceWidth);
            Assert.AreEqual(1920f, HudLayoutConfig.ReferenceHeight);
            Assert.AreEqual(360f, HudLayoutConfig.MinimumWidth);
            Assert.AreEqual(640f, HudLayoutConfig.MinimumHeight);
            Assert.AreEqual(0.5f, HudLayoutConfig.CanvasMatch);
        }

        [Test]
        public void EveryHudRectIsValidAndInsideTheSafeArea()
        {
            foreach (var rect in HudLayoutConfig.HudRects)
            {
                Assert.IsTrue(rect.IsValid, "rect is not a normalised rectangle: " + rect);
                Assert.IsTrue(HudLayoutConfig.IsHudRectSafe(rect),
                    "rect leaves the safe area: " + rect);
            }
        }

        [Test]
        public void NoHudRectEntersThePlatformReserves()
        {
            foreach (var rect in HudLayoutConfig.HudRects)
            {
                Assert.IsFalse(HudLayoutConfig.Overlaps(rect, HudLayoutConfig.RightReserve),
                    "rect overlaps the right chrome reserve: " + rect);
                Assert.IsFalse(HudLayoutConfig.Overlaps(rect, HudLayoutConfig.BottomReserve),
                    "rect overlaps the bottom chrome reserve: " + rect);
            }
        }

        [Test]
        public void DocumentedRegionsMatchThePlan()
        {
            // 07_UNITY_ART_UX.md: top HUD 5..14%, commands x 8..75% y 76..86%,
            // reserve right x>82% and bottom y>88%.
            Assert.AreEqual(0.05f, HudLayoutConfig.TopHud.YMin, 0.0001f);
            Assert.AreEqual(0.14f, HudLayoutConfig.TopHud.YMax, 0.0001f);
            Assert.AreEqual(0.08f, HudLayoutConfig.Commands.XMin, 0.0001f);
            Assert.AreEqual(0.75f, HudLayoutConfig.Commands.XMax, 0.0001f);
            Assert.AreEqual(0.76f, HudLayoutConfig.Commands.YMin, 0.0001f);
            Assert.AreEqual(0.86f, HudLayoutConfig.Commands.YMax, 0.0001f);
            Assert.AreEqual(0.82f, HudLayoutConfig.RightReserve.XMin, 0.0001f);
            Assert.AreEqual(0.88f, HudLayoutConfig.BottomReserve.YMin, 0.0001f);
        }

        [Test]
        public void CommandAndPrimaryTextMeetMinimumSizes()
        {
            Assert.GreaterOrEqual(HudLayoutConfig.MinimumCommandFontPx, 16);
            Assert.GreaterOrEqual(HudLayoutConfig.MinimumPrimaryFontPx, 14);
            Assert.GreaterOrEqual(HudLayoutConfig.MinimumSecondaryFontPx, 12);
            Assert.Less(HudLayoutConfig.MinimumSecondaryFontPx, HudLayoutConfig.MinimumPrimaryFontPx);
        }

        [Test]
        public void AnchorsAreDerivedFromTheRect()
        {
            HudLayoutConfig.ToAnchors(HudLayoutConfig.Commands, out var minX, out var minY, out var maxX, out var maxY);
            Assert.AreEqual(0.08f, minX, 0.0001f);
            Assert.AreEqual(0.76f, minY, 0.0001f);
            Assert.AreEqual(0.75f, maxX, 0.0001f);
            Assert.AreEqual(0.86f, maxY, 0.0001f);
        }

        [Test]
        public void InvalidRectIsRejected()
        {
            Assert.Throws<System.ArgumentOutOfRangeException>(() =>
                HudLayoutConfig.ToAnchors(new NormalizedRect(0.5f, 0.5f, 0.4f, 0.6f),
                    out _, out _, out _, out _));
        }

        [Test]
        public void SafeAreaContainsBothEdgeCases()
        {
            // Normalised layout values are resolution independent, so the same
            // contract holds at 1080x1920 and 360x640.
            Assert.IsTrue(HudLayoutConfig.SafeArea.Contains(0.0f, 0.0f));
            Assert.IsTrue(HudLayoutConfig.SafeArea.Contains(0.82f, 0.88f));
            Assert.IsFalse(HudLayoutConfig.SafeArea.Contains(0.9f, 0.5f));
            Assert.IsFalse(HudLayoutConfig.SafeArea.Contains(0.5f, 0.95f));
        }
    }
}
