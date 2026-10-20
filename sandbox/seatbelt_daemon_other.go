//go:build !windows

package sandbox

import "codex_core/codexuds"

func sharedDaemonSocketDirectory() (string, bool) {
	directory, err := codexuds.SharedDaemonSocketDirectory()
	if err != nil || directory == "" {
		return "", false
	}
	return directory, true
}
