package actions

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/siliconflow/bizyair-cli/lib"
)

type webappFakeAPI struct {
	lib.BizyAPI
	versionResp  *lib.Response[lib.WebAppVersionDetail]
	modelResp    *lib.Response[lib.BizyModelDetail]
	webappResp   *lib.Response[lib.WebAppDetail]
	createdReq   lib.WebAppTaskCreateReq
	listResp     *lib.Response[lib.BizyModelListResp]
	versionResps map[int64]*lib.Response[lib.WebAppVersionDetail]
}

func (a *webappFakeAPI) ListWebAppsContext(context.Context, int, int, string) (*lib.Response[lib.BizyModelListResp], error) {
	return a.listResp, nil
}

func (a *webappFakeAPI) GetWebappVersionDetailContext(_ context.Context, versionID int64) (*lib.Response[lib.WebAppVersionDetail], error) {
	if len(a.versionResps) > 0 {
		return a.versionResps[versionID], nil
	}
	return a.versionResp, nil
}

func (a *webappFakeAPI) GetBizyModelDetailContext(context.Context, int64) (*lib.Response[lib.BizyModelDetail], error) {
	return a.modelResp, nil
}

func (a *webappFakeAPI) GetWebappDetailByVersionContext(context.Context, int64) (*lib.Response[lib.WebAppDetail], error) {
	return a.webappResp, nil
}

func (a *webappFakeAPI) CreateWebappComfyTaskContext(_ context.Context, req lib.WebAppTaskCreateReq) (*lib.Response[lib.WebAppComfyTaskResp], error) {
	a.createdReq = req
	return &lib.Response[lib.WebAppComfyTaskResp]{
		Data: lib.WebAppComfyTaskResp{TaskID: 11, TaskStatus: "Pending"},
	}, nil
}

func TestGetAIAppDetailMergesWebappInputNodes(t *testing.T) {
	api := &webappFakeAPI{
		versionResp: &lib.Response[lib.WebAppVersionDetail]{
			Data: lib.WebAppVersionDetail{
				Id: 7, BizyModelId: 3, RefBizyModelId: 5, BaseModel: "base",
				Description: "desc", CreatedAt: "2024-01-01",
			},
		},
		modelResp: &lib.Response[lib.BizyModelDetail]{
			Data: lib.BizyModelDetail{Name: "wf-name", UserName: "author", UserId: "u1"},
		},
		webappResp: &lib.Response[lib.WebAppDetail]{
			Data: lib.WebAppDetail{
				Id:   7,
				Name: "App",
				InputNodes: []lib.WebAppInputNode{{
					FieldName: "prompt", VariableName: "v1", FieldType: "string", FieldLabel: "Prompt",
				}},
			},
		},
	}

	result := GetAIAppDetail(context.Background(), api, 7)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	d := result.Detail
	if d == nil {
		t.Fatal("detail is nil")
	}
	if len(d.InputNodes) != 1 || d.InputNodes[0].FieldName != "prompt" {
		t.Fatalf("InputNodes = %#v, want the webapp input node", d.InputNodes)
	}
	if d.Name != "App" {
		t.Fatalf("Name = %q, want webapp detail Name to take precedence", d.Name)
	}
	if d.Id != 7 {
		t.Fatalf("Id = %d, want 7", d.Id)
	}
}

func TestGetAIAppDetailKeepsVersionIdWhenWebappDetailMissing(t *testing.T) {
	api := &webappFakeAPI{
		versionResp: &lib.Response[lib.WebAppVersionDetail]{
			Data: lib.WebAppVersionDetail{Id: 7, BizyModelId: 3, RefBizyModelId: 5},
		},
		modelResp: &lib.Response[lib.BizyModelDetail]{
			Data: lib.BizyModelDetail{Name: "wf-name"},
		},
		webappResp: &lib.Response[lib.WebAppDetail]{
			Data: lib.WebAppDetail{Id: 9},
		},
	}

	result := GetAIAppDetail(context.Background(), api, 7)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Detail == nil {
		t.Fatal("detail is nil")
	}
	if result.Detail.Id != 7 {
		t.Fatalf("Id = %d, want version id 7 preserved", result.Detail.Id)
	}
}

