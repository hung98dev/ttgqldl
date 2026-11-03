using System.IO;
using Ttgqldl.Bootstrap;
using Ttgqldl.Presentation;
using Ttgqldl.UI;
using UnityEditor;
using UnityEditor.Build;
using UnityEditor.Events;
using UnityEditor.SceneManagement;
using UnityEngine;
using UnityEngine.EventSystems;
using UnityEngine.UI;

namespace Ttgqldl.Editor
{
    /// <summary>
    /// Builds the three approved scenes from code so the layout constants in
    /// <see cref="HudLayoutConfig"/> are the single source of truth and the scene
    /// files cannot drift from the documented safe areas.
    ///
    /// Run with: Unity -batchmode -projectPath unity -executeMethod
    /// Ttgqldl.Editor.ProjectSetup.Configure -quit
    /// </summary>
    public static class SceneFactory
    {
        public const string ScenesFolder = "Assets/_TTGQLDL/Scenes";
        public const string BootstrapPath = ScenesFolder + "/00_Bootstrap.unity";
        public const string BroadcastPath = ScenesFolder + "/01_BroadcastWorld.unity";
        public const string RecoveryPath = ScenesFolder + "/02_OfflineRecoveryPreview.unity";

        private static readonly Color Background = new Color(0.055f, 0.062f, 0.086f, 1f);
        private static readonly Color PanelColor = new Color(0.10f, 0.11f, 0.16f, 0.85f);
        private static readonly Color TextColor = new Color(0.96f, 0.96f, 0.98f, 1f);
        private static readonly Color WarningColor = new Color(1f, 0.86f, 0.35f, 1f);

        private static Font _font;

        public static void CreateAll()
        {
            EnsureFolder(ScenesFolder);
            CreateBootstrapScene();
            CreateBroadcastScene();
            CreateRecoveryScene();
            AssetDatabase.SaveAssets();
            AssetDatabase.Refresh();
        }

        private static void CreateBootstrapScene()
        {
            var scene = EditorSceneManager.NewScene(NewSceneSetup.EmptyScene, NewSceneMode.Single);
            CreateCamera();
            CreateEventSystem();

            var go = new GameObject("Bootstrap");
            go.AddComponent<BootstrapSceneController>();

            EditorSceneManager.SaveScene(scene, BootstrapPath);
        }

        private static void CreateRecoveryScene()
        {
            var scene = EditorSceneManager.NewScene(NewSceneSetup.EmptyScene, NewSceneMode.Single);
            CreateCamera();
            CreateEventSystem();

            var canvas = CreateCanvas();
            var safe = CreateSafeArea(canvas.transform);

            var panel = CreatePanel(safe, "RecoveryPanel", new NormalizedRect(0.05f, 0.05f, 0.82f, 0.88f), PanelColor);
            var text = CreateText(panel, "DiagnosticsText", new Vector2(0.04f, 0.20f), new Vector2(0.96f, 0.96f),
                22, TextAnchor.UpperLeft, "OFFLINE RECOVERY PREVIEW");
            text.horizontalOverflow = HorizontalWrapMode.Wrap;
            text.verticalOverflow = VerticalWrapMode.Overflow;

            var buttonRoot = new GameObject("ReturnButton", typeof(RectTransform), typeof(Image), typeof(Button));
            buttonRoot.transform.SetParent(panel, false);
            var buttonRect = (RectTransform)buttonRoot.transform;
            buttonRect.anchorMin = new Vector2(0.04f, 0.05f);
            buttonRect.anchorMax = new Vector2(0.60f, 0.16f);
            buttonRect.offsetMin = Vector2.zero;
            buttonRect.offsetMax = Vector2.zero;
            buttonRoot.GetComponent<Image>().color = new Color(0.16f, 0.22f, 0.34f, 1f);
            var buttonLabel = CreateText((RectTransform)buttonRoot.transform, "Label",
                Vector2.zero, Vector2.one, 24, TextAnchor.MiddleCenter, "RETURN TO BROADCAST");

            var controller = new GameObject("RecoveryController").AddComponent<OfflineRecoveryPreviewController>();
            controller.EditorBind(text);
            var button = buttonRoot.GetComponent<Button>();
            UnityEventTools.AddPersistentListener(button.onClick, controller.ReturnToBroadcast);
            button.targetGraphic = buttonRoot.GetComponent<Image>();
            _ = buttonLabel;

            EditorSceneManager.SaveScene(scene, RecoveryPath);
        }

