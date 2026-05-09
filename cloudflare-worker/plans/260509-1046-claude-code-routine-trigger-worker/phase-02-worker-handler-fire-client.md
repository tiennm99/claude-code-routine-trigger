---
phase: 2
title: Worker handler + fire client
status: completed
priority: P1
effort: 2-3h
dependencies:
  - 1
---

# Phase 2: Worker handler + fire client

## Overview
Implement the `scheduled` handler in plain JS with JSDoc type annotations: render text template, POST to `/fire`, log structured JSON. Single `worker.js` file. No `fetch` handler (per locked decision: scheduled-only).

## Requirements
- **Functional:**
  - On cron tick, POSTs to `ROUTINE_FIRE_URL` with bearer token, beta header, JSON body.
  - Logs `claude_code_session_url` on 2xx; logs status + body on non-2xx; never throws (would crash CF retry).
  - Renders `TEXT_TEMPLATE` with `{LocalTime}`, `{Cron}`, `{ISO}` placeholders.
  - No retry (each `/fire` = new session).
- **Non-functional:**
  - Token never logged.
  - Use `console.log(JSON.stringify({...}))` for structured logs (CF picks these up).
  - JSDoc `@typedef Env` for environment bindings + `@param`/`@returns` on all exports.
  - Total file ≤ 200 lines (split if exceeds).

## Architecture

```
ScheduledEvent (cron, scheduledTime)
    │
    ▼
fireRoutine(env, controller)
    ├── renderText(template, vars)         # simple {Tok} substitution
    ├── fetch(URL, POST, headers, body)
    └── log(json)                          # success or error
```

### Mapping from Go daemon
| Go (claude-code-routine-cron)              | JavaScript (this worker)                            |
|---|---|
| `Config.FireURL` env                        | `env.ROUTINE_FIRE_URL` secret                       |
| `Config.Token` env                          | `env.ROUTINE_FIRE_TOKEN` secret                     |
| `Config.Schedules` env                      | `wrangler.toml` `[triggers].crons` (literal)        |
| `Config.Location` (TZ)                      | `env.TZ` plain var, default `UTC`                   |
| `Config.Template` (Go text/template)        | `env.TEXT_TEMPLATE` plain string + `{Tok}` substitution |
| `FireClient.Fire()`                         | `fireRoutine()` function                            |
| `cron.New()` scheduler                      | CF Workers scheduled handler (built-in)             |
| `slog` JSON logs                            | `console.log(JSON.stringify(...))`                  |

### JSDoc typedefs
```js
/**
 * @typedef {object} Env
 * @property {string} ROUTINE_FIRE_URL  - secret: Anthropic /fire endpoint
 * @property {string} ROUTINE_FIRE_TOKEN - secret: per-routine bearer token
 * @property {string} [TEXT_TEMPLATE]   - plain var: prompt template, supports {LocalTime} {Cron} {ISO}
 * @property {string} [TZ]              - plain var: IANA tz, default 'UTC'
 */

/**
 * @typedef {object} FireResponse
 * @property {string} type
 * @property {string} claude_code_session_id
 * @property {string} claude_code_session_url
 */
```

### Headers (verbatim from siblings)
```
Authorization: Bearer ${token}
anthropic-version: 2023-06-01
anthropic-beta: experimental-cc-routine-2026-04-01
Content-Type: application/json
```

### Body
```json
{ "text": "<rendered template>" }
```

### Template substitution (KISS — not Go text/template)
| Token         | Value                                                 |
|---------------|-------------------------------------------------------|
| `{ISO}`       | `new Date(scheduledTime).toISOString()`               |
| `{LocalTime}` | formatted via `Intl.DateTimeFormat(env.TZ ?? 'UTC')`  |
| `{Cron}`      | `controller.cron`                                     |

Default template: `Scheduled trigger at {LocalTime}` — same default as Go daemon.

## Related Code Files
- Create: `worker.js` (export default `{ scheduled }`, JSDoc-annotated)
- (Optional split if >200 lines) `fire-client.js`, `template-renderer.js`

## Implementation Steps
1. Top of `worker.js`: write JSDoc `@typedef Env` + `@typedef FireResponse` blocks.
2. Implement `renderText(template, vars)` with JSDoc:
   ```js
   /**
    * @param {string} template
    * @param {Record<string, string>} vars
    * @returns {string}
    */
   ```
   Replaces `{Key}` tokens, leaves unknown tokens intact.
3. Implement `formatLocalTime(date, tz)` using `Intl.DateTimeFormat`. JSDoc: `@param {Date} date`, `@param {string} tz`, `@returns {string}`.
4. Implement `fireRoutine(env, controller)` with JSDoc `@param {Env} env`, `@param {ScheduledController} controller`, `@returns {Promise<void>}`:
   - Build `text` via `renderText`.
   - `fetch(env.ROUTINE_FIRE_URL, { method: 'POST', headers, body: JSON.stringify({text}) })`.
   - On `!response.ok`: `console.log(JSON.stringify({level:'error', cron, status, body}))`.
   - On `ok`: parse JSON, log `{level:'info', cron, session_url, session_id}`.
   - Wrap in try/catch — log network errors, don't rethrow.
5. Export default with JSDoc `@type {ExportedHandler<Env>}`:
   ```js
   /** @type {ExportedHandler<Env>} */
   export default {
     async scheduled(controller, env, ctx) {
       ctx.waitUntil(fireRoutine(env, controller));
     },
   };
   ```
6. `npx tsc --noEmit --allowJs --checkJs worker.js --target es2022 --module esnext --moduleResolution bundler --types @cloudflare/workers-types` (one-shot smoke check; not added to scripts to avoid TS dependency creep). Optional — skip if undesired.
7. Local smoke: `wrangler dev --test-scheduled`, then `curl "http://localhost:8787/__scheduled?cron=*+*+*+*+*"` — confirm log line.

## Success Criteria
- [ ] `worker.js` ≤ 200 lines (split if needed)
- [ ] All exported functions have JSDoc with `@param` + `@returns`
- [ ] `wrangler deploy --dry-run` validates module
- [ ] Token never appears in logged output (manual grep on `wrangler dev` output)
- [ ] Local `__scheduled` test produces structured JSON log

## Risk Assessment
- **`Intl.DateTimeFormat` TZ support on Workers runtime:** supported since 2023; verify with smoke test.
- **`ctx.waitUntil` vs awaiting in `scheduled`:** `waitUntil` ensures fire completes even if handler returns; preferred per CF docs.
- **No compile-time type errors:** JSDoc + editor LSP catches most; runtime tests (phase 4) catch the rest.
- **Template injection:** `text` is sent only to Anthropic API; no XSS / SQL surface — safe.

## Security Considerations
- Token comes only from `env.ROUTINE_FIRE_TOKEN` (CF Secret), never bundled.
- `console.log` body must not include `Authorization` header value.
- TLS to `api.anthropic.com` is automatic (Workers `fetch` validates certs).
