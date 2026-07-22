package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/internal/i18n"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func Upload(c *cli.Context) error {
	args := parseArgument(c, meta.CmdUpload)
	setLogVerbose(args.Verbose)
	logArguments(args)

	// 检查是否使用 YAML 配置文件批量上传
	if args.FilePath != "" {
		// 使用 YAML 配置文件
		if err := uploadFromYaml(c.Context, args.FilePath, args); err != nil {
			return cli.Exit(err, meta.LoadError)
		}
		return nil
	}

	// 获取 API Key
	apiKey := args.ApiKey
	if apiKey == "" {
		var err error
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return cli.Exit(err, meta.LoadError)
		}
	}

	// 准备版本输入参数
	versions := make([]actions.VersionInput, len(args.Path))
	for i := range args.Path {
		// 优先从文件读取 intro，其次使用直接提供的 intro
		intro := ""
		introPath := getStringAt(args.IntroPath, i, "")
		if introPath != "" {
			// 从文件读取
			if err := lib.ValidateIntroFile(introPath); err != nil {
				return cli.Exit(i18n.NewError("cli.upload.intro_validation_failed", map[string]any{"Version": i + 1}, err), meta.LoadError)
			}
			content, err := lib.ReadIntroFile(introPath)
			if err != nil {
				return cli.Exit(i18n.NewError("cli.upload.intro_read_failed", map[string]any{"Version": i + 1}, err), meta.LoadError)
			}
			intro = content
			fmt.Fprintln(os.Stdout, i18n.T("cli.upload.intro_loaded", map[string]any{
				"Current": i + 1, "Total": len(args.Path), "Path": introPath,
			}))
		} else {
			// 使用直接提供的 intro
			intro = getStringAt(args.Intro, i, "")
		}

		// 验证 intro 不能为空
		if strings.TrimSpace(intro) == "" {
			return cli.Exit(i18n.NewError("cli.upload.intro_required", map[string]any{"Version": i + 1}, nil), meta.LoadError)
		}

		versions[i] = actions.VersionInput{
			Version:      getVersionAt(args.ModelVersion, i, fmt.Sprintf("v%d.0", i+1)),
			Path:         args.Path[i],
			BaseModel:    getStringAt(args.BaseModel, i, ""),
			Introduction: intro,
			CoverUrl:     getStringAt(args.CoverUrls, i, ""),
			Public:       getBoolAt(args.VersionPublic, i, false),
		}
	}

	// 准备上传输入参数
	input := actions.UploadInput{
		ApiKey:     apiKey,
		BaseDomain: args.BaseDomain,
		ModelType:  args.Type,
		ModelName:  args.Name,
		Versions:   versions,
		Overwrite:  args.Overwrite,
		Context:    c.Context,
	}

	client := lib.NewClient(args.BaseDomain, apiKey)
	resp, err := client.GetBaseModelTypesContext(c.Context)
	if err != nil {
		return cli.Exit(i18n.NewError("error.base_models.fetch_failed", nil, err), meta.ServerError)
	}

	if resp.Data != nil {
		for _, item := range resp.Data {
			input.AllowedBaseModels = append(input.AllowedBaseModels, item.Value)
		}
	}

	// 创建CLI回调
	callback := &cliUploadCallback{}

	// 执行上传
	fmt.Fprintln(os.Stdout, i18n.TN("cli.upload.start", len(versions), map[string]any{"Count": len(versions), "Concurrency": 3}))

	result := actions.ExecuteUpload(client, input, callback)

	// 处理结果
	if !result.Success {
		if result.CanceledByUser {
			fmt.Fprintf(os.Stdout, "\n%s\n", i18n.T("cli.upload.canceled"))
			folder, _ := lib.GetCheckpointDir()
			if folder != "" {
				fmt.Fprintln(os.Stdout, i18n.T("cli.upload.checkpoint_saved"))
				fmt.Fprintln(os.Stdout, i18n.T("cli.upload.checkpoint_location", map[string]any{"Path": folder}))
			}
			return nil
		}

		fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("cli.upload.failure_list"))
		for _, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		return cli.Exit(i18n.T("cli.upload.failed"), meta.ServerError)
	}

	fmt.Fprintf(os.Stdout, "\n✓ %s\n", i18n.T("cli.upload.success"))
	if result.SuccessCount < result.TotalCount {
		fmt.Fprintln(os.Stdout, i18n.T("cli.upload.partial", map[string]any{
			"Success": result.SuccessCount, "Total": result.TotalCount,
		}))
		for _, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		return nil
	}

	// 全部成功时，显示模型详情
	displayUploadedModelDetail(c.Context, apiKey, args.BaseDomain, result.ModelName, result.ModelType)
	return nil
}

