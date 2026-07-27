package actions

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

func ExecuteUpload(api lib.BizyAPI, input UploadInput, callback UploadCallback) UploadResult {
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	total := len(input.Versions)

	if errors.Is(ctx.Err(), context.Canceled) {
		return canceledUploadResult(total, 0)
	}

	if input.ApiKey == "" {
		return UploadResult{
			Success: false,
			Errors:  []error{lib.WithStep(i18n.T("step.upload"), i18n.NewError("error.auth.api_key_missing", nil, nil))},
		}
	}

	if err := validateUploadInput(input); err != nil {
		return UploadResult{
			Success: false,
			Errors:  []error{err},
		}
	}

	exists, err := api.CheckModelExistsContext(ctx, input.ModelName, input.ModelType)
	if err != nil {
		if uploadWasCanceled(ctx, err) {
			return canceledUploadResult(total, 0)
		}
		return UploadResult{
			Success: false,
			Errors:  []error{lib.WithStep(i18n.T("step.check_model"), err)},
		}
	}
	if exists {
		return UploadResult{
			Success: false,
			Errors: []error{
				lib.WithStep(i18n.T("step.check_model"),
					i18n.NewError("validation.model_exists", map[string]any{"Name": input.ModelName}, nil)),
			},
		}
	}

	return uploadVersionsConcurrently(ctx, api, input, callback)
}

func validateUploadInput(input UploadInput) error {
	if err := lib.ValidateModelType(input.ModelType); err != nil {
		return lib.WithStep(i18n.T("step.validate_parameters"), i18n.NewError("validation.model_type_invalid", nil, err))
	}

	if err := lib.ValidateModelName(input.ModelName); err != nil {
		return lib.WithStep(i18n.T("step.validate_parameters"), i18n.NewError("validation.model_name_invalid", nil, err))
	}

	if len(input.Versions) == 0 {
		return lib.WithStep(i18n.T("step.validate_parameters"), i18n.NewError("validation.version_required", nil, nil))
	}

	for i, ver := range input.Versions {
		if ver.Path == "" {
			return lib.WithStep(i18n.T("step.validate_parameters"), i18n.NewError("validation.version_path_required", map[string]any{"Version": i + 1}, nil))
		}

		stat, err := os.Stat(ver.Path)
		if err != nil {
			return lib.WithStep(i18n.T("step.validate_parameters"), i18n.NewError("validation.version_path_invalid", map[string]any{"Version": i + 1}, err))
		}

		if stat.IsDir() {
			return lib.WithStep(i18n.T("step.validate_parameters"), i18n.NewError("validation.version_directory_unsupported", map[string]any{"Version": i + 1}, nil))
		}

		if ver.CoverUrl == "" {
			return lib.WithStep(i18n.T("step.validate_parameters"), i18n.NewError("validation.version_cover_required", map[string]any{"Version": i + 1}, nil))
		}

		// 验证基础模型 - 使用从API获取的列表
		if ver.BaseModel != "" {
			if err := lib.ValidateBaseModel(ver.BaseModel, input.AllowedBaseModels); err != nil {
				return lib.WithStep(i18n.T("step.validate_parameters"), i18n.NewError("validation.version_base_model_invalid", map[string]any{"Version": i + 1}, err))
			}
		}

		if ver.Version == "" {
			return lib.WithStep(i18n.T("step.validate_parameters"), i18n.NewError("validation.version_name_required", map[string]any{"Version": i + 1}, nil))
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

	recordContextError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		if errors.Is(err, context.Canceled) {
			canceled = true
			return
		}
		uploadErrors = append(uploadErrors, lib.WithStep(i18n.T("step.upload"), err))
	}

	for i, ver := range input.Versions {
		wg.Add(1)
		idx := i
		version := ver

		go func() {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				recordContextError(ctx.Err())
				return
			}
			defer func() { <-semaphore }()

			if err := ctx.Err(); err != nil {
				recordContextError(err)
				return
			}

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

	successVersions := make([]*lib.ModelVersion, 0, total)
	for _, mv := range versionList {
		if mv != nil {
			successVersions = append(successVersions, mv)
		}
	}

	if canceled || errors.Is(ctx.Err(), context.Canceled) {
		return canceledUploadResult(total, len(successVersions))
	}

	if err := ctx.Err(); err != nil {
		if len(uploadErrors) == 0 {
			uploadErrors = append(uploadErrors, lib.WithStep(i18n.T("step.upload"), err))
		}
		return UploadResult{
			Success:      false,
			TotalCount:   total,
			SuccessCount: len(successVersions),
			Errors:       uploadErrors,
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

	_, err := api.CommitModelV2Context(ctx, input.ModelName, input.ModelType, successVersions)
	if err != nil {
		if uploadWasCanceled(ctx, err) {
			return canceledUploadResult(total, len(successVersions))
		}
		return UploadResult{
			Success:      false,
			SuccessCount: len(successVersions),
			TotalCount:   total,
			Errors:       []error{lib.WithStep(i18n.T("step.commit_model"), err)},
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

func uploadWasCanceled(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) ||
		(ctx != nil && errors.Is(ctx.Err(), context.Canceled))
}

func canceledUploadResult(total, success int) UploadResult {
	return UploadResult{
		Success:        false,
		SuccessCount:   success,
		TotalCount:     total,
		CanceledByUser: true,
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
			Error: lib.WithStep(i18n.T("step.version_cover_upload", map[string]any{"Version": index + 1}), err),
		}
	}

	stat, err := os.Stat(version.Path)
	if err != nil {
		return singleVersionResult{
			Error: lib.WithStep(i18n.T("step.version_read_file", map[string]any{"Version": index + 1}), err),
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
		File:      file,
		Client:    api,
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
			Error: lib.WithStep(i18n.T("step.version_file_upload", map[string]any{"Version": index + 1}), err),
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
