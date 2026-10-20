package win

import (
	"testing"

	"codex_core/sandbox/windowssandbox"
)

func TestWithReadACLMutexRejectsNil(t *testing.T) {
	if err := WithReadACLMutex(nil); err != windowssandbox.ErrInvalidRequest {
		t.Fatalf("WithReadACLMutex(nil) error = %v", err)
	}
}
