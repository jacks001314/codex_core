package main

import (
	"context"
	"fmt"
	"log"

	"codex_core/core"
)

func main() {
	client, err := core.NewFromConfig(context.Background(), &core.ConfigOptions{Stream: true})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	events, err := client.Stream(context.Background(), &core.Request{Prompt: "Explain the project briefly."})
	if err != nil {
		log.Fatal(err)
	}
	for event := range events {
		if event.Type == core.EventModel && event.ModelEvent != nil {
			fmt.Print(event.ModelEvent.Delta)
		}
		if event.Type == core.EventFailed {
			log.Fatal(event.Error)
		}
	}
	fmt.Println()
}
