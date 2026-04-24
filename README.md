# claude-code-routine-trigger

> [!IMPORTANT]
> **No longer used by the author.** Claude Code routines now ship with a **built-in cron trigger** in the routine editor — that's what I use now. Prefer the native scheduler; it runs on Anthropic's infra (no GitHub Actions delay/drop issues), needs no repo, no secrets, no workflow file.
>
> This repo is kept as a reference for anyone who still wants to fire routines from an external scheduler (GitHub Actions, own cron, Cloud Scheduler, etc.).

Scheduled GitHub Actions workflow that fires a [Claude Code routine](https://code.claude.com/docs/en/routines) via the [`/fire` API](https://platform.claude.com/docs/en/api/claude-code/routines-fire). Ships with a 5×-daily default cron and exposes a `workflow_dispatch` button for on-demand runs. Fire URL and per-routine token live in repo secrets.

**Why:** GitHub cron is the cheapest way to run a Claude Code routine unattended on a schedule — no servers, no extra scheduler, no billable infra.

Default schedule (UTC+7, `Asia/Ho_Chi_Minh`):

| Local time | UTC cron      |
| ---------- | ------------- |
| 00:00      | `0 17 * * *`  |
| 05:00      | `0 22 * * *`  |
| 10:00      | `0 3 * * *`   |
| 15:00      | `0 8 * * *`   |
| 20:00      | `0 13 * * *`  |

Manual `workflow_dispatch` is supported with an optional `text` input for ad-hoc runs. Customize the cron list to match your own cadence — see below.

## Customize the schedule

GitHub Actions requires the cron list to be **literal** in the workflow file — it cannot be read from secrets, repo variables, or inputs. To change when the routine fires, edit `.github/workflows/trigger-routine.yml` and update the `on.schedule` block:

```yaml
on:
  schedule:
    - cron: '0 17 * * *' # 00:00 UTC+7
    # add / remove / edit these lines as needed
```

Tips:
- Cron runs in **UTC**. Convert your local time: `UTC = local − offset` (e.g. 09:00 UTC+7 → 02:00 UTC → `0 2 * * *`).
- Validate expressions at <https://crontab.guru/>.
- Schedules only activate on the default branch after the file is pushed.
- **Avoid minute `0`.** Pick scattered minutes like `7`, `13`, `23`, `37`, `47` — see the warning below.

> [!WARNING]
> **GitHub cron delays can be severe at minute `0`.** Top-of-the-hour is the most contested slot on GitHub's shared runners — scheduled jobs there are commonly delayed by 30 min – 2+ hours, and during high load GitHub may **drop the run entirely** (see [Actions docs](https://docs.github.com/en/actions/writing-workflows/choosing-when-your-workflow-runs/events-that-trigger-workflows#schedule)).
>
> Observed in this repo: a `0 13 * * *` cron fired at **14:46 UTC — 1h46m late** ([run 24895643143](https://github.com/tiennm99/claude-code-routine-trigger/actions/runs/24895643143)).
>
> The default schedule above still uses minute `0` for readability. If you care about timing, rewrite them — e.g. `7 17 * * *`, `13 22 * * *`, `19 3 * * *`, `23 8 * * *`, `37 13 * * *`. Do **not** rely on GitHub cron for precise timing; if you need it, trigger `/fire` from a real scheduler (own cron, Cloud Scheduler, etc.).

## Setup

1. Create a routine at <https://claude.ai/code/routines> (requires a Pro/Max/Team/Enterprise plan with Claude Code on the web enabled).
2. In the routine editor, add an **API** trigger and generate a token. Copy both the fire URL and the token — the token is shown once.
3. Add two repository secrets (`Settings → Secrets and variables → Actions`):
   - `ROUTINE_FIRE_URL` — e.g. `https://api.anthropic.com/v1/claude_code/routines/trig_01ABo4hmfydBLFDgRMnKwEKy/fire`
   - `ROUTINE_FIRE_TOKEN` — e.g. `sk-ant-oat01-...`
4. Enable Actions for the repo. Scheduled runs start on the next matching UTC tick.

## Manual run

`Actions → Trigger Claude Code Routine → Run workflow`. Leave `text` blank to get a timestamped default, or pass custom context (alert body, log line, etc.).

## Request shape

```bash
curl -X POST "$ROUTINE_FIRE_URL" \
  -H "Authorization: Bearer $ROUTINE_FIRE_TOKEN" \
  -H "anthropic-version: 2023-06-01" \
  -H "anthropic-beta: experimental-cc-routine-2026-04-01" \
  -H "Content-Type: application/json" \
  -d '{"text": "Scheduled trigger ..."}'
```

Beta header pinned to `experimental-cc-routine-2026-04-01`. Two previous dated beta versions keep working while migrating — bump when Anthropic ships a new one.

## Notes

- GitHub cron is best-effort: delays of 30+ minutes are common at minute `0`, and runs can be silently skipped under load. Acceptable for housekeeping-style routines, not for precise timing — see the warning in *Customize the schedule*.
- Token is scoped to a single routine; a leak can only fire that one routine.
- Each POST creates a new session (no idempotency). Avoid retry loops that would multiply sessions.
- 429 responses include `Retry-After`; the workflow fails loud rather than retrying silently.

## Use cases

Anything a Claude Code routine already does, on a clock:
- Daily repo triage (stale issues, dependabot PRs, unreviewed drafts).
- Morning / evening digests written to a file in the routine's attached repo.
- Scheduled codebase audits (unused exports, TODO sweep, lint drift).
- Periodic doc freshness checks.

For event-driven triggers (CI failure, webhook, alert) call `/fire` from the producing system directly — a cron scheduler isn't the right shape.

## License

Apache-2.0 — see [LICENSE](./LICENSE).
