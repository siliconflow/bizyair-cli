package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

// modelUploadResult 单个模型的上传结果
type modelUploadResult struct {
	ModelName      string
	ModelType      string
	Success        bool
	CanceledByUser bool
	Error          error
	VersionSuccess int
	VersionTotal   int
}

type yamlModelUploadFunc func(
	context.Context,
	string,
	string,
	string,
	string,
	[]config.YamlVersion,
) modelUploadResult

// uploadFromYaml 从 YAML 配置文件批量上传模型
func uploadFromYaml(ctx context.Context, yamlPath string, args *config.Argument) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// 1. 加载 YAML 配置
	fmt.Fprintln(os.Stdout, i18n.T("cli.batch.loading", map[string]any{"Path": yamlPath}))
	cfg, err := config.LoadYamlConfig(yamlPath)
	if err != nil {
		return i18n.NewError("cli.batch.load_failed", nil, err)
	}

	// 2. 规范化路径（相对路径转为基于 YAML 文件所在目录的路径）
	yamlDir := filepath.Dir(yamlPath)
	if err := config.NormalizeModelPaths(cfg, yamlDir); err != nil {
		return i18n.NewError("cli.batch.normalize_failed", nil, err)
	}

	// 3. 验证配置
	if err := config.ValidateYamlConfig(cfg); err != nil {
		return i18n.NewError("cli.batch.validation_failed", nil, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	// 4. 获取 API Key
	apiKey := args.ApiKey
	if apiKey == "" {
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return i18n.NewError("error.auth.not_logged_in", nil, err)
		}
	}

	results, err := uploadYamlModels(ctx, cfg.Models, args, apiKey, processModelUpload)
	if err != nil {
		return err
	}

	// 5. 显示汇总结果
	displayBatchUploadSummary(results)

	// 6. 根据结果决定退出码
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	if successCount == 0 {
		return i18n.NewError("cli.batch.all_failed", nil, nil)
	}

	return nil
}

func uploadYamlModels(
	ctx context.Context,
	models []config.YamlModel,
	args *config.Argument,
	apiKey string,
	uploadModel yamlModelUploadFunc,
) ([]modelUploadResult, error) {
	// 开始批量上传
	totalModels := len(models)
	fmt.Fprintf(os.Stdout, "\n%s\n", i18n.TN("cli.batch.start", totalModels, map[string]any{"Count": totalModels}))
	fmt.Fprintln(os.Stdout, strings.Repeat("=", 40))

	// 记录所有模型的上传结果
	results := make([]modelUploadResult, 0, totalModels)

	// 串行上传每个模型（避免过多并发）
	for i, model := range models {
		if err := ctx.Err(); err != nil {
			return results, err
		}

		fmt.Fprintf(os.Stdout, "\n%s\n", i18n.T("cli.batch.uploading_model", map[string]any{
			"Current": i + 1, "Total": totalModels, "Name": model.Name, "Type": model.Type,
		}))
		fmt.Fprintln(os.Stdout, i18n.TN("cli.batch.version_count", len(model.Versions), map[string]any{"Count": len(model.Versions)}))

		// 自动递增版本号
		versions := config.AutoIncrementVersionNames(model.Versions)

		// 转换为 VersionInput 并执行上传
		result := uploadModel(ctx, apiKey, args.BaseDomain, model.Name, model.Type, versions)
		results = append(results, result)
		if result.CanceledByUser {
			return results, context.Canceled
		}
		if err := ctx.Err(); err != nil {
			return results, err
		}

		// 显示结果
		if result.Success {
			fmt.Fprintf(os.Stdout, "\n%s\n", i18n.T("cli.batch.model_success", map[string]any{
				"Name": result.ModelName, "Success": result.VersionSuccess, "Total": result.VersionTotal,
			}))
			// 显示模型详情
			displayUploadedModelDetail(ctx, apiKey, args.BaseDomain, result.ModelName, result.ModelType)
		} else {
			fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("cli.batch.model_failed", map[string]any{"Name": result.ModelName, "Cause": result.Error}))
		}
	}

	if err := ctx.Err(); err != nil {
		return results, err
	}

	return results, nil
}

