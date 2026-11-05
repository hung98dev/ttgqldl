using System;
using System.Collections.Generic;
using UnityEngine;

namespace TTGQLDL.Showcase.Audio
{
    /// <summary>
    /// Presentation-only event-driven audio dispatcher for the isolated ART-008 vertical slice.
    /// Strictly non-authoritative: never advances gameplay ticks, mutates simulation state, or creates durable ACKs.
    /// Preallocates 12 pooled AudioSources (8 Combat SFX + 2 Priority Alerts + 1 Ambient + 1 Music muted -80 dB by default).
    /// </summary>
    public sealed class BroadcastAudioDispatcher : MonoBehaviour
    {
        public const int CombatSfxPoolSize = 8;
        public const int PriorityAlertsPoolSize = 2;
        public const int TotalPreallocatedVoices = 12;
        public const int RingBufferCapacity = 256;

        [Serializable]
        public sealed class CueSpec
        {
            public string cueId;
            public string clipName;
            public string bus; // "SFX", "Alerts", "Ambient", "Music"
            public int priority0Highest;
            public double cooldownMs;
            public double assaultCooldownMs;
            public float assaultAttenuationDb;
            public bool suppressDuringPriorityAlerts;
            public int maxConcurrentInstances = 1;
            public bool loop;
            public bool mutedByDefault;
        }

        private sealed class PooledVoice
        {
            public AudioSource source;
            public string activeCueId;
            public int priority0Highest;
            public double startedTimeMs;
            public double endTimeMs;
        }

        private static readonly HashSet<string> GenuineOncePerWaveMilestones = new HashSet<string>(StringComparer.Ordinal)
        {
            "vote_closed",
            "boss_incoming",
            "boss_spawned",
            "wave_result"
        };

        private readonly List<PooledVoice> _sfxPool = new List<PooledVoice>(CombatSfxPoolSize);
        private readonly List<PooledVoice> _alertsPool = new List<PooledVoice>(PriorityAlertsPoolSize);
        private AudioSource _ambientSource;
        private AudioSource _musicSource;

        private readonly Dictionary<string, CueSpec> _cues = new Dictionary<string, CueSpec>(StringComparer.Ordinal);
        private readonly Dictionary<string, AudioClip> _clips = new Dictionary<string, AudioClip>(StringComparer.Ordinal);
        private readonly Dictionary<string, double> _lastCuePlayMs = new Dictionary<string, double>(StringComparer.Ordinal);

        // Two-tier deduplication state (ttg-audio-dedupe-v2)
        private readonly Queue<string> _ringOrder = new Queue<string>(RingBufferCapacity);
        private readonly HashSet<string> _ringSet = new HashSet<string>(StringComparer.Ordinal);
        private readonly Dictionary<string, long> _sessionMaxStreamSeq = new Dictionary<string, long>(StringComparer.Ordinal);
        private readonly HashSet<string> _waveMilestones = new HashSet<string>(StringComparer.Ordinal);

        // Bus gain state (in dB)
        private readonly Dictionary<string, float> _busVolumeDb = new Dictionary<string, float>(StringComparer.Ordinal)
        {
            { "Master", 0.0f },
            { "Alerts", 0.0f },
            { "SFX", -1.5f },
            { "Ambient", -3.0f },
            { "Music", -80.0f }
        };

        private bool _globalMute;
        private bool _musicRightsClearedForTestOnly;
        private int _voiceStealCount;

        public bool IsInitialized { get; private set; }
        public bool GlobalMute => _globalMute;
        public bool IsMusicMutedByDefault => _musicSource != null && _musicSource.mute && _busVolumeDb["Music"] <= -79.9f;
        public int VoiceStealCount => _voiceStealCount;
        public int TotalChildAudioSources => GetComponentsInChildren<AudioSource>(true).Length;

