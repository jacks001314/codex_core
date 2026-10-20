package core

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"codex_core/sandbox"
	"codex_core/tool"
	"codex_core/turn"
)

type ToolPreset string

const (
	ToolsNone      ToolPreset = "none"
	ToolsReadOnly  ToolPreset = "read-only"
	ToolsWorkspace ToolPreset = "workspace"
)

type ApprovalHandler interface {
	ApproveShell(context.Context, *tool.ShellApprovalRequest) (tool.ShellApprovalDecision, error)
	ApprovePatch(context.Context, *tool.ApplyPatchApprovalRequest) (tool.ApplyPatchApprovalDecision, error)
}

type ToolOptions struct {
	Preset           ToolPreset
	CWD              string
	Approval         ApprovalHandler
	DefaultTimeoutMS uint64
	MaxOutputTokens  *int
	Additional       []tool.Executor
}

func BuildToolRouter(options *ToolOptions) (*tool.Router, error) {
	if options == nil || options.Preset == "" || options.Preset == ToolsNone {
		return routerFromExecutors(nil, optionalExecutors(options)...)
	}
	cwd := strings.TrimSpace(options.CWD)
	if cwd == "" {
		cwd = "."
	}
	absolute, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	cwd = filepath.Clean(absolute)
	registryOptions := &turn.ToolRegistryOptions{
		EnableCore:       true,
		EnableToolSearch: true,
		EnableMCP:        false,
		EnableAgents:     false,
		ThreadID:         "core",
		Shell: &tool.ShellExecutorOptions{
			Validation: tool.ShellValidationOptions{
				CWD: cwd, DefaultTimeoutMS: options.DefaultTimeoutMS,
				ApprovalPolicy:               sandbox.ApprovalOnRequest,
				AdditionalPermissionsAllowed: false,
			},
			MaxOutputTokens: options.MaxOutputTokens,
		},
	}
	switch options.Preset {
	case ToolsReadOnly:
		profile := sandbox.ReadOnlyPermissionProfile()
		registryOptions.EnableShell = true
		registryOptions.Shell.Validation.PermissionProfileID = sandbox.BuiltInPermissionProfileReadOnly
		registryOptions.Shell.Validation.PermissionProfile = &profile
	case ToolsWorkspace:
		if options.Approval == nil {
			return nil, ErrApprovalNeed
		}
		profile := sandbox.WorkspaceWritePermissionProfile()
		registryOptions.EnableShell = true
		registryOptions.EnableApplyPatch = true
		registryOptions.Shell.Validation.PermissionProfileID = sandbox.BuiltInPermissionProfileWorkspace
		registryOptions.Shell.Validation.PermissionProfile = &profile
		registryOptions.Shell.Approval = options.Approval.ApproveShell
		registryOptions.ApplyPatch = &tool.ApplyPatchExecutorOptions{CWD: cwd, Approval: options.Approval.ApprovePatch}
	default:
		return nil, errors.New("unsupported core tool preset: " + string(options.Preset))
	}
	registry, err := turn.BuildToolRegistry(registryOptions)
	if err != nil {
		return nil, err
	}
	for _, executor := range options.Additional {
		if err := registry.Register(executor); err != nil {
			return nil, err
		}
	}
	return tool.NewRouter(registry), nil
}

func routerFromExecutors(router *tool.Router, executors ...tool.Executor) (*tool.Router, error) {
	if router != nil {
		return router, nil
	}
	if len(executors) == 0 {
		return nil, nil
	}
	registry := tool.NewRegistry()
	for _, executor := range executors {
		if err := registry.Register(executor); err != nil {
			return nil, err
		}
	}
	return tool.NewRouter(registry), nil
}

func optionalExecutors(options *ToolOptions) []tool.Executor {
	if options == nil {
		return nil
	}
	return options.Additional
}
