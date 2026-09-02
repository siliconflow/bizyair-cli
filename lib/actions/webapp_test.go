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
	versionResp *lib.Response[lib.WebAppVersionDetail]
	modelResp   *lib.Response[lib.BizyModelDetail]
	webappResp  *lib.Response[lib.WebAppDetail]
	createdReq  lib.WebAppTaskCreateReq
}

func (a *webappFakeAPI) GetWebappVersionDetailContext(context.Context, int64) (*lib.Response[lib.WebAppVersionDetail], error) {
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