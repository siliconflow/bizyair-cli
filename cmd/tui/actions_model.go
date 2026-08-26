package tui

import (
	"context"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

func fetchMyModelsList(api lib.BizyAPI, apiKey, baseDomain string, input myModelsInputs) myModelsDoneMsg {
	var baseModels []string
	if input.baseModelFilter != "" {
		baseModels = []string{input.baseModelFilter}
	}

	result := actions.ListModels(api, actions.ListModelsInput{
		Context:    context.Background(),
		ApiKey:     apiKey,
		BaseDomain: baseDomain,
		ModelType:  input.typeFilter,
		BaseModels: baseModels,
		Keyword:    input.search.Value(),
		Sort:       input.sortBy,
		Current:    1,
		PageSize:   100,
	})
	if result.Error != nil {
		return myModelsDoneMsg{err: result.Error}
	}
	return myModelsDoneMsg{models: result.Models, total: result.Total}
}

func fetchModelDetail(api lib.BizyAPI, modelId int64) modelDetailDoneMsg {
	resp, err := api.GetBizyModelDetailContext(context.Background(), modelId)
	if err != nil {
		return modelDetailDoneMsg{
			err: lib.WithStep(i18n.T("step.model_detail"), err),
		}
	}

	if resp == nil || resp.Data.Id == 0 {
		return modelDetailDoneMsg{
			err: lib.WithStep(i18n.T("step.model_detail"), i18n.NewError("error.model.detail_missing", nil, nil)),
		}
	}

	return modelDetailDoneMsg{
		detail: &resp.Data,
	}
}

func deleteModelCmd(api lib.BizyAPI, modelId int64) modelDeletedMsg {
	_, err := api.DeleteBizyModelByIdContext(context.Background(), modelId)
	if err != nil {
		return modelDeletedMsg{
			success: false,
			err:     lib.WithStep(i18n.T("step.delete_model"), err),
		}
	}

	return modelDeletedMsg{
		success: true,
	}
}

func toggleModelPublicCmd(api lib.BizyAPI, versionIDs []int64, public bool) modelPublicToggledMsg {
	err := actions.ToggleModelPublicContext(context.Background(), api, versionIDs, public)
	if err != nil {
		return modelPublicToggledMsg{
			success: false,
			err:     lib.WithStep(i18n.T("step.toggle_model_public"), err),
		}
	}

	return modelPublicToggledMsg{
		success: true,
	}
}
