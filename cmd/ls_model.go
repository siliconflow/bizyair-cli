package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
	"github.com/siliconflow/bizyair-cli/lib/actions"
	"github.com/siliconflow/bizyair-cli/meta"
	"github.com/urfave/cli/v2"
)

func ListModel(c *cli.Context) error {
	args, err := globalArgs.Parse(c, meta.CmdLs)
	if err != nil {
		return cli.Exit(err, meta.LoadError)
	}
	setLogVerbose(args.Verbose)
	logs.Debugf("args: %#v\n", args)

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

	// 准备输入参数
	input := actions.ListModelsInput{
		ApiKey:     apiKey,
		BaseDomain: args.BaseDomain,
		ModelType:  args.Type,
		Public:     args.Public,
		Current:    1,
		PageSize:   100,
		Sort:       "Recently",
	}

	// 调用统一的业务逻辑
	result := actions.ListModels(input)
	if result.Error != nil {
		return cli.Exit(result.Error, meta.ServerError)
	}

	// 格式化输出
	if len(result.Models) < 1 {
		fmt.Fprintln(os.Stdout, "No models found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tVERSIONS\tUSED\tFORKED\tLIKED\tUPDATED AT\t")

	// 打印数据行
	for _, model := range result.Models {
		// 获取版本信息
		versionCount := len(model.Versions)
		var versionStr string
		if versionCount > 0 {
			versions := []string{}
			for _, v := range model.Versions {
				versions = append(versions, v.Version)
			}
			versionStr = strings.Join(versions, ",")
			if len(versionStr) > 20 {
				versionStr = versionStr[:17] + "..."
			}
		} else {
			versionStr = "-"
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\t%d\t%s\t\n",
			model.Id,
			model.Name,
			model.Type,
			versionStr,
			model.Counter.UsedCount,
			model.Counter.ForkedCount,
			model.Counter.LikedCount,
			model.UpdatedAt,
		)
	}
	w.Flush()

	// 显示统计信息
	fmt.Fprintf(os.Stdout, "\nTotal: %d models (showing page 1)\n", result.Total)

	return nil
}
