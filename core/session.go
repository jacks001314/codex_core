package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"codex_core/session"
	"codex_core/turn"

	"github.com/google/uuid"
)

func (c *Client) prepareSessionRequest(request *Request) error {
	if c == nil || c.sessions == nil || request == nil {
		return nil
	}
	if strings.TrimSpace(request.ThreadID) == "" {
		request.ThreadID = uuid.NewString()
	}
	if strings.TrimSpace(request.TurnID) == "" {
		request.TurnID = uuid.NewString()
	}
	record, err := c.sessions.Read(session.ThreadID(request.ThreadID), true, true)
	if errors.Is(err, session.ErrThreadNotFound) {
		now := c.now().UTC()
		return c.sessions.Create(&session.Record{
			ID: session.ThreadID(request.ThreadID), CreatedAt: now, UpdatedAt: now, RecencyAt: now,
			Metadata: session.Metadata{Model: request.Model, ModelProvider: request.ProviderID},
		})
	}
	if err != nil {
		return err
	}
	history := session.InputItemsFromRecord(record, &session.HistoryBuildOptions{IncludeToolOutputs: true, CWD: record.Metadata.CWD})
	if len(history) > 0 {
		request.InputItems = append(history, request.InputItems...)
	}
	if strings.TrimSpace(request.PreviousResponseID) == "" {
		request.PreviousResponseID = firstNonEmpty(record.Metadata.LastResponseID, record.Metadata.PreviousResponseID)
	}
	if strings.TrimSpace(request.Model) == "" {
		request.Model = record.Metadata.Model
	}
	if strings.TrimSpace(request.ProviderID) == "" {
		request.ProviderID = record.Metadata.ModelProvider
	}
	return nil
}

func (c *Client) persistSessionResult(request *Request, result *turn.AgentLoopResult, started, ended time.Time) error {
	if c == nil || c.sessions == nil || request == nil || result == nil {
		return nil
	}
	items := make([]session.Item, 0, 1+len(result.InputItems))
	if prompt := strings.TrimSpace(request.Prompt); prompt != "" {
		items = append(items, session.Item{ID: uuid.NewString(), Type: "message", Role: "user", Text: prompt, CreatedAt: started, Metadata: map[string]any{"turn_id": request.TurnID}})
	}
	for _, response := range result.ModelResponses() {
		if response == nil {
			continue
		}
		if len(response.Items) == 0 && strings.TrimSpace(response.Message) != "" {
			items = append(items, session.Item{ID: uuid.NewString(), Type: "message", Role: "assistant", Text: response.Message, ResponseID: response.ResponseID, CreatedAt: ended, Metadata: map[string]any{"turn_id": request.TurnID}})
		}
		for i := range response.Items {
			item := response.Items[i]
			raw, err := json.Marshal(&item)
			if err != nil {
				return err
			}
			role := ""
			if item.Type == "message" {
				role = "assistant"
			}
			items = append(items, session.Item{
				ID: firstNonEmpty(item.ID, uuid.NewString()), Type: item.Type, Role: role, Text: item.Text,
				Name: item.Name, Namespace: item.Namespace, CallID: item.CallID, Status: item.Status,
				Data: cloneAnyMap(item.Data), Raw: raw, ResponseID: response.ResponseID, CreatedAt: ended,
				Metadata: map[string]any{"turn_id": request.TurnID},
			})
		}
	}
	for i := range result.ToolExecutions {
		execution := result.ToolExecutions[i]
		if execution.Output == nil {
			continue
		}
		payload := map[string]any{
			"type": "function_call_output", "call_id": execution.Output.CallID,
			"output": execution.Output.Body,
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		items = append(items, session.Item{
			ID: uuid.NewString(), Type: "function_call_output", CallID: execution.Output.CallID,
			Text: execution.Output.Body, Data: cloneAnyMap(execution.Output.Data), Raw: raw,
			CreatedAt: execution.FinishedAt, Metadata: map[string]any{"turn_id": request.TurnID},
		})
	}
	if len(items) > 0 {
		if _, err := c.sessions.AppendItems(session.ThreadID(request.ThreadID), items); err != nil {
			return err
		}
	}
	lastResponseID := ""
	if result.Response != nil {
		lastResponseID = result.Response.ResponseID
	}
	modelID := request.Model
	providerID := request.ProviderID
	_, err := c.sessions.UpdateMetadata(session.ThreadID(request.ThreadID), &session.MetadataPatch{
		Model: &modelID, ModelProvider: &providerID, LastResponseID: &lastResponseID,
	}, true)
	return err
}

func (c *Client) ResumeThread(ctx context.Context, threadID string, request *Request) (*Result, error) {
	if request == nil {
		request = &Request{}
	}
	clone := *request
	clone.ThreadID = threadID
	return c.Run(ctx, &clone)
}

func (c *Client) CreateThread(record *session.Record) (*session.Record, error) {
	if c == nil {
		return nil, ErrNilClient
	}
	if c.sessions == nil {
		return nil, fmt.Errorf("session store is not configured")
	}
	if record == nil {
		record = &session.Record{}
	}
	clone := *record
	if clone.ID == "" {
		clone.ID = session.ThreadID(uuid.NewString())
	}
	now := c.now().UTC()
	if clone.CreatedAt.IsZero() {
		clone.CreatedAt = now
	}
	if clone.UpdatedAt.IsZero() {
		clone.UpdatedAt = now
	}
	if clone.RecencyAt.IsZero() {
		clone.RecencyAt = now
	}
	if err := c.sessions.Create(&clone); err != nil {
		return nil, err
	}
	return c.sessions.Read(clone.ID, true, true)
}

func (c *Client) ReadThread(threadID string, includeArchived, includeHistory bool) (*session.Record, error) {
	if c == nil {
		return nil, ErrNilClient
	}
	if c.sessions == nil {
		return nil, fmt.Errorf("session store is not configured")
	}
	return c.sessions.Read(session.ThreadID(threadID), includeArchived, includeHistory)
}

func (c *Client) ListThreads(options session.ListOptions) (*session.Page, error) {
	if c == nil {
		return nil, ErrNilClient
	}
	if c.sessions == nil {
		return nil, fmt.Errorf("session store is not configured")
	}
	return c.sessions.List(options)
}

func (c *Client) ForkThread(threadID string, options session.ForkOptions) (*session.Record, error) {
	if c == nil {
		return nil, ErrNilClient
	}
	if c.sessions == nil {
		return nil, fmt.Errorf("session store is not configured")
	}
	return c.sessions.Fork(session.ThreadID(threadID), options)
}

func (c *Client) ArchiveThread(threadID string) error {
	if c == nil {
		return ErrNilClient
	}
	if c.sessions == nil {
		return fmt.Errorf("session store is not configured")
	}
	return c.sessions.Archive(session.ThreadID(threadID))
}

func (c *Client) DeleteThread(threadID string) error {
	if c == nil {
		return ErrNilClient
	}
	if c.sessions == nil {
		return fmt.Errorf("session store is not configured")
	}
	return c.sessions.Delete(session.ThreadID(threadID))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func cloneAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
