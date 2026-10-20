# codex_core

`codex_core` is an embeddable Go implementation of the Codex runtime extracted from `codex_go`. It exposes model execution, tool loops, safe command policies, configuration/authentication, streaming events, and session persistence without a TUI, CLI parser, app-server, or daemon.

## Basic use

```go
client, err := core.NewFromConfig(ctx, &core.ConfigOptions{})
result, err := client.Run(ctx, &core.Request{Prompt: "Summarize this repository."})
```

Authentication precedence for `NewFromConfig` is:

1. `ConfigOptions.Auth`
2. `ConfigOptions.APIKey`
3. `OPENAI_API_KEY`, `CODEX_API_KEY`, or `CODEX_ACCESS_TOKEN`
4. `auth.json` under `CodexHome`

Explicit provider/model values override loaded TOML configuration.

## Safe tools

The default client has no local tools. Tool execution must be explicitly enabled:

- `ToolsNone`: no built-in local tools.
- `ToolsReadOnly`: core handlers and sandboxed read-only shell access. Escalation is not automatically approved.
- `ToolsWorkspace`: workspace shell and apply-patch; requires an `ApprovalHandler`.
- Custom executors can be supplied independently.

```go
Tools: &core.ToolOptions{
    Preset: core.ToolsWorkspace,
    CWD: workspace,
    Approval: handler,
}
```

Never implement an approval handler that blindly approves model-generated commands in an untrusted workspace.

## Events

`Client.Stream` emits:

- `started`
- `model` for Responses API stream events, including text/reasoning deltas and usage
- `tool.started`
- `tool.completed`
- `completed`
- `failed`

Events are delivered through a channel and never written to stdout/stderr by the library.

## Sessions

Set `Persist: true` in `ConfigOptions` or inject a `session.Store` through `core.Options`. `Run` creates IDs when absent and records prompts, responses, tool outputs, model/provider metadata, and the last response ID. Use `ResumeThread`, `ForkThread`, `ListThreads`, `ArchiveThread`, and `DeleteThread` for lifecycle operations.

## Examples

- `examples/basic`
- `examples/streaming`
- `examples/structured_output`
- `examples/custom_tool`
- `examples/workspace_agent`
- `examples/session_resume`
- `examples/session_lifecycle`
- `examples/event_sink` (offline; shows event delivery without a model endpoint)

## Validation

```powershell
go list -buildvcs=false ./...
go test ./...
go test -race ./...
go vet ./...
```
