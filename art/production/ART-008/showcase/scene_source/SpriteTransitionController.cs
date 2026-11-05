using System;
using System.Collections.Generic;
using UnityEngine;

namespace TTGQLDL.Showcase.Presentation
{
    /// <summary>
    /// Bounded <=20 one-shot 12 FPS VFX burst pool for the isolated ART-008 vertical slice.
    /// Automatically deactivates and clears sprite reference after the final clip frame so zero residue persists.
    /// </summary>
    public sealed class BoundedVfxPool : MonoBehaviour
    {
        public const int MaxActiveBursts = 20;
        public const double FrameStepMs = 1000.0 / 12.0;

        private sealed class BurstSlot
        {
            public GameObject go;
            public SpriteRenderer renderer;
            public Sprite[] frames;
            public int frameIndex;
            public double accumMs;
            public bool active;
        }

        private readonly List<BurstSlot> _slots = new List<BurstSlot>(MaxActiveBursts);

        public int ActiveBurstCount
        {
            get
            {
                int c = 0;
                for (int i = 0; i < _slots.Count; i++) if (_slots[i].active) c++;
                return c;
            }
        }

        public void Initialize()
        {
            if (_slots.Count > 0) return;
            for (int i = 0; i < MaxActiveBursts; i++)
            {
                var go = new GameObject($"PooledVfxBurst_{i:D2}");
                go.transform.SetParent(transform, false);
                var sr = go.AddComponent<SpriteRenderer>();
                sr.sortingLayerName = "VFX";
                sr.sortingOrder = 10 + i;
                go.SetActive(false);
                _slots.Add(new BurstSlot { go = go, renderer = sr, active = false });
            }
        }

        public bool SpawnBurst(Sprite[] frames, Vector3 worldPos, float uniformScale, string sortingLayerName = "VFX", int sortingOrder = 15, bool flipX = false)
        {
            if (frames == null || frames.Length == 0) return false;
            BurstSlot chosen = null;
            for (int i = 0; i < _slots.Count; i++)
            {
                if (!_slots[i].active)
                {
                    chosen = _slots[i];
                    break;
                }
            }
            if (chosen == null)
            {
                // Bounded pool safety: recycle oldest nearly-finished burst rather than allocating unbounded GameObjects
                int bestIdx = 0;
                int maxIdx = -1;
                for (int i = 0; i < _slots.Count; i++)
                {
                    if (_slots[i].frameIndex > maxIdx)
                    {
                        maxIdx = _slots[i].frameIndex;
                        bestIdx = i;
                    }
                }
                chosen = _slots[bestIdx];
            }

            chosen.frames = frames;
            chosen.frameIndex = 0;
            chosen.accumMs = 0.0;
            chosen.active = true;
            chosen.go.transform.position = worldPos;
            chosen.go.transform.localScale = new Vector3(uniformScale, uniformScale, 1f);
            chosen.renderer.sortingLayerName = sortingLayerName;
            chosen.renderer.sortingOrder = sortingOrder;
            chosen.renderer.flipX = flipX;
            chosen.renderer.sprite = frames[0];
            chosen.go.SetActive(true);
            return true;
        }

        public void StepSimulationMs(double deltaMs)
        {
            for (int i = 0; i < _slots.Count; i++)
            {
                var s = _slots[i];
                if (!s.active) continue;
                s.accumMs += deltaMs;
                while (s.accumMs >= FrameStepMs)
                {
                    s.accumMs -= FrameStepMs;
                    s.frameIndex++;
                    if (s.frameIndex >= s.frames.Length)
                    {
                        s.active = false;
                        s.renderer.sprite = null;
                        s.go.SetActive(false);
                        break;
                    }
                    s.renderer.sprite = s.frames[s.frameIndex];
                }
            }
        }
    }

    /// <summary>
    /// Controlled 12 FPS sprite transition animator for all 7 Bastion Dawn character classes.
    /// Guarantees clean transitions (Idle -> Walk -> Action -> Recovery -> Idle, Idle -> Hit -> Idle),
    /// exact impact-frame VFX/audio hooks, zero accidental action looping, and Y-sorted order updates.
    /// </summary>
    public sealed class SpriteTransitionController : MonoBehaviour
    {
        public const double FrameStepMs = 1000.0 / 12.0;

