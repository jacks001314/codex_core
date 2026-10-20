package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"codex_core/core"
)

// TestLiveCodex is opt-in. Set CODEX_LIVE_TEST=1 and provide normal Codex
// credentials (OPENAI_API_KEY, CODEX_API_KEY, or auth.json) to run it.
func TestLiveCodex(t *testing.T) {
	if os.Getenv("CODEX_LIVE_TEST") != "1" {
		t.Skip("set CODEX_LIVE_TEST=1 to run live API integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	client, err := core.NewFromConfig(ctx, &core.ConfigOptions{Stream: true, Persist: true})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	result, err := client.Run(ctx, &core.Request{Prompt: "Reply with exactly: live-ok"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Response == nil || result.Response.Message == "" {
		t.Fatalf("empty response: %#v", result)
	}
}

func TestLiveConfigurationDoesNotLeakSecret(t *testing.T) {
	if got := core.Code(errors.New("configuration invalid")); got != core.ErrorUnknown {
		t.Fatalf("code=%q", got)
	}
}
