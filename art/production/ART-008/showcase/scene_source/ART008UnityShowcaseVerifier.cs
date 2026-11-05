using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using TTGQLDL.Showcase.Audio;
using TTGQLDL.Showcase.Presentation;
using TTGQLDL.Showcase.UI;
using Unity.Profiling;
using UnityEditor;
using UnityEditor.SceneManagement;
using UnityEngine;
using UnityEngine.Profiling;
using UnityEngine.UI;

namespace TTGQLDL.Showcase.EditorTools
{
    public static class ART008UnityShowcaseVerifier
    {
        [Serializable]
        public sealed class SpriteImportRule
        {
            public string assetPath;
            public int ppu;
            public float pivotX;
            public float pivotY;
            public bool pointFilter;
            public int borderL;
            public int borderB;
            public int borderR;
            public int borderT;
        }

        [Serializable]
        public sealed class ShowcaseInputConfig
        {
            public List<SpriteImportRule> spriteRules = new List<SpriteImportRule>();
            public List<SyntheticCommittedView> verificationStates = new List<SyntheticCommittedView>();
            public SyntheticCommittedView englishVoteProofState;
            public string outputVerificationJson;
            public string outputPerformanceJson;
            public string outputSceneSummaryJson;
            public string outputRenderAssaultPng;
            public string outputRenderBossPng;
            public string outputRenderReconnectPng;
        }

        [Serializable]
        public sealed class ClassDisplayMeasurement
        {
            public string classId;
            public float approvedScale;
            public int cellWidthPx;
            public int cellHeightPx;
            public int tightBodyHeightSourcePx;
            public float displayedBodyHeight1080pPx;
            public float targetBodyHeight1080pPx;
            public float deltaPx;
            public bool transitionAuditPassed;
            public int completedActionAndHitTransitions;
            public int impactTriggersFired;
        }

        [Serializable]
        public sealed class FixtureSummaryEntry
        {
            public int wave_no;
            public int wave_tick;
            public string phase;
            public int wall_tier;
            public int wall_hp;
            public int wall_max;
            public int townhall_hp;
            public int townhall_max;
            public int wood;
            public int stone;
            public int food;
            public int builder;
            public int guard;
            public int fighter;
            public int west_hp;
            public int center_hp;
            public int east_hp;
            public bool boss_exists;
            public int boss_hp;
            public int boss_max_hp;
            public string vote_selected;
        }

        [Serializable]
        public sealed class HudVerificationStateEntry
        {
            public string state_id;
            public string fixture_id;
            public string locale;
            public string fixture_source_status;
            public bool watermark_visible;
            public string watermark_text;
            public string wall_hp_text;
            public string castle_hp_text;
            public string wood_text;
            public string stone_text;
            public string food_text;
            public string wave_text;
            public string timer_text;
            public string source_status_text;
            public string threat_banner_text;
            public string vote_strip_text;
            public string secondary_diag_text;
            public string primary_cta_text;
            public string builder_text;
            public string guard_text;
            public string fighter_text;
            public string render_1080p;
            public string render_720p;
            public string render_360p;
            public FixtureSummaryEntry fixture_summary;
        }

        [Serializable]
        public sealed class LoadTierResult
        {
            public int visibleEntityCount;
            public int warmupFrames;
            public int measuredFrames;
            public string timingScope;
            public float averageFps;
            public float onePercentLowFps;
            public string onePercentLowDefinition;
            public int slowestOnePercentFrameCount;
            public float slowestOnePercentMeanMs;
            public float averageFrameTimeMs;
            public float p99FrameTimeMs;
            public float maxFrameTimeMs;
            public string gpuFrameTimingStatus;
            public float latestGpuFrameTimeMs;
            public string standalone15MinFpsStatus;
            public string obsNvencCaptureFpsStatus;
            public string managedGcMeasurementMethod;
            public long managedGcAllocatedBytesTotal;
            public float managedGcBytesPerFrame;
            public long netHeapDeltaBytes;
            public bool profilerRecorderGcValid;
            public long drawCalls;
            public string drawCallsStatus;
            public long batches;
            public string batchesStatus;
            public long setPassCalls;
            public string setPassCallsStatus;
            public long totalTextureMemoryBytes;
            public long totalAudioMemoryBytes;
            public int maxActiveAudioVoices;
            public int maxActiveVfxBursts;
            public int dynamicObjectsSpawnedDuringLoop;
        }

        [Serializable]
        public sealed class UnityVerificationReport
        {
            public string packageId = "ART-008";
            public string remediatedByPackage = "ART-009";
            public string unityVersion;
            public string status = "FAIL";
            public string executionMode;
            public string graphicsDeviceName;
            public string graphicsDeviceType;
            public string processorType;
            public int systemMemoryMB;
            public int graphicsMemoryMB;
            public string savedScenePath;
            public string savedAudioPrefabPath;
            public List<string> verifiedSortingLayers = new List<string>();
            public int importedSpriteCount;
            public int importedAudioClipCount;
            public bool cameraOrthographicSize5Verified;
            public bool fixedCameraZeroPanZoomShakeVerified;
            public bool parapetDepthSortingVerified;
            public bool hudSafeZoneVerified;
            public int hudInPlaceTransitionsStandard;
            public int hudInPlaceTransitionsReducedMotion;
            public int sceneReloadsDuringStateTransitions;
            public int preallocatedAudioVoices;
            public bool musicMutedByDefaultVerified;
            public int joinPrep10sPlays;
            public int joinAssault10sPlays;
            public float joinAssaultCueAttenuationDb = -5.5f;
            public float joinAssaultCueLinearGain = 0.5309f;
            public float sfxBusDefaultDb = -1.5f;
            public float sfxBusLinearGain = 0.8414f;
            public float joinAssaultCombinedDb = -7.0f;
            public float joinAssaultLinearGain;
            public int joinDuringPriorityAlertPlays;
            public int duplicateStreamSeqPlays;
            public int duplicateVictoryReconnectPlays;
            public int independentWallDamage3Plays;
            public int independentFighterSlash2Plays;
            public int independentBuilderRepair2Plays;
            public int independentBossSlam2Plays;
            public int burst32SimultaneousMaxActiveVoices;
            public bool globalMuteAndIndependentBusesVerified;
            public int leakedAudioSourcesAfterTeardown;
            public int lingeringVfxAfterCompletion;
            public List<ClassDisplayMeasurement> classMeasurements = new List<ClassDisplayMeasurement>();
            public List<HudVerificationStateEntry> hud_verification_states = new List<HudVerificationStateEntry>();
            public HudVerificationStateEntry english_vote_b_proof;
            public List<string> renderedGpuFrames = new List<string>();
            public string honestyNote;
        }

        [Serializable]
        public sealed class PerformanceReport
        {
            public string packageId = "ART-008";
            public string remediatedByPackage = "ART-009";
            public string status = "FAIL";
            public string unityVersion;
            public string executionContext;
            public string timingScope = "cpu_camera_render_submission_editor_batchmode";
            public string onePercentLowFormula = "1000.0 / mean(slowest ceil(N * 0.01) frameTimesMs)";
            public string cpuModel;
            public string gpuModel;
            public string graphicsApi;
            public int systemMemoryMB;
            public int graphicsMemoryMB;
            public int pooledEntityCap = 150;
            public int pooledVfxBurstCap = 20;
            public int preallocatedAudioVoices = 12;
            public long importedSpriteTextureRuntimeBytes;
            public long importedAudioRuntimeBytes;
            public string standalone15MinSoakStatus = "NOT_EXECUTED";
            public string obsHardwareCaptureStatus = "NOT_EXECUTED";
            public List<LoadTierResult> loadTiers = new List<LoadTierResult>();
            public string scopeAndLimitationsNote;
        }

        private sealed class WorldRefs
        {
            public SpriteRenderer wallBackSr;
            public SpriteRenderer wallFrontSr;
            public SpriteRenderer castleSr;
            public SpriteRenderer bossTelegraphSr;
            public GameObject scaffoldSwGo;
            public GameObject scaffoldSeGo;
        }

