package main

import (
	"fmt"
	"log"

	"codex_core/guardiancontext"
)

// evidenceSection is a minimal section contributor. Real contributors read the
// parent conversation from SectionInput.History.
type evidenceSection struct {
	scope guardiancontext.SectionScope
	kind  guardiancontext.ConversationTranscriptEntryKind
	text  string
}

func (s evidenceSection) Scope() guardiancontext.SectionScope { return s.scope }

func (s evidenceSection) Contribute(guardiancontext.SectionInput) (*guardiancontext.ContextSection, error) {
	return &guardiancontext.ContextSection{
		Items: []guardiancontext.ConversationTranscriptEntry{{
			Kind:          s.kind,
			Text:          s.text,
			OriginalBytes: len(s.text),
		}},
	}, nil
}

func main() {
	var registry guardiancontext.SectionRegistry
	registry.Register(evidenceSection{
		scope: guardiancontext.SectionScopeShared,
		kind:  guardiancontext.UserKind(),
		text:  "Refactor the parser without changing behavior.",
	})
	registry.Register(evidenceSection{
		scope: guardiancontext.SectionScopeSyncOnly,
		kind:  guardiancontext.ToolOutputKind("shell"),
		text:  "go test ./... -> ok",
	})
	registry.Register(evidenceSection{
		scope: guardiancontext.SectionScopeAsyncOnly,
		kind:  guardiancontext.AssistantKind(),
		text:  "Scored after the turn completed.",
	})

	for _, target := range []guardiancontext.ContextTarget{
		guardiancontext.ContextTargetSync,
		guardiancontext.ContextTargetAsync,
	} {
		sections, err := registry.Collect(guardiancontext.SectionInput{Target: target})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s context: %d section(s)\n", target, len(sections))
		for _, section := range sections {
			for _, item := range section.Items {
				fmt.Printf("  %-10s %s\n", item.Kind.Role(), guardiancontext.TruncateText(item.Text, 80))
			}
		}
	}
}
