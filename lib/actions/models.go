package actions

import (
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

// ListModels 查询模型列表
func ListModels(input ListModelsInput) ListModelsResult {
	// 参数验证
	if input.ApiKey == "" {
		return ListModelsResult{
			Error: lib.WithStep("查询模型列表", lib.NewValidationError("未登录或缺少API Key")),
		}
	}

	// 设置默认值
	if input.BaseDomain == "" {
		input.BaseDomain = meta.DefaultDomain
	}
	if input.Current == 0 {
		input.Current = 1
	}
	if input.PageSize == 0 {
		input.PageSize = 100
	}
	if input.Sort == "" {
		input.Sort = "Recently"
	}

	// 构建模型类型列表
	var modelTypes []string
	if input.ModelType != "" {
		modelTypes = []string{input.ModelType}
	} else if len(input.ModelTypes) > 0 {
		modelTypes = input.ModelTypes
	} else {
		// 如果未指定类型，查询所有类型
		for _, t := range meta.ModelTypes {
			modelTypes = append(modelTypes, string(t))
		}
	}

	// 调用API
	client := lib.NewClient(input.BaseDomain, input.ApiKey)
	resp, err := client.ListModel(
		input.Current,
		input.PageSize,
		input.Keyword,
		input.Sort,
		modelTypes,
		input.BaseModels,
	)
	if err != nil {
		return ListModelsResult{
			Error: lib.WithStep("查询模型列表", err),
		}
	}

	return ListModelsResult{
		Models: resp.Data.List,
		Total:  resp.Data.Total,
	}
}

// GetModelDetail 获取模型详情
func GetModelDetail(apiKey, baseDomain string, modelId int64) ModelDetailResult {
	if apiKey == "" {
		return ModelDetailResult{
			Error: lib.WithStep("查询模型详情", lib.NewValidationError("未登录或缺少API Key")),
		}
	}

	if baseDomain == "" {
		baseDomain = meta.DefaultDomain
	}

	client := lib.NewClient(baseDomain, apiKey)
	resp, err := client.GetBizyModelDetail(modelId)
	if err != nil {
		return ModelDetailResult{
			Error: lib.WithStep("查询模型详情", err),
		}
	}

	if resp == nil || resp.Data.Id == 0 {
		return ModelDetailResult{
			Error: lib.WithStep("查询模型详情", lib.NewValidationError("未获取到模型详情")),
		}
	}

	return ModelDetailResult{
		Detail: &resp.Data,
	}
}

// DeleteModel 删除模型
func DeleteModel(apiKey, baseDomain string, modelId int64) DeleteModelResult {
	if apiKey == "" {
		return DeleteModelResult{
			Success: false,
			Error:   lib.WithStep("删除模型", lib.NewValidationError("未登录或缺少API Key")),
		}
	}

	if baseDomain == "" {
		baseDomain = meta.DefaultDomain
	}

	client := lib.NewClient(baseDomain, apiKey)
	_, err := client.DeleteBizyModelById(modelId)
	if err != nil {
		return DeleteModelResult{
			Success: false,
			Error:   lib.WithStep("删除模型", err),
		}
	}

	return DeleteModelResult{
		Success: true,
	}
}
