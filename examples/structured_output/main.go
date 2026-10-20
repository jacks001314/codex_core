package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"codex_core/core"
)

type languageSummary struct {
	Language   string   `json:"language"`
	Highlights []string `json:"highlights"`
}

func main() {
	ctx := context.Background()
	client, err := core.NewFromConfig(ctx, &core.ConfigOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	result, err := client.Run(ctx, &core.Request{
		Prompt: "Describe the Go programming language as structured data.",
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"language":   map[string]any{"type": "string"},
				"highlights": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required":             []string{"language", "highlights"},
			"additionalProperties": false,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	var summary languageSummary
	if err := json.Unmarshal([]byte(result.Response.Message), &summary); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("language: %s\nhighlights: %v\n", summary.Language, summary.Highlights)
}
