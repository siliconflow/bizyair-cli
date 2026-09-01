package actions

import (
	"context"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

func ListModelzooEndpoints(ctx context.Context, api lib.BizyAPI, keyword, billingUnit, sort string, showDeprecated bool, current, pageSize int) ModelzooEndpointsResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if current < 1 {
		current = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	listResp, err := api.GetModelzooListContext(ctx, current, pageSize, keyword, sort)
	if err != nil {
		return ModelzooEndpointsResult{
			Error: lib.WithStep(i18n.T("step.list_modelzoo_endpoints", nil), err),
		}
	}

	priceResp, priceErr := api.GetModelzooPriceTableFlatContext(ctx, keyword, billingUnit, sort, showDeprecated)
	_ = priceErr

	priceByEndpoint := make(map[string]*lib.ModelzooModelFlat)
	if priceResp != nil {
		for i := range priceResp.Data.Models {
			priceByEndpoint[priceResp.Data.Models[i].Endpoint] = &priceResp.Data.Models[i]
		}
	}

	total := 0
	var models []lib.ModelzooModelFlat
	if listResp != nil {
		total = listResp.Data.Total
		for _, item := range listResp.Data.List {
			flat := lib.ModelzooModelFlat{
				Endpoint:     item.Endpoint,
				DisplayName:  item.DisplayName,
				Manufacturer: item.Manufacturer,
				ModelName:    item.ModelName,
				Category:     item.Category,
				MinCredits:   item.MinCredits,
				IconURL:      item.IconURL,
				Description:  item.Description,
				ModelVersion: item.Edition,
				Series:       lib.SeriesFromEndpoint(item.Endpoint), // 推荐优先取价格表接口字段,此处先以 endpoint 兜底
			}
			if pm, ok := priceByEndpoint[item.Endpoint]; ok {
				flat.BillingUnit = pm.BillingUnit
				flat.PriceTables = pm.PriceTables
				flat.SimplePriceText = pm.SimplePriceText
				flat.IndicativePrice = pm.IndicativePrice
				if pm.Series != "" {
					flat.Series = pm.Series
				}
				if flat.ModelVersion == "" {
					flat.ModelVersion = pm.ModelVersion
				}
				flat.Tags = pm.Tags
				flat.Deprecated = pm.Deprecated
				flat.SubCategory = pm.SubCategory
				if pm.MinCredits > 0 && flat.MinCredits == 0 {
					flat.MinCredits = pm.MinCredits
				}
			}
			models = append(models, flat)
		}
	}
	if total < len(models) {
		total = len(models)
	}
	return ModelzooEndpointsResult{
		Models: models,
		Total:  total,
	}
}

func ListModelzooTags(ctx context.Context, api lib.BizyAPI) ModelzooTagsResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.GetModelzooTagsContext(ctx)
	if err != nil {
		return ModelzooTagsResult{
			Error: lib.WithStep(i18n.T("step.list_modelzoo_tags", nil), err),
		}
	}
	var tags []lib.ModelzooTag
	if resp != nil {
		tags = resp.Data.Tags
	}
	return ModelzooTagsResult{
		Tags: tags,
	}
}

func ListModelzooCategories(ctx context.Context, api lib.BizyAPI) ModelzooCategoriesResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.GetModelzooCategoriesContext(ctx, "", false)
	if err != nil {
		return ModelzooCategoriesResult{
			Error: lib.WithStep(i18n.T("step.list_modelzoo_categories", nil), err),
		}
	}
	var categories []lib.ModelzooCategoryItem
	if resp != nil {
		categories = resp.Data.List
	}
	return ModelzooCategoriesResult{
		Categories: categories,
	}
}

func GetEndpointDetail(ctx context.Context, api lib.BizyAPI, endpoint string) ModelzooEndpointDetailResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.GetModelzooEndpointDetailContext(ctx, endpoint)
	if err != nil {
		return ModelzooEndpointDetailResult{
			Error: lib.WithStep(i18n.T("step.get_endpoint_detail", nil), err),
		}
	}
	var detail *lib.ModelzooEndpointDetail
	if resp != nil {
		detail = &resp.Data
	}
	return ModelzooEndpointDetailResult{
		Detail: detail,
	}
}

func GetPriceTable(ctx context.Context, api lib.BizyAPI, endpoint string) PriceTableResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.GetModelzooPriceTableContext(ctx, endpoint)
	if err != nil {
		return PriceTableResult{
			Error: lib.WithStep(i18n.T("step.get_price_table", nil), err),
		}
	}
	var pts []lib.PriceTable
	if resp != nil {
		pts = resp.Data.PriceTables
	}
	return PriceTableResult{
		PriceTables: pts,
	}
}

func CreateTask(ctx context.Context, api lib.BizyAPI, endpoint string, params map[string]any) CreateTaskResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.CreateModelZooTaskContext(ctx, endpoint, params)
	if err != nil {
		return CreateTaskResult{
			Error: lib.WithStep(i18n.T("step.create_task", nil), err),
		}
	}
	var requestID string
	if resp != nil {
		requestID = resp.Data.RequestID
	}
	return CreateTaskResult{
		RequestID: requestID,
	}
}

func GetTaskStatus(ctx context.Context, api lib.BizyAPI, requestID string) TaskStatusResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.GetModelZooTaskStatusContext(ctx, requestID)
	if err != nil {
		return TaskStatusResult{
			Error: lib.WithStep(i18n.T("step.get_task_status", nil), err),
		}
	}
	var status *lib.ModelZooTaskStatusResp
	if resp != nil {
		status = &resp.Data
	}
	return TaskStatusResult{
		Status: status,
	}
}

func GetTaskCancel(ctx context.Context, api lib.BizyAPI, requestID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := api.CancelModelZooTaskContext(ctx, requestID)
	if err != nil {
		return lib.WithStep(i18n.T("step.cancel_task", nil), err)
	}
	return nil
}
