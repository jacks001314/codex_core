//go:build !windows

package mxc

import "codex_core/sandbox"

func PrepareFileSystemPolicy(
	*sandbox.PermissionProfile,
	string,
	map[string]string,
	string,
) (*FileSystemPolicy, error) {
	return nil, errUnsupportedPlatform()
}