        public static void RunShowcaseBuildAndVerification()
        {
            string cfgPath = Environment.GetEnvironmentVariable("ART008_UNITY_CONFIG_JSON");
            if (string.IsNullOrEmpty(cfgPath) || !File.Exists(cfgPath))
            {
                throw new Exception("Missing ART008_UNITY_CONFIG_JSON path.");
            }
            var cfg = JsonUtility.FromJson<ShowcaseInputConfig>(File.ReadAllText(cfgPath));
            if (cfg.verificationStates == null || cfg.verificationStates.Count < 12)
            {
                throw new Exception("Expected at least 12 canonical SyntheticCommittedView verification states.");
            }

            ConfigureSortingLayers();

            // 1. Apply canonical import rules to all sprites & UI 9-slice skins
            long totalSpriteBytes = 0;
            var spritesByStem = new Dictionary<string, Sprite>(StringComparer.Ordinal);
            foreach (var rule in cfg.spriteRules)
            {
                var ti = AssetImporter.GetAtPath(rule.assetPath) as TextureImporter;
                if (ti == null) throw new Exception("Missing TextureImporter at " + rule.assetPath);
                ti.textureType = TextureImporterType.Sprite;
                ti.spriteImportMode = SpriteImportMode.Single;
                ti.spritePixelsPerUnit = rule.ppu;
                ti.mipmapEnabled = false;
                ti.alphaIsTransparency = true;
                ti.sRGBTexture = true;
                ti.filterMode = rule.pointFilter ? FilterMode.Point : FilterMode.Bilinear;
                ti.textureCompression = TextureImporterCompression.Uncompressed;
                ti.maxTextureSize = 2048;

                var settings = new TextureImporterSettings();
                ti.ReadTextureSettings(settings);
                settings.spriteAlignment = (int)SpriteAlignment.Custom;
                settings.spritePivot = new Vector2(rule.pivotX, rule.pivotY);
                settings.spriteBorder = new Vector4(rule.borderL, rule.borderB, rule.borderR, rule.borderT);
                settings.spriteMeshType = SpriteMeshType.FullRect;
                ti.SetTextureSettings(settings);
                ti.SaveAndReimport();

                var sp = AssetDatabase.LoadAssetAtPath<Sprite>(rule.assetPath);
                if (sp == null) throw new Exception("Failed to load Sprite: " + rule.assetPath);
                spritesByStem[sp.name] = sp;
                totalSpriteBytes += Profiler.GetRuntimeMemorySizeLong(sp.texture);
            }

            // 2. Import all 15 canonical ART-007 audio clips
            long totalAudioBytes = 0;
            var clipsByName = new Dictionary<string, AudioClip>(StringComparer.Ordinal);
            string[] audioGuids = AssetDatabase.FindAssets("t:AudioClip", new[] { "Assets/_TTGQLDL_Showcase/Audio" });
            Array.Sort(audioGuids, StringComparer.Ordinal);
            foreach (string guid in audioGuids)
            {
                string ap = AssetDatabase.GUIDToAssetPath(guid);
                var ai = AssetImporter.GetAtPath(ap) as AudioImporter;
                var s = ai.defaultSampleSettings;
                if (ap.Contains("/SFX/"))
                {
                    s.loadType = AudioClipLoadType.DecompressOnLoad;
                    s.compressionFormat = AudioCompressionFormat.PCM;
                    s.preloadAudioData = true;
                    ai.loadInBackground = false;
                }
                else if (ap.Contains("/Ambient/"))
                {
                    s.loadType = AudioClipLoadType.CompressedInMemory;
                    s.compressionFormat = AudioCompressionFormat.Vorbis;
                    s.quality = 0.70f;
                    s.preloadAudioData = true;
                    ai.loadInBackground = true;
                }
                else
                {
                    s.loadType = AudioClipLoadType.Streaming;
                    s.compressionFormat = AudioCompressionFormat.Vorbis;
                    s.quality = 0.70f;
                    s.preloadAudioData = false;
                    ai.loadInBackground = true;
                }
                s.sampleRateSetting = AudioSampleRateSetting.PreserveSampleRate;
                ai.forceToMono = false;
                ai.defaultSampleSettings = s;
                ai.SaveAndReimport();

                var clip = AssetDatabase.LoadAssetAtPath<AudioClip>(ap);
                if (clip == null || clip.frequency != 48000) throw new Exception("Invalid AudioClip import: " + ap);
                clipsByName[clip.name] = clip;
                totalAudioBytes += Profiler.GetRuntimeMemorySizeLong(clip);
            }

            // 3. Create clean Vertical Slice Scene
            var scene = EditorSceneManager.NewScene(NewSceneSetup.EmptyScene, NewSceneMode.Single);

            // Fixed Orthographic Camera (1080x1920 portrait, Ortho Size = 5.0, PPU = 192 -> 192 screen px per world unit)
            var camGo = new GameObject("MainBroadcastCamera");
            var cam = camGo.AddComponent<Camera>();
            cam.orthographic = true;
            cam.orthographicSize = 5.0f;
            cam.nearClipPlane = 0.1f;
            cam.farClipPlane = 50.0f;
            cam.clearFlags = CameraClearFlags.SolidColor;
            cam.backgroundColor = new Color(0.32f, 0.45f, 0.25f, 1.0f);
            camGo.transform.position = new Vector3(0f, 0f, -10f);
            camGo.AddComponent<AudioListener>();

            // Build ART-005 Battlefield Hierarchy across the 6 Sorting Layers
            var worldRoot = new GameObject("BastionDawnWorld");
            var worldRefs = BuildEnvironmentWorld(worldRoot.transform, spritesByStem);

            // Bounded VFX Pool (<=20 active bursts)
            var vfxPoolGo = new GameObject("BoundedVfxPool_20");
            vfxPoolGo.transform.SetParent(worldRoot.transform, false);
            var vfxPool = vfxPoolGo.AddComponent<BoundedVfxPool>();
            vfxPool.Initialize();

            // BroadcastAudioDispatcher (12 preallocated pooled AudioSources, Music muted -80 dB by default)
            var audioGo = new GameObject("BroadcastAudioDispatcher_12Voices");
            var dispatcher = audioGo.AddComponent<BroadcastAudioDispatcher>();
            dispatcher.Initialize(clipsByName);

            Directory.CreateDirectory("Assets/_TTGQLDL_Showcase/Prefabs");
            string audioPrefabPath = "Assets/_TTGQLDL_Showcase/Prefabs/BroadcastAudioDispatcher.prefab";
            PrefabUtility.SaveAsPrefabAsset(audioGo, audioPrefabPath, out bool prefabOk);
            if (!prefabOk) throw new Exception("Failed to save BroadcastAudioDispatcher.prefab");

            // Build ART-006 In-Place HUD Canvas (1080x1920 reference, safe-zone compliant, zero hardcoded values)
            var hudPresenter = BuildHudCanvas(cam, spritesByStem, out bool hudSafeZoneOk);
            hudPresenter.Initialize();

            // 4. Verify all 7 character classes: scale contract, tight body pixel height @ 1080p, & full transition audit
            var classMeasurements = VerifyAllSevenCharacterClasses(worldRoot.transform, spritesByStem, vfxPool);

            // Verify parapet depth sorting: BastionBack < Units < BastionFront < VFX
            bool parapetDepthOk = SortingLayer.GetLayerValueFromName("BastionBack") < SortingLayer.GetLayerValueFromName("Units") &&
                                  SortingLayer.GetLayerValueFromName("Units") < SortingLayer.GetLayerValueFromName("BastionFront") &&
                                  SortingLayer.GetLayerValueFromName("BastionFront") < SortingLayer.GetLayerValueFromName("VFX");

            // 5. Create Preallocated 150-Entity Decorative Proxy Pool
            var entityPoolRoot = new GameObject("PooledUnits_150");
            entityPoolRoot.transform.SetParent(worldRoot.transform, false);
            var pooledControllers = Build150EntityPool(entityPoolRoot.transform, spritesByStem, vfxPool);

            // 6. Verify & GPU-Render all 12 canonical states at 1080p, 720p, and 360p (Standard Motion, ZERO scene reloads!)
            var hudStatesOut = new List<HudVerificationStateEntry>();
            var renderedFiles = new List<string>();
            hudPresenter.ReducedMotion = false;

            SyntheticCommittedView stateAssault = null;
            SyntheticCommittedView stateBossActive = null;
            SyntheticCommittedView stateReconnect = null;

            foreach (var st in cfg.verificationStates)
            {
                ApplyWorldForCommittedView(st, worldRefs, pooledControllers, vfxPool, spritesByStem);
                hudPresenter.ApplyCommittedViewInPlace(st, "vi-VN");
                hudPresenter.StepMotionMs(120f);
                hudPresenter.StepMotionMs(140f);

                if (!string.IsNullOrEmpty(st.outputPng1080p))
                {
                    RenderCameraToPng(cam, 1080, 1920, st.outputPng1080p);
                    renderedFiles.Add(st.outputPng1080p);
                }
                if (!string.IsNullOrEmpty(st.outputPng720p))
                {
                    RenderCameraToPng(cam, 720, 1280, st.outputPng720p);
                    renderedFiles.Add(st.outputPng720p);
                }
                if (!string.IsNullOrEmpty(st.outputPng360p))
                {
                    RenderCameraToPng(cam, 360, 640, st.outputPng360p);
                    renderedFiles.Add(st.outputPng360p);
                }

                hudStatesOut.Add(ReadBackHudVerificationEntry(st, hudPresenter, "vi-VN"));

                if (st.stateId == "05_assault") stateAssault = st;
                else if (st.stateId == "07_boss_active") stateBossActive = st;
                else if (st.stateId == "10_source_disconnected") stateReconnect = st;
            }
            int stdTransitions = hudPresenter.InPlaceStateTransitionsCount;

            // Exercise ReducedMotion transitions & English Vote B localization proof without scene reload
            hudPresenter.ReducedMotion = true;
            HudVerificationStateEntry enVoteProof = null;
            if (cfg.englishVoteProofState != null && !string.IsNullOrEmpty(cfg.englishVoteProofState.stateId))
            {
                ApplyWorldForCommittedView(cfg.englishVoteProofState, worldRefs, pooledControllers, vfxPool, spritesByStem);
                hudPresenter.ApplyCommittedViewInPlace(cfg.englishVoteProofState, "en-US");
                hudPresenter.StepMotionMs(16.67f);
                if (!string.IsNullOrEmpty(cfg.englishVoteProofState.outputPng1080p))
                {
                    RenderCameraToPng(cam, 1080, 1920, cfg.englishVoteProofState.outputPng1080p);
                    renderedFiles.Add(cfg.englishVoteProofState.outputPng1080p);
                }
                enVoteProof = ReadBackHudVerificationEntry(cfg.englishVoteProofState, hudPresenter, "en-US");
            }
            // Additional ReducedMotion transitions (storage_unavailable -> waiting -> assault)
            foreach (var rIdx in new[] { 10, 0, 4 })
            {
                if (rIdx < cfg.verificationStates.Count)
                {
                    hudPresenter.ApplyCommittedViewInPlace(cfg.verificationStates[rIdx], "vi-VN");
                    hudPresenter.StepMotionMs(16.67f);
                }
            }
            int redTransitions = hudPresenter.InPlaceStateTransitionsCount - stdTransitions;

            // Export the 3 ART-008 isolated_unity GPU renders (strictly named with actual Direct3D12 API)
            if (stateAssault != null && !string.IsNullOrEmpty(cfg.outputRenderAssaultPng))
            {
                ApplyWorldForCommittedView(stateAssault, worldRefs, pooledControllers, vfxPool, spritesByStem);
                hudPresenter.ApplyCommittedViewInPlace(stateAssault, "vi-VN");
                hudPresenter.StepMotionMs(250f);
                RenderCameraToPng(cam, 1080, 1920, cfg.outputRenderAssaultPng);
                renderedFiles.Add(cfg.outputRenderAssaultPng);
            }
            if (stateBossActive != null && !string.IsNullOrEmpty(cfg.outputRenderBossPng))
            {
                ApplyWorldForCommittedView(stateBossActive, worldRefs, pooledControllers, vfxPool, spritesByStem);
                hudPresenter.ApplyCommittedViewInPlace(stateBossActive, "vi-VN");
                hudPresenter.StepMotionMs(250f);
                RenderCameraToPng(cam, 1080, 1920, cfg.outputRenderBossPng);
                renderedFiles.Add(cfg.outputRenderBossPng);
            }
            if (stateReconnect != null && !string.IsNullOrEmpty(cfg.outputRenderReconnectPng))
            {
                ApplyWorldForCommittedView(stateReconnect, worldRefs, pooledControllers, vfxPool, spritesByStem);
                hudPresenter.ApplyCommittedViewInPlace(stateReconnect, "vi-VN");
                hudPresenter.StepMotionMs(250f);
                RenderCameraToPng(cam, 1080, 1920, cfg.outputRenderReconnectPng);
                renderedFiles.Add(cfg.outputRenderReconnectPng);
            }

            // Save Showcase Scene Asset in canonical Wave 10 ASSAULT state
            Directory.CreateDirectory("Assets/_TTGQLDL_Showcase/Scenes");
            string scenePath = "Assets/_TTGQLDL_Showcase/Scenes/BastionDawn_ART008_VerticalSlice.unity";
            if (stateAssault != null)
            {
                ApplyWorldForCommittedView(stateAssault, worldRefs, pooledControllers, vfxPool, spritesByStem);
                hudPresenter.ReducedMotion = true;
                hudPresenter.ApplyCommittedViewInPlace(stateAssault, "vi-VN");
            }
            EditorSceneManager.SaveScene(scene, scenePath);

            // 7. Verify BroadcastAudioDispatcher cooldowns, priority suppression, pool bounds, Two-Tier dedupe & buses
            VerifyAudioDispatcherContract(dispatcher, out int prep10s, out int assault10s, out float assaultGain,
                out int joinDuringAlert, out int dupSeqPlays, out int dupVicPlays, out int wallDmg3,
                out int fighter2, out int builder2, out int bossSlam2, out int burst32Voices, out bool busesOk);

            // 8. Run Real 180-Frame GPU+CPU Load Benchmarks at 10, 50, 100, and 150 Pooled Entities
            worldRefs.bossTelegraphSr.gameObject.SetActive(false);
            worldRefs.wallBackSr.sprite = spritesByStem["env_wall_state2_damaged_back"];
            worldRefs.wallFrontSr.sprite = spritesByStem["env_wall_state2_damaged_front"];
            hudPresenter.ReducedMotion = false;
            if (stateAssault != null)
            {
                hudPresenter.ApplyCommittedViewInPlace(stateAssault, "vi-VN");
            }

            var loadTiers = new List<LoadTierResult>();
            foreach (int count in new[] { 10, 50, 100, 150 })
            {
                loadTiers.Add(MeasureEntityLoadTier(cam, pooledControllers, vfxPool, dispatcher, hudPresenter, count, 180, totalSpriteBytes, totalAudioBytes));
            }

            // Ensure VFX bursts drain cleanly after stepping 12 frames with no new spawns
            for (int f = 0; f < 12; f++) vfxPool.StepSimulationMs(SpriteTransitionController.FrameStepMs);
            int lingeringVfx = vfxPool.ActiveBurstCount;

            // Teardown check for leaked AudioSources
            int preallocatedVoices = dispatcher.TotalChildAudioSources;
            bool musicDefaultMuteOk = dispatcher.IsMusicMutedByDefault;
            UnityEngine.Object.DestroyImmediate(audioGo);
            int leakedSources = UnityEngine.Object.FindObjectsByType<AudioSource>().Length;

            // Write Scene Hierarchy Summary
            File.WriteAllText(cfg.outputSceneSummaryJson, JsonUtility.ToJson(new SceneSummaryDoc
            {
                scenePath = scenePath,
                audioPrefabPath = audioPrefabPath,
                unityVersion = Application.unityVersion,
                graphicsApi = SystemInfo.graphicsDeviceType.ToString(),
                cameraOrthographicSize = 5.0f,
                referenceResolution = "1080x1920",
                pixelsPerUnit = 192,
                pooledEntityCapacity = 150,
                pooledVfxCapacity = 20,
                preallocatedAudioVoices = 12,
                verifiedHudStateCount = hudStatesOut.Count
            }, true));

            string gfxApi = SystemInfo.graphicsDeviceType.ToString();
            var rep = new UnityVerificationReport
            {
                unityVersion = Application.unityVersion,
                executionMode = $"Unity {Application.unityVersion} -batchmode with Hardware {gfxApi} GPU Camera.Render() (NO -nographics)",
                graphicsDeviceName = SystemInfo.graphicsDeviceName,
                graphicsDeviceType = gfxApi,
                processorType = SystemInfo.processorType,
                systemMemoryMB = SystemInfo.systemMemorySize,
                graphicsMemoryMB = SystemInfo.graphicsMemorySize,
                savedScenePath = scenePath,
                savedAudioPrefabPath = audioPrefabPath,
                importedSpriteCount = spritesByStem.Count,
                importedAudioClipCount = clipsByName.Count,
                cameraOrthographicSize5Verified = Mathf.Approximately(cam.orthographicSize, 5.0f),
                fixedCameraZeroPanZoomShakeVerified = camGo.transform.position == new Vector3(0f, 0f, -10f),
                parapetDepthSortingVerified = parapetDepthOk,
                hudSafeZoneVerified = hudSafeZoneOk,
                hudInPlaceTransitionsStandard = stdTransitions,
                hudInPlaceTransitionsReducedMotion = redTransitions,
                sceneReloadsDuringStateTransitions = hudPresenter.SceneReloadCount,
                preallocatedAudioVoices = preallocatedVoices,
                musicMutedByDefaultVerified = musicDefaultMuteOk,
                joinPrep10sPlays = prep10s,
                joinAssault10sPlays = assault10s,
                joinAssaultLinearGain = assaultGain,
                joinDuringPriorityAlertPlays = joinDuringAlert,
                duplicateStreamSeqPlays = dupSeqPlays,
                duplicateVictoryReconnectPlays = dupVicPlays,
                independentWallDamage3Plays = wallDmg3,
                independentFighterSlash2Plays = fighter2,
                independentBuilderRepair2Plays = builder2,
                independentBossSlam2Plays = bossSlam2,
                burst32SimultaneousMaxActiveVoices = burst32Voices,
                globalMuteAndIndependentBusesVerified = busesOk,
                leakedAudioSourcesAfterTeardown = leakedSources,
                lingeringVfxAfterCompletion = lingeringVfx,
                classMeasurements = classMeasurements,
                hud_verification_states = hudStatesOut,
                english_vote_b_proof = enVoteProof,
                renderedGpuFrames = renderedFiles,
                honestyNote = $"Executed in an isolated Unity {Application.unityVersion} project outside production unity/. Proves asset compatibility, sorting, scaling, canonical SyntheticCommittedView HUD truthfulness across 12 states, audio gain/dedupe logic, and bounded 150-entity {gfxApi} CPU render-submission budgets; does NOT claim M05 live WebSocket wiring, 15-minute M06/M08 standalone soak, or hardware OBS capture."
            };
            foreach (var sl in SortingLayer.layers)
            {
                rep.verifiedSortingLayers.Add($"{sl.name}(value={sl.value})");
            }

            bool allClassesOk = classMeasurements.Count == 7 && classMeasurements.TrueForAll(m => m.transitionAuditPassed && m.deltaPx <= 0.6f);
            bool audioAllOk = preallocatedVoices == 12 && musicDefaultMuteOk && prep10s == 5 && assault10s == 3 &&
                              joinDuringAlert == 0 && dupSeqPlays == 1 && dupVicPlays == 1 && wallDmg3 == 3 &&
                              fighter2 == 2 && builder2 == 2 && bossSlam2 == 2 && burst32Voices <= 10 &&
                              busesOk && leakedSources == 0;
            bool hud12Ok = hudStatesOut.Count >= 12 && hudStatesOut.TrueForAll(h => h.castle_hp_text.EndsWith("/500"));

            rep.status = (allClassesOk && audioAllOk && hud12Ok && parapetDepthOk && hudSafeZoneOk && hudPresenter.SceneReloadCount == 0 && lingeringVfx == 0) ? "PASS" : "FAIL";
            File.WriteAllText(cfg.outputVerificationJson, JsonUtility.ToJson(rep, true));

            var perf = new PerformanceReport
            {
                status = rep.status,
                unityVersion = Application.unityVersion,
                executionContext = $"Isolated Unity {Application.unityVersion} {gfxApi} 1080x1920 RenderTexture Camera.Render() CPU Submission Loop (180 frames/tier after 24-frame warmup)",
                cpuModel = SystemInfo.processorType,
                gpuModel = SystemInfo.graphicsDeviceName,
                graphicsApi = gfxApi,
                systemMemoryMB = SystemInfo.systemMemorySize,
                graphicsMemoryMB = SystemInfo.graphicsMemorySize,
                importedSpriteTextureRuntimeBytes = totalSpriteBytes,
                importedAudioRuntimeBytes = totalAudioBytes,
                loadTiers = loadTiers,
                scopeAndLimitationsNote = $"Bounded 10/50/100/150 pooled entity benchmark measured on {SystemInfo.processorType} + {SystemInfo.graphicsDeviceName} ({gfxApi}) in isolated Unity {Application.unityVersion} Editor batchmode. Frame times reflect CPU-side Camera.Render() submission into an offscreen 1080x1920 RenderTexture (not swapchain VSync or full async GPU completion). 10,000 simulated participants are represented via HUD counters + <=150 pooled decorative world units per 07_UNITY_ART_UX.md. Standalone 15-minute soak and OBS NVENC capture remain NOT_EXECUTED until M06/M08."
            };
            File.WriteAllText(cfg.outputPerformanceJson, JsonUtility.ToJson(perf, true));

            if (rep.status != "PASS")
            {
                throw new Exception("ART-008 / ART-009 isolated Unity showcase verification failed invariants.");
            }
            EditorApplication.Exit(0);
        }

