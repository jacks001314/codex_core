package core_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"codex_core/core"
	"codex_core/model"
	"codex_core/session"
	"codex_core/tool"
)

type streamAgent struct{}

func (streamAgent) Run(_ context.Context, request *model.AgentRequest) (*model.AgentResponse, error) {
	if request.StreamHandler != nil {
		request.StreamHandler(&model.ResponsesStreamEvent{Kind: model.ResponsesStreamEventOutputText, Delta: "hello"})
	}
	return &model.AgentResponse{ResponseID: "resp", Message: "hello"}, nil
}

func TestStreamForwardsModelEvents(t *testing.T) {
	client, err := core.New(&core.Options{Agent: streamAgent{}})
	if err != nil {
		t.Fatal(err)
	}
	events, err := client.Stream(context.Background(), &core.Request{Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	var types []core.EventType
	for event := range events {
		types = append(types, event.Type)
	}
	want := []core.EventType{core.EventStarted, core.EventModel, core.EventCompleted}
	if !reflect.DeepEqual(types, want) {
		t.Fatalf("types=%v want=%v", types, want)
	}
}

func TestSessionPersistenceAndResumeInput(t *testing.T) {
	store := session.NewStore(t.TempDir())
	agent := &recordingAgent{response: &model.AgentResponse{ResponseID: "resp-1", Message: "answer"}}
	client, err := core.New(&core.Options{Agent: agent, SessionStore: store})
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.Run(context.Background(), &core.Request{Prompt: "first", Model: "m", ProviderID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if first.ThreadID == "" || first.TurnID == "" {
		t.Fatalf("ids missing: %#v", first)
	}
	record, err := store.Read(session.ThreadID(first.ThreadID), true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Items) < 2 || record.Metadata.LastResponseID != "resp-1" {
		t.Fatalf("record=%#v", record)
	}
	agent.response = &model.AgentResponse{ResponseID: "resp-2", Message: "next"}
	_, err = client.ResumeThread(context.Background(), first.ThreadID, &core.Request{Prompt: "continue"})
	if err != nil {
		t.Fatal(err)
	}
	if len(agent.last.InputItems) == 0 || agent.last.PreviousResponseID != "resp-1" {
		t.Fatalf("resume request=%#v", agent.last)
	}
}

type recordingAgent struct {
	last     *model.AgentRequest
	response *model.AgentResponse
}

func (a *recordingAgent) Run(_ context.Context, request *model.AgentRequest) (*model.AgentResponse, error) {
	a.last = request
	return a.response, nil
}

func TestSafeToolPresets(t *testing.T) {
	if router, err := core.BuildToolRouter(&core.ToolOptions{Preset: core.ToolsNone}); err != nil || router != nil {
		t.Fatalf("none=%v %v", router, err)
	}
	if _, err := core.BuildToolRouter(&core.ToolOptions{Preset: core.ToolsWorkspace, CWD: t.TempDir()}); !errors.Is(err, core.ErrApprovalNeed) {
		t.Fatalf("workspace error=%v", err)
	}
	router, err := core.BuildToolRouter(&core.ToolOptions{Preset: core.ToolsReadOnly, CWD: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if router == nil || len(router.ModelVisibleSpecs()) == 0 {
		t.Fatal("read-only router has no tools")
	}
}

func TestNewFromConfigWithInMemoryValues(t *testing.T) {
	home := t.TempDir()
	client, err := core.NewFromConfig(context.Background(), &core.ConfigOptions{
		CodexHome: home, APIKey: "sk-test", Model: "gpt-5.4", Values: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Run(context.Background(), &core.Request{Prompt: "x"}); !errors.Is(err, core.ErrClosed) {
		t.Fatalf("closed error=%v", err)
	}
}

func TestCustomOnlyToolPreset(t *testing.T) {
	executor := tool.NewExecutorFunc(tool.Spec{Name: tool.PlainName("x")}, func(context.Context, *tool.Invocation) (*tool.Output, error) { return &tool.Output{Success: true}, nil })
	router, err := core.BuildToolRouter(&core.ToolOptions{Preset: core.ToolsNone, Additional: []tool.Executor{executor}})
	if err != nil {
		t.Fatal(err)
	}
	if got := router.ModelVisibleSpecs(); len(got) != 1 || got[0].Name.Name != "x" {
		t.Fatalf("specs=%#v", got)
	}
}

func TestSessionFilesRemainInsideRoot(t *testing.T) {
	root := t.TempDir()
	store := session.NewStore(root)
	client, err := core.New(&core.Options{Agent: &recordingAgent{response: &model.AgentResponse{Message: "ok"}}, SessionStore: store})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Run(context.Background(), &core.Request{Prompt: "safe"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, result.ThreadID+".json")); err != nil {
		t.Fatal(err)
	}
}
