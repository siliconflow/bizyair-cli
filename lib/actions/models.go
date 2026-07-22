package actions

import (
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/meta"
)

func ListModels(api lib.BizyAPI, input ListModelsInput) ListModelsResult {
	if input.ApiKey == "" {
		return ListModelsResult{
			Error: lib.WithStep(i18n.T("step.list_models"), i18n.NewError("error.auth.api_key_missing", nil, nil)),
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
			Error: lib.WithStep(i18n.T("step.list_models"), err),
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
			Error: lib.WithStep(i18n.T("step.model_detail"), err),
		}
	}

	if resp == nil || resp.Data.Id == 0 {
		return ModelDetailResult{
			Error: lib.WithStep(i18n.T("step.model_detail"), i18n.NewError("error.model.detail_missing", nil, nil)),
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
			Error:   lib.WithStep(i18n.T("step.delete_model"), err),
		}
	}

	return DeleteModelResult{
		Success: true,
	}
}
