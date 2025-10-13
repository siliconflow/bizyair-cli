package lib

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"unicode"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/samber/lo"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

// Client bizyair client
type Client struct {
	Domain string
	ApiKey string
}

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
	return &Client{
		Domain: domain,
		ApiKey: apiKey,
	}
}

func (c *Client) UserInfo() (*Response[UserInfo], error) {
	serverUrl := fmt.Sprintf("%s/%s/user/info", c.Domain, meta.APIv1)
	body, statusCode, err := c.doGet(serverUrl, nil, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}

	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}
	return handleResponse[UserInfo](body)
}

func (c *Client) OssSign(signature string, modelType string) (*Response[FilesResp], error) {

	// if file exists, return file.id, else return oss certificate
	// serverUrl := fmt.Sprintf("%s:%s/%s/sign?%s=%s&type=%s", c.Host, c.Port, meta.BizyUrl, meta.SignMethod, signature, modelType)
	serverUrl := fmt.Sprintf("%s/x/%s/files/%s", c.Domain, meta.APIv1, signature)
	body, statusCode, err := c.doGet(serverUrl, OssSignReq{Type: modelType}, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}
	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[FilesResp](body)

}

func (c *Client) CommitFileV2(signature string, objectKey string, md5_hash string, modelType string) (*Response[FilesResp], error) {
	serverUrl := fmt.Sprintf("%s/x/%s/files", c.Domain, meta.APIv1)
	body, statusCode, err := c.doPost(serverUrl, FileCommitReqV2{
		Sign:      signature,
		ObjectKey: objectKey,
		Md5Hash:   md5_hash,
		ModelType: modelType,
	}, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}

	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[FilesResp](body)
}

func (c *Client) CommitModelV2(modelName string, modelType string, modelVersion []*ModelVersion) (*Response[ModelCommitResp], error) {
	serverUrl := fmt.Sprintf("%s/x/%s/bizy_models", c.Domain, meta.APIv1)
	body, statusCode, err := c.doPost(serverUrl, ModelCommitReqV2{
		Name:     modelName,
		Type:     modelType,
		Versions: modelVersion,
	}, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}

	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[ModelCommitResp](body)
}

func (c *Client) ListModel(current int, pageSize int, keyword string, sort string, modelTypes []string, baseModels []string) (*Response[BizyModelListResp], error) {
	// mode 固定为 "my"（我发布的模型）
	serverUrl := fmt.Sprintf("%s/x/%s/bizy_models/my", c.Domain, meta.APIv1)
	param := BizyModelListReq{
		Current:    current,
		PageSize:   pageSize,
		Keyword:    keyword,
		Sort:       sort,
		ModelTypes: modelTypes,
		BaseModels: baseModels,
	}
	body, statusCode, err := c.doGet(serverUrl, param, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}

	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[BizyModelListResp](body)
}

func (c *Client) ListModelFiles(modelType string, modelName string, extName string, public bool) (*Response[ModelListFilesResp], error) {
	serverUrl := fmt.Sprintf("%s/x/%s/models/files", c.Domain, meta.APIv1)
	param := ModelListFilesReq{
		Type:    modelType,
		Name:    modelName,
		ExtName: extName,
		Public:  public,
	}
	body, statusCode, err := c.doGet(serverUrl, param, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}

	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[ModelListFilesResp](body)
}

// GetBizyModelDetail 根据 bizy_model_id 获取模型详情
func (c *Client) GetBizyModelDetail(bizyModelId int64) (*Response[BizyModelDetail], error) {
	serverUrl := fmt.Sprintf("%s/x/%s/bizy_models/%d/detail", c.Domain, meta.APIv1, bizyModelId)
	body, statusCode, err := c.doGet(serverUrl, nil, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}
	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[BizyModelDetail](body)
}

func (c *Client) RemoveModel(modelType string, modelName string) (*Response[ModelDeleteResp], error) {
	serverUrl := fmt.Sprintf("%s/x/%s/models", c.Domain, meta.APIv1)
	body, statusCode, err := c.doDelete(serverUrl, ModelDeleteReq{
		Name: modelName,
		Type: modelType,
	}, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}

	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[ModelDeleteResp](body)
}

// DeleteBizyModelById 通过 bizy_model_id 删除模型
func (c *Client) DeleteBizyModelById(bizyModelId int64) (*Response[interface{}], error) {
	serverUrl := fmt.Sprintf("%s/x/%s/bizy_models/%d", c.Domain, meta.APIv1, bizyModelId)
	body, statusCode, err := c.doDelete(serverUrl, nil, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}
	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[interface{}](body)
}

func (c *Client) CheckModel(modelType string, modelName string) (*Response[CheckModelResp], error) {
	serverUrl := fmt.Sprintf("%s/x/%s/models/check", c.Domain, meta.APIv1)
	body, statusCode, err := c.doGet(serverUrl, ModelQueryReq{
		Name: modelName,
		Type: modelType,
	}, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}

	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[CheckModelResp](body)
}

// GetUploadToken 获取临时上传凭证（inputs）
func (c *Client) GetUploadToken(fileName, fileType string) (*Response[FilesResp], error) {
	serverUrl := fmt.Sprintf("%s/x/%s/upload/token", c.Domain, meta.APIv1)
	body, statusCode, err := c.doGet(serverUrl, UploadTokenReq{FileName: fileName, FileType: fileType}, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}
	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[FilesResp](body)
}

