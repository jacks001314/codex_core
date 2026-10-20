package main

import (
	"context"
	"fmt"
	"log"

	"codex_core/core"
)

func main() {
	client, err := core.NewFromConfig(context.Background(), &core.ConfigOptions{
		Tools: &core.ToolOptions{Preset: core.ToolsReadOnly, CWD: "."},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	result, err := client.Run(context.Background(), &core.Request{
		Prompt: "Inspect this workspace and summarize its Go packages without changing files.",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Response.Message)
}
