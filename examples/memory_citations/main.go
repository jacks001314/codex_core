package main

import (
	"fmt"
	"log"

	"codex_core/memories"
)

func main() {
	citation := memories.ParseCitation([]string{
		"<citation_entries>\nREADME.md:10-14 |note=[documents the embeddable runtime]\nruntime/demo.go:3-8 |note=[shows the tool loop]\n</citation_entries>",
		"<rollout_ids>\nthread-abc123\n../escape\nthread-abc123\nthread/with/slash\n</rollout_ids>",
	})
	if citation == nil {
		log.Fatal("no citation parsed")
	}

	fmt.Println("evidence:")
	for _, entry := range citation.Entries {
		fmt.Printf("  %s:%d-%d %s\n", entry.Path, entry.LineStart, entry.LineEnd, entry.Note)
	}
	fmt.Printf("referenced rollout ids: %v\n", citation.RolloutIDs)
	fmt.Printf("safe thread ids: %v\n", memories.ThreadIDsFromCitation(citation))
}
