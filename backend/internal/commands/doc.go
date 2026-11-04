// Package commands owns parsing and normalization of player comments. It turns
// untrusted UTF-8 text into exactly one recognized, typed action or a stable
// rejection; it never touches identity, persistence, scheduling or simulation
// state.
//
// The contract (02_GAME_RULES.md "Exact player action model",
// 06_LIVE_SECURITY.md "Adapter specification", 09_QA_ACCEPTANCE.md QA-CMD-01):
//
//   - Role tokens: XAY/XÂY (builder), THU/THỦ (guard), DANH/ĐÁNH (fighter),
//     BUILD (builder), DEFEND (guard), ATTACK (fighter); case-insensitive.
//   - Ballot tokens: A, B, C (a distinct ballot action, never a role).
//   - Exactly one token after normalization. Punctuation, substring and
//     multi-word input are rejected; there is no prefix or substring matching.
//   - Unicode-equivalent forms that normalize exactly to an authorized token are
//     accepted; deceptive mixed-script lookalikes are not.
//   - Maximum comment length is 500 UTF-8 bytes, checked before any Unicode
//     processing.
//
// Normalization is a fixed, locale-independent pipeline:
//
//  1. byte length (reject >500) and UTF-8 validity;
//  2. reject forbidden controls, Unicode format characters (bidi overrides,
//     isolates, zero-width marks) and excessive combining marks, while allowing
//     legitimate Unicode whitespace;
//  3. Unicode NFKC compatibility normalization;
//  4. deterministic Unicode lower-casing (strings.ToLower, locale-independent);
//  5. Vietnamese diacritic folding (NFD, drop combining marks, map đ/Đ to d);
//  6. Unicode whitespace normalization to ASCII space, then trim;
//  7. exact allowlist match, then mapping to the domain role or vote enum.
//
// The parser is pure, deterministic and thread-safe: no clock, network, database,
// locale, randomness, global mutable state or logging of rejected input. It does
// not award XP, register participants, enforce cooldowns, deduplicate provider
// events, generate actor keys, schedule input sequences or publish events; those
// belong to later M03 work packages and the existing reducer.
package commands