        private static void CreateBroadcastScene()
        {
            var scene = EditorSceneManager.NewScene(NewSceneSetup.EmptyScene, NewSceneMode.Single);
            CreateCamera();
            CreateEventSystem();

            var canvas = CreateCanvas();
            var safe = CreateSafeArea(canvas.transform);

            var topHud = CreatePanel(safe, "TopHud", HudLayoutConfig.TopHud, PanelColor);
            var water = CreateText(topHud, "WaveText", new Vector2(0f, 0.60f), new Vector2(0.55f, 1f),
                40, TextAnchor.MiddleLeft, "WAVE --");
            var timer = CreateText(topHud, "TimerText", new Vector2(0.55f, 0.60f), new Vector2(1f, 1f),
                40, TextAnchor.MiddleRight, "--:--");
            var phase = CreateText(topHud, "PhaseText", new Vector2(0f, 0.30f), new Vector2(0.55f, 0.60f),
                32, TextAnchor.MiddleLeft, "CHUẨN BỊ");
            var lifecycle = CreateText(topHud, "LifecycleText", new Vector2(0f, 0f), new Vector2(0.55f, 0.30f),
                24, TextAnchor.MiddleLeft, "WAITING");
            var status = CreateText(topHud, "StatusText", new Vector2(0.55f, 0.30f), new Vector2(1f, 0.60f),
                28, TextAnchor.MiddleRight, "MOCK");
            var stale = CreateText(topHud, "StaleIndicator", new Vector2(0.55f, 0f), new Vector2(1f, 0.30f),
                24, TextAnchor.MiddleRight, "STALE");
            stale.color = WarningColor;

            var world = CreatePanel(safe, "World", HudLayoutConfig.World, PanelColor);
            var connection = CreateText(world, "ConnectionBanner", new Vector2(0f, 0.94f), new Vector2(1f, 1f),
                22, TextAnchor.MiddleCenter, "");
            connection.color = WarningColor;
            var wall = CreateText(world, "WallText", new Vector2(0f, 0.84f), new Vector2(1f, 0.92f),
                30, TextAnchor.MiddleCenter, "TƯỜNG --/--");
            var hall = CreateText(world, "TownhallText", new Vector2(0f, 0.76f), new Vector2(1f, 0.84f),
                28, TextAnchor.MiddleCenter, "NHÀ CHÍNH --/--");
            var lanes = new Text[3];
            for (var i = 0; i < 3; i++)
            {
                var yMax = 0.64f - (0.10f * i);
                lanes[i] = CreateText(world, "LaneText" + i,
                    new Vector2(0f, yMax - 0.08f), new Vector2(1f, yMax),
                    26, TextAnchor.MiddleCenter, "LANE");
            }

            var badges = new GameObject[3];
            var badgeNames = new[] { "ClearBadge", "VictoryBadge", "DefeatBadge" };
            var badgeTexts = new[] { "CLEAR", "VICTORY", "DEFEAT" };
            for (var i = 0; i < badges.Length; i++)
            {
                var text = CreateText(world, badgeNames[i], new Vector2(0.10f, 0.36f), new Vector2(0.90f, 0.42f),
                    34, TextAnchor.MiddleCenter, badgeTexts[i]);
                text.color = i == 2 ? new Color(1f, 0.55f, 0.55f, 1f) : WarningColor;
                badges[i] = text.gameObject;
                badges[i].SetActive(false);
            }

            var votePanel = CreatePanel(world, "VotePanel", new NormalizedRect(0f, 0.20f, 1f, 0.34f),
                new Color(0.14f, 0.16f, 0.22f, 0.9f));
            var vote = CreateText(votePanel, "VoteText", Vector2.zero, Vector2.one, 24, TextAnchor.MiddleCenter, "");
            votePanel.gameObject.SetActive(false);

            var resources = CreateText(world, "ResourcesText", new Vector2(0f, 0.02f), new Vector2(1f, 0.10f),
                24, TextAnchor.MiddleCenter, "GỖ --   ĐÁ --   LƯƠNG --");
            var spotlight = CreateText(world, "SpotlightText", new Vector2(0f, 0.10f), new Vector2(1f, 0.18f),
                22, TextAnchor.MiddleCenter, "");
            var banner = CreateText(world, "BannerText", new Vector2(0f, 0.92f), new Vector2(1f, 1f),
                20, TextAnchor.MiddleCenter, "");
            banner.gameObject.SetActive(false);

            var commands = CreatePanel(safe, "Commands", HudLayoutConfig.Commands,
                new Color(0.12f, 0.15f, 0.20f, 0.95f));
            var roles = CreateText(commands, "RoleCountsText", Vector2.zero, Vector2.one,
                34, TextAnchor.MiddleCenter, "XÂY --   THỦ --   ĐÁNH --");

            var watermark = CreateText(safe, "MockWatermark",
                new Vector2(HudLayoutConfig.Watermark.XMin, HudLayoutConfig.Watermark.YMin),
                new Vector2(HudLayoutConfig.Watermark.XMax, HudLayoutConfig.Watermark.YMax),
                22, TextAnchor.MiddleLeft, "MOCK — OFFLINE FIXTURE");
            watermark.color = WarningColor;

            var hudRoot = new GameObject("HudView");
            hudRoot.transform.SetParent(safe, false);
            var hud = hudRoot.AddComponent<HudView>();
            hud.EditorBind(water, timer, phase, lifecycle, status, wall, hall, resources, roles,
                lanes, votePanel.gameObject, vote, spotlight, banner, watermark, stale, connection,
                badges[0], badges[1], badges[2]);

            var controller = new GameObject("Broadcast").AddComponent<BroadcastSceneController>();
            controller.EditorBind(hud);

            EditorSceneManager.SaveScene(scene, BroadcastPath);
        }

