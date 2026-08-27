package lib

import "github.com/siliconflow/bizyair-cli/domain"

type FileReq struct {
	Sign string `json:"sign,omitempty" form:"sign" query:"sign"`
}

// User/account types（定义见 domain/user.go）
type UserInfo = domain.UserInfo
type PlanInfo = domain.PlanInfo
type PlanNextTier = domain.PlanNextTier
type PlanWalletInfo = domain.PlanWalletInfo
type PlanRollingWindow = domain.PlanRollingWindow
type PlanOverviewResp = domain.PlanOverviewResp
type WalletInfo = domain.WalletInfo
type WalletResp = domain.WalletResp
type CreditItem = domain.CreditItem
type CreditsListResp = domain.CreditsListResp
type CreditsReq = domain.CreditsReq
type CostByTimeResult = domain.CostByTimeResult
type DayCostResp = domain.DayCostResp

type FilesResp struct {
	File    *FileInfo    `json:"file,omitempty" form:"file" query:"file"`
	Storage *StorageInfo `json:"storage,omitempty" form:"storage" query:"storage"`
}

type FileInfo struct {
	Sign            string `json:"sign,omitempty" form:"sign" query:"sign"`
	ObjectKey       string `json:"object_key,omitempty" form:"object_key" query:"object_key"`
	AccessKeyId     string `json:"access_key_id,omitempty" form:"access_key_id" query:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret,omitempty" form:"access_key_secret" query:"access_key_secret"`
	Expiration      string `json:"expiration,omitempty" form:"expiration" query:"expiration"`
	SecurityToken   string `json:"security_token,omitempty" form:"security_token" query:"security_token"`
	Id              int64  `json:"id,omitempty" form:"id" query:"id"`
}

type StorageInfo struct {
	Endpoint string `json:"endpoint,omitempty" form:"endpoint" query:"endpoint"`
	Bucket   string `json:"bucket,omitempty" form:"bucket" query:"bucket"`
	Region   string `json:"region,omitempty" form:"region" query:"region"`
}

type FileCommitReqV2 struct {
	Sign      string `json:"sign,omitempty" form:"sign" query:"sign"`
	ObjectKey string `json:"object_key,omitempty" form:"object_key" query:"object_key"`
	Md5Hash   string `json:"md5_hash,omitempty" form:"md5_hash" query:"md5_hash"`
	ModelType string `json:"type,omitempty" form:"type" query:"type"`
}

type ModelCommitReqV2 struct {
	Name     string          `json:"name,omitempty" form:"name" query:"name"`
	Type     string          `json:"type,omitempty" form:"type" query:"type"`
	Versions []*ModelVersion `json:"versions,omitempty" form:"versions" query:"versions"`
}

type ModelFile struct {
	Sign string `json:"sign,omitempty" form:"sign" query:"sign"`
	Path string `json:"path,omitempty" form:"path" query:"path"`
}

type ModelVersion struct {
	Version   string `json:"version,omitempty" form:"version" query:"version"`
	BaseModel string `json:"base_model,omitempty" form:"base_model" query:"base_model"`
	// Introduction is the user-facing model introduction. The current API
	// contract names this version-level field "description".
	Introduction string   `json:"description,omitempty" form:"description" query:"description"`
	Public       bool     `json:"public,omitempty" form:"public" query:"public"`
	Sign         string   `json:"sign,omitempty" form:"sign" query:"sign"`
	Path         string   `json:"path,omitempty" form:"path" query:"path"`
	CoverUrls    []string `json:"cover_urls,omitempty" form:"cover_urls" query:"cover_urls"`
}

type OssSignReq struct {
	Type string `json:"type,omitempty" form:"type" query:"type"`
}

type ModelCommitResp struct {
}

type ModelQueryReq struct {
	Name string `json:"name,omitempty" form:"name" query:"name"`
	Type string `json:"type,omitempty" form:"type" query:"type"`
}

type ModelInfo struct {
	Name      string `json:"name,omitempty" form:"name" query:"name"`
	Type      string `json:"type,omitempty" form:"type" query:"type"`
	FileNum   int    `json:"file_num,omitempty" form:"file_num" query:"file_num"`
	Available bool   `json:"available,omitempty" form:"available" query:"available"`
	UpdatedAt string `json:"updated_at,omitempty" form:"updated_at" query:"updated_at"`
}

