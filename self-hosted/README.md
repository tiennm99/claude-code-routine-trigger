# claude-code-routine-cron

Self-hosted Go daemon that fires a [Claude Code routine](https://code.claude.com/docs/en/routines) on a precise cron schedule via the [`/fire` API](https://platform.claude.com/docs/en/api/claude-code/routines-fire). Packaged as a multi-arch Docker image on GHCR.

> [!TIP]
> **Anthropic's routine editor now ships a built-in cron trigger** — that runs on Anthropic's infra, no setup. Use it first.
>
> This repo is for users who explicitly want **self-hosted scheduling**: precise timing (no GitHub Actions delays), behind-firewall, on-prem audit trail, or to integrate with their own infra.

## Why this vs `claude-code-routine-trigger`

|                  | [claude-code-routine-trigger](https://github.com/tiennm99/claude-code-routine-trigger) | **claude-code-routine-cron** (this repo) |
| ---------------- | -------------------------------------------------------------------------------------- | ---------------------------------------- |
| Runs on          | GitHub Actions runners                                                                 | your infra (Docker host, k8s, NAS, RPi)  |
| Cost             | free (within GitHub minutes)                                                           | minimal (your infra)                     |
| Cron precision   | ±30 min – 2 h, can drop runs                                                           | sub-second                               |
| Setup            | fork + 2 repo secrets                                                                  | 2 env vars + cron + Docker               |
| Secret storage   | GitHub repo secrets                                                                    | host env / Docker secrets / k8s Secret   |
| Audit trail      | GitHub Actions runs page                                                               | container stdout                         |

## Quickstart

```bash
docker run -d --name claude-routine \
  --restart unless-stopped \
  -e ROUTINE_FIRE_URL='https://api.anthropic.com/v1/claude_code/routines/trig_.../fire' \
  -e ROUTINE_FIRE_TOKEN='sk-ant-oat01-...' \
  -e CRON_SCHEDULE='0 9 * * *;0 18 * * *' \
  -e TZ='Asia/Ho_Chi_Minh' \
  ghcr.io/tiennm99/claude-code-routine-cron:latest
```

Tail logs:

```bash
docker logs -f claude-routine
```

A successful fire logs a JSON line with `claude_code_session_url`. Open it to watch the run.

## Environment variables

| Name                 | Required | Default                                  | Notes |
| -------------------- | :------: | ---------------------------------------- | ----- |
| `ROUTINE_FIRE_URL`   | yes      | —                                        | Anthropic `/fire` endpoint. From routine editor → API trigger. |
| `ROUTINE_FIRE_TOKEN` | yes      | —                                        | `sk-ant-oat01-...` per-routine token. Shown once in the editor. |
| `CRON_SCHEDULE`      | yes      | —                                        | One or more standard 5-field cron expressions. Split on `;` or newlines. |
| `TZ`                 | no       | `UTC`                                    | IANA tz name (`Asia/Ho_Chi_Minh`, `America/New_York`, …). Cron evaluates in this zone. |
| `TEXT_TEMPLATE`      | no       | `Scheduled trigger at {{.LocalTime}}`    | Go [text/template](https://pkg.go.dev/text/template); see *Templates*. |
| `LOG_LEVEL`          | no       | `info`                                   | `debug`, `info`, `warn`, `error`. |

Validation is fail-fast: missing required vars or malformed crons/timezones/templates cause the daemon to exit with a clear error before any HTTP traffic.

## Multiple schedules

Use `;` (or newlines, in YAML block scalars) to register multiple crons:

```yaml
environment:
  CRON_SCHEDULE: |
    0 9 * * *
    0 13 * * *
    0 18 * * *
```

Each schedule fires independently; all use the same `TEXT_TEMPLATE`, URL, and token.

## Templates

The `TEXT_TEMPLATE` env var is rendered as a Go `text/template` per fire with these variables:

| Var          | Type        | Example                          |
| ------------ | ----------- | -------------------------------- |
| `.Now`       | `time.Time` | UTC time of the fire             |
| `.LocalTime` | `string`    | `2026-05-08 23:30 +07`           |
| `.Cron`      | `string`    | `0 9 * * *` — the expression that fired |

Example:

```bash
-e TEXT_TEMPLATE='Daily digest at {{.LocalTime}} (cron {{.Cron}})'
```

## docker-compose

See [`docker-compose.example.yml`](./docker-compose.example.yml) and [`.env.example`](./.env.example):

```bash
cp docker-compose.example.yml docker-compose.yml
cp .env.example .env       # then edit secrets
docker compose up -d
```

## Security

- The token is **per-routine**: a leak only fires that one routine.
- Token never appears in logs (verified by tests).
- No retry on failure — each `/fire` POST creates a new session, retries would multiply sessions and burn quota.
- Image is `gcr.io/distroless/static-debian12:nonroot` — no shell, no package manager, runs as UID 65532.
- TLS to `api.anthropic.com` uses standard Go cert verification.

## Beta header

The request pins `anthropic-beta: experimental-cc-routine-2026-04-01`. When Anthropic ships a new dated beta, bump it via a new release. Older dated values keep working for a transition window per Anthropic's beta policy.

## Operational notes

- **Time accuracy**: cron correctness depends on the host clock. Run NTP / sync on the Docker host.
- **Restart policy**: use `--restart unless-stopped` (or k8s `Deployment`) — the daemon does not self-restart on panic.
- **Pin tags in production**: prefer `:vX.Y.Z` over `:latest`.
- **No idempotency**: avoid retry loops in upstream automation that POSTs the same alert twice.

## Build from source

```bash
git clone https://github.com/tiennm99/claude-code-routine-cron
cd claude-code-routine-cron
go build .
./claude-code-routine-cron     # uses env vars from your shell
```

Tests:

```bash
go test -race -cover ./...
```

## License

[Apache-2.0](./LICENSE)