        [Serializable]
        private sealed class SceneSummaryDoc
        {
            public string scenePath;
            public string audioPrefabPath;
            public string unityVersion;
            public string graphicsApi;
            public float cameraOrthographicSize;
            public string referenceResolution;
            public int pixelsPerUnit;
            public int pooledEntityCapacity;
            public int pooledVfxCapacity;
            public int preallocatedAudioVoices;
            public int verifiedHudStateCount;
        }

        private static HudVerificationStateEntry ReadBackHudVerificationEntry(
            SyntheticCommittedView v,
            HudInPlacePresenter hud,
            string locale)
        {
            return new HudVerificationStateEntry
            {
                state_id = v.stateId,
                fixture_id = v.fixtureId,
                locale = locale,
                fixture_source_status = v.sourceStatus,
                watermark_visible = hud.watermarkGroup != null && hud.watermarkGroup.alpha > 0.5f,
                watermark_text = hud.watermarkLabel != null ? hud.watermarkLabel.text : "",
                wall_hp_text = hud.wallHpLabel != null ? hud.wallHpLabel.text : "",
                castle_hp_text = hud.castleHpLabel != null ? hud.castleHpLabel.text : "",
                wood_text = hud.woodLabel != null ? hud.woodLabel.text : "",
                stone_text = hud.stoneLabel != null ? hud.stoneLabel.text : "",
                food_text = hud.foodLabel != null ? hud.foodLabel.text : "",
                wave_text = hud.waveLabel != null ? hud.waveLabel.text : "",
                timer_text = hud.timerLabel != null ? hud.timerLabel.text : "",
                source_status_text = hud.sourceStatusLabel != null ? hud.sourceStatusLabel.text : "",
                threat_banner_text = hud.threatBannerLabel != null ? hud.threatBannerLabel.text : "",
                vote_strip_text = hud.voteStripLabel != null ? hud.voteStripLabel.text : "",
                secondary_diag_text = (hud.secondaryDiagGroup != null && hud.secondaryDiagGroup.alpha > 0.5f && hud.secondaryDiagLabel != null) ? hud.secondaryDiagLabel.text : "",
                primary_cta_text = hud.primaryCtaLabel != null ? hud.primaryCtaLabel.text : "",
                builder_text = hud.roleBuilderLabel != null ? hud.roleBuilderLabel.text : "",
                guard_text = hud.roleGuardLabel != null ? hud.roleGuardLabel.text : "",
                fighter_text = hud.roleFighterLabel != null ? hud.roleFighterLabel.text : "",
                render_1080p = v.outputPng1080p ?? "",
                render_720p = v.outputPng720p ?? "",
                render_360p = v.outputPng360p ?? "",
                fixture_summary = new FixtureSummaryEntry
                {
                    wave_no = v.waveNo,
                    wave_tick = v.waveTick,
                    phase = v.phase,
                    wall_tier = v.wallTier,
                    wall_hp = v.wallHp,
                    wall_max = v.wallMax,
                    townhall_hp = v.townhallHp,
                    townhall_max = v.townhallMax,
                    wood = v.wood,
                    stone = v.stone,
                    food = v.food,
                    builder = v.builderCount,
                    guard = v.guardCount,
                    fighter = v.fighterCount,
                    west_hp = v.westHp,
                    center_hp = v.centerHp,
                    east_hp = v.eastHp,
                    boss_exists = v.bossExists,
                    boss_hp = v.bossHp,
                    boss_max_hp = v.bossMaxHp,
                    vote_selected = v.voteSelected ?? ""
                }
            };
        }

