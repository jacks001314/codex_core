package core_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"codex_core/core"
	"codex_core/model"
	"codex_core/tool"
)

type fakeAgent struct {
	request  *model.AgentRequest
	response *model.AgentResponse
	err      error
}

func (f *fakeAgent) Run(ctx context.Context, request *model.AgentRequest) (*model.AgentResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.request = request
	return f.response, f.err
}

func TestClientRunBuildsAgentRequestAndEvents(t *testing.T) {
	agent := &fakeAgent{response: &model.AgentResponse{ResponseID: "resp-1", Message: "done"}}
	var events []core.Event
	client, err := core.New(&core.Options{Agent: agent, EventSink: func(event core.Event) error { events = append(events, event); return nil }})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Run(context.Background(), &core.Request{Prompt: "hello", ThreadID: "thread-1", TurnID: "turn-1", Model: "test-model"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Response.ResponseID != "resp-1" || agent.request.Model != "test-model" {
		t.Fatalf("unexpected result/request: %#v %#v", result, agent.request)
	}
	if got := []core.EventType{events[0].Type, events[1].Type}; !reflect.DeepEqual(got, []core.EventType{core.EventStarted, core.EventCompleted}) {
		t.Fatalf("events = %#v", got)
	}
	if result.Duration < 0 {
		t.Fatalf("negative duration: %v", result.Duration)
	}
}

func TestClientRunReportsFailure(t *testing.T) {
	want := errors.New("upstream failed")
	client, err := core.New(&core.Options{Agent: &fakeAgent{err: want}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Run(context.Background(), &core.Request{Prompt: "hello"})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
}

func TestClientStreamCloses(t *testing.T) {
	client, err := core.New(&core.Options{Agent: &fakeAgent{response: &model.AgentResponse{Message: "ok"}}, Now: func() time.Time { return time.Unix(1, 0) }})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := client.Stream(context.Background(), &core.Request{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	var got []core.EventType
	for event := range stream {
		got = append(got, event.Type)
	}
	want := []core.EventType{core.EventStarted, core.EventCompleted}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestClientValidatesInputs(t *testing.T) {
	if _, err := core.New(nil); !errors.Is(err, core.ErrNilAgent) {
		t.Fatalf("New error = %v", err)
	}
	client, err := core.New(&core.Options{Agent: &fakeAgent{response: &model.AgentResponse{}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Run(context.Background(), nil); !errors.Is(err, core.ErrNilRequest) {
		t.Fatalf("nil request error = %v", err)
	}
	if _, err := client.Run(context.Background(), &core.Request{}); !errors.Is(err, core.ErrEmptyPrompt) {
		t.Fatalf("empty prompt error = %v", err)
	}
}

func TestClientRunDispatchesRegisteredTool(t *testing.T) {
	calls := 0
	executor := tool.NewExecutorFunc(tool.Spec{
		Name: tool.PlainName("echo"), InputSchema: map[string]any{"type": "object"},
	}, func(ctx context.Context, invocation *tool.Invocation) (*tool.Output, error) {
		calls++
		return &tool.Output{Success: true, Body: "tool result"}, nil
	})
	agent := &sequenceAgent{responses: []*model.AgentResponse{
		{Items: []model.AgentItem{{Type: "function_call", Name: "echo", CallID: "call-1", Arguments: `{}`}}},
		{Message: "finished"},
	}}
	client, err := core.New(&core.Options{Agent: agent, Tools: []tool.Executor{executor}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Run(context.Background(), &core.Request{Prompt: "use echo"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || result.Iterations != 2 || len(result.ToolExecutions) != 1 || result.Response.Message != "finished" {
		t.Fatalf("calls=%d result=%#v", calls, result)
	}
}

type sequenceAgent struct {
	responses []*model.AgentResponse
	index     int
}

func (a *sequenceAgent) Run(ctx context.Context, request *model.AgentRequest) (*model.AgentResponse, error) {
	if a.index >= len(a.responses) {
		return nil, errors.New("unexpected model call")
	}
	response := a.responses[a.index]
	a.index++
	return response, nil
}
