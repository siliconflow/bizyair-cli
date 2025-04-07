package meta

const (
	ApiKeyInvalid      = "Invalid api key, please check your api key."
	ModelNotFound      = "Model not found."
	NoModelFound       = "No model found."
	ModelFileNotFound  = "The model file is not found."
	NotLoggedIn        = "Not logged in"
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
