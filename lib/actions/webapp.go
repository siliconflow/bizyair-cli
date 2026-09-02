package actions

import (
	"context"
	"strings"
	"sync"

	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
)

// ListAIApplications 获取社区 AI 应用列表。
func ListAIApplications(ctx context.Context, api lib.BizyAPI, keyword, sort string, baseModels []string, current, pageSize int) AIApplicationsResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if current < 1 {
		current = 1
	}
	if pageSize < 1 {
		pageSize = 24
	}
	_ = baseModels
	_ = sort
	resp, err := api.ListWebAppsContext(ctx, current, pageSize, keyword)
	if err != nil {
		return AIApplicationsResult{
			Error: lib.WithStep(i18n.T("step.list_ai_apps", nil), err),
		}
	}
	total := 0
	var apps []*lib.BizyModelInfo
	if resp != nil {
		total = resp.Data.Total
		apps = resp.Data.List
	}
	if total < len(apps) {
		total = len(apps)
	}
	return AIApplicationsResult{
		Apps:  apps,
		Total: total,
	}
}

// EnrichAIApplications 并发回填列表应用的发布时间。
// 社区列表接口不返回 created_at，需逐个拉取版本详情；
// 单条失败不中断，未取到时间的应用保持原值（显示 "-"）。
func EnrichAIApplications(ctx context.Context, api lib.BizyAPI, apps []*lib.BizyModelInfo, workers int) {
	if ctx == nil {
		ctx = context.Background()
	}
	if workers < 1 {
		workers = 8
	}
	if len(apps) == 0 || workers > len(apps) {
		workers = len(apps)
	}
	if workers < 1 {
		return
	}
	var wg sync.WaitGroup
	jobs := make(chan *lib.BizyModelInfo)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for app := range jobs {
				enrichAIApplicationCreatedAt(ctx, api, app)
			}
		}()
	}
	for _, app := range apps {
		jobs <- app
	}
	close(jobs)
	wg.Wait()
}

// enrichAIApplicationCreatedAt 用版本详情/WebApp 详情回填单个应用的发布时间。
func enrichAIApplicationCreatedAt(ctx context.Context, api lib.BizyAPI, app *lib.BizyModelInfo) {
	if app == nil || app.CreatedAt != "" || len(app.Versions) == 0 || app.Versions[0] == nil {
		return
	}
	versionID := app.Versions[0].Id
	if versionID > 0 {
		if resp, err := api.GetWebappVersionDetailContext(ctx, versionID); err == nil && resp != nil {
			if resp.Data.CreatedAt != "" {
				app.CreatedAt = resp.Data.CreatedAt
				return
			}
		}
		if resp, err := api.GetWebappDetailByVersionContext(ctx, versionID); err == nil && resp != nil {
			if resp.Data.CreatedAt != "" {
				app.CreatedAt = resp.Data.CreatedAt
			}
		}
	}
}

// ResolveAIAppByNameResult 按名称解析 AI 应用的结果。
type ResolveAIAppByNameResult struct {
	App        *lib.BizyModelInfo
	Candidates []*lib.BizyModelInfo
	Error      error
}

// ResolveAIAppByName 按名称（大小写不敏感）在社区列表中精确匹配应用。
// 无匹配时返回错误；多个同名应用时返回候选列表交由调用方提示用户区分。
func ResolveAIAppByName(ctx context.Context, api lib.BizyAPI, name string) ResolveAIAppByNameResult {
	name = strings.TrimSpace(name)
	if name == "" {
		return ResolveAIAppByNameResult{
			Error: i18n.NewError("cli.app.id_required", nil, nil),
		}
	}
	resp, err := api.ListWebAppsContext(ctx, 1, 100, name)
	if err != nil {
		return ResolveAIAppByNameResult{
			Error: lib.WithStep(i18n.T("step.list_ai_apps", nil), err),
		}
	}
	var matched []*lib.BizyModelInfo
	if resp != nil {
		for _, app := range resp.Data.List {
			if app != nil && strings.EqualFold(app.Name, name) {
				matched = append(matched, app)
			}
		}
	}
	switch len(matched) {
	case 0:
		return ResolveAIAppByNameResult{
			Error: i18n.NewError("cli.app.not_found", map[string]any{"ID": name}, nil),
		}
	case 1:
		return ResolveAIAppByNameResult{App: matched[0]}
	default:
		return ResolveAIAppByNameResult{Candidates: matched}
	}
}

// GetWebAppVersionDetail 获取 AI 应用版本详情。
func GetWebAppVersionDetail(ctx context.Context, api lib.BizyAPI, versionID int64) WebAppVersionDetailResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.GetWebappVersionDetailContext(ctx, versionID)
	if err != nil {
		return WebAppVersionDetailResult{
			Error: lib.WithStep(i18n.T("step.get_webapp_detail", nil), err),
		}
	}
	var detail *lib.WebAppVersionDetail
	if resp != nil {
		detail = &resp.Data
	}
	return WebAppVersionDetailResult{
		Detail: detail,
	}
}

