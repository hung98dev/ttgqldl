using System;

namespace Ttgqldl.Presentation
{
    /// <summary>A rectangle in normalised canvas coordinates (0..1, origin bottom-left).</summary>
    public readonly struct NormalizedRect
    {
        public NormalizedRect(float xMin, float yMin, float xMax, float yMax)
        {
            XMin = xMin;
            YMin = yMin;
            XMax = xMax;
            YMax = yMax;
        }

        public float XMin { get; }
        public float YMin { get; }
        public float XMax { get; }
        public float YMax { get; }
        public float Width => XMax - XMin;
        public float Height => YMax - YMin;

        public bool IsValid =>
            XMin >= 0f && YMin >= 0f && XMax <= 1f && YMax <= 1f && XMax > XMin && YMax > YMin;

        public bool Contains(float x, float y) => x >= XMin && x <= XMax && y >= YMin && y <= YMax;

        /// <summary>True when this rectangle is fully inside <paramref name="outer"/>.</summary>
        public bool IsWithin(NormalizedRect outer) =>
            XMin >= outer.XMin && YMin >= outer.YMin && XMax <= outer.XMax && YMax <= outer.YMax;

        public override string ToString() =>
            "[" + XMin.ToString("0.###") + "," + YMin.ToString("0.###") + " → " +
            XMax.ToString("0.###") + "," + YMax.ToString("0.###") + "]";
    }

    /// <summary>
    /// The documented portrait layout contract from docs/plan/07_UNITY_ART_UX.md.
    ///
    /// Reference canvas is 1080x1920 with CanvasScaler match 0.5; the right
    /// 18% and the bottom 12% stay free for TikTok chrome until a real OBS
    /// overlay screenshot proves otherwise. These are layout numbers only and
    /// apply identically at 1080x1920 and 360x640.
    /// </summary>
    public static class HudLayoutConfig
    {
        public const float ReferenceWidth = 1080f;
        public const float ReferenceHeight = 1920f;
        public const float MinimumWidth = 360f;
        public const float MinimumHeight = 640f;

        /// <summary>CanvasScaler.matchWidthOrHeight value from the plan.</summary>
        public const float CanvasMatch = 0.5f;

        /// <summary>Minimum text sizes at 360x640, in reference-independent px.</summary>
        public const int MinimumCommandFontPx = 16;
        public const int MinimumPrimaryFontPx = 14;
        public const int MinimumSecondaryFontPx = 12;

        /// <summary>Reserved for platform chrome; no essential HUD element may enter it.</summary>
        public static readonly NormalizedRect RightReserve = new NormalizedRect(0.82f, 0f, 1f, 1f);
        public static readonly NormalizedRect BottomReserve = new NormalizedRect(0f, 0.88f, 1f, 1f);

        /// <summary>The area the HUD may use after the platform reserves.</summary>
        public static readonly NormalizedRect SafeArea = new NormalizedRect(0f, 0f, 0.82f, 0.88f);

        public static readonly NormalizedRect TopHud = new NormalizedRect(0.05f, 0.05f, 0.82f, 0.14f);
        public static readonly NormalizedRect World = new NormalizedRect(0.05f, 0.20f, 0.82f, 0.74f);
        public static readonly NormalizedRect Commands = new NormalizedRect(0.08f, 0.76f, 0.75f, 0.86f);
        public static readonly NormalizedRect Watermark = new NormalizedRect(0.05f, 0.86f, 0.60f, 0.875f);

        /// <summary>Every rect the HUD draws into, for containment tests.</summary>
        public static readonly NormalizedRect[] HudRects =
        {
            TopHud, World, Commands, Watermark,
        };

        public static bool IsHudRectSafe(NormalizedRect rect) => rect.IsValid && rect.IsWithin(SafeArea);

        public static bool Overlaps(NormalizedRect a, NormalizedRect b) =>
            a.XMin < b.XMax && b.XMin < a.XMax && a.YMin < b.YMax && b.YMin < a.YMax;

        /// <summary>
        /// Converts a normalised rect into anchorMin/anchorMax values for a
        /// RectTransform, leaving a margin expressed in normalised units.
        /// </summary>
        public static void ToAnchors(NormalizedRect rect, out float anchorMinX, out float anchorMinY,
            out float anchorMaxX, out float anchorMaxY)
        {
            if (!rect.IsValid)
            {
                throw new ArgumentOutOfRangeException(nameof(rect), rect.ToString());
            }

            anchorMinX = rect.XMin;
            anchorMinY = rect.YMin;
            anchorMaxX = rect.XMax;
            anchorMaxY = rect.YMax;
        }
    }
}