func TestEnrichAIApplicationsBackfillsCreatedAt(t *testing.T) {
	api := &webappFakeAPI{
		versionResps: map[int64]*lib.Response[lib.WebAppVersionDetail]{
			10: {Data: lib.WebAppVersionDetail{Id: 10, CreatedAt: "2026-08-26T14:31:08Z"}},
			11: {Data: lib.WebAppVersionDetail{Id: 11}},
		},
	}
	apps := []*lib.BizyModelInfo{
		{Id: 1, Name: "a", Versions: []*lib.BizyModelVersion{{Id: 10}}},
		{Id: 2, Name: "b", Versions: []*lib.BizyModelVersion{{Id: 11}}},
		{Id: 3, Name: "c"}, // 无版本，跳过
		{Id: 4, Name: "d", CreatedAt: "2026-01-01", Versions: []*lib.BizyModelVersion{{Id: 12}}}, // 已有时间，跳过
	}
	EnrichAIApplications(context.Background(), api, apps, 2)
	if apps[0].CreatedAt != "2026-08-26T14:31:08Z" {
		t.Fatalf("apps[0].CreatedAt = %q, want backfilled time", apps[0].CreatedAt)
	}
	if apps[1].CreatedAt != "" {
		t.Fatalf("apps[1].CreatedAt = %q, want empty when version detail lacks time", apps[1].CreatedAt)
	}
	if apps[2].CreatedAt != "" {
		t.Fatalf("apps[2].CreatedAt = %q, want empty when app has no versions", apps[2].CreatedAt)
	}
	if apps[3].CreatedAt != "2026-01-01" {
		t.Fatalf("apps[3].CreatedAt = %q, want preserved existing time", apps[3].CreatedAt)
	}
}

func TestEnrichAIApplicationsFallsBackToWebappDetail(t *testing.T) {
	api := &webappFakeAPI{
		webappResp: &lib.Response[lib.WebAppDetail]{
			Data: lib.WebAppDetail{Id: 10, CreatedAt: "2026-08-26T15:00:00Z"},
		},
	}
	apps := []*lib.BizyModelInfo{
		{Id: 1, Name: "a", Versions: []*lib.BizyModelVersion{{Id: 10}}},
	}
	// versionResp 为 nil 且返回 nil：先失败，回退 webapp 详情补全。
	EnrichAIApplications(context.Background(), api, apps, 2)
	if apps[0].CreatedAt != "2026-08-26T15:00:00Z" {
		t.Fatalf("apps[0].CreatedAt = %q, want fallback time from webapp detail", apps[0].CreatedAt)
	}
}

func TestEnrichAIApplicationsEmptyInput(t *testing.T) {
	api := &webappFakeAPI{}
	EnrichAIApplications(context.Background(), api, nil, 8)
	EnrichAIApplications(context.Background(), api, []*lib.BizyModelInfo{}, 8)
}

func TestResolveAIAppByNameExactMatchCaseInsensitive(t *testing.T) {
	api := &webappFakeAPI{
		listResp: &lib.Response[lib.BizyModelListResp]{
			Data: lib.BizyModelListResp{
				List: []*lib.BizyModelInfo{
					{Id: 1, Name: "LTX 2.5 Image-to-Video", Versions: []*lib.BizyModelVersion{{Id: 10}}},
					{Id: 2, Name: "Other App", Versions: []*lib.BizyModelVersion{{Id: 20}}},
				},
			},
		},
	}
	result := ResolveAIAppByName(context.Background(), api, "ltx 2.5 image-to-video")
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.App == nil || result.App.Id != 1 {
		t.Fatalf("App = %#v, want id 1", result.App)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("Candidates = %#v, want none", result.Candidates)
	}
}

func TestResolveAIAppByNameNoMatch(t *testing.T) {
	api := &webappFakeAPI{
		listResp: &lib.Response[lib.BizyModelListResp]{
			Data: lib.BizyModelListResp{
				List: []*lib.BizyModelInfo{
					{Id: 1, Name: "LTX 2.5 Image-to-Video"},
				},
			},
		},
	}
	result := ResolveAIAppByName(context.Background(), api, "nope")
	if result.Error == nil {
		t.Fatal("want error for no match")
	}
	if result.Candidates != nil || result.App != nil {
		t.Fatalf("want no app/candidates, got %#v / %#v", result.App, result.Candidates)
	}
}

func TestResolveAIAppByNameAmbiguous(t *testing.T) {
	api := &webappFakeAPI{
		listResp: &lib.Response[lib.BizyModelListResp]{
			Data: lib.BizyModelListResp{
				List: []*lib.BizyModelInfo{
					{Id: 1, Name: "Forked App", Versions: []*lib.BizyModelVersion{{Id: 10}}},
					{Id: 2, Name: "Forked App", Versions: []*lib.BizyModelVersion{{Id: 20}}},
				},
			},
		},
	}
	result := ResolveAIAppByName(context.Background(), api, "Forked App")
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("Candidates = %#v, want 2", result.Candidates)
	}
}

func TestCreateWebAppTaskAlwaysSendsInputValues(t *testing.T) {
	api := &webappFakeAPI{}
	result := CreateWebAppTask(context.Background(), api, lib.WebAppTaskCreateReq{
		WebAppId:    7,
		InputValues: map[string]any{},
	})
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.TaskID != 11 {
		t.Fatalf("TaskID = %d, want 11", result.TaskID)
	}
	body, err := json.Marshal(api.createdReq)
	if err != nil {
		t.Fatalf("marshal req: %v", err)
	}
	if !strings.Contains(string(body), `"input_values":{}`) {
		t.Fatalf("request body %s does not include input_values", string(body))
	}
}