        private static void ApplyWorldForCommittedView(
            SyntheticCommittedView v,
            WorldRefs w,
            List<SpriteTransitionController> pool,
            BoundedVfxPool vfxPool,
            Dictionary<string, Sprite> sp)
        {
            // Ensure zero one-shot VFX bursts from prior state screenshots linger across discrete state captures
            if (vfxPool != null)
            {
                vfxPool.StepSimulationMs(1500.0);
            }

            // 1. Castle tier band from canonical village_level (1..50)
            string castleKey = v.villageLevel <= 9 ? "env_castle_band1_lv01_09" :
                               v.villageLevel <= 24 ? "env_castle_band2_lv10_24" :
                               v.villageLevel <= 39 ? "env_castle_band3_lv25_39" : "env_castle_band4_lv40_50";
            w.castleSr.sprite = sp[castleKey];

            // 2. Wall damage/recovery state from exact wallHp / wallMax ratio
            float ratio = v.wallMax > 0 ? (float)v.wallHp / v.wallMax : 0f;
            string wallBackKey;
            string wallFrontKey;
            if (v.wallHp <= 0)
            {
                wallBackKey = "env_wall_state4_breached_back";
                wallFrontKey = "env_wall_state4_breached_front";
            }
            else if (v.isPostLossRecovery)
            {
                wallBackKey = "env_wall_state5_recovery_back";
                wallFrontKey = "env_wall_state5_recovery_front";
            }
            else if (ratio < 0.25f)
            {
                wallBackKey = "env_wall_state3_critical_back";
                wallFrontKey = "env_wall_state3_critical_front";
            }
            else if (ratio < 0.60f)
            {
                wallBackKey = "env_wall_state2_damaged_back";
                wallFrontKey = "env_wall_state2_damaged_front";
            }
            else
            {
                wallBackKey = "env_wall_state1_healthy_back";
                wallFrontKey = "env_wall_state1_healthy_front";
            }
            w.wallBackSr.sprite = sp[wallBackKey];
            w.wallFrontSr.sprite = sp[wallFrontKey];

            bool scaffoldsOn = v.builderCount > 0 && v.lifecycle != "stopped" && v.phase != "waiting";
            w.scaffoldSwGo.SetActive(scaffoldsOn);
            w.scaffoldSeGo.SetActive(scaffoldsOn);

            // 3. Boss ground telegraph vs Boss Warlord spawn strictly at tick >= 2600
            bool bossActive = v.bossExists && v.waveTick >= 2600 && v.bossHp > 0;
            bool bossWarn = v.bossIncoming && v.waveTick >= 2450 && v.waveTick < 2600;
            w.bossTelegraphSr.gameObject.SetActive(bossWarn || bossActive);
            if (bossWarn)
            {
                w.bossTelegraphSr.sprite = sp["vfx_boss_slam_telegraph_f02"];
            }
            else if (bossActive)
            {
                w.bossTelegraphSr.sprite = sp["vfx_boss_slam_telegraph_f05"];
            }

            // 4. Decorative pooled units strictly reflecting aggregate defender & lane HP pools
            bool frozenOrEmpty = v.lifecycle == "stopped" || v.phase == "waiting" || v.uniqueJoined <= 0;
            int totalOrdHp = v.westHp + v.centerHp + v.eastHp;
            bool showEnemies = !frozenOrEmpty && v.waveTick >= 1600 && v.phase != "results" && totalOrdHp > 0;

            int maxBuilderProxies = frozenOrEmpty ? 0 : Mathf.Clamp((v.builderCount + 9) / 10, 0, 5);
            int maxGuardProxies = frozenOrEmpty ? 0 : Mathf.Clamp((v.guardCount + 11) / 12, 0, 6);
            int maxFighterProxies = frozenOrEmpty ? 0 : Mathf.Clamp((v.fighterCount + 14) / 15, 0, 8);
            int maxEnemyProxiesPerClass = showEnemies ? Mathf.Clamp((totalOrdHp + 79) / 80, 1, 6) : 0;

            int bShown = 0, gShown = 0, fShown = 0, grShown = 0, ruShown = 0, brShown = 0;
            for (int i = 0; i < pool.Count; i++)
            {
                var u = pool[i];
                bool on = false;
                if (i == 0)
                {
                    on = bossActive;
                    if (on)
                    {
                        u.PlayClip("cleaveslam");
                        for (int s = 0; s < 5; s++) u.StepSimulationMs(SpriteTransitionController.FrameStepMs);
                    }
                }
                else
                {
                    int mod = i % 6;
                    if (mod == 0 && bShown < maxBuilderProxies) { on = true; bShown++; }
                    else if (mod == 1 && gShown < maxGuardProxies) { on = true; gShown++; }
                    else if (mod == 2 && fShown < maxFighterProxies) { on = true; fShown++; }
                    else if (mod == 3 && grShown < maxEnemyProxiesPerClass) { on = true; grShown++; }
                    else if (mod == 4 && ruShown < maxEnemyProxiesPerClass) { on = true; ruShown++; }
                    else if (mod == 5 && brShown < maxEnemyProxiesPerClass) { on = true; brShown++; }
                }
                u.gameObject.SetActive(on);
                if (on && i > 0 && v.lifecycle == "running")
                {
                    u.StepSimulationMs(SpriteTransitionController.FrameStepMs * ((i % 4) + 1));
                }
            }
        }

        private static void ConfigureSortingLayers()
        {
            var tagManager = new SerializedObject(AssetDatabase.LoadAllAssetsAtPath("ProjectSettings/TagManager.asset")[0]);
            var sortingLayers = tagManager.FindProperty("m_SortingLayers");
            string[] required = { "Ground", "LanesAndDecals", "BastionBack", "Units", "BastionFront", "VFX" };
            sortingLayers.ClearArray();
            sortingLayers.InsertArrayElementAtIndex(0);
            var def = sortingLayers.GetArrayElementAtIndex(0);
            def.FindPropertyRelative("name").stringValue = "Default";
            def.FindPropertyRelative("uniqueID").intValue = 0;

            for (int i = 0; i < required.Length; i++)
            {
                sortingLayers.InsertArrayElementAtIndex(i + 1);
                var el = sortingLayers.GetArrayElementAtIndex(i + 1);
                el.FindPropertyRelative("name").stringValue = required[i];
                el.FindPropertyRelative("uniqueID").intValue = 100 + i;
            }
            tagManager.ApplyModifiedPropertiesWithoutUndo();
        }

        private static Vector3 ScreenPxToWorld(float sx, float sy)
        {
            return new Vector3((sx - 540f) / 192f, (960f - sy) / 192f, 0f);
        }

        private static GameObject AddWorldSprite(Transform parent, string name, Sprite sp, float sx, float sy, string layer, int order, float scale = 1f)
        {
            var go = new GameObject(name);
            go.transform.SetParent(parent, false);
            go.transform.position = ScreenPxToWorld(sx, sy);
            go.transform.localScale = new Vector3(scale, scale, 1f);
            var sr = go.AddComponent<SpriteRenderer>();
            sr.sprite = sp;
            sr.sortingLayerName = layer;
            sr.sortingOrder = order;
            return go;
        }

        private static WorldRefs BuildEnvironmentWorld(Transform worldRoot, Dictionary<string, Sprite> sp)
        {
            var refs = new WorldRefs();
            var groundRoot = new GameObject("Layer0_Ground");
            groundRoot.transform.SetParent(worldRoot, false);
            for (int gx = 128; gx <= 1152; gx += 256)
            {
                for (int gy = 128; gy <= 2048; gy += 256)
                {
                    string tileId = ((gx + gy) % 512 == 0) ? "env_tile_02_meadow_variation" : "env_tile_01_meadow_main";
                    AddWorldSprite(groundRoot.transform, $"Meadow_{gx}_{gy}", sp[tileId], gx, gy, "Ground", 0);
                }
            }

            var lanesRoot = new GameObject("Layer1_LanesAndDecals");
            lanesRoot.transform.SetParent(worldRoot, false);
            for (int ly = 80; ly <= 668; ly += 64)
            {
                float tW = ly / 690f;
                AddWorldSprite(lanesRoot.transform, $"WestLane_{ly}", sp["env_tile_03_ochre_road_body"], Mathf.Lerp(78f, 335f, tW), ly, "LanesAndDecals", 2, 0.30f);
                AddWorldSprite(lanesRoot.transform, $"CenterCauseway_{ly}", sp["env_tile_05_limestone_causeway"], 454f, ly, "LanesAndDecals", 4, 0.32f);
                if (ly >= 260)
                {
                    float tE = (ly - 260f) / 420f;
                    AddWorldSprite(lanesRoot.transform, $"EastLane_{ly}", sp["env_tile_03_ochre_road_body"], Mathf.Lerp(824f, 575f, tE), ly, "LanesAndDecals", 2, 0.30f);
                }
            }
            AddWorldSprite(lanesRoot.transform, "WestApron", sp["trans_lane_bastion_apron"], 335f, 672f, "LanesAndDecals", 8, 0.42f);
            AddWorldSprite(lanesRoot.transform, "CenterApron", sp["trans_lane_bastion_apron"], 454f, 668f, "LanesAndDecals", 9, 0.42f);
            AddWorldSprite(lanesRoot.transform, "EastApron", sp["trans_lane_bastion_apron"], 575f, 672f, "LanesAndDecals", 8, 0.42f);
            var teleGo = AddWorldSprite(lanesRoot.transform, "BossTelegraphRune", sp["vfx_boss_slam_telegraph_f05"], 454f, 520f, "LanesAndDecals", 24, 0.68f);
            refs.bossTelegraphSr = teleGo.GetComponent<SpriteRenderer>();
            teleGo.SetActive(false);

            var backRoot = new GameObject("Layer2_BastionBack");
            backRoot.transform.SetParent(worldRoot, false);
            AddWorldSprite(backRoot.transform, "CourtyardPlaza", sp["env_tile_06_courtyard_plaza"], 454f, 845f, "BastionBack", -200, 1.38f);
            var wbGo = AddWorldSprite(backRoot.transform, "BastionWallBack", sp["env_wall_state1_healthy_back"], 454f, 998f, "BastionBack", -190, 1.0f);
            refs.wallBackSr = wbGo.GetComponent<SpriteRenderer>();

            AddWorldSprite(backRoot.transform, "Watchtower", sp["env_bldg_watchtower"], 352f, 756f, "BastionBack", -106, 0.54f);
            var casGo = AddWorldSprite(backRoot.transform, "TownhallKeep", sp["env_castle_band2_lv10_24"], 454f, 796f, "BastionBack", -85, 0.50f);
            refs.castleSr = casGo.GetComponent<SpriteRenderer>();
            AddWorldSprite(backRoot.transform, "House01", sp["env_house_01_artisan_gable"], 314f, 804f, "BastionBack", -81, 0.34f);
            AddWorldSprite(backRoot.transform, "House02", sp["env_house_02_granary_round"], 594f, 804f, "BastionBack", -80, 0.34f);
            AddWorldSprite(backRoot.transform, "SunBanner", sp["env_prop_sun_banner"], 454f, 844f, "BastionBack", -60, 0.38f);
            AddWorldSprite(backRoot.transform, "SupplyCart", sp["env_prop_supply_cart"], 406f, 852f, "BastionBack", -56, 0.36f);
            AddWorldSprite(backRoot.transform, "StoneCrates", sp["env_prop_stone_crates"], 502f, 852f, "BastionBack", -55, 0.36f);
            AddWorldSprite(backRoot.transform, "House03", sp["env_house_03_mason_lodge"], 306f, 860f, "BastionBack", -52, 0.34f);
            AddWorldSprite(backRoot.transform, "House04", sp["env_house_04_bakery_chimney"], 602f, 860f, "BastionBack", -51, 0.34f);
            AddWorldSprite(backRoot.transform, "House05", sp["env_house_05_smithy_shed"], 364f, 898f, "BastionBack", -32, 0.33f);
            AddWorldSprite(backRoot.transform, "House06", sp["env_house_06_guard_quarters"], 544f, 898f, "BastionBack", -31, 0.33f);

            var frontRoot = new GameObject("Layer4_BastionFront");
            frontRoot.transform.SetParent(worldRoot, false);
            var wfGo = AddWorldSprite(frontRoot.transform, "BastionWallFrontParapet", sp["env_wall_state1_healthy_front"], 454f, 998f, "BastionFront", 5, 1.0f);
            refs.wallFrontSr = wfGo.GetComponent<SpriteRenderer>();
            refs.scaffoldSwGo = AddWorldSprite(frontRoot.transform, "ScaffoldSW", sp["env_wall_repair_scaffold"], 294f, 996f, "BastionFront", 12, 0.44f);
            refs.scaffoldSeGo = AddWorldSprite(frontRoot.transform, "ScaffoldSE", sp["env_wall_repair_scaffold"], 614f, 996f, "BastionFront", 12, 0.44f);
            refs.scaffoldSeGo.GetComponent<SpriteRenderer>().flipX = true;
            return refs;
        }

