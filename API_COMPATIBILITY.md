# Changelog

## 0.1.0 - Unreleased

- Added embeddable `core.Client` and `NewFromConfig`.
- Added model/tool streaming events and multi-turn tool execution.
- Added safe read-only/workspace tool presets with injected approvals.
- Added session create/read/resume/fork/list/archive/delete APIs.
- Added opt-in live integration test, examples, CI checks, and security policy.
  ''@ | Set-Content -Encoding UTF8 .\CHANGELOG.md
  @'

# API compatibility

The supported application-facing API is the `core` package. Lower-level packages (`model`, `turn`, `tool`, `auth`, `session`, and others) are implementation building blocks and may evolve independently.

For the `core` package:

- Existing exported types and fields are kept source-compatible within the 0.x series when practical.
- New event types and optional fields may be added.
- Existing event meanings and error codes are not changed silently.
- `Close` is idempotent; calls after close return `core.ErrClosed`.
- Context cancellation takes precedence over new work.
- Session JSON files are treated as a versioned persistence format, not a wire protocol.

The first stable release should start after the live integration flow and security review pass.
