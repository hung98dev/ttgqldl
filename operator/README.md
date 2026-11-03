# TTGQLDL operator console

React 19 + TypeScript (strict) + Vite operator console for the TTGQLDL live-village
backend. This is the M01-W0105 foundation: a real, buildable, tested interface that
talks only to the endpoints the Go backend actually implements.

## What works today

| Capability | State | Notes |
|---|---|---|
| `GET /healthz` | AVAILABLE | liveness of the Go process |
| `GET /readyz` | AVAILABLE | readiness; 503 is shown as DEGRADED with the backend reason |
| Everything under `/admin/v1` | UNAVAILABLE | contracts exist, calls do not; actions are disabled with their owning milestone |
| LIVE source | BLOCKED | GATE-LIVE: no permission, approval or credential exists |

No authentication is implemented. The shipped adapter reports
`AUTH_UNAVAILABLE`; nothing is stored in `localStorage` except the theme choice.

## Requirements

- Node.js **24 LTS** (pinned 24.21.0), npm 11
- The Go backend running on `http://127.0.0.1:8080` (optional: the console still
  renders and shows OFFLINE when it is down)

## Commands

```bash
npm ci                 # exact lockfile install
npm run dev            # Vite on http://127.0.0.1:5173 (proxies /healthz, /readyz)
npm run typecheck      # tsc --noEmit
npm run lint           # eslint
npm run test -- --run  # Vitest + Testing Library
npm run build          # typecheck + production bundle in dist/
```

From the repository root, `tooling/operator-test.ps1|sh` runs the same gates, and
`tooling/operator-run.ps1|sh` starts the dev server (add `-Demo` / `--demo` for the
fixture preview).

## Demo mode

`VITE_OPERATOR_DEMO=true` (or `operator-run --demo`) renders fixed fixtures behind
a persistent **DEMO / DỮ LIỆU MÔ PHỎNG** banner. Demo data is never described as
real players, a real stream or an authenticated operator, and demo mutations are
labelled as local simulation. Production builds default to real-service mode and
show `UNAVAILABLE` states instead of fabricated rows.

## Architecture

```
src/contracts  wire types (health, villages, sessions, moderation, capabilities)
src/api        typed HTTP client, response guards, capability matrix, demo fixtures
src/app        composition root, routing, providers (api/auth/health/theme), i18n
src/components layout + reusable UI (cards, tables, dialogs, states, boundary)
src/features   one folder per operator workflow
```

Rules the code follows: no `any`, every external payload is validated before it
reaches a component, no direct `fetch` outside `src/api`, no global mutable state
outside React context, no dangerous HTML injection, and every request is abortable.

## Known gaps (by design)

- Authentication, sessions, moderation and admin actions belong to M05/M07.
- No browser E2E runner is configured in this repository; flows are covered by
  component tests in jsdom.
- Serving the SPA, TLS and CSP headers are deployment work (M08/M10).
