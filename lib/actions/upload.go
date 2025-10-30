package actions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

// ExecuteUpload 执行上传操作
// 这是统一的业务编排逻辑，CLI和TUI都调用这个函数
func ExecuteUpload(input UploadInput, callback UploadCallback) UploadResult {
	// 使用传入的context，如果没有则使用Background
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}

	if input.ApiKey == "" {
		return UploadResult{
			Success: false,
			Errors:  []error{lib.WithStep("上传", lib.NewValidationError("未登录或缺少API Key"))},
		}
	}

	// 1. 参数验证
	if err := validateUploadInput(input); err != nil {
		return UploadResult{
			Success: false,
			Errors:  []error{err},
		}
	}

	// 2. 设置默认值
	if input.BaseDomain == "" {
		input.BaseDomain = meta.DefaultDomain
	}

	// 3. 创建客户端
	client := lib.NewClient(input.BaseDomain, input.ApiKey)

	// 4. 检查模型是否存在
	if !input.Overwrite {
		exists, err := client.CheckModelExists(input.ModelName, input.ModelType)
		if err != nil {
			return UploadResult{
				Success: false,
				Errors:  []error{lib.WithStep("检查模型", err)},
			}
		}
		if exists {
			return UploadResult{
				Success: false,
				Errors: []error{
					lib.WithStep("检查模型",
						lib.NewValidationError(fmt.Sprintf("模型名 '%s' 已存在，请使用不同的名称或启用覆盖", input.ModelName))),
				},
			}
		}
	}

	// 5. 并发上传所有版本
	return uploadVersionsConcurrently(ctx, client, input, callback)
}

// validateUploadInput 验证上传参数
func validateUploadInput(input UploadInput) error {
	// 验证模型类型
	if err := lib.ValidateModelType(input.ModelType); err != nil {
		return lib.WithStep("参数验证", fmt.Errorf("模型类型无效: %w", err))
	}

	// 验证模型名称
	if err := lib.ValidateModelName(input.ModelName); err != nil {
		return lib.WithStep("参数验证", fmt.Errorf("模型名称无效: %w", err))
	}

	// 验证版本信息
	if len(input.Versions) == 0 {
		return lib.WithStep("参数验证", lib.NewValidationError("至少需要一个版本"))
	}

	for i, ver := range input.Versions {
		// 验证路径
		if ver.Path == "" {
			return lib.WithStep("参数验证", lib.NewValidationError(fmt.Sprintf("版本 %d: 路径不能为空", i+1)))
		}

		stat, err := os.Stat(ver.Path)
		if err != nil {
			return lib.WithStep("参数验证", fmt.Errorf("版本 %d: 路径无效: %w", i+1, err))
		}

		if stat.IsDir() {
			return lib.WithStep("参数验证", lib.NewValidationError(fmt.Sprintf("版本 %d: 不支持目录上传，仅支持文件", i+1)))
		}

		// 验证封面
		if ver.CoverUrl == "" {
			return lib.WithStep("参数验证", lib.NewValidationError(fmt.Sprintf("版本 %d: 封面是必填项", i+1)))
		}

		// 验证基础模型
		if ver.BaseModel != "" {
			if err := lib.ValidateBaseModel(ver.BaseModel); err != nil {
				return lib.WithStep("参数验证", fmt.Errorf("版本 %d: 基础模型无效: %w", i+1, err))
			}
		}

		// 验证版本号
		if ver.Version == "" {
			return lib.WithStep("参数验证", lib.NewValidationError(fmt.Sprintf("版本 %d: 版本号不能为空", i+1)))
		}
	}

	return nil
}