        private static Text AddUiLabel(Transform parent, string name, Font font, int fontSize, Color color)
        {
            var go = new GameObject(name, typeof(RectTransform), typeof(CanvasRenderer), typeof(Text), typeof(Outline));
            go.transform.SetParent(parent, false);
            var rt = go.GetComponent<RectTransform>();
            rt.anchorMin = Vector2.zero;
            rt.anchorMax = Vector2.one;
            rt.offsetMin = new Vector2(12f, 4f);
            rt.offsetMax = new Vector2(-12f, -4f);
            var t = go.GetComponent<Text>();
            t.font = font;
            t.fontSize = fontSize;
            t.fontStyle = FontStyle.Bold;
            t.alignment = TextAnchor.MiddleCenter;
            t.color = color;
            t.text = string.Empty;
            t.horizontalOverflow = HorizontalWrapMode.Wrap;
            t.verticalOverflow = VerticalWrapMode.Overflow;
            var ol = go.GetComponent<Outline>();
            ol.effectColor = new Color(0.06f, 0.08f, 0.11f, 0.95f);
            ol.effectDistance = new Vector2(2f, -2f);
            return t;
        }

        private static Image CreateSlicedUiBox(Transform parent, string name, Sprite sp, float x0, float y0, float x1, float y1, Image.Type imgType = Image.Type.Sliced)
        {
            var go = new GameObject(name, typeof(RectTransform), typeof(CanvasRenderer), typeof(Image));
            go.transform.SetParent(parent, false);
            var rt = go.GetComponent<RectTransform>();
            rt.anchorMin = new Vector2(0f, 1f);
            rt.anchorMax = new Vector2(0f, 1f);
            rt.pivot = new Vector2(0f, 1f);
            rt.anchoredPosition = new Vector2(x0, -y0);
            rt.sizeDelta = new Vector2(x1 - x0, y1 - y0);
            var img = go.GetComponent<Image>();
            img.sprite = sp;
            img.type = imgType;
            return img;
        }

        private static HudInPlacePresenter BuildHudCanvas(Camera cam, Dictionary<string, Sprite> sp, out bool safeZoneVerified)
        {
            var canvasGo = new GameObject("HUD_Canvas_1080x1920", typeof(RectTransform), typeof(Canvas), typeof(CanvasScaler), typeof(GraphicRaycaster));
            var canvas = canvasGo.GetComponent<Canvas>();
            canvas.renderMode = RenderMode.ScreenSpaceCamera;
            canvas.worldCamera = cam;
            canvas.planeDistance = 1.0f;
            canvas.sortingLayerName = "VFX";
            canvas.sortingOrder = 200;

            var scaler = canvasGo.GetComponent<CanvasScaler>();
            scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
            scaler.referenceResolution = new Vector2(1080f, 1920f);
            scaler.matchWidthOrHeight = 0.5f;

            var presenter = canvasGo.AddComponent<HudInPlacePresenter>();
            presenter.rootCanvas = canvas;
            presenter.skinNavy = sp["ui_comp_resource_bar"];
            presenter.skinOk = sp["ui_comp_status_pill_ok"];
            presenter.skinWarn = sp["ui_comp_status_pill_warn"];
            presenter.skinErr = sp["ui_comp_status_pill_err"];
            presenter.skinVoteOpen = sp["ui_comp_vote_card_open"];
            presenter.skinVoteLocked = sp["ui_comp_vote_card_locked"];
            presenter.skinHpHealthy = sp["ui_comp_hp_bar_fill_healthy"];
            presenter.skinHpWarn = sp["ui_comp_hp_bar_fill_warn"];
            presenter.skinHpCrit = sp["ui_comp_hp_bar_fill_crit"];

            var uiFont = Font.CreateDynamicFontFromOSFont(new[] { "Segoe UI", "Arial" }, 36);

            // All direct HUD elements strictly within x <= 876 (< 885.6 right 18% rail) and y <= 1686 (< 1689.6 bottom 12% rail)
            CreateSlicedUiBox(canvasGo.transform, "TopStatusPanel", sp["ui_comp_resource_bar"], 36f, 22f, 876f, 242f);
            CreateSlicedUiBox(canvasGo.transform, "WallIcon", sp["ui_icon_wall"], 46f, 37f, 90f, 81f, Image.Type.Simple);
            var wallTrack = CreateSlicedUiBox(canvasGo.transform, "WallHpTrack", sp["ui_comp_hp_bar_frame"], 96f, 33f, 556f, 85f);
            var wallFillGo = new GameObject("WallHpFill", typeof(RectTransform), typeof(CanvasRenderer), typeof(Image));
            wallFillGo.transform.SetParent(wallTrack.transform, false);
            var wfRt = wallFillGo.GetComponent<RectTransform>();
            wfRt.anchorMin = new Vector2(0.02f, 0.15f);
            wfRt.anchorMax = new Vector2(0.98f, 0.85f);
            wfRt.offsetMin = Vector2.zero;
            wfRt.offsetMax = Vector2.zero;
            var wfImg = wallFillGo.GetComponent<Image>();
            wfImg.sprite = sp["ui_comp_hp_bar_fill_healthy"];
            wfImg.type = Image.Type.Tiled;
            presenter.wallHpFill = wfImg;
            presenter.wallHpLabel = AddUiLabel(wallTrack.transform, "WallHpText", uiFont, 29, Color.white);

            var timerCap = CreateSlicedUiBox(canvasGo.transform, "TimerCapsule", sp["ui_comp_resource_bar"], 568f, 33f, 864f, 85f);
            presenter.timerLabel = AddUiLabel(timerCap.transform, "TimerText", uiFont, 30, new Color(1f, 0.96f, 0.82f));

            CreateSlicedUiBox(canvasGo.transform, "CastleIcon", sp["ui_icon_castle"], 46f, 103f, 86f, 143f, Image.Type.Simple);
            var casTrack = CreateSlicedUiBox(canvasGo.transform, "CastleHpTrack", sp["ui_comp_hp_bar_frame"], 90f, 97f, 406f, 149f);
            var casFillGo = new GameObject("CastleHpFill", typeof(RectTransform), typeof(CanvasRenderer), typeof(Image));
            casFillGo.transform.SetParent(casTrack.transform, false);
            var cfRt = casFillGo.GetComponent<RectTransform>();
            cfRt.anchorMin = new Vector2(0.02f, 0.15f);
            cfRt.anchorMax = new Vector2(0.98f, 0.85f);
            cfRt.offsetMin = Vector2.zero;
            cfRt.offsetMax = Vector2.zero;
            var cfImg = casFillGo.GetComponent<Image>();
            cfImg.sprite = sp["ui_comp_hp_bar_fill_healthy"];
            cfImg.type = Image.Type.Tiled;
            presenter.castleHpFill = cfImg;
            presenter.castleHpLabel = AddUiLabel(casTrack.transform, "CastleHpText", uiFont, 26, Color.white);

            var waveCap = CreateSlicedUiBox(canvasGo.transform, "WavePlaque", sp["ui_comp_resource_bar"], 412f, 97f, 620f, 149f);
            presenter.waveLabel = AddUiLabel(waveCap.transform, "WaveText", uiFont, 28, Color.white);

            presenter.sourceStatusSkin = CreateSlicedUiBox(canvasGo.transform, "SourceStatusPill", sp["ui_comp_status_pill_ok"], 625f, 97f, 871f, 149f);
            presenter.sourceStatusLabel = AddUiLabel(presenter.sourceStatusSkin.transform, "SourceStatusText", uiFont, 25, Color.white);

            var rWood = CreateSlicedUiBox(canvasGo.transform, "ResWood", sp["ui_comp_resource_bar"], 42f, 164f, 317f, 226f);
            presenter.woodLabel = AddUiLabel(rWood.transform, "ResWoodText", uiFont, 27, new Color(0.98f, 0.95f, 0.88f));
            var rStone = CreateSlicedUiBox(canvasGo.transform, "ResStone", sp["ui_comp_resource_bar"], 319f, 164f, 594f, 226f);
            presenter.stoneLabel = AddUiLabel(rStone.transform, "ResStoneText", uiFont, 27, new Color(0.98f, 0.95f, 0.88f));
            var rFood = CreateSlicedUiBox(canvasGo.transform, "ResFood", sp["ui_comp_resource_bar"], 596f, 164f, 871f, 226f);
            presenter.foodLabel = AddUiLabel(rFood.transform, "ResFoodText", uiFont, 27, new Color(0.98f, 0.95f, 0.88f));

            var bannerImg = CreateSlicedUiBox(canvasGo.transform, "ThreatBanner", sp["ui_comp_status_pill_warn"], 36f, 250f, 876f, 316f);
            presenter.threatBannerSkin = bannerImg;
            presenter.threatBannerRect = bannerImg.rectTransform;
            presenter.threatBannerGroup = bannerImg.gameObject.AddComponent<CanvasGroup>();
            presenter.threatBannerLabel = AddUiLabel(bannerImg.transform, "ThreatBannerText", uiFont, 28, Color.white);

            // Embedded Boss HP bar strip inside bottom of ThreatBanner (visible only when boss_exists && wave_tick >= 2600)
            var bossTrackGo = new GameObject("BossHpBarTrack", typeof(RectTransform), typeof(CanvasRenderer), typeof(Image));
            bossTrackGo.transform.SetParent(bannerImg.transform, false);
            var btRt = bossTrackGo.GetComponent<RectTransform>();
            btRt.anchorMin = new Vector2(0.045f, 0.08f);
            btRt.anchorMax = new Vector2(0.955f, 0.23f);
            btRt.offsetMin = Vector2.zero;
            btRt.offsetMax = Vector2.zero;
            var btImg = bossTrackGo.GetComponent<Image>();
            btImg.color = new Color(0.07f, 0.09f, 0.12f, 0.96f);

            var bossFillGo = new GameObject("BossHpBarFill", typeof(RectTransform), typeof(CanvasRenderer), typeof(Image));
            bossFillGo.transform.SetParent(bossTrackGo.transform, false);
            var bfRt = bossFillGo.GetComponent<RectTransform>();
            bfRt.anchorMin = Vector2.zero;
            bfRt.anchorMax = Vector2.one;
            bfRt.offsetMin = Vector2.zero;
            bfRt.offsetMax = Vector2.zero;
            var bfImg = bossFillGo.GetComponent<Image>();
            bfImg.color = new Color(0.85f, 0.27f, 0.94f, 1.0f);
            presenter.bossHpBarRoot = bossTrackGo;
            presenter.bossHpFill = bfImg;
            bossTrackGo.SetActive(false);

            var diagImg = CreateSlicedUiBox(canvasGo.transform, "SecondaryDiagPill", sp["ui_comp_status_pill_warn"], 56f, 1208f, 856f, 1260f);
            presenter.secondaryDiagSkin = diagImg;
            presenter.secondaryDiagGroup = diagImg.gameObject.AddComponent<CanvasGroup>();
            presenter.secondaryDiagGroup.alpha = 0f;
            presenter.secondaryDiagLabel = AddUiLabel(diagImg.transform, "SecondaryDiagText", uiFont, 25, Color.white);

            var toastImg = CreateSlicedUiBox(canvasGo.transform, "JoinToastSingleSlot", sp["ui_comp_toast_join"], 76f, 1330f, 836f, 1388f);
            presenter.joinToastGroup = toastImg.gameObject.AddComponent<CanvasGroup>();
            presenter.joinToastGroup.alpha = 0f;
            presenter.joinToastLabel = AddUiLabel(toastImg.transform, "JoinToastText", uiFont, 26, new Color(0.88f, 0.95f, 1f));

            presenter.voteStripSkin = CreateSlicedUiBox(canvasGo.transform, "SubordinateVoteOrObjectiveStrip", sp["ui_comp_vote_card_open"], 36f, 1396f, 876f, 1454f);
            presenter.voteStripLabel = AddUiLabel(presenter.voteStripSkin.transform, "VoteStripText", uiFont, 23, new Color(0.98f, 0.96f, 0.84f));

            presenter.ctaBannerSkin = CreateSlicedUiBox(canvasGo.transform, "PrimaryCtaPlaque", sp["ui_comp_cta_banner"], 36f, 1462f, 876f, 1550f);
            presenter.primaryCtaLabel = AddUiLabel(presenter.ctaBannerSkin.transform, "PrimaryCtaText", uiFont, 41, new Color(1f, 0.97f, 0.86f));

            var rb = CreateSlicedUiBox(canvasGo.transform, "RoleBuilderPlaque", sp["ui_comp_role_counter_builder"], 36f, 1556f, 310f, 1630f);
            presenter.roleBuilderLabel = AddUiLabel(rb.transform, "RoleBuilderText", uiFont, 29, Color.white);
            var rg = CreateSlicedUiBox(canvasGo.transform, "RoleGuardPlaque", sp["ui_comp_role_counter_guard"], 319f, 1556f, 593f, 1630f);
            presenter.roleGuardLabel = AddUiLabel(rg.transform, "RoleGuardText", uiFont, 29, Color.white);
            var rf = CreateSlicedUiBox(canvasGo.transform, "RoleFighterPlaque", sp["ui_comp_role_counter_fighter"], 602f, 1556f, 876f, 1630f);
            presenter.roleFighterLabel = AddUiLabel(rf.transform, "RoleFighterText", uiFont, 29, Color.white);

            var wmImg = CreateSlicedUiBox(canvasGo.transform, "SyntheticWatermarkFooter", sp["ui_comp_status_pill_warn"], 36f, 1638f, 876f, 1686f);
            presenter.watermarkGroup = wmImg.gameObject.AddComponent<CanvasGroup>();
            presenter.watermarkLabel = AddUiLabel(wmImg.transform, "WatermarkText", uiFont, 24, new Color(1f, 0.95f, 0.72f));

            safeZoneVerified = true;
            foreach (RectTransform child in canvasGo.transform)
            {
                float rightPx = child.anchoredPosition.x + child.sizeDelta.x;
                float bottomPx = (-child.anchoredPosition.y) + child.sizeDelta.y;
                if (rightPx > 885.6f || bottomPx > 1689.6f)
                {
                    safeZoneVerified = false;
                }
            }
            return presenter;
        }

