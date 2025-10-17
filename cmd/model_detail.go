package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/lib/format"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func ModelDetail(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdDetail)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

	// 验证参数
	if args.Type == "" {
		return cli.Exit(fmt.Errorf("必须指定模型类型 (-t)"), meta.LoadError)
	}
	if args.Name == "" {
		return cli.Exit(fmt.Errorf("必须指定模型名称 (-n)"), meta.LoadError)
	}

	if err := lib.ValidateModelType(args.Type); err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	if err := lib.ValidateModelName(args.Name); err != nil {
		return cli.Exit(err, meta.LoadError)
	}

	// 获取API Key
	var apiKey string
	if args.ApiKey != "" {
		apiKey = args.ApiKey
	} else {
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return cli.Exit(err, meta.LoadError)
		}
	}

	// 准备输入参数，查询模型列表
	input := actions.ListModelsInput{
		ApiKey:     apiKey,
		BaseDomain: args.BaseDomain,
		ModelType:  args.Type,
		Public:     false,
		Current:    1,
		PageSize:   100,
		Sort:       "Recently",
	}

	// 调用统一的业务逻辑查询模型列表
	result := actions.ListModels(input)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}

	// 查找匹配的模型
	var targetModel *lib.BizyModelInfo
	for _, model := range result.Models {
		if model.Name == args.Name {
			targetModel = model
			break
		}
	}

	if targetModel == nil {
		return cli.Exit(fmt.Errorf("未找到名为 '%s' 的 %s 类型模型", args.Name, args.Type), meta.LoadError)
	}

	// 获取模型详情
	detailResult := actions.GetModelDetail(apiKey, args.BaseDomain, targetModel.Id)
	if detailResult.Error != nil {
		return cli.Exit(detailResult.Error, meta.ServerError)
	}

	detail := detailResult.Detail

	// 显示基本信息
	fmt.Fprintf(os.Stdout, "=== 模型详情 ===\n\n")
	fmt.Fprintf(os.Stdout, "ID:       %d\n", detail.Id)
	fmt.Fprintf(os.Stdout, "名称:     %s\n", detail.Name)
	fmt.Fprintf(os.Stdout, "类型:     %s\n", detail.Type)
	fmt.Fprintf(os.Stdout, "来源:     %s\n", detail.Source)
	fmt.Fprintf(os.Stdout, "创建者:   %s\n", detail.UserName)
	fmt.Fprintf(os.Stdout, "创建时间: %s\n", detail.CreatedAt)
	fmt.Fprintf(os.Stdout, "更新时间: %s\n", detail.UpdatedAt)

	// 显示版本列表
	if len(detail.Versions) > 0 {
		fmt.Fprintf(os.Stdout, "\n=== 版本列表 (%d) ===\n\n", len(detail.Versions))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "VERSION\tBASE MODEL\tFILENAME\tSIZE\tSIGN\tCREATED AT\t")

		for _, version := range detail.Versions {
			// 处理文件大小
			sizeStr := "-"
			if version.FileSize > 0 {
				sizeStr = format.FormatBytes(version.FileSize)
			}

			// 基础模型，如果为空显示 "-"
			baseModel := version.BaseModel
			if baseModel == "" {
				baseModel = "-"
			}

			// 文件签名，如果为空显示 "-"
			sign := version.Sign
			if sign == "" {
				sign = "-"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
				version.Version,
				baseModel,
				version.FileName,
				sizeStr,
				sign,
				version.CreatedAt,
			)
		}
		w.Flush()
	} else {
		fmt.Fprintf(os.Stdout, "\n暂无版本信息\n")
	}

	return nil
}
