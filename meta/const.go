package meta

import (
	"strings"

	"github.com/samber/lo"
)

const (
	CmdLogin      = "login"
	CmdWhoami     = "whoami"
	CmdLogout     = "logout"
	CmdUpload     = "upload"
	CmdUploadFile = "upload-file"
	CmdModel      = "model"
	CmdLs         = "ls"
	CmdLsFiles    = "ls-files"
	CmdRm         = "rm"
	CmdCommit     = "commit"
)

const (
	ModelNameFileName = "name.txt"
	ModelTypeFileName = "type.txt"
	ContentFileName   = "content.*"
	BaseModelFileName = "basemodel.*"
	IntroFileName     = "description.*"
	PublicFileName    = "public.*"
	CoverFileName	  = "cover*.*"
)

const (
	DefaultDomain = "https://bizyair-api.siliconflow.cn"
	UATDomain     = "https://uat-api.bizyair.cn"
	AuthDomain    = "https://api.siliconflow.cn"
)

const (
	LoadError   = 1
	ServerError = 2
	HttpError   = 3
)

type UploadFileType string

const (
	TypeCheckpoint   UploadFileType = "Checkpoint"
	TypeVae          UploadFileType = "bizyair/vae"
	TypeLora         UploadFileType = "LoRA"
	TypeControlNet   UploadFileType = "Controlnet"
	TypeEmbedding    UploadFileType = "bizyair/embedding"
	TypeHyperNetwork UploadFileType = "bizyair/hypernetwork"
	TypeClip         UploadFileType = "bizyair/clip"
	TypeClipVision   UploadFileType = "bizyair/clip_vision"
	TypeUpscale      UploadFileType = "bizyair/upscale"
	TypeDataset      UploadFileType = "bizyair/dataset"
	TypeOther        UploadFileType = "Other"
	TypeWorkflow     UploadFileType = "Workflow"
)

var ModelTypes = []UploadFileType{
	TypeCheckpoint,
	TypeLora,
	TypeControlNet,
	TypeOther,
	TypeWorkflow,
}

var ModelTypesStr = func(arr []UploadFileType) string {
	strs := lo.Map[UploadFileType, string](arr, func(v UploadFileType, _ int) string {
		return string(v)
	})
	return "'" + strings.Join(strs, "','") + "'"
}(ModelTypes)

const (
	PercentEncode           = "%2F"
	HTTPGet                 = "GET"
	HTTPPost                = "POST"
	HTTPPut                 = "PUT"
	HTTPDelete              = "DELETE"
	HeaderAuthorization     = "Authorization"
	HeaderContentType       = "Content-Type"
	HeaderSiliconCliVersion = "X-Silicon-CLI-Version"
	JsonContentType         = "application/json"
	APIv1                   = "v1"
	SfFolder                = ".siliconflow"
	SfApiKey                = "apikey"
	OSWindows               = "windows"
	EnvUserProfile          = "USERPROFILE"
	EnvHome                 = "HOME"
	EnvAPIKey               = "SF_API_KEY"
	OSSObjectKey            = "https://%s.%s.aliyuncs.com/%s"
	OKCode                  = 20000
)

// IgnoreUploadDirs ignore files when upload
var IgnoreUploadDirs = []string{
	".git",
	".idea",
}

var SupportedBaseModels = map[string]bool{
	"Flux.1 D":  true,
	"SDXL":      true,
	"SD 1.5":    true,
	"SD 3.5":    true,
	"Pony":      true,
	"Kolors":    true,
	"Hunyuan 1": true,
	"Other":     true,
}

var BaseModelStr = parseMapKey(SupportedBaseModels)

func parseMapKey[T any](myMap map[string]T) string {
	strs := make([]string, 0)
	for k, _ := range myMap {
		strs = append(strs, k)
	}
	return "'" + strings.Join(strs, "','") + "'"
}
