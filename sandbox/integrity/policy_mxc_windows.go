package integrity

import (
	"codex_core/sandbox"
	mxcsandbox "codex_core/sandbox/mxc"
)

func preparePlatformIntegrityPolicy(request *RunnerRequest) error {
	if request == nil || request.SandboxType != sandbox.SandboxTypeWindowsMxc {
		return nil
	}
	_, err := mxcsandbox.PrepareEntriesPolicy(request.Entries, request.CWD, request.Env, request.CWD)
	return err
}