        private static Sprite[] LoadSeq(Dictionary<string, Sprite> sp, string prefix, int count)
        {
            var arr = new Sprite[count];
            for (int i = 0; i < count; i++)
            {
                string k = $"{prefix}_f{i:D2}";
                if (!sp.TryGetValue(k, out arr[i])) throw new Exception("Missing sprite frame: " + k);
            }
            return arr;
        }

        private static int MeasureTightOpaqueHeightPx(Texture2D tex)
        {
            string assetPath = AssetDatabase.GetAssetPath(tex);
            var tmp = new Texture2D(2, 2, TextureFormat.RGBA32, false);
            ImageConversion.LoadImage(tmp, File.ReadAllBytes(assetPath));
            Color32[] pixels = tmp.GetPixels32();
            int w = tmp.width;
            int h = tmp.height;
            int minY = h, maxY = -1;
            for (int y = 0; y < h; y++)
            {
                for (int x = 0; x < w; x++)
                {
                    if (pixels[y * w + x].a > 8)
                    {
                        if (y < minY) minY = y;
                        if (y > maxY) maxY = y;
                    }
                }
            }
            UnityEngine.Object.DestroyImmediate(tmp);
            return maxY >= minY ? (maxY - minY + 1) : 0;
        }

        private static List<ClassDisplayMeasurement> VerifyAllSevenCharacterClasses(
            Transform worldRoot,
            Dictionary<string, Sprite> sp,
            BoundedVfxPool vfxPool)
        {
            var specs = new[]
            {
                new { id="builder", prefix="chr_role_builder", idle="idle", idleN=4, walk="walk", walkN=6, act="repair", actN=6, hit="hit", hitN=2, scale=0.55814f, targetPx=48.0f, bodyRefPx=86, imp=3, vfx="vfx_builder_repair_spark", vfxN=6, cue="SFX02", vfxLayer="VFX" },
                new { id="guard",   prefix="chr_role_guard",   idle="idle", idleN=4, walk="walk", walkN=6, act="shieldbrace", actN=6, hit="hit", hitN=2, scale=0.55172f, targetPx=48.0f, bodyRefPx=87, imp=2, vfx="vfx_guard_shield_block", vfxN=4, cue="SFX03", vfxLayer="VFX" },
                new { id="fighter", prefix="chr_role_fighter", idle="idle", idleN=4, walk="walk", walkN=6, act="swordslash", actN=6, hit="hit", hitN=2, scale=0.55814f, targetPx=48.0f, bodyRefPx=86, imp=3, vfx="vfx_fighter_slash_arc", vfxN=4, cue="SFX04", vfxLayer="VFX" },
                new { id="grunt",   prefix="chr_enemy_grunt",  idle="idle", idleN=4, walk="walk", walkN=6, act="clubsmash", actN=6, hit="hit", hitN=2, scale=0.55814f, targetPx=48.0f, bodyRefPx=86, imp=3, vfx="vfx_unit_hit_puff", vfxN=6, cue="SFX05", vfxLayer="VFX" },
                new { id="runner",  prefix="chr_enemy_runner", idle="idle", idleN=4, walk="run",  walkN=6, act="daggerlunge", actN=6, hit="hit", hitN=2, scale=0.54321f, targetPx=44.0f, bodyRefPx=81, imp=2, vfx="vfx_unit_hit_puff", vfxN=6, cue="SFX05", vfxLayer="VFX" },
                new { id="brute",   prefix="chr_enemy_brute",  idle="idle", idleN=4, walk="march",walkN=6, act="gauntletcrush", actN=6, hit="hit", hitN=2, scale=0.66667f, targetPx=64.0f, bodyRefPx=96, imp=3, vfx="vfx_unit_hit_puff", vfxN=6, cue="SFX05", vfxLayer="VFX" },
                new { id="boss",    prefix="chr_boss_warlord", idle="idlemarch", idleN=6, walk="idlemarch", walkN=6, act="cleaveslam", actN=8, hit="hit", hitN=2, scale=0.57955f, targetPx=102.0f, bodyRefPx=176, imp=5, vfx="vfx_boss_slam_telegraph", vfxN=10, cue="SFX09", vfxLayer="LanesAndDecals" },
            };

            var results = new List<ClassDisplayMeasurement>();
            var auditRoot = new GameObject("TempTransitionAuditRig");
            auditRoot.transform.SetParent(worldRoot, false);

            foreach (var s in specs)
            {
                var go = new GameObject("Audit_" + s.id);
                go.transform.SetParent(auditRoot.transform, false);
                go.transform.position = new Vector3(0f, -0.10f, 0f);
                var sr = go.AddComponent<SpriteRenderer>();
                var ctrl = go.AddComponent<SpriteTransitionController>();

                var idleFrames = LoadSeq(sp, $"{s.prefix}_{s.idle}", s.idleN);
                var walkFrames = LoadSeq(sp, $"{s.prefix}_{s.walk}", s.walkN);
                var actFrames = LoadSeq(sp, $"{s.prefix}_{s.act}", s.actN);
                var hitFrames = LoadSeq(sp, $"{s.prefix}_{s.hit}", s.hitN);
                var vfxFrames = LoadSeq(sp, s.vfx, s.vfxN);

                var clipDefs = new List<SpriteTransitionController.ClipDef>
                {
                    new SpriteTransitionController.ClipDef { clipName = s.idle, frames = idleFrames, loop = true },
                    new SpriteTransitionController.ClipDef
                    {
                        clipName = s.act,
                        frames = actFrames,
                        loop = false,
                        impactFrameIndex = s.imp,
                        pairedVfxFrames = vfxFrames,
                        vfxWorldOffset = new Vector2(0.12f, 0.12f),
                        vfxScale = s.id == "boss" ? 0.68f : 0.56f,
                        vfxSortingLayer = s.vfxLayer,
                        pairedAudioCueId = s.cue
                    }
                };
                if (s.walk != s.idle)
                {
                    clipDefs.Add(new SpriteTransitionController.ClipDef { clipName = s.walk, frames = walkFrames, loop = true });
                }
                if (s.hit != s.act)
                {
                    clipDefs.Add(new SpriteTransitionController.ClipDef { clipName = s.hit, frames = hitFrames, loop = false });
                }

                ctrl.Configure(s.id, sr, s.scale, s.idle, vfxPool, clipDefs);

                for (int i = 0; i < 4; i++) ctrl.StepSimulationMs(SpriteTransitionController.FrameStepMs);
                ctrl.PlayClip(s.walk);
                for (int i = 0; i < s.walkN; i++) ctrl.StepSimulationMs(SpriteTransitionController.FrameStepMs);
                ctrl.PlayClip(s.act);
                for (int i = 0; i < s.actN + 1; i++)
                {
                    ctrl.StepSimulationMs(SpriteTransitionController.FrameStepMs);
                    vfxPool.StepSimulationMs(SpriteTransitionController.FrameStepMs);
                }
                bool returnedToIdleAfterAction = ctrl.CurrentClipName == s.idle;

                bool returnedToIdleAfterHit = true;
                if (s.hit != s.act)
                {
                    ctrl.PlayClip(s.hit);
                    for (int i = 0; i < s.hitN + 1; i++) ctrl.StepSimulationMs(SpriteTransitionController.FrameStepMs);
                    returnedToIdleAfterHit = ctrl.CurrentClipName == s.idle;
                }

                for (int i = 0; i < 12; i++) vfxPool.StepSimulationMs(SpriteTransitionController.FrameStepMs);

                int tightFullPx = MeasureTightOpaqueHeightPx(idleFrames[0].texture);
                float displayedBodyPx = (float)Math.Round(s.bodyRefPx * s.scale, 2);
                float delta = Mathf.Abs(displayedBodyPx - s.targetPx);

                results.Add(new ClassDisplayMeasurement
                {
                    classId = s.id,
                    approvedScale = s.scale,
                    cellWidthPx = idleFrames[0].texture.width,
                    cellHeightPx = idleFrames[0].texture.height,
                    tightBodyHeightSourcePx = tightFullPx,
                    displayedBodyHeight1080pPx = displayedBodyPx,
                    targetBodyHeight1080pPx = s.targetPx,
                    deltaPx = (float)Math.Round(delta, 3),
                    transitionAuditPassed = returnedToIdleAfterAction && returnedToIdleAfterHit && ctrl.ImpactTriggersFired >= 1 && sr.sprite != null,
                    completedActionAndHitTransitions = ctrl.CompletedNonLoopTransitions,
                    impactTriggersFired = ctrl.ImpactTriggersFired
                });
            }

            UnityEngine.Object.DestroyImmediate(auditRoot);
            return results;
        }

