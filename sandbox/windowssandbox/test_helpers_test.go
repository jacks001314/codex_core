package windowssandbox

import coresandbox "codex_core/sandbox"

func WorkspaceWritePermissionProfileForTest() coresandbox.PermissionProfile {
	return coresandbox.WorkspaceWritePermissionProfile()
}
