---
phase: 1
title: Repo scaffold
status: completed
priority: P1
effort: 1h
dependencies: []
---

# Phase 1: Repo scaffold

## Overview
Greenfield repo init. Apache-2.0 license, README skeleton, JS + wrangler toolchain, gitignore. No transpile step — Workers runtime executes ES modules natively.

## Requirements
- **Functional:** `npm install` succeeds; `wrangler deploy --dry-run` validates an empty `worker.js` handler.
- **Non-functional:** No secrets in repo. Pin wrangler major version. kebab-case file names.

## Architecture
Flat layout — single-purpose worker, no need for src/ subdirs (KISS).

```
claude-code-routine-trigger-worker/
├── .github/workflows/ci.yml      # phase 4
├── .gitignore
├── LICENSE                       # Apache-2.0
├── README.md                     # phase 5
├── package.json
├── wrangler.toml                 # phase 3
├── worker.js                     # phase 2
└── worker.test.js                # phase 4
```

## Related Code Files
- Create: `LICENSE` (Apache-2.0 verbatim)
- Create: `.gitignore` (node_modules, .dev.vars, .wrangler, dist, .env*)
- Create: `package.json` (name, version 0.0.0, `"type": "module"`, scripts: test, deploy, dev)
- Create: `README.md` (one-liner placeholder; full content in phase 5)

## Implementation Steps
1. `cd /config/workspace/tiennm99/claude-code-routine-trigger-worker && git init -b main`
2. Write `LICENSE` (copy Apache-2.0 from sibling `claude-code-routine-cron/LICENSE`).
3. Write `.gitignore`:
   ```
   node_modules/
   dist/
   .wrangler/
   .dev.vars
   .env
   .env.*
   *.log
   ```
4. Write `package.json` with `"type": "module"`, devDeps: `wrangler` (^4), `vitest` (phase 4), `@cloudflare/vitest-pool-workers` (phase 4).
   Scripts:
   - `dev`: `wrangler dev`
   - `deploy`: `wrangler deploy`
   - `test`: `vitest run`
5. Run `npm install` — confirm lockfile generated.
6. Stub `README.md` with one-liner: `> Cloudflare Workers cron port of claude-code-routine-cron — fires Claude Code routines on a schedule.`
7. First commit: `chore: init repo scaffold`.

## Success Criteria
- [ ] `npm install` exits 0
- [ ] `wrangler deploy --dry-run` validates empty `worker.js` exporting `export default {}`
- [ ] `git status` clean after first commit
- [ ] LICENSE file matches Apache-2.0 exactly
- [ ] `package.json` has `"type": "module"`

## Risk Assessment
- **Wrangler version drift:** pin to current major (e.g. `^4.0.0`); document upgrade path in README.
- **Node version:** wrangler 4 needs Node ≥18; document in README prereqs.
- **No type safety:** mitigate by JSDoc type hints in `worker.js` for `Env` and handler signatures (cheap, optional, no toolchain).
