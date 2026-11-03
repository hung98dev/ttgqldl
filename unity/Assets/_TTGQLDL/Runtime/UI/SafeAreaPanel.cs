using Ttgqldl.Presentation;
using UnityEngine;

namespace Ttgqldl.UI
{
    /// <summary>
    /// Fits a RectTransform to the device safe area and to the platform chrome
    /// reserves documented in docs/plan/07_UNITY_ART_UX.md (right 18%, bottom
    /// 12%). It runs as a canvas transition, never as a scene load.
    /// </summary>
    [RequireComponent(typeof(RectTransform))]
    public sealed class SafeAreaPanel : MonoBehaviour
    {
        private RectTransform _rect;
        private Rect _lastSafeArea;
        private int _lastWidth;
        private int _lastHeight;

        private void Awake()
        {
            _rect = (RectTransform)transform;
            ApplyIfChanged();
        }

        private void Update()
        {
            ApplyIfChanged();
        }

        private void ApplyIfChanged()
        {
            var safeArea = Screen.safeArea;
            if (_rect == null || (safeArea == _lastSafeArea && Screen.width == _lastWidth && Screen.height == _lastHeight))
            {
                return;
            }

            _lastSafeArea = safeArea;
            _lastWidth = Screen.width;
            _lastHeight = Screen.height;
            ApplyAnchors(_rect, safeArea, Screen.width, Screen.height);
        }

        /// <summary>
        /// Pure anchor computation, separated so it can be tested without a
        /// running player. Returns anchorMin and anchorMax in 0..1 canvas space.
        /// </summary>
        public static void ComputeAnchors(Rect safeArea, int screenWidth, int screenHeight,
            out Vector2 anchorMin, out Vector2 anchorMax)
        {
            if (screenWidth <= 0 || screenHeight <= 0)
            {
                anchorMin = Vector2.zero;
                anchorMax = Vector2.one;
                return;
            }

            var xMin = Mathf.Clamp01(safeArea.xMin / screenWidth);
            var yMin = Mathf.Clamp01(safeArea.yMin / screenHeight);
            var xMax = Mathf.Clamp01(safeArea.xMax / screenWidth);
            var yMax = Mathf.Clamp01(safeArea.yMax / screenHeight);

            // Never expand past the platform reserves, whatever the device reports.
            xMax = Mathf.Min(xMax, HudLayoutConfig.SafeArea.XMax);
            yMax = Mathf.Min(yMax, HudLayoutConfig.SafeArea.YMax);

            if (xMax <= xMin)
            {
                xMax = Mathf.Min(1f, xMin + 0.01f);
            }

            if (yMax <= yMin)
            {
                yMax = Mathf.Min(1f, yMin + 0.01f);
            }

            anchorMin = new Vector2(xMin, yMin);
            anchorMax = new Vector2(xMax, yMax);
        }

        private static void ApplyAnchors(RectTransform rect, Rect safeArea, int screenWidth, int screenHeight)
        {
            ComputeAnchors(safeArea, screenWidth, screenHeight, out var min, out var max);
            rect.anchorMin = min;
            rect.anchorMax = max;
            rect.offsetMin = Vector2.zero;
            rect.offsetMax = Vector2.zero;
        }
    }
}
