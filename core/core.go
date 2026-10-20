package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"codex_core/model"
	"codex_core/session"
	"codex_core/tool"
	"codex_core/turn"
)

var (
	ErrNilClient    = errors.New("codex core client is nil")
	ErrNilRequest   = errors.New("codex core request is nil")
	ErrNilAgent     = errors.New("codex core agent is nil")
	ErrEmptyPrompt  = errors.New("codex core prompt is empty")
	ErrClosed       = errors.New("codex core client is closed")
	ErrApprovalNeed = errors.New("codex core approval handler is required")
)

type EventType string

const (
	EventStarted       EventType = "started"
	EventModel         EventType = "model"
	EventToolStarted   EventType = "tool.started"
	EventToolCompleted EventType = "tool.completed"
	EventCompleted     EventType = "completed"
	EventFailed        EventType = "failed"
)

type Event struct {
	Type       EventType
	ThreadID   string
	TurnID     string
	At         time.Time
	Response   *model.AgentResponse
	ModelEvent *model.ResponsesStreamEvent
	Invocation *tool.Invocation
	ToolOutput *tool.Output
	Usage      *model.AgentUsage
	Error      error
}

type EventSink func(Event) error

type Options struct {
	Agent           model.AgentRunner
	Tools           []tool.Executor
	Router          *tool.Router
	EventSink       EventSink
	Now             func() time.Time
	MaxTurns        int
	DefaultModel    string
	DefaultProvider string
	SessionStore    *session.Store
}

type Request struct {
	Prompt             string
	Instructions       string
	InputItems         []any
	Model              string
	ProviderID         string
	ThreadID           string
	TurnID             string
	PreviousResponseID string
	OutputSchema       any
	Tools              []any
}

type Result struct {
	Response       *model.AgentResponse
	Responses      []*model.AgentResponse
	Usage          model.AgentUsage
	ThreadID       string
	TurnID         string
	StartedAt      time.Time
	EndedAt        time.Time
	Duration       time.Duration
	Iterations     int
	ToolExecutions []turn.ToolExecutionResult
}

type Client struct {
	agent           model.AgentRunner
	eventSink       EventSink
	now             func() time.Time
	router          *tool.Router
	maxTurns        int
	defaultModel    string
	defaultProvider string
	sessions        *session.Store
	sessionMu       *sync.Mutex

	mu     sync.RWMutex
	closed bool
}

