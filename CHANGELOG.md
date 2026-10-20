# Changelog

## 0.1.0 - 2026-07-18

- Added embeddable `core.Client` and `NewFromConfig`.
- Added Responses model streaming events and multi-turn tool execution.
- Added safe read-only/workspace tool presets with injected approvals.
- Added session create/read/resume/fork/list/archive/delete APIs.
- Added opt-in live integration testing, examples, CI checks, and security policy.
- Added symlink-escape protection for workspace patches and serialized session persistence.

Live API integration tests are opt-in with `CODEX_LIVE_TEST=1`.