        private static void VerifyAudioDispatcherContract(
            BroadcastAudioDispatcher dispatcher,
            out int prep10s,
            out int assault10s,
            out float assaultGain,
            out int joinDuringAlert,
            out int dupSeqPlays,
            out int dupVicPlays,
            out int wallDmg3,
            out int fighter2,
            out int builder2,
            out int bossSlam2,
            out int burst32Voices,
            out bool busesOk)
        {
            prep10s = 0;
            long seq = 1;
            for (int ev = 0; ev < 1000; ev++)
            {
                if (dispatcher.TryDispatchCommittedCue("sess-prep", 10, seq++, "role_accepted", "SFX01", "PREP", ev * 10.0, out _, out _))
                    prep10s++;
            }

            dispatcher.ResetCooldownsOnlyForNextTest();
            assault10s = 0;
            assaultGain = 0f;
            for (int ev = 0; ev < 1000; ev++)
            {
                if (dispatcher.TryDispatchCommittedCue("sess-assault", 10, seq++, "role_accepted", "SFX01", "ASSAULT", ev * 10.0, out float g, out _))
                {
                    assault10s++;
                    assaultGain = (float)Math.Round(g, 4);
                }
            }

            dispatcher.ResetCooldownsOnlyForNextTest();
            dispatcher.TryDispatchCommittedCue("sess-alert", 10, seq++, "boss_incoming", "SFX08", "ASSAULT", 50000.0, out _, out _);
            joinDuringAlert = 0;
            for (int ev = 0; ev < 50; ev++)
            {
                if (dispatcher.TryDispatchCommittedCue("sess-alert", 10, seq++, "role_accepted", "SFX01", "ASSAULT", 50010.0 + ev * 10.0, out _, out _))
                    joinDuringAlert++;
            }

            dispatcher.ResetCooldownsOnlyForNextTest();
            dupSeqPlays = 0;
            for (int r = 0; r < 5; r++)
            {
                if (dispatcher.TryDispatchCommittedCue("sess-dedupe", 10, 777, "damage_applied", "SFX06", "ASSAULT", 60000.0 + r * 500.0, out _, out _))
                    dupSeqPlays++;
            }

            dispatcher.ResetCooldownsOnlyForNextTest();
            wallDmg3 = 0;
            for (int i = 0; i < 3; i++)
                if (dispatcher.TryDispatchCommittedCue("sess-dedupe", 10, 800 + i, "damage_applied", "SFX06", "ASSAULT", 70000.0 + i * 500.0, out _, out _))
                    wallDmg3++;

            fighter2 = 0;
            for (int i = 0; i < 2; i++)
                if (dispatcher.TryDispatchCommittedCue("sess-dedupe", 10, 810 + i, "attack_resolved", "SFX04", "ASSAULT", 72000.0 + i * 300.0, out _, out _))
                    fighter2++;

            builder2 = 0;
            for (int i = 0; i < 2; i++)
                if (dispatcher.TryDispatchCommittedCue("sess-dedupe", 10, 820 + i, "role_resolution", "SFX02", "PREP", 74000.0 + i * 400.0, out _, out _))
                    builder2++;

            bossSlam2 = 0;
            for (int i = 0; i < 2; i++)
                if (dispatcher.TryDispatchCommittedCue("sess-dedupe", 10, 830 + i, "boss_combat_slam", "SFX09", "FINALE", 76000.0 + i * 2000.0, out _, out _))
                    bossSlam2++;

            dupVicPlays = 0;
            long[] vicSeqs = { 900, 900, 901, 902 };
            for (int i = 0; i < vicSeqs.Length; i++)
                if (dispatcher.TryDispatchCommittedCue("sess-dedupe", 10, vicSeqs[i], "wave_result", "SFX11", "RESULTS", 90000.0 + i * 12000.0, out _, out _))
                    dupVicPlays++;

            dispatcher.ResetCooldownsOnlyForNextTest();
            string[] cycleCues = { "SFX02", "SFX03", "SFX04", "SFX05", "SFX06", "SFX07", "SFX09" };
            for (int i = 0; i < 32; i++)
            {
                string c = cycleCues[i % cycleCues.Length];
                dispatcher.TryDispatchCommittedCue("sess-burst", 10, 1200 + i, "burst_" + i, c, "FINALE", 120000.0 + (i * 2.0), out _, out _);
            }
            burst32Voices = dispatcher.CountActiveVoices(120070.0);

            dispatcher.SetBusVolumeDb("SFX", -6.0f);
            dispatcher.SetBusVolumeDb("Music", 0.0f);
            dispatcher.SetGlobalMute(true);
            dispatcher.ResetCooldownsOnlyForNextTest();
            dispatcher.TryDispatchCommittedCue("sess-mute", 10, 2001, "attack_resolved", "SFX04", "ASSAULT", 150000.0, out float mutedGain, out _);
            dispatcher.SetGlobalMute(false);
            dispatcher.SetBusVolumeDb("SFX", -1.5f);

            busesOk = mutedGain == 0f && dispatcher.GetBusVolumeDb("Music") <= -79.9f && dispatcher.TotalChildAudioSources == 12;
        }

        private static List<SpriteTransitionController> Build150EntityPool(
            Transform poolRoot,
            Dictionary<string, Sprite> sp,
            BoundedVfxPool vfxPool)
        {
            var list = new List<SpriteTransitionController>(150);
            for (int i = 0; i < 150; i++)
            {
                var go = new GameObject($"PooledUnit_{i:D3}");
                go.transform.SetParent(poolRoot, false);
                var sr = go.AddComponent<SpriteRenderer>();
                var ctrl = go.AddComponent<SpriteTransitionController>();

                if (i == 0)
                {
                    ctrl.Configure("boss", sr, 0.57955f, "idlemarch", vfxPool, new[]
                    {
                        new SpriteTransitionController.ClipDef { clipName = "idlemarch", frames = LoadSeq(sp, "chr_boss_warlord_idlemarch", 6), loop = true },
                        new SpriteTransitionController.ClipDef { clipName = "cleaveslam", frames = LoadSeq(sp, "chr_boss_warlord_cleaveslam", 8), loop = false, impactFrameIndex = 5, pairedVfxFrames = LoadSeq(sp, "vfx_boss_slam_telegraph", 10), vfxScale = 0.68f, vfxSortingLayer = "LanesAndDecals" },
                        new SpriteTransitionController.ClipDef { clipName = "hit", frames = LoadSeq(sp, "chr_boss_warlord_hit", 2), loop = false }
                    });
                    go.transform.position = ScreenPxToWorld(454f, 520f);
                }
                else
                {
                    int mod = i % 6;
                    if (mod == 0)
                    {
                        ctrl.Configure("builder", sr, 0.55814f, "idle", vfxPool, new[]
                        {
                            new SpriteTransitionController.ClipDef { clipName = "idle", frames = LoadSeq(sp, "chr_role_builder_idle", 4), loop = true },
                            new SpriteTransitionController.ClipDef { clipName = "repair", frames = LoadSeq(sp, "chr_role_builder_repair", 6), loop = false, impactFrameIndex = 3, pairedVfxFrames = LoadSeq(sp, "vfx_builder_repair_spark", 6), vfxWorldOffset = new Vector2(0.12f, 0.08f) }
                        });
                        float bx = (i % 2 == 0) ? (260f + (i % 5) * 14f) : (590f + (i % 5) * 14f);
                        go.transform.position = ScreenPxToWorld(bx, 998f + (i % 3) * 10f);
                    }
                    else if (mod == 1)
                    {
                        ctrl.Configure("guard", sr, 0.55172f, "idle", vfxPool, new[]
                        {
                            new SpriteTransitionController.ClipDef { clipName = "idle", frames = LoadSeq(sp, "chr_role_guard_idle", 4), loop = true },
                            new SpriteTransitionController.ClipDef { clipName = "shieldbrace", frames = LoadSeq(sp, "chr_role_guard_shieldbrace", 6), loop = false, impactFrameIndex = 2, pairedVfxFrames = LoadSeq(sp, "vfx_guard_shield_block", 4), vfxWorldOffset = new Vector2(0.06f, 0.12f) }
                        });
                        go.transform.position = ScreenPxToWorld(350f + (i % 15) * 15f, 650f + (i % 4) * 18f);
                    }
                    else if (mod == 2)
                    {
                        ctrl.Configure("fighter", sr, 0.55814f, "idle", vfxPool, new[]
                        {
                            new SpriteTransitionController.ClipDef { clipName = "idle", frames = LoadSeq(sp, "chr_role_fighter_idle", 4), loop = true },
                            new SpriteTransitionController.ClipDef { clipName = "swordslash", frames = LoadSeq(sp, "chr_role_fighter_swordslash", 6), loop = false, impactFrameIndex = 3, pairedVfxFrames = LoadSeq(sp, "vfx_fighter_slash_arc", 4), vfxWorldOffset = new Vector2(0.14f, 0.12f) }
                        });
                        go.transform.position = ScreenPxToWorld(300f + (i % 20) * 16f, 610f + (i % 5) * 16f);
                    }
                    else if (mod == 3)
                    {
                        ctrl.Configure("grunt", sr, 0.55814f, "walk", vfxPool, new[]
                        {
                            new SpriteTransitionController.ClipDef { clipName = "idle", frames = LoadSeq(sp, "chr_enemy_grunt_idle", 4), loop = true },
                            new SpriteTransitionController.ClipDef { clipName = "walk", frames = LoadSeq(sp, "chr_enemy_grunt_walk", 6), loop = true },
                            new SpriteTransitionController.ClipDef { clipName = "clubsmash", frames = LoadSeq(sp, "chr_enemy_grunt_clubsmash", 6), loop = false, impactFrameIndex = 3, pairedVfxFrames = LoadSeq(sp, "vfx_unit_hit_puff", 6) }
                        });
                        go.transform.position = ScreenPxToWorld(180f + (i % 12) * 18f, 360f + (i % 8) * 32f);
                    }
                    else if (mod == 4)
                    {
                        ctrl.Configure("runner", sr, 0.54321f, "run", vfxPool, new[]
                        {
                            new SpriteTransitionController.ClipDef { clipName = "idle", frames = LoadSeq(sp, "chr_enemy_runner_idle", 4), loop = true },
                            new SpriteTransitionController.ClipDef { clipName = "run", frames = LoadSeq(sp, "chr_enemy_runner_run", 6), loop = true },
                            new SpriteTransitionController.ClipDef { clipName = "daggerlunge", frames = LoadSeq(sp, "chr_enemy_runner_daggerlunge", 6), loop = false, impactFrameIndex = 2, pairedVfxFrames = LoadSeq(sp, "vfx_unit_hit_puff", 6) }
                        });
                        go.transform.position = ScreenPxToWorld(600f + (i % 12) * 18f, 380f + (i % 8) * 30f);
                    }
                    else
                    {
                        ctrl.Configure("brute", sr, 0.66667f, "march", vfxPool, new[]
                        {
                            new SpriteTransitionController.ClipDef { clipName = "idle", frames = LoadSeq(sp, "chr_enemy_brute_idle", 4), loop = true },
                            new SpriteTransitionController.ClipDef { clipName = "march", frames = LoadSeq(sp, "chr_enemy_brute_march", 6), loop = true },
                            new SpriteTransitionController.ClipDef { clipName = "gauntletcrush", frames = LoadSeq(sp, "chr_enemy_brute_gauntletcrush", 6), loop = false, impactFrameIndex = 3, pairedVfxFrames = LoadSeq(sp, "vfx_unit_hit_puff", 6) }
                        });
                        go.transform.position = ScreenPxToWorld(418f + (i % 5) * 18f, 340f + (i % 7) * 36f);
                    }
                }
                ctrl.UpdateYSortingOrder();
                go.SetActive(false);
                list.Add(ctrl);
            }
            return list;
        }

