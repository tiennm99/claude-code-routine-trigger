# claude-code-routine-trigger-worker

Cloudflare Worker that fires a [Claude Code routine](https://code.claude.com/docs/en/routines) on a precise cron schedule via the [`/fire` API](https://platform.claude.com/docs/en/api/claude-code/routines-fire). Runs on Cloudflare's free tier — no servers, no GitHub minutes, no Docker host.

> [!TIP]
> **Anthropic's routine editor now ships a built-in cron trigger** — it runs on Anthropic's infra, no setup. Use it first.
>
> If you need an external scheduler (Anthropic cron disabled, custom payloads, audit trail outside Anthropic), I recommend **[cron-job.org](https://cron-job.org)** — free, no infra, ~2-min setup. I tried it on this exact routine and am satisfied, so I've parked this worker (cron triggers are commented out in `wrangler.toml`) to keep my CF cron-trigger quota free for other projects. See [Recommended: cron-job.org](#recommended-cron-joborg).
>
> This worker is still a valid path if you specifically want secrets in CF / logs in CF / schedule in code review. See [Using Cloudflare Workers](#using-cloudflare-workers).

## Recommended: cron-job.org

[cron-job.org](https://cron-job.org) is a free hosted cron service with a clean dashboard, per-fire history, and 1-min granularity on the free tier. Setup:

1. Sign up at [console.cron-job.org/signup](https://console.cron-job.org/signup).
2. Click **CREATE CRONJOB**.
3. **Common** tab:
   - **Title**: anything (e.g. `claude-code-routine`)
   - **URL**: paste the `/fire` URL from the Anthropic routine editor → *API trigger* → *URL* (looks like `https://api.anthropic.com/v1/claude_code/routines/trig_.../fire`)
   - **Schedule**: pick your timezone and the times to fire — cron-job.org accepts both UI selectors and raw cron syntax
4. **Advanced** tab → set **Request method** to `POST`.
5. **Headers** tab → add four headers:

   | Key                 | Value                                          |
   | ------------------- | ---------------------------------------------- |
   | `Authorization`     | `Bearer sk-ant-oat01-...` (your routine token) |
   | `anthropic-version` | `2023-06-01`                                   |
   | `anthropic-beta`    | `experimental-cc-routine-2026-04-01`           |
   | `Content-Type`      | `application/json`                             |

6. **Body** tab → select `Raw` and paste:
   ```json
   {"text": "Scheduled trigger"}
   ```
7. **Notifications** tab (optional) → enable email on failure so you know if the token expires.
8. **Save**.

Each fire shows up in *History* with status code and response body — open `claude_code_session_url` from the response JSON to watch the run.

**Operational notes:**
- **Token rotation**: edit the cronjob → swap the `Authorization` header value. No redeploy.
- **Beta header**: when Anthropic ships a new dated `anthropic-beta` value, update the header. Older dated values keep working for a transition window per Anthropic's beta policy.
- **No retry on failure**: each `/fire` POST creates a new Claude Code session, so retrying would multiply sessions and burn quota. cron-job.org's default is one attempt per fire, which is what you want.
- **Limits**: cron-job.org's free tier allows up to 50 cronjobs and unlimited executions at 1-min granularity — way more than enough for routine triggering.

## Why this vs the siblings

|                  | [claude-code-routine-trigger](https://github.com/tiennm99/claude-code-routine-trigger) | [claude-code-routine-cron](https://github.com/tiennm99/claude-code-routine-cron) | **claude-code-routine-trigger-worker** (this) |
| ---------------- | -------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | --------------------------------------------- |
| Runs on          | GitHub Actions runners                                                                 | your infra (Docker host, k8s, NAS, RPi)                                          | Cloudflare edge                               |
| Cost             | free (within GitHub minutes)                                                           | minimal (your infra)                                                             | free (CF free tier)                           |
| Cron precision   | ±30 min – 2 h, can drop runs                                                           | sub-second                                                                       | within ~15 sec                                |
| Setup            | fork + 2 repo secrets                                                                  | env vars + Docker                                                                | `wrangler deploy` + 2 secrets                 |
| Audit trail      | GitHub Actions runs page                                                               | container stdout                                                                 | CF dashboard / `wrangler tail`                |

## Using Cloudflare Workers

> The default `wrangler.toml` ships with the `[triggers]` block **commented out** — see the note in the file. To activate this worker, uncomment the block (and edit the schedule) before deploying.

### Quickstart

```bash
git clone https://github.com/tiennm99/claude-code-routine-trigger-worker
cd claude-code-routine-trigger-worker
npm install

# Upload secrets (from Anthropic routine editor → API trigger)
echo -n 'https://api.anthropic.com/v1/claude_code/routines/trig_.../fire' \
  | npx wrangler secret put ROUTINE_FIRE_URL
echo -n 'sk-ant-oat01-...' \
  | npx wrangler secret put ROUTINE_FIRE_TOKEN

# Uncomment [triggers].crons in wrangler.toml, edit the schedule + timezone, then:
npx wrangler deploy
```

Tail logs:

```bash
npx wrangler tail
```

A successful fire logs a JSON line with `session_url`. Open it to watch the run.

### Environment variables

Configured in `wrangler.toml` (`[vars]` for plain values) or via `wrangler secret put` (for secrets).

| Name                 | Type   | Required | Default                                  | Notes |
| -------------------- | ------ | :------: | ---------------------------------------- | ----- |
| `ROUTINE_FIRE_URL`   | secret | yes      | —                                        | Anthropic `/fire` endpoint. From routine editor → API trigger. |
| `ROUTINE_FIRE_TOKEN` | secret | yes      | —                                        | `sk-ant-oat01-...` per-routine token. Shown once in the editor. |
| `TEXT_TEMPLATE`      | var    | no       | `Scheduled trigger at {LocalTime}`       | Token-substitution template. See *Templates*. |
| `TZ`                 | var    | no       | `UTC`                                    | IANA tz name (`Asia/Ho_Chi_Minh`, `America/New_York`, …). Used for `{LocalTime}` formatting. |

### Customize the schedule

Uncomment and edit `wrangler.toml` `[triggers].crons` — Cloudflare requires literal cron expressions (same constraint as GitHub Actions). To change when the routine fires:

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

### Templates

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

### Local development

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

### Tests

```bash
npm test
```

Vitest runs in the Workers runtime via `@cloudflare/vitest-pool-workers`. Tests mock `fetch`, so they consume no Anthropic quota.

### Secret rotation

`wrangler secret put` overwrites silently. Rotate by re-running it with the new value:

```bash
echo -n 'sk-ant-oat01-newvalue' | npx wrangler secret put ROUTINE_FIRE_TOKEN
```

The change takes effect on the next deploy or within a few seconds via the live config.

### Beta header

The request pins `anthropic-beta: experimental-cc-routine-2026-04-01` (constant in `worker.js`). When Anthropic ships a new dated beta, bump it via a release. Older dated values keep working for a transition window per Anthropic's beta policy.

### Operational notes

- **Time accuracy**: cron precision on CF Workers is within ~15 seconds — adequate for routine triggering.
- **No retry**: each `/fire` POST creates a new Claude Code session — retrying multiplies sessions and burns quota. The worker logs failures and moves on.
- **Logs / traces**: `[observability]` is enabled in `wrangler.toml` with `head_sampling_rate = 1` and `invocation_logs = true` — every invocation produces a structured log + trace, retained per the [Workers Logs retention policy](https://developers.cloudflare.com/workers/observability/logs/workers-logs/) (3 days on the free plan). View live with `npx wrangler tail`, or browse history in the CF dashboard under **Workers & Pages → your worker → Logs**.
- **Cost**: scheduled handlers count against the Workers Free plan's 100k requests/day budget. 5 daily fires × 30 days = 150 requests/month — negligible.

### Security

- The token is **per-routine**: a leak only fires that one routine.
- Secrets live in CF's encrypted secret store, never in the bundle, never in logs (verified by tests).
- TLS to `api.anthropic.com` uses CF's standard cert verification.
- `worker.js` has no `fetch` HTTP handler — the worker is unreachable from the public internet, only fires on cron tick.

## License

[Apache-2.0](./LICENSE)
