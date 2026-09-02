package lib

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/samber/lo"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/meta"
)

// BizyAPI defines the interface for BizyAir API operations.
// Use this interface in business logic (e.g., actions) to allow mocking in tests.
type BizyAPI interface {
	UserInfoContext(ctx context.Context) (*Response[UserInfo], error)
	OssSignContext(ctx context.Context, signature, modelType string) (*Response[FilesResp], error)
	CommitFileV2Context(ctx context.Context, signature, objectKey, md5Hash, modelType string) (*Response[FilesResp], error)
	CommitModelV2Context(ctx context.Context, modelName, modelType string, versions []*ModelVersion) (*Response[ModelCommitResp], error)
	ListModelContext(ctx context.Context, current, pageSize int, keyword, sort string, modelTypes, baseModels []string) (*Response[BizyModelListResp], error)
	GetBizyModelDetailContext(ctx context.Context, bizyModelId int64) (*Response[BizyModelDetail], error)
	DeleteBizyModelByIdContext(ctx context.Context, bizyModelId int64) (*Response[interface{}], error)
	BatchUpdateVersionPublicContext(ctx context.Context, versionIDs []int64, public bool) (*Response[BatchUpdatePublicResp], error)
	CheckModelExistsContext(ctx context.Context, modelName, modelType string) (bool, error)
	GetUploadTokenContext(ctx context.Context, fileName, fileType string) (*Response[FilesResp], error)
	CommitInputResourceContext(ctx context.Context, name, objectKey string) (*Response[InputResourceCommitResp], error)
	GetBaseModelTypesContext(ctx context.Context) (*Response[[]*BaseModelTypeItem], error)
	GetPlanOverviewContext(ctx context.Context) (*Response[PlanOverviewResp], error)
	GetWalletContext(ctx context.Context) (*Response[WalletResp], error)
	GetCreditsContext(ctx context.Context, current, pageSize, expireDays int) (*Response[CreditsListResp], error)
	GetDayCostContext(ctx context.Context, date string) (*Response[DayCostResp], error)

	// ModelZoo
	GetModelzooListContext(ctx context.Context, current, pageSize int, keyword, sort string) (*Response[ModelzooListResp], error)
	GetModelzooPriceTableFlatContext(ctx context.Context, keyword, billingUnit, sort string, showDeprecated bool) (*Response[ModelzooPriceTableFlatResp], error)
	GetModelzooTagsContext(ctx context.Context) (*Response[ModelzooTagsResp], error)
	GetModelzooCategoriesContext(ctx context.Context, keyword string, showDeprecated bool) (*Response[ModelzooCategoriesResp], error)
	GetModelzooEndpointDetailContext(ctx context.Context, endpoint string) (*Response[ModelzooEndpointDetail], error)
	GetModelzooPriceTableContext(ctx context.Context, endpoint string) (*Response[ModelzooPriceTableResp], error)
	CreateModelZooTaskContext(ctx context.Context, endpoint string, params map[string]any) (*Response[ModelZooTaskResp], error)
	GetModelZooTaskStatusContext(ctx context.Context, requestID string) (*Response[ModelZooTaskStatusResp], error)

	// AI Applications
	ListWebAppsContext(ctx context.Context, current, pageSize int, keyword string) (*Response[BizyModelListResp], error)
	GetWebappVersionDetailContext(ctx context.Context, versionID int64) (*Response[WebAppVersionDetail], error)
	GetWebappDetailByVersionContext(ctx context.Context, webAppID int64) (*Response[WebAppDetail], error)
	CreateWebappComfyTaskContext(ctx context.Context, req WebAppTaskCreateReq) (*Response[WebAppComfyTaskResp], error)
	GetComfyTaskDetailContext(ctx context.Context, taskID int64) (*Response[ComfyTaskStatusData], error)
	GetWebappTaskOutputsContext(ctx context.Context, requestID string) (*Response[WebAppTaskOutputsResp], error)
}

