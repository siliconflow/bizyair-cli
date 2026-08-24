package tui

import (
	"context"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
)

func fetchMyModelsList(api lib.BizyAPI, input myModelsInputs) myModelsDoneMsg {
	ctx := context.Background()

	var modelTypes []string
	if input.typeFilter != "" {
		modelTypes = []string{input.typeFilter}
	} else {
		for _, t := range []string{"Checkpoint", "LoRA", "Controlnet", "VAE", "UNet", "Upscaler", "Detection", "Other"} {
			modelTypes = append(modelTypes, t)
		}
	}

	var baseModels []string
	if input.baseModelFilter != "" {
		baseModels = []string{input.baseModelFilter}
	}

	sort := input.sortBy
	if sort == "" {
		sort = "Recently"
	}

	resp, err := api.ListModelContext(
		ctx,
		1,
		100,
		input.searchQuery,
		sort,
		modelTypes,
		baseModels,
	)
	if err != nil {
		return myModelsDoneMsg{
			err: lib.WithStep(i18n.T("step.list_models"), err),
		}
	}

	return myModelsDoneMsg{
		models: resp.Data.List,
		total:  resp.Data.Total,
	}
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