func New(options *Options) (*Client, error) {
	if options == nil {
		options = &Options{}
	}
	if options.Agent == nil {
		return nil, ErrNilAgent
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	router := options.Router
	if router == nil && len(options.Tools) > 0 {
		registry := tool.NewRegistry()
		for _, executor := range options.Tools {
			if err := registry.Register(executor); err != nil {
				return nil, err
			}
		}
		router = tool.NewRouter(registry)
	}
	return &Client{
		agent: options.Agent, eventSink: options.EventSink, now: now, router: router,
		maxTurns: options.MaxTurns, defaultModel: strings.TrimSpace(options.DefaultModel),
		defaultProvider: strings.TrimSpace(options.DefaultProvider), sessions: options.SessionStore,
		sessionMu: &sync.Mutex{},
	}, nil
}

func (c *Client) Run(ctx context.Context, request *Request) (*Result, error) {
	if err := c.validateRequest(request); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req := *request
	req.InputItems = append([]any(nil), request.InputItems...)
	req.Tools = append([]any(nil), request.Tools...)
	if strings.TrimSpace(req.Model) == "" {
		req.Model = c.defaultModel
	}
	if strings.TrimSpace(req.ProviderID) == "" {
		req.ProviderID = c.defaultProvider
	}
	if c.sessions != nil {
		if c.sessionMu == nil {
			c.sessionMu = &sync.Mutex{}
		}
		c.sessionMu.Lock()
		defer c.sessionMu.Unlock()
		if err := c.prepareSessionRequest(&req); err != nil {
			return nil, err
		}
	}
	started := c.now().UTC()
	if err := c.emit(Event{Type: EventStarted, ThreadID: req.ThreadID, TurnID: req.TurnID, At: started}); err != nil {
		return nil, err
	}

	collector := &eventErrorCollector{}
	agent := &eventAgent{base: c.agent, emit: func(event *model.ResponsesStreamEvent) {
		if event == nil {
			return
		}
		collector.capture(c.emit(Event{Type: EventModel, ThreadID: req.ThreadID, TurnID: req.TurnID, At: c.now().UTC(), ModelEvent: event, Usage: event.Usage}))
	}}
	loopResult, err := turn.NewRuntime(&turn.RuntimeOptions{
		Agent: agent, Router: c.router, Now: c.now, MaxTurns: c.maxTurns,
	}).Run(ctx, &turn.AgentLoopRequest{
		Prompt: req.Prompt, Instructions: req.Instructions, InputItems: req.InputItems,
		Tools: req.Tools, Model: req.Model, ProviderID: req.ProviderID, ThreadID: req.ThreadID,
		TurnID: req.TurnID, PreviousResponseID: req.PreviousResponseID, OutputSchema: req.OutputSchema,
		OnToolStarted: func(_ context.Context, invocation *tool.Invocation, at time.Time) {
			collector.capture(c.emit(Event{Type: EventToolStarted, ThreadID: req.ThreadID, TurnID: req.TurnID, At: at, Invocation: invocation}))
		},
		PostToolInputItems: func(_ context.Context, invocation *tool.Invocation, output *tool.Output) []any {
			collector.capture(c.emit(Event{Type: EventToolCompleted, ThreadID: req.ThreadID, TurnID: req.TurnID, At: c.now().UTC(), Invocation: invocation, ToolOutput: output}))
			return nil
		},
	})
	ended := c.now().UTC()
	if err == nil {
		err = collector.err()
	}
	if err != nil {
		_ = c.emit(Event{Type: EventFailed, ThreadID: req.ThreadID, TurnID: req.TurnID, At: ended, Error: err})
		return nil, err
	}
	if loopResult == nil || loopResult.Response == nil {
		err = errors.New("codex core agent returned nil response")
		_ = c.emit(Event{Type: EventFailed, ThreadID: req.ThreadID, TurnID: req.TurnID, At: ended, Error: err})
		return nil, err
	}
	if c.sessions != nil {
		if err := c.persistSessionResult(&req, loopResult, started, ended); err != nil {
			return nil, err
		}
	}
	if err := c.emit(Event{Type: EventCompleted, ThreadID: req.ThreadID, TurnID: req.TurnID, At: ended, Response: loopResult.Response, Usage: &loopResult.Usage}); err != nil {
		return nil, err
	}
	return &Result{
		Response: loopResult.Response, Responses: loopResult.ModelResponses(), Usage: loopResult.Usage,
		ThreadID: req.ThreadID, TurnID: req.TurnID, StartedAt: started, EndedAt: ended,
		Duration: ended.Sub(started), Iterations: loopResult.Iterations, ToolExecutions: loopResult.ToolExecutions,
	}, nil
}

func (c *Client) Stream(ctx context.Context, request *Request) (<-chan Event, error) {
	if err := c.validateRequest(request); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	events := make(chan Event, 16)
	go func() {
		defer close(events)
		streamClient := &Client{
			agent: c.agent, now: c.now, router: c.router, maxTurns: c.maxTurns,
			defaultModel: c.defaultModel, defaultProvider: c.defaultProvider, sessions: c.sessions,
			sessionMu: c.sessionMu,
		}
		streamClient.eventSink = func(event Event) error {
			select {
			case events <- event:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		_, _ = streamClient.Run(ctx, request)
	}()
	return events, nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return nil
}

func (c *Client) validateRequest(request *Request) error {
	if c == nil {
		return ErrNilClient
	}
	c.mu.RLock()
	closed := c.closed
	c.mu.RUnlock()
	if closed {
		return ErrClosed
	}
	if request == nil {
		return ErrNilRequest
	}
	if strings.TrimSpace(request.Prompt) == "" && len(request.InputItems) == 0 {
		return ErrEmptyPrompt
	}
	return nil
}

func (c *Client) emit(event Event) error {
	if c.eventSink == nil {
		return nil
	}
	if err := c.eventSink(event); err != nil {
		return fmt.Errorf("emit %s event: %w", event.Type, err)
	}
	return nil
}

type eventAgent struct {
	base model.AgentRunner
	emit model.ResponsesStreamHandler
}

func (a *eventAgent) Run(ctx context.Context, request *model.AgentRequest) (*model.AgentResponse, error) {
	clone := *request
	original := request.StreamHandler
	clone.StreamHandler = func(event *model.ResponsesStreamEvent) {
		if original != nil {
			original(event)
		}
		if a.emit != nil {
			a.emit(event)
		}
	}
	return a.base.Run(ctx, &clone)
}

type eventErrorCollector struct {
	mu    sync.Mutex
	first error
}

func (c *eventErrorCollector) capture(err error) {
	if err == nil {
		return
	}
	c.mu.Lock()
	if c.first == nil {
		c.first = err
	}
	c.mu.Unlock()
}

func (c *eventErrorCollector) err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.first
}
