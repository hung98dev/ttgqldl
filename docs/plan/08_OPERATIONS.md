# DEVELOPMENT, CI, DEPLOYMENT AND INCIDENTS

## Runtime reference and toolchains
Windows 11 host for Unity OBS with >=6 physical CPU cores, >=16 GiB RAM and discrete GPU approximately RTX3060-class as reference only; Linux amd64 backend reference dedicated 8vCPU, 16GiB RAM, NVMe SSD and PostgreSQL 18.6 over private loopback/network. These are NOT verified current user's equipment; tests record real CPU/GPU/storage before claiming numbers. Go 1.27.3, Unity 6000.6.7f1, Postgres18.6. Pin exact external libs, node LTS, Docker digests and Unity packages during M01; no moving latest tags. Windows PowerShell and Linux Bash entrypoints run from repo root.
Single-main workflow: pre-commit quality gates, then commit main, then verify GitHub Actions and revert/repair. No auto-merge, PRs or branches unless explicitly requested.

## Required configuration register
TTG_ENV=dev|stage|prod (default dev)
TTG_SOURCE=mock|replay|authorized_live (default mock; live fails closed until permission verification)
TTG_HTTP_ADDR=127.0.0.1:8080 (default local)
TTG_DATABASE_URL: secret, mandatory all modes
TTG_HMAC_KEY_VERSION: nonempty and TTG_HMAC_SECRET: secret mandatory
TTG_ADMIN_PASSWORD_HASH: Argon2id mandatory on stage/prod, seed dev creds only in local environment
TTG_STREAM_SIGNING_SECRET: random >=32 bytes, mandatory stage/prod
TTG_SOURCE_CREDENTIAL_REF: mandatory in authorized_live
TTG_LOG_LEVEL=info; TTG_METRICS_ADDR=127.0.0.1:9090; TTG_MAX_QUEUE=20000; TTG_STREAM_RATE_HZ=5; TTG_SNAPSHOT_TICKS=600
Invalid enum, weak secret, missing key or migration mismatch => fail startup. .env.example placeholders ONLY, real secrets gitignored and scanned.

## Mandatory commands/scripts
tooling/doctor, up, down, migrate-up, migrate-down, test-unit, test-race, test-integration, test-schema, test-golden, test-load, test-unity, test-operator, test-security, backup, restore-verify, capture-obs; both .ps1 and .sh wrappers or portable Go CLI where possible.
doctor prints versions and refuses mismatched stable pins; up starts PostgreSQL/backend/operator and mock source. down preserves persistent state unless --wipe-dev-data explicit with interactive warning. All scripts write JSON evidence UTC timestamp, SHA, host, versions, command, exit status. If a test cannot run due to missing device/licensing, mark BLOCKED not PASS.

## Deployment
Docker Compose local single-host baseline; Postgres binds 127.0.0.1/private only, public API behind TLS reverse proxy with auth and fixed limits; do not expose admin or DB on public IP. Production image pinned by digest, semver, rollback. Source must be explicit and accepted at startup; never downgrade real-source sessions silently to mock. Stage exercises mock/replay, real integration stage after authorization. Runtime CPU/memory/pid limits explicit in Compose and bench results.

## Observability/alarms
Counters input_received/accepted/duplicate/ignored/cooldown/invalid/dropped, actual distinct player registrations, source health, writer epoch, committed tick, queue depth, websocket sessions, stream lag, DB tx retries and errors, Unity fps/audio peak. Histogram backend receive-to-commit/publish p50/p95/p99. External TikTok-to-backend latency is separately UNKNOWN until measurable.
Alert: queue >75% for30s, ANY dropped valid game action, internal p95>1s sustained5min, missing checkpoint >90 simulated seconds, DB disk>80%, source unexpectedly offline >5s, unauthorized operator attempt spike. Operator must see prominent warning not falsified healthy status.

## Availability/recovery
Nightly encrypted PostgreSQL backups and continuous WAL archive; aim RPO<=5min, RTO<=30min measured on target hardware. Keep >=7 restore points, weekly restore verification with privacy-erasure tombstone reapply; no reliance on nightly backups alone for 5min RPO. Event log+checkpoints determine exact last committed logical tick; never resume based on elapsed real wall time. Store default snapshot every600 ticks and wave boundary; rollback restores image digest + compatible schema, forbid unsafe downgrade that loses data.
Provider disconnect: freeze ticks, show banner, retry backoff 1/2/4/8/16/30s, require healthy source and authenticated operator resume after sustained connectivity >=10s. DB outage: stop acknowledgements and tick advance, enter STORAGE_UNAVAILABLE; replay after recovery. Unity reconnect: discard stale view, fetch public snapshot with new stream sequence, resync. Queue overload: drop invalid/non-actionable first, flag any genuine game command lost as DEGRADED and pause for operator if sustained 30s. Emergency stop: operator action audited, no further simulated ticks.
Privacy erase: queue, immediate alias suppression, anonymized checkpoint at safe wave boundary; retain deletion tombstone ledger for future restore. Incident retrospective required with timeline, root cause and reproduction.

## Production LIVE human prerequisites
Confirm approved provider event source, owner account/region and streaming rights; verify full overlay and latency on LIVE preview; original audio/art IP rights, privacy notice and deletion mechanism, staffed moderation, emergency stop, 2h rehearsal, physical workstation and recovery drills. Engineer automation CANNOT grant platform privileges or human legal approvals. If not met, mark LIVE release BLOCKED, allow standalone simulator deliverable only.
