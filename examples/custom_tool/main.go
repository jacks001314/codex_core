package main

import (
	"context"
	"fmt"
	"log"

	"codex_core/core"
	"codex_core/tool"
)

func main() {
	echo := tool.NewExecutorFunc(tool.Spec{
		Name: tool.PlainName("echo"), Description: "Returns the supplied text.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{"type": "string"},
			},
			"required":             []string{"text"},
			"additionalProperties": false,
		},
	}, func(_ context.Context, invocation *tool.Invocation) (*tool.Output, error) {
		var args struct {
			Text string `json:"text"`
		}
		if err := invocation.DecodeArguments(&args); err != nil {
			return nil, err
		}
		return &tool.Output{Success: true, Body: args.Text}, nil
	})
	client, err := core.NewFromConfig(context.Background(), &core.ConfigOptions{
		Client: core.Options{Tools: []tool.Executor{echo}},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	result, err := client.Run(context.Background(), &core.Request{Prompt: "Call echo with text hello"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Response.Message)
}
