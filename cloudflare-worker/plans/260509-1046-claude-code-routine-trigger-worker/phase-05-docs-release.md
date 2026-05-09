---
phase: 5
title: Docs + release
status: completed
priority: P2
effort: 1.5h
dependencies:
  - 1
  - 2
  - 3
  - 4
---

# Phase 5: Docs + release

## Overview
Full README mirroring sibling repos' tone, comparison-table back-link in both siblings, v0.1.0 GitHub release, public deployment of the author's own instance.

## Requirements
- **Functional:**
  - README covers: why-this-vs-siblings, quickstart, env vars, secret upload, customize schedule, beta header policy, security, license.
  - GitHub release `v0.1.0` tagged.
  - Both sibling READMEs updated with a row pointing to this repo.
- **Non-functional:**
  - Same prose voice as siblings (`> [!TIP]` / `> [!WARNING]` callouts, terse).
  - Code blocks copy-pasteable.
  - All claims about CF cron precision sourced (link CF docs).

## Architecture

### README sections (in order)
1. **Header + tagline** — one-liner.
2. **Why this vs siblings** — three-column comparison table.
3. **Quickstart** — `wrangler deploy` + 2 secret-puts + done.
4. **Environment variables** — table (name, required, default, notes).
5. **Multiple schedules** — `wrangler.toml` `[triggers].crons` array.
6. **Templates** — `{LocalTime}`, `{Cron}`, `{ISO}`.
7. **Customize schedule** — edit `wrangler.toml`, `wrangler deploy`.
8. **Local development** — `.dev.vars` + `wrangler dev --test-scheduled`.
9. **Secret rotation** — `wrangler secret put` overwrites.
10. **CF Workers free tier** — link, note 5-cron-per-worker limit.
11. **Beta header** — pinned value, bump policy.
12. **Operational notes** — log access (`wrangler tail`), no idempotency.
13. **License** — Apache-2.0.

### Sibling repo updates
- Edit `claude-code-routine-trigger/README.md` — add row in the "no longer used by author" callout listing `trigger-worker` as another option.
- Edit `claude-code-routine-cron/README.md` — extend "Why this vs `claude-code-routine-trigger`" table to a 3-way comparison including `trigger-worker`.

## Related Code Files
- Modify: `README.md` (this repo)
- Modify (out of repo, requires PR/commit): `/config/workspace/tiennm99/claude-code-routine-trigger/README.md`
- Modify (out of repo): `/config/workspace/tiennm99/claude-code-routine-cron/README.md`
- Create: `CHANGELOG.md` (Keep-A-Changelog format)
- Optional: `.github/release.yml` for release-notes automation

## Implementation Steps
1. Draft README per architecture sections — use sibling READMEs verbatim where applicable (Apache-2.0 license clauses, beta header note, security note about per-routine token scope).
2. Write `CHANGELOG.md` with `## [0.1.0] - 2026-MM-DD` initial entry listing features.
3. Local proof: deploy author's instance via `wrangler deploy`, attach a non-prod routine, observe one cron tick, capture `session_url` for README screenshot.
4. Tag and release:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   gh release create v0.1.0 --generate-notes
   ```
5. Open PRs (or direct commits if author owns) to update sibling READMEs with the comparison row.
6. Final commit: `docs: README + v0.1.0 release notes`.

## Success Criteria
- [ ] README renders cleanly on GitHub
- [ ] All code blocks in README are runnable (copy-paste tested)
- [ ] Comparison table accurately reflects all 3 options
- [ ] Both sibling READMEs reference this repo
- [ ] `v0.1.0` tag pushed and GitHub release published
- [ ] Author's own deployment fired at least one successful cron tick (proof-of-life)

## Risk Assessment
- **Beta header drift:** if Anthropic ships new dated beta between phase 2 and release, bump in code + CHANGELOG before tagging.
- **Sibling repo edits cause merge conflicts:** branch + PR if user has unmerged work; otherwise direct commit.
- **CF dashboard URL format changes:** don't hardcode dashboard URLs in README; use `wrangler tail` (CLI) which is stable.

## Security Considerations
- README must NOT include real `ROUTINE_FIRE_URL` or token — only `sk-ant-oat01-...` placeholder format.
- Document that pushing to a public fork preserves `.dev.vars` exclusion (gitignore line).

## Next Steps (post v0.1.0)
- v0.2: optional `WEBHOOK_URL` to mirror fire output to Slack/Discord (low priority — most users use `wrangler tail`).
- v0.3: `wrangler.toml.example` with multi-routine deployment helper script.
