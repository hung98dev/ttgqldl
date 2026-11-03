# BUILD / OPERATE / RECOVER — NORMATIVE RUNBOOK

## Boot
M01 provides pinned tools with scripts doctor, dev-up, migrate-up, backend-test, integration-test, unity-test, ui-build, load-test, backup, restore-verify and dev-down, with Windows PowerShell and Linux shell wrappers. README must document ports, secrets via .env.example (never real .env), exact versions and clean machine setup.
Docker Compose binds PostgreSQL to 127.0.0.1 by default; admin panel local only without hardened reverse proxy; do not expose database/admin publicly.

## Environment
dev = mock adapter with persistent dev database and explicit MOCK watermark. stage = replay/live approval sandbox with de-identified fixtures; prod = only approved live adapter and fully audited access. Stage/production fail startup when required creds or schema migration are missing, never quietly downgrade to mock while claiming live.

## Operational readiness
Health liveness, readiness DB+single-writer ownership, queue capacity, viewer active unique count, accepted/dropped events, state publishing lag, last snapshot, command p95, WS clients, frame time/GC, provider health. Dashboard alerts on missing events while platform expected online, backlog>75%, state lag>1s p95 sustained, snapshot failure, DB storage >80%, unauthorized access anomalies.
Streamer controls: pause, resume, stop, manually hide alias, mute sounds, force full sync, toggle low-VFX. Every privileged operation audited.

## Persistence & disaster recovery
Snapshot every 60 simulated seconds and at end of each wave. Backups daily + PostgreSQL WAL/PITR strategy sized to deployment; target RPO <=5min, RTO <=30min measured via rehearsal. Restore in clean environment before production; checksum replay. Crash midwave: resume from last committed sequence/tick and replay in order without double contribution. No silent wipe.
Releases use versioned image digest and rollback runbook. Before migration backup, test forward migration and rollback/restore on stage; deploy backend+client protocol compat within specified supported schema versions; if incompatible abort deployment.

## Incident scenarios
Provider disconnect: prominent status, freeze accepting viewer actions; configurable simulation pause default true; maintain world state.
Queue overflow: drop non-command non-actionable events first, then explicitly record lost commands if capacity exceeded, display degraded status.
PG outage: stop authoritative tick after safe checkpoint, set stream unavailable, retry with backoff, never send fabricated success.
Unity crash: reconnect -> new full snapshot and replay only noncritical visuals.
Abuse: remove display aliases, terminate participant session if permitted, mute event source, emergency broadcast pause.
Each incident includes detection, owner, containment, evidence and recovery retrospective.

## Go-live checklist
All matrix rows PASS; real provider access authorized and proven; real 9:16 capture tested, backups restore verified, load evidence archived, moderator staffed, privacy notice and viewer opt-outs ready, security review signed, rollback verified.
