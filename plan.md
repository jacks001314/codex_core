# codex_core Development Plan

## 1. Goal

Extract the reusable Codex runtime from `D:\qax\reagent\dev\codex_go` into an independently buildable Go module. Applications must call it directly through Go APIs and receive structured results, errors, and streaming events.

This module will not include interactive UI, TUI, CLI command parsing, HTTP/JSON-RPC app-server, daemon, remote-control, or any other service process.

## 2. Reference analysis

The reference project's reusable core is centered on:

- `model`: Agent request/response, Responses API transport, streaming, providers, model catalog.
- `turn`: Agent loop, tool-call iterations, cancellation, max iterations, timing.
- `tool`: tool schema, registry, router, parallel dispatch, built-in executors.
- `config`, `auth`, `sandbox`: configuration, credentials, provider auth, permissions, execution policy.
- `session`, `rollout`, `context`, `compact`: persistence, recovery, context construction, compaction.
- `mcp`, `plugin`, `agent`: optional extensions.

`exec.Runner` must not become the public API: it is coupled to `cli`, stdin/stdout, terminal event rendering, and CLI configuration assembly. Its reusable orchestration logic will be reassembled behind a library facade.

## 3. Target architecture

```text
core/       Stable embedded facade: Client, Request, Result, Event
model/      Models, providers, Responses API, stream parsing
turn/       Agent Loop and Runtime
tool/       Tool protocol, registry, router, executors
auth/       Credentials and authentication
config/     Config, profiles, overrides, features
sandbox/    Permission policies and platform execution
session/    Session/history persistence
rollout/    JSONL rollout recording
context/    Context/history building
compact/    Token budget and compaction
protocol/   Library-facing structured events
docs/       API, migration, and security documentation
```

`core` is the recommended dependency for applications. Lower-level packages must remain testable and replaceable. HTTP, storage, approval, clock, and event delivery are injected through interfaces/options; the runtime never writes UI output itself.

## 4. Phased implementation

### Phase 0 - Baseline and dependency audit

1. Establish module conventions, package layout, README, license, and changelog.
2. Inventory reference files, tests, build tags, and third-party dependencies.
3. Maintain a migration matrix: source file, target file, migrate/adapt/exclude decision, rationale, behavior gap.
4. Establish CI checks: `go list -buildvcs=false ./...`, `go test ./...`, and `go vet ./...`.

**Acceptance:** The empty module builds and tests independently; it has no UI or service dependencies.

### Phase 1 - Minimum agent loop (P0)

1. Define `core.Client`, `Options`, `Request`, `Result`, `Event`, usage, and stable error codes.
2. Migrate `model.AgentRunner`, agent input/output types, Responses API transport, and streaming parsing.
3. Migrate `turn.AgentLoop` and `turn.Runtime`: model sampling, tool-loop continuation, max turns, context cancellation, and timing.
4. Build fake-agent tests before wiring real HTTP Responses calls.
5. Expose both blocking execution and channel/callback streaming without stdin/stdout.

**Acceptance:** A host application can perform `prompt -> model response` using only Go APIs; fake and HTTP-mocked providers both pass tests.

### Phase 2 - Tools and controlled execution (P0)

1. Migrate `tool.Registry`, `Executor`, `Router`, schema, namespaces, visibility, and lifecycle events.
2. Let hosts register custom tools with parallelism declarations.
3. Migrate the minimum built-ins: shell, apply_patch, read/search files.
4. Replace terminal approvals with an injected `ApprovalHandler`.
5. Migrate permission profiles and command-run plans for unrestricted, read-only, and workspace-write modes.
6. Standardize timeout, cancellation, exit-code, output-truncation, concurrency, and panic handling.

**Acceptance:** Tools participate in the Agent Loop; unapproved high-risk calls are not run; policy and platform behavior are tested.

### Phase 3 - Config, auth, and providers (P0)

1. Migrate TOML loading, profile selection, project config, and key/value overrides.
2. Support in-memory configuration; explicit `Options` override environment and file configuration.
3. Migrate API key, ChatGPT/OAuth tokens, secret-store abstraction, and provider auth command support.
4. Migrate provider parsing, OpenAI/Codex Responses API, compatible providers, and model catalog behavior.
5. Support injected HTTP client, base URL, retry policy, stream idle timeout, and request metadata.
6. Classify configuration, auth, network, protocol, tool, and cancellation errors while preserving `errors.Is/As`.

**Acceptance:** Explicit provider/API-key operation works; missing credentials, invalid config, unknown provider, and network failures are distinguishable.

### Phase 4 - Sessions, rollout, and recovery (P1)

1. Migrate session create/read/append/update/archive/delete/fork/list operations.
2. Migrate rollout JSONL recording for session metadata, items, turn status, errors, and compaction.
3. Map model responses and tool input/output to common history items.
4. Support resume/fork with prior response ID and restored context.
5. Make persistence optional so pure in-memory runs remain possible.
6. Test file permissions, atomic writes, corrupt records, and concurrent append behavior.

### Phase 5 - Context and compaction (P1)

1. Migrate non-UI portions of `context`, `prompt`, and `compact`.
2. Provide a replaceable token-budget estimator.
3. Implement threshold-triggered compaction, fallback behavior, and retention of recent user input and critical tool results.
4. Layer permissions, environment context, user instructions, developer instructions, and host-supplied context.
5. Treat model, MCP, plugin, file, and shell content as untrusted data.

### Phase 6 - Optional extensions (P2)

