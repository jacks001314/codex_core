# Security policy

## Defaults

`core.New` has no tools unless explicitly supplied. `ToolsWorkspace` requires an approval handler. Full unrestricted execution is not exposed as a default preset.

## Credentials

Do not include API keys, bearer tokens, or auth snapshots in logs, events, errors, examples, or session metadata. Use environment variables or the auth store. Live tests are opt-in and must never be enabled in CI.

## Filesystem

Patch paths are relative to the configured workspace. Absolute paths, traversal paths, and existing symlink escapes are rejected. Use read-only tools when a workspace is not trusted.

## Reporting

Report suspected vulnerabilities privately to the project maintainers. Include reproduction steps without real credentials or private workspace content.
