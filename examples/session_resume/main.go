package main

import (
	"context"
	"fmt"
	"log"

	"codex_core/core"
)

func main() {
	client, err := core.NewFromConfig(context.Background(), &core.ConfigOptions{Persist: true})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	first, err := client.Run(context.Background(), &core.Request{Prompt: "Remember that the project color is blue."})
	if err != nil {
		log.Fatal(err)
	}
	second, err := client.ResumeThread(context.Background(), first.ThreadID, &core.Request{Prompt: "What is the project color?"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(second.Response.Message)
}