// CommitInputResource 提交输入资源，返回可用 url
func (c *Client) CommitInputResource(name, objectKey string) (*Response[InputResourceCommitResp], error) {
	serverUrl := fmt.Sprintf("%s/x/%s/input_resource/commit", c.Domain, meta.APIv1)
	body, statusCode, err := c.doPost(serverUrl, InputResourceCommitReq{Name: name, ObjectKey: objectKey}, c.authHeader())
	if err != nil {
		return nil, cli.Exit(err, meta.ServerError)
	}
	if statusCode != http.StatusOK {
		return nil, cli.Exit(handleError(body, statusCode), meta.ServerError)
	}
	return handleResponse[InputResourceCommitResp](body)
}

// CheckModelExists 检查模型名是否已存在
// 返回 true 表示模型名已存在（HTTP 200），false 表示不存在（HTTP 404）
func (c *Client) CheckModelExists(modelName string, modelType string) (bool, error) {
	serverUrl := fmt.Sprintf("%s/x/%s/bizy_models/exists", c.Domain, meta.APIv1)
	body, statusCode, err := c.doGet(serverUrl, ModelQueryReq{
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
	// 使用固定的社区API地址
	serverUrl := "https://bizyair.cn/api/special/community/base_model_types"
	body, statusCode, err := c.doGet(serverUrl, nil, nil)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		return nil, handleError(body, statusCode)
	}

	return handleResponse[[]*BaseModelTypeItem](body)
}

func (c *Client) authHeader() map[string]string {
	header := make(map[string]string)
	header[meta.HeaderAuthorization] = fmt.Sprintf("Bearer %s", c.ApiKey)
	return header
}

// doGet do get request
func (c *Client) doGet(urlStr string, queryParams interface{}, header map[string]string) ([]byte, int, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	client := &http.Client{Transport: tr}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, -1, err
	}

	if queryParams != nil {
		v := reflect.ValueOf(queryParams)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		query := parsedURL.Query()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldName := v.Type().Field(i).Name
			snakeName := lo.SnakeCase(fieldName)

			// 处理数组/切片类型
			if field.Kind() == reflect.Slice {
				for j := 0; j < field.Len(); j++ {
					elemValue := fmt.Sprintf("%v", field.Index(j).Interface())
					if elemValue != "" {
						query.Add(snakeName, elemValue)
					}
				}
			} else {
				// 处理普通类型
				fieldValue := fmt.Sprintf("%v", field.Interface())
				if fieldValue != "" && fieldValue != "0" && fieldValue != "false" && fieldValue != "[]" {
					query.Add(snakeName, fieldValue)
				}
			}
		}
		parsedURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequest(meta.HTTPGet, parsedURL.String(), nil)
	if err != nil {
		return nil, -1, err
	}

	if len(header) > 0 {
		for key, value := range header {
			req.Header.Set(key, value)
		}
	}
	req.Header.Set(meta.HeaderSiliconCliVersion, meta.Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// doPost do post request
func (c *Client) doPost(url string, data interface{}, header map[string]string) ([]byte, int, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	client := &http.Client{Transport: tr}

	// 将数据编码为JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, -1, err
	}

	req, err := http.NewRequest(meta.HTTPPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, -1, err
	}

	if len(header) > 0 {
		for key, value := range header {
			req.Header.Set(key, value)
		}
	}
	req.Header.Set(meta.HeaderSiliconCliVersion, meta.Version)
	req.Header.Set(meta.HeaderContentType, meta.JsonContentType)

	resp, err := client.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// doDelete do delete request
func (c *Client) doDelete(url string, data interface{}, header map[string]string) ([]byte, int, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	client := &http.Client{Transport: tr}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, -1, err
	}

	req, err := http.NewRequest(meta.HTTPDelete, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, -1, err
	}

	if len(header) > 0 {
		for key, value := range header {
			req.Header.Set(key, value)
		}
	}
	req.Header.Set(meta.HeaderContentType, meta.JsonContentType)

	resp, err := client.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func handleError(responseBody []byte, statusCode int) error {
	rawMessage := string(responseBody)
	if statusCode == http.StatusNotFound {
		return cli.Exit(fmt.Errorf("server not found, you can use \"--base_domain\" to specify the target domain"), meta.LoadError)
	}
	var parsedResponse Response[interface{}]
	err := json.Unmarshal(responseBody, &parsedResponse)
	if err != nil {
		rawMessage = strings.TrimFunc(rawMessage, func(r rune) bool {
			return unicode.Is(unicode.Quotation_Mark, r)
		})
		if rawMessage == "" {
			return cli.Exit(meta.NewErrNo("Unknown server error"), meta.ServerError)
		}
		return cli.Exit(meta.NewErrNo(rawMessage), meta.ServerError)
	}

	if errno, exists := meta.ServerErrors[parsedResponse.Code]; exists {
		return cli.Exit(errno, meta.ServerError)
	}

	return fmt.Errorf("unexcepted http status code: %d, message: %s", statusCode, rawMessage)
}

func handleResponse[T any](responseBody []byte) (*Response[T], error) {
	var parsedResponse Response[T]
	err := json.Unmarshal(responseBody, &parsedResponse)
	if err != nil {
		logs.Debugf("error: %s\n", err)
		return nil, cli.Exit(err, meta.ServerError)
	}

	if parsedResponse.Code != meta.OKCode {
		if errno, exists := meta.ServerErrors[parsedResponse.Code]; exists {
			return nil, cli.Exit(errno, meta.ServerError)
		}
		return nil, cli.Exit(fmt.Errorf("server error: %s", parsedResponse.Message), meta.ServerError)
	}
	return &parsedResponse, nil
}