// processModelUpload 处理单个模型的上传（包括转换和上传）
func processModelUpload(
	ctx context.Context,
	apiKey string,
	baseDomain string,
	modelName string,
	modelType string,
	versions []config.YamlVersion,
) modelUploadResult {
	// 转换为 VersionInput
	versionInputs := make([]actions.VersionInput, len(versions))
	for j, ver := range versions {
		// 获取介绍文本
		intro, err := ver.GetIntroduction()
		if err != nil {
			return modelUploadResult{
				ModelName:    modelName,
				ModelType:    modelType,
				Success:      false,
				Error:        i18n.NewError("cli.batch.version_intro_failed", map[string]any{"Version": j + 1}, err),
				VersionTotal: len(versions),
			}
		}

		versionInputs[j] = actions.VersionInput{
			Version:      ver.Name,
			Path:         ver.ModelPath,
			BaseModel:    ver.BaseModel,
			Introduction: intro,
			CoverUrl:     ver.GetCoverInput(), // cover_path 或 cover_url
			Public:       ver.GetPublic(),
		}
	}

	// 执行上传
	return uploadSingleModelFromYaml(ctx, apiKey, baseDomain, modelName, modelType, versionInputs)
}

// uploadSingleModelFromYaml 上传单个模型（从 YAML 配置）
func uploadSingleModelFromYaml(
	ctx context.Context,
	apiKey string,
	baseDomain string,
	modelName string,
	modelType string,
	versions []actions.VersionInput,
) modelUploadResult {
	// 创建客户端
	client := lib.NewClient(baseDomain, apiKey)

	// 从API获取基础模型列表
	allowedModels := []string{}
	resp, err := client.GetBaseModelTypesContext(ctx)
	if err != nil {
		return modelUploadResult{
			ModelName:      modelName,
			ModelType:      modelType,
			Success:        false,
			CanceledByUser: errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled),
			Error:          i18n.NewError("error.base_models.fetch_failed", nil, err),
		}
	}
	if resp.Data != nil {
		for _, item := range resp.Data {
			allowedModels = append(allowedModels, item.Value)
		}
	}

	// 准备上传输入
	input := actions.UploadInput{
		Context:           ctx,
		ApiKey:            apiKey,
		BaseDomain:        baseDomain,
		ModelType:         modelType,
		ModelName:         modelName,
		Versions:          versions,
		AllowedBaseModels: allowedModels, // 传入从API获取的列表
	}

	// 创建回调
	callback := &cliUploadCallback{}

	// 执行上传
	fmt.Fprintln(os.Stdout, i18n.TN("cli.upload.start", len(versions), map[string]any{"Count": len(versions), "Concurrency": 3}))

	uploadResult := actions.ExecuteUpload(client, input, callback)

	// 返回结果
	return modelUploadResult{
		ModelName:      modelName,
		ModelType:      modelType,
		Success:        uploadResult.Success,
		CanceledByUser: uploadResult.CanceledByUser,
		Error:          combineErrors(uploadResult.Errors),
		VersionSuccess: uploadResult.SuccessCount,
		VersionTotal:   uploadResult.TotalCount,
	}
}

// displayBatchUploadSummary 显示批量上传的汇总结果
func displayBatchUploadSummary(results []modelUploadResult) {
	fmt.Fprintln(os.Stdout, "\n"+strings.Repeat("=", 40))
	fmt.Fprintln(os.Stdout, i18n.T("cli.batch.complete"))

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}

	fmt.Fprintln(os.Stdout, i18n.TN("cli.batch.total", len(results), map[string]any{"Count": len(results)}))
	fmt.Fprintln(os.Stdout, i18n.TN("cli.batch.succeeded", successCount, map[string]any{"Count": successCount}))
	fmt.Fprintln(os.Stdout, i18n.TN("cli.batch.failed_count", failCount, map[string]any{"Count": failCount}))

	// 如果有失败，显示失败详情
	if failCount > 0 {
		fmt.Fprintln(os.Stdout, "\n"+i18n.T("cli.batch.failure_details"))
		for _, r := range results {
			if !r.Success {
				errorMsg := i18n.T("error.unknown")
				if r.Error != nil {
					errorMsg = r.Error.Error()
				}
				fmt.Fprintf(os.Stderr, "  - %s (%s): %s\n", r.ModelName, r.ModelType, errorMsg)
			}
		}
	}
}

// combineErrors 合并多个错误为一个错误
func combineErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}
	if len(errors) == 1 {
		return errors[0]
	}

	// 合并多个错误
	messages := make([]string, len(errors))
	for i, err := range errors {
		messages[i] = err.Error()
	}
	return fmt.Errorf("%s", strings.Join(messages, "; "))
}
