# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-05-09

### Added
- Cloudflare Workers `scheduled` handler that POSTs to the Claude Code routine `/fire` endpoint.
- Default 5×daily cron at UTC+7 (00:07 / 05:13 / 10:19 / 15:23 / 20:37) — scattered minutes for shared-infra hygiene.
- Token-substitution template with `{ISO}`, `{LocalTime}`, `{Cron}` (default: `Scheduled trigger at {LocalTime}`).
- IANA timezone support via `TZ` env var (default: `UTC`).
- Structured JSON logs for fire success / non-2xx / network failure.
- Vitest test suite covering template substitution, fire paths, header correctness, token redaction.
- GitHub Actions CI: `npm test` + `wrangler deploy --dry-run` on push and PR.
- Apache-2.0 license.
