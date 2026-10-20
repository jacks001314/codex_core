//go:build !windows

package unified_exec

import "codex_core/sandbox/windowssandbox"

func SpawnWindowsSandboxLiveSessionForLevel(req *WindowsSandboxSessionRequest) (*LiveSession, error) {
	_ = req
	return nil, windowssandbox.Unsupported("unified_exec.live_legacy")
}