        public void Initialize(Dictionary<string, AudioClip> clipsByName)
        {
            if (IsInitialized) return;

            foreach (var kv in clipsByName)
            {
                _clips[kv.Key] = kv.Value;
            }
            RegisterCanonicalCues();

            var sfxRoot = new GameObject("CombatSfxPool_8");
            sfxRoot.transform.SetParent(transform, false);
            for (int i = 0; i < CombatSfxPoolSize; i++)
            {
                var go = new GameObject($"SfxVoice_{i:D2}");
                go.transform.SetParent(sfxRoot.transform, false);
                var src = go.AddComponent<AudioSource>();
                src.playOnAwake = false;
                src.loop = false;
                src.spatialBlend = 0f;
                src.priority = 110;
                _sfxPool.Add(new PooledVoice { source = src, priority0Highest = 255, endTimeMs = -1.0 });
            }

            var alertRoot = new GameObject("PriorityAlertsPool_2");
            alertRoot.transform.SetParent(transform, false);
            for (int i = 0; i < PriorityAlertsPoolSize; i++)
            {
                var go = new GameObject($"AlertVoice_{i:D2}");
                go.transform.SetParent(alertRoot.transform, false);
                var src = go.AddComponent<AudioSource>();
                src.playOnAwake = false;
                src.loop = false;
                src.spatialBlend = 0f;
                src.priority = 16;
                _alertsPool.Add(new PooledVoice { source = src, priority0Highest = 255, endTimeMs = -1.0 });
            }

            var ambGo = new GameObject("AmbientLoopVoice_1");
            ambGo.transform.SetParent(transform, false);
            _ambientSource = ambGo.AddComponent<AudioSource>();
            _ambientSource.playOnAwake = false;
            _ambientSource.loop = true;
            _ambientSource.spatialBlend = 0f;
            _ambientSource.priority = 140;
            if (_clips.TryGetValue("amb_village_dawn_loop", out var ambClip))
            {
                _ambientSource.clip = ambClip;
            }

            var musGo = new GameObject("MusicLoopVoice_1_MutedByDefault");
            musGo.transform.SetParent(transform, false);
            _musicSource = musGo.AddComponent<AudioSource>();
            _musicSource.playOnAwake = false;
            _musicSource.loop = true;
            _musicSource.spatialBlend = 0f;
            _musicSource.priority = 160;
            _musicSource.mute = true;
            _musicSource.volume = 0f;
            if (_clips.TryGetValue("music_village_prep_loop", out var musClip))
            {
                _musicSource.clip = musClip;
            }

            RefreshLoopVolumes();
            IsInitialized = true;
        }

        private void RegisterCanonicalCues()
        {
            _cues["SFX01"] = new CueSpec { cueId = "SFX01", clipName = "sfx_role_join", bus = "SFX", priority0Highest = 96, cooldownMs = 2000.0, assaultCooldownMs = 3500.0, assaultAttenuationDb = -5.5f, suppressDuringPriorityAlerts = true, maxConcurrentInstances = 1 };
            _cues["SFX02"] = new CueSpec { cueId = "SFX02", clipName = "sfx_builder_repair", bus = "SFX", priority0Highest = 110, cooldownMs = 220.0, assaultCooldownMs = 220.0, maxConcurrentInstances = 2 };
            _cues["SFX03"] = new CueSpec { cueId = "SFX03", clipName = "sfx_guard_block", bus = "SFX", priority0Highest = 84, cooldownMs = 180.0, assaultCooldownMs = 180.0, maxConcurrentInstances = 2 };
            _cues["SFX04"] = new CueSpec { cueId = "SFX04", clipName = "sfx_fighter_slash", bus = "SFX", priority0Highest = 82, cooldownMs = 160.0, assaultCooldownMs = 160.0, maxConcurrentInstances = 3 };
            _cues["SFX05"] = new CueSpec { cueId = "SFX05", clipName = "sfx_enemy_attack", bus = "SFX", priority0Highest = 120, cooldownMs = 150.0, assaultCooldownMs = 150.0, maxConcurrentInstances = 3 };
            _cues["SFX06"] = new CueSpec { cueId = "SFX06", clipName = "sfx_wall_damage", bus = "SFX", priority0Highest = 64, cooldownMs = 350.0, assaultCooldownMs = 350.0, maxConcurrentInstances = 2 };
            _cues["SFX07"] = new CueSpec { cueId = "SFX07", clipName = "sfx_wall_breach", bus = "Alerts", priority0Highest = 24, cooldownMs = 2000.0, assaultCooldownMs = 2000.0, maxConcurrentInstances = 1 };
            _cues["SFX08"] = new CueSpec { cueId = "SFX08", clipName = "sfx_boss_warning", bus = "Alerts", priority0Highest = 16, cooldownMs = 5000.0, assaultCooldownMs = 5000.0, maxConcurrentInstances = 1 };
            _cues["SFX09"] = new CueSpec { cueId = "SFX09", clipName = "sfx_boss_slam", bus = "Alerts", priority0Highest = 20, cooldownMs = 800.0, assaultCooldownMs = 800.0, maxConcurrentInstances = 1 };
            _cues["SFX10"] = new CueSpec { cueId = "SFX10", clipName = "sfx_vote_closed", bus = "Alerts", priority0Highest = 40, cooldownMs = 5000.0, assaultCooldownMs = 5000.0, maxConcurrentInstances = 1 };
            _cues["SFX11"] = new CueSpec { cueId = "SFX11", clipName = "sfx_wave_victory", bus = "Alerts", priority0Highest = 8, cooldownMs = 10000.0, assaultCooldownMs = 10000.0, maxConcurrentInstances = 1 };
            _cues["SFX12"] = new CueSpec { cueId = "SFX12", clipName = "sfx_wave_defeat", bus = "Alerts", priority0Highest = 8, cooldownMs = 10000.0, assaultCooldownMs = 10000.0, maxConcurrentInstances = 1 };
            _cues["AMB01"] = new CueSpec { cueId = "AMB01", clipName = "amb_village_dawn_loop", bus = "Ambient", priority0Highest = 140, loop = true };
            _cues["MUS01"] = new CueSpec { cueId = "MUS01", clipName = "music_village_prep_loop", bus = "Music", priority0Highest = 160, loop = true, mutedByDefault = true };
            _cues["MUS02"] = new CueSpec { cueId = "MUS02", clipName = "music_village_assault_loop", bus = "Music", priority0Highest = 160, loop = true, mutedByDefault = true };
        }