// GetAIAppDetail 获取 AI 应用详情：版本详情 + 工作流详情 + webapp 详情合并。
// 版本 ID（web_app_id）即可用于任务创建，ref_bizy_model_id 提供作者/简介等展示信息，
// webapp 详情（GET /v1/webapp/{bizy_model_id}）提供 input_nodes 输入节点定义。
// GetWebAppDetail 是 GetAIAppDetail 的别名（CLI 以 app_id/版本 ID 作为入参）。
func GetWebAppDetail(ctx context.Context, api lib.BizyAPI, versionID int64) WebAppDetailResult {
	return GetAIAppDetail(ctx, api, versionID)
}

func GetAIAppDetail(ctx context.Context, api lib.BizyAPI, versionID int64) WebAppDetailResult {
	if ctx == nil {
		ctx = context.Background()
	}
	versionResult := GetWebAppVersionDetail(ctx, api, versionID)
	if versionResult.Error != nil {
		return WebAppDetailResult{Error: versionResult.Error}
	}
	if versionResult.Detail == nil {
		return WebAppDetailResult{Detail: nil}
	}
	version := versionResult.Detail

	var wfResp *lib.Response[lib.BizyModelDetail]
	if version.RefBizyModelId > 0 {
		wfResp, _ = api.GetBizyModelDetailContext(ctx, version.RefBizyModelId)
	}

	detail := &lib.WebAppDetail{
		Id:          version.Id,
		BizyModelId: version.BizyModelId,
		BaseModel:   version.BaseModel,
		Description: version.Description,
		CreatedAt:   version.CreatedAt,
		UpdatedAt:   version.UpdatedAt,
		CoverUrls:   version.CoverUrls,
	}
	if wfResp != nil {
		detail.Name = wfResp.Data.Name
		detail.NickName = wfResp.Data.UserName
		detail.Creator = wfResp.Data.UserId
		if detail.BaseModel == "" && len(wfResp.Data.Versions) > 0 {
			detail.BaseModel = wfResp.Data.Versions[0].BaseModel
		}
		if detail.Description == "" && len(wfResp.Data.Versions) > 0 {
			detail.Description = wfResp.Data.Versions[0].Description
		}
		if detail.CreatedAt == "" {
			detail.CreatedAt = wfResp.Data.CreatedAt
		}
	}
	// webapp 详情提供 input_nodes 输入节点定义与应用权威信息
	// （GET /v1/webapp/{web_app_id}/detail，web_app_id 即版本 ID）。
	var webAppResp *lib.Response[lib.WebAppDetail]
	webAppResp, _ = api.GetWebappDetailByVersionContext(ctx, version.Id)
	if webAppResp != nil {
		d := webAppResp.Data
		detail.InputNodes = d.InputNodes
		if d.Name != "" {
			detail.Name = d.Name
		}
		if d.NickName != "" {
			detail.NickName = d.NickName
		}
		if d.Creator != "" {
			detail.Creator = d.Creator
		}
		if detail.Id == 0 && d.Id > 0 {
			detail.Id = d.Id
		}
		if detail.BaseModel == "" && d.BaseModel != "" {
			detail.BaseModel = d.BaseModel
		}
		if detail.Description == "" && d.Description != "" {
			detail.Description = d.Description
		}
		if detail.CreatedAt == "" && d.CreatedAt != "" {
			detail.CreatedAt = d.CreatedAt
		}
		if len(d.CoverUrls) > 0 {
			detail.CoverUrls = d.CoverUrls
		}
	}
	return WebAppDetailResult{
		Detail: detail,
	}
}

// CreateWebAppTask 异步创建 AI 应用任务，返回 task_id 与初始状态。
func CreateWebAppTask(ctx context.Context, api lib.BizyAPI, req lib.WebAppTaskCreateReq) CreateWebAppTaskResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.CreateWebappComfyTaskContext(ctx, req)
	if err != nil {
		return CreateWebAppTaskResult{
			Error: lib.WithStep(i18n.T("step.create_task", nil), err),
		}
	}
	var result CreateWebAppTaskResult
	if resp != nil {
		result.TaskID = resp.Data.TaskID
		result.TaskStatus = resp.Data.TaskStatus
	}
	return result
}

// GetWebAppTaskStatus 按 comfy task_id 查询 AI 应用任务状态。
func GetWebAppTaskStatus(ctx context.Context, api lib.BizyAPI, taskID int64) WebAppTaskStatusResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.GetComfyTaskDetailContext(ctx, taskID)
	if err != nil {
		return WebAppTaskStatusResult{
			Error: lib.WithStep(i18n.T("step.get_task_status", nil), err),
		}
	}
	var status *lib.ComfyTaskStatusData
	if resp != nil {
		status = &resp.Data
	}
	return WebAppTaskStatusResult{
		Status: status,
	}
}

// GetWebAppTaskOutputs 按 request_id 获取 AI 应用任务输出 URL 列表。
func GetWebAppTaskOutputs(ctx context.Context, api lib.BizyAPI, requestID string) WebAppTaskOutputsResult {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := api.GetWebappTaskOutputsContext(ctx, requestID)
	if err != nil {
		return WebAppTaskOutputsResult{
			Error: lib.WithStep(i18n.T("step.get_task_outputs", nil), err),
		}
	}
	var outputs []lib.WebAppTaskOutput
	if resp != nil {
		outputs = resp.Data.Outputs
	}
	return WebAppTaskOutputsResult{
		Outputs: outputs,
	}
}
