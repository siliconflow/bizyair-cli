package lib

import (
	"errors"
	"fmt"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
)

// APIError is a structured server failure. KnownCodeMessageID is populated
// for stable BizyAir error codes; unknown errors retain their raw detail.
type APIError struct {
	HTTPStatus         int
	Code               int32
	RawMessage         string
	KnownCodeMessageID string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.KnownCodeMessageID != "" {
		return i18n.T(e.KnownCodeMessageID)
	}
	data := map[string]any{
		"Status": e.HTTPStatus,
		"Code":   e.Code,
		"Detail": e.RawMessage,
	}
	if e.RawMessage == "" {
		return i18n.T("error.server.request_failed", data)
	}
	if e.HTTPStatus > 0 {
		return i18n.T("error.server.request_failed_detail", data)
	}
	return i18n.T("error.server.response_failed_detail", data)
}

func (e *APIError) DebugString() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("HTTP status=%d code=%d message=%q", e.HTTPStatus, e.Code, e.RawMessage)
}

// FormatError keeps normal output localized and appends preserved server
// diagnostics only when verbose output was explicitly requested.
func FormatError(err error, verbose bool) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if !verbose {
		return message
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return message
	}
	detail := apiErr.DebugString()
	if detail == "" || detail == message {
		return message
	}
	return message + "\n" + detail
}
