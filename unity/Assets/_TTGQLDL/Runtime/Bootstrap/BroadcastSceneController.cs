using System;
using Ttgqldl.Networking;
using Ttgqldl.Presentation;
using Ttgqldl.UI;
using UnityEngine;

namespace Ttgqldl.Bootstrap
{
    /// <summary>
    /// Long-lived broadcast scene controller.
    ///
    /// The scene is loaded once and never reloaded for pause, source loss or
    /// storage loss: those are presentation states applied to the existing HUD.
    /// </summary>
    public sealed class BroadcastSceneController : MonoBehaviour
    {
        [SerializeField] private HudView hud;

        private TtgqldlRuntime _runtime;
        private Action<HudViewModel> _onViewChanged;

        public HudView Hud => hud;
        public TtgqldlRuntime Runtime => _runtime;

#if UNITY_EDITOR
        /// <summary>Editor-only wiring used by the scene factory.</summary>
        public void EditorBind(HudView view) => hud = view;
#endif

        private void Start()
        {
            if (hud == null)
            {
                hud = FindAnyObjectByType<HudView>();
            }

            // Opening the broadcast scene directly (for example during an
            // operator preview) must still produce a labelled offline runtime.
            _runtime = TtgqldlRuntime.Instance ?? TtgqldlRuntime.Initialize(new OfflineFixtureSource());

            _onViewChanged = OnViewChanged;
            _runtime.ViewChanged += _onViewChanged;
            _runtime.Recompute();
        }

        private void Update()
        {
            if (_runtime != null)
            {
                _runtime.Tick(Time.unscaledDeltaTime);
            }
        }

        private void OnDestroy()
        {
            if (_runtime != null && _onViewChanged != null)
            {
                _runtime.ViewChanged -= _onViewChanged;
            }

            _onViewChanged = null;
        }

        private void OnViewChanged(HudViewModel model)
        {
            if (hud != null)
            {
                hud.Apply(model);
            }
        }
    }
}