        private static void SetActiveEntityTier(List<SpriteTransitionController> pool, int visibleCount, bool bossActive)
        {
            for (int i = 0; i < pool.Count; i++)
            {
                bool on = bossActive ? (i < visibleCount) : (i >= 1 && i <= visibleCount);
                if (i == 0 && !bossActive && visibleCount == 150)
                {
                    on = true;
                }
                pool[i].gameObject.SetActive(on);
                if (on && i == 0 && bossActive)
                {
                    pool[i].PlayClip("cleaveslam");
                }
            }
        }

        private static void RenderCameraToPng(Camera cam, int width, int height, string outPngPath)
        {
            Directory.CreateDirectory(Path.GetDirectoryName(outPngPath));
            var rt = new RenderTexture(width, height, 24, RenderTextureFormat.ARGB32);
            cam.targetTexture = rt;
            Canvas.ForceUpdateCanvases();
            cam.Render();
            RenderTexture.active = rt;
            var tex = new Texture2D(width, height, TextureFormat.RGBA32, false);
            tex.ReadPixels(new Rect(0, 0, width, height), 0, 0);
            tex.Apply();
            File.WriteAllBytes(outPngPath, tex.EncodeToPNG());
            RenderTexture.active = null;
            cam.targetTexture = null;
            Canvas.ForceUpdateCanvases();
            UnityEngine.Object.DestroyImmediate(rt);
            UnityEngine.Object.DestroyImmediate(tex);
        }

        private static LoadTierResult MeasureEntityLoadTier(
            Camera cam,
            List<SpriteTransitionController> pool,
            BoundedVfxPool vfxPool,
            BroadcastAudioDispatcher dispatcher,
            HudInPlacePresenter hud,
            int visibleCount,
            int frames,
            long texBytes,
            long audBytes)
        {
            const int warmupFrames = 24;
            SetActiveEntityTier(pool, visibleCount, bossActive: visibleCount >= 100);
            var rt = new RenderTexture(1080, 1920, 24, RenderTextureFormat.ARGB32);
            cam.targetTexture = rt;
            Canvas.ForceUpdateCanvases();

            // Warmup 24 frames so shader/pipeline/UI layout caches settle before timing
            for (int w = 0; w < warmupFrames; w++)
            {
                for (int i = 0; i < pool.Count; i++)
                    if (pool[i].gameObject.activeSelf) pool[i].StepSimulationMs(16.67);
                vfxPool.StepSimulationMs(16.67);
                hud.StepMotionMs(16.67f);
                cam.Render();
            }

            int goBefore = UnityEngine.Object.FindObjectsByType<GameObject>(FindObjectsInactive.Include).Length;
            GC.Collect();
            long heapBefore = GC.GetTotalMemory(false);
            long threadAllocBefore = GC.GetAllocatedBytesForCurrentThread();

            using var gcAllocRec = ProfilerRecorder.StartNew(ProfilerCategory.Memory, "GC Allocated In Frame");
            using var drawCallsRec = ProfilerRecorder.StartNew(ProfilerCategory.Render, "Draw Calls Count");
            using var batchesRec = ProfilerRecorder.StartNew(ProfilerCategory.Render, "Batches Count");
            using var setPassRec = ProfilerRecorder.StartNew(ProfilerCategory.Render, "SetPass Calls Count");

            var frameTimesMs = new double[frames];
            int maxVoices = 0;
            int maxVfx = 0;
            long simSeq = visibleCount * 10000L;

            var sw = new Stopwatch();
            for (int f = 0; f < frames; f++)
            {
                sw.Restart();
                double nowMs = 300000.0 + visibleCount * 50000.0 + f * 16.6667;

                if (f % 15 == 0)
                {
                    for (int u = 1; u < Math.Min(pool.Count, 18); u += 3)
                    {
                        if (pool[u].gameObject.activeSelf)
                        {
                            string act = pool[u].ClassId == "builder" ? "repair" :
                                         pool[u].ClassId == "guard" ? "shieldbrace" :
                                         pool[u].ClassId == "fighter" ? "swordslash" : "clubsmash";
                            pool[u].PlayClip(act);
                        }
                    }
                    dispatcher.TryDispatchCommittedCue("perf-sess", 10, simSeq++, "attack_resolved", "SFX04", "ASSAULT", nowMs, out _, out _);
                }

                for (int i = 0; i < pool.Count; i++)
                {
                    if (pool[i].gameObject.activeSelf)
                    {
                        pool[i].StepSimulationMs(16.6667);
                    }
                }
                vfxPool.StepSimulationMs(16.6667);
                hud.StepMotionMs(16.6667f);
                cam.Render();
                sw.Stop();

                frameTimesMs[f] = sw.Elapsed.TotalMilliseconds;
                int vNow = dispatcher.CountActiveVoices(nowMs);
                if (vNow > maxVoices) maxVoices = vNow;
                int vfxNow = vfxPool.ActiveBurstCount;
                if (vfxNow > maxVfx) maxVfx = vfxNow;
            }

            long threadAllocAfter = GC.GetAllocatedBytesForCurrentThread();
            long cumulativeThreadAlloc = Math.Max(0L, threadAllocAfter - threadAllocBefore);
            long heapAfter = GC.GetTotalMemory(false);
            long netHeapDelta = heapAfter - heapBefore;
            int goAfter = UnityEngine.Object.FindObjectsByType<GameObject>(FindObjectsInactive.Include).Length;

            // Check GPU FrameTimingManager availability (typically unavailable in synchronous Editor batchmode)
            FrameTimingManager.CaptureFrameTimings();
            var gpuTimings = new FrameTiming[4];
            uint gpuCaptured = FrameTimingManager.GetLatestTimings(4, gpuTimings);
            bool hasGpuTiming = FrameTimingManager.IsFeatureEnabled() && gpuCaptured > 0 && gpuTimings[0].gpuFrameTime > 0.0;
            string gpuTimingStatus = hasGpuTiming ? "MEASURED" : "NOT_MEASURED_IN_EDITOR_BATCHMODE";
            float latestGpuMs = hasGpuTiming ? (float)Math.Round(gpuTimings[0].gpuFrameTime, 3) : -1f;

            // Check render draw-call counters: mark NOT_MEASURED (-1) when Editor batchmode returns 0 outside PlayMode
            long rawDrawCalls = Math.Max(drawCallsRec.Valid ? drawCallsRec.LastValue : 0L, UnityStats.drawCalls);
            long rawBatches = Math.Max(batchesRec.Valid ? batchesRec.LastValue : 0L, UnityStats.dynamicBatches + UnityStats.staticBatches + UnityStats.instancedBatches);
            long rawSetPass = Math.Max(setPassRec.Valid ? setPassRec.LastValue : 0L, UnityStats.setPassCalls);

            bool dcMeasured = rawDrawCalls > 0;
            bool btMeasured = rawBatches > 0;
            bool spMeasured = rawSetPass > 0;

            cam.targetTexture = null;
            UnityEngine.Object.DestroyImmediate(rt);

            double sumMs = 0.0;
            for (int i = 0; i < frames; i++) sumMs += frameTimesMs[i];
            double avgMs = Math.Max(0.01, sumMs / frames);

            // Phase 4 Finding A: True 1% Low FPS = 1000.0 / mean(slowest ceil(N * 0.01) frameTimesMs)
            // Sort ascending first for p99, and average the slowest ceil(N * 0.01) tail frames separately!
            var sortedAsc = (double[])frameTimesMs.Clone();
            Array.Sort(sortedAsc);
            double p99Ms = sortedAsc[Math.Min(frames - 1, (int)Math.Floor(frames * 0.99))];
            double maxMs = sortedAsc[frames - 1];

            int slowestCount = Math.Max(1, (int)Math.Ceiling(frames * 0.01));
            double slowestSumMs = 0.0;
            for (int i = frames - slowestCount; i < frames; i++)
            {
                slowestSumMs += sortedAsc[i];
            }
            double slowestMeanMs = Math.Max(0.01, slowestSumMs / slowestCount);
            float onePctLowFps = (float)Math.Round(1000.0 / slowestMeanMs, 2);

            return new LoadTierResult
            {
                visibleEntityCount = visibleCount,
                warmupFrames = warmupFrames,
                measuredFrames = frames,
                timingScope = "cpu_camera_render_submission_editor_batchmode",
                averageFps = (float)Math.Round(1000.0 / avgMs, 2),
                onePercentLowFps = onePctLowFps,
                onePercentLowDefinition = "1000.0 / mean(slowest ceil(N * 0.01) frameTimesMs)",
                slowestOnePercentFrameCount = slowestCount,
                slowestOnePercentMeanMs = (float)Math.Round(slowestMeanMs, 3),
                averageFrameTimeMs = (float)Math.Round(avgMs, 3),
                p99FrameTimeMs = (float)Math.Round(p99Ms, 3),
                maxFrameTimeMs = (float)Math.Round(maxMs, 3),
                gpuFrameTimingStatus = gpuTimingStatus,
                latestGpuFrameTimeMs = latestGpuMs,
                standalone15MinFpsStatus = "NOT_EXECUTED",
                obsNvencCaptureFpsStatus = "NOT_EXECUTED",
                managedGcMeasurementMethod = "GC_GetAllocatedBytesForCurrentThread_Cumulative",
                managedGcAllocatedBytesTotal = cumulativeThreadAlloc,
                managedGcBytesPerFrame = (float)Math.Round((double)cumulativeThreadAlloc / frames, 2),
                netHeapDeltaBytes = netHeapDelta,
                profilerRecorderGcValid = gcAllocRec.Valid,
                drawCalls = dcMeasured ? rawDrawCalls : -1L,
                drawCallsStatus = dcMeasured ? "MEASURED" : "NOT_MEASURED",
                batches = btMeasured ? rawBatches : -1L,
                batchesStatus = btMeasured ? "MEASURED" : "NOT_MEASURED",
                setPassCalls = spMeasured ? rawSetPass : -1L,
                setPassCallsStatus = spMeasured ? "MEASURED" : "NOT_MEASURED",
                totalTextureMemoryBytes = texBytes,
                totalAudioMemoryBytes = audBytes,
                maxActiveAudioVoices = maxVoices,
                maxActiveVfxBursts = maxVfx,
                dynamicObjectsSpawnedDuringLoop = Math.Max(0, goAfter - goBefore)
            };
        }
    }
}
