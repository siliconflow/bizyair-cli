package lib

// ProgressCallback 进度回调函数类型
type ProgressCallback func(consumed, total int64)

// ProgressAdapter 进度适配器接口
type ProgressAdapter interface {
	OnProgress(fileIndex string, fileName string, consumed, total int64)
}