Migrate MCP client/runtime, plugins, subagents, web search, image generation, view-image, memory, and telemetry as independently enabled extensions. None may break the baseline text-and-tool runtime if unavailable.

## 5. Public API draft

```go
type Client struct{}

func New(options *Options) (*Client, error)
func (c *Client) Run(ctx context.Context, request *Request) (*Result, error)
func (c *Client) Stream(ctx context.Context, request *Request) (<-chan Event, error)
func (c *Client) Close() error

type Options struct {
    Config     *Config
    CodexHome  string
    Agent      model.AgentRunner
    Tools      []tool.Executor
    Approval   ApprovalHandler
    EventSink  EventSink
    HTTPClient model.HTTPDoer
    MaxTurns   int
    Persist    bool
}

type Request struct {
    Prompt             string
    InputItems         []any
    Instructions       string
    Model              string
    ProviderID         string
    ThreadID           string
    TurnID             string
    OutputSchema       any
    PreviousResponseID string
}
```

Freeze fields only after migrated types are validated. Do not expose reference-project app-server or CLI transport types as part of the stable facade.

## 6. Testing and quality

- Migrate relevant reference tests first, then add facade-level integration tests.
- Use fake Agent, fake clock, fake approval, fake filesystem, and `httptest.Server`; unit tests must not use real network, credentials, or user home.
- Cover cancellation/deadline, max turns, duplicate tools, parallel failure, output truncation, invalid JSON, stream disconnects, missing auth, and corrupt sessions.
- Build and test platform-specific files on Windows, Linux, and macOS.
- Close each phase with `gofmt -w .`, `go list -buildvcs=false ./...`, `go test ./...`, `go test -race ./...`, and `go vet ./...`.

## 7. Security and compatibility

- Never emit credentials in logs, ordinary errors, or events.
- Default tool execution to least privilege; approval is a host callback, never a terminal fallback.
- Validate paths, commands, environment values, URLs, schemas, and dynamic tool names.
- Keep new event fields optional and use stable error codes.
- Use behavior tests to lock compatibility with `codex_go`; do not chase every CLI-only feature.

## 8. Milestones and immediate next step

- **M0:** Architecture, dependency audit, test baseline.
- **M1:** `model + turn + core` minimum loop without tools.
- **M2:** Tool registry/router, approval, and sandbox.
- **M3:** Config, auth, provider, HTTP error handling.
- **M4:** Session, rollout, recovery, compaction.
- **M5:** MCP, plugins, agents, hosted tools.

Start with M0: create Phase-0 package/test scaffolding; verify `model.AgentRunner`, `turn.AgentLoop`, and `tool.Registry/Router` contracts; implement a mock-only `core.Client` loop; maintain the migration matrix for every copied or excluded reference file.

## 9. Production hardening work packages

### P0-A - Configured client construction

- Add a high-level constructor that loads optional TOML/profile overrides or accepts an in-memory provider definition.
- Resolve credentials without requiring callers to manually construct `model.AgentRunner`.
- Build `model.RuntimeProvider`, `model.ResponsesAgentRunner`, HTTP transport, model defaults, and provider metadata internally.
- Preserve dependency injection: an explicit Agent or HTTP client always overrides automatic construction.

### P0-B - Safe tool presets

- Provide no-tools, read-only, workspace, and custom-only presets.
- Default to no shell execution; workspace mutation requires an explicit preset and approval callback/policy.
- Configure command timeout, output token limit, workspace root, sandbox permission profile, and unified exec lifecycle.
- Never silently downgrade an unsupported sandbox to unrestricted execution.

### P0-C - Structured streaming events

- Forward model stream events, tool start/completion, usage, rate-limit metadata, final result, and errors through the `core` event API.
- Keep event delivery independent from stdout/stderr and define backpressure/cancellation behavior.
- Preserve raw model stream events for advanced callers while exposing normalized event types.

### P1-A - Session API

- Connect `session.Store` to `core.Client` with create/resume/fork/list/archive/delete operations.
- Persist model input/output, tool calls/results, response IDs, model/provider metadata, and timestamps.
- Allow persistence to be disabled or replaced by a host-provided store.

### P1-B - Public API stabilization

- Keep applications on `core` types rather than `turn` or app-server transport structs.
- Add stable error codes, lifecycle-safe `Close`, concurrency documentation, and option validation.
- Keep MCP, plugins, agents, telemetry, and hosted tools opt-in.

### P1-C - Documentation and examples

- Add basic, streaming, custom-tool, workspace-agent, and session-resume examples.
- Document authentication precedence, safe tool defaults, event semantics, and migration from direct lower-level package use.

### Completion criteria

- All examples compile.
- `go mod tidy` leaves no accidental UI/app-server dependency.
- `go list -buildvcs=false ./...`, `go test ./...`, `go test -race ./...`, and `go vet ./...` pass.

## 10. Release-readiness status

Completed:

- Live API integration test is available under `integration/` and is opt-in via `CODEX_LIVE_TEST=1`.
- `scripts/integration-test.ps1` provides a repeatable local integration command.
- Session persistence is serialized per client; patch paths reject traversal and existing symlink escapes.
- Public API compatibility, security policy, changelog, and cross-platform GitHub Actions workflow are documented.
- Examples compile and the dependency graph is normalized with `go mod tidy`.

Before tagging a release:

1. Run the live integration test with non-production test credentials.
2. Review tool approval behavior on a disposable workspace.
3. Confirm Windows/Linux/macOS CI is green.
4. Decide whether lower-level copied packages should be marked internal or retained as supported extension points.
5. Tag the first release only after the API compatibility document is reviewed.
