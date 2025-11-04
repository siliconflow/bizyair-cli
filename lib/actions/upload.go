package actions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/siliconflow/bizyair-cli/lib"
)

func ExecuteUpload(api lib.BizyAPI, input UploadInput, callback UploadCallback) UploadResult {
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

	if err := validateUploadInput(input); err != nil {
		return UploadResult{
			Success: false,
			Errors:  []error{err},
		}
	}

	if !input.Overwrite {
		exists, err := api.CheckModelExists(input.ModelName, input.ModelType)
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

	return uploadVersionsConcurrently(ctx, api, input, callback)
}

func validateUploadInput(input UploadInput) error {
	if err := lib.ValidateModelType(input.ModelType); err != nil {
		return lib.WithStep("参数验证", fmt.Errorf("模型类型无效: %w", err))
	}

	if err := lib.ValidateModelName(input.ModelName); err != nil {
		return lib.WithStep("参数验证", fmt.Errorf("模型名称无效: %w", err))
	}

	if len(input.Versions) == 0 {
		return lib.WithStep("参数验证", lib.NewValidationError("至少需要一个版本"))
	}

	for i, ver := range input.Versions {
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

		if ver.CoverUrl == "" {
			return lib.WithStep("参数验证", lib.NewValidationError(fmt.Sprintf("版本 %d: 封面是必填项", i+1)))
		}

		// 验证基础模型 - 使用从API获取的列表
		if ver.BaseModel != "" {
			if err := lib.ValidateBaseModel(ver.BaseModel, input.AllowedBaseModels); err != nil {
				return lib.WithStep("参数验证", fmt.Errorf("版本 %d: 基础模型无效: %w", i+1, err))
			}
		}

		if ver.Version == "" {
			return lib.WithStep("参数验证", lib.NewValidationError(fmt.Sprintf("版本 %d: 版本号不能为空", i+1)))
		}
	}

	return nil
}

func uploadVersionsConcurrently(
	ctx context.Context,
	api lib.BizyAPI,
	input UploadInput,
	callback UploadCallback,
) UploadResult {
	total := len(input.Versions)
	versionList := make([]*lib.ModelVersion, total)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3)
	var mu sync.Mutex
	var uploadErrors []error
	var canceled bool

	for i, ver := range input.Versions {
		wg.Add(1)
		idx := i
		version := ver

		go func() {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if callback != nil {
				callback.OnVersionStart(idx, total, filepath.Base(version.Path))
			}

			result := uploadSingleVersion(
				ctx, api, input.ModelType, version, idx, total, callback,
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

			mu.Lock()
			versionList[idx] = result.ModelVersion
			mu.Unlock()

			if callback != nil {
				callback.OnVersionComplete(idx, total, filepath.Base(version.Path), nil)
			}
		}()
	}

	wg.Wait()

	if canceled {
		return UploadResult{
			Success:        false,
			CanceledByUser: true,
			TotalCount:     total,
		}
	}

	successVersions := make([]*lib.ModelVersion, 0, total)
	for _, mv := range versionList {
		if mv != nil {
			successVersions = append(successVersions, mv)
		}
	}

	if len(successVersions) == 0 {
		return UploadResult{
			Success:      false,
			TotalCount:   total,
			SuccessCount: 0,
			Errors:       uploadErrors,
		}
	}

	_, err := api.CommitModelV2(input.ModelName, input.ModelType, successVersions)
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

type singleVersionResult struct {
	ModelVersion *lib.ModelVersion
	Error        error
	Canceled     bool
}

func uploadSingleVersion(
	ctx context.Context,
	api lib.BizyAPI,
	modelType string,
	version VersionInput,
	index int,
	total int,
	callback UploadCallback,
) singleVersionResult {
	var coverStatusCallback func(status, message string)
	if callback != nil {
		coverStatusCallback = func(status, message string) {
			callback.OnCoverStatus(index, total, status, message)
		}
	}

	coverUrl, err := lib.UploadCover(api, version.CoverUrl, ctx, coverStatusCallback)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return singleVersionResult{Canceled: true}
		}
		return singleVersionResult{
			Error: lib.WithStep(fmt.Sprintf("版本%d封面上传", index+1), err),
		}
	}

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

	_, err = lib.UnifiedUpload(lib.UploadOptions{
		File:         file,
		Client:       api,
		ModelType:    modelType,
		Context:      ctx,
		FileIndex:    fmt.Sprintf("%d/%d", index+1, total),
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
