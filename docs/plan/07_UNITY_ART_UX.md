# UNITY 9:16, ART, INTERACTION

## Locked visual product
Unity 6000.6.7f1, 2D pseudo-isometric village (not 3D). Vertical 1080x1920 reference, target 30 FPS minimum on reference capture host; fixed orthographic camera, no manual pan. Render <=150 representative pooled entities (NOT 10k player objects). Default quota 40 builders, 40 guards, 40 fighters, 30 enemies; quotas adapt to real on-screen threat, <=150 total, boss always visible. Unity receives only server truth; client visual pools and interpolated HP cannot change combat. Prefer Unity Built-in 2D pipeline, URP only after approved measured ADR.
World: centered castle+wall occupying central 44% width; WEST/CENTER/EAST approach lanes visibly distinct; enemy silhouettes at edges; builders repair, guards defend and fighters battle. UI responsive anchored Canvas, never baked essential text.

## Responsive safe areas
Reference 1080x1920, CanvasScaler matchWidthOrHeight=0.5. Top HUD y=5..14%, castle/world y=20..74%, commands x=8..75% y=76..86%. Reserve right x>82% and bottom y>88% for TikTok chrome until actual live OBS overlay verified; platform overlays vary by region/client and require screenshots.
At 360x640 equivalent, commands >=16px, timer and castle HP >=14px, secondary text>=12px. Text contrast>=4.5:1; white/light outlined text on dark scrim; no reliance on color alone. Max 12 simultaneous player alias labels, truncated to <=12 visible characters plus ellipsis.

## Mandatory visual states
ATTRACT, WAITING_FOR_VIEWER, PREP, PREP_60_WARN, PREP_15_WARN, ASSAULT, FINALE, BOSS, RESULTS_WIN, RESULTS_LOSS, VOTE_OPEN, SOURCE_RECONNECTING, PAUSED, STORAGE_UNAVAILABLE, DEGRADED_DROPS, MAINTENANCE. Mock/replay watermark always displayed outside real authorized live mode. Source disconnect and unsaved state are not disguised as valid gameplay.
Healthy valid join must update aggregate count on next stream publication (<=200ms publication cadence, backend SLO separate); show rotating spotlight of filtered alias if permitted, not a promise every avatar appears. Repeated identical comment silent; changing role affects next 20s tick. Vote shows A/B/C and integer counts/close timer; result lasts 20s. No dynamic camera movement obscures the village.

## Animation and content manifest (mandatory M06)
ART001 player builder yellow apron/hammer, guard blue shield/cape, fighter red weapon/helmet; easily distinguished in grayscale and 48px preview; 3/4 facing center.
ART002 grunt round green, runner slim purple, brute square orange, boss black-violet oversized high-contrast silhouette.
ART003 wall 5 visual damage/tier states mapped across 20 tiers; townhall, one watchtower, 6 houses, 3 decorations.
ART004 3 approach lanes, 6 environmental tile variants, 4 lighting palettes (visual only), atlas and background.
ART005 projectile, hit, repair spark, shield effect, boss telegraph, no more than 20 simultaneous VFX bursts.
UI001 command icons, resource icons x3, wall, hall, boss, danger, timer, vote, source health; Vietnamese default and English localization.
SFX001 12 original/cleared short effects, one original ambient loop, two short music loops with rights recorded; music mute by default until stream rights confirmed.
Every player role: idle4, walk6, action6, hit2 frames at 12fps; enemies idle4, walk6, attack6, hit2; boss idle6, attack8, hit2. Source frame 128x128 PNG RGBA with pivot=(0.5,0.08), consistent 3/4 direction; Unity atlas <=2048x2048, filter Point for pixel-silhouette art. Atlas packing and pooling must be profiled in capture build.
Each asset needs exact source, provenance, right-to-use, dimensions, animation import preset, atlas, screenshot and license metadata. Placeholder assets M01 only; M06 PASS requires production original/licensed visual package, no third-party copyrighted characters.
Animations and effects not tied to ticking game logic. Never manufacture fictitious active viewer indicators, names or history.

