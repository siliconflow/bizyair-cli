package cmd

import (
	"fmt"
	"os"

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
		versions[i] = actions.VersionInput{
			Version:      getVersionAt(args.ModelVersion, i, fmt.Sprintf("v%d", i)),
			Path:         args.Path[i],
			BaseModel:    getStringAt(args.BaseModel, i, ""),
			Introduction: getStringAt(args.Intro, i, ""),
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
	}
	return nil
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