// 新版模型列表请求参数
type BizyModelListReq struct {
	Current    int      `json:"current,omitempty" form:"current" query:"current"`
	PageSize   int      `json:"page_size,omitempty" form:"page_size" query:"page_size"`
	Keyword    string   `json:"keyword,omitempty" form:"keyword" query:"keyword"`
	Sort       string   `json:"sort,omitempty" form:"sort" query:"sort"`
	ModelTypes []string `json:"model_types,omitempty" form:"model_types" query:"model_types"`
	BaseModels []string `json:"base_models,omitempty" form:"base_models" query:"base_models"`
}

// 模型统计信息
type ModelCounter struct {
	UsedCount       int `json:"used_count,omitempty"`
	ForkedCount     int `json:"forked_count,omitempty"`
	LikedCount      int `json:"liked_count,omitempty"`
	DownloadedCount int `json:"downloaded_count,omitempty"`
	ViewCount       int `json:"view_count,omitempty"`
}

// 模型版本信息
type BizyModelVersion struct {
	Id           int64        `json:"id,omitempty"`
	Version      string       `json:"version,omitempty"`
	CoverUrls    []string     `json:"cover_urls,omitempty"`
	FileUrl      string       `json:"file_url,omitempty"`
	FileName     string       `json:"file_name,omitempty"`
	FileSize     int64        `json:"file_size,omitempty"`
	Public       bool         `json:"public,omitempty"`
	DraftId      string       `json:"draft_id,omitempty"`
	CreatedAt    string       `json:"created_at,omitempty"`
	Forked       bool         `json:"forked,omitempty"`
	Liked        bool         `json:"liked,omitempty"`
	Counter      ModelCounter `json:"counter,omitempty"`
	BaseModel    string       `json:"base_model,omitempty"`
	TriggerWords []string     `json:"trigger_words,omitempty"`
	Description  string       `json:"description,omitempty"`
}