        public void SetGlobalMute(bool muted)
        {
            _globalMute = muted;
            foreach (var v in _sfxPool) v.source.mute = muted;
            foreach (var v in _alertsPool) v.source.mute = muted;
            if (_ambientSource != null) _ambientSource.mute = muted;
            if (_musicSource != null) _musicSource.mute = muted || !_musicRightsClearedForTestOnly;
            RefreshLoopVolumes();
        }

        public void SetBusVolumeDb(string bus, float volumeDb)
        {
            if (!_busVolumeDb.ContainsKey(bus)) throw new ArgumentException("Unknown audio bus: " + bus);
            if (bus == "Music" && !_musicRightsClearedForTestOnly)
            {
                // Enforce mandatory GATE-LEGAL default mute unless explicitly unlocked in offline QA
                _busVolumeDb["Music"] = -80.0f;
                return;
            }
            _busVolumeDb[bus] = Mathf.Clamp(volumeDb, -80.0f, 0.0f);
            RefreshLoopVolumes();
        }

        public float GetBusVolumeDb(string bus) => _busVolumeDb[bus];

        private float DbToLinear(float db) => db <= -79.5f ? 0f : Mathf.Pow(10f, db / 20f);

        private void RefreshLoopVolumes()
        {
            if (_ambientSource != null)
            {
                _ambientSource.volume = _globalMute ? 0f : DbToLinear(_busVolumeDb["Master"] + _busVolumeDb["Ambient"]);
            }
            if (_musicSource != null)
            {
                bool musMuted = _globalMute || !_musicRightsClearedForTestOnly || _busVolumeDb["Music"] <= -79.5f;
                _musicSource.mute = musMuted;
                _musicSource.volume = musMuted ? 0f : DbToLinear(_busVolumeDb["Master"] + _busVolumeDb["Music"]);
            }
        }

        public bool IsAnyPriorityAlertActive(double nowMs)
        {
            for (int i = 0; i < _alertsPool.Count; i++)
            {
                if (_alertsPool[i].endTimeMs > nowMs) return true;
            }
            return false;
        }

        public int CountActiveVoices(double nowMs)
        {
            int active = 0;
            for (int i = 0; i < _sfxPool.Count; i++) if (_sfxPool[i].endTimeMs > nowMs) active++;
            for (int i = 0; i < _alertsPool.Count; i++) if (_alertsPool[i].endTimeMs > nowMs) active++;
            if (_ambientSource != null && _ambientSource.isPlaying && !_ambientSource.mute) active++;
            if (_musicSource != null && _musicSource.isPlaying && !_musicSource.mute) active++;
            return active;
        }

