---
title: claude-code-routine-trigger-worker — CF Workers cron port
description: >-
  Cloudflare Workers port of claude-code-routine-cron: scheduled handler fires
  Claude Code routine /fire endpoint, zero infra, free tier
status: completed
completed: 2026-05-09
priority: P2
created: 2026-05-09T00:00:00.000Z
target_repo: /config/workspace/tiennm99/claude-code-routine-trigger-worker
reference_repos:
  - /config/workspace/tiennm99/claude-code-routine-trigger
  - /config/workspace/tiennm99/claude-code-routine-cron
license: Apache-2.0
blockedBy: []
blocks: []
---

# claude-code-routine-trigger-worker — CF Workers cron port

## Problem
Two existing siblings cover the routine-fire space, both with trade-offs:
- `claude-code-routine-trigger` (GH Actions): free, but cron unreliable (30 min – 2h delays, occasional drops at minute `0`).
- `claude-code-routine-cron` (Go daemon, Docker): sub-second precision, but requires self-hosted infra.

Gap: a free, no-infra option with reliable timing.

## Goal
Cloudflare Workers port: scheduled handler POSTs to the Claude Code routine `/fire` endpoint on a fixed cron. Free tier (CF Workers cron triggers cost $0 within free limits). Single TypeScript file, deploy via `wrangler deploy`.

## Locked Decisions
| # | Decision | Choice |
|---|----------|--------|
| 1 | Repo name | Completed |
| 2 | Scope | Completed |
| 3 | Cron config | Completed |
| 4 | Manual HTTP fire | Completed |
| 5 | Language | Completed |
| 6 | Secret management | `wrangler secret put` — `ROUTINE_FIRE_URL`, `ROUTINE_FIRE_TOKEN` |
| 7 | Optional config | `TEXT_TEMPLATE` (env var, plain string with placeholders), `TZ` (env var) |
| 8 | License | Apache-2.0 (matches siblings) |

## Non-goals
- Manual `/fire` HTTP endpoint (KISS — scheduled only)
- Multiple routines per worker (deploy N workers for N routines)
- Retry on failure (each POST = new session; retries multiply sessions)
- Durable Objects / KV / D1 (stateless)
- Custom log shipping (CF dashboard `wrangler tail` is enough)
- Go-style `text/template` (use simple `{token}` substitution — KISS)

## Comparison Matrix (post-implementation)
|                  | trigger (GH Actions) | cron (Go daemon)         | **trigger-worker (this)** |
| ---------------- | -------------------- | ------------------------ | ------------------------- |
| Runs on          | GitHub runners       | self-hosted Docker       | Cloudflare edge           |
| Cost             | free (GH minutes)    | minimal (own infra)      | free (CF free tier)       |
| Cron precision   | ±30 min – 2 h        | sub-second               | ±15 sec (CF SLA)          |
| Setup            | fork + 2 secrets     | env vars + Docker        | wrangler deploy + secrets |
| Audit trail      | Actions runs page    | container stdout         | CF dashboard / `tail`     |

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Repo scaffold](./phase-01-repo-scaffold.md) | Completed |
| 2 | [Worker handler + fire client](./phase-02-worker-handler-fire-client.md) | Completed |
| 3 | [wrangler config + secrets](./phase-03-wrangler-config-secrets.md) | Completed |
| 4 | [Tests + CI](./phase-04-tests-ci.md) | Completed |
| 5 | [Docs + release](./phase-05-docs-release.md) | Completed |

## Dependencies
None. Greenfield repo. Reference repos read-only.

## Risk Register
| Risk | Severity | Mitigation |
|---|---|---|
| CF Workers cron precision in practice | Low | Documented ±15 sec — adequate for routine triggering |
| Wrangler config drift with new CF features | Low | Pin wrangler major version in `package.json` |
| Beta header (`experimental-cc-routine-2026-04-01`) churn | Medium | Constant in code; bump via release; siblings have same exposure |
| Free tier cron limit (5 unique cron expressions per worker) | Low | One-routine-per-worker model means single-digit crons typical |
| Secrets accidentally committed | High | `.dev.vars` gitignored; secrets only via `wrangler secret put` in CI |

## Success Criteria
- [ ] `wrangler deploy` ships worker; first cron tick fires `/fire` and logs `claude_code_session_url`.
- [ ] `wrangler tail` shows JSON-structured logs for each fire.
- [ ] All CF Workers cron expressions in `wrangler.toml` validated by `wrangler dev` locally.
- [ ] CI runs typecheck + tests on push.
- [ ] README documents quickstart, env vars, secret rotation, beta header policy.
- [ ] Release `v0.1.0` published with `wrangler.toml` example.

## References
- Sibling Go daemon: `/config/workspace/tiennm99/claude-code-routine-cron`
- Sibling GH Actions: `/config/workspace/tiennm99/claude-code-routine-trigger`
- Anthropic `/fire` API: https://platform.claude.com/docs/en/api/claude-code/routines-fire
- CF Workers cron: https://developers.cloudflare.com/workers/configuration/cron-triggers/
