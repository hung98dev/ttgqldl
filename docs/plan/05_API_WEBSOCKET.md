# BACKEND API AND WEBSOCKET PROTOCOL

## Conventions and error model
All endpoints are /admin/v1 unless stated. JSON UTF-8, Content-Type application/json, no floats where integer required. UUID lowercase, UTC RFC3339 timestamps. Accept maximum HTTP request 16KiB (mock events aggregate max 64KiB). Success is 200/201/204, validation 400 code=INVALID_ARGUMENT, unauthenticated 401 UNAUTHENTICATED, forbidden 403 FORBIDDEN, missing 404 NOT_FOUND, wrong lifecycle/duplicate conflicting session 409 FAILED_PRECONDITION, throttle 429 RESOURCE_EXHAUSTED with Retry-After, DB unavailable 503 UNAVAILABLE. Every error JSON: {"error":{"code":"ENUM","message":"safe_text","request_id":"uuid","retryable":false}}. Never include secrets/provider identifiers or SQL errors. Idempotent mutations require Idempotency-Key UUID stored per actor/action for 24h; replay returns original response.
All admin write requests: same-site Secure HttpOnly session cookie, CSRF X-CSRF-Token linked to server session and Origin allowlist; RBAC roles viewer(read only), operator(start/pause/resume/mute), admin(accounts/data deletion/settings). Login protected with per-user and global rate limits. Unity broadcast client accesses separate view-only short-lived stream token via operator panel; token lifetime 300s and rotate without losing connection.

## Routes (exact request/response behavior)
- GET /healthz -> 200 {"status":"alive"}; no DB check.
- GET /readyz -> 200 {"status":"ready","writer_epoch":integer} or 503; protected metrics not exposed.
- GET /metrics -> Prometheus exposition, internal network only.
- POST /admin/v1/auth/login {username,password} ->200 {user:{id,role},csrf_token}; Set-Cookie secure session. Wrong creds ->401 generic; 5 attempts/5min ->429.
- POST /admin/v1/auth/logout {} ->204 revoke cookie.
- GET /admin/v1/me ->200 {id,role,csrf_token}; 401 anonymous.
- GET /admin/v1/sessions ->200 {items:[SessionSummary],next_cursor?}; bounded page size <=50.
- POST /admin/v1/sessions/start {village_id:UUID,source:"mock"|"replay"|"authorized_live"} ->201 {session_id,status:"waiting",wave_no:1}; 409 if already active; authorized_live requires verified gate flag and adapter auth, else403 PLATFORM_NOT_AUTHORIZED.
- POST /admin/v1/sessions/{id}/pause {reason:string<=256} ->200 SessionSummary; waiting/paused idempotent; forbidden status returns409.
- POST /admin/v1/sessions/{id}/resume {} ->200 SessionSummary; source_disconnected or DB unhealthy ->409.
- POST /admin/v1/sessions/{id}/stop {reason:string<=256} ->200 SessionSummary with stopped=true; idempotent if already stopped.
- GET /admin/v1/sessions/{id}/state ->200 PublicState plus internal safe operator stats but no raw stable provider IDs.
- GET /admin/v1/sessions/{id}/metrics ->200 counts, p50/p95 backend latency, ingress dropped breakdown, queue, ws.
- POST /admin/v1/sessions/{id}/stream-token {} ->200 {token,expires_at,websocket_url}; viewed only by operator/admin.
- POST /admin/v1/sessions/{id}/moderation {actor_public_ref:string,action:"hide"|"unhide"|"block",reason:string<=256} ->200 {applied:true}; never expose ActorKey via public wire.
- POST /admin/v1/sessions/{id}/privacy/erase {actor_public_ref:string,reason:string<=256} ->202 {status:"queued",request_id:UUID,status_url:string}; admin only, append auditable erasure workflow and a privacy_erase_requests row.
- GET /admin/v1/sessions/{id}/privacy/erase/{request_id} ->200 {request_id,status:"queued"|"applied"|"blocked"|"failed",requested_at,applied_at?,privacy_epoch?}; operator/admin read. This is the only way an operator can prove erasure completed, so M07 evidence depends on it.
- GET /admin/v1/audit?cursor=&limit=1..50 ->200 {items:[{id,actor_public_ref,action,target,reason,created_at}],next_cursor:null|string}; admin only, bounded page size, never returns secrets, password hashes, actor_key or raw provider IDs.
- POST /dev/v1/events {session_id:UUID,events:[{event_id:string,viewer_id:string,alias:string,comment:string}]} ->202 {accepted:int,ignored:int,invalid:int}; dev mode only, route not registered prod/stage.
- POST /admin/v1/sessions/{id}/settings {low_vfx:bool,mute_audio:bool} ->200 settings; operator permitted; changes do not alter combat/official gameplay.
Every request/response schema must be materialized as machine-readable OpenAPI 3.1 and verified via schema-conformance tests in M03; these exact routes are normative, not examples.

