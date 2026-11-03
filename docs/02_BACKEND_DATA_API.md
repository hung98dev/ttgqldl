# BACKEND, DATABASE AND PROTOCOL — HARD CONTRACT

## Component ownership
Go modular monolith. Ingest adapter validates provenance; parser converts normalized commands; identity service computes scoped HMAC actor key; command service dedupes and upserts effective action keyed session/window/actor; domain simulation only consumes immutable ordered action-resolution batches and seeded RNG; PostgreSQL owns durable state; WebSocket publishes bounded snapshots and events. A single writer per village must hold a Postgres advisory lock or equivalent fencing token. Multi-process race cannot run two authoritative simulations.

## Data model (migrations M01-M03)
villages(id uuid primary key, broadcaster_key text unique not null,level int check level between 1 and 50,wall_tier int check 1..20,created_at timestamptz);
sessions(id uuid primary key,village_id uuid fk,phase text,wave int,sim_tick bigint,rule_version text,rng_seed bigint,last_sequence bigint,status text,started_at timestamptz,stopped_at timestamptz);
players(id uuid primary key,village_id uuid fk,actor_key text not null,alias text,hide_alias bool,xp bigint,role text,last_seen_at timestamptz,unique(village_id,actor_key));
ingest_receipts(session_id uuid,source text,source_event_id text,received_at timestamptz,primary key(session_id,source,source_event_id));
window_actions(session_id uuid,window_index bigint,actor_key text,role text,updated_sequence bigint,primary key(session_id,window_index,actor_key));
ballots(session_id uuid,vote_id text,actor_key text,option text,updated_sequence bigint,primary key(session_id,vote_id,actor_key));
accepted_events(session_id uuid,sequence bigint,kind text,actor_key text,payload jsonb,rule_version text,created_at timestamptz,primary key(session_id,sequence));
snapshots(session_id uuid,sequence bigint,state jsonb,sha256 text,created_at timestamptz,primary key(session_id,sequence));
operator_audit(id uuid primary key,operator_id text,action text,reason text,created_at timestamptz);
All timestamps UTC, migrations reversible when feasible, constraints/indexes tested; no unnecessary raw comment archive. Separate secret/key rotation mapping; data deletion process.

## Transaction ordering
Atomic ingest receipt insert + validated accepted event/window upsert + outbox record in one database transaction, ack after commit. Duplicate unique violation => idempotent no-op. Sequence allocated inside serialized per-village transaction. Replay from verified last snapshot + accepted_events in ascending sequence; snapshot SHA = sha256(canonical state encoding) where fields and order are defined in a golden schema. Never use system time or nondeterministic maps in combat.

## API v1
GET /healthz (process only), GET /readyz (DB and ownership), GET /metrics (restricted)
POST /admin/v1/sessions/start, /pause, /resume, /stop (authenticated streamer/admin, CSRF protected browser auth, audit)
GET /admin/v1/state and /admin/v1/sessions (authenticated)
POST /dev/v1/events (mock mode only, disabled in prod builds/config)
GET /stream/v1/villages/{id} WebSocket requiring short-lived stream token.
Message envelope {schema_version:1,type,session_id,sequence,sent_at,payload}; types snapshot, state_diff, visual_event, connection_status, error. Full snapshot on connect/resync; monotonic sequence; old/unknown schema rejected with actionable diagnostic. Snapshot publisher <=5 Hz default; visual event rate cap 20/s; max client payload 128KiB; bounded write queue drop cosmetic events before state, disconnect slow readers then resync.

## Network/security
All remote traffic HTTPS/WSS; localhost ports not publicly bound by default. TLS termination via reverse proxy with documented config. Admin RBAC and audit, secret environment variables/secret files never Git, no SQL string composition, prepared pgx queries, Go context deadlines and graceful SIGTERM. Inputs bound: message <=500 UTF8 chars, source alias <=64 graphemes sanitised; reject malformed huge payload. No HTML execution in Unity/dashboard. HMAC keys are rotated without losing historic player identity via explicit mapping migration.

## Reliability and performance
Bounded ingest queue default 20,000 events; explicitly count rejected/dropped and present to streamer. Never claim all comments processed under backpressure. Tick 10Hz but authoritative role resolution at 20s. Reconnect with full snapshot. Snapshot every 60s and every completed wave, events after snapshot used for replay. A crash and replay must produce identical state digest.

## Deliverables
Actual executable SQL migrations, sqlc config+queries, OpenAPI 3.1 with full request/response/error models, websocket JSON schema, generated typed client, diagram, threat model and fixtures; prose alone is never completion.
