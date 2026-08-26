package meta

import (
	"sort"
	"strings"

	"github.com/samber/lo"
)

const (
	CmdLogin   = "login"
	CmdLogout  = "logout"
	CmdUpload  = "upload"
	CmdModel   = "model"
	CmdLs      = "ls"
	CmdDetail  = "detail"
	CmdRm      = "rm"
	CmdCommit  = "commit"
	CmdUpgrade = "upgrade"
	CmdWhoami  = "whoami"
	CmdPlan    = "plan"
	CmdWallet  = "wallet"
	CmdDayCost = "day-cost"
)

const (
	// DefaultServiceHost is the only Go default that needs to change for a
	// future BizyAir API domain migration.
	DefaultServiceHost = "bizyair.ai"

	// DefaultBaseDomain is the configurable root used to derive BizyAir API,
	// metadata, web, and general storage service domains.
	DefaultBaseDomain = "https://" + DefaultServiceHost

	// Deprecated: use DefaultBaseDomain. Kept for source compatibility.
	DefaultDomain = DefaultBaseDomain
	StorageDomain = "https://storage." + DefaultServiceHost
)

const (
	LoadError           = 1
	ServerError         = 2
	HttpError           = 3
	InterruptedExitCode = 130
)

const (
	// 分片上传配置
	MultipartPartSize  = 5 * 1024 * 1024   // 每个分片5MB（与前端一致）
	MultipartParallel  = 3                 // 并发上传3个分片（与前端一致）
	MultipartThreshold = 100 * 1024 * 1024 // 超过100MB使用分片上传
	CheckpointFolder   = "uploads"         // checkpoint文件夹名称

	// ManifestURL intentionally does not follow DefaultBaseDomain. CLI
	// releases are published to the fixed .ai storage domain.
	ManifestURL         = "https://storage.bizyair.ai/cli/releases/manifest.json"
	UpgradeBackupSuffix = ".backup"
	UpgradeMaxRetries   = 3
)

type UploadFileType string

const (
	TypeCheckpoint UploadFileType = "Checkpoint"
	TypeVae        UploadFileType = "VAE"
	TypeUNet       UploadFileType = "UNet"
	TypeLora       UploadFileType = "LoRA"
	TypeControlNet UploadFileType = "Controlnet"
	TypeClip       UploadFileType = "CLIP"
	TypeUpscale    UploadFileType = "Upscaler"
	TypeDetection  UploadFileType = "Detection"
	TypeOther      UploadFileType = "Other"
)

var ModelTypes = []UploadFileType{
	TypeCheckpoint,
	TypeVae,
	TypeUNet,
	TypeLora,
	TypeControlNet,
	TypeClip,
	TypeUpscale,
	TypeDetection,
	TypeOther,
}

var ModelTypesStr = func(arr []UploadFileType) string {
	strs := lo.Map(arr, func(v UploadFileType, _ int) string {
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
	SfFolder                = ".bizyair"
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

// SupportedBaseModels is the offline fallback used by the TUI and upload help.
// Keep it in sync with the base_models field of the /v1/dict API.
var SupportedBaseModels = map[string]bool{
	"FLUX.1 D":       true,
	"FLUX.1 Kontext": true,
	"FLUX.1 S":       true,
	"SDXL":           true,
	"SD 1.5":         true,
	"SD 3.5":         true,
	"Pony":           true,
	"Illustrious":    true,
	"NoobAI":         true,
	"Anima":          true,
	"FLUX.2 D":       true,
	"FLUX.2 Klein":   true,
	"ERNIE-Image":    true,
	"Ideogram":       true,
	"Krea":           true,
	"Kolors":         true,
	"MiniMax H3":     true,
	"Hunyuan Video":  true,
	"Wan Video":      true,
	"Qwen-Image":     true,
	"Qwen-Edit":      true,
	"Z-Image":        true,
	"Ovis":           true,
	"LTX-2":          true,
	"Nano Banana":    true,
	"Seedream":       true,
	"Seedance":       true,
	"Veo":            true,
	"Kling":          true,
	"Hailuo":         true,
	"GPT-Image":      true,
	"Vidu":           true,
	"Grok":           true,
	"Happy Horse":    true,
	"Other":          true,
}

var BaseModelStr = parseMapKey(SupportedBaseModels)

func parseMapKey[T any](myMap map[string]T) string {
	strs := make([]string, 0)
	for k := range myMap {
		strs = append(strs, k)
	}
	sort.Strings(strs)
	return "'" + strings.Join(strs, "','") + "'"
}