// 模型信息（新版）
type BizyModelInfo struct {
	Id          int64               `json:"id,omitempty"`
	Name        string              `json:"name,omitempty"`
	Type        string              `json:"type,omitempty"`
	Description string              `json:"description,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	CreatedAt   string              `json:"created_at,omitempty"`
	UpdatedAt   string              `json:"updated_at,omitempty"`
	UserId      string              `json:"user_id,omitempty"`
	Username    string              `json:"username,omitempty"`
	Counter     ModelCounter        `json:"counter,omitempty"`
	Versions    []*BizyModelVersion `json:"versions,omitempty"`
}

// 新版模型列表响应
type BizyModelListResp struct {
	List     []*BizyModelInfo `json:"list,omitempty"`
	Total    int              `json:"total,omitempty"`
	Current  int              `json:"current,omitempty"`
	PageSize int              `json:"page_size,omitempty"`
}

// Upload Token（inputs）
type UploadTokenReq struct {
	FileName string `json:"file_name,omitempty" form:"file_name" query:"file_name"`
	FileType string `json:"file_type,omitempty" form:"file_type" query:"file_type"`
}

// Input Resource Commit
type InputResourceCommitReq struct {
	Name      string `json:"name,omitempty" form:"name" query:"name"`
	ObjectKey string `json:"object_key,omitempty" form:"object_key" query:"object_key"`
}

type InputResourceCommitResp struct {
	Id   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Ext  string `json:"ext,omitempty"`
	Url  string `json:"url,omitempty"`
}

// 详情页：模型版本（与列表版本字段略有不同，按接口保持独立定义）
type BizyModelDetailVersion struct {
	Id          int64  `json:"id,omitempty"`
	Version     string `json:"version,omitempty"`
	BaseModel   string `json:"base_model,omitempty"`
	Description string `json:"description,omitempty"`
	// Intro is retained for compatibility with older responses.
	Intro       string       `json:"intro,omitempty"`
	Sign        string       `json:"sign,omitempty"`
	Path        string       `json:"path,omitempty"`
	Available   bool         `json:"available,omitempty"`
	FileName    string       `json:"file_name,omitempty"`
	BizyModelId int64        `json:"bizy_model_id,omitempty"`
	UserId      string       `json:"user_id,omitempty"`
	UserName    string       `json:"user_name,omitempty"`
	UserAvatar  string       `json:"user_avatar,omitempty"`
	Counter     ModelCounter `json:"counter,omitempty"`
	ModelId     int64        `json:"model_id,omitempty"`
	FileSize    int64        `json:"file_size,omitempty"`
	Public      bool         `json:"public,omitempty"`
	CreatedAt   string       `json:"created_at,omitempty"`
	UpdatedAt   string       `json:"updated_at,omitempty"`
	CoverUrls   []string     `json:"cover_urls,omitempty"`
}

// 详情页：模型详情
type BizyModelDetail struct {
	Id         int64                    `json:"id,omitempty"`
	Name       string                   `json:"name,omitempty"`
	Type       string                   `json:"type,omitempty"`
	UserId     string                   `json:"user_id,omitempty"`
	UserName   string                   `json:"user_name,omitempty"`
	UserAvatar string                   `json:"user_avatar,omitempty"`
	Versions   []BizyModelDetailVersion `json:"versions,omitempty"`
	Counter    ModelCounter             `json:"counter,omitempty"`
	CreatedAt  string                   `json:"created_at,omitempty"`
	UpdatedAt  string                   `json:"updated_at,omitempty"`
	Source     string                   `json:"source,omitempty"`
}

// 基础模型类型项
type BaseModelTypeItem struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
}

// 批量更新公开状态响应
type BatchUpdatePublicResp struct {
	ErrMsg    string  `json:"err_msg,omitempty"`
	ErrCnMsg  string  `json:"err_cn_msg,omitempty"`
	FailedIDs []int64 `json:"failed_ids,omitempty"`
}

// ---- ModelZoo 数据模型 ----

// ModelzooModelFlat 模型广场扁平化模型（含价格表）
type ModelzooModelFlat struct {
	Endpoint        string      `json:"endpoint,omitempty"`
	DisplayName     string      `json:"display_name,omitempty"`
	Manufacturer    string      `json:"manufacturer,omitempty"`
	ModelName       string      `json:"model_name,omitempty"`
	BillingUnit     string      `json:"billing_unit,omitempty"`
	MinCredits      int64       `json:"min_credits,omitempty"`
	Category        string      `json:"category,omitempty"`
	IndicativePrice bool        `json:"indicative_price,omitempty"`
	PriceTable      *PriceTable `json:"price_table,omitempty"`
	Series          string      `json:"series,omitempty"`
	ModelVersion    string      `json:"model_version,omitempty"`
	Tags            []string    `json:"tags,omitempty"`
	Description     string      `json:"description,omitempty"`
	Deprecated      bool        `json:"deprecated,omitempty"`
	IconURL         string      `json:"icon_url,omitempty"`
	SimplePriceText string      `json:"simple_price_text,omitempty"`
	SubCategory     string      `json:"sub_category,omitempty"`
}

// ModelzooTag 能力标签
type ModelzooTag struct {
	ID  int64  `json:"id,omitempty"`
	Tag string `json:"tag,omitempty"`
}

// ModelzooTagsResp 标签列表响应
type ModelzooTagsResp struct {
	Tags []ModelzooTag `json:"tags,omitempty"`
}

// PriceTable 价格表
type PriceTable struct {
	Columns []PriceTableColumn `json:"columns,omitempty"`
	Cells   [][]PriceTableCell `json:"cells,omitempty"`
}

// PriceTableColumn 价格表列定义
type PriceTableColumn struct {
	FieldLabel string `json:"field_label,omitempty"`
	FieldName  string `json:"field_name,omitempty"`
}

// PriceTableCell 价格表单元格
type PriceTableCell struct {
	ValueStr string  `json:"value_str,omitempty"`
	UnitName string  `json:"unit_name,omitempty"`
	UnitKey  string  `json:"unit_key,omitempty"`
	Amount   float64 `json:"amount,omitempty"`
}

// ModelzooEndpointDetail 端点详情（含输入参数列表）
type ModelzooEndpointDetail struct {
	Endpoint     string               `json:"endpoint,omitempty"`
	DisplayName  string               `json:"display_name,omitempty"`
	Description  string               `json:"description,omitempty"`
	Manufacturer string               `json:"manufacturer,omitempty"`
	Category     string               `json:"category,omitempty"`
	Edition      string               `json:"edition,omitempty"`
	BillingUnit  string               `json:"billing_unit,omitempty"`
	MinCredits   int64                `json:"min_credits,omitempty"`
	InputParams  []ModelzooInputParam `json:"input_params,omitempty"`
}

// ModelzooInputParam 输入参数定义
type ModelzooInputParam struct {
	VariableName string                `json:"variable_name,omitempty"`
	VariableType string                `json:"variable_type,omitempty"`
	FieldName    string                `json:"field_name,omitempty"`
	FieldType    string                `json:"field_type,omitempty"`
	FieldValue   any                   `json:"field_value,omitempty"`
	FieldLabel   string                `json:"field_label,omitempty"`
	Required     bool                  `json:"required,omitempty"`
	Sort         int                   `json:"sort,omitempty"`
	FieldTooltip string                `json:"field_tooltip,omitempty"`
	FieldOptions *ModelzooFieldOptions `json:"field_options,omitempty"`
}

// ParamKey 返回参数键名（优先 FieldName，回退 VariableName）
func (p ModelzooInputParam) ParamKey() string {
	if p.FieldName != "" {
		return p.FieldName
	}
	return p.VariableName
}

// ModelzooFieldOptions 字段选项（枚举/像素/宽高比）
type ModelzooFieldOptions struct {
	Values         []any    `json:"values,omitempty"`
	Options        []any    `json:"options,omitempty"`
	Choices        []any    `json:"choices,omitempty"`
	MinPixels      *int     `json:"min_pixels,omitempty"`
	MaxPixels      *int     `json:"max_pixels,omitempty"`
	MinAspectRatio *float64 `json:"min_aspect_ratio,omitempty"`
	MaxAspectRatio *float64 `json:"max_aspect_ratio,omitempty"`
}

// EnumValues 返回可选枚举值
func (o ModelzooFieldOptions) EnumValues() []any {
	if len(o.Values) > 0 {
		return o.Values
	}
	if len(o.Options) > 0 {
		return o.Options
	}
	return o.Choices
}

// ModelzooListItem POST /v1/modelzoo/list 返回的单个模型
type ModelzooListItem struct {
	Id            int64   `json:"id"`
	DisplayName   string  `json:"display_name"`
	Description   string  `json:"description"`
	Category      string  `json:"category"`
	ModelName     string  `json:"model_name"`
	Endpoint      string  `json:"endpoint"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	Owner         string  `json:"owner"`
	Manufacturer  string  `json:"manufacturer"`
	IconURL       string  `json:"icon_url"`
	BackgroundURL string  `json:"background_url"`
	Edition       string  `json:"edition"`
	MinCredits    int64   `json:"min_credits"`
	DiscountRate  float64 `json:"discount_rate"`
}

// ModelzooListResp 模型广场列表响应
type ModelzooListResp struct {
	List     []ModelzooListItem `json:"list"`
	Total    int                `json:"total"`
	Current  int                `json:"current"`
	PageSize int                `json:"page_size"`
}

// ModelzooPriceTableFlatResp 扁平价格表响应
type ModelzooPriceTableFlatResp struct {
	Models []ModelzooModelFlat `json:"models,omitempty"`
}

// ---- ModelZoo 任务模型 ----

// ModelZooTaskResp 任务创建响应
type ModelZooTaskResp struct {
	RequestID string `json:"request_id,omitempty"`
}

// ModelZooTaskStatusResp 任务状态查询响应
type ModelZooTaskStatusResp struct {
	RequestID string `json:"request_id,omitempty"`
	Status    string `json:"status,omitempty"`
	Outputs   any    `json:"outputs,omitempty"`
}

// ModelZoo 任务状态常量
const (
	TaskStatusSuccess      = "Success"
	TaskStatusFailed       = "Failed"
	TaskStatusRunning      = "Running"
	TaskStatusQueued       = "Queued"
	TaskStatusCancelled    = "Cancelled"
	TaskStatusTransferring = "Transferring"
)
