package main

import (
	"context"
	"fmt"
	"log"

	"codex_core/core"
	"codex_core/model"
)

// staticAgent keeps the example runnable without network access. Production
// applications normally use core.NewFromConfig to build a model-backed agent.
type staticAgent struct{}

func (staticAgent) Run(_ context.Context, request *model.AgentRequest) (*model.AgentResponse, error) {
	if request.StreamHandler != nil {
		request.StreamHandler(&model.ResponsesStreamEvent{
			Kind:  model.ResponsesStreamEventOutputText,
			Delta: "embedded ",
		})
		request.StreamHandler(&model.ResponsesStreamEvent{
			Kind:  model.ResponsesStreamEventOutputText,
			Delta: "runtime",
		})
	}
	return &model.AgentResponse{
		ResponseID: "resp-example",
		Message:    "embedded runtime",
		Usage:      model.AgentUsage{OutputTokens: 2, TotalTokens: 2},
	}, nil
}

func main() {
	ctx := context.Background()
	client, err := core.New(&core.Options{
		Agent: staticAgent{},
		EventSink: func(event core.Event) error {
			switch event.Type {
			case core.EventModel:
				if event.ModelEvent != nil && event.ModelEvent.Delta != "" {
					fmt.Printf("delta: %q\n", event.ModelEvent.Delta)
				}
			case core.EventCompleted:
				fmt.Printf("completed: %s\n", event.Response.Message)
			default:
				fmt.Printf("event: %s\n", event.Type)
			}
			return nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	result, err := client.Run(ctx, &core.Request{Prompt: "Describe the embedding boundary."})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("tokens: %d\n", result.Usage.TotalTokens)
}