        // --- primitives -----------------------------------------------------

        private static void CreateCamera()
        {
            var go = new GameObject("Main Camera", typeof(Camera));
            go.tag = "MainCamera";
            var camera = go.GetComponent<Camera>();
            camera.orthographic = true;
            camera.orthographicSize = 5f;
            camera.clearFlags = CameraClearFlags.SolidColor;
            camera.backgroundColor = Background;
            go.transform.position = new Vector3(0f, 0f, -10f);
        }

        private static void CreateEventSystem()
        {
            var go = new GameObject("EventSystem", typeof(EventSystem), typeof(StandaloneInputModule));
            go.transform.position = Vector3.zero;
        }

        private static Canvas CreateCanvas()
        {
            var go = new GameObject("Canvas", typeof(Canvas), typeof(CanvasScaler), typeof(GraphicRaycaster));
            var canvas = go.GetComponent<Canvas>();
            canvas.renderMode = RenderMode.ScreenSpaceOverlay;

            var scaler = go.GetComponent<CanvasScaler>();
            scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
            scaler.referenceResolution = new Vector2(HudLayoutConfig.ReferenceWidth, HudLayoutConfig.ReferenceHeight);
            scaler.screenMatchMode = CanvasScaler.ScreenMatchMode.MatchWidthOrHeight;
            scaler.matchWidthOrHeight = HudLayoutConfig.CanvasMatch;
            return canvas;
        }

        private static RectTransform CreateSafeArea(Transform parent)
        {
            var go = new GameObject("SafeArea", typeof(RectTransform), typeof(SafeAreaPanel));
            go.transform.SetParent(parent, false);
            var rect = (RectTransform)go.transform;
            rect.anchorMin = Vector2.zero;
            rect.anchorMax = Vector2.one;
            rect.offsetMin = Vector2.zero;
            rect.offsetMax = Vector2.zero;
            return rect;
        }

        private static RectTransform CreatePanel(Transform parent, string name, NormalizedRect rect, Color color)
        {
            var go = new GameObject(name, typeof(RectTransform), typeof(Image));
            go.transform.SetParent(parent, false);
            var rt = (RectTransform)go.transform;
            rt.anchorMin = new Vector2(rect.XMin, rect.YMin);
            rt.anchorMax = new Vector2(rect.XMax, rect.YMax);
            rt.offsetMin = Vector2.zero;
            rt.offsetMax = Vector2.zero;
            go.GetComponent<Image>().color = color;
            return rt;
        }

        private static Text CreateText(Transform parent, string name, Vector2 anchorMin, Vector2 anchorMax,
            int fontSize, TextAnchor alignment, string content)
        {
            var go = new GameObject(name, typeof(RectTransform), typeof(Text));
            go.transform.SetParent(parent, false);
            var rect = (RectTransform)go.transform;
            rect.anchorMin = anchorMin;
            rect.anchorMax = anchorMax;
            rect.offsetMin = Vector2.zero;
            rect.offsetMax = Vector2.zero;

            var text = go.GetComponent<Text>();
            text.font = Font();
            text.fontSize = fontSize;
            text.alignment = alignment;
            text.color = TextColor;
            text.text = content;
            text.raycastTarget = false;
            text.horizontalOverflow = HorizontalWrapMode.Overflow;
            text.verticalOverflow = VerticalWrapMode.Overflow;
            return text;
        }

        private static Font Font()
        {
            if (_font != null)
            {
                return _font;
            }

            _font = Resources.GetBuiltinResource<Font>("LegacyRuntime.ttf");
            if (_font == null)
            {
                _font = Resources.GetBuiltinResource<Font>("Arial.ttf");
            }

            return _font;
        }

        private static void EnsureFolder(string path)
        {
            if (AssetDatabase.IsValidFolder(path))
            {
                return;
            }

            var parent = Path.GetDirectoryName(path)?.Replace('\\', '/');
            var leaf = Path.GetFileName(path);
            if (!string.IsNullOrEmpty(parent) && !AssetDatabase.IsValidFolder(parent))
            {
                EnsureFolder(parent);
            }

            AssetDatabase.CreateFolder(parent ?? "Assets", leaf);
        }

        /// <summary>Named build targets helper kept here so the build script shares it.</summary>
        public static NamedBuildTarget StandaloneTarget => NamedBuildTarget.Standalone;
    }
}
