package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/siliconflow/bizyair-cli/config"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

// modelUploadResult 单个模型的上传结果
type modelUploadResult struct {
	ModelName      string
	ModelType      string
	Success        bool
	Error          error
	VersionSuccess int
	VersionTotal   int
}

// uploadFromYaml 从 YAML 配置文件批量上传模型
func uploadFromYaml(yamlPath string, args *config.Argument) error {
	// 1. 加载 YAML 配置
	fmt.Fprintf(os.Stdout, "正在加载配置文件: %s\n", yamlPath)
	cfg, err := config.LoadYamlConfig(yamlPath)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %w", err)
	}

	// 2. 规范化路径（相对路径转为基于 YAML 文件所在目录的路径）
	yamlDir := filepath.Dir(yamlPath)
	if err := config.NormalizeModelPaths(cfg, yamlDir); err != nil {
		return fmt.Errorf("规范化路径失败: %w", err)
	}

	// 3. 验证配置
	if err := config.ValidateYamlConfig(cfg); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 4. 获取 API Key
	apiKey := args.ApiKey
	if apiKey == "" {
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return fmt.Errorf("未登录或缺少 API Key: %w", err)
		}
	}

	// 5. 开始批量上传
	totalModels := len(cfg.Models)
	fmt.Fprintf(os.Stdout, "\n开始批量上传，共 %d 个模型\n", totalModels)
	fmt.Fprintln(os.Stdout, strings.Repeat("=", 40))

	// 记录所有模型的上传结果
	results := make([]modelUploadResult, 0, totalModels)

	// 6. 串行上传每个模型（避免过多并发）
	for i, model := range cfg.Models {
		fmt.Fprintf(os.Stdout, "\n[%d/%d] 正在上传模型: %s (%s)\n", i+1, totalModels, model.Name, model.Type)
		fmt.Fprintf(os.Stdout, "  - 共 %d 个版本\n", len(model.Versions))

		// 自动递增版本号
		versions := config.AutoIncrementVersionNames(model.Versions)

		// 转换为 VersionInput 并执行上传
		result := processModelUpload(apiKey, args.BaseDomain, model.Name, model.Type, versions, args.Overwrite)
		results = append(results, result)

		// 显示结果
		if result.Success {
			fmt.Fprintf(os.Stdout, "\n✓ 模型 '%s' 上传成功！(%d/%d 版本成功)\n",
				result.ModelName, result.VersionSuccess, result.VersionTotal)
			// 显示模型详情
			displayUploadedModelDetail(apiKey, args.BaseDomain, result.ModelName, result.ModelType)
		} else {
			fmt.Fprintf(os.Stderr, "\n✗ 模型 '%s' 上传失败: %v\n", result.ModelName, result.Error)
		}
	}

	// 7. 显示汇总结果
	displayBatchUploadSummary(results)

	// 8. 根据结果决定退出码
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("所有模型上传失败")
	}

	return nil
}

// processModelUpload 处理单个模型的上传（包括转换和上传）
func processModelUpload(
	apiKey string,
	baseDomain string,
	modelName string,
	modelType string,
	versions []config.YamlVersion,
	overwrite bool,
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
				Error:        fmt.Errorf("读取版本 %d 介绍失败: %w", j+1, err),
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
	return uploadSingleModelFromYaml(apiKey, baseDomain, modelName, modelType, versionInputs, overwrite)
}

// uploadSingleModelFromYaml 上传单个模型（从 YAML 配置）
func uploadSingleModelFromYaml(
	apiKey string,
	baseDomain string,
	modelName string,
	modelType string,
	versions []actions.VersionInput,
	overwrite bool,
) modelUploadResult {
	// 准备上传输入
	input := actions.UploadInput{
		ApiKey:     apiKey,
		BaseDomain: baseDomain,
		ModelType:  modelType,
		ModelName:  modelName,
		Versions:   versions,
		Overwrite:  overwrite,
	}

	// 创建回调
	callback := &cliUploadCallback{}

	// 执行上传
	fmt.Fprintf(os.Stdout, "开始上传 %d 个文件（并发数：3）\n", len(versions))
	uploadResult := actions.ExecuteUpload(input, callback)

	// 返回结果
	return modelUploadResult{
		ModelName:      modelName,
		ModelType:      modelType,
		Success:        uploadResult.Success,
		Error:          combineErrors(uploadResult.Errors),
		VersionSuccess: uploadResult.SuccessCount,
		VersionTotal:   uploadResult.TotalCount,
	}
}

// displayBatchUploadSummary 显示批量上传的汇总结果
func displayBatchUploadSummary(results []modelUploadResult) {
	fmt.Fprintln(os.Stdout, "\n"+strings.Repeat("=", 40))
	fmt.Fprintln(os.Stdout, "批量上传完成！")

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}

	fmt.Fprintf(os.Stdout, "总计: %d 个模型\n", len(results))
	fmt.Fprintf(os.Stdout, "  ✓ 成功: %d 个\n", successCount)
	fmt.Fprintf(os.Stdout, "  ✗ 失败: %d 个\n", failCount)

	// 如果有失败，显示失败详情
	if failCount > 0 {
		fmt.Fprintln(os.Stdout, "\n失败详情：")
		for _, r := range results {
			if !r.Success {
				errorMsg := "未知错误"
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
