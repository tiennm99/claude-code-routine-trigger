# claude-code-routine-trigger-worker

Cloudflare Worker that fires a [Claude Code routine](https://code.claude.com/docs/en/routines) on a precise cron schedule via the [`/fire` API](https://platform.claude.com/docs/en/api/claude-code/routines-fire). Runs on Cloudflare's free tier — no servers, no GitHub minutes, no Docker host.

> [!TIP]
> **Anthropic's routine editor now ships a built-in cron trigger** — it runs on Anthropic's infra, no setup. Use it first.
>
> This worker is for users who want self-hosted-like control without operating their own infra: secrets in CF, logs in CF, schedule in code review.

## Why this vs the siblings

|                  | [claude-code-routine-trigger](https://github.com/tiennm99/claude-code-routine-trigger) | [claude-code-routine-cron](https://github.com/tiennm99/claude-code-routine-cron) | **claude-code-routine-trigger-worker** (this) |
| ---------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | --------------------------------------------- |
| Runs on          | GitHub Actions runners                                                                 | your infra (Docker host, k8s, NAS, RPi)                                          | Cloudflare edge                               |
| Cost             | free (within GitHub minutes)                                                           | minimal (your infra)                                                             | free (CF free tier)                           |
| Cron precision   | ±30 min – 2 h, can drop runs                                                           | sub-second                                                                       | within ~15 sec                                |
| Setup            | fork + 2 repo secrets                                                                  | env vars + Docker                                                                | `wrangler deploy` + 2 secrets                 |
| Audit trail      | GitHub Actions runs page                                                               | container stdout                                                                 | CF dashboard / `wrangler tail`                |

## Quickstart

```bash
git clone https://github.com/tiennm99/claude-code-routine-trigger-worker
cd claude-code-routine-trigger-worker
npm install

# Upload secrets (from Anthropic routine editor → API trigger)
echo -n 'https://api.anthropic.com/v1/claude_code/routines/trig_.../fire' \
  | npx wrangler secret put ROUTINE_FIRE_URL
echo -n 'sk-ant-oat01-...' \
  | npx wrangler secret put ROUTINE_FIRE_TOKEN

# Edit the cron schedule and timezone in wrangler.toml, then:
npx wrangler deploy
```

Tail logs:

```bash
npx wrangler tail
```

A successful fire logs a JSON line with `session_url`. Open it to watch the run.

## Environment variables

Configured in `wrangler.toml` (`[vars]` for plain values) or via `wrangler secret put` (for secrets).

| Name                 | Type   | Required | Default                                  | Notes |
| -------------------- | ------ | :------: | ---------------------------------------- | ----- |
| `ROUTINE_FIRE_URL`   | secret | yes      | —                                        | Anthropic `/fire` endpoint. From routine editor → API trigger. |
| `ROUTINE_FIRE_TOKEN` | secret | yes      | —                                        | `sk-ant-oat01-...` per-routine token. Shown once in the editor. |
| `TEXT_TEMPLATE`      | var    | no       | `Scheduled trigger at {LocalTime}`       | Token-substitution template. See *Templates*. |
| `TZ`                 | var    | no       | `UTC`                                    | IANA tz name (`Asia/Ho_Chi_Minh`, `America/New_York`, …). Used for `{LocalTime}` formatting. |

## Customize the schedule

Edit `wrangler.toml` `[triggers].crons` — Cloudflare requires literal cron expressions (same constraint as GitHub Actions). To change when the routine fires:

```toml
[triggers]
crons = [
  "0 0-17,22-23 * * *", # UTC+7: 00:00 + 05:00..23:00 hourly
]
```

Then redeploy: `npx wrangler deploy`.

Tips:
- Cron runs in **UTC**. Convert your local time: `UTC = local − offset` (e.g. 09:00 UTC+7 → 02:00 UTC → `0 2 * * *`).
- Validate expressions at <https://crontab.guru/>.
- **Free tier limit:** 5 cron expressions per worker. Default config uses 1.
- Standard 5-field syntax — supports `*`, `,`, `-`, `/`. Comma lists (`22,23,0,1,2`) and ranges (`3-7`) let you cram many fires into one expression.

## Templates

`TEXT_TEMPLATE` supports these `{Token}` substitutions, rendered per fire. Unknown tokens are left intact in the output.

| Token         | Example                                |
| ------------- | -------------------------------------- |
| `{ISO}`       | `2026-05-09T03:19:00.000Z`             |
| `{LocalTime}` | `2026-05-09 10:19 GMT+7`               |
| `{Cron}`      | `19 3 * * *` — the expression that fired |

Example:

```toml
[vars]
TEXT_TEMPLATE = "Daily digest at {LocalTime} (cron {Cron})"
TZ = "America/New_York"
```

## Local development

Copy `.dev.vars.example` to `.dev.vars` and fill in your routine credentials:

```bash
cp .dev.vars.example .dev.vars
# edit .dev.vars

npx wrangler dev --test-scheduled
```

Then trigger a scheduled run:

```bash
curl "http://localhost:8787/__scheduled?cron=*+*+*+*+*"
```

`.dev.vars` is gitignored — never commit it.

## Tests

```bash
npm test
```

Vitest runs in the Workers runtime via `@cloudflare/vitest-pool-workers`. Tests mock `fetch`, so they consume no Anthropic quota.

## Secret rotation

`wrangler secret put` overwrites silently. Rotate by re-running it with the new value:

```bash
echo -n 'sk-ant-oat01-newvalue' | npx wrangler secret put ROUTINE_FIRE_TOKEN
```

The change takes effect on the next deploy or within a few seconds via the live config.

## Beta header

The request pins `anthropic-beta: experimental-cc-routine-2026-04-01` (constant in `worker.js`). When Anthropic ships a new dated beta, bump it via a release. Older dated values keep working for a transition window per Anthropic's beta policy.

## Operational notes

- **Time accuracy**: cron precision on CF Workers is within ~15 seconds — adequate for routine triggering.
- **No retry**: each `/fire` POST creates a new Claude Code session — retrying multiplies sessions and burns quota. The worker logs failures and moves on.
- **Logs / traces**: `[observability]` is enabled in `wrangler.toml` with `head_sampling_rate = 1` and `invocation_logs = true` — every invocation produces a structured log + trace, retained per the [Workers Logs retention policy](https://developers.cloudflare.com/workers/observability/logs/workers-logs/) (3 days on the free plan). View live with `npx wrangler tail`, or browse history in the CF dashboard under **Workers & Pages → your worker → Logs**.
- **Cost**: scheduled handlers count against the Workers Free plan's 100k requests/day budget. 5 daily fires × 30 days = 150 requests/month — negligible.

## Security

- The token is **per-routine**: a leak only fires that one routine.
- Secrets live in CF's encrypted secret store, never in the bundle, never in logs (verified by tests).
- TLS to `api.anthropic.com` uses CF's standard cert verification.
- `worker.js` has no `fetch` HTTP handler — the worker is unreachable from the public internet, only fires on cron tick.

## License

[Apache-2.0](./LICENSE)