        /// <summary>
        /// Dispatches a committed non-speculative presentation event through Two-Tier Deduplication,
        /// phase-aware cooldowns, priority-alert suppression, and the fixed 12-voice pool.
        /// </summary>
        public bool TryDispatchCommittedCue(
            string sessionId,
            int waveNo,
            long streamSeq,
            string canonicalEventType,
            string cueId,
            string phase,
            double nowMs,
            out float scheduledLinearGain,
            out string selectedClipName)
        {
            scheduledLinearGain = 0f;
            selectedClipName = string.Empty;

            if (!_cues.TryGetValue(cueId, out var spec)) return false;
            if (!_clips.TryGetValue(spec.clipName, out var clip) || clip == null) return false;

            // 1. Tier-1 exact outbox (session_id, stream_seq) deduplication (256-entry ring + monotonic cursor)
            string deliveryKey = $"{sessionId}:{streamSeq}";
            if (_ringSet.Contains(deliveryKey)) return false;
            if (_sessionMaxStreamSeq.TryGetValue(sessionId, out long maxSeq) && streamSeq <= maxSeq - RingBufferCapacity)
            {
                return false;
            }
            RecordDeliveryKey(sessionId, streamSeq, deliveryKey);

            // 2. Tier-2 genuine once-per-wave milestone suppression ONLY for vote_closed, boss_incoming, boss_spawned, wave_result
            if (GenuineOncePerWaveMilestones.Contains(canonicalEventType))
            {
                string waveMilestoneKey = $"{sessionId}:{waveNo}:{canonicalEventType}";
                if (!_waveMilestones.Add(waveMilestoneKey))
                {
                    return false;
                }
            }

            // 3. Priority-alert suppression for low-priority chime cues (SFX01 Role Join)
            if (spec.suppressDuringPriorityAlerts && IsAnyPriorityAlertActive(nowMs))
            {
                return false;
            }

            // 4. Phase-aware cooldown check (PREP 2000ms vs ASSAULT/FINALE 3500ms for SFX01)
            bool isAssaultLike = string.Equals(phase, "ASSAULT", StringComparison.OrdinalIgnoreCase) ||
                                 string.Equals(phase, "FINALE", StringComparison.OrdinalIgnoreCase);
            double requiredCooldown = isAssaultLike ? spec.assaultCooldownMs : spec.cooldownMs;
            if (_lastCuePlayMs.TryGetValue(cueId, out double lastMs) && (nowMs - lastMs) < requiredCooldown)
            {
                return false;
            }

            // 5. Max concurrent instances per cue check
            var targetPool = spec.bus == "Alerts" ? _alertsPool : _sfxPool;
            int activeSameCue = 0;
            for (int i = 0; i < targetPool.Count; i++)
            {
                if (targetPool[i].endTimeMs > nowMs && targetPool[i].activeCueId == cueId)
                {
                    activeSameCue++;
                }
            }
            if (activeSameCue >= spec.maxConcurrentInstances) return false;

            // 6. Voice allocation (free slot first; otherwise priority voice stealing in SFX pool)
            PooledVoice chosen = null;
            for (int i = 0; i < targetPool.Count; i++)
            {
                if (targetPool[i].endTimeMs <= nowMs)
                {
                    chosen = targetPool[i];
                    break;
                }
            }
            if (chosen == null)
            {
                // Steal lowest-priority (highest numeric priority0Highest), oldest voice if strictly lower priority than incoming cue
                int worstIdx = -1;
                int worstPrio = spec.priority0Highest;
                double oldestStart = double.MaxValue;
                for (int i = 0; i < targetPool.Count; i++)
                {
                    var cand = targetPool[i];
                    if (cand.priority0Highest > worstPrio || (cand.priority0Highest == worstPrio && cand.startedTimeMs < oldestStart && spec.bus == "Alerts"))
                    {
                        worstPrio = cand.priority0Highest;
                        oldestStart = cand.startedTimeMs;
                        worstIdx = i;
                    }
                }
                if (worstIdx < 0) return false;
                chosen = targetPool[worstIdx];
                _voiceStealCount++;
            }

            float phaseAttenDb = isAssaultLike ? spec.assaultAttenuationDb : 0f;
            float totalDb = _busVolumeDb["Master"] + _busVolumeDb[spec.bus] + phaseAttenDb;
            scheduledLinearGain = _globalMute ? 0f : DbToLinear(totalDb);

            chosen.source.Stop();
            chosen.source.clip = clip;
            chosen.source.priority = spec.priority0Highest;
            chosen.source.mute = _globalMute;
            chosen.source.volume = scheduledLinearGain;
            chosen.source.Play();

            chosen.activeCueId = cueId;
            chosen.priority0Highest = spec.priority0Highest;
            chosen.startedTimeMs = nowMs;
            chosen.endTimeMs = nowMs + (clip.length * 1000.0);

            _lastCuePlayMs[cueId] = nowMs;
            selectedClipName = clip.name;
            return true;
        }

        public void ResetCooldownsOnlyForNextTest()
        {
            _lastCuePlayMs.Clear();
            for (int i = 0; i < _sfxPool.Count; i++) _sfxPool[i].endTimeMs = -1.0;
            for (int i = 0; i < _alertsPool.Count; i++) _alertsPool[i].endTimeMs = -1.0;
        }

        private void RecordDeliveryKey(string sessionId, long streamSeq, string deliveryKey)
        {
            if (_ringOrder.Count >= RingBufferCapacity)
            {
                string oldest = _ringOrder.Dequeue();
                _ringSet.Remove(oldest);
            }
            _ringOrder.Enqueue(deliveryKey);
            _ringSet.Add(deliveryKey);

            if (!_sessionMaxStreamSeq.TryGetValue(sessionId, out long prev) || streamSeq > prev)
            {
                _sessionMaxStreamSeq[sessionId] = streamSeq;
            }
        }
    }
}
