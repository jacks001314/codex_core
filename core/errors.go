package core

import (
	"context"
	"errors"

	"codex_core/model"
	"codex_core/session"
	"codex_core/tool"
)

type ErrorCode string

const (
	ErrorInvalidRequest ErrorCode = "invalid_request"
	ErrorClosed         ErrorCode = "closed"
	ErrorAuthentication ErrorCode = "authentication"
	ErrorConfiguration  ErrorCode = "configuration"
	ErrorModel          ErrorCode = "model"
	ErrorTool           ErrorCode = "tool"
	ErrorSession        ErrorCode = "session"
	ErrorCancelled      ErrorCode = "cancelled"
	ErrorUnknown        ErrorCode = "unknown"
)

func Code(err error) ErrorCode {
	var apiErr *model.ResponsesAPIError
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return ErrorCancelled
	case errors.Is(err, ErrNilClient), errors.Is(err, ErrNilRequest), errors.Is(err, ErrNilAgent), errors.Is(err, ErrEmptyPrompt):
		return ErrorInvalidRequest
	case errors.Is(err, ErrClosed):
		return ErrorClosed
	case errors.As(err, &apiErr) && (apiErr.StatusCode == 401 || apiErr.StatusCode == 403):
		return ErrorAuthentication
	case errors.As(err, &apiErr):
		return ErrorModel
	case errors.Is(err, model.ErrInvalidAgentRequest), errors.Is(err, model.ErrInvalidModelRequest):
		return ErrorModel
	case errors.Is(err, tool.ErrToolNotFound), errors.Is(err, tool.ErrToolInvalidCall), errors.Is(err, tool.ErrToolCancelled):
		return ErrorTool
	case errors.Is(err, session.ErrInvalidThreadID), errors.Is(err, session.ErrThreadNotFound), errors.Is(err, session.ErrConflict):
		return ErrorSession
	default:
		return ErrorUnknown
	}
}
