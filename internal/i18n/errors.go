package i18n

// LocalizedError keeps a stable message ID and an optional wrapped cause.
type LocalizedError struct {
	MessageID string
	Data      map[string]any
	Cause     error
}

func (e *LocalizedError) Error() string {
	if e == nil {
		return ""
	}
	data := cloneData(e.Data)
	if e.Cause != nil {
		data["Cause"] = e.Cause
	}
	return T(e.MessageID, data)
}

func (e *LocalizedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewError creates a localizable error while preserving the wrapped cause.
func NewError(messageID string, data map[string]any, cause error) error {
	return &LocalizedError{MessageID: messageID, Data: cloneData(data), Cause: cause}
}

func cloneData(data map[string]any) map[string]any {
	clone := make(map[string]any, len(data)+1)
	for key, value := range data {
		clone[key] = value
	}
	return clone
}
