package domain

// WebAppInputNode 表示 AI 应用详情中 input_nodes 的单个输入参数定义。
type WebAppInputNode struct {
	ID           int64  `json:"id,omitempty"`
	NodeID       int64  `json:"node_id,omitempty"`
	NodeName     string `json:"node_name,omitempty"`
	NodeType     string `json:"node_type,omitempty"`
	FieldName    string `json:"field_name,omitempty"`
	FieldType    string `json:"field_type,omitempty"`
	FieldOptions string `json:"field_options,omitempty"` // JSON 字符串
	FieldLabel   string `json:"field_label,omitempty"`
	FieldValue   any    `json:"field_value,omitempty"`
	Sort         int    `json:"sort,omitempty"`
	VariableName string `json:"variable_name,omitempty"`
}

// ParamKey 返回参数键名（优先 FieldName，回退 VariableName）。
func (n WebAppInputNode) ParamKey() string {
	if n.FieldName != "" {
		return n.FieldName
	}
	return n.VariableName
}

// WebAppDetail 表示 AI 应用详情（名称/基座模型/作者/简介/输入节点等）。
type WebAppDetail struct {
	Id              int64             `json:"id,omitempty"`
	BizyModelId     int64             `json:"bizy_model_id,omitempty"`
	Name            string            `json:"name,omitempty"`
	BaseModel       string            `json:"base_model,omitempty"`
	NickName        string            `json:"nick_name,omitempty"`
	UserDescription string            `json:"user_description,omitempty"`
	Description     string            `json:"description,omitempty"`
	CoverUrls       []string          `json:"cover_urls,omitempty"`
	CreatedAt       string            `json:"created_at,omitempty"`
	UpdatedAt       string            `json:"updated_at,omitempty"`
	Creator         string            `json:"creator,omitempty"`
	InputNodes      []WebAppInputNode `json:"input_nodes,omitempty"`
}

// WebAppVersionDetail 是 AI 应用版本详情（GET /v1/bizy_models/versions/{version_id}）。
// 版本 ID 同时是任务创建时使用的 web_app_id；ref_bizy_model_id 指向工作流详情页。
type WebAppVersionDetail struct {
	Id                int64    `json:"id,omitempty"`
	Version           string   `json:"version,omitempty"`
	BaseModel         string   `json:"base_model,omitempty"`
	Description       string   `json:"description,omitempty"`
	Sign              string   `json:"sign,omitempty"`
	Path              string   `json:"path,omitempty"`
	Public            bool     `json:"public,omitempty"`
	Available         bool     `json:"available,omitempty"`
	FileName          string   `json:"file_name,omitempty"`
	BizyModelId       int64    `json:"bizy_model_id,omitempty"`
	UserId            string   `json:"user_id,omitempty"`
	ModelId           int64    `json:"model_id,omitempty"`
	CreatedAt         string   `json:"created_at,omitempty"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
	CoverUrls         []string `json:"cover_urls,omitempty"`
	DraftId           int64    `json:"draft_id,omitempty"`
	RefBizyModelId    int64    `json:"ref_bizy_model_id,omitempty"`
	RefWorkflowId     int64    `json:"ref_workflow_id,omitempty"`
	PromptContentTpl  string   `json:"web_app_prompt_content_tpl,omitempty"`
}

// WebAppTaskCreateReq 是创建 AI 应用任务的请求体。
type WebAppTaskCreateReq struct {
	WebAppId    int64          `json:"web_app_id,omitempty"`
	BackendId   int64          `json:"backend_id,omitempty"`
	InputValues map[string]any `json:"input_values"`
}

// WebAppTaskCreateResp 是同步创建 AI 应用任务的响应（扁平 JSON）。
type WebAppTaskCreateResp struct {
	RequestID string             `json:"request_id,omitempty"`
	Status    string             `json:"status,omitempty"`
	Outputs   []WebAppTaskOutput `json:"outputs,omitempty"`
	Type      string             `json:"type,omitempty"`
	CreatedAt string             `json:"created_at,omitempty"`
	UpdatedAt string             `json:"updated_at,omitempty"`
	ExpiredAt string             `json:"expired_at,omitempty"`
}

// WebAppTaskStatusResp 是 AI 应用任务状态查询响应（openapi/{request_id}）。
type WebAppTaskStatusResp struct {
	RequestID string `json:"request_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Outputs   any    `json:"outputs,omitempty"`
}

// WebAppComfyTaskResp 是异步创建 AI 应用任务的响应 data
// （POST /v1/webapp/task/create 标准信封 data）。
type WebAppComfyTaskResp struct {
	TaskID     int64  `json:"task_id,omitempty"`
	TaskStatus string `json:"task_status,omitempty"`
	WssURL     string `json:"wss_url,omitempty"`
}

// ComfyTaskStatusData 是 Comfy 任务状态响应的 data（GET /v1/comfy/task/{task_id}）。
type ComfyTaskStatusData struct {
	ID          int64  `json:"id,omitempty"`
	Status      string `json:"status,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	Type        string `json:"type,omitempty"`
	DraftName   string `json:"draft_name,omitempty"`
	DraftID     int64  `json:"draft_id,omitempty"`
	WebAppID    int64  `json:"web_app_id,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	ExpiredAt   string `json:"expired_at,omitempty"`
	EndedAt     string `json:"ended_at,omitempty"`
}

// WebAppTaskOutputsResp 是 AI 应用任务输出接口的响应 data
// （GET /v1/webapp/task/openapi/{request_id}/outputs）。
type WebAppTaskOutputsResp struct {
	Outputs []WebAppTaskOutput `json:"outputs,omitempty"`
}

// WebAppTaskOutput 表示单个任务输出。
type WebAppTaskOutput struct {
	ID        int64  `json:"id,omitempty"`
	ObjectURL string `json:"object_url,omitempty"`
	OutputExt string `json:"output_ext,omitempty"`
}