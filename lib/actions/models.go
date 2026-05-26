package actions

import (
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

func ListModels(api lib.BizyAPI, input ListModelsInput) ListModelsResult {
	if input.ApiKey == "" {
		return ListModelsResult{
			Error: lib.WithStep("查询模型列表", lib.NewValidationError("未登录或缺少API Key")),
		}
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

	var modelTypes []string
	if input.ModelType != "" {
		modelTypes = []string{input.ModelType}
	} else if len(input.ModelTypes) > 0 {
		modelTypes = input.ModelTypes
	} else {
		for _, t := range meta.ModelTypes {
			modelTypes = append(modelTypes, string(t))
		}
	}

	resp, err := api.ListModel(
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

func GetModelDetail(api lib.BizyAPI, modelId int64) ModelDetailResult {
	resp, err := api.GetBizyModelDetail(modelId)
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

func DeleteModel(api lib.BizyAPI, modelId int64) DeleteModelResult {
	_, err := api.DeleteBizyModelById(modelId)
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
