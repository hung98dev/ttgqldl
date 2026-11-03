# ENDPOINT AND STREAMING CONTRACT — VERSION 1

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
- POST /admin/v1/sessions/{id}/privacy/erase {actor_public_ref:string,reason:string<=256} ->202 {status:"queued",request_id:UUID}; admin only, append auditable erasure workflow.
- POST /dev/v1/events {session_id:UUID,events:[{event_id:string,viewer_id:string,alias:string,comment:string}]} ->202 {accepted:int,ignored:int,invalid:int}; dev mode only, route not registered prod/stage.
- POST /admin/v1/sessions/{id}/settings {low_vfx:bool,mute_audio:bool} ->200 settings; operator permitted; changes do not alter combat/official gameplay.
Every request/response schema must be materialized as machine-readable OpenAPI 3.1 and verified via schema-conformance tests in M03; these exact routes are normative, not examples.

## WebSocket
GET /stream/v1/sessions/{session_id}?token=<opaque>; for security prefer Sec-WebSocket-Protocol bearer token in production instead of URL query (reject URL tokens in prod to avoid log leak). TLS/WSS for remote. Upon connect send full public_snapshot within 1s, then public_state_diff at most 5Hz. Every envelope JSON:
{"schema_version":1,"type":"public_snapshot|public_state_diff|visual_event|connection_status|error","session_id":"uuid","stream_seq":"decimal-string","revision":"decimal-string","sent_at":"RFC3339","payload":{}}
stream_seq monotonic per session only; no gaps silently ignored. On gap, client requests {"schema_version":1,"type":"resync","after_stream_seq":"decimal-string"}; server sends public_snapshot with latest stream_seq (NOT unbounded delta backfill). Client discards older revisions and any unexpected schema version; present STALE/RECONNECTING when >3s without snapshot or heartbeat. Server ping every10s; idle timeout30s. max single outbound frame 128KiB, stream outbox max bounded 20 messages/client; drop cosmetic events first; if still full disconnect slow subscriber with 1013 and require fresh snapshot.
public_snapshot payload fixed fields: {wave_no:int,phase:enum,remaining_seconds:int,wall_hp:int,wall_max:int,townhall_hp:int,townhall_max:500,wood:int,stone:int,food:int,counts:{builder:int,guard:int,fighter:int,unique_joined:int},lanes:[{id:"west|center|east",ordinary_hp:int,boss_hp:int}],vote:{open:bool,options:[A,B,C],counts:{A:int,B:int,C:int},selected?:char},source_status:"connected|disconnected|mock|replay",spotlights:[{public_ref,display_alias,role}],event_banner:{kind,text_key,expiry_ms}}. Stream diff uses RFC6902 JSON Patch only if payload<=128KiB and base_revision matches; otherwise server sends full snapshot. Fixed text keys map to client localization catalog, not arbitrary HTML or user input.

## Time/latency
Internal latency from backend event received to committed publish, p95 <=1.0s at measured target; external platform-to-backend not controllable and must be separately shown as UNKNOWN/MEASURED. UI acknowledgement can appear at next publish once valid command committed; domain effect occurs at 20s boundary. Max concurrent WebSocket clients initially 4 trusted devices, not 10k player sockets; viewers watch TikTok video, they are not websocket clients.
