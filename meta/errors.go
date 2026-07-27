package meta

const (
	// Deprecated: use the localized API error returned by lib.Client.
	ApiKeyInvalid = "Invalid api key, please check your api key."
	// Deprecated: use the localized API error returned by lib.Client.
	ModelNotFound = "Model not found."
	// Deprecated: use the localized API error returned by lib.Client.
	NoModelFound = "No model found."
	// Deprecated: use the localized API error returned by lib.Client.
	ModelFileNotFound = "The model file is not found."
	// Deprecated: use the localized API error returned by lib.Client.
	NotLoggedIn = "Not logged in"
	// Deprecated: use the localized API error returned by lib.Client.
	ModelAlreadyExists = "Model already exists."
)

type ErrNo struct {
	ErrMsg string
}

func (e ErrNo) Error() string {
	return e.ErrMsg
}

func NewErrNo(msg string) ErrNo {
	return ErrNo{msg}
}

var ServerErrors = map[int32]ErrNo{
	20004: NewErrNo(ApiKeyInvalid),
	20224: NewErrNo(ModelNotFound),
	20225: NewErrNo(ModelFileNotFound),
	20226: NewErrNo(NoModelFound),
	20227: NewErrNo(ModelAlreadyExists),
}

// ServerErrorMessageIDs maps stable server error codes to local message IDs.
// The legacy ServerErrors map remains available for source compatibility.
var ServerErrorMessageIDs = map[int32]string{
	20004: "error.server.api_key_invalid",
	20224: "error.server.model_not_found",
	20225: "error.server.model_file_not_found",
	20226: "error.server.no_models",
	20227: "error.server.model_exists",
}
