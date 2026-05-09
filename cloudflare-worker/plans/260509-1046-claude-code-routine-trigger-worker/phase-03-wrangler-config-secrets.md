---
phase: 3
title: wrangler config + secrets
status: completed
priority: P1
effort: 1h
dependencies:
  - 1
  - 2
---

# Phase 3: wrangler config + secrets

## Overview
Author `wrangler.toml` with literal `[triggers].crons`, define plain vars vs secrets, document deployment flow. CF Workers cron triggers fire from `wrangler.toml` only — same constraint as GH Actions.

## Requirements
- **Functional:**
  - `wrangler deploy` succeeds with valid `wrangler.toml`.
  - At least one cron expression in `[triggers]` (default 5×daily mirroring sibling repos).
  - Secrets uploaded out-of-band (not in `wrangler.toml`).
- **Non-functional:**
  - All comments above each block explain the field.
  - Default crons use scattered minutes (avoid minute `0`) per `claude-code-routine-trigger` README warning — though CF cron is more reliable than GH, scattering is still good hygiene.

## Architecture

### `wrangler.toml` shape
```toml
name = "claude-code-routine-trigger-worker"
main = "worker.js"
compatibility_date = "2026-05-09"
compatibility_flags = ["nodejs_compat"]   # only if Intl needs polyfill — verify in phase 2

# Plain vars (visible in dashboard, fine for non-secrets)
[vars]
TEXT_TEMPLATE = "Scheduled trigger at {LocalTime}"
TZ = "Asia/Ho_Chi_Minh"

# Cron triggers — LITERAL, cannot read from env/secrets/vars
[triggers]
crons = [
  "7 17 * * *",   # 00:07 UTC+7
  "13 22 * * *",  # 05:13 UTC+7
  "19 3 * * *",   # 10:19 UTC+7
  "23 8 * * *",   # 15:23 UTC+7
  "37 13 * * *",  # 20:37 UTC+7
]

# Observability — opt-in, free tier supports basic logs
[observability]
enabled = true
```

### Secrets (uploaded via `wrangler secret put`)
| Name                 | Source                              |
|----------------------|-------------------------------------|
| `ROUTINE_FIRE_URL`   | Anthropic routine editor → API trigger |
| `ROUTINE_FIRE_TOKEN` | Anthropic routine editor (shown once) |

### Local dev: `.dev.vars`
```
ROUTINE_FIRE_URL=https://api.anthropic.com/v1/claude_code/routines/trig_.../fire
ROUTINE_FIRE_TOKEN=sk-ant-oat01-...
```
Gitignored. Used by `wrangler dev` only.

### `wrangler.toml.example`
Committed copy with placeholder values for documentation; real `wrangler.toml` ships with author's defaults but secrets are NEVER literal.

## Related Code Files
- Create: `wrangler.toml`
- Create: `wrangler.toml.example` (or document in README — KISS, prefer README)
- Create: `.dev.vars.example` with placeholder secret names + dummy values

## Implementation Steps
1. Write `wrangler.toml` per architecture block above.
   - Pick `compatibility_date` = today (2026-05-09).
   - Decide on `nodejs_compat` flag — only enable if phase 2 needs it (`Intl` is built-in, likely **don't need it**).
2. Write `.dev.vars.example`:
   ```
   ROUTINE_FIRE_URL=https://api.anthropic.com/v1/claude_code/routines/trig_REPLACE/fire
   ROUTINE_FIRE_TOKEN=sk-ant-oat01-REPLACE
   ```
3. Verify `.gitignore` includes `.dev.vars` (added in phase 1).
4. Local validation: `wrangler dev` — must boot without error.
5. Document secret-upload flow in phase 5 README:
   ```bash
   echo -n 'https://...' | wrangler secret put ROUTINE_FIRE_URL
   echo -n 'sk-ant-oat01-...' | wrangler secret put ROUTINE_FIRE_TOKEN
   ```
6. Commit: `feat: wrangler config with default 5x-daily crons`.

## Success Criteria
- [ ] `wrangler dev` starts cleanly with `.dev.vars` populated
- [ ] `wrangler deploy --dry-run` validates `wrangler.toml` without errors
- [ ] All 5 default crons match sibling repos' cadence (UTC+7 daily 00/05/10/15/20)
- [ ] No secret values in any committed file
- [ ] `.dev.vars` confirmed in `.gitignore`

## Risk Assessment
- **CF cron expression syntax:** CF Workers cron supports standard 5-field; `*/N` and ranges work. Avoid `?` (Quartz-only).
- **Minute-`0` GH issue does not apply to CF** but scattering minutes is still recommended (free tier shared infra benefits).
- **Free tier limit:** 5 cron triggers per worker. Default config uses exactly 5 — at limit. Document that adding a 6th requires another worker.

## Security Considerations
- Secrets never in `wrangler.toml` (only `[vars]` for non-secrets).
- `.dev.vars` is gitignored.
- `wrangler secret put` accepts stdin, avoiding shell history leak (use `echo -n ... | wrangler secret put`).
