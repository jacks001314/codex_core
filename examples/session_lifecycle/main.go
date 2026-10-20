package main

import (
	"context"
	"fmt"
	"log"

	"codex_core/core"
	"codex_core/session"
)

func main() {
	ctx := context.Background()
	client, err := core.NewFromConfig(ctx, &core.ConfigOptions{Persist: true})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Running a turn on a fresh client auto-creates and persists a thread.
	first, err := client.Run(ctx, &core.Request{Prompt: "Create a todo list with three items."})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("original thread: %s\n", first.ThreadID)

	// The stored thread can be re-read with its materialized history.
	record, err := client.ReadThread(first.ThreadID, true, true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("stored history items: %d\n", record.ItemCount)

	// Threads can be listed with pagination.
	page, err := client.ListThreads(session.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("threads in store: %d\n", len(page.Records))

	// Fork creates an independent branch of the conversation.
	fork, err := client.ForkThread(first.ThreadID, session.ForkOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("forked thread: %s\n", fork.ID)

	// Lifecycle cleanup: archive then delete the original branch.
	if err := client.ArchiveThread(first.ThreadID); err != nil {
		log.Fatal(err)
	}
	if err := client.DeleteThread(first.ThreadID); err != nil {
		log.Fatal(err)
	}
	fmt.Println("original thread archived and deleted")
}
