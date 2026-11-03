# UNITY PORTRAIT / CONTENT / INTERACTION CONTRACT

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