        public sealed class ClipDef
        {
            public string clipName;
            public Sprite[] frames;
            public bool loop;
            public int impactFrameIndex = -1;
            public Sprite[] pairedVfxFrames;
            public Vector2 vfxWorldOffset;
            public float vfxScale = 0.56f;
            public string vfxSortingLayer = "VFX";
            public string pairedAudioCueId;
        }

        private SpriteRenderer _renderer;
        private readonly Dictionary<string, ClipDef> _clips = new Dictionary<string, ClipDef>(StringComparer.Ordinal);
        private string _defaultIdleClip = "idle";
        private ClipDef _current;
        private int _frameIndex;
        private double _accumMs;
        private BoundedVfxPool _vfxPool;

        public string ClassId { get; private set; }
        public string CurrentClipName => _current != null ? _current.clipName : string.Empty;
        public int CurrentFrameIndex => _frameIndex;
        public int CompletedNonLoopTransitions { get; private set; }
        public int ImpactTriggersFired { get; private set; }
        public event Action<string, string, int> OnImpactFrameReached; // (classId, audioCueId, frameIndex)

        public void Configure(
            string classId,
            SpriteRenderer renderer,
            float approvedVisualScale,
            string defaultIdleClip,
            BoundedVfxPool vfxPool,
            IEnumerable<ClipDef> clips)
        {
            ClassId = classId;
            _renderer = renderer;
            _defaultIdleClip = defaultIdleClip;
            _vfxPool = vfxPool;
            transform.localScale = new Vector3(approvedVisualScale, approvedVisualScale, 1f);

            _clips.Clear();
            foreach (var c in clips)
            {
                _clips[c.clipName] = c;
            }
            UpdateYSortingOrder();
            PlayClip(_defaultIdleClip);
        }

        public void UpdateYSortingOrder()
        {
            if (_renderer != null)
            {
                _renderer.sortingLayerName = "Units";
                _renderer.sortingOrder = -Mathf.RoundToInt(transform.position.y * 100f);
            }
        }

        public bool PlayClip(string clipName)
        {
            if (!_clips.TryGetValue(clipName, out var next)) return false;
            _current = next;
            _frameIndex = 0;
            _accumMs = 0.0;
            _renderer.sprite = _current.frames[0];
            CheckImpactFrame();
            return true;
        }

        public void StepSimulationMs(double deltaMs)
        {
            if (_current == null || _current.frames == null || _current.frames.Length == 0) return;
            _accumMs += deltaMs;
            while (_accumMs >= FrameStepMs)
            {
                _accumMs -= FrameStepMs;
                int nextIdx = _frameIndex + 1;
                if (nextIdx >= _current.frames.Length)
                {
                    if (_current.loop)
                    {
                        _frameIndex = 0;
                        _renderer.sprite = _current.frames[0];
                    }
                    else
                    {
                        // Non-looping action or hit clip cleanly returns to default idle stance without flickering
                        CompletedNonLoopTransitions++;
                        PlayClip(_defaultIdleClip);
                        continue;
                    }
                }
                else
                {
                    _frameIndex = nextIdx;
                    _renderer.sprite = _current.frames[_frameIndex];
                }
                CheckImpactFrame();
            }
        }

        private void CheckImpactFrame()
        {
            if (_current == null || _frameIndex != _current.impactFrameIndex) return;
            ImpactTriggersFired++;
            if (_vfxPool != null && _current.pairedVfxFrames != null && _current.pairedVfxFrames.Length > 0)
            {
                float dir = (_renderer != null && _renderer.flipX) ? -1f : 1f;
                Vector3 vfxPos = transform.position + new Vector3(_current.vfxWorldOffset.x * dir, _current.vfxWorldOffset.y, 0f);
                int order = _current.vfxSortingLayer == "LanesAndDecals" ? 28 : 20;
                _vfxPool.SpawnBurst(_current.pairedVfxFrames, vfxPos, _current.vfxScale, _current.vfxSortingLayer, order, _renderer != null && _renderer.flipX);
            }
            if (!string.IsNullOrEmpty(_current.pairedAudioCueId))
            {
                OnImpactFrameReached?.Invoke(ClassId, _current.pairedAudioCueId, _frameIndex);
            }
        }
    }
}