// displayUploadedModelDetail 显示刚上传的模型详情
func displayUploadedModelDetail(ctx context.Context, apiKey, baseDomain, modelName, modelType string) {
	if ctx == nil {
		ctx = context.Background()
	}
	// 后端需要时间处理，先等待1秒
	select {
	case <-time.After(time.Second):
	case <-ctx.Done():
		return
	}

	// 查询模型列表，带重试逻辑
	listInput := actions.ListModelsInput{
		Context:    ctx,
		ApiKey:     apiKey,
		BaseDomain: baseDomain,
		ModelType:  modelType,
		Current:    1,
		PageSize:   100,
		Sort:       "Recently",
	}

	client := lib.NewClient(baseDomain, apiKey)

	var targetModel *lib.BizyModelInfo
	maxRetries := 3
	retryDelay := time.Second

	// 重试查询模型
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			// 后续重试等待
			select {
			case <-time.After(retryDelay):
			case <-ctx.Done():
				return
			}
		}

		listResult := actions.ListModels(client, listInput)
		if listResult.Error != nil {
			fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("cli.upload.model_id_failed", map[string]any{"Cause": listResult.Error}))
			return
		}

		// 查找匹配的模型
		for _, model := range listResult.Models {
			if model.Name == modelName {
				targetModel = model
				break
			}
		}

		if targetModel != nil {
			break
		}

		// 如果不是最后一次重试，显示等待信息
		if i < maxRetries-1 {
			logs.Debugf("Model not found; retrying in %d second(s).\n", retryDelay/time.Second)
		}
	}

	if targetModel == nil {
		fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("cli.upload.published_model_not_found", map[string]any{"Retries": maxRetries}))
		return
	}

	// 构建模型详情页面 URL
	endpoints, err := lib.ResolveServiceEndpoints(baseDomain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s\n", err)
		return
	}
	modelURL := endpoints.ModelDetailURL(targetModel.Id)

	// 显示成功提示和链接
	fmt.Fprintf(os.Stdout, "\n%s\n", i18n.T("cli.upload.published"))
	fmt.Fprintln(os.Stdout, i18n.T("cli.upload.model_link", map[string]any{"URL": modelURL}))

	// 尝试在浏览器中打开模型详情页面
	msg, err := lib.OpenBrowser(modelURL)
	if err != nil {
		// 如果无法打开浏览器，只是提示，不影响整体流程
		fmt.Fprintln(os.Stdout, i18n.T("cli.upload.browser_hint"))
	} else {
		fmt.Fprintf(os.Stdout, "%s\n", msg)
	}
}

// cliUploadCallback CLI的进度回调实现
type cliUploadCallback struct{}

func (c *cliUploadCallback) OnProgress(progress actions.UploadProgress) {
	if progress.Total > 0 {
		percent := float64(progress.Consumed) / float64(progress.Total)
		bar := renderProgressBar(percent)
		fmt.Printf("\r(%d/%d) %s %s %.1f%% (%s/%s)",
			progress.VersionIndex+1,
			progress.VersionTotal,
			progress.FileName,
			bar,
			percent*100,
			format.FormatBytes(progress.Consumed),
			format.FormatBytes(progress.Total))

		if percent >= 1.0 {
			fmt.Println()
		}
	}
}

func (c *cliUploadCallback) OnVersionStart(index, total int, fileName string) {
	fmt.Println(i18n.T("cli.upload.version_start", map[string]any{"Current": index + 1, "Total": total, "File": fileName}))
}

func (c *cliUploadCallback) OnVersionComplete(index, total int, fileName string, err error) {
	if err != nil {
		fmt.Println(i18n.T("cli.upload.version_failed", map[string]any{
			"Current": index + 1, "Total": total, "File": fileName, "Cause": err,
		}))
	} else {
		fmt.Println(i18n.T("cli.upload.version_complete", map[string]any{"Current": index + 1, "Total": total, "File": fileName}))
	}
}

func (c *cliUploadCallback) OnCoverStatus(index, total int, status, message string) {
	// CLI模式输出警告信息
	if status == "fallback" {
		fmt.Fprintln(os.Stderr, i18n.T("cli.upload.cover_warning", map[string]any{
			"Current": index + 1, "Total": total, "Message": message,
		}))
	}
}

// 辅助函数
func getVersionAt(versions []string, index int, defaultValue string) string {
	if index < len(versions) && versions[index] != "" {
		return versions[index]
	}
	return defaultValue
}

func getStringAt(slice []string, index int, defaultValue string) string {
	if index < len(slice) {
		return slice[index]
	}
	return defaultValue
}

func getBoolAt(slice []string, index int, defaultValue bool) bool {
	if index < len(slice) {
		return slice[index] == "true" || slice[index] == "True" || slice[index] == "1"
	}
	return defaultValue
}
