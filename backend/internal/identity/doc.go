// Package identity owns scoped pseudonymous actor keys and the display-alias privacy
// boundary defined by docs/plan/06_LIVE_SECURITY.md. M03-W0303 implements pure,
// versioned HMAC-SHA256 derivation, candidate derivation across retained key versions,
// session-scoped ephemeral identities and structural alias sanitization.
//
// ActorKey contract: exactly 64 lowercase hexadecimal characters, produced by
//
//	HMAC-SHA256(secret,
//	    lp("ttg-actor-key") || u64be(1) || lp(key_version) ||
//	    lp(provider) || lp(provider_version) || lp(broadcaster_key) || lp(viewer_id))
//
// where lp(s) = u64be(len(s)) || s. The tuple binds provider, provider identity
// version, broadcaster/village scope, stable provider viewer id and key version.
// Display alias, comment text, timestamps, event sequence, gift counts, client IP,
// fingerprints and per-message random ids are deliberately absent.
//
// Security properties:
//
//   - Derivation is pure, deterministic and thread-safe; keys are immutable and
//     supplied explicitly (no default production secret, no global registry, no
//     database, HTTP or TikTok dependency).
//   - Secret material is private to Key/KeySet; String/GoString are redacted and
//     errors never include identities or secrets.
//   - Missing, short, degenerate or invalid key material fails closed; duplicate and
//     unknown key versions are rejected.
//   - Ephemeral identities are bound to one session and an explicit injected synthetic
//     secret, labelled synthetic, expire explicitly and never imply provider-backed
//     continuity. Missing stable LIVE viewer ids must fail closed; they are never
//     silently replaced by ephemeral identities.
//   - ActorKey is private. Raw provider viewer ids and ActorKeys must not reach the
//     public stream, Unity or user-visible logs.
//
// Rotation boundary: an old HMAC digest cannot recover the original viewer id when the
// secret changes. Candidates are derived transiently from a lawful raw viewer id under
// the active and retained key versions; durable alias resolution, alias append and
// retirement are M04 persistence work and are NOT implemented here. This package does
// not claim QA-SEC-03 rotation/erasure completion.
//
// Alias sanitization is structural only (UTF-8, length/grapheme bounds,
// controls/bidi/mark floods, URL-like/markup rejection, opt-out/fallback). Lexical
// moderation, profanity policy and audit remain later M05/M08 gates; accepting an
// alias at the source boundary does not authorize publication.
package identity
