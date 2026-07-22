package lib

import (
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

// Format keeps normal user output localized while exposing preserved server
// details to verbose diagnostics through the conventional %+v format.
func (e *APIError) Format(state fmt.State, verb rune) {
	if e == nil {
		return
	}
	if verb == 'v' && state.Flag('+') {
		_, _ = fmt.Fprint(state, e.DebugString())
		return
	}
	if verb == 'q' {
		_, _ = fmt.Fprintf(state, "%q", e.Error())
		return
	}
	_, _ = fmt.Fprint(state, e.Error())
}