// uploadVersionsConcurrently 并发上传多个版本
func uploadVersionsConcurrently(
	ctx context.Context,
	client *lib.Client,
	input UploadInput,
	callback UploadCallback,
) UploadResult {
	total := len(input.Versions)
	versionList := make([]*lib.ModelVersion, total)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // 最多3个并发
	var mu sync.Mutex
	var uploadErrors []error
	var canceled bool

	for i, ver := range input.Versions {
		wg.Add(1)
		idx := i
		version := ver

		go func() {
			defer wg.Done()

			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()

			// 通知开始上传
			if callback != nil {
				callback.OnVersionStart(idx, total, filepath.Base(version.Path))
			}

			// 上传单个版本
			result := uploadSingleVersion(
				ctx, client, input.ModelType, version, idx, total, callback,
			)

			if result.Canceled {
				mu.Lock()
				canceled = true
				mu.Unlock()
				return
			}

			if result.Error != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, result.Error)
				mu.Unlock()

				if callback != nil {
					callback.OnVersionComplete(idx, total, filepath.Base(version.Path), result.Error)
				}
				return
			}

			// 保存结果
			mu.Lock()
			versionList[idx] = result.ModelVersion
			mu.Unlock()

			if callback != nil {
				callback.OnVersionComplete(idx, total, filepath.Base(version.Path), nil)
			}
		}()
	}

	wg.Wait()

	// 处理取消
	if canceled {
		return UploadResult{
			Success:        false,
			CanceledByUser: true,
			TotalCount:     total,
		}
	}

	// 过滤成功的版本
	successVersions := make([]*lib.ModelVersion, 0, total)
	for _, mv := range versionList {
		if mv != nil {
			successVersions = append(successVersions, mv)
		}
	}

	// 如果全部失败
	if len(successVersions) == 0 {
		return UploadResult{
			Success:      false,
			TotalCount:   total,
			SuccessCount: 0,
			Errors:       uploadErrors,
		}
	}

	// 提交模型
	_, err := client.CommitModelV2(input.ModelName, input.ModelType, successVersions)
	if err != nil {
		return UploadResult{
			Success: false,
			Errors:  []error{lib.WithStep("提交模型", err)},
		}
	}

	return UploadResult{
		Success:      true,
		SuccessCount: len(successVersions),
		TotalCount:   total,
		Errors:       uploadErrors,
		ModelName:    input.ModelName,
		ModelType:    input.ModelType,
	}
}

// singleVersionResult 单个版本上传的结果
type singleVersionResult struct {
	ModelVersion *lib.ModelVersion
	Error        error
	Canceled     bool
}

// uploadSingleVersion 上传单个版本
func uploadSingleVersion(
	ctx context.Context,
	client *lib.Client,
	modelType string,
	version VersionInput,
	index int,
	total int,
	callback UploadCallback,
) singleVersionResult {
	// 1. 上传封面
	var coverStatusCallback func(status, message string)
	if callback != nil {
		coverStatusCallback = func(status, message string) {
			callback.OnCoverStatus(index, total, status, message)
		}
	}

	coverUrl, err := lib.UploadCover(client, version.CoverUrl, ctx, coverStatusCallback)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return singleVersionResult{Canceled: true}
		}
		return singleVersionResult{
			Error: lib.WithStep(fmt.Sprintf("版本%d封面上传", index+1), err),
		}
	}

	// 2. 准备文件信息
	stat, err := os.Stat(version.Path)
	if err != nil {
		return singleVersionResult{
			Error: lib.WithStep(fmt.Sprintf("版本%d读取文件", index+1), err),
		}
	}

	relPath, _ := filepath.Rel(filepath.Dir(version.Path), version.Path)
	if relPath == "" || relPath == "." {
		relPath = filepath.Base(version.Path)
	}

	file := &lib.FileToUpload{
		Path:    filepath.ToSlash(version.Path),
		RelPath: filepath.ToSlash(relPath),
		Size:    stat.Size(),
	}

	// 3. 上传文件（带进度回调）
	_, err = lib.UnifiedUpload(lib.UploadOptions{
		File:      file,
		Client:    client,
		ModelType: modelType,
		Context:   ctx,
		FileIndex: fmt.Sprintf("%d/%d", index+1, total),
		ProgressFunc: func(consumed, fileTotal int64) {
			if callback != nil {
				callback.OnProgress(UploadProgress{
					VersionIndex: index,
					VersionTotal: total,
					FileName:     filepath.Base(file.RelPath),
					Consumed:     consumed,
					Total:        fileTotal,
				})
			}
		},
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return singleVersionResult{Canceled: true}
		}
		return singleVersionResult{
			Error: lib.WithStep(fmt.Sprintf("版本%d文件上传", index+1), err),
		}
	}

	// 4. 构建版本信息
	var coverUrls []string
	if coverUrl != "" {
		coverUrls = []string{coverUrl}
	}

	modelVersion := &lib.ModelVersion{
		Version:      version.Version,
		BaseModel:    version.BaseModel,
		Introduction: version.Introduction,
		Public:       version.Public,
		Sign:         file.Signature,
		Path:         version.Path,
		CoverUrls:    coverUrls,
	}

	return singleVersionResult{ModelVersion: modelVersion}
}
