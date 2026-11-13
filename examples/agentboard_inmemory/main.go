package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"codex_core/agent"
	"codex_core/agentboard"
)

// demoHost is the minimal agentboard.Host the in-memory backend needs. A host
// owns membership, the posting clock and notification delivery, so the board
// never invents identities or timestamps on its own.
type demoHost struct {
	members  map[string]agent.AgentPath
	clock    time.Time
	notified []string
}

func newDemoHost(members map[string]agent.AgentPath) *demoHost {
	return &demoHost{
		members: members,
		clock:   time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC),
	}
}

func (h *demoHost) AgentPath(_ context.Context, caller string) (agent.AgentPath, error) {
	path, ok := h.members[caller]
	if !ok {
		return "", fmt.Errorf("unknown agent %q", caller)
	}
	return path, nil
}

func (h *demoHost) ResolveAgent(_ context.Context, path agent.AgentPath) (string, error) {
	for id, member := range h.members {
		if member == path {
			return id, nil
		}
	}
	return "", fmt.Errorf("unknown agent path %q", path)
}

func (h *demoHost) CurrentTime(context.Context, string) (time.Time, error) {
	h.clock = h.clock.Add(time.Minute)
	return h.clock, nil
}

func (h *demoHost) Notify(_ context.Context, recipient string, post agentboard.PostPreview) (agentboard.NotificationDelivery, error) {
	h.notified = append(h.notified, recipient+" <- "+post.MessageID)
	return agentboard.NotificationAccepted, nil
}

func main() {
	ctx := context.Background()

	rootPath, err := agent.NewAgentPath("/root")
	if err != nil {
		log.Fatal(err)
	}
	workerPath, err := agent.NewAgentPath("/root/worker")
	if err != nil {
		log.Fatal(err)
	}
	host := newDemoHost(map[string]agent.AgentPath{
		"root":   rootPath,
		"worker": workerPath,
	})

	// The in-memory registry stands in for the local SQLite backend: one
	// identity is one board tree, and every caller shares its channels.
	boards := &agentboard.InMemoryMessageBoards{}
	board := boards.Open("demo-tree", host)
	defer board.Close()

	if _, err := board.CreateChannel(ctx, "root", agentboard.CreateChannelRequest{
		ChannelName:  "general",
		Subscription: agentboard.Subscribe,
	}); err != nil {
		log.Fatal(err)
	}

	root, err := board.Post(ctx, "root", agentboard.PostRequest{
		RequestID:      "req-root-1",
		Destination:    agentboard.PostDestination{Kind: "channel", Name: "general"},
		Text:           "worker: summarise the build status in one line",
		AgentsToNotify: []agent.AgentPath{workerPath},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("root post: %s by %s\n", root.MessageID, root.Author)

	// Replies target the root post's thread instead of the channel.
	reply, err := board.Post(ctx, "worker", agentboard.PostRequest{
		RequestID:   "req-worker-1",
		Destination: agentboard.PostDestination{Kind: "thread", ThreadID: root.MessageID},
		Text:        "linux green, windows pending, macOS skipped",
	})
	if err != nil {
		log.Fatal(err)
	}

	thread, err := board.ReadThread(ctx, "root", agentboard.ReadThreadRequest{
		ThreadID: root.MessageID,
		Page:     agentboard.PageRequest{Limit: 20},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("thread content: %q with %d replies\n", thread.RootPost.TextPreview, len(thread.Replies.Results))

	// Reads are bounded so a host can page through long posts.
	chunk, err := board.ReadPost(ctx, "worker", agentboard.ReadPostRequest{
		MessageID:  reply.MessageID,
		LimitChars: 12,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("chunk: %q (next offset %d)\n", chunk.Text, chunk.NextOffsetChars)

	channels, err := board.ListChannels(ctx, "worker", agentboard.ChannelQuery{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("channels: %d, first has %d messages\n", len(channels.Results), channels.Results[0].MessageCount)

	fmt.Printf("notifications: %v\n", host.notified)
}
