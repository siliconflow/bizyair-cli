package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func Upload(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdUpload)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	// 检查是否使用 YAML 配置文件批量上传
	if args.FilePath != "" {
		// 使用 YAML 配置文件
		if err := uploadFromYaml(args.FilePath, args); err != nil {
			return cli.Exit(err, meta.LoadError)
		}
		return nil
	}

	// 获取 API Key
	apiKey := args.ApiKey
	if apiKey == "" {
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
				return cli.Exit(fmt.Errorf("intro 文件验证失败 [版本 %d]: %w", i+1, err), meta.LoadError)
			}
			content, err := lib.ReadIntroFile(introPath)
			if err != nil {
				return cli.Exit(fmt.Errorf("读取 intro 文件失败 [版本 %d]: %w", i+1, err), meta.LoadError)
			}
			intro = content
			fmt.Fprintf(os.Stdout, "已从文件读取 intro (版本 %d/%d): %s\n", i+1, len(args.Path), introPath)
		} else {
			// 使用直接提供的 intro
			intro = getStringAt(args.Intro, i, "")
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
	}

	// 创建CLI回调
	callback := &cliUploadCallback{}

	// 执行上传
	fmt.Fprintf(os.Stdout, "开始上传 %d 个文件（并发数：3）\n", len(versions))
	result := actions.ExecuteUpload(input, callback)

	// 处理结果
	if !result.Success {
		if result.CanceledByUser {
			fmt.Fprintln(os.Stdout, "\n上传已取消")
			folder, _ := lib.GetCheckpointDir()
			if folder != "" {
				fmt.Fprintf(os.Stdout, "已上传的部分已保存checkpoint，下次上传相同文件时会自动续传。\n")
				fmt.Fprintf(os.Stdout, "Checkpoint文件位置: %s\n", folder)
			}
			return nil
		}

		fmt.Fprintf(os.Stderr, "\n上传失败，错误列表：\n")
		for _, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		return cli.Exit("上传失败", meta.ServerError)
	}

	fmt.Fprintf(os.Stdout, "\n✓ 上传成功！\n")
	if result.SuccessCount < result.TotalCount {
		fmt.Fprintf(os.Stdout, "部分版本失败：成功 %d/%d\n",
			result.SuccessCount, result.TotalCount)
		for _, err := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", err)
		}
		return nil
	}

	// 全部成功时，显示模型详情
	displayUploadedModelDetail(apiKey, args.BaseDomain, result.ModelName, result.ModelType)
	return nil
}

// displayUploadedModelDetail 显示刚上传的模型详情
func displayUploadedModelDetail(apiKey, baseDomain, modelName, modelType string) {
	// 后端需要时间处理，先等待1秒
	time.Sleep(time.Second)

	// 查询模型列表，带重试逻辑
	listInput := actions.ListModelsInput{
		ApiKey:     apiKey,
		BaseDomain: baseDomain,
		ModelType:  modelType,
		Current:    1,
		PageSize:   100,
		Sort:       "Recently",
	}

	var targetModel *lib.BizyModelInfo
	maxRetries := 3
	retryDelay := time.Second

	// 重试查询模型
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			// 后续重试等待
			time.Sleep(retryDelay)
		}

		listResult := actions.ListModels(listInput)
		if listResult.Error != nil {
			fmt.Fprintf(os.Stderr, "\n获取模型ID失败: %v\n", listResult.Error)
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
			logs.Debugf("未找到模型，%d秒后重试...\n", retryDelay/time.Second)
		}
	}

	if targetModel == nil {
		fmt.Fprintf(os.Stderr, "\n未找到刚上传的模型（已重试%d次），请稍后手动通过 TUI 或 API 查看\n", maxRetries)
		return
	}

	// 显示成功提示和链接
	fmt.Fprintf(os.Stdout, "\n模型发布成功，请在浏览器登录 https://bizyair.cn 在：https://bizyair.cn/community/models/my/%d 查看\n", targetModel.Id)
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
	fmt.Printf("开始上传 (%d/%d): %s\n", index+1, total, fileName)
}

func (c *cliUploadCallback) OnVersionComplete(index, total int, fileName string, err error) {
	if err != nil {
		fmt.Printf("✗ (%d/%d) %s 上传失败: %v\n", index+1, total, fileName, err)
	} else {
		fmt.Printf("✓ (%d/%d) %s 上传完成\n", index+1, total, fileName)
	}
}

func (c *cliUploadCallback) OnCoverStatus(index, total int, status, message string) {
	// CLI模式输出警告信息
	if status == "fallback" {
		fmt.Fprintf(os.Stderr, "⚠ 警告 - 版本 %d/%d: %s\n", index+1, total, message)
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