## WebSocket
GET /stream/v1/sessions/{session_id} with a short-lived token validated from Sec-WebSocket-Protocol; query-string tokens are forbidden in all environments to avoid leaks. TLS/WSS for remote. Upon connect send full public_snapshot within 1s, then public_state_diff at most 5Hz. Every envelope JSON:
{"schema_version":1,"type":"public_snapshot|public_state_diff|visual_event|connection_status|error","session_id":"uuid","stream_seq":"decimal-string","revision":"decimal-string","sent_at":"RFC3339","payload":{}}
stream_seq monotonic per session only; no gaps silently ignored. On gap, client requests {"schema_version":1,"type":"resync","after_stream_seq":"decimal-string"}; server sends public_snapshot with latest stream_seq (NOT unbounded delta backfill). Client discards older revisions and any unexpected schema version; present STALE/RECONNECTING when >3s without snapshot or heartbeat. Server ping every10s; idle timeout30s. max single outbound frame 128KiB, stream outbox max bounded 20 messages/client; drop cosmetic events first; if still full disconnect slow subscriber with 1013 and require fresh snapshot.
public_snapshot payload fixed fields: {wave_no:int,lifecycle:enum,pause_reasons:enum[],phase:enum,remaining_seconds:int,wall_hp:int,wall_max:int,townhall_hp:int,townhall_max:500,wood:int,stone:int,food:int,counts:{builder:int,guard:int,fighter:int,unique_joined:int},lanes:[{id:"west|center|east",ordinary_hp:int,boss_hp:int}],vote:{open:bool,options:[A,B,C],counts:{A:int,B:int,C:int},selected?:char},source_status:"connected|disconnected|mock|replay",spotlights:[{public_ref,display_alias,role}],event_banner:{kind,text_key,expiry_ms}}. Stream diff uses RFC6902 JSON Patch with explicit base_revision decimal-string only if payload<=128KiB and base_revision matches; otherwise server sends full snapshot. Fixed text keys map to client localization catalog, not arbitrary HTML or user input.

## Time/latency
Internal latency from backend event received to committed publish, p95 <=1.0s at measured target; external platform-to-backend not controllable and must be separately shown as UNKNOWN/MEASURED. UI acknowledgement can appear at next publish once valid command committed; domain effect occurs at 20s boundary. Max concurrent WebSocket clients initially 4 trusted devices, not 10k player sockets; viewers watch TikTok video, they are not websocket clients.

## Mandatory bootstrap, endpoints and protocol details
- POST /admin/v1/villages {display_name:string(1..64),source:"mock"|"replay"|"authorized_live"} ->201 {id:UUID,display_name,level:1,wall_tier:1}; requires admin and idempotency key. The server creates a random mock/replay broadcaster key; LIVE creation requires authorized source binding. GET /admin/v1/villages?cursor=&limit=1..50 ->200 {items:[VillageSummary],next_cursor:null|string}; GET /admin/v1/villages/{id} ->200 VillageSummary. Without these APIs a new clean installation cannot call sessions/start.
- GET /admin/v1/sessions/{id}/participants?query=... ->200 paginated max50 moderated alias and per-session random actor_public_ref, operator/admin only; no actor_key/provider IDs exposed. This makes block/hide/privacy work for players beyond the 12 public spotlights.
- Initial operator admin is created ONLY with TTG_BOOTSTRAP_ADMIN_USER and TTG_ADMIN_PASSWORD_HASH (Argon2id) on first startup; fail stage/prod startup if no admin and no secure bootstrap. Never embed default production admin passwords.
- Idempotency-Key must be valid UUID. Same principal+route+key+same SHA256 normalized body within24h returns original response, different body ->409 IDEMPOTENCY_CONFLICT. Password/cookie/stream token must never be echoed in idempotency storage.
- SessionSummary must include session_id,village_id,wave_no,wave_tick,lifecycle,phase,source_mode,pause_reasons. Distinguish operator PAUSED, source DISCONNECTED and storage UNAVAILABLE; all blockers must be cleared to resume.
- Unity connects via Sec-WebSocket-Protocol values ttg.v1,ttg.auth.<base64url_token>; server echoes only ttg.v1 and redacts any authorization protocol header from logs. Token view-only scoped per session, TTL300s, can be rotated without dropping an existing connection; URL tokens NEVER permitted.
- Snapshot phase enum waiting|prep|assault|finale|results. Include boss_incoming boolean; client must never show CLEAR while boss spawn pending. Diff envelope includes base_revision decimal-string and is rejected/resynced on mismatch. Send connection_status/heartbeat at least every2s, UI becomes STALE after3s without valid message.
