package main

import (
	"context"
	"fmt"
	"log"

	"codex_core/core"
)

func main() {
	client, err := core.NewFromConfig(context.Background(), &core.ConfigOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	result, err := client.Run(context.Background(), &core.Request{
		Prompt: "In one sentence, explain what an embeddable agent runtime does.",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Response.Message)
}
