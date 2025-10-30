package actions

import (
	"context"

	"github.com/siliconflow/bizyair-cli/lib"
)

// LoginResult 登录操作的结果
type LoginResult struct {
	Success bool
	ApiKey  string
	Error   error
}

// ListModelsInput 查询模型列表的输入参数
type ListModelsInput struct {
	ApiKey     string
	BaseDomain string
	ModelType  string   // 为空则查询所有类型
	ModelTypes []string // 多个类型
	BaseModels []string
	Keyword    string
	Sort       string
	Current    int
	PageSize   int
}

// ListModelsResult 查询模型列表的结果
type ListModelsResult struct {
	Models []*lib.BizyModelInfo
	Total  int
	Error  error
}

// ModelDetailResult 查询模型详情的结果
type ModelDetailResult struct {
	Detail *lib.BizyModelDetail
	Error  error
}

// DeleteModelResult 删除模型的结果
type DeleteModelResult struct {
	Success bool
	Error   error
}

// VersionInput 单个版本的输入参数
type VersionInput struct {
	Version      string
	Path         string
	BaseModel    string
	Introduction string
	CoverUrl     string
	Public       bool
}

// UploadInput 上传操作的输入参数
type UploadInput struct {
	ApiKey     string
	BaseDomain string
	ModelType  string
	ModelName  string
	Versions   []VersionInput
	Overwrite  bool
	Context    context.Context // 用于取消操作
}

// UploadProgress 上传进度信息
type UploadProgress struct {
	VersionIndex int    // 当前版本索引（0-based）
	VersionTotal int    // 总版本数
	FileName     string // 文件名
	Consumed     int64  // 已上传字节数
	Total        int64  // 文件总大小
}

// UploadResult 上传操作的结果
type UploadResult struct {
	Success        bool
	SuccessCount   int
	TotalCount     int
	Errors         []error
	CanceledByUser bool
	ModelName      string // 模型名称
	ModelType      string // 模型类型
}

// UploadCallback 上传过程的回调接口
// CLI和TUI需要实现此接口来接收上传进度和状态更新
type UploadCallback interface {
	// OnProgress 上传进度更新
	OnProgress(progress UploadProgress)

	// OnVersionStart 某个版本开始上传
	OnVersionStart(index, total int, fileName string)

	// OnVersionComplete 某个版本上传完成（成功或失败）
	OnVersionComplete(index, total int, fileName string, err error)

	// OnCoverStatus 封面处理状态更新
	// status: "converting" (转换中), "ready" (已准备), "fallback" (回退原格式), "done" (完成)
	OnCoverStatus(index, total int, status, message string)
}