## Accessibility and comprehension
Reduced-motion disables camera shake/strobe. No flashes>3Hz. At 360x640 with actual chat overlay, five uncoached viewers watch 3 seconds with audio off; >=4 correctly identify threat and one comment command. Capture two muted 15min sessions at 720x1280 and 1080x1920 plus 1440x2560 crop; document real TikTok overlays. Test source disconnection and empty-room states as first-class UX.
This is a test requirement, not a claim the research already happened.

## Moment-to-moment broadcast design (hard user-experience contract)
- All phases must provide a meaningful, truthful, readable change at least every 30s: countdown/state updates every second; observable builder progress at t=20/40/60/80/100/120/140/160; even-wave poll [80,120); enemy warning at t=130/145; assault begins t=160; combat impacts t=180,200,220,240; boss warning t=245 and boss spawn t=260 (every 5th wave); final hit/result at t=280; results+next-wave CTA [280,300). At 10Hz these seconds map to ticks 600,800,...,3000 in 03_SIMULATION.md, so the presentation MUST use tick 2600 for boss spawn and tick 2800 for the final combat/result and must never display a boss or a result one boundary early. On odd waves without poll, show defensible village-building objective and role distribution at t=80 rather than inventing a choice.
- The first 3 seconds of an unfamiliar clip must show (i) castle/wall condition (ii) imminent threat/time (iii) a single plain-language command: `Bình luận XÂY / THỦ / ĐÁNH`. Do not show multiple competing calls to action. Current role counts must be adjacent to role labels and update from server after registration; visual activity must not imply a contribution that server did not accept.
- On successful first join, display a tiny 2-second feedback toast or aggregate count change without obscuring the castle. Spotlight selection must be fair over eligible non-hidden actors: stable round-robin by joined sequence, maximum one 2-second spotlight per 5s, 12 simultaneous alias labels max. In a stream with 10,000 commenters no guarantee of individual recognition; visible aggregate group statistics are the fallback, not fabricated personal acknowledgements.
- Right side and bottom safe overlays are tuned in OBS by operator for account/device, not assumed to be universal. An adjustable inspector panel stores safe-rectangle margins, persists settings, and cannot hide main attack command or core HP even with platform chat enabled.
- Every phase must have a comprehensible non-video-audio version. Do not use anxiety-inducing flashing, countdown sounds above baseline, graphic injury or shame callouts for nonparticipants. No forced chat spam or paid boost prompts. Show success AND failure honestly; every failure offers a fair recovery without destruction of viewer cosmetic progress.
- Boss anticipation: `BOSS_INCOMING` shown even if ordinary enemies were already defeated before t2600. Victory flair only after server authoritative `RESULTS_WIN` at t2800; premature CLEAR forbidden. Show militia as `Tự động hỗ trợ` with no viewer nickname.
- Scenes required: 00_Bootstrap, 01_BroadcastWorld, 02_OfflineRecoveryPreview. Only BroadcastWorld is streamed; operator UI is separate protected web application. The Windows standalone build runs 1080x1920 borderless/windowed, OBS Game Capture/Window Capture preset documented. Do not assume TikTok LIVE Studio connector exists.
- Art production gate: assets must be complete, rights-cleared and legible after H.264 compression and scaled to 360x640; avoid generic featureless colored boxes as final characters. All animated roles and boss remain distinct at 48px high and across grayscale. Minimum 15min visual regression capture with realistic mock 10/100/1000/10000 audience trajectories and overlay.

## Unity 6.6.7f1 known-issue mitigation
Unity official release notes for 6000.6.7f1 report a known issue where repeated SceneManager.LoadSceneAsync transitions can hang a Player. Keep 01_BroadcastWorld loaded for the whole broadcast; display source loss, pause and recovery as Canvas states, not repeated async scene switches. Bootstrap/recovery scenes may load once with verified testing. Official source: https://unity.com/releases/editor/whats-new/6000.6.7f1 . QA-VIS-02 tests 2-hour continuous mock stream with repeated source disconnect/reconnect, no scene-reload hang.

