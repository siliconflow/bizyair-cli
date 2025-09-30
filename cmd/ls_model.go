package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/cloudwego/hertz/cmd/hz/util/logs"
	"github.com/siliconflow/bizyair-cli/lib"
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

	var apiKey string
	if args.ApiKey != "" {
		apiKey = args.ApiKey
	} else {
		apiKey, err = lib.NewSfFolder().GetKey()
		if err != nil {
			return err
		}
	}

	client := lib.NewClient(args.BaseDomain, apiKey)

	// 构建模型类型列表
	var modelTypes []string
	if args.Type != "" {
		modelTypes = []string{args.Type}
	} else {
		// 如果未指定类型，查询所有类型
		for _, t := range meta.ModelTypes {
			modelTypes = append(modelTypes, string(t))
		}
	}

	// 调用新接口
	// current=1, pageSize=100, keyword="", sort="Recently", modelTypes, baseModels=[]
	modelResp, err := client.ListModel(1, 100, "", "Recently", modelTypes, []string{})
	if err != nil {
		return err
	}

	modelList := modelResp.Data.List

	if len(modelList) < 1 {
		fmt.Fprintln(os.Stdout, "No models found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tVERSIONS\tUSED\tFORKED\tLIKED\tUPDATED AT\t")

	// 打印数据行
	for _, model := range modelList {
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
	fmt.Fprintf(os.Stdout, "\nTotal: %d models (showing page 1)\n", modelResp.Data.Total)

	return nil
}
