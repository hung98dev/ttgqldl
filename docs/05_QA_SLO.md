# QUALITY, TESTS AND RELEASE BLOCKERS

## Required tests
Unit: commands (Vietnamese normalization, unicode, exact grammar), vote, windows, combat, inventory caps, balance math, state machine, reward caps.
Property/fuzz: no negative HP/resources, max one action per actor/window, caps, schema parsing, no panics, deterministic outcome.
Integration: migrations/rollback, transactional receipt dedupe, event sequence, snapshot/replay checksum, WebSocket reconnect, identity rotation, RBAC and audit.
Soak: 10k logical players, 1k messages/s sustained 10 minutes, bursts 5k/s 10 seconds; report actual input, accepted, dropped, CPU, RAM, DB query p95, internal accepted-to-visible p95/p99, WebSocket resync and FPS. Profile bottleneck; no unbounded queues.
Failure: kill -9 backend mid wave, cut PG network, reconnect provider, duplicate deliveries, queue full, restart operator, restore DB backup into clean environment, break Unity socket, stale schema mismatch.
UI: record 720x1280, 1080x1920, 1440x2560, phone crop and TikTok overlays; watch on 360x640; 5-person 3-second comprehension test; no essential HUD clipping.
Security: OWASP ASVS-inspired verification, dependencies CVE scan, secret scan, moderation abuse, CSRF, authz, no unsafe source integration.

## Acceptance IDs
AC001 first valid role creates actor once.
AC002 100 same-window commands yield 1 role contribution.
AC003 XAY then THU same window yields THU only.
AC004 duplicated source event cannot alter state twice.
AC005 3 lanes and types resolve reproducibly across 100 seeds.
AC006 pause freezes server wave clock.
AC007 restart from snapshot/events yields same hash.
AC008 defeat retains XP/progression and grants recovery.
AC009 10k logical players without 10k visual objects.
AC010 over-capacity queue reports accurate drops.
AC011 disconnected provider unmistakably visible and no fabricated input.
AC012 vote one effective ballot per actor.
AC013 no gift event affects combat/XP.
AC014 admin endpoints reject unauthenticated/unauthorized clients.
AC015 export, erase and hide alias work.
AC016 30 FPS benchmark on recorded reference hardware (not unqualified claim).
AC017 OBS essential HUD fits with real platform overlays.
AC018 migration & clean restore PASS.
AC019 source rights and TikTok policy evidence attached; else TikTok release BLOCKED.
AC020 all released builds pinned and reproducible.

## Evidence protocol
Every acceptance row records commit SHA, command, environment/versions/hardware, dataset, UTC run time, output artifact path/link, PASS/FAIL/BLOCKED and reviewer. Mock tests cannot close external authorization gate. CI must include go test ./..., race tests, fuzz smoke, integration PG18, Unity editmode/playmode and security scans. A claim of DONE without evidence = FAIL.