// Client bizyair client
type Client struct {
	Domain     string
	Endpoints  ServiceEndpoints
	ApiKey     string
	httpClient *http.Client
}

const defaultAPIRequestTimeout = 30 * time.Second

// Response the response of bizyair
type Response[T any] struct {
	RequestId string `json:"requestId,omitempty"`
	Code      int32  `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
	Status    bool   `json:"status,omitempty"`
	Data      T      `json:"data"`
	Error     string `json:"error,omitempty"`
}

// NewClient New Client
func NewClient(domain string, apiKey string) *Client {
	endpoints, err := ResolveServiceEndpoints(domain)
	if err != nil {
		// Preserve the old constructor signature. Invalid input remains invalid
		// and will produce a request creation/transport error rather than
		// silently falling back to production.
		raw := strings.TrimRight(strings.TrimSpace(domain), "/")
		endpoints = ServiceEndpoints{Base: raw, API: raw, Meta: raw, Web: raw, Storage: raw}
	}
	return NewClientWithEndpoints(domain, endpoints, apiKey)
}

// NewClientWithEndpoints constructs a client from an explicitly resolved
// service map. It is primarily useful for application wiring and tests.
func NewClientWithEndpoints(domain string, endpoints ServiceEndpoints, apiKey string) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
		transport.TLSClientConfig.MinVersion = tls.VersionTLS12
	}
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 10
	transport.IdleConnTimeout = 90 * time.Second

	return &Client{
		Domain:    domain,
		Endpoints: endpoints,
		ApiKey:    apiKey,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   defaultAPIRequestTimeout,
		},
	}
}

func (c *Client) UserInfo() (*Response[UserInfo], error) {
	return c.UserInfoContext(context.Background())
}

func (c *Client) UserInfoContext(ctx context.Context) (*Response[UserInfo], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/user/info")
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[UserInfo](body)
}

func (c *Client) OssSign(signature string, modelType string) (*Response[FilesResp], error) {
	return c.OssSignContext(context.Background(), signature, modelType)
}

func (c *Client) OssSignContext(ctx context.Context, signature string, modelType string) (*Response[FilesResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, fmt.Sprintf("/v1/files/%s", url.PathEscape(signature)))
	body, statusCode, err := c.doGet(ctx, serverURL, OssSignReq{Type: modelType}, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[FilesResp](body)
}

func (c *Client) CommitFileV2(signature string, objectKey string, md5_hash string, modelType string) (*Response[FilesResp], error) {
	return c.CommitFileV2Context(context.Background(), signature, objectKey, md5_hash, modelType)
}

func (c *Client) CommitFileV2Context(ctx context.Context, signature string, objectKey string, md5Hash string, modelType string) (*Response[FilesResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/files")
	body, statusCode, err := c.doPost(ctx, serverURL, FileCommitReqV2{
		Sign:      signature,
		ObjectKey: objectKey,
		Md5Hash:   md5Hash,
		ModelType: modelType,
	}, c.authHeader())
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[FilesResp](body)
}

func (c *Client) CommitModelV2(modelName string, modelType string, modelVersion []*ModelVersion) (*Response[ModelCommitResp], error) {
	return c.CommitModelV2Context(context.Background(), modelName, modelType, modelVersion)
}

func (c *Client) CommitModelV2Context(ctx context.Context, modelName string, modelType string, modelVersion []*ModelVersion) (*Response[ModelCommitResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/bizy_models")
	body, statusCode, err := c.doPost(ctx, serverURL, ModelCommitReqV2{
		Name:     modelName,
		Type:     modelType,
		Versions: modelVersion,
	}, c.authHeader())
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ModelCommitResp](body)
}

func (c *Client) ListModel(current int, pageSize int, keyword string, sort string, modelTypes []string, baseModels []string) (*Response[BizyModelListResp], error) {
	return c.ListModelContext(context.Background(), current, pageSize, keyword, sort, modelTypes, baseModels)
}

func (c *Client) ListModelContext(ctx context.Context, current int, pageSize int, keyword string, sort string, modelTypes []string, baseModels []string) (*Response[BizyModelListResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/bizy_models/my")
	param := BizyModelListReq{
		Current:    current,
		PageSize:   pageSize,
		Keyword:    keyword,
		Sort:       sort,
		ModelTypes: modelTypes,
		BaseModels: baseModels,
	}
	body, statusCode, err := c.doGet(ctx, serverURL, param, c.authHeader())
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[BizyModelListResp](body)
}

// GetBizyModelDetail 根据 bizy_model_id 获取模型详情
func (c *Client) GetBizyModelDetail(bizyModelId int64) (*Response[BizyModelDetail], error) {
	return c.GetBizyModelDetailContext(context.Background(), bizyModelId)
}

func (c *Client) GetBizyModelDetailContext(ctx context.Context, bizyModelId int64) (*Response[BizyModelDetail], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, fmt.Sprintf("/v1/bizy_models/%d/detail", bizyModelId))
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[BizyModelDetail](body)
}

// DeleteBizyModelById 通过 bizy_model_id 删除模型
func (c *Client) DeleteBizyModelById(bizyModelId int64) (*Response[interface{}], error) {
	return c.DeleteBizyModelByIdContext(context.Background(), bizyModelId)
}

func (c *Client) DeleteBizyModelByIdContext(ctx context.Context, bizyModelId int64) (*Response[interface{}], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, fmt.Sprintf("/v1/bizy_models/%d", bizyModelId))
	body, statusCode, err := c.doDelete(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[interface{}](body)
}

// BatchUpdateVersionPublicContext 批量更新模型版本公开状态
func (c *Client) BatchUpdateVersionPublicContext(ctx context.Context, versionIDs []int64, public bool) (*Response[BatchUpdatePublicResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/bizy_models/versions/batch_update_public")
	body, statusCode, err := c.do(ctx, meta.HTTPPut, serverURL, nil, map[string]any{
		"ids":    versionIDs,
		"public": public,
	}, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	resp, err := handleResponse[BatchUpdatePublicResp](body)
	if err != nil {
		return nil, err
	}
	if len(resp.Data.FailedIDs) > 0 {
		msg := resp.Data.ErrCnMsg
		if msg == "" {
			msg = resp.Data.ErrMsg
		}
		return resp, i18n.NewError("error.model.batch_public_failed", map[string]any{"Msg": msg, "IDs": resp.Data.FailedIDs}, nil)
	}
	return resp, nil
}

// GetUploadToken 获取临时上传凭证（inputs）
func (c *Client) GetUploadToken(fileName, fileType string) (*Response[FilesResp], error) {
	return c.GetUploadTokenContext(context.Background(), fileName, fileType)
}

func (c *Client) GetUploadTokenContext(ctx context.Context, fileName, fileType string) (*Response[FilesResp], error) {
	serverURL := joinEndpoint(c.Endpoints.API, "/v1/upload/token")
	body, statusCode, err := c.doGet(ctx, serverURL, UploadTokenReq{FileName: fileName, FileType: fileType}, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[FilesResp](body)
}

// CommitInputResource 提交输入资源，返回可用 url
func (c *Client) CommitInputResource(name, objectKey string) (*Response[InputResourceCommitResp], error) {
	return c.CommitInputResourceContext(context.Background(), name, objectKey)
}

func (c *Client) CommitInputResourceContext(ctx context.Context, name, objectKey string) (*Response[InputResourceCommitResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/input_resource/commit")
	body, statusCode, err := c.doPost(ctx, serverURL, InputResourceCommitReq{Name: name, ObjectKey: objectKey}, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[InputResourceCommitResp](body)
}

// CheckModelExists 检查模型名是否已存在
// 返回 true 表示模型名已存在（HTTP 200），false 表示不存在（HTTP 404）
func (c *Client) CheckModelExists(modelName string, modelType string) (bool, error) {
	return c.CheckModelExistsContext(context.Background(), modelName, modelType)
}

func (c *Client) CheckModelExistsContext(ctx context.Context, modelName string, modelType string) (bool, error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/bizy_models/exists")
	body, statusCode, err := c.doGet(ctx, serverURL, ModelQueryReq{
		Name: modelName,
		Type: modelType,
	}, c.authHeader())
	if err != nil {
		return false, err
	}

	// HTTP 200 表示模型名已存在
	if statusCode == http.StatusOK {
		return true, nil
	}

	// HTTP 404 表示模型名不存在
	if statusCode == http.StatusNotFound {
		return false, nil
	}

	// 其他状态码作为错误处理
	return false, handleError(body, statusCode)
}

// GetBaseModelTypes 获取基础模型类型列表
func (c *Client) GetBaseModelTypes() (*Response[[]*BaseModelTypeItem], error) {
	return c.GetBaseModelTypesContext(context.Background())
}

func (c *Client) GetBaseModelTypesContext(ctx context.Context) (*Response[[]*BaseModelTypeItem], error) {
	serverURL := c.Endpoints.BaseModelTypesURL()
	body, statusCode, err := c.doGet(ctx, serverURL, nil, nil)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}

	return handleResponse[[]*BaseModelTypeItem](body)
}

// GetPlanOverview 获取套餐概览
func (c *Client) GetPlanOverview() (*Response[PlanOverviewResp], error) {
	return c.GetPlanOverviewContext(context.Background())
}

func (c *Client) GetPlanOverviewContext(ctx context.Context) (*Response[PlanOverviewResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/user/plan_overview")
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[PlanOverviewResp](body)
}

// GetWallet 获取钱包余额
func (c *Client) GetWallet() (*Response[WalletResp], error) {
	return c.GetWalletContext(context.Background())
}

func (c *Client) GetWalletContext(ctx context.Context) (*Response[WalletResp], error) {
	serverURL := joinEndpoint(c.Endpoints.FinanceURL(), "/v1/wallet")
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[WalletResp](body)
}

// GetCredits 获取积分明细
func (c *Client) GetCredits(current, pageSize, expireDays int) (*Response[CreditsListResp], error) {
	return c.GetCreditsContext(context.Background(), current, pageSize, expireDays)
}

func (c *Client) GetCreditsContext(ctx context.Context, current, pageSize, expireDays int) (*Response[CreditsListResp], error) {
	serverURL := joinEndpoint(c.Endpoints.FinanceURL(), "/v1/credits")
	body, statusCode, err := c.doGet(ctx, serverURL, CreditsReq{
		Current:    current,
		PageSize:   pageSize,
		ExpireDays: expireDays,
	}, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[CreditsListResp](body)
}

// GetDayCost 获取每日消费记录
func (c *Client) GetDayCost(date string) (*Response[DayCostResp], error) {
	return c.GetDayCostContext(context.Background(), date)
}

func (c *Client) GetDayCostContext(ctx context.Context, date string) (*Response[DayCostResp], error) {
	serverURL := joinEndpoint(c.Endpoints.FinanceURL(), "/v1/bills/day_cost")
	// API requires ISO 8601 date format (e.g. "2026-08-10T00:00:00Z").
	// Default to today if no date provided.
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02T00:00:00Z")
	} else if len(date) == 10 && date[4] == '-' && date[7] == '-' {
		// Convert plain date like "2026-08-10" to ISO 8601.
		date = date + "T00:00:00Z"
	}
	body, statusCode, err := c.doGet(ctx, serverURL, struct {
		Date string `form:"date" query:"date"`
	}{Date: date}, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[DayCostResp](body)
}

func (c *Client) GetModelzooListContext(ctx context.Context, current, pageSize int, keyword, sort string) (*Response[ModelzooListResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/modelzoo/list")
	body, statusCode, err := c.doPost(ctx, serverURL, map[string]any{
		"current":   current,
		"page_size": pageSize,
		"keyword":   keyword,
		"sort":      sort,
	}, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ModelzooListResp](body)
}

func (c *Client) GetModelzooPriceTableFlatContext(ctx context.Context, keyword, billingUnit, sort string, showDeprecated bool) (*Response[ModelzooPriceTableFlatResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/modelzoo/price_table_flat")
	query := url.Values{}
	if keyword != "" {
		query.Set("keyword", keyword)
	}
	if billingUnit != "" {
		query.Set("billing_unit", billingUnit)
	}
	if sort != "" {
		query.Set("sort", sort)
	}
	query.Set("show_deprecated", fmt.Sprintf("%v", showDeprecated))
	if len(query) > 0 {
		serverURL = serverURL + "?" + query.Encode()
	}
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ModelzooPriceTableFlatResp](body)
}

func (c *Client) GetModelzooTagsContext(ctx context.Context) (*Response[ModelzooTagsResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/modelzoo/tags")
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ModelzooTagsResp](body)
}

func (c *Client) GetModelzooCategoriesContext(ctx context.Context, keyword string, showDeprecated bool) (*Response[ModelzooCategoriesResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/modelzoo/categories")
	query := url.Values{}
	if keyword != "" {
		query.Set("keyword", keyword)
	}
	query.Set("show_deprecated", fmt.Sprintf("%v", showDeprecated))
	if len(query) > 0 {
		serverURL = serverURL + "?" + query.Encode()
	}
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ModelzooCategoriesResp](body)
}

func (c *Client) GetModelzooEndpointDetailContext(ctx context.Context, endpoint string) (*Response[ModelzooEndpointDetail], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, fmt.Sprintf("/v1/modelzoo/detail/%s", url.PathEscape(endpoint)))
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ModelzooEndpointDetail](body)
}

func (c *Client) GetModelzooPriceTableContext(ctx context.Context, endpoint string) (*Response[ModelzooPriceTableResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, fmt.Sprintf("/v1/modelzoo/price_table/%s", url.PathEscape(endpoint)))
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ModelzooPriceTableResp](body)
}

func (c *Client) CreateModelZooTaskContext(ctx context.Context, endpoint string, params map[string]any) (*Response[ModelZooTaskResp], error) {
	serverURL := joinEndpoint(c.Endpoints.API, fmt.Sprintf("/v1/modelzoo/tasks/openapi/%s", url.PathEscape(endpoint)))
	body, statusCode, err := c.doPost(ctx, serverURL, params, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ModelZooTaskResp](body)
}

func (c *Client) GetModelZooTaskStatusContext(ctx context.Context, requestID string) (*Response[ModelZooTaskStatusResp], error) {
	serverURL := joinEndpoint(c.Endpoints.API, fmt.Sprintf("/v1/modelzoo/tasks/%s", url.PathEscape(requestID)))
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ModelZooTaskStatusResp](body)
}

// ---- AI Applications ----

// ListWebAppsContext 获取社区 AI 应用列表（GET /v1/bizy_models/community?model_types=Application）。
func (c *Client) ListWebAppsContext(ctx context.Context, current, pageSize int, keyword string) (*Response[BizyModelListResp], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, "/v1/bizy_models/community")
	param := BizyModelListReq{
		Current:    current,
		PageSize:   pageSize,
		Keyword:    keyword,
		Sort:       "Recently",
		ModelTypes: []string{"Application"},
	}
	body, statusCode, err := c.doGet(ctx, serverURL, param, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[BizyModelListResp](body)
}

// GetWebappVersionDetailContext 获取 AI 应用版本详情（GET /v1/bizy_models/versions/{version_id}）。
// 版本 ID 同时是任务创建时使用的 web_app_id；其 ref_bizy_model_id 指向工作流详情。
func (c *Client) GetWebappVersionDetailContext(ctx context.Context, versionID int64) (*Response[WebAppVersionDetail], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, fmt.Sprintf("/v1/bizy_models/versions/%d", versionID))
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[WebAppVersionDetail](body)
}

// GetWebappDetailByVersionContext 按 web_app_id（版本 ID）获取 AI 应用详情
// （GET /v1/webapp/{web_app_id}/detail），返回包含 input_nodes 输入节点定义。
func (c *Client) GetWebappDetailByVersionContext(ctx context.Context, webAppID int64) (*Response[WebAppDetail], error) {
	serverURL := joinEndpoint(c.Endpoints.Meta, fmt.Sprintf("/v1/webapp/%d/detail", webAppID))
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[WebAppDetail](body)
}

// CreateWebappComfyTaskContext 异步创建 AI 应用任务（POST /v1/webapp/task/create，标准信封）。
func (c *Client) CreateWebappComfyTaskContext(ctx context.Context, req WebAppTaskCreateReq) (*Response[WebAppComfyTaskResp], error) {
	serverURL := joinEndpoint(c.Endpoints.API, "/v1/webapp/task/create")
	body, statusCode, err := c.doPost(ctx, serverURL, req, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[WebAppComfyTaskResp](body)
}

// GetComfyTaskDetailContext 按 Comfy task_id 查询任务状态（GET /v1/comfy/task/{task_id}）。
func (c *Client) GetComfyTaskDetailContext(ctx context.Context, taskID int64) (*Response[ComfyTaskStatusData], error) {
	serverURL := joinEndpoint(c.Endpoints.API, fmt.Sprintf("/v1/comfy/task/%d", taskID))
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[ComfyTaskStatusData](body)
}

// GetWebappTaskOutputsContext 获取 AI 应用任务输出（GET /v1/webapp/task/openapi/{request_id}/outputs）。
func (c *Client) GetWebappTaskOutputsContext(ctx context.Context, requestID string) (*Response[WebAppTaskOutputsResp], error) {
	serverURL := joinEndpoint(c.Endpoints.API, fmt.Sprintf("/v1/webapp/task/openapi/%s/outputs", url.PathEscape(requestID)))
	body, statusCode, err := c.doGet(ctx, serverURL, nil, c.authHeader())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[WebAppTaskOutputsResp](body)
}

func (c *Client) authHeader() map[string]string {
	header := make(map[string]string)
	header[meta.HeaderAuthorization] = fmt.Sprintf("Bearer %s", c.ApiKey)
	return header
}

const maxAPIResponseSize = 16 * 1024 * 1024

func (c *Client) doGet(ctx context.Context, urlStr string, queryParams interface{}, header map[string]string) ([]byte, int, error) {
	return c.do(ctx, meta.HTTPGet, urlStr, queryParams, nil, header)
}

func (c *Client) doPost(ctx context.Context, urlStr string, data interface{}, header map[string]string) ([]byte, int, error) {
	return c.do(ctx, meta.HTTPPost, urlStr, nil, data, header)
}

func (c *Client) doDelete(ctx context.Context, urlStr string, data interface{}, header map[string]string) ([]byte, int, error) {
	return c.do(ctx, meta.HTTPDelete, urlStr, nil, data, header)
}

func (c *Client) doPut(ctx context.Context, urlStr string, data interface{}, header map[string]string) ([]byte, int, error) {
	return c.do(ctx, meta.HTTPPut, urlStr, nil, data, header)
}

func (c *Client) do(ctx context.Context, method, urlStr string, queryParams, data interface{}, header map[string]string) ([]byte, int, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, -1, i18n.NewError("error.network.invalid_url", map[string]any{"URL": urlStr}, err)
	}

	if queryParams != nil {
		v := reflect.ValueOf(queryParams)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		query := parsedURL.Query()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			structField := v.Type().Field(i)
			fieldName := strings.Split(structField.Tag.Get("query"), ",")[0]
			if fieldName == "" {
				fieldName = lo.SnakeCase(structField.Name)
			}
			if fieldName == "-" {
				continue
			}
			if field.IsZero() {
				continue
			}

			if field.Kind() == reflect.Slice {
				for j := 0; j < field.Len(); j++ {
					elemValue := fmt.Sprintf("%v", field.Index(j).Interface())
					if elemValue != "" {
						query.Add(fieldName, elemValue)
					}
				}
			} else {
				fieldValue := fmt.Sprintf("%v", field.Interface())
				query.Add(fieldName, fieldValue)
			}
		}
		parsedURL.RawQuery = query.Encode()
	}

	if ctx == nil {
		ctx = context.Background()
	}
	var requestBody io.Reader
	if data != nil {
		jsonData, marshalErr := json.Marshal(data)
		if marshalErr != nil {
			return nil, -1, i18n.NewError("error.network.encode_request", map[string]any{"Method": method, "URL": parsedURL.String()}, marshalErr)
		}
		requestBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), requestBody)
	if err != nil {
		return nil, -1, i18n.NewError("error.network.create_request", map[string]any{"Method": method, "URL": parsedURL.String()}, err)
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}
	req.Header.Set(meta.HeaderSiliconCliVersion, meta.Version)
	if data != nil {
		req.Header.Set(meta.HeaderContentType, meta.JsonContentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, -1, i18n.NewError("error.network.request_failed", map[string]any{"Method": method, "URL": parsedURL.String()}, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponseSize+1))
	if err != nil {
		return nil, resp.StatusCode, i18n.NewError("error.network.response_read_failed", map[string]any{"URL": parsedURL.String()}, err)
	}
	if len(body) > maxAPIResponseSize {
		return nil, resp.StatusCode, i18n.NewError("error.network.response_too_large", map[string]any{"URL": parsedURL.String(), "Limit": maxAPIResponseSize}, nil)
	}
	return body, resp.StatusCode, nil
}

func handleError(responseBody []byte, statusCode int) error {
	rawMessage := strings.TrimSpace(string(responseBody))
	var parsedResponse Response[interface{}]
	err := json.Unmarshal(responseBody, &parsedResponse)
	if err == nil {
		if messageID, exists := meta.ServerErrorMessageIDs[parsedResponse.Code]; exists {
			return &APIError{
				HTTPStatus: statusCode, Code: parsedResponse.Code, RawMessage: parsedResponse.Message, KnownCodeMessageID: messageID,
			}
		}
		detail := parsedResponse.Message
		if detail == "" {
			detail = rawMessage
		}
		if statusCode == http.StatusNotFound {
			return &APIError{HTTPStatus: statusCode, Code: parsedResponse.Code, KnownCodeMessageID: "error.server.not_found", RawMessage: detail}
		}
		return &APIError{HTTPStatus: statusCode, Code: parsedResponse.Code, RawMessage: detail}
	}

	rawMessage = strings.TrimFunc(rawMessage, func(r rune) bool {
		return unicode.Is(unicode.Quotation_Mark, r)
	})
	if statusCode == http.StatusNotFound {
		return &APIError{HTTPStatus: statusCode, KnownCodeMessageID: "error.server.not_found", RawMessage: rawMessage}
	}
	return &APIError{HTTPStatus: statusCode, RawMessage: rawMessage}
}

func handleResponse[T any](responseBody []byte) (*Response[T], error) {
	var parsedResponse Response[T]
	err := json.Unmarshal(responseBody, &parsedResponse)
	if err != nil {
		logs.Debugf("error: %s\n", err)
		return nil, &APIError{HTTPStatus: http.StatusOK, RawMessage: strings.TrimSpace(string(responseBody))}
	}

	if parsedResponse.Code != meta.OKCode {
		if messageID, exists := meta.ServerErrorMessageIDs[parsedResponse.Code]; exists {
			return nil, &APIError{HTTPStatus: http.StatusOK, Code: parsedResponse.Code, RawMessage: parsedResponse.Message, KnownCodeMessageID: messageID}
		}
		return nil, &APIError{HTTPStatus: http.StatusOK, Code: parsedResponse.Code, RawMessage: parsedResponse.Message}
	}
	return &parsedResponse, nil
}
