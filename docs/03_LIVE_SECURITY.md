# LIVE INTEGRATION, PLATFORM COMPLIANCE AND SECURITY GATES

## External dependency gate: BLOCKED until verified
Do not assume TikTok provides public real-time comment API for this account. Production integration requires official or otherwise expressly authorized access, verified credentials, current documentation, terms, rate limits, identity and event ID fields, privacy/security review, and broadcaster permission. No scraping/automation bypass. If not verifiable, mock/replay mode is permitted but TikTok production release FAILS. Source: TikTok developer portal/official terms must be attached to M08 evidence.

## Adapters
Source interface Start(ctx,emit), Stop(ctx), Health(); capability metadata stable_identity,stable_event_id,event_clock,reconnect_cursor,moderation_data. Implement mock and deterministic NDJSON replay first; separately gated AuthorizedTikTokAdapter. Source event {source,source_session_id,source_event_id?,source_user_id?,display_name?,comment,received_at,source_at?}. Session must match active broadcaster. Missing stable ID => ephemeral session identity, no permanent player record, present limitation. Never use display name as identity.
Duplicates discarded using provider IDs when available. Out-of-order delivery uses arrival order to assign authoritative monotonic accepted sequence, not assumed provider order. Source loss shows CONNECTIVITY LOST; do not simulate comments as real input. UI cannot display fake viewer/user counts.

## Fairness and abuse resistance
Only 1 effective role action per 20s/actor; latest role overwrites. Only 1 effective vote per decision/actor. Bursts may be throttled per actor and globally; accurate dropped metric. Moderator blocklists, alias sanitization, manual name suppression, pause, mute, reconnect and emergency stop. Abuse-resistant controls MUST not log raw IPs needlessly. Gift events excluded from gameplay stats entirely.

## Privacy
Hash scoped provider identity with HMAC secret; encrypt backups, minimize data, document retention (raw diagnostic payload <=24h; identity/XP until user deletion/retention policy), account removal flow and opt-out name. Consent and policies must be reviewed under applicable law. Log no credentials or unredacted tokens.

## Attack tests
Duplicate/late events, same user 1,000 comments, 5k/s burst, malformed UTF-8, 1MB messages, injection markup, offensive alias, session switch, stolen admin cookie, compromised token, reconnect storms, missing provider IDs, persistence crash, secret rotation; record PASS evidence. Security release gate must reject any unresolved critical/high vulnerability.

## Human approval
Streamer/account owner approves TikTok event source, production ToS, age/region conditions, privacy policy, and broadcast rights. Legal approval is an external human gate. Agent cannot self-certify authorization.
